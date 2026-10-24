package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"pajak/internal/finance"
)

// Barang di rincian transaksi harus ada di Stok & Gudang pada saat disimpan (di bawah kunci),
// di setiap jalur simpan: formulir, Catat cepat, persetujuan usulan koreksi, impor data awal.

const pesanBarangTidakAda = "barang yang dipilih tidak ada di Stok & Gudang"

func harusBarangTidakAda(t *testing.T, code int, body string) {
	t.Helper()
	if code != http.StatusBadRequest || strings.TrimSpace(body) != pesanBarangTidakAda {
		t.Fatalf("harus 400 %q, dapat %d %q", pesanBarangTidakAda, code, body)
	}
}

func siapKoreksiPerpetual(t *testing.T) (*Server, *http.Cookie, *http.Cookie, finance.StokItem) {
	t.Helper()
	srv, _, qa, k := siapKlien(t)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/business", manufakturPerpetual); rec.Code != http.StatusOK {
		t.Fatalf("profil perpetual: %d %s", rec.Code, rec.Body)
	}
	kain := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kain","kategori":"Bahan Baku","satuan":"m"}`)
	return srv, qa, k, kain
}

func usulBeliKain(id int) string {
	return `{"transaksi_id":1,"jenis":"ubah","alasan":"Beli kain 10 m","usulan":{"tanggal":"2026-10-13","jenis":"Beli Bahan / Stok","nominal":4500000,"keterangan":"Beli kain","lunas":true,"rincian":[{"stok_id":` + itoa(id) + `,"qty_stok":10000}]}}`
}

func harusKoreksiTidakDiterapkan(t *testing.T, srv *Server, qa *http.Cookie) {
	t.Helper()
	if got := txsOf(t, srv, qa); got[0].Jenis != "Perawatan & Perlengkapan" {
		t.Errorf("usulan tertolak tetap mengubah pembukuan: %+v", got[0])
	}
	if n := jumlahGerakAwal(t, srv, qa); n != 0 {
		t.Errorf("usulan tertolak tidak boleh memulai kartu stok (%d)", n)
	}
	var ks []koreksiRow
	_ = json.NewDecoder(do(srv, qa, http.MethodGet, "/api/v1/koreksi", "").Body).Decode(&ks)
	if len(ks) == 0 || ks[0].Status != finance.KoreksiMenunggu {
		t.Errorf("usulan tertolak harus tetap menunggu: %+v", ks)
	}
}

func TestBarangAda_UsulanKoreksiBarangHantuDitolak(t *testing.T) {
	srv, qa, k, _ := siapKoreksiPerpetual(t)
	rec := do(srv, k, http.MethodPost, "/api/v1/master/klien/2/koreksi", usulBeliKain(999))
	harusBarangTidakAda(t, rec.Code, rec.Body.String())
	var ks []koreksiRow
	_ = json.NewDecoder(do(srv, qa, http.MethodGet, "/api/v1/koreksi", "").Body).Decode(&ks)
	if len(ks) != 0 {
		t.Fatalf("usulan barang hantu tidak boleh tersimpan: %+v", ks)
	}
}

func TestBarangAda_SetujuiKoreksiBarangHantuDitolak(t *testing.T) {
	srv, qa, k, kain := siapKoreksiPerpetual(t)
	usul(t, srv, k, usulBeliKain(kain.ID))
	// Usulan lama (sebelum cek saat usul) yang menunjuk barang yang tidak pernah ada.
	tn := tenantByEmail(t, srv, qa)
	tn.mu.Lock()
	tn.koreksi[0].Usulan.Rincian[0].StokID = 999
	tn.mu.Unlock()
	rec := do(srv, qa, http.MethodPost, "/api/v1/koreksi/1/setujui", `{}`)
	harusBarangTidakAda(t, rec.Code, rec.Body.String())
	harusKoreksiTidakDiterapkan(t, srv, qa)
}

func TestBarangAda_SetujuiKoreksiBarangDihapusDitolak(t *testing.T) {
	srv, qa, k, kain := siapKoreksiPerpetual(t)
	usul(t, srv, k, usulBeliKain(kain.ID))
	if rec := do(srv, qa, http.MethodDelete, "/api/v1/stok/"+itoa(kain.ID), ""); rec.Code/100 != 2 {
		t.Fatalf("hapus barang: %d %s", rec.Code, rec.Body)
	}
	rec := do(srv, qa, http.MethodPost, "/api/v1/koreksi/1/setujui", `{}`)
	harusBarangTidakAda(t, rec.Code, rec.Body.String())
	harusKoreksiTidakDiterapkan(t, srv, qa)
}

// Formulir: validStok berjalan sebelum kunci; barang yang dihapus di antaranya ditolak saat
// simpan di bawah kunci.
func TestBarangAda_FormulirDicekDiBawahKunci(t *testing.T) {
	srv, qa, kain := perpetualKartuMulai(t, "2026-01-01")
	tn := tenantByEmail(t, srv, qa)
	tx := finance.Transaction{Tanggal: "2026-02-01", Jenis: finance.JenisBeliStok, Nominal: 100000, Lunas: true,
		Rincian: []finance.RincianBarang{{StokID: kain.ID + 100, QtyStok: 1000}}}
	tn.mu.Lock()
	_, kode, err := tn.cekSimpanTxLocked(tx, -1)
	tn.mu.Unlock()
	if kode != http.StatusBadRequest || err == nil || err.Error() != pesanBarangTidakAda {
		t.Fatalf("formulir barang hantu: %d %v", kode, err)
	}
}

func TestBarangAda_CatatDicekDiBawahKunci(t *testing.T) {
	srv, qa, kain := perpetualKartuMulai(t, "2026-01-01")
	tn := tenantByEmail(t, srv, qa)
	tx := finance.Transaction{Tanggal: "2026-02-01", Jenis: finance.JenisBeliStok, Nominal: 100000, Lunas: true, Grup: 77,
		Rincian: []finance.RincianBarang{{StokID: kain.ID + 100, QtyStok: 1000}}}
	tn.mu.Lock()
	n := len(tn.txs)
	_, kode, err := tn.simpanCatatLocked(77, []finance.Transaction{tx}, "uji")
	sesudah := len(tn.txs)
	tn.mu.Unlock()
	if kode != http.StatusBadRequest || err == nil || err.Error() != pesanBarangTidakAda {
		t.Fatalf("catat barang hantu: %d %v", kode, err)
	}
	if sesudah != n {
		t.Errorf("catat tertolak tetap menyimpan transaksi")
	}
}

// --- Impor Data Awal ---

func imporDataAwal(t *testing.T, srv *Server, c *http.Cookie, commit bool, sheets ...finance.ImportSheet) (int, onboardingResp, string) {
	t.Helper()
	body, ctype := workbookUpload(t, sheets)
	path := "/api/v1/onboarding/import"
	if commit {
		path += "?commit=1"
	}
	rec := upload(srv, c, path, body, ctype)
	raw := rec.Body.String()
	var res onboardingResp
	_ = json.Unmarshal([]byte(raw), &res)
	return rec.Code, res, raw
}

func sheetTransaksiUji(baris ...[]string) finance.ImportSheet {
	return finance.ImportSheet{Nama: "Transaksi", Rows: append([][]string{{"Tanggal", "Jenis", "Nominal", "Keterangan", "Barang", "jumlah_barang"}}, baris...)}
}

func TestImporDataAwal_PerpetualTanpaBarangGalatPerBaris(t *testing.T) {
	srv, qa, _ := perpetualKartuMulai(t, "2026-01-01")
	n := len(txsOf(t, srv, qa))
	sh := sheetTransaksiUji([]string{"2026-02-01", "Penjualan", "100000", "tanpa barang", "", ""})
	code, res, raw := imporDataAwal(t, srv, qa, false, sh)
	if code != http.StatusOK || res.TransaksiCount != 0 || !adaError(res.Errors, "Transaksi", errWajibBarang.Error()) {
		t.Fatalf("pratinjau: %d %s", code, raw)
	}
	if res.Errors[0].Baris != 2 {
		t.Errorf("galat harus menunjuk baris 2: %+v", res.Errors)
	}
	code, res, raw = imporDataAwal(t, srv, qa, true, sh)
	if code != http.StatusOK || !res.Imported {
		t.Fatalf("commit: %d %s", code, raw)
	}
	if got := len(txsOf(t, srv, qa)); got != n {
		t.Errorf("baris tanpa barang tidak boleh tersimpan: %d → %d", n, got)
	}
}

func TestImporDataAwal_PerpetualBerbarangTersimpanDenganRincian(t *testing.T) {
	srv, qa, kain := perpetualKartuMulai(t, "2026-01-01")
	sh := sheetTransaksiUji(
		[]string{"2026-02-01", "Penjualan", "100000", "jual kain", "Kain", "2"},
		[]string{"2025-12-20", "Penjualan", "50000", "sebelum kartu", "", ""},
	)
	code, res, raw := imporDataAwal(t, srv, qa, true, sh)
	if code != http.StatusOK || !res.Imported || res.TransaksiCount != 2 || len(res.Errors) != 0 {
		t.Fatalf("commit: %d %s", code, raw)
	}
	var jual *finance.Transaction
	for _, tx := range txsOf(t, srv, qa) {
		if tx.Keterangan == "jual kain" {
			jual = &tx
		}
	}
	if jual == nil {
		t.Fatal("transaksi berbarang tidak tersimpan")
	}
	if b := jual.BarisBarang(); len(b) != 1 || b[0].StokID != kain.ID || b[0].QtyStok != 2000 {
		t.Fatalf("rincian barang: %+v", jual.Rincian)
	}
}

func TestImporDataAwal_PerpetualStokMinusDitolak(t *testing.T) {
	srv, qa, _ := perpetualKartuMulai(t, "2026-01-01") // stok awal 10 m
	n := len(txsOf(t, srv, qa))
	sh := sheetTransaksiUji([]string{"2026-02-01", "Penjualan", "100000", "jual kain", "Kain", "11"})
	code, res, raw := imporDataAwal(t, srv, qa, false, sh)
	if code != http.StatusOK || res.BisaDisimpan || !adaError(res.Errors, "Transaksi", "") {
		t.Fatalf("pratinjau stok minus: %d %s", code, raw)
	}
	if code, _, raw = imporDataAwal(t, srv, qa, true, sh); code/100 == 2 {
		t.Fatalf("commit stok minus harus ditolak: %d %s", code, raw)
	}
	if got := len(txsOf(t, srv, qa)); got != n {
		t.Errorf("impor tertolak tetap menyimpan transaksi: %d → %d", n, got)
	}
}

// Kartu belum dimulai: transaksi berbarang pertama hasil impor data awal memulainya.
func TestImporDataAwal_PerpetualMemulaiKartu(t *testing.T) {
	srv, qa := usahaManufakturPerpetual(t)
	buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kain","kategori":"Bahan Baku","satuan":"m","jumlah":5,"harga_pokok":10000}`)
	sh := sheetTransaksiUji([]string{"2026-02-01", "Beli Bahan / Stok", "100000", "beli kain", "Kain", "3"})
	code, res, raw := imporDataAwal(t, srv, qa, true, sh)
	if code != http.StatusOK || !res.Imported || res.TransaksiCount != 1 {
		t.Fatalf("commit: %d %s", code, raw)
	}
	if n := jumlahGerakAwal(t, srv, qa); n != 1 {
		t.Errorf("impor berbarang pertama harus memulai kartu stok (%d)", n)
	}
}

// Sheet Stok di berkas yang sama menggantikan daftar barang: barang transaksi dicocokkan ke
// daftar baru itu.
func TestImporDataAwal_BarangDariSheetStokBerkasYangSama(t *testing.T) {
	srv, qa := usahaManufakturPerpetual(t)
	stok := finance.ImportSheet{Nama: "Stok", Rows: [][]string{
		{"SKU", "Nama Barang", "Kategori", "Jumlah", "Satuan", "Harga Pokok"},
		{"BNG-1", "Benang", "Bahan Baku", "20", "gulung", "5000"},
	}}
	sh := sheetTransaksiUji([]string{"2026-02-01", "Penjualan", "100000", "jual benang", "BNG-1", "4"})
	code, res, raw := imporDataAwal(t, srv, qa, true, stok, sh)
	if code != http.StatusOK || !res.Imported || res.TransaksiCount != 1 || len(res.Errors) != 0 {
		t.Fatalf("commit: %d %s", code, raw)
	}
	tn := tenantByEmail(t, srv, qa)
	tn.mu.RLock()
	defer tn.mu.RUnlock()
	tx := tn.txs[len(tn.txs)-1]
	if b := tx.BarisBarang(); len(b) != 1 || len(tn.stok) != 1 || b[0].StokID != tn.stok[0].ID || b[0].QtyStok != 4000 {
		t.Fatalf("rincian harus menunjuk barang sheet Stok: %+v, stok %+v", tx.Rincian, tn.stok)
	}
}

// Periodik: kolom barang opsional dan tidak dicek ketat, sama dengan impor transaksi.
func TestImporDataAwal_PeriodikTidakBerubah(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	n := len(txsOf(t, srv, owner))
	sh := sheetTransaksiUji(
		[]string{"2026-08-29", "Penjualan", "3000000", "tanpa barang", "", ""},
		[]string{"2026-08-30", "Penjualan", "1000000", "barang tak dikenal", "Sepatu", "1"},
	)
	code, res, raw := imporDataAwal(t, srv, owner, true, sh)
	if code != http.StatusOK || !res.Imported || res.TransaksiCount != 2 || len(res.Errors) != 0 {
		t.Fatalf("commit periodik: %d %s", code, raw)
	}
	if got := len(txsOf(t, srv, owner)); got != n+2 {
		t.Errorf("transaksi periodik: %d → %d", n, got)
	}
}
