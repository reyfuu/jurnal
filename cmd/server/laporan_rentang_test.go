package main

import (
	"net/http"
	"strings"
	"testing"

	"pajak/internal/finance"
)

// Rentang satu bulan penuh = laporan bulanan; dua paruh bulan berjumlah sama dengan sebulan
// (penyusutan ikut di paruh yang memuat akhir bulan, tidak dobel).
func TestLaporanRentang_KonsistenDenganBulanan(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	bulan := getJSON[finance.IncomeStatement](t, srv, owner, "/api/v1/reports/financial?period=2026-08")
	kas := getJSON[finance.CashFlow](t, srv, owner, "/api/v1/reports/cashflow?period=2026-08")
	penuh := getJSON[finance.LaporanRentang](t, srv, owner, "/api/v1/reports/rentang?dari=2026-08-01&sampai=2026-08-31")
	if penuh.Income.NetProfit != bulan.NetProfit || penuh.Income.Omzet != bulan.Omzet || penuh.CashFlow.ClosingBalance != kas.ClosingBalance || penuh.CashFlow.OpeningBalance != kas.OpeningBalance {
		t.Fatalf("rentang sebulan beda dengan bulanan: laba %d/%d, kas akhir %d/%d", penuh.Income.NetProfit, bulan.NetProfit, penuh.CashFlow.ClosingBalance, kas.ClosingBalance)
	}
	a := getJSON[finance.LaporanRentang](t, srv, owner, "/api/v1/reports/rentang?dari=2026-08-01&sampai=2026-08-15")
	b := getJSON[finance.LaporanRentang](t, srv, owner, "/api/v1/reports/rentang?dari=2026-08-16&sampai=2026-08-31")
	if a.Income.Depreciation != 0 || b.Income.Depreciation != bulan.Depreciation {
		t.Fatalf("penyusutan harus di paruh yang memuat akhir bulan: %d / %d (sebulan %d)", a.Income.Depreciation, b.Income.Depreciation, bulan.Depreciation)
	}
	if a.Income.NetProfit+b.Income.NetProfit != bulan.NetProfit || b.CashFlow.OpeningBalance != a.CashFlow.ClosingBalance {
		t.Fatalf("dua paruh: laba %d+%d≠%d atau kas tidak bersambung (%d→%d)", a.Income.NetProfit, b.Income.NetProfit, bulan.NetProfit, a.CashFlow.ClosingBalance, b.CashFlow.OpeningBalance)
	}
	if a.Label != "1 Agustus 2026 – 15 Agustus 2026" {
		t.Fatalf("label %q", a.Label)
	}
	for _, q := range []string{"dari=2026-08-10&sampai=2026-08-01", "dari=2026-08-01", "dari=2020-01-01&sampai=2026-08-01"} {
		if rec := do(srv, owner, http.MethodGet, "/api/v1/reports/rentang?"+q, ""); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d", q, rec.Code)
		}
	}
	rec := do(srv, owner, http.MethodGet, "/api/v1/reports/pdf?period=2026-08&bagian=laba-rugi&dari=2026-08-01&sampai=2026-08-15", "")
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Body.String(), "%PDF") || !strings.Contains(rec.Header().Get("Content-Disposition"), "2026-08-01_2026-08-15") {
		t.Fatalf("pdf rentang: %d %q", rec.Code, rec.Header().Get("Content-Disposition"))
	}
}
