package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	if me.User.Role != auth.Kasir || len(me.Views) != 2 {
		t.Fatalf("me: %+v", me)
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
