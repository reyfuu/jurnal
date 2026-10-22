package finance

import "testing"

func TestCekUnik(t *testing.T) {
	stok := []StokItem{{ID: 1, SKU: "RM-001", Nama: "Biji Kopi Arabica"}, {ID: 2, SKU: "", Nama: "Gula Aren"}}
	for _, c := range []struct {
		id        int
		sku, nama string
		salah     bool
	}{
		{0, "rm-001", "Kopi Baru", true},             // SKU sama (huruf kecil)
		{0, "RM-009", "  biji  kopi ARABICA ", true}, // nama sama (spasi & huruf)
		{0, "", "Gula Aren", true},                   // nama sama tanpa SKU
		{0, "RM-009", "Susu", false},
		{0, "", "Susu", false},                    // SKU kosong tidak dibandingkan
		{1, "RM-001", "Biji Kopi Arabica", false}, // ubah diri sendiri
		{2, "RM-001", "Gula Aren", true},          // ubah ke SKU barang lain
	} {
		if err := CekUnik(stok, InfoStok, c.id, c.sku, c.nama, "barang"); (err != nil) != c.salah {
			t.Errorf("%+v: err = %v", c, err)
		}
	}
	rows := [][]string{{"SKU", "Nama Barang", "Jumlah"}, {"A1", "Kopi", "1"}, {"a1", "Teh", "1"}, {"B2", "kopi", "1"}, {"C3", "Susu", "2"}}
	list, errs := parseStokRows(rows)
	if len(list) != 2 || len(errs) != 2 {
		t.Errorf("baris kembar di berkas harus ditolak: %d valid, %v", len(list), errs)
	}
}
