package finance

// ponytail: single-month ledger, no multi-period carry-over; add period filter when real DB lands.

const (
	KindIncome  = "INCOME"
	KindExpense = "EXPENSE"

	CatSales       = "SALES"
	CatOtherIncome = "OTHER_INCOME"
	CatCapital     = "CAPITAL" // setor modal (bukan laba)
	CatLoan        = "LOAN"    // terima pinjaman (jadi utang)
	CatCOGS        = "COGS"
	CatChannelFee  = "CHANNEL_FEE"
	CatOpex        = "OPEX"
	CatAsset       = "ASSET"        // beli alat/aset (bukan biaya langsung)
	CatLoanPayment = "LOAN_PAYMENT" // bayar cicilan (mengurangi utang)
	CatPrive       = "PRIVE"        // ambil uang pribadi (mengurangi modal)
	// expense category lain dihitung sebagai Opex
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

// nonOperating: pos yang menggerakkan kas tapi bukan laba rugi. Nilai = efek bertanda ke kas.
func nonOperating(e LedgerEntry) (int64, bool) {
	switch e.Category {
	case CatCapital, CatLoan:
		return e.Gross, true
	case CatAsset, CatLoanPayment, CatPrive:
		return -e.Gross, true
	}
	return 0, false
}

type IncomeStatement struct {
	Omzet       int64  `json:"omzet"`
	ChannelFees int64  `json:"channel_fees"`
	NetSales    int64  `json:"net_sales"`
	COGS        int64  `json:"cogs"`
	GrossProfit int64  `json:"gross_profit"`
	Opex        int64  `json:"opex"`
	OtherIncome int64  `json:"other_income"`
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
	// NonOperating = modal & pinjaman masuk − aset, cicilan, prive (bertanda, lunas maupun belum).
	// Laba − piutang baru + utang baru + NonOperating = NetCash.
	NonOperating int64 `json:"non_operating"`
}

type BalanceSheetAssets struct {
	Cash        int64 `json:"cash"`
	Receivables int64 `json:"receivables"`
	Inventory   int64 `json:"inventory"`
	FixedAssets int64 `json:"fixed_assets"`
	Total       int64 `json:"total"`
}

type BalanceSheetLiabilities struct {
	Payables int64 `json:"payables"`
	Loans    int64 `json:"loans"`
	Total    int64 `json:"total"`
}

type BalanceSheetEquity struct {
	OpeningCapital int64 `json:"opening_capital"`
	AddedCapital   int64 `json:"added_capital"` // setor modal − ambil pribadi
	CurrentProfit  int64 `json:"current_profit"`
	Total          int64 `json:"total"`
}

type BalanceSheet struct {
	Assets      BalanceSheetAssets      `json:"assets"`
	Liabilities BalanceSheetLiabilities `json:"liabilities"`
	Equity      BalanceSheetEquity      `json:"equity"`
	IsBalanced  bool                    `json:"is_balanced"`
}

// ponytail: aset tetap belum disusutkan; tambahkan penyusutan bulanan bila dibutuhkan laporan bank.
func BuildBalanceSheet(entries []LedgerEntry, openingCash, inventory, openingEquity int64) BalanceSheet {
	is := BuildIncomeStatement(entries)
	cf := BuildCashFlow(entries, openingCash)
	var fixed, loans, added int64
	for _, e := range entries {
		switch e.Category {
		case CatAsset:
			fixed += e.Gross
		case CatLoan:
			loans += e.Gross
		case CatLoanPayment:
			loans -= e.Gross
		case CatCapital:
			added += e.Gross
		case CatPrive:
			added -= e.Gross
		}
	}
	a := BalanceSheetAssets{Cash: cf.ClosingBalance, Receivables: cf.NewReceivables, Inventory: inventory, FixedAssets: fixed}
	a.Total = a.Cash + a.Receivables + a.Inventory + a.FixedAssets
	l := BalanceSheetLiabilities{Payables: cf.NewPayables, Loans: loans}
	l.Total = l.Payables + l.Loans
	q := BalanceSheetEquity{OpeningCapital: openingEquity, AddedCapital: added, CurrentProfit: is.NetProfit}
	q.Total = q.OpeningCapital + q.AddedCapital + q.CurrentProfit
	return BalanceSheet{Assets: a, Liabilities: l, Equity: q, IsBalanced: a.Total == l.Total+q.Total}
}

func BuildIncomeStatement(entries []LedgerEntry) IncomeStatement {
	var is IncomeStatement
	opexIdx := map[string]int{}
	for _, e := range entries {
		if _, skip := nonOperating(e); skip {
			continue
		}
		switch {
		case e.Kind == KindIncome && e.Category == CatSales:
			is.Omzet += e.Gross
			is.ChannelFees += e.Fee
		case e.Kind == KindIncome:
			is.OtherIncome += e.Gross - e.Fee
		case e.Category == CatChannelFee:
			is.ChannelFees += e.Gross
		case e.Category == CatCOGS:
			is.COGS += e.Gross
		default:
			is.Opex += e.Gross
			if i, ok := opexIdx[e.Title]; ok {
				is.OpexLines[i].Amount += e.Gross
			} else {
				opexIdx[e.Title] = len(is.OpexLines)
				is.OpexLines = append(is.OpexLines, Line{e.Title, e.Gross})
			}
		}
	}
	is.NetSales = is.Omzet - is.ChannelFees
	is.GrossProfit = is.NetSales - is.COGS
	is.NetProfit = is.GrossProfit - is.Opex + is.OtherIncome
	if is.Omzet > 0 {
		is.MarginBps = int(is.NetProfit * 10000 / is.Omzet)
	}
	return is
}

func BuildCashFlow(entries []LedgerEntry, opening int64) CashFlow {
	cf := CashFlow{OpeningBalance: opening}
	for _, e := range entries {
		if v, ok := nonOperating(e); ok {
			cf.NonOperating += v
		}
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
