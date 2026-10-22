package finance

import "testing"

// Skenario: titik nol 10 Jan (kopi 10 kg @200rb = 2 jt). Januari: beli 10 kg 2,5 jt (lunas),
// jual 2,5 kg; Februari: beli bahan tanpa barang 300rb (tetap biaya), opname kurang 0,5 kg.
func skenarioStok() ([]StokItem, []Transaction, []GerakStok) {
	stok := []StokItem{{ID: 1, Nama: "Kopi", Satuan: "kg", HargaPokok: 200_000}}
	gerak := []GerakStok{
		{ID: 1, Tanggal: "2026-01-10", Jenis: GerakAwal, Baris: []BarisGerak{{StokID: 1, Fisik: 10_000, HargaUnit: 200_000}}},
		{ID: 2, Tanggal: "2026-02-20", Jenis: GerakOpname, Baris: []BarisGerak{{StokID: 1, Fisik: 17_000}}},
	}
	txs := []Transaction{
		{ID: 1, Tanggal: "2026-01-02", Jenis: "Beli Bahan / Stok", Nominal: 700_000, Lunas: true, StokID: 1, QtyStok: 1_000}, // sebelum titik nol: tetap biaya
		{ID: 2, Tanggal: "2026-01-12", Jenis: "Beli Bahan / Stok", Nominal: 2_500_000, Lunas: true, StokID: 1, QtyStok: 10_000},
		{ID: 3, Tanggal: "2026-01-15", Jenis: "Penjualan", Nominal: 3_000_000, Lunas: true, StokID: 1, QtyStok: 2_500},
		{ID: 4, Tanggal: "2026-02-03", Jenis: "Beli Bahan / Stok", Nominal: 300_000, Lunas: true},
		{ID: 5, Tanggal: "2026-02-05", Jenis: "Penjualan", Nominal: 1_000_000, Lunas: true},
	}
	return stok, txs, gerak
}

func TestStokPeriode(t *testing.T) {
	stok, txs, gerak := skenarioStok()
	k := HitungKartu(stok, txs, gerak)

	if p := k.Periode("2025-12"); p.Aktif {
		t.Errorf("sebelum titik nol tidak aktif: %+v", p)
	}
	jan := k.Periode("2026-01")
	// awal = nilai titik nol; beli 2,5 jt; jual 2,5 kg @225rb = 562.500; akhir 17,5 kg = 3.937.500
	if !jan.Aktif || jan.PersediaanAwal != 2_000_000 || jan.Kapitalisasi != 2_500_000 || jan.HPPJual != 562_500 || jan.PersediaanAkhir != 3_937_500 {
		t.Errorf("januari %+v", jan)
	}
	feb := k.Periode("2026-02")
	// opname 17 kg dari 17,5 → kurang 0,5 kg × 225rb = 112.500
	if feb.PersediaanAwal != 3_937_500 || feb.SelisihOpname != -112_500 || feb.PersediaanAkhir != 3_825_000 || feb.Kapitalisasi != 0 {
		t.Errorf("februari %+v", feb)
	}
	for _, p := range []StokPeriode{jan, feb} {
		if p.PersediaanAkhir-p.PersediaanAwal != p.Kapitalisasi-p.HPPJual+p.SelisihOpname+p.Koreksi {
			t.Errorf("tidak berimbang %+v", p)
		}
	}
}

func TestLaporanDenganKartuStok(t *testing.T) {
	stok, txs, gerak := skenarioStok()
	k := HitungKartu(stok, txs, gerak)
	opening := ReportOpening{Cash: 10_000_000, Inventory: 5_000_000} // persediaan lama (angka tetap)

	lama := BuildPeriodReport(txs, nil, "2026-01", opening)
	jan := BuildPeriodReportStok(txs, nil, "2026-01", opening, k.Periode("2026-01"))
	// Lama: HPP = semua belanja bahan (700rb + 2,5 jt). Baru: 700rb (sebelum titik nol) + HPP terjual 562.500.
	if lama.Income.COGS != 3_200_000 || jan.Income.COGS != 1_262_500 {
		t.Fatalf("HPP lama %d baru %d", lama.Income.COGS, jan.Income.COGS)
	}
	if jan.Income.NetProfit != 3_000_000-1_262_500 || jan.Income.HPPStok == nil || jan.Income.HPPStok.HPPJual != 562_500 {
		t.Errorf("laba %d rincian %+v", jan.Income.NetProfit, jan.Income.HPPStok)
	}
	// Neraca: persediaan akhir dari kartu, tetap seimbang; kas sama dengan cara lama.
	if jan.BalanceSheet.Assets.Inventory != 3_937_500 || !jan.BalanceSheet.IsBalanced || jan.CashFlow.ClosingBalance != lama.CashFlow.ClosingBalance {
		t.Errorf("neraca %+v kas %d vs %d", jan.BalanceSheet, jan.CashFlow.ClosingBalance, lama.CashFlow.ClosingBalance)
	}
	// Arus kas: metode tidak langsung memuat kenaikan persediaan dan tetap sama dengan langsung.
	if jan.CashFlow.OperasiTidakLangsung.Net != jan.CashFlow.OperasiLangsung.Net || jan.CashFlow.KenaikanPersediaan != 1_937_500 {
		t.Errorf("arus kas langsung %d tidak langsung %d kenaikan %d", jan.CashFlow.OperasiLangsung.Net, jan.CashFlow.OperasiTidakLangsung.Net, jan.CashFlow.KenaikanPersediaan)
	}
	ada := false
	for _, l := range jan.CashFlow.OperasiTidakLangsung.Lines {
		if l.Label == "Kenaikan persediaan" && l.Amount == -1_937_500 {
			ada = true
		}
		if l.NonKas && l.Label == "Perubahan persediaan stok — belum dihitung per periode" {
			t.Error("catatan 'belum dihitung' masih ada")
		}
	}
	if !ada {
		t.Errorf("baris kenaikan persediaan tidak ada: %+v", jan.CashFlow.OperasiTidakLangsung.Lines)
	}

	feb := BuildPeriodReportStok(txs, nil, "2026-02", opening, k.Periode("2026-02"))
	// HPP Februari = belanja tanpa barang 300rb + kurang opname 112.500
	if feb.Income.COGS != 412_500 || !feb.BalanceSheet.IsBalanced || feb.BalanceSheet.Assets.Inventory != 3_825_000 {
		t.Errorf("februari HPP %d neraca %+v", feb.Income.COGS, feb.BalanceSheet)
	}
	if feb.CashFlow.OperasiTidakLangsung.Net != feb.CashFlow.OperasiLangsung.Net {
		t.Error("februari arus kas tidak cocok")
	}

	// Periode sebelum titik nol tidak berubah sama sekali.
	des := BuildPeriodReportStok(txs, nil, "2025-12", opening, k.Periode("2025-12"))
	if des.Income.HPPStok != nil || des.BalanceSheet.Assets.Inventory != 5_000_000 {
		t.Errorf("sebelum titik nol berubah: %+v", des.BalanceSheet.Assets)
	}
}
