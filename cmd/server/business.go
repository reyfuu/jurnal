package main

import (
	"cmp"
	"encoding/json"
	"net/http"
	"os"

	"pajak/internal/finance"
	"pajak/internal/tax"
)

func loadProfile(p string) *finance.BusinessProfile {
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

func saveProfile(p string, prof finance.BusinessProfile) error {
	if p == "" {
		return nil
	}
	return writeFileAtomic(p, prof)
}

// kluOptions: daftar KLU untuk picker onboarding, dengan jenis usaha untuk penyaringan.
var kluOptions = func() []map[string]string {
	klu := tax.BenchmarkAktif()
	out := make([]map[string]string, 0, len(klu))
	for _, k := range klu {
		out = append(out, map[string]string{"kode": k.Kode, "nama": k.Nama, "jenis": tax.JenisKLU(k.Kode)})
	}
	return out
}()

// handleWilayah: kabupaten/kota beserta kecamatannya untuk satu provinsi (combobox lokasi).
func handleWilayah(w http.ResponseWriter, r *http.Request) {
	kota := finance.KotaDi(r.URL.Query().Get("provinsi"))
	if kota == nil {
		http.Error(w, "provinsi tidak dikenal", http.StatusBadRequest)
		return
	}
	writeJSON(w, kota)
}

func (s *tenant) handleGetBusiness(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	prof := s.profile
	s.mu.RUnlock()
	writeJSON(w, map[string]any{
		"profile": prof,
		"options": map[string]any{
			"jenis_usaha":       finance.JenisUsahaOptions,
			"bentuk_usaha":      finance.BentukUsahaOptions,
			"ptkp":              tax.PTKPCodes,
			"metode_persediaan": finance.MetodePersediaanOptions,
			"provinsi":          finance.ProvinsiOptions,
			"klu":               kluOptions,
		},
	})
}

// handleCekAwal: cek benchmark & risiko untuk profil yang belum disimpan (ringkasan onboarding).
func (s *tenant) handleCekAwal(w http.ResponseWriter, r *http.Request) {
	var prof finance.BusinessProfile
	if err := json.NewDecoder(r.Body).Decode(&prof); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	writeJSON(w, prof.CekAwal())
}

func (s *tenant) handlePutBusiness(w http.ResponseWriter, r *http.Request) {
	var prof finance.BusinessProfile
	if err := json.NewDecoder(r.Body).Decode(&prof); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := cmp.Or(prof.Validate(), prof.ValidateWilayah()); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := saveProfile(s.file("business"), prof); err != nil {
		http.Error(w, "gagal menyimpan profil usaha", http.StatusInternalServerError)
		return
	}
	s.mu.Lock()
	if s.profile == nil || s.profile.JenisUsaha != prof.JenisUsaha {
		s.coaTemplateID = 0 // jenis usaha berganti: ikut template terbaru jenis yang baru
	}
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
func (s *tenant) annualized() (omzet, marginBps int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var total, laba int64
	periods := finance.AvailablePeriods()
	for _, p := range periods {
		// Laba bersih sudah dikurangi penyusutan & amortisasi, jadi margin yang dipakai
		// Perencanaan Pajak sama dengan yang tampil di Laporan Keuangan.
		is := finance.BuildPeriodReport(s.txs, s.assets, p.ID, s.openingFor(p.ID)).Income
		total += is.Omzet
		laba += is.NetProfit
	}
	if total <= 0 {
		return 0, 0
	}
	return total * 12 / int64(len(periods)), laba * 10_000 / total
}

func (s *tenant) handleTaxPlanning(w http.ResponseWriter, r *http.Request) {
	var req taxPlanningReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	omzet, margin := s.annualized()
	s.mu.RLock()
	if p := s.profile; omzet == 0 && p != nil && p.OmzetTahunan > 0 {
		// belum ada transaksi: pakai perkiraan dari onboarding
		omzet, margin = p.OmzetTahunan, p.LabaSebelumPajak*10_000/p.OmzetTahunan
	}
	s.mu.RUnlock()
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
