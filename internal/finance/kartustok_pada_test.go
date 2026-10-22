package finance

import "testing"

// Checklist tester No 22: posisi kartu stok per tanggal, salinan (posisi sekarang tidak berubah).
func TestKartuPada(t *testing.T) {
	k := KartuStok{Mulai: "2026-10-01", Baris: []BarisKartu{
		{Tanggal: "2026-10-01", StokID: 1, Qty: 10_000, Nilai: 100_000, SaldoQty: 10_000, SaldoNilai: 100_000},
		{Tanggal: "2026-10-10", StokID: 1, Qty: -4_000, Nilai: -40_000, SaldoQty: 6_000, SaldoNilai: 60_000},
		{Tanggal: "2026-10-20", StokID: 1, Qty: 2_000, Nilai: 22_000, SaldoQty: 8_000, SaldoNilai: 82_000},
	}, Posisi: map[int]PosisiStok{1: {Qty: 8_000, Nilai: 82_000}}}
	for _, c := range []struct {
		tgl string
		mau PosisiStok
	}{{"2026-09-30", PosisiStok{}}, {"2026-10-15", PosisiStok{6_000, 60_000}}, {"2027-01-01", PosisiStok{8_000, 82_000}}} {
		if got := k.Pada(c.tgl).Posisi[1]; got != c.mau {
			t.Errorf("%s: %+v, mau %+v", c.tgl, got, c.mau)
		}
	}
	if k.Posisi[1] != (PosisiStok{8_000, 82_000}) {
		t.Fatal("Pada tidak boleh mengubah kartu asli")
	}
}
