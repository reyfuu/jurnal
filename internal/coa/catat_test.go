package coa

import (
	"testing"

	"pajak/internal/finance"
)

// Contoh preview: nota belanja dipecah ke 3 akun, dibayar sebagian tunai sisanya utang.
func TestPecahCatat_SebagianTunaiSisanyaUtang(t *testing.T) {
	p := finance.BusinessProfile{JenisUsaha: "manufaktur", BentukUsaha: "op", MetodePersediaan: "periodik"}
	peta := PetaDari(Manufaktur)
	baris := []BarisCatat{
		{Akun: "1301", Keterangan: "Gula", Debit: 87_500},
		{Akun: "1302", Keterangan: "Plastik", Debit: 30_000},
		{Akun: "6106", Keterangan: "Ongkir", Debit: 10_500},
		{Akun: "1102", Kredit: 100_000},
		{Akun: "2101", Kredit: 28_000, JatuhTempo: "2026-11-21"},
	}
	txs, err := PecahCatat(baris, Manufaktur, p, peta)
	if err != nil {
		t.Fatal(err)
	}
	type ringkas struct {
		akun, lawan string
		n           int64
		lunas       bool
	}
	var got []ringkas
	for _, x := range txs {
		got = append(got, ringkas{x.Akun, x.Lawan, x.Nominal, x.Lunas})
	}
	want := []ringkas{{"1301", "1102", 87_500, true}, {"1302", "1102", 12_500, true}, {"1302", "", 17_500, false}, {"6106", "", 10_500, false}}
	if len(got) != len(want) {
		t.Fatalf("%+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("transaksi %d = %+v, mau %+v", i, got[i], want[i])
		}
	}
	if txs[0].Jenis != "Beli Bahan / Stok" || txs[3].Jenis != "Ongkir & Transport" || txs[2].JatuhTempo != "2026-11-21" {
		t.Fatalf("jenis/tempo: %+v", txs)
	}
	// Susun ulang untuk layar ubah: baris sama per akun.
	balik := SusunCatat(txs, p, peta)
	total := map[string][2]int64{}
	for _, b := range balik {
		v := total[b.Akun]
		total[b.Akun] = [2]int64{v[0] + b.Debit, v[1] + b.Kredit}
	}
	if total["1302"] != [2]int64{30_000, 0} || total["1102"] != [2]int64{0, 100_000} || total["2101"] != [2]int64{0, 28_000} {
		t.Fatalf("susun ulang: %+v", balik)
	}
}

func TestPecahCatat_PenjualanDanPenolakan(t *testing.T) {
	p := finance.BusinessProfile{JenisUsaha: "perdagangan", BentukUsaha: "op", MetodePersediaan: "periodik"}
	peta := PetaDari(Master)
	txs, err := PecahCatat([]BarisCatat{{Akun: "1102", Debit: 600_000}, {Akun: "1201", Debit: 400_000}, {Akun: "4101", Kredit: 1_000_000}}, Master, p, peta)
	if err != nil || len(txs) != 2 || txs[0].Jenis != "Penjualan" || !txs[0].Lunas || txs[0].Lawan != "1102" || txs[1].Lunas {
		t.Fatalf("penjualan: %+v %v", txs, err)
	}
	for nama, b := range map[string][]BarisCatat{
		"tidak seimbang": {{Akun: "6190", Debit: 10}, {Akun: "1101", Kredit: 9}},
		"dua sisi pokok": {{Akun: "6190", Debit: 10}, {Akun: "4101", Kredit: 10}},
		"utang di debit": {{Akun: "2101", Debit: 10}, {Akun: "4101", Kredit: 10}},
		"debit & kredit": {{Akun: "6190", Debit: 10, Kredit: 10}, {Akun: "1101", Kredit: 0}},
	} {
		if _, err := PecahCatat(b, Master, p, peta); err == nil {
			t.Errorf("%s harus ditolak", nama)
		}
	}
}
