package coa

import (
	"testing"

	"pajak/internal/finance"
)

// Jalur buku berjalan yang dipakai Buku Besar & laporan per akun (JournalBerjalan + SaldoBawaan
// per bulan) dan laporan keuangan (BuildLaporanBerjalan), 10.000 transaksi setahun (agent.md §3).
// Jalankan sebelum/sesudah mengubah jalur Transaksi → Jurnal → Laporan.
func BenchmarkJournalBerjalan(b *testing.B) {
	txs, p := benchTxs(10_000), benchProfile()
	peta := PetaDari(Master)
	b.ResetTimer()
	for range b.N {
		_ = JournalBerjalan(txs, "2026-06", p, peta)
		_ = SaldoBawaan(txs, "2026-06-01", p, peta)
	}
}

func BenchmarkLaporanBerjalan(b *testing.B) {
	txs := benchTxs(10_000)
	ob := finance.OpeningBalance{Tanggal: "2026-01-01", Kas: 100_000_000}
	b.ResetTimer()
	for range b.N {
		_ = finance.BuildLaporanBerjalan(txs, nil, "2026-06", ob, finance.KartuStok{})
	}
}
