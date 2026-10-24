package main

import (
	"net/http"
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
