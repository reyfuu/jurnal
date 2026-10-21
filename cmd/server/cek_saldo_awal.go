package main

import (
	"fmt"
	"net/http"

	"pajak/internal/finance"
)

// Pemeriksaan kecocokan saldo awal dengan rinciannya: angka yang diisi di Saldo Awal harus sama
// dengan daftar yang menjadi sumbernya — aset tetap = nilai buku daftar aset, persediaan = nilai
// daftar barang / hitung fisik pertama kartu stok, dan Harta = Kewajiban + Modal. Selisih tidak
// diblokir (data bisa diisi bertahap), tapi diperingatkan di Saldo Awal, Langkah memulai, dan
// Tutup Buku, karena selisih yang dibiarkan ikut ke semua laporan.

type cekKecocokan struct {
	Kode         string `json:"kode"` // aset | persediaan | seimbang
	Judul        string `json:"judul"`
	Status       string `json:"status"` // ok | selisih | info
	SaldoAwal    int64  `json:"saldo_awal"`
	Pembanding   int64  `json:"pembanding"`
	LabelBanding string `json:"label_banding"`
	Selisih      int64  `json:"selisih"`
	Pesan        string `json:"pesan"`
	View         string `json:"view"`
	Langkah      string `json:"langkah"` // kode Langkah memulai yang terkait
}

// pembandingPersediaanLocked: nilai persediaan yang seharusnya ada di saldo awal — hitung fisik
// pertama kartu stok bila sudah dimulai, selain itu nilai daftar barang. s.mu dipegang.
func (s *tenant) pembandingPersediaanLocked() (int64, string) {
	if k := s.kartuLocked(); k.Mulai != "" {
		return k.Periode(k.Mulai[:7]).PersediaanAwal, "Nilai hitung fisik pertama " + finance.FormatTanggalID(k.Mulai)
	}
	return finance.SummarizeStok(s.stok).NilaiPersediaan, "Nilai daftar barang (jumlah × harga pokok)"
}

// cekSaldoAwalLocked: pemanggil memegang s.mu. Kosong bila saldo awal belum diisi.
func (s *tenant) cekSaldoAwalLocked() []cekKecocokan {
	if s.opening == nil {
		return nil
	}
	ob := *s.opening
	rp := finance.FormatRupiahPenuh
	var out []cekKecocokan

	// Aset tetap: hanya bila saldo awal menyatakan nilainya (per akun / template); saldo awal
	// lama tanpa baris aset tetap memakai nilai buku daftar aset langsung.
	nbv := finance.AssetsForPeriod(s.assets, s.txs, ob.Period()).NilaiBukuAwal
	c := cekKecocokan{Kode: "aset", Judul: "Aset tetap = daftar aset", SaldoAwal: ob.AsetTetap, Pembanding: nbv,
		LabelBanding: "Nilai buku daftar aset per " + finance.FormatTanggalID(ob.Tanggal), View: "saldo-awal", Langkah: "aset"}
	switch {
	case len(ob.Rincian) == 0 && ob.AsetTetap == 0:
		c.Status, c.SaldoAwal = "info", nbv
		c.Pesan = "Saldo awal memakai nilai buku daftar aset langsung (" + rp(nbv) + ")."
	case ob.AsetTetap == nbv:
		c.Status, c.Pesan = "ok", "Aset tetap di saldo awal sama dengan daftar aset ("+rp(nbv)+")."
	default:
		c.Status, c.Selisih = "selisih", ob.AsetTetap-nbv
		c.Pesan = fmt.Sprintf("Saldo awal mencatat aset tetap %s, tapi daftar aset (yang dibeli sebelum %s) bernilai buku %s — selisih %s. %s",
			rp(ob.AsetTetap), finance.FormatTanggalID(ob.Tanggal), rp(nbv), rp(c.Selisih), saranSelisihAset(c.Selisih))
	}
	out = append(out, c)

	// Persediaan: dengan kartu stok dibandingkan dengan hitung fisik pertama (titik nol),
	// tanpa kartu stok dengan nilai daftar barang di Stok & Gudang.
	k := s.kartuLocked()
	c = cekKecocokan{Kode: "persediaan", Judul: "Persediaan = daftar barang", SaldoAwal: ob.Persediaan, View: "stok-gudang", Langkah: "barang"}
	c.Pembanding, c.LabelBanding = s.pembandingPersediaanLocked()
	if k.Mulai != "" {
		c.Judul, c.Langkah = "Persediaan = hitung fisik pertama (kartu stok)", "kartu"
	}
	switch {
	case ob.Persediaan == 0 && c.Pembanding == 0:
		c.Status, c.Pesan = "info", "Saldo awal tanpa persediaan dan belum ada barang bernilai."
	case ob.Persediaan == c.Pembanding:
		c.Status, c.Pesan = "ok", "Persediaan di saldo awal sama dengan "+lower(c.LabelBanding)+" ("+rp(c.Pembanding)+")."
	default:
		c.Status, c.Selisih = "selisih", ob.Persediaan-c.Pembanding
		c.Pesan = fmt.Sprintf("Saldo awal mencatat persediaan %s, sedangkan %s %s — selisih %s. Samakan jumlah / harga pokok barang, atau ubah persediaan di Saldo Awal.",
			rp(ob.Persediaan), lower(c.LabelBanding), rp(c.Pembanding), rp(c.Selisih))
		if k.Mulai != "" {
			c.Pesan += " Selama belum sama, selisihnya dicatat sebagai tambahan modal di neraca."
		}
	}
	out = append(out, c)

	// Harta = Kewajiban + Modal (saldo awal per akun).
	if d := ob.SelisihModal(nbv); d != 0 {
		out = append(out, cekKecocokan{Kode: "seimbang", Judul: "Harta = Kewajiban + Modal", Status: "selisih", Selisih: d,
			SaldoAwal: ob.Modal(), Pembanding: ob.ModalAwal(nbv), LabelBanding: "Harta − Kewajiban", View: "saldo-awal", Langkah: "saldo",
			Pesan: fmt.Sprintf("Modal di saldo awal %s, sedangkan Harta − Kewajiban %s — selisih %s. Biasanya karena daftar aset berubah setelah saldo awal disimpan; buka Saldo Awal lalu samakan.",
				rp(ob.Modal()), rp(ob.ModalAwal(nbv)), rp(d))})
	}
	return out
}

func saranSelisihAset(selisih int64) string {
	if selisih > 0 {
		return "Ada aset yang belum didaftarkan (atau tanggal belinya setelah tanggal saldo awal): tambahkan di Laporan Keuangan → Aset & Penyusutan, atau kurangi aset tetap di Saldo Awal."
	}
	return "Daftar aset lebih besar: hapus aset yang tidak ada, atau naikkan aset tetap di Saldo Awal (Modal ikut disesuaikan)."
}

func lower(s string) string {
	if s == "" {
		return s
	}
	return string(s[0]|0x20) + s[1:]
}

func (s *tenant) handleCekSaldoAwal(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	out := s.cekSaldoAwalLocked()
	s.mu.RUnlock()
	if out == nil {
		out = []cekKecocokan{}
	}
	writeJSON(w, out)
}
