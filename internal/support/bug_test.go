package support_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pajak/internal/support"
)

func storeTetap() *support.Store {
	s := support.NewStore()
	s.Now = func() time.Time { return time.Date(2026, 10, 19, 9, 30, 0, 0, time.UTC) }
	return s
}

func laporan(judul string, tenant int) support.Laporan {
	return support.Laporan{
		Judul: judul, Deskripsi: "Angka di laporan tidak cocok dengan transaksi.",
		Keparahan: support.Mengganggu, TenantID: tenant, Email: "owner@contoh.id",
	}
}

func TestCreate_MengisiIDWaktuStatus(t *testing.T) {
	s := storeTetap()
	l, err := s.Create(laporan("Laba rugi beda", 1))
	if err != nil {
		t.Fatal(err)
	}
	if l.ID != 1 || l.Status != support.StatusBaru {
		t.Errorf("laporan baru: %+v", l)
	}
	if l.Dibuat != "2026-10-19T09:30:00Z" {
		t.Errorf("waktu dibuat = %q", l.Dibuat)
	}
	l2, _ := s.Create(laporan("Lain", 1))
	if l2.ID != 2 {
		t.Errorf("ID harus berurut, got %d", l2.ID)
	}
}

func TestValidate(t *testing.T) {
	s := storeTetap()
	for _, c := range []struct {
		nama string
		l    support.Laporan
	}{
		{"tanpa judul", support.Laporan{Deskripsi: "cukup panjang untuk lolos"}},
		{"deskripsi terlalu pendek", support.Laporan{Judul: "Ada bug", Deskripsi: "rusak"}},
		{"judul kepanjangan", support.Laporan{Judul: strings.Repeat("a", 151), Deskripsi: "cukup panjang untuk lolos"}},
		{"deskripsi kepanjangan", support.Laporan{Judul: "Ada bug", Deskripsi: strings.Repeat("a", 4001)}},
	} {
		if _, err := s.Create(c.l); err == nil {
			t.Errorf("%s harus ditolak", c.nama)
		}
	}
	// Keparahan tak dikenal dibetulkan ke default, bukan ditolak: pelapor tidak perlu
	// terhalang hanya karena nilai dropdown tidak cocok.
	l, err := s.Create(support.Laporan{Judul: "Ada bug", Deskripsi: "cukup panjang untuk lolos", Keparahan: "ngawur"})
	if err != nil || l.Keparahan != support.Mengganggu {
		t.Errorf("keparahan tak dikenal → %q, err %v", l.Keparahan, err)
	}
}

// Pemilik usaha hanya boleh melihat laporan dari usahanya sendiri; ID Master melihat semua.
func TestList_DipisahPerUsaha(t *testing.T) {
	s := storeTetap()
	for _, c := range []struct {
		judul  string
		tenant int
	}{{"A1", 1}, {"A2", 1}, {"B1", 2}} {
		if _, err := s.Create(laporan(c.judul, c.tenant)); err != nil {
			t.Fatal(err)
		}
	}
	if n := len(s.List(1)); n != 2 {
		t.Errorf("usaha 1 melihat %d laporan, want 2", n)
	}
	if n := len(s.List(2)); n != 1 {
		t.Errorf("usaha 2 melihat %d laporan, want 1", n)
	}
	semua := s.List(0)
	if len(semua) != 3 {
		t.Fatalf("ID Master melihat %d laporan, want 3", len(semua))
	}
	// Terbaru lebih dulu.
	if semua[0].Judul != "B1" {
		t.Errorf("urutan: yang terbaru harus di atas, got %q", semua[0].Judul)
	}
}

func TestTanggapi(t *testing.T) {
	s := storeTetap()
	l, _ := s.Create(laporan("Bug", 1))
	got, err := s.Tanggapi(l.ID, support.StatusSelesai, " sudah diperbaiki di versi berikutnya ")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != support.StatusSelesai || got.Balasan != "sudah diperbaiki di versi berikutnya" {
		t.Errorf("tanggapan: %+v", got)
	}
	if _, err := s.Tanggapi(999, support.StatusDibaca, ""); err != support.ErrNotFound {
		t.Errorf("laporan tidak ada → %v", err)
	}
	if _, err := s.Tanggapi(l.ID, "ngawur", ""); err == nil {
		t.Error("status tak dikenal harus ditolak")
	}
}

func TestSnapshotRestore(t *testing.T) {
	s := storeTetap()
	s.Create(laporan("A", 1))
	s.Create(laporan("B", 2))
	snap := s.Snapshot()

	lain := support.NewStore()
	lain.Restore(snap)
	if len(lain.List(0)) != 2 {
		t.Fatalf("restore kehilangan laporan")
	}
	// Deret ID berlanjut setelah restart, tidak mulai dari 1 lagi menimpa yang ada.
	l, _ := lain.Create(laporan("C", 1))
	if l.ID != 3 {
		t.Errorf("ID setelah restore = %d, want 3", l.ID)
	}
}

func TestRekap(t *testing.T) {
	s := storeTetap()
	a, _ := s.Create(laporan("A", 1))
	s.Create(laporan("B", 1))
	s.Tanggapi(a.ID, support.StatusSelesai, "")
	r := s.Rekap(1)
	if r.Total != 2 || r.PerStatus[support.StatusBaru] != 1 || r.PerStatus[support.StatusSelesai] != 1 {
		t.Errorf("rekap: %+v", r)
	}
}

// Tanpa BUG_WEBHOOK_URL, penerusan mati dan tidak ada yang dikirim ke mana pun.
func TestPengirim_TidakAktifTanpaURL(t *testing.T) {
	t.Setenv("BUG_WEBHOOK_URL", "")
	p := support.NewPengirim()
	if p.Aktif() {
		t.Fatal("tanpa URL penerusan harus mati")
	}
	if err := p.Kirim(context.Background(), laporan("A", 1)); err != support.ErrTidakAktif {
		t.Errorf("Kirim tanpa URL → %v", err)
	}
	// URL non-HTTP diabaikan, bukan dipakai.
	t.Setenv("BUG_WEBHOOK_URL", "file:///etc/passwd")
	if support.NewPengirim().Aktif() {
		t.Error("URL non-HTTP tidak boleh dianggap aktif")
	}
}

func TestPengirim_KirimKeWebhook(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	t.Setenv("BUG_WEBHOOK_URL", srv.URL)
	p := support.NewPengirim()
	if !p.Aktif() {
		t.Fatal("harus aktif")
	}
	l := laporan("Laba rugi beda", 1)
	l.ID = 7
	if err := p.Kirim(context.Background(), l); err != nil {
		t.Fatal(err)
	}
	// "text" untuk Slack, "content" untuk Discord, "laporan" untuk endpoint sendiri.
	for _, k := range []string{"text", "content", "laporan"} {
		if _, ok := body[k]; !ok {
			t.Errorf("payload tidak punya %q: %v", k, body)
		}
	}
	if teks, _ := body["text"].(string); !strings.Contains(teks, "Laba rugi beda") || !strings.Contains(teks, "#7") {
		t.Errorf("ringkasan tidak memuat judul & nomor: %q", teks)
	}
}

func TestPengirim_WebhookGagal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("BUG_WEBHOOK_URL", srv.URL)
	err := support.NewPengirim().Kirim(context.Background(), laporan("A", 1))
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("kegagalan webhook harus dilaporkan: %v", err)
	}
	// Laporan tetap tersimpan walau penerusan gagal — itu yang dicatat TandaiTerkirim.
	s := storeTetap()
	l, _ := s.Create(laporan("A", 1))
	s.TandaiTerkirim(l.ID, err)
	got := s.List(1)[0]
	if got.Diteruskan || got.GagalKirim == "" {
		t.Errorf("kegagalan harus tercatat di laporan: %+v", got)
	}
	s.TandaiTerkirim(l.ID, nil)
	if got := s.List(1)[0]; !got.Diteruskan || got.GagalKirim != "" {
		t.Errorf("setelah berhasil, catatan gagal harus bersih: %+v", got)
	}
}

// Tujuan hanya menyebut host: webhook Slack/Discord memuat token rahasia di path-nya.
func TestPengirim_TujuanTidakMembocorkanToken(t *testing.T) {
	t.Setenv("BUG_WEBHOOK_URL", "https://hooks.slack.com/services/T000/B000/XXXXrahasiaXXXX")
	tujuan := support.NewPengirim().Tujuan()
	if tujuan != "hooks.slack.com" {
		t.Errorf("Tujuan() = %q, want hooks.slack.com", tujuan)
	}
	if strings.Contains(tujuan, "rahasia") {
		t.Error("token di path tidak boleh ikut ditampilkan")
	}
}
