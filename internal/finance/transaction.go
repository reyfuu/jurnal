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
	// Pindah Dana: uang pindah antar rekening usaha sendiri (Lawan = asal, Akun = tujuan). Total kas,
	// laba rugi, dan ringkasan masuk/keluar tidak berubah; hanya saldo per rekening.
	{JenisPindahDana, "keluar", PosPindah},
	// PPN keluaran (usaha PKP): dipungut dari pembeli bersama invoice, lalu disetor ke negara.
	{"PPN Keluaran", "masuk", "ppn"},
	{"Setor PPN", "keluar", "setor_ppn"},
	// PPh Final 0,5% (PP 55/2022) yang disetor sendiri; beban pajak seperti "Pajak & Izin",
	// dipisah supaya bisa direkonsiliasi per masa (lihat RekonsiliasiTahun).
	{JenisSetorPPhFinal, "keluar", "biaya"},
}

const (
	JenisPindahDana = "Pindah Dana"
	PosPindah       = "pindah"
)

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
	"aset": CatAsset, "cicilan": CatLoanPayment, "prive": CatPrive, "ppn": CatPPN, "setor_ppn": CatPPN,
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
	TanggalLunas string `json:"tanggal_lunas"`  // kosong = lunas di tanggal transaksi
	Akun         string `json:"akun,omitempty"` // kode CoA koreksi user; kosong = otomatis dari jenis
	// InvoiceID: transaksi dibuat & dijaga otomatis oleh invoice ini (penjualan DPP atau PPN
	// keluarannya); JatuhTempo dari invoice, dipakai daftar piutang.
	InvoiceID  int    `json:"invoice_id,omitempty"`
	JatuhTempo string `json:"jatuh_tempo,omitempty"`
	// Grup: transaksi ini bagian dari satu catatan jurnal multi-akun (Catat cepat); diatur server.
	// NoBukti: nomor nota / bukti catatan itu. Lawan: kode akun kas/bank persisnya (kosong =
	// otomatis dari Lunas & Channel).
	// Sumber: asal pencatatan bila bukan input manual — "mutasi" (impor mutasi bank) atau "foto"
	// (dibaca AI dari foto nota); dipakai pengecekan transaksi dobel.
	Sumber string `json:"sumber,omitempty"`
	// NoJurnal: nomor jurnal tetap JU-YYYYMM-NNNN, diberikan server (cmd/server/nomor_jurnal.go).
	NoJurnal string `json:"no_jurnal,omitempty"`
	Grup     int    `json:"grup,omitempty"`
	NoBukti  string `json:"no_bukti,omitempty"`
	Lawan    string `json:"lawan,omitempty"`
	// StokID & QtyStok: barang yang dibeli (Beli Bahan / Stok) atau dijual (Penjualan) dan
	// jumlahnya per seribu satuan; menggerakkan kartu stok setelah titik nol.
	StokID  int   `json:"stok_id,omitempty"`
	QtyStok int64 `json:"qty_stok,omitempty"`
	// Rincian: beberapa barang dalam satu transaksi (bentuk baru; menggantikan StokID/QtyStok
	// tunggal — keduanya tidak boleh terisi bersamaan). BukanBarang: penanda eksplisit bahwa
	// penjualan/pembelian ini memang bukan barang persediaan (mis. jasa); tak boleh bersama barang.
	Rincian     []RincianBarang `json:"rincian,omitempty"`
	BukanBarang bool            `json:"bukan_barang,omitempty"`
	// MasaPajak: masa (YYYY-MM) yang dilunasi setoran pajak (Setor PPh Final / Setor PPN);
	// bawaan bulan sebelum tanggal setor. Kosong untuk jenis lain.
	MasaPajak string `json:"masa_pajak,omitempty"`
}

// RincianBarang: satu baris barang dalam transaksi. QtyStok per seribu satuan (TakaranSkala);
// Nilai = bagian Nominal transaksi untuk baris ini (0 = belum dibagi).
type RincianBarang struct {
	StokID  int   `json:"stok_id"`
	QtyStok int64 `json:"qty_stok"`
	Nilai   int64 `json:"nilai,omitempty"`
}

// BarisBarang: barang transaksi ini — Rincian bila ada; data lama (StokID tunggal) dibaca sebagai
// satu baris bernilai Nominal; selain itu nil. Baris tunggal bernilai 0 memakai Nominal.
func (t Transaction) BarisBarang() []RincianBarang {
	if len(t.Rincian) > 0 {
		out := slices.Clone(t.Rincian)
		if len(out) == 1 && out[0].Nilai == 0 {
			out[0].Nilai = t.Nominal
		}
		return out
	}
	if t.StokID > 0 {
		return []RincianBarang{{StokID: t.StokID, QtyStok: t.QtyStok, Nilai: t.Nominal}}
	}
	return nil
}

// NormalkanBarang mengubah StokID/QtyStok tunggal menjadi Rincian satu baris (nilai = Nominal)
// lalu mengosongkan keduanya. Dipanggil sesudah Validate; BarisBarang() tidak berubah.
func NormalkanBarang(t *Transaction) {
	if t.StokID <= 0 || len(t.Rincian) > 0 {
		return
	}
	t.Rincian = t.BarisBarang()
	t.StokID, t.QtyStok = 0, 0
}

// Validate merapikan isian dan menolak data yang tidak bisa dibukukan.
func Validate(t *Transaction) error {
	for _, s := range []*string{&t.Tanggal, &t.Jenis, &t.Keterangan, &t.Item, &t.Channel, &t.Pihak, &t.TanggalLunas, &t.Akun, &t.NoBukti, &t.Lawan, &t.Sumber, &t.MasaPajak} {
		*s = strings.TrimSpace(*s)
	}
	if _, err := time.Parse(time.DateOnly, t.Tanggal); err != nil {
		return errors.New("tanggal wajib diisi (format YYYY-MM-DD)")
	}
	if _, ok := jenisOf(t.Jenis); !ok {
		return errors.New("jenis transaksi tidak dikenal")
	}
	if t.Jenis == JenisPindahDana && (t.Akun == "" || t.Lawan == "" || t.Akun == t.Lawan || !t.Lunas) {
		return errors.New("pindah dana dicatat lewat Catat: pilih rekening asal dan rekening tujuan yang berbeda")
	}
	if t.Nominal <= 0 {
		return errors.New("nominal harus lebih dari 0")
	}
	if err := rapikanMasa(t); err != nil {
		return err
	}
	if t.Qty < 0 {
		return errors.New("jumlah tidak boleh negatif")
	}
	if len(t.Rincian) == 0 {
		t.Rincian = nil
	}
	if len(t.Rincian) > 0 && t.StokID != 0 {
		return errors.New("pilih satu cara: barang tunggal atau rincian barang")
	}
	adaBarang := t.StokID != 0 || len(t.Rincian) > 0
	if !adaBarang {
		t.QtyStok = 0
	}
	if (adaBarang || t.BukanBarang) && t.Jenis != JenisJual && t.Jenis != JenisBeliStok {
		return errors.New("barang hanya bisa dipilih untuk Penjualan atau Beli Bahan / Stok")
	}
	if t.BukanBarang && adaBarang {
		return errors.New("transaksi bertanda bukan barang tidak boleh berisi barang")
	}
	if t.StokID != 0 && t.QtyStok <= 0 {
		return errors.New("jumlah barang harus lebih dari 0")
	}
	var jumlahNilai int64
	for _, b := range t.Rincian {
		if b.StokID <= 0 {
			return errors.New("barang pada rincian wajib dipilih")
		}
		if b.QtyStok <= 0 {
			return errors.New("jumlah barang harus lebih dari 0")
		}
		if b.Nilai < 0 {
			return errors.New("nilai barang tidak boleh negatif")
		}
		// pembelian beberapa barang: nilai tiap baris menentukan akun & kartu barangnya
		if t.Jenis == JenisBeliStok && len(t.Rincian) >= 2 && b.Nilai == 0 {
			return errors.New("isi nilai tiap barang pada pembelian beberapa barang")
		}
		jumlahNilai += b.Nilai
	}
	if jumlahNilai > t.Nominal {
		return errors.New("jumlah nilai barang melebihi nominal transaksi")
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
		if j.Pos == PosPindah {
			continue // bukan pemasukan/pengeluaran
		}
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

// FormatTanggalID: "2026-08-05" → "05 Agu 2026".
func FormatTanggalID(d string) string { return formatTanggal(d) }

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
	k.Omzet.Value, k.Omzet.Display = is.Omzet, FormatRupiahPenuh(is.Omzet)
	k.LabaBersih.Value, k.LabaBersih.Display = is.NetProfit, FormatRupiahPenuh(is.NetProfit)
	k.SaldoKas.Value, k.SaldoKas.Display = cf.ClosingBalance, FormatRupiahPenuh(cf.ClosingBalance)
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
		k.TargetOmzet.SurplusDisplay = "Lebih " + FormatRupiahPenuh(k.TargetOmzet.SurplusValue)
	} else {
		k.TargetOmzet.SurplusDisplay = "Kurang " + FormatRupiahPenuh(-k.TargetOmzet.SurplusValue)
	}

	snap.Tenant.TotalTransactions += len(periodTxs) - seedPeriodTx

	recent := slices.Clone(periodTxs)
	slices.SortFunc(recent, func(a, b Transaction) int {
		if c := strings.Compare(b.Tanggal, a.Tanggal); c != 0 {
			return c
		}
		return b.ID - a.ID
	})
	snap.RecentTransactions = []TransactionItem{}
	for _, t := range recent[:min(5, len(recent))] {
		j, _ := jenisOf(t.Jenis)
		item := TransactionItem{ID: fmt.Sprintf("TX-%d", t.ID), Date: formatTanggal(t.Tanggal), Type: KindExpense, Category: t.Jenis,
			Title: cmp.Or(t.Keterangan, t.Jenis), Channel: t.Channel, Gross: t.Nominal, Net: -t.Nominal}
		if j.Arah == "masuk" {
			item.Type, item.Net = KindIncome, t.Nominal
		}
		if j.Pos == PosPindah {
			item.Net = 0 // antar rekening sendiri: bukan pengeluaran
		}
		snap.RecentTransactions = append(snap.RecentTransactions, item)
	}
}
