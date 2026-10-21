package coa

import "slices"

// Peran akun: jurnal otomatis & laporan mencari akun lewat perannya, bukan kode tetap, supaya
// setiap jenis usaha boleh memakai CoA dengan penomoran sendiri (mis. CoA manufaktur: 1102 =
// Kas Besar, bank di 1111). Setiap template memberi tahu akun mana yang memegang tiap peran
// (Account.Peran); template lama tanpa kolom peran memakai kode bawaan CoA dagang.

const (
	PeranKas              = "kas"
	PeranBank             = "bank"
	PeranMarketplace      = "saldo_marketplace"
	PeranPiutang          = "piutang"
	PeranPersediaan       = "persediaan"
	PeranDibayarDimuka    = "dibayar_dimuka"
	PeranPeralatan        = "aset_peralatan"
	PeranAkumPeralatan    = "akum_peralatan"
	PeranGedung           = "aset_gedung"
	PeranAkumGedung       = "akum_gedung"
	PeranKendaraan        = "aset_kendaraan"
	PeranAkumKendaraan    = "akum_kendaraan"
	PeranTakBerwujud      = "aset_tak_berwujud"
	PeranAkumAmortisasi   = "akum_amortisasi"
	PeranUtangUsaha       = "utang_usaha"
	PeranPinjaman         = "pinjaman"
	PeranUtangLain        = "utang_lain"
	PeranModal            = "modal"
	PeranPrive            = "prive"
	PeranLabaDitahan      = "laba_ditahan"
	PeranPenjualan        = "penjualan"
	PeranHPP              = "hpp"
	PeranPembelian        = "pembelian"
	PeranKemasan          = "kemasan"
	PeranBebanPenyusutan  = "beban_penyusutan"
	PeranBebanAmortisasi  = "beban_amortisasi"
	PeranPendapatanLain   = "pendapatan_lain"
	PeranBebanPajak       = "beban_pajak"
	PeranPotonganAplikasi = "beban_marketplace"
	PeranOngkir           = "beban_ongkir"
	PeranGaji             = "beban_gaji"
	PeranSewa             = "beban_sewa"
	PeranListrik          = "beban_listrik"
	PeranIklan            = "beban_iklan"
	PeranPerawatan        = "beban_perawatan"
	PeranBiayaLain        = "beban_lain"
)

// PeranInfo: peran beserta kode bawaannya di CoA dagang (Master) dan penjelasannya untuk Panel Master.
type PeranInfo struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Bawaan string `json:"bawaan"`
}

// DaftarPeran: semua peran yang dipakai jurnal otomatis & laporan; semuanya wajib terisi di
// template yang diterbitkan.
var DaftarPeran = []PeranInfo{
	{PeranKas, "Kas", "1101"},
	{PeranBank, "Bank", "1102"},
	{PeranMarketplace, "Saldo marketplace / payment gateway", "1104"},
	{PeranPiutang, "Piutang usaha", "1201"},
	{PeranPersediaan, "Persediaan", "1301"},
	{PeranDibayarDimuka, "Biaya dibayar di muka (saldo awal aset lancar lain)", "1401"},
	{PeranPeralatan, "Aset tetap: peralatan & mesin", "1501"},
	{PeranAkumPeralatan, "Akumulasi penyusutan peralatan", "1502"},
	{PeranGedung, "Aset tetap: bangunan", "1503"},
	{PeranAkumGedung, "Akumulasi penyusutan bangunan", "1504"},
	{PeranKendaraan, "Aset tetap: kendaraan", "1505"},
	{PeranAkumKendaraan, "Akumulasi penyusutan kendaraan", "1506"},
	{PeranTakBerwujud, "Harta tak berwujud", "1507"},
	{PeranAkumAmortisasi, "Akumulasi amortisasi", "1508"},
	{PeranUtangUsaha, "Utang usaha", "2101"},
	{PeranPinjaman, "Pinjaman / utang bank", "2102"},
	{PeranUtangLain, "Utang lain-lain (saldo awal kewajiban lain)", "2105"},
	{PeranModal, "Modal pemilik / disetor", "3101"},
	{PeranPrive, "Prive / dividen", "3102"},
	{PeranLabaDitahan, "Laba ditahan", "3201"},
	{PeranPenjualan, "Penjualan", "4101"},
	{PeranHPP, "Harga pokok penjualan (perpetual)", "5101"},
	{PeranPembelian, "Pembelian / pemakaian bahan (periodik)", "5102"},
	{PeranKemasan, "Beban kemasan", "5104"},
	{PeranBebanPenyusutan, "Beban penyusutan", "6113"},
	{PeranBebanAmortisasi, "Beban amortisasi", "6118"},
	{PeranPendapatanLain, "Pendapatan lain-lain", "8101"},
	{PeranBebanPajak, "Beban pajak & perizinan", "9103"},
	{PeranPotonganAplikasi, "Beban marketplace & payment gateway", "6108"},
	{PeranOngkir, "Beban transportasi & ongkir", "6106"},
	{PeranGaji, "Beban gaji & upah", "6101"},
	{PeranSewa, "Beban sewa", "6104"},
	{PeranListrik, "Beban listrik, air & internet", "6105"},
	{PeranIklan, "Beban iklan & promosi", "6102"},
	{PeranPerawatan, "Beban pemeliharaan & perlengkapan", "6112"},
	{PeranBiayaLain, "Beban operasional lain-lain", "6190"},
}

// Peta: peran → kode akun untuk satu template.
type Peta map[string]string

// Kode: kode akun pemegang peran ("" bila tidak ada).
func (p Peta) Kode(peran string) string { return p[peran] }

// PetaDari menyusun peta peran dari daftar akun: akun yang bertanda Peran dipakai; peran yang
// belum dipegang siapa pun jatuh ke kode bawaan CoA dagang bila kode itu ada di daftar.
func PetaDari(accs []Account) Peta {
	p := Peta{}
	for _, a := range accs {
		if a.Peran != "" {
			if _, sudah := p[a.Peran]; !sudah {
				p[a.Peran] = a.Kode
			}
		}
	}
	for _, r := range DaftarPeran {
		if _, ada := p[r.ID]; ada {
			continue
		}
		if slices.ContainsFunc(accs, func(a Account) bool { return a.Kode == r.Bawaan }) {
			p[r.ID] = r.Bawaan
		}
	}
	return p
}

// PetaMaster: peta CoA dagang bawaan.
var PetaMaster = PetaDari(Master)

// PeranKurang: peran wajib yang tidak dipegang akun mana pun di daftar.
func PeranKurang(accs []Account) []string {
	p := PetaDari(accs)
	var out []string
	for _, r := range DaftarPeran {
		if p[r.ID] == "" {
			out = append(out, r.ID)
		}
	}
	return out
}
