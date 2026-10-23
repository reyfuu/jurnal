package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"pajak/internal/auth"
	"pajak/internal/finance"
)

// No. Referensi & pembatalan transaksi. Hapus transaksi = batalkan: nomornya tetap tercatat
// (siapa, kapan, alasan) dan tidak dipakai ulang; transaksinya keluar dari laporan.

// beriRefLocked memberi No. Referensi pada transaksi baru yang belum bernomor. s.mu dipegang.
func (s *tenant) beriRefLocked() { finance.BeriRef(s.txs, s.txBatal, s.refMulai) }

// batalkanLocked memindahkan transaksi yang cocok ke daftar batal. s.mu dipegang.
func (s *tenant) batalkanLocked(cocok func(finance.Transaction) bool, asal, oleh, alasan string) {
	waktu := time.Now().In(wib).Format("2006-01-02 15:04")
	s.txs = slices.DeleteFunc(s.txs, func(t finance.Transaction) bool {
		if !cocok(t) {
			return false
		}
		t.Batal = &finance.Pembatalan{Oleh: oleh, Waktu: waktu, Alasan: alasan, Asal: asal}
		s.txBatal = append(s.txBatal, t)
		return true
	})
}

var wib = func() *time.Location {
	if l, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return l
	}
	return time.FixedZone("WIB", 7*3600)
}()

// alasanBatal: alasan pembatalan dari body {"alasan": "..."} atau ?alasan=; wajib diisi.
func alasanBatal(r *http.Request) (string, error) {
	var b struct {
		Alasan string `json:"alasan"`
	}
	_ = json.NewDecoder(r.Body).Decode(&b)
	a := strings.TrimSpace(b.Alasan)
	if a == "" {
		a = strings.TrimSpace(r.URL.Query().Get("alasan"))
	}
	if a == "" {
		return "", errors.New("isi alasan pembatalan transaksi")
	}
	if len(a) > 300 {
		return "", errors.New("alasan pembatalan paling panjang 300 huruf")
	}
	return a, nil
}

// handleTransaksiBatal: daftar transaksi yang dibatalkan (untuk filter "Tampilkan yang batal").
func (s *tenant) handleTransaksiBatal(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	out := slices.Clone(s.txBatal)
	s.mu.RUnlock()
	writeJSON(w, nonNil(out))
}

// handlePulihkanTransaksi: Owner memulihkan transaksi yang dibatalkan (satu nomor sekaligus),
// dengan nomor yang sama. Pembatalan karena invoice / aset / produksi kembali lewat dokumennya.
func (s *tenant) handlePulihkanTransaksi(w http.ResponseWriter, r *http.Request) {
	if currentUser(r).Role != auth.Owner {
		http.Error(w, "hanya Owner yang bisa memulihkan transaksi yang dibatalkan", http.StatusForbidden)
		return
	}
	id, _ := strconv.Atoi(r.PathValue("id"))
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.txBatal, func(t finance.Transaction) bool { return t.ID == id })
	if i < 0 {
		http.Error(w, "transaksi batal tidak ditemukan", http.StatusNotFound)
		return
	}
	x := s.txBatal[i]
	if !x.Batal.BisaDipulihkan() {
		http.Error(w, "transaksi ini dibatalkan bersama "+x.Batal.Asal+"-nya; pulihkan lewat menu itu", http.StatusConflict)
		return
	}
	sama := func(t finance.Transaction) bool {
		if x.Ref != "" {
			return t.Ref == x.Ref
		}
		return t.ID == x.ID || (x.Grup != 0 && t.Grup == x.Grup)
	}
	for _, t := range s.txBatal {
		if sama(t) {
			if err := s.cekTerbukaLocked(tanggalTx(t)...); err != nil {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
		}
	}
	var pulih []finance.Transaction
	s.txBatal = slices.DeleteFunc(s.txBatal, func(t finance.Transaction) bool {
		if !sama(t) {
			return false
		}
		t.Batal = nil
		pulih = append(pulih, t)
		return true
	})
	s.txs = append(s.txs, pulih...)
	writeJSON(w, pulih)
}
