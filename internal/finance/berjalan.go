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

var posOperasi = map[string]bool{"omzet": true, "lain": true, "hpp": true, "potongan": true, "biaya": true}

// Posisi: saldo akun neraca (selain persediaan & aset tetap) pada akhir suatu tanggal.
type Posisi struct {
	Kas, Piutang, Utang, Pinjaman int64
	// Bagian operasional piutang/utang (modal kerja) — selebihnya dari pendanaan/investasi.
	PiutangOps, UtangOps int64
	SetorModal, Prive    int64 // kumulatif sejak saldo awal
}

// PosisiPada menghitung saldo pada akhir tanggal sampai (YYYY-MM-DD; "YYYY-MM-00" = sebelum
// bulan itu) dari saldo awal ob dan semua transaksi.
func PosisiPada(ob OpeningBalance, txs []Transaction, sampai string) Posisi {
	p := Posisi{Kas: ob.Kas, Piutang: ob.Piutang, Utang: ob.Utang, Pinjaman: ob.Pinjaman, PiutangOps: ob.Piutang, UtangOps: ob.Utang}
	for _, t := range txs {
		if t.Tanggal > sampai {
			continue
		}
		j, ok := jenisOf(t.Jenis)
		if !ok {
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
	mulai := ob.Period()
	if mulai == "" || period < mulai {
		mulai = period
	}
	sp := kartu.Periode(period)
	is, as := labaBulan(txs, assets, period, sp)

	p0 := PosisiPada(ob, txs, period+"-00")
	p1 := PosisiPada(ob, txs, period+"-31")

	// Arus kas: transaksi yang uangnya bergerak bulan ini (tanggal pelunasan).
	var kas []LedgerEntry
	for _, t := range txs {
		if d := tanggalKas(t); t.Lunas && d[:min(7, len(d))] == period {
			kas = append(kas, ToLedger([]Transaction{t})...)
		}
	}
	cf := BuildCashFlow(kas, is, p0.Kas)
	cf.NewReceivables = p1.PiutangOps - p0.PiutangOps
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
		FixedAssetsCost: as.TotalHargaPerolehan, AccumDepreciation: as.TotalAkumulasi, FixedAssets: as.TotalNilaiBuku,
	}
	a.Total = a.Cash + a.Receivables + a.Inventory + a.FixedAssets
	l := BalanceSheetLiabilities{Payables: p1.Utang, Loans: p1.Pinjaman}
	l.Total = l.Payables + l.Loans
	q := BalanceSheetEquity{
		OpeningCapital:   ob.Kas + ob.Piutang + ob.Persediaan + nilaiBukuAwal - ob.Utang - ob.Pinjaman,
		AddedCapital:     p1.SetorModal - p1.Prive + asetTanpaKas + selisihTitikNol,
		RetainedEarnings: labaDitahan,
		CurrentProfit:    is.NetProfit,
	}
	q.Total = q.OpeningCapital + q.AddedCapital + q.RetainedEarnings + q.CurrentProfit
	cf.SetorModal, cf.Prive = p1.SetorModal-p0.SetorModal, p1.Prive-p0.Prive
	return PeriodReport{
		Period: period, Income: is, CashFlow: cf, Assets: as,
		BalanceSheet: BalanceSheet{Assets: a, Liabilities: l, Equity: q, IsBalanced: a.Total == l.Total+q.Total},
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
	awal := ob.Utang
	if piutang {
		awal = ob.Piutang
	}
	if awal > 0 && ob.Tanggal <= per {
		tambah(BarisUmur{Tanggal: ob.Tanggal, Pihak: "Saldo awal", Keterangan: "Saldo awal pembukuan", Nilai: awal})
	}
	for _, t := range txs {
		j, ok := jenisOf(t.Jenis)
		if !ok || t.Tanggal > per || lunasPada(t, per) || (j.Arah == "masuk") != piutang {
			continue
		}
		tambah(BarisUmur{TransaksiID: t.ID, Tanggal: t.Tanggal, Pihak: cmp.Or(t.Pihak, "(tanpa nama)"), Keterangan: cmp.Or(t.Keterangan, t.Item), Jenis: t.Jenis, Nilai: t.Nominal})
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
