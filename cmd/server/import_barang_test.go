package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"pajak/internal/finance"
)

var kepalaImpor = []string{"Tanggal", "Jenis", "Nominal", "Keterangan", "Barang", "Jumlah Barang"}

func imporXlsx(t *testing.T, srv *Server, qa *http.Cookie, commit bool, rows ...[]string) (int, importResp, string) {
	t.Helper()
	body, ctype := xlsxUpload(t, append([][]string{kepalaImpor}, rows...))
	url := "/api/v1/transactions/import"
	if commit {
		url += "?commit=1"
	}
	rec := upload(srv, qa, url, body, ctype)
	var res importResp
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	return rec.Code, res, strings.TrimSpace(rec.Body.String())
}

func TestImporTransaksi_BarangDariSKUDanNama(t *testing.T) {
	srv, qa, kaos, kain := perpetualKaos(t, 10000, 0)
	_ = do(srv, qa, http.MethodPut, "/api/v1/stok/"+itoa(kaos.ID), `{"sku":"KS-01","nama":"Kaos","kategori":"Produk Jadi","satuan":"pcs"}`)
	code, res, body := imporXlsx(t, srv, qa, true,
		[]string{"2026-01-10", "Penjualan", "100000", "per SKU", "ks-01", "2,5"},
		[]string{"2026-01-11", "Beli Bahan / Stok", "50000", "per nama", "  KAIN ", "3.5"},
	)
	if code != http.StatusOK || res.Imported != 2 || len(res.Errors) != 0 {
		t.Fatalf("impor: %d %s", code, body)
	}
	got := txsOf(t, srv, qa)
	if len(got) != 2 {
		t.Fatalf("tersimpan %d", len(got))
	}
	if b := got[0].BarisBarang(); len(b) != 1 || b[0].StokID != kaos.ID || b[0].QtyStok != 2500 {
		t.Errorf("baris 1: %+v", got[0])
	}
	if b := got[1].BarisBarang(); len(b) != 1 || b[0].StokID != kain.ID || b[0].QtyStok != 3500 {
		t.Errorf("baris 2: %+v", got[1])
	}
	_, res, _ = imporXlsx(t, srv, qa, false, []string{"2026-01-12", "Penjualan", "1000", "", "Sepatu", "1"})
	if res.Valid != 0 || len(res.Errors) != 1 || res.Errors[0].Baris != 2 || res.Errors[0].Pesan != `barang "Sepatu" tidak ada di daftar barang` {
		t.Fatalf("barang tak dikenal: %+v", res)
	}
}

func TestImporTransaksi_PerpetualTanpaBarangGalatPerBaris(t *testing.T) {
	srv, qa, kaos, _ := perpetualKaos(t, 10000, 0)
	code, res, body := imporXlsx(t, srv, qa, true,
		[]string{"2026-01-10", "Penjualan", "100000", "tanpa barang", "", ""},
		[]string{"2026-01-11", "Penjualan", "100000", "berbarang", "Kaos", "1"},
		[]string{"2026-01-12", "Gaji & Upah", "5000", "bukan jual", "", ""},
	)
	if code != http.StatusOK || res.Imported != 2 || len(res.Errors) != 1 || res.Errors[0].Baris != 2 || !strings.Contains(res.Errors[0].Pesan, errWajibBarang.Error()) {
		t.Fatalf("hasil: %d %s", code, body)
	}
	for _, tx := range txsOf(t, srv, qa) {
		if tx.Jenis == "Penjualan" && tx.BarisBarang()[0].StokID != kaos.ID {
			t.Errorf("tx salah: %+v", tx)
		}
	}
}

func TestImporTransaksi_JasaBoleh(t *testing.T) {
	srv, qa, _, _ := perpetualKaos(t, 10000, 0)
	code, res, body := imporXlsx(t, srv, qa, true, []string{"2026-01-10", "Penjualan", "100000", "servis", "Jasa", ""})
	if code != http.StatusOK || res.Imported != 1 || len(res.Errors) != 0 {
		t.Fatalf("impor: %d %s", code, body)
	}
	if tx := txsOf(t, srv, qa)[0]; !tx.BukanBarang || len(tx.BarisBarang()) != 0 {
		t.Errorf("harus bukan barang: %+v", tx)
	}
}

func TestImporTransaksi_KolomLamaTetapJalan(t *testing.T) {
	srv, qa := usahaManufaktur(t)
	body, ctype := xlsxUpload(t, [][]string{
		{"Tanggal", "Jenis", "Nominal", "Keterangan"},
		{"2026-01-10", "Penjualan", "100000", "lama"},
	})
	rec := upload(srv, qa, "/api/v1/transactions/import?commit=1", body, ctype)
	var res importResp
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if rec.Code != http.StatusOK || res.Imported != 1 || len(res.Errors) != 0 || res.Galat != "" {
		t.Fatalf("impor lama: %d %s", rec.Code, rec.Body)
	}
}

func TestImporTransaksi_MelebihiStokDitolak(t *testing.T) {
	srv, qa, _, _ := perpetualKaos(t, 5000, 0)
	rows := [][]string{
		{"2026-01-10", "Penjualan", "100000", "", "Kaos", "3"},
		{"2026-01-11", "Penjualan", "100000", "", "Kaos", "3"},
	}
	_, res, _ := imporXlsx(t, srv, qa, false, rows...)
	if !strings.Contains(res.Galat, "stok Kaos tidak cukup pada 11 Jan 2026") {
		t.Fatalf("pratinjau harus memberi tahu: %+v", res)
	}
	code, _, body := imporXlsx(t, srv, qa, true, rows...)
	cekMinus(t, code, body, "stok Kaos tidak cukup pada 11 Jan 2026: tersedia 2 pcs, dibutuhkan 3 pcs")
	if n := len(txsOf(t, srv, qa)); n != 0 {
		t.Fatalf("impor tertolak tetap menyimpan %d transaksi", n)
	}
}

func TestImporTransaksi_KartuDimulaiOtomatis(t *testing.T) {
	srv, qa := usahaManufakturPerpetual(t)
	buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kaos","kategori":"Produk Jadi","satuan":"pcs","jumlah":10,"harga_pokok":50000}`)
	code, res, body := imporXlsx(t, srv, qa, true,
		[]string{"2026-02-10", "Penjualan", "100000", "", "Kaos", "4"},
		[]string{"2026-02-05", "Penjualan", "100000", "", "Kaos", "5"},
	)
	if code != http.StatusOK || res.Imported != 2 {
		t.Fatalf("impor: %d %s", code, body)
	}
	gerak := getJSON[[]finance.GerakStok](t, srv, qa, "/api/v1/stok/gerak")
	if len(gerak) != 1 || gerak[0].Jenis != finance.GerakAwal || gerak[0].Tanggal != "2026-02-05" {
		t.Errorf("kartu harus mulai otomatis di tanggal impor paling awal: %+v", gerak)
	}
}
