package finance_test

import (
	"testing"

	"pajak/internal/finance"
)

func TestIncomeStatement_Agustus2026(t *testing.T) {
	is := finance.BuildIncomeStatement(finance.SeedLedger("2026-08"))

	checks := []struct {
		name      string
		got, want int64
	}{
		{"Omzet", is.Omzet, 92_000_000},
		{"ChannelFees", is.ChannelFees, 4_600_000},
		{"NetSales", is.NetSales, 87_400_000},
		{"COGS", is.COGS, 26_000_000},
		{"GrossProfit", is.GrossProfit, 61_400_000},
		{"Opex", is.Opex, 36_400_000},
		{"NetProfit", is.NetProfit, 25_000_000},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: want %d, got %d", c.name, c.want, c.got)
		}
	}
}

func TestCashFlow_ReconcilesProfitToCash(t *testing.T) {
	entries := finance.SeedLedger("2026-08")
	is := finance.BuildIncomeStatement(entries)
	cf := finance.BuildCashFlow(entries, 80_000_000)

	if cf.NetCash != 17_000_000 {
		t.Fatalf("NetCash: want 17_000_000, got %d", cf.NetCash)
	}
	if cf.ClosingBalance != 97_000_000 {
		t.Fatalf("ClosingBalance: want 97_000_000, got %d", cf.ClosingBalance)
	}
	if cf.NewReceivables != 11_000_000 || cf.NewPayables != 3_000_000 {
		t.Fatalf("unexpected receivables/payables: %+v", cf)
	}
	// Invariant: NetProfit - piutang baru + utang baru == kenaikan kas.
	if got := is.NetProfit - cf.NewReceivables + cf.NewPayables; got != cf.NetCash {
		t.Fatalf("reconciliation broken: profit-bridge=%d, netcash=%d", got, cf.NetCash)
	}
}

func TestCashFlow_EmptyLedger(t *testing.T) {
	cf := finance.BuildCashFlow(nil, 5_000_000)
	if cf.NetCash != 0 || cf.ClosingBalance != 5_000_000 {
		t.Fatalf("empty ledger must not change cash: %+v", cf)
	}
}

func TestBalanceSheet_Agustus2026(t *testing.T) {
	const inventoryValue int64 = 8_250_000
	const openingEquity int64 = 88_250_000 // Modal Awal = Aset Awal (80jt kas + 8.25jt stok) - Utang Awal (0)

	bs := finance.BuildBalanceSheet(finance.SeedLedger("2026-08"), 80_000_000, inventoryValue, openingEquity)

	if bs.Assets.Cash != 97_000_000 {
		t.Errorf("Assets.Cash: want 97_000_000, got %d", bs.Assets.Cash)
	}
	if bs.Assets.Receivables != 11_000_000 {
		t.Errorf("Assets.Receivables: want 11_000_000, got %d", bs.Assets.Receivables)
	}
	if bs.Assets.Total != 97_000_000+11_000_000+8_250_000 {
		t.Errorf("Assets.Total: want %d, got %d", 97_000_000+11_000_000+8_250_000, bs.Assets.Total)
	}
	if bs.Liabilities.Payables != 3_000_000 {
		t.Errorf("Liabilities.Payables: want 3_000_000, got %d", bs.Liabilities.Payables)
	}
	if !bs.IsBalanced {
		t.Errorf("Balance sheet not balanced: Total Assets %d != Liabilities (%d) + Equity (%d) [Sum=%d]",
			bs.Assets.Total, bs.Liabilities.Total, bs.Equity.Total, bs.Liabilities.Total+bs.Equity.Total)
	}
}
