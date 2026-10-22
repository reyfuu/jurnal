package finance

import (
	"strconv"
	"time"
)

// Laporan laba rugi & arus kas untuk rentang tanggal bebas (mis. 15 Januari – 15 Desember).
// Neraca tetap per periode (akhir bulan). Penyusutan & amortisasi dibukukan per bulan pada akhir
// bulan, jadi satu bulan ikut dihitung penuh bila tanggal akhir bulannya masuk rentang: 15–31
// Januari memuat penyusutan Januari, 15 Januari – 15 Februari juga hanya Januari. Rentang yang
// bersambung tidak menghitung penyusutan dua kali.

// LaporanRentang: laba rugi & arus kas satu rentang tanggal.
type LaporanRentang struct {
	Dari     string          `json:"dari"`
	Sampai   string          `json:"sampai"`
	Label    string          `json:"label"` // "15 Januari 2026 – 15 Desember 2026"
	Income   IncomeStatement `json:"income"`
	CashFlow CashFlow        `json:"cashflow"`
}

var namaBulanPanjang = [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

// TanggalPanjang: "2026-01-15" → "15 Januari 2026" (judul laporan & PDF).
func TanggalPanjang(d string) string {
	t, err := time.Parse(time.DateOnly, d)
	if err != nil {
		return d
	}
	return strconv.Itoa(t.Day()) + " " + namaBulanPanjang[t.Month()-1] + " " + strconv.Itoa(t.Year())
}

// LabelRentang: "2026-01-15", "2026-12-15" → "15 Januari 2026 – 15 Desember 2026".
func LabelRentang(dari, sampai string) string {
	return TanggalPanjang(dari) + " – " + TanggalPanjang(sampai)
}

func akhirBulan(period string) string {
	t, err := time.Parse("2006-01", period)
	if err != nil {
		return period + "-31"
	}
	return t.AddDate(0, 1, -1).Format(time.DateOnly)
}

func hariSebelum(d string) string {
	t, err := time.Parse(time.DateOnly, d)
	if err != nil {
		return d
	}
	return t.AddDate(0, 0, -1).Format(time.DateOnly)
}

// BuildLaporanRentang menghitung laba rugi (akrual, tanggal transaksi) dan arus kas (tanggal
// uang bergerak) untuk [dari, sampai], dengan saldo berjalan sejak saldo awal ob.
func BuildLaporanRentang(txs []Transaction, assets []Asset, dari, sampai string, ob OpeningBalance, kartu KartuStok) LaporanRentang {
	var dalam []Transaction
	for _, t := range txs {
		if t.Tanggal >= dari && t.Tanggal <= sampai {
			dalam = append(dalam, t)
		}
	}
	var susut, amortisasi int64
	for m := dari[:7]; m <= sampai[:7]; m = geserBulan(m, 1) {
		if akhir := akhirBulan(m); akhir >= dari && akhir <= sampai {
			as := AssetsForPeriod(assets, txs, m)
			susut += as.BebanPenyusutan
			amortisasi += as.BebanAmortisasi
		}
	}
	is := BuildIncomeStatement(ToLedger(dalam), susut, amortisasi)
	sp := kartu.Rentang(dari, sampai)
	if sp.Aktif {
		is.COGS += -sp.Kapitalisasi + sp.HPPJual - sp.SelisihOpname - sp.Koreksi
		is.turunan()
		rincian := sp
		is.HPPStok = &rincian
	}

	p0 := PosisiPada(ob, txs, hariSebelum(dari))
	p1 := PosisiPada(ob, txs, sampai)
	var kas []LedgerEntry
	for _, t := range txs {
		if d := tanggalKas(t); t.Lunas && d >= dari && d <= sampai {
			kas = append(kas, ToLedger([]Transaction{t})...)
		}
	}
	for _, pa := range ob.Pelunasan {
		if pa.Tanggal < dari || pa.Tanggal > sampai {
			continue
		}
		if pa.Jenis == "piutang" {
			kas = append(kas, LedgerEntry{Kind: KindIncome, Category: CatSales, Title: "Pelunasan piutang saldo awal", Channel: "Pelunasan saldo awal", Gross: pa.Nominal, Settled: true})
		} else {
			kas = append(kas, LedgerEntry{Kind: KindExpense, Category: CatCOGS, Title: "Pelunasan utang saldo awal", Gross: pa.Nominal, Settled: true})
		}
	}
	cf := BuildCashFlow(kas, is, p0.Kas)
	cf.NewReceivables = p1.PiutangOps - p0.PiutangOps
	cf.KenaikanUtangPPN = p1.UtangPPN - p0.UtangPPN
	cf.NewPayables = p1.UtangOps - p0.UtangOps
	cf.NewReceivablesOther = (p1.Piutang - p1.PiutangOps) - (p0.Piutang - p0.PiutangOps)
	cf.NewPayablesOther = (p1.Utang - p1.UtangOps) - (p0.Utang - p0.UtangOps)
	cf.susunTidakLangsung(is)
	if sp.Aktif {
		cf.terapkanPersediaan(sp.PersediaanAkhir - sp.PersediaanAwal)
	}
	return LaporanRentang{Dari: dari, Sampai: sampai, Label: LabelRentang(dari, sampai), Income: is, CashFlow: cf}
}

// TanggalAkhirPeriode: "2026-10" → "31 Oktober 2026", tanggal posisi neraca satu periode.
func TanggalAkhirPeriode(p string) string {
	t, err := time.Parse("2006-01", p)
	if err != nil {
		return p
	}
	return TanggalPanjang(t.AddDate(0, 1, -1).Format(time.DateOnly))
}

// tanggalAkhirPendek: "2026-10" → "31 Okt 2026" untuk judul kolom neraca perbandingan.
func tanggalAkhirPendek(p string) string {
	t, err := time.Parse("2006-01", p)
	if err != nil {
		return p
	}
	return strconv.Itoa(t.AddDate(0, 1, -1).Day()) + " " + labelPeriodePendek(p)
}
