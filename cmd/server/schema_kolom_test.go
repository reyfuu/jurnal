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
	for _, m := range regexp.MustCompile(`ALTER TABLE ` + tabel + ` ADD COLUMN IF NOT EXISTS (\w+)`).FindAllStringSubmatch(sql, -1) {
		out[m[1]] = true
	}
	return out
}

// Setiap field JSON struct yang disimpan per kolom harus punya kolomnya; kalau tidak,
// jsonb_populate_recordset membuangnya diam-diam saat simpan ke Postgres.
func TestSchema_SetiapFieldPunyaKolom(t *testing.T) {
	for tabel, v := range map[string]any{"stok": finance.StokItem{}, "transactions": finance.Transaction{}} {
		kolom := kolomTabel(t, tabel)
		ty := reflect.TypeOf(v)
		for i := range ty.NumField() {
			nama, _, _ := strings.Cut(ty.Field(i).Tag.Get("json"), ",")
			if nama == "" || nama == "-" {
				continue
			}
			if !kolom[nama] {
				t.Errorf("tabel %s tidak punya kolom %q untuk field %s", tabel, nama, ty.Field(i).Name)
			}
		}
	}
}
