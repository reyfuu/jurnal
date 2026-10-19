package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Laporan bug dari menu Lapor Bug. Satu berkas global (bugs.json) untuk semua usaha supaya
// operator cukup membaca satu tempat di VPS; tiap usaha hanya melihat laporannya sendiri.

type bugReport struct {
	ID        int       `json:"id"`
	TenantID  int       `json:"tenant_id"`
	UserEmail string    `json:"user_email"`
	Judul     string    `json:"judul"`
	Deskripsi string    `json:"deskripsi"`
	Halaman   string    `json:"halaman"`
	UserAgent string    `json:"user_agent"`
	Waktu     time.Time `json:"waktu"`
}

type bugStore struct {
	mu   sync.Mutex
	path string
	list []bugReport
}

func loadBugs() *bugStore {
	b := &bugStore{path: dataPath("bugs.json")}
	if raw, err := os.ReadFile(b.path); err == nil {
		if err := json.Unmarshal(raw, &b.list); err != nil {
			log.Fatalf("bugs.json rusak, perbaiki atau pindahkan dulu: %v", err)
		}
	}
	return b
}

func (s *Server) handleCreateBug(w http.ResponseWriter, r *http.Request) {
	var req struct{ Judul, Deskripsi, Halaman string }
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	req.Judul, req.Deskripsi = strings.TrimSpace(req.Judul), strings.TrimSpace(req.Deskripsi)
	if req.Judul == "" || req.Deskripsi == "" {
		http.Error(w, "judul dan deskripsi wajib diisi", http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(req.Judul) > 200 || utf8.RuneCountInString(req.Deskripsi) > 5000 {
		http.Error(w, "judul maks 200 karakter, deskripsi maks 5000 karakter", http.StatusBadRequest)
		return
	}
	u := currentUser(r)
	b := s.bugs
	b.mu.Lock()
	defer b.mu.Unlock()
	rep := bugReport{
		ID: len(b.list) + 1, TenantID: u.TenantID, UserEmail: u.Email,
		Judul: req.Judul, Deskripsi: req.Deskripsi, Halaman: req.Halaman,
		UserAgent: r.UserAgent(), Waktu: time.Now(),
	}
	b.list = append(b.list, rep)
	if b.path != "" {
		if err := writeFileAtomic(b.path, b.list); err != nil {
			b.list = b.list[:len(b.list)-1]
			http.Error(w, "gagal menyimpan laporan", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, rep)
}

func (s *Server) handleListBugs(w http.ResponseWriter, r *http.Request) {
	tid := currentUser(r).TenantID
	b := s.bugs
	b.mu.Lock()
	out := []bugReport{}
	for i := len(b.list) - 1; i >= 0; i-- { // terbaru dulu
		if b.list[i].TenantID == tid {
			out = append(out, b.list[i])
		}
	}
	b.mu.Unlock()
	writeJSON(w, out)
}
