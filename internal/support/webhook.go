package support

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Penerusan laporan ke pengembang. Slot-nya disiapkan tapi dibiarkan kosong: tanpa
// BUG_WEBHOOK_URL laporan tetap tersimpan di server dan bisa dibaca ID Master di aplikasi,
// hanya tidak dikirim ke mana-mana. Tidak ada data yang keluar dari server kecuali operator
// memang mengisi URL-nya sendiri.
//
// Payload memuat "text" dan "content" sekaligus supaya webhook Slack maupun Discord langsung
// menampilkannya, plus objek laporan utuh untuk endpoint sendiri.

type Pengirim struct {
	URL    string
	client *http.Client
}

// NewPengirim membaca BUG_WEBHOOK_URL. URL non-HTTP(S) diabaikan, bukan dipakai.
func NewPengirim() *Pengirim {
	u := strings.TrimSpace(os.Getenv("BUG_WEBHOOK_URL"))
	if u != "" && !strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "http://") {
		u = ""
	}
	timeout := 10 * time.Second
	if d, err := time.ParseDuration(os.Getenv("BUG_WEBHOOK_TIMEOUT")); err == nil && d > 0 {
		timeout = d
	}
	return &Pengirim{URL: u, client: &http.Client{Timeout: timeout}}
}

func (p *Pengirim) Aktif() bool { return p != nil && p.URL != "" }

// Tujuan menyebut host tujuan untuk ditampilkan ke pengguna — bukan URL penuhnya, karena
// webhook Slack/Discord memuat token di dalam path-nya.
func (p *Pengirim) Tujuan() string {
	if !p.Aktif() {
		return ""
	}
	s := strings.TrimPrefix(strings.TrimPrefix(p.URL, "https://"), "http://")
	if i := strings.IndexByte(s, '/'); i > 0 {
		return s[:i]
	}
	return s
}

var ErrTidakAktif = errors.New("penerusan laporan belum diaktifkan (BUG_WEBHOOK_URL kosong)")

func Ringkas(l Laporan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "🐞 Laporan bug #%d [%s] — %s\n", l.ID, l.Keparahan, l.Judul)
	fmt.Fprintf(&b, "Pelapor: %s <%s> (%s, usaha %d)\n", l.Pelapor, l.Email, l.Peran, l.TenantID)
	if l.Halaman != "" {
		fmt.Fprintf(&b, "Halaman: %s\n", l.Halaman)
	}
	fmt.Fprintf(&b, "\n%s\n", l.Deskripsi)
	if l.Langkah != "" {
		fmt.Fprintf(&b, "\nLangkah:\n%s\n", l.Langkah)
	}
	if l.Versi != "" || l.UserAgent != "" {
		fmt.Fprintf(&b, "\nVersi %s · %s", l.Versi, l.UserAgent)
	}
	return b.String()
}

func (p *Pengirim) Kirim(ctx context.Context, l Laporan) error {
	if !p.Aktif() {
		return ErrTidakAktif
	}
	teks := Ringkas(l)
	body, err := json.Marshal(map[string]any{"text": teks, "content": teks, "laporan": l})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := p.client.Do(req)
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
