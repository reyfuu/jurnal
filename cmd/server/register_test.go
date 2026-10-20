package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

const registerBody = `{"name":"QA Satu","email":"qa1@x.id","password":"rahasia123","wa":"0812-3456-7890","setuju_pdp":true}`

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

// UU PDP: pendaftaran wajib menyetujui kebijakan privasi dan mencantumkan nomor WA yang sah.
func TestRegisterWajibWAdanPersetujuanPDP(t *testing.T) {
	srv := NewServer()
	for name, body := range map[string]string{
		"tanpa persetujuan": `{"name":"QA","email":"qa@x.id","password":"rahasia123","wa":"081234567890"}`,
		"tolak persetujuan": `{"name":"QA","email":"qa@x.id","password":"rahasia123","wa":"081234567890","setuju_pdp":false}`,
		"tanpa WA":          `{"name":"QA","email":"qa@x.id","password":"rahasia123","setuju_pdp":true}`,
		"WA tidak sah":      `{"name":"QA","email":"qa@x.id","password":"rahasia123","wa":"12345","setuju_pdp":true}`,
	} {
		if rec := do(srv, nil, http.MethodPost, "/api/v1/auth/register", body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: want 400, got %d %s", name, rec.Code, rec.Body)
		}
	}
	if len(srv.tenants) != 1 {
		t.Fatalf("pendaftaran gagal tidak boleh membuat usaha, ada %d", len(srv.tenants))
	}

	var me meResp
	_ = json.NewDecoder(do(srv, register(t, srv, registerBody), http.MethodGet, "/api/v1/auth/me", "").Body).Decode(&me)
	if me.User.WA != "6281234567890" {
		t.Fatalf("WA harus dinormalkan ke 62…, got %q", me.User.WA)
	}
	if time.Since(me.User.PersetujuanPDP) > time.Minute || me.User.VersiPDP != versiPDP {
		t.Fatalf("persetujuan PDP harus tercatat (waktu & versi): %q %q", me.User.PersetujuanPDP, me.User.VersiPDP)
	}
}
