package finance_test

import (
	"testing"

	"pajak/internal/finance"
)

func TestParseImport(t *testing.T) {
	rows := [][]string{
		{"Data transaksi Kedai Kopi Senja"}, // judul di atas header boleh
		{"No", "Tanggal", "Jenis Transaksi", "Nominal (Rp)", "Keterangan", "Produk / Bahan", "Jumlah", "Channel", "Pelanggan / Supplier", "Status Bayar", "Tanggal Lunas"},
		{"1", "2026-08-01", "Penjualan", "Rp 1.250.000", "Rekap kasir", "Es Kopi Susu", "50", "Dine-in", "", "Lunas", ""},
		{"2", "46236", "gaji", "12500000", "Gaji barista"},                                             // tanggal seri Excel, alias jenis
		{"3", "03/08/2026", "kulakan", "2.000.000,00", "", "Susu", "", "", "Toko Susu", "belum lunas"}, // dd/mm/yyyy, alias, desimal
		{},                                       // baris kosong dilewati
		{"5", "2026-08-05", "Sedekah", "100000"}, // jenis tidak dikenal
		{"6", "", "Penjualan", "100000"},         // tanggal kosong
		{"7", "2026-08-07", "Penjualan", "nol"},  // nominal tidak valid
		{"8", "2026-08-08", "Sewa", "8000000", "Sewa ruko", "", "", "", "", "Lunas", "2026-08-10"},
	}
	txs, errs := finance.ParseImport(rows)

	if len(txs) != 4 {
		t.Fatalf("want 4 valid rows, got %d: %+v (errs %+v)", len(txs), txs, errs)
	}
	first := txs[0]
	if first.Tanggal != "2026-08-01" || first.Jenis != "Penjualan" || first.Nominal != 1_250_000 || first.Item != "Es Kopi Susu" ||
		first.Qty != 50 || first.Channel != "Dine-in" || !first.Lunas {
		t.Fatalf("row 3 parsed wrong: %+v", first)
	}
	if txs[1].Tanggal != "2026-08-02" || txs[1].Jenis != "Gaji & Upah" {
		t.Fatalf("excel serial date / alias: %+v", txs[1])
	}
	if txs[2].Tanggal != "2026-08-03" || txs[2].Jenis != "Beli Bahan / Stok" || txs[2].Nominal != 2_000_000 || txs[2].Lunas {
		t.Fatalf("dd/mm/yyyy, alias, decimal, belum lunas: %+v", txs[2])
	}
	if txs[3].Jenis != "Sewa Tempat" || txs[3].TanggalLunas != "2026-08-10" {
		t.Fatalf("sewa: %+v", txs[3])
	}

	gotRows := map[int]bool{}
	for _, e := range errs {
		gotRows[e.Baris] = true
		if e.Pesan == "" {
			t.Errorf("row %d: empty message", e.Baris)
		}
	}
	for _, r := range []int{7, 8, 9} { // nomor baris Excel (1-based)
		if !gotRows[r] {
			t.Errorf("want error on Excel row %d, errs=%+v", r, errs)
		}
	}
	if len(errs) != 3 {
		t.Fatalf("want 3 errors, got %+v", errs)
	}
}

func TestParseImport_MissingHeader(t *testing.T) {
	_, errs := finance.ParseImport([][]string{{"foo", "bar"}, {"1", "2"}})
	if len(errs) != 1 || errs[0].Baris != 0 {
		t.Fatalf("want one header error, got %+v", errs)
	}
}

func TestTemplateRows_RoundTrip(t *testing.T) {
	rows := finance.TemplateRows()
	txs, errs := finance.ParseImport(rows)
	if len(errs) != 0 || len(txs) != len(rows)-1 {
		t.Fatalf("template examples must import cleanly: %d txs, errs %+v", len(txs), errs)
	}
}

func TestParseImportBaris_KolomBarang(t *testing.T) {
	rows := [][]string{
		{"Tanggal", "Jenis", "Nominal", "Barang", "Jumlah Barang"},
		{"2026-01-10", "Penjualan", "1000", "Kaos", "2,5"},
		{"2026-01-11", "Penjualan", "1000", "jasa", ""},
		{"2026-01-12", "Penjualan", "1000", "Kaos", "abc"},
		{"2026-01-13", "Penjualan", "1000", "Kaos", "1.250,5"},
	}
	got, errs := finance.ParseImportBaris(rows)
	if len(got) != 3 || len(errs) != 1 || errs[0].Baris != 4 {
		t.Fatalf("hasil: %+v %+v", got, errs)
	}
	if got[0].Baris != 2 || got[0].BarangTeks != "Kaos" || got[0].JumlahBarang != 2500 || !got[0].AdaJumlah {
		t.Errorf("baris 2: %+v", got[0])
	}
	if !finance.JasaTeks(got[1].BarangTeks) || got[1].AdaJumlah {
		t.Errorf("jasa: %+v", got[1])
	}
	if got[2].JumlahBarang != 1250500 {
		t.Errorf("1.250,5: %d", got[2].JumlahBarang)
	}
}
