package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStateSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DATA_DIR", dir)
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")

	if rec := do(srv, owner, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-08-20","jenis":"Penjualan","nominal":123456,"keterangan":"Tes persist","lunas":true}`); rec.Code != http.StatusCreated {
		t.Fatalf("create tx: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, owner, http.MethodPost, "/api/v1/users", `{"name":"Tono","email":"tono@x.id","role":"kasir","active":true,"password":"rahasia123"}`); rec.Code != http.StatusCreated {
		t.Fatalf("create user: %d %s", rec.Code, rec.Body)
	}
	if info, err := os.Stat(filepath.Join(dir, "state.json")); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("state.json must exist with 0600: %v %v", err, info)
	}

	srv2 := NewServer() // "deploy ulang"
	rec := do(srv2, owner, http.MethodGet, "/api/v1/transactions", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("old session must stay valid after restart, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Tes persist") {
		t.Fatal("transaction lost after restart")
	}
	login(t, srv2, "owner@kedaikopisenja.id")
	if rec := do(srv2, nil, http.MethodPost, "/api/v1/auth/login", `{"email":"tono@x.id","password":"rahasia123"}`); rec.Code != http.StatusOK {
		t.Fatalf("new user must log in after restart, got %d", rec.Code)
	}
}

func TestNoDataDirStaysInMemory(t *testing.T) {
	t.Setenv("DATA_DIR", "")
	srv := NewServer()
	if rec := do(srv, login(t, srv, "owner@kedaikopisenja.id"), http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-08-20","jenis":"Penjualan","nominal":1,"lunas":true}`); rec.Code != http.StatusCreated {
		t.Fatalf("create without DATA_DIR: %d", rec.Code)
	}
}
