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

// Sama untuk invoice_items (Task 9): kolom stok_id & bukan_barang ada, dan kolom yang field
// InvoiceItem-nya omitempty tidak NOT NULL.
func TestSchema_KolomOmitemptyInvoiceItemsNullable(t *testing.T) {
	cekKolomOmitemptyNullable(t, "invoice_items", reflect.TypeFor[finance.InvoiceItem](), "stok_id", "bukan_barang")
}

// Sama untuk invoices: rekening_lunas (pelunasan lewat impor mutasi) ada dan nullable.
func TestSchema_KolomOmitemptyInvoicesNullable(t *testing.T) {
	cekKolomOmitemptyNullable(t, "invoices", reflect.TypeFor[finance.Invoice](), "rekening_lunas")
}

// cekKolomOmitemptyNullable: kolom ALTER tabel yang field Go-nya omitempty tidak NOT NULL, dan
// kolom wajib ada sebagai ALTER dengan field omitempty.
func cekKolomOmitemptyNullable(t *testing.T, tabel string, rt reflect.Type, wajib ...string) {
	t.Helper()
	sql, err := os.ReadFile("schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	omit := map[string]bool{}
	for i := range rt.NumField() {
		if nama, opsi, _ := strings.Cut(rt.Field(i).Tag.Get("json"), ","); strings.Contains(opsi, "omitempty") {
			omit[nama] = true
		}
	}
	re := regexp.MustCompile(`(?im)^ALTER TABLE ` + tabel + ` ADD COLUMN IF NOT EXISTS (\w+)([^;]*);`)
	ada := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(string(sql), -1) {
		ada[m[1]] = true
		if omit[m[1]] && strings.Contains(strings.ToUpper(m[2]), "NOT NULL") {
			t.Errorf("kolom %s.%s omitempty tetapi NOT NULL", tabel, m[1])
		}
	}
	for _, k := range wajib {
		if !ada[k] || !omit[k] {
			t.Errorf("%s.%s: kolom ada %v, field omitempty %v", tabel, k, ada[k], omit[k])
		}
	}
}
