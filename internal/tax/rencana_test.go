package tax

import (
	"errors"
	"testing"
)

func TestHitungNPPN_NormaKLUWilayah(t *testing.T) {
	r, err := HitungNPPN(NPPNInput{Omzet: 1_200_000_000, KLU: "01111", Wilayah: WilayahIbukota10, PTKP: "K/1"})
	if err != nil {
		t.Fatal(err)
	}
	// 15% x 1,2 M = 180 jt; - PTKP 63 jt = 117 jt; 60 jt x 5% + 57 jt x 15% = 11,55 jt
	if r.NormaBps != 1500 || r.Neto != 180_000_000 || r.PKP != 117_000_000 || r.PPh != 11_550_000 {
		t.Fatalf("got %+v", r)
	}
	if r.PPhFinal != 3_500_000 || !r.Boleh { // (1,2 M - 500 jt) x 0,5%
		t.Fatalf("pembanding PPh final %d, boleh %v", r.PPhFinal, r.Boleh)
	}
	if len(r.Langkah) == 0 {
		t.Error("langkah kosong")
	}
}

func TestHitungNPPN_NormaManualDanTidakTersedia(t *testing.T) {
	r, err := HitungNPPN(NPPNInput{Omzet: 1_000_000_000, NormaBps: 2000, PTKP: "TK/0"})
	if err != nil || r.Neto != 200_000_000 || r.Norma != nil {
		t.Fatalf("norma manual: %+v %v", r, err)
	}
	if _, err := HitungNPPN(NPPNInput{Omzet: 1_000_000_000, KLU: "01113", Wilayah: WilayahIbukotaLain, PTKP: "TK/0"}); !errors.Is(err, ErrNormaTidakAda) {
		t.Errorf("sel kosong di tabel norma harus ditolak, bukan ditebak: %v", err)
	}
	if _, err := HitungNPPN(NPPNInput{Omzet: 1_000_000_000, KLU: "99999", Wilayah: WilayahIbukota10, PTKP: "TK/0"}); !errors.Is(err, ErrNormaTidakAda) {
		t.Errorf("KLU tak dikenal: %v", err)
	}
	r, _ = HitungNPPN(NPPNInput{Omzet: 5_000_000_000, NormaBps: 2000, PTKP: "TK/0"})
	if r.Boleh {
		t.Error("omzet di atas Rp4,8 M tidak boleh memakai norma")
	}
}

// cekGrossUp: nilai kontrak terkecil yang membuat vendor menerima target, dekat sampel workbook.
func cekGrossUp(t *testing.T, in GrossUpInput, kira int64) GrossUpResult {
	t.Helper()
	r, err := HitungGrossUp(in)
	if err != nil {
		t.Fatal(err)
	}
	if r.Diterima < in.Diterima || r.Diterima-in.Diterima > 3 {
		t.Errorf("diterima %d, target %d", r.Diterima, in.Diterima)
	}
	if r.NilaiKontrak-r.Pajak != r.Diterima {
		t.Errorf("kontrak %d - pajak %d != diterima %d", r.NilaiKontrak, r.Pajak, r.Diterima)
	}
	// minimal: satu rupiah lebih kecil sudah di bawah target
	if d := r.NilaiKontrak - 1 - pajakKontrak(t, in, r.NilaiKontrak-1); d >= in.Diterima {
		t.Errorf("nilai kontrak %d bukan yang terkecil", r.NilaiKontrak)
	}
	if diff := r.NilaiKontrak - kira; diff < -2 || diff > 2 {
		t.Errorf("nilai kontrak %d, sampel workbook %d", r.NilaiKontrak, kira)
	}
	return r
}

func pajakKontrak(t *testing.T, in GrossUpInput, k int64) int64 {
	t.Helper()
	p, err := pajakGrossUp(in, k)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestHitungGrossUp_SampelWorkbook(t *testing.T) {
	// Badan Gross Up: jasa 2% tanpa NPWP (x2) → 200 jt / 96% = 208.333.333
	cekGrossUp(t, GrossUpInput{Jenis: "pph23", Objek: "23-jasa", Diterima: 200_000_000}, 208_333_333)
	// Sewa tanah & bangunan 10% final → 20 jt / 90%
	cekGrossUp(t, GrossUpInput{Jenis: "pph4-2", Objek: "42-sewa-tb", Diterima: 20_000_000, PunyaNPWP: true}, 22_222_222)
	// OP Gross Up: PPh 21 bukan pegawai, DPP 50%, tarif Pasal 17, tanpa NPWP 120%
	r := cekGrossUp(t, GrossUpInput{Jenis: JenisPPh21BukanPegawai, Diterima: 500_000_000}, 544_470_588)
	if r.DPP != r.NilaiKontrak/2 {
		t.Errorf("DPP bukan pegawai %d", r.DPP)
	}
}

func TestHitungGrossUp_TolakPPNDanAngkaNgawur(t *testing.T) {
	for _, in := range []GrossUpInput{
		{Jenis: "ppn", Objek: "ppn-umum", Diterima: 1_000_000},
		{Jenis: "pph23", Objek: "23-tidak-ada", Diterima: 1_000_000},
		{Jenis: "pph23", Objek: "23-jasa", Diterima: -1},
	} {
		if _, err := HitungGrossUp(in); err == nil {
			t.Errorf("%+v diterima", in)
		}
	}
}

func skemaPPN(t *testing.T, r PPNHargaResult, id string) SkemaPPN {
	t.Helper()
	for _, s := range r.Skema {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("skema %s tidak ada", id)
	return SkemaPPN{}
}

func TestHitungPPNHarga_SampelPricingPPN(t *testing.T) {
	// Sheet "Simulasi Pricing PPN": pembelian 1 M belum PPN, 100% dari PKP, markup 10%, opex 2%
	r, err := HitungPPNHarga(PPNHargaInput{Pembelian: 1_000_000_000, PorsiDariPKPBps: 10_000, MarkupBps: 1000, OpexBps: 200})
	if err != nil {
		t.Fatal(err)
	}
	non := skemaPPN(t, r, SkemaNonPKP)
	if non.HPP != 1_110_000_000 || non.HargaPembeli != 1_221_000_000 || non.Opex != 24_420_000 || non.Laba != 86_580_000 || non.PPNSetor != 0 {
		t.Errorf("non-PKP %+v", non)
	}
	beban := skemaPPN(t, r, SkemaPKPDibebankan)
	if beban.DPP != 1_100_000_000 || beban.PPNKeluaran != 121_000_000 || beban.PPNMasukan != 110_000_000 || beban.PPNSetor != 11_000_000 || beban.Laba != 78_000_000 || beban.HargaPembeli != 1_221_000_000 {
		t.Errorf("PKP dibebankan %+v", beban)
	}
	tanggung := skemaPPN(t, r, SkemaPKPDitanggung)
	// harga ke pembeli 1,1 M sudah termasuk PPN 11%: DPP = 1,1 M / 1,11
	if tanggung.HargaPembeli != 1_100_000_000 || tanggung.DPP != 990_990_990 || tanggung.PPNKeluaran != 109_009_010 || tanggung.Laba != 990_990_990-1_000_000_000-19_819_819 {
		t.Errorf("PKP ditanggung %+v", tanggung)
	}
}

func TestHitungPPNHarga_PembelianTermasukPPN(t *testing.T) {
	// 1,11 M sudah termasuk PPN, separuh dari PKP: PPN masukan = 555 jt x 11/111 = 55 jt
	r, err := HitungPPNHarga(PPNHargaInput{Pembelian: 1_110_000_000, TermasukPPN: true, PorsiDariPKPBps: 5000, MarkupBps: 0, OpexNominal: 1_000_000})
	if err != nil {
		t.Fatal(err)
	}
	b := skemaPPN(t, r, SkemaPKPDibebankan)
	if b.PPNMasukan != 55_000_000 || b.HPP != 1_055_000_000 || b.Opex != 1_000_000 {
		t.Errorf("%+v", b)
	}
}

func TestHitungBeliSewa_SampelWorkbook(t *testing.T) {
	r, err := HitungBeliSewa(BeliSewaInput{Bentuk: "badan", Omzet: 1_000_000_000, BiayaLain: 400_000_000,
		HargaAset: 10_000_000, UmurTahun: 4, Metode: MetodeGarisLurus, SewaPerTahun: 4_000_000})
	if err != nil {
		t.Fatal(err)
	}
	// tahun pertama sama dengan workbook: PPh 31E 65.725.000 (beli) vs 65.560.000 (sewa)
	if r.Beli.Tahun[0].PPh != 65_725_000 || r.Sewa.Tahun[0].PPh != 65_560_000 || r.PotongSewa != 80_000 {
		t.Fatalf("tahun 1: beli %d sewa %d potong %d", r.Beli.Tahun[0].PPh, r.Sewa.Tahun[0].PPh, r.PotongSewa)
	}
	// selama umur aset: beli 10 jt + 4 x 65.725.000; sewa 4 x 4 jt + 4 x 65.560.000
	if r.Beli.TotalKas != 272_900_000 || r.Sewa.TotalKas != 278_240_000 || r.Rekomendasi != "beli" {
		t.Fatalf("total beli %d sewa %d → %s", r.Beli.TotalKas, r.Sewa.TotalKas, r.Rekomendasi)
	}
}

func TestHitungBeliSewa_SaldoMenurunHabisDiTahunTerakhir(t *testing.T) {
	r, err := HitungBeliSewa(BeliSewaInput{Bentuk: "badan", Omzet: 2_000_000_000, BiayaLain: 1_000_000_000,
		HargaAset: 100_000_000, UmurTahun: 8, Metode: MetodeSaldoMenurun, SewaPerTahun: 20_000_000})
	if err != nil {
		t.Fatal(err)
	}
	want := []int64{25_000_000, 18_750_000, 14_062_500, 10_546_875, 7_910_156, 5_932_617, 4_449_463, 13_348_389}
	var total int64
	for i, y := range r.Beli.Tahun {
		if y.Beban != want[i] {
			t.Errorf("tahun %d penyusutan %d, want %d", i+1, y.Beban, want[i])
		}
		total += y.Beban
	}
	if total != 100_000_000 {
		t.Errorf("total penyusutan %d", total)
	}
}

func TestHitungBeliSewa_BangunanGarisLurusDanPotong10Persen(t *testing.T) {
	r, err := HitungBeliSewa(BeliSewaInput{Bentuk: "op", PTKP: "TK/0", Omzet: 6_000_000_000, BiayaLain: 5_000_000_000,
		HargaAset: 2_000_000_000, UmurTahun: 20, Metode: MetodeSaldoMenurun, Bangunan: true, SewaPerTahun: 150_000_000})
	if err != nil {
		t.Fatal(err)
	}
	if r.Metode != MetodeGarisLurus || r.Beli.Tahun[0].Beban != 100_000_000 || r.PotongSewa != 15_000_000 {
		t.Fatalf("metode %s, penyusutan %d, potong %d", r.Metode, r.Beli.Tahun[0].Beban, r.PotongSewa)
	}
	// OP: Pasal 17 atas laba - PTKP. Beli: 1 M - 100 jt - 54 jt = 846 jt
	if r.Beli.Tahun[0].PPh != Pasal17(846_000_000) {
		t.Errorf("PPh OP %d", r.Beli.Tahun[0].PPh)
	}
}
