package main

import (
	"net/http"
	"slices"
	"strings"

	"pajak/internal/finance"
)

// Riwayat per pelanggan / supplier (checklist tester No 25). Dicocokkan lewat nama pihak di
// transaksi (invoice yang dibukukan juga menjadi transaksi berpihak pelanggannya); huruf
// besar/kecil dan spasi ganda diabaikan, sama dengan aturan nama unik daftar master.

type riwayatPihak struct {
	Nama           string                `json:"nama"`
	Dari           string                `json:"dari"`
	Sampai         string                `json:"sampai"`
	Transaksi      []finance.Transaction `json:"transaksi"`
	TotalMasuk     int64                 `json:"total_masuk"`
	TotalKeluar    int64                 `json:"total_keluar"`
	PiutangTerbuka int64                 `json:"piutang_terbuka"` // semua tanggal, belum lunas
	UtangTerbuka   int64                 `json:"utang_terbuka"`
}

func (s *tenant) handleRiwayatPihak(w http.ResponseWriter, r *http.Request) {
	nama := strings.TrimSpace(r.URL.Query().Get("nama"))
	if nama == "" {
		http.Error(w, "nama pelanggan / supplier wajib diisi", http.StatusBadRequest)
		return
	}
	dari, sampai, err := rentangDari(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	k := finance.KunciNama(nama)
	out := riwayatPihak{Nama: nama, Dari: dari, Sampai: sampai, Transaksi: []finance.Transaction{}}
	s.mu.RLock()
	for _, t := range s.txs {
		if finance.KunciNama(t.Pihak) != k {
			continue
		}
		j, _ := finance.LookupJenis(t.Jenis)
		masuk := j.Arah == "masuk"
		if !t.Lunas && masuk {
			out.PiutangTerbuka += t.Nominal
		} else if !t.Lunas {
			out.UtangTerbuka += t.Nominal
		}
		if t.Tanggal < dari || t.Tanggal > sampai {
			continue
		}
		out.Transaksi = append(out.Transaksi, t)
		if masuk {
			out.TotalMasuk += t.Nominal
		} else {
			out.TotalKeluar += t.Nominal
		}
	}
	s.mu.RUnlock()
	slices.SortStableFunc(out.Transaksi, func(a, b finance.Transaction) int { return strings.Compare(a.Tanggal, b.Tanggal) })
	writeJSON(w, out)
}
