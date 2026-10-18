package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pajak/internal/finance"
)

func TestDashboardEndpoint(t *testing.T) {
	srv := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard?period=2026-08", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var snap finance.DashboardSnapshot
	if err := json.NewDecoder(rec.Body).Decode(&snap); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if snap.Tenant.Name != "Kedai Kopi Senja" {
		t.Fatalf("expected tenant 'Kedai Kopi Senja', got %q", snap.Tenant.Name)
	}
	if snap.KPIs.Omzet.Display != "Rp92 jt" {
		t.Fatalf("expected Omzet 'Rp92 jt', got %q", snap.KPIs.Omzet.Display)
	}
}

func TestPriceSimulationEndpoint(t *testing.T) {
	srv := NewServer()
	body := `{"cogs":10000,"target_profit":8000,"fee_bps":2000,"fixed_fee":0}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pricing/simulate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var res finance.PriceSimulationResult
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode err: %v", err)
	}
	if res.RecommendedPrice != 22500 {
		t.Fatalf("expected recommended price 22500, got %d", res.RecommendedPrice)
	}
}

func TestReportsEndpoints(t *testing.T) {
	srv := NewServer()

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/reports/financial?period=2026-08", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("financial: status %d", rec.Code)
	}
	var is finance.IncomeStatement
	if err := json.NewDecoder(rec.Body).Decode(&is); err != nil {
		t.Fatal(err)
	}
	if is.NetProfit != 25_000_000 {
		t.Fatalf("NetProfit: want 25_000_000, got %d", is.NetProfit)
	}

	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/reports/cashflow?period=2026-08", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("cashflow: status %d", rec.Code)
	}
	var cf finance.CashFlow
	if err := json.NewDecoder(rec.Body).Decode(&cf); err != nil {
		t.Fatal(err)
	}
	if cf.ClosingBalance != 97_000_000 {
		t.Fatalf("ClosingBalance: want 97_000_000, got %d", cf.ClosingBalance)
	}
}
