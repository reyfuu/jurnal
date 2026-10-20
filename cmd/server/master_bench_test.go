package main

import (
	"fmt"
	"testing"
	"time"

	"pajak/internal/finance"
	"pajak/internal/master"
)

// Benchmark Panel Master pada skala target agent.md §3: 1.000 tenant dengan
// 4.000–10.000 transaksi/tenant/tahun. Memvalidasi kandidat smokinggun
// cmd/server/master.go:279 (nested-or-callback-loop): klienRows memindai
// seluruh transaksi setiap tenant pada tiap request panel.

func benchTenants(b *testing.B, n, txPerTenant int) *Server {
	b.Helper()
	b.Setenv("MASTER_EMAIL", "andi@masterpajak.id")
	b.Setenv("MASTER_PASSWORD", "rahasia-master-1")
	srv := NewServer()
	srv.tenantsMu.Lock()
	for id := 2; id <= n+1; id++ {
		t := newTenant(id)
		txs := make([]finance.Transaction, 0, txPerTenant)
		for i := 0; i < txPerTenant; i++ {
			txs = append(txs, finance.Transaction{
				ID:      i + 1,
				Tanggal: fmt.Sprintf("2026-%02d-%02d", 1+i%12, 1+i%28),
				Jenis:   "Penjualan",
				Nominal: int64(10_000 + i),
				Lunas:   true,
			})
		}
		t.mu.Lock()
		t.txs = txs
		t.nextTxID = txPerTenant + 1
		t.mu.Unlock()
		srv.tenants[id] = t
	}
	srv.tenantsMu.Unlock()
	return srv
}

func BenchmarkKlienRows(b *testing.B) {
	for _, c := range []struct{ tenants, tx int }{{100, 4_000}, {1_000, 4_000}, {1_000, 10_000}} {
		srv := benchTenants(b, c.tenants, c.tx)
		ak := master.Akses{Role: master.Role{ID: master.RoleMaster}}
		now := time.Now()

		// hitungTx=true: daftar klien, satu-satunya halaman yang menampilkan kolom itu.
		b.Run(fmt.Sprintf("daftar/tenant=%d/tx=%d", c.tenants, c.tx), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if len(srv.klienRows(ak, now, true)) == 0 {
					b.Fatal("tidak ada klien")
				}
			}
		})

		// hitungTx=false: ringkasan, pembayaran, tim.
		b.Run(fmt.Sprintf("ringkasan/tenant=%d/tx=%d", c.tenants, c.tx), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if len(srv.klienRows(ak, now, false)) == 0 {
					b.Fatal("tidak ada klien")
				}
			}
		})

		// satu klien: halaman detail.
		b.Run(fmt.Sprintf("detail/tenant=%d/tx=%d", c.tenants, c.tx), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if len(srv.klienRowsFilter(ak, now, true, 2)) != 1 {
					b.Fatal("detail harus satu klien")
				}
			}
		})
	}
}
