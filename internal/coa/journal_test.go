package coa

import (
	"testing"

	"pajak/internal/finance"
)

func TestAutoAccount(t *testing.T) {
	cvPer := profile("cv", "periodik")
	ptPerp := profile("pt", "perpetual")
	for _, c := range []struct {
		jenis string
		p     finance.BusinessProfile
		want  string
	}{
		{"Penjualan", cvPer, "4101"},
		{"Sewa Tempat", cvPer, "6104"},
		{"Pajak & Izin", cvPer, "9103"},
		{"Beli Bahan / Stok", cvPer, "5102"},
		{"Beli Bahan / Stok", ptPerp, "1301"},
		{"Ambil Uang Pribadi", cvPer, "3102"},
		{"Ambil Uang Pribadi", ptPerp, "3201"},
	} {
		if got := AutoAccount(c.jenis, c.p); got != c.want {
			t.Errorf("AutoAccount(%q, %s/%s) = %s, want %s", c.jenis, c.p.BentukUsaha, c.p.MetodePersediaan, got, c.want)
		}
	}
	// setiap jenis transaksi punya akun yang aktif di CoA
	for _, p := range []finance.BusinessProfile{cvPer, ptPerp} {
		active := Active(p)
		for _, j := range finance.JenisList {
			if _, ok := Find(active, AutoAccount(j.Nama, p)); !ok {
				t.Errorf("%s/%s: jenis %q dipetakan ke akun tidak aktif %q", p.BentukUsaha, p.MetodePersediaan, j.Nama, AutoAccount(j.Nama, p))
			}
		}
	}
}

func TestCounterAccount(t *testing.T) {
	for _, c := range []struct {
		tx   finance.Transaction
		want string
	}{
		{finance.Transaction{Jenis: "Penjualan", Channel: "Dine-in", Lunas: true}, "1101"},
		{finance.Transaction{Jenis: "Penjualan", Channel: "GoFood", Lunas: true}, "1104"},
		{finance.Transaction{Jenis: "Gaji & Upah", Channel: "BCA Bisnis", Lunas: true}, "1102"},
		{finance.Transaction{Jenis: "Penjualan", Channel: "B2B Tempo", Lunas: false}, "1201"},
		{finance.Transaction{Jenis: "Beli Bahan / Stok", Lunas: false}, "2101"},
	} {
		if got := CounterAccount(c.tx); got != c.want {
			t.Errorf("CounterAccount(%+v) = %s, want %s", c.tx, got, c.want)
		}
	}
}

func TestJournalAndReports_SeedAugust(t *testing.T) {
	p := profile("cv", "periodik")
	txs := finance.ForPeriod(finance.SeedTransactions(), "2026-08")
	lines := Journal(txs, p)
	if len(lines) != 2*len(txs) {
		t.Fatalf("setiap transaksi 2 baris jurnal, got %d for %d tx", len(lines), len(txs))
	}
	opening := Opening(finance.OpeningCash("2026-08"), 8_250_000)
	tb := TrialBalance(Active(p), opening, lines)
	var d, k int64
	for _, b := range tb {
		d += b.Debit
		k += b.Kredit
	}
	if d != k || d == 0 {
		t.Fatalf("neraca saldo tidak seimbang: debit %d kredit %d", d, k)
	}

	is := IncomeStatement(tb, p)
	old := finance.BuildIncomeStatement(finance.ToLedger(txs))
	if is.LabaBersih != old.NetProfit {
		t.Fatalf("laba bersih CoA %d != laporan lama %d", is.LabaBersih, old.NetProfit)
	}
	if is.PenjualanBersih != old.Omzet {
		t.Fatalf("penjualan bersih %d != omzet %d", is.PenjualanBersih, old.Omzet)
	}

	bs := BalanceSheet(tb, is.LabaBersih)
	if !bs.Seimbang || bs.TotalAset != bs.TotalLiabilitas+bs.TotalEkuitas {
		t.Fatalf("neraca tidak seimbang: %+v", bs)
	}
	oldBS := finance.BuildBalanceSheet(finance.ToLedger(txs), finance.OpeningCash("2026-08"), 8_250_000, finance.OpeningCash("2026-08")+8_250_000)
	if bs.TotalAset != oldBS.Assets.Total {
		t.Fatalf("total aset CoA %d != neraca lama %d", bs.TotalAset, oldBS.Assets.Total)
	}
}

func TestJournal_OverrideAkun(t *testing.T) {
	p := profile("cv", "periodik")
	lines := Journal([]finance.Transaction{{ID: 7, Tanggal: "2026-08-02", Jenis: "Biaya Lain", Nominal: 500, Lunas: true, Akun: "6114"}}, p)
	if lines[0].Kode != "6114" || lines[0].Debit != 500 || lines[1].Kode != "1101" || lines[1].Kredit != 500 {
		t.Fatalf("override akun harus dipakai: %+v", lines)
	}
}
