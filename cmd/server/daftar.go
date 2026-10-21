package main

import (
	"bytes"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"pajak/internal/finance"
	"pajak/internal/xlsx"
)

// Unduh semua daftar: piutang & utang (per tanggal akhir), invoice yang terbit dalam rentang,
// barang (posisi stok), pelanggan (penjualan dalam rentang + piutangnya), dan pemasok (+ utangnya)
// dalam satu Excel atau satu PDF. Karyawan tidak ikut: data gaji punya hak akses sendiri.

var labelStatusInvoice = map[string]string{finance.StatusDraft: "Draft", finance.StatusTerkirim: "Belum dibayar", finance.StatusLunas: "Lunas"}

func (t *tenant) daftarUsaha(dari, sampai string) []finance.Daftar {
	t.mu.RLock()
	ob := t.bukuAwalLocked()
	piutang := finance.UmurSaldo(ob, t.txs, sampai, true)
	utang := finance.UmurSaldo(ob, t.txs, sampai, false)
	pelanggan := finance.RekapPelanggan(t.txs, dari, sampai)
	invoices := slices.Clone(t.invoices)
	stok := finance.TerapkanKartu(slices.Clone(t.stok), t.kartuLocked())
	pemasok := slices.Clone(t.supplier)
	t.mu.RUnlock()

	rp := func(n int64) string { return strconv.FormatInt(n, 10) }
	umur := func(judul, pihak string, u finance.UmurSaldoResult) finance.Daftar {
		d := finance.Daftar{Judul: judul, Kolom: []string{"Tanggal", pihak, "Keterangan", "Jatuh Tempo", "Umur (hari)", "Sisa (Rp)"}, Rupiah: []int{5}}
		for _, b := range u.Baris {
			d.Baris = append(d.Baris, []string{b.Tanggal, b.Pihak, b.Keterangan, b.JatuhTempo, strconv.Itoa(b.Hari), rp(b.Nilai)})
		}
		d.Baris = append(d.Baris, []string{"Total", "", "", "", "", rp(u.Total)})
		return d
	}
	sisaPer := func(u finance.UmurSaldoResult) map[string]int64 {
		m := map[string]int64{}
		for _, p := range u.Pihak {
			m[strings.ToLower(strings.TrimSpace(p.Nama))] += p.Total
		}
		return m
	}

	inv := finance.Daftar{Judul: "Invoice", Kolom: []string{"Nomor", "Tanggal", "Jatuh Tempo", "Pelanggan", "Status", "DPP (Rp)", "PPN (Rp)", "Total (Rp)", "Tanggal Lunas"}, Rupiah: []int{5, 6, 7}}
	var dpp, ppn, total int64
	for _, x := range invoices {
		if x.Tanggal < dari || x.Tanggal > sampai {
			continue
		}
		tot := x.Totals()
		dpp, ppn, total = dpp+tot.DPP, ppn+tot.PPN, total+tot.Total
		inv.Baris = append(inv.Baris, []string{x.Nomor, x.Tanggal, x.JatuhTempo, x.PelangganNama, labelStatusInvoice[x.Status], rp(tot.DPP), rp(tot.PPN), rp(tot.Total), x.TanggalLunas})
	}
	inv.Baris = append(inv.Baris, []string{"Total", "", "", "", "", rp(dpp), rp(ppn), rp(total), ""})

	barang := finance.Daftar{Judul: "Barang", Kolom: []string{"SKU", "Nama Barang", "Kategori", "Jumlah", "Satuan", "Harga Pokok (Rp)", "Nilai (Rp)"}, Rupiah: []int{5, 6}}
	var nilaiStok int64
	for _, it := range stok {
		nilai := it.Jumlah * it.HargaPokok / finance.TakaranSkala
		nilaiStok += nilai
		barang.Baris = append(barang.Baris, []string{it.SKU, it.Nama, it.Kategori, finance.FormatQty(it.Jumlah), it.Satuan, rp(it.HargaPokok), rp(nilai)})
	}
	barang.Baris = append(barang.Baris, []string{"Total", "", "", "", "", "", rp(nilaiStok)})

	// Pelanggan: yang berbelanja dalam rentang, lalu yang hanya punya piutang.
	sisa := sisaPer(piutang)
	plg := finance.Daftar{Judul: "Pelanggan", Kolom: []string{"Nama", "Jumlah Transaksi", "Penjualan (Rp)", "Piutang per " + sampai + " (Rp)"}, Rupiah: []int{2, 3}}
	sudah := map[string]bool{}
	for _, p := range pelanggan {
		k := strings.ToLower(p.Nama)
		sudah[k] = true
		plg.Baris = append(plg.Baris, []string{p.Nama, strconv.Itoa(p.Jumlah), rp(p.Penjualan), rp(sisa[k])})
	}
	for _, p := range piutang.Pihak {
		if k := strings.ToLower(strings.TrimSpace(p.Nama)); !sudah[k] {
			sudah[k] = true
			plg.Baris = append(plg.Baris, []string{p.Nama, "0", "0", rp(p.Total)})
		}
	}

	sisa = sisaPer(utang)
	sup := finance.Daftar{Judul: "Pemasok", Kolom: []string{"Nama", "Kontak", "Telepon", "Email", "Alamat", "Tempo (hari)", "Utang per " + sampai + " (Rp)"}, Rupiah: []int{6}}
	sudah = map[string]bool{}
	for _, s := range pemasok {
		k := strings.ToLower(strings.TrimSpace(s.Nama))
		sudah[k] = true
		sup.Baris = append(sup.Baris, []string{s.Nama, s.Kontak, s.Telepon, s.Email, s.Alamat, strconv.Itoa(s.TempoHari), rp(sisa[k])})
	}
	for _, p := range utang.Pihak { // ada utang tetapi belum terdaftar sebagai pemasok
		if k := strings.ToLower(strings.TrimSpace(p.Nama)); !sudah[k] {
			sup.Baris = append(sup.Baris, []string{p.Nama, "", "", "", "", "", rp(p.Total)})
		}
	}
	return []finance.Daftar{umur("Piutang", "Pelanggan", piutang), umur("Utang", "Pemasok", utang), inv, barang, plg, sup}
}

func (t *tenant) handleDaftarXlsx(w http.ResponseWriter, r *http.Request) {
	dari, sampai, err := rentangDari(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var sheets []xlsx.Sheet
	for _, d := range t.daftarUsaha(dari, sampai) {
		sheets = append(sheets, xlsx.Sheet{Name: d.Judul, Rows: append([][]string{d.Kolom}, d.Baris...)})
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="daftar-%s_%s.xlsx"`, dari, sampai))
	_ = xlsx.Write(w, sheets)
}

func (t *tenant) handleDaftarPDF(w http.ResponseWriter, r *http.Request) {
	dari, sampai, err := rentangDari(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sub := "Invoice & penjualan: " + labelRentang(dari, sampai) + ". Piutang, utang & stok: posisi per " + finance.FormatTanggalID(sampai) + ". Dalam Rupiah."
	var buf bytes.Buffer
	if err := finance.RenderDaftarPDF(&buf, t.currentProfile().NamaUsaha, sub, t.daftarUsaha(dari, sampai)); err != nil {
		http.Error(w, "gagal menyusun PDF", http.StatusInternalServerError)
		return
	}
	kirimPDF(w, fmt.Sprintf("daftar-%s_%s.pdf", dari, sampai), buf.Bytes())
}
