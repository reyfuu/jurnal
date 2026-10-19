package finance

import (
	"errors"
	"strings"
	"time"
)

// Neraca awal onboarding. Laporan keuangan di aplikasi ini diturunkan dari transaksi, jadi
// yang diimpor saat onboarding bukan neraca/arus kas jadi, melainkan posisi awalnya: dari
// sini neraca & arus kas tiap periode terbentuk sendiri dan tidak mungkin bertabrakan dengan
// hasil hitung transaksi.

// OpeningBalance adalah posisi keuangan pada hari pertama usaha memakai aplikasi.
type OpeningBalance struct {
	Tanggal    string `json:"tanggal"` // YYYY-MM-DD; menentukan periode onboarding
	Kas        int64  `json:"kas"`
	Piutang    int64  `json:"piutang"`
	Persediaan int64  `json:"persediaan"`
	Utang      int64  `json:"utang"`
	Pinjaman   int64  `json:"pinjaman"`
	// ModalDisetor opsional: hanya dipakai untuk dicocokkan dengan modal hasil hitung,
	// supaya salah input ketahuan. Modal yang dipakai laporan selalu hasil hitung.
	ModalDisetor int64 `json:"modal_disetor"`
}

func ValidateOpeningBalance(o *OpeningBalance) error {
	o.Tanggal = strings.TrimSpace(o.Tanggal)
	if _, err := time.Parse(time.DateOnly, o.Tanggal); err != nil {
		return errors.New("tanggal mulai pembukuan wajib diisi (format YYYY-MM-DD)")
	}
	for _, c := range []struct {
		nama string
		v    int64
	}{
		{"kas & bank", o.Kas}, {"piutang usaha", o.Piutang}, {"persediaan", o.Persediaan},
		{"utang usaha", o.Utang}, {"pinjaman", o.Pinjaman}, {"modal disetor", o.ModalDisetor},
	} {
		if c.v < 0 {
			return errors.New("nilai " + c.nama + " tidak boleh negatif")
		}
	}
	return nil
}

// Period mengembalikan periode onboarding (YYYY-MM) dari tanggal mulai.
func (o OpeningBalance) Period() string {
	if len(o.Tanggal) < 7 {
		return ""
	}
	return o.Tanggal[:7]
}

// ModalAwal = aktiva awal − kewajiban awal. nilaiBukuAset diambil dari register aset.
func (o OpeningBalance) ModalAwal(nilaiBukuAset int64) int64 {
	return o.Kas + o.Piutang + o.Persediaan + nilaiBukuAset - o.Utang - o.Pinjaman
}

// SelisihModal: 0 bila ModalDisetor tidak diisi atau sudah cocok. Nilai bukan nol berarti
// isian neraca awal tidak seimbang dan perlu diperiksa user.
func (o OpeningBalance) SelisihModal(nilaiBukuAset int64) int64 {
	if o.ModalDisetor == 0 {
		return 0
	}
	return o.ModalDisetor - o.ModalAwal(nilaiBukuAset)
}

// SeedInventory: nilai persediaan data demo.
//
// ponytail: tetap sampai modul stok menghitung persediaan dari transaksi; tenant nyata
// memakai angka dari neraca awal onboarding.
const SeedInventory int64 = 8_250_000

// SeedOpening: saldo awal periode untuk data demo, dipakai selama tenant belum mengisi
// neraca awal onboarding.
//
// Saldo ini SENGAJA tidak berantai: seed menulis saldo awal tiap periode sendiri-sendiri untuk
// memperagakan kas yang turun walau laba positif, jadi kas akhir satu periode tidak sama dengan
// kas awal periode berikutnya. Tenant nyata memakai OpeningBalance.At yang berantai dengan benar.
func SeedOpening(period string) ReportOpening {
	return ReportOpening{Cash: OpeningCash(period), Inventory: SeedInventory}
}

// ReportOpening adalah saldo awal satu periode yang dipakai BuildPeriodReport. Untuk data demo
// diisi dari seed; untuk tenant nyata diturunkan dari OpeningBalance lewat At.
type ReportOpening struct {
	Cash        int64 `json:"cash"`
	Receivables int64 `json:"receivables"`
	Payables    int64 `json:"payables"`
	Inventory   int64 `json:"inventory"`
	Loans       int64 `json:"loans"`
}

// netCashBefore: kas bersih seluruh transaksi lunas sebelum period.
//
// ponytail: kas dianggap bergerak di Tanggal transaksi (TanggalLunas belum dipakai), sama
// seperti BuildCashFlow dan CashWarnings. Perbaiki bersamaan bila pelunasan terpisah dibutuhkan.
func netCashBefore(txs []Transaction, period string) int64 {
	var total int64
	for _, t := range txs {
		if !t.Lunas || len(t.Tanggal) < 7 || t.Tanggal[:7] >= period {
			continue
		}
		if j, ok := jenisOf(t.Jenis); ok {
			if j.Arah == "masuk" {
				total += t.Nominal
			} else {
				total -= t.Nominal
			}
		}
	}
	return total
}

// At menurunkan saldo awal satu periode dari neraca awal. Kas ikut berjalan: saldo awal kas
// ditambah kas bersih semua transaksi sebelum period.
//
// ponytail: piutang, utang, dan pinjaman awal hanya dipakai di periode onboarding karena
// pelunasannya lintas periode belum ditelusuri (umur piutang per periode, Fase D). Periode
// setelahnya memakai akrual periode itu saja — neraca tetap seimbang karena modal awal
// dihitung dari komposisi yang sama.
func (o OpeningBalance) At(txs []Transaction, period string) ReportOpening {
	r := ReportOpening{Cash: o.Kas + netCashBefore(txs, period), Inventory: o.Persediaan}
	if period == o.Period() {
		r.Receivables, r.Payables, r.Loans = o.Piutang, o.Utang, o.Pinjaman
	}
	return r
}
