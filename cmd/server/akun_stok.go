package main

import (
	"net/http"
	"slices"

	"pajak/internal/coa"
	"pajak/internal/finance"
)

// akunStokLocked: akun persediaan efektif per barang. Akun pilihan dipakai bila masih ada di
// CoA; selain itu (kosong, atau akunnya sudah hilang) ditebak dari kategori. StokID tak dikenal
// -> akun peran persediaan. Pemanggil memegang s.mu.
func (s *tenant) akunStokLocked(all []coa.Account, peta coa.Peta) func(stokID int) string {
	punya := map[string]bool{}
	for _, a := range coa.AkunPersediaan(all) {
		punya[a.Kode] = true
	}
	return func(stokID int) string {
		i := slices.IndexFunc(s.stok, func(x finance.StokItem) bool { return x.ID == stokID })
		if i < 0 {
			return peta.Kode(coa.PeranPersediaan)
		}
		it := s.stok[i]
		if punya[it.Akun] {
			return it.Akun
		}
		return coa.AkunPersediaanBarang(it.Kategori, all, peta)
	}
}

// akunStokValid: akun yang dikirim harus salah satu akun persediaan CoA usaha ini.
func (s *tenant) akunStokValid(w http.ResponseWriter, it finance.StokItem) bool {
	if it.Akun == "" {
		return true
	}
	all, _, _ := s.coaAkun()
	for _, a := range coa.AkunPersediaan(all) {
		if a.Kode == it.Akun {
			return true
		}
	}
	http.Error(w, "akun persediaan tidak dikenal", http.StatusBadRequest)
	return false
}

type opsiAkunStok struct {
	Kode string `json:"kode"`
	Nama string `json:"nama"`
}

type stokBaris struct {
	finance.StokRow
	AkunEfektif string `json:"akun_efektif"`
}

type stokDaftar struct {
	finance.StokSummary
	Items       []stokBaris   `json:"items"`
	PilihanAkun []opsiAkunStok `json:"pilihan_akun"`
}

// denganAkun: ringkasan stok ditambah akun efektif per barang dan daftar akun persediaan aktif.
func denganAkun(sum finance.StokSummary, akun func(int) string, aktif []coa.Account) stokDaftar {
	out := stokDaftar{StokSummary: sum, Items: make([]stokBaris, len(sum.Items)), PilihanAkun: []opsiAkunStok{}}
	for i, r := range sum.Items {
		out.Items[i] = stokBaris{StokRow: r, AkunEfektif: akun(r.ID)}
	}
	for _, a := range coa.AkunPersediaan(aktif) {
		out.PilihanAkun = append(out.PilihanAkun, opsiAkunStok{Kode: a.Kode, Nama: a.Nama})
	}
	return out
}
