package finance

import "errors"

// BreakEvenResult: titik impas (BEP) per bulan dari biaya tetap dan margin kontribusi per unit.
type BreakEvenResult struct {
	Price              int64 `json:"price"`
	VariableCost       int64 `json:"variable_cost"`       // HPP + komisi per unit
	ContributionMargin int64 `json:"contribution_margin"` // harga - biaya variabel
	FixedCost          int64 `json:"fixed_cost"`          // per bulan (gaji, sewa, listrik, ...)
	BEPUnits           int64 `json:"bep_units"`
	BEPUnitsPerDay     int64 `json:"bep_units_per_day"` // asumsi 30 hari buka
	BEPRevenue         int64 `json:"bep_revenue"`
	MonthlyVolume      int64 `json:"monthly_volume"`
	NetProfitAtVolume  int64 `json:"net_profit_at_volume"`
	TargetNetProfit    int64 `json:"target_net_profit"`
	UnitsForTarget     int64 `json:"units_for_target"`
}

var ErrPriceBelowVariableCost = errors.New("harga jual di bawah atau sama dengan biaya variabel, setiap penjualan menambah rugi")

func ceilDiv(a, b int64) int64 { return (a + b - 1) / b }

func BreakEven(price, variableCost, fixedCost, monthlyVolume, targetNetProfit int64) (BreakEvenResult, error) {
	margin := price - variableCost
	if margin <= 0 {
		return BreakEvenResult{}, ErrPriceBelowVariableCost
	}
	units := ceilDiv(fixedCost, margin)
	return BreakEvenResult{
		Price: price, VariableCost: variableCost, ContributionMargin: margin, FixedCost: fixedCost,
		BEPUnits: units, BEPUnitsPerDay: ceilDiv(units, 30), BEPRevenue: units * price,
		MonthlyVolume: monthlyVolume, NetProfitAtVolume: margin*monthlyVolume - fixedCost,
		TargetNetProfit: targetNetProfit, UnitsForTarget: ceilDiv(fixedCost+targetNetProfit, margin),
	}, nil
}
