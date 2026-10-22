package main

import (
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
		st := getJSON[finance.StokSummary](t, srv, c, "/api/v1/stok?pada="+tgl)
		if st.Mulai != "" && st.NilaiPersediaan != bs.Assets.Inventory {
			t.Fatalf("%s stok: %d vs neraca %d", tgl, st.NilaiPersediaan, bs.Assets.Inventory)
		}
	}
}
