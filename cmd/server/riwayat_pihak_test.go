package main

import (
	"net/http"
	"testing"

	"pajak/internal/finance"
)

// Checklist tester No 25: riwayat transaksi per pelanggan / supplier (dicocokkan lewat nama,
// huruf besar/kecil & spasi diabaikan), dengan ringkasan dan yang belum lunas.
func TestRiwayatPihak(t *testing.T) {
	srv, qa := usahaBaru(t)
	for _, tx := range []string{
		`{"tanggal":"2026-10-01","jenis":"Penjualan","nominal":100000,"lunas":true,"pihak":"PT Maju"}`,
		`{"tanggal":"2026-10-05","jenis":"Penjualan","nominal":50000,"lunas":false,"pihak":"pt  maju"}`,
		`{"tanggal":"2026-09-01","jenis":"Penjualan","nominal":7000,"lunas":true,"pihak":"PT Maju"}`,
		`{"tanggal":"2026-10-02","jenis":"Penjualan","nominal":9000,"lunas":true,"pihak":"Toko Lain"}`,
	} {
		buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", tx)
	}
	r := getJSON[riwayatPihak](t, srv, qa, "/api/v1/pihak/riwayat?nama=PT%20Maju&dari=2026-10-01&sampai=2026-10-31")
	if len(r.Transaksi) != 2 || r.Transaksi[0].Tanggal != "2026-10-01" || r.TotalMasuk != 150_000 || r.PiutangTerbuka != 50_000 {
		t.Fatalf("riwayat: %+v", r)
	}
	if rec := do(srv, qa, http.MethodGet, "/api/v1/pihak/riwayat?nama=", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("nama kosong harus ditolak: %d", rec.Code)
	}
}
