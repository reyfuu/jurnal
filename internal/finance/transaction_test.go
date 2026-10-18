package finance_test

import (
	"testing"

	"pajak/internal/finance"
)

func tx(tanggal, jenis string, nominal int64, lunas bool) finance.Transaction {
	return finance.Transaction{Tanggal: tanggal, Jenis: jenis, Nominal: nominal, Lunas: lunas}
}

func TestValidate(t *testing.T) {
	ok := finance.Transaction{Tanggal: "2026-08-10", Jenis: "Penjualan", Nominal: 1, Keterangan: "  Rekap kasir  ", Lunas: true}
	if err := finance.Validate(&ok); err != nil {
		t.Fatalf("valid tx rejected: %v", err)
	}
	if ok.Keterangan != "Rekap kasir" {
		t.Fatalf("keterangan not trimmed: %q", ok.Keterangan)
	}

	bad := []finance.Transaction{
		{Tanggal: "10-08-2026", Jenis: "Penjualan", Nominal: 1, Lunas: true},
		{Tanggal: "2026-08-10", Jenis: "Jualan Online", Nominal: 1, Lunas: true},
		{Tanggal: "2026-08-10", Jenis: "Penjualan", Nominal: 0, Lunas: true},
		{Tanggal: "2026-08-10", Jenis: "Penjualan", Nominal: 1, Qty: -1, Lunas: true},
		{Tanggal: "2026-08-10", Jenis: "Penjualan", Nominal: 1, Lunas: false, TanggalLunas: "2026-08-12"},
		{Tanggal: "2026-08-10", Jenis: "Penjualan", Nominal: 1, Lunas: true, TanggalLunas: "2026-08-01"},
	}
	for i, b := range bad {
		if err := finance.Validate(&b); err == nil {
			t.Errorf("case %d: want error, got nil", i)
		}
	}
}

func TestSeedTransactions_PerPeriod(t *testing.T) {
	all := finance.SeedTransactions()
	total := 0
	for _, p := range finance.AvailablePeriods() {
		n := len(finance.ForPeriod(all, p.ID))
		if n == 0 {
			t.Errorf("period %s has no seed transactions", p.ID)
		}
		total += n
	}
	if total != len(all) {
		t.Fatalf("transactions outside seed periods: %d of %d", len(all)-total, len(all))
	}
	seen := map[int]bool{}
	for _, x := range all {
		if err := finance.Validate(&x); err != nil {
			t.Fatalf("seed tx %d invalid: %v", x.ID, err)
		}
		if seen[x.ID] || x.ID == 0 {
			t.Fatalf("seed IDs must be unique and non-zero, got %d", x.ID)
		}
		seen[x.ID] = true
	}
}

func TestIncomeStatement_OpexGroupedByJenis(t *testing.T) {
	is := finance.BuildIncomeStatement(finance.SeedLedger("2026-08"))
	for _, l := range is.OpexLines {
		if l.Label == "Listrik, Air, Internet" {
			if l.Amount != 15_900_000 { // utilitas 12,9 jt + tagihan listrik belum dibayar 3 jt
				t.Fatalf("Listrik line: want 15_900_000, got %d", l.Amount)
			}
			return
		}
	}
	t.Fatalf("no grouped Listrik line in %+v", is.OpexLines)
}

// Semua jenis non-laba (modal, pinjaman, aset, cicilan, prive) dan pendapatan lain,
// lunas maupun belum: neraca tetap seimbang dan rekonsiliasi laba→kas tetap utuh.
func TestNonOperating_KeepsBooksBalanced(t *testing.T) {
	txs := append(finance.ForPeriod(finance.SeedTransactions(), "2026-08"),
		tx("2026-08-02", "Setor Modal", 10_000_000, true),
		tx("2026-08-03", "Terima Pinjaman", 20_000_000, true),
		tx("2026-08-04", "Beli Alat / Aset", 15_000_000, true),
		tx("2026-08-05", "Beli Alat / Aset", 4_000_000, false),
		tx("2026-08-06", "Bayar Cicilan", 2_000_000, true),
		tx("2026-08-07", "Ambil Uang Pribadi", 3_000_000, true),
		tx("2026-08-08", "Pendapatan Lain", 1_000_000, false),
	)
	entries := finance.ToLedger(txs)
	is := finance.BuildIncomeStatement(entries)
	cf := finance.BuildCashFlow(entries, 80_000_000)
	bs := finance.BuildBalanceSheet(entries, 80_000_000, 8_250_000, 88_250_000)

	if is.OtherIncome != 1_000_000 || is.NetProfit != 26_000_000 {
		t.Fatalf("other income must count in profit: other=%d net=%d", is.OtherIncome, is.NetProfit)
	}
	if got := is.NetProfit - cf.NewReceivables + cf.NewPayables + cf.NonOperating; got != cf.NetCash {
		t.Fatalf("reconciliation broken: %d != NetCash %d (%+v)", got, cf.NetCash, cf)
	}
	if !bs.IsBalanced {
		t.Fatalf("balance sheet not balanced: %+v", bs)
	}
	if bs.Assets.FixedAssets != 19_000_000 || bs.Liabilities.Loans != 18_000_000 || bs.Equity.AddedCapital != 7_000_000 {
		t.Fatalf("non-operating lines wrong: %+v", bs)
	}
}

func TestApplyKPIs(t *testing.T) {
	txs := finance.ForPeriod(finance.SeedTransactions(), "2026-08")
	entries := finance.ToLedger(txs)
	snap := finance.SeedKedaiKopiSenja("2026-08")
	want := snap.KPIs

	finance.ApplyKPIs(&snap, finance.BuildIncomeStatement(entries), finance.BuildCashFlow(entries, 80_000_000), txs, len(txs))
	k := snap.KPIs
	if k.Omzet.Display != want.Omzet.Display || k.LabaBersih.Display != want.LabaBersih.Display ||
		k.MarginLaba.Display != want.MarginLaba.Display || k.SaldoKas.Display != want.SaldoKas.Display ||
		k.TargetOmzet.Display != want.TargetOmzet.Display || k.TargetOmzet.SurplusDisplay != want.TargetOmzet.SurplusDisplay {
		t.Fatalf("seed KPIs must be unchanged by recompute:\nwant %+v\ngot  %+v", want, k)
	}
	if len(snap.RecentTransactions) != 5 || snap.RecentTransactions[0].Date < snap.RecentTransactions[4].Date {
		t.Fatalf("recent transactions: %+v", snap.RecentTransactions)
	}

	// Omzet turun di bawah target → "Kurang".
	low := finance.ToLedger([]finance.Transaction{tx("2026-08-01", "Penjualan", 50_000_000, true)})
	finance.ApplyKPIs(&snap, finance.BuildIncomeStatement(low), finance.BuildCashFlow(low, 80_000_000), nil, 1)
	if snap.KPIs.Omzet.Display != "Rp50 jt" || snap.KPIs.TargetOmzet.SurplusDisplay[:6] != "Kurang" {
		t.Fatalf("low omzet KPIs: %+v", snap.KPIs)
	}
}
