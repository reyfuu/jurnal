package main

import (
	"cmp"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"pajak/internal/auth"
	"pajak/internal/master"
)

// Kelola data dari Panel Master: usaha klien (ubah, arsipkan = soft delete, hapus permanen),
// pengguna klien (ubah kontak, nonaktif = soft delete, reset password, hapus permanen), dan
// hapus permanen anggota tim. Hapus permanen hanya untuk data yang sudah diarsipkan /
// dinonaktifkan dan wajib konfirmasi nama/email — tidak ada jalan kembali selain backup DB.

const pesanArsip = "Usaha ini diarsipkan oleh tim Master Pajak. Hubungi tim Master Pajak untuk memulihkannya."

func (s *Server) tenantByID(id int) *tenant {
	s.tenantsMu.RLock()
	defer s.tenantsMu.RUnlock()
	return s.tenants[id]
}

// diarsip: usaha tenantID diarsipkan; usaha 0 (tim) tidak pernah.
func (s *Server) diarsip(tenantID int) bool {
	t := s.tenantByID(tenantID)
	if tenantID == 0 || t == nil {
		return false
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return !t.arsip.IsZero()
}

// klienBoleh: usaha id ada dan boleh dilihat anggota; selain itu menulis 404.
func (s *Server) klienBoleh(w http.ResponseWriter, r *http.Request) (*tenant, bool) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	t := s.tenantByID(id)
	if t == nil || !aksesOf(r).BolehKlien(id) {
		http.Error(w, "usaha tidak ditemukan", http.StatusNotFound)
		return nil, false
	}
	return t, true
}

func sama(a, b string) bool { return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b)) }

// --- usaha klien ---

// handleUbahKlien: identitas usaha (nama, alamat, kontak). Struktur pajak (bentuk, PTKP, PKP)
// tetap diubah pemilik di Profil Usaha.
func (s *Server) handleUbahKlien(w http.ResponseWriter, r *http.Request) {
	t, ok := s.klienBoleh(w, r)
	if !ok {
		return
	}
	var req struct {
		NamaUsaha string `json:"nama_usaha"`
		Alamat    string `json:"alamat"`
		Kecamatan string `json:"kecamatan"`
		Kota      string `json:"kota"`
		Provinsi  string `json:"provinsi"`
		NPWP      string `json:"npwp"`
		Telepon   string `json:"telepon"`
		Email     string `json:"email"`
	}
	if !decodeBody(w, r, &req) {
		return
	}
	t.mu.Lock()
	if t.profile == nil {
		t.mu.Unlock()
		http.Error(w, "usaha ini belum mengisi profil", http.StatusConflict)
		return
	}
	p := *t.profile
	p.NamaUsaha, p.Alamat, p.Kecamatan, p.Kota, p.Provinsi = strings.TrimSpace(req.NamaUsaha), strings.TrimSpace(req.Alamat), strings.TrimSpace(req.Kecamatan), strings.TrimSpace(req.Kota), req.Provinsi
	p.NPWP, p.Telepon, p.Email = strings.TrimSpace(req.NPWP), strings.TrimSpace(req.Telepon), strings.TrimSpace(req.Email)
	if err := cmp.Or(p.Validate(), p.ValidateWilayah()); err != nil {
		t.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := saveProfile(t.file("business"), p); err != nil {
		t.mu.Unlock()
		http.Error(w, "gagal menyimpan profil usaha", http.StatusInternalServerError)
		return
	}
	t.profile = &p
	t.mu.Unlock()
	if err := s.saveFor(t); err != nil {
		writeMasterErr(w, fmt.Errorf("%w: %w", errSimpan, err))
		return
	}
	if err := s.catat(r, "Ubah data usaha "+p.NamaUsaha, t.id, ""); err != nil {
		writeMasterErr(w, err)
		return
	}
	writeJSON(w, p)
}

func (s *Server) ubahArsip(w http.ResponseWriter, r *http.Request, arsip bool) {
	t, ok := s.klienBoleh(w, r)
	if !ok {
		return
	}
	var req struct{ Alasan string }
	if arsip {
		if !decodeBody(w, r, &req) {
			return
		}
		if req.Alasan = strings.TrimSpace(req.Alasan); req.Alasan == "" {
			http.Error(w, "alasan mengarsipkan wajib diisi", http.StatusBadRequest)
			return
		}
		if t.demo {
			http.Error(w, "usaha contoh tidak bisa diarsipkan", http.StatusConflict)
			return
		}
	}
	t.mu.Lock()
	if arsip {
		t.arsip, t.arsipAlasan = time.Now().UTC().Truncate(time.Second), req.Alasan
	} else {
		t.arsip, t.arsipAlasan = time.Time{}, ""
	}
	t.mu.Unlock()
	if err := s.saveFor(t); err != nil {
		writeMasterErr(w, fmt.Errorf("%w: %w", errSimpan, err))
		return
	}
	aksi := "Pulihkan usaha dari arsip"
	if arsip {
		aksi = "Arsipkan usaha"
	}
	if err := s.catat(r, aksi+" "+namaKlien(t, ""), t.id, req.Alasan); err != nil {
		writeMasterErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleHapusKlien: hapus permanen usaha yang sudah diarsipkan beserta semua datanya & user-nya.
func (s *Server) handleHapusKlien(w http.ResponseWriter, r *http.Request) {
	t, ok := s.klienBoleh(w, r)
	if !ok {
		return
	}
	var req struct{ Konfirmasi string }
	if !decodeBody(w, r, &req) {
		return
	}
	nama := namaKlien(t, "")
	t.mu.RLock()
	diarsip := !t.arsip.IsZero()
	if t.profile != nil {
		nama = cmp.Or(t.profile.NamaUsaha, nama)
	}
	t.mu.RUnlock()
	switch {
	case t.demo || t.id == 1:
		http.Error(w, "usaha contoh tidak bisa dihapus", http.StatusConflict)
		return
	case !diarsip:
		http.Error(w, "arsipkan usaha dulu sebelum menghapus permanen", http.StatusConflict)
		return
	case !sama(req.Konfirmasi, nama):
		http.Error(w, "konfirmasi tidak cocok: ketik nama usaha persis", http.StatusBadRequest)
		return
	}
	s.tenantsMu.Lock()
	delete(s.tenants, t.id)
	s.tenantsMu.Unlock()
	nUser := s.users.HapusUsaha(t.id)
	if s.db != nil {
		s.saveMu.Lock()
		_, err := s.db.Exec(`DELETE FROM tenants WHERE id = $1`, t.id) // ON DELETE CASCADE: semua data usaha
		s.saveMu.Unlock()
		if err != nil {
			log.Printf("GAGAL menghapus usaha %d dari Postgres: %v", t.id, err)
			writeMasterErr(w, fmt.Errorf("%w: %w", errSimpan, err))
			return
		}
	} else if dataPath("") != "" {
		for _, f := range []string{"state", "business"} {
			if err := os.Remove(t.file(f)); err != nil && !errors.Is(err, os.ErrNotExist) {
				log.Printf("GAGAL menghapus %s: %v", t.file(f), err)
			}
		}
	}
	if err := s.saveFor(nil); err != nil {
		writeMasterErr(w, fmt.Errorf("%w: %w", errSimpan, err))
		return
	}
	if err := s.catat(r, "Hapus permanen usaha "+nama, t.id, fmt.Sprintf("%d pengguna ikut dihapus", nUser)); err != nil {
		writeMasterErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- pengguna klien ---

type penggunaRow struct {
	auth.User
	RoleLabel string `json:"role_label"`
	Usaha     string `json:"usaha"`
	Kota      string `json:"kota,omitempty"`
	Alamat    string `json:"alamat,omitempty"`
	Telepon   string `json:"telepon,omitempty"` // telepon usaha (profil)
	Arsip     bool   `json:"arsip"`
	// KlienID & Langganan: usaha pengguna ini dan status langganannya, supaya Master bisa
	// mengatur paket langsung dari daftar pengguna.
	KlienID   int             `json:"klien_id"`
	Langganan langgananStatus `json:"langganan"`
	Demo      bool            `json:"demo,omitempty"`
}

func (s *Server) handleMasterPengguna(w http.ResponseWriter, r *http.Request) {
	ak := aksesOf(r)
	out := []penggunaRow{}
	for _, u := range s.users.All() {
		if u.Role == auth.MP || !ak.BolehKlien(u.TenantID) {
			continue
		}
		row := penggunaRow{User: u, RoleLabel: auth.Roles[u.Role].Label}
		if t := s.tenantByID(u.TenantID); t != nil {
			t.mu.RLock()
			if p := t.profile; p != nil {
				row.Usaha, row.Kota, row.Alamat, row.Telepon = p.NamaUsaha, p.Kota, p.Alamat, p.Telepon
			}
			row.Arsip = !t.arsip.IsZero()
			row.KlienID, row.Demo = t.id, t.id == 1
			t.mu.RUnlock()
			row.Usaha = namaKlien(t, row.Usaha)
			row.Langganan = t.langgananStatus(time.Now())
		}
		out = append(out, row)
	}
	writeJSON(w, out)
}

// penggunaBoleh: user klien id yang boleh dikelola anggota; selain itu menulis 404.
func (s *Server) penggunaBoleh(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	u, ok := s.users.Get(id)
	if !ok || u.Role == auth.MP || !aksesOf(r).BolehKlien(u.TenantID) {
		http.Error(w, "pengguna tidak ditemukan", http.StatusNotFound)
		return auth.User{}, false
	}
	return u, true
}

func (s *Server) handleUbahPengguna(w http.ResponseWriter, r *http.Request) {
	u, ok := s.penggunaBoleh(w, r)
	if !ok {
		return
	}
	var req struct {
		Name, Email, WA string
		Aktif           bool
	}
	if !decodeBody(w, r, &req) {
		return
	}
	baru, err := s.users.UbahKontak(u.ID, req.Name, req.Email, req.WA)
	if err == nil && req.Aktif != u.Active {
		baru, err = s.users.Update(u.TenantID, u.ID, auth.Input{Name: baru.Name, Email: baru.Email, Role: u.Role, Active: req.Aktif})
	}
	if errors.Is(err, auth.ErrLastOwner) {
		http.Error(w, "ini satu-satunya Owner aktif usaha tersebut; arsipkan usahanya bila ingin menghentikan akses", http.StatusConflict)
		return
	}
	if err != nil {
		writeAuthErr(w, err)
		return
	}
	if err := s.saveFor(nil); err != nil {
		writeMasterErr(w, fmt.Errorf("%w: %w", errSimpan, err))
		return
	}
	ket := ""
	if req.Aktif != u.Active {
		ket = map[bool]string{true: "diaktifkan", false: "dinonaktifkan"}[req.Aktif]
	}
	if err := s.catat(r, "Ubah pengguna klien "+baru.Email, u.TenantID, ket); err != nil {
		writeMasterErr(w, err)
		return
	}
	writeJSON(w, baru)
}

func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	u, ok := s.penggunaBoleh(w, r)
	if !ok {
		return
	}
	pw, err := s.users.ResetPassword(u.ID)
	if err != nil {
		writeAuthErr(w, err)
		return
	}
	if err := s.saveFor(nil); err != nil {
		writeMasterErr(w, fmt.Errorf("%w: %w", errSimpan, err))
		return
	}
	if err := s.catat(r, "Reset password "+u.Email, u.TenantID, "password sementara dibuat; semua sesi diakhiri"); err != nil {
		writeMasterErr(w, err)
		return
	}
	writeJSON(w, map[string]string{"password": pw})
}

func (s *Server) handleHapusPengguna(w http.ResponseWriter, r *http.Request) {
	u, ok := s.penggunaBoleh(w, r)
	if !ok {
		return
	}
	var req struct{ Konfirmasi string }
	if !decodeBody(w, r, &req) {
		return
	}
	switch {
	case u.Active:
		http.Error(w, "nonaktifkan pengguna dulu sebelum menghapus permanen", http.StatusConflict)
		return
	case !sama(req.Konfirmasi, u.Email):
		http.Error(w, "konfirmasi tidak cocok: ketik email pengguna persis", http.StatusBadRequest)
		return
	}
	if err := s.users.Delete(u.TenantID, u.ID); err != nil {
		writeAuthErr(w, err)
		return
	}
	if err := s.saveFor(nil); err != nil {
		writeMasterErr(w, fmt.Errorf("%w: %w", errSimpan, err))
		return
	}
	if err := s.catat(r, "Hapus permanen pengguna "+u.Email, u.TenantID, ""); err != nil {
		writeMasterErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- anggota tim ---

func (s *Server) handleHapusAnggota(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	u, ok := s.users.Get(id)
	if !ok || u.Role != auth.MP {
		http.Error(w, "anggota tidak ditemukan", http.StatusNotFound)
		return
	}
	var req struct{ Konfirmasi string }
	if !decodeBody(w, r, &req) {
		return
	}
	switch {
	case id == currentUser(r).ID:
		http.Error(w, "tidak bisa menghapus akun sendiri", http.StatusConflict)
		return
	case u.Active:
		http.Error(w, "nonaktifkan anggota dulu sebelum menghapus permanen", http.StatusConflict)
		return
	case !sama(req.Konfirmasi, u.Email):
		http.Error(w, "konfirmasi tidak cocok: ketik email anggota persis", http.StatusBadRequest)
		return
	}
	if err := s.ubahMaster(func(st *master.State) error { return st.LepasAnggota(id) }); err != nil {
		writeMasterErr(w, err)
		return
	}
	if err := s.users.Delete(0, id); err != nil {
		writeAuthErr(w, err)
		return
	}
	if err := s.saveFor(nil); err != nil {
		writeMasterErr(w, fmt.Errorf("%w: %w", errSimpan, err))
		return
	}
	if err := s.catat(r, "Hapus permanen anggota tim "+u.Email, 0, ""); err != nil {
		writeMasterErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) kelolaRoutes() {
	m := s.mux
	m.HandleFunc("PUT /api/v1/master/klien/{id}", s.guardMP(s.handleUbahKlien, master.KlienUbah))
	m.HandleFunc("POST /api/v1/master/klien/{id}/arsip", s.guardMP(func(w http.ResponseWriter, r *http.Request) { s.ubahArsip(w, r, true) }, master.KlienUbah))
	m.HandleFunc("DELETE /api/v1/master/klien/{id}/arsip", s.guardMP(func(w http.ResponseWriter, r *http.Request) { s.ubahArsip(w, r, false) }, master.KlienUbah))
	m.HandleFunc("DELETE /api/v1/master/klien/{id}", s.guardMP(s.handleHapusKlien, master.HapusPermanen))
	m.HandleFunc("GET /api/v1/master/pengguna", s.guardMP(s.handleMasterPengguna, master.PenggunaLihat))
	m.HandleFunc("PUT /api/v1/master/pengguna/{id}", s.guardMP(s.handleUbahPengguna, master.PenggunaKelola))
	m.HandleFunc("POST /api/v1/master/pengguna/{id}/reset-password", s.guardMP(s.handleResetPassword, master.PenggunaReset))
	m.HandleFunc("DELETE /api/v1/master/pengguna/{id}", s.guardMP(s.handleHapusPengguna, master.HapusPermanen))
	m.HandleFunc("DELETE /api/v1/master/tim/{id}", s.guardMP(s.handleHapusAnggota, master.HapusPermanen))
	m.HandleFunc("GET /api/v1/akses-tim", s.guard("pengguna", s.handleRiwayatAksesTim))
	s.catatanRoutes()
	s.jadwalRoutes()
	s.kartuStokRoutes()
	s.umurRoutes()
	s.alurKerjaRoutes()
}
