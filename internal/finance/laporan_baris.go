package finance

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Laporan keuangan dalam bentuk baris siap cetak. Satu susunan dipakai PDF per periode dan
// tabel perbandingan multi-periode, supaya keduanya tidak bisa berbeda dari layar Laporan
// Keuangan. Beban ditulis negatif (mengurangi laba / kas) seperti di layar.

const (
	GayaJudul = "judul" // judul kelompok, tanpa angka
	GayaBaris = "baris"
	GayaSub   = "sub"   // rincian di bawah baris
	GayaTotal = "total" // subtotal / total, dicetak tebal
)

// Agregat menentukan isi kolom Total saat beberapa periode dibandingkan.
const (
	AgregatJumlah = "jumlah" // arus satu periode (laba rugi, arus kas): dijumlahkan
	AgregatAwal   = "awal"   // saldo awal: diambil dari periode paling awal
	AgregatAkhir  = "akhir"  // posisi akhir (neraca, saldo kas akhir): periode paling akhir
)

type BarisLaporan struct {
	Kunci   string `json:"kunci"` // sama antar-periode, untuk menjajarkan kolom perbandingan
	Label   string `json:"label"`
	Nilai   int64  `json:"nilai"`
	Gaya    string `json:"gaya"`
	Agregat string `json:"agregat,omitempty"`
}

type BagianLaporan struct {
	Kode  string         `json:"kode"` // laba-rugi | neraca | arus-kas
	Judul string         `json:"judul"`
	Baris []BarisLaporan `json:"baris"`
}

type susun struct {
	b       BagianLaporan
	agregat string
}

func (s *susun) judul(kunci, label string) {
	s.b.Baris = append(s.b.Baris, BarisLaporan{Kunci: kunci, Label: label, Gaya: GayaJudul})
}

func (s *susun) tambah(gaya, kunci, label string, nilai int64) {
	s.b.Baris = append(s.b.Baris, BarisLaporan{Kunci: kunci, Label: label, Nilai: nilai, Gaya: gaya, Agregat: s.agregat})
}

func BagianLabaRugi(is IncomeStatement) BagianLaporan {
	s := susun{b: BagianLaporan{Kode: "laba-rugi", Judul: "Laporan Laba Rugi"}, agregat: AgregatJumlah}
	s.tambah(GayaBaris, "omzet", "Omzet (penjualan kotor)", is.Omzet)
	s.tambah(GayaBaris, "potongan", "Potongan channel / aplikasi", -is.ChannelFees)
	s.tambah(GayaTotal, "penjualan_bersih", "Penjualan bersih", is.NetSales)
	s.tambah(GayaBaris, "hpp", "HPP (bahan baku)", -is.COGS)
	s.tambah(GayaTotal, "laba_kotor", "Laba kotor", is.GrossProfit)
	s.judul("judul_beban", "Beban operasional")
	for _, l := range is.OpexLines {
		s.tambah(GayaBaris, "beban:"+l.Label, l.Label, -l.Amount)
	}
	s.tambah(GayaTotal, "total_beban", "Total beban operasional (tunai)", -is.Opex)
	s.tambah(GayaTotal, "ebitda", "EBITDA (laba sebelum penyusutan & amortisasi)", is.EBITDA)
	s.judul("judul_penutupan", "Penyusutan & amortisasi (jurnal penutupan bulan)")
	s.tambah(GayaBaris, "penyusutan", "Beban penyusutan aset tetap", -is.Depreciation)
	s.tambah(GayaBaris, "amortisasi", "Beban amortisasi harta tak berwujud", -is.Amortization)
	s.tambah(GayaTotal, "ebit", "Laba usaha (EBIT)", is.EBIT)
	if is.OtherIncome != 0 {
		s.tambah(GayaBaris, "pendapatan_lain", "Pendapatan lain", is.OtherIncome)
	}
	s.tambah(GayaTotal, "laba_bersih", "Laba bersih", is.NetProfit)
	return s.b
}

func BagianNeraca(bs BalanceSheet) BagianLaporan {
	s := susun{b: BagianLaporan{Kode: "neraca", Judul: "Neraca (Posisi Keuangan)"}, agregat: AgregatAkhir}
	a := bs.Assets
	s.judul("judul_aset", "Aset (Aktiva)")
	s.tambah(GayaBaris, "kas", "Saldo kas & bank", a.Cash)
	s.tambah(GayaBaris, "piutang", "Piutang belum tertagih", a.Receivables)
	s.tambah(GayaBaris, "persediaan", "Nilai persediaan stok", a.Inventory)
	s.tambah(GayaBaris, "aset_tetap", "Aset tetap & tak berwujud (nilai buku)", a.FixedAssets)
	s.tambah(GayaSub, "harga_perolehan", "Harga perolehan", a.FixedAssetsCost)
	s.tambah(GayaSub, "akumulasi", "Akumulasi penyusutan & amortisasi", -a.AccumDepreciation)
	s.tambah(GayaTotal, "total_aset", "Total aset", a.Total)
	s.judul("judul_kewajiban", "Kewajiban (Utang)")
	s.tambah(GayaBaris, "utang_usaha", "Utang usaha / operasional", bs.Liabilities.Payables)
	s.tambah(GayaBaris, "pinjaman", "Pinjaman", bs.Liabilities.Loans)
	s.tambah(GayaTotal, "total_kewajiban", "Total kewajiban", bs.Liabilities.Total)
	s.judul("judul_modal", "Modal (Ekuitas)")
	s.tambah(GayaBaris, "modal_awal", "Modal awal", bs.Equity.OpeningCapital)
	s.tambah(GayaBaris, "tambahan_modal", "Tambahan modal - ambil pribadi", bs.Equity.AddedCapital)
	s.tambah(GayaBaris, "laba_ditahan", "Laba ditahan (bulan-bulan sebelumnya)", bs.Equity.RetainedEarnings)
	s.tambah(GayaBaris, "laba_berjalan", "Laba bersih periode berjalan", bs.Equity.CurrentProfit)
	s.tambah(GayaTotal, "total_modal", "Total modal", bs.Equity.Total)
	s.tambah(GayaTotal, "total_pasiva", "Total pasiva (utang + modal)", bs.Liabilities.Total+bs.Equity.Total)
	return s.b
}

// BagianArusKas menyajikan arus kas metode langsung, atau metode tidak langsung bila
// tidakLangsung; Investasi & Pendanaan sama di kedua metode.
func BagianArusKas(cf CashFlow, tidakLangsung bool) BagianLaporan {
	judul, operasi := "Laporan Arus Kas (Metode Langsung)", cf.OperasiLangsung
	if tidakLangsung {
		judul, operasi = "Laporan Arus Kas (Metode Tidak Langsung)", cf.OperasiTidakLangsung
	}
	s := susun{b: BagianLaporan{Kode: "arus-kas", Judul: judul}, agregat: AgregatJumlah}
	bagian := func(kode, label, total string, sec CashFlowSection) {
		s.judul("judul_"+kode, label)
		for _, l := range sec.Lines {
			if l.NonKas {
				continue // baris keterangan Rp 0; tidak ada uang yang bergerak
			}
			s.tambah(GayaBaris, kode+":"+l.Label, l.Label, l.Amount)
			for _, d := range l.Detail {
				s.tambah(GayaSub, kode+":"+l.Label+":"+d.Label, d.Label, d.Amount)
			}
		}
		s.tambah(GayaTotal, kode+"_bersih", total, sec.Net)
	}
	bagian("operasi", "A. Aktivitas operasi", "Kas bersih dari aktivitas operasi", operasi)
	bagian("investasi", "B. Aktivitas investasi", "Kas bersih dari aktivitas investasi", cf.Investasi)
	bagian("pendanaan", "C. Aktivitas pendanaan", "Kas bersih dari aktivitas pendanaan", cf.Pendanaan)
	s.tambah(GayaTotal, "kenaikan_kas", "Kenaikan kas bersih (A + B + C)", cf.NetCash)
	s.agregat = AgregatAwal
	s.tambah(GayaBaris, "saldo_awal", "Saldo kas awal periode", cf.OpeningBalance)
	s.agregat = AgregatAkhir
	s.tambah(GayaTotal, "saldo_akhir", "Saldo kas akhir periode", cf.ClosingBalance)
	return s.b
}

// --- Perbandingan multi-periode ---

type PeriodeLaporan struct {
	Periode string
	Bagian  []BagianLaporan // urutan bagian sama untuk setiap periode
}

type KolomBanding struct {
	Periode string `json:"periode"`
	Label   string `json:"label"`
}

type BarisBanding struct {
	Kunci   string  `json:"kunci"`
	Label   string  `json:"label"`
	Gaya    string  `json:"gaya"`
	Agregat string  `json:"agregat,omitempty"`
	Nilai   []int64 `json:"nilai"` // satu per kolom
	Total   int64   `json:"total"`
}

type BagianBanding struct {
	Kode  string         `json:"kode"`
	Judul string         `json:"judul"`
	Baris []BarisBanding `json:"baris"`
}

type Banding struct {
	Kolom  []KolomBanding  `json:"kolom"`
	Bagian []BagianBanding `json:"bagian"`
}

// MaksPeriodeBanding: satu tahun penuh; lebih dari itu tabel PDF tidak terbaca.
const MaksPeriodeBanding = 12

var polaPeriode = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

func PeriodeSah(p string) bool { return polaPeriode.MatchString(p) }

// LabelPeriode: "2026-08" → "Agustus 2026".
func LabelPeriode(p string) string {
	if !PeriodeSah(p) {
		return p
	}
	var bulan int
	fmt.Sscanf(p[5:], "%d", &bulan)
	return bulanID[bulan-1] + " " + p[:4]
}

// ValidasiPeriodeBanding mengurai daftar "YYYY-MM" dipisah koma, menolak yang salah format
// atau ganda, lalu mengurutkannya dari yang paling awal.
func ValidasiPeriodeBanding(daftar string) ([]string, error) {
	var hasil []string
	for _, p := range strings.Split(daftar, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !PeriodeSah(p) {
			return nil, fmt.Errorf("periode %q tidak dikenal (format YYYY-MM)", p)
		}
		if slices.Contains(hasil, p) {
			return nil, fmt.Errorf("periode %s dipilih dua kali", p)
		}
		hasil = append(hasil, p)
	}
	switch {
	case len(hasil) == 0:
		return nil, errors.New("pilih minimal satu bulan")
	case len(hasil) > MaksPeriodeBanding:
		return nil, fmt.Errorf("maksimal %d bulan sekaligus", MaksPeriodeBanding)
	}
	slices.Sort(hasil)
	return hasil, nil
}

// SusunBanding menjajarkan laporan beberapa periode baris demi baris. Baris yang hanya ada
// di sebagian periode (mis. beban baru) tetap muncul, bernilai 0 di periode lain, dan
// disisipkan setelah baris yang mendahuluinya. Kolom Total mengikuti Agregat tiap baris.
func SusunBanding(laporan []PeriodeLaporan) Banding {
	laporan = slices.Clone(laporan)
	slices.SortFunc(laporan, func(a, b PeriodeLaporan) int { return strings.Compare(a.Periode, b.Periode) })
	var out Banding
	for _, l := range laporan {
		out.Kolom = append(out.Kolom, KolomBanding{Periode: l.Periode, Label: LabelPeriode(l.Periode)})
	}
	if len(laporan) == 0 {
		return out
	}
	n := len(laporan)
	for bi, contoh := range laporan[0].Bagian {
		bg := BagianBanding{Kode: contoh.Kode, Judul: contoh.Judul}
		for ki, l := range laporan {
			if bi >= len(l.Bagian) {
				continue
			}
			sebelum := -1
			for _, r := range l.Bagian[bi].Baris {
				i := slices.IndexFunc(bg.Baris, func(x BarisBanding) bool { return x.Kunci == r.Kunci })
				if i < 0 {
					i = sebelum + 1
					bg.Baris = slices.Insert(bg.Baris, i, BarisBanding{
						Kunci: r.Kunci, Label: r.Label, Gaya: r.Gaya, Agregat: r.Agregat, Nilai: make([]int64, n)})
				}
				bg.Baris[i].Nilai[ki] = r.Nilai
				sebelum = i
			}
		}
		for i := range bg.Baris {
			r := &bg.Baris[i]
			switch r.Agregat {
			case AgregatJumlah:
				for _, v := range r.Nilai {
					r.Total += v
				}
			case AgregatAwal:
				r.Total = r.Nilai[0]
			case AgregatAkhir:
				r.Total = r.Nilai[n-1]
			}
		}
		out.Bagian = append(out.Bagian, bg)
	}
	return out
}
