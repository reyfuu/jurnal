package finance

import "fmt"

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

func FormatJt(rp int64) string {
	jt := float64(rp) / 1_000_000.0
	if rp%1_000_000 == 0 {
		return fmt.Sprintf("Rp%d jt", rp/1_000_000)
	}
	return fmt.Sprintf("Rp%.1f jt", jt)
}

func CalculateUMKMTax(entityType string, monthlyOmzet []int64) UMKMTaxReport {
	const opExemption int64 = 500_000_000
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

		// PPh Final 0.5% = taxable * 5 / 1000
		tax := (taxable * 5) / 1000
		totalTaxable += taxable
		totalTax += tax

		status := "Kena PPh 0,5%"
		if tax == 0 {
			status = "Bebas Pajak (Kuota ≤ Rp500 jt)"
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
			TaxDue:          tax,
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

func SeedKedaiKopiSenja(period string) DashboardSnapshot {
	trend := []MonthlyTrendPoint{
		{Month: "Okt 25", Omzet: 72_000_000, TotalBiaya: 51_000_000, LabaBersih: 21_000_000, EndingCash: 55_000_000},
		{Month: "Nov 25", Omzet: 75_000_000, TotalBiaya: 56_000_000, LabaBersih: 19_000_000, EndingCash: 62_000_000},
		{Month: "Des 25", Omzet: 77_000_000, TotalBiaya: 59_000_000, LabaBersih: 18_000_000, EndingCash: 65_000_000},
		{Month: "Jan 26", Omzet: 79_000_000, TotalBiaya: 58_000_000, LabaBersih: 21_000_000, EndingCash: 68_000_000},
		{Month: "Feb 26", Omzet: 73_000_000, TotalBiaya: 59_000_000, LabaBersih: 14_000_000, EndingCash: 63_000_000},
		{Month: "Mar 26", Omzet: 82_000_000, TotalBiaya: 76_000_000, LabaBersih: 6_000_000, EndingCash: 53_000_000},
		{Month: "Apr 26", Omzet: 80_000_000, TotalBiaya: 62_000_000, LabaBersih: 18_000_000, EndingCash: 59_000_000},
		{Month: "Mei 26", Omzet: 85_000_000, TotalBiaya: 63_000_000, LabaBersih: 22_000_000, EndingCash: 69_000_000},
		{Month: "Jun 26", Omzet: 84_000_000, TotalBiaya: 65_000_000, LabaBersih: 19_000_000, EndingCash: 67_000_000},
		{Month: "Jul 26", Omzet: 88_500_000, TotalBiaya: 66_000_000, LabaBersih: 22_500_000, EndingCash: 80_000_000},
		{Month: "Agu 26", Omzet: 92_000_000, TotalBiaya: 67_000_000, LabaBersih: 25_000_000, EndingCash: 97_000_000},
	}

	dailyValues := []int64{
		3_800_000, 3_100_000, 2_500_000, 2_300_000, 2_100_000,
		2_800_000, 3_600_000, 4_900_000, 3_700_000, 2_400_000,
		2_200_000, 2_000_000, 2_900_000, 3_400_000, 3_600_000,
		3_300_000, 2_200_000, 2_500_000, 2_600_000, 3_200_000,
		3_800_000, 4_700_000, 4_500_000, 3_100_000, 3_700_000,
		2_500_000, 2_700_000, 3_100_000, 4_400_000, 3_500_000,
		1_900_000,
	}
	const dailyTarget int64 = 3_000_000
	dailyPoints := make([]DailyOmzetPoint, 0, len(dailyValues))
	for idx, val := range dailyValues {
		dailyPoints = append(dailyPoints, DailyOmzetPoint{
			Day:         idx + 1,
			Omzet:       val,
			AboveTarget: val >= dailyTarget,
		})
	}

	omzet2026 := []int64{
		79_000_000, 73_000_000, 82_000_000, 80_000_000,
		85_000_000, 84_000_000, 88_500_000, 92_000_000,
	}

	periodLabel := "Agustus 2026"
	if period == "2026-07" {
		periodLabel = "Juli 2026"
	}

	return DashboardSnapshot{
		Tenant: TenantInfo{
			Name:              "Kedai Kopi Senja",
			Initials:          "KK",
			Category:          "FnB",
			TotalTransactions: 4425,
			IsDemo:            true,
		},
		Period:      period,
		PeriodLabel: periodLabel,
		KPIs: DashboardKPIs{
			Omzet: MetricCard{
				Value:     92_000_000,
				Display:   "Rp92 jt",
				BadgeText: "▲ 4%",
				SubText:   "vs bulan lalu",
				Trend:     "up",
				Sparkline: []float64{72, 75, 77, 79, 73, 82, 80, 85, 84, 88.5, 92},
			},
			LabaBersih: MetricCard{
				Value:     25_000_000,
				Display:   "Rp25 jt",
				BadgeText: "▲ 11%",
				SubText:   "vs bulan lalu",
				Trend:     "up",
				Sparkline: []float64{21, 19, 18, 21, 14, 6, 18, 22, 19, 22.5, 25},
			},
			MarginLaba: MetricCard{
				Value:     27,
				Display:   "27%",
				BadgeText: "+1,7 poin",
				SubText:   "vs bulan lalu",
				Trend:     "up",
				Sparkline: []float64{29, 25, 23, 26, 19, 7, 22, 26, 23, 25.3, 27},
			},
			SaldoKas: MetricCard{
				Value:     97_000_000,
				Display:   "Rp97 jt",
				BadgeText: "▲ 21%",
				SubText:   "sejak awal bulan",
				Trend:     "up",
				Sparkline: []float64{55, 62, 65, 68, 63, 53, 59, 69, 67, 80, 97},
			},
			TargetOmzet: TargetCard{
				AchievedPct:    108,
				Display:        "108%",
				SurplusValue:   6_800_000,
				SurplusDisplay: "Lebih Rp6,8 jt",
			},
			Piutang: PiutangCard{
				UnpaidTotal:          11_000_000,
				UnpaidDisplay:        "Rp11 jt",
				InvoiceCount:         10,
				Overdue30dTotal:      9_600_000,
				Overdue30dDisplay:    "Rp9,6 jt lewat 30 hari",
				PayableUnpaidTotal:   7_400_000,
				PayableUnpaidDisplay: "Utang belum dibayar Rp7,4 jt",
			},
		},
		MonthlyTrend11M: trend,
		DailyAverage:    3_000_000,
		DailyTarget:     dailyTarget,
		DailyOmzet:      dailyPoints,
		TotalProducts:   7,
		TopProducts: []ProductPerf{
			{Name: "Es Kopi Susu Senja", Qty: 1920, Omzet: 34_560_000, Pct: 38},
			{Name: "Americano Arabica Gayo", Qty: 940, Omzet: 18_800_000, Pct: 20},
			{Name: "Matcha Oat Latte", Qty: 620, Omzet: 15_500_000, Pct: 17},
			{Name: "Croissant Butter Pairing", Qty: 510, Omzet: 12_750_000, Pct: 14},
			{Name: "Kopi Literan Keluarga 1L", Qty: 148, Omzet: 10_390_000, Pct: 11},
		},
		TotalAppFees:      4_600_000,
		TotalAppFeesLabel: "Potongan aplikasi bulan ini Rp4,6 jt",
		Channels: []ChannelPerf{
			{Name: "Dine-in & Takeaway (QRIS/Tunai)", Gross: 52_000_000, Fee: 0, Net: 52_000_000, Pct: 57},
			{Name: "GoFood (Komisi 20%)", Gross: 23_000_000, Fee: 2_600_000, Net: 20_400_000, Pct: 25},
			{Name: "GrabFood (Komisi 20%)", Gross: 12_000_000, Fee: 1_400_000, Net: 10_600_000, Pct: 13},
			{Name: "ShopeeFood & Catering B2B", Gross: 5_000_000, Fee: 600_000, Net: 4_400_000, Pct: 5},
		},
		RecentTransactions: []TransactionItem{
			{ID: "TX-4425", Date: "31 Agu 2026", Type: "INCOME", Category: "Penjualan", Title: "Settlement GoFood Harian (84 cup)", Channel: "GoFood", Gross: 2_100_000, Fee: 420_000, Net: 1_680_000},
			{ID: "TX-4424", Date: "31 Agu 2026", Type: "INCOME", Category: "Penjualan", Title: "Kasir POS Dine-in & QRIS (112 cup)", Channel: "Dine-in", Gross: 2_850_000, Fee: 0, Net: 2_850_000},
			{ID: "TX-4423", Date: "30 Agu 2026", Type: "EXPENSE", Category: "Bahan Baku (HPP)", Title: "Restock Susu Fresh Milk & Oatside 12 Karton", Channel: "Supplier", Gross: 3_450_000, Fee: 0, Net: -3_450_000},
			{ID: "TX-4422", Date: "29 Agu 2026", Type: "INCOME", Category: "Catering B2B", Title: "Invoice #INV-2026-089 PT Nusantara Kreatif", Channel: "B2B Tempo", Gross: 4_200_000, Fee: 0, Net: 4_200_000},
			{ID: "TX-4421", Date: "28 Agu 2026", Type: "EXPENSE", Category: "Operasional", Title: "Gaji Barista & Roaster (4 Staf)", Channel: "BCA Bisnis", Gross: 12_500_000, Fee: 0, Net: -12_500_000},
		},
		Obligations: []ObligationItem{
			{InvoiceNo: "INV-2026-061", Counterparty: "PT Karya Digital Mandiri (Kopi Rapat Bulanan)", Kind: "RECEIVABLE", DueDate: "10 Jul 2026", DaysOverdue: 52, Amount: 5_400_000},
			{InvoiceNo: "INV-2026-068", Counterparty: "Event Organizer Senayan Fest", Kind: "RECEIVABLE", DueDate: "22 Jul 2026", DaysOverdue: 40, Amount: 4_200_000},
			{InvoiceNo: "INV-2026-089", Counterparty: "Coworking Space Kemang", Kind: "RECEIVABLE", DueDate: "05 Sep 2026", DaysOverdue: 0, Amount: 1_400_000},
			{InvoiceNo: "PO-SUP-112", Counterparty: "Roastery Biji Kopi Gayo Lestari", Kind: "PAYABLE", DueDate: "08 Sep 2026", DaysOverdue: 0, Amount: 4_800_000},
			{InvoiceNo: "PO-SUP-118", Counterparty: " Kemasan Cup & Sablon Senja", Kind: "PAYABLE", DueDate: "12 Sep 2026", DaysOverdue: 0, Amount: 2_600_000},
		},
		Inventory: []InventoryItem{
			{SKU: "RM-001", Name: "Biji Kopi Arabica Gayo (Roasted)", Type: "Bahan Baku", Stock: "42,5 kg", COGS: 185_000, SellPrice: 0, Status: "Aman"},
			{SKU: "RM-002", Name: "Fresh Milk Full Cream 1L", Type: "Bahan Baku", Stock: "18 liter", COGS: 21_500, SellPrice: 0, Status: "Stok Menipis"},
			{SKU: "FG-001", Name: "Es Kopi Susu Senja (12oz)", Type: "Produk Jadi", Stock: "Resep BOM Aktif", COGS: 6_800, SellPrice: 18_000, Status: "Margin 62%"},
			{SKU: "FG-002", Name: "Matcha Oat Latte (12oz)", Type: "Produk Jadi", Stock: "Resep BOM Aktif", COGS: 9_500, SellPrice: 25_000, Status: "Margin 62%"},
		},
		TaxReport: CalculateUMKMTax("OP", omzet2026),
	}
}
