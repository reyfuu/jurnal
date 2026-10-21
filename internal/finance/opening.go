package finance

import (
	"errors"
	"fmt"
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
	// ModalDisetor & LabaDitahan: sisi modal neraca awal. Untuk saldo awal per akun (Rincian)
	// keduanya wajib dan harus membuat Harta = Kewajiban + Modal; saldo awal lama yang hanya
	// mengisi ModalDisetor untuk dicocokkan tetap terbaca seperti dulu.
	ModalDisetor int64 `json:"modal_disetor"`
	LabaDitahan  int64 `json:"laba_ditahan,omitempty"` // boleh negatif (rugi yang belum tertutup)
	// AsetLain: harta lancar selain kas, piutang usaha, dan stok (uang muka, biaya dibayar di
	// muka, pajak dibayar di muka). KewajibanLain: selain utang usaha & pinjaman (utang pajak,
	// beban yang masih harus dibayar). Keduanya tetap sampai ada transaksi yang menyentuhnya.
	AsetLain      int64 `json:"aset_lain,omitempty"`
	KewajibanLain int64 `json:"kewajiban_lain,omitempty"`
	// AsetTetap: nilai buku aset tetap yang DINYATAKAN di saldo awal; harus sama dengan nilai
	// buku register aset per tanggal mulai (aset tetap sendiri dicatat di register, bukan di sini).
	AsetTetap int64 `json:"aset_tetap,omitempty"`
	// Rincian: saldo awal per akun (template CoA / menu Saldo Awal). Bila ada, semua angka di
	// atas diturunkan dari sini lewat TerapkanRincian.
	Rincian []SaldoAkun `json:"rincian,omitempty"`
	// Pelunasan: cicilan pelunasan piutang/utang saldo awal (tidak punya transaksi asal).
	Pelunasan []PelunasanAwal `json:"pelunasan,omitempty"`
}

// PelunasanAwal: pelanggan melunasi piutang saldo awal (Jenis "piutang") atau usaha membayar
// utang saldo awal ("utang") pada Tanggal; kas bergerak, laba tidak.
type PelunasanAwal struct {
	Jenis      string `json:"jenis"`
	Tanggal    string `json:"tanggal"`
	Nominal    int64  `json:"nominal"`
	Keterangan string `json:"keterangan,omitempty"`
}

// sisaAwal: piutang/utang saldo awal yang belum dilunasi sampai tanggal itu.
func (o OpeningBalance) sisaAwal(piutang bool, sampai string) int64 {
	sisa, jenis := o.Utang, "utang"
	if piutang {
		sisa, jenis = o.Piutang, "piutang"
	}
	for _, p := range o.Pelunasan {
		if p.Jenis == jenis && p.Tanggal <= sampai {
			sisa -= p.Nominal
		}
	}
	return sisa
}

// TambahPelunasan memvalidasi lalu mencatat pelunasan saldo awal.
func (o *OpeningBalance) TambahPelunasan(p PelunasanAwal) error {
	p.Keterangan = strings.TrimSpace(p.Keterangan)
	switch {
	case p.Jenis != "piutang" && p.Jenis != "utang":
		return errors.New("jenis pelunasan harus piutang atau utang")
	case p.Nominal <= 0:
		return errors.New("nominal pelunasan harus lebih dari 0")
	case p.Tanggal < o.Tanggal:
		return errors.New("tanggal pelunasan tidak boleh sebelum tanggal saldo awal")
	}
	if _, err := time.Parse(time.DateOnly, p.Tanggal); err != nil {
		return errors.New("tanggal pelunasan wajib diisi (YYYY-MM-DD)")
	}
	if p.Nominal > o.sisaAwal(p.Jenis == "piutang", "9999-12-31") {
		return errors.New("nominal melebihi sisa " + p.Jenis + " saldo awal")
	}
	o.Pelunasan = append(o.Pelunasan, p)
	return nil
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
	return o.Harta(nilaiBukuAset) - o.Kewajiban()
}

// Harta / Kewajiban / Modal: tiga sisi neraca awal.
func (o OpeningBalance) Harta(nilaiBukuAset int64) int64 {
	return o.Kas + o.Piutang + o.Persediaan + o.AsetLain + nilaiBukuAset
}
func (o OpeningBalance) Kewajiban() int64 { return o.Utang + o.Pinjaman + o.KewajibanLain }
func (o OpeningBalance) Modal() int64     { return o.ModalDisetor + o.LabaDitahan }

// SelisihModal = Modal yang diisi − (Harta − Kewajiban); 0 berarti Harta = Kewajiban + Modal.
// Saldo awal lama tanpa rincian dan tanpa modal (modal dulu dihitung otomatis) dianggap cocok.
func (o OpeningBalance) SelisihModal(nilaiBukuAset int64) int64 {
	if len(o.Rincian) == 0 && o.ModalDisetor == 0 && o.LabaDitahan == 0 {
		return 0
	}
	return o.Modal() - o.ModalAwal(nilaiBukuAset)
}

// CekSeimbang: saldo awal baru wajib Harta = Kewajiban + Modal, dan aset tetap yang dinyatakan
// harus sama dengan nilai buku register aset. Tanpa ini selisihnya diam-diam menjadi "modal"
// dan terbawa ke semua laporan sesudahnya.
func (o OpeningBalance) CekSeimbang(nilaiBukuAset int64) error {
	if o.AsetTetap != nilaiBukuAset {
		return fmt.Errorf("aset tetap (nilai buku) di saldo awal %s, sedangkan menurut daftar aset per %s nilai bukunya %s. Samakan dengan daftar aset; aset tetap dicatat di sheet / menu Aset",
			formatRpID(o.AsetTetap), o.Tanggal, formatRpID(nilaiBukuAset))
	}
	if d := o.Harta(nilaiBukuAset) - o.Kewajiban() - o.Modal(); d != 0 {
		return fmt.Errorf("saldo awal belum seimbang: Harta %s, Kewajiban %s, Modal %s; selisih %s. Harta harus sama dengan Kewajiban + Modal",
			formatRpID(o.Harta(nilaiBukuAset)), formatRpID(o.Kewajiban()), formatRpID(o.Modal()), formatRpID(d))
	}
	return nil
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
