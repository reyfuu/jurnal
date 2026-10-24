package main

import (
	"encoding/json"
	"net/http"
	"strings"
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

// siapProduksiKaos: manufaktur perpetual dengan Kain 100 m @10.000 dan produk Kaos (10 pcs per batch, 1 m kain per pcs).
func siapProduksiKaos(t *testing.T) (*Server, *http.Cookie, finance.StokItem, finance.Produk) {
	t.Helper()
	return siapProduksiKaosDengan(t, usahaManufakturPerpetual)
}

func siapProduksiKaosDengan(t *testing.T, usaha func(*testing.T) (*Server, *http.Cookie)) (*Server, *http.Cookie, finance.StokItem, finance.Produk) {
	t.Helper()
	srv, qa := usaha(t)
	kain := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kain","kategori":"Bahan Baku","satuan":"m"}`)
	kaos := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kaos","kategori":"Produk Jadi","satuan":"pcs"}`)
	buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/awal", `{"tanggal":"2026-01-01","baris":[`+
		`{"stok_id":`+itoa(kain.ID)+`,"fisik":100000,"harga_unit":10000},`+
		`{"stok_id":`+itoa(kaos.ID)+`,"fisik":0,"harga_unit":0}]}`)
	pKaos := buat[finance.Produk](t, srv, qa, "/api/v1/produksi/produk", `{"nama":"Kaos","satuan":"pcs","hasil_per_batch":10,"bahan":[{"stok_id":`+itoa(kain.ID)+`,"takaran":10000}]}`)
	return srv, qa, kain, pKaos
}

func txBiayaProduksi(t *testing.T, srv *Server, qa *http.Cookie) []finance.Transaction {
	t.Helper()
	var out []finance.Transaction
	for _, x := range getJSON[txListResp](t, srv, qa, "/api/v1/transactions").Transactions {
		if strings.HasPrefix(x.Keterangan, "Biaya produksi") {
			out = append(out, x)
		}
	}
	return out
}

// Perpetual: biaya lain tanpa akun masuk ke akun persediaan barang hasil dan menambah nilainya.
func TestCatatProduksi_BiayaKeAkunHasil(t *testing.T) {
	srv, qa, kain, pKaos := siapProduksiKaos(t)
	body := `{"tanggal":"2026-01-05","produk":[{"produk_id":` + itoa(pKaos.ID) + `,"hasil":10000,"bahan":[{"stok_id":` + itoa(kain.ID) + `,"qty":10000}],` +
		`"biaya":[{"keterangan":"Upah jahit","nominal":20000}]}]}`
	rec := do(srv, qa, http.MethodPost, "/api/v1/stok/produksi/catat", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("catat produksi: %d %s", rec.Code, rec.Body)
	}
	var out produksiCatatResp
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Gerak) != 1 || out.Gerak[0].Biaya != 20_000 {
		t.Fatalf("g.Biaya harus 20000: %+v", out)
	}
	txs := txBiayaProduksi(t, srv, qa)
	if len(txs) != 1 || txs[0].Akun != "1305" {
		t.Fatalf("transaksi biaya harus berakun 1305: %+v", txs)
	}
}

// Akun beban pilihan pengguna tetap beban: tidak dikapitalisasi ke nilai barang hasil.
func TestCatatProduksi_BiayaAkunBebanTidakDikapitalisasi(t *testing.T) {
	srv, qa, kain, pKaos := siapProduksiKaos(t)
	body := `{"tanggal":"2026-01-05","produk":[{"produk_id":` + itoa(pKaos.ID) + `,"hasil":10000,"bahan":[{"stok_id":` + itoa(kain.ID) + `,"qty":10000}],` +
		`"biaya":[{"keterangan":"Upah jahit","akun":"5103","nominal":15000}]}]}`
	rec := do(srv, qa, http.MethodPost, "/api/v1/stok/produksi/catat", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("catat produksi: %d %s", rec.Code, rec.Body)
	}
	var out produksiCatatResp
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Gerak) != 1 || out.Gerak[0].Biaya != 0 || out.Transaksi != 1 {
		t.Fatalf("g.Biaya harus 0: %+v", out)
	}
	txs := txBiayaProduksi(t, srv, qa)
	if len(txs) != 1 || txs[0].Akun != "5103" {
		t.Fatalf("transaksi biaya harus berakun 5103: %+v", txs)
	}
	// Nilai Kaos = bahan saja: 10 m x 10.000.
	var nilai int64
	for _, it := range getJSON[finance.StokSummary](t, srv, qa, "/api/v1/stok").Items {
		if it.Nama == "Kaos" {
			nilai = it.Nilai
		}
	}
	if nilai != 100_000 {
		t.Fatalf("nilai Kaos %d, mau 100000 (bahan saja)", nilai)
	}
	// Transaksi beban tetap satu grup dengan dokumen produksinya: hapus dokumen ikut menghapus biaya.
	if out.Gerak[0].Grup == 0 || txs[0].Grup != out.Gerak[0].Grup {
		t.Fatalf("grup dokumen %d, grup transaksi %d", out.Gerak[0].Grup, txs[0].Grup)
	}
	if rec := do(srv, qa, http.MethodDelete, "/api/v1/stok/gerak/"+itoa(out.Gerak[0].ID), ""); rec.Code != http.StatusNoContent {
		t.Fatalf("hapus produksi: %d %s", rec.Code, rec.Body)
	}
	if len(txBiayaProduksi(t, srv, qa)) != 0 {
		t.Fatal("biaya beban harus ikut terhapus")
	}
}

// Periodik: akun beban pilihan pengguna juga tidak dikapitalisasi, jadi nilai kartu produk = bahan
// saja dan laba CoA tetap sama dengan laba ringkas.
func TestCatatProduksi_BiayaAkunBebanTidakDikapitalisasiPeriodik(t *testing.T) {
	srv, qa, kain, pKaos := siapProduksiKaosDengan(t, usahaManufaktur)
	body := `{"tanggal":"2026-01-05","produk":[{"produk_id":` + itoa(pKaos.ID) + `,"hasil":10000,"bahan":[{"stok_id":` + itoa(kain.ID) + `,"qty":10000}],` +
		`"biaya":[{"keterangan":"Upah jahit","akun":"5103","nominal":15000}]}]}`
	rec := do(srv, qa, http.MethodPost, "/api/v1/stok/produksi/catat", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("catat produksi: %d %s", rec.Code, rec.Body)
	}
	var out produksiCatatResp
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Gerak) != 1 || out.Gerak[0].Biaya != 0 {
		t.Fatalf("g.Biaya harus 0: %+v", out)
	}
	var nilai int64
	for _, it := range getJSON[finance.StokSummary](t, srv, qa, "/api/v1/stok").Items {
		if it.Nama == "Kaos" {
			nilai = it.Nilai
		}
	}
	if nilai != 100_000 {
		t.Fatalf("nilai Kaos %d, mau 100000 (bahan saja)", nilai)
	}
	c := getJSON[coaReport](t, srv, qa, "/api/v1/reports/coa?period=2026-01")
	is := getJSON[finance.IncomeStatement](t, srv, qa, "/api/v1/reports/financial?period=2026-01")
	if c.IncomeStatement == nil || c.IncomeStatement.LabaBersih != is.NetProfit {
		t.Fatalf("laba CoA harus = laba ringkas %d: %+v", is.NetProfit, c.IncomeStatement)
	}
	neracaSaldoSeimbang(t, c)
}
