package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// Label laporan mengikuti nama akun CoA usaha (Piutang Usaha, Utang Usaha, Beban Sewa, …).
func TestLaporanPakaiNamaAkunCoA(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	do(srv, owner, http.MethodPut, "/api/v1/business", dagangCV)

	var nama map[string]string
	_ = json.NewDecoder(do(srv, owner, http.MethodGet, "/api/v1/reports/nama-akun", "").Body).Decode(&nama)
	for k, mau := range map[string]string{"piutang": "Piutang Usaha", "utang_usaha": "Utang Usaha", "kas": "Kas & Bank",
		"pinjaman": "Utang Bank", "omzet": "Penjualan", "hpp": "Harga Pokok Penjualan", "beban:Sewa Tempat": "Beban Sewa",
		"beban:Gaji & Upah": "Beban Gaji & Upah", "penyusutan": "Beban Penyusutan"} {
		if nama[k] != mau {
			t.Errorf("%s = %q, mau %q", k, nama[k], mau)
		}
	}
	rec := do(srv, owner, http.MethodGet, "/api/v1/reports/pdf?period=2026-08&bagian=semua", "")
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, "Piutang Usaha") || strings.Contains(body, "Piutang belum tertagih") {
		t.Errorf("PDF belum memakai nama akun CoA (%d)", rec.Code)
	}
	var b struct {
		Bagian []struct {
			Baris []struct{ Kunci, Label string } `json:"baris"`
		} `json:"bagian"`
	}
	_ = json.NewDecoder(do(srv, owner, http.MethodGet, "/api/v1/reports/banding?periode=2026-07,2026-08", "").Body).Decode(&b)
	ada := false
	for _, bg := range b.Bagian {
		for _, r := range bg.Baris {
			if r.Kunci == "utang_usaha" {
				ada = r.Label == "Utang Usaha"
			}
		}
	}
	if !ada {
		t.Error("perbandingan belum memakai nama akun CoA untuk utang usaha")
	}
}
