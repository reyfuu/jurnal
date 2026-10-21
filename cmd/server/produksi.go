package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"pajak/internal/finance"
)

// Produksi & Supplier: resep per produk, HPP hasil hitung, dan daftar supplier.
//
// HPP dihitung di sini setiap kali daftar dibaca, bukan disimpan bersama produk. Harga bahan
// di Stok & Gudang bisa berubah kapan saja; HPP yang disimpan akan diam-diam jadi basi dan
// tetap dipakai menetapkan harga jual.

func (s *tenant) handleListProduksi(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	produk := slices.Clone(s.produk)
	supplier := slices.Clone(s.supplier)
	// Setelah titik nol, harga bahan & HPP di stok memakai nilai kartu stok (FIFO / rata-rata).
	stok := finance.TerapkanKartu(slices.Clone(s.stok), s.kartuLocked())
	// Utang per supplier dari buku besar (pembelian belum dibayar), sama dengan Neraca.
	tagihan := s.tagihanBukuLocked(time.Now().In(finance.WIB).Format(time.DateOnly))
	s.mu.RUnlock()
	writeJSON(w, finance.SummarizeProduksi(produk, supplier, stok, tagihan))
}

func decodeProduk(w http.ResponseWriter, r *http.Request) (finance.Produk, bool) {
	var p finance.Produk
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "data produk tidak valid", http.StatusBadRequest)
		return p, false
	}
	if err := finance.ValidateProduk(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return p, false
	}
	return p, true
}

func (s *tenant) handleCreateProduk(w http.ResponseWriter, r *http.Request) {
	p, ok := decodeProduk(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	if s.nextProdukID == 0 {
		s.nextProdukID = 1
	}
	p.ID = s.nextProdukID
	s.nextProdukID++
	s.produk = append(s.produk, p)
	s.mu.Unlock()
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, p)
}

// produkIndex harus dipanggil dengan s.mu terkunci.
func (s *tenant) produkIndex(r *http.Request) int {
	id, _ := strconv.Atoi(r.PathValue("id"))
	return slices.IndexFunc(s.produk, func(x finance.Produk) bool { return x.ID == id })
}

func (s *tenant) handleUpdateProduk(w http.ResponseWriter, r *http.Request) {
	p, ok := decodeProduk(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.produkIndex(r)
	if i < 0 {
		http.Error(w, "produk tidak ditemukan", http.StatusNotFound)
		return
	}
	p.ID = s.produk[i].ID
	s.produk[i] = p
	writeJSON(w, p)
}

func (s *tenant) handleDeleteProduk(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.produkIndex(r)
	if i < 0 {
		http.Error(w, "produk tidak ditemukan", http.StatusNotFound)
		return
	}
	s.produk = slices.Delete(s.produk, i, i+1)
	w.WriteHeader(http.StatusNoContent)
}

// handleTerapkanHPP menyalin HPP hasil resep ke harga pokok barang di Stok & Gudang.
//
// Disengaja sebagai aksi terpisah, bukan otomatis: harga pokok di stok ikut menentukan nilai
// persediaan di neraca, jadi perubahannya harus keputusan pengguna, bukan efek samping dari
// menyunting resep.
func (s *tenant) handleTerapkanHPP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.produkIndex(r)
	if i < 0 {
		http.Error(w, "produk tidak ditemukan", http.StatusNotFound)
		return
	}
	p := s.produk[i]
	if p.SKU == "" {
		http.Error(w, "produk belum punya SKU, jadi barangnya di Stok & Gudang tidak bisa dicocokkan", http.StatusBadRequest)
		return
	}
	hpp := finance.HitungHPP(p, s.stok)
	j := slices.IndexFunc(s.stok, func(x finance.StokItem) bool { return strings.EqualFold(x.SKU, p.SKU) })
	if j < 0 {
		http.Error(w, "SKU "+p.SKU+" belum ada di Stok & Gudang; tambahkan barangnya dulu", http.StatusBadRequest)
		return
	}
	s.stok[j].HargaPokok = hpp.PerUnit
	writeJSON(w, map[string]any{"sku": p.SKU, "harga_pokok": hpp.PerUnit})
}

// --- Supplier ---

func decodeSupplier(w http.ResponseWriter, r *http.Request) (finance.Supplier, bool) {
	var sp finance.Supplier
	if err := json.NewDecoder(r.Body).Decode(&sp); err != nil {
		http.Error(w, "data supplier tidak valid", http.StatusBadRequest)
		return sp, false
	}
	if err := finance.ValidateSupplier(&sp); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return sp, false
	}
	return sp, true
}

func (s *tenant) handleCreateSupplier(w http.ResponseWriter, r *http.Request) {
	sp, ok := decodeSupplier(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	if s.nextSupplierID == 0 {
		s.nextSupplierID = 1
	}
	sp.ID = s.nextSupplierID
	s.nextSupplierID++
	s.supplier = append(s.supplier, sp)
	s.mu.Unlock()
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, sp)
}

func (s *tenant) supplierIndex(r *http.Request) int {
	id, _ := strconv.Atoi(r.PathValue("id"))
	return slices.IndexFunc(s.supplier, func(x finance.Supplier) bool { return x.ID == id })
}

func (s *tenant) handleUpdateSupplier(w http.ResponseWriter, r *http.Request) {
	sp, ok := decodeSupplier(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.supplierIndex(r)
	if i < 0 {
		http.Error(w, "supplier tidak ditemukan", http.StatusNotFound)
		return
	}
	sp.ID = s.supplier[i].ID
	s.supplier[i] = sp
	writeJSON(w, sp)
}

func (s *tenant) handleDeleteSupplier(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.supplierIndex(r)
	if i < 0 {
		http.Error(w, "supplier tidak ditemukan", http.StatusNotFound)
		return
	}
	s.supplier = slices.Delete(s.supplier, i, i+1)
	w.WriteHeader(http.StatusNoContent)
}
