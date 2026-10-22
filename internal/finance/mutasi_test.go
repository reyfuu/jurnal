package finance_test

import (
	"strings"
	"testing"

	"pajak/internal/finance"
)

// Bentuk BCA: ada baris kop sebelum header, satu kolom "Jumlah" dengan penanda CR/DB
// menempel di angkanya, dan tanggal tanpa tahun.
func mutasiBCA() [][]string {
	return [][]string{
		{"REKENING", "1234567890"},
		{"NAMA", "KEDAI KOPI SENJA"},
		{"PERIODE", "SEPTEMBER 2026"},
		{},
		{"TANGGAL", "KETERANGAN", "CBG", "JUMLAH", "SALDO"},
		{"01/09", "TRSF E-BANKING CR 0109/FTSCY PT KARYA DIGITAL", "0000", "5.400.000,00 CR", "52.400.000,00"},
		{"02/09", "QRIS CR 0209 MERCHANT SENJA", "0000", "1.850.000,00 CR", "54.250.000,00"},
		{"03/09", "BIAYA ADM", "0000", "15.000,00 DB", "54.235.000,00"},
		{"05/09", "TRSF E-BANKING DB 0509/FTSCY ROASTERY GAYO LESTARI", "0000", "4.800.000,00 DB", "49.435.000,00"},
		{"10/09", "PEMBAYARAN GAJI KARYAWAN SEPTEMBER", "0000", "14.350.000,00 DB", "35.085.000,00"},
		{"15/09", "PLN PREPAID 0151234567", "0000", "500.000,00 DB", "34.585.000,00"},
		{"20/09", "GOFOOD SETTLEMENT 2009", "0000", "3.200.000,00 CR", "37.785.000,00"},
		{"28/09", "BUNGA", "0000", "12.500,00 CR", "37.797.500,00"},
		{"28/09", "PAJAK BUNGA", "0000", "2.500,00 DB", "37.795.000,00"},
		{"30/09", "TARIK TUNAI ATM 0930", "0000", "2.000.000,00 DB", "35.795.000,00"},
		{},
		{"", "SALDO AKHIR", "", "", "35.795.000,00"},
	}
}

func rowOf(t *testing.T, h finance.MutasiHasil, cari string) finance.MutasiRow {
	t.Helper()
	for _, r := range h.Rows {
		if strings.Contains(r.Keterangan, cari) {
			return r
		}
	}
	t.Fatalf("baris dengan keterangan %q tidak ada", cari)
	return finance.MutasiRow{}
}

func TestParseMutasi_BentukBCA(t *testing.T) {
	h := finance.ParseMutasi(mutasiBCA(), 2026, nil)
	if len(h.Errors) != 0 {
		t.Fatalf("tidak boleh ada error: %+v", h.Errors)
	}
	if len(h.Rows) != 10 {
		t.Fatalf("10 baris mutasi, got %d", len(h.Rows))
	}
	// Kop di atas header dan baris "SALDO AKHIR" di bawah tidak boleh ikut terbaca.
	if h.TotalMasuk != 10_462_500 {
		t.Errorf("total masuk = %d, want 10.462.500", h.TotalMasuk)
	}
	if h.TotalKeluar != 21_667_500 {
		t.Errorf("total keluar = %d, want 21.667.500", h.TotalKeluar)
	}
	for _, c := range []struct {
		cari, jenis, arah, keyakinan string
	}{
		{"PT KARYA DIGITAL", "Penjualan", "masuk", finance.KeyakinanRendah},
		{"QRIS", "Penjualan", "masuk", finance.KeyakinanTinggi},
		{"BIAYA ADM", "Biaya Lain", "keluar", finance.KeyakinanTinggi},
		{"ROASTERY", "Beli Bahan / Stok", "keluar", finance.KeyakinanSedang},
		{"GAJI", "Gaji & Upah", "keluar", finance.KeyakinanTinggi},
		{"PLN", "Listrik, Air, Internet", "keluar", finance.KeyakinanTinggi},
		{"GOFOOD", "Penjualan", "masuk", finance.KeyakinanTinggi},
		{"TARIK TUNAI", "Ambil Uang Pribadi", "keluar", finance.KeyakinanRendah},
	} {
		r := rowOf(t, h, c.cari)
		if r.Jenis != c.jenis || r.Arah != c.arah || r.Keyakinan != c.keyakinan {
			t.Errorf("%q → %q/%s/%s, want %q/%s/%s", c.cari, r.Jenis, r.Arah, r.Keyakinan, c.jenis, c.arah, c.keyakinan)
		}
	}
	// Tanggal tanpa tahun memakai tahun periode berkas.
	if r := rowOf(t, h, "QRIS"); r.Tanggal != "2026-09-02" {
		t.Errorf("tanggal tanpa tahun = %q, want 2026-09-02", r.Tanggal)
	}
}

// "PAJAK BUNGA" memuat kata "bunga" (pendapatan) sekaligus "pajak" (beban). Aturan yang lebih
// khas harus menang, dan arah uangnya memang keluar.
func TestTebakJenis_PajakBungaBukanPendapatan(t *testing.T) {
	j, yakin, _ := finance.TebakJenis("PAJAK BUNGA", "keluar")
	if j != "Pajak & Izin" || yakin != finance.KeyakinanTinggi {
		t.Errorf("PAJAK BUNGA keluar → %q/%s, want Pajak & Izin/tinggi", j, yakin)
	}
	if j, _, _ := finance.TebakJenis("BUNGA", "masuk"); j != "Pendapatan Lain" {
		t.Errorf("BUNGA masuk → %q, want Pendapatan Lain", j)
	}
}

// Kata pendek dicocokkan sebagai kata utuh: "adm" tidak boleh cocok di tengah kata lain.
func TestTebakJenis_KataPendekTidakCocokDiTengahKata(t *testing.T) {
	// "ADMIRAL" tidak boleh cocok dengan kata kunci "adm". Jenis yang keluar sama-sama
	// "Biaya Lain" — yang membedakan keyakinannya: ini jatuh ke default arah keluar
	// (rendah, wajib diperiksa), bukan hasil kata kunci yang diklaim pasti.
	j, yakin, alasan := finance.TebakJenis("PEMBAYARAN KE PT ADMIRAL NUSANTARA", "keluar")
	if yakin != finance.KeyakinanRendah {
		t.Errorf(`"ADMIRAL" dibaca sebagai kata kunci: %q/%s — %s`, j, yakin, alasan)
	}
	j, yakin, _ = finance.TebakJenis("BIAYA ADM", "keluar")
	if j != "Biaya Lain" || yakin != finance.KeyakinanTinggi {
		t.Errorf("BIAYA ADM → %q/%s, want Biaya Lain/tinggi", j, yakin)
	}
}

// Arah uang menentukan aturan mana yang berlaku: kata "pinjaman" masuk = terima pinjaman,
// keluar = bayar cicilan.
func TestTebakJenis_ArahMembedakanAturan(t *testing.T) {
	if j, _, _ := finance.TebakJenis("PENCAIRAN KUR BRI", "masuk"); j != "Terima Pinjaman" {
		t.Errorf("pencairan KUR masuk → %q", j)
	}
	if j, _, _ := finance.TebakJenis("ANGSURAN KUR BRI", "keluar"); j != "Bayar Cicilan" {
		t.Errorf("angsuran KUR keluar → %q", j)
	}
}

// Bentuk Mandiri: kolom Debet & Kredit terpisah, tanggal lengkap.
func TestParseMutasi_KolomDebetKreditTerpisah(t *testing.T) {
	rows := [][]string{
		{"Tanggal Transaksi", "Uraian Transaksi", "Debet", "Kredit", "Saldo"},
		{"01/09/2026", "SETORAN TUNAI KASIR", "", "2.500.000", "40.000.000"},
		{"02/09/2026", "BIAYA ADMINISTRASI", "15.000", "", "39.985.000"},
		{"03/09/2026", "IPL SEWA RUKO SEPTEMBER", "8.000.000", "", "31.985.000"},
	}
	h := finance.ParseMutasi(rows, 0, nil)
	if len(h.Errors) != 0 || len(h.Rows) != 3 {
		t.Fatalf("%d baris, errors %+v", len(h.Rows), h.Errors)
	}
	if h.Rows[0].Arah != "masuk" || h.Rows[0].Nominal != 2_500_000 {
		t.Errorf("baris kredit: %+v", h.Rows[0])
	}
	if h.Rows[2].Jenis != "Sewa Tempat" || h.Rows[2].Arah != "keluar" {
		t.Errorf("sewa ruko: %+v", h.Rows[2])
	}
}

func TestParseMutasi_TandaMinusSebagaiArah(t *testing.T) {
	rows := [][]string{
		{"Date", "Description", "Amount"},
		{"2026-09-01", "QRIS SETTLEMENT", "1500000"},
		{"2026-09-02", "BIAYA ADM", "-15000"},
		{"2026-09-03", "PEMBELIAN BAHAN", "(250000)"},
	}
	h := finance.ParseMutasi(rows, 0, nil)
	if len(h.Rows) != 3 {
		t.Fatalf("%d baris, errors %+v", len(h.Rows), h.Errors)
	}
	for i, want := range []string{"masuk", "keluar", "keluar"} {
		if h.Rows[i].Arah != want {
			t.Errorf("baris %d arah %q, want %q", i+1, h.Rows[i].Arah, want)
		}
	}
	if h.Rows[2].Nominal != 250_000 {
		t.Errorf("nominal dalam tanda kurung = %d, want 250000", h.Rows[2].Nominal)
	}
}

func TestParseMutasi_DuplikatTidakDicentang(t *testing.T) {
	existing := []finance.Transaction{{Tanggal: "2026-09-03", Jenis: "Biaya Lain", Nominal: 15_000, Lunas: true}}
	h := finance.ParseMutasi(mutasiBCA(), 2026, existing)
	if h.Duplikat != 1 {
		t.Fatalf("1 dugaan duplikat, got %d", h.Duplikat)
	}
	r := rowOf(t, h, "BIAYA ADM")
	if r.Pilih {
		t.Error("baris yang diduga duplikat tidak boleh dicentang otomatis")
	}
	// Baris lain tetap dicentang.
	if !rowOf(t, h, "QRIS").Pilih {
		t.Error("baris non-duplikat harus dicentang")
	}
}

func TestParseMutasi_HeaderTidakDikenali(t *testing.T) {
	h := finance.ParseMutasi([][]string{{"A", "B"}, {"1", "2"}}, 0, nil)
	if len(h.Errors) != 1 || !strings.Contains(h.Errors[0].Pesan, "kolom mutasi tidak dikenali") {
		t.Errorf("harus menolak dengan jelas: %+v", h.Errors)
	}
	if len(h.Rows) != 0 {
		t.Error("tidak boleh ada baris dari berkas yang tak dikenali")
	}
}

func TestMutasiKeTransaksi(t *testing.T) {
	h := finance.ParseMutasi(mutasiBCA(), 2026, nil)
	txs, errs := finance.MutasiKeTransaksi(h.Rows)
	if len(errs) != 0 {
		t.Fatalf("errors: %+v", errs)
	}
	if len(txs) != 10 {
		t.Fatalf("%d transaksi, want 10", len(txs))
	}
	for _, tx := range txs {
		if tx.Channel != "Mutasi Bank" {
			t.Errorf("asal transaksi harus ditandai, got channel %q", tx.Channel)
		}
		if !tx.Lunas {
			t.Error("mutasi bank selalu sudah lunas: uangnya sudah berpindah")
		}
	}
}

func TestMutasiKeTransaksi_HanyaYangDipilih(t *testing.T) {
	rows := []finance.MutasiRow{
		{Baris: 2, Tanggal: "2026-09-01", Jenis: "Penjualan", Nominal: 100_000, Arah: "masuk", Pilih: true},
		{Baris: 3, Tanggal: "2026-09-02", Jenis: "Penjualan", Nominal: 200_000, Arah: "masuk", Pilih: false},
	}
	txs, _ := finance.MutasiKeTransaksi(rows)
	if len(txs) != 1 || txs[0].Nominal != 100_000 {
		t.Errorf("hanya baris terpilih yang dibukukan: %+v", txs)
	}
}

// Jenis yang arahnya bertentangan dengan arah uang di rekening harus ditolak, bukan
// dibukukan terbalik.
func TestMutasiKeTransaksi_ArahBertentanganDitolak(t *testing.T) {
	rows := []finance.MutasiRow{
		{Baris: 7, Tanggal: "2026-09-01", Jenis: "Penjualan", Nominal: 100_000, Arah: "keluar", Pilih: true},
	}
	txs, errs := finance.MutasiKeTransaksi(rows)
	if len(txs) != 0 {
		t.Error("tidak boleh dibukukan")
	}
	if len(errs) != 1 || !strings.Contains(errs[0].Pesan, "uang masuk") {
		t.Errorf("pesan harus menjelaskan pertentangannya: %+v", errs)
	}
}

func TestParseMutasi_RingkasanPerJenis(t *testing.T) {
	h := finance.ParseMutasi(mutasiBCA(), 2026, nil)
	per := map[string]finance.MutasiGrup{}
	for _, g := range h.Ringkasan {
		per[g.Jenis] = g
	}
	if g := per["Penjualan"]; g.Jumlah != 3 || g.Total != 10_450_000 {
		t.Errorf("Penjualan: %d baris, total %d; want 3 baris, 10.450.000", g.Jumlah, g.Total)
	}
	if g := per["Pajak & Izin"]; g.Jumlah != 1 || g.Total != 2_500 {
		t.Errorf("Pajak & Izin: %+v", g)
	}
	if h.PerluDiperiksa != 2 {
		t.Errorf("baris berkeyakinan rendah = %d, want 2 (transfer masuk tanpa penanda + tarik tunai)", h.PerluDiperiksa)
	}
}

func TestMutasi_KelompokRendahDanSaranAI(t *testing.T) {
	h := finance.MutasiHasil{Rows: []finance.MutasiRow{
		{Keterangan: "TRSF E-BANKING CR 1234567890 BUDI SANTOSO", Arah: "masuk", Nominal: 500_000, Jenis: "Penjualan", Keyakinan: finance.KeyakinanRendah},
		{Keterangan: "TRSF E-BANKING CR 9876543210 BUDI SANTOSO", Arah: "masuk", Nominal: 750_000, Jenis: "Penjualan", Keyakinan: finance.KeyakinanRendah},
		{Keterangan: "BIAYA ADM", Arah: "keluar", Nominal: 10_000, Jenis: "Biaya Lain", Keyakinan: finance.KeyakinanTinggi},
		{Keterangan: "DB 0011 TOKO SINAR JAYA", Arah: "keluar", Nominal: 2_000_000, Jenis: "Biaya Lain", Keyakinan: finance.KeyakinanRendah},
	}, PerluDiperiksa: 3}
	g := h.KelompokRendah()
	if len(g) != 2 { // dua baris Budi dengan nomor rekening berbeda = satu kelompok
		t.Fatalf("kelompok = %+v", g)
	}
	if g[0].Keterangan != "TRSF E-BANKING CR # BUDI SANTOSO" || len(g[0].Baris) != 2 {
		t.Errorf("nomor panjang harus disamarkan & baris digabung: %+v", g[0])
	}
	h.TerapkanSaran(map[string]finance.SaranJenis{
		g[0].Kunci: {Jenis: "Setor Modal", Alasan: "transfer pribadi pemilik"},
		g[1].Kunci: {Jenis: "Penjualan", Alasan: "salah arah"}, // Penjualan bukan jenis keluar → diabaikan
	})
	if r := h.Rows[0]; r.Jenis != "Setor Modal" || r.Keyakinan != finance.KeyakinanAI || r.Alasan != "Saran AI: transfer pribadi pemilik" {
		t.Errorf("baris 1 = %+v", r)
	}
	if h.Rows[3].Keyakinan != finance.KeyakinanRendah || h.Rows[2].Keyakinan != finance.KeyakinanTinggi {
		t.Error("saran salah arah / baris yakin tidak boleh berubah")
	}
	if h.PerluDiperiksa != 1 || h.DibantuAI != 2 {
		t.Errorf("perlu diperiksa %d, dibantu AI %d", h.PerluDiperiksa, h.DibantuAI)
	}
}

func tanggalSemua(h finance.MutasiHasil) string {
	var s []string
	for _, r := range h.Rows {
		s = append(s, r.Tanggal)
	}
	return strings.Join(s, ",")
}

func TestParseMutasi_TanggalHariBulanDanPerBulan(t *testing.T) {
	h := finance.ParseMutasi([][]string{
		{"Tanggal", "Keterangan", "Debet", "Kredit"},
		{"15/01/2026", "QRIS penjualan", "", "100.000"},
		{"31/01/2026", "Listrik", "50.000", ""},
		{"03/02/2026", "QRIS penjualan", "", "70.000"},
	}, 2026, nil)
	if got := tanggalSemua(h); got != "2026-01-15,2026-01-31,2026-02-03" {
		t.Fatalf("tanggal %s", got)
	}
	if len(h.PerBulan) != 2 || h.PerBulan[0].Bulan != "2026-01" || h.PerBulan[0].Jumlah != 2 || h.PerBulan[1].Label != "Februari 2026" {
		t.Errorf("per bulan %+v", h.PerBulan)
	}
}

func TestParseMutasi_FormatBulanTanggalTerdeteksi(t *testing.T) {
	// 01/25/2026 tidak mungkin hari/bulan, jadi seluruh berkas dibaca bulan/hari.
	h := finance.ParseMutasi([][]string{
		{"Date", "Description", "Debit", "Credit"},
		{"01/05/2026", "Transfer masuk", "", "100.000"},
		{"01/25/2026", "Transfer masuk", "", "100.000"},
		{"02/03/2026", "Transfer masuk", "", "100.000"},
	}, 2026, nil)
	if got := tanggalSemua(h); got != "2026-01-05,2026-01-25,2026-02-03" {
		t.Fatalf("tanggal %s", got)
	}
	if !strings.Contains(strings.Join(h.Peringatan, " "), "bulan/tanggal") {
		t.Errorf("harus ada peringatan format terbalik: %v", h.Peringatan)
	}
}

func TestParseMutasi_LintasTahunTanpaTahun(t *testing.T) {
	// Cetakan 25 Des – 5 Jan tanpa tahun; tahun yang diisi = tahun Januari-nya.
	h := finance.ParseMutasi([][]string{
		{"TANGGAL", "KETERANGAN", "JUMLAH"},
		{"28/12", "QRIS CR", "100.000,00 CR"},
		{"31/12", "QRIS CR", "100.000,00 CR"},
		{"02/01", "QRIS CR", "100.000,00 CR"},
	}, 2026, nil)
	if got := tanggalSemua(h); got != "2025-12-28,2025-12-31,2026-01-02" {
		t.Fatalf("tanggal %s", got)
	}
}

func TestParseMutasi_PeringatanTanggalMasaDepan(t *testing.T) {
	h := finance.ParseMutasi([][]string{
		{"Tanggal", "Keterangan", "Debet", "Kredit"},
		{"15/01/2099", "QRIS", "", "100.000"},
	}, 2099, nil)
	if !strings.Contains(strings.Join(h.Peringatan, " "), "masa depan") {
		t.Errorf("peringatan %v", h.Peringatan)
	}
}
