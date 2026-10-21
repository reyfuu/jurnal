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
		if got := AutoAccount(c.jenis, c.p, PetaMaster); got != c.want {
			t.Errorf("AutoAccount(%q, %s/%s) = %s, want %s", c.jenis, c.p.BentukUsaha, c.p.MetodePersediaan, got, c.want)
		}
	}
	// setiap jenis transaksi punya akun yang aktif di CoA
	for _, p := range []finance.BusinessProfile{cvPer, ptPerp} {
		active := Active(p)
		for _, j := range finance.JenisList {
			if _, ok := Find(active, AutoAccount(j.Nama, p, PetaMaster)); !ok {
				t.Errorf("%s/%s: jenis %q dipetakan ke akun tidak aktif %q", p.BentukUsaha, p.MetodePersediaan, j.Nama, AutoAccount(j.Nama, p, PetaMaster))
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
		if got := CounterAccount(c.tx, PetaMaster); got != c.want {
			t.Errorf("CounterAccount(%+v) = %s, want %s", c.tx, got, c.want)
		}
	}
}

func TestJournalAndReports_SeedAugust(t *testing.T) {
	p := profile("cv", "periodik")
	txs := finance.ForPeriod(finance.SeedTransactions(), "2026-08")
	lines := Journal(txs, p, PetaMaster)
	if len(lines) != 2*len(txs) {
		t.Fatalf("setiap transaksi 2 baris jurnal, got %d for %d tx", len(lines), len(txs))
	}
	// Register aset ikut dijurnal: laporan per akun kini satu tab dengan laba rugi & neraca,
	// jadi angkanya harus sama persis, bukan sekadar mirip.
	rep := finance.BuildPeriodReport(txs, finance.SeedAssets(), "2026-08", finance.SeedOpening("2026-08"))
	lines = append(lines, DepreciationJournal(rep.Assets, "2026-08", PetaMaster, p)...)
	opening := Opening(finance.OpeningCash("2026-08"), 8_250_000, OpeningAssets(rep.Assets, "2026-08", PetaMaster), PetaMaster)
	tb := TrialBalance(Active(p), opening, lines)
	var d, k int64
	for _, b := range tb {
		d += b.Debit
		k += b.Kredit
	}
	if d != k || d == 0 {
		t.Fatalf("neraca saldo tidak seimbang: debit %d kredit %d", d, k)
	}

	is := IncomeStatement(tb, p, PetaMaster)
	old := rep.Income
	if is.LabaBersih != old.NetProfit {
		t.Fatalf("laba bersih CoA %d != laporan lama %d", is.LabaBersih, old.NetProfit)
	}
	if is.LabaBersih != 18_450_000 {
		t.Errorf("laba bersih Agustus setelah penyusutan: want 18_450_000, got %d", is.LabaBersih)
	}
	if is.PenjualanBersih != old.Omzet {
		t.Fatalf("penjualan bersih %d != omzet %d", is.PenjualanBersih, old.Omzet)
	}

	bs := BalanceSheet(tb, is.LabaBersih)
	if !bs.Seimbang || bs.TotalAset != bs.TotalLiabilitas+bs.TotalEkuitas {
		t.Fatalf("neraca tidak seimbang: %+v", bs)
	}
	oldBS := rep.BalanceSheet
	if bs.TotalAset != oldBS.Assets.Total {
		t.Fatalf("total aset CoA %d != neraca lama %d", bs.TotalAset, oldBS.Assets.Total)
	}
}

func TestJournal_OverrideAkun(t *testing.T) {
	p := profile("cv", "periodik")
	lines := Journal([]finance.Transaction{{ID: 7, Tanggal: "2026-08-02", Jenis: "Biaya Lain", Nominal: 500, Lunas: true, Akun: "6114"}}, p, PetaMaster)
	if lines[0].Kode != "6114" || lines[0].Debit != 500 || lines[1].Kode != "1101" || lines[1].Kredit != 500 {
		t.Fatalf("override akun harus dipakai: %+v", lines)
	}
}
