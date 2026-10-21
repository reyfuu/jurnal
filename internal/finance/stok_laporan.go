package finance

// Kartu stok → laporan keuangan. Mulai titik nol, pembelian barang yang dilacak tidak lagi
// langsung jadi HPP tapi masuk persediaan; HPP = modal barang yang benar-benar terjual
// (FIFO / rata-rata bergerak) ditambah barang hilang/rusak dari opname. Persediaan di neraca = nilai
// kartu stok akhir periode, dan arus kas metode tidak langsung memuat kenaikan persediaan.
// Periode sebelum titik nol tidak berubah.

// StokPeriode: efek kartu stok pada satu bulan.
type StokPeriode struct {
	Aktif           bool   `json:"aktif"`
	Mulai           string `json:"mulai"`
	PersediaanAwal  int64  `json:"persediaan_awal"`
	PersediaanAkhir int64  `json:"persediaan_akhir"`
	// Kapitalisasi: pembelian barang (setelah titik nol) yang masuk persediaan, bukan HPP.
	Kapitalisasi  int64 `json:"kapitalisasi"`
	HPPJual       int64 `json:"hpp_jual"`
	SelisihOpname int64 `json:"selisih_opname"` // + lebih, − kurang (hilang/rusak)
	Koreksi       int64 `json:"koreksi"`        // penilaian ulang saat stok minus tertutup
}

// Periode merekap kartu stok untuk satu bulan (YYYY-MM). Bulan yang memuat titik nol memakai
// nilai titik nol sebagai persediaan awal.
func (k KartuStok) Periode(period string) StokPeriode {
	dari, sampai := period+"-01", period+"-31"
	if k.Mulai == "" || k.Mulai > sampai {
		return StokPeriode{}
	}
	p := StokPeriode{Aktif: true, Mulai: k.Mulai}
	sebelum, akhir := map[int]int64{}, map[int]int64{}
	mulaiDiSini := k.Mulai >= dari
	for _, b := range k.Baris {
		if b.Tanggal > sampai {
			break
		}
		akhir[b.StokID] = b.SaldoNilai
		if b.Tanggal < dari {
			sebelum[b.StokID] = b.SaldoNilai
			continue
		}
		switch b.Jenis {
		case KartuAwal:
			p.PersediaanAwal += b.Nilai
			continue
		case KartuBeli:
			p.Kapitalisasi += b.Nilai
		case KartuJual:
			p.HPPJual -= b.Nilai
		case KartuOpname:
			p.SelisihOpname += b.Nilai
		}
		p.Koreksi += b.Koreksi
	}
	if !mulaiDiSini {
		for _, v := range sebelum {
			p.PersediaanAwal += v
		}
	}
	for _, v := range akhir {
		p.PersediaanAkhir += v
	}
	return p
}

// BuildPeriodReportStok: BuildPeriodReport plus efek kartu stok bila sp.Aktif.
func BuildPeriodReportStok(txs []Transaction, assets []Asset, period string, opening ReportOpening, sp StokPeriode) PeriodReport {
	if !sp.Aktif {
		return BuildPeriodReport(txs, assets, period, opening)
	}
	opening.Inventory = sp.PersediaanAwal
	entries := ToLedger(ForPeriod(txs, period))
	as := AssetsForPeriod(assets, txs, period)
	is := BuildIncomeStatement(entries, as.BebanPenyusutan, as.BebanAmortisasi)
	is.COGS += -sp.Kapitalisasi + sp.HPPJual - sp.SelisihOpname - sp.Koreksi
	is.turunan()
	rincian := sp
	is.HPPStok = &rincian
	cf := BuildCashFlow(entries, is, opening.Cash)
	cf.terapkanPersediaan(sp.PersediaanAkhir - sp.PersediaanAwal)
	return PeriodReport{
		Period: period, Income: is, CashFlow: cf, Assets: as,
		BalanceSheet: BuildBalanceSheet(entries, is, cf, as, opening),
	}
}

// terapkanPersediaan: kenaikan persediaan mengurangi kas operasi metode tidak langsung
// (laba sudah naik karena pembelian tidak lagi dibebankan langsung).
func (cf *CashFlow) terapkanPersediaan(kenaikan int64) {
	cf.KenaikanPersediaan = kenaikan
	s := &cf.OperasiTidakLangsung
	lines := s.Lines[:0:0]
	for _, l := range s.Lines {
		if l.NonKas && l.Label == catatanPersediaan {
			continue
		}
		lines = append(lines, l)
	}
	s.Lines = lines
	s.add("Kenaikan persediaan stok", -kenaikan)
}
