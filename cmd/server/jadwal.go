package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"pajak/internal/finance"
	"pajak/internal/master"
)

// Jadwal pengingat laporan & pajak (tutup buku, setor/lapor pajak): diatur Panel Master untuk
// semua klien, tampil di aplikasi klien sebagai banner mulai H-n. ponytail: kanal WA & email
// menyusul, memakai DaftarJadwal.Aktif yang sama.

func (s *Server) daftarJadwal() finance.DaftarJadwal {
	s.masterMu.Lock()
	defer s.masterMu.Unlock()
	return s.master.DaftarJadwal()
}

type jadwalBody struct {
	Item []finance.JadwalPengingat `json:"item"`
}

func (s *Server) handleMasterJadwal(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, jadwalBody{Item: s.daftarJadwal().Item})
}

func (s *Server) handleAturJadwal(w http.ResponseWriter, r *http.Request) {
	var b jadwalBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "data tidak terbaca", http.StatusBadRequest)
		return
	}
	var hasil finance.DaftarJadwal
	err := s.ubahMaster(func(st *master.State) error {
		d := st.DaftarJadwal()
		if err := d.Ganti(b.Item); err != nil {
			return err
		}
		st.Jadwal, hasil = &d, d
		return nil
	})
	if errors.Is(err, finance.ErrJadwal) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "gagal menyimpan jadwal", http.StatusInternalServerError)
		return
	}
	s.catat(r, "Ubah jadwal pengingat", 0, fmt.Sprintf("%d jadwal", len(hasil.Item)))
	writeJSON(w, jadwalBody{Item: hasil.Item})
}

// jadwalAktif: pengingat jadwal yang sedang berlaku untuk usaha ini (untuk banner di /auth/me).
func (s *Server) jadwalAktif(t *tenant) []finance.PengingatJadwal {
	d := s.daftarJadwal()
	t.mu.RLock()
	var p *finance.BusinessProfile
	if t.profile != nil {
		salin := *t.profile
		p = &salin
	}
	t.mu.RUnlock()
	return d.Aktif(p, time.Now())
}

func (s *Server) jadwalRoutes() {
	s.mux.HandleFunc("GET /api/v1/master/jadwal", s.guardMP(s.handleMasterJadwal, master.Pengaturan, master.CatatanKonsultan, master.KlienSemua))
	s.mux.HandleFunc("PUT /api/v1/master/jadwal", s.guardMP(s.handleAturJadwal, master.Pengaturan))
}
