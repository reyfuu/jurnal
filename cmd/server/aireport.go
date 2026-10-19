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
		snap = finance.EmptySnapshot(period)
	}
	invoices := append([]finance.Invoice(nil), t.invoices...)
	karyawan := append([]finance.Karyawan(nil), t.karyawan...)
	txs := append([]finance.Transaction(nil), t.txs...)
	assets := append([]finance.Asset(nil), t.assets...)
	profile := finance.BusinessProfile{}
	if t.profile != nil {
		profile = *t.profile
	}
	t.mu.RUnlock()

	riwayat := finance.TrendHistory(txs, assets, period, t.openingFor)
	facts := finance.CollectFacts(rep, snap, profile, invoices, karyawan, riwayat)

	writeJSON(w, map[string]any{
		"facts":    facts,
		"analysis": ai.Analyze(r.Context(), s.llm, facts),
		"llm":      ai.Status(s.llm),
	})
}
