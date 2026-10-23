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
	Bukti      string `json:"bukti"` // No. Bukti yang sama dengan Jurnal Umum
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
	// BolehMinus: saldo minus akun ini wajar (laba/rugi), jadi tidak ditandai salah.
	BolehMinus bool `json:"boleh_minus,omitempty"`
}

type bukuBesarResp struct {
	Supported   bool            `json:"supported"`
	Period      string          `json:"period"`
	PeriodLabel string          `json:"period_label"`
	Dari        string          `json:"dari"`
	Sampai      string          `json:"sampai"`
	Akun        []akunBukuBesar `json:"akun"`
	TotalDebit  int64           `json:"total_debit"`
	TotalKredit int64           `json:"total_kredit"`
	// Seimbang: total debit = total kredit seluruh jurnal dalam rentang.
	Seimbang bool `json:"seimbang"`
	// Pilihan: semua akun CoA usaha (untuk memilih riwayat satu akun).
	Pilihan []pilihanAkun `json:"pilihan"`
}

type pilihanAkun struct {
	Kode string `json:"kode"`
	Nama string `json:"nama"`
}

// bukuBesar menyusun buku besar per akun untuk rentang tanggal (boleh lintas bulan), urut sesuai
// CoA. Saldo awal = saldo awal bulan pertama ditambah mutasi bulan itu sebelum tanggal dari;
// mutasi bulan-bulan berikutnya menyambung, jadi saldo berjalan tidak terputus. akun ≠ "" =
// hanya riwayat akun itu.
func (s *tenant) bukuBesar(dari, sampai, akun string) bukuBesarResp {
	out := bukuBesarResp{Period: dari[:7], PeriodLabel: labelRentang(dari, sampai), Dari: dari, Sampai: sampai, Akun: []akunBukuBesar{}, Pilihan: []pilihanAkun{}}
	var urutan []coa.Account
	akunMap := map[string]*akunBukuBesar{}
	pengisi := map[string]bool{} // akun nol yang ditambahkan, belum pernah muncul di neraca saldo
	nb := s.penomoranBukti()
	for bulan := dari[:7]; bulan <= sampai[:7]; bulan = bulanBerikut(bulan) {
		j, ok := s.jurnalPeriode(bulan)
		if !ok {
			return out
		}
		out.Supported = true
		if bulan == dari[:7] {
			urutan = j.all
			for _, a := range j.all {
				out.Pilihan = append(out.Pilihan, pilihanAkun{a.Kode, a.Nama})
			}
		}
		for _, b := range j.tb {
			if a := akunMap[b.Kode]; a == nil || (pengisi[b.Kode] && a.Debit == 0 && a.Kredit == 0 && a.SaldoAwal == 0) {
				delete(pengisi, b.Kode)
				a := &akunBukuBesar{Kode: b.Kode, Nama: b.Nama, Kelompok: b.Kelompok, Normal: b.Normal, Baris: []barisBukuBesar{}, BolehMinus: bolehMinus(b.Subtype, b.Kode)}
				// Akun yang baru muncul di bulan berikutnya: saldo awalnya dari bulan itu (0
				// sebelum ada mutasi apa pun).
				a.SaldoAwal = tandaNormal(b.Normal) * j.saldoAwal[b.Kode]
				a.SaldoAkhir = a.SaldoAwal
				akunMap[b.Kode] = a
			}
		}
		// Akun aktif tanpa saldo & mutasi tetap dikirim (bernilai nol) supaya "Tampilkan akun
		// tanpa saldo" menampilkan seluruh CoA, termasuk usaha yang belum punya transaksi.
		for _, a := range j.active {
			if akunMap[a.Kode] == nil {
				akunMap[a.Kode] = &akunBukuBesar{Kode: a.Kode, Nama: a.Nama, Kelompok: a.Kelompok, Normal: a.Normal, Baris: []barisBukuBesar{}, BolehMinus: bolehMinus(a.Subtype, a.Kode)}
				pengisi[a.Kode] = true
			}
		}
		// Nomor bukti dihitung dari urutan asli j.entri (sama dengan Jurnal Umum), baru diurutkan.
		grup, bukti := nb.kelompokkan(j.entri, bulan)
		urut := make([]int, len(j.entri))
		for i := range urut {
			urut[i] = i
		}
		slices.SortStableFunc(urut, func(x, y int) int { return cmp.Compare(j.entri[x].Tanggal, j.entri[y].Tanggal) })
		for _, x := range urut {
			e := j.entri[x]
			a := akunMap[e.Kode]
			if a == nil {
				continue
			}
			geser := tandaNormal(a.Normal) * (e.Debit - e.Kredit)
			if e.Tanggal < dari {
				a.SaldoAwal += geser // sebelum rentang: masuk saldo awal
				a.SaldoAkhir = a.SaldoAwal
				continue
			}
			if e.Tanggal > sampai {
				continue
			}
			a.SaldoAkhir += geser
			a.Debit += e.Debit
			a.Kredit += e.Kredit
			out.TotalDebit += e.Debit
			out.TotalKredit += e.Kredit
			a.Baris = append(a.Baris, barisBukuBesar{Tanggal: e.Tanggal, Bukti: bukti[grup[x]], TxID: e.TxID, Keterangan: e.Ket, Debit: e.Debit, Kredit: e.Kredit, Saldo: a.SaldoAkhir})
		}
	}
	out.Seimbang = out.TotalDebit == out.TotalKredit
	for _, acc := range urutan {
		if a := akunMap[acc.Kode]; a != nil && (akun == "" || akun == a.Kode) {
			out.Akun = append(out.Akun, *a)
		}
	}
	return out
}

func tandaNormal(n coa.Normal) int64 {
	if n == coa.Kredit {
		return -1
	}
	return 1
}

func bulanBerikut(bulan string) string {
	t, _ := time.Parse("2006-01", bulan)
	return t.AddDate(0, 1, 0).Format("2006-01")
}

// labelRentang: "Agustus 2026" bila tepat satu bulan penuh, selain itu "5 Agu 2026 – 20 Agu 2026".
func labelRentang(dari, sampai string) string {
	a, _ := time.Parse(time.DateOnly, dari)
	b, _ := time.Parse(time.DateOnly, sampai)
	if a.Day() == 1 && b.Equal(a.AddDate(0, 1, -1)) {
		return finance.LabelPeriode(dari[:7])
	}
	return finance.FormatTanggalID(dari) + " – " + finance.FormatTanggalID(sampai)
}

func (s *tenant) handleBukuBesar(w http.ResponseWriter, r *http.Request) {
	dari, sampai, err := rentangDari(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, s.bukuBesar(dari, sampai, r.URL.Query().Get("akun")))
}

func (s *tenant) handleBukuBesarXlsx(w http.ResponseWriter, r *http.Request) {
	dari, sampai, err := rentangDari(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	bb := s.bukuBesar(dari, sampai, r.URL.Query().Get("akun"))
	if !bb.Supported {
		http.Error(w, "CoA untuk jenis usaha ini belum tersedia", http.StatusNotFound)
		return
	}
	ringkas := [][]string{{"Kode", "Akun", "Kelompok", "Saldo Awal", "Debit", "Kredit", "Saldo Akhir"}}
	rinci := [][]string{{"Kode", "Akun", "Tanggal", "No. Bukti", "Keterangan", "Debit", "Kredit", "Saldo"}}
	for _, a := range bb.Akun {
		ringkas = append(ringkas, []string{a.Kode, a.Nama, string(a.Kelompok), rp(a.SaldoAwal), rp(a.Debit), rp(a.Kredit), rp(a.SaldoAkhir)})
		if len(a.Baris) == 0 && a.SaldoAwal == 0 {
			continue
		}
		rinci = append(rinci, []string{a.Kode, a.Nama, dari, "", "Saldo awal", "", "", rp(a.SaldoAwal)})
		for _, b := range a.Baris {
			rinci = append(rinci, []string{a.Kode, a.Nama, b.Tanggal, b.Bukti, b.Keterangan, rp(b.Debit), rp(b.Kredit), rp(b.Saldo)})
		}
		rinci = append(rinci, []string{a.Kode, a.Nama, "", "", "Saldo akhir", rp(a.Debit), rp(a.Kredit), rp(a.SaldoAkhir)}, []string{})
	}
	ringkas = append(ringkas, []string{"", "TOTAL MUTASI", "", "", rp(bb.TotalDebit), rp(bb.TotalKredit), ""})
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="buku-besar-%s_%s.xlsx"`, dari, sampai))
	_ = xlsx.Write(w, []xlsx.Sheet{{Name: "Ringkasan", Rows: ringkas}, {Name: "Buku Besar", Rows: rinci}})
}
