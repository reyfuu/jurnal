package finance_test

import (
	"testing"

	"pajak/internal/finance"
)

const (
	seedInventory   int64 = finance.SeedInventory
	seedOpeningCash int64 = 80_000_000
)

// seedOpening: saldo awal Agustus 2026 dari data demo.
func seedOpening() finance.ReportOpening {
	return finance.ReportOpening{Cash: seedOpeningCash, Inventory: seedInventory}
}

// agustus2026: laporan lengkap Agustus 2026 dari transaksi & register aset seed.
func agustus2026() finance.PeriodReport {
	return finance.BuildPeriodReport(finance.SeedTransactions(), finance.SeedAssets(), "2026-08", seedOpening())
}

func TestIncomeStatement_Agustus2026(t *testing.T) {
	is := agustus2026().Income

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
		{"Depreciation", is.Depreciation, 6_175_000},
		{"Amortization", is.Amortization, 375_000},
		{"TotalOpex", is.TotalOpex, 42_950_000},
		{"NetProfit", is.NetProfit, 18_450_000},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: want %d, got %d", c.name, c.want, c.got)
		}
	}
	if is.MarginBps != 2005 {
		t.Errorf("MarginBps: want 2005 (20,05%%), got %d", is.MarginBps)
	}
}

// Penyusutan menurunkan laba tapi tidak menyentuh kas sama sekali.
func TestIncomeStatement_PenyusutanTidakMengubahKas(t *testing.T) {
	entries := finance.SeedLedger("2026-08")
	tanpa := finance.BuildIncomeStatement(entries, 0, 0)
	dengan := finance.BuildIncomeStatement(entries, 6_175_000, 375_000)

	if tanpa.NetProfit != 25_000_000 {
		t.Fatalf("laba tanpa penyusutan: want 25_000_000, got %d", tanpa.NetProfit)
	}
	if dengan.NetProfit != tanpa.NetProfit-6_550_000 {
		t.Errorf("laba harus turun sebesar beban penyusutan: %d vs %d", dengan.NetProfit, tanpa.NetProfit)
	}
	cfTanpa := finance.BuildCashFlow(entries, tanpa, seedOpeningCash)
	cfDengan := finance.BuildCashFlow(entries, dengan, seedOpeningCash)
	if cfTanpa.ClosingBalance != cfDengan.ClosingBalance {
		t.Errorf("saldo kas akhir tidak boleh berubah: %d vs %d", cfTanpa.ClosingBalance, cfDengan.ClosingBalance)
	}
}

// Metode langsung dan tidak langsung wajib menghasilkan kas operasi yang sama.
func TestCashFlow_DuaMetodeCocok(t *testing.T) {
	rep := agustus2026()
	cf := rep.CashFlow

	if cf.NetCash != 17_000_000 {
		t.Fatalf("NetCash: want 17_000_000, got %d", cf.NetCash)
	}
	if cf.ClosingBalance != 97_000_000 {
		t.Fatalf("ClosingBalance: want 97_000_000, got %d", cf.ClosingBalance)
	}
	if cf.NewReceivables != 11_000_000 || cf.NewPayables != 3_000_000 {
		t.Fatalf("piutang/utang baru: got %d / %d", cf.NewReceivables, cf.NewPayables)
	}
	if cf.OperasiLangsung.Net != cf.OperasiTidakLangsung.Net {
		t.Errorf("kas operasi dua metode beda: langsung %d, tidak langsung %d",
			cf.OperasiLangsung.Net, cf.OperasiTidakLangsung.Net)
	}
	if cf.OperasiLangsung.Net != 17_000_000 {
		t.Errorf("kas operasi: want 17_000_000, got %d", cf.OperasiLangsung.Net)
	}
	// Jembatan metode tidak langsung: laba + beban non-kas − piutang + utang.
	bridge := rep.Income.NetProfit + rep.Income.Depreciation + rep.Income.Amortization - cf.NewReceivables + cf.NewPayables
	if bridge != cf.OperasiTidakLangsung.Net {
		t.Errorf("rekonsiliasi laba ke kas rusak: bridge=%d, operasi=%d", bridge, cf.OperasiTidakLangsung.Net)
	}
	// Operasi + Investasi + Pendanaan harus sama dengan kenaikan kas bersih.
	if got := cf.OperasiLangsung.Net + cf.Investasi.Net + cf.Pendanaan.Net; got != cf.NetCash {
		t.Errorf("jumlah tiga aktivitas: want %d, got %d", cf.NetCash, got)
	}
}

// Metode langsung merinci penerimaan per channel; totalnya harus sama dengan baris induknya.
func TestCashFlow_RincianPerChannel(t *testing.T) {
	cf := agustus2026().CashFlow
	pelanggan := cf.OperasiLangsung.Lines[0]

	if pelanggan.Amount != 81_000_000 {
		t.Errorf("penerimaan dari pelanggan (tanpa catering tempo 11jt): want 81_000_000, got %d", pelanggan.Amount)
	}
	var total int64
	for _, d := range pelanggan.Detail {
		total += d.Amount
	}
	if total != pelanggan.Amount {
		t.Errorf("rincian channel %d tidak sama dengan totalnya %d", total, pelanggan.Amount)
	}
	if len(pelanggan.Detail) == 0 {
		t.Error("penerimaan pelanggan harus dirinci per channel")
	}
}

func TestCashFlow_EmptyLedger(t *testing.T) {
	cf := finance.BuildCashFlow(nil, finance.IncomeStatement{}, 5_000_000)
	if cf.NetCash != 0 || cf.ClosingBalance != 5_000_000 {
		t.Fatalf("buku besar kosong tidak boleh mengubah kas: %+v", cf)
	}
	if cf.OperasiLangsung.Net != 0 || cf.OperasiTidakLangsung.Net != 0 {
		t.Fatalf("kas operasi harus nol: %d / %d", cf.OperasiLangsung.Net, cf.OperasiTidakLangsung.Net)
	}
}

func TestBalanceSheet_Agustus2026(t *testing.T) {
	bs := agustus2026().BalanceSheet

	checks := []struct {
		name      string
		got, want int64
	}{
		{"Assets.Cash", bs.Assets.Cash, 97_000_000},
		{"Assets.Receivables", bs.Assets.Receivables, 11_000_000},
		{"Assets.Inventory", bs.Assets.Inventory, seedInventory},
		{"Assets.FixedAssetsCost", bs.Assets.FixedAssetsCost, 246_000_000},
		{"Assets.AccumDepreciation", bs.Assets.AccumDepreciation, 45_475_000},
		{"Assets.FixedAssets", bs.Assets.FixedAssets, 200_525_000},
		{"Assets.Total", bs.Assets.Total, 316_775_000},
		{"Liabilities.Payables", bs.Liabilities.Payables, 3_000_000},
		{"Equity.OpeningCapital", bs.Equity.OpeningCapital, 295_325_000},
		{"Equity.CurrentProfit", bs.Equity.CurrentProfit, 18_450_000},
		{"Equity.Total", bs.Equity.Total, 313_775_000},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: want %d, got %d", c.name, c.want, c.got)
		}
	}
	if !bs.IsBalanced {
		t.Errorf("neraca tidak seimbang: aktiva %d, pasiva %d (utang %d + modal %d)",
			bs.Assets.Total, bs.Liabilities.Total+bs.Equity.Total, bs.Liabilities.Total, bs.Equity.Total)
	}
}

// Nilai buku aset harus naik sebesar harga perolehan dikurangi penyusutan, dan neraca
// tetap seimbang di setiap periode seed — termasuk bulan aset pertama kali masuk.
func TestBalanceSheet_SeimbangSemuaPeriode(t *testing.T) {
	txs, assets := finance.SeedTransactions(), finance.SeedAssets()
	for _, p := range finance.AvailablePeriods() {
		rep := finance.BuildPeriodReport(txs, assets, p.ID, finance.SeedOpening(p.ID))
		if !rep.BalanceSheet.IsBalanced {
			bs := rep.BalanceSheet
			t.Errorf("%s tidak seimbang: aktiva %d vs pasiva %d", p.ID, bs.Assets.Total, bs.Liabilities.Total+bs.Equity.Total)
		}
	}
}

// Beli aset secara tunai: kas turun, aset naik, neraca tetap seimbang walau masih draft.
func TestBalanceSheet_SeimbangSetelahBeliAset(t *testing.T) {
	txs := append(finance.SeedTransactions(), finance.Transaction{
		ID: 99_999, Tanggal: "2026-08-15", Jenis: finance.JenisBeliAset, Nominal: 12_000_000,
		Keterangan: "Oven Deck 2 Pintu", Lunas: true,
	})
	rep := finance.BuildPeriodReport(txs, finance.SeedAssets(), "2026-08", seedOpening())

	if !rep.BalanceSheet.IsBalanced {
		bs := rep.BalanceSheet
		t.Fatalf("neraca tidak seimbang setelah beli aset: aktiva %d vs pasiva %d",
			bs.Assets.Total, bs.Liabilities.Total+bs.Equity.Total)
	}
	if rep.CashFlow.Investasi.Net != -12_000_000 {
		t.Errorf("pembelian aset harus masuk aktivitas investasi: got %d", rep.CashFlow.Investasi.Net)
	}
	if rep.CashFlow.ClosingBalance != 97_000_000-12_000_000 {
		t.Errorf("saldo kas akhir: want 85_000_000, got %d", rep.CashFlow.ClosingBalance)
	}
	// Pembelian aset bukan beban, jadi laba bersih tidak berubah.
	if rep.Income.NetProfit != 18_450_000 {
		t.Errorf("laba bersih tidak boleh terpengaruh pembelian aset: got %d", rep.Income.NetProfit)
	}
}

// Modal, pinjaman, cicilan, dan prive masuk aktivitas pendanaan — bukan laba rugi.
func TestCashFlow_AktivitasPendanaan(t *testing.T) {
	entries := []finance.LedgerEntry{
		{Kind: finance.KindIncome, Category: finance.CatSales, Title: "Jual", Gross: 10_000_000, Settled: true},
		{Kind: finance.KindIncome, Category: finance.CatCapital, Title: "Setor modal", Gross: 20_000_000, Settled: true},
		{Kind: finance.KindIncome, Category: finance.CatLoan, Title: "Pinjaman bank", Gross: 30_000_000, Settled: true},
		{Kind: finance.KindExpense, Category: finance.CatLoanPayment, Title: "Cicilan", Gross: 5_000_000, Settled: true},
		{Kind: finance.KindExpense, Category: finance.CatPrive, Title: "Ambil pribadi", Gross: 2_000_000, Settled: true},
		{Kind: finance.KindExpense, Category: finance.CatAsset, Title: "Beli oven", Gross: 8_000_000, Settled: true},
	}
	is := finance.BuildIncomeStatement(entries, 0, 0)
	cf := finance.BuildCashFlow(entries, is, 0)

	if is.NetProfit != 10_000_000 {
		t.Errorf("hanya penjualan yang masuk laba rugi: want 10_000_000, got %d", is.NetProfit)
	}
	if cf.OperasiLangsung.Net != 10_000_000 {
		t.Errorf("kas operasi: want 10_000_000, got %d", cf.OperasiLangsung.Net)
	}
	if cf.Investasi.Net != -8_000_000 {
		t.Errorf("kas investasi: want -8_000_000, got %d", cf.Investasi.Net)
	}
	if cf.Pendanaan.Net != 20_000_000+30_000_000-5_000_000-2_000_000 {
		t.Errorf("kas pendanaan: want 43_000_000, got %d", cf.Pendanaan.Net)
	}
	if cf.NetCash != 45_000_000 {
		t.Errorf("kenaikan kas bersih: want 45_000_000, got %d", cf.NetCash)
	}
	if cf.OperasiLangsung.Net != cf.OperasiTidakLangsung.Net {
		t.Errorf("dua metode beda: %d vs %d", cf.OperasiLangsung.Net, cf.OperasiTidakLangsung.Net)
	}
}
