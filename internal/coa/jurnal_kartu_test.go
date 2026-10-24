package coa

import (
	"slices"
	"testing"

	"pajak/internal/finance"
)

// akunUji: barang 1 Kain (1301), 2 Benang (1302), 3 Potongan (1303), 4 Kaos (1305).
func akunUji(id int) string {
	return map[int]string{1: "1301", 2: "1302", 3: "1303", 4: "1305"}[id]
}

func saldoBaris(lines []Line) (map[string]int64, int64, int64) {
	net := map[string]int64{}
	var d, k int64
	for _, l := range lines {
		net[l.Kode] += l.Debit - l.Kredit
		d, k = d+l.Debit, k+l.Kredit
	}
	return net, d, k
}

func TestJurnalKartu_ProduksiBertahap(t *testing.T) {
	peta := PetaDari(Manufaktur)
	baris := []finance.BarisKartu{
		{Tanggal: "2026-01-01", StokID: 1, Jenis: finance.KartuAwal, Nilai: 100_000},
		{Tanggal: "2026-01-03", StokID: 1, Jenis: finance.KartuBeli, TransaksiID: 5, Nilai: 50_000},
		{Tanggal: "2026-01-05", StokID: 1, Jenis: finance.KartuProduksiPakai, GerakID: 1, Nilai: -60_000},
		{Tanggal: "2026-01-05", StokID: 3, Jenis: finance.KartuProduksiMasuk, GerakID: 1, Nilai: 80_000, Biaya: 20_000},
		{Tanggal: "2026-01-08", StokID: 3, Jenis: finance.KartuProduksiPakai, GerakID: 2, Nilai: -80_000},
		{Tanggal: "2026-01-08", StokID: 2, Jenis: finance.KartuProduksiPakai, GerakID: 2, Nilai: -10_000},
		{Tanggal: "2026-01-08", StokID: 4, Jenis: finance.KartuProduksiMasuk, GerakID: 2, Nilai: 90_000},
		{Tanggal: "2026-01-12", StokID: 4, Jenis: finance.KartuJual, TransaksiID: 9, Nilai: -45_000},
		{Tanggal: "2026-01-20", StokID: 2, Jenis: finance.KartuOpname, GerakID: 3, Nilai: -1_000},
		{Tanggal: "2026-02-02", StokID: 4, Jenis: finance.KartuJual, TransaksiID: 10, Nilai: -7_000}, // di luar periode
	}
	lines := JurnalKartu(baris, "2026-01-01", "2026-01-31", akunUji, peta)
	net, d, k := saldoBaris(lines)
	hpp := peta.Kode(PeranHPP)
	for kode, mau := range map[string]int64{"1301": -60_000, "1303": -20_000, "1302": -11_000, "1305": 45_000, hpp: 46_000} {
		if net[kode] != mau {
			t.Errorf("akun %s = %d, harus %d", kode, net[kode], mau)
		}
	}
	if d != 196_000 || k != 196_000 {
		t.Errorf("debit %d kredit %d, harus 196.000", d, k)
	}
	// Gerak 1 seimbang tanpa biaya: Dr 1303 60.000 = Cr 1301 60.000.
	var g1d, g1k int64
	for _, l := range lines {
		if l.Tanggal == "2026-01-05" {
			if l.Kode == "1303" {
				g1d += l.Debit
			}
			if l.Kode == "1301" {
				g1k += l.Kredit
			}
		}
	}
	if g1d != 60_000 || g1k != 60_000 {
		t.Errorf("gerak 1: Dr 1303 %d Cr 1301 %d", g1d, g1k)
	}
	for _, l := range lines {
		if l.Kode == hpp && l.Tanggal == "2026-01-12" && l.TxID != 9 {
			t.Errorf("baris HPP penjualan TxID %d, harus 9", l.TxID)
		}
		if l.Tanggal == "2026-01-12" && l.Kode == hpp && l.Debit != 45_000 {
			t.Errorf("HPP penjualan %d", l.Debit)
		}
	}
	// Keterangan Jurnal Umum.
	for _, l := range JurnalKartuKet(baris, "2026-01-01", "2026-01-31", akunUji, peta) {
		var mau string
		switch l.Tanggal {
		case "2026-01-05", "2026-01-08":
			mau = "Produksi"
		case "2026-01-12":
			mau = "HPP"
		case "2026-01-20":
			mau = "Selisih stok opname"
		}
		if l.Ket != mau {
			t.Errorf("ket %s %s = %q, harus %q", l.Tanggal, l.Kode, l.Ket, mau)
		}
	}
}

func TestJurnalKartu_KetProduksi(t *testing.T) {
	peta := PetaDari(Manufaktur)
	baris := []finance.BarisKartu{
		{Tanggal: "2026-01-05", StokID: 1, Jenis: finance.KartuProduksiPakai, GerakID: 1, Keterangan: "Potong kain", Nilai: -60_000},
		{Tanggal: "2026-01-05", StokID: 3, Jenis: finance.KartuProduksiMasuk, GerakID: 1, Keterangan: "Potong kain", Nilai: 60_000},
	}
	for _, l := range JurnalKartuKet(baris, "2026-01-01", "2026-01-31", akunUji, peta) {
		if l.Ket != "Produksi — Potong kain" {
			t.Errorf("ket %q", l.Ket)
		}
	}
}

func TestJurnalKartu_PembulatanSeimbang(t *testing.T) {
	peta := PetaDari(Manufaktur)
	for _, kasus := range []struct {
		nama   string
		a, b   int64
		bahan  int64
		maxDeb int64
	}{
		{"pas", 33_333, 33_334, 66_667, 33_334},
		{"kurang 1", 33_333, 33_333, 66_667, 33_334}, // selisih ke baris debit terbesar (seri → yang pertama)
	} {
		baris := []finance.BarisKartu{
			{Tanggal: "2026-01-05", StokID: 1, Jenis: finance.KartuProduksiPakai, GerakID: 7, Nilai: -kasus.bahan},
			{Tanggal: "2026-01-05", StokID: 3, Jenis: finance.KartuProduksiMasuk, GerakID: 7, Nilai: kasus.a},
			{Tanggal: "2026-01-05", StokID: 4, Jenis: finance.KartuProduksiMasuk, GerakID: 7, Nilai: kasus.b},
		}
		lines := JurnalKartu(baris, "2026-01-01", "2026-01-31", akunUji, peta)
		_, d, k := saldoBaris(lines)
		if d != k || d != kasus.bahan {
			t.Errorf("%s: debit %d kredit %d", kasus.nama, d, k)
		}
		var maks int64
		for _, l := range lines {
			maks = max(maks, l.Debit)
		}
		if maks != kasus.maxDeb {
			t.Errorf("%s: debit terbesar %d, harus %d", kasus.nama, maks, kasus.maxDeb)
		}
	}
}

// akunStok nil: semua barang ke akun peran persediaan.
func TestJurnalKartu_TanpaAkunBarang(t *testing.T) {
	peta := PetaDari(Master)
	baris := []finance.BarisKartu{{Tanggal: "2026-01-12", StokID: 4, Jenis: finance.KartuJual, TransaksiID: 9, Nilai: -45_000}}
	lines := JurnalKartu(baris, "2026-01-01", "2026-01-31", nil, peta)
	if len(lines) != 2 || lines[1].Kode != peta.Kode(PeranPersediaan) || lines[1].Kredit != 45_000 || lines[0].Kode != peta.Kode(PeranHPP) {
		t.Errorf("baris %+v", lines)
	}
}

func TestJournalBerjalan_PembelianKeAkunBarang(t *testing.T) {
	peta := PetaDari(Manufaktur)
	p := finance.BusinessProfile{JenisUsaha: "manufaktur", MetodePersediaan: "perpetual"}
	txs := []finance.Transaction{
		{ID: 1, Tanggal: "2026-01-03", Jenis: finance.JenisBeliStok, Nominal: 50_000, Lunas: true, StokID: 2, QtyStok: 1000},
		{ID: 2, Tanggal: "2026-01-04", Jenis: finance.JenisBeliStok, Nominal: 10_000, Lunas: true},
		{ID: 3, Tanggal: "2026-01-05", Jenis: finance.JenisBeliStok, Nominal: 7_000, Lunas: true, StokID: 2, Akun: "1305"},
	}
	akun := map[int]string{}
	for _, l := range JournalBerjalan(txs, "2026-01", p, peta, akunUji, nil) {
		if l.Debit > 0 {
			akun[l.TxID] = l.Kode
		}
	}
	if akun[1] != "1302" || akun[2] != peta.Kode(PeranPersediaan) || akun[3] != "1305" {
		t.Errorf("akun debit %+v", akun)
	}
	for _, l := range JournalBerjalan(txs[:1], "2026-01", p, peta, nil, nil) {
		if l.Debit > 0 && l.Kode != peta.Kode(PeranPersediaan) {
			t.Errorf("akunStok nil: debit %s", l.Kode)
		}
	}
}

// Opname lebih: Dr akun barang / Cr HPP. Baris bernilai 0 tidak menghasilkan jurnal.
func TestJurnalKartu_OpnameLebihDanNilaiNol(t *testing.T) {
	peta := PetaDari(Manufaktur)
	hpp := peta.Kode(PeranHPP)
	lines := JurnalKartu([]finance.BarisKartu{
		{Tanggal: "2026-01-20", StokID: 2, Jenis: finance.KartuOpname, GerakID: 3, Nilai: 2_500},
	}, "2026-01-01", "2026-01-31", akunUji, peta)
	if len(lines) != 2 || lines[0].Kode != "1302" || lines[0].Debit != 2_500 || lines[1].Kode != hpp || lines[1].Kredit != 2_500 {
		t.Errorf("opname lebih: %+v", lines)
	}
	nol := JurnalKartu([]finance.BarisKartu{
		{Tanggal: "2026-01-12", StokID: 4, Jenis: finance.KartuJual, TransaksiID: 9},
		{Tanggal: "2026-01-20", StokID: 2, Jenis: finance.KartuOpname, GerakID: 3},
	}, "2026-01-01", "2026-01-31", akunUji, peta)
	if len(nol) != 0 {
		t.Errorf("baris bernilai 0 harus tanpa jurnal: %+v", nol)
	}
}

// Data yang melewati Validate (Σ nilai > Nominal): jatuh ke satu debit sebesar Nominal ke akun
// barang baris pertama, jadi jurnal tetap seimbang.
func TestJournalBerjalan_NilaiRincianMelebihiNominal(t *testing.T) {
	peta := PetaDari(Manufaktur)
	p := finance.BusinessProfile{JenisUsaha: "manufaktur", MetodePersediaan: "perpetual"}
	txs := []finance.Transaction{{ID: 1, Tanggal: "2026-01-03", Jenis: finance.JenisBeliStok, Nominal: 40_000, Lunas: true,
		Rincian: []finance.RincianBarang{{StokID: 2, QtyStok: 1000, Nilai: 30_000}, {StokID: 1, QtyStok: 1000, Nilai: 20_000}}}}
	lines := JournalBerjalan(txs, "2026-01", p, peta, akunUji, nil)
	var dr []Line
	var d, k int64
	for _, l := range lines {
		d, k = d+l.Debit, k+l.Kredit
		if l.Debit > 0 {
			dr = append(dr, l)
		}
	}
	if d != k || d != 40_000 || len(dr) != 1 || dr[0].Kode != "1302" {
		t.Errorf("jurnal %+v (debit %d kredit %d)", lines, d, k)
	}
}

// Fix round 1 (R5): pembelian berbarang dengan akun pilihan persediaan (mis. dari Catat cepat)
// dipecah ke akun barangnya seperti tanpa akun pilihan; akun pilihan jadi akun sisa. Akun
// pilihan bukan persediaan (HPP) tetap satu debit.
func TestJournalBerjalan_AkunPilihanPersediaanDipecah(t *testing.T) {
	peta := PetaDari(Manufaktur)
	p := finance.BusinessProfile{JenisUsaha: "manufaktur", MetodePersediaan: "perpetual"}
	rincian := []finance.RincianBarang{{StokID: 2, QtyStok: 1000, Nilai: 30_000}, {StokID: 1, QtyStok: 1000, Nilai: 5_000}}
	persediaan := func(kode string) bool {
		return slices.ContainsFunc(AkunPersediaan(Manufaktur), func(a Account) bool { return a.Kode == kode })
	}
	debit := func(akun string) []Line {
		txs := []finance.Transaction{{ID: 1, Tanggal: "2026-01-03", Jenis: finance.JenisBeliStok, Nominal: 40_000, Lunas: true, Akun: akun, Rincian: rincian}}
		var dr []Line
		var d, k int64
		for _, l := range JournalBerjalan(txs, "2026-01", p, peta, akunUji, persediaan) {
			d, k = d+l.Debit, k+l.Kredit
			if l.Debit > 0 {
				l.TxID, l.Tanggal = 0, ""
				dr = append(dr, l)
			}
		}
		if d != k {
			t.Errorf("akun %s: debit %d ≠ kredit %d", akun, d, k)
		}
		return dr
	}
	if got, mau := debit("1305"), []Line{{Kode: "1302", Debit: 30_000}, {Kode: "1301", Debit: 5_000}, {Kode: "1305", Debit: 5_000}}; !slices.Equal(got, mau) {
		t.Errorf("akun pilihan persediaan: Dr %+v, mau %+v", got, mau)
	}
	hpp := peta.Kode(PeranHPP)
	if got, mau := debit(hpp), []Line{{Kode: hpp, Debit: 40_000}}; !slices.Equal(got, mau) {
		t.Errorf("akun pilihan HPP: Dr %+v, mau %+v", got, mau)
	}
}
