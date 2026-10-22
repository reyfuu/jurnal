package finance

import (
	"cmp"
	"slices"
	"time"
)

// Laporan berjalan: neraca dan arus kas dari saldo awal pembukuan yang terus dibawa dari
// bulan ke bulan, seperti buku besar.
//
//   - Kas bergerak di tanggal pelunasan (TanggalLunas), bukan tanggal transaksi; penjualan
//     tempo menjadi piutang sampai dilunasi, pembelian tempo menjadi utang sampai dibayar.
//   - Piutang, utang, pinjaman, setoran modal, dan prive berjalan sejak saldo awal.
//   - Penutupan buku tiap akhir bulan: laba (rugi) bulan-bulan sebelumnya menjadi Laba Ditahan;
//     penyusutan & amortisasi sudah dibebankan tiap bulan dari register aset.
//   - Persediaan: angka saldo awal sampai titik nol kartu stok, setelah itu nilai kartu stok;
//     selisih penilaian di titik nol masuk modal (bukan laba).

// tanggalKas: tanggal uang benar-benar masuk/keluar.
func tanggalKas(t Transaction) string { return cmp.Or(t.TanggalLunas, t.Tanggal) }

func lunasPada(t Transaction, sampai string) bool { return t.Lunas && tanggalKas(t) <= sampai }

var posOperasi = map[string]bool{"omzet": true, "lain": true, "hpp": true, "potongan": true, "biaya": true, "ppn": true, "setor_ppn": true}

// Posisi: saldo akun neraca (selain persediaan & aset tetap) pada akhir suatu tanggal.
type Posisi struct {
	Kas, Piutang, Utang, Pinjaman int64
	// Bagian operasional piutang/utang (modal kerja) — selebihnya dari pendanaan/investasi.
	PiutangOps, UtangOps int64
	SetorModal, Prive    int64 // kumulatif sejak saldo awal
	UtangPPN             int64 // PPN keluaran dipungut − disetor
}

// PosisiPada menghitung saldo pada akhir tanggal sampai (YYYY-MM-DD; "YYYY-MM-00" = sebelum
// bulan itu) dari saldo awal ob dan semua transaksi.
func PosisiPada(ob OpeningBalance, txs []Transaction, sampai string) Posisi {
	ap, au := ob.sisaAwal(true, sampai), ob.sisaAwal(false, sampai)
	p := Posisi{Kas: ob.Kas + (ob.Piutang - ap) - (ob.Utang - au), Piutang: ap, Utang: au, Pinjaman: ob.Pinjaman, PiutangOps: ap, UtangOps: au}
	for _, t := range txs {
		if t.Tanggal > sampai {
			continue
		}
		j, ok := jenisOf(t.Jenis)
		if !ok || j.Pos == PosPindah { // pindah dana: total kas tetap
			continue
		}
		masuk, n := j.Arah == "masuk", t.Nominal
		switch {
		case lunasPada(t, sampai) && masuk:
			p.Kas += n
		case lunasPada(t, sampai):
			p.Kas -= n
		case masuk:
			p.Piutang += n
			if posOperasi[j.Pos] {
				p.PiutangOps += n
			}
		default:
			p.Utang += n
			if posOperasi[j.Pos] {
				p.UtangOps += n
			}
		}
		switch j.Pos {
		case "pinjaman":
			p.Pinjaman += n
		case "cicilan":
			p.Pinjaman -= n
		case "modal":
			p.SetorModal += n
		case "prive":
			p.Prive += n
		case "ppn":
			p.UtangPPN += n
		case "setor_ppn":
			p.UtangPPN -= n
		}
	}
	return p
}

// labaBulan: laba rugi satu bulan (akrual) termasuk penyusutan dan efek kartu stok.
func labaBulan(txs []Transaction, assets []Asset, period string, sp StokPeriode) (IncomeStatement, AssetSummary) {
	as := AssetsForPeriod(assets, txs, period)
	is := BuildIncomeStatement(ToLedger(ForPeriod(txs, period)), as.BebanPenyusutan, as.BebanAmortisasi)
	if sp.Aktif {
		is.COGS += -sp.Kapitalisasi + sp.HPPJual - sp.SelisihOpname - sp.Koreksi
		is.turunan()
		rincian := sp
		is.HPPStok = &rincian
	}
	return is, as
}

// BuildLaporanBerjalan: laporan satu bulan dengan saldo berjalan sejak saldo awal ob.
// Bulan sebelum bulan saldo awal tidak dilaporkan berjalan (pakai bulan saldo awal).
func BuildLaporanBerjalan(txs []Transaction, assets []Asset, period string, ob OpeningBalance, kartu KartuStok) PeriodReport {
	return buildBerjalan(txs, assets, period, period+"-31", ob, kartu)
}

// BuildNeracaPada: neraca per tanggal bebas (mis. 15 Agustus 2026). Transaksi, aset, dan kartu
// stok sesudah tanggal itu tidak dihitung; penyusutan & amortisasi bulan itu belum masuk karena
// dibukukan di akhir bulan.
func BuildNeracaPada(txs []Transaction, assets []Asset, tanggal string, ob OpeningBalance, kartu KartuStok) BalanceSheet {
	period := tanggal[:7]
	if tanggal >= akhirBulan(period) {
		return BuildLaporanBerjalan(txs, assets, period, ob, kartu).BalanceSheet
	}
	var sd []Transaction
	for _, t := range txs {
		if t.Tanggal <= tanggal {
			sd = append(sd, t)
		}
	}
	var as []Asset
	for _, a := range assets {
		if a.TanggalPerolehan <= tanggal {
			as = append(as, a)
		}
	}
	return buildBerjalan(sd, as, period, tanggal, ob, kartu).BalanceSheet
}

// buildBerjalan: laporan bulan period dengan posisi akhir pada tanggal sampai (period+"-31" =
// sebulan penuh; tanggal lebih awal = sebagian bulan, tanpa penyusutan bulan itu).
func buildBerjalan(txs []Transaction, assets []Asset, period, sampai string, ob OpeningBalance, kartu KartuStok) PeriodReport {
	mulai := ob.Period()
	if mulai == "" || period < mulai {
		mulai = period
	}
	penuh := sampai >= period+"-31" || sampai >= akhirBulan(period)
	sp := kartu.Periode(period)
	if !penuh {
		sp = kartu.Rentang(period+"-01", sampai)
	}
	is, as := labaBulan(txs, assets, period, sp)
	var susutBulanIni int64
	if !penuh { // penyusutan dibukukan di akhir bulan: belum ada pada tanggal ini
		susutBulanIni = as.BebanPenyusutan + as.BebanAmortisasi
		is.Depreciation, is.Amortization = 0, 0
		is.turunan()
	}

	p0 := PosisiPada(ob, txs, period+"-00")
	p1 := PosisiPada(ob, txs, sampai)

	// Arus kas: transaksi yang uangnya bergerak bulan ini (tanggal pelunasan).
	var kas []LedgerEntry
	for _, t := range txs {
		if d := tanggalKas(t); t.Lunas && d[:min(7, len(d))] == period {
			kas = append(kas, ToLedger([]Transaction{t})...)
		}
	}
	for _, pa := range ob.Pelunasan {
		if len(pa.Tanggal) >= 7 && pa.Tanggal[:7] == period {
			if pa.Jenis == "piutang" {
				kas = append(kas, LedgerEntry{Kind: KindIncome, Category: CatSales, Title: "Pelunasan piutang saldo awal", Channel: "Pelunasan saldo awal", Gross: pa.Nominal, Settled: true})
			} else {
				kas = append(kas, LedgerEntry{Kind: KindExpense, Category: CatCOGS, Title: "Pelunasan utang saldo awal", Gross: pa.Nominal, Settled: true})
			}
		}
	}
	cf := BuildCashFlow(kas, is, p0.Kas)
	cf.NewReceivables = p1.PiutangOps - p0.PiutangOps
	cf.KenaikanUtangPPN = p1.UtangPPN - p0.UtangPPN
	cf.NewPayables = p1.UtangOps - p0.UtangOps
	cf.NewReceivablesOther = (p1.Piutang - p1.PiutangOps) - (p0.Piutang - p0.PiutangOps)
	cf.NewPayablesOther = (p1.Utang - p1.UtangOps) - (p0.Utang - p0.UtangOps)
	cf.susunTidakLangsung(is)

	persediaanAkhir := ob.Persediaan
	var selisihTitikNol int64
	if sp.Aktif {
		persediaanAkhir = sp.PersediaanAkhir
		cf.terapkanPersediaan(sp.PersediaanAkhir - sp.PersediaanAwal)
		selisihTitikNol = kartu.Periode(kartu.Mulai[:7]).PersediaanAwal - ob.Persediaan
	}

	// Penutupan buku: laba bulan-bulan sebelumnya sejak saldo awal → Laba Ditahan; aset yang
	// masuk tanpa kas (impor data awal) → tambahan modal.
	var labaDitahan, asetTanpaKas int64
	for m := mulai; m < period; m = geserBulan(m, 1) {
		isM, asM := labaBulan(txs, assets, m, kartu.Periode(m))
		labaDitahan += isM.NetProfit
		asetTanpaKas += asM.TambahanTanpaKas
	}
	asetTanpaKas += as.TambahanTanpaKas
	nilaiBukuAwal := AssetsForPeriod(assets, txs, mulai).NilaiBukuAwal

	a := BalanceSheetAssets{
		Cash: p1.Kas, Receivables: p1.Piutang, Inventory: persediaanAkhir,
		FixedAssetsCost: as.TotalHargaPerolehan, AccumDepreciation: as.TotalAkumulasi - susutBulanIni, FixedAssets: as.TotalNilaiBuku + susutBulanIni,
		OtherAssets: ob.AsetLain,
	}
	a.Total = a.Cash + a.Receivables + a.Inventory + a.FixedAssets + a.OtherAssets
	l := BalanceSheetLiabilities{Payables: p1.Utang, Loans: p1.Pinjaman, Other: ob.KewajibanLain + p1.UtangPPN}
	l.Total = l.Payables + l.Loans + l.Other
	// Modal awal = yang dimiliki − kewajiban − laba ditahan saldo awal; untuk saldo awal yang
	// seimbang sama dengan modal disetor. Laba ditahan saldo awal digabung dengan laba
	// bulan-bulan sesudahnya.
	q := BalanceSheetEquity{
		OpeningCapital:   ob.ModalAwal(nilaiBukuAwal) - ob.LabaDitahan,
		AddedCapital:     p1.SetorModal - p1.Prive + asetTanpaKas + selisihTitikNol,
		RetainedEarnings: labaDitahan + ob.LabaDitahan,
		CurrentProfit:    is.NetProfit,
	}
	q.Total = q.OpeningCapital + q.AddedCapital + q.RetainedEarnings + q.CurrentProfit
	cf.SetorModal, cf.Prive = p1.SetorModal-p0.SetorModal, p1.Prive-p0.Prive
	return PeriodReport{
		Period: period, Income: is, CashFlow: cf, Assets: as,
		BalanceSheet: BalanceSheet{Assets: a, Liabilities: l, Equity: q, IsBalanced: a.Total == l.Total+q.Total, Tanggal: min(sampai, akhirBulan(period))},
	}
}

// --- Umur piutang / utang ---

type KelompokUmur struct {
	Label  string `json:"label"`
	Nilai  int64  `json:"nilai"`
	Jumlah int    `json:"jumlah"`
}

type BarisUmur struct {
	TransaksiID int    `json:"transaksi_id,omitempty"` // 0 = saldo awal
	Tanggal     string `json:"tanggal"`
	Pihak       string `json:"pihak"`
	Keterangan  string `json:"keterangan"`
	Jenis       string `json:"jenis"`
	Nilai       int64  `json:"nilai"`
	Hari        int    `json:"hari"`
	Kelompok    int    `json:"kelompok"`
	InvoiceID   int    `json:"invoice_id,omitempty"`
	JatuhTempo  string `json:"jatuh_tempo,omitempty"`
}

type PihakUmur struct {
	Nama  string  `json:"nama"`
	Nilai []int64 `json:"nilai"` // per kelompok
	Total int64   `json:"total"`
}

type UmurSaldoResult struct {
	Per      string         `json:"per"`
	Piutang  bool           `json:"piutang"`
	Kelompok []KelompokUmur `json:"kelompok"`
	Pihak    []PihakUmur    `json:"pihak"`
	Baris    []BarisUmur    `json:"baris"`
	Total    int64          `json:"total"`
}

var batasUmur = []struct {
	label string
	hari  int
}{{"0–30 hari", 30}, {"31–60 hari", 60}, {"61–90 hari", 90}, {"> 90 hari", 1 << 30}}

// UmurSaldo: piutang (piutang=true) atau utang yang belum lunas per tanggal, dikelompokkan
// menurut umur sejak tanggal transaksi. Totalnya sama dengan piutang/utang di neraca.
func UmurSaldo(ob OpeningBalance, txs []Transaction, per string, piutang bool) UmurSaldoResult {
	out := UmurSaldoResult{Per: per, Piutang: piutang, Baris: []BarisUmur{}, Pihak: []PihakUmur{}}
	for _, b := range batasUmur {
		out.Kelompok = append(out.Kelompok, KelompokUmur{Label: b.label})
	}
	tPer, err := time.Parse(time.DateOnly, per)
	if err != nil {
		return out
	}
	tambah := func(b BarisUmur) {
		if t, err := time.Parse(time.DateOnly, b.Tanggal); err == nil {
			b.Hari = max(0, int(tPer.Sub(t).Hours()/24))
		}
		for i, x := range batasUmur {
			if b.Hari <= x.hari {
				b.Kelompok = i
				break
			}
		}
		out.Baris = append(out.Baris, b)
		out.Kelompok[b.Kelompok].Nilai += b.Nilai
		out.Kelompok[b.Kelompok].Jumlah++
		out.Total += b.Nilai
	}
	if awal := ob.sisaAwal(piutang, per); awal > 0 && ob.Tanggal <= per {
		tambah(BarisUmur{Tanggal: ob.Tanggal, Pihak: "Saldo awal", Keterangan: "Saldo awal pembukuan", Nilai: awal})
	}
	// Satu invoice = satu baris tagihan: penjualan (DPP) dan PPN keluarannya digabung.
	var baris []BarisUmur
	perInvoice := map[int]int{}
	for _, t := range txs {
		j, ok := jenisOf(t.Jenis)
		if !ok || t.Tanggal > per || lunasPada(t, per) || (j.Arah == "masuk") != piutang {
			continue
		}
		if i, ada := perInvoice[t.InvoiceID]; ada && t.InvoiceID != 0 {
			baris[i].Nilai += t.Nominal
			continue
		}
		if t.InvoiceID != 0 {
			perInvoice[t.InvoiceID] = len(baris)
		}
		baris = append(baris, BarisUmur{TransaksiID: t.ID, Tanggal: t.Tanggal, Pihak: cmp.Or(t.Pihak, "(tanpa nama)"), Keterangan: cmp.Or(t.Keterangan, t.Item),
			Jenis: t.Jenis, Nilai: t.Nominal, InvoiceID: t.InvoiceID, JatuhTempo: t.JatuhTempo})
	}
	for _, b := range baris {
		tambah(b)
	}
	slices.SortStableFunc(out.Baris, func(a, b BarisUmur) int { return cmp.Compare(b.Hari, a.Hari) })
	idx := map[string]int{}
	for _, b := range out.Baris {
		i, ok := idx[b.Pihak]
		if !ok {
			i = len(out.Pihak)
			idx[b.Pihak] = i
			out.Pihak = append(out.Pihak, PihakUmur{Nama: b.Pihak, Nilai: make([]int64, len(batasUmur))})
		}
		out.Pihak[i].Nilai[b.Kelompok] += b.Nilai
		out.Pihak[i].Total += b.Nilai
	}
	slices.SortStableFunc(out.Pihak, func(a, b PihakUmur) int { return cmp.Compare(b.Total, a.Total) })
	return out
}
