package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"

	"pajak/internal/finance"
)

// Register aset: sumber tunggal penyusutan & amortisasi untuk Laba Rugi, Neraca, dan Arus Kas.
// Hak akses mengikuti "laporan-keuangan" karena tab Aset ada di dalam menu itu.

func (s *tenant) handleListAssets(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.report(periodOrDefault(r)).Assets)
}

// metodeUsaha: metode penyusutan profil; "" = belum dipilih. Dipanggil dengan s.mu terkunci
// (baca atau tulis) — profile hanya diganti di bawah kunci tulis.
func (s *tenant) metodeUsaha() string {
	if s.profile == nil {
		return ""
	}
	return s.profile.MetodePenyusutan
}

// decodeAsset membaca dan memvalidasi body; menulis 400 dan mengembalikan false bila gagal.
// Bila usaha sudah memilih metode penyusutan, metode di body diabaikan.
func (s *tenant) decodeAsset(w http.ResponseWriter, r *http.Request) (finance.Asset, bool) {
	var a finance.Asset
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, "data aset tidak valid", http.StatusBadRequest)
		return a, false
	}
	s.mu.RLock()
	m := s.metodeUsaha()
	s.mu.RUnlock()
	if m != "" {
		a.Metode = ""
	}
	if err := finance.ValidateAsset(&a); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return a, false
	}
	a.Metode = finance.MetodeUntuk(m, a)
	return a, true
}

// handlePilihMetode: Owner memilih metode penyusutan usaha satu kali; semua aset dihitung ulang.
func (s *tenant) handlePilihMetode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Metode string `json:"metode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "data tidak valid", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case s.profile == nil:
		http.Error(w, "lengkapi profil usaha dulu", http.StatusConflict)
		return
	case s.profile.MetodePenyusutan != "":
		http.Error(w, errMetodeTerkunci, http.StatusConflict)
		return
	}
	prof := *s.profile
	prof.MetodePenyusutan = req.Metode
	if req.Metode == "" {
		prof.MetodePenyusutan = "-" // kosong bukan pilihan
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
	finance.TerapkanMetode(s.assets, prof.MetodePenyusutan)
	writeJSON(w, prof)
}

const errMetodeTerkunci = "Metode penyusutan sudah dipilih dan dikunci untuk semua aset. Perubahan metode memerlukan persetujuan DJP; hubungi tim Master Pajak."

func (s *tenant) handleCreateAsset(w http.ResponseWriter, r *http.Request) {
	a, ok := s.decodeAsset(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	a.ID = s.nextAssetID
	s.nextAssetID++
	s.assets = append(s.assets, a)
	s.mu.Unlock()
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, a)
}

// assetIndex harus dipanggil dengan s.mu terkunci.
func (s *tenant) assetIndex(r *http.Request) int {
	id, _ := strconv.Atoi(r.PathValue("id"))
	return slices.IndexFunc(s.assets, func(a finance.Asset) bool { return a.ID == id })
}

func (s *tenant) handleUpdateAsset(w http.ResponseWriter, r *http.Request) {
	a, ok := s.decodeAsset(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.assetIndex(r)
	if i < 0 {
		http.Error(w, "aset tidak ditemukan", http.StatusNotFound)
		return
	}
	a.ID = s.assets[i].ID
	s.assets[i] = a
	writeJSON(w, a)
}

func (s *tenant) handleDeleteAsset(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.assetIndex(r)
	if i < 0 {
		http.Error(w, "aset tidak ditemukan", http.StatusNotFound)
		return
	}
	s.assets = slices.Delete(s.assets, i, i+1)
	w.WriteHeader(http.StatusNoContent)
}
