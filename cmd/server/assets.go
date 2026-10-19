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

// decodeAsset membaca dan memvalidasi body; menulis 400 dan mengembalikan false bila gagal.
func decodeAsset(w http.ResponseWriter, r *http.Request) (finance.Asset, bool) {
	var a finance.Asset
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, "data aset tidak valid", http.StatusBadRequest)
		return a, false
	}
	if err := finance.ValidateAsset(&a); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return a, false
	}
	return a, true
}

func (s *tenant) handleCreateAsset(w http.ResponseWriter, r *http.Request) {
	a, ok := decodeAsset(w, r)
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
	a, ok := decodeAsset(w, r)
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
