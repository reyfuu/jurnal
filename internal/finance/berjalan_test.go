package finance

import "testing"

// Skenario 3 bulan usaha dagang dengan saldo awal, penjualan & pembelian tempo yang dilunasi
// bulan berikutnya, pinjaman, prive, aset, dan kartu stok mulai Februari.
func skenarioBerjalan() (OpeningBalance, []Transaction, []Asset, []StokItem, []GerakStok) {
	ob := OpeningBalance{Tanggal: "2026-01-01", Kas: 10_000_000, Piutang: 1_000_000, Persediaan: 2_000_000, Utang: 500_000, Pinjaman: 3_000_000}
	txs := []Transaction{
		{ID: 1, Tanggal: "2026-01-05", Jenis: "Penjualan", Nominal: 4_000_000, Lunas: true},
		{ID: 2, Tanggal: "2026-01-20", Jenis: "Penjualan", Nominal: 1_500_000, Lunas: true, TanggalLunas: "2026-02-10"}, // dilunasi Februari
		{ID: 3, Tanggal: "2026-01-21", Jenis: "Penjualan", Nominal: 800_000},                                            // belum dibayar sampai akhir
		{ID: 4, Tanggal: "2026-01-25", Jenis: "Beli Bahan / Stok", Nominal: 1_200_000, Lunas: true, TanggalLunas: "2026-02-03"},
		{ID: 5, Tanggal: "2026-01-28", Jenis: "Gaji & Upah", Nominal: 900_000, Lunas: true},
		{ID: 6, Tanggal: "2026-01-29", Jenis: "Beli Alat / Aset", Nominal: 1_200_000, Lunas: true},
		{ID: 7, Tanggal: "2026-02-05", Jenis: "Terima Pinjaman", Nominal: 2_000_000, Lunas: true},
		{ID: 8, Tanggal: "2026-02-12", Jenis: "Beli Bahan / Stok", Nominal: 2_500_000, Lunas: false, StokID: 1, QtyStok: 10_000},
		{ID: 9, Tanggal: "2026-02-15", Jenis: "Penjualan", Nominal: 3_000_000, Lunas: true, StokID: 1, QtyStok: 4_000},
		{ID: 10, Tanggal: "2026-02-20", Jenis: "Ambil Uang Pribadi", Nominal: 400_000, Lunas: true},
		{ID: 11, Tanggal: "2026-02-25", Jenis: "Bayar Cicilan", Nominal: 500_000, Lunas: true},
		{ID: 12, Tanggal: "2026-03-02", Jenis: "Setor Modal", Nominal: 1_000_000, Lunas: true},
		{ID: 13, Tanggal: "2026-03-05", Jenis: "Penjualan", Nominal: 2_000_000, Lunas: true, StokID: 1, QtyStok: 2_000},
		{ID: 14, Tanggal: "2026-03-06", Jenis: "Listrik, Air, Internet", Nominal: 300_000, Lunas: false},
	}
	assets := []Asset{{ID: 1, Nama: "Etalase", Kelompok: "k1", Metode: MetodeGarisLurus, TanggalPerolehan: "2026-01-29", HargaPerolehan: 1_200_000, TransaksiID: 6}}
	stok := []StokItem{{ID: 1, Nama: "Kopi", Satuan: "kg", HargaPokok: 200_000}}
	gerak := []GerakStok{{ID: 1, Tanggal: "2026-02-01", Jenis: GerakAwal, Baris: []BarisGerak{{StokID: 1, Fisik: 5_000, HargaUnit: 200_000}}}}
	return ob, txs, assets, stok, gerak
}

func TestLaporanBerjalan_Invarian(t *testing.T) {
	ob, txs, assets, stok, gerak := skenarioBerjalan()
	k := HitungKartu(stok, txs, gerak)
	var lalu PeriodReport
	for i, p := range []string{"2026-01", "2026-02", "2026-03"} {
		r := BuildLaporanBerjalan(txs, assets, p, ob, k)
		bs, cf, is := r.BalanceSheet, r.CashFlow, r.Income
		if !bs.IsBalanced {
			t.Errorf("%s neraca tidak seimbang: aset %d, kewajiban+modal %d (%+v)", p, bs.Assets.Total, bs.Liabilities.Total+bs.Equity.Total, bs)
		}
		if cf.OperasiLangsung.Net != cf.OperasiTidakLangsung.Net {
			t.Errorf("%s kas operasi langsung %d ≠ tidak langsung %d", p, cf.OperasiLangsung.Net, cf.OperasiTidakLangsung.Net)
		}
		if cf.ClosingBalance != bs.Assets.Cash || cf.OpeningBalance+cf.OperasiLangsung.Net+cf.Investasi.Net+cf.Pendanaan.Net != cf.ClosingBalance {
			t.Errorf("%s arus kas tidak menutup ke kas neraca", p)
		}
		if is.EBITDA != is.GrossProfit-is.Opex || is.EBIT != is.EBITDA-is.Depreciation-is.Amortization || is.NetProfit != is.EBIT+is.OtherIncome {
			t.Errorf("%s susunan EBITDA salah: %+v", p, is)
		}
		if i > 0 {
			// Saldo berlanjut: kas awal = kas akhir bulan lalu; laba ditahan = laba ditahan + laba bulan lalu.
			if cf.OpeningBalance != lalu.CashFlow.ClosingBalance {
				t.Errorf("%s kas awal %d ≠ kas akhir bulan lalu %d", p, cf.OpeningBalance, lalu.CashFlow.ClosingBalance)
			}
			if bs.Equity.RetainedEarnings != lalu.BalanceSheet.Equity.RetainedEarnings+lalu.Income.NetProfit {
				t.Errorf("%s laba ditahan tidak berlanjut", p)
			}
		}
		lalu = r
	}
}

func TestLaporanBerjalan_PiutangUtangPelunasan(t *testing.T) {
	ob, txs, assets, stok, gerak := skenarioBerjalan()
	k := HitungKartu(stok, txs, gerak)
	jan := BuildLaporanBerjalan(txs, assets, "2026-01", ob, k)
	feb := BuildLaporanBerjalan(txs, assets, "2026-02", ob, k)
	mar := BuildLaporanBerjalan(txs, assets, "2026-03", ob, k)
	// Januari: piutang awal 1 jt + 1,5 jt (lunas Feb) + 0,8 jt; utang awal 0,5 jt + bahan 1,2 jt (lunas Feb).
	if jan.BalanceSheet.Assets.Receivables != 3_300_000 || jan.BalanceSheet.Liabilities.Payables != 1_700_000 {
		t.Errorf("jan piutang %d utang %d", jan.BalanceSheet.Assets.Receivables, jan.BalanceSheet.Liabilities.Payables)
	}
	// Februari: pelunasan masuk kas Februari; piutang tinggal 1 jt + 0,8 jt; utang 0,5 jt + bahan stok tempo 2,5 jt.
	if feb.BalanceSheet.Assets.Receivables != 1_800_000 || feb.BalanceSheet.Liabilities.Payables != 3_000_000 {
		t.Errorf("feb piutang %d utang %d", feb.BalanceSheet.Assets.Receivables, feb.BalanceSheet.Liabilities.Payables)
	}
	var terimaPelanggan int64
	for _, l := range feb.CashFlow.OperasiLangsung.Lines {
		if l.Label == "Penerimaan kas dari pelanggan" {
			terimaPelanggan = l.Amount
		}
	}
	if terimaPelanggan != 1_500_000+3_000_000 {
		t.Errorf("penerimaan Februari %d, mau pelunasan 1,5 jt + jual tunai 3 jt", terimaPelanggan)
	}
	// Pinjaman berjalan: 3 jt + 2 jt − 0,5 jt; Maret tetap.
	if feb.BalanceSheet.Liabilities.Loans != 4_500_000 || mar.BalanceSheet.Liabilities.Loans != 4_500_000 {
		t.Errorf("pinjaman feb %d mar %d", feb.BalanceSheet.Liabilities.Loans, mar.BalanceSheet.Liabilities.Loans)
	}
	// Maret: listrik tempo menambah utang 0,3 jt.
	if mar.BalanceSheet.Liabilities.Payables != 3_300_000 {
		t.Errorf("mar utang %d", mar.BalanceSheet.Liabilities.Payables)
	}
}

func TestUmurPiutang(t *testing.T) {
	ob, txs, _, _, _ := skenarioBerjalan()
	txs[2].Pihak = "Toko Maju"
	u := UmurSaldo(ob, txs, "2026-03-31", true)
	// Belum lunas per 31 Mar: saldo awal 1 jt (sejak 1 Jan, 89 hari) + 0,8 jt dari 21 Jan (69 hari) → 61–90 hari.
	if u.Total != 1_800_000 || u.Kelompok[2].Nilai != 1_800_000 || u.Kelompok[3].Nilai != 0 {
		t.Errorf("umur piutang %+v", u)
	}
	if len(u.Pihak) == 0 || u.Pihak[0].Nama == "" {
		t.Errorf("rincian per pihak %+v", u.Pihak)
	}
	utang := UmurSaldo(ob, txs, "2026-03-31", false)
	if utang.Total != 3_300_000 {
		t.Errorf("umur utang total %d", utang.Total)
	}
}

func TestPelunasanSaldoAwal(t *testing.T) {
	ob, txs, assets, stok, gerak := skenarioBerjalan()
	ob.Pelunasan = []PelunasanAwal{{Jenis: "piutang", Tanggal: "2026-02-08", Nominal: 600_000}, {Jenis: "utang", Tanggal: "2026-03-01", Nominal: 500_000}}
	k := HitungKartu(stok, txs, gerak)
	feb := BuildLaporanBerjalan(txs, assets, "2026-02", ob, k)
	mar := BuildLaporanBerjalan(txs, assets, "2026-03", ob, k)
	// Piutang Februari 1,8 jt − 0,6 jt pelunasan saldo awal; utang Maret 3,3 jt − 0,5 jt.
	if feb.BalanceSheet.Assets.Receivables != 1_200_000 || mar.BalanceSheet.Liabilities.Payables != 2_800_000 {
		t.Errorf("piutang feb %d utang mar %d", feb.BalanceSheet.Assets.Receivables, mar.BalanceSheet.Liabilities.Payables)
	}
	for _, r := range []PeriodReport{feb, mar} {
		if !r.BalanceSheet.IsBalanced || r.CashFlow.OperasiLangsung.Net != r.CashFlow.OperasiTidakLangsung.Net || r.CashFlow.ClosingBalance != r.BalanceSheet.Assets.Cash {
			t.Errorf("%s tidak konsisten", r.Period)
		}
	}
	u := UmurSaldo(ob, txs, "2026-03-31", true)
	sisaAwal := int64(-1)
	for _, b := range u.Baris {
		if b.TransaksiID == 0 {
			sisaAwal = b.Nilai
		}
	}
	if u.Total != 1_200_000 || sisaAwal != 400_000 {
		t.Errorf("umur setelah pelunasan saldo awal %+v", u)
	}
	if err := ob.TambahPelunasan(PelunasanAwal{Jenis: "piutang", Tanggal: "2026-03-01", Nominal: 500_000}); err == nil {
		t.Error("pelunasan melebihi sisa saldo awal (0,4 jt) harus ditolak")
	}
}

func TestTagihanDariBuku(t *testing.T) {
	ob, txs, _, _, _ := skenarioBerjalan()
	p, u := UmurSaldo(ob, txs, "2026-03-31", true), UmurSaldo(ob, txs, "2026-03-31", false)
	tg := TagihanDariBuku(p, u)
	r := SummarizeTagihan(tg, mustTgl("2026-03-31"))
	if r.TotalPiutang != p.Total || r.TotalUtang != u.Total || r.PiutangTerlambat != p.Total {
		t.Errorf("rekap %+v (piutang %d utang %d)", r, p.Total, u.Total)
	}
}

// Neraca per tanggal bebas: seimbang setiap hari, transaksi sesudah tanggal belum dihitung,
// penyusutan bulan itu belum masuk (dibukukan akhir bulan), dan akhir bulan = laporan bulanan.
func TestNeracaPada(t *testing.T) {
	ob, txs, assets, stok, gerak := skenarioBerjalan()
	k := HitungKartu(stok, txs, gerak)
	for d := 1; d <= 28; d++ {
		tgl := "2026-02-" + string(rune('0'+d/10)) + string(rune('0'+d%10))
		bs := BuildNeracaPada(txs, assets, tgl, ob, k)
		if !bs.IsBalanced || bs.Tanggal != tgl {
			t.Fatalf("%s tidak seimbang / tanggal salah: aset %d, pasiva %d, tanggal %s", tgl, bs.Assets.Total, bs.Liabilities.Total+bs.Equity.Total, bs.Tanggal)
		}
	}
	// 14 Feb: pinjaman 5 Feb masuk, penjualan 15 Feb belum; akumulasi = posisi akhir Januari
	b14 := BuildNeracaPada(txs, assets, "2026-02-14", ob, k)
	b15 := BuildNeracaPada(txs, assets, "2026-02-15", ob, k)
	if b15.Assets.Cash-b14.Assets.Cash != 3_000_000 {
		t.Errorf("penjualan tunai 15 Feb: kas 14 Feb %d, 15 Feb %d", b14.Assets.Cash, b15.Assets.Cash)
	}
	jan := BuildLaporanBerjalan(txs, assets, "2026-01", ob, k).BalanceSheet
	if b14.Assets.AccumDepreciation != jan.Assets.AccumDepreciation {
		t.Errorf("akumulasi 14 Feb %d, akhir Januari %d", b14.Assets.AccumDepreciation, jan.Assets.AccumDepreciation)
	}
	if b14.Liabilities.Loans != 3_000_000+2_000_000 {
		t.Errorf("pinjaman 14 Feb %d", b14.Liabilities.Loans)
	}
	feb := BuildLaporanBerjalan(txs, assets, "2026-02", ob, k).BalanceSheet
	if akhir := BuildNeracaPada(txs, assets, "2026-02-28", ob, k); akhir != feb {
		t.Errorf("akhir bulan harus sama dengan laporan bulanan:\n%+v\n%+v", akhir, feb)
	}
}
