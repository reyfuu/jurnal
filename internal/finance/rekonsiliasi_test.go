package finance

import (
	"strings"
	"testing"
)

func TestValidate_MasaPajak(t *testing.T) {
	s := Transaction{Tanggal: "2026-07-10", Jenis: JenisSetorPPhFinal, Nominal: 500_000, Lunas: true}
	if err := Validate(&s); err != nil || s.MasaPajak != "2026-06" {
		t.Fatalf("bawaan masa = bulan sebelum setor: %q %v", s.MasaPajak, err)
	}
	s.MasaPajak = "2026-08"
	if err := Validate(&s); err == nil {
		t.Fatal("masa sesudah bulan setor harus ditolak")
	}
	s.MasaPajak = "2026-13"
	if err := Validate(&s); err == nil {
		t.Fatal("masa tidak sah harus ditolak")
	}
	j := Transaction{Tanggal: "2026-07-10", Jenis: "Penjualan", Nominal: 1, Lunas: true, MasaPajak: "2026-06"}
	if err := Validate(&j); err != nil || j.MasaPajak != "" {
		t.Fatalf("masa hanya untuk setoran pajak: %q %v", j.MasaPajak, err)
	}
}

// OP: Rp100 jt/bulan, kuota bebas Rp500 jt habis di Mei, Juni mulai terutang Rp500 rb/bulan.
func TestRekonsiliasiTahun_OP(t *testing.T) {
	var omzet [12]int64
	for i := range 10 { // Januari–Oktober
		omzet[i] = 100_000_000
	}
	setor := func(tgl string, n int64, masa string) Transaction {
		return Transaction{Tanggal: tgl, Jenis: JenisSetorPPhFinal, Nominal: n, Lunas: true, MasaPajak: masa}
	}
	txs := []Transaction{
		setor("2026-07-10", 500_000, "2026-06"),
		setor("2026-08-14", 400_000, "2026-07"),
		setor("2026-09-12", 600_000, "2026-08"),
		{Tanggal: "2026-10-05", Jenis: "Pajak & Izin", Nominal: 500_000, Lunas: true},
		setor("2025-12-10", 999, "2025-11"), // tahun lain: diabaikan
	}
	r := RekonsiliasiTahun(2026, omzet, txs, BusinessProfile{BentukUsaha: "op"}, "2026-10-23")
	if r.Subjek != "OP" || r.BebasOP != 500_000_000 || len(r.Bulan) != 12 {
		t.Fatalf("kepala: %+v", r)
	}
	mau := map[int]struct {
		status          string
		terutang, setor int64
		pajakLain       int64
	}{
		4:  {"nihil", 0, 0, 0},
		5:  {"cocok", 500_000, 500_000, 0},
		6:  {"kurang", 500_000, 400_000, 0},
		7:  {"lebih", 500_000, 600_000, 0},
		8:  {"kurang", 500_000, 0, 500_000},
		9:  {"berjalan", 500_000, 0, 0},
		11: {"berjalan", 0, 0, 0},
	}
	for i, m := range mau {
		b := r.Bulan[i]
		if b.Status != m.status || b.PPhTerutang != m.terutang || b.PPhDisetor != m.setor || b.PajakLain != m.pajakLain {
			t.Errorf("bulan %d (%s): %+v, mau %+v", i+1, b.Masa, b, m)
		}
	}
	if b := r.Bulan[8]; b.JatuhTempo != "2026-10-15" || b.Selisih != -500_000 {
		t.Errorf("September: %+v", b)
	}
	if r.TotalOmzet != 1_000_000_000 || r.TotalTerutang != 2_500_000 || r.TotalDisetor != 1_500_000 || r.KurangBayar != 500_000 || r.Selesai || r.PerluDicek != 3 {
		t.Errorf("total: %+v", r)
	}
	if len(r.Peringatan) != 0 {
		t.Errorf("OP di bawah Rp4,8 M tanpa peringatan: %v", r.Peringatan)
	}
}

// CV: tanpa kuota bebas, terutang sejak rupiah pertama; peringatan jangka waktu PP 55 dan
// batas Rp4,8 M; setoran dalam toleransi pembulatan dianggap cocok; belum jatuh tempo.
func TestRekonsiliasiTahun_CV(t *testing.T) {
	var omzet [12]int64
	omzet[0], omzet[8] = 5_000_000_000, 10_000_000
	txs := []Transaction{
		{Tanggal: "2026-02-15", Jenis: JenisSetorPPhFinal, Nominal: 24_999_950, Lunas: true, MasaPajak: "2026-01"},
	}
	r := RekonsiliasiTahun(2026, omzet, txs, BusinessProfile{BentukUsaha: "cv"}, "2026-10-10")
	if r.Subjek != "Badan" || r.BebasOP != 0 {
		t.Fatalf("subjek: %+v", r)
	}
	if b := r.Bulan[0]; b.Status != "cocok" || b.PPhTerutang != 25_000_000 {
		t.Errorf("Januari: %+v", b)
	}
	if b := r.Bulan[8]; b.Status != "belum_jatuh_tempo" || b.PPhTerutang != 50_000 {
		t.Errorf("September sebelum tgl 15 Oktober: %+v", b)
	}
	teks := strings.Join(r.Peringatan, " | ")
	if !strings.Contains(teks, "jangka waktu") || !strings.Contains(teks, "4,8 miliar") {
		t.Errorf("peringatan: %v", r.Peringatan)
	}
}

// PKP: PPN Keluaran per bulan transaksi, Setor PPN per masa (data lama tanpa masa = bulan
// sebelum tanggal setor).
func TestRekonsiliasiTahun_PPN(t *testing.T) {
	txs := []Transaction{
		{Tanggal: "2026-03-20", Jenis: "PPN Keluaran", Nominal: 1_100_000},
		{Tanggal: "2026-04-10", Jenis: "Setor PPN", Nominal: 1_100_000, Lunas: true},
	}
	r := RekonsiliasiTahun(2026, [12]int64{}, txs, BusinessProfile{BentukUsaha: "pt_perorangan", PKP: true}, "2026-10-23")
	if b := r.Bulan[2]; !r.PKP || b.PPNKeluaran != 1_100_000 || b.PPNDisetor != 1_100_000 {
		t.Fatalf("PPN Maret: %+v", b)
	}
	if r.Subjek != "Badan" {
		t.Fatalf("PT perorangan = badan tanpa kuota bebas: %s", r.Subjek)
	}
}

// Masa Desember dibayar Januari tahun berikutnya tetap masuk Desember; tahun selesai setelah
// jatuh tempo Desember (15 Januari).
func TestRekonsiliasiTahun_DesemberDanSelesai(t *testing.T) {
	var omzet [12]int64
	omzet[11] = 10_000_000
	txs := []Transaction{{Tanggal: "2027-01-10", Jenis: JenisSetorPPhFinal, Nominal: 50_000, Lunas: true, MasaPajak: "2026-12"}}
	r := RekonsiliasiTahun(2026, omzet, txs, BusinessProfile{BentukUsaha: "pt_perorangan"}, "2027-01-20")
	if b := r.Bulan[11]; b.Status != "cocok" || b.JatuhTempo != "2027-01-15" || !r.Selesai || r.KurangBayar != 0 {
		t.Fatalf("Desember: %+v selesai=%v kurang=%d", b, r.Selesai, r.KurangBayar)
	}
	if d := r.Daftar(); d.Baris[12][8] != "Cocok (SPT Tahunan)" {
		t.Fatalf("baris setahun: %v", d.Baris[12])
	}
}

func TestTebakJenis_SetorPPhFinal(t *testing.T) {
	if j, _, _ := TebakJenis("SETORAN PAJAK MPN 411128-420 PPH FINAL", "keluar"); j != JenisSetorPPhFinal {
		t.Fatalf("kode billing PPh Final UMKM: %s", j)
	}
	if j, _, _ := TebakJenis("PAJAK BUNGA GIRO 411128-404", "keluar"); j == JenisSetorPPhFinal {
		t.Fatalf("PPh final bunga bukan PPh Final UMKM: %s", j)
	}
}
