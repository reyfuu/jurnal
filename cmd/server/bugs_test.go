package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestBugReportPerTenantAndPersisted(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	kasir := login(t, srv, "kasir@kedaikopisenja.id")
	if rec := do(srv, kasir, http.MethodPost, "/api/v1/bugs", `{"judul":" ","deskripsi":"x"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("judul kosong: want 400, got %d", rec.Code)
	}
	if rec := do(srv, kasir, http.MethodPost, "/api/v1/bugs", `{"judul":"Tombol simpan mati","deskripsi":"Tidak terjadi apa-apa","halaman":"transaksi"}`); rec.Code != http.StatusCreated {
		t.Fatalf("lapor bug: %d %s", rec.Code, rec.Body)
	}
	qa := register(t, srv, registerBody)
	if body := do(srv, qa, http.MethodGet, "/api/v1/bugs", "").Body.String(); strings.Contains(body, "Tombol simpan") {
		t.Fatal("laporan bug usaha lain bocor")
	}

	var list []bugReport
	srv2 := NewServer()
	_ = json.NewDecoder(do(srv2, login(t, srv2, "owner@kedaikopisenja.id"), http.MethodGet, "/api/v1/bugs", "").Body).Decode(&list)
	if len(list) != 1 || list[0].UserEmail != "kasir@kedaikopisenja.id" || list[0].Halaman != "transaksi" {
		t.Fatalf("laporan harus bertahan setelah restart: %+v", list)
	}
}

func TestCekAwalEndpoint(t *testing.T) {
	srv := NewServer()
	qa := register(t, srv, registerBody)
	rec := do(srv, qa, http.MethodPost, "/api/v1/business/cek-awal", `{"jenis_usaha":"perdagangan","bentuk_usaha":"op","ptkp":"TK/0","omzet_tahunan":6000000000,"laba_sebelum_pajak":60000000}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"di_bawah"`) || !strings.Contains(rec.Body.String(), "Wajib menjadi PKP") {
		t.Fatalf("cek awal: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, nil, http.MethodPost, "/api/v1/business/cek-awal", `{}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa login: want 401, got %d", rec.Code)
	}
}
