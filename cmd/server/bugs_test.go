package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// png1x1: PNG valid terkecil untuk uji lampiran.
var png1x1 = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\rIDATx\x9cc\xf8\x0f\x00\x00\x01\x01\x00\x05\x18\xd8N\x00\x00\x00\x00IEND\xaeB`\x82")

// postBug mengirim laporan sebagai multipart; img nil = tanpa screenshot.
func postBug(srv *Server, c *http.Cookie, fields map[string]string, img []byte) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	if img != nil {
		fw, _ := mw.CreateFormFile("screenshot", "layar.png")
		_, _ = fw.Write(img)
	}
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bugs", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(c)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func TestBugReportPerTenantAndPersisted(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	kasir := login(t, srv, "kasir@kedaikopisenja.id")
	if rec := postBug(srv, kasir, map[string]string{"judul": " ", "deskripsi": "x"}, nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("judul kosong: want 400, got %d", rec.Code)
	}
	if rec := postBug(srv, kasir, map[string]string{"judul": "a", "deskripsi": "b"}, []byte("bukan gambar")); rec.Code != http.StatusBadRequest {
		t.Fatalf("lampiran bukan gambar: want 400, got %d", rec.Code)
	}
	if rec := postBug(srv, kasir, map[string]string{"judul": "Tombol simpan mati", "deskripsi": "Tidak terjadi apa-apa", "halaman": "transaksi"}, png1x1); rec.Code != http.StatusCreated {
		t.Fatalf("lapor bug: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, kasir, http.MethodGet, "/api/v1/bugs/1/screenshot", ""); rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" || !bytes.Equal(rec.Body.Bytes(), png1x1) {
		t.Fatalf("unduh screenshot: %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	qa := register(t, srv, registerBody)
	if body := do(srv, qa, http.MethodGet, "/api/v1/bugs", "").Body.String(); strings.Contains(body, "Tombol simpan") {
		t.Fatal("laporan bug usaha lain bocor")
	}
	if rec := do(srv, qa, http.MethodGet, "/api/v1/bugs/1/screenshot", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("screenshot usaha lain: want 404, got %d", rec.Code)
	}

	var list []bugReport
	srv2 := NewServer()
	_ = json.NewDecoder(do(srv2, login(t, srv2, "owner@kedaikopisenja.id"), http.MethodGet, "/api/v1/bugs", "").Body).Decode(&list)
	if len(list) != 1 || list[0].UserEmail != "kasir@kedaikopisenja.id" || list[0].Halaman != "transaksi" || list[0].Screenshot != "1.png" {
		t.Fatalf("laporan harus bertahan setelah restart: %+v", list)
	}
}

func TestCekAwalEndpoint(t *testing.T) {
	srv := NewServer()
	qa := register(t, srv, registerBody)
	rec := do(srv, qa, http.MethodPost, "/api/v1/business/cek-awal", `{"jenis_usaha":"perdagangan","bentuk_usaha":"op","ptkp":"TK/0","klu":"471","omzet_tahunan":6000000000,"laba_sebelum_pajak":60000000}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"di_bawah"`) || !strings.Contains(rec.Body.String(), "Wajib menjadi PKP") {
		t.Fatalf("cek awal: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, qa, http.MethodPut, "/api/v1/business", `{"nama_usaha":"X","jenis_usaha":"jasa","bentuk_usaha":"cv","klu":"999"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("KLU tidak dikenal: want 400, got %d", rec.Code)
	}
	if body := do(srv, qa, http.MethodGet, "/api/v1/business", "").Body.String(); !strings.Contains(body, `"kode":"471"`) {
		t.Fatal("options.klu harus berisi daftar KLU")
	}
	if rec := do(srv, nil, http.MethodPost, "/api/v1/business/cek-awal", `{}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa login: want 401, got %d", rec.Code)
	}
}

// Laporan harus benar-benar sampai ke pengembang, bukan cuma mengendap di bugs.json.
func TestBugDiteruskanKeWebhook(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	var diterima map[string]any
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&diterima)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer hook.Close()
	t.Setenv("BUG_WEBHOOK_URL", hook.URL)

	srv := NewServer()
	kasir := login(t, srv, "kasir@kedaikopisenja.id")
	rec := postBug(srv, kasir, map[string]string{"judul": "Tombol simpan mati", "deskripsi": "Tidak terjadi apa-apa", "halaman": "transaksi"}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("lapor bug: %d %s", rec.Code, rec.Body)
	}
	var rep bugReport
	_ = json.NewDecoder(rec.Body).Decode(&rep)
	if !rep.Diteruskan || rep.GagalKirim != "" {
		t.Fatalf("harus diteruskan: %+v", rep)
	}
	// "text" untuk Slack, "content" untuk Discord, "laporan" untuk endpoint sendiri.
	for _, k := range []string{"text", "content", "laporan"} {
		if _, ok := diterima[k]; !ok {
			t.Errorf("payload webhook tidak punya %q", k)
		}
	}
	if teks, _ := diterima["text"].(string); !strings.Contains(teks, "Tombol simpan mati") {
		t.Errorf("ringkasan tidak memuat judul: %q", teks)
	}
	// Status penerusan ikut tersimpan, jadi masih terbaca setelah restart.
	var list []bugReport
	srv2 := NewServer()
	_ = json.NewDecoder(do(srv2, login(t, srv2, "kasir@kedaikopisenja.id"), http.MethodGet, "/api/v1/bugs", "").Body).Decode(&list)
	if len(list) != 1 || !list[0].Diteruskan {
		t.Fatalf("status penerusan harus ikut tersimpan: %+v", list)
	}
}

// Webhook gagal tidak boleh membuat pelaporan ikut gagal: laporannya sudah tersimpan.
func TestBugWebhookGagalTetapTersimpan(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer hook.Close()
	t.Setenv("BUG_WEBHOOK_URL", hook.URL)

	srv := NewServer()
	kasir := login(t, srv, "kasir@kedaikopisenja.id")
	rec := postBug(srv, kasir, map[string]string{"judul": "Angka salah", "deskripsi": "Laba rugi tidak cocok"}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("pelaporan harus tetap berhasil walau penerusan gagal: %d %s", rec.Code, rec.Body)
	}
	var rep bugReport
	_ = json.NewDecoder(rec.Body).Decode(&rep)
	if rep.Diteruskan || !strings.Contains(rep.GagalKirim, "500") {
		t.Fatalf("kegagalan harus dilaporkan apa adanya: %+v", rep)
	}
	var list []bugReport
	_ = json.NewDecoder(do(srv, kasir, http.MethodGet, "/api/v1/bugs", "").Body).Decode(&list)
	if len(list) != 1 {
		t.Fatalf("laporan tidak boleh hilang: %+v", list)
	}
}

// Tanpa BUG_WEBHOOK_URL tidak ada data yang keluar dari server.
func TestBugInfo_TanpaWebhook(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("BUG_WEBHOOK_URL", "")
	srv := NewServer()
	var info bugInfo
	_ = json.NewDecoder(do(srv, login(t, srv, "kasir@kedaikopisenja.id"), http.MethodGet, "/api/v1/bugs/info", "").Body).Decode(&info)
	if info.PenerusanAktif || info.PenerusanKe != "" || info.Master {
		t.Fatalf("tanpa webhook & MASTER_EMAIL: %+v", info)
	}
	// URL non-HTTP diabaikan, bukan dipakai.
	t.Setenv("BUG_WEBHOOK_URL", "file:///etc/passwd")
	if newBugHook().aktif() {
		t.Error("URL non-HTTP tidak boleh dianggap aktif")
	}
}

// Tujuan hanya menyebut host: webhook Slack/Discord memuat token di path-nya, dan nilai ini
// ditampilkan ke pengguna.
func TestBugHook_TujuanTidakMembocorkanToken(t *testing.T) {
	t.Setenv("BUG_WEBHOOK_URL", "https://hooks.slack.com/services/T000/B000/XXXXrahasiaXXXX")
	if got := newBugHook().tujuan(); got != "hooks.slack.com" {
		t.Errorf("tujuan() = %q, want hooks.slack.com", got)
	}
}

// ID Master membaca laporan seluruh usaha; pengguna biasa tetap hanya usahanya sendiri.
func TestBugMasterLihatSemuaUsaha(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("MASTER_EMAIL", "owner@kedaikopisenja.id")
	srv := NewServer()
	kasir := login(t, srv, "kasir@kedaikopisenja.id")
	postBug(srv, kasir, map[string]string{"judul": "Dari usaha satu", "deskripsi": "Keterangan cukup panjang"}, png1x1)
	qa := register(t, srv, registerBody)
	postBug(srv, qa, map[string]string{"judul": "Dari usaha dua", "deskripsi": "Keterangan cukup panjang"}, nil)

	master := login(t, srv, "owner@kedaikopisenja.id")
	var list []bugReport
	_ = json.NewDecoder(do(srv, master, http.MethodGet, "/api/v1/bugs", "").Body).Decode(&list)
	if len(list) != 2 {
		t.Fatalf("ID Master harus melihat laporan semua usaha, got %d", len(list))
	}
	var info bugInfo
	_ = json.NewDecoder(do(srv, master, http.MethodGet, "/api/v1/bugs/info", "").Body).Decode(&info)
	if !info.Master {
		t.Error("owner yang cocok MASTER_EMAIL harus ditandai master")
	}
	// Lampiran usaha lain ikut bisa dibuka ID Master, tapi tidak oleh usaha lain.
	if rec := do(srv, master, http.MethodGet, "/api/v1/bugs/1/screenshot", ""); rec.Code != http.StatusOK {
		t.Errorf("ID Master buka screenshot: %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodGet, "/api/v1/bugs/1/screenshot", ""); rec.Code != http.StatusNotFound {
		t.Errorf("usaha lain tetap 404, got %d", rec.Code)
	}
	// Tanpa MASTER_EMAIL, owner yang sama kembali hanya melihat usahanya.
	t.Setenv("MASTER_EMAIL", "")
	var biasa []bugReport
	_ = json.NewDecoder(do(srv, master, http.MethodGet, "/api/v1/bugs", "").Body).Decode(&biasa)
	if len(biasa) != 1 {
		t.Fatalf("tanpa MASTER_EMAIL harus kembali per usaha, got %d", len(biasa))
	}
}
