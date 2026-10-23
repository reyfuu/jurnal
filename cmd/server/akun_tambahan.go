package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"

	"pajak/internal/coa"
	"pajak/internal/finance"
)

// Akun buatan klien di Daftar Akun: klien mengisi nomor akun & nama. Kelompok, saldo normal, dan
// posisi di laporan dibaca dari seri nomornya (akun lain di ratusan yang sama); seri baru memilih
// kelompoknya sekali. Nomor tidak bisa diubah, nama bisa.

func (s *tenant) handleTambahAkun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Kode     string `json:"kode"`
		Nama     string `json:"nama"`
		Kelompok string `json:"kelompok"` // hanya untuk seri nomor yang belum ada di daftar
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "data tidak valid", http.StatusBadRequest)
		return
	}
	req.Kode, req.Nama = strings.TrimSpace(req.Kode), strings.TrimSpace(req.Nama)
	if req.Nama == "" || len(req.Nama) > 80 {
		http.Error(w, "nama akun wajib diisi (maks. 80 huruf)", http.StatusBadRequest)
		return
	}
	all, _, _ := s.coaAkun()
	if all == nil {
		http.Error(w, "daftar akun belum tersedia untuk jenis usaha ini", http.StatusConflict)
		return
	}
	induk, err := coa.IndukNomor(all, req.Kode, coa.Kelompok(req.Kelompok))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if slices.ContainsFunc(all, func(a coa.Account) bool { return strings.EqualFold(a.Nama, req.Nama) }) {
		http.Error(w, "nama akun sudah ada di daftar", http.StatusConflict)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.profile == nil {
		http.Error(w, "lengkapi profil usaha dulu", http.StatusConflict)
		return
	}
	// induk berupa akun tambahan lain: rujuk induk aslinya di template
	for _, t := range s.profile.AkunTambahan {
		if t.Kode == induk {
			induk = t.Induk
		}
	}
	prof := *s.profile
	baru := finance.AkunTambahan{Kode: req.Kode, Nama: req.Nama, Induk: induk}
	prof.AkunTambahan = append(slices.Clone(prof.AkunTambahan), baru)
	if err := saveProfile(s.file("business"), prof); err != nil {
		http.Error(w, "gagal menyimpan akun", http.StatusInternalServerError)
		return
	}
	s.profile = &prof
	writeJSON(w, baru)
}

// handleUbahNamaAkun: nama akun buatan sendiri boleh diubah; nomornya tetap.
func (s *tenant) handleUbahNamaAkun(w http.ResponseWriter, r *http.Request) {
	kode := r.PathValue("kode")
	var req struct {
		Nama string `json:"nama"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "data tidak valid", http.StatusBadRequest)
		return
	}
	req.Nama = strings.TrimSpace(req.Nama)
	if req.Nama == "" || len(req.Nama) > 80 {
		http.Error(w, "nama akun wajib diisi (maks. 80 huruf)", http.StatusBadRequest)
		return
	}
	all, _, _ := s.coaAkun()
	if slices.ContainsFunc(all, func(a coa.Account) bool { return a.Kode != kode && strings.EqualFold(a.Nama, req.Nama) }) {
		http.Error(w, "nama akun sudah ada di daftar", http.StatusConflict)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.profile == nil {
		http.Error(w, "akun tidak ditemukan", http.StatusNotFound)
		return
	}
	i := slices.IndexFunc(s.profile.AkunTambahan, func(t finance.AkunTambahan) bool { return t.Kode == kode })
	if i < 0 {
		http.Error(w, "hanya nama akun yang ditambahkan sendiri yang bisa diubah", http.StatusNotFound)
		return
	}
	prof := *s.profile
	prof.AkunTambahan = slices.Clone(prof.AkunTambahan)
	prof.AkunTambahan[i].Nama = req.Nama
	if err := saveProfile(s.file("business"), prof); err != nil {
		http.Error(w, "gagal menyimpan akun", http.StatusInternalServerError)
		return
	}
	s.profile = &prof
	writeJSON(w, prof.AkunTambahan[i])
}

func (s *tenant) handleHapusAkun(w http.ResponseWriter, r *http.Request) {
	kode := r.PathValue("kode")
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.profile == nil {
		http.Error(w, "akun tidak ditemukan", http.StatusNotFound)
		return
	}
	i := slices.IndexFunc(s.profile.AkunTambahan, func(t finance.AkunTambahan) bool { return t.Kode == kode })
	if i < 0 {
		http.Error(w, "hanya akun yang ditambahkan sendiri yang bisa dihapus", http.StatusNotFound)
		return
	}
	if slices.ContainsFunc(s.txs, func(t finance.Transaction) bool { return t.Akun == kode || t.Lawan == kode }) {
		http.Error(w, "akun masih dipakai transaksi; pindahkan transaksinya dulu", http.StatusConflict)
		return
	}
	if slices.ContainsFunc(s.profile.AkunTambahan, func(t finance.AkunTambahan) bool { return t.Induk == kode }) {
		http.Error(w, "akun ini menjadi induk akun lain", http.StatusConflict)
		return
	}
	prof := *s.profile
	prof.AkunTambahan = slices.Delete(slices.Clone(prof.AkunTambahan), i, i+1)
	if err := saveProfile(s.file("business"), prof); err != nil {
		http.Error(w, "gagal menghapus akun", http.StatusInternalServerError)
		return
	}
	s.profile = &prof
	w.WriteHeader(http.StatusNoContent)
}
