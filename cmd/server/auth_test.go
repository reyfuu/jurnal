package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"

	"pajak/internal/auth"
)

// login mengembalikan cookie sesi untuk email demo.
func login(t *testing.T, srv *Server, email string) *http.Cookie {
	t.Helper()
	body := `{"email":"` + email + `","password":"` + auth.DemoPassword + `"}`
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("login %s: status %d %s", email, rec.Code, rec.Body)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookie {
			if !c.HttpOnly {
				t.Fatal("session cookie must be HttpOnly")
			}
			return c
		}
	}
	t.Fatal("no session cookie")
	return nil
}

// asOwner membungkus server agar setiap request membawa sesi Owner.
func asOwner(t *testing.T, srv *Server) http.Handler {
	c := login(t, srv, "owner@kedaikopisenja.id")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.AddCookie(c)
		srv.ServeHTTP(w, r)
	})
}

func do(srv *Server, c *http.Cookie, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if c != nil {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func TestAPIRequiresLogin(t *testing.T) {
	srv := NewServer()
	if rec := do(srv, nil, http.MethodGet, "/api/v1/dashboard", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("dashboard without session: want 401, got %d", rec.Code)
	}
	if rec := do(srv, nil, http.MethodPost, "/api/v1/auth/login", `{"email":"owner@kedaikopisenja.id","password":"salah"}`); rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad password: want 401, got %d", rec.Code)
	}
}

func TestMeAndLogout(t *testing.T) {
	srv := NewServer()
	c := login(t, srv, "kasir@kedaikopisenja.id")
	rec := do(srv, c, http.MethodGet, "/api/v1/auth/me", "")
	var me meResp
	if err := json.NewDecoder(rec.Body).Decode(&me); err != nil {
		t.Fatal(err)
	}
	// Menu kasir disebut satu per satu, bukan dihitung: jumlah menu berubah tiap kali ada
	// fitur baru, sementara yang penting di sini adalah batas aksesnya.
	if me.User.Role != auth.Kasir {
		t.Fatalf("me: %+v", me)
	}
	for _, v := range []string{"dashboard", "transaksi", "lapor-bug"} {
		if !slices.Contains(me.Views, v) {
			t.Errorf("kasir harus punya menu %q, got %v", v, me.Views)
		}
	}
	for _, v := range []string{"transaksi-kelola", "pengguna", "laporan-keuangan", "profil-usaha"} {
		if slices.Contains(me.Views, v) {
			t.Errorf("kasir tidak boleh punya menu %q", v)
		}
	}
	do(srv, c, http.MethodPost, "/api/v1/auth/logout", "")
	if rec := do(srv, c, http.MethodGet, "/api/v1/auth/me", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("after logout: want 401, got %d", rec.Code)
	}
}

func TestRoleAccess(t *testing.T) {
	srv := NewServer()
	kasir := login(t, srv, "kasir@kedaikopisenja.id")
	admin := login(t, srv, "admin@kedaikopisenja.id")
	cases := []struct {
		c      *http.Cookie
		method string
		path   string
		want   int
	}{
		{kasir, http.MethodGet, "/api/v1/dashboard", http.StatusOK},
		{kasir, http.MethodGet, "/api/v1/reports/financial", http.StatusForbidden},
		{kasir, http.MethodGet, "/api/v1/users", http.StatusForbidden},
		{admin, http.MethodGet, "/api/v1/reports/financial", http.StatusOK},
		{admin, http.MethodGet, "/api/v1/users", http.StatusForbidden},
	}
	for _, tc := range cases {
		if rec := do(srv, tc.c, tc.method, tc.path, ""); rec.Code != tc.want {
			t.Errorf("%s %s: want %d, got %d", tc.method, tc.path, tc.want, rec.Code)
		}
	}
}

func TestUserCRUD(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")

	rec := do(srv, owner, http.MethodPost, "/api/v1/users", `{"name":"Budi","email":"budi@kopi.id","role":"kasir","active":true,"password":"rahasia123"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	var u auth.User
	_ = json.NewDecoder(rec.Body).Decode(&u)

	if rec := do(srv, owner, http.MethodPost, "/api/v1/users", `{"name":"X","email":"budi@kopi.id","role":"kasir","active":true,"password":"rahasia123"}`); rec.Code != http.StatusConflict {
		t.Fatalf("duplicate: want 409, got %d", rec.Code)
	}
	if rec := do(srv, owner, http.MethodPost, "/api/v1/users", `{"name":"X","email":"x@kopi.id","role":"dewa","active":true,"password":"rahasia123"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad role: want 400, got %d", rec.Code)
	}

	path := "/api/v1/users/" + strconv.Itoa(u.ID)
	if rec := do(srv, owner, http.MethodPut, path, `{"name":"Budi","email":"budi@kopi.id","role":"admin","active":true}`); rec.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec.Code, rec.Body)
	}

	var list usersResp
	_ = json.NewDecoder(do(srv, owner, http.MethodGet, "/api/v1/users", "").Body).Decode(&list)
	if len(list.Users) != 4 || list.Users[3].Role != auth.Admin || len(list.Roles) != 3 {
		t.Fatalf("list: %+v", list)
	}

	if rec := do(srv, owner, http.MethodDelete, path, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}
	if rec := do(srv, owner, http.MethodDelete, "/api/v1/users/1", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("delete self: want 400, got %d", rec.Code)
	}
}

func TestLoginCookieSecureBehindProxy(t *testing.T) {
	srv := NewServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"email":"owner@kedaikopisenja.id","password":"`+auth.DemoPassword+`"}`))
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	cs := rec.Result().Cookies()
	if len(cs) == 0 || !cs[0].Secure || cs[0].MaxAge <= 0 {
		t.Fatalf("want Secure cookie with MaxAge behind https proxy, got %+v", cs)
	}
}

func TestLoginRateLimit(t *testing.T) {
	srv := NewServer()
	bad := func(ip string) int {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
			strings.NewReader(`{"email":"owner@kedaikopisenja.id","password":"salah"}`))
		req.Header.Set("X-Real-IP", ip)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		return rec.Code
	}
	for i := range maxLoginFailures {
		if code := bad("203.0.113.9"); code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: want 401, got %d", i+1, code)
		}
	}
	if code := bad("203.0.113.9"); code != http.StatusTooManyRequests {
		t.Fatalf("want 429 after %d failures, got %d", maxLoginFailures, code)
	}
	if code := bad("203.0.113.10"); code != http.StatusUnauthorized {
		t.Fatalf("other IP must not be blocked, got %d", code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	NewServer().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil))
	for k, v := range map[string]string{"X-Content-Type-Options": "nosniff", "X-Frame-Options": "DENY"} {
		if got := rec.Header().Get(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
}
