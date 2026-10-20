package coa

import (
	"fmt"
	"testing"

	"pajak/internal/finance"
)

// Benchmark untuk memvalidasi kandidat hotspot yang dilaporkan smokinggun di
// jalur Transaksi → Jurnal → Neraca Saldo. Skala mengikuti target di agent.md §3:
// 4.000–10.000 transaksi per tenant per tahun.
//
// Kandidat yang diuji:
//   - internal/coa/journal.go:185 (membership-in-loop: Find di dalam loop Master)
//   - internal/coa/reports.go:27 (membership-in-loop)
//   - internal/finance/transaction.go:212 (membership-in-loop: jenisOf per transaksi)

func benchTxs(n int) []finance.Transaction {
	jenis := finance.JenisList
	out := make([]finance.Transaction, 0, n)
	for i := 0; i < n; i++ {
		j := jenis[i%len(jenis)]
		out = append(out, finance.Transaction{
			ID:         i + 1,
			Tanggal:    fmt.Sprintf("2026-%02d-%02d", 1+i%12, 1+i%28),
			Jenis:      j.Nama,
			Nominal:    int64(10_000 + i*37),
			Keterangan: "benchmark",
			Lunas:      i%3 != 0,
		})
	}
	return out
}

func benchProfile() finance.BusinessProfile {
	return finance.BusinessProfile{
		NamaUsaha:        "Benchmark",
		JenisUsaha:       "perdagangan",
		BentukUsaha:      "cv",
		MetodePersediaan: "periodik",
	}
}

func BenchmarkJournalTrialBalance(b *testing.B) {
	for _, n := range []int{4_000, 10_000} {
		b.Run(fmt.Sprintf("tx=%d", n), func(b *testing.B) {
			p := benchProfile()
			txs := benchTxs(n)
			active := Active(p)
			opening := Opening(50_000_000, 10_000_000, nil)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				lines := Journal(txs, p)
				tb := TrialBalance(active, opening, lines)
				if len(tb) == 0 {
					b.Fatal("neraca saldo kosong")
				}
			}
		})
	}
}

func BenchmarkTrialBalanceOnly(b *testing.B) {
	p := benchProfile()
	active := Active(p)
	opening := Opening(50_000_000, 10_000_000, nil)
	lines := Journal(benchTxs(10_000), p)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if len(TrialBalance(active, opening, lines)) == 0 {
			b.Fatal("neraca saldo kosong")
		}
	}
}
