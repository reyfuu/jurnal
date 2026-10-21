package finance_test

import (
	"testing"

	"pajak/internal/finance"
)

// Contoh pengguna: laptop 15 jt, Kelompok 1 (48 bulan), diperoleh 2025-01-01, saldo awal
// 2026-01-01 → 12 bulan × 312.500 = 3.750.000; nilai buku 11.250.000. Sama dengan rumus
// Akumulasi Dihitung di template Excel, dengan atau tanpa akumulasi manual yang sama.
func TestAkumulasiAwal_LaptopContohPengguna(t *testing.T) {
	laptop := finance.Asset{Nama: "Laptop", Kelompok: "k1", Metode: finance.MetodeGarisLurus, TanggalPerolehan: "2025-01-01", HargaPerolehan: 15_000_000}
	manual := laptop
	manual.AkumulasiAwal, manual.PerTanggal = 3_750_000, "2026-01-01"
	for _, a := range []finance.Asset{laptop, manual} {
		if err := finance.ValidateAsset(&a); err != nil {
			t.Fatal(err)
		}
		r := finance.AssetsForPeriod([]finance.Asset{a}, nil, "2026-01")
		if r.NilaiBukuAwal != 11_250_000 {
			t.Errorf("akumulasi awal %d: nilai buku awal %d, want 11.250.000", a.AkumulasiAwal, r.NilaiBukuAwal)
		}
		if got := finance.AssetSchedule(a); len(got) != 48 || sum(got) != 15_000_000 || got[12] != 312_500 {
			t.Errorf("jadwal: %d bulan, total %d, bulan ke-13 %d", len(got), sum(got), got[12])
		}
	}
}

// Akumulasi manual berbeda dari hitungan sistem: nilai buku awal mengikuti angka manual, sisa
// harga disusutkan di sisa masa manfaat (garis lurus & saldo menurun), total tetap = harga.
func TestAkumulasiAwal_ManualSisaDisusutkanSisaMasa(t *testing.T) {
	a := finance.Asset{Nama: "Laptop", Kelompok: "k1", Metode: finance.MetodeGarisLurus, TanggalPerolehan: "2025-01-01", HargaPerolehan: 15_000_000,
		AkumulasiAwal: 5_000_000, PerTanggal: "2026-01-01"}
	if err := finance.ValidateAsset(&a); err != nil {
		t.Fatal(err)
	}
	if r := finance.AssetsForPeriod([]finance.Asset{a}, nil, "2026-01"); r.NilaiBukuAwal != 10_000_000 || r.BebanPenyusutan != 10_000_000/36 {
		t.Fatalf("GL: nilai buku awal %d (want 10.000.000), beban Jan %d (want %d)", r.NilaiBukuAwal, r.BebanPenyusutan, int64(10_000_000/36))
	}
	gl := finance.AssetSchedule(a)
	if len(gl) != 48 || sum(gl[:12]) != 5_000_000 || sum(gl) != 15_000_000 {
		t.Fatalf("GL: %d bulan, 12 bulan pertama %d, total %d", len(gl), sum(gl[:12]), sum(gl))
	}
	if r := finance.AssetsForPeriod([]finance.Asset{a}, nil, "2028-12"); r.TotalNilaiBuku != 0 {
		t.Errorf("GL akhir masa manfaat: nilai buku %d, want 0", r.TotalNilaiBuku)
	}

	a.Metode = finance.MetodeSaldoMenurun
	sm := finance.AssetSchedule(a)
	if len(sm) != 48 || sum(sm[:12]) != 5_000_000 || sum(sm) != 15_000_000 {
		t.Fatalf("SM: %d bulan, 12 bulan pertama %d, total %d", len(sm), sum(sm[:12]), sum(sm))
	}
	// Tahun 2026: 50% × sisa 10 jt = 5 jt, dibagi 12 bulan.
	if got := sum(sm[12:24]); got != 5_000_000 {
		t.Errorf("SM 2026: want 5.000.000, got %d", got)
	}
	if r := finance.AssetsForPeriod([]finance.Asset{a}, nil, "2026-01"); r.NilaiBukuAwal != 10_000_000 {
		t.Errorf("SM nilai buku awal %d", r.NilaiBukuAwal)
	}

	// Saldo awal di tengah tahun: blok saldo menurun berikutnya hanya sampai Desember.
	b := finance.Asset{Nama: "Mesin", Kelompok: "k1", Metode: finance.MetodeSaldoMenurun, TanggalPerolehan: "2025-03-01", HargaPerolehan: 12_000_000,
		AkumulasiAwal: 4_000_000, PerTanggal: "2025-09-15"}
	sb := finance.AssetSchedule(b)
	if len(sb) != 48 || sum(sb[:6]) != 4_000_000 || sum(sb) != 12_000_000 {
		t.Fatalf("SM tengah tahun: %d bulan, 6 bulan pertama %d, total %d", len(sb), sum(sb[:6]), sum(sb))
	}
	if got, want := sum(sb[6:10]), int64(8_000_000*5000*4/(10_000*12)); got != want {
		t.Errorf("SM Sep–Des 2025 atas sisa 8 jt: got %d, want %d", got, want)
	}
}

// Masa manfaat sudah habis per tanggal saldo awal: aset habis disusutkan, angka manual diabaikan.
func TestAkumulasiAwal_LewatMasaManfaat(t *testing.T) {
	a := finance.Asset{Nama: "Laptop Lama", Kelompok: "k1", TanggalPerolehan: "2020-01-01", HargaPerolehan: 4_800_000,
		AkumulasiAwal: 4_800_000, PerTanggal: "2026-01-01"}
	if r := finance.AssetsForPeriod([]finance.Asset{a}, nil, "2026-01"); r.NilaiBukuAwal != 0 || sum(finance.AssetSchedule(a)) != 4_800_000 {
		t.Errorf("nilai buku awal %d", r.NilaiBukuAwal)
	}
}

func TestAkumulasiAwal_Validasi(t *testing.T) {
	base := finance.Asset{Nama: "X", Kelompok: "k1", TanggalPerolehan: "2025-01-01", HargaPerolehan: 1_000_000}
	for _, c := range []struct {
		akum int64
		per  string
	}{{-1, "2026-01-01"}, {1_000_001, "2026-01-01"}, {500, ""}, {500, "01-01-2026"}} {
		a := base
		a.AkumulasiAwal, a.PerTanggal = c.akum, c.per
		if err := finance.ValidateAsset(&a); err == nil {
			t.Errorf("akumulasi %d per %q harus ditolak", c.akum, c.per)
		}
	}
	ok := base
	ok.AkumulasiAwal, ok.PerTanggal = 0, "2026-01-01"
	if err := finance.ValidateAsset(&ok); err != nil {
		t.Errorf("akumulasi 0 (belum disusutkan) harus diterima: %v", err)
	}
}
