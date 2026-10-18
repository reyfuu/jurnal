package finance_test

import (
	"testing"

	"pajak/internal/finance"
)

func TestCalculateDashboardKPIs_Agustus2026(t *testing.T) {
	snapshot := finance.SeedKedaiKopiSenja("2026-08")

	if snapshot.KPIs.Omzet.Value != 92_000_000 {
		t.Fatalf("expected Omzet 92_000_000, got %d", snapshot.KPIs.Omzet.Value)
	}
	if snapshot.KPIs.Omzet.Display != "Rp92 jt" {
		t.Fatalf("expected Omzet display 'Rp92 jt', got %q", snapshot.KPIs.Omzet.Display)
	}
	if snapshot.KPIs.LabaBersih.Value != 25_000_000 {
		t.Fatalf("expected LabaBersih 25_000_000, got %d", snapshot.KPIs.LabaBersih.Value)
	}
	if snapshot.KPIs.MarginLaba.Display != "27%" {
		t.Fatalf("expected MarginLaba display '27%%', got %q", snapshot.KPIs.MarginLaba.Display)
	}
	if snapshot.KPIs.SaldoKas.Value != 97_000_000 {
		t.Fatalf("expected SaldoKas 97_000_000, got %d", snapshot.KPIs.SaldoKas.Value)
	}
	if snapshot.KPIs.TargetOmzet.AchievedPct != 108 {
		t.Fatalf("expected TargetOmzet 108%%, got %d", snapshot.KPIs.TargetOmzet.AchievedPct)
	}
	if snapshot.KPIs.Piutang.UnpaidTotal != 11_000_000 || snapshot.KPIs.Piutang.Overdue30dTotal != 9_600_000 {
		t.Fatalf("unexpected Piutang totals: %+v", snapshot.KPIs.Piutang)
	}
}

func TestCalculateUMKMTax_OrangPribadi_500JtThreshold(t *testing.T) {
	// PP 55/2022: WP Orang Pribadi bebas PPh Final 0.5% untuk 500 juta pertama per tahun.
	monthlyOmzet := []int64{
		80_000_000, // Jan: Cum 80jt -> Tax 0
		85_000_000, // Feb: Cum 165jt -> Tax 0
		90_000_000, // Mar: Cum 255jt -> Tax 0
		80_000_000, // Apr: Cum 335jt -> Tax 0
		85_000_000, // Mei: Cum 420jt -> Tax 0
		90_000_000, // Jun: Cum 510jt -> Taxable 10jt * 0.5% = Rp50.000
		92_000_000, // Jul: Cum 602jt -> Taxable 92jt * 0.5% = Rp460.000
	}

	report := finance.CalculateUMKMTax("OP", monthlyOmzet)

	if report.Months[4].TaxDue != 0 {
		t.Fatalf("expected Mei tax 0 (cum 420jt <= 500jt), got %d", report.Months[4].TaxDue)
	}
	if report.Months[5].TaxableAmount != 10_000_000 || report.Months[5].TaxDue != 50_000 {
		t.Fatalf("expected Jun taxable 10jt & tax 50_000, got taxable=%d tax=%d",
			report.Months[5].TaxableAmount, report.Months[5].TaxDue)
	}
	if report.Months[6].TaxDue != 460_000 {
		t.Fatalf("expected Jul tax 460_000, got %d", report.Months[6].TaxDue)
	}
	if report.TotalTaxDue != 510_000 {
		t.Fatalf("expected TotalTaxDue 510_000, got %d", report.TotalTaxDue)
	}
}

func TestSimulateChannelPrice(t *testing.T) {
	// HPP Rp10.000, Target Laba Bersih Rp8.000, Komisi Aplikasi 20% (2000 bps)
	// Harga Jual harus Rp22.500 karena 22.500 - (20% * 22.500 = 4.500) = 18.000 -> 18.000 - 10.000 = 8.000
	res := finance.SimulateChannelPrice(10_000, 8_000, 2000, 0)
	if res.RecommendedPrice != 22_500 {
		t.Fatalf("expected RecommendedPrice 22500, got %d", res.RecommendedPrice)
	}
	if res.ChannelFee != 4_500 {
		t.Fatalf("expected ChannelFee 4500, got %d", res.ChannelFee)
	}
}
