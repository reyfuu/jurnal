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
