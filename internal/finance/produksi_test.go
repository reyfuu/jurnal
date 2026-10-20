package finance_test

import (
	"strings"
	"testing"

	"pajak/internal/finance"
)

func produkSeed(t *testing.T, sku string) finance.Produk {
	t.Helper()
	for _, p := range finance.SeedProduk() {
		if p.SKU == sku {
			return p
		}
	}
	t.Fatalf("produk contoh %s tidak ada", sku)
	return finance.Produk{}
}

func TestHitungHPP_DataContoh(t *testing.T) {
	h := finance.HitungHPP(produkSeed(t, "FG-001"), finance.SeedStok())
	for _, c := range []struct {
		nama string
		got  int64
		want int64
	}{
		{"total bahan", h.TotalBahan, 75_400}, // 33.300 + 25.800 + 4.800 + 11.500
		{"total per batch", h.TotalBatch, 83_000},
		{"HPP per unit", h.PerUnit, 8_300},
		{"laba per unit", h.LabaPerUnit, 9_700},
	} {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.nama, c.got, c.want)
		}
	}
	if h.MarginBps != 5_388 {
		t.Errorf("margin = %d bps, want 5388", h.MarginBps)
	}
	if len(h.Peringatan) != 0 {
		t.Errorf("resep contoh harus lengkap, ada peringatan: %v", h.Peringatan)
	}
	// Takaran per seribu: 0,18 kg biji @185.000/kg = 33.300, bukan 185.000 x 180.
	if got := h.Bahan[0].Subtotal; got != 33_300 {
		t.Errorf("biji kopi 0,18 kg = %d, want 33300", got)
	}
}

// Inti fiturnya: HPP ikut naik ketika harga bahan di Stok & Gudang naik, tanpa resep disentuh.
func TestHitungHPP_IkutHargaBahan(t *testing.T) {
	p := produkSeed(t, "FG-001")
	stok := finance.SeedStok()
	sebelum := finance.HitungHPP(p, stok).PerUnit
	for i := range stok {
		if stok[i].SKU == "RM-001" {
			stok[i].HargaPokok = 220_000 // biji kopi naik dari 185.000
		}
	}
	sesudah := finance.HitungHPP(p, stok)
	// 0,18 kg x selisih 35.000 = 6.300 per batch 10 cup = 630 per cup.
	if sesudah.PerUnit != sebelum+630 {
		t.Errorf("HPP setelah bahan naik = %d, want %d", sesudah.PerUnit, sebelum+630)
	}
	if sesudah.PerUnit <= sebelum {
		t.Error("HPP harus ikut naik saat harga bahan naik")
	}
}

func TestHitungHPP_PembulatanKeAtas(t *testing.T) {
	// 10.000 per batch dibagi 3 unit = 3.333,33. Dibulatkan ke atas supaya HPP tidak
	// dilaporkan lebih murah dari kenyataan, yang akan membuat harga jual kurang.
	p := finance.Produk{Nama: "Uji", HasilPerBatch: 3, BiayaOverhead: 10_000, HargaJual: 5_000}
	h := finance.HitungHPP(p, nil)
	if h.PerUnit != 3_334 {
		t.Errorf("HPP per unit = %d, want 3334 (dibulatkan ke atas dari 3333,33)", h.PerUnit)
	}
}

func TestHitungHPP_BahanHilangTidakMenggagalkan(t *testing.T) {
	p := finance.Produk{Nama: "Uji", HasilPerBatch: 1, Bahan: []finance.ResepBahan{
		{StokID: 9999, Takaran: finance.TakaranSkala}, // barangnya sudah dihapus dari stok
		{Nama: "Garam", Takaran: finance.TakaranSkala, HargaManual: 500},
	}}
	h := finance.HitungHPP(p, finance.SeedStok())
	if h.TotalBahan != 500 {
		t.Errorf("bahan hilang dihitung 0, total = %d, want 500", h.TotalBahan)
	}
	if len(h.Peringatan) != 1 || !strings.Contains(h.Peringatan[0], "tidak ada lagi di Stok") {
		t.Errorf("peringatan bahan hilang tidak muncul: %v", h.Peringatan)
	}
	if h.Bahan[1].Sumber != "manual" {
		t.Errorf("bahan tanpa StokID bersumber manual, got %q", h.Bahan[1].Sumber)
	}
}

func TestHitungHPP_HargaJualDiBawahHPP(t *testing.T) {
	p := finance.Produk{Nama: "Uji", HasilPerBatch: 1, BiayaOverhead: 10_000, HargaJual: 8_000}
	h := finance.HitungHPP(p, nil)
	if h.LabaPerUnit != -2_000 {
		t.Errorf("laba = %d, want -2000", h.LabaPerUnit)
	}
	var rugi bool
	for _, w := range h.Peringatan {
		if strings.Contains(w, "di bawah HPP") {
			rugi = true
		}
	}
	if !rugi {
		t.Errorf("harus memperingatkan jual di bawah HPP: %v", h.Peringatan)
	}
}

func TestSummarizeProduksi_DataContoh(t *testing.T) {
	s := finance.SummarizeProduksi(finance.SeedProduk(), finance.SeedSupplier(), finance.SeedStok(), finance.SeedTagihan())
	if s.TotalProduk != 2 || s.TotalSupplier != 3 {
		t.Fatalf("contoh: %d produk, %d supplier", s.TotalProduk, s.TotalSupplier)
	}
	if s.ProdukRugi != 0 || s.ProdukTanpaHPP != 0 {
		t.Errorf("data contoh harus sehat: rugi %d, resep tak lengkap %d", s.ProdukRugi, s.ProdukTanpaHPP)
	}
	// Harga pokok produk jadi di Stok & Gudang harus sama dengan hasil resep.
	for _, p := range s.Produk {
		if !p.AdaDiStok {
			t.Errorf("%s harus tercocokkan ke stok lewat SKU", p.SKU)
			continue
		}
		if p.Selisih != 0 {
			t.Errorf("%s: HPP resep %d vs stok %d, selisih %d", p.SKU, p.HPP.PerUnit, p.HPPStok, p.Selisih)
		}
	}
	// Utang supplier dijumlahkan dari tagihan PAYABLE yang belum lunas, bukan dari piutang.
	if s.UtangSupplier != 7_400_000 {
		t.Errorf("utang supplier = %d, want 7.400.000", s.UtangSupplier)
	}
	for _, sp := range s.Supplier {
		if sp.Nama == "CV Susu Segar Nusantara" && sp.UtangBelumDibayar != 0 {
			t.Errorf("supplier tanpa tagihan harus 0, got %d", sp.UtangBelumDibayar)
		}
	}
}

func TestSummarizeProduksi_UtangLunasTidakDihitung(t *testing.T) {
	tagihan := finance.SeedTagihan()
	for i := range tagihan {
		if tagihan[i].Nomor == "PO-SUP-112" {
			tagihan[i].Lunas = true
		}
	}
	s := finance.SummarizeProduksi(nil, finance.SeedSupplier(), nil, tagihan)
	if s.UtangSupplier != 2_600_000 {
		t.Errorf("setelah PO-SUP-112 lunas, utang = %d, want 2.600.000", s.UtangSupplier)
	}
}

func TestSummarizeProduksi_HPPStokBasiTerdeteksi(t *testing.T) {
	stok := finance.SeedStok()
	for i := range stok {
		if stok[i].SKU == "FG-001" {
			stok[i].HargaPokok = 6_800 // angka lama, sebelum harga bahan naik
		}
	}
	s := finance.SummarizeProduksi(finance.SeedProduk(), nil, stok, nil)
	var row finance.ProdukRow
	for _, p := range s.Produk {
		if p.SKU == "FG-001" {
			row = p
		}
	}
	if row.Selisih != 1_500 {
		t.Errorf("selisih HPP resep (8.300) vs stok (6.800) = %d, want 1500", row.Selisih)
	}
}

func TestValidateProduk(t *testing.T) {
	p := finance.Produk{Nama: " Kopi Susu ", HasilPerBatch: 1}
	if err := finance.ValidateProduk(&p); err != nil {
		t.Fatalf("produk minimal harus valid: %v", err)
	}
	if p.Nama != "Kopi Susu" || p.Satuan != "pcs" || p.Kategori != "Produk Jadi" {
		t.Errorf("normalisasi: %+v", p)
	}
	for _, c := range []struct {
		nama string
		p    finance.Produk
	}{
		{"tanpa nama", finance.Produk{HasilPerBatch: 1}},
		{"hasil per batch 0", finance.Produk{Nama: "X"}},
		{"takaran 0", finance.Produk{Nama: "X", HasilPerBatch: 1, Bahan: []finance.ResepBahan{{Nama: "Gula", Takaran: 0}}}},
	} {
		p := c.p
		if err := finance.ValidateProduk(&p); err == nil {
			t.Errorf("%s harus ditolak", c.nama)
		}
	}
	// Baris bahan kosong dari form dibuang, bukan dianggap kesalahan.
	p = finance.Produk{Nama: "X", HasilPerBatch: 1, Bahan: []finance.ResepBahan{{}, {Nama: "Gula", Takaran: 500, HargaManual: 100}}}
	if err := finance.ValidateProduk(&p); err != nil {
		t.Fatalf("baris kosong harus diabaikan: %v", err)
	}
	if len(p.Bahan) != 1 {
		t.Errorf("baris bahan kosong harus dibuang, sisa %d", len(p.Bahan))
	}
}
