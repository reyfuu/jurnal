package finance_test

import (
	"slices"
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

// Kolom Barang (opsional, header persis seperti impor transaksi): teksnya dibawa per item untuk
// dicocokkan server; "jasa" langsung menjadi Bukan barang.
func TestParseInvoiceFile_KolomBarang(t *testing.T) {
	h := []string{"Nomor", "Tanggal", "Jatuh Tempo", "Pelanggan", "Deskripsi", "Qty", "Harga Satuan", "Barang"}
	rows := [][]string{h,
		{"INV-B-1", "2026-09-01", "2026-09-30", "PT Satu", "Kain", "2", "20000", "KN-01"},
		{"INV-B-1", "", "", "", "Ongkir", "1", "10000", " Jasa "},
		{"INV-B-1", "", "", "", "Bonus", "1", "0", ""},
	}
	list, errs, _ := finance.ParseInvoiceFile([]finance.ImportSheet{{Nama: "Invoice", Rows: rows}}, 0)
	if len(list) != 1 || len(errs) != 0 {
		t.Fatalf("%d invoice, errors %+v", len(list), errs)
	}
	x := list[0]
	if len(x.Barang) != 3 || x.Barang[0] != "KN-01" || x.Barang[1] != "" || x.Barang[2] != "" {
		t.Errorf("teks barang per item: %q", x.Barang)
	}
	if it := x.Invoice.Items; it[0].BukanBarang || !it[1].BukanBarang || it[2].BukanBarang || it[0].StokID != 0 {
		t.Errorf("tanda bukan barang: %+v", it)
	}
	// Header "Nama Barang" bukan kolom barang (berkas lama tetap terbaca seperti dulu).
	h2 := []string{"Nomor", "Tanggal", "Jatuh Tempo", "Pelanggan", "Nama Barang", "Deskripsi", "Qty", "Harga Satuan"}
	list, _, _ = finance.ParseInvoiceFile([]finance.ImportSheet{{Nama: "Invoice", Rows: [][]string{h2,
		{"INV-B-2", "2026-09-01", "2026-09-30", "PT Dua", "jasa", "Kopi", "1", "1000"}}}}, 0)
	if len(list) != 1 || list[0].Invoice.Items[0].BukanBarang || list[0].Barang[0] != "" {
		t.Errorf("header Nama Barang tidak boleh dibaca sebagai kolom barang: %+v", list)
	}
	// Template memuat kolom Barang.
	if tpl := finance.InvoiceTemplate(); !slices.Contains(tpl[0].Rows[0], "Barang") {
		t.Errorf("template tanpa kolom Barang: %q", tpl[0].Rows[0])
	}
}
