package finance

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// ponytail: transaksi in-memory; pindah ke tabel `transactions` (tenant_id, tanggal) saat Postgres masuk.

// Jenis transaksi dalam bahasa pemilik usaha; Pos menentukan ke mana ia masuk di laporan.
type Jenis struct {
	Nama string `json:"nama"`
	Arah string `json:"arah"` // "masuk" | "keluar"
	Pos  string `json:"pos"`
}

var JenisList = []Jenis{
	{"Penjualan", "masuk", "omzet"},
	{"Pendapatan Lain", "masuk", "lain"},
	{"Setor Modal", "masuk", "modal"},
	{"Terima Pinjaman", "masuk", "pinjaman"},
	{"Beli Bahan / Stok", "keluar", "hpp"},
	{"Kemasan", "keluar", "biaya"},
	{"Potongan Aplikasi", "keluar", "potongan"},
	{"Ongkir & Transport", "keluar", "biaya"},
	{"Gaji & Upah", "keluar", "biaya"},
	{"Sewa Tempat", "keluar", "biaya"},
	{"Listrik, Air, Internet", "keluar", "biaya"},
	{"Iklan & Promosi", "keluar", "biaya"},
	{"Perawatan & Perlengkapan", "keluar", "biaya"},
	{"Pajak & Izin", "keluar", "biaya"},
	{"Biaya Lain", "keluar", "biaya"},
	{"Beli Alat / Aset", "keluar", "aset"},
	{"Bayar Cicilan", "keluar", "cicilan"},
	{"Ambil Uang Pribadi", "keluar", "prive"},
}

// LookupJenis: info arah & pos untuk nama jenis transaksi.
func LookupJenis(nama string) (Jenis, bool) { return jenisOf(nama) }

func jenisOf(nama string) (Jenis, bool) {
	i := slices.IndexFunc(JenisList, func(j Jenis) bool { return j.Nama == nama })
	if i < 0 {
		return Jenis{}, false
	}
	return JenisList[i], true
}

var posCategory = map[string]string{
	"omzet": CatSales, "lain": CatOtherIncome, "modal": CatCapital, "pinjaman": CatLoan,
	"hpp": CatCOGS, "potongan": CatChannelFee, "biaya": CatOpex,
	"aset": CatAsset, "cicilan": CatLoanPayment, "prive": CatPrive,
}

type Transaction struct {
	ID           int    `json:"id"`
	Tanggal      string `json:"tanggal"` // YYYY-MM-DD
	Jenis        string `json:"jenis"`
	Nominal      int64  `json:"nominal"`
	Keterangan   string `json:"keterangan"`
	Item         string `json:"item"`
	Qty          int    `json:"qty"`
	Channel      string `json:"channel"`
	Pihak        string `json:"pihak"`
	Lunas        bool   `json:"lunas"`
	TanggalLunas string `json:"tanggal_lunas"` // kosong = lunas di tanggal transaksi
	Akun         string `json:"akun,omitempty"` // kode CoA koreksi user; kosong = otomatis dari jenis
}

// Validate merapikan isian dan menolak data yang tidak bisa dibukukan.
func Validate(t *Transaction) error {
	for _, s := range []*string{&t.Tanggal, &t.Jenis, &t.Keterangan, &t.Item, &t.Channel, &t.Pihak, &t.TanggalLunas, &t.Akun} {
		*s = strings.TrimSpace(*s)
	}
	if _, err := time.Parse(time.DateOnly, t.Tanggal); err != nil {
		return errors.New("tanggal wajib diisi (format YYYY-MM-DD)")
	}
	if _, ok := jenisOf(t.Jenis); !ok {
		return errors.New("jenis transaksi tidak dikenal")
	}
	if t.Nominal <= 0 {
		return errors.New("nominal harus lebih dari 0")
	}
	if t.Qty < 0 {
		return errors.New("jumlah tidak boleh negatif")
	}
	if t.TanggalLunas != "" {
		if !t.Lunas {
			return errors.New("tanggal lunas hanya untuk transaksi yang sudah lunas")
		}
		if _, err := time.Parse(time.DateOnly, t.TanggalLunas); err != nil || t.TanggalLunas < t.Tanggal {
			return errors.New("tanggal lunas tidak boleh sebelum tanggal transaksi")
		}
	}
	return nil
}

func ForPeriod(txs []Transaction, period string) []Transaction {
	var out []Transaction
	for _, t := range txs {
		if strings.HasPrefix(t.Tanggal, period) {
			out = append(out, t)
		}
	}
	return out
}

// ToLedger menurunkan entri buku besar dari transaksi; laporan dan AI tetap memakai LedgerEntry.
func ToLedger(txs []Transaction) []LedgerEntry {
	out := make([]LedgerEntry, 0, len(txs))
	for _, t := range txs {
		j, _ := jenisOf(t.Jenis)
		kind := KindExpense
		if j.Arah == "masuk" {
			kind = KindIncome
		}
		title := t.Keterangan
		if j.Pos == "biaya" || title == "" {
			title = j.Nama // baris beban dikelompokkan per jenis di laba rugi
		}
		out = append(out, LedgerEntry{Kind: kind, Category: posCategory[j.Pos], Title: title, Channel: t.Channel, Gross: t.Nominal, Settled: t.Lunas})
	}
	return out
}

// SeedTransactions mengurai ringkasan ledger tiap periode seed (seed.go) menjadi transaksi bertanggal,
// sehingga total laporan per bulan sama persis dengan angka dashboard seed.
// seedMenu: komposisi penjualan Kedai Kopi Senja per item (bobot %, harga per pcs).
var seedMenu = []struct {
	Item  string
	Pct   int64
	Harga int64
}{
	{"Kopi Susu Gula Aren", 32, 18_000},
	{"Americano", 18, 15_000},
	{"Cappuccino", 16, 22_000},
	{"Croissant Butter", 14, 20_000},
	{"Matcha Latte", 12, 24_000},
	{"Roti Bakar Cokelat", 8, 17_000},
}

var seedB2B = []string{"Hotel Santika Bandung", "PT Kreatif Digital", "Kantor Notaris Wijaya"}

// seedSalesLines memecah satu baris omzet seed menjadi baris per item menu; total nominal tetap sama.
func seedSalesLines(e LedgerEntry, n int) []Transaction {
	pihak := ""
	if e.Channel == "B2B Tempo" {
		pihak = seedB2B[n%len(seedB2B)]
	}
	var out []Transaction
	sisa := e.Gross
	for i, m := range seedMenu {
		nominal := e.Gross * m.Pct / 100
		if i == len(seedMenu)-1 {
			nominal = sisa
		}
		sisa -= nominal
		out = append(out, Transaction{
			Jenis: "Penjualan", Nominal: nominal, Keterangan: e.Title, Item: m.Item, Qty: int(nominal / m.Harga),
			Channel: e.Channel, Pihak: pihak, Lunas: e.Settled,
		})
	}
	return out
}

func SeedTransactions() []Transaction {
	periods := AvailablePeriods()
	slices.SortFunc(periods, func(a, b struct {
		ID    string `json:"id"`
		Label string `json:"label"`
	}) int {
		return strings.Compare(a.ID, b.ID)
	})
	var out []Transaction
	add := func(t Transaction) { t.ID = len(out) + 1; out = append(out, t) }
	for _, p := range periods {
		entries := periodsData[p.ID].Ledger
		for i, e := range entries {
			date := fmt.Sprintf("%s-%02d", p.ID, 1+i*27/max(1, len(entries)-1))
			if e.Category == CatSales {
				for _, t := range seedSalesLines(e, len(out)) {
					t.Tanggal = date
					add(t)
				}
			} else {
				add(Transaction{Tanggal: date, Jenis: seedJenis(e), Nominal: e.Gross, Keterangan: e.Title, Channel: e.Channel, Lunas: e.Settled})
			}
			if e.Fee > 0 {
				add(Transaction{Tanggal: date, Jenis: "Potongan Aplikasi", Nominal: e.Fee, Keterangan: "Komisi " + e.Channel, Channel: e.Channel, Lunas: e.Settled})
			}
		}
	}
	return out
}

func seedJenis(e LedgerEntry) string {
	switch e.Category {
	case CatSales:
		return "Penjualan"
	case CatCOGS:
		return "Beli Bahan / Stok"
	}
	title := strings.ToLower(e.Title)
	for _, m := range [][2]string{
		{"gaji", "Gaji & Upah"}, {"thr", "Gaji & Upah"}, {"sewa", "Sewa Tempat"},
		{"listrik", "Listrik, Air, Internet"}, {"utilitas", "Listrik, Air, Internet"}, {"kemasan", "Kemasan"},
	} {
		if strings.Contains(title, m[0]) {
			return m[1]
		}
	}
	return "Biaya Lain"
}

var bulanSingkat = [...]string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}

func formatTanggal(d string) string {
	t, err := time.Parse(time.DateOnly, d)
	if err != nil {
		return d
	}
	return fmt.Sprintf("%02d %s %d", t.Day(), bulanSingkat[t.Month()-1], t.Year())
}

// ApplyKPIs menghitung ulang kartu KPI dari laporan periode; badge, sparkline, dan grafik tetap seed.
// snap harus snapshot seed periode itu: target omzet dibaca dari kartunya, seedPeriodTx = jumlah transaksi seed periode.
func ApplyKPIs(snap *DashboardSnapshot, is IncomeStatement, cf CashFlow, periodTxs []Transaction, seedPeriodTx int) {
	k := &snap.KPIs
	target := k.Omzet.Value - k.TargetOmzet.SurplusValue
	k.Omzet.Value, k.Omzet.Display = is.Omzet, FormatJt(is.Omzet)
	k.LabaBersih.Value, k.LabaBersih.Display = is.NetProfit, FormatJt(is.NetProfit)
	k.SaldoKas.Value, k.SaldoKas.Display = cf.ClosingBalance, FormatJt(cf.ClosingBalance)
	k.MarginLaba.Value = 0
	if is.Omzet > 0 {
		k.MarginLaba.Value = (is.NetProfit*100 + is.Omzet/2) / is.Omzet
	}
	k.MarginLaba.Display = fmt.Sprintf("%d%%", k.MarginLaba.Value)

	if target > 0 {
		k.TargetOmzet.AchievedPct = int(is.Omzet * 100 / target)
	}
	k.TargetOmzet.Display = fmt.Sprintf("%d%%", k.TargetOmzet.AchievedPct)
	k.TargetOmzet.SurplusValue = is.Omzet - target
	if k.TargetOmzet.SurplusValue >= 0 {
		k.TargetOmzet.SurplusDisplay = "Lebih " + FormatJt(k.TargetOmzet.SurplusValue)
	} else {
		k.TargetOmzet.SurplusDisplay = "Kurang " + FormatJt(-k.TargetOmzet.SurplusValue)
	}

	snap.Tenant.TotalTransactions += len(periodTxs) - seedPeriodTx

	recent := slices.Clone(periodTxs)
	slices.SortFunc(recent, func(a, b Transaction) int {
		if c := strings.Compare(b.Tanggal, a.Tanggal); c != 0 {
			return c
		}
		return b.ID - a.ID
	})
	snap.RecentTransactions = nil
	for _, t := range recent[:min(5, len(recent))] {
		j, _ := jenisOf(t.Jenis)
		item := TransactionItem{ID: fmt.Sprintf("TX-%d", t.ID), Date: formatTanggal(t.Tanggal), Type: KindExpense, Category: t.Jenis,
			Title: cmp.Or(t.Keterangan, t.Jenis), Channel: t.Channel, Gross: t.Nominal, Net: -t.Nominal}
		if j.Arah == "masuk" {
			item.Type, item.Net = KindIncome, t.Nominal
		}
		snap.RecentTransactions = append(snap.RecentTransactions, item)
	}
}
