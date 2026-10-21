package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"pajak/internal/ai"
	"pajak/internal/coa"
	"pajak/internal/finance"
)

// Saran akun saldo awal: begitu pengguna mengetik nama akun, layar menerima kategori neraca dan
// kode akunnya. Urutan: akun CoA usaha yang namanya cocok → AI (bila aktif & kuota ada) →
// kata kunci. Kode baru diberikan bila barisnya bukan akun CoA yang sudah ada.

type saranSaldoReq struct {
	Nama     string   `json:"nama"`
	Kategori string   `json:"kategori"` // diisi = kategori dipilih sendiri; cukup cari kodenya
	Dipakai  []string `json:"dipakai"`  // kode yang sudah ada di layar
}

type saranSaldo struct {
	Kategori string `json:"kategori"`
	Kode     string `json:"kode"`
	Akun     string `json:"akun,omitempty"` // nama akun CoA yang cocok
	Baru     bool   `json:"baru"`           // kode baru, belum ada di CoA
	Sumber   string `json:"sumber"`         // coa | ai | aturan | pilihan | ""
	Alasan   string `json:"alasan,omitempty"`
}

// maxSaranSaldoAI: panggilan AI saran kategori per usaha per 24 jam.
const maxSaranSaldoAI = 200

func (s *Server) handleSaranSaldo(w http.ResponseWriter, r *http.Request) {
	var req saranSaldoReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "data tidak valid", http.StatusBadRequest)
		return
	}
	req.Nama = strings.TrimSpace(req.Nama)
	if utf8.RuneCountInString(req.Nama) > 120 || len(req.Dipakai) > 1000 {
		http.Error(w, "nama akun terlalu panjang", http.StatusBadRequest)
		return
	}
	t := s.tenantOf(r)
	all, active, p := t.akunSaldoAwal()
	peta := coa.PetaDari(all)
	out := saranSaldo{}

	if k, ok := finance.KategoriSaldoDari(req.Kategori); ok && req.Kategori != "" {
		out.Kategori, out.Sumber = k.ID, "pilihan"
	} else if req.Nama != "" {
		if a, persis, ok := coa.CocokAkunSaldo(req.Nama, active, peta); ok {
			out.Kategori, out.Akun, out.Sumber = coa.KategoriSaldoAkun(a, peta), a.Nama, "coa"
			if persis {
				out.Kode = a.Kode
			}
		}
		if out.Kategori == "" && s.llm != nil && s.llm.Enabled() && s.jatahAI("saldo|"+strconv.Itoa(t.id), maxSaranSaldoAI) {
			usaha := strings.TrimSpace(p.JenisUsaha + " · " + p.NamaUsaha)
			if kat, alasan, err := ai.SaranKategoriSaldo(r.Context(), s.llm, usaha, req.Nama); err == nil && kat != "" {
				out.Kategori, out.Alasan, out.Sumber = kat, alasan, "ai"
			}
		}
		if out.Kategori == "" {
			if k := coa.TebakKategoriNama(req.Nama); k != "" {
				out.Kategori, out.Sumber = k, "aturan"
			}
		}
	}
	if out.Kategori != "" && out.Kode == "" && out.Kategori != finance.KatAsetTetap {
		out.Kode, out.Baru = coa.KodeSaldo(out.Kategori, active, peta, req.Dipakai)
	}
	writeJSON(w, out)
}

// akunSaldoAwal: CoA usaha untuk layar Saldo Awal; usaha yang belum memilih jenis (belum punya
// template) memakai CoA dagang bawaan. Jangan dipanggil dengan s.mu terkunci (coaTemplate
// mengunci sendiri).
func (s *tenant) akunSaldoAwal() (all, active []coa.Account, p finance.BusinessProfile) {
	all, active, p = s.coaAkun()
	if len(all) == 0 {
		all, active = coa.Master, coa.ActiveFrom(coa.Master, p)
	}
	return all, active, p
}
