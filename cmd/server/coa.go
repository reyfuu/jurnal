package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"pajak/internal/coa"
	"pajak/internal/finance"
	"pajak/internal/xlsx"
)

func (s *tenant) currentProfile() finance.BusinessProfile {
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

func (s *tenant) handleCoA(w http.ResponseWriter, r *http.Request) {
	_, active, p := s.coaAkun()
	var mapping []mappingRow
	if active != nil {
		for _, j := range finance.JenisList {
			kode := coa.AutoAccount(j.Nama, p, coa.PetaDari(active))
			acc, _ := coa.Find(active, kode)
			mapping = append(mapping, mappingRow{j.Nama, kode, acc.Nama})
		}
	}
	out := map[string]any{"supported": active != nil, "accounts": active, "mapping": mapping, "kelompok": coa.Kelompoks}
	if t, baru := s.pembaruanCOA(p); t.ID != 0 {
		out["template"] = map[string]any{"id": t.ID, "jenis": t.Jenis, "versi": t.Versi}
		out["pembaruan"] = baru
	}
	writeJSON(w, out)
}

type coaReport struct {
	Supported       bool                       `json:"supported"`
	TrialBalance    []coa.Balance              `json:"trial_balance"`
	IncomeStatement *coa.IncomeStatementResult `json:"income_statement,omitempty"`
	BalanceSheet    *coa.BalanceSheetResult    `json:"balance_sheet,omitempty"`
}

func (s *tenant) buildCoAReport(r *http.Request) coaReport {
	all, active, p := s.coaAkun()
	if active == nil {
		return coaReport{}
	}
	period := periodOrDefault(r)
	// Buku berjalan: saldo awal akun = posisi akhir bulan lalu; transaksi bulan ini dan
	// pelunasan transaksi bulan lalu dijurnal di bulan ini.
	s.mu.RLock()
	peta := coa.PetaDari(all)
	lines := coa.JournalBerjalan(s.txs, period, p, peta)
	// Pelunasan piutang/utang saldo awal bulan ini: Dr Kas / Cr Piutang, atau Dr Utang / Cr Kas.
	kas, piutang, utang := peta.Kode(coa.PeranKas), peta.Kode(coa.PeranPiutang), peta.Kode(coa.PeranUtangUsaha)
	for _, pa := range s.bukuAwalLocked().Pelunasan {
		if len(pa.Tanggal) < 7 || pa.Tanggal[:7] != period {
			continue
		}
		if pa.Jenis == "piutang" {
			lines = append(lines, coa.Line{Tanggal: pa.Tanggal, Kode: kas, Debit: pa.Nominal}, coa.Line{Tanggal: pa.Tanggal, Kode: piutang, Kredit: pa.Nominal})
		} else {
			lines = append(lines, coa.Line{Tanggal: pa.Tanggal, Kode: utang, Debit: pa.Nominal}, coa.Line{Tanggal: pa.Tanggal, Kode: kas, Kredit: pa.Nominal})
		}
	}
	opening := s.openingFor(period)
	saldoLain := coa.SaldoAwalLain(s.bukuAwalLocked(), all, peta)
	sp := s.kartuLocked().Periode(period)
	rep := s.reportLocked(period)
	s.mu.RUnlock()
	// Register aset ikut dijurnal supaya laba bersih & total aset di laporan per akun sama
	// dengan tab Laba Rugi / Neraca, yang memang satu menu.
	as := rep.Assets
	lines = append(lines, coa.DepreciationJournal(as, period, peta, p)...)
	saldoAwal := coa.OpeningBerjalan(opening.Cash, opening.Receivables, opening.Inventory, opening.Payables, opening.Loans,
		rep.BalanceSheet.Equity.RetainedEarnings, coa.OpeningAssets(as, period, peta), saldoLain, peta, p)
	tb := coa.TrialBalanceFrom(all, active, saldoAwal, lines)
	// Kartu stok: sesuaikan akun persediaan ke nilai persediaan akhir; selisihnya masuk HPP.
	var penyesuaian int64
	if sp.Aktif {
		var saldoPersediaan int64
		for _, b := range tb {
			if b.Kode == peta.Kode(coa.PeranPersediaan) {
				saldoPersediaan = b.Saldo
			}
		}
		if penyesuaian = sp.PersediaanAkhir - saldoPersediaan; penyesuaian != 0 {
			lines = append(lines, coa.PenyesuaianPersediaan(period+"-28", penyesuaian, p, peta)...)
			tb = coa.TrialBalanceFrom(all, active, saldoAwal, lines)
		}
	}
	is := coa.IncomeStatementKartu(tb, p, sp.Aktif, penyesuaian, peta)
	bs := coa.BalanceSheet(tb, is.LabaBersih)
	return coaReport{true, tb, &is, &bs}
}

func (s *tenant) handleReportCoA(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.buildCoAReport(r))
}

func rp(n int64) string { return strconv.FormatInt(n, 10) }

func (s *tenant) handleReportCoAXlsx(w http.ResponseWriter, r *http.Request) {
	period := periodOrDefault(r) // masuk ke header Content-Disposition: harus YYYY-MM
	if _, err := time.Parse("2006-01", period); err != nil {
		http.Error(w, "period harus YYYY-MM", http.StatusBadRequest)
		return
	}
	rep := s.buildCoAReport(r)
	all, _, _ := s.coaAkun()
	kodePersediaan := coa.PetaDari(all).Kode(coa.PeranPersediaan)
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
		lr = append(lr, []string{kodePersediaan, "Persediaan Awal", rp(is.PersediaanAwal)})
	}
	add("Harga Pokok Penjualan", is.HPPLines)
	if is.Metode == "periodik" {
		lr = append(lr, []string{kodePersediaan, "Persediaan Akhir", rp(-is.PersediaanAkhir)})
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
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="laporan-keuangan-%s.xlsx"`, period))
	_ = xlsx.Write(w, []xlsx.Sheet{{Name: "Laba Rugi", Rows: lr}, {Name: "Neraca", Rows: nr}, {Name: "Neraca Saldo", Rows: ns}})
}

// validAkun: koreksi akun harus ada di CoA aktif; tanpa CoA (jasa/manufaktur) tidak boleh diisi.
func (s *tenant) validAkun(t finance.Transaction) bool {
	if t.Akun == "" {
		return true
	}
	_, active, _ := s.coaAkun()
	_, ok := coa.Find(active, t.Akun)
	return ok
}
