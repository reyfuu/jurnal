package finance_test

import (
	"slices"
	"strings"
	"testing"

	"pajak/internal/finance"
)

// BarangPenjualan: rincian barang transaksi Penjualan invoice. Diskon dibagi proporsional ke
// semua baris (sisa pembulatan ke baris terbesar), lalu hanya baris berbarang yang masuk rincian.
func TestInvoice_BarangPenjualan(t *testing.T) {
	inv := finance.Invoice{DiskonRp: 10_000, Items: []finance.InvoiceItem{
		{Deskripsi: "Kain", Qty: 3, HargaSatuan: 20_000, StokID: 1},
		{Deskripsi: "Kaos", Qty: 2, HargaSatuan: 50_000, StokID: 2},
		{Deskripsi: "Jahit", Qty: 1, HargaSatuan: 50_000, BukanBarang: true},
	}}
	rinci, jasa := inv.BarangPenjualan()
	want := []finance.RincianBarang{{StokID: 1, QtyStok: 3000, Nilai: 57_143}, {StokID: 2, QtyStok: 2000, Nilai: 95_237}}
	if jasa || !slices.Equal(rinci, want) {
		t.Fatalf("rincian %+v bukan barang %v, harus %+v", rinci, jasa, want)
	}
	// Diskon melebihi subtotal: DPP 0, nilai 0 semua.
	inv.DiskonRp = 1_000_000
	if r, _ := inv.BarangPenjualan(); r[0].Nilai != 0 || r[1].Nilai != 0 {
		t.Errorf("diskon penuh: %+v", r)
	}
	// Semua jasa → bukan barang; tanpa tanda apa pun → tidak keduanya.
	semua := finance.Invoice{Items: []finance.InvoiceItem{{Qty: 1, HargaSatuan: 1, BukanBarang: true}}}
	if r, j := semua.BarangPenjualan(); len(r) != 0 || !j {
		t.Errorf("semua jasa: %+v %v", r, j)
	}
	polos := finance.Invoice{Items: []finance.InvoiceItem{{Qty: 1, HargaSatuan: 1}}}
	if r, j := polos.BarangPenjualan(); len(r) != 0 || j {
		t.Errorf("tanpa tanda: %+v %v", r, j)
	}
	// Nilai besar tidak meluap.
	besar := finance.Invoice{DiskonRp: 3_000_000_000_000, Items: []finance.InvoiceItem{
		{Qty: 1_000_000, HargaSatuan: 9_000_000_000, StokID: 1}, {Qty: 1, HargaSatuan: 1, StokID: 2}}}
	if r, _ := besar.BarangPenjualan(); r[0].Nilai+r[1].Nilai != besar.Totals().DPP || r[0].Nilai < 0 {
		t.Errorf("nilai besar: %+v, DPP %d", r, besar.Totals().DPP)
	}
}

func TestValidateInvoice_BarangDanJasaBersamaan(t *testing.T) {
	inv := finance.Invoice{PelangganNama: "PT", Tanggal: "2026-01-01", JatuhTempo: "2026-01-31",
		Items: []finance.InvoiceItem{{Deskripsi: "a", Qty: 1}, {Deskripsi: "b", Qty: 1, StokID: 1, BukanBarang: true}}}
	if err := finance.ValidateInvoice(&inv); err == nil || !strings.HasPrefix(err.Error(), "baris 2:") {
		t.Errorf("barang + bukan barang harus ditolak per baris: %v", err)
	}
	inv.Items[1].BukanBarang, inv.Items[1].StokID = false, -1
	if err := finance.ValidateInvoice(&inv); err == nil {
		t.Error("stok_id negatif harus ditolak")
	}
}
