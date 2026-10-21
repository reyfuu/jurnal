package finance

import "testing"

// Contoh yang dijelaskan ke pengguna: 10 kg @10rb, beli 10 kg @12rb, jual 15 kg.
// FIFO: HPP 10×10rb + 5×12rb = 160rb, sisa 5 kg = 60rb. Rata-rata: 15×11rb = 165rb, sisa 55rb.
func TestKartu_FIFOvsRataRata(t *testing.T) {
	gerak := []GerakStok{{ID: 1, Tanggal: "2026-01-01", Jenis: GerakAwal, Baris: []BarisGerak{{StokID: 1, Fisik: 10_000, HargaUnit: 10_000}}}}
	txs := []Transaction{
		{ID: 1, Tanggal: "2026-01-02", Jenis: JenisBeliStok, Nominal: 120_000, StokID: 1, QtyStok: 10_000},
		{ID: 2, Tanggal: "2026-01-03", Jenis: JenisJual, Nominal: 300_000, StokID: 1, QtyStok: 15_000},
	}
	for _, c := range []struct {
		metode     string
		hpp, nilai int64
	}{{MetodeFIFO, 160_000, 60_000}, {MetodeRataRata, 165_000, 55_000}, {"", 165_000, 55_000}} {
		k := HitungKartuMetode(c.metode, stokUji(), txs, gerak)
		l := k.Mutasi(stokUji(), "2026-01-01", "2026-01-31")
		if l.HPPJual != c.hpp || k.Posisi[1] != (PosisiStok{Qty: 5_000, Nilai: c.nilai}) {
			t.Errorf("%q: HPP %d (mau %d), posisi %+v (mau nilai %d)", c.metode, l.HPPJual, c.hpp, k.Posisi[1], c.nilai)
		}
	}
}

// FIFO: penjualan melebihi stok dinilai harga pokok terakhir; pembelian berikutnya menutup
// minusnya dan selisih harganya jadi koreksi, sehingga saldo akhir = lapis yang tersisa.
func TestKartu_FIFOStokMinusDikoreksiPembelian(t *testing.T) {
	gerak := []GerakStok{{ID: 1, Tanggal: "2026-01-01", Jenis: GerakAwal, Baris: []BarisGerak{{StokID: 1, Fisik: 2_000, HargaUnit: 10_000}}}}
	txs := []Transaction{
		{ID: 1, Tanggal: "2026-01-02", Jenis: JenisJual, Nominal: 50_000, StokID: 1, QtyStok: 3_000},     // 2 @10rb + 1 @10rb (perkiraan)
		{ID: 2, Tanggal: "2026-01-03", Jenis: JenisBeliStok, Nominal: 48_000, StokID: 1, QtyStok: 4_000}, // @12rb: 1 menutup minus
	}
	k := HitungKartuMetode(MetodeFIFO, stokUji(), txs, gerak)
	jual, beli := k.Baris[1], k.Baris[2]
	if jual.Nilai != -30_000 || jual.SaldoQty != -1_000 || !jual.Minus {
		t.Errorf("jual %+v", jual)
	}
	if beli.Koreksi != -2_000 || beli.SaldoQty != 3_000 || beli.SaldoNilai != 36_000 {
		t.Errorf("beli menutup minus: %+v", beli)
	}
	if k.Posisi[1] != (PosisiStok{Qty: 3_000, Nilai: 36_000}) {
		t.Errorf("posisi %+v", k.Posisi[1])
	}
}

// FIFO di produksi & opname: bahan keluar dari lapis tertua; barang lebih saat opname dinilai
// harga beli terakhir.
func TestKartu_FIFOProduksiDanOpname(t *testing.T) {
	gerak := []GerakStok{
		{ID: 1, Tanggal: "2026-01-01", Jenis: GerakAwal, Baris: []BarisGerak{{StokID: 1, Fisik: 1_000, HargaUnit: 200_000}}},
		{ID: 2, Tanggal: "2026-01-03", Jenis: GerakProduksi, Baris: []BarisGerak{{StokID: 1, Qty: -1_500}, {StokID: 2, Qty: 10_000}}},
		{ID: 3, Tanggal: "2026-01-05", Jenis: GerakOpname, Baris: []BarisGerak{{StokID: 1, Fisik: 2_000}}},
	}
	txs := []Transaction{{ID: 1, Tanggal: "2026-01-02", Jenis: JenisBeliStok, Nominal: 250_000, StokID: 1, QtyStok: 1_000}}
	k := HitungKartuMetode(MetodeFIFO, stokUji(), txs, gerak)
	// bahan: 1 kg @200rb + 0,5 kg @250rb = 325rb → nilai 10 cup
	if p := k.Posisi[2]; p.Nilai != 325_000 {
		t.Errorf("barang jadi %+v", p)
	}
	// sisa 0,5 kg @250rb = 125rb; opname fisik 2 kg → lebih 1,5 kg @250rb = 375rb
	if p := k.Posisi[1]; p.Qty != 2_000 || p.Nilai != 500_000 {
		t.Errorf("bahan setelah opname %+v", p)
	}
}
