package tax_test

import (
	"errors"
	"testing"

	"pajak/internal/tax"
)

// Contoh baku: gaji Rp 10 juta/bulan, TK/0, tanpa iuran pensiun.
// Bruto setahun 120.000.000; biaya jabatan 5% = 6.000.000 (kena batas);
// neto 114.000.000; PTKP TK/0 54.000.000; PKP 60.000.000;
// Pasal 17: 60.000.000 x 5% = 3.000.000 setahun, 250.000 sebulan.
func TestHitungPPh21_ContohBaku(t *testing.T) {
	r, err := tax.HitungPPh21(tax.PPh21Input{PenghasilanBruto: 10_000_000, PTKPStatus: "TK/0", PunyaNPWP: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		nama      string
		got, want int64
	}{
		{"bruto setahun", r.BrutoSetahun, 120_000_000},
		{"biaya jabatan", r.BiayaJabatan, 6_000_000},
		{"neto setahun", r.NetoSetahun, 114_000_000},
		{"PTKP", r.PTKP, 54_000_000},
		{"PKP", r.PKP, 60_000_000},
		{"PPh setahun", r.PPhSetahun, 3_000_000},
		{"PPh sebulan", r.PPhSebulan, 250_000},
	} {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.nama, c.got, c.want)
		}
	}
}

// Biaya jabatan dibatasi Rp 6 juta setahun; gaji besar tidak boleh menembus batas itu.
func TestHitungPPh21_BiayaJabatanDibatasi(t *testing.T) {
	r, _ := tax.HitungPPh21(tax.PPh21Input{PenghasilanBruto: 50_000_000, PTKPStatus: "TK/0", PunyaNPWP: true})
	if r.BiayaJabatan != 6_000_000 {
		t.Errorf("biaya jabatan = %d, want dibatasi 6.000.000", r.BiayaJabatan)
	}
}

// Penghasilan di bawah PTKP tidak terutang pajak sama sekali.
func TestHitungPPh21_DibawahPTKPNol(t *testing.T) {
	r, _ := tax.HitungPPh21(tax.PPh21Input{PenghasilanBruto: 4_000_000, PTKPStatus: "TK/0", PunyaNPWP: true})
	if r.PKP != 0 || r.PPhSetahun != 0 {
		t.Errorf("di bawah PTKP: PKP %d, PPh %d, keduanya harus 0", r.PKP, r.PPhSetahun)
	}
}

// Tanpa NPWP dipotong 20% lebih tinggi.
func TestHitungPPh21_TanpaNPWPLebihTinggi(t *testing.T) {
	ber, _ := tax.HitungPPh21(tax.PPh21Input{PenghasilanBruto: 10_000_000, PTKPStatus: "TK/0", PunyaNPWP: true})
	tanpa, _ := tax.HitungPPh21(tax.PPh21Input{PenghasilanBruto: 10_000_000, PTKPStatus: "TK/0"})
	if want := ber.PPhSetahun * 120 / 100; tanpa.PPhSetahun != want {
		t.Errorf("tanpa NPWP = %d, want %d (120%% dari %d)", tanpa.PPhSetahun, want, ber.PPhSetahun)
	}
	if tanpa.TambahanTanpaNPWP != tanpa.PPhSetahun-ber.PPhSetahun {
		t.Errorf("tambahan tanpa NPWP tidak konsisten: %d", tanpa.TambahanTanpaNPWP)
	}
}

// Gross up: tunjangan pajak harus tepat menutup pajaknya sendiri, dan karyawan menerima penuh.
func TestHitungPPh21_GrossUpMenutupPajaknyaSendiri(t *testing.T) {
	r, err := tax.HitungPPh21(tax.PPh21Input{PenghasilanBruto: 10_000_000, PTKPStatus: "TK/0", PunyaNPWP: true, GrossUp: true})
	if err != nil {
		t.Fatal(err)
	}
	if r.TunjanganPajak != r.PPhSetahun {
		t.Errorf("tunjangan pajak %d harus sama dengan PPh setahun %d", r.TunjanganPajak, r.PPhSetahun)
	}
	if r.TakeHomeSebulan != 10_000_000 {
		t.Errorf("pada gross up karyawan menerima penuh: take home %d, want 10.000.000", r.TakeHomeSebulan)
	}
	// Pajaknya lebih besar daripada skema gross karena tunjangan ikut jadi penghasilan.
	gross, _ := tax.HitungPPh21(tax.PPh21Input{PenghasilanBruto: 10_000_000, PTKPStatus: "TK/0", PunyaNPWP: true})
	if r.PPhSetahun <= gross.PPhSetahun {
		t.Errorf("gross up %d harus > gross %d", r.PPhSetahun, gross.PPhSetahun)
	}
}

func TestHitungPPh21_StatusPTKPTidakDikenal(t *testing.T) {
	if _, err := tax.HitungPPh21(tax.PPh21Input{PenghasilanBruto: 1, PTKPStatus: "X/9"}); err == nil {
		t.Error("status PTKP ngawur harus ditolak")
	}
}

// Iuran pensiun mengurangi penghasilan neto, jadi pajaknya turun.
func TestHitungPPh21_IuranPensiunMengurangi(t *testing.T) {
	tanpa, _ := tax.HitungPPh21(tax.PPh21Input{PenghasilanBruto: 10_000_000, PTKPStatus: "TK/0", PunyaNPWP: true})
	dengan, _ := tax.HitungPPh21(tax.PPh21Input{PenghasilanBruto: 10_000_000, PTKPStatus: "TK/0", PunyaNPWP: true, IuranPensiun: 200_000})
	if dengan.PPhSetahun >= tanpa.PPhSetahun {
		t.Errorf("iuran pensiun harus menurunkan pajak: %d vs %d", dengan.PPhSetahun, tanpa.PPhSetahun)
	}
	if dengan.NetoSetahun != tanpa.NetoSetahun-2_400_000 {
		t.Errorf("neto = %d, want berkurang 2.400.000", dengan.NetoSetahun)
	}
}

// === TER ===

// TER menolak menghitung di luar lapisan yang sudah terverifikasi, bukan menebak.
func TestHitungTER_MenolakDiLuarLapisanTerverifikasi(t *testing.T) {
	if _, pph, err := tax.HitungTER(5_000_000, "TK/0"); err != nil || pph != 0 {
		t.Errorf("di bawah batas 0%%: pph %d err %v", pph, err)
	}
	if bps, pph, err := tax.HitungTER(5_500_000, "TK/0"); err != nil || bps != 25 || pph != 13_750 {
		t.Errorf("lapisan 0,25%%: bps %d pph %d err %v", bps, pph, err)
	}
	// Gaji besar jatuh di lapisan yang belum diisi: harus menolak, bukan mengembalikan 0.
	if _, _, err := tax.HitungTER(50_000_000, "TK/0"); !errors.Is(err, tax.ErrTERBelumLengkap) {
		t.Errorf("lapisan belum lengkap harus ditolak terang-terangan, got %v", err)
	}
}

func TestTERBelumLengkap_DilaporkanKeUI(t *testing.T) {
	kurang := tax.TERBelumLengkap()
	if len(kurang) != 3 {
		t.Errorf("ketiga kategori belum lengkap, got %v", kurang)
	}
}

// === Potong-pungut ===

func TestHitungPotong_TarifDasarBenar(t *testing.T) {
	for _, c := range []struct {
		jenis, objek string
		bruto        int64
		npwp         bool
		wantPajak    int64
		wantDiterima int64
	}{
		{"pph23", "23-jasa", 10_000_000, true, 200_000, 9_800_000},                  // 2%
		{"pph23", "23-jasa", 10_000_000, false, 400_000, 9_600_000},                 // tanpa NPWP 2x
		{"pph23", "23-dividen", 10_000_000, true, 1_500_000, 8_500_000},             // 15%
		{"pph4-2", "42-sewa-tb", 50_000_000, true, 5_000_000, 45_000_000},           // 10% final
		{"pph4-2", "42-umkm", 100_000_000, true, 500_000, 99_500_000},               // 0,5%
		{"pph4-2", "42-konstruksi-kecil", 100_000_000, true, 1_750_000, 98_250_000}, // 1,75%
		{"pph22", "22-impor-api", 100_000_000, true, 2_500_000, 97_500_000},         // 2,5%
		{"pph15", "15-pelayaran-dn", 100_000_000, true, 1_200_000, 98_800_000},      // 1,2%
	} {
		r, err := tax.HitungPotong(tax.PotongInput{Jenis: c.jenis, Objek: c.objek, NilaiBruto: c.bruto, PunyaNPWP: c.npwp})
		if err != nil {
			t.Errorf("%s/%s: %v", c.jenis, c.objek, err)
			continue
		}
		if r.Pajak != c.wantPajak {
			t.Errorf("%s/%s pajak = %d, want %d", c.jenis, c.objek, r.Pajak, c.wantPajak)
		}
		if r.Diterima != c.wantDiterima {
			t.Errorf("%s/%s diterima = %d, want %d", c.jenis, c.objek, r.Diterima, c.wantDiterima)
		}
	}
}

// PPN tarif umum: 12% atas DPP nilai lain 11/12, bebannya setara 11% dari harga jual.
func TestHitungPotong_PPNDPPNilaiLain(t *testing.T) {
	r, err := tax.HitungPotong(tax.PotongInput{Jenis: "ppn", Objek: "ppn-umum", NilaiBruto: 100_000_000, PunyaNPWP: true})
	if err != nil {
		t.Fatal(err)
	}
	// 100 juta x 91,66% = 91.660.000; x 12% = 10.999.200 — setara 11%.
	if r.Pajak < 10_900_000 || r.Pajak > 11_100_000 {
		t.Errorf("PPN = %d, seharusnya mendekati 11%% dari harga jual", r.Pajak)
	}
	if r.DPP >= r.NilaiBruto {
		t.Errorf("DPP nilai lain %d harus lebih kecil dari harga jual %d", r.DPP, r.NilaiBruto)
	}
}

// Tanpa NPWP hanya berlaku pada jenis yang mengenal aturan itu; PPh 4(2) final tidak.
func TestHitungPotong_TanpaNPWPTidakBerlakuDiFinal(t *testing.T) {
	a, _ := tax.HitungPotong(tax.PotongInput{Jenis: "pph4-2", Objek: "42-sewa-tb", NilaiBruto: 10_000_000, PunyaNPWP: true})
	b, _ := tax.HitungPotong(tax.PotongInput{Jenis: "pph4-2", Objek: "42-sewa-tb", NilaiBruto: 10_000_000})
	if a.Pajak != b.Pajak {
		t.Errorf("PPh 4(2) tidak mengenal kenaikan tanpa NPWP: %d vs %d", a.Pajak, b.Pajak)
	}
}

func TestHitungPotong_TolakMasukanNgawur(t *testing.T) {
	if _, err := tax.HitungPotong(tax.PotongInput{Jenis: "entah", Objek: "x", NilaiBruto: 1}); err == nil {
		t.Error("jenis pajak tak dikenal harus ditolak")
	}
	if _, err := tax.HitungPotong(tax.PotongInput{Jenis: "pph23", Objek: "42-sewa-tb", NilaiBruto: 1}); err == nil {
		t.Error("objek dari jenis lain harus ditolak")
	}
	if _, err := tax.HitungPotong(tax.PotongInput{Jenis: "pph23", Objek: "23-jasa", NilaiBruto: -1}); err == nil {
		t.Error("nilai negatif harus ditolak")
	}
}

// === PPh Badan ===

func TestHitungPPhBadan_PKPTidakMelebihiPeredaranBruto(t *testing.T) {
	if _, err := tax.HitungPPhBadan(tax.PPhBadanInput{PeredaranBruto: 1_000_000_000, PenghasilanKenaPajak: 1_000_000_001}); err == nil {
		t.Errorf("PKP > peredaran bruto harus ditolak: %v", err)
	}
	if _, err := tax.HitungPPhBadan(tax.PPhBadanInput{PeredaranBruto: 1_000_000_000, PenghasilanKenaPajak: 1_000_000_000}); err != nil {
		t.Errorf("PKP = peredaran bruto sah: %v", err)
	}
}

func TestHitungPPhBadan_FasilitasPasal31E(t *testing.T) {
	// Omzet 4 miliar (di bawah 4,8 M): seluruh PKP dapat tarif efektif 11%.
	kecil, _ := tax.HitungPPhBadan(tax.PPhBadanInput{PeredaranBruto: 4_000_000_000, PenghasilanKenaPajak: 500_000_000})
	if kecil.PPhTerutang != 55_000_000 {
		t.Errorf("omzet 4 M: PPh = %d, want 55.000.000 (11%% dari 500 juta)", kecil.PPhTerutang)
	}
	// Omzet 60 miliar (di atas 50 M): tanpa fasilitas, 22% penuh.
	besar, _ := tax.HitungPPhBadan(tax.PPhBadanInput{PeredaranBruto: 60_000_000_000, PenghasilanKenaPajak: 500_000_000})
	if besar.PPhTerutang != 110_000_000 {
		t.Errorf("omzet 60 M: PPh = %d, want 110.000.000 (22%% dari 500 juta)", besar.PPhTerutang)
	}
	if besar.DapatFasilitas {
		t.Error("omzet di atas 50 M tidak dapat fasilitas")
	}
	// Di antaranya: sebagian dapat fasilitas, hasilnya di antara keduanya.
	tengah, _ := tax.HitungPPhBadan(tax.PPhBadanInput{PeredaranBruto: 10_000_000_000, PenghasilanKenaPajak: 500_000_000})
	if tengah.PPhTerutang <= kecil.PPhTerutang || tengah.PPhTerutang >= besar.PPhTerutang {
		t.Errorf("omzet 10 M: PPh %d harus di antara %d dan %d", tengah.PPhTerutang, kecil.PPhTerutang, besar.PPhTerutang)
	}
}

// Setiap objek harus punya rujukan aturan dan tarif yang masuk akal.
func TestJenisPajakList_DataSehat(t *testing.T) {
	kode := map[string]bool{}
	for _, j := range tax.JenisPajakList {
		if len(j.Objek) == 0 {
			t.Errorf("%s tidak punya objek", j.ID)
		}
		for _, o := range j.Objek {
			if kode[o.Kode] {
				t.Errorf("kode objek %q dipakai dua kali", o.Kode)
			}
			kode[o.Kode] = true
			if o.TarifBps <= 0 || o.TarifBps > 20_000 {
				t.Errorf("%s: tarif %d bps tidak masuk akal", o.Kode, o.TarifBps)
			}
			if o.DasarBps <= 0 || o.DasarBps > 10_000 {
				t.Errorf("%s: dasar %d bps tidak masuk akal", o.Kode, o.DasarBps)
			}
			if o.Dasar == "" {
				t.Errorf("%s: rujukan aturan wajib diisi agar bisa dicek ulang", o.Kode)
			}
		}
	}
}
