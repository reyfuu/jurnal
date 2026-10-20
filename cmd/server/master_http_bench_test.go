package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pajak/internal/finance"
)

// Benchmark Panel Master di batas HTTP yang sebenarnya dipakai UI: lewat
// srv.ServeHTTP, termasuk guard, akses, dan serialisasi JSON. Ini melengkapi
// BenchmarkKlienRows yang memanggil fungsi internal.
//
// Sengaja hanya memakai endpoint publik, supaya file ini bisa dijalankan apa
// adanya pada revisi sebelum maupun sesudah perbaikan `hitungTx` — jadi angka
// sebelum/sesudah benar-benar sebanding.
//
// Skala mengikuti agent.md §3: 1.000 usaha, 4.000 transaksi/usaha/tahun.

func benchMasterServer(b *testing.B, tenants, txPerTenant int) (*Server, *http.Cookie) {
	b.Helper()
	b.Setenv("MASTER_EMAIL", "andi@masterpajak.id")
	b.Setenv("MASTER_PASSWORD", "rahasia-master-1")
	srv := NewServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"email":"andi@masterpajak.id","password":"rahasia-master-1"}`))
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		b.Fatalf("login master: %d %s", rec.Code, rec.Body)
	}
	var ck *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookie {
			ck = c
		}
	}
	if ck == nil {
		b.Fatal("tanpa cookie sesi")
	}

	srv.tenantsMu.Lock()
	for id := 2; id <= tenants+1; id++ {
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
	return srv, ck
}

func BenchmarkMasterEndpoints(b *testing.B) {
	const tenants, tx = 1_000, 4_000
	srv, ck := benchMasterServer(b, tenants, tx)

	for _, c := range []struct{ name, path string }{
		{"ringkasan", "/api/v1/master/ringkasan"},
		{"pembayaran", "/api/v1/master/pembayaran"},
		{"tim", "/api/v1/master/tim"},
		{"detail-satu-klien", "/api/v1/master/klien/2"},
		{"daftar-klien", "/api/v1/master/klien"},
	} {
		b.Run(fmt.Sprintf("%s/tenant=%d/tx=%d", c.name, tenants, tx), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				rec := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, c.path, nil)
				req.AddCookie(ck)
				srv.ServeHTTP(rec, req)
				if rec.Code != http.StatusOK {
					b.Fatalf("GET %s: %d %s", c.path, rec.Code, rec.Body)
				}
			}
		})
	}
}
