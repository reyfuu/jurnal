package finance

import (
	"strings"
	"testing"
)

// Kopi (id 1, kg) & Kopi Susu (id 2, cup). Qty per seribu satuan.
func stokUji() []StokItem {
	return []StokItem{
		{ID: 1, SKU: "BHN-KOPI", Nama: "Biji kopi", Kategori: "Bahan Baku", Satuan: "kg", HargaPokok: 200_000},
		{ID: 2, SKU: "KS-1", Nama: "Kopi Susu", Kategori: "Produk Jadi", Satuan: "cup", HargaPokok: 0},
	}
}

func posisi(t *testing.T, k KartuStok, id int) PosisiStok {
	t.Helper()
	return k.Posisi[id]
}

func TestKartu_BelumMulaiKosong(t *testing.T) {
	k := HitungKartu(stokUji(), []Transaction{{ID: 1, Tanggal: "2026-01-02", Jenis: "Beli Bahan / Stok", Nominal: 400_000, StokID: 1, QtyStok: 2_000}}, nil)
	if k.Mulai != "" || len(k.Baris) != 0 {
		t.Fatalf("tanpa stok awal kartu harus kosong: %+v", k)
	}
}

func TestKartu_RataRataBergerakBeliJualOpname(t *testing.T) {
	gerak := []GerakStok{
		{ID: 1, Tanggal: "2026-01-01", Jenis: GerakAwal, Baris: []BarisGerak{{StokID: 1, Fisik: 10_000, HargaUnit: 200_000}}},
		{ID: 2, Tanggal: "2026-01-20", Jenis: GerakOpname, Baris: []BarisGerak{{StokID: 1, Fisik: 6_000}}},
	}
	txs := []Transaction{
		// sebelum titik nol: diabaikan
		{ID: 9, Tanggal: "2025-12-30", Jenis: "Beli Bahan / Stok", Nominal: 999, StokID: 1, QtyStok: 1_000},
		// beli 10 kg @250rb → 20 kg, nilai 4,5 jt, rata-rata 225rb
		{ID: 10, Tanggal: "2026-01-05", Jenis: "Beli Bahan / Stok", Nominal: 2_500_000, StokID: 1, QtyStok: 10_000},
		// jual 2,5 kg → HPP 562.500
		{ID: 11, Tanggal: "2026-01-10", Jenis: "Penjualan", Nominal: 1_000_000, StokID: 1, QtyStok: 2_500, Channel: "Toko"},
		// transaksi tanpa barang tidak menggerakkan stok
		{ID: 12, Tanggal: "2026-01-11", Jenis: "Penjualan", Nominal: 500_000},
	}
	k := HitungKartu(stokUji(), txs, gerak)
	if k.Mulai != "2026-01-01" {
		t.Fatalf("mulai %q", k.Mulai)
	}
	var jual BarisKartu
	for _, b := range k.Baris {
		if b.TransaksiID == 9 {
			t.Error("transaksi sebelum titik nol ikut dihitung")
		}
		if b.TransaksiID == 11 {
			jual = b
		}
	}
	if jual.Jenis != KartuJual || jual.Qty != -2_500 || jual.Nilai != -562_500 || jual.Channel != "Toko" {
		t.Errorf("jual %+v", jual)
	}
	// sisa sistem 17,5 kg nilai 3.937.500; opname fisik 6 kg → selisih −11,5 kg × 225rb
	p := posisi(t, k, 1)
	if p.Qty != 6_000 || p.Nilai != 1_350_000 {
		t.Errorf("posisi akhir %+v", p)
	}
	op := k.Baris[len(k.Baris)-1]
	if op.Jenis != KartuOpname || op.Qty != -11_500 || op.Nilai != -2_587_500 {
		t.Errorf("opname %+v", op)
	}
	if p := k.PosisiPada(1, "2026-01-09"); p.Qty != 20_000 || p.Nilai != 4_500_000 {
		t.Errorf("posisi 9 Jan %+v", p)
	}
}

func TestKartu_ProduksiMemakaiBahanMenambahBarangJadi(t *testing.T) {
	gerak := []GerakStok{
		{ID: 1, Tanggal: "2026-01-01", Jenis: GerakAwal, Baris: []BarisGerak{{StokID: 1, Fisik: 2_000, HargaUnit: 200_000}}},
		// 50 cup memakai 0,9 kg kopi (18 g/cup)
		{ID: 2, Tanggal: "2026-01-03", Jenis: GerakProduksi, Baris: []BarisGerak{{StokID: 1, Qty: -900}, {StokID: 2, Qty: 50_000}}},
	}
	txs := []Transaction{{ID: 1, Tanggal: "2026-01-04", Jenis: "Penjualan", Nominal: 360_000, StokID: 2, QtyStok: 20_000}}
	k := HitungKartu(stokUji(), txs, gerak)
	if p := posisi(t, k, 1); p.Qty != 1_100 || p.Nilai != 220_000 {
		t.Errorf("bahan %+v", p)
	}
	// 50 cup bernilai 180.000 (3.600/cup); jual 20 cup → HPP 72.000, sisa 30 cup 108.000
	if p := posisi(t, k, 2); p.Qty != 30_000 || p.Nilai != 108_000 {
		t.Errorf("barang jadi %+v", p)
	}
}

func TestKartu_StokMinusTetapDicatatDanDitandai(t *testing.T) {
	gerak := []GerakStok{{ID: 1, Tanggal: "2026-01-01", Jenis: GerakAwal, Baris: []BarisGerak{{StokID: 1, Fisik: 1_000, HargaUnit: 200_000}}}}
	txs := []Transaction{
		{ID: 1, Tanggal: "2026-01-02", Jenis: "Penjualan", Nominal: 1, StokID: 1, QtyStok: 3_000}, // minus 2 kg
		{ID: 2, Tanggal: "2026-01-03", Jenis: "Beli Bahan / Stok", Nominal: 1_500_000, StokID: 1, QtyStok: 5_000},
	}
	k := HitungKartu(stokUji(), txs, gerak)
	jual := k.Baris[1]
	// keluar 3 kg dinilai rata-rata terakhir 200rb/kg
	if jual.Nilai != -600_000 || jual.SaldoQty != -2_000 || !jual.Minus {
		t.Errorf("jual minus %+v", jual)
	}
	// setelah beli 5 kg @300rb, sisa 3 kg dinilai harga masuk: 900rb; selisih jadi koreksi HPP
	p := posisi(t, k, 1)
	if p.Qty != 3_000 || p.Nilai != 900_000 {
		t.Errorf("setelah menutup minus %+v", p)
	}
	beli := k.Baris[2]
	if beli.Nilai != 1_500_000 || beli.Koreksi != 900_000-(1_500_000-400_000) {
		t.Errorf("beli %+v", beli)
	}
}

func TestLaporanMutasiBulanan(t *testing.T) {
	gerak := []GerakStok{
		{ID: 1, Tanggal: "2026-01-01", Jenis: GerakAwal, Baris: []BarisGerak{{StokID: 1, Fisik: 10_000, HargaUnit: 200_000}}},
		{ID: 2, Tanggal: "2026-02-10", Jenis: GerakProduksi, Baris: []BarisGerak{{StokID: 1, Qty: -1_000}, {StokID: 2, Qty: 50_000}}},
	}
	txs := []Transaction{
		{ID: 1, Tanggal: "2026-01-15", Jenis: "Penjualan", Nominal: 1, StokID: 1, QtyStok: 2_000},
		{ID: 2, Tanggal: "2026-02-05", Jenis: "Beli Bahan / Stok", Nominal: 1_000_000, StokID: 1, QtyStok: 5_000},
		{ID: 3, Tanggal: "2026-02-20", Jenis: "Penjualan", Nominal: 1, StokID: 2, QtyStok: 10_000},
	}
	k := HitungKartu(stokUji(), txs, gerak)
	l := k.Mutasi(stokUji(), "2026-02-01", "2026-02-28")
	var kopi MutasiItem
	for _, it := range l.Items {
		if it.StokID == 1 {
			kopi = it
		}
	}
	if kopi.AwalQty != 8_000 || kopi.BeliQty != 5_000 || kopi.PakaiQty != 1_000 || kopi.AkhirQty != 12_000 {
		t.Errorf("kopi %+v", kopi)
	}
	if kopi.AwalNilai+kopi.BeliNilai-kopi.PakaiNilai-kopi.JualNilai+kopi.OpnameNilai+kopi.KoreksiNilai != kopi.AkhirNilai {
		t.Errorf("nilai tidak berimbang %+v", kopi)
	}
	if l.Total.JualNilai == 0 || l.HPPJual != l.Total.JualNilai {
		t.Errorf("total/hpp %+v %d", l.Total, l.HPPJual)
	}
}

func TestValidasiTransaksiBarang(t *testing.T) {
	ok := Transaction{Tanggal: "2026-01-01", Jenis: "Penjualan", Nominal: 1, StokID: 1, QtyStok: 1_000}
	if err := Validate(&ok); err != nil {
		t.Fatal(err)
	}
	for _, salah := range []Transaction{
		{Tanggal: "2026-01-01", Jenis: "Gaji & Upah", Nominal: 1, StokID: 1, QtyStok: 1_000},
		{Tanggal: "2026-01-01", Jenis: "Penjualan", Nominal: 1, StokID: 1, QtyStok: 0},
	} {
		if err := Validate(&salah); err == nil || !strings.Contains(err.Error(), "barang") {
			t.Errorf("harus ditolak: %+v (%v)", salah, err)
		}
	}
	tanpa := Transaction{Tanggal: "2026-01-01", Jenis: "Penjualan", Nominal: 1, QtyStok: 5}
	if err := Validate(&tanpa); err != nil || tanpa.QtyStok != 0 {
		t.Errorf("tanpa barang: qty dibuang (%v, %d)", err, tanpa.QtyStok)
	}
}

func TestRingkasStokIkutKartu(t *testing.T) {
	gerak := []GerakStok{{ID: 1, Tanggal: "2026-01-01", Jenis: GerakAwal, Baris: []BarisGerak{{StokID: 1, Fisik: 2_500, HargaUnit: 200_000}}}}
	txs := []Transaction{{ID: 1, Tanggal: "2026-01-02", Jenis: "Penjualan", Nominal: 1, StokID: 2, QtyStok: 1_000}}
	s := RingkasStok(stokUji(), HitungKartu(stokUji(), txs, gerak))
	kopi, cup := s.Items[0], s.Items[1]
	if s.Mulai != "2026-01-01" || !kopi.Dilacak || kopi.JumlahMili != 2_500 || kopi.Nilai != 500_000 || kopi.HargaPokok != 200_000 || kopi.Jumlah != 2 {
		t.Errorf("kopi %+v", kopi)
	}
	if !cup.Minus || cup.Status != "Minus" || s.Minus != 1 || s.NilaiPersediaan != kopi.Nilai+cup.Nilai {
		t.Errorf("cup %+v ringkas minus %d nilai %d", cup, s.Minus, s.NilaiPersediaan)
	}
	if lama := RingkasStok(stokUji(), KartuStok{}); lama.Mulai != "" || lama.Items[0].Dilacak {
		t.Error("tanpa titik nol harus sama seperti sebelumnya")
	}
}
