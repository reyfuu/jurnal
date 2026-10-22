package main

import (
	"net/http"
	"strconv"

	"pajak/internal/finance"
)

// Cegah jurnal dobel (lihat finance.CariMirip): pengecekan sebelum menyimpan di halaman Catat
// transaksi, dan daftar pasangan transaksi yang kemungkinan dobel per periode.

// handleCariMirip: ?tanggal=&nominal=&arah=masuk|keluar[&kecuali_grup=N] → transaksi mirip.
func (s *tenant) handleCariMirip(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	nominal, _ := strconv.ParseInt(q.Get("nominal"), 10, 64)
	arah := q.Get("arah")
	if !tanggalSah(q.Get("tanggal")) || nominal <= 0 || (arah != "masuk" && arah != "keluar") {
		http.Error(w, "isi tanggal, nominal, dan arah (masuk/keluar)", http.StatusBadRequest)
		return
	}
	grup, _ := strconv.Atoi(q.Get("kecuali_grup"))
	s.mu.RLock()
	kecuali := map[int]bool{}
	for _, t := range s.txs {
		if grup > 0 && t.Grup == grup {
			kecuali[t.ID] = true
		}
	}
	m := finance.CariMirip(s.txs, q.Get("tanggal"), nominal, arah, kecuali)
	s.mu.RUnlock()
	writeJSON(w, nonNil(m))
}

// handleDobel: pasangan transaksi lunas di periode yang kemungkinan tercatat dua kali.
func (s *tenant) handleDobel(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	p := finance.PasanganMirip(s.txs, periodOrDefault(r))
	s.mu.RUnlock()
	writeJSON(w, nonNil(p))
}
