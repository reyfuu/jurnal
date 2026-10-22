package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"pajak/internal/coa"
)

// Daftar Akun: klien menambah akun sendiri; jenisnya mengikuti induk, kodenya kode kosong
// berikutnya, ikut di CoA, bertahan saat profil disimpan ulang, dan baru bisa dihapus bila
// tidak dipakai transaksi.
func TestAkunTambahan(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	do(srv, owner, http.MethodPut, "/api/v1/business", dagangCV)

	akun := func() []coa.Account {
		var c struct {
			Accounts []coa.Account `json:"accounts"`
		}
		_ = json.NewDecoder(do(srv, owner, http.MethodGet, "/api/v1/coa", "").Body).Decode(&c)
		return c.Accounts
	}
	induk, ok := coa.Find(akun(), "1101")
	if !ok {
		t.Fatal("akun 1101 tidak ada")
	}
	rec := do(srv, owner, http.MethodPost, "/api/v1/coa/akun", `{"nama":"Kas Cabang Bandung","induk":"1101"}`)
	var baru struct{ Kode, Nama, Induk string }
	_ = json.NewDecoder(rec.Body).Decode(&baru)
	if rec.Code != http.StatusOK || baru.Kode == "" || baru.Kode[:2] != "11" {
		t.Fatalf("tambah akun: %d %+v", rec.Code, baru)
	}
	a, ok := coa.Find(akun(), baru.Kode)
	if !ok || a.Nama != "Kas Cabang Bandung" || a.Kelompok != induk.Kelompok || a.Normal != induk.Normal || a.Subtype != induk.Subtype || a.Peran != "" {
		t.Fatalf("akun baru di CoA: %+v (induk %+v)", a, induk)
	}
	if rec := do(srv, owner, http.MethodPost, "/api/v1/coa/akun", `{"nama":"kas cabang bandung","induk":"1101"}`); rec.Code != http.StatusConflict {
		t.Errorf("nama ganda harus ditolak: %d", rec.Code)
	}
	if rec := do(srv, owner, http.MethodPost, "/api/v1/coa/akun", `{"nama":"X","induk":"9999"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("induk tidak dikenal harus ditolak: %d", rec.Code)
	}
	// simpan profil lagi (formulir tidak mengirim akun_tambahan): akun tetap ada
	do(srv, owner, http.MethodPut, "/api/v1/business", dagangCV)
	if _, ok := coa.Find(akun(), baru.Kode); !ok {
		t.Fatal("akun tambahan hilang setelah profil disimpan ulang")
	}
	// dipakai transaksi → tidak bisa dihapus
	do(srv, owner, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-08-03","jenis":"Penjualan","nominal":1000,"lunas":true,"akun":"`+baru.Kode+`"}`)
	if rec := do(srv, owner, http.MethodDelete, "/api/v1/coa/akun/"+baru.Kode, ""); rec.Code != http.StatusConflict {
		t.Errorf("akun terpakai harus ditolak dihapus: %d", rec.Code)
	}
	// akun template tidak bisa dihapus lewat endpoint ini
	if rec := do(srv, owner, http.MethodDelete, "/api/v1/coa/akun/1101", ""); rec.Code != http.StatusNotFound {
		t.Errorf("akun template: %d", rec.Code)
	}
	rec = do(srv, owner, http.MethodPost, "/api/v1/coa/akun", `{"nama":"Kas Kecil Gudang","induk":"1101"}`)
	var kedua struct{ Kode string }
	_ = json.NewDecoder(rec.Body).Decode(&kedua)
	if rec := do(srv, owner, http.MethodDelete, "/api/v1/coa/akun/"+kedua.Kode, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("hapus akun tidak terpakai: %d", rec.Code)
	}
	if _, ok := coa.Find(akun(), kedua.Kode); ok {
		t.Error("akun yang dihapus masih ada")
	}
}
