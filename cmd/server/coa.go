package main

import (
	"cmp"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
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

// entriJurnal: satu baris jurnal beserta keterangannya, untuk Buku Besar.
type entriJurnal struct {
	coa.Line
	Ket string
}

// jurnalBulan: semua yang membentuk laporan per akun satu periode — saldo awal akun, baris jurnal
// (transaksi, pelunasan saldo awal, penyusutan, penyesuaian kartu stok) dengan keterangannya, dan
// neraca saldo. Laporan per akun dan Buku Besar memakai fungsi ini, jadi angkanya selalu sama.
type jurnalBulan struct {
	all, active []coa.Account
	p           finance.BusinessProfile
	peta        coa.Peta
	saldoAwal   map[string]int64
	entri       []entriJurnal
	tb          []coa.Balance
	penyesuaian int64
	kartuAktif  bool
}

func (s *tenant) jurnalPeriode(period string) (jurnalBulan, bool) {
	all, active, p := s.coaAkun()
	if active == nil {
		return jurnalBulan{}, false
	}
	j := jurnalBulan{all: all, active: active, p: p, peta: coa.PetaDari(all)}
	peta := j.peta
	// Buku berjalan: saldo awal akun = posisi akhir bulan lalu; transaksi bulan ini dan
	// pelunasan transaksi bulan lalu dijurnal di bulan ini.
	s.mu.RLock()
	ketTx, bulanTx := map[int]string{}, map[int]string{}
	for _, t := range s.txs {
		if len(t.Tanggal) >= 7 {
			bulanTx[t.ID] = t.Tanggal[:7]
		}
		ket := t.Jenis
		if k := strings.TrimSpace(cmp.Or(t.Keterangan, t.Item)); k != "" {
			ket += " — " + k
		}
		if t.Pihak != "" {
			ket += " (" + t.Pihak + ")"
		}
		ketTx[t.ID] = ket
	}
	k := s.kartuLocked()
	sp := k.Periode(period)
	akunOf := s.akunStokLocked(all, peta)
	// Perpetual dengan kartu stok aktif: pembelian barang langsung ke akun persediaan barangnya.
	var akunBeli func(int) string
	if sp.Aktif {
		akunBeli = akunOf
	}
	for _, l := range coa.JournalBerjalan(s.txs, period, p, peta, akunBeli) {
		ket := ketTx[l.TxID]
		if bulanTx[l.TxID] < period {
			ket = "Pelunasan: " + ket // transaksi bulan lalu yang dibayar bulan ini
		}
		j.entri = append(j.entri, entriJurnal{l, ket})
	}
	// Perpetual dengan kartu stok aktif: penjualan, produksi, dan stok opname dijurnal di tanggal
	// kejadiannya ke akun barangnya; penyesuaian sisa di bawah hanya menyapu sisanya (pembelian
	// tanpa barang, koreksi stok minus, biaya lain yang tidak berakun barang hasil).
	if sp.Aktif && p.MetodePersediaan == "perpetual" {
		for _, l := range coa.JurnalKartuKet(k.Baris, period+"-01", finance.AkhirBulan(period), akunOf, peta) {
			ket := l.Ket
			if l.TxID != 0 && ketTx[l.TxID] != "" {
				ket = ketHPP + ketTx[l.TxID]
			}
			j.entri = append(j.entri, entriJurnal{l.Line, ket})
		}
	}
	// Pelunasan piutang/utang saldo awal bulan ini: Dr Kas / Cr Piutang, atau Dr Utang / Cr Kas.
	kas, piutang, utang := peta.Kode(coa.PeranKas), peta.Kode(coa.PeranPiutang), peta.Kode(coa.PeranUtangUsaha)
	for _, pa := range s.bukuAwalLocked().Pelunasan {
		if len(pa.Tanggal) < 7 || pa.Tanggal[:7] != period {
			continue
		}
		ket := cmp.Or(pa.Keterangan, "Pelunasan "+pa.Jenis+" saldo awal")
		rek := cmp.Or(pa.Akun, kas) // rekening yang dipilih / bawaan Bank; pelunasan lama = Kas
		if pa.Jenis == "piutang" {
			j.entri = append(j.entri, entriJurnal{coa.Line{Tanggal: pa.Tanggal, Kode: rek, Debit: pa.Nominal}, ket}, entriJurnal{coa.Line{Tanggal: pa.Tanggal, Kode: piutang, Kredit: pa.Nominal}, ket})
		} else {
			j.entri = append(j.entri, entriJurnal{coa.Line{Tanggal: pa.Tanggal, Kode: utang, Debit: pa.Nominal}, ket}, entriJurnal{coa.Line{Tanggal: pa.Tanggal, Kode: rek, Kredit: pa.Nominal}, ket})
		}
	}
	opening := s.openingFor(period)
	saldoLain := coa.SaldoAwalLain(s.bukuAwalLocked(), all, peta)
	// Kas di bank / saldo marketplace & prive dibawa per akun; kas sisanya di akun Kas. Baris kas
	// saldo awal yang berakun sendiri (Bank, rekening tambahan) tetap di akunnya.
	kasAwal := opening.Cash
	for kode, v := range coa.KasAwalLain(s.bukuAwalLocked(), all, peta) {
		saldoLain[kode] += v
		kasAwal -= v
	}
	// pelunasan saldo awal bulan-bulan sebelumnya lewat rekening selain Kas dibawa per akun
	for _, pa := range s.bukuAwalLocked().Pelunasan {
		if pa.Akun == "" || pa.Akun == kas || pa.Tanggal >= period+"-01" {
			continue
		}
		v := pa.Nominal
		if pa.Jenis != "piutang" {
			v = -v
		}
		saldoLain[pa.Akun] += v
		kasAwal -= v
	}
	for kode, v := range coa.SaldoBawaan(s.txs, period+"-01", p, peta) {
		saldoLain[kode] += v
		if kode != peta.Kode(coa.PeranPrive) {
			kasAwal -= v
		}
	}
	persediaanAwal := s.persediaanAwalPerAkunLocked(period, all, peta)
	var target map[string]int64 // nilai kartu per akun di akhir bulan
	if sp.Aktif {
		target = k.NilaiPerAkun(period+"-31", akunOf)
	}
	rep := s.reportLocked(period)
	s.mu.RUnlock()
	// Register aset ikut dijurnal supaya laba bersih & total aset di laporan per akun sama
	// dengan tab Laba Rugi / Neraca, yang memang satu menu.
	as := rep.Assets
	// DepreciationJournal menulis pasangan baris: beban/akumulasi (penyusutan) atau harta/modal
	// (aset masuk tanpa transaksi kas).
	susut := coa.DepreciationJournal(as, period, peta, p)
	for i := 0; i+1 < len(susut); i += 2 {
		ket := "Penyusutan & amortisasi bulan ini (register aset)"
		if susut[i+1].Kode == kodeModal(peta, p) {
			ket = "Aset masuk tanpa transaksi kas (setoran modal berupa aset)"
		}
		j.entri = append(j.entri, entriJurnal{susut[i], ket}, entriJurnal{susut[i+1], ket})
	}
	j.saldoAwal = coa.OpeningBerjalan(kasAwal, opening.Receivables, persediaanAwal, opening.Payables, opening.Loans,
		rep.BalanceSheet.Equity.RetainedEarnings, coa.OpeningAssets(as, period, peta), saldoLain, peta, p)
	j.tb = coa.TrialBalanceFrom(all, active, j.saldoAwal, j.lines())
	// Kartu stok: sesuaikan tiap akun persediaan ke nilai kartu barang-barangnya di akhir bulan;
	// selisihnya masuk HPP. Akun persediaan yang terisi transaksi tanpa barang ikut ditutup ke
	// nilai kartu, jadi persediaan tidak terhitung dua kali.
	if sp.Aktif {
		j.kartuAktif = true
		saldo := map[string]int64{}
		for _, b := range j.tb {
			if b.Subtype == "inventory" {
				saldo[b.Kode] = b.Saldo
			}
		}
		kode := map[string]bool{}
		for _, a := range coa.AkunPersediaan(active) {
			kode[a.Kode] = true
		}
		for kd := range target {
			kode[kd] = true
		}
		for kd := range saldo {
			kode[kd] = true
		}
		for kd := range persediaanAwal {
			kode[kd] = true
		}
		// penyesuaian = perubahan bersih akun persediaan bulan ini (akhir kartu − saldo awal),
		// bukan jumlah selisih: persediaan yang terisi transaksi tanpa barang tetap tampil
		// sebagai pembelian, dan persediaan awal di laba rugi = saldo awal sebenarnya.
		disesuaikan := false
		tanggal := finance.AkhirBulan(period) // hari terakhir bulan: urut paling akhir di jurnal
		for _, kd := range slices.Sorted(maps.Keys(kode)) {
			j.penyesuaian += target[kd] - persediaanAwal[kd]
			selisih := target[kd] - saldo[kd]
			if selisih == 0 {
				continue
			}
			disesuaikan = true
			for _, l := range coa.PenyesuaianPersediaan(tanggal, kd, selisih, p, peta) {
				j.entri = append(j.entri, entriJurnal{l, "Penyesuaian persediaan ke nilai kartu stok"})
			}
		}
		if disesuaikan {
			j.tb = coa.TrialBalanceFrom(all, active, j.saldoAwal, j.lines())
		}
	}
	return j, true
}

// persediaanAwalPerAkunLocked: saldo awal persediaan per akun pada awal periode. Kartu aktif:
// nilai kartu per akun di akhir bulan lalu, ditambah baris stok awal (titik nol / barang baru)
// bulan ini — sama dengan PersediaanAwal kartu. Selain itu baris persediaan saldo awal
// berkode akun persediaan masuk ke akunnya, sisanya ke akun peran. Jumlahnya selalu sama
// dengan openingFor(period).Inventory. Pemanggil memegang s.mu.
func (s *tenant) persediaanAwalPerAkunLocked(period string, all []coa.Account, peta coa.Peta) map[string]int64 {
	peran := peta.Kode(coa.PeranPersediaan)
	k := s.kartuLocked()
	if k.Periode(period).Aktif {
		akunOf := s.akunStokLocked(all, peta)
		out := k.NilaiPerAkun(period+"-00", akunOf) // semua baris sebelum tanggal 1
		out[peran] += 0                              // akun peran selalu tampil, seperti sebelumnya
		for _, b := range k.Baris {
			if b.Jenis == finance.KartuAwal && b.Tanggal >= period+"-01" && b.Tanggal <= period+"-31" {
				out[cmp.Or(akunOf(b.StokID), peran)] += b.Nilai
			}
		}
		return out
	}
	ob := s.bukuAwalLocked()
	punya := map[string]bool{}
	for _, a := range coa.AkunPersediaan(all) {
		punya[a.Kode] = true
	}
	out := map[string]int64{}
	sisa := ob.Persediaan
	for _, r := range ob.Rincian {
		if r.Kategori == finance.KatPersediaan && punya[r.Kode] && r.Kode != peran {
			out[r.Kode] += r.Nominal
			sisa -= r.Nominal
		}
	}
	out[peran] += sisa
	return out
}

// kodeModal: akun modal usaha ini (sama dengan pilihan jurnal otomatis).
func kodeModal(peta coa.Peta, p finance.BusinessProfile) string {
	if p.BentukUsaha == "pt" || p.BentukUsaha == "pt_perorangan" {
		if k := peta["modal_pt"]; k != "" {
			return k
		}
	}
	return peta.Kode(coa.PeranModal)
}

func (j jurnalBulan) lines() []coa.Line {
	out := make([]coa.Line, len(j.entri))
	for i, e := range j.entri {
		out[i] = e.Line
	}
	return out
}

func (s *tenant) buildCoAReport(r *http.Request) coaReport {
	j, ok := s.jurnalPeriode(periodOrDefault(r))
	if !ok {
		return coaReport{}
	}
	is := coa.IncomeStatementKartu(j.tb, j.p, j.kartuAktif, j.penyesuaian, j.peta)
	bs := coa.BalanceSheet(j.tb, is.LabaBersih)
	return coaReport{true, j.tb, &is, &bs}
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
	all, active, _ := s.coaAkun()
	// Persediaan awal/akhir = jumlah semua akun persediaan; kode akun hanya ditulis bila usaha
	// ini memang punya satu akun persediaan.
	kodePersediaan, namaAwal, namaAkhir := coa.PetaDari(all).Kode(coa.PeranPersediaan), "Persediaan Awal", "Persediaan Akhir"
	if len(coa.AkunPersediaan(active)) > 1 {
		kodePersediaan, namaAwal, namaAkhir = "", "Persediaan Awal (semua akun)", "Persediaan Akhir (semua akun)"
	}
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
		lr = append(lr, []string{kodePersediaan, namaAwal, rp(is.PersediaanAwal)})
	}
	add("Harga Pokok Penjualan", is.HPPLines)
	if is.Metode == "periodik" {
		lr = append(lr, []string{kodePersediaan, namaAkhir, rp(-is.PersediaanAkhir)})
	}
	lr = append(lr, []string{"", "Harga Pokok Penjualan", rp(is.HPP)}, []string{"", "Laba (Rugi) Kotor", rp(is.LabaKotor)})
	add("Beban Operasional", is.BebanOperasional)
	lr = append(lr, []string{"", "Laba (Rugi) Usaha", rp(is.LabaUsaha)})
	add("Pendapatan Lain-lain", is.PendapatanLain)
	add("Beban Lain-lain", is.BebanLain)
	lr = append(lr, []string{"", "Laba (Rugi) Bersih Tahun Berjalan", rp(is.LabaBersih)})

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
