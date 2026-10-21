package finance

import (
	"strings"
	"testing"
)

func TestUangTeks_DuaKebiasaan(t *testing.T) {
	for in, want := range map[string]int64{
		"1,500,000.00": 1_500_000, "1.500.000,00": 1_500_000, "150.000": 150_000,
		"2,500.50": 2_501, "10,000": 10_000, "Rp1.250.000": 1_250_000, "-75.000,00": 75_000,
	} {
		if got, ok := uangTeks(in); !ok || got != want {
			t.Errorf("uangTeks(%q) = %d, %v; want %d", in, got, ok, want)
		}
	}
}

// Cetakan gaya BCA: tanggal tanpa tahun, satu kolom mutasi + saldo, arah dari penanda DB /
// perubahan saldo, keterangan berlanjut ke baris berikutnya.
func TestMutasiDariTeks_GayaBCA(t *testing.T) {
	baris := []string{
		"REKENING TAHAPAN",
		"PERIODE : SEPTEMBER 2026",
		"TANGGAL KETERANGAN CBG MUTASI SALDO",
		"01/09 SALDO AWAL 10,000,000.00",
		"02/09 TRSF E-BANKING CR 0209/FTSCY/WS95031 1,500,000.00 11,500,000.00",
		"BUDI SANTOSO",
		"03/09 TARIKAN ATM 0938 500,000.00 DB 11,000,000.00",
		"05/09 KR OTOMATIS 2,000,000.00 13,000,000.00",
		"LLG-ANTAR BANK",
		"Bersambung ke halaman berikut",
		"SALDO AKHIR 13,000,000.00",
	}
	rows := MutasiDariTeks(baris)
	h := ParseMutasi(rows, 2026, nil)
	if len(h.Errors) != 0 || len(h.Rows) != 3 {
		t.Fatalf("rows=%v errors=%v hasil=%+v", rows, h.Errors, h.Rows)
	}
	want := []struct {
		tgl, arah string
		n         int64
	}{{"2026-09-02", "masuk", 1_500_000}, {"2026-09-03", "keluar", 500_000}, {"2026-09-05", "masuk", 2_000_000}}
	for i, w := range want {
		r := h.Rows[i]
		if r.Tanggal != w.tgl || r.Arah != w.arah || r.Nominal != w.n {
			t.Errorf("baris %d = %s %s %d; want %+v", i, r.Tanggal, r.Arah, r.Nominal, w)
		}
	}
	if h.Rows[0].Keterangan != "TRSF E-BANKING CR 0209/FTSCY/WS95031 BUDI SANTOSO" {
		t.Errorf("keterangan lanjutan: %q", h.Rows[0].Keterangan)
	}
}

// Cetakan gaya Mandiri/BRI: tanggal lengkap + jam, kolom Debet, Kredit, Saldo terpisah.
func TestMutasiDariTeks_KolomDebetKredit(t *testing.T) {
	baris := []string{
		"Tanggal Keterangan Debit Kredit Saldo",
		"01/10/2026 08:15:02 Transfer dari ANI 0.00 750,000.00 1,750,000.00",
		"02 Okt 2026 Biaya Admin 12.500,00 0,00 1.737.500,00",
	}
	h := ParseMutasi(MutasiDariTeks(baris), 2026, nil)
	if len(h.Errors) != 0 || len(h.Rows) != 2 {
		t.Fatalf("errors=%v rows=%+v", h.Errors, h.Rows)
	}
	if r := h.Rows[0]; r.Tanggal != "2026-10-01" || r.Arah != "masuk" || r.Nominal != 750_000 || r.Keterangan != "Transfer dari ANI" {
		t.Errorf("baris 1: %+v", r)
	}
	if r := h.Rows[1]; r.Tanggal != "2026-10-02" || r.Arah != "keluar" || r.Nominal != 12_500 {
		t.Errorf("baris 2: %+v", r)
	}
}

// Arah yang tidak bisa dipastikan dilaporkan, bukan diam-diam dianggap uang masuk.
func TestMutasiDariTeks_ArahTidakJelasDilaporkan(t *testing.T) {
	h := ParseMutasi(MutasiDariTeks([]string{"04/09/2026 PEMBAYARAN QRIS 25.000,00"}), 2026, nil)
	if len(h.Rows) != 0 || len(h.Errors) != 1 {
		t.Fatalf("rows=%+v errors=%v", h.Rows, h.Errors)
	}
}

func TestMutasiDariTeks_TanpaTransaksi(t *testing.T) {
	if MutasiDariTeks([]string{"Surat pemberitahuan", "Terima kasih"}) != nil {
		t.Fatal("teks tanpa baris bertanggal harus nil")
	}
}

// Cetakan e-Statement BRImo: tanggal dd/mm/yy + jam, kolom Teller, Debet/Kredit/Saldo, dan jam
// cetak di kaki halaman berformat bulan/tanggal ("07/18/2025 08:58:23") yang bukan transaksi.
func TestMutasiDariTeks_GayaBRImo(t *testing.T) {
	baris := []string{
		"Halaman 1 dari 2",
		"Periode Transaksi : 01/01/25 - 31/01/25",
		"Tanggal Transaksi Uraian Transaksi Teller Debet Kredit Saldo",
		"01/01/25 12:40:51 Transfer Dari Budi via BRImo 8888110 0.00 999,000.00 17,237,544.00",
		"07/18/2025 08:58:23",
		"Halaman 2 dari 2",
		"15/01/25 23:59:59 Pajak 496.00 0.00 17,237,048.00",
		"29/01/25 14:40:20 Transfer BI-Fast dari BANK MANDIRI (PERSERO), PT 8888698 0.00 999,999.00 18,237,047.00",
		"- SITI",
		"Saldo Awal Total Transaksi Debet Total Transaksi Kredit Saldo Akhir",
		"16,238,544.00 496.00 1,998,999.00 18,237,047.00",
		"07/18/2025 08:58:23",
	}
	h := ParseMutasi(MutasiDariTeks(baris), 2026, nil)
	if len(h.Errors) != 0 || len(h.Rows) != 3 || len(h.Peringatan) != 0 {
		t.Fatalf("errors=%v peringatan=%v rows=%+v", h.Errors, h.Peringatan, h.Rows)
	}
	if h.Rows[0].Tanggal != "2025-01-01" || h.Rows[1].Tanggal != "2025-01-15" || h.Rows[2].Tanggal != "2025-01-29" {
		t.Errorf("tanggal: %s %s %s", h.Rows[0].Tanggal, h.Rows[1].Tanggal, h.Rows[2].Tanggal)
	}
	if h.TotalMasuk != 1_998_999 || h.TotalKeluar != 496 {
		t.Errorf("total masuk %d keluar %d", h.TotalMasuk, h.TotalKeluar)
	}
	if h.Rows[2].Keterangan != "Transfer BI-Fast dari BANK MANDIRI (PERSERO), PT 8888698 - SITI" {
		t.Errorf("keterangan lanjutan: %q", h.Rows[2].Keterangan)
	}
}

// Variasi tata letak antar bank. Tiap kasus: baris teks PDF → transaksi yang diharapkan.
func TestMutasiDariTeks_BerbagaiBank(t *testing.T) {
	type tx struct {
		tgl, arah string
		n         int64
	}
	for _, c := range []struct {
		nama  string
		baris []string
		want  []tx
	}{
		{"BCA Bisnis: tanggal ISO + jam, IDR, DB/CR di kanan", []string{
			"Opening Balance : IDR 1,946,256.00",
			"Date FT Number Description Currency Amount DB CR Balance",
			"2026-08-31 17:05:01 8284488867.73.20260831 Biaya Administrasi IDR 15,000.00 DB 1,931,256.00",
			"Page 1/1",
		}, []tx{{"2026-08-31", "keluar", 15_000}}},
		{"Nomor urut di depan, kolom Debit/Kredit/Saldo", []string{
			"No Tanggal Uraian Debit Kredit Saldo",
			"1 03/08/2026 Setoran Tunai 0,00 2.500.000,00 7.500.000,00",
			"2 04/08/2026 Transfer ke Supplier 1.000.000,00 0,00 6.500.000,00",
		}, []tx{{"2026-08-03", "masuk", 2_500_000}, {"2026-08-04", "keluar", 1_000_000}}},
		{"Bank digital: tanda +/− dan Rp menempel", []string{
			"05 Agu 2026 Transfer masuk dari Andi +Rp1.250.000",
			"06 Agu 2026 Pembayaran QRIS -Rp45.000",
		}, []tx{{"2026-08-05", "masuk", 1_250_000}, {"2026-08-06", "keluar", 45_000}}},
		{"Penanda di kiri angka & kata Kredit/Debit", []string{
			"07/08/2026 Bunga Kredit 12.345,00",
			"08/08/2026 Biaya Debit 5.000,00",
		}, []tx{{"2026-08-07", "masuk", 12_345}, {"2026-08-08", "keluar", 5_000}}},
		{"Tanggal tanpa tahun: tahun dari kop periode, arah dari saldo", []string{
			"PERIODE : AGUSTUS 2025",
			"SALDO AWAL 1,000,000.00",
			"10/08 TRANSFER DARI BUDI 500,000.00 1,500,000.00",
			"11/08 TARIK TUNAI 200,000.00 1,300,000.00",
		}, []tx{{"2025-08-10", "masuk", 500_000}, {"2025-08-11", "keluar", 200_000}}},
	} {
		rows, peringatan := MutasiDariTeksCek(c.baris)
		h := ParseMutasi(rows, 2026, nil)
		if len(h.Errors) != 0 || len(h.Rows) != len(c.want) || len(peringatan) != 0 {
			t.Errorf("%s: errors=%v peringatan=%v rows=%+v", c.nama, h.Errors, peringatan, h.Rows)
			continue
		}
		for i, w := range c.want {
			if r := h.Rows[i]; r.Tanggal != w.tgl || r.Arah != w.arah || r.Nominal != w.n {
				t.Errorf("%s baris %d: %s %s %d; want %+v", c.nama, i+1, r.Tanggal, r.Arah, r.Nominal, w)
			}
		}
	}
}

// Saldo berjalan yang tidak cocok (angka salah baca) dilaporkan per tanggal.
func TestMutasiDariTeks_SaldoTidakCocokDiperingatkan(t *testing.T) {
	_, p := MutasiDariTeksCek([]string{
		"SALDO AWAL 1,000,000.00",
		"10/08/2026 TRANSFER CR 500,000.00 1,500,000.00",
		"11/08/2026 TARIK TUNAI DB 200,000.00 1,350,000.00",
		"12/08/2026 TRANSFER CR 100,000.00 1,450,000.00",
		"13/08/2026 TRANSFER CR 100,000.00 1,550,000.00",
	})
	if len(p) != 1 || !strings.Contains(p[0], "11/08/2026") {
		t.Fatalf("peringatan saldo: %v", p)
	}
}
