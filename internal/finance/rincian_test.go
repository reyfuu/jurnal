package finance

import (
	"slices"
	"testing"
)

func TestBarisBarang_DataLamaJadiSatuBaris(t *testing.T) {
	tx := Transaction{Jenis: JenisJual, Nominal: 50_000, StokID: 3, QtyStok: 2000}
	want := []RincianBarang{{StokID: 3, QtyStok: 2000, Nilai: 50_000}}
	if got := tx.BarisBarang(); !slices.Equal(got, want) {
		t.Errorf("BarisBarang = %+v, want %+v", got, want)
	}
	if got := (Transaction{Nominal: 10}).BarisBarang(); got != nil {
		t.Errorf("tanpa barang harus nil, dapat %+v", got)
	}
}

func TestNormalkanBarang(t *testing.T) {
	tx := Transaction{Jenis: JenisBeliStok, Nominal: 80_000, StokID: 5, QtyStok: 1500}
	sebelum := tx.BarisBarang()
	NormalkanBarang(&tx)
	if tx.StokID != 0 || tx.QtyStok != 0 {
		t.Errorf("StokID/QtyStok harus kosong: %+v", tx)
	}
	if !slices.Equal(tx.BarisBarang(), sebelum) || !slices.Equal(tx.Rincian, sebelum) {
		t.Errorf("sesudah %+v, sebelum %+v", tx.Rincian, sebelum)
	}
	// sudah berrincian: tidak diubah
	dua := Transaction{Jenis: JenisJual, Nominal: 100, Rincian: []RincianBarang{{1, 1000, 60}, {2, 1000, 40}}}
	NormalkanBarang(&dua)
	if len(dua.Rincian) != 2 {
		t.Errorf("rincian berubah: %+v", dua.Rincian)
	}
}

func TestValidate_Rincian(t *testing.T) {
	dasar := func() Transaction {
		return Transaction{Tanggal: "2026-01-05", Jenis: JenisJual, Nominal: 100_000, Lunas: true}
	}
	cases := []struct {
		nama string
		ubah func(*Transaction)
		err  bool
	}{
		{"rincian + stok_id", func(t *Transaction) {
			t.StokID, t.QtyStok = 1, 1000
			t.Rincian = []RincianBarang{{StokID: 2, QtyStok: 1000}}
		}, true},
		{"qty 0", func(t *Transaction) { t.Rincian = []RincianBarang{{StokID: 1, QtyStok: 0}} }, true},
		{"stok_id 0", func(t *Transaction) { t.Rincian = []RincianBarang{{StokID: 0, QtyStok: 1000}} }, true},
		{"nilai negatif", func(t *Transaction) { t.Rincian = []RincianBarang{{StokID: 1, QtyStok: 1000, Nilai: -1}} }, true},
		{"jumlah nilai melebihi nominal", func(t *Transaction) {
			t.Rincian = []RincianBarang{{StokID: 1, QtyStok: 1000, Nilai: 70_000}, {StokID: 2, QtyStok: 1000, Nilai: 50_000}}
		}, true},
		{"jenis bukan jual/beli stok", func(t *Transaction) {
			t.Jenis = "Gaji & Upah"
			t.Rincian = []RincianBarang{{StokID: 1, QtyStok: 1000}}
		}, true},
		{"bukan_barang + rincian", func(t *Transaction) {
			t.BukanBarang = true
			t.Rincian = []RincianBarang{{StokID: 1, QtyStok: 1000}}
		}, true},
		{"bukan_barang + stok_id", func(t *Transaction) {
			t.BukanBarang = true
			t.StokID, t.QtyStok = 1, 1000
		}, true},
		{"bukan_barang jenis lain", func(t *Transaction) { t.Jenis = "Gaji & Upah"; t.BukanBarang = true }, true},
		{"bukan_barang sah", func(t *Transaction) { t.BukanBarang = true }, false},
		{"dua baris sah", func(t *Transaction) {
			t.Rincian = []RincianBarang{{StokID: 1, QtyStok: 1000, Nilai: 60_000}, {StokID: 2, QtyStok: 2000, Nilai: 40_000}}
		}, false},
		{"jual dua baris nilai 0 sah", func(t *Transaction) {
			t.Rincian = []RincianBarang{{StokID: 1, QtyStok: 1000}, {StokID: 2, QtyStok: 2000}}
		}, false},
		{"beli dua baris nilai 0", func(t *Transaction) {
			t.Jenis = JenisBeliStok
			t.Rincian = []RincianBarang{{StokID: 1, QtyStok: 1000, Nilai: 60_000}, {StokID: 2, QtyStok: 2000}}
		}, true},
		{"beli satu baris nilai 0 sah", func(t *Transaction) {
			t.Jenis = JenisBeliStok
			t.Rincian = []RincianBarang{{StokID: 1, QtyStok: 1000}}
		}, false},
		{"beli dua baris bernilai sah", func(t *Transaction) {
			t.Jenis = JenisBeliStok
			t.Rincian = []RincianBarang{{StokID: 1, QtyStok: 1000, Nilai: 60_000}, {StokID: 2, QtyStok: 2000, Nilai: 40_000}}
		}, false},
	}
	for _, c := range cases {
		tx := dasar()
		c.ubah(&tx)
		if err := Validate(&tx); (err != nil) != c.err {
			t.Errorf("%s: err = %v, want error=%v", c.nama, err, c.err)
		}
	}
}

func TestValidate_BeliBeberapaBarangWajibNilai(t *testing.T) {
	tx := Transaction{Tanggal: "2026-01-05", Jenis: JenisBeliStok, Nominal: 100_000, Lunas: true,
		Rincian: []RincianBarang{{StokID: 1, QtyStok: 1000}, {StokID: 2, QtyStok: 2000, Nilai: 40_000}}}
	if err := Validate(&tx); err == nil || err.Error() != "isi nilai tiap barang pada pembelian beberapa barang" {
		t.Errorf("err = %v", err)
	}
}

func TestSama_RincianNilSamaDenganKosong(t *testing.T) {
	a := Transaction{ID: 1, Nominal: 5}
	b := a
	b.Rincian = []RincianBarang{}
	if !sama(a, b) {
		t.Error("nil dan kosong harus dianggap sama")
	}
	b.Rincian = []RincianBarang{{StokID: 1, QtyStok: 1}}
	if sama(a, b) {
		t.Error("rincian berbeda harus tidak sama")
	}
	v := Transaction{Tanggal: "2026-01-05", Jenis: JenisJual, Nominal: 5, Rincian: []RincianBarang{}}
	if err := Validate(&v); err != nil || v.Rincian != nil {
		t.Errorf("Validate harus menormalkan rincian kosong ke nil: %v %#v", err, v.Rincian)
	}
}
