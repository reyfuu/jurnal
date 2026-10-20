package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"pajak/internal/finance"
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
	if rec.Code != http.StatusOK || r.Input.Omzet <= 0 || r.Input.NetMarginBps <= 0 || r.Input.PTKP != "K/1" || len(r.BentukUsaha.Skema) != 6 || len(r.Entitas) != 6 {
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

func TestWilayahEndpoint(t *testing.T) {
	srv := NewServer()
	if c := do(srv, nil, http.MethodGet, "/api/v1/wilayah?provinsi=Bali", "").Code; c != http.StatusUnauthorized {
		t.Fatalf("tanpa login: want 401, got %d", c)
	}
	owner := login(t, srv, "owner@kedaikopisenja.id")
	var kota []finance.Kota
	rec := do(srv, owner, http.MethodGet, "/api/v1/wilayah?provinsi=Jawa+Barat", "")
	if err := json.NewDecoder(rec.Body).Decode(&kota); err != nil || rec.Code != http.StatusOK || len(kota) != 27 {
		t.Fatalf("Jawa Barat: %d %d kab/kota %v", rec.Code, len(kota), err)
	}
	if c := do(srv, owner, http.MethodGet, "/api/v1/wilayah?provinsi=Atlantis", "").Code; c != http.StatusBadRequest {
		t.Fatalf("provinsi asing: want 400, got %d", c)
	}
	bad := `{"nama_usaha":"X","jenis_usaha":"jasa","bentuk_usaha":"op","ptkp":"K/1","provinsi":"Jawa Barat","kota":"Kota Bandung","kecamatan":"Kebayoran Baru"}`
	if c := do(srv, owner, http.MethodPut, "/api/v1/business", bad).Code; c != http.StatusBadRequest {
		t.Fatalf("kecamatan bukan di kota itu: want 400, got %d", c)
	}
	good := strings.Replace(bad, "Kebayoran Baru", "Sukasari", 1)
	if rec := do(srv, owner, http.MethodPut, "/api/v1/business", good); rec.Code != http.StatusOK {
		t.Fatalf("wilayah valid: %d %s", rec.Code, rec.Body)
	}
}

func TestSaranKLU_TanpaAIPakaiPencarian(t *testing.T) {
	srv := NewServer()
	qa := register(t, srv, registerBody) // belum berlangganan: tetap boleh (onboarding)
	rec := do(srv, qa, http.MethodPost, "/api/v1/business/klu-saran", `{"deskripsi":"laundry kiloan dekat kampus","jenis":"jasa"}`)
	var body struct {
		AI    bool `json:"ai"`
		Saran []struct {
			Kode   string   `json:"kode"`
			Contoh []string `json:"contoh"`
		} `json:"saran"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if rec.Code != http.StatusOK || body.AI || len(body.Saran) == 0 || body.Saran[0].Kode != "962" || len(body.Saran[0].Contoh) == 0 {
		t.Fatalf("saran KLU: %d %+v", rec.Code, body)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/business/klu-saran", `{"deskripsi":"ab"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("deskripsi terlalu pendek: %d", rec.Code)
	}
}

func TestJatahSaranKLU(t *testing.T) {
	srv := NewServer()
	for i := 0; i < maxSaranKLUAI; i++ {
		if !srv.jatahSaranKLU(7) {
			t.Fatalf("jatah ke-%d ditolak", i+1)
		}
	}
	if srv.jatahSaranKLU(7) || !srv.jatahSaranKLU(8) {
		t.Fatal("kuota per user tidak berlaku")
	}
}
