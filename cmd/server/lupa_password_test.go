package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type surat struct{ ke, isi string }

func tangkapSurel(srv *Server) chan surat {
	ch := make(chan surat, 10)
	srv.kirimSurel = func(k, _, i string) error { ch <- surat{k, i}; return nil }
	return ch
}

func tokenDari(t *testing.T, ch chan surat) (string, string) {
	t.Helper()
	select {
	case s := <-ch:
		const awal = "https://contoh.id/#/atur-password?token="
		i := strings.Index(s.isi, awal)
		if i < 0 {
			t.Fatalf("tautan tidak ada: %q", s.isi)
		}
		tok, _ := url.QueryUnescape(strings.Fields(s.isi[i+len(awal):])[0])
		return s.ke, tok
	case <-time.After(2 * time.Second):
		t.Fatal("email tidak terkirim")
		return "", ""
	}
}

// Checklist tester No 15: lupa password dari halaman Masuk.
func TestLupaPassword(t *testing.T) {
	t.Setenv("APP_URL", "https://contoh.id/")
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	ch := tangkapSurel(srv)

	tidakAda := do(srv, nil, http.MethodPost, "/api/v1/auth/lupa-password", `{"email":"tidak-ada@contoh.id"}`)
	ada := do(srv, nil, http.MethodPost, "/api/v1/auth/lupa-password", `{"email":"Owner@KedaiKopiSenja.id"}`)
	if tidakAda.Code != http.StatusAccepted || ada.Code != http.StatusAccepted || tidakAda.Body.String() != ada.Body.String() {
		t.Fatalf("balasan harus sama: %d %q / %d %q", tidakAda.Code, tidakAda.Body, ada.Code, ada.Body)
	}
	ke, lama := tokenDari(t, ch)
	if ke != "owner@kedaikopisenja.id" {
		t.Fatalf("ke %q", ke)
	}
	// permintaan baru membatalkan tautan lama
	do(srv, nil, http.MethodPost, "/api/v1/auth/lupa-password", `{"email":"owner@kedaikopisenja.id"}`)
	_, token := tokenDari(t, ch)
	if rec := do(srv, nil, http.MethodPost, "/api/v1/auth/atur-password", `{"token":"`+lama+`","password":"password-baru-9"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("token lama harus gugur: %d", rec.Code)
	}
	if rec := do(srv, nil, http.MethodPost, "/api/v1/auth/atur-password", `{"token":"`+token+`","password":"pendek"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("password pendek: %d", rec.Code)
	}
	if rec := do(srv, nil, http.MethodPost, "/api/v1/auth/atur-password", `{"token":"`+token+`","password":"password-baru-9"}`); rec.Code != http.StatusNoContent {
		t.Fatalf("atur password: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, nil, http.MethodPost, "/api/v1/auth/atur-password", `{"token":"`+token+`","password":"password-baru-9"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("token sekali pakai: %d", rec.Code)
	}
	// password baru tersimpan: tetap berlaku setelah server dimuat ulang
	srv2 := NewServer()
	if rec := do(srv2, nil, http.MethodPost, "/api/v1/auth/login", `{"email":"owner@kedaikopisenja.id","password":"password-baru-9"}`); rec.Code != http.StatusOK {
		t.Fatalf("login password baru setelah muat ulang: %d", rec.Code)
	}
}

// Batas per email tidak memberi 429 (tidak bisa dipakai mengunci reset orang lain / menebak akun):
// balasan tetap sama, hanya email berhenti dikirim.
func TestLupaPassword_BatasPerEmailDiam(t *testing.T) {
	t.Setenv("APP_URL", "https://contoh.id")
	srv := NewServer()
	ch := tangkapSurel(srv)
	for i := range 7 {
		req := `{"email":"admin@kedaikopisenja.id"}`
		rec := do(srv, nil, http.MethodPost, "/api/v1/auth/lupa-password", req)
		if rec.Code != http.StatusAccepted && !(i >= 5 && rec.Code == http.StatusTooManyRequests) {
			t.Fatalf("permintaan %d: %d", i+1, rec.Code)
		}
	}
	time.Sleep(200 * time.Millisecond)
	if n := len(ch); n > 5 {
		t.Fatalf("email terkirim %d kali, batas 5", n)
	}
}
