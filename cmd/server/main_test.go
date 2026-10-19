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
	// Laba bersih sudah dikurangi penyusutan & amortisasi register aset seed (Rp6,55 jt).
	if is.NetProfit != 18_450_000 || is.Depreciation != 6_175_000 || is.Amortization != 375_000 {
		t.Fatalf("laba rugi: net=%d penyusutan=%d amortisasi=%d", is.NetProfit, is.Depreciation, is.Amortization)
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
	if cf.OperasiLangsung.Net != cf.OperasiTidakLangsung.Net {
		t.Fatalf("kas operasi dua metode beda: langsung %d, tidak langsung %d",
			cf.OperasiLangsung.Net, cf.OperasiTidakLangsung.Net)
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
	body := `{"tanggal":"2026-08-20","jenis":"Penjualan","nominal":5000000,"keterangan":"Penjualan Paket Event","channel":"Dine-in","lunas":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	asOwner(t, srv).ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
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

func TestAIChatEndpoint(t *testing.T) {
	srv := NewServer()

	body := `{"query":"Ringkas laporan bulan ini","period":"2026-08"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	asOwner(t, srv).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode err: %v", err)
	}
	ans, _ := res["answer"].(string)
	if !strings.Contains(ans, "Ringkasan Laporan Bisnis") {
		t.Fatalf("expected answer to contain summary, got: %s", ans)
	}
}

func TestPriceSimulationBreakEven(t *testing.T) {
	srv := asOwner(t, NewServer())
	post := func(body string) map[string]any {
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/pricing/simulate?period=2026-08", strings.NewReader(body)))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d %s", rec.Code, rec.Body)
		}
		var m map[string]any
		_ = json.NewDecoder(rec.Body).Decode(&m)
		return m
	}
	m := post(`{"cogs":6800,"target_profit":7600,"fee_bps":2000,"fixed_cost":38000000,"monthly_volume":1500,"target_net_profit":15200000}`)
	be, ok := m["break_even"].(map[string]any)
	if !ok || be["bep_units"].(float64) != 5000 || m["recommended_price"].(float64) != 18000 {
		t.Fatalf("unexpected response %v", m)
	}
	// tanpa fixed_cost: pakai total beban operasional periode berjalan
	m = post(`{"cogs":6800,"target_profit":7600,"fee_bps":2000}`)
	if be := m["break_even"].(map[string]any); be["fixed_cost"].(float64) <= 0 {
		t.Fatalf("default fixed cost should come from period opex, got %v", be)
	}
}

func TestSalesReportEndpoint(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	rec := do(srv, owner, http.MethodGet, "/api/v1/reports/sales?period=2026-08&granularity=week", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body)
	}
	var r finance.SalesReportResult
	_ = json.NewDecoder(rec.Body).Decode(&r)
	if r.Summary.Omzet <= 0 || len(r.Trend) == 0 || len(r.TopItems) == 0 || r.Granularity != "week" {
		t.Fatalf("unexpected report %+v", r.Summary)
	}
	// month = tren seluruh periode
	rec = do(srv, owner, http.MethodGet, "/api/v1/reports/sales?granularity=month", "")
	_ = json.NewDecoder(rec.Body).Decode(&r)
	if len(r.Trend) != len(finance.AvailablePeriods()) {
		t.Fatalf("month trend should cover all periods, got %d", len(r.Trend))
	}
	if rec := do(srv, login(t, srv, "kasir@kedaikopisenja.id"), http.MethodGet, "/api/v1/reports/sales", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("kasir must not see sales report, got %d", rec.Code)
	}
}
