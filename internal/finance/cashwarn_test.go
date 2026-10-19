package finance

import (
	"reflect"
	"testing"
)

func TestCashWarnings(t *testing.T) {
	txs := []Transaction{
		{ID: 1, Tanggal: "2026-03-01", Jenis: "Penjualan", Nominal: 2_000_000, Lunas: true},
		{ID: 2, Tanggal: "2026-03-02", Jenis: "Sewa Tempat", Nominal: 5_000_000, Lunas: true},
		{ID: 3, Tanggal: "2026-03-02", Jenis: "Gaji & Upah", Nominal: 1_000_000, Lunas: true},
		{ID: 4, Tanggal: "2026-03-03", Jenis: "Beli Bahan / Stok", Nominal: 500_000, Lunas: false}, // utang, kas tidak bergerak
		{ID: 5, Tanggal: "2026-03-03", Jenis: "Penjualan", Nominal: 900_000, Lunas: true},
		{ID: 6, Tanggal: "2026-03-01", Jenis: "Penjualan", Nominal: 3_000_000, Lunas: true, TanggalLunas: "2026-03-05"}, // kas masuk tgl 5
	}
	got := CashWarnings(txs, 1_000_000)
	want := []CashWarning{
		{Tanggal: "2026-03-02", Saldo: -3_000_000, Penyebab: []int{2, 3}},
		{Tanggal: "2026-03-03", Saldo: -2_100_000, Penyebab: nil},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

func TestCashWarnings_NoneWhenPositive(t *testing.T) {
	txs := []Transaction{{ID: 1, Tanggal: "2026-03-01", Jenis: "Biaya Lain", Nominal: 100, Lunas: true}}
	if got := CashWarnings(txs, 1_000); len(got) != 0 {
		t.Fatalf("want no warnings, got %+v", got)
	}
}
