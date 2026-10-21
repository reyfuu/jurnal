package finance

import (
	"fmt"

	"pajak/internal/tax"
)

// ponytail: in-memory deterministic seed + int64 Rupiah computation for 100-user scale; swap to pgx pool when persistent DB is attached.

type TenantInfo struct {
	Name              string `json:"name"`
	Initials          string `json:"initials"`
	Category          string `json:"category"`
	TotalTransactions int    `json:"total_transactions"`
	IsDemo            bool   `json:"is_demo"`
}

type MetricCard struct {
	Value     int64     `json:"value"`
	Display   string    `json:"display"`
	BadgeText string    `json:"badge_text"`
	SubText   string    `json:"sub_text,omitempty"`
	Trend     string    `json:"trend"`
	Sparkline []float64 `json:"sparkline"`
}

type TargetCard struct {
	AchievedPct    int    `json:"achieved_pct"`
	Display        string `json:"display"`
	SurplusValue   int64  `json:"surplus_value"`
	SurplusDisplay string `json:"surplus_display"`
}

type PiutangCard struct {
	UnpaidTotal          int64  `json:"unpaid_total"`
	UnpaidDisplay        string `json:"unpaid_display"`
	InvoiceCount         int    `json:"invoice_count"`
	Overdue30dTotal      int64  `json:"overdue_30d_total"`
	Overdue30dDisplay    string `json:"overdue_30d_display"`
	PayableUnpaidTotal   int64  `json:"payable_unpaid_total"`
	PayableUnpaidDisplay string `json:"payable_unpaid_display"`
}

type DashboardKPIs struct {
	Omzet       MetricCard  `json:"omzet"`
	LabaBersih  MetricCard  `json:"laba_bersih"`
	MarginLaba  MetricCard  `json:"margin_laba"`
	SaldoKas    MetricCard  `json:"saldo_kas"`
	TargetOmzet TargetCard  `json:"target_omzet"`
	Piutang     PiutangCard `json:"piutang"`
}

type MonthlyTrendPoint struct {
	Month      string `json:"month"`
	Omzet      int64  `json:"omzet"`
	TotalBiaya int64  `json:"total_biaya"`
	LabaBersih int64  `json:"laba_bersih"`
	EndingCash int64  `json:"ending_cash"`
}

type DailyOmzetPoint struct {
	Day         int   `json:"day"`
	Omzet       int64 `json:"omzet"`
	AboveTarget bool  `json:"above_target"`
}

type ProductPerf struct {
	Name  string `json:"name"`
	Qty   int    `json:"qty"`
	Omzet int64  `json:"omzet"`
	Pct   int    `json:"pct"`
}

type ChannelPerf struct {
	Name  string `json:"name"`
	Gross int64  `json:"gross"`
	Fee   int64  `json:"fee"`
	Net   int64  `json:"net"`
	Pct   int    `json:"pct"`
}

type TransactionItem struct {
	ID       string `json:"id"`
	Date     string `json:"date"`
	Type     string `json:"type"` // INCOME | EXPENSE
	Category string `json:"category"`
	Title    string `json:"title"`
	Channel  string `json:"channel"`
	Gross    int64  `json:"gross"`
	Fee      int64  `json:"fee"`
	Net      int64  `json:"net"`
}

type ObligationItem struct {
	InvoiceNo    string `json:"invoice_no"`
	Counterparty string `json:"counterparty"`
	Kind         string `json:"kind"` // RECEIVABLE | PAYABLE
	DueDate      string `json:"due_date"`
	DaysOverdue  int    `json:"days_overdue"`
	Amount       int64  `json:"amount"`
}

type InventoryItem struct {
	SKU       string `json:"sku"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Stock     string `json:"stock"`
	COGS      int64  `json:"cogs"`
	SellPrice int64  `json:"sell_price"`
	Status    string `json:"status"`
}

type DashboardSnapshot struct {
	CashWarnings       []CashWarning       `json:"cash_warnings"`
	Tenant             TenantInfo          `json:"tenant"`
	Period             string              `json:"period"`
	PeriodLabel        string              `json:"period_label"`
	KPIs               DashboardKPIs       `json:"kpis"`
	MonthlyTrend11M    []MonthlyTrendPoint `json:"monthly_trend_11m"`
	DailyAverage       int64               `json:"daily_average"`
	DailyTarget        int64               `json:"daily_target"`
	DailyOmzet         []DailyOmzetPoint   `json:"daily_omzet"`
	TotalProducts      int                 `json:"total_products"`
	TopProducts        []ProductPerf       `json:"top_products"`
	TotalAppFees       int64               `json:"total_app_fees"`
	TotalAppFeesLabel  string              `json:"total_app_fees_label"`
	Channels           []ChannelPerf       `json:"channels"`
	RecentTransactions []TransactionItem   `json:"recent_transactions"`
	Obligations        []ObligationItem    `json:"obligations"`
	Inventory          []InventoryItem     `json:"inventory"`
	TaxReport          UMKMTaxReport       `json:"tax_report"`
}

type UMKMTaxMonth struct {
	MonthName       string `json:"month_name"`
	GrossOmzet      int64  `json:"gross_omzet"`
	CumulativeOmzet int64  `json:"cumulative_omzet"`
	TaxableAmount   int64  `json:"taxable_amount"`
	TaxDue          int64  `json:"tax_due"`
	Status          string `json:"status"`
}

type UMKMTaxReport struct {
	EntityType      string         `json:"entity_type"`
	YearlyExemption int64          `json:"yearly_exemption"`
	TotalGross      int64          `json:"total_gross"`
	TotalTaxable    int64          `json:"total_taxable"`
	TotalTaxDue     int64          `json:"total_tax_due"`
	Months          []UMKMTaxMonth `json:"months"`
}

type PriceSimulationResult struct {
	COGS             int64 `json:"cogs"`
	TargetProfit     int64 `json:"target_profit"`
	FeeBps           int   `json:"fee_bps"`
	FixedFee         int64 `json:"fixed_fee"`
	RecommendedPrice int64 `json:"recommended_price"`
	ChannelFee       int64 `json:"channel_fee"`
	NetTakeHome      int64 `json:"net_take_home"`
	NetMarginPct     int   `json:"net_margin_pct"`
}

func CalculateUMKMTax(entityType string, monthlyOmzet []int64) UMKMTaxReport {
	pr := tax.P()
	opExemption := pr.UMKMBebasOP
	monthNames := []string{
		"Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember",
	}

	var cum int64
	var totalTaxable int64
	var totalTax int64
	months := make([]UMKMTaxMonth, 0, len(monthlyOmzet))

	for i, omzet := range monthlyOmzet {
		prevCum := cum
		cum += omzet
		var taxable int64
		if entityType == "OP" {
			if cum <= opExemption {
				taxable = 0
			} else if prevCum < opExemption {
				taxable = cum - opExemption
			} else {
				taxable = omzet
			}
		} else {
			taxable = omzet
		}

		// PPh final UMKM = taxable x tarif (bawaan 0,5%)
		pph := taxable * pr.UMKMTarifBps / 10_000
		totalTaxable += taxable
		totalTax += pph

		status := "Kena PPh " + persen(int(pr.UMKMTarifBps))
		if pph == 0 {
			status = "Bebas Pajak (Kuota ≤ " + FormatRupiahPenuh(opExemption) + ")"
		}

		mName := fmt.Sprintf("Bulan %d", i+1)
		if i < len(monthNames) {
			mName = monthNames[i]
		}

		months = append(months, UMKMTaxMonth{
			MonthName:       mName,
			GrossOmzet:      omzet,
			CumulativeOmzet: cum,
			TaxableAmount:   taxable,
			TaxDue:          pph,
			Status:          status,
		})
	}

	exemption := int64(0)
	if entityType == "OP" {
		exemption = opExemption
	}

	return UMKMTaxReport{
		EntityType:      entityType,
		YearlyExemption: exemption,
		TotalGross:      cum,
		TotalTaxable:    totalTaxable,
		TotalTaxDue:     totalTax,
		Months:          months,
	}
}

func SimulateChannelPrice(cogs int64, targetProfit int64, feeBps int, fixedFee int64) PriceSimulationResult {
	if feeBps >= 10000 {
		feeBps = 9900
	}
	requiredBeforePct := cogs + targetProfit + fixedFee
	// price * (10000 - feeBps) / 10000 = requiredBeforePct
	denom := int64(10000 - feeBps)
	price := (requiredBeforePct*10000 + denom - 1) / denom
	channelFee := (price*int64(feeBps))/10000 + fixedFee
	netTakeHome := price - channelFee
	marginPct := 0
	if price > 0 {
		marginPct = int(((netTakeHome - cogs) * 100) / price)
	}
	return PriceSimulationResult{
		COGS:             cogs,
		TargetProfit:     targetProfit,
		FeeBps:           feeBps,
		FixedFee:         fixedFee,
		RecommendedPrice: price,
		ChannelFee:       channelFee,
		NetTakeHome:      netTakeHome,
		NetMarginPct:     marginPct,
	}
}
