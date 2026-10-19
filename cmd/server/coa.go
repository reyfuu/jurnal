package main

import (
	"fmt"
	"net/http"
	"strconv"

	"pajak/internal/coa"
	"pajak/internal/finance"
	"pajak/internal/xlsx"
)

func (s *Server) currentProfile() finance.BusinessProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.profile == nil {
		return finance.BusinessProfile{}
	}
	return *s.profile
}

type mappingRow struct {
	Jenis string `json:"jenis"`
	Kode  string `json:"kode"`
	Nama  string `json:"nama"`
}

func (s *Server) handleCoA(w http.ResponseWriter, r *http.Request) {
	p := s.currentProfile()
	active := coa.Active(p)
	var mapping []mappingRow
	if active != nil {
		for _, j := range finance.JenisList {
			kode := coa.AutoAccount(j.Nama, p)
			acc, _ := coa.Find(active, kode)
			mapping = append(mapping, mappingRow{j.Nama, kode, acc.Nama})
		}
	}
	writeJSON(w, map[string]any{"supported": active != nil, "accounts": active, "mapping": mapping, "kelompok": coa.Kelompoks})
}

type coaReport struct {
	Supported       bool                       `json:"supported"`
	TrialBalance    []coa.Balance              `json:"trial_balance"`
	IncomeStatement *coa.IncomeStatementResult `json:"income_statement,omitempty"`
	BalanceSheet    *coa.BalanceSheetResult    `json:"balance_sheet,omitempty"`
}

func (s *Server) buildCoAReport(r *http.Request) coaReport {
	p := s.currentProfile()
	active := coa.Active(p)
	if active == nil {
		return coaReport{}
	}
	period := periodOrDefault(r)
	s.mu.RLock()
	lines := coa.Journal(finance.ForPeriod(s.txs, period), p)
	opening := s.openingFor(period) // ikut neraca awal onboarding bila sudah diisi
	s.mu.RUnlock()
	// Register aset ikut dijurnal supaya laba bersih & total aset di laporan per akun sama
	// dengan tab Laba Rugi / Neraca, yang memang satu menu.
	as := s.report(period).Assets
	lines = append(lines, coa.DepreciationJournal(as, period)...)
	tb := coa.TrialBalance(active, coa.Opening(opening.Cash, opening.Inventory, coa.OpeningAssets(as, period)), lines)
	is := coa.IncomeStatement(tb, p)
	bs := coa.BalanceSheet(tb, is.LabaBersih)
	return coaReport{true, tb, &is, &bs}
}

func (s *Server) handleReportCoA(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.buildCoAReport(r))
}

func rp(n int64) string { return strconv.FormatInt(n, 10) }

func (s *Server) handleReportCoAXlsx(w http.ResponseWriter, r *http.Request) {
	rep := s.buildCoAReport(r)
	if !rep.Supported {
		http.Error(w, "CoA untuk jenis usaha ini belum tersedia", http.StatusNotFound)
		return
	}
	is, bs := rep.IncomeStatement, rep.BalanceSheet
	lr := [][]string{{"Kode", "Akun", "Jumlah (Rp)"}}
	add := func(title string, lines []coa.ReportLine) {
		lr = append(lr, []string{"", title, ""})
		for _, l := range lines {
			lr = append(lr, []string{l.Kode, l.Nama, rp(l.Jumlah)})
		}
	}
	add("Pendapatan Usaha", is.Pendapatan)
	lr = append(lr, []string{"", "Penjualan Bersih", rp(is.PenjualanBersih)})
	if is.Metode == "periodik" {
		lr = append(lr, []string{"1301", "Persediaan Awal", rp(is.PersediaanAwal)})
	}
	add("Harga Pokok Penjualan", is.HPPLines)
	if is.Metode == "periodik" {
		lr = append(lr, []string{"1301", "Persediaan Akhir", rp(-is.PersediaanAkhir)})
	}
	lr = append(lr, []string{"", "Harga Pokok Penjualan", rp(is.HPP)}, []string{"", "Laba Kotor", rp(is.LabaKotor)})
	add("Beban Operasional", is.BebanOperasional)
	lr = append(lr, []string{"", "Laba Usaha", rp(is.LabaUsaha)})
	add("Pendapatan Lain-lain", is.PendapatanLain)
	add("Beban Lain-lain", is.BebanLain)
	lr = append(lr, []string{"", "Laba Bersih", rp(is.LabaBersih)})

	nr := [][]string{{"Kode", "Akun", "Jumlah (Rp)"}}
	for _, sec := range append(append(bs.Aset, bs.Liabilitas...), bs.Ekuitas) {
		nr = append(nr, []string{"", string(sec.Kelompok), ""})
		for _, l := range sec.Lines {
			nr = append(nr, []string{l.Kode, l.Nama, rp(l.Jumlah)})
		}
		nr = append(nr, []string{"", "Total " + string(sec.Kelompok), rp(sec.Total)})
	}
	nr = append(nr, []string{"", "TOTAL ASET", rp(bs.TotalAset)}, []string{"", "TOTAL LIABILITAS + EKUITAS", rp(bs.TotalLiabilitas + bs.TotalEkuitas)})

	ns := [][]string{{"Kode", "Akun", "Debit (Rp)", "Kredit (Rp)"}}
	var d, k int64
	for _, b := range rep.TrialBalance {
		ns = append(ns, []string{b.Kode, b.Nama, rp(b.Debit), rp(b.Kredit)})
		d, k = d+b.Debit, k+b.Kredit
	}
	ns = append(ns, []string{"", "TOTAL", rp(d), rp(k)})

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="laporan-keuangan-%s.xlsx"`, periodOrDefault(r)))
	_ = xlsx.Write(w, []xlsx.Sheet{{Name: "Laba Rugi", Rows: lr}, {Name: "Neraca", Rows: nr}, {Name: "Neraca Saldo", Rows: ns}})
}

// validAkun: koreksi akun harus ada di CoA aktif; tanpa CoA (jasa/manufaktur) tidak boleh diisi.
func (s *Server) validAkun(t finance.Transaction) bool {
	if t.Akun == "" {
		return true
	}
	_, ok := coa.Find(coa.Active(s.currentProfile()), t.Akun)
	return ok
}
