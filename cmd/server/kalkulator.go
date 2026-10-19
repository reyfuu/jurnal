package main

import (
	"encoding/json"
	"net/http"

	"pajak/internal/tax"
)

// handleKalkulatorMaster menyajikan seluruh jenis & objek pajak yang bisa dihitung, supaya
// daftar tarif hanya hidup di satu tempat (internal/tax) dan frontend tidak menyalinnya.
func (s *Server) handleKalkulatorMaster(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{
		"jenis":      tax.JenisPajakList,
		"ptkp_codes": tax.PTKPCodes,
		"ptkp":       tax.PTKP,
		"ter": map[string]any{
			"kategori":      tax.TERKategori,
			"lapisan":       tax.TERLapisan,
			"belum_lengkap": tax.TERBelumLengkap(),
		},
		"tarif_badan_bps": tax.TarifBadanBps,
	})
}

func (s *Server) handleHitungPPh21(w http.ResponseWriter, r *http.Request) {
	var in tax.PPh21Input
	if !decodeBody(w, r, &in) {
		return
	}
	res, err := tax.HitungPPh21(in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// TER disajikan berdampingan bila lapisannya sudah terisi, supaya pengguna bisa
	// membandingkan; bila belum, alasannya dinyatakan apa adanya.
	out := map[string]any{"hasil": res}
	if bps, pph, err := tax.HitungTER(in.PenghasilanBruto, in.PTKPStatus); err == nil {
		out["ter"] = map[string]any{"tersedia": true, "tarif_bps": bps, "pph_sebulan": pph}
	} else {
		out["ter"] = map[string]any{"tersedia": false, "alasan": err.Error()}
	}
	writeJSON(w, out)
}

func (s *Server) handleHitungPotong(w http.ResponseWriter, r *http.Request) {
	var in tax.PotongInput
	if !decodeBody(w, r, &in) {
		return
	}
	res, err := tax.HitungPotong(in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, res)
}

func (s *Server) handleHitungPPhBadan(w http.ResponseWriter, r *http.Request) {
	var in tax.PPhBadanInput
	if !decodeBody(w, r, &in) {
		return
	}
	res, err := tax.HitungPPhBadan(in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, res)
}

func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		http.Error(w, "data perhitungan tidak valid", http.StatusBadRequest)
		return false
	}
	return true
}
