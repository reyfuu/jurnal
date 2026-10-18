package finance_test

import (
	"testing"

	"pajak/internal/finance"
)

func TestAllPeriodsSeed(t *testing.T) {
	periods := []string{
		"2026-08", "2026-07", "2026-06", "2026-05",
		"2026-04", "2026-03", "2026-02", "2026-01",
	}

	for _, p := range periods {
		snap := finance.SeedKedaiKopiSenja(p)
		if snap.Period != p {
			t.Errorf("expected period %s, got %s", p, snap.Period)
		}
		if snap.KPIs.Omzet.Value <= 0 {
			t.Errorf("period %s: omzet must be positive, got %d", p, snap.KPIs.Omzet.Value)
		}
		if snap.KPIs.LabaBersih.Value <= 0 {
			t.Errorf("period %s: laba bersih must be positive, got %d", p, snap.KPIs.LabaBersih.Value)
		}

		ledger := finance.SeedLedger(p)
		if len(ledger) == 0 {
			t.Errorf("period %s: ledger must not be empty", p)
		}

		is := finance.BuildIncomeStatement(ledger)
		if is.Omzet != snap.KPIs.Omzet.Value {
			t.Errorf("period %s: ledger omzet %d != snapshot omzet %d", p, is.Omzet, snap.KPIs.Omzet.Value)
		}
	}
}
