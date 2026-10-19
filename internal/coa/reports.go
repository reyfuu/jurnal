package coa

import (
	"slices"

	"pajak/internal/finance"
)

type ReportLine struct {
	Kode   string `json:"kode"`
	Nama   string `json:"nama"`
	Jumlah int64  `json:"jumlah"` // searah kelompok: akun kontra bernilai negatif
}

// amount: saldo akun searah saldo normal kelompoknya (kontra jadi negatif).
func amount(b Balance) int64 {
	if b.Normal == b.Kelompok.Normal() {
		return b.Saldo
	}
	return -b.Saldo
}

func linesOf(tb []Balance, k Kelompok, skip ...string) ([]ReportLine, int64) {
	var out []ReportLine
	var total int64
	for _, b := range tb {
		if b.Kelompok != k || slices.Contains(skip, b.Kode) {
			continue
		}
		v := amount(b)
		out = append(out, ReportLine{b.Kode, b.Nama, v})
		total += v
	}
	return out, total
}

type IncomeStatementResult struct {
	Pendapatan       []ReportLine `json:"pendapatan"` // termasuk diskon & retur (negatif)
	PenjualanBersih  int64        `json:"penjualan_bersih"`
	Metode           string       `json:"metode"`
	PersediaanAwal   int64        `json:"persediaan_awal"`
	HPPLines         []ReportLine `json:"hpp_lines"`
	PersediaanAkhir  int64        `json:"persediaan_akhir"`
	HPP              int64        `json:"hpp"`
	LabaKotor        int64        `json:"laba_kotor"`
	BebanOperasional []ReportLine `json:"beban_operasional"`
	TotalBebanOps    int64        `json:"total_beban_operasional"`
	LabaUsaha        int64        `json:"laba_usaha"`
	PendapatanLain   []ReportLine `json:"pendapatan_lain"`
	BebanLain        []ReportLine `json:"beban_lain"`
	LabaBersih       int64        `json:"laba_bersih"`
	Catatan          []string     `json:"catatan"`
}

// IncomeStatement: laba rugi format perdagangan dari neraca saldo.
func IncomeStatement(tb []Balance, p finance.BusinessProfile) IncomeStatementResult {
	r := IncomeStatementResult{Metode: "periodik"}
	r.Pendapatan, r.PenjualanBersih = linesOf(tb, PendapatanUsaha)
	var hppLines int64
	r.HPPLines, hppLines = linesOf(tb, HPP)
	if perpetual(p) {
		r.Metode = "perpetual"
		r.HPP = hppLines
		r.Catatan = append(r.Catatan, "Metode perpetual: pembelian tercatat ke Persediaan (1301). HPP per penjualan akan terisi setelah modul kartu stok tersedia.")
	} else {
		// ponytail: persediaan akhir = awal sampai stok opname/kartu stok dicatat di aplikasi.
		for _, b := range tb {
			if b.Kode == "1301" {
				r.PersediaanAwal, r.PersediaanAkhir = b.Saldo, b.Saldo
			}
		}
		r.HPP = r.PersediaanAwal + hppLines - r.PersediaanAkhir
		r.Catatan = append(r.Catatan, "Metode periodik: persediaan akhir sementara disamakan dengan persediaan awal sampai stok opname dicatat.")
	}
	r.LabaKotor = r.PenjualanBersih - r.HPP
	r.BebanOperasional, r.TotalBebanOps = linesOf(tb, BebanOperasional)
	r.LabaUsaha = r.LabaKotor - r.TotalBebanOps
	var lain, bebanLain int64
	r.PendapatanLain, lain = linesOf(tb, PendapatanLain)
	r.BebanLain, bebanLain = linesOf(tb, BebanLain)
	r.LabaBersih = r.LabaUsaha + lain - bebanLain
	return r
}

type BalanceSection struct {
	Kelompok Kelompok     `json:"kelompok"`
	Lines    []ReportLine `json:"lines"`
	Total    int64        `json:"total"`
}

type BalanceSheetResult struct {
	Aset            []BalanceSection `json:"aset"`
	Liabilitas      []BalanceSection `json:"liabilitas"`
	Ekuitas         BalanceSection   `json:"ekuitas"`
	TotalAset       int64            `json:"total_aset"`
	TotalLiabilitas int64            `json:"total_liabilitas"`
	TotalEkuitas    int64            `json:"total_ekuitas"`
	Seimbang        bool             `json:"seimbang"`
}

// BalanceSheet: neraca dari akun 1-3; laba berjalan masuk ekuitas.
func BalanceSheet(tb []Balance, labaBerjalan int64) BalanceSheetResult {
	sec := func(k Kelompok) BalanceSection {
		lines, total := linesOf(tb, k)
		return BalanceSection{k, lines, total}
	}
	r := BalanceSheetResult{
		Aset:       []BalanceSection{sec(AsetLancar), sec(AsetTidakLancar)},
		Liabilitas: []BalanceSection{sec(LiabilitasPendek), sec(LiabilitasJangka)},
		Ekuitas:    sec(Ekuitas),
	}
	r.Ekuitas.Lines = append(r.Ekuitas.Lines, ReportLine{"", "Laba Tahun Berjalan", labaBerjalan})
	r.Ekuitas.Total += labaBerjalan
	for _, s := range r.Aset {
		r.TotalAset += s.Total
	}
	for _, s := range r.Liabilitas {
		r.TotalLiabilitas += s.Total
	}
	r.TotalEkuitas = r.Ekuitas.Total
	r.Seimbang = r.TotalAset == r.TotalLiabilitas+r.TotalEkuitas
	return r
}
