package main

import (
	"cmp"
	"fmt"
	"net/http"
	"slices"
	"time"

	"pajak/internal/coa"
	"pajak/internal/finance"
	"pajak/internal/xlsx"
)

// Buku Besar: setiap akun dengan saldo awal, mutasi debit/kredit per baris jurnal, dan saldo
// berjalan. Dibentuk dari jurnal yang sama dengan Laba Rugi, Neraca & laporan per akun
// (jurnalPeriode), jadi saldo akhirnya sama dengan neraca saldo bulan itu.

type barisBukuBesar struct {
	Tanggal    string `json:"tanggal"`
	TxID       int    `json:"tx_id,omitempty"`
	Keterangan string `json:"keterangan"`
	Debit      int64  `json:"debit"`
	Kredit     int64  `json:"kredit"`
	Saldo      int64  `json:"saldo"` // menurut saldo normal akun
}

type akunBukuBesar struct {
	Kode       string           `json:"kode"`
	Nama       string           `json:"nama"`
	Kelompok   coa.Kelompok     `json:"kelompok"`
	Normal     coa.Normal       `json:"normal"`
	SaldoAwal  int64            `json:"saldo_awal"`
	Debit      int64            `json:"debit"`
	Kredit     int64            `json:"kredit"`
	SaldoAkhir int64            `json:"saldo_akhir"`
	Baris      []barisBukuBesar `json:"baris"`
}

type bukuBesarResp struct {
	Supported   bool            `json:"supported"`
	Period      string          `json:"period"`
	PeriodLabel string          `json:"period_label"`
	Akun        []akunBukuBesar `json:"akun"`
	TotalDebit  int64           `json:"total_debit"`
	TotalKredit int64           `json:"total_kredit"`
	// Seimbang: total debit = total kredit seluruh jurnal bulan ini.
	Seimbang bool `json:"seimbang"`
}

// bukuBesar menyusun buku besar per akun, urut sesuai CoA. Akun tanpa saldo awal & tanpa mutasi
// tidak ikut, kecuali akun aktif yang memang selalu tampil di neraca saldo.
func (s *tenant) bukuBesar(period string) bukuBesarResp {
	j, ok := s.jurnalPeriode(period)
	out := bukuBesarResp{Supported: ok, Period: period, PeriodLabel: finance.LabelPeriode(period), Akun: []akunBukuBesar{}}
	if !ok {
		return out
	}
	perAkun := map[string][]entriJurnal{}
	for _, e := range j.entri {
		perAkun[e.Kode] = append(perAkun[e.Kode], e)
		out.TotalDebit += e.Debit
		out.TotalKredit += e.Kredit
	}
	out.Seimbang = out.TotalDebit == out.TotalKredit
	for _, b := range j.tb {
		a := akunBukuBesar{Kode: b.Kode, Nama: b.Nama, Kelompok: b.Kelompok, Normal: b.Normal, Baris: []barisBukuBesar{}}
		tanda := int64(1) // saldo menurut posisi normal: kredit-normal dibalik
		if b.Normal == coa.Kredit {
			tanda = -1
		}
		a.SaldoAwal = tanda * j.saldoAwal[b.Kode]
		saldo := a.SaldoAwal
		entri := perAkun[b.Kode]
		slices.SortStableFunc(entri, func(x, y entriJurnal) int { return cmp.Compare(x.Tanggal, y.Tanggal) })
		for _, e := range entri {
			saldo += tanda * (e.Debit - e.Kredit)
			a.Debit += e.Debit
			a.Kredit += e.Kredit
			a.Baris = append(a.Baris, barisBukuBesar{Tanggal: e.Tanggal, TxID: e.TxID, Keterangan: e.Ket, Debit: e.Debit, Kredit: e.Kredit, Saldo: saldo})
		}
		a.SaldoAkhir = saldo
		out.Akun = append(out.Akun, a)
	}
	return out
}

func (s *tenant) handleBukuBesar(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.bukuBesar(periodOrDefault(r)))
}

func (s *tenant) handleBukuBesarXlsx(w http.ResponseWriter, r *http.Request) {
	period := periodOrDefault(r) // masuk ke nama berkas: harus YYYY-MM
	if _, err := time.Parse("2006-01", period); err != nil {
		http.Error(w, "period harus YYYY-MM", http.StatusBadRequest)
		return
	}
	bb := s.bukuBesar(period)
	if !bb.Supported {
		http.Error(w, "CoA untuk jenis usaha ini belum tersedia", http.StatusNotFound)
		return
	}
	ringkas := [][]string{{"Kode", "Akun", "Kelompok", "Saldo Awal", "Debit", "Kredit", "Saldo Akhir"}}
	rinci := [][]string{{"Kode", "Akun", "Tanggal", "Keterangan", "Debit", "Kredit", "Saldo"}}
	for _, a := range bb.Akun {
		ringkas = append(ringkas, []string{a.Kode, a.Nama, string(a.Kelompok), rp(a.SaldoAwal), rp(a.Debit), rp(a.Kredit), rp(a.SaldoAkhir)})
		if len(a.Baris) == 0 && a.SaldoAwal == 0 {
			continue
		}
		rinci = append(rinci, []string{a.Kode, a.Nama, "", "Saldo awal", "", "", rp(a.SaldoAwal)})
		for _, b := range a.Baris {
			rinci = append(rinci, []string{a.Kode, a.Nama, b.Tanggal, b.Keterangan, rp(b.Debit), rp(b.Kredit), rp(b.Saldo)})
		}
		rinci = append(rinci, []string{a.Kode, a.Nama, "", "Saldo akhir", rp(a.Debit), rp(a.Kredit), rp(a.SaldoAkhir)}, []string{})
	}
	ringkas = append(ringkas, []string{"", "TOTAL MUTASI", "", "", rp(bb.TotalDebit), rp(bb.TotalKredit), ""})
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="buku-besar-%s.xlsx"`, period))
	_ = xlsx.Write(w, []xlsx.Sheet{{Name: "Ringkasan", Rows: ringkas}, {Name: "Buku Besar", Rows: rinci}})
}
