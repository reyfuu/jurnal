package finance_test

import (
	"strings"
	"testing"

	"pajak/internal/finance"
)

// Impor invoice: satu baris per item, baris bernomor sama = satu invoice; nomor kosong +
// pelanggan = invoice baru bernomor otomatis, nomor & pelanggan kosong = lanjutan invoice
// sebelumnya. Status kosong = terkirim; PPN kosong = tarif usaha, 0 = tanpa PPN.
func TestParseInvoiceFile_Pengelompokan(t *testing.T) {
	h := []string{"Nomor", "Tanggal", "Jatuh Tempo", "Pelanggan", "Alamat", "NPWP", "No. PO", "Status", "Tanggal Lunas", "Deskripsi", "Qty", "Satuan", "Harga Satuan", "Diskon (Rp)", "PPN (%)", "Catatan"}
	rows := [][]string{h,
		{"INV-A-1", "2026-09-01", "2026-09-30", "PT Satu", "", "", "", "", "", "Kopi literan", "10", "liter", "120000", "", "", ""},
		{"INV-A-1", "", "", "", "", "", "", "", "", "Croissant", "20", "pcs", "25000", "", "", ""},
		{"", "2026-09-02", "2026-09-16", "CV Dua", "", "", "", "Lunas", "2026-09-10", "Katering rapat", "1", "paket", "1500000", "100000", "0", ""},
		{"", "", "", "", "", "", "", "", "", "Ongkir", "1", "", "50000", "", "", ""},
		{"INV-A-3", "2026-09-03", "2026-09-20", "PT Tiga", "", "", "", "draft", "", "Biji kopi", "2,5", "kg", "200000", "", "11%", ""},
		{"INV-A-4", "2026-09-04", "2026-09-01", "PT Empat", "", "", "", "", "", "Kopi", "1", "", "10000", "", "", ""},
	}
	list, errs, nama := finance.ParseInvoiceFile([]finance.ImportSheet{{Nama: "Invoice", Rows: rows}, {Nama: "Panduan", Rows: [][]string{{"x"}}}}, 1100)
	if nama != "Invoice" || len(list) != 2 {
		t.Fatalf("sheet %q, %d invoice, errors %+v", nama, len(list), errs)
	}
	a, b := list[0].Invoice, list[1].Invoice
	if a.Nomor != "INV-A-1" || len(a.Items) != 2 || a.Status != finance.StatusTerkirim || a.PPNBps != 1100 || list[0].Baris != 2 {
		t.Errorf("invoice 1: %+v (baris %d)", a, list[0].Baris)
	}
	if b.Nomor != "" || b.PelangganNama != "CV Dua" || len(b.Items) != 2 || b.Status != finance.StatusLunas || b.PPNBps != 0 || b.DiskonRp != 100_000 || b.TanggalLunas != "2026-09-10" {
		t.Errorf("invoice 2 (nomor otomatis): %+v", b)
	}
	salah := map[int]string{}
	for _, e := range errs {
		salah[e.Baris] = e.Pesan
	}
	if !strings.Contains(salah[6], "qty") || !strings.Contains(salah[7], "jatuh tempo") || len(errs) != 2 {
		t.Errorf("qty pecahan (baris 6) & jatuh tempo sebelum terbit (baris 7) harus ditolak: %+v", errs)
	}
	if _, errs, _ := finance.ParseInvoiceFile(finance.InvoiceTemplate(), 1100); len(errs) != 0 {
		t.Errorf("template sendiri harus bersih: %+v", errs)
	}
}
