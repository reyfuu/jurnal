package main

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"pajak/internal/finance"
)

// kolomTabel: nama kolom tabel di schema.sql, dari CREATE TABLE dan ALTER TABLE ... ADD COLUMN.
func kolomTabel(t *testing.T, tabel string) map[string]bool {
	t.Helper()
	b, err := os.ReadFile("schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ReplaceAll(string(b), "\r\n", "\n")
	out := map[string]bool{}
	buat := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS ` + tabel + ` \((.*?)\n\);`).FindStringSubmatch(sql)
	if buat == nil {
		t.Fatalf("CREATE TABLE %s tidak ditemukan", tabel)
	}
	for _, baris := range strings.Split(buat[1], "\n") {
		f := strings.Fields(strings.TrimSpace(baris))
		if len(f) > 0 && !strings.HasPrefix(f[0], "--") && f[0] != "PRIMARY" && f[0] != "UNIQUE" && f[0] != "FOREIGN" {
			out[f[0]] = true
		}
	}
	for _, m := range regexp.MustCompile(`ALTER TABLE `+tabel+` ADD COLUMN IF NOT EXISTS (\w+)`).FindAllStringSubmatch(sql, -1) {
		out[m[1]] = true
	}
	return out
}

// fieldJSON: nama field JSON struct, termasuk field struct yang di-embed tanpa tag (mis.
// invoiceItemRow meng-embed finance.InvoiceItem: field-nya jadi kolom invoice_items).
func fieldJSON(ty reflect.Type) map[string]string {
	out := map[string]string{}
	for i := range ty.NumField() {
		f := ty.Field(i)
		tag := f.Tag.Get("json")
		if f.Anonymous && tag == "" {
			for k, v := range fieldJSON(f.Type) {
				out[k] = v
			}
			continue
		}
		nama, _, _ := strings.Cut(tag, ",")
		if nama == "" || nama == "-" || !f.IsExported() {
			continue
		}
		out[nama] = f.Name
	}
	return out
}

// Setiap field JSON struct yang disimpan per kolom harus punya kolomnya; kalau tidak,
// jsonb_populate_recordset membuangnya diam-diam saat simpan ke Postgres. Tabel & tipe sama
// dengan saveTenantDB (pgstore.go).
func TestSchema_SetiapFieldPunyaKolom(t *testing.T) {
	for _, c := range []struct {
		tabel string
		v     any
		// bukanKolom: field yang sengaja tidak punya kolom karena pgstore menyimpannya di
		// tempat lain (alasan per field).
		bukanKolom map[string]string
	}{
		{"stok", finance.StokItem{}, nil},
		{"transactions", finance.Transaction{}, nil},
		{"invoices", finance.Invoice{}, map[string]string{
			"items": "baris faktur disimpan per baris di invoice_items (saveTenantDB → invoiceItemRow)",
		}},
		{"invoice_items", invoiceItemRow{}, nil},
	} {
		kolom := kolomTabel(t, c.tabel)
		for nama, field := range fieldJSON(reflect.TypeOf(c.v)) {
			if _, ok := c.bukanKolom[nama]; ok {
				if kolom[nama] {
					t.Errorf("tabel %s kini punya kolom %q: hapus pengecualiannya", c.tabel, nama)
				}
				continue
			}
			if !kolom[nama] {
				t.Errorf("tabel %s tidak punya kolom %q untuk field %s", c.tabel, nama, field)
			}
		}
	}
}
