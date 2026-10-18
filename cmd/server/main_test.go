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

	asOwner(t, srv).ServeHTTP(rec, req)

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

	asOwner(t, srv).ServeHTTP(rec, req)

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
	asOwner(t, srv).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/reports/financial?period=2026-08", nil))
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
	asOwner(t, srv).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/reports/cashflow?period=2026-08", nil))
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

func TestBalanceSheetEndpoint(t *testing.T) {
	srv := NewServer()
	rec := httptest.NewRecorder()
	asOwner(t, srv).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/reports/balance-sheet?period=2026-08", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("balance-sheet: status %d", rec.Code)
	}
	var bs finance.BalanceSheet
	if err := json.NewDecoder(rec.Body).Decode(&bs); err != nil {
		t.Fatal(err)
	}
	if !bs.IsBalanced {
		t.Fatalf("expected balance sheet to balance: %+v", bs)
	}
}

func TestAddTransactionSyncsLedgerAndReports(t *testing.T) {
	srv := NewServer()

	// Tambah transaksi penjualan cash Rp5.000.000
	body := `{"title":"Penjualan Paket Event","type":"INCOME","channel":"Dine-in","gross":5000000,"fee_bps":0}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	asOwner(t, srv).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("add tx failed: status %d", rec.Code)
	}

	// Cek Laporan Keuangan apakah omzet naik jadi 97.000.000 (92jt + 5jt)
	recFin := httptest.NewRecorder()
	asOwner(t, srv).ServeHTTP(recFin, httptest.NewRequest(http.MethodGet, "/api/v1/reports/financial?period=2026-08", nil))
	var is finance.IncomeStatement
	if err := json.NewDecoder(recFin.Body).Decode(&is); err != nil {
		t.Fatal(err)
	}
	if is.Omzet != 97_000_000 {
		t.Fatalf("expected updated omzet 97_000_000, got %d", is.Omzet)
	}
}
