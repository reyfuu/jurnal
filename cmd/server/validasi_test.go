package main

import (
	"net/http"
	"strings"
	"testing"
)

// Margin laba di antara -100% dan 100% (laba vs omzet) dan PKP ≤ peredaran bruto ditolak server
// juga, bukan hanya di formulir.
func TestValidasiMarginLabaDanPKP(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	profil := func(omzet, laba string) string {
		return strings.Replace(profileBody, `}`, `,"omzet_tahunan":`+omzet+`,"laba_sebelum_pajak":`+laba+`}`, 1)
	}
	cases := []struct {
		method, path, body string
		want               int
	}{
		{http.MethodPut, "/api/v1/business", profil("1000000000", "1500000000"), http.StatusBadRequest},  // laba > omzet
		{http.MethodPut, "/api/v1/business", profil("1000000000", "-1000000001"), http.StatusBadRequest}, // rugi > omzet
		{http.MethodPut, "/api/v1/business", profil("1000000000", "1000000000"), http.StatusOK},          // margin 100%
		{http.MethodPost, "/api/v1/tax/planning", `{"omzet":1000000000,"net_margin_bps":15000}`, http.StatusBadRequest},
		{http.MethodPost, "/api/v1/tax/planning", `{"omzet":1000000000,"net_margin_bps":-10001}`, http.StatusBadRequest},
		{http.MethodPost, "/api/v1/tax/planning", `{}`, http.StatusOK}, // margin turunan tidak divalidasi
		{http.MethodPost, "/api/v1/kalkulator/pph-badan", `{"peredaran_bruto":1000000000,"penghasilan_kena_pajak":2000000000}`, http.StatusBadRequest},
		{http.MethodPost, "/api/v1/kalkulator/pph-badan", `{"peredaran_bruto":1000000000,"penghasilan_kena_pajak":1000000000}`, http.StatusOK},
	}
	for _, c := range cases {
		rec := do(srv, owner, c.method, c.path, c.body)
		if rec.Code != c.want {
			t.Errorf("%s %s %s: want %d, got %d %s", c.method, c.path, c.body, c.want, rec.Code, rec.Body)
		}
		if c.want == http.StatusBadRequest && !strings.Contains(rec.Body.String(), "lebih besar") && !strings.Contains(rec.Body.String(), "-100% dan 100%") {
			t.Errorf("%s: pesan harus meminta revisi input, dapat %q", c.path, rec.Body)
		}
	}
}
