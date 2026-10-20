package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"slices"
	"strconv"
	"sync/atomic"
	"time"

	"pajak/internal/coa"
	"pajak/internal/finance"
	"pajak/internal/katalog"
	"pajak/internal/master"
	"pajak/internal/tax"
	"pajak/internal/xlsx"
)

// Katalog Panel Master: template COA per jenis usaha & benchmark NPM nasional, berversi. Klien
// dipatok ke versi template-nya (tenant.coaTemplate) supaya versi baru tidak mengubah pembukuan
// yang sudah berjalan; Owner memilih kapan memperbarui (POST /coa/perbarui).

// katalogAktif: salinan katalog yang dibaca tenant (resolusi COA) tanpa mengunci Server.
// ponytail: satu server per proses, sama seperti benchmark aktif di internal/tax.
var katalogAktif atomic.Pointer[katalog.State]

// pasangKatalog memasang st sebagai katalog aktif aplikasi (COA tenant & benchmark nasional).
func pasangKatalog(st katalog.State) {
	katalogAktif.Store(&st)
	b := st.BenchmarkAktif()
	tax.AktifkanBenchmark(b.Baris, b.Sumber)
	tax.AktifkanParameter(st.ParameterAktif().Isi)
}

// segarkanBawaan: versi 1 (bawaan) selalu mengikuti kode, jadi perbaikan coa.Master / data DJP
// di kode tetap berlaku sampai tim menerbitkan versi sendiri.
func segarkanBawaan(st *katalog.State) {
	baku := katalog.NewState()
	for i := range st.Template {
		if st.Template[i].ID == 1 {
			st.Template[i].Akun = baku.Template[0].Akun
		}
	}
	for i := range st.Benchmark {
		if st.Benchmark[i].ID == 1 {
			st.Benchmark[i].Baris, st.Benchmark[i].Sumber = baku.Benchmark[0].Baris, baku.Benchmark[0].Sumber
		}
	}
	for i := range st.Parameter {
		if st.Parameter[i].ID == 1 {
			st.Parameter[i].Isi, st.Parameter[i].Dasar = baku.Parameter[0].Isi, baku.Parameter[0].Dasar
		}
	}
}

func katalogFile() string { return dataPath("katalog.json") }

func loadKatalogFile() katalog.State {
	st := katalog.NewState()
	if p := katalogFile(); p != "" {
		if raw, err := os.ReadFile(p); err == nil {
			if err := json.Unmarshal(raw, &st); err != nil {
				log.Fatalf("katalog.json rusak, perbaiki atau pindahkan dulu: %v", err)
			}
		}
	}
	segarkanBawaan(&st)
	return st
}

func (s *Server) saveKatalog(st katalog.State) error {
	if s.db != nil {
		return saveKatalogDB(s.db, st)
	}
	if p := katalogFile(); p != "" {
		return writeFileAtomic(p, st)
	}
	return nil
}

// ubahKatalog: salin, ubah, simpan, lalu pasang. Gagal simpan = katalog lama tetap.
func (s *Server) ubahKatalog(f func(st *katalog.State) error) error {
	s.katalogMu.Lock()
	defer s.katalogMu.Unlock()
	st := s.katalog.Clone()
	if err := f(&st); err != nil {
		return err
	}
	if err := s.saveKatalog(st); err != nil {
		log.Printf("GAGAL menyimpan katalog: %v", err)
		return fmt.Errorf("%w: %w", errSimpan, err)
	}
	s.katalog = st
	pasangKatalog(st)
	return nil
}

// --- COA per tenant ---

// coaTemplate: template yang berlaku untuk usaha ini — versi yang dipatok bila jenis usahanya
// masih sama, selain itu versi terbit terbaru. ok=false bila jenis usaha belum punya template.
func (s *tenant) coaTemplate(p finance.BusinessProfile) (katalog.Template, bool) {
	st := katalogAktif.Load()
	if st == nil {
		return katalog.Template{}, false
	}
	s.mu.RLock()
	pin := s.coaTemplateID
	s.mu.RUnlock()
	if t, ok := st.TemplateByID(pin); ok && pin > 0 && t.Jenis == p.JenisUsaha && t.Status != katalog.Draft {
		return t, true
	}
	return st.TemplateAktif(p.JenisUsaha)
}

// coaAkun: semua akun template (urutan neraca saldo) dan akun yang aktif untuk profil.
func (s *tenant) coaAkun() (all, active []coa.Account, p finance.BusinessProfile) {
	p = s.currentProfile()
	if t, ok := s.coaTemplate(p); ok {
		return t.Akun, coa.ActiveFrom(t.Akun, p), p
	}
	return nil, nil, p
}

type coaPembaruan struct {
	ID    int              `json:"id"`
	Versi int              `json:"versi"`
	Diff  katalog.DiffAkun `json:"diff"`
}

// pembaruanCOA: versi terbit lebih baru dari yang dipakai usaha ini, beserta bedanya.
func (s *tenant) pembaruanCOA(p finance.BusinessProfile) (katalog.Template, *coaPembaruan) {
	t, ok := s.coaTemplate(p)
	if !ok {
		return t, nil
	}
	if a, _ := katalogAktif.Load().TemplateAktif(p.JenisUsaha); a.ID != t.ID && a.Versi > t.Versi {
		return t, &coaPembaruan{a.ID, a.Versi, katalog.Bandingkan(t.Akun, a.Akun)}
	}
	return t, nil
}

// handlePerbaruiCOA: Owner memindahkan usaha ke versi template terbaru. Ditolak bila ada
// transaksi yang memakai akun yang dihapus di versi baru.
func (s *tenant) handlePerbaruiCOA(w http.ResponseWriter, r *http.Request) {
	p := s.currentProfile()
	_, baru := s.pembaruanCOA(p)
	if baru == nil {
		http.Error(w, "sudah memakai versi terbaru", http.StatusConflict)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var dipakai []string
	for _, h := range baru.Diff.Hapus {
		if slices.ContainsFunc(s.txs, func(t finance.Transaction) bool { return t.Akun == h.Kode }) {
			dipakai = append(dipakai, h.Kode+" "+h.Nama)
		}
	}
	if len(dipakai) > 0 {
		http.Error(w, fmt.Sprintf("Akun berikut dihapus di versi baru tapi masih dipakai transaksi Anda: %v. Ganti akunnya dulu.", dipakai), http.StatusConflict)
		return
	}
	s.coaTemplateID = baru.ID
	writeJSON(w, baru)
}

// --- Panel Master ---

type katalogCOAResp struct {
	Jenis     []string           `json:"jenis"`
	Template  []katalog.Template `json:"template"`
	KodeWajib []string           `json:"kode_wajib"`
	// KlienPerTemplate: jumlah usaha yang memakai tiap versi.
	KlienPerTemplate map[int]int `json:"klien_per_template"`
}

func (s *Server) handleKatalogCOA(w http.ResponseWriter, r *http.Request) {
	s.katalogMu.Lock()
	st := s.katalog.Clone()
	s.katalogMu.Unlock()
	out := katalogCOAResp{Jenis: katalog.JenisUsaha, Template: st.Template, KodeWajib: coa.KodeWajib(), KlienPerTemplate: map[int]int{}}
	for _, t := range s.semuaTenant() {
		if tm, ok := t.coaTemplate(t.currentProfile()); ok {
			out.KlienPerTemplate[tm.ID]++
		}
	}
	writeJSON(w, out)
}

func (s *Server) semuaTenant() []*tenant {
	s.tenantsMu.RLock()
	defer s.tenantsMu.RUnlock()
	out := make([]*tenant, 0, len(s.tenants))
	for _, t := range s.tenants {
		out = append(out, t)
	}
	slices.SortFunc(out, func(a, b *tenant) int { return a.id - b.id })
	return out
}

// barisUnggahan: baris sheet pertama berkas .xlsx di field "file".
func barisUnggahan(w http.ResponseWriter, r *http.Request) ([][]string, bool) {
	sheets, ok := uploadedSheets(w, r)
	if !ok {
		return nil, false
	}
	if len(sheets) == 0 {
		http.Error(w, "berkas tidak berisi sheet", http.StatusBadRequest)
		return nil, false
	}
	return sheets[0].Rows, true
}

func writeKatalogErr(w http.ResponseWriter, err error) {
	var hilang katalog.ErrKLUHilang
	switch {
	case errors.As(err, &hilang):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, katalog.ErrTidakValid), errors.Is(err, coa.ErrTemplate), errors.Is(err, tax.ErrParameter):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		writeMasterErr(w, err)
	}
}

// handleDraftCOA: draft dari Excel (multipart: file, jenis, catatan) atau JSON {jenis, akun, catatan}.
func (s *Server) handleDraftCOA(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Jenis, Catatan string
		Akun           []coa.Account
	}
	if mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mt == "multipart/form-data" {
		rows, ok := barisUnggahan(w, r)
		if !ok {
			return
		}
		var err error
		if req.Akun, err = katalog.ParseTemplate(rows); err != nil {
			writeKatalogErr(w, err)
			return
		}
		req.Jenis, req.Catatan = r.FormValue("jenis"), r.FormValue("catatan")
	} else if !decodeBody(w, r, &req) {
		return
	}
	u := currentUser(r)
	var d katalog.Template
	err := s.ubahKatalog(func(st *katalog.State) (err error) {
		d, err = st.SimpanDraft(req.Jenis, req.Akun, req.Catatan, u.Name, time.Now().UTC().Truncate(time.Second))
		return err
	})
	if err != nil {
		writeKatalogErr(w, err)
		return
	}
	if err := s.catat(r, "Simpan draft template COA "+d.Jenis, 0, fmt.Sprintf("%d akun", len(d.Akun))); err != nil {
		writeMasterErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, d)
}

func (s *Server) handleHapusDraftCOA(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	if err := s.ubahKatalog(func(st *katalog.State) error { return st.HapusDraft(id) }); err != nil {
		writeKatalogErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleTerbitkanCOA: draft jadi versi aktif. Usaha yang selama ini mengikuti versi terbaru
// dipatok ke versi lama dulu, supaya pembukuannya tidak berubah tanpa persetujuan Owner.
func (s *Server) handleTerbitkanCOA(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	u := currentUser(r)
	var baru katalog.Template
	var lamaID int
	err := s.ubahKatalog(func(st *katalog.State) error {
		var lama *katalog.Template
		var err error
		if baru, lama, err = st.Terbitkan(id, u.Name, time.Now().UTC().Truncate(time.Second)); err != nil {
			return err
		}
		if lama != nil {
			lamaID = lama.ID
		}
		return nil
	})
	if err != nil {
		writeKatalogErr(w, err)
		return
	}
	if lamaID > 0 {
		for _, t := range s.semuaTenant() {
			p := t.currentProfile()
			t.mu.Lock()
			pin := p.JenisUsaha == baru.Jenis && t.coaTemplateID == 0
			if pin {
				t.coaTemplateID = lamaID
			}
			t.mu.Unlock()
			if pin {
				if err := s.saveFor(t); err != nil {
					writeMasterErr(w, err)
					return
				}
			}
		}
	}
	if err := s.catat(r, fmt.Sprintf("Terbitkan template COA %s v%d", baru.Jenis, baru.Versi), 0, baru.Catatan); err != nil {
		writeMasterErr(w, err)
		return
	}
	writeJSON(w, baru)
}

func (s *Server) handleUnduhCOA(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	s.katalogMu.Lock()
	t, ok := s.katalog.TemplateByID(id)
	s.katalogMu.Unlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="coa-%s-v%d.xlsx"`, t.Jenis, t.Versi))
	_ = xlsx.Write(w, []xlsx.Sheet{{Name: "COA", Rows: katalog.BarisTemplate(t.Akun)}})
}

type katalogBenchmarkResp struct {
	Versi       []katalog.Benchmark `json:"versi"`
	KlienPerKLU map[string]int      `json:"klien_per_klu"`
}

func (s *Server) kluDipakai() map[string]int {
	out := map[string]int{}
	for _, t := range s.semuaTenant() {
		if k := t.currentProfile().KLU; k != "" {
			out[k]++
		}
	}
	return out
}

func (s *Server) handleKatalogBenchmark(w http.ResponseWriter, r *http.Request) {
	s.katalogMu.Lock()
	st := s.katalog.Clone()
	s.katalogMu.Unlock()
	writeJSON(w, katalogBenchmarkResp{st.Benchmark, s.kluDipakai()})
}

// handleImporBenchmark: multipart (file, nama, sumber) → versi draft.
func (s *Server) handleImporBenchmark(w http.ResponseWriter, r *http.Request) {
	rows, ok := barisUnggahan(w, r)
	if !ok {
		return
	}
	baris, err := katalog.ParseBenchmark(rows)
	if err != nil {
		writeKatalogErr(w, err)
		return
	}
	u := currentUser(r)
	var b katalog.Benchmark
	err = s.ubahKatalog(func(st *katalog.State) (err error) {
		b, err = st.SimpanBenchmark(r.FormValue("nama"), r.FormValue("sumber"), baris, u.Name, time.Now().UTC().Truncate(time.Second))
		return err
	})
	if err != nil {
		writeKatalogErr(w, err)
		return
	}
	if err := s.catat(r, "Impor benchmark "+b.Nama, 0, fmt.Sprintf("%d KLU", len(b.Baris))); err != nil {
		writeMasterErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, b)
}

func (s *Server) handleAktifkanBenchmark(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	var dipakai []string
	for k := range s.kluDipakai() {
		dipakai = append(dipakai, k)
	}
	var b katalog.Benchmark
	err := s.ubahKatalog(func(st *katalog.State) error {
		if err := st.AktifkanBenchmark(id, dipakai); err != nil {
			return err
		}
		b, _ = st.BenchmarkByID(id)
		return nil
	})
	if err != nil {
		writeKatalogErr(w, err)
		return
	}
	if err := s.catat(r, "Aktifkan benchmark "+b.Nama, 0, b.Sumber); err != nil {
		writeMasterErr(w, err)
		return
	}
	writeJSON(w, b)
}

func (s *Server) handleUnduhBenchmark(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	s.katalogMu.Lock()
	b, ok := s.katalog.BenchmarkByID(id)
	s.katalogMu.Unlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="benchmark-djp-%d.xlsx"`, b.ID))
	_ = xlsx.Write(w, []xlsx.Sheet{{Name: "Benchmark", Rows: katalog.BarisBenchmark(b.Baris)}})
}

func (s *Server) katalogRoutes() {
	m := s.mux
	m.HandleFunc("GET /api/v1/master/coa", s.guardMP(s.handleKatalogCOA, master.COATemplate))
	m.HandleFunc("POST /api/v1/master/coa/draft", s.guardMP(s.handleDraftCOA, master.COATemplate))
	m.HandleFunc("DELETE /api/v1/master/coa/{id}", s.guardMP(s.handleHapusDraftCOA, master.COATemplate))
	m.HandleFunc("POST /api/v1/master/coa/{id}/terbitkan", s.guardMP(s.handleTerbitkanCOA, master.COATemplate))
	m.HandleFunc("GET /api/v1/master/coa/{id}/xlsx", s.guardMP(s.handleUnduhCOA, master.COATemplate))
	m.HandleFunc("GET /api/v1/master/benchmark", s.guardMP(s.handleKatalogBenchmark, master.Benchmark))
	m.HandleFunc("POST /api/v1/master/benchmark", s.guardMP(s.handleImporBenchmark, master.Benchmark))
	m.HandleFunc("POST /api/v1/master/benchmark/{id}/aktifkan", s.guardMP(s.handleAktifkanBenchmark, master.Benchmark))
	m.HandleFunc("GET /api/v1/master/benchmark/{id}/xlsx", s.guardMP(s.handleUnduhBenchmark, master.Benchmark))
	s.parameterRoutes()
	s.mux.HandleFunc("POST /api/v1/coa/perbarui", s.persisting(s.guard("profil-usaha", s.on((*tenant).handlePerbaruiCOA))))
}
