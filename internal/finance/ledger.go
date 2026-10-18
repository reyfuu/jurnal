package finance

// ponytail: single-month ledger, no multi-period carry-over; add period filter when real DB lands.

const (
	KindIncome  = "INCOME"
	KindExpense = "EXPENSE"

	CatSales = "SALES"
	CatCOGS  = "COGS"
	// any other expense category counts as Opex
)

// LedgerEntry is one money movement. Settled=false means piutang (income) or utang (expense).
type LedgerEntry struct {
	Kind     string `json:"kind"`
	Category string `json:"category"`
	Title    string `json:"title"`
	Channel  string `json:"channel,omitempty"`
	Gross    int64  `json:"gross"`
	Fee      int64  `json:"fee"`
	Settled  bool   `json:"settled"`
}

type IncomeStatement struct {
	Omzet       int64  `json:"omzet"`
	ChannelFees int64  `json:"channel_fees"`
	NetSales    int64  `json:"net_sales"`
	COGS        int64  `json:"cogs"`
	GrossProfit int64  `json:"gross_profit"`
	Opex        int64  `json:"opex"`
	NetProfit   int64  `json:"net_profit"`
	MarginBps   int    `json:"margin_bps"`
	OpexLines   []Line `json:"opex_lines"`
}

type Line struct {
	Label  string `json:"label"`
	Amount int64  `json:"amount"`
}

type CashFlow struct {
	OpeningBalance int64 `json:"opening_balance"`
	CashIn         int64 `json:"cash_in"`
	CashOut        int64 `json:"cash_out"`
	NetCash        int64 `json:"net_cash"`
	ClosingBalance int64 `json:"closing_balance"`
	NewReceivables int64 `json:"new_receivables"`
	NewPayables    int64 `json:"new_payables"`
}

func BuildIncomeStatement(entries []LedgerEntry) IncomeStatement {
	var is IncomeStatement
	for _, e := range entries {
		switch {
		case e.Kind == KindIncome && e.Category == CatSales:
			is.Omzet += e.Gross
			is.ChannelFees += e.Fee
		case e.Kind == KindExpense && e.Category == CatCOGS:
			is.COGS += e.Gross
		case e.Kind == KindExpense:
			is.Opex += e.Gross
			is.OpexLines = append(is.OpexLines, Line{e.Title, e.Gross})
		}
	}
	is.NetSales = is.Omzet - is.ChannelFees
	is.GrossProfit = is.NetSales - is.COGS
	is.NetProfit = is.GrossProfit - is.Opex
	if is.Omzet > 0 {
		is.MarginBps = int(is.NetProfit * 10000 / is.Omzet)
	}
	return is
}

func BuildCashFlow(entries []LedgerEntry, opening int64) CashFlow {
	cf := CashFlow{OpeningBalance: opening}
	for _, e := range entries {
		switch e.Kind {
		case KindIncome:
			net := e.Gross - e.Fee
			if e.Settled {
				cf.CashIn += net
			} else {
				cf.NewReceivables += net
			}
		case KindExpense:
			if e.Settled {
				cf.CashOut += e.Gross
			} else {
				cf.NewPayables += e.Gross
			}
		}
	}
	cf.NetCash = cf.CashIn - cf.CashOut
	cf.ClosingBalance = opening + cf.NetCash
	return cf
}

// SeedLedger returns the Kedai Kopi Senja demo month; totals match the dashboard snapshot.
func SeedLedger(period string) []LedgerEntry {
	return []LedgerEntry{
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
	}
}
