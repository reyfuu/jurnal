package main

import (
	"cmp"
	"encoding/json"
	"errors"
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
	t, kode, err := s.lunasiTxLocked(id, b.Tanggal)
	if err != nil {
		http.Error(w, err.Error(), kode)
		return
	}
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

// lunasiTxLocked menandai transaksi belum lunas sebagai lunas pada tanggal; transaksi invoice
// melunasi invoice-nya (DPP & PPN bersama). kode = status HTTP bila gagal. s.mu dipegang.
func (s *tenant) lunasiTxLocked(id int, tanggal string) (finance.Transaction, int, error) {
	i := slices.IndexFunc(s.txs, func(t finance.Transaction) bool { return t.ID == id })
	if i < 0 {
		return finance.Transaction{}, http.StatusNotFound, errors.New("transaksi tidak ditemukan")
	}
	t := s.txs[i]
	if err := s.cekTerbukaLocked(tanggal); err != nil {
		return t, http.StatusConflict, err
	}
	if t.Lunas {
		return t, http.StatusConflict, errors.New("transaksi ini sudah lunas")
	}
	if tanggal < t.Tanggal {
		return t, http.StatusBadRequest, errors.New("tanggal pelunasan tidak boleh sebelum tanggal transaksi")
	}
	if t.InvoiceID != 0 {
		// Piutang dari invoice: invoice ikut lunas, transaksi DPP & PPN-nya lunas bersama.
		if err := s.lunasiInvoiceLocked(t.InvoiceID, tanggal); err != nil {
			return t, http.StatusConflict, err
		}
		if j := slices.IndexFunc(s.txs, func(x finance.Transaction) bool { return x.ID == id }); j >= 0 {
			return s.txs[j], 0, nil
		}
	}
	t.Lunas = true
	s.pakaiRekeningBawaanLocked(&t, "") // dilunasi lewat Bank kecuali rekeningnya sudah dipilih
	if tanggal != t.Tanggal {
		t.TanggalLunas = tanggal
	}
	s.txs[i] = t
	return t, 0, nil
}
