package main

import (
	"encoding/json"
	"net/http"
	"os"

	"pajak/internal/finance"
	"pajak/internal/tax"
)

func profilePath() string { return dataPath("business.json") }

func loadProfile() *finance.BusinessProfile {
	p := profilePath()
	if p == "" {
		return nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var prof finance.BusinessProfile
	if json.Unmarshal(b, &prof) != nil || prof.Validate() != nil {
		return nil
	}
	return &prof
}

func saveProfile(prof finance.BusinessProfile) error {
	p := profilePath()
	if p == "" {
		return nil
	}
	return writeFileAtomic(p, prof)
}

func (s *Server) handleGetBusiness(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	prof := s.profile
	s.mu.RUnlock()
	writeJSON(w, map[string]any{
		"profile": prof,
		"options": map[string]any{
			"jenis_usaha":  finance.JenisUsahaOptions,
			"bentuk_usaha": finance.BentukUsahaOptions,
			"ptkp":         tax.PTKPCodes,
		},
	})
}

func (s *Server) handlePutBusiness(w http.ResponseWriter, r *http.Request) {
	var prof finance.BusinessProfile
	if err := json.NewDecoder(r.Body).Decode(&prof); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := prof.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := saveProfile(prof); err != nil {
		http.Error(w, "gagal menyimpan profil usaha", http.StatusInternalServerError)
		return
	}
	s.mu.Lock()
	s.profile = &prof
	s.mu.Unlock()
	writeJSON(w, prof)
}

type taxPlanningReq struct {
	Omzet        int64  `json:"omzet"`          // per tahun; 0 = proyeksi dari data
	NetMarginBps int64  `json:"net_margin_bps"` // 0 = dari laba rugi
	PTKP         string `json:"ptkp"`           // kosong = dari profil usaha
}

// annualized: omzet setahun = rata-rata omzet bulanan semua periode x 12, margin = laba bersih / omzet.
func (s *Server) annualized() (omzet, marginBps int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var total, laba int64
	periods := finance.AvailablePeriods()
	for _, p := range periods {
		is := finance.BuildIncomeStatement(finance.ToLedger(finance.ForPeriod(s.txs, p.ID)))
		total += is.Omzet
		laba += is.NetProfit
	}
	if total <= 0 {
		return 0, 0
	}
	return total * 12 / int64(len(periods)), laba * 10_000 / total
}

func (s *Server) handleTaxPlanning(w http.ResponseWriter, r *http.Request) {
	var req taxPlanningReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	omzet, margin := s.annualized()
	if req.Omzet <= 0 {
		req.Omzet = omzet
	}
	if req.NetMarginBps == 0 {
		req.NetMarginBps = margin
	}
	if req.PTKP == "" {
		s.mu.RLock()
		if s.profile != nil {
			req.PTKP = s.profile.PTKP
		}
		s.mu.RUnlock()
	}
	if req.PTKP == "" {
		req.PTKP = "TK/0"
	}
	bu, err := tax.CompareBentukUsaha(req.Omzet, req.NetMarginBps, req.PTKP)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{
		"input":        req,
		"bentuk_usaha": bu,
		"entitas":      tax.CompareEntitas(req.Omzet, 10_000-req.NetMarginBps),
	})
}
