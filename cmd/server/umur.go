package main

import (
	"cmp"
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"time"

	"pajak/internal/finance"
)

// Umur piutang & utang dari transaksi yang belum lunas (+ saldo awal) — totalnya sama dengan
// piutang/utang di Neraca berjalan. Pelunasan dicatat dengan tanggal bayar sehingga kas masuk
// ke bulan pembayaran.

func (s *tenant) handleUmurSaldo(w http.ResponseWriter, r *http.Request) {
	per := cmp.Or(r.URL.Query().Get("per"), time.Now().In(finance.WIB).Format(time.DateOnly))
	if !tanggalSah(per) {
		http.Error(w, "tanggal tidak valid", http.StatusBadRequest)
		return
	}
	piutang := r.URL.Query().Get("jenis") != "utang"
	s.mu.RLock()
	out := finance.UmurSaldo(s.bukuAwalLocked(), s.txs, per, piutang)
	s.mu.RUnlock()
	writeJSON(w, out)
}

// handleLunasi: tandai transaksi lunas pada tanggal bayar (pelunasan piutang / pembayaran utang).
func (s *tenant) handleLunasi(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Tanggal string `json:"tanggal"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || !tanggalSah(b.Tanggal) {
		http.Error(w, "tanggal pelunasan wajib diisi (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	id, _ := strconv.Atoi(r.PathValue("id"))
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.txs, func(t finance.Transaction) bool { return t.ID == id })
	if i < 0 {
		http.Error(w, "transaksi tidak ditemukan", http.StatusNotFound)
		return
	}
	t := s.txs[i]
	if err := s.cekTerbukaLocked(b.Tanggal); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if t.Lunas {
		http.Error(w, "transaksi ini sudah lunas", http.StatusConflict)
		return
	}
	if b.Tanggal < t.Tanggal {
		http.Error(w, "tanggal pelunasan tidak boleh sebelum tanggal transaksi", http.StatusBadRequest)
		return
	}
	t.Lunas = true
	if b.Tanggal != t.Tanggal {
		t.TanggalLunas = b.Tanggal
	}
	s.txs[i] = t
	writeJSON(w, s.withWarnings(t))
}

// tagihanBukuLocked: piutang & utang belum lunas per tanggal dari buku besar. Pemanggil memegang s.mu.
func (s *tenant) tagihanBukuLocked(per string) []finance.Tagihan {
	ob := s.bukuAwalLocked()
	return finance.TagihanDariBuku(finance.UmurSaldo(ob, s.txs, per, true), finance.UmurSaldo(ob, s.txs, per, false))
}

func (s *Server) umurRoutes() {
	s.mux.HandleFunc("GET /api/v1/reports/umur", s.guard("umur-piutang", s.on((*tenant).handleUmurSaldo)))
	s.mux.HandleFunc("POST /api/v1/transactions/{id}/lunas", s.persisting(s.guard("transaksi-kelola", s.on((*tenant).handleLunasi))))
}
