package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"pajak/internal/support"
)

// Menu Lapor Bug. Laporan selalu disimpan di server lebih dulu; penerusan ke pengembang
// lewat BUG_WEBHOOK_URL menyusul dan kegagalannya tidak membuat pelaporan ikut gagal —
// laporan yang sudah tersimpan masih bisa dibaca ID Master di aplikasi.

// masterEmail: pemilik "ID Master" (pengembang) yang boleh melihat laporan seluruh usaha dan
// menanggapinya. Kosong = tidak ada yang punya akses lintas usaha; laporan tetap tersimpan.
func masterEmail() string { return strings.ToLower(strings.TrimSpace(os.Getenv("MASTER_EMAIL"))) }

func isMaster(r *http.Request) bool {
	m := masterEmail()
	return m != "" && strings.EqualFold(currentUser(r).Email, m)
}

type bugInfo struct {
	Laporan []support.Laporan `json:"laporan"`
	Rekap   support.Rekap     `json:"rekap"`
	// Master true = yang login adalah ID Master, jadi daftarnya lintas usaha & bisa ditanggapi.
	Master bool `json:"master"`
	// Penerusan menjelaskan ke mana laporan dikirim; kosong = hanya tersimpan di server.
	PenerusanAktif bool     `json:"penerusan_aktif"`
	PenerusanKe    string   `json:"penerusan_ke"`
	Keparahan      []string `json:"keparahan"`
	StatusList     []string `json:"status_list"`
	// Konteks yang akan ikut terkirim, supaya pelapor tahu sebelum mengirim.
	Konteks map[string]string `json:"konteks"`
}

func (s *Server) handleBugInfo(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	master := isMaster(r)
	tenantID := u.TenantID
	if master {
		tenantID = 0 // 0 = semua usaha
	}
	writeJSON(w, bugInfo{
		Laporan:        s.bugs.List(tenantID),
		Rekap:          s.bugs.Rekap(tenantID),
		Master:         master,
		PenerusanAktif: s.bugHook.Aktif(),
		PenerusanKe:    s.bugHook.Tujuan(),
		Keparahan:      support.KeparahanList,
		StatusList:     support.StatusList,
		Konteks: map[string]string{
			"pelapor": u.Name, "email": u.Email, "peran": string(u.Role),
			"usaha": strconv.Itoa(u.TenantID), "versi": appVersion(),
		},
	})
}

type bugReq struct {
	Judul     string `json:"judul"`
	Deskripsi string `json:"deskripsi"`
	Langkah   string `json:"langkah"`
	Keparahan string `json:"keparahan"`
	Halaman   string `json:"halaman"`
}

func appVersion() string {
	if v := strings.TrimSpace(os.Getenv("APP_VERSION")); v != "" {
		return v
	}
	return "dev"
}

func (s *Server) handleCreateBug(w http.ResponseWriter, r *http.Request) {
	var req bugReq
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		http.Error(w, "data laporan tidak valid", http.StatusBadRequest)
		return
	}
	u := currentUser(r)
	l, err := s.bugs.Create(support.Laporan{
		Judul: req.Judul, Deskripsi: req.Deskripsi, Langkah: req.Langkah,
		Keparahan: req.Keparahan, Halaman: req.Halaman,
		Pelapor: u.Name, Email: u.Email, Peran: string(u.Role), TenantID: u.TenantID,
		// User-Agent dibaca dari header, tidak dari body: supaya tidak bisa dipalsukan
		// menjadi teks panjang sembarang oleh klien.
		UserAgent: potong(r.UserAgent(), 200),
		Versi:     appVersion(),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Penerusan dicoba di sini (bukan di latar) supaya pelapor langsung tahu apakah
	// laporannya sudah sampai ke pengembang atau baru tersimpan di server.
	if s.bugHook.Aktif() {
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		err := s.bugHook.Kirim(ctx, l)
		s.bugs.TandaiTerkirim(l.ID, err)
		l.Diteruskan = err == nil
		if err != nil {
			l.GagalKirim = err.Error()
		}
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, l)
}

func potong(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

type tanggapiReq struct {
	Status  string `json:"status"`
	Balasan string `json:"balasan"`
}

func (s *Server) handleTanggapiBug(w http.ResponseWriter, r *http.Request) {
	if !isMaster(r) {
		http.Error(w, "hanya ID Master (pengembang) yang bisa menanggapi laporan", http.StatusForbidden)
		return
	}
	var req tanggapiReq
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		http.Error(w, "data tanggapan tidak valid", http.StatusBadRequest)
		return
	}
	id, _ := strconv.Atoi(r.PathValue("id"))
	l, err := s.bugs.Tanggapi(id, req.Status, req.Balasan)
	if err != nil {
		code := http.StatusBadRequest
		if err == support.ErrNotFound {
			code = http.StatusNotFound
		}
		http.Error(w, err.Error(), code)
		return
	}
	writeJSON(w, l)
}
