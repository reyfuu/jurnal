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
	bank, _ := coa.Find(akun(), "1104") // akun terdekat di bawah 1105 dalam seri 11xx
	// nomor diketik sendiri; kelompok & saldo normal dari seri 11xx
	rec := do(srv, owner, http.MethodPost, "/api/v1/coa/akun", `{"kode":"1105","nama":"Bank BRI Operasional"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("tambah akun: %d %s", rec.Code, rec.Body)
	}
	a, ok := coa.Find(akun(), "1105")
	if !ok || a.Nama != "Bank BRI Operasional" || a.Kelompok != bank.Kelompok || a.Normal != bank.Normal || a.Subtype != bank.Subtype || a.Peran != "" {
		t.Fatalf("akun baru di CoA: %+v (bank %+v)", a, bank)
	}
	for _, c := range []struct {
		body string
		kode int
	}{
		{`{"kode":"1105","nama":"Lain"}`, http.StatusBadRequest},               // nomor sudah dipakai
		{`{"kode":"11A3","nama":"Lain"}`, http.StatusBadRequest},               // bukan 4 angka
		{`{"kode":"1109","nama":"bank bri operasional"}`, http.StatusConflict}, // nama ganda
		{`{"kode":"1701","nama":"Deposit Sewa"}`, http.StatusBadRequest},       // seri baru tanpa kelompok
	} {
		if rec := do(srv, owner, http.MethodPost, "/api/v1/coa/akun", c.body); rec.Code != c.kode {
			t.Errorf("%s: %d, mau %d (%s)", c.body, rec.Code, c.kode, rec.Body)
		}
	}
	// seri baru dengan kelompok pilihan
	if rec := do(srv, owner, http.MethodPost, "/api/v1/coa/akun", `{"kode":"1701","nama":"Deposit Sewa","kelompok":"`+string(bank.Kelompok)+`"}`); rec.Code != http.StatusOK {
		t.Fatalf("seri baru: %d %s", rec.Code, rec.Body)
	}
	if d, ok := coa.Find(akun(), "1701"); !ok || d.Kelompok != bank.Kelompok {
		t.Errorf("akun seri baru: %+v", d)
	}
	// ubah nama: boleh; nomor tetap
	if rec := do(srv, owner, http.MethodPut, "/api/v1/coa/akun/1105", `{"nama":"Bank BRI"}`); rec.Code != http.StatusOK {
		t.Fatalf("ubah nama: %d %s", rec.Code, rec.Body)
	}
	if a, _ := coa.Find(akun(), "1105"); a.Nama != "Bank BRI" {
		t.Errorf("nama baru: %q", a.Nama)
	}
	if rec := do(srv, owner, http.MethodPut, "/api/v1/coa/akun/1101", `{"nama":"Kas Besar"}`); rec.Code != http.StatusNotFound {
		t.Errorf("nama akun template tidak diubah di sini: %d", rec.Code)
	}
	// simpan profil lagi: akun tetap ada
	do(srv, owner, http.MethodPut, "/api/v1/business", dagangCV)
	if _, ok := coa.Find(akun(), "1105"); !ok {
		t.Fatal("akun tambahan hilang setelah profil disimpan ulang")
	}
	// dipakai transaksi → tidak bisa dihapus; yang belum dipakai bisa
	do(srv, owner, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-08-03","jenis":"Penjualan","nominal":1000,"lunas":true,"akun":"1105"}`)
	if rec := do(srv, owner, http.MethodDelete, "/api/v1/coa/akun/1105", ""); rec.Code != http.StatusConflict {
		t.Errorf("akun terpakai harus ditolak dihapus: %d", rec.Code)
	}
	if rec := do(srv, owner, http.MethodDelete, "/api/v1/coa/akun/1701", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("hapus akun tidak terpakai: %d", rec.Code)
	}
}

// Data master tidak boleh kembar: SKU atau nama barang / produk, nama supplier.
func TestMasterTidakKembar(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	if rec := do(srv, owner, http.MethodPost, "/api/v1/stok", `{"sku":"ZZ-1","nama":"Barang Uji","satuan":"pcs","jumlah":1}`); rec.Code != http.StatusCreated {
		t.Fatalf("barang pertama: %d %s", rec.Code, rec.Body)
	}
	for _, body := range []string{`{"sku":"zz-1","nama":"Lain","satuan":"pcs","jumlah":1}`, `{"sku":"ZZ-2","nama":"barang  uji","satuan":"pcs","jumlah":1}`} {
		if rec := do(srv, owner, http.MethodPost, "/api/v1/stok", body); rec.Code != http.StatusConflict {
			t.Errorf("barang kembar %s: %d", body, rec.Code)
		}
	}
	if rec := do(srv, owner, http.MethodPost, "/api/v1/produksi/supplier", `{"nama":"Supplier Uji"}`); rec.Code != http.StatusCreated {
		t.Fatalf("supplier: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, owner, http.MethodPost, "/api/v1/produksi/supplier", `{"nama":"supplier uji"}`); rec.Code != http.StatusConflict {
		t.Errorf("supplier kembar: %d", rec.Code)
	}
}
