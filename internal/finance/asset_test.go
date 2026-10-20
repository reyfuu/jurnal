package finance_test

import (
	"slices"
	"testing"

	"pajak/internal/finance"
)

func sum(xs []int64) int64 {
	var t int64
	for _, x := range xs {
		t += x
	}
	return t
}

func assetByName(t *testing.T, as finance.AssetSummary, nama string) finance.AssetReport {
	t.Helper()
	i := slices.IndexFunc(as.Assets, func(r finance.AssetReport) bool { return r.Nama == nama })
	if i < 0 {
		t.Fatalf("aset %q tidak ada di register", nama)
	}
	return as.Assets[i]
}

// Garis lurus Kelompok 1 (tarif 25%, masa manfaat 48 bulan): beban rata, habis tepat di bulan ke-48.
func TestAssetSchedule_GarisLurus(t *testing.T) {
	a := finance.Asset{Nama: "Grinder", Kelompok: "k1", Metode: finance.MetodeGarisLurus,
		TanggalPerolehan: "2026-02-01", HargaPerolehan: 24_000_000}

	sched := finance.AssetSchedule(a)
	if len(sched) != 48 {
		t.Fatalf("panjang jadwal: want 48 bulan, got %d", len(sched))
	}
	if sched[0] != 500_000 || sched[47] != 500_000 {
		t.Errorf("beban per bulan: want 500_000, got %d dan %d", sched[0], sched[47])
	}
	if got := sum(sched); got != 24_000_000 {
		t.Errorf("total penyusutan harus sama dengan harga perolehan: want 24_000_000, got %d", got)
	}
}

// Saldo menurun Kelompok 1 (tarif 50%) atas mesin Rp90 jt yang diperoleh Februari:
// tahun pertama hanya 11 bulan, dan di akhir masa manfaat sisa nilai buku disusutkan
// sekaligus (PMK 72/2023 Pasal 2 ayat 2) sehingga nilai buku berakhir nol.
func TestAssetSchedule_SaldoMenurun(t *testing.T) {
	a := finance.Asset{Nama: "Mesin Espresso", Kelompok: "k1", Metode: finance.MetodeSaldoMenurun,
		TanggalPerolehan: "2026-02-01", HargaPerolehan: 90_000_000}

	sched := finance.AssetSchedule(a)
	if len(sched) != 48 {
		t.Fatalf("panjang jadwal: want 48 bulan, got %d", len(sched))
	}
	if got := sum(sched); got != 90_000_000 {
		t.Fatalf("total penyusutan: want 90_000_000, got %d", got)
	}
	// Tahun pertama 50% x 90jt x 11/12 = 41,25jt → 3,75jt per bulan.
	if sched[0] != 3_750_000 {
		t.Errorf("beban bulan pertama: want 3_750_000, got %d", sched[0])
	}
	if got := sum(sched[:11]); got != 41_250_000 {
		t.Errorf("penyusutan 2026 (11 bulan): want 41_250_000, got %d", got)
	}

	jadwal := finance.AssetsForPeriod([]finance.Asset{a}, nil, "2026-08").Assets[0].Jadwal
	want := []finance.AssetYear{
		{Tahun: 2026, Bulan: 11, DasarAwal: 90_000_000, Penyusutan: 41_250_000, SisaAkhir: 48_750_000},
		{Tahun: 2027, Bulan: 12, DasarAwal: 48_750_000, Penyusutan: 24_375_000, SisaAkhir: 24_375_000},
		{Tahun: 2028, Bulan: 12, DasarAwal: 24_375_000, Penyusutan: 12_187_500, SisaAkhir: 12_187_500},
		{Tahun: 2029, Bulan: 12, DasarAwal: 12_187_500, Penyusutan: 6_093_750, SisaAkhir: 6_093_750},
		{Tahun: 2030, Bulan: 1, DasarAwal: 6_093_750, Penyusutan: 6_093_750, SisaAkhir: 0},
	}
	if !slices.Equal(jadwal, want) {
		t.Errorf("jadwal tahunan saldo menurun:\n got %+v\nwant %+v", jadwal, want)
	}
}

// Perbandingan dua metode atas aset yang sama: total sama, waktu pengakuan beban berbeda.
func TestAssetSchedule_SaldoMenurunLebihBeratDiAwal(t *testing.T) {
	base := finance.Asset{Kelompok: "k1", TanggalPerolehan: "2026-02-01", HargaPerolehan: 90_000_000}
	gl, sm := base, base
	gl.Metode, sm.Metode = finance.MetodeGarisLurus, finance.MetodeSaldoMenurun

	schedGL, schedSM := finance.AssetSchedule(gl), finance.AssetSchedule(sm)
	if schedGL[0] != 1_875_000 || schedSM[0] != 3_750_000 {
		t.Errorf("beban bulan pertama: garis lurus %d, saldo menurun %d", schedGL[0], schedSM[0])
	}
	if sum(schedGL) != sum(schedSM) {
		t.Errorf("total kedua metode harus sama: GL %d, SM %d", sum(schedGL), sum(schedSM))
	}
}

// Bangunan hanya boleh garis lurus; tarif bangunan tidak permanen 10% (masa manfaat 10 tahun).
func TestAsset_BangunanHanyaGarisLurus(t *testing.T) {
	a := finance.Asset{Nama: "Renovasi", JenisHarta: "bangunan_tidak_permanen",
		Metode: finance.MetodeSaldoMenurun, TanggalPerolehan: "2026-02-01", HargaPerolehan: 36_000_000}
	if err := finance.ValidateAsset(&a); err == nil {
		t.Fatal("saldo menurun atas bangunan harus ditolak")
	}

	a.Metode = finance.MetodeGarisLurus
	if err := finance.ValidateAsset(&a); err != nil {
		t.Fatalf("garis lurus atas bangunan harus diterima: %v", err)
	}
	sched := finance.AssetSchedule(a)
	if len(sched) != 120 || sched[0] != 300_000 {
		t.Errorf("bangunan tidak permanen: want 120 bulan @300_000, got %d bulan @%d", len(sched), sched[0])
	}
}

func TestValidateAsset_IsiKelompokDariJenisHarta(t *testing.T) {
	a := finance.Asset{Nama: "Software POS", JenisHarta: "software",
		TanggalPerolehan: "2026-02-01", HargaPerolehan: 18_000_000}
	if err := finance.ValidateAsset(&a); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if a.Kelompok != "k1" || !a.TakBerwujud || a.Metode != finance.MetodeGarisLurus {
		t.Errorf("kelompok/amortisasi/metode default tidak terisi: %+v", a)
	}
}

func TestValidateAsset_Penolakan(t *testing.T) {
	ok := finance.Asset{Nama: "Mesin", Kelompok: "k1", TanggalPerolehan: "2026-02-01", HargaPerolehan: 1_000_000}
	cases := []struct {
		name  string
		patch func(*finance.Asset)
	}{
		{"nama kosong", func(a *finance.Asset) { a.Nama = "  " }},
		{"tanggal tidak valid", func(a *finance.Asset) { a.TanggalPerolehan = "Februari 2026" }},
		{"harga nol", func(a *finance.Asset) { a.HargaPerolehan = 0 }},
		{"kelompok tidak dikenal", func(a *finance.Asset) { a.Kelompok = "k9" }},
		{"jenis harta tidak dikenal", func(a *finance.Asset) { a.JenisHarta = "pesawat" }},
		{"metode tidak dikenal", func(a *finance.Asset) { a.Metode = "anuitas" }},
	}
	for _, c := range cases {
		a := ok
		c.patch(&a)
		if err := finance.ValidateAsset(&a); err == nil {
			t.Errorf("%s: harus ditolak", c.name)
		}
	}
}

// Register aset seed per Agustus 2026: espresso saldo menurun, sisanya garis lurus.
func TestAssetsForPeriod_Agustus2026(t *testing.T) {
	as := finance.AssetsForPeriod(finance.SeedAssets(), nil, "2026-08")

	if len(as.Assets) != 6 || as.DraftCount != 0 {
		t.Fatalf("register: 6 aset tanpa draft, got %d aset / %d draft", len(as.Assets), as.DraftCount)
	}
	checks := []struct {
		name      string
		got, want int64
	}{
		{"TotalHargaPerolehan", as.TotalHargaPerolehan, 246_000_000},
		{"TotalAkumulasi", as.TotalAkumulasi, 45_475_000},
		{"TotalNilaiBuku", as.TotalNilaiBuku, 200_525_000},
		{"NilaiBukuAwal", as.NilaiBukuAwal, 207_075_000},
		{"BebanPenyusutan", as.BebanPenyusutan, 6_175_000},
		{"BebanAmortisasi", as.BebanAmortisasi, 375_000},
		{"BebanTotal", as.BebanTotal, 6_550_000},
		{"TambahanTanpaKas", as.TambahanTanpaKas, 0},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: want %d, got %d", c.name, c.want, c.got)
		}
	}
	if as.TotalHargaPerolehan-as.TotalAkumulasi != as.TotalNilaiBuku {
		t.Errorf("nilai buku harus harga perolehan dikurangi akumulasi")
	}

	// Mesin espresso: 7 bulan (Feb s/d Agu) x 3,75jt dengan saldo menurun 50%.
	mesin := assetByName(t, as, "Mesin Espresso La Marzocco Linea")
	if mesin.BebanBulanIni != 3_750_000 || mesin.Akumulasi != 26_250_000 || mesin.NilaiBuku != 63_750_000 {
		t.Errorf("mesin espresso: %+v", mesin)
	}
	if mesin.TarifBps != 5000 || mesin.MasaManfaat != 48 {
		t.Errorf("tarif saldo menurun Kelompok 1: want 5000 bps / 48 bulan, got %d / %d", mesin.TarifBps, mesin.MasaManfaat)
	}
	// Kulkas diperoleh Maret, jadi baru 6 bulan penyusutan.
	kulkas := assetByName(t, as, "Kulkas & Chiller Display")
	if kulkas.Akumulasi != 2_250_000 || kulkas.BebanBulanIni != 375_000 {
		t.Errorf("kulkas (perolehan Maret): %+v", kulkas)
	}
}

// Aset yang belum diperoleh pada periode yang dilihat tidak boleh ikut dihitung.
func TestAssetsForPeriod_SebelumPerolehan(t *testing.T) {
	as := finance.AssetsForPeriod(finance.SeedAssets(), nil, "2026-01")
	if len(as.Assets) != 0 || as.BebanTotal != 0 || as.TotalNilaiBuku != 0 {
		t.Fatalf("Januari 2026 belum punya aset: %d aset, beban %d, nilai buku %d",
			len(as.Assets), as.BebanTotal, as.TotalNilaiBuku)
	}
}

// Transaksi "Beli Alat / Aset" muncul sebagai draft: masuk neraca sebesar harga perolehan
// supaya tidak hilang, tapi belum menghasilkan beban sampai kelompoknya dilengkapi.
func TestAssetsForPeriod_DraftDariTransaksi(t *testing.T) {
	txs := []finance.Transaction{
		{ID: 77, Tanggal: "2026-08-10", Jenis: finance.JenisBeliAset, Nominal: 12_000_000,
			Keterangan: "Oven Deck 2 Pintu", Lunas: true},
	}
	as := finance.AssetsForPeriod(nil, txs, "2026-08")
	if len(as.Assets) != 1 || as.DraftCount != 1 {
		t.Fatalf("want 1 draft, got %d aset / %d draft", len(as.Assets), as.DraftCount)
	}
	d := as.Assets[0]
	if !d.Draft || d.Nama != "Oven Deck 2 Pintu" || d.NilaiBuku != 12_000_000 || d.BebanBulanIni != 0 {
		t.Errorf("draft: %+v", d)
	}
	if as.TambahanTanpaKas != 0 {
		t.Errorf("draft berasal dari transaksi kas, bukan setoran aset: got %d", as.TambahanTanpaKas)
	}

	// Setelah dilengkapi (kelompok + metode, tertaut ke transaksi yang sama) draft hilang.
	lengkap := []finance.Asset{{ID: 1, Nama: "Oven Deck 2 Pintu", Kelompok: "k1", Metode: finance.MetodeGarisLurus,
		TanggalPerolehan: "2026-08-10", HargaPerolehan: 12_000_000, TransaksiID: 77}}
	as = finance.AssetsForPeriod(lengkap, txs, "2026-08")
	if len(as.Assets) != 1 || as.DraftCount != 0 || as.BebanPenyusutan != 250_000 {
		t.Fatalf("aset lengkap: %d aset / %d draft / beban %d", len(as.Assets), as.DraftCount, as.BebanPenyusutan)
	}
}

func TestMetodeUsaha_SemuaAsetIkutKecualiBangunan(t *testing.T) {
	as := finance.SeedAssets()
	finance.TerapkanMetode(as, finance.MetodeSaldoMenurun)
	for _, a := range as {
		want := finance.MetodeSaldoMenurun
		if a.Kelompok == "bangunan_tidak_permanen" {
			want = finance.MetodeGarisLurus // PMK 72/2023: bangunan hanya garis lurus
		}
		if a.Metode != want {
			t.Errorf("%s: metode %s, want %s", a.Nama, a.Metode, want)
		}
	}
	finance.TerapkanMetode(as, "") // belum dipilih: metode per aset dibiarkan
	if as[0].Metode != finance.MetodeSaldoMenurun {
		t.Error("metode kosong mengubah aset")
	}
}

func TestMetodeSeragam(t *testing.T) {
	as := finance.SeedAssets() // espresso saldo menurun, sisanya garis lurus
	if m, ok := finance.MetodeSeragam(as); ok || m != "" {
		t.Errorf("seed campur: %q %v", m, ok)
	}
	as[0].Metode = finance.MetodeGarisLurus
	if m, ok := finance.MetodeSeragam(as); !ok || m != finance.MetodeGarisLurus {
		t.Errorf("seragam garis lurus: %q %v", m, ok)
	}
	// bangunan & draft tidak ikut menentukan; tanpa aset lain = belum ada pilihan
	if m, ok := finance.MetodeSeragam([]finance.Asset{as[4], {Nama: "draft"}}); !ok || m != "" {
		t.Errorf("hanya bangunan/draft: %q %v", m, ok)
	}
}

func TestProfil_MetodePenyusutanValid(t *testing.T) {
	p := finance.BusinessProfile{NamaUsaha: "A", JenisUsaha: "jasa", BentukUsaha: "pt", MetodePenyusutan: "cepat"}
	if err := p.Validate(); err == nil {
		t.Fatal("metode ngawur diterima")
	}
	p.MetodePenyusutan = finance.MetodeSaldoMenurun
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
}
