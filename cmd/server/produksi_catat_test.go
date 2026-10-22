package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"pajak/internal/finance"
)

// Produksi pertama tanpa kartu stok: daftar barang jadi stok awal otomatis; biaya lain dibayar
// dari kas dan ikut jadi nilai produk jadi tanpa terhitung dua kali di laba rugi.
func TestCatatProduksi_LangsungTanpaKartuStok(t *testing.T) {
	srv, qa := usahaBaru(t)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/opening", `{"tanggal":"2026-01-01","rincian":[{"nama":"Kas","kategori":"kas","nominal":5000000},{"nama":"Persediaan","kategori":"persediaan","nominal":600000},{"nama":"Modal Pemilik","kategori":"modal","nominal":5600000}]}`); rec.Code != http.StatusOK {
		t.Fatalf("saldo awal: %d %s", rec.Code, rec.Body)
	}
	singkong := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Singkong","kategori":"Bahan Baku","satuan":"kg","jumlah":100,"harga_pokok":6000}`)
	keripik := buat[finance.Produk](t, srv, qa, "/api/v1/produksi/produk", `{"nama":"Keripik","satuan":"pcs","hasil_per_batch":40,"harga_jual":15000,"bahan":[{"stok_id":`+itoa(singkong.ID)+`,"takaran":10000}]}`)

	body := `{"tanggal":"2026-01-10","produk":[{"produk_id":` + itoa(keripik.ID) + `,"hasil":40000,
		"bahan":[{"stok_id":` + itoa(singkong.ID) + `,"qty":10000}],
		"biaya":[{"keterangan":"Tenaga harian","nominal":100000},{"keterangan":"Gas","nominal":20000}]}]}`
	rec := do(srv, qa, http.MethodPost, "/api/v1/stok/produksi/catat", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("catat produksi: %d %s", rec.Code, rec.Body)
	}
	var out produksiCatatResp
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if !out.StokAwal || len(out.Gerak) != 1 || out.Gerak[0].Biaya != 120_000 || out.Transaksi != 2 {
		t.Fatalf("hasil: %+v", out)
	}
	// Nilai 40 pcs keripik = 10 kg singkong @6.000 + biaya lain 120.000 = 180.000 (4.500/pcs).
	stok := getJSON[finance.StokSummary](t, srv, qa, "/api/v1/stok")
	var nilai int64
	for _, it := range stok.Items {
		if it.Nama == "Keripik" {
			nilai = it.Nilai
		}
	}
	if nilai != 180_000 {
		t.Fatalf("nilai keripik %d, mau 180000", nilai)
	}
	is := getJSON[finance.IncomeStatement](t, srv, qa, "/api/v1/reports/financial?period=2026-01")
	bs := getJSON[finance.BalanceSheet](t, srv, qa, "/api/v1/reports/balance-sheet?period=2026-01")
	if is.COGS != 0 || is.NetProfit != 0 || !bs.IsBalanced {
		t.Fatalf("belum ada penjualan: HPP %d laba %d seimbang %v", is.COGS, is.NetProfit, bs.IsBalanced)
	}
	// Hapus dokumen produksi: transaksi biaya lainnya ikut terhapus.
	if rec := do(srv, qa, http.MethodDelete, "/api/v1/stok/gerak/"+itoa(out.Gerak[0].ID), ""); rec.Code != http.StatusNoContent {
		t.Fatalf("hapus produksi: %d %s", rec.Code, rec.Body)
	}
	if is := getJSON[finance.IncomeStatement](t, srv, qa, "/api/v1/reports/financial?period=2026-01"); is.COGS != 0 || is.TotalOpex != 0 {
		t.Fatalf("biaya lain harus ikut terhapus: %+v", is)
	}
}
