package finance_test

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"pajak/internal/finance"
)

// Template yang kita bagikan harus lolos parsernya sendiri; kalau tidak, user ikut gagal.
func TestOnboardingTemplate_LolosParsernyaSendiri(t *testing.T) {
	d := finance.ParseOnboarding(finance.OnboardingTemplate(""))

	if len(d.Errors) != 0 {
		t.Fatalf("template harus bebas error, got %+v", d.Errors)
	}
	if d.Opening == nil {
		t.Fatal("saldo awal rinci tidak terbaca")
	}
	o := d.Opening
	if o.Tanggal != "2026-01-01" || o.Kas != 45_000_000 || o.Piutang != 8_500_000 || o.Persediaan != 8_250_000 ||
		o.AsetLain != 7_050_000 || o.Utang != 5_200_000 || o.Pinjaman != 20_000_000 || o.KewajibanLain != 450_000 || o.LabaDitahan != 10_000_000 {
		t.Errorf("saldo awal: %+v", o)
	}
	if len(d.Assets) != 5 || len(d.Karyawan) != 3 || len(d.Transaksi) != 3 || len(d.Stok) != 3 {
		t.Fatalf("jumlah baris: %d aset, %d karyawan, %d transaksi, %d stok", len(d.Assets), len(d.Karyawan), len(d.Transaksi), len(d.Stok))
	}
	// Persediaan ikut mengisi Stok & Gudang (kode barang → SKU, harga pokok per satuan).
	if s := d.Stok[0]; s.SKU != "RM-001" || s.Jumlah != 30 || s.HargaPokok != 185_000 || s.Satuan != "kg" || s.Kategori != "Bahan Baku" {
		t.Errorf("stok pertama: %+v", s)
	}
	// Kelompok dari Kelompok Fiskal, jenis harta tetap tertaut bila cocok.
	if a := d.Assets[0]; a.Nama != "Laptop Kasir" || a.Kelompok != "k1" || a.JenisHarta != "komputer" || a.Metode != finance.MetodeGarisLurus {
		t.Errorf("aset pertama: %+v", a)
	}
	if !d.Assets[4].TakBerwujud {
		t.Errorf("software harus ditandai harta tak berwujud: %+v", d.Assets[4])
	}
	// Renovasi ruko = bangunan tidak permanen, wajib garis lurus.
	if d.Assets[3].Kelompok != "bangunan_tidak_permanen" || d.Assets[3].Metode != finance.MetodeGarisLurus {
		t.Errorf("renovasi: %+v", d.Assets[3])
	}
	terbaca := strings.Join(d.Terbaca, "|")
	for _, s := range []string{"Petunjuk", "Kas dan Bank", "Piutang", "Persediaan", "Perlengkapan", "Aset Lancar Lain", "Aset Tetap", "Utang", "Modal", "Karyawan", "Transaksi"} {
		if !strings.Contains(terbaca, s) {
			t.Errorf("sheet %s tidak terbaca: %s", s, terbaca)
		}
	}
}

// Contoh template harus seimbang (Harta = Kewajiban + Modal) dan aset tetapnya sama dengan
// nilai buku sheet Aset Tetap, untuk metode penyusutan apa pun yang dipakai usaha.
func TestOnboardingTemplate_SaldoAwalSeimbang(t *testing.T) {
	for _, metode := range []string{"", finance.MetodeGarisLurus, finance.MetodeSaldoMenurun} {
		d := finance.ParseOnboarding(finance.OnboardingTemplate(metode))
		if len(d.Errors) != 0 || d.Opening == nil {
			t.Fatalf("%q: %+v", metode, d.Errors)
		}
		finance.TerapkanMetode(d.Assets, metode)
		nbv := finance.AssetsForPeriod(d.Assets, d.Transaksi, d.Opening.Period()).NilaiBukuAwal
		d.Opening.LengkapiAsetTetap(nbv)
		if nbv == 0 || d.Opening.AsetTetap != nbv {
			t.Errorf("%q: aset tetap %d, nilai buku sheet Aset Tetap %d", metode, d.Opening.AsetTetap, nbv)
		}
		if err := d.Opening.CekSeimbang(nbv); err != nil {
			t.Errorf("%q: %v", metode, err)
		}
		// Saldo menurun: akumulasi manual contoh = hitungan sistem, selain bangunan (garis lurus).
		for _, a := range d.Assets {
			manual := a.AkumulasiAwal > 0 && a.PerTanggal == "2026-01-01"
			if want := metode == finance.MetodeSaldoMenurun && a.Kelompok != "bangunan_tidak_permanen"; manual != want {
				t.Errorf("%q: %s akumulasi manual %d per %q", metode, a.Nama, a.AkumulasiAwal, a.PerTanggal)
			}
		}
	}
}

// Berkas yang sudah disimpan Excel membawa nilai rumus (Akumulasi Dipakai, Nilai Buku). Nilai
// itu dipakai sebagai akumulasi awal & nilai buku yang dinyatakan, dan harus tetap seimbang.
func TestOnboardingTemplate_NilaiRumusTersimpanExcel(t *testing.T) {
	sheets := finance.OnboardingTemplate("")
	for _, sh := range sheets {
		if sh.Nama != "Aset Tetap" {
			continue
		}
		for _, row := range sh.Rows[6:] {
			harga, _ := strconv.ParseInt(row[5], 10, 64)
			p, _ := time.Parse(time.DateOnly, row[4])
			bulan := (2026-p.Year())*12 + 1 - int(p.Month())
			n := map[string]int64{"Kelompok 1 (4 th)": 48, "Bangunan Tidak Permanen (10 th)": 120}[row[3]]
			akum := harga / n * int64(bulan) // rumus Excel: INT(harga/n) × bulan
			row[9], row[10] = strconv.FormatInt(akum, 10), strconv.FormatInt(harga-akum, 10)
		}
	}
	d := finance.ParseOnboarding(sheets)
	if len(d.Errors) != 0 || d.Opening == nil {
		t.Fatalf("%+v", d.Errors)
	}
	nbv := finance.AssetsForPeriod(d.Assets, nil, "2026-01").NilaiBukuAwal
	if d.Opening.AsetTetap != nbv || d.Assets[0].AkumulasiAwal != 3_750_000 {
		t.Fatalf("aset tetap dinyatakan %d, register %d, akumulasi laptop %d", d.Opening.AsetTetap, nbv, d.Assets[0].AkumulasiAwal)
	}
	if err := d.Opening.CekSeimbang(nbv); err != nil {
		t.Fatal(err)
	}
}

// Isian rincian: nilai turunan dihitung ulang dari kolom isian; cadangan kerugian & prive
// mengurangi pos-nya; baris bermasalah dilaporkan dengan nomor baris Excel.
func TestParseOnboarding_RinciDihitungUlangDanError(t *testing.T) {
	sheets := finance.OnboardingTemplate("")
	set := func(nama string, baris int, kol int, v string) {
		for _, sh := range sheets {
			if sh.Nama == nama {
				for len(sh.Rows) < baris {
					sh.Rows = append(sh.Rows, nil)
				}
				row := sh.Rows[baris-1]
				for len(row) <= kol {
					row = append(row, "")
				}
				row[kol] = v
				sh.Rows[baris-1] = row
				return
			}
		}
		t.Fatalf("sheet %s tidak ada", nama)
	}
	set("Piutang", 4, 8, "500000")   // Cadangan Kerugian
	set("Piutang", 7, 7, "999")      // Sisa Piutang (rumus) palsu: diabaikan
	set("Modal", 8, 2, "Prive")      // saldo laba contoh → prive 10 jt
	set("Utang", 8, 8, "900000")     // sudah dibayar > nilai 450 rb → error baris 8
	set("Aset Tetap", 7, 3, "Tanah") // tanah belum didukung → error baris 7
	d := finance.ParseOnboarding(sheets)

	want := map[string]int{"Utang": 8, "Aset Tetap": 7}
	for _, e := range d.Errors {
		if want[e.Sheet] != e.Baris {
			t.Errorf("error tak terduga: %+v", e)
		}
		delete(want, e.Sheet)
	}
	if len(want) != 0 {
		t.Fatalf("error tidak dilaporkan: %v (semua: %+v)", want, d.Errors)
	}
	o := d.Opening
	if o == nil {
		t.Fatal("saldo awal tetap terbentuk dari baris yang valid")
	}
	if o.Piutang != 8_000_000 || o.LabaDitahan != 0 || o.Kas != 45_000_000 || o.KewajibanLain != 0 {
		t.Errorf("piutang %d (8,5 jt − cadangan 0,5 jt), laba ditahan %d, kas %d, kewajiban lain %d",
			o.Piutang, o.LabaDitahan, o.Kas, o.KewajibanLain)
	}
	prive := false
	for _, r := range o.Rincian {
		prive = prive || (r.Kategori == finance.KatModal && r.Nominal == -10_000_000)
	}
	if !prive {
		t.Errorf("prive 10 jt harus jadi baris modal minus: %+v", o.Rincian)
	}
	if len(d.Assets) != 4 {
		t.Errorf("aset tanah tidak ikut: %d aset", len(d.Assets))
	}
}

// Tanggal Saldo Awal wajib; tanpa itu tidak ada saldo awal yang dibentuk.
func TestParseOnboarding_RinciTanpaTanggal(t *testing.T) {
	sheets := finance.OnboardingTemplate("")
	sheets[0].Rows[5][2] = ""
	d := finance.ParseOnboarding(sheets)
	if d.Opening != nil || len(d.Errors) != 1 || d.Errors[0].Sheet != "Petunjuk" || d.Errors[0].Baris != 6 {
		t.Fatalf("opening %+v, errors %+v", d.Opening, d.Errors)
	}
	// Nomor seri tanggal Excel (sel tanggal asli) juga diterima.
	sheets[0].Rows[5][2] = "46023"
	if d := finance.ParseOnboarding(sheets); d.Opening == nil || d.Opening.Tanggal != "2026-01-01" {
		t.Fatalf("seri Excel: %+v %+v", d.Opening, d.Errors)
	}
}

// Baris tambahan klien (akun di luar template) dibaca dari Kategori-nya; saldo awal yang tidak
// seimbang ditolak dengan angka yang jelas.
func TestSaldoAwalPerAkun_BarisTambahanDanTidakSeimbang(t *testing.T) {
	rows := [][]string{
		{"Saldo Awal"},
		{"Tanggal mulai pembukuan", "2026-01-01"},
		{"Kode Akun", "Nama Akun", "Kategori Akun", "Nominal (Rp)", "Keterangan"},
		{"1102", "Bank BCA", "Harta: Kas & Bank", "10.000.000", ""},
		{"", "Rekening Mandiri kedua", "Harta: Kas & Bank", "5000000", "ditambah sendiri"},
		{"", "Utang ke saudara", "Kewajiban: Lain-lain", "3000000", ""},
		{"3101", "Modal Pemilik", "Modal: Modal Disetor", "12000000", ""},
	}
	d := finance.ParseOnboarding([]finance.ImportSheet{{Nama: "Saldo Awal", Rows: rows}})
	if len(d.Errors) != 0 || d.Opening == nil || d.Opening.Kas != 15_000_000 || d.Opening.KewajibanLain != 3_000_000 || len(d.Opening.Rincian) != 4 {
		t.Fatalf("%+v %+v", d.Errors, d.Opening)
	}
	if err := d.Opening.CekSeimbang(0); err != nil {
		t.Fatalf("15 jt = 3 jt + 12 jt harus seimbang: %v", err)
	}
	d.Opening.ModalDisetor = 10_000_000
	if err := d.Opening.CekSeimbang(0); err == nil || !strings.Contains(err.Error(), "2.000.000") {
		t.Fatalf("selisih 2 jt harus ditolak: %v", err)
	}
	if err := d.Opening.CekSeimbang(5_000_000); err == nil || !strings.Contains(err.Error(), "nilai buku") {
		t.Fatalf("aset tetap tidak sama dengan register harus ditolak: %v", err)
	}
}

// Nama sheet dan label bebas selama mengandung kata kuncinya, dan format angka/tanggal
// mengikuti kebiasaan Indonesia.
func TestParseOnboarding_LabelDanFormatBebas(t *testing.T) {
	d := finance.ParseOnboarding([]finance.ImportSheet{
		{Nama: "NERACA AWAL per 1 Jan", Rows: [][]string{
			{"Pos", "Jumlah"},
			{"Tanggal mulai", "01/01/2026"},
			{"Saldo Kas dan Bank", "Rp 45.000.000"},
			{"Piutang pelanggan", "8.500.000"},
			{"Nilai stok gudang", "8.250.000"},
			{"Hutang supplier", "5.200.000"},
			{"Pinjaman bank", "0"},
		}},
		{Nama: "Daftar Pegawai", Rows: [][]string{
			{"Nama Pegawai", "Posisi", "Mulai Kerja", "Upah Pokok", "Allowance", "PTKP"},
			{"Dewi Lestari", "Barista", "05/01/2026", "Rp 5.500.000", "750.000", "K/1"},
		}},
	})

	if len(d.Errors) != 0 {
		t.Fatalf("errors: %+v", d.Errors)
	}
	if d.Opening == nil || d.Opening.Tanggal != "2026-01-01" || d.Opening.Kas != 45_000_000 ||
		d.Opening.Piutang != 8_500_000 || d.Opening.Persediaan != 8_250_000 || d.Opening.Utang != 5_200_000 {
		t.Fatalf("saldo awal: %+v", d.Opening)
	}
	if len(d.Karyawan) != 1 || d.Karyawan[0].GajiPokok != 5_500_000 || d.Karyawan[0].Tunjangan != 750_000 {
		t.Fatalf("karyawan: %+v", d.Karyawan)
	}
	if d.Karyawan[0].PenghasilanBulanan() != 6_250_000 || !d.Karyawan[0].Aktif {
		t.Errorf("penghasilan bulanan / status: %+v", d.Karyawan[0])
	}
}

// Baris bermasalah dilaporkan dengan nomor baris Excel dan tidak ikut diimpor; baris lain tetap masuk.
func TestParseOnboarding_BarisBermasalahDilaporkan(t *testing.T) {
	d := finance.ParseOnboarding([]finance.ImportSheet{
		{Nama: "Aset", Rows: [][]string{
			{"Nama Aset", "Jenis Harta", "Tanggal Perolehan", "Harga Perolehan"},
			{"Oven Deck", "Peralatan usaha restoran / rumah makan", "2026-02-01", "24000000"},
			{"Pesawat Jet", "Pesawat terbang pribadi", "2026-02-01", "10000000000"},
			{"Kompor Gas", "Mesin pengolah makanan & minuman", "bukan tanggal", "3000000"},
			{"Blender", "Mesin pengolah makanan & minuman", "2026-02-01", "gratis"},
		}},
	})

	if len(d.Assets) != 1 || d.Assets[0].Nama != "Oven Deck" {
		t.Fatalf("hanya baris valid yang masuk: %+v", d.Assets)
	}
	if len(d.Errors) != 3 {
		t.Fatalf("want 3 error, got %+v", d.Errors)
	}
	for i, baris := range []int{3, 4, 5} {
		if d.Errors[i].Baris != baris || d.Errors[i].Sheet != "Aset" {
			t.Errorf("error %d menunjuk baris %d sheet %q, want baris %d", i, d.Errors[i].Baris, d.Errors[i].Sheet, baris)
		}
	}
}

func TestParseOnboarding_SheetTidakDikenali(t *testing.T) {
	d := finance.ParseOnboarding([]finance.ImportSheet{{Nama: "Sheet1", Rows: [][]string{{"a", "b"}}}})
	if len(d.Errors) != 1 || d.Errors[0].Baris != 0 {
		t.Fatalf("want satu error header, got %+v", d.Errors)
	}
}

// Modal awal dihitung dari komposisi saldo awal; ModalDisetor hanya untuk dicocokkan.
func TestOpeningBalance_ModalAwalDanSelisih(t *testing.T) {
	o := finance.OpeningBalance{
		Tanggal: "2026-01-01", Kas: 45_000_000, Piutang: 8_500_000, Persediaan: 8_250_000,
		Utang: 5_200_000, Pinjaman: 0,
	}
	const nbvAset int64 = 100_000_000

	if got := o.ModalAwal(nbvAset); got != 156_550_000 {
		t.Errorf("modal awal: want 156_550_000, got %d", got)
	}
	if got := o.SelisihModal(nbvAset); got != 0 {
		t.Errorf("tanpa modal disetor tidak ada selisih, got %d", got)
	}
	o.ModalDisetor = 150_000_000
	if got := o.SelisihModal(nbvAset); got != -6_550_000 {
		t.Errorf("selisih modal: want -6_550_000, got %d", got)
	}
}

// Saldo kas awal periode ikut berjalan: saldo awal + kas bersih semua transaksi sebelumnya.
func TestOpeningBalance_KasBerjalanAntarPeriode(t *testing.T) {
	o := finance.OpeningBalance{Tanggal: "2026-01-01", Kas: 20_000_000, Piutang: 3_000_000, Utang: 1_000_000, Persediaan: 5_000_000}
	txs := []finance.Transaction{
		{ID: 1, Tanggal: "2026-01-10", Jenis: "Penjualan", Nominal: 9_000_000, Lunas: true},
		{ID: 2, Tanggal: "2026-01-20", Jenis: "Gaji & Upah", Nominal: 4_000_000, Lunas: true},
		{ID: 3, Tanggal: "2026-02-05", Jenis: "Penjualan", Nominal: 7_000_000, Lunas: true},
		{ID: 4, Tanggal: "2026-01-25", Jenis: "Penjualan", Nominal: 5_000_000, Lunas: false}, // piutang, belum kas
	}

	jan := o.At(txs, "2026-01")
	if jan.Cash != 20_000_000 {
		t.Errorf("kas awal Januari = saldo awal: want 20_000_000, got %d", jan.Cash)
	}
	if jan.Receivables != 3_000_000 || jan.Payables != 1_000_000 {
		t.Errorf("piutang & utang awal hanya di periode onboarding: %+v", jan)
	}

	feb := o.At(txs, "2026-02")
	if feb.Cash != 25_000_000 {
		t.Errorf("kas awal Februari = 20jt + 9jt − 4jt: want 25_000_000, got %d", feb.Cash)
	}
	if feb.Receivables != 0 || feb.Payables != 0 {
		t.Errorf("periode setelah onboarding tidak mengulang piutang/utang awal: %+v", feb)
	}
	if feb.Inventory != 5_000_000 {
		t.Errorf("persediaan tetap dibawa: got %d", feb.Inventory)
	}
}

// Neraca harus tetap seimbang ketika saldo awal onboarding dipakai, termasuk di periode
// onboarding yang membawa piutang, utang, dan pinjaman awal.
func TestBalanceSheet_SeimbangDenganNeracaAwal(t *testing.T) {
	o := finance.OpeningBalance{
		Tanggal: "2026-01-01", Kas: 45_000_000, Piutang: 8_500_000, Persediaan: 8_250_000,
		Utang: 5_200_000, Pinjaman: 12_000_000,
	}
	txs, assets := finance.SeedTransactions(), finance.SeedAssets()
	for _, p := range finance.AvailablePeriods() {
		rep := finance.BuildPeriodReport(txs, assets, p.ID, o.At(txs, p.ID))
		bs := rep.BalanceSheet
		if !bs.IsBalanced {
			t.Errorf("%s tidak seimbang: aktiva %d vs pasiva %d", p.ID, bs.Assets.Total, bs.Liabilities.Total+bs.Equity.Total)
		}
	}

	jan := finance.BuildPeriodReport(txs, assets, "2026-01", o.At(txs, "2026-01")).BalanceSheet
	if jan.Liabilities.Loans != 12_000_000 {
		t.Errorf("pinjaman awal masuk kewajiban: got %d", jan.Liabilities.Loans)
	}
	if jan.Assets.Inventory != 8_250_000 {
		t.Errorf("persediaan dari neraca awal: got %d", jan.Assets.Inventory)
	}
}

func TestValidateOpeningBalance_Penolakan(t *testing.T) {
	cases := []struct {
		name string
		o    finance.OpeningBalance
	}{
		{"tanggal kosong", finance.OpeningBalance{Kas: 1}},
		{"tanggal salah format", finance.OpeningBalance{Tanggal: "01-01-2026"}},
		{"kas negatif", finance.OpeningBalance{Tanggal: "2026-01-01", Kas: -1}},
		{"utang negatif", finance.OpeningBalance{Tanggal: "2026-01-01", Utang: -5}},
	}
	for _, c := range cases {
		o := c.o
		if err := finance.ValidateOpeningBalance(&o); err == nil {
			t.Errorf("%s: harus ditolak", c.name)
		}
	}
}

func TestValidateKaryawan(t *testing.T) {
	ok := finance.Karyawan{Nama: " Dewi ", TanggalMasuk: "2026-01-05", GajiPokok: 5_000_000}
	if err := finance.ValidateKaryawan(&ok); err != nil {
		t.Fatalf("karyawan valid ditolak: %v", err)
	}
	if ok.Nama != "Dewi" || ok.PTKP != "TK/0" {
		t.Errorf("nama dirapikan & PTKP default TK/0: %+v", ok)
	}

	bad := []finance.Karyawan{
		{TanggalMasuk: "2026-01-05", GajiPokok: 1},
		{Nama: "A", TanggalMasuk: "05 Januari", GajiPokok: 1},
		{Nama: "A", TanggalMasuk: "2026-01-05", GajiPokok: 0},
		{Nama: "A", TanggalMasuk: "2026-01-05", GajiPokok: 1, Tunjangan: -1},
		{Nama: "A", TanggalMasuk: "2026-01-05", GajiPokok: 1, PTKP: "X/9"},
	}
	for i, k := range bad {
		if err := finance.ValidateKaryawan(&k); err == nil {
			t.Errorf("case %d harus ditolak", i)
		}
	}
}

func TestSummarizeKaryawan_HanyaHitungYangAktif(t *testing.T) {
	s := finance.SummarizeKaryawan([]finance.Karyawan{
		{Nama: "A", GajiPokok: 5_000_000, Tunjangan: 500_000, Aktif: true},
		{Nama: "B", GajiPokok: 4_000_000, Aktif: true},
		{Nama: "C", GajiPokok: 9_000_000, Aktif: false},
	})
	if s.TotalAktif != 2 || s.BebanGajiBulanan != 9_500_000 {
		t.Errorf("rekap: %d aktif, beban %d", s.TotalAktif, s.BebanGajiBulanan)
	}
	if len(s.PTKPCodes) == 0 {
		t.Error("daftar kode PTKP harus dikirim untuk form")
	}
}

// Regresi: kata "piutang" memuat "utang". Pemeriksaan yang salah urutan membuat SELURUH
// piutang terbaca sebagai utang, dan ini tidak terlihat dari jumlah baris yang berhasil
// diimpor — semua baris "berhasil", hanya arahnya terbalik.
func TestImporTagihan_PiutangTidakTerbacaSebagaiUtang(t *testing.T) {
	rows := [][]string{
		{"Jenis", "Nomor", "Pelanggan / Supplier", "Jatuh Tempo", "Nominal", "Status"},
		{"Piutang", "INV-1", "PT Pelanggan Setia", "2026-07-10", "5400000", "Belum Lunas"},
		{"Utang", "PO-1", "Supplier Kopi", "2026-09-08", "4800000", "Belum Lunas"},
		{"Hutang", "PO-2", "Supplier Gula", "2026-09-10", "1200000", "Belum Lunas"},
		{"", "INV-2", "Tanpa kolom jenis", "2026-08-01", "900000", ""},
		{"", "PO-3", "Nomor berawalan PO", "2026-08-01", "700000", ""},
	}
	d := finance.ParseOnboarding([]finance.ImportSheet{{Nama: "Piutang & Utang", Rows: rows}})
	if len(d.Errors) > 0 {
		t.Fatalf("tidak boleh ada error: %+v", d.Errors)
	}
	if len(d.Tagihan) != 5 {
		t.Fatalf("want 5 baris, got %d", len(d.Tagihan))
	}
	want := []string{
		finance.TagihanPiutang, finance.TagihanUtang, finance.TagihanUtang,
		finance.TagihanPiutang, // tanpa kolom jenis, nomor INV dianggap piutang
		finance.TagihanUtang,   // nomor berawalan PO dianggap utang
	}
	for i, w := range want {
		if d.Tagihan[i].Jenis != w {
			t.Errorf("baris %d (%s): jenis %q, want %q", i+1, d.Tagihan[i].Nomor, d.Tagihan[i].Jenis, w)
		}
	}
}

func TestImporStok_KolomOpsionalDanStokMinimum(t *testing.T) {
	rows := [][]string{
		{"SKU", "Nama Barang", "Kategori", "Jumlah", "Satuan", "Stok Minimum", "Harga Pokok", "Harga Jual"},
		{"RM-001", "Biji Kopi", "Bahan Baku", "42", "kg", "15", "185000", ""},
		{"RM-002", "Susu", "Bahan Baku", "18", "liter", "24", "21500", ""},
		{"FG-001", "Es Kopi Susu", "Produk Jadi", "0", "cup", "30", "6800", "18000"},
		{"", "Tanpa SKU & harga", "", "5", "pcs", "", "", ""},
	}
	d := finance.ParseOnboarding([]finance.ImportSheet{{Nama: "Stok & Gudang", Rows: rows}})
	if len(d.Errors) > 0 {
		t.Fatalf("kolom kosong tidak boleh jadi error: %+v", d.Errors)
	}
	if len(d.Stok) != 4 {
		t.Fatalf("want 4 baris, got %d", len(d.Stok))
	}
	// "Stok Minimum" juga memuat kata "stok"; kalau pemetaan kolomnya salah, nilai minimum
	// akan tertukar dengan jumlah.
	if d.Stok[0].Jumlah != 42 || d.Stok[0].StokMin != 15 {
		t.Errorf("jumlah %d stok_min %d, want 42 dan 15", d.Stok[0].Jumlah, d.Stok[0].StokMin)
	}
	if got := d.Stok[1].Status(); got != "Stok Menipis" {
		t.Errorf("18 dari minimum 24 harus menipis, got %q", got)
	}
	if got := d.Stok[2].Status(); got != "Habis" {
		t.Errorf("jumlah 0 harus habis, got %q", got)
	}
	if d.Stok[3].Kategori != "Barang" {
		t.Errorf("kategori kosong harus diisi default, got %q", d.Stok[3].Kategori)
	}
}

func TestParseTagihanFile_AbaikanSheetLain(t *testing.T) {
	// Berkas berisi beberapa sheet diunggah di halaman Piutang & Utang: hanya sheet
	// piutang/utang yang dibaca, sheet lain tidak boleh jadi error.
	var karyawan finance.ImportSheet
	for _, sh := range finance.OnboardingTemplate("") {
		if sh.Nama == "Karyawan" {
			karyawan = sh
		}
	}
	list, errs, nama := finance.ParseTagihanFile(append(finance.TagihanTemplate(), karyawan))
	if len(errs) != 0 {
		t.Fatalf("tidak boleh ada error, got %+v", errs)
	}
	if nama != "Piutang & Utang" {
		t.Errorf("sheet terbaca %q", nama)
	}
	if len(list) != 3 {
		t.Fatalf("3 baris contoh, got %d", len(list))
	}
	var piutang, utang int
	for _, x := range list {
		if x.Jenis == finance.TagihanPiutang {
			piutang++
		} else {
			utang++
		}
	}
	if piutang != 2 || utang != 1 {
		t.Errorf("want 2 piutang 1 utang, got %d/%d", piutang, utang)
	}
}

func TestParseStokFile_SheetTanpaNamaYangDikenali(t *testing.T) {
	rows := finance.StokTemplate()[0].Rows
	// "Sheet1" dari Google Sheets / Save As harus tetap terbaca selama itu satu-satunya sheet.
	stok, errs, _ := finance.ParseStokFile([]finance.ImportSheet{{Nama: "Sheet1", Rows: rows}})
	if len(errs) != 0 || len(stok) != 3 {
		t.Fatalf("satu sheet tanpa nama dikenali harus dipakai: %d item, errs %+v", len(stok), errs)
	}
	// Dua sheet asing sekaligus: tidak boleh menebak yang mana.
	_, errs, _ = finance.ParseStokFile([]finance.ImportSheet{{Nama: "Sheet1", Rows: rows}, {Nama: "Sheet2", Rows: rows}})
	if len(errs) == 0 {
		t.Error("dua sheet asing harus ditolak, bukan ditebak")
	}
}

func TestStokTemplate_RoundTrip(t *testing.T) {
	list, errs, _ := finance.ParseStokFile(finance.StokTemplate())
	if len(errs) != 0 {
		t.Fatalf("template sendiri harus bersih: %+v", errs)
	}
	s := finance.SummarizeStok(list)
	if s.TotalItem != 3 || s.Menipis != 1 {
		t.Errorf("rekap template: %+v", s)
	}
}

// Jenis di luar daftar dropdown diterima (dropdown hanya saran): barisnya jadi akun bernama
// jenis itu, uraiannya pindah ke keterangan; kode akun diberikan server dari CoA usaha.
func TestRinci_JenisBebas(t *testing.T) {
	sheets := finance.OnboardingTemplate("")
	set := func(nama string, baris, kol int, v string) {
		for _, sh := range sheets {
			if sh.Nama == nama {
				sh.Rows[baris-1][kol] = v
				return
			}
		}
		t.Fatalf("sheet %s tidak ada", nama)
	}
	set("Kas dan Bank", 8, 2, "Deposito Mandiri")
	set("Aset Lancar Lain", 7, 2, "Uang Jaminan")
	set("Utang", 8, 2, "Utang ke Pemegang Saham")
	set("Utang", 9, 2, "Kredit Koperasi")
	d := finance.ParseOnboarding(sheets)
	if len(d.Errors) != 0 || d.Opening == nil {
		t.Fatalf("jenis bebas harus diterima: %+v", d.Errors)
	}
	cari := func(nama string) finance.SaldoAkun {
		for _, r := range d.Opening.Rincian {
			if r.Nama == nama {
				return r
			}
		}
		t.Fatalf("baris %q tidak ada: %+v", nama, d.Opening.Rincian)
		return finance.SaldoAkun{}
	}
	if r := cari("Deposito Mandiri"); r.Kategori != finance.KatKas || r.Kode != "" || r.Nominal != 40_000_000 || !strings.Contains(r.Keterangan, "Bank BCA Operasional") {
		t.Errorf("deposito: %+v", r)
	}
	if r := cari("Uang Jaminan"); r.Kategori != finance.KatAsetLain || r.Kode != "" || !strings.Contains(r.Keterangan, "Sewa ruko") {
		t.Errorf("uang jaminan: %+v", r)
	}
	if r := cari("Utang ke Pemegang Saham"); r.Kategori != finance.KatKewajibanLain || r.Kode != "" {
		t.Errorf("utang pemegang saham: %+v", r)
	}
	if r := cari("Kredit Koperasi"); r.Kategori != finance.KatPinjaman {
		t.Errorf("kredit koperasi harus kategori pinjaman: %+v", r)
	}
	// Total per pos tidak berubah: hanya pengelompokan akunnya.
	if o := d.Opening; o.Kas != 45_000_000 || o.AsetLain != 7_050_000 || o.Pinjaman != 20_000_000 {
		t.Errorf("total pos berubah: %+v", o)
	}
}

// Status SEIMBANG tampil paling atas Neraca Saldo Awal (dan di Petunjuk), berwarna; kolom Jenis
// menerima ketikan bebas.
func TestOnboardingTemplate_StatusDiAtas(t *testing.T) {
	byNama := map[string]finance.ImportSheet{}
	for _, sh := range finance.OnboardingTemplate("") {
		byNama[sh.Nama] = sh
	}
	n := byNama["Neraca Saldo Awal"]
	if !strings.Contains(n.Rows[3][3], "SEIMBANG") || !slices.Contains(n.Format.Status, "D4") {
		t.Fatalf("status di D4: %q, status %v", n.Rows[3], n.Format.Status)
	}
	p := byNama["Petunjuk"]
	ada := false
	for i, row := range p.Rows {
		if len(row) > 2 && strings.Contains(row[2], "'Neraca Saldo Awal'!$D$4") {
			ada = slices.Contains(p.Format.Status, fmt.Sprintf("C%d", i+1))
		}
	}
	if !ada {
		t.Error("Petunjuk harus menampilkan status Neraca Saldo Awal berwarna")
	}
	for nama, kol := range map[string]int{"Kas dan Bank": 2, "Aset Lancar Lain": 2, "Utang": 2, "Modal": 2} {
		if f := byNama[nama].Format; !slices.Contains(f.Bebas, kol) {
			t.Errorf("%s: kolom Jenis harus bebas: %v", nama, f.Bebas)
		}
	}
}
