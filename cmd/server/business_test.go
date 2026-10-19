package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"pajak/internal/tax"
)

const profileBody = `{"nama_usaha":"Kedai Kopi Senja","jenis_usaha":"manufaktur","bentuk_usaha":"op","ptkp":"K/1","pkp":false}`

func TestBusinessProfileFlow(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")

	var got struct {
		Profile *struct {
			JenisUsaha string `json:"jenis_usaha"`
		} `json:"profile"`
		Options map[string]any `json:"options"`
	}
	rec := do(srv, owner, http.MethodGet, "/api/v1/business", "")
	_ = json.NewDecoder(rec.Body).Decode(&got)
	if rec.Code != http.StatusOK || got.Profile != nil || got.Options["jenis_usaha"] == nil {
		t.Fatalf("new server must have empty profile + options, got %d %+v", rec.Code, got)
	}
	if rec := do(srv, login(t, srv, "admin@kedaikopisenja.id"), http.MethodPut, "/api/v1/business", profileBody); rec.Code != http.StatusForbidden {
		t.Fatalf("admin must not edit profile, got %d", rec.Code)
	}
	if rec := do(srv, owner, http.MethodPut, "/api/v1/business", `{"nama_usaha":"X","jenis_usaha":"tambang","bentuk_usaha":"op"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid profile must be 400, got %d", rec.Code)
	}
	if rec := do(srv, owner, http.MethodPut, "/api/v1/business", profileBody); rec.Code != http.StatusOK {
		t.Fatalf("owner save: %d %s", rec.Code, rec.Body)
	}

	// server baru dengan DATA_DIR sama memuat profil yang tersimpan
	srv2 := NewServer()
	rec = do(srv2, login(t, srv2, "kasir@kedaikopisenja.id"), http.MethodGet, "/api/v1/business", "")
	_ = json.NewDecoder(rec.Body).Decode(&got)
	if got.Profile == nil || got.Profile.JenisUsaha != "manufaktur" {
		t.Fatalf("profile not persisted: %+v", got)
	}
}

func TestTaxPlanningEndpoint(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	do(srv, owner, http.MethodPut, "/api/v1/business", profileBody)

	var r struct {
		BentukUsaha tax.BentukUsahaResult `json:"bentuk_usaha"`
		Entitas     []tax.Entitas         `json:"entitas"`
		Input       struct {
			Omzet        int64  `json:"omzet"`
			NetMarginBps int64  `json:"net_margin_bps"`
			PTKP         string `json:"ptkp"`
		} `json:"input"`
	}
	rec := do(srv, owner, http.MethodPost, "/api/v1/tax/planning", `{}`)
	_ = json.NewDecoder(rec.Body).Decode(&r)
	if rec.Code != http.StatusOK || r.Input.Omzet <= 0 || r.Input.NetMarginBps <= 0 || r.Input.PTKP != "K/1" || len(r.BentukUsaha.Skema) != 6 || len(r.Entitas) != 5 {
		t.Fatalf("defaults from data/profile: %d %+v", rec.Code, r.Input)
	}
	rec = do(srv, owner, http.MethodPost, "/api/v1/tax/planning", `{"omzet":2400000000,"net_margin_bps":700,"ptkp":"K/3"}`)
	_ = json.NewDecoder(rec.Body).Decode(&r)
	if r.BentukUsaha.Termurah != tax.SkemaPasal17OP {
		t.Fatalf("template sample should recommend Pasal 17 OP, got %s", r.BentukUsaha.Termurah)
	}
	if rec := do(srv, login(t, srv, "kasir@kedaikopisenja.id"), http.MethodPost, "/api/v1/tax/planning", `{}`); rec.Code != http.StatusForbidden {
		t.Fatalf("kasir must get 403, got %d", rec.Code)
	}
}
