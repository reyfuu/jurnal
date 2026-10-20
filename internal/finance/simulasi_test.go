package finance_test

import (
	"testing"

	"pajak/internal/finance"
)

// dasarGambar: angka dari contoh Simulasi & Harga (rata-rata Jul–Sep 2026).
func dasarGambar() finance.DasarSimulasi {
	return finance.DasarSimulasi{Omzet: 87_307_333, ModalBahan: 29_979_479, BiayaVariabel: 6_472_400, BiayaTetap: 26_488_292, Penyusutan: 1_427_083, HariBuka: 31}
}

func TestSimulasikan_TanpaPerubahanSamaDenganDasar(t *testing.T) {
	h := finance.Simulasikan(dasarGambar(), finance.Skenario{})
	if h.Laba != 22_940_079 || h.Omzet != 87_307_333 || h.MarginBps != 2627 {
		t.Fatalf("hasil = %+v", h)
	}
	bep := finance.HitungTitikImpas(h, 31)
	if !bep.Ada || bep.PerBulan != 47_924_397 || bep.SisaPer100rb != 58_249 || bep.DiAtasBps != 4510 || bep.BiayaTetap != 27_915_375 {
		t.Fatalf("titik impas = %+v", bep)
	}
	if bep.PerHari != 1_545_949 { // 47.924.397 / 31 hari buka, dibulatkan ke atas
		t.Errorf("per hari = %d", bep.PerHari)
	}
}

func TestSimulasikan_Skenario(t *testing.T) {
	d := dasarGambar()
	// Naik harga 5%: omzet naik 5%, modal bahan tetap, biaya variabel ikut omzet (komisi %).
	h := finance.Simulasikan(d, finance.Skenario{HargaBps: 500})
	if h.Omzet != 91_672_700 || h.ModalBahan != d.ModalBahan || h.BiayaVariabel != 6_796_020 {
		t.Fatalf("naik harga 5%% = %+v", h)
	}
	// Jualan naik 20%: omzet, modal bahan, dan biaya variabel naik 20%; biaya tetap tidak.
	h = finance.Simulasikan(d, finance.Skenario{VolumeBps: 2000})
	if h.Omzet != 104_768_800 || h.ModalBahan != 35_975_375 || h.BiayaTetap != d.BiayaTetap {
		t.Fatalf("jualan naik 20%% = %+v", h)
	}
	// Hemat bahan 10% + tambah karyawan Rp3 jt.
	h = finance.Simulasikan(d, finance.Skenario{ModalBps: -1000, TambahTetap: 3_000_000})
	if h.ModalBahan != 26_981_531 || h.BiayaTetap != 29_488_292 || h.Laba != 22_940_079+2_997_948-3_000_000 {
		t.Fatalf("hemat bahan + karyawan = %+v", h)
	}
	// Harga turun 90%: tiap penjualan rugi → tidak ada titik impas.
	h = finance.Simulasikan(d, finance.Skenario{HargaBps: -9000})
	if bep := finance.HitungTitikImpas(h, 31); bep.Ada {
		t.Errorf("harga di bawah modal tetap punya titik impas: %+v", bep)
	}
}

func TestSimulasikan_AngkaBesarTidakOverflow(t *testing.T) {
	d := finance.DasarSimulasi{Omzet: 900_000_000_000, ModalBahan: 400_000_000_000, BiayaVariabel: 90_000_000_000, BiayaTetap: 100_000_000_000}
	h := finance.Simulasikan(d, finance.Skenario{HargaBps: 5000, VolumeBps: 5000})
	if h.Omzet != 2_025_000_000_000 || h.BiayaVariabel != 202_500_000_000 {
		t.Fatalf("hasil = %+v", h)
	}
}

func TestRataRataSimulasi(t *testing.T) {
	bulan := []finance.BulanSimulasi{
		{Periode: "2026-07", Income: finance.IncomeStatement{Omzet: 90_000_000, COGS: 30_000_000, ChannelFees: 6_000_000, Opex: 26_000_000, Depreciation: 1_400_000, NetProfit: 26_600_000}, HariBuka: 31},
		{Periode: "2026-08", Income: finance.IncomeStatement{Omzet: 84_000_001, COGS: 29_000_000, ChannelFees: 7_000_000, Opex: 27_000_000, Depreciation: 1_400_000, Amortization: 100_000, NetProfit: 19_500_001}, HariBuka: 30},
	}
	d := finance.RataRataSimulasi(bulan)
	if d.Omzet != 87_000_001 || d.ModalBahan != 29_500_000 || d.BiayaVariabel != 6_500_000 || d.BiayaTetap != 26_500_000 || d.Penyusutan != 1_450_000 || d.HariBuka != 31 || len(d.Periode) != 2 {
		t.Fatalf("rata-rata = %+v", d)
	}
	if finance.RataRataSimulasi(nil).Omzet != 0 {
		t.Error("tanpa data harus nol")
	}
}

func TestPorsiBalikModal(t *testing.T) {
	d := dasarGambar()
	// Es kopi Rp22.000, modal Rp8.300; biaya variabel 7,41% dari omzet = Rp1.631 per porsi.
	p := finance.PorsiBalikModal(d, 22_000, 8_300)
	if !p.Ada || p.Sisa != 22_000-8_300-1_631 || p.PorsiPerHari != 75 {
		t.Fatalf("porsi = %+v", p)
	}
	if p := finance.PorsiBalikModal(d, 9_000, 8_500); p.Ada {
		t.Errorf("harga di bawah modal + variabel tetap dihitung: %+v", p)
	}
}

func TestHargaUntukMargin(t *testing.T) {
	// Contoh gambar: modal 8.000, toko (tanpa komisi), margin 60% → Rp20.000.
	r, err := finance.HargaUntukMargin(8_000, 0, 0, 6000)
	if err != nil || r.RecommendedPrice != 20_000 || r.NetTakeHome-r.COGS != 12_000 {
		t.Fatalf("toko = %+v %v", r, err)
	}
	// GoFood 20%: harga harus menutup komisi juga.
	r, _ = finance.HargaUntukMargin(8_000, 2000, 0, 6000)
	if r.RecommendedPrice != 40_000 || r.ChannelFee != 8_000 {
		t.Fatalf("gofood = %+v", r)
	}
	if _, err := finance.HargaUntukMargin(8_000, 5000, 0, 6000); err == nil {
		t.Error("komisi + margin ≥ 100% harus ditolak")
	}
}
