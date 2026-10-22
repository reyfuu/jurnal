package main

import (
	"net/http"
	"strings"
	"testing"

	"pajak/internal/finance"
)

func usahaPT(t *testing.T) (*Server, *http.Cookie) {
	t.Helper()
	srv, qa := usahaBaru(t)
	pt := strings.Replace(dagangCV, `"bentuk_usaha":"cv"`, `"bentuk_usaha":"pt"`, 1)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/business", pt); rec.Code != http.StatusOK {
		t.Fatalf("profil PT: %d %s", rec.Code, rec.Body)
	}
	return srv, qa
}

// Bugslist 22 Okt #5: "Pembagian Dividen" hanya untuk PT; CV memakai Ambil Uang Pribadi.
func TestDividen_HanyaPT(t *testing.T) {
	srv, qa := usahaBaru(t) // CV
	rec := do(srv, qa, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-02-15","jenis":"Pembagian Dividen","nominal":1000000}`)
	if rec.Code == http.StatusCreated || !strings.Contains(rec.Body.String(), "PT") {
		t.Fatalf("CV mencatat dividen: %d %s", rec.Code, rec.Body)
	}
	var cv struct{ Jenis []finance.Jenis }
	cv = getJSON[struct{ Jenis []finance.Jenis }](t, srv, qa, "/api/v1/transactions")
	for _, j := range cv.Jenis {
		if j.Nama == finance.JenisDividen {
			t.Fatal("pilihan jenis CV tidak boleh memuat Pembagian Dividen")
		}
	}
}

// Dividen PT yang belum dibayar masuk Utang Dividen (2115) di neraca & neraca saldo, lalu
// hilang setelah dibayar; neraca saldo seimbang di kedua bulan.
func TestDividen_PTUtangDividen(t *testing.T) {
	srv, qa := usahaPT(t)
	for _, b := range []string{
		`{"tanggal":"2026-01-10","jenis":"Setor Modal","nominal":10000000,"lunas":true}`,
		`{"tanggal":"2026-01-12","jenis":"Penjualan","nominal":5000000,"lunas":true}`,
		`{"tanggal":"2026-02-15","jenis":"Pembagian Dividen","nominal":2000000,"lunas":true,"tanggal_lunas":"2026-03-10"}`,
	} {
		buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", b)
	}
	bs := getJSON[finance.BalanceSheet](t, srv, qa, "/api/v1/reports/balance-sheet?period=2026-02")
	if !bs.PT || bs.Liabilities.DividendsPayable != 2_000_000 || bs.Equity.Dividen != 2_000_000 || !bs.IsBalanced {
		t.Fatalf("neraca Feb: %+v", bs)
	}
	for p, mau := range map[string]int64{"2026-02": 2_000_000, "2026-03": 0} {
		var d, k, utangDividen int64
		for _, b := range getJSON[coaReport](t, srv, qa, "/api/v1/reports/coa?period="+p).TrialBalance {
			d, k = d+b.Debit, k+b.Kredit
			if b.Kode == "2115" {
				utangDividen = b.Saldo
			}
		}
		if d != k || utangDividen != mau {
			t.Errorf("%s: neraca saldo %d vs %d, 2115 = %d (mau %d)", p, d, k, utangDividen, mau)
		}
	}
}
