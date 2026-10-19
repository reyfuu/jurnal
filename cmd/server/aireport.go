package main

import (
	"net/http"

	"pajak/internal/ai"
	"pajak/internal/finance"
)

// handleAIReport menyajikan seluruh angka laporan satu periode plus analisis model.
//
// Fakta dan analisis dikirim bersama, dari satu sumber yang sama, supaya yang dibaca model
// dan yang dilihat pengguna tidak mungkin berbeda. Tanpa LLM_API_KEY fakta tetap lengkap dan
// hanya bagian analisis yang kosong.
func (s *Server) handleAIReport(w http.ResponseWriter, r *http.Request) {
	period := periodOrDefault(r)
	rep := s.report(period)

	s.mu.RLock()
	snap, ok := s.snapshots[period]
	if !ok {
		snap = s.snapshots["2026-08"]
	}
	invoices := append([]finance.Invoice(nil), s.invoices...)
	karyawan := append([]finance.Karyawan(nil), s.karyawan...)
	txs := append([]finance.Transaction(nil), s.txs...)
	assets := append([]finance.Asset(nil), s.assets...)
	s.mu.RUnlock()

	riwayat := finance.TrendHistory(txs, assets, period, s.openingFor)
	facts := finance.CollectFacts(rep, snap, s.currentProfile(), invoices, karyawan, riwayat)

	writeJSON(w, map[string]any{
		"facts":    facts,
		"analysis": ai.Analyze(r.Context(), s.llm, facts),
		"llm":      ai.Status(s.llm),
	})
}
