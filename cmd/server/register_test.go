package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

const registerBody = `{"name":"QA Satu","email":"qa1@x.id","password":"rahasia123"}`

// register mendaftarkan akun baru dan mengembalikan cookie sesinya.
func register(t *testing.T, srv *Server, body string) *http.Cookie {
	t.Helper()
	rec := do(srv, nil, http.MethodPost, "/api/v1/auth/register", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", rec.Code, rec.Body)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookie {
			return c
		}
	}
	t.Fatal("no session cookie")
	return nil
}

func TestRegisterCreatesEmptyIsolatedBusiness(t *testing.T) {
	srv := NewServer()
	qa := register(t, srv, registerBody)
	owner := login(t, srv, "owner@kedaikopisenja.id")

	var me meResp
	_ = json.NewDecoder(do(srv, qa, http.MethodGet, "/api/v1/auth/me", "").Body).Decode(&me)
	if me.User.Role != "owner" || me.User.TenantID < 2 {
		t.Fatalf("pendaftar harus Owner usaha baru: %+v", me.User)
	}
	var txs txListResp
	_ = json.NewDecoder(do(srv, qa, http.MethodGet, "/api/v1/transactions", "").Body).Decode(&txs)
	if len(txs.Transactions) != 0 {
		t.Fatalf("usaha baru harus kosong, ada %d transaksi", len(txs.Transactions))
	}
	if body := do(srv, qa, http.MethodGet, "/api/v1/business", "").Body.String(); !strings.Contains(body, `"profile":null`) {
		t.Fatalf("usaha baru belum punya profil (masuk onboarding): %s", body)
	}
	if rec := do(srv, qa, http.MethodGet, "/api/v1/dashboard?period=2026-08", ""); rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "Kedai Kopi Senja") {
		t.Fatalf("dashboard usaha baru tidak boleh berisi data contoh: %d", rec.Code)
	}

	aktifkan(t, srv, qa)
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-08-20","jenis":"Penjualan","nominal":5000,"keterangan":"Punya QA","lunas":true}`); rec.Code != http.StatusCreated {
		t.Fatalf("create tx: %d %s", rec.Code, rec.Body)
	}
	if strings.Contains(do(srv, owner, http.MethodGet, "/api/v1/transactions", "").Body.String(), "Punya QA") {
		t.Fatal("transaksi usaha baru bocor ke usaha 1")
	}

	var users usersResp
	_ = json.NewDecoder(do(srv, qa, http.MethodGet, "/api/v1/users", "").Body).Decode(&users)
	if len(users.Users) != 1 {
		t.Fatalf("pendaftar hanya boleh melihat user usahanya, dapat %d", len(users.Users))
	}
	if rec := do(srv, qa, http.MethodPut, "/api/v1/users/1", `{"name":"x","email":"owner@kedaikopisenja.id","role":"kasir","active":true}`); rec.Code != http.StatusNotFound {
		t.Fatalf("ubah user usaha lain: want 404, got %d", rec.Code)
	}
	if rec := do(srv, nil, http.MethodPost, "/api/v1/auth/register", registerBody); rec.Code != http.StatusConflict {
		t.Fatalf("email sama: want 409, got %d", rec.Code)
	}
}

func TestRegisteredBusinessSurvivesRestart(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	qa := register(t, srv, registerBody)
	aktifkan(t, srv, qa)
	do(srv, qa, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-08-20","jenis":"Penjualan","nominal":5000,"keterangan":"Punya QA","lunas":true}`)
	nDemo := len(demoTenant().txs)

	srv2 := NewServer()
	if !strings.Contains(do(srv2, qa, http.MethodGet, "/api/v1/transactions", "").Body.String(), "Punya QA") {
		t.Fatal("transaksi usaha baru hilang setelah restart")
	}
	var txs txListResp
	_ = json.NewDecoder(do(srv2, login(t, srv2, "owner@kedaikopisenja.id"), http.MethodGet, "/api/v1/transactions", "").Body).Decode(&txs)
	if len(txs.Transactions) != nDemo {
		t.Fatalf("usaha 1 berubah: want %d transaksi, got %d", nDemo, len(txs.Transactions))
	}
	// usaha berikutnya tidak boleh memakai ulang id usaha yang sudah ada
	c := register(t, srv2, strings.Replace(registerBody, "qa1@", "qa2@", 1))
	var me meResp
	_ = json.NewDecoder(do(srv2, c, http.MethodGet, "/api/v1/auth/me", "").Body).Decode(&me)
	if me.User.TenantID != 3 {
		t.Fatalf("usaha kedua harus id 3, got %d", me.User.TenantID)
	}
}
