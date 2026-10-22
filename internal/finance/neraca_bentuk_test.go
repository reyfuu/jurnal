package finance

import "testing"

func cekInvarianNeraca(t *testing.T, r PeriodReport) {
	t.Helper()
	bs, cf := r.BalanceSheet, r.CashFlow
	if !bs.IsBalanced {
		t.Errorf("%s neraca tidak seimbang: aset %d, kewajiban+modal %d", r.Period, bs.Assets.Total, bs.Liabilities.Total+bs.Equity.Total)
	}
	if cf.OperasiLangsung.Net != cf.OperasiTidakLangsung.Net {
		t.Errorf("%s kas operasi langsung %d ≠ tidak langsung %d", r.Period, cf.OperasiLangsung.Net, cf.OperasiTidakLangsung.Net)
	}
	if cf.ClosingBalance != bs.Assets.Cash || cf.OpeningBalance+cf.OperasiLangsung.Net+cf.Investasi.Net+cf.Pendanaan.Net != cf.ClosingBalance {
		t.Errorf("%s arus kas tidak menutup ke kas neraca", r.Period)
	}
}

// Bugslist 22 Okt #5 (CV/OP): prive tampil sebagai baris sendiri yang mengurangi modal, bukan
// dilebur ke "tambahan modal".
func TestNeraca_PriveBarisSendiri(t *testing.T) {
	ob := OpeningBalance{Tanggal: "2026-01-01", Kas: 10_000_000, ModalDisetor: 10_000_000}
	txs := []Transaction{
		{ID: 1, Tanggal: "2026-01-05", Jenis: "Setor Modal", Nominal: 2_000_000, Lunas: true},
		{ID: 2, Tanggal: "2026-01-20", Jenis: "Ambil Uang Pribadi", Nominal: 500_000, Lunas: true},
	}
	r := BuildLaporanBerjalan(txs, nil, "2026-01", ob, KartuStok{})
	if q := r.BalanceSheet.Equity; q.AddedCapital != 2_000_000 || q.Prive != 500_000 || q.Total != 11_500_000 {
		t.Fatalf("modal: %+v", q)
	}
	cekInvarianNeraca(t, r)
}

// Bugslist 22 Okt #5 (PT): dividen yang diumumkan tapi belum dibayar = Utang Dividen di neraca;
// dibayar bulan berikutnya → kas keluar, utang dividen hilang. Dividen bukan beban (laba tetap).
func TestNeraca_UtangDividenPT(t *testing.T) {
	ob := OpeningBalance{Tanggal: "2026-01-01", Kas: 10_000_000, ModalDisetor: 10_000_000}
	txs := []Transaction{
		{ID: 1, Tanggal: "2026-01-10", Jenis: "Penjualan", Nominal: 5_000_000, Lunas: true},
		{ID: 2, Tanggal: "2026-02-15", Jenis: JenisDividen, Nominal: 2_000_000, Lunas: true, TanggalLunas: "2026-03-10"},
	}
	feb := BuildLaporanBerjalan(txs, nil, "2026-02", ob, KartuStok{})
	l, q := feb.BalanceSheet.Liabilities, feb.BalanceSheet.Equity
	if l.DividendsPayable != 2_000_000 || l.Payables != 0 || q.Dividen != 2_000_000 || feb.Income.NetProfit != 0 || feb.BalanceSheet.Assets.Cash != 15_000_000 {
		t.Fatalf("feb: utang %+v modal %+v laba %d kas %d", l, q, feb.Income.NetProfit, feb.BalanceSheet.Assets.Cash)
	}
	cekInvarianNeraca(t, feb)
	mar := BuildLaporanBerjalan(txs, nil, "2026-03", ob, KartuStok{})
	if l := mar.BalanceSheet.Liabilities; l.DividendsPayable != 0 || mar.BalanceSheet.Assets.Cash != 13_000_000 || mar.BalanceSheet.Equity.Dividen != 2_000_000 {
		t.Fatalf("mar: utang %+v kas %d dividen %d", l, mar.BalanceSheet.Assets.Cash, mar.BalanceSheet.Equity.Dividen)
	}
	cekInvarianNeraca(t, mar)
}

// Struktur tampilan neraca mengikuti bentuk usaha, walau nilainya 0.
func TestBagianNeraca_PerBentukUsaha(t *testing.T) {
	ada := func(b BagianLaporan, kunci string) bool {
		for _, r := range b.Baris {
			if r.Kunci == kunci {
				return true
			}
		}
		return false
	}
	cv := BagianNeraca(BalanceSheet{})
	if !ada(cv, "prive") || ada(cv, "utang_dividen") || ada(cv, "dividen") {
		t.Errorf("CV: harus ada Prive, tanpa dividen")
	}
	pt := BagianNeraca(BalanceSheet{PT: true})
	if !ada(pt, "utang_dividen") || !ada(pt, "dividen") || ada(pt, "prive") {
		t.Errorf("PT: harus ada Utang Dividen & Dividen, tanpa Prive (bila 0)")
	}
}
