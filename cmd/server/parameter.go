package main

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"pajak/internal/katalog"
	"pajak/internal/master"
	"pajak/internal/tax"
	"pajak/internal/xlsx"
)

// Parameter pajak di Panel Master: angka aturan (tarif, PTKP, batas, TER, objek potong, norma
// NPPN) berversi. Formulir mengubah semuanya kecuali norma; norma (±1.400 KLU) lewat Excel.

// parameterRingkas: versi tanpa tabel norma, supaya daftar & formulir tetap ringan.
type parameterRingkas struct {
	katalog.ParameterVersi
	JumlahNorma int `json:"jumlah_norma"`
}

func ringkas(v katalog.ParameterVersi) parameterRingkas {
	n := len(v.Isi.Norma)
	v.Isi = v.Isi.Salin()
	v.Isi.Norma = nil
	return parameterRingkas{v, n}
}

func (s *Server) parameterVersi(id int) (katalog.ParameterVersi, bool) {
	s.katalogMu.Lock()
	defer s.katalogMu.Unlock()
	return s.katalog.ParameterByID(id)
}

func (s *Server) handleDaftarParameter(w http.ResponseWriter, r *http.Request) {
	s.katalogMu.Lock()
	vs := slices.Clone(s.katalog.Parameter)
	s.katalogMu.Unlock()
	out := make([]parameterRingkas, len(vs))
	for i, v := range vs {
		out[i] = ringkas(v)
	}
	writeJSON(w, map[string]any{"versi": out, "objek_terikat": tax.ObjekTerikat, "ptkp_codes": append(slices.Clone(tax.PTKPCodes), "K/I/0", "K/I/1", "K/I/2", "K/I/3")})
}

func idPath(r *http.Request) int {
	id, _ := strconv.Atoi(r.PathValue("id"))
	return id
}

// handleSalinParameter: draft baru berisi salinan versi id (titik awal setiap perubahan).
func (s *Server) handleSalinParameter(w http.ResponseWriter, r *http.Request) {
	asal, ok := s.parameterVersi(idPath(r))
	if !ok {
		http.NotFound(w, r)
		return
	}
	u := currentUser(r)
	var d katalog.ParameterVersi
	err := s.ubahKatalog(func(st *katalog.State) (err error) {
		d, err = st.SimpanParameter(0, "Draft dari "+asal.Nama, asal.Dasar, asal.Isi, u.Name, time.Now().UTC().Truncate(time.Second))
		return err
	})
	if err != nil {
		writeKatalogErr(w, err)
		return
	}
	if err := s.catat(r, "Buat draft parameter pajak dari "+asal.Nama, 0, ""); err != nil {
		writeMasterErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, ringkas(d))
}

// handleUbahParameter: ganti isi draft. Norma tidak ikut formulir; yang tersimpan dipertahankan.
func (s *Server) handleUbahParameter(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Nama, Dasar string
		Isi         tax.Parameter
	}
	if !decodeBody(w, r, &req) {
		return
	}
	id := idPath(r)
	u := currentUser(r)
	var d katalog.ParameterVersi
	err := s.ubahKatalog(func(st *katalog.State) (err error) {
		lama, ok := st.ParameterByID(id)
		if !ok {
			return fmt.Errorf("%w: versi tidak ditemukan", katalog.ErrTidakValid)
		}
		req.Isi.Norma = lama.Isi.Norma
		d, err = st.SimpanParameter(id, req.Nama, req.Dasar, req.Isi, u.Name, time.Now().UTC().Truncate(time.Second))
		return err
	})
	if err != nil {
		writeKatalogErr(w, err)
		return
	}
	writeJSON(w, ringkas(d))
}

func (s *Server) handleHapusParameter(w http.ResponseWriter, r *http.Request) {
	if err := s.ubahKatalog(func(st *katalog.State) error { return st.HapusDraftParameter(idPath(r)) }); err != nil {
		writeKatalogErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAktifkanParameter(w http.ResponseWriter, r *http.Request) {
	id := idPath(r)
	u := currentUser(r)
	err := s.ubahKatalog(func(st *katalog.State) error {
		return st.AktifkanParameter(id, u.Name, time.Now().UTC().Truncate(time.Second))
	})
	if err != nil {
		writeKatalogErr(w, err)
		return
	}
	v, _ := s.parameterVersi(id)
	if err := s.catat(r, "Aktifkan parameter pajak "+v.Nama, 0, v.Dasar); err != nil {
		writeMasterErr(w, err)
		return
	}
	writeJSON(w, ringkas(v))
}

// handleImporNorma: Excel norma menggantikan tabel norma draft id.
func (s *Server) handleImporNorma(w http.ResponseWriter, r *http.Request) {
	rows, ok := barisUnggahan(w, r)
	if !ok {
		return
	}
	ns, err := katalog.ParseNorma(rows)
	if err != nil {
		writeKatalogErr(w, err)
		return
	}
	id := idPath(r)
	u := currentUser(r)
	var d katalog.ParameterVersi
	err = s.ubahKatalog(func(st *katalog.State) (err error) {
		lama, ok := st.ParameterByID(id)
		if !ok {
			return fmt.Errorf("%w: versi tidak ditemukan", katalog.ErrTidakValid)
		}
		isi := lama.Isi.Salin()
		isi.Norma = ns
		d, err = st.SimpanParameter(id, lama.Nama, lama.Dasar, isi, u.Name, time.Now().UTC().Truncate(time.Second))
		return err
	})
	if err != nil {
		writeKatalogErr(w, err)
		return
	}
	if err := s.catat(r, "Impor norma NPPN ke "+d.Nama, 0, fmt.Sprintf("%d KLU", len(ns))); err != nil {
		writeMasterErr(w, err)
		return
	}
	writeJSON(w, ringkas(d))
}

func (s *Server) handleUnduhNorma(w http.ResponseWriter, r *http.Request) {
	v, ok := s.parameterVersi(idPath(r))
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="norma-nppn-%d.xlsx"`, v.ID))
	_ = xlsx.Write(w, []xlsx.Sheet{{Name: "Norma", Rows: katalog.BarisNorma(v.Isi.Norma)}})
}

// --- kalkulator klien ---

// handleCariNorma: KLU 5 digit untuk kalkulator NPPN (cari kode atau uraian, maks. 20).
func (s *Server) handleCariNorma(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	out := []tax.Norma{}
	if q != "" {
		for _, n := range tax.P().Norma {
			if strings.HasPrefix(n.KLU, q) || strings.Contains(strings.ToLower(n.Uraian), q) {
				if out = append(out, n); len(out) == 20 {
					break
				}
			}
		}
	}
	writeJSON(w, out)
}

// hitung: handler POST generik untuk kalkulator perencanaan (input JSON → hasil atau 400).
func hitung[I, O any](f func(I) (O, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in I
		if !decodeBody(w, r, &in) {
			return
		}
		out, err := f(in)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, out)
	}
}

func (s *Server) parameterRoutes() {
	m := s.mux
	g := func(h http.HandlerFunc) http.HandlerFunc { return s.guardMP(h, master.ParameterPajak) }
	m.HandleFunc("GET /api/v1/master/parameter", g(s.handleDaftarParameter))
	m.HandleFunc("POST /api/v1/master/parameter/{id}/salin", g(s.handleSalinParameter))
	m.HandleFunc("PUT /api/v1/master/parameter/{id}", g(s.handleUbahParameter))
	m.HandleFunc("DELETE /api/v1/master/parameter/{id}", g(s.handleHapusParameter))
	m.HandleFunc("POST /api/v1/master/parameter/{id}/aktifkan", g(s.handleAktifkanParameter))
	m.HandleFunc("POST /api/v1/master/parameter/{id}/norma", g(s.handleImporNorma))
	m.HandleFunc("GET /api/v1/master/parameter/{id}/norma.xlsx", g(s.handleUnduhNorma))

	k := func(h http.HandlerFunc) http.HandlerFunc { return s.guard("kalkulator-pajak", h) }
	m.HandleFunc("GET /api/v1/kalkulator/norma", k(s.handleCariNorma))
	m.HandleFunc("POST /api/v1/kalkulator/nppn", k(hitung(tax.HitungNPPN)))
	m.HandleFunc("POST /api/v1/kalkulator/gross-up", k(hitung(tax.HitungGrossUp)))
	m.HandleFunc("POST /api/v1/kalkulator/ppn-harga", k(hitung(tax.HitungPPNHarga)))
	m.HandleFunc("POST /api/v1/kalkulator/beli-sewa", k(hitung(tax.HitungBeliSewa)))
}
