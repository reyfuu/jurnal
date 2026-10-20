package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"pajak/internal/ai"
	"pajak/internal/tax"
)

// maxSaranKLUAI: pemakaian AI untuk saran bidang usaha per user per 24 jam; lewat dari itu
// tetap dijawab dengan pencarian kata (tanpa biaya model).
const maxSaranKLUAI = 20

func (s *Server) jatahSaranKLU(userID int) bool {
	return s.jatahAI("klu|"+strconv.Itoa(userID), maxSaranKLUAI)
}

// jatahAI: memotong satu jatah pemakaian AI untuk kunci dalam 24 jam terakhir; false = habis.
// ponytail: di memori — restart mengosongkan hitungan; cukup untuk mengerem biaya model.
func (s *Server) jatahAI(kunci string, maks int) bool {
	s.kluMu.Lock()
	defer s.kluMu.Unlock()
	if s.kuotaAI == nil {
		s.kuotaAI = map[string][]time.Time{}
	}
	batas := time.Now().Add(-24 * time.Hour)
	pakai := slices.DeleteFunc(s.kuotaAI[kunci], func(t time.Time) bool { return t.Before(batas) })
	if len(pakai) >= maks {
		s.kuotaAI[kunci] = pakai
		return false
	}
	s.kuotaAI[kunci] = append(pakai, time.Now())
	return true
}

// handleSaranKLU: {deskripsi, jenis} → 1–3 saran KLU. Dengan AI bila aktif & kuota ada; selain
// itu (atau bila AI gagal) hasil pencarian kata tax.CariKLU.
func (s *Server) handleSaranKLU(w http.ResponseWriter, r *http.Request) {
	var req struct{ Deskripsi, Jenis string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "data tidak valid", http.StatusBadRequest)
		return
	}
	req.Deskripsi = strings.TrimSpace(req.Deskripsi)
	if n := utf8.RuneCountInString(req.Deskripsi); n < 3 || n > 300 {
		http.Error(w, "ceritakan usaha Anda dalam 3–300 huruf", http.StatusBadRequest)
		return
	}
	out := struct {
		AI    bool          `json:"ai"`
		Model string        `json:"model,omitempty"`
		Saran []ai.SaranKLU `json:"saran"`
	}{Saran: []ai.SaranKLU{}}
	if s.llm != nil && s.llm.Enabled() && s.jatahSaranKLU(currentUser(r).ID) {
		if saran, err := ai.PilihKLU(r.Context(), s.llm, req.Deskripsi, req.Jenis); err == nil {
			out.AI, out.Model, out.Saran = true, s.llm.Model(), saran
		}
	}
	if !out.AI {
		for _, k := range tax.CariKLU(req.Deskripsi, req.Jenis, 3) {
			out.Saran = append(out.Saran, ai.SaranKLU{Kode: k.Kode, Nama: k.Nama, Jenis: k.Jenis, Contoh: k.Contoh})
		}
	}
	writeJSON(w, out)
}
