package main

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"pajak/internal/coa"
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

// pembandingPersediaanPerAkunLocked: pembanding persediaan per akun persediaan efektif barang —
// hitung fisik pertama bila kartu stok sudah dimulai, selain itu jumlah × harga pokok daftar
// barang. Jumlahnya sama dengan pembandingPersediaanLocked. s.mu dipegang.
func (s *tenant) pembandingPersediaanPerAkunLocked(all []coa.Account, peta coa.Peta) map[string]int64 {
	if k := s.kartuLocked(); k.Mulai != "" {
		return s.persediaanAwalPerAkunLocked(k.Mulai[:7], all, peta)
	}
	akunOf := s.akunStokLocked(all, peta)
	out := map[string]int64{}
	for _, it := range s.stok {
		out[akunOf(it.ID)] += it.Jumlah * it.HargaPokok
	}
	return out
}

// persediaanSaldoAwalPerAkun: persediaan saldo awal per akun. Baris berkode akun persediaan masuk
// akunnya, sisanya (baris lain, saldo awal tanpa rincian) ke akun peran — sama dengan jurnal.
func persediaanSaldoAwalPerAkun(ob finance.OpeningBalance, all []coa.Account, peta coa.Peta) map[string]int64 {
	peran := peta.Kode(coa.PeranPersediaan)
	punya := map[string]bool{}
	for _, a := range coa.AkunPersediaan(all) {
		punya[a.Kode] = true
	}
	out := map[string]int64{}
	sisa := ob.Persediaan
	for _, r := range ob.Rincian {
		if r.Kategori == finance.KatPersediaan && punya[r.Kode] && r.Kode != peran {
			out[r.Kode] += r.Nominal
			sisa -= r.Nominal
		}
	}
	out[peran] += sisa
	return out
}

// barisPersediaanAkun: pembanding per akun sebagai baris saldo awal (urut kode, hanya yang
// bernilai), untuk baris persediaan form Saldo Awal yang diisi dari daftar barang.
func barisPersediaanAkun(per map[string]int64, all []coa.Account, label string) []finance.SaldoAkun {
	out := []finance.SaldoAkun{}
	for _, kd := range slices.Sorted(maps.Keys(per)) {
		if per[kd] == 0 {
			continue
		}
		nama := "Persediaan"
		if a, ok := coa.Find(all, kd); ok {
			nama = a.Nama
		}
		out = append(out, finance.SaldoAkun{Kode: kd, Nama: nama, Kategori: finance.KatPersediaan, Nominal: per[kd], Keterangan: label})
	}
	return out
}

// akunCekLocked: akun CoA usaha untuk pemeriksaan saldo awal, termasuk akun tambahan klien
// (mis. persediaan turunan 1301), sama dengan daftar di layar Saldo Awal. s.mu dipegang.
func (s *tenant) akunCekLocked() ([]coa.Account, coa.Peta) {
	all := s.akunLocked()
	return all, coa.PetaDari(all)
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

	// Saldo awal yang disimpan paksa walau belum seimbang: selisihnya di akun penampung.
	if i := slices.IndexFunc(ob.Rincian, barisSelisihSaldoAwal); i >= 0 {
		r := ob.Rincian[i]
		out = append(out, cekKecocokan{Kode: "paksa", Judul: "Selisih saldo awal belum dibereskan", Status: "selisih", SaldoAwal: r.Nominal,
			LabelBanding: r.Kode + " " + r.Nama, Selisih: r.Nominal, View: "saldo-awal", Langkah: "saldo",
			Pesan: fmt.Sprintf("Saldo awal disimpan walau belum seimbang; selisih %s ditampung di akun %s %s. Periksa lagi angka per akun lalu simpan ulang sampai seimbang.", rp(r.Nominal), r.Kode, r.Nama)})
	}

	// Persediaan: dengan kartu stok dibandingkan dengan hitung fisik pertama (titik nol),
	// tanpa kartu stok dengan nilai daftar barang di Stok & Gudang.
	k := s.kartuLocked()
	c = cekKecocokan{Kode: "persediaan", Judul: "Persediaan = daftar barang", SaldoAwal: ob.Persediaan, View: "stok-gudang", Langkah: "barang"}
	c.Pembanding, c.LabelBanding = s.pembandingPersediaanLocked()
	if k.Mulai != "" {
		c.Judul, c.Langkah = "Persediaan = hitung fisik pertama (kartu stok)", "kartu"
	}
	// Beberapa akun persediaan terisi (bahan baku, barang jadi, …): satu entri per akun yang
	// berbeda, karena total yang sama belum tentu sama per akunnya.
	all, peta := s.akunCekLocked()
	simpan, banding := persediaanSaldoAwalPerAkun(ob, all, peta), s.pembandingPersediaanPerAkunLocked(all, peta)
	akun := map[string]bool{}
	for _, m := range []map[string]int64{simpan, banding} {
		for kd, v := range m {
			if v != 0 {
				akun[kd] = true
			}
		}
	}
	if len(akun) > 1 {
		var beda []cekKecocokan
		for _, kd := range slices.Sorted(maps.Keys(akun)) {
			if simpan[kd] == banding[kd] {
				continue
			}
			nama := kd
			if a, ok := coa.Find(all, kd); ok {
				nama = kd + " " + a.Nama
			}
			ck := c
			ck.Kode, ck.Status, ck.SaldoAwal, ck.Pembanding, ck.Selisih = "persediaan_"+kd, "selisih", simpan[kd], banding[kd], simpan[kd]-banding[kd]
			ck.Judul = "Persediaan " + nama + " = " + strings.TrimPrefix(c.Judul, "Persediaan = ")
			ck.Pesan = fmt.Sprintf("Saldo awal mencatat %s sebesar %s, sedangkan %s untuk akun ini %s — selisih %s. Samakan jumlah / harga pokok / akun barang, atau ubah baris akun ini di Saldo Awal.",
				nama, rp(ck.SaldoAwal), lower(c.LabelBanding), rp(ck.Pembanding), rp(ck.Selisih))
			if k.Mulai != "" {
				ck.Pesan += " Selama belum sama, selisihnya dicatat sebagai tambahan modal di neraca."
			}
			beda = append(beda, ck)
		}
		if len(beda) > 0 {
			out = append(out, beda...)
		} else {
			c.Status, c.Pesan = "ok", "Persediaan di saldo awal sama dengan "+lower(c.LabelBanding)+" per akun ("+rp(c.Pembanding)+")."
			out = append(out, c)
		}
	} else {
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
	}

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
