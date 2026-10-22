package main

import (
	"errors"
	"net/http"
	"time"

	"pajak/internal/finance"
)

// Laba rugi & arus kas untuk rentang tanggal bebas (Laporan Keuangan → Dari / Sampai tanggal).

const maksHariRentang = 3 * 366

// rentangLaporan membaca ?dari=&sampai= (YYYY-MM-DD). ok=false bila keduanya kosong.
func rentangLaporan(r *http.Request) (dari, sampai string, ok bool, err error) {
	dari, sampai = r.URL.Query().Get("dari"), r.URL.Query().Get("sampai")
	if dari == "" && sampai == "" {
		return "", "", false, nil
	}
	d, e1 := time.Parse(time.DateOnly, dari)
	s, e2 := time.Parse(time.DateOnly, sampai)
	switch {
	case e1 != nil || e2 != nil:
		return "", "", false, errors.New("tanggal dari & sampai wajib diisi (YYYY-MM-DD)")
	case s.Before(d):
		return "", "", false, errors.New("tanggal sampai tidak boleh sebelum tanggal dari")
	case s.Sub(d) > maksHariRentang*24*time.Hour:
		return "", "", false, errors.New("rentang paling panjang 3 tahun")
	}
	return dari, sampai, true, nil
}

func (s *tenant) laporanRentang(dari, sampai string) finance.LaporanRentang {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return finance.BuildLaporanRentang(s.txs, s.assets, dari, sampai, s.bukuAwalLocked(), s.kartuLocked())
}

func (s *tenant) handleLaporanRentang(w http.ResponseWriter, r *http.Request) {
	dari, sampai, ok, err := rentangLaporan(r)
	if err == nil && !ok {
		err = errors.New("tanggal dari & sampai wajib diisi (YYYY-MM-DD)")
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, s.laporanRentang(dari, sampai))
}
