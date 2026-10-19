package tax

import (
	"errors"
	"fmt"
	"strings"
)

// Kalkulator pajak: PPh 21, PPh 23, PPh 4 ayat (2), PPh Final UMKM, dan PPN.
//
// Semua tarif ditulis dalam basis poin (bps): 1100 = 11%, 50 = 0,5%. Nilai Rupiah selalu
// bilangan bulat; pembulatan ke bawah mengikuti kebiasaan pemotongan pajak.

const (
	// BiayaJabatanBps: 5% dari penghasilan bruto, dibatasi BiayaJabatanMaxTahun.
	BiayaJabatanBps      = 500
	BiayaJabatanMaxTahun = 6_000_000 // Rp 500.000/bulan

	// TanpaNPWPTambahanBps: pemotongan 20% lebih tinggi untuk PPh 21 tanpa NPWP,
	// dan 100% lebih tinggi untuk PPh 23.
	TanpaNPWPPPh21Bps = 12_000 // 120% dari tarif normal
	TanpaNPWPPPh23Bps = 20_000 // 200% dari tarif normal
)

// ===== PPh 21 =====

type PPh21Input struct {
	// PenghasilanBruto per bulan: gaji pokok + tunjangan tetap.
	PenghasilanBruto int64  `json:"penghasilan_bruto"`
	PTKPStatus       string `json:"ptkp_status"`
	// IuranPensiun per bulan yang dibayar karyawan; mengurangi penghasilan neto.
	IuranPensiun int64 `json:"iuran_pensiun"`
	// GrossUp: pajaknya ditanggung perusahaan dan ikut jadi penghasilan karyawan.
	GrossUp bool `json:"gross_up"`
	// PunyaNPWP: tanpa NPWP dipotong 20% lebih tinggi.
	PunyaNPWP bool `json:"punya_npwp"`
}

type PPh21Result struct {
	BrutoSetahun      int64 `json:"bruto_setahun"`
	BiayaJabatan      int64 `json:"biaya_jabatan"`
	IuranPensiun      int64 `json:"iuran_pensiun_setahun"`
	NetoSetahun       int64 `json:"neto_setahun"`
	PTKP              int64 `json:"ptkp"`
	PKP               int64 `json:"pkp"` // dibulatkan ke bawah ke ribuan penuh
	PPhSetahun        int64 `json:"pph_setahun"`
	PPhSebulan        int64 `json:"pph_sebulan"`
	TambahanTanpaNPWP int64 `json:"tambahan_tanpa_npwp"`
	// TunjanganPajak terisi pada skema gross up: jumlah yang ditambahkan ke penghasilan.
	TunjanganPajak  int64     `json:"tunjangan_pajak"`
	TakeHomeSebulan int64     `json:"take_home_sebulan"`
	Langkah         []Langkah `json:"langkah"`
}

// Langkah menjelaskan perhitungan baris demi baris, supaya hasilnya bisa ditelusuri pengguna
// dan tidak tampak seperti angka yang muncul begitu saja.
type Langkah struct {
	Label      string `json:"label"`
	Nilai      int64  `json:"nilai"`
	Keterangan string `json:"keterangan,omitempty"`
}

// HitungPPh21 memakai metode tahunan Pasal 17: bruto setahun dikurangi biaya jabatan, iuran
// pensiun, dan PTKP, lalu dikenakan tarif progresif; sebulan = setahun dibagi 12.
//
// ponytail: TER bulanan (PMK 168/2023) belum dipakai karena Lampiran tarifnya belum
// diverifikasi baris per baris — lihat TERBelumLengkap. Metode tahunan ini yang dipakai untuk
// penghitungan ulang Desember, jadi totalnya setahun tetap benar; yang bisa berbeda hanya
// pembagian beban antar bulan.
func HitungPPh21(in PPh21Input) (PPh21Result, error) {
	ptkp, ok := PTKP[strings.TrimSpace(in.PTKPStatus)]
	if !ok {
		return PPh21Result{}, fmt.Errorf("status PTKP %q tidak dikenal", in.PTKPStatus)
	}
	if in.PenghasilanBruto < 0 || in.IuranPensiun < 0 {
		return PPh21Result{}, errors.New("penghasilan dan iuran tidak boleh negatif")
	}

	hitung := func(brutoBulanan int64) PPh21Result {
		var r PPh21Result
		r.PTKP = ptkp
		r.BrutoSetahun = brutoBulanan * 12
		r.BiayaJabatan = minI64(r.BrutoSetahun*BiayaJabatanBps/10_000, BiayaJabatanMaxTahun)
		r.IuranPensiun = in.IuranPensiun * 12
		r.NetoSetahun = r.BrutoSetahun - r.BiayaJabatan - r.IuranPensiun
		pkp := r.NetoSetahun - ptkp
		if pkp < 0 {
			pkp = 0
		}
		r.PKP = pkp / 1000 * 1000 // PKP dibulatkan ke bawah ke ribuan penuh
		r.PPhSetahun = Pasal17(r.PKP)
		if !in.PunyaNPWP && r.PPhSetahun > 0 {
			normal := r.PPhSetahun
			r.PPhSetahun = normal * TanpaNPWPPPh21Bps / 10_000
			r.TambahanTanpaNPWP = r.PPhSetahun - normal
		}
		r.PPhSebulan = r.PPhSetahun / 12
		return r
	}

	r := hitung(in.PenghasilanBruto)

	if in.GrossUp {
		// Gross up: tunjangan pajak ikut menambah penghasilan, yang menaikkan pajaknya lagi.
		// Diselesaikan dengan iterasi sampai berhenti berubah — lebih sederhana dan lebih
		// tahan terhadap perubahan tarif daripada rumus lapisan yang harus ditulis ulang
		// setiap kali tarif berganti. Konvergen dalam beberapa putaran karena tarif < 100%.
		tunjangan := int64(0)
		for i := 0; i < 40; i++ {
			next := hitung(in.PenghasilanBruto + tunjangan/12)
			if next.PPhSetahun == tunjangan {
				break
			}
			tunjangan = next.PPhSetahun
		}
		r = hitung(in.PenghasilanBruto + tunjangan/12)
		r.TunjanganPajak = r.PPhSetahun
	}

	bruto := in.PenghasilanBruto
	if in.GrossUp {
		bruto += r.TunjanganPajak / 12
	}
	r.TakeHomeSebulan = bruto - in.IuranPensiun - r.PPhSebulan
	if in.GrossUp {
		// Pada gross up pajak ditanggung perusahaan, jadi karyawan menerima penuh.
		r.TakeHomeSebulan = in.PenghasilanBruto - in.IuranPensiun
	}

	r.Langkah = []Langkah{
		{"Penghasilan bruto setahun", r.BrutoSetahun, "penghasilan sebulan x 12"},
		{"Biaya jabatan", -r.BiayaJabatan, "5% dari bruto, maksimal Rp 6.000.000 setahun"},
		{"Iuran pensiun setahun", -r.IuranPensiun, ""},
		{"Penghasilan neto setahun", r.NetoSetahun, ""},
		{"PTKP " + in.PTKPStatus, -r.PTKP, ""},
		{"Penghasilan kena pajak", r.PKP, "dibulatkan ke bawah ke ribuan penuh"},
		{"PPh 21 setahun", r.PPhSetahun, tarifKeterangan(in.PunyaNPWP)},
		{"PPh 21 sebulan", r.PPhSebulan, "setahun dibagi 12"},
	}
	return r, nil
}

func tarifKeterangan(punyaNPWP bool) string {
	if punyaNPWP {
		return "tarif progresif Pasal 17"
	}
	return "tarif Pasal 17 + 20% karena tanpa NPWP"
}

// ===== TER (PMK 168/2023) =====

// TERKategori memetakan status PTKP ke kategori tarif efektif bulanan.
var TERKategori = map[string]string{
	"TK/0": "A", "TK/1": "A", "K/0": "A",
	"TK/2": "B", "TK/3": "B", "K/1": "B", "K/2": "B",
	"K/3": "C",
}

// === SATU-SATUNYA TEMPAT MELENGKAPI TABEL TER ===
//
// Lapisan di bawah ini BELUM LENGKAP. Lampiran PMK 168/2023 memuat 44 lapis untuk kategori A
// serta sekitar 40 lapis untuk B dan C; yang tercantum di sini hanya baris yang sudah
// terverifikasi. Sengaja tidak ditebak: tarif pajak yang salah lebih berbahaya daripada
// kalkulator yang berkata "belum bisa dihitung".
//
// Cara melengkapi:
//  1. Buka Lampiran PMK 168/2023 bagian Tarif Efektif Bulanan.
//  2. Tambahkan baris {BatasBawah, BatasAtas, TarifBps} berurutan menaik. BatasAtas -1 = tanpa batas.
//  3. Tarif dalam basis poin: 0,25% = 25, 5% = 500, 34% = 3400.
//  4. Setelah satu kategori utuh dari 0 sampai tanpa batas, HitungTER otomatis memakainya.
var TERLapisan = map[string][]TERBaris{
	"A": {
		{0, 5_400_000, 0},
		{5_400_001, 5_650_000, 25},
		{5_650_001, 5_950_000, 50},
		{5_950_001, 6_300_000, 75},
		{6_300_001, 6_750_000, 100},
	},
	"B": {
		{0, 6_200_000, 0},
		{6_200_001, 6_500_000, 25},
	},
	"C": {
		{0, 6_600_000, 0},
		{6_600_001, 6_950_000, 25},
	},
}

type TERBaris struct {
	BatasBawah int64 `json:"batas_bawah"`
	BatasAtas  int64 `json:"batas_atas"` // -1 = tanpa batas atas
	TarifBps   int   `json:"tarif_bps"`
}

// TERBelumLengkap melaporkan kategori yang lapisannya belum mencapai tanpa batas, supaya UI
// bisa menyatakannya terang-terangan dan tidak menyajikan angka yang keliru.
func TERBelumLengkap() []string {
	var out []string
	for _, k := range []string{"A", "B", "C"} {
		l := TERLapisan[k]
		if len(l) == 0 || l[len(l)-1].BatasAtas != -1 {
			out = append(out, k)
		}
	}
	return out
}

var ErrTERBelumLengkap = errors.New("lapisan TER belum lengkap untuk penghasilan sebesar ini; lengkapi TERLapisan dari Lampiran PMK 168/2023")

// HitungTER mengembalikan tarif efektif bulanan dan PPh 21 sebulan menurut TER.
// Bila penghasilan jatuh di luar lapisan yang sudah terverifikasi, fungsi ini menolak
// menghitung alih-alih menebak.
func HitungTER(brutoBulanan int64, ptkpStatus string) (tarifBps int, pph int64, err error) {
	kat, ok := TERKategori[strings.TrimSpace(ptkpStatus)]
	if !ok {
		return 0, 0, fmt.Errorf("status PTKP %q tidak dikenal", ptkpStatus)
	}
	for _, b := range TERLapisan[kat] {
		if brutoBulanan >= b.BatasBawah && (b.BatasAtas < 0 || brutoBulanan <= b.BatasAtas) {
			return b.TarifBps, brutoBulanan * int64(b.TarifBps) / 10_000, nil
		}
	}
	return 0, 0, ErrTERBelumLengkap
}

func minI64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
