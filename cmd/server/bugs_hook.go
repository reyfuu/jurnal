package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Penerusan laporan bug ke pengembang.
//
// bugs.go sudah menyimpan laporan ke bugs.json, dan itu memang dilakukan LEBIH DULU: penerusan
// bisa gagal karena jaringan atau URL salah, dan laporan yang hilang karena itu hilang
// selamanya sementara pelapor sudah merasa melapor. Jadi penerusan hanya menambah jalur
// pengiriman, tidak pernah menjadi satu-satunya.
//
// Slot-nya disiapkan tapi dibiarkan kosong: tanpa BUG_WEBHOOK_URL tidak ada data yang keluar
// dari server, dan laporan tetap terbaca di halaman Lapor Bug.

type bugHook struct {
	url    string
	client *http.Client
}

// newBugHook membaca BUG_WEBHOOK_URL. URL non-HTTP(S) diabaikan, bukan dipakai.
func newBugHook() *bugHook {
	u := strings.TrimSpace(os.Getenv("BUG_WEBHOOK_URL"))
	if u != "" && !strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "http://") {
		u = ""
	}
	timeout := 10 * time.Second
	if d, err := time.ParseDuration(os.Getenv("BUG_WEBHOOK_TIMEOUT")); err == nil && d > 0 {
		timeout = d
	}
	return &bugHook{url: u, client: &http.Client{Timeout: timeout}}
}

func (h *bugHook) aktif() bool { return h != nil && h.url != "" }

// tujuan hanya menyebut host, bukan URL penuh: webhook Slack/Discord memuat token di path-nya,
// dan halaman Lapor Bug menampilkan nilai ini ke pengguna.
func (h *bugHook) tujuan() string {
	if !h.aktif() {
		return ""
	}
	s := strings.TrimPrefix(strings.TrimPrefix(h.url, "https://"), "http://")
	if i := strings.IndexByte(s, '/'); i > 0 {
		return s[:i]
	}
	return s
}

func ringkasBug(b bugReport, punyaScreenshot bool) string {
	var w strings.Builder
	fmt.Fprintf(&w, "🐞 Laporan bug #%d — %s\n", b.ID, b.Judul)
	fmt.Fprintf(&w, "Pelapor: %s (usaha %d)\n", b.UserEmail, b.TenantID)
	if b.Halaman != "" {
		fmt.Fprintf(&w, "Halaman: %s\n", b.Halaman)
	}
	fmt.Fprintf(&w, "\n%s\n", b.Deskripsi)
	if punyaScreenshot {
		fmt.Fprintf(&w, "\n(ada lampiran screenshot — buka di menu Lapor Bug)\n")
	}
	if b.UserAgent != "" {
		fmt.Fprintf(&w, "\n%s", b.UserAgent)
	}
	return w.String()
}

// kirim meneruskan laporan. Gambarnya tidak ikut dikirim: webhook Slack/Discord tidak menerima
// lampiran lewat jalur ini, dan mengirim gambar pengguna ke pihak luar tanpa itu diperlukan
// bukan hal yang perlu dilakukan diam-diam.
func (h *bugHook) kirim(ctx context.Context, b bugReport) error {
	teks := ringkasBug(b, b.Screenshot != "")
	body, err := json.Marshal(map[string]any{"text": teks, "content": teks, "laporan": b})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4<<10))
	if res.StatusCode >= 300 {
		return fmt.Errorf("webhook menjawab %s", res.Status)
	}
	return nil
}

// masterEmail: pemilik "ID Master" (pengembang) yang boleh membaca laporan seluruh usaha.
// Kosong = tidak ada yang punya akses lintas usaha; laporan tetap tersimpan dan diteruskan.
func masterEmail() string { return strings.ToLower(strings.TrimSpace(os.Getenv("MASTER_EMAIL"))) }

func isMaster(r *http.Request) bool {
	m := masterEmail()
	return m != "" && strings.EqualFold(currentUser(r).Email, m)
}

// bugInfo menyertai daftar laporan: ke mana laporan dikirim dan apakah yang login ID Master.
// Ditampilkan ke pelapor supaya jelas laporannya sampai ke mana — bukan sekadar "terkirim".
type bugInfo struct {
	Master bool `json:"master"`
	// PenerusanAktif false = laporan hanya tersimpan di server ini.
	PenerusanAktif bool   `json:"penerusan_aktif"`
	PenerusanKe    string `json:"penerusan_ke"`
}
