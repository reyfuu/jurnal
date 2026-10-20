package main

import (
	"net/http"
	"strings"
	"time"

	"pajak/internal/auth"
	"pajak/internal/master"
)

// Lihat sebagai klien: anggota tim membuka data satu usaha apa adanya, HANYA BACA, selama
// lihatDurasi. Alasan wajib dan tercatat di audit log. Selama aktif, permintaan akun tim ke
// endpoint data usaha dijalankan sebagai user sementara ber-role auth.Lihat atas usaha itu
// (lihat guard); permintaan pengubah dan ekspor ditolak.

const lihatDurasi = 2 * time.Hour

type lihatGrant struct {
	KlienID int
	Alasan  string
	Sampai  time.Time
}

// lihatInfo menyertai /auth/me selama mode lihat-sebagai.
type lihatInfo struct {
	KlienID int       `json:"klien_id"`
	Nama    string    `json:"nama"`
	Alasan  string    `json:"alasan"`
	Sampai  time.Time `json:"sampai"`
}

type lihatKey struct{}

// sedangLihat: request ini berjalan dalam mode lihat-sebagai.
func sedangLihat(r *http.Request) bool { return r.Context().Value(lihatKey{}) != nil }

// lihatUntuk: izin lihat-sebagai milik user yang masih berlaku. Penugasan & izin dicek ulang di
// setiap request, jadi mencabutnya di Tim & Akses langsung menghentikan akses.
func (s *Server) lihatUntuk(userID int, now time.Time) (lihatGrant, bool) {
	s.masterMu.Lock()
	defer s.masterMu.Unlock()
	g, ok := s.lihat[userID]
	if !ok {
		return g, false
	}
	ak, anggota := s.master.Akses(userID)
	if !now.Before(g.Sampai) || !anggota || !ak.Bisa(master.LihatSebagai) || !ak.BolehKlien(g.KlienID) {
		delete(s.lihat, userID)
		return g, false
	}
	return g, true
}

// sebagaiKlien: user sementara untuk endpoint data usaha selama lihat-sebagai.
func sebagaiKlien(u auth.User, g lihatGrant) auth.User {
	return auth.User{ID: u.ID, Name: u.Name, Email: u.Email, Role: auth.Lihat, Active: true, TenantID: g.KlienID}
}

func (s *Server) lihatInfoFor(u auth.User) *lihatInfo {
	g, ok := s.lihatUntuk(u.ID, time.Now())
	if !ok {
		return nil
	}
	s.tenantsMu.RLock()
	t := s.tenants[g.KlienID]
	s.tenantsMu.RUnlock()
	nama := ""
	if t != nil {
		t.mu.RLock()
		if t.profile != nil {
			nama = t.profile.NamaUsaha
		}
		nama = namaKlien(t, nama)
		t.mu.RUnlock()
	}
	return &lihatInfo{g.KlienID, nama, g.Alasan, g.Sampai}
}

func (s *Server) handleMulaiLihat(w http.ResponseWriter, r *http.Request) {
	t := s.klienDari(w, r)
	if t == nil {
		return
	}
	var req struct{ Alasan string }
	if !decodeBody(w, r, &req) {
		return
	}
	req.Alasan = clip(strings.TrimSpace(req.Alasan), 300)
	if req.Alasan == "" {
		http.Error(w, "alasan wajib diisi", http.StatusBadRequest)
		return
	}
	if err := s.catat(r, "Lihat sebagai klien", t.id, req.Alasan); err != nil {
		writeMasterErr(w, err)
		return
	}
	g := lihatGrant{t.id, req.Alasan, time.Now().Add(lihatDurasi)}
	s.masterMu.Lock()
	s.lihat[currentUser(r).ID] = g
	s.masterMu.Unlock()
	writeJSON(w, s.lihatInfoFor(currentUser(r)))
}

func (s *Server) handleSelesaiLihat(w http.ResponseWriter, r *http.Request) {
	s.masterMu.Lock()
	delete(s.lihat, currentUser(r).ID)
	s.masterMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}
