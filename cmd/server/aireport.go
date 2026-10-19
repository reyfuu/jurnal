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
func (s *Server) handleAIReport(t *tenant, w http.ResponseWriter, r *http.Request) {
	period := periodOrDefault(r)
	rep := t.report(period)

	t.mu.RLock()
	snap, ok := t.snapshots[period]
	if !ok {
		snap = t.snapshots["2026-08"]
	}
	invoices := append([]finance.Invoice(nil), t.invoices...)
	karyawan := append([]finance.Karyawan(nil), t.karyawan...)
	riwayat := finance.TrendHistory(t.txs, t.assets, period, t.openingFor)
	t.mu.RUnlock()

	facts := finance.CollectFacts(rep, snap, t.currentProfile(), invoices, karyawan, riwayat)

	writeJSON(w, map[string]any{
		"facts":    facts,
		"analysis": ai.Analyze(r.Context(), s.llm, facts),
		"llm":      ai.Status(s.llm),
	})
}
