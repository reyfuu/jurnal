package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"pajak/internal/ai"
	"pajak/internal/master"
	"pajak/internal/pengetahuan"
	"pajak/internal/tax"
	"pajak/internal/xlsx"
)

// Artikel pajak & basis pengetahuan AI (Panel Master), Pustaka Pajak (klien), dan analisa AI
// per klien yang merujuk sumber-sumber itu.

func pengetahuanFile() string { return dataPath("pengetahuan.json") }

func loadPengetahuanFile() pengetahuan.State {
	st := pengetahuan.NewState()
	if p := pengetahuanFile(); p != "" {
		if raw, err := os.ReadFile(p); err == nil {
			if err := json.Unmarshal(raw, &st); err != nil {
				log.Fatalf("pengetahuan.json rusak, perbaiki atau pindahkan dulu: %v", err)
			}
		}
	}
	return st
}

func (s *Server) ubahPengetahuan(f func(st *pengetahuan.State) error) error {
	s.pengetahuanMu.Lock()
	defer s.pengetahuanMu.Unlock()
	st := s.pengetahuan.Clone()
	if err := f(&st); err != nil {
		return err
	}
	var err error
	switch {
	case s.db != nil:
		err = savePengetahuanDB(s.db, st)
	case pengetahuanFile() != "":
		err = writeFileAtomic(pengetahuanFile(), st)
	}
	if err != nil {
		log.Printf("GAGAL menyimpan basis pengetahuan: %v", err)
		return fmt.Errorf("%w: %w", errSimpan, err)
	}
	s.pengetahuan = st
	return nil
}

func (s *Server) salinPengetahuan() pengetahuan.State {
	s.pengetahuanMu.Lock()
	defer s.pengetahuanMu.Unlock()
	return s.pengetahuan.Clone()
}

func writePengetahuanErr(w http.ResponseWriter, err error) {
	if errors.Is(err, pengetahuan.ErrTidakValid) || errors.Is(err, pengetahuan.ErrPDFKosong) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeMasterErr(w, err)
}

// --- artikel ---

func (s *Server) handleListArtikel(w http.ResponseWriter, r *http.Request) {
	st := s.salinPengetahuan()
	writeJSON(w, nonNil(st.Artikel))
}

// handleSimpanArtikel: POST = baru, PUT /{id} = ubah. Menerbitkan (atau mengubah artikel yang
// sudah terbit) hanya untuk izin artikel_terbit.
func (s *Server) handleSimpanArtikel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		pengetahuan.Artikel
		Terbit bool `json:"terbit"`
	}
	if !decodeBody(w, r, &req) {
		return
	}
	a := req.Artikel
	a.ID, _ = strconv.Atoi(r.PathValue("id"))
	ak, u := aksesOf(r), currentUser(r)
	bolehTerbit := ak.Bisa(master.ArtikelTerbit)
	err := s.ubahPengetahuan(func(st *pengetahuan.State) error {
		if i := slices.IndexFunc(st.Artikel, func(x pengetahuan.Artikel) bool { return x.ID == a.ID }); i >= 0 && st.Artikel[i].Status == pengetahuan.Terbit && !bolehTerbit {
			return errBukanPenerbit
		}
		if req.Terbit && !bolehTerbit {
			return errBukanPenerbit
		}
		var err error
		a, err = st.SimpanArtikel(a, req.Terbit, u.Name, time.Now().UTC().Truncate(time.Second))
		return err
	})
	if errors.Is(err, errBukanPenerbit) {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	if err != nil {
		writePengetahuanErr(w, err)
		return
	}
	aksi := "Simpan draft artikel "
	if a.Status == pengetahuan.Terbit {
		aksi = "Terbitkan artikel "
	}
	if err := s.catat(r, aksi+pengetahuan.RefArtikel(a.ID), 0, a.Judul); err != nil {
		writeMasterErr(w, err)
		return
	}
	if r.Method == http.MethodPost {
		w.WriteHeader(http.StatusCreated)
	}
	writeJSON(w, a)
}

var errBukanPenerbit = errors.New("menerbitkan atau mengubah artikel terbit butuh izin verifikasi artikel")

// handlePustaka: artikel terbit yang ditandai tampil ke klien.
func (s *Server) handlePustaka(w http.ResponseWriter, r *http.Request) {
	out := []pengetahuan.Artikel{}
	for _, a := range s.salinPengetahuan().Artikel {
		if a.Status == pengetahuan.Terbit && a.TampilKlien {
			out = append(out, a)
		}
	}
	slices.SortStableFunc(out, func(a, b pengetahuan.Artikel) int { return b.Waktu.Compare(a.Waktu) })
	writeJSON(w, out)
}

// --- sumber pengetahuan ---

type sumberRow struct {
	pengetahuan.Sumber
	Potongan int    `json:"potongan"`
	Cuplikan string `json:"cuplikan"`
}

type pengetahuanResp struct {
	Sumber          []sumberRow `json:"sumber"`
	ArtikelTerbit   int         `json:"artikel_terbit"`
	PotonganDipakai int         `json:"potongan_dipakai"`
}

func (s *Server) handleListPengetahuan(w http.ResponseWriter, r *http.Request) {
	st := s.salinPengetahuan()
	out := pengetahuanResp{Sumber: []sumberRow{}, PotonganDipakai: len(st.Potongan())}
	for _, src := range st.Sumber {
		row := sumberRow{Sumber: src, Potongan: pengetahuan.JumlahPotongan(src.Isi), Cuplikan: clip(src.Isi, 280)}
		row.Isi = "" // isi lengkap bisa jutaan karakter; cukup cuplikan
		out.Sumber = append(out.Sumber, row)
	}
	for _, a := range st.Artikel {
		if a.Status == pengetahuan.Terbit {
			out.ArtikelTerbit++
		}
	}
	writeJSON(w, out)
}

// handleTambahPengetahuan: JSON {judul, isi, topik} (tulis manual) atau multipart file .xlsx/.pdf
// + judul + topik (pisah koma).
func (s *Server) handleTambahPengetahuan(w http.ResponseWriter, r *http.Request) {
	var src pengetahuan.Sumber
	if mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mt == "multipart/form-data" {
		r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
		f, hdr, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "unggah berkas .xlsx atau .pdf (maks 20 MB) di field file", http.StatusBadRequest)
			return
		}
		defer f.Close()
		data, err := io.ReadAll(f)
		if err != nil {
			http.Error(w, "gagal membaca berkas", http.StatusBadRequest)
			return
		}
		src.Judul, src.Topik, src.Kategori = r.FormValue("judul"), strings.Split(r.FormValue("topik"), ","), r.FormValue("kategori")
		switch strings.ToLower(filepath.Ext(hdr.Filename)) {
		case ".pdf":
			src.Jenis = "pdf"
			if src.Isi, err = pengetahuan.TeksPDF(data); err != nil {
				writePengetahuanErr(w, err)
				return
			}
		case ".xlsx":
			src.Jenis = "excel"
			sheets, err := xlsx.ReadAll(data)
			if err != nil || len(sheets) == 0 {
				http.Error(w, "berkas Excel tidak bisa dibaca", http.StatusBadRequest)
				return
			}
			var topik []string
			if src.Isi, topik, err = pengetahuan.ParseExcel(sheets[0].Rows); err != nil {
				writePengetahuanErr(w, err)
				return
			}
			src.Topik = append(src.Topik, topik...)
		default:
			http.Error(w, "hanya .xlsx atau .pdf", http.StatusBadRequest)
			return
		}
	} else {
		if !decodeBody(w, r, &src) {
			return
		}
		src.Jenis = "manual"
	}
	u := currentUser(r)
	err := s.ubahPengetahuan(func(st *pengetahuan.State) (err error) {
		src, err = st.TambahSumber(src, u.Name, time.Now().UTC().Truncate(time.Second))
		return err
	})
	if err != nil {
		writePengetahuanErr(w, err)
		return
	}
	if err := s.catat(r, "Tambah sumber pengetahuan "+pengetahuan.RefSumber(src.ID), 0, src.Judul); err != nil {
		writeMasterErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, sumberRow{Sumber: src, Potongan: pengetahuan.JumlahPotongan(src.Isi)})
}

func (s *Server) handleUbahPengetahuan(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	u := currentUser(r)
	var aksi string
	var err error
	switch {
	case r.Method == http.MethodDelete:
		aksi, err = "Hapus sumber pengetahuan ", s.ubahPengetahuan(func(st *pengetahuan.State) error { return st.HapusSumber(id) })
	case strings.HasSuffix(r.URL.Path, "/verifikasi"):
		aksi, err = "Verifikasi sumber pengetahuan ", s.ubahPengetahuan(func(st *pengetahuan.State) error { return st.Verifikasi(id, u.Name) })
	default:
		var req struct {
			DipakaiAI bool `json:"dipakai_ai"`
		}
		if !decodeBody(w, r, &req) {
			return
		}
		aksi = "Lepas sumber dari AI "
		if req.DipakaiAI {
			aksi = "Pakai sumber untuk AI "
		}
		err = s.ubahPengetahuan(func(st *pengetahuan.State) error { return st.SetDipakai(id, req.DipakaiAI) })
	}
	if err != nil {
		writePengetahuanErr(w, err)
		return
	}
	if err := s.catat(r, aksi+pengetahuan.RefSumber(id), 0, ""); err != nil {
		writeMasterErr(w, err)
		return
	}
	if r.Method == http.MethodDelete {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

// handleCariPengetahuan: "uji pertanyaan" — sumber yang akan ditemukan AI untuk q.
func (s *Server) handleCariPengetahuan(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, nonNil(pengetahuan.Cari(s.salinPengetahuan().Potongan(), r.URL.Query().Get("q"), 5)))
}

// --- analisa AI per klien ---

type analisaKlienResp struct {
	Fakta   ai.FaktaKlien          `json:"fakta"`
	Sumber  []pengetahuan.Potongan `json:"sumber"`
	Analisa ai.AnalisaKlienHasil   `json:"analisa"`
}

// faktaKlien: angka dihitung di sini (Go), bukan oleh model.
func faktaKlien(t *tenant) ai.FaktaKlien {
	p := t.currentProfile()
	omzet, margin := t.annualized()
	cek := tax.CekAwal(tax.CekAwalInput{BentukUsaha: p.BentukUsaha, PTKP: p.PTKP, KLU: p.KLU, PKP: p.PKP, Omzet: omzet, Laba: omzet * margin / 10_000})
	t.mu.RLock()
	nama := ""
	if t.profile != nil {
		nama = t.profile.NamaUsaha
	}
	nama = namaKlien(t, nama)
	t.mu.RUnlock()
	f := ai.FaktaKlien{Nama: nama, JenisUsaha: p.JenisUsaha, BentukUsaha: p.BentukUsaha, PKP: p.PKP, KLU: p.KLU,
		OmzetSetahun: omzet, NPMBps: margin, StatusNPM: cek.Status, SumberBenchmark: cek.Sumber}
	if cek.KLU != nil {
		f.KLUNama, f.BenchmarkNPMBps, f.BenchmarkCTTORBps = cek.KLU.Nama, cek.KLU.NPMBps, cek.KLU.CTTORBps
	}
	for _, rk := range cek.Risiko {
		f.Risiko = append(f.Risiko, rk.Judul)
	}
	return f
}

func (s *Server) handleAnalisaKlien(w http.ResponseWriter, r *http.Request) {
	t := s.klienDari(w, r)
	if t == nil {
		return
	}
	f := faktaKlien(t)
	sumber := nonNil(pengetahuan.Cari(s.salinPengetahuan().Potongan(), ai.QueryKlien(f, f.Risiko), 6))
	if err := s.catat(r, "Analisa AI klien", t.id, ""); err != nil {
		writeMasterErr(w, err)
		return
	}
	writeJSON(w, analisaKlienResp{f, sumber, ai.AnalisaKlien(r.Context(), s.llm, f, sumber)})
}

func (s *Server) pengetahuanRoutes() {
	m := s.mux
	m.HandleFunc("GET /api/v1/master/artikel", s.guardMP(s.handleListArtikel, master.Artikel, master.ArtikelTerbit))
	m.HandleFunc("POST /api/v1/master/artikel", s.guardMP(s.handleSimpanArtikel, master.Artikel, master.ArtikelTerbit))
	m.HandleFunc("PUT /api/v1/master/artikel/{id}", s.guardMP(s.handleSimpanArtikel, master.Artikel, master.ArtikelTerbit))
	m.HandleFunc("GET /api/v1/master/pengetahuan", s.guardMP(s.handleListPengetahuan, master.Pengetahuan))
	m.HandleFunc("POST /api/v1/master/pengetahuan", s.guardMP(s.handleTambahPengetahuan, master.Pengetahuan))
	m.HandleFunc("GET /api/v1/master/pengetahuan/cari", s.guardMP(s.handleCariPengetahuan, master.Pengetahuan))
	m.HandleFunc("POST /api/v1/master/pengetahuan/{id}/verifikasi", s.guardMP(s.handleUbahPengetahuan, master.Pengetahuan))
	m.HandleFunc("PUT /api/v1/master/pengetahuan/{id}", s.guardMP(s.handleUbahPengetahuan, master.Pengetahuan))
	m.HandleFunc("DELETE /api/v1/master/pengetahuan/{id}", s.guardMP(s.handleUbahPengetahuan, master.Pengetahuan))
	m.HandleFunc("GET /api/v1/master/klien/{id}/analisa", s.guardMP(s.handleAnalisaKlien, master.LihatSebagai))
	m.HandleFunc("GET /api/v1/pustaka", s.guard("pustaka", s.handlePustaka))
}
