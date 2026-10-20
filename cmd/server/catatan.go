package main

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"time"

	"pajak/internal/auth"
	"pajak/internal/finance"
	"pajak/internal/master"
)

// Catatan konsultan: ditulis tim (biasanya saat lihat sebagai) pada halaman & periode laporan
// klien; klien menandai sudah diperbaiki. Catatan terbuka diingatkan lagi di aplikasi sesuai
// aturan Pengingat Panel Master (satu aturan untuk semua klien). ponytail: kanal WA & email
// belum ada — jadwalnya sama (PerluDiingatkan), tinggal ditambah pengirimnya.

var labelTingkat = map[string]string{finance.TingkatInfo: "info", finance.TingkatPerbaiki: "perlu diperbaiki", finance.TingkatPenting: "penting"}

func (s *Server) aturanPengingat() finance.Pengingat {
	s.masterMu.Lock()
	defer s.masterMu.Unlock()
	return s.master.AturanPengingat()
}

// --- tim Master Pajak ---

func (s *Server) handleTulisCatatan(w http.ResponseWriter, r *http.Request) {
	t := s.klienDari(w, r)
	if t == nil {
		return
	}
	var c finance.CatatanKonsultan
	if !decodeBody(w, r, &c) {
		return
	}
	if err := finance.RapikanCatatan(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	u, ak := currentUser(r), aksesOf(r)
	t.mu.Lock()
	c.ID, c.OlehID, c.Oleh, c.Peran = t.nextCatatanID, u.ID, u.Name, ak.Role.Nama
	c.Dibuat, c.Status = time.Now().UTC().Truncate(time.Second), finance.CatatanTerbuka
	c.SelesaiOleh, c.SelesaiPada, c.DiingatkanPada, c.JumlahPengingat = "", time.Time{}, time.Time{}, 0
	t.nextCatatanID++
	t.catatan = append(t.catatan, c)
	t.mu.Unlock()
	if err := s.saveFor(t); err != nil {
		t.mu.Lock()
		t.catatan, t.nextCatatanID = t.catatan[:len(t.catatan)-1], t.nextCatatanID-1
		t.mu.Unlock()
		writeMasterErr(w, fmt.Errorf("%w: %w", errSimpan, err))
		return
	}
	label := namaHalaman[c.Halaman]
	if label == "" {
		label = c.Halaman
	}
	if err := s.catat(r, "Tulis catatan konsultan ("+labelTingkat[c.Tingkat]+") di "+label, t.id, c.Isi); err != nil {
		writeMasterErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, c)
}

type catatanRow struct {
	finance.CatatanKonsultan
	KlienID   int    `json:"klien_id"`
	KlienNama string `json:"klien_nama"`
}

func (s *Server) handleMasterCatatan(w http.ResponseWriter, r *http.Request) {
	ak := aksesOf(r)
	out := []catatanRow{}
	for _, t := range s.semuaTenant() {
		if !ak.BolehKlien(t.id) {
			continue
		}
		nama := namaKlien(t, t.currentProfile().NamaUsaha)
		t.mu.RLock()
		for _, c := range t.catatan {
			out = append(out, catatanRow{c, t.id, nama})
		}
		t.mu.RUnlock()
	}
	slices.SortStableFunc(out, func(a, b catatanRow) int { return b.Dibuat.Compare(a.Dibuat) })
	writeJSON(w, map[string]any{"catatan": out, "pengingat": s.aturanPengingat()})
}

func (s *Server) handleAturPengingat(w http.ResponseWriter, r *http.Request) {
	var p finance.Pengingat
	if !decodeBody(w, r, &p) {
		return
	}
	if err := p.Validasi(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.ubahMaster(func(st *master.State) error { st.Pengingat = &p; return nil }); err != nil {
		writeMasterErr(w, err)
		return
	}
	status := "nonaktif"
	if p.Aktif {
		status = fmt.Sprintf("setiap %d hari, mulai tingkat %s", p.IntervalHari, labelTingkat[p.TingkatMin])
	}
	if err := s.catat(r, "Ubah pengingat catatan konsultan", 0, status); err != nil {
		writeMasterErr(w, err)
		return
	}
	writeJSON(w, p)
}

// --- klien ---

type catatanKlienResp struct {
	Catatan []finance.CatatanKonsultan `json:"catatan"`
	// Pengingat: catatan yang jatuh tempo diingatkan sekarang (ditampilkan sebagai pop-up).
	Pengingat []int             `json:"pengingat"`
	Aturan    finance.Pengingat `json:"aturan"`
}

func (s *Server) handleCatatanKlien(w http.ResponseWriter, r *http.Request) {
	t := s.tenantOf(r)
	aturan, now := s.aturanPengingat(), time.Now()
	out := catatanKlienResp{Catatan: []finance.CatatanKonsultan{}, Pengingat: []int{}, Aturan: aturan}
	t.mu.RLock()
	for i := len(t.catatan) - 1; i >= 0; i-- {
		c := t.catatan[i]
		out.Catatan = append(out.Catatan, c)
		if !sedangLihat(r) && c.PerluDiingatkan(aturan, now) {
			out.Pengingat = append(out.Pengingat, c.ID)
		}
	}
	t.mu.RUnlock()
	writeJSON(w, out)
}

// bolehTindakLanjut: Owner & Admin usaha (bukan Kasir) menandai catatan selesai.
func bolehTindakLanjut(w http.ResponseWriter, u auth.User) bool {
	if u.Role != auth.Owner && u.Role != auth.Admin {
		http.Error(w, "hanya Owner atau Admin yang menindaklanjuti catatan konsultan", http.StatusForbidden)
		return false
	}
	return true
}

func (s *Server) handleCatatanSelesai(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if !bolehTindakLanjut(w, u) {
		return
	}
	id, _ := strconv.Atoi(r.PathValue("id"))
	t := s.tenantOf(r)
	t.mu.Lock()
	i := slices.IndexFunc(t.catatan, func(c finance.CatatanKonsultan) bool { return c.ID == id })
	if i < 0 {
		t.mu.Unlock()
		http.Error(w, "catatan tidak ditemukan", http.StatusNotFound)
		return
	}
	if t.catatan[i].Status == finance.CatatanSelesai {
		t.mu.Unlock()
		http.Error(w, "catatan sudah ditandai selesai", http.StatusConflict)
		return
	}
	t.catatan[i].Status, t.catatan[i].SelesaiOleh, t.catatan[i].SelesaiPada = finance.CatatanSelesai, u.Name, time.Now().UTC().Truncate(time.Second)
	c := t.catatan[i]
	t.mu.Unlock()
	_ = s.ubahMaster(func(st *master.State) error {
		st.Catat(master.Catatan{UserID: u.ID, Nama: u.Name, Peran: "Owner klien", Aksi: "Tandai catatan konsultan sudah diperbaiki", KlienID: t.id, Keterangan: c.Isi})
		return nil
	})
	writeJSON(w, c)
}

// handleCatatanDiingatkan: pop-up pengingat sudah ditampilkan; jeda pengingat dihitung ulang.
func (s *Server) handleCatatanDiingatkan(w http.ResponseWriter, r *http.Request) {
	var req struct{ IDs []int }
	if !decodeBody(w, r, &req) {
		return
	}
	t, now := s.tenantOf(r), time.Now().UTC().Truncate(time.Second)
	t.mu.Lock()
	for i := range t.catatan {
		if slices.Contains(req.IDs, t.catatan[i].ID) && t.catatan[i].Status == finance.CatatanTerbuka {
			t.catatan[i].DiingatkanPada = now
			t.catatan[i].JumlahPengingat++
		}
	}
	t.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// catatanTerbuka: jumlah catatan terbuka & yang jatuh tempo diingatkan (untuk banner di /auth/me).
func (s *Server) catatanTerbuka(t *tenant) (terbuka, ingat int) {
	aturan, now := s.aturanPengingat(), time.Now()
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, c := range t.catatan {
		if c.Status == finance.CatatanTerbuka {
			terbuka++
			if c.PerluDiingatkan(aturan, now) {
				ingat++
			}
		}
	}
	return
}

func (s *Server) catatanRoutes() {
	m := s.mux
	m.HandleFunc("POST /api/v1/master/klien/{id}/catatan", s.guardMP(s.handleTulisCatatan, master.CatatanKonsultan))
	m.HandleFunc("GET /api/v1/master/catatan", s.guardMP(s.handleMasterCatatan, master.CatatanKonsultan, master.Pengaturan, master.KlienSemua))
	m.HandleFunc("PUT /api/v1/master/pengingat", s.guardMP(s.handleAturPengingat, master.Pengaturan))
	m.HandleFunc("GET /api/v1/catatan", s.guard("", s.handleCatatanKlien))
	m.HandleFunc("POST /api/v1/catatan/{id}/selesai", s.persisting(s.guard("", s.handleCatatanSelesai)))
	m.HandleFunc("POST /api/v1/catatan/diingatkan", s.persisting(s.guard("", s.handleCatatanDiingatkan)))
}
