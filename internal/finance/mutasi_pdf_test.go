package finance

import "testing"

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
