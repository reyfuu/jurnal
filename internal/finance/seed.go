package finance

import "fmt"

type MonthSeedData struct {
	Label        string
	Omzet        int64
	FeeApp       int64
	COGS         int64
	Opex         int64
	LabaBersih   int64
	MarginBps    int
	OpeningCash  int64
	ClosingCash  int64
	Transactions int
	TargetOmzet  int64
	UnpaidAR     int64
	Overdue30d   int64
	UnpaidAP     int64
	SparkOmzet   []float64
	SparkLaba    []float64
	SparkCash    []float64
	Channels     []ChannelPerf
	RecentTxs    []TransactionItem
	Ledger       []LedgerEntry // ringkasan bulan; diurai jadi transaksi oleh SeedTransactions
}

var periodsData = map[string]MonthSeedData{
	"2026-08": {
		Label: "Agustus 2026", Omzet: 92_000_000, FeeApp: 4_600_000, COGS: 26_000_000, Opex: 36_400_000,
		LabaBersih: 25_000_000, MarginBps: 2717, OpeningCash: 80_000_000, ClosingCash: 97_000_000,
		Transactions: 4425, TargetOmzet: 85_000_000, UnpaidAR: 11_000_000, Overdue30d: 9_600_000, UnpaidAP: 7_400_000,
		SparkOmzet: []float64{72, 75, 77, 79, 73, 82, 80, 85, 84, 88.5, 92},
		SparkLaba:  []float64{21, 19, 18, 21, 14, 6, 18, 22, 19, 22.5, 25},
		SparkCash:  []float64{55, 62, 65, 68, 63, 53, 59, 69, 67, 80, 97},
		Channels: []ChannelPerf{
			{Name: "Dine-in & Takeaway (QRIS/Tunai)", Gross: 52_000_000, Fee: 0, Net: 52_000_000, Pct: 57},
			{Name: "GoFood (Komisi 20%)", Gross: 23_000_000, Fee: 2_600_000, Net: 20_400_000, Pct: 25},
			{Name: "GrabFood (Komisi 20%)", Gross: 12_000_000, Fee: 1_400_000, Net: 10_600_000, Pct: 13},
			{Name: "ShopeeFood & Catering B2B", Gross: 5_000_000, Fee: 600_000, Net: 4_400_000, Pct: 5},
		},
		RecentTxs: []TransactionItem{
			{ID: "TX-4425", Date: "31 Agu 2026", Type: "INCOME", Category: "Penjualan", Title: "Settlement GoFood Harian (84 cup)", Channel: "GoFood", Gross: 2_100_000, Fee: 420_000, Net: 1_680_000},
			{ID: "TX-4424", Date: "31 Agu 2026", Type: "INCOME", Category: "Penjualan", Title: "Kasir POS Dine-in & QRIS (112 cup)", Channel: "Dine-in", Gross: 2_850_000, Fee: 0, Net: 2_850_000},
			{ID: "TX-4423", Date: "30 Agu 2026", Type: "EXPENSE", Category: "Bahan Baku (HPP)", Title: "Restock Susu Fresh Milk & Oatside 12 Karton", Channel: "Supplier", Gross: 3_450_000, Fee: 0, Net: -3_450_000},
			{ID: "TX-4422", Date: "29 Agu 2026", Type: "INCOME", Category: "Catering B2B", Title: "Invoice #INV-2026-089 PT Nusantara Kreatif", Channel: "B2B Tempo", Gross: 4_200_000, Fee: 0, Net: 4_200_000},
			{ID: "TX-4421", Date: "28 Agu 2026", Type: "EXPENSE", Category: "Operasional", Title: "Gaji Barista & Roaster (4 Staf)", Channel: "BCA Bisnis", Gross: 12_500_000, Fee: 0, Net: -12_500_000},
			{ID: "TX-4420", Date: "27 Agu 2026", Type: "INCOME", Category: "Penjualan", Title: "Settlement GrabFood Harian (56 cup)", Channel: "GrabFood", Gross: 1_400_000, Fee: 280_000, Net: 1_120_000},
			{ID: "TX-4419", Date: "26 Agu 2026", Type: "INCOME", Category: "Penjualan", Title: "Kasir POS Sore & Malam", Channel: "Dine-in", Gross: 1_950_000, Fee: 0, Net: 1_950_000},
			{ID: "TX-4418", Date: "25 Agu 2026", Type: "EXPENSE", Category: "Operasional", Title: "Listrik PLN & Internet Biznet", Channel: "BCA Bisnis", Gross: 2_400_000, Fee: 0, Net: -2_400_000},
		},
		Ledger: []LedgerEntry{
			{KindIncome, CatSales, "Kasir Dine-in & Takeaway", "Dine-in", 41_000_000, 0, true},
			{KindIncome, CatSales, "Catering B2B (tempo)", "B2B Tempo", 11_000_000, 0, false},
			{KindIncome, CatSales, "GoFood", "GoFood", 23_000_000, 2_600_000, true},
			{KindIncome, CatSales, "GrabFood", "GrabFood", 12_000_000, 1_400_000, true},
			{KindIncome, CatSales, "ShopeeFood", "ShopeeFood", 5_000_000, 600_000, true},
			{KindExpense, CatCOGS, "Bahan baku (kopi, susu, cup)", "", 26_000_000, 0, true},
			{KindExpense, "OPEX", "Gaji barista & roaster", "", 12_500_000, 0, true},
			{KindExpense, "OPEX", "Sewa tempat", "", 8_000_000, 0, true},
			{KindExpense, "OPEX", "Listrik, internet, marketing, lain-lain", "", 12_900_000, 0, true},
			{KindExpense, "OPEX", "Tagihan listrik (belum dibayar)", "", 3_000_000, 0, false},
		},
	},
	"2026-07": {
		Label: "Juli 2026", Omzet: 88_500_000, FeeApp: 4_400_000, COGS: 25_100_000, Opex: 36_500_000,
		LabaBersih: 22_500_000, MarginBps: 2542, OpeningCash: 67_000_000, ClosingCash: 80_000_000,
		Transactions: 4210, TargetOmzet: 85_000_000, UnpaidAR: 9_500_000, Overdue30d: 5_400_000, UnpaidAP: 6_200_000,
		SparkOmzet: []float64{72, 75, 77, 79, 73, 82, 80, 85, 84, 88.5},
		SparkLaba:  []float64{21, 19, 18, 21, 14, 6, 18, 22, 19, 22.5},
		SparkCash:  []float64{55, 62, 65, 68, 63, 53, 59, 69, 67, 80},
		Channels: []ChannelPerf{
			{Name: "Dine-in & Takeaway (QRIS/Tunai)", Gross: 50_000_000, Fee: 0, Net: 50_000_000, Pct: 56},
			{Name: "GoFood (Komisi 20%)", Gross: 22_500_000, Fee: 2_500_000, Net: 20_000_000, Pct: 25},
			{Name: "GrabFood (Komisi 20%)", Gross: 11_500_000, Fee: 1_300_000, Net: 10_200_000, Pct: 13},
			{Name: "ShopeeFood & Catering B2B", Gross: 4_500_000, Fee: 600_000, Net: 3_900_000, Pct: 6},
		},
		RecentTxs: []TransactionItem{
			{ID: "TX-4180", Date: "31 Jul 2026", Type: "INCOME", Category: "Penjualan", Title: "Settlement POS Dine-in Weekend", Channel: "Dine-in", Gross: 3_100_000, Fee: 0, Net: 3_100_000},
			{ID: "TX-4179", Date: "30 Jul 2026", Type: "INCOME", Category: "Penjualan", Title: "Settlement GoFood Promo Liburan", Channel: "GoFood", Gross: 2_400_000, Fee: 480_000, Net: 1_920_000},
			{ID: "TX-4178", Date: "29 Jul 2026", Type: "EXPENSE", Category: "Bahan Baku (HPP)", Title: "Biji Kopi Arabica Gayo 25kg", Channel: "Supplier", Gross: 4_625_000, Fee: 0, Net: -4_625_000},
			{ID: "TX-4177", Date: "28 Jul 2026", Type: "EXPENSE", Category: "Operasional", Title: "Gaji Staf Barista Juli", Channel: "BCA Bisnis", Gross: 12_500_000, Fee: 0, Net: -12_500_000},
		},
		Ledger: []LedgerEntry{
			{KindIncome, CatSales, "Dine-in & QRIS", "Dine-in", 40_500_000, 0, true},
			{KindIncome, CatSales, "Pesanan Catering Kantor", "B2B Tempo", 9_500_000, 0, false},
			{KindIncome, CatSales, "GoFood", "GoFood", 22_500_000, 2_500_000, true},
			{KindIncome, CatSales, "GrabFood", "GrabFood", 11_500_000, 1_300_000, true},
			{KindIncome, CatSales, "ShopeeFood", "ShopeeFood", 4_500_000, 600_000, true},
			{KindExpense, CatCOGS, "Bahan baku kopi, susu, sirup", "", 25_100_000, 0, true},
			{KindExpense, "OPEX", "Gaji staf", "", 12_500_000, 0, true},
			{KindExpense, "OPEX", "Sewa tempat", "", 8_000_000, 0, true},
			{KindExpense, "OPEX", "Operasional rutin", "", 13_500_000, 0, true},
			{KindExpense, "OPEX", "Utang kemasan paper cup", "", 2_500_000, 0, false},
		},
	},
	"2026-06": {
		Label: "Juni 2026", Omzet: 84_000_000, FeeApp: 4_200_000, COGS: 24_800_000, Opex: 36_000_000,
		LabaBersih: 19_000_000, MarginBps: 2261, OpeningCash: 69_000_000, ClosingCash: 67_000_000,
		Transactions: 3950, TargetOmzet: 80_000_000, UnpaidAR: 8_000_000, Overdue30d: 4_200_000, UnpaidAP: 5_500_000,
		SparkOmzet: []float64{72, 75, 77, 79, 73, 82, 80, 85, 84},
		SparkLaba:  []float64{21, 19, 18, 21, 14, 6, 18, 22, 19},
		SparkCash:  []float64{55, 62, 65, 68, 63, 53, 59, 69, 67},
		Channels: []ChannelPerf{
			{Name: "Dine-in & Takeaway (QRIS/Tunai)", Gross: 48_000_000, Fee: 0, Net: 48_000_000, Pct: 57},
			{Name: "GoFood (Komisi 20%)", Gross: 21_000_000, Fee: 2_400_000, Net: 18_600_000, Pct: 25},
			{Name: "GrabFood (Komisi 20%)", Gross: 11_000_000, Fee: 1_300_000, Net: 9_700_000, Pct: 13},
			{Name: "ShopeeFood & Catering B2B", Gross: 4_000_000, Fee: 500_000, Net: 3_500_000, Pct: 5},
		},
		RecentTxs: []TransactionItem{
			{ID: "TX-3920", Date: "30 Jun 2026", Type: "INCOME", Category: "Penjualan", Title: "Kasir POS Akhir Bulan", Channel: "Dine-in", Gross: 2_600_000, Fee: 0, Net: 2_600_000},
			{ID: "TX-3919", Date: "28 Jun 2026", Type: "EXPENSE", Category: "Operasional", Title: "Gaji Karyawan Juni", Channel: "BCA Bisnis", Gross: 12_500_000, Fee: 0, Net: -12_500_000},
		},
		Ledger: []LedgerEntry{
			{KindIncome, CatSales, "Dine-in POS", "Dine-in", 40_000_000, 0, true},
			{KindIncome, CatSales, "Pesanan Catering", "B2B Tempo", 8_000_000, 0, false},
			{KindIncome, CatSales, "GoFood", "GoFood", 21_000_000, 2_400_000, true},
			{KindIncome, CatSales, "GrabFood", "GrabFood", 11_000_000, 1_300_000, true},
			{KindIncome, CatSales, "ShopeeFood", "ShopeeFood", 4_000_000, 500_000, true},
			{KindExpense, CatCOGS, "Bahan baku", "", 24_800_000, 0, true},
			{KindExpense, "OPEX", "Gaji barista", "", 12_500_000, 0, true},
			{KindExpense, "OPEX", "Sewa tempat", "", 8_000_000, 0, true},
			{KindExpense, "OPEX", "Biaya listrik & utilitas", "", 13_500_000, 0, true},
			{KindExpense, "OPEX", "Tagihan supplier", "", 2_000_000, 0, false},
		},
	},
	"2026-05": {
		Label: "Mei 2026", Omzet: 85_000_000, FeeApp: 4_300_000, COGS: 24_200_000, Opex: 34_500_000,
		LabaBersih: 22_000_000, MarginBps: 2588, OpeningCash: 59_000_000, ClosingCash: 69_000_000,
		Transactions: 4050, TargetOmzet: 80_000_000, UnpaidAR: 7_500_000, Overdue30d: 3_100_000, UnpaidAP: 4_800_000,
		SparkOmzet: []float64{72, 75, 77, 79, 73, 82, 80, 85},
		SparkLaba:  []float64{21, 19, 18, 21, 14, 6, 18, 22},
		SparkCash:  []float64{55, 62, 65, 68, 63, 53, 59, 69},
		Channels: []ChannelPerf{
			{Name: "Dine-in & Takeaway (QRIS/Tunai)", Gross: 49_000_000, Fee: 0, Net: 49_000_000, Pct: 58},
			{Name: "GoFood (Komisi 20%)", Gross: 21_500_000, Fee: 2_400_000, Net: 19_100_000, Pct: 25},
			{Name: "GrabFood (Komisi 20%)", Gross: 10_500_000, Fee: 1_200_000, Net: 9_300_000, Pct: 12},
			{Name: "ShopeeFood & Catering B2B", Gross: 4_000_000, Fee: 700_000, Net: 3_300_000, Pct: 5},
		},
		RecentTxs: []TransactionItem{
			{ID: "TX-3680", Date: "31 Mei 2026", Type: "INCOME", Category: "Penjualan", Title: "Kasir POS Akhir Pekan", Channel: "Dine-in", Gross: 2_900_000, Fee: 0, Net: 2_900_000},
			{ID: "TX-3679", Date: "29 Mei 2026", Type: "EXPENSE", Category: "Operasional", Title: "Gaji Barista Mei", Channel: "BCA Bisnis", Gross: 12_500_000, Fee: 0, Net: -12_500_000},
		},
		Ledger: []LedgerEntry{
			{KindIncome, CatSales, "Dine-in POS", "Dine-in", 41_500_000, 0, true},
			{KindIncome, CatSales, "Catering B2B", "B2B Tempo", 7_500_000, 0, false},
			{KindIncome, CatSales, "GoFood", "GoFood", 21_500_000, 2_400_000, true},
			{KindIncome, CatSales, "GrabFood", "GrabFood", 10_500_000, 1_200_000, true},
			{KindIncome, CatSales, "ShopeeFood", "ShopeeFood", 4_000_000, 700_000, true},
			{KindExpense, CatCOGS, "Bahan baku", "", 24_200_000, 0, true},
			{KindExpense, "OPEX", "Gaji barista", "", 12_500_000, 0, true},
			{KindExpense, "OPEX", "Sewa tempat", "", 8_000_000, 0, true},
			{KindExpense, "OPEX", "Operasional umum", "", 12_000_000, 0, true},
			{KindExpense, "OPEX", "Tagihan supplier", "", 2_000_000, 0, false},
		},
	},
	"2026-04": {
		Label: "April 2026", Omzet: 80_000_000, FeeApp: 4_000_000, COGS: 23_500_000, Opex: 34_500_000,
		LabaBersih: 18_000_000, MarginBps: 2250, OpeningCash: 53_000_000, ClosingCash: 59_000_000,
		Transactions: 3750, TargetOmzet: 75_000_000, UnpaidAR: 6_000_000, Overdue30d: 2_000_000, UnpaidAP: 4_000_000,
		SparkOmzet: []float64{72, 75, 77, 79, 73, 82, 80},
		SparkLaba:  []float64{21, 19, 18, 21, 14, 6, 18},
		SparkCash:  []float64{55, 62, 65, 68, 63, 53, 59},
		Channels: []ChannelPerf{
			{Name: "Dine-in & Takeaway (QRIS/Tunai)", Gross: 46_000_000, Fee: 0, Net: 46_000_000, Pct: 58},
			{Name: "GoFood (Komisi 20%)", Gross: 20_000_000, Fee: 2_300_000, Net: 17_700_000, Pct: 25},
			{Name: "GrabFood (Komisi 20%)", Gross: 10_000_000, Fee: 1_200_000, Net: 8_800_000, Pct: 12},
			{Name: "ShopeeFood & Catering B2B", Gross: 4_000_000, Fee: 500_000, Net: 3_500_000, Pct: 5},
		},
		RecentTxs: []TransactionItem{
			{ID: "TX-3400", Date: "30 Apr 2026", Type: "INCOME", Category: "Penjualan", Title: "Kasir POS Pasca Lebaran", Channel: "Dine-in", Gross: 2_700_000, Fee: 0, Net: 2_700_000},
		},
		Ledger: []LedgerEntry{
			{KindIncome, CatSales, "Dine-in", "Dine-in", 40_000_000, 0, true},
			{KindIncome, CatSales, "Catering", "B2B Tempo", 6_000_000, 0, false},
			{KindIncome, CatSales, "GoFood", "GoFood", 20_000_000, 2_300_000, true},
			{KindIncome, CatSales, "GrabFood", "GrabFood", 10_000_000, 1_200_000, true},
			{KindIncome, CatSales, "ShopeeFood", "ShopeeFood", 4_000_000, 500_000, true},
			{KindExpense, CatCOGS, "Bahan baku", "", 23_500_000, 0, true},
			{KindExpense, "OPEX", "Gaji staf", "", 12_500_000, 0, true},
			{KindExpense, "OPEX", "Sewa tempat", "", 8_000_000, 0, true},
			{KindExpense, "OPEX", "Biaya listrik & utilitas", "", 12_500_000, 0, true},
			{KindExpense, "OPEX", "Tagihan vendor", "", 1_500_000, 0, false},
		},
	},
	"2026-03": {
		Label: "Maret 2026", Omzet: 82_000_000, FeeApp: 4_100_000, COGS: 24_900_000, Opex: 47_000_000,
		LabaBersih: 6_000_000, MarginBps: 731, OpeningCash: 63_000_000, ClosingCash: 53_000_000,
		Transactions: 3600, TargetOmzet: 80_000_000, UnpaidAR: 5_000_000, Overdue30d: 1_200_000, UnpaidAP: 3_500_000,
		SparkOmzet: []float64{72, 75, 77, 79, 73, 82},
		SparkLaba:  []float64{21, 19, 18, 21, 14, 6},
		SparkCash:  []float64{55, 62, 65, 68, 63, 53},
		Channels: []ChannelPerf{
			{Name: "Dine-in & Takeaway (QRIS/Tunai)", Gross: 47_000_000, Fee: 0, Net: 47_000_000, Pct: 57},
			{Name: "GoFood (Komisi 20%)", Gross: 21_000_000, Fee: 2_400_000, Net: 18_600_000, Pct: 26},
			{Name: "GrabFood (Komisi 20%)", Gross: 10_000_000, Fee: 1_200_000, Net: 8_800_000, Pct: 12},
			{Name: "ShopeeFood & Catering B2B", Gross: 4_000_000, Fee: 500_000, Net: 3_500_000, Pct: 5},
		},
		RecentTxs: []TransactionItem{
			{ID: "TX-3100", Date: "30 Mar 2026", Type: "EXPENSE", Category: "Operasional", Title: "Pencairan THR Barista & Karyawan", Channel: "BCA Bisnis", Gross: 12_500_000, Fee: 0, Net: -12_500_000},
		},
		Ledger: []LedgerEntry{
			{KindIncome, CatSales, "Dine-in", "Dine-in", 42_000_000, 0, true},
			{KindIncome, CatSales, "Catering B2B", "B2B Tempo", 5_000_000, 0, false},
			{KindIncome, CatSales, "GoFood", "GoFood", 21_000_000, 2_400_000, true},
			{KindIncome, CatSales, "GrabFood", "GrabFood", 10_000_000, 1_200_000, true},
			{KindIncome, CatSales, "ShopeeFood", "ShopeeFood", 4_000_000, 500_000, true},
			{KindExpense, CatCOGS, "Bahan baku", "", 24_900_000, 0, true},
			{KindExpense, "OPEX", "Gaji bulanan", "", 12_500_000, 0, true},
			{KindExpense, "OPEX", "Tunjangan Hari Raya (THR)", "", 12_500_000, 0, true},
			{KindExpense, "OPEX", "Sewa tempat", "", 8_000_000, 0, true},
			{KindExpense, "OPEX", "Utilitas & operasional", "", 12_000_000, 0, true},
			{KindExpense, "OPEX", "Tagihan tertunda", "", 2_000_000, 0, false},
		},
	},
	"2026-02": {
		Label: "Februari 2026", Omzet: 73_000_000, FeeApp: 3_800_000, COGS: 21_200_000, Opex: 34_000_000,
		LabaBersih: 14_000_000, MarginBps: 1918, OpeningCash: 68_000_000, ClosingCash: 63_000_000,
		Transactions: 3300, TargetOmzet: 75_000_000, UnpaidAR: 4_500_000, Overdue30d: 0, UnpaidAP: 3_000_000,
		SparkOmzet: []float64{72, 75, 77, 79, 73},
		SparkLaba:  []float64{21, 19, 18, 21, 14},
		SparkCash:  []float64{55, 62, 65, 68, 63},
		Channels: []ChannelPerf{
			{Name: "Dine-in & Takeaway (QRIS/Tunai)", Gross: 43_000_000, Fee: 0, Net: 43_000_000, Pct: 59},
			{Name: "GoFood (Komisi 20%)", Gross: 18_000_000, Fee: 2_200_000, Net: 15_800_000, Pct: 25},
			{Name: "GrabFood (Komisi 20%)", Gross: 8_500_000, Fee: 1_100_000, Net: 7_400_000, Pct: 12},
			{Name: "ShopeeFood & Catering B2B", Gross: 3_500_000, Fee: 500_000, Net: 3_000_000, Pct: 4},
		},
		RecentTxs: []TransactionItem{
			{ID: "TX-2800", Date: "28 Feb 2026", Type: "INCOME", Category: "Penjualan", Title: "Kasir POS Akhir Bulan", Channel: "Dine-in", Gross: 2_100_000, Fee: 0, Net: 2_100_000},
		},
		Ledger: []LedgerEntry{
			{KindIncome, CatSales, "Dine-in", "Dine-in", 38_500_000, 0, true},
			{KindIncome, CatSales, "Catering", "B2B Tempo", 4_500_000, 0, false},
			{KindIncome, CatSales, "GoFood", "GoFood", 18_000_000, 2_200_000, true},
			{KindIncome, CatSales, "GrabFood", "GrabFood", 8_500_000, 1_100_000, true},
			{KindIncome, CatSales, "ShopeeFood", "ShopeeFood", 3_500_000, 500_000, true},
			{KindExpense, CatCOGS, "Bahan baku", "", 21_200_000, 0, true},
			{KindExpense, "OPEX", "Gaji barista", "", 12_500_000, 0, true},
			{KindExpense, "OPEX", "Sewa tempat", "", 8_000_000, 0, true},
			{KindExpense, "OPEX", "Utilitas & operasional", "", 12_500_000, 0, true},
			{KindExpense, "OPEX", "Tagihan listrik tertunda", "", 1_000_000, 0, false},
		},
	},
	"2026-01": {
		Label: "Januari 2026", Omzet: 79_000_000, FeeApp: 4_000_000, COGS: 22_500_000, Opex: 31_500_000,
		LabaBersih: 21_000_000, MarginBps: 2658, OpeningCash: 65_000_000, ClosingCash: 68_000_000,
		Transactions: 3500, TargetOmzet: 75_000_000, UnpaidAR: 4_000_000, Overdue30d: 0, UnpaidAP: 2_500_000,
		SparkOmzet: []float64{72, 75, 77, 79},
		SparkLaba:  []float64{21, 19, 18, 21},
		SparkCash:  []float64{55, 62, 65, 68},
		Channels: []ChannelPerf{
			{Name: "Dine-in & Takeaway (QRIS/Tunai)", Gross: 47_000_000, Fee: 0, Net: 47_000_000, Pct: 60},
			{Name: "GoFood (Komisi 20%)", Gross: 19_500_000, Fee: 2_400_000, Net: 17_100_000, Pct: 24},
			{Name: "GrabFood (Komisi 20%)", Gross: 8_500_000, Fee: 1_100_000, Net: 7_400_000, Pct: 11},
			{Name: "ShopeeFood & Catering B2B", Gross: 4_000_000, Fee: 500_000, Net: 3_500_000, Pct: 5},
		},
		RecentTxs: []TransactionItem{
			{ID: "TX-2400", Date: "31 Jan 2026", Type: "INCOME", Category: "Penjualan", Title: "Kasir POS Awal Tahun", Channel: "Dine-in", Gross: 2_500_000, Fee: 0, Net: 2_500_000},
		},
		Ledger: []LedgerEntry{
			{KindIncome, CatSales, "Dine-in", "Dine-in", 43_000_000, 0, true},
			{KindIncome, CatSales, "Catering B2B", "B2B Tempo", 4_000_000, 0, false},
			{KindIncome, CatSales, "GoFood", "GoFood", 19_500_000, 2_400_000, true},
			{KindIncome, CatSales, "GrabFood", "GrabFood", 8_500_000, 1_100_000, true},
			{KindIncome, CatSales, "ShopeeFood", "ShopeeFood", 4_000_000, 500_000, true},
			{KindExpense, CatCOGS, "Bahan baku kopi & susu", "", 22_500_000, 0, true},
			{KindExpense, "OPEX", "Gaji staf", "", 12_500_000, 0, true},
			{KindExpense, "OPEX", "Sewa tempat", "", 8_000_000, 0, true},
			{KindExpense, "OPEX", "Biaya listrik & utilitas", "", 9_500_000, 0, true},
			{KindExpense, "OPEX", "Tagihan supplier", "", 1_500_000, 0, false},
		},
	},
}

// AvailablePeriods returns list of available period IDs and labels
func AvailablePeriods() []struct {
	ID    string `json:"id"`
	Label string `json:"label"`
} {
	return []struct {
		ID    string `json:"id"`
		Label string `json:"label"`
	}{
		{"2026-08", "Agustus 2026"},
		{"2026-07", "Juli 2026"},
		{"2026-06", "Juni 2026"},
		{"2026-05", "Mei 2026"},
		{"2026-04", "April 2026"},
		{"2026-03", "Maret 2026"},
		{"2026-02", "Februari 2026"},
		{"2026-01", "Januari 2026"},
	}
}

// SeedLedger: buku besar periode, diturunkan dari transaksi seed (sumber tunggal laporan).
func SeedLedger(period string) []LedgerEntry {
	if _, ok := periodsData[period]; !ok {
		period = "2026-08"
	}
	return ToLedger(ForPeriod(SeedTransactions(), period))
}

// OpeningCash: saldo kas awal periode; periode tak dikenal memakai Agustus 2026.
func OpeningCash(period string) int64 {
	if m, ok := periodsData[period]; ok {
		return m.OpeningCash
	}
	return periodsData["2026-08"].OpeningCash
}

func SeedKedaiKopiSenja(period string) DashboardSnapshot {
	m, ok := periodsData[period]
	if !ok {
		m = periodsData["2026-08"]
		period = "2026-08"
	}

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

	marginPct := m.MarginBps / 100
	achievedPct := int((m.Omzet * 100) / m.TargetOmzet)
	surplus := m.Omzet - m.TargetOmzet
	surplusDisplay := fmt.Sprintf("Lebih %s", FormatJt(surplus))
	if surplus < 0 {
		surplusDisplay = fmt.Sprintf("Kurang %s", FormatJt(-surplus))
	}

	return DashboardSnapshot{
		Tenant: TenantInfo{
			Name:              "Kedai Kopi Senja",
			Initials:          "KK",
			Category:          "FnB",
			TotalTransactions: m.Transactions,
			IsDemo:            true,
		},
		Period:      period,
		PeriodLabel: m.Label,
		KPIs: DashboardKPIs{
			Omzet: MetricCard{
				Value:     m.Omzet,
				Display:   FormatJt(m.Omzet),
				BadgeText: "▲ Stabil",
				SubText:   "vs target",
				Trend:     "up",
				Sparkline: m.SparkOmzet,
			},
			LabaBersih: MetricCard{
				Value:     m.LabaBersih,
				Display:   FormatJt(m.LabaBersih),
				BadgeText: fmt.Sprintf("%d%% Margin", marginPct),
				SubText:   "laba bersih riil",
				Trend:     "up",
				Sparkline: m.SparkLaba,
			},
			MarginLaba: MetricCard{
				Value:     int64(marginPct),
				Display:   fmt.Sprintf("%d%%", marginPct),
				BadgeText: "Sehat",
				SubText:   "target min 20%",
				Trend:     "up",
				Sparkline: []float64{26, 19, 7, 22, 26, 23, 25.3, 27},
			},
			SaldoKas: MetricCard{
				Value:     m.ClosingCash,
				Display:   FormatJt(m.ClosingCash),
				BadgeText: "Likuid",
				SubText:   "kas operasional",
				Trend:     "up",
				Sparkline: m.SparkCash,
			},
			TargetOmzet: TargetCard{
				AchievedPct:    achievedPct,
				Display:        fmt.Sprintf("%d%%", achievedPct),
				SurplusValue:   surplus,
				SurplusDisplay: surplusDisplay,
			},
			Piutang: PiutangCard{
				UnpaidTotal:          m.UnpaidAR,
				UnpaidDisplay:        FormatJt(m.UnpaidAR),
				InvoiceCount:         8,
				Overdue30dTotal:      m.Overdue30d,
				Overdue30dDisplay:    fmt.Sprintf("%s lewat 30 hari", FormatJt(m.Overdue30d)),
				PayableUnpaidTotal:   m.UnpaidAP,
				PayableUnpaidDisplay: fmt.Sprintf("Utang supplier %s", FormatJt(m.UnpaidAP)),
			},
		},
		MonthlyTrend11M: trend,
		DailyAverage:    m.Omzet / 30,
		DailyTarget:     dailyTarget,
		DailyOmzet:      dailyPoints,
		TotalProducts:   7,
		TopProducts: []ProductPerf{
			{Name: "Es Kopi Susu Senja", Qty: 1920, Omzet: (m.Omzet * 38) / 100, Pct: 38},
			{Name: "Americano Arabica Gayo", Qty: 940, Omzet: (m.Omzet * 20) / 100, Pct: 20},
			{Name: "Matcha Oat Latte", Qty: 620, Omzet: (m.Omzet * 17) / 100, Pct: 17},
			{Name: "Croissant Butter Pairing", Qty: 510, Omzet: (m.Omzet * 14) / 100, Pct: 14},
			{Name: "Kopi Literan Keluarga 1L", Qty: 148, Omzet: (m.Omzet * 11) / 100, Pct: 11},
		},
		TotalAppFees:       m.FeeApp,
		TotalAppFeesLabel:  fmt.Sprintf("Potongan aplikasi bulan ini %s", FormatJt(m.FeeApp)),
		Channels:           m.Channels,
		RecentTransactions: m.RecentTxs,
		Obligations: []ObligationItem{
			{InvoiceNo: "INV-2026-061", Counterparty: "PT Karya Digital Mandiri (Kopi Rapat Bulanan)", Kind: "RECEIVABLE", DueDate: "10 Jul 2026", DaysOverdue: 52, Amount: 5_400_000},
			{InvoiceNo: "INV-2026-068", Counterparty: "Event Organizer Senayan Fest", Kind: "RECEIVABLE", DueDate: "22 Jul 2026", DaysOverdue: 40, Amount: 4_200_000},
			{InvoiceNo: "INV-2026-089", Counterparty: "Coworking Space Kemang", Kind: "RECEIVABLE", DueDate: "05 Sep 2026", DaysOverdue: 0, Amount: 1_400_000},
			{InvoiceNo: "PO-SUP-112", Counterparty: "Roastery Biji Kopi Gayo Lestari", Kind: "PAYABLE", DueDate: "08 Sep 2026", DaysOverdue: 0, Amount: 4_800_000},
			{InvoiceNo: "PO-SUP-118", Counterparty: "Kemasan Cup & Sablon Senja", Kind: "PAYABLE", DueDate: "12 Sep 2026", DaysOverdue: 0, Amount: 2_600_000},
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

// EmptySnapshot: kerangka dashboard tanpa data contoh untuk usaha baru (hasil register).
// KPI, tren, dan transaksi terbaru diisi dari transaksi usaha itu oleh ApplyKPIs/ApplyTrend.
func EmptySnapshot(period string) DashboardSnapshot {
	seed := SeedKedaiKopiSenja(period)
	return DashboardSnapshot{
		Tenant:             TenantInfo{Name: "Usaha Baru", Initials: "UB"},
		Period:             seed.Period,
		PeriodLabel:        seed.PeriodLabel,
		CashWarnings:       []CashWarning{},
		MonthlyTrend11M:    []MonthlyTrendPoint{},
		DailyOmzet:         []DailyOmzetPoint{},
		TopProducts:        []ProductPerf{},
		Channels:           []ChannelPerf{},
		RecentTransactions: []TransactionItem{},
		Obligations:        []ObligationItem{},
		Inventory:          []InventoryItem{},
		TaxReport:          CalculateUMKMTax("OP", nil),
	}
}

// SeedAssets: register aset contoh Kedai Kopi Senja. Dipakai agar demo menunjukkan kedua
// metode PMK 72/2023 — mesin espresso saldo menurun, sisanya garis lurus, renovasi bangunan
// tidak permanen (wajib garis lurus), dan satu harta tak berwujud (amortisasi).
func SeedAssets() []Asset {
	return []Asset{
		{ID: 1, Nama: "Mesin Espresso La Marzocco Linea", JenisHarta: "mesin_makanan", Kelompok: "k1",
			Metode: MetodeSaldoMenurun, TanggalPerolehan: "2026-02-01", HargaPerolehan: 90_000_000},
		{ID: 2, Nama: "Grinder Mazzer Super Jolly (2 unit)", JenisHarta: "mesin_makanan", Kelompok: "k1",
			Metode: MetodeGarisLurus, TanggalPerolehan: "2026-02-01", HargaPerolehan: 24_000_000},
		{ID: 3, Nama: "Kulkas & Chiller Display", JenisHarta: "peralatan_restoran", Kelompok: "k1",
			Metode: MetodeGarisLurus, TanggalPerolehan: "2026-03-01", HargaPerolehan: 18_000_000},
		{ID: 4, Nama: "Furnitur & Interior Kedai", JenisHarta: "mebel_kayu", Kelompok: "k1",
			Metode: MetodeGarisLurus, TanggalPerolehan: "2026-02-01", HargaPerolehan: 60_000_000},
		{ID: 5, Nama: "Renovasi & Instalasi Listrik", JenisHarta: "bangunan_tidak_permanen", Kelompok: "bangunan_tidak_permanen",
			Metode: MetodeGarisLurus, TanggalPerolehan: "2026-02-01", HargaPerolehan: 36_000_000},
		{ID: 6, Nama: "Software POS & Lisensi 3 Tahun", JenisHarta: "software", Kelompok: "k1", TakBerwujud: true,
			Metode: MetodeGarisLurus, TanggalPerolehan: "2026-02-01", HargaPerolehan: 18_000_000},
	}
}
