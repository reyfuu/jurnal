package primbon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func metadataDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	data, err := os.ReadFile(filepath.Join(fixture, "index.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "index.csv"), data, 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}
func TestMetadataTagsWithoutNaskah(t *testing.T) {
	dir := metadataDir(t)
	d, err := Muat(dir)
	if err != nil {
		t.Fatal(err)
	}
	id := d.list[0].UUID
	raw := `{"` + id + `":{"tag":["PPh 21","Payroll","payroll",""],"status":"Berlaku","perubahan":"Diubah","mulai_berlaku":" 2026-01-01 ","status_pada":" 2026-10-22 ","diverifikasi_pada":"2026-10-23","sumber_verifikasi":"https://jdih.kemenkeu.go.id/"}}`
	if err = os.WriteFile(filepath.Join(dir, "metadata.json"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	d, err = Muat(dir)
	if err != nil {
		t.Fatal(err)
	}
	p := ambil(t, d, id)
	if p.Format != "" || len(p.Tag) != 2 || p.Perubahan != "Diubah" || p.StatusPada != "2026-10-22" || p.Berlaku != "2026-01-01" {
		t.Fatalf("metadata without corpus: %+v", p)
	}
	h := d.Cari(Query{Q: "Payroll"})
	if h.Total != 1 || h.Items[0].UUID != id {
		t.Fatalf("tag not searchable: %+v", h)
	}
}
func TestMetadataReloadRetainsLastGood(t *testing.T) {
	dir := metadataDir(t)
	catalog := Buka(dir)
	d, err := catalog.Data()
	if err != nil {
		t.Fatal(err)
	}
	id := d.list[0].UUID
	write := func(raw string, offset int) {
		t.Helper()
		p := filepath.Join(dir, "metadata.json")
		if err := os.WriteFile(p, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		stamp := time.Now().Add(time.Duration(offset) * time.Second)
		if err := os.Chtimes(p, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	write(`{"`+id+`":{"tag":["Payroll"]}}`, 1)
	d, err = catalog.Data()
	if err != nil || len(ambil(t, d, id).Tag) != 1 {
		t.Fatal("metadata change ignored", err)
	}
	write(`{"`+id+`":{"status_pada":"2026-02-31"}}`, 2)
	d, err = catalog.Data()
	if err != nil || len(ambil(t, d, id).Tag) != 1 {
		t.Fatal("bad metadata discarded last good state", err)
	}
	write(`{"`+id+`":{"tag":["Valid"]}}`, 2)
	d, err = catalog.Data()
	if err != nil || ambil(t, d, id).Tag[0] != "Valid" {
		t.Fatal("invalid reload was not retried", err)
	}
	if err = os.Remove(filepath.Join(dir, "metadata.json")); err != nil {
		t.Fatal(err)
	}
	d, err = catalog.Data()
	if err != nil || len(ambil(t, d, id).Tag) != 0 {
		t.Fatal("metadata removal ignored", err)
	}
}
func TestMetadataRejectsUnsafeOrInvalidInput(t *testing.T) {
	for _, fragment := range []string{`"sumber_verifikasi":"javascript:alert(1)"`, `"diverifikasi_pada":"bad-date"`, `"unknown_field":true`, `"tag":["` + strings.Repeat("a", 81) + `"]`} {
		t.Run(fragment[:min(25, len(fragment))], func(t *testing.T) {
			dir := metadataDir(t)
			if err := os.WriteFile(filepath.Join(dir, "metadata.json"), []byte(`{"id":{`+fragment+`}}`), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Muat(dir); err == nil {
				t.Fatal("invalid metadata accepted")
			}
		})
	}
}
