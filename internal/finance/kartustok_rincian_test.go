package finance

import (
	"reflect"
	"testing"
)

// Kain (id 1) & Benang (id 2) untuk transaksi berbarang banyak.
func stokRincianUji() []StokItem {
	return []StokItem{
		{ID: 1, Nama: "Kain", Kategori: "Bahan Baku", Satuan: "m"},
		{ID: 2, Nama: "Benang", Kategori: "Bahan Penolong", Satuan: "m"},
	}
}

func gerakAwalRincianUji() []GerakStok {
	return []GerakStok{{ID: 1, Tanggal: "2026-01-01", Jenis: GerakAwal, Baris: []BarisGerak{
		{StokID: 1, Fisik: 1_000, HargaUnit: 10_000}, {StokID: 2, Fisik: 1_000, HargaUnit: 5_000}}}}
}

// Satu pembelian dua barang: tiap baris rincian menjadi baris kartu sendiri, TransaksiID sama,
// nilai = nilai barisnya (sisa Nominal, mis. ongkir, tidak masuk kartu).
func TestKartu_RincianDuaBarang(t *testing.T) {
	txs := []Transaction{{ID: 7, Tanggal: "2026-01-05", Jenis: JenisBeliStok, Nominal: 40_000,
		Rincian: []RincianBarang{{StokID: 1, QtyStok: 3_000, Nilai: 30_000}, {StokID: 2, QtyStok: 1_000, Nilai: 5_000}}}}
	k := HitungKartu(stokRincianUji(), txs, gerakAwalRincianUji())
	var beli []BarisKartu
	for _, b := range k.Baris {
		if b.Jenis == KartuBeli {
			beli = append(beli, b)
		}
	}
	if len(beli) != 2 {
		t.Fatalf("harus dua baris beli, dapat %+v", beli)
	}
	if beli[0].StokID != 1 || beli[0].TransaksiID != 7 || beli[0].Qty != 3_000 || beli[0].Nilai != 30_000 {
		t.Errorf("baris kain %+v", beli[0])
	}
	if beli[1].StokID != 2 || beli[1].TransaksiID != 7 || beli[1].Qty != 1_000 || beli[1].Nilai != 5_000 {
		t.Errorf("baris benang %+v", beli[1])
	}
	if p := k.Posisi[1]; p.Qty != 4_000 || p.Nilai != 40_000 {
		t.Errorf("posisi kain %+v", p)
	}
	if p := k.Posisi[2]; p.Qty != 2_000 || p.Nilai != 10_000 {
		t.Errorf("posisi benang %+v", p)
	}
}

// Penjualan dua barang: dua baris jual dengan TransaksiID sama, urut sesuai rincian.
func TestKartu_RincianJualDuaBarang(t *testing.T) {
	txs := []Transaction{{ID: 8, Tanggal: "2026-01-06", Jenis: JenisJual, Nominal: 50_000, Channel: "Toko",
		Rincian: []RincianBarang{{StokID: 2, QtyStok: 500, Nilai: 10_000}, {StokID: 1, QtyStok: 500, Nilai: 40_000}}}}
	k := HitungKartu(stokRincianUji(), txs, gerakAwalRincianUji())
	var jual []BarisKartu
	for _, b := range k.Baris {
		if b.Jenis == KartuJual {
			jual = append(jual, b)
		}
	}
	if len(jual) != 2 || jual[0].StokID != 2 || jual[1].StokID != 1 || jual[0].TransaksiID != 8 || jual[1].TransaksiID != 8 ||
		jual[0].Nilai != -2_500 || jual[1].Nilai != -5_000 || jual[0].Channel != "Toko" {
		t.Errorf("baris jual %+v", jual)
	}
}

// Data lama (StokID tunggal) menghasilkan kartu yang persis sama dengan Rincian setara.
func TestKartu_DataLamaSamaDenganRincian(t *testing.T) {
	lama := []Transaction{
		{ID: 1, Tanggal: "2026-01-03", Jenis: JenisBeliStok, Nominal: 25_000, StokID: 1, QtyStok: 2_000},
		{ID: 2, Tanggal: "2026-01-04", Jenis: JenisJual, Nominal: 60_000, StokID: 1, QtyStok: 1_500, Channel: "Toko"},
		{ID: 3, Tanggal: "2026-01-04", Jenis: JenisBeliStok, Nominal: 7_000, StokID: 2, QtyStok: 1_000},
	}
	baru := make([]Transaction, len(lama))
	for i, t := range lama {
		NormalkanBarang(&t)
		baru[i] = t
	}
	for _, m := range []string{MetodeRataRata, MetodeFIFO} {
		a := HitungKartuMetode(m, stokRincianUji(), lama, gerakAwalRincianUji())
		b := HitungKartuMetode(m, stokRincianUji(), baru, gerakAwalRincianUji())
		if len(a.Baris) == 0 || !reflect.DeepEqual(a.Baris, b.Baris) || !reflect.DeepEqual(a.Posisi, b.Posisi) {
			t.Errorf("%s: kartu data lama ≠ rincian\nlama %+v\nbaru %+v", m, a.Baris, b.Baris)
		}
	}
}
