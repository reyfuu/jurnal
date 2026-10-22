package coa

import (
	"testing"

	"pajak/internal/finance"
)

func saldoPerAkun(lines []Line) map[string]int64 {
	out := map[string]int64{}
	for _, l := range lines {
		out[l.Kode] += l.Debit - l.Kredit
	}
	return out
}

// Penjualan tempo yang dibayar sebagian di bulan yang sama tetap lewat Piutang: Dr Piutang di
// tanggal jual, Dr Bank / Cr Piutang di tanggal bayar (rekening pilihan pembayaran).
func TestJournalBerjalan_CicilanSebulan(t *testing.T) {
	p := finance.BusinessProfile{BentukUsaha: "cv", MetodePersediaan: "periodik"}
	peta := PetaDari(Master)
	txs := []finance.Transaction{{ID: 1, Tanggal: "2026-01-05", Jenis: "Penjualan", Nominal: 300_000}}
	txs, err := finance.Bayar(txs, finance.Pembayaran{ID: 7, Tanggal: "2026-01-20", Jenis: finance.BayarPiutang, Lawan: "1102",
		Alokasi: []finance.Alokasi{{TransaksiID: 1, Nominal: 200_000}}})
	if err != nil {
		t.Fatal(err)
	}
	lines := JournalBerjalan(txs, "2026-01", p, peta)
	s := saldoPerAkun(lines)
	if s["1201"] != 100_000 || s["1102"] != 200_000 || s["4101"] != -300_000 || s["1101"] != 0 {
		t.Fatalf("saldo per akun %v", s)
	}
	for _, l := range lines {
		if l.Kode == "1102" && l.Tanggal != "2026-01-20" {
			t.Errorf("kas bergerak di tanggal bayar, bukan %s", l.Tanggal)
		}
	}
	// Bulan berikutnya: bank 200rb dibawa di akun bank (saldo bawaan), bukan di Kas.
	if b := SaldoBawaan(txs, "2026-02-01", p, peta); b["1102"] != 200_000 {
		t.Fatalf("saldo bawaan %v", b)
	}
}

// Transaksi lunas lama yang tanggal lunasnya sesudah tanggal transaksi (bulan sama) juga lewat
// Piutang; yang lunas di tanggal transaksi langsung ke kas, memakai rekening pilihan (Lawan).
func TestJournalBerjalan_LunasTanggalBeda(t *testing.T) {
	p := finance.BusinessProfile{BentukUsaha: "cv", MetodePersediaan: "periodik"}
	peta := PetaDari(Master)
	txs := []finance.Transaction{
		{ID: 1, Tanggal: "2026-01-05", Jenis: "Penjualan", Nominal: 300_000, Lunas: true, TanggalLunas: "2026-01-25"},
		{ID: 2, Tanggal: "2026-01-06", Jenis: "Biaya Lain", Nominal: 50_000, Lunas: true, Lawan: "1102"},
	}
	var dariPiutang, bankBiaya bool
	for _, l := range JournalBerjalan(txs, "2026-01", p, peta) {
		dariPiutang = dariPiutang || (l.TxID == 1 && l.Kode == "1201" && l.Tanggal == "2026-01-05" && l.Debit == 300_000)
		bankBiaya = bankBiaya || (l.TxID == 2 && l.Kode == "1102" && l.Kredit == 50_000)
	}
	if !dariPiutang || !bankBiaya {
		t.Fatalf("lewat piutang %v, biaya dari bank pilihan %v", dariPiutang, bankBiaya)
	}
}
