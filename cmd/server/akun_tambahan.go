package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"

	"pajak/internal/coa"
	"pajak/internal/finance"
)

// Akun buatan klien di Daftar Akun: jenisnya mengikuti akun induk (kelompok, saldo normal,
// posisi di laporan), kodenya kode kosong berikutnya setelah induk.

func (s *tenant) handleTambahAkun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Nama  string `json:"nama"`
		Induk string `json:"induk"`
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
	all, active, _ := s.coaAkun()
	if all == nil {
		http.Error(w, "daftar akun belum tersedia untuk jenis usaha ini", http.StatusConflict)
		return
	}
	if _, ok := coa.Find(active, req.Induk); !ok {
		http.Error(w, "pilih akun induk dari daftar", http.StatusBadRequest)
		return
	}
	if slices.ContainsFunc(all, func(a coa.Account) bool { return strings.EqualFold(a.Nama, req.Nama) }) {
		http.Error(w, "nama akun sudah ada di daftar", http.StatusConflict)
		return
	}
	// induk turunan (akun tambahan) tetap merujuk induk aslinya di template
	induk := req.Induk
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.profile == nil {
		http.Error(w, "lengkapi profil usaha dulu", http.StatusConflict)
		return
	}
	for _, t := range s.profile.AkunTambahan {
		if t.Kode == induk {
			induk = t.Induk
		}
	}
	kode := coa.KodeTurunan(req.Induk, all)
	if kode == "" {
		http.Error(w, "kode di kelompok akun ini sudah penuh; pilih akun induk lain", http.StatusConflict)
		return
	}
	prof := *s.profile
	prof.AkunTambahan = append(slices.Clone(prof.AkunTambahan), finance.AkunTambahan{Kode: kode, Nama: req.Nama, Induk: induk})
	if err := saveProfile(s.file("business"), prof); err != nil {
		http.Error(w, "gagal menyimpan akun", http.StatusInternalServerError)
		return
	}
	s.profile = &prof
	writeJSON(w, finance.AkunTambahan{Kode: kode, Nama: req.Nama, Induk: induk})
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
