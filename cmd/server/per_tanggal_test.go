package main

import (
	"net/http"
	"testing"

	"pajak/internal/finance"
)

// Checklist tester No 22: daftar aset dan nilai persediaan per tanggal bebas, sama dengan baris
// Neraca per tanggal yang sama (tengah bulan: penyusutan bulan itu belum dibukukan).
func TestAsetDanStokPerTanggal(t *testing.T) {
	srv := NewServer()
	c := login(t, srv, "owner@kedaikopisenja.id")
	for _, tgl := range []string{"2026-08-15", "2026-08-31", "2026-07-10"} {
		bs := getJSON[finance.BalanceSheet](t, srv, c, "/api/v1/reports/balance-sheet?tanggal="+tgl)
		as := getJSON[finance.AssetSummary](t, srv, c, "/api/v1/assets?pada="+tgl)
		if as.TotalNilaiBuku != bs.Assets.FixedAssets || as.TotalHargaPerolehan != bs.Assets.FixedAssetsCost {
			t.Fatalf("%s aset: nilai buku %d vs neraca %d, perolehan %d vs %d", tgl, as.TotalNilaiBuku, bs.Assets.FixedAssets, as.TotalHargaPerolehan, bs.Assets.FixedAssetsCost)
		}
		// usaha demo belum memulai kartu stok: per tanggal ditolak jelas, bukan semua barang "Habis"
		// (perhitungan per tanggal diuji di finance.TestKartuPada)
		if getJSON[finance.StokSummary](t, srv, c, "/api/v1/stok").Mulai == "" {
			if rec := do(srv, c, http.MethodGet, "/api/v1/stok?pada="+tgl, ""); rec.Code != http.StatusBadRequest {
				t.Fatalf("%s stok tanpa kartu: %d", tgl, rec.Code)
			}
		} else if st := getJSON[finance.StokSummary](t, srv, c, "/api/v1/stok?pada="+tgl); st.NilaiPersediaan != bs.Assets.Inventory {
			t.Fatalf("%s stok: %d vs neraca %d", tgl, st.NilaiPersediaan, bs.Assets.Inventory)
		}
	}
}

func TestPerTanggal_TanggalSalahDitolak(t *testing.T) {
	srv := NewServer()
	c := login(t, srv, "owner@kedaikopisenja.id")
	for _, u := range []string{"/api/v1/assets?pada=2026-13-01", "/api/v1/stok?pada=x"} {
		if rec := do(srv, c, http.MethodGet, u, ""); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d, mau 400", u, rec.Code)
		}
	}
}

// Aset yang dibeli sesudah tanggal yang dilihat (bulan yang sama) tidak ikut, termasuk lewat
// transaksi "Beli Alat / Aset"-nya: sama dengan Neraca per tanggal itu.
func TestAsetPerTanggal_PembelianSesudahnyaTidakIkut(t *testing.T) {
	srv := NewServer()
	c := login(t, srv, "owner@kedaikopisenja.id")
	buat[finance.Transaction](t, srv, c, "/api/v1/transactions", `{"tanggal":"2026-08-20","jenis":"Beli Alat / Aset","nominal":7000000,"lunas":true,"keterangan":"Mesin kopi baru"}`)
	bs := getJSON[finance.BalanceSheet](t, srv, c, "/api/v1/reports/balance-sheet?tanggal=2026-08-15")
	as := getJSON[finance.AssetSummary](t, srv, c, "/api/v1/assets?pada=2026-08-15")
	if as.TotalHargaPerolehan != bs.Assets.FixedAssetsCost || as.TotalNilaiBuku != bs.Assets.FixedAssets {
		t.Fatalf("perolehan %d vs %d, nilai buku %d vs %d", as.TotalHargaPerolehan, bs.Assets.FixedAssetsCost, as.TotalNilaiBuku, bs.Assets.FixedAssets)
	}
	for _, a := range as.Assets {
		if a.TanggalPerolehan > "2026-08-15" {
			t.Fatalf("aset sesudah tanggal ikut: %+v", a.Asset)
		}
	}
}
