package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"

	"pajak/internal/finance"
)

// Daftar pelanggan (checklist tester No 24): CRUD seperti supplier, plus ambil dari data lama.

func decodePelanggan(w http.ResponseWriter, r *http.Request) (finance.Pelanggan, bool) {
	var p finance.Pelanggan
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "data pelanggan tidak valid", http.StatusBadRequest)
		return p, false
	}
	if err := finance.ValidatePelanggan(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return p, false
	}
	return p, true
}

func (s *tenant) handleListPelanggan(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeJSON(w, nonNil(slices.Clone(s.pelanggan)))
}

// tambahPelangganLocked: s.mu dipegang.
func (s *tenant) tambahPelangganLocked(p finance.Pelanggan) finance.Pelanggan {
	p.ID = max(s.nextPelangganID, 1)
	s.nextPelangganID = p.ID + 1
	s.pelanggan = append(s.pelanggan, p)
	return p
}

func (s *tenant) handleCreatePelanggan(w http.ResponseWriter, r *http.Request) {
	p, ok := decodePelanggan(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := finance.CekUnik(s.pelanggan, finance.InfoPelanggan, 0, "", p.Nama, "pelanggan"); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, s.tambahPelangganLocked(p))
}

// handlePelangganDariData menambahkan nama pelanggan dari transaksi penjualan & invoice yang
// belum ada di daftar; yang sudah ada dilewati.
func (s *tenant) handlePelangganDariData(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, p := range finance.PelangganDariData(s.txs, s.invoices) {
		if finance.CekUnik(s.pelanggan, finance.InfoPelanggan, 0, "", p.Nama, "pelanggan") == nil {
			s.tambahPelangganLocked(p)
			n++
		}
	}
	writeJSON(w, map[string]int{"ditambah": n})
}

func (s *tenant) pelangganIndex(r *http.Request) int {
	id, _ := strconv.Atoi(r.PathValue("id"))
	return slices.IndexFunc(s.pelanggan, func(x finance.Pelanggan) bool { return x.ID == id })
}

func (s *tenant) handleUpdatePelanggan(w http.ResponseWriter, r *http.Request) {
	p, ok := decodePelanggan(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.pelangganIndex(r)
	if i < 0 {
		http.Error(w, "pelanggan tidak ditemukan", http.StatusNotFound)
		return
	}
	if err := finance.CekUnik(s.pelanggan, finance.InfoPelanggan, s.pelanggan[i].ID, "", p.Nama, "pelanggan"); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	p.ID = s.pelanggan[i].ID
	s.pelanggan[i] = p
	writeJSON(w, p)
}

func (s *tenant) handleDeletePelanggan(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.pelangganIndex(r)
	if i < 0 {
		http.Error(w, "pelanggan tidak ditemukan", http.StatusNotFound)
		return
	}
	s.pelanggan = slices.Delete(s.pelanggan, i, i+1)
	w.WriteHeader(http.StatusNoContent)
}
