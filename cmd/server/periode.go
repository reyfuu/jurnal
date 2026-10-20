package main

import (
	"net/http"

	"pajak/internal/finance"
)

// Periode laporan per usaha: dari data paling awal sampai bulan berjalan (finance.DaftarPeriode).

// periodeLocked: s.mu harus terkunci (baca).
func (s *tenant) periodeLocked() []finance.PilihanPeriode {
	var ekstra []string
	if s.opening != nil {
		ekstra = append(ekstra, s.opening.Period())
	}
	if k := s.kartuLocked(); k.Mulai != "" {
		ekstra = append(ekstra, k.Mulai[:7])
	}
	if s.demo {
		for _, p := range finance.AvailablePeriods() {
			ekstra = append(ekstra, p.ID)
		}
	}
	return finance.DaftarPeriode(s.txs, ekstra, finance.BulanBerjalan())
}

// snapshotLocked: kerangka dashboard periode itu; periode tanpa kerangka (di luar data contoh)
// memakai kerangka kosong — angka diisi dari transaksi oleh ApplyKPIs, bukan dari periode lain.
func (s *tenant) snapshotLocked(period string) finance.DashboardSnapshot {
	if snap, ok := s.snapshots[period]; ok {
		return snap
	}
	snap := finance.EmptySnapshot(period)
	for _, x := range s.snapshots {
		snap.Tenant = x.Tenant // nama usaha sama untuk semua periode
		break
	}
	return snap
}

func (s *Server) handlePeriods(w http.ResponseWriter, r *http.Request) {
	t := s.tenantOf(r)
	if t == nil {
		writeJSON(w, finance.DaftarPeriode(nil, nil, finance.BulanBerjalan()))
		return
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	writeJSON(w, t.periodeLocked())
}
