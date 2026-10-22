package finance

import "cmp"

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
	// CatPPN: PPN keluaran yang dipungut dari pembeli (masuk) atau disetor ke negara (keluar).
	// Titipan untuk negara: menggerakkan kas operasi & utang pajak, bukan laba rugi.
	CatPPN = "PPN"
	// expense category lain dihitung sebagai Opex
)

// Aktivitas arus kas (PSAK 2): operasi, investasi, pendanaan.
const (
	AktivitasOperasi   = "operasi"
	AktivitasInvestasi = "investasi"
	AktivitasPendanaan = "pendanaan"
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

// aktivitasOf memetakan entri ke aktivitas arus kas; dipakai juga untuk menyaring pos
// yang menggerakkan kas tapi bukan laba rugi.
func aktivitasOf(e LedgerEntry) string {
	switch e.Category {
	case CatAsset:
		return AktivitasInvestasi
	case CatCapital, CatLoan, CatLoanPayment, CatPrive:
		return AktivitasPendanaan
	}
	return AktivitasOperasi
}

type IncomeStatement struct {
	Omzet       int64 `json:"omzet"`
	ChannelFees int64 `json:"channel_fees"`
	NetSales    int64 `json:"net_sales"`
	COGS        int64 `json:"cogs"`
	GrossProfit int64 `json:"gross_profit"`
	// Opex = beban operasional yang mengeluarkan kas (tanpa penyusutan).
	Opex int64 `json:"opex"`
	// Depreciation & Amortization = beban non-kas dari register aset (PMK 72/2023).
	Depreciation int64  `json:"depreciation"`
	Amortization int64  `json:"amortization"`
	TotalOpex    int64  `json:"total_opex"` // Opex + Depreciation + Amortization
	OtherIncome  int64  `json:"other_income"`
	NetProfit    int64  `json:"net_profit"`
	MarginBps    int    `json:"margin_bps"`
	OpexLines    []Line `json:"opex_lines"`
	// EBITDA = laba kotor − beban operasional tunai (sebelum penyusutan & amortisasi);
	// EBIT = EBITDA − penyusutan − amortisasi (laba usaha); laba bersih = EBIT + pendapatan lain.
	EBITDA int64 `json:"ebitda"`
	EBIT   int64 `json:"ebit"`
	// HPPStok: rincian HPP dari kartu stok (setelah titik nol); nil = HPP = belanja bahan.
	HPPStok *StokPeriode `json:"hpp_stok,omitempty"`
}

type Line struct {
	Label  string `json:"label"`
	Amount int64  `json:"amount"`
}

// CashFlowLine: Amount bertanda — positif menambah kas, negatif mengurangi.
type CashFlowLine struct {
	Label  string `json:"label"`
	Amount int64  `json:"amount"`
	Detail []Line `json:"detail,omitempty"`
	// NonKas menandai baris informasi yang sengaja Rp 0 karena tidak menggerakkan kas.
	NonKas bool `json:"non_kas,omitempty"`
}

type CashFlowSection struct {
	Lines []CashFlowLine `json:"lines"`
	Net   int64          `json:"net"`
}

func (s *CashFlowSection) add(label string, amount int64, detail ...Line) {
	s.Lines = append(s.Lines, CashFlowLine{Label: label, Amount: amount, Detail: detail})
	s.Net += amount
}

func (s *CashFlowSection) note(label string) {
	s.Lines = append(s.Lines, CashFlowLine{Label: label, NonKas: true})
}

// CashFlow menyajikan arus kas satu periode dalam dua metode. Bagian Operasi berbeda
// (langsung menelusuri kas, tidak langsung mulai dari laba bersih) tapi Net-nya wajib sama;
// Investasi & Pendanaan identik di kedua metode.
type CashFlow struct {
	OpeningBalance int64 `json:"opening_balance"`
	CashIn         int64 `json:"cash_in"`
	CashOut        int64 `json:"cash_out"`
	NetCash        int64 `json:"net_cash"`
	ClosingBalance int64 `json:"closing_balance"`
	// NewReceivables & NewPayables hanya mencakup akrual operasional, supaya penyesuaian
	// modal kerja di metode tidak langsung tepat sama dengan kas operasi.
	NewReceivables int64 `json:"new_receivables"`
	NewPayables    int64 `json:"new_payables"`
	// NewPayablesOther & NewReceivablesOther: akrual dari aktivitas investasi & pendanaan,
	// mis. aset dibeli secara kredit. Tetap jadi utang/piutang di neraca, tapi bukan
	// penyesuaian modal kerja di metode tidak langsung.
	NewPayablesOther    int64 `json:"new_payables_other"`
	NewReceivablesOther int64 `json:"new_receivables_other"`
	Depreciation        int64 `json:"depreciation"`
	Amortization        int64 `json:"amortization"`
	// SetorModal & Prive dipakai neraca untuk menghitung tambahan modal.
	SetorModal int64 `json:"setor_modal"`
	Prive      int64 `json:"prive"`
	// KenaikanPersediaan: nilai persediaan akhir − awal dari kartu stok (0 sebelum titik nol).
	KenaikanPersediaan int64 `json:"kenaikan_persediaan"`
	// KenaikanUtangPPN: PPN keluaran dipungut − disetor (akrual) periode ini.
	KenaikanUtangPPN int64 `json:"kenaikan_utang_ppn,omitempty"`

	OperasiLangsung      CashFlowSection `json:"operasi_langsung"`
	OperasiTidakLangsung CashFlowSection `json:"operasi_tidak_langsung"`
	Investasi            CashFlowSection `json:"investasi"`
	Pendanaan            CashFlowSection `json:"pendanaan"`
}

type BalanceSheetAssets struct {
	Cash        int64 `json:"cash"`
	Receivables int64 `json:"receivables"`
	Inventory   int64 `json:"inventory"`
	// FixedAssetsCost & AccumDepreciation berasal dari register aset; FixedAssets = nilai buku.
	FixedAssetsCost   int64 `json:"fixed_assets_cost"`
	AccumDepreciation int64 `json:"accum_depreciation"`
	FixedAssets       int64 `json:"fixed_assets"`
	// OtherAssets: aset lancar lain dari saldo awal (uang muka, biaya dibayar di muka, dll.).
	OtherAssets int64 `json:"other_assets,omitempty"`
	Total       int64 `json:"total"`
}

type BalanceSheetLiabilities struct {
	Payables int64 `json:"payables"`
	Loans    int64 `json:"loans"`
	// Other: kewajiban lain dari saldo awal (utang pajak, beban yang masih harus dibayar, dll.).
	Other int64 `json:"other,omitempty"`
	Total int64 `json:"total"`
}

type BalanceSheetEquity struct {
	OpeningCapital int64 `json:"opening_capital"`
	AddedCapital   int64 `json:"added_capital"` // setor modal − ambil pribadi + aset masuk tanpa kas
	// RetainedEarnings: laba (rugi) bulan-bulan sebelumnya sejak saldo awal — hasil penutupan
	// buku tiap akhir bulan. 0 pada laporan per periode lama (BuildPeriodReport).
	RetainedEarnings int64 `json:"retained_earnings"`
	CurrentProfit    int64 `json:"current_profit"`
	Total            int64 `json:"total"`
}

type BalanceSheet struct {
	Assets      BalanceSheetAssets      `json:"assets"`
	Liabilities BalanceSheetLiabilities `json:"liabilities"`
	Equity      BalanceSheetEquity      `json:"equity"`
	IsBalanced  bool                    `json:"is_balanced"`
	// Tanggal: tanggal posisi neraca (YYYY-MM-DD); akhir bulan untuk laporan per periode.
	Tanggal string `json:"tanggal,omitempty"`
}

func BuildIncomeStatement(entries []LedgerEntry, depreciation, amortization int64) IncomeStatement {
	is := IncomeStatement{Depreciation: depreciation, Amortization: amortization}
	opexIdx := map[string]int{}
	for _, e := range entries {
		if aktivitasOf(e) != AktivitasOperasi || e.Category == CatPPN {
			continue // modal, pinjaman, aset, cicilan, prive, PPN titipan bukan laba rugi
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
	is.turunan()
	return is
}

// turunan menghitung ulang angka turunan setelah komponen laba rugi berubah.
func (is *IncomeStatement) turunan() {
	is.NetSales = is.Omzet - is.ChannelFees
	is.GrossProfit = is.NetSales - is.COGS
	is.TotalOpex = is.Opex + is.Depreciation + is.Amortization
	is.EBITDA = is.GrossProfit - is.Opex
	is.EBIT = is.EBITDA - is.Depreciation - is.Amortization
	is.NetProfit = is.EBIT + is.OtherIncome
	is.MarginBps = 0
	if is.Omzet > 0 {
		is.MarginBps = int(is.NetProfit * 10000 / is.Omzet)
	}
}

// catatanPersediaan: keterangan arus kas tidak langsung sebelum kartu stok dimulai.
const catatanPersediaan = "Perubahan persediaan stok — belum dihitung (mulai kartu stok di Stok & Gudang)"

// BuildCashFlow menyusun arus kas dari entri yang sudah lunas; is dipakai untuk metode
// tidak langsung (laba bersih dan beban non-kas).
func BuildCashFlow(entries []LedgerEntry, is IncomeStatement, opening int64) CashFlow {
	cf := CashFlow{OpeningBalance: opening, Depreciation: is.Depreciation, Amortization: is.Amortization}

	var pelanggan, lainnya, potongan, hpp, opex, beliAset, modal, pinjaman, cicilan, prive, ppnMasuk, ppnSetor int64
	channel := newTally()
	opexDetail := newTally()

	for _, e := range entries {
		kas := e.Gross
		if e.Kind == KindIncome {
			kas = e.Gross - e.Fee
		}
		if e.Category == CatPPN { // utang PPN berubah saat dipungut / disetor (akrual)
			if e.Kind == KindIncome {
				cf.KenaikanUtangPPN += e.Gross
			} else {
				cf.KenaikanUtangPPN -= e.Gross
			}
		}
		if !e.Settled {
			// Belum menyentuh kas. Hanya akrual operasional yang jadi penyesuaian modal kerja;
			// aset yang dibeli kredit tetap jadi utang di neraca tapi bukan kas operasi.
			operasi := aktivitasOf(e) == AktivitasOperasi
			switch {
			case e.Kind == KindIncome && operasi:
				cf.NewReceivables += kas
			case e.Kind == KindIncome:
				cf.NewReceivablesOther += kas
			case operasi:
				cf.NewPayables += kas
			default:
				cf.NewPayablesOther += kas
			}
			continue
		}
		if e.Kind == KindIncome {
			cf.CashIn += kas
		} else {
			cf.CashOut += kas
		}
		switch {
		case e.Category == CatPPN && e.Kind == KindIncome:
			ppnMasuk += kas
		case e.Category == CatPPN:
			ppnSetor += kas
		case e.Category == CatSales:
			pelanggan += kas
			channel.add(cmp.Or(e.Channel, "Lainnya"), kas)
		case e.Category == CatCapital:
			modal += kas
		case e.Category == CatLoan:
			pinjaman += kas
		case e.Category == CatAsset:
			beliAset += kas
		case e.Category == CatLoanPayment:
			cicilan += kas
		case e.Category == CatPrive:
			prive += kas
		case e.Kind == KindIncome:
			lainnya += kas
		case e.Category == CatChannelFee:
			potongan += kas
		case e.Category == CatCOGS:
			hpp += kas
		default:
			opex += kas
			opexDetail.add(e.Title, kas)
		}
	}
	cf.NetCash = cf.CashIn - cf.CashOut
	cf.ClosingBalance = opening + cf.NetCash
	cf.SetorModal, cf.Prive = modal, prive

	cf.OperasiLangsung.add("Penerimaan kas dari pelanggan", pelanggan, channel.lines()...)
	cf.OperasiLangsung.add("Penerimaan pendapatan lain", lainnya)
	cf.OperasiLangsung.add("Pembayaran potongan channel / aplikasi", -potongan)
	cf.OperasiLangsung.add("Pembayaran ke supplier bahan baku (HPP)", -hpp)
	cf.OperasiLangsung.add("Pembayaran beban operasional", -opex, opexDetail.lines()...)
	if ppnMasuk != 0 || ppnSetor != 0 {
		cf.OperasiLangsung.add("PPN dipungut dari pembeli (titipan negara)", ppnMasuk)
		cf.OperasiLangsung.add("PPN disetor ke negara", -ppnSetor)
	}
	cf.OperasiLangsung.note("Penjualan tempo & biaya belum dibayar — belum menyentuh kas")
	cf.OperasiLangsung.note("Penyusutan & amortisasi — tidak mengeluarkan kas")

	cf.susunTidakLangsung(is)

	cf.Investasi.add("Beli alat / aset tetap", -beliAset)
	cf.Pendanaan.add("Setor modal", modal)
	cf.Pendanaan.add("Terima pinjaman", pinjaman)
	cf.Pendanaan.add("Bayar cicilan pinjaman", -cicilan)
	cf.Pendanaan.add("Ambil uang pribadi (prive)", -prive)
	return cf
}

// susunTidakLangsung: kas operasi metode tidak langsung dari laba bersih, beban non-kas, dan
// perubahan modal kerja (NewReceivables/NewPayables harus sudah terisi).
func (cf *CashFlow) susunTidakLangsung(is IncomeStatement) {
	cf.OperasiTidakLangsung = CashFlowSection{}
	cf.OperasiTidakLangsung.add("Laba (rugi) bersih (dari laporan laba rugi)", is.NetProfit)
	cf.OperasiTidakLangsung.add("Penyusutan aset tetap", is.Depreciation)
	cf.OperasiTidakLangsung.add("Amortisasi harta tak berwujud", is.Amortization)
	cf.OperasiTidakLangsung.add("Kenaikan piutang usaha", -cf.NewReceivables)
	cf.OperasiTidakLangsung.add("Kenaikan utang usaha", cf.NewPayables)
	if cf.KenaikanUtangPPN != 0 {
		cf.OperasiTidakLangsung.add("Kenaikan utang PPN (dipungut − disetor)", cf.KenaikanUtangPPN)
	}
	cf.OperasiTidakLangsung.note(catatanPersediaan)
}

// tally menjumlahkan per label sambil menjaga urutan kemunculan.
type tally struct {
	idx   map[string]int
	items []Line
}

func newTally() *tally { return &tally{idx: map[string]int{}} }

func (t *tally) add(label string, amount int64) {
	if i, ok := t.idx[label]; ok {
		t.items[i].Amount += amount
		return
	}
	t.idx[label] = len(t.items)
	t.items = append(t.items, Line{label, amount})
}

func (t *tally) lines() []Line { return t.items }

// BuildBalanceSheet menyusun neraca. Modal awal dihitung dari komposisi saldo awal itu sendiri
// (aktiva awal − kewajiban awal), sehingga aktiva selalu sama dengan pasiva.
//
// Persediaan = saldo awal + kenaikan dari kartu stok; sebelum titik nol kenaikannya 0, jadi
// persediaan tetap angka saldo awal.
func BuildBalanceSheet(entries []LedgerEntry, is IncomeStatement, cf CashFlow, as AssetSummary, opening ReportOpening) BalanceSheet {
	var loans, utangPPN int64
	for _, e := range entries {
		switch {
		case e.Category == CatLoan:
			loans += e.Gross
		case e.Category == CatLoanPayment:
			loans -= e.Gross
		case e.Category == CatPPN && e.Kind == KindIncome:
			utangPPN += e.Gross
		case e.Category == CatPPN:
			utangPPN -= e.Gross
		}
	}
	a := BalanceSheetAssets{
		Cash:            cf.ClosingBalance,
		Receivables:     opening.Receivables + cf.NewReceivables + cf.NewReceivablesOther,
		Inventory:       opening.Inventory + cf.KenaikanPersediaan,
		FixedAssetsCost: as.TotalHargaPerolehan, AccumDepreciation: as.TotalAkumulasi, FixedAssets: as.TotalNilaiBuku,
	}
	a.Total = a.Cash + a.Receivables + a.Inventory + a.FixedAssets
	l := BalanceSheetLiabilities{
		Payables: opening.Payables + cf.NewPayables + cf.NewPayablesOther,
		Loans:    opening.Loans + loans,
		Other:    utangPPN,
	}
	l.Total = l.Payables + l.Loans + l.Other
	q := BalanceSheetEquity{
		OpeningCapital: opening.Cash + opening.Receivables + opening.Inventory + as.NilaiBukuAwal - opening.Payables - opening.Loans,
		AddedCapital:   cf.SetorModal - cf.Prive + as.TambahanTanpaKas,
		CurrentProfit:  is.NetProfit,
	}
	q.Total = q.OpeningCapital + q.AddedCapital + q.CurrentProfit
	return BalanceSheet{Assets: a, Liabilities: l, Equity: q, IsBalanced: a.Total == l.Total+q.Total}
}

// PeriodReport adalah seluruh laporan satu periode dari satu sumber data, supaya laba rugi,
// arus kas, neraca, dan register aset tidak pernah saling bertentangan.
type PeriodReport struct {
	Period       string          `json:"period"`
	Income       IncomeStatement `json:"income"`
	CashFlow     CashFlow        `json:"cash_flow"`
	BalanceSheet BalanceSheet    `json:"balance_sheet"`
	Assets       AssetSummary    `json:"assets"`
}

// BuildPeriodReport menghitung semua laporan periode dari transaksi dan register aset.
// txs adalah seluruh transaksi; penyaringan per periode dilakukan di sini.
func BuildPeriodReport(txs []Transaction, assets []Asset, period string, opening ReportOpening) PeriodReport {
	entries := ToLedger(ForPeriod(txs, period))
	as := AssetsForPeriod(assets, txs, period)
	is := BuildIncomeStatement(entries, as.BebanPenyusutan, as.BebanAmortisasi)
	cf := BuildCashFlow(entries, is, opening.Cash)
	return PeriodReport{
		Period: period, Income: is, CashFlow: cf, Assets: as,
		BalanceSheet: BuildBalanceSheet(entries, is, cf, as, opening),
	}
}
