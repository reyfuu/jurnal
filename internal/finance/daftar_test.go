package finance_test

import (
	"testing"

	"pajak/internal/finance"
)

// Pelanggan = pihak transaksi penjualan dalam rentang (nama tanpa beda huruf besar/kecil),
// urut penjualan terbesar; transaksi di luar rentang, bukan penjualan, atau tanpa nama diabaikan.
func TestRekapPelanggan_RentangDanUrutan(t *testing.T) {
	tx := func(tgl, jenis, pihak string, n int64) finance.Transaction {
		return finance.Transaction{Tanggal: tgl, Jenis: jenis, Pihak: pihak, Nominal: n}
	}
	got := finance.RekapPelanggan([]finance.Transaction{
		tx("2026-08-01", "Penjualan", "Toko Andi", 100),
		tx("2026-08-02", "Penjualan", "toko andi ", 50),
		tx("2026-08-03", "Penjualan", "CV Budi", 200),
		tx("2026-07-31", "Penjualan", "CV Lama", 999),
		tx("2026-08-04", "Biaya Lain", "PLN", 300),
		tx("2026-08-05", "Penjualan", "", 70),
	}, "2026-08-01", "2026-08-31")
	if len(got) != 2 || got[0].Nama != "CV Budi" || got[0].Penjualan != 200 || got[1].Nama != "Toko Andi" || got[1].Penjualan != 150 || got[1].Jumlah != 2 {
		t.Fatalf("%+v", got)
	}
}
