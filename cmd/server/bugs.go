package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
	// Screenshot: nama berkas di data/bugs/; kosong = tanpa lampiran.
	Screenshot string `json:"screenshot,omitempty"`
}

type bugStore struct {
	mu   sync.Mutex
	path string
	list []bugReport
	mem  map[string][]byte // screenshot saat tanpa DATA_DIR (test, dev lokal)
}

// screenshotExt: hanya gambar yang diterima, ditentukan dari isi berkas (bukan nama).
var screenshotExt = map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/webp": ".webp", "image/gif": ".gif"}

func (b *bugStore) dir() string {
	if b.path == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(b.path), "bugs")
}

func loadBugs() *bugStore {
	b := &bugStore{path: dataPath("bugs.json"), mem: map[string][]byte{}}
	if raw, err := os.ReadFile(b.path); err == nil {
		if err := json.Unmarshal(raw, &b.list); err != nil {
			log.Fatalf("bugs.json rusak, perbaiki atau pindahkan dulu: %v", err)
		}
	}
	return b
}

func (s *Server) handleCreateBug(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		http.Error(w, "kirim sebagai form; screenshot maks 5 MB", http.StatusBadRequest)
		return
	}
	judul, deskripsi := strings.TrimSpace(r.FormValue("judul")), strings.TrimSpace(r.FormValue("deskripsi"))
	if judul == "" || deskripsi == "" {
		http.Error(w, "judul dan deskripsi wajib diisi", http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(judul) > 200 || utf8.RuneCountInString(deskripsi) > 5000 {
		http.Error(w, "judul maks 200 karakter, deskripsi maks 5000 karakter", http.StatusBadRequest)
		return
	}
	var img []byte
	var ext string
	if f, _, err := r.FormFile("screenshot"); err == nil {
		defer f.Close()
		img, err = io.ReadAll(io.LimitReader(f, 5<<20+1))
		if err != nil || len(img) > 5<<20 {
			http.Error(w, "screenshot maks 5 MB", http.StatusBadRequest)
			return
		}
		var ok bool
		if ext, ok = screenshotExt[http.DetectContentType(img)]; !ok {
			http.Error(w, "screenshot harus gambar PNG, JPG, WEBP, atau GIF", http.StatusBadRequest)
			return
		}
	}

	u := currentUser(r)
	b := s.bugs
	b.mu.Lock()
	defer b.mu.Unlock()
	rep := bugReport{
		ID: len(b.list) + 1, TenantID: u.TenantID, UserEmail: u.Email,
		Judul: judul, Deskripsi: deskripsi, Halaman: r.FormValue("halaman"),
		UserAgent: r.UserAgent(), Waktu: time.Now(),
	}
	if img != nil {
		rep.Screenshot = strconv.Itoa(rep.ID) + ext
		if d := b.dir(); d != "" {
			if err := os.MkdirAll(d, 0o700); err != nil || os.WriteFile(filepath.Join(d, rep.Screenshot), img, 0o600) != nil {
				http.Error(w, "gagal menyimpan screenshot", http.StatusInternalServerError)
				return
			}
		} else {
			b.mem[rep.Screenshot] = img
		}
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

// handleBugScreenshot: hanya laporan milik usaha user sendiri; usaha lain dapat 404.
func (s *Server) handleBugScreenshot(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	b := s.bugs
	b.mu.Lock()
	var name string
	if id >= 1 && id <= len(b.list) && b.list[id-1].TenantID == currentUser(r).TenantID {
		name = b.list[id-1].Screenshot
	}
	img := b.mem[name]
	b.mu.Unlock()
	if name == "" {
		http.NotFound(w, r)
		return
	}
	if d := b.dir(); d != "" {
		var err error
		if img, err = os.ReadFile(filepath.Join(d, name)); err != nil {
			http.NotFound(w, r)
			return
		}
	}
	w.Header().Set("Content-Type", http.DetectContentType(img))
	w.Header().Set("Cache-Control", "private, max-age=86400")
	_, _ = w.Write(img)
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
