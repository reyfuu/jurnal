package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"pajak/internal/finance"
	"pajak/internal/xlsx"
)

func buat[T any](t *testing.T, srv *Server, c *http.Cookie, path, body string) T {
	t.Helper()
	rec := do(srv, c, http.MethodPost, path, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST %s: %d %s", path, rec.Code, rec.Body)
	}
	var v T
	_ = json.NewDecoder(rec.Body).Decode(&v)
	return v
}

func TestKartuStok_AlurLengkap(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	qa := register(t, srv, registerBody)
	aktifkan(t, srv, qa)

	kopi := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"sku":"BHN-KOPI","nama":"Biji kopi","kategori":"Bahan Baku","satuan":"kg","harga_pokok":200000}`)
	produk := buat[finance.Produk](t, srv, qa, "/api/v1/produksi/produk",
		`{"sku":"KS-1","nama":"Kopi Susu","satuan":"cup","hasil_per_batch":50,"harga_jual":18000,"bahan":[{"stok_id":`+itoa(kopi.ID)+`,"nama":"Biji kopi","satuan":"kg","takaran":900}]}`)

	// Sebelum titik nol: opname & produksi ditolak.
	if rec := do(srv, qa, http.MethodPost, "/api/v1/stok/opname", `{"tanggal":"2026-01-05","baris":[{"stok_id":1,"fisik":1000}]}`); rec.Code != http.StatusBadRequest {
		t.Errorf("opname sebelum titik nol: %d", rec.Code)
	}
	buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/awal", `{"tanggal":"2026-01-01","baris":[{"stok_id":`+itoa(kopi.ID)+`,"fisik":10000,"harga_unit":200000}]}`)
	if rec := do(srv, qa, http.MethodPost, "/api/v1/stok/awal", `{"tanggal":"2026-01-02","baris":[{"stok_id":1,"fisik":1}]}`); rec.Code != http.StatusConflict {
		t.Errorf("stok awal kedua: %d", rec.Code)
	}

	// Beli 10 kg lewat transaksi biasa; barang tidak dikenal ditolak; jenis lain tidak boleh memilih barang.
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-01-05","jenis":"Beli Bahan / Stok","nominal":2500000,"lunas":true,"stok_id":99,"qty_stok":1000}`); rec.Code != http.StatusBadRequest {
		t.Errorf("barang tidak dikenal: %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-01-05","jenis":"Gaji & Upah","nominal":1,"lunas":true,"stok_id":1,"qty_stok":1000}`); rec.Code != http.StatusBadRequest {
		t.Errorf("gaji dengan barang: %d", rec.Code)
	}
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-01-05","jenis":"Beli Bahan / Stok","nominal":2500000,"lunas":true,"stok_id":`+itoa(kopi.ID)+`,"qty_stok":10000}`)

	// Produksi 2 batch: 1,8 kg kopi → 100 cup; barang jadi dibuat otomatis.
	buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/produksi", `{"tanggal":"2026-01-06","produk_id":`+itoa(produk.ID)+`,"batch":2}`)
	ringkas := getJSON[finance.StokSummary](t, srv, qa, "/api/v1/stok")
	var cup finance.StokRow
	for _, r := range ringkas.Items {
		if r.SKU == "KS-1" {
			cup = r
		}
	}
	if !cup.Dilacak || cup.JumlahMili != 100_000 || cup.Kategori != "Produk Jadi" {
		t.Fatalf("barang jadi %+v", cup)
	}

	// Jual 30 cup lewat transaksi Penjualan.
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-01-07","jenis":"Penjualan","nominal":540000,"lunas":true,"channel":"Toko","stok_id":`+itoa(cup.ID)+`,"qty_stok":30000}`)
	// Opname kopi: sistem 18,2 kg, fisik 18 kg.
	buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/opname", `{"tanggal":"2026-01-31","baris":[{"stok_id":`+itoa(kopi.ID)+`,"fisik":18000}]}`)

	l := getJSON[finance.LaporanMutasi](t, srv, qa, "/api/v1/stok/mutasi?dari=2026-01-01&sampai=2026-01-31")
	var mk, mc finance.MutasiItem
	for _, m := range l.Items {
		switch m.StokID {
		case kopi.ID:
			mk = m
		case cup.ID:
			mc = m
		}
	}
	if mk.AwalQty != 10_000 || mk.BeliQty != 10_000 || mk.PakaiQty != 1_800 || mk.OpnameQty != -200 || mk.AkhirQty != 18_000 {
		t.Errorf("mutasi kopi %+v", mk)
	}
	// 1,8 kg @225rb = 405rb untuk 100 cup → 30 cup = 121.500
	if mc.ProduksiQty != 100_000 || mc.JualQty != 30_000 || mc.JualNilai != 121_500 || l.HPPJual != 121_500 {
		t.Errorf("mutasi cup %+v hpp %d", mc, l.HPPJual)
	}

	lp := getJSON[laporanProduksi](t, srv, qa, "/api/v1/produksi/laporan?dari=2026-01-01&sampai=2026-01-31")
	if len(lp.Baris) != 1 || lp.Baris[0].Produk != "Kopi Susu" || lp.Baris[0].Batch != 2 || lp.Baris[0].HasilQty != 100_000 ||
		lp.Baris[0].Nilai != 405_000 || lp.Baris[0].PerUnit != 4_050 || len(lp.Baris[0].Bahan) != 1 || lp.Baris[0].Bahan[0].Qty != 1_800 || lp.Total != 405_000 {
		t.Errorf("laporan produksi %+v", lp)
	}
	if rec := do(srv, qa, http.MethodGet, "/api/v1/produksi/laporan.xlsx?dari=2026-01-01&sampai=2026-01-31", ""); rec.Code != http.StatusOK {
		t.Errorf("ekspor produksi %d", rec.Code)
	}

	kartu := getJSON[kartuBarangResp](t, srv, qa, "/api/v1/stok/kartu/"+itoa(kopi.ID)+"?dari=2026-01-02&sampai=2026-01-31")
	if kartu.Awal.Qty != 10_000 || len(kartu.Baris) != 3 {
		t.Errorf("kartu kopi awal %+v baris %d", kartu.Awal, len(kartu.Baris))
	}

	// Barang yang sudah punya riwayat tidak bisa dihapus; stok awal tidak bisa dihapus selagi ada opname.
	if rec := do(srv, qa, http.MethodDelete, "/api/v1/stok/"+itoa(kopi.ID), ""); rec.Code != http.StatusConflict {
		t.Errorf("hapus barang berriwayat: %d", rec.Code)
	}
	gerak := getJSON[[]finance.GerakStok](t, srv, qa, "/api/v1/stok/gerak")
	if len(gerak) != 3 || gerak[0].Jenis != finance.GerakOpname {
		t.Fatalf("gerak %+v", gerak)
	}
	if rec := do(srv, qa, http.MethodDelete, "/api/v1/stok/gerak/1", ""); rec.Code != http.StatusConflict {
		t.Errorf("hapus stok awal: %d", rec.Code)
	}

	if rec := do(srv, qa, http.MethodPost, "/api/v1/users", `{"name":"Kasir","email":"kasir@qa.id","role":"kasir","active":true,"password":"rahasia123"}`); rec.Code != http.StatusCreated {
		t.Fatalf("buat kasir: %d %s", rec.Code, rec.Body)
	}
	kasir := loginPass(t, srv, "kasir@qa.id", "rahasia123")
	if rec := do(srv, kasir, http.MethodGet, "/api/v1/stok/pilihan", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Biji kopi") {
		t.Errorf("kasir pilihan barang: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, kasir, http.MethodGet, "/api/v1/stok/mutasi", ""); rec.Code != http.StatusForbidden {
		t.Errorf("kasir laporan mutasi: %d", rec.Code)
	}

	// Produksi aktual: kopi terpakai 0,5 kg (resep 0,9), hasil 48 cup (standar 50).
	buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/produksi", `{"tanggal":"2026-01-20","produk_id":`+itoa(produk.ID)+`,"batch":1,"hasil":48000,"keterangan":"ada yang tumpah",
		"bahan":[{"stok_id":`+itoa(kopi.ID)+`,"qty":500}]}`)
	lp2 := getJSON[laporanProduksi](t, srv, qa, "/api/v1/produksi/laporan?dari=2026-01-20&sampai=2026-01-20")
	if len(lp2.Baris) != 1 || lp2.Baris[0].HasilQty != 48_000 || lp2.Baris[0].HasilStd != 50_000 ||
		lp2.Baris[0].Bahan[0].Qty != 500 || lp2.Baris[0].Bahan[0].Standar != 900 || lp2.Baris[0].Keterangan != "ada yang tumpah" {
		t.Errorf("produksi aktual %+v", lp2.Baris)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/stok/produksi", `{"tanggal":"2026-01-20","produk_id":`+itoa(produk.ID)+`,"batch":1,"bahan":[{"stok_id":99,"qty":1}]}`); rec.Code != http.StatusBadRequest {
		t.Errorf("bahan tak dikenal: %d", rec.Code)
	}

	rec := do(srv, qa, http.MethodGet, "/api/v1/stok/mutasi.xlsx?dari=2026-01-01&sampai=2026-01-31", "")
	rows, err := xlsx.Read(rec.Body.Bytes())
	if rec.Code != http.StatusOK || err != nil || !strings.Contains(strings.Join(rows[4], "|"), "Biji kopi") {
		t.Errorf("ekspor: %d %v %v", rec.Code, err, rows)
	}
}
