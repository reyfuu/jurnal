package finance

import "testing"

func TestBreakEven(t *testing.T) {
	// harga 18.000, biaya variabel 10.400 (HPP 6.800 + komisi 3.600) → margin 7.600/pcs
	got, err := BreakEven(18_000, 10_400, 38_000_000, 1_500, 15_200_000)
	if err != nil {
		t.Fatal(err)
	}
	want := BreakEvenResult{
		Price: 18_000, VariableCost: 10_400, ContributionMargin: 7_600, FixedCost: 38_000_000,
		BEPUnits: 5_000, BEPUnitsPerDay: 167, BEPRevenue: 90_000_000,
		MonthlyVolume: 1_500, NetProfitAtVolume: 7_600*1_500 - 38_000_000,
		TargetNetProfit: 15_200_000, UnitsForTarget: 7_000,
	}
	if got != want {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

func TestBreakEven_RoundsUp(t *testing.T) {
	got, _ := BreakEven(10_000, 7_000, 10_000, 0, 0)
	if got.BEPUnits != 4 { // 10.000 / 3.000 = 3,33 → 4 pcs
		t.Fatalf("BEP units = %d, want 4", got.BEPUnits)
	}
}

func TestBreakEven_PriceBelowVariableCost(t *testing.T) {
	if _, err := BreakEven(10_000, 10_000, 5_000_000, 0, 0); err == nil {
		t.Fatal("want error when price <= variable cost")
	}
}
