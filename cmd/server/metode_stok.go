package main

import (
	"encoding/json"
	"net/http"
	"slices"

	"pajak/internal/finance"
)

// Metode nilai stok (FIFO / rata-rata bergerak) diperlakukan seperti metode penyusutan: dipilih
// sekali per usaha lalu dikunci, karena perubahan metode pembukuan butuh persetujuan DJP.
// Bawaan FIFO. Usaha yang memulai kartu stok sebelum pilihan ini ada selama ini dihitung
// rata-rata bergerak, jadi tercatat & terkunci di rata-rata supaya angka lamanya tidak berubah.

const errMetodeStokTerkunci = "Metode nilai stok sudah dipilih dan dikunci. Perubahan metode memerlukan persetujuan DJP; hubungi tim MasterPajak AI."

// metodeStokLocked: metode yang berlaku dan apakah sudah terkunci. s.mu dipegang.
func (s *tenant) metodeStokLocked() (metode string, terkunci bool) {
	if s.profile != nil && s.profile.MetodeNilaiStok != "" {
		return s.profile.MetodeNilaiStok, true
	}
	if slices.ContainsFunc(s.gerak, func(g finance.GerakStok) bool { return g.Jenis == finance.GerakAwal }) {
		return finance.MetodeRataRata, true // kartu stok dimulai sebelum ada pilihan metode
	}
	return finance.MetodeFIFO, false
}

// kunciMetodeStokLocked: kartu stok dimulai tanpa memilih → metode bawaan dicatat & dikunci.
func (s *tenant) kunciMetodeStokLocked() error {
	if s.profile == nil || s.profile.MetodeNilaiStok != "" {
		return nil
	}
	prof := *s.profile
	prof.MetodeNilaiStok, _ = s.metodeStokLocked()
	if err := saveProfile(s.file("business"), prof); err != nil {
		return err
	}
	s.profile = &prof
	return nil
}

type metodeStokResp struct {
	Metode   string           `json:"metode"`
	Terkunci bool             `json:"terkunci"`
	Pilihan  []finance.Option `json:"pilihan"`
}

func (s *tenant) handleGetMetodeStok(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	m, kunci := s.metodeStokLocked()
	s.mu.RUnlock()
	writeJSON(w, metodeStokResp{m, kunci, finance.MetodeNilaiStokOptions})
}

// handlePilihMetodeStok: Owner memilih metode nilai stok satu kali.
func (s *tenant) handlePilihMetodeStok(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Metode string `json:"metode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "data tidak valid", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.profile == nil {
		http.Error(w, "lengkapi profil usaha dulu", http.StatusConflict)
		return
	}
	if _, kunci := s.metodeStokLocked(); kunci {
		http.Error(w, errMetodeStokTerkunci, http.StatusConflict)
		return
	}
	prof := *s.profile
	prof.MetodeNilaiStok = req.Metode
	if req.Metode == "" {
		prof.MetodeNilaiStok = "-" // kosong bukan pilihan
	}
	if err := prof.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := saveProfile(s.file("business"), prof); err != nil {
		http.Error(w, "gagal menyimpan profil usaha", http.StatusInternalServerError)
		return
	}
	s.profile = &prof
	writeJSON(w, metodeStokResp{prof.MetodeNilaiStok, true, finance.MetodeNilaiStokOptions})
}
