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
	// Belum ada transaksi (baru daftar / belum berlangganan): satu-satunya data adalah perkiraan
	// form sign-up, jadi analisis memakai profil — terbatas, dan halaman menyebutnya.
	t.mu.RLock()
	kosong := len(t.txs) == 0
	t.mu.RUnlock()
	if p := t.currentProfile(); kosong && p.NamaUsaha != "" {
		f := ai.FaktaDariProfil(p)
		writeJSON(w, map[string]any{
			"sumber":       "profil",
			"profil":       f,
			"profil_usaha": p,
			"analysis":     ai.AnalisaProfil(r.Context(), s.llm, f),
			"llm":          ai.Status(s.llm),
		})
		return
	}
	rep := t.report(period)

	t.mu.RLock()
	snap := t.snapshotLocked(period)
	invoices := append([]finance.Invoice(nil), t.invoices...)
	karyawan := append([]finance.Karyawan(nil), t.karyawan...)
	riwayat := finance.TrendHistory(t.txs, t.assets, period, t.openingFor, t.kartuLocked().Periode)
	t.mu.RUnlock()

	facts := finance.CollectFacts(rep, snap, t.currentProfile(), invoices, karyawan, riwayat)

	writeJSON(w, map[string]any{
		"sumber":   "transaksi",
		"facts":    facts,
		"analysis": ai.Analyze(r.Context(), s.llm, facts),
		"llm":      ai.Status(s.llm),
	})
}
