package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"pajak/internal/finance"
)

// Rekening bawaan Bank: transaksi baru, pelunasan utang, dan pelunasan saldo awal bergerak di
// akun Bank (1102), bukan Kas (1101); laporan per akun tetap seimbang di bulan berikutnya.
func TestRekeningBawaanBank(t *testing.T) {
	srv, qa := usahaBaru(t)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/opening", `{"tanggal":"2026-01-01","rincian":[{"nama":"Bank","kategori":"kas","nominal":10000000},{"nama":"Utang Usaha","kategori":"utang","nominal":2000000},{"nama":"Modal Pemilik","kategori":"modal","nominal":8000000}]}`); rec.Code != http.StatusOK {
		t.Fatalf("saldo awal: %d %s", rec.Code, rec.Body)
	}
	sewa := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-01-06","jenis":"Sewa Tempat","nominal":1000000,"lunas":true}`)
	if sewa.Lawan != "1102" {
		t.Fatalf("transaksi baru lawan %q, mau 1102 Bank", sewa.Lawan)
	}
	gofood := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-01-07","jenis":"Penjualan","nominal":300000,"lunas":true,"channel":"GoFood"}`)
	if gofood.Lawan != "" {
		t.Errorf("penjualan GoFood tetap ke saldo marketplace, lawan %q", gofood.Lawan)
	}
	bahan := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-01-08","jenis":"Biaya Lain","nominal":500000,"lunas":false}`)
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions/"+strconv.Itoa(bahan.ID)+"/lunas", `{"tanggal":"2026-02-03"}`); rec.Code != http.StatusOK {
		t.Fatalf("pelunasan utang: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/opening/pelunasan", `{"jenis":"utang","tanggal":"2026-01-20","nominal":2000000}`); rec.Code != http.StatusCreated {
		t.Fatalf("pelunasan saldo awal: %d %s", rec.Code, rec.Body)
	}
	saldo := func(period string) map[string]int64 {
		var rep struct {
			TrialBalance []struct {
				Kode  string `json:"kode"`
				Saldo int64  `json:"saldo"`
			} `json:"trial_balance"`
			BalanceSheet struct {
				Seimbang bool `json:"seimbang"`
			} `json:"balance_sheet"`
		}
		_ = json.Unmarshal(do(srv, qa, http.MethodGet, "/api/v1/reports/coa?period="+period, "").Body.Bytes(), &rep)
		if !rep.BalanceSheet.Seimbang {
			t.Errorf("%s laporan per akun tidak seimbang", period)
		}
		out := map[string]int64{}
		for _, b := range rep.TrialBalance {
			out[b.Kode] = b.Saldo
		}
		return out
	}
	// Januari: bank 10 jt - sewa 1 jt - pelunasan saldo awal 2 jt = 7 jt; kas tidak bergerak
	if s := saldo("2026-01"); s["1102"] != 7_000_000 || s["1101"] != 0 {
		t.Errorf("Januari: bank %d kas %d", s["1102"], s["1101"])
	}
	// Februari: utang bahan dilunasi lewat bank → 6,5 jt; saldo bank Januari terbawa
	if s := saldo("2026-02"); s["1102"] != 6_500_000 || s["1101"] != 0 {
		t.Errorf("Februari: bank %d kas %d", s["1102"], s["1101"])
	}
}
