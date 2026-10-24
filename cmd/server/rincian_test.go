package main

import (
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"pajak/internal/finance"
)

func TestStok_TidakBolehHapusBarangDiRincian(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	qa := register(t, srv, registerBody)
	aktifkan(t, srv, qa)

	kopi := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"sku":"BHN-KOPI","nama":"Biji kopi","kategori":"Bahan Baku","satuan":"kg","harga_pokok":200000}`)
	// barang hanya muncul di rincian, bukan di stok_id
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions",
		`{"tanggal":"2026-01-05","jenis":"Beli Bahan / Stok","nominal":50000,"lunas":true,"rincian":[{"stok_id":`+itoa(kopi.ID)+`,"qty_stok":5000}]}`)
	if rec := do(srv, qa, http.MethodDelete, "/api/v1/stok/"+itoa(kopi.ID), ""); rec.Code != http.StatusConflict {
		t.Errorf("hapus barang di rincian: %d, want 409", rec.Code)
	}
	// barang tak dikenal di rincian ditolak
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions",
		`{"tanggal":"2026-01-05","jenis":"Beli Bahan / Stok","nominal":50000,"lunas":true,"rincian":[{"stok_id":999,"qty_stok":5000}]}`); rec.Code == http.StatusCreated {
		t.Errorf("barang tak dikenal di rincian harus ditolak: %d", rec.Code)
	}
}

// Kolom transactions yang ditambah lewat ALTER dan field Go-nya omitempty tidak boleh NOT NULL:
// jsonb_populate_recordset mengisi NULL untuk key yang hilang, jadi NOT NULL menggagalkan simpan.
func TestSchema_KolomOmitemptyTransactionsNullable(t *testing.T) {
	sql, err := os.ReadFile("schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	omit := map[string]bool{}
	rt := reflect.TypeFor[finance.Transaction]()
	for i := range rt.NumField() {
		f := rt.Field(i)
		tag := f.Tag.Get("json")
		if nama, opsi, _ := strings.Cut(tag, ","); strings.Contains(opsi, "omitempty") {
			omit[nama] = true
		}
	}
	re := regexp.MustCompile(`(?im)^ALTER TABLE transactions ADD COLUMN IF NOT EXISTS (\w+)([^;]*);`)
	n := 0
	for _, m := range re.FindAllStringSubmatch(string(sql), -1) {
		n++
		if omit[m[1]] && strings.Contains(strings.ToUpper(m[2]), "NOT NULL") {
			t.Errorf("kolom %s omitempty tetapi NOT NULL", m[1])
		}
	}
	if n == 0 {
		t.Fatal("tidak ada ALTER transactions terbaca")
	}
}
