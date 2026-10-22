package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// Checklist tester No 15: lupa password dari halaman Masuk.
func TestLupaPassword(t *testing.T) {
	t.Setenv("APP_URL", "https://contoh.id/")
	srv := NewServer()
	var ke, isi string
	srv.kirimSurel = func(k, _, i string) error { ke, isi = k, i; return nil }

	tidakAda := do(srv, nil, http.MethodPost, "/api/v1/auth/lupa-password", `{"email":"tidak-ada@contoh.id"}`)
	ada := do(srv, nil, http.MethodPost, "/api/v1/auth/lupa-password", `{"email":"Owner@KedaiKopiSenja.id"}`)
	if tidakAda.Code != http.StatusAccepted || ada.Code != http.StatusAccepted || tidakAda.Body.String() != ada.Body.String() {
		t.Fatalf("balasan harus sama: %d %q / %d %q", tidakAda.Code, tidakAda.Body, ada.Code, ada.Body)
	}
	i := strings.Index(isi, "https://contoh.id/#/atur-password?token=")
	if ke != "owner@kedaikopisenja.id" || i < 0 {
		t.Fatalf("email: ke=%q isi=%q", ke, isi)
	}
	token, _ := url.QueryUnescape(strings.Fields(isi[i+len("https://contoh.id/#/atur-password?token="):])[0])
	if rec := do(srv, nil, http.MethodPost, "/api/v1/auth/atur-password", `{"token":"`+token+`","password":"pendek"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("password pendek: %d", rec.Code)
	}
	if rec := do(srv, nil, http.MethodPost, "/api/v1/auth/atur-password", `{"token":"`+token+`","password":"password-baru-9"}`); rec.Code != http.StatusNoContent {
		t.Fatalf("atur password: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, nil, http.MethodPost, "/api/v1/auth/atur-password", `{"token":"`+token+`","password":"password-baru-9"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("token sekali pakai: %d", rec.Code)
	}
	if rec := do(srv, nil, http.MethodPost, "/api/v1/auth/login", `{"email":"owner@kedaikopisenja.id","password":"password-baru-9"}`); rec.Code != http.StatusOK {
		t.Fatalf("login password baru: %d", rec.Code)
	}
}
