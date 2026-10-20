package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"pajak/internal/finance"
)

func TestTutupBuku_MenguncidanBukaKembali(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	qa := register(t, srv, registerBody)
	aktifkan(t, srv, qa)
	tx := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-01-10","jenis":"Penjualan","nominal":1000000,"lunas":false}`)

	r := getJSON[tutupBukuResp](t, srv, qa, "/api/v1/tutup-buku?period=2026-01")
	if r.Terkunci || !r.BisaTutup || len(r.Cek) < 5 {
		t.Fatalf("sebelum tutup %+v", r)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/tutup-buku", `{"periode":"2026-01"}`); rec.Code != http.StatusOK {
		t.Fatalf("tutup: %d %s", rec.Code, rec.Body)
	}
	// Januari terkunci: tambah, ubah, hapus, dan pelunasan bertanggal Januari ditolak.
	for _, c := range []struct{ m, p, b string }{
		{http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-01-20","jenis":"Penjualan","nominal":1,"lunas":true}`},
		{http.MethodPut, "/api/v1/transactions/" + itoa(tx.ID), `{"tanggal":"2026-02-01","jenis":"Penjualan","nominal":1,"lunas":true}`},
		{http.MethodDelete, "/api/v1/transactions/" + itoa(tx.ID), ``},
		{http.MethodPost, "/api/v1/transactions/" + itoa(tx.ID) + "/lunas", `{"tanggal":"2026-01-25"}`},
	} {
		if rec := do(srv, qa, c.m, c.p, c.b); rec.Code != http.StatusConflict {
			t.Errorf("%s %s: %d, mau 409", c.m, c.p, rec.Code)
		}
	}
	// Februari masih terbuka, termasuk pelunasan piutang Januari di bulan Februari.
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-02-03","jenis":"Penjualan","nominal":1,"lunas":true}`)
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions/"+itoa(tx.ID)+"/lunas", `{"tanggal":"2026-02-05"}`); rec.Code != http.StatusOK {
		t.Errorf("pelunasan Februari: %d %s", rec.Code, rec.Body)
	}

	// Hanya Owner yang membuka kembali.
	if rec := do(srv, qa, http.MethodPost, "/api/v1/users", `{"name":"Admin","email":"adm@qa.id","role":"admin","active":true,"password":"rahasia123"}`); rec.Code != http.StatusCreated {
		t.Fatalf("buat admin: %d", rec.Code)
	}
	adm := loginPass(t, srv, "adm@qa.id", "rahasia123")
	if rec := do(srv, adm, http.MethodDelete, "/api/v1/tutup-buku", ""); rec.Code != http.StatusForbidden {
		t.Errorf("admin buka kembali: %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodDelete, "/api/v1/tutup-buku", ""); rec.Code != http.StatusOK {
		t.Fatalf("owner buka kembali: %d %s", rec.Code, rec.Body)
	}
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-01-21","jenis":"Penjualan","nominal":1,"lunas":true}`)
}

func TestPelunasanSaldoAwal_DanLangkahMulai(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	qa := register(t, srv, registerBody)
	aktifkan(t, srv, qa)

	langkah := getJSON[[]langkahMulai](t, srv, qa, "/api/v1/mulai")
	if len(langkah) < 4 || langkah[0].Kode != "profil" || langkah[1].Selesai {
		t.Fatalf("langkah awal %+v", langkah)
	}
	if rec := do(srv, qa, http.MethodPut, "/api/v1/business", dagangCV); rec.Code != http.StatusOK {
		t.Fatalf("profil %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodPut, "/api/v1/opening", `{"tanggal":"2026-01-01","kas":5000000,"piutang":2000000,"utang":800000}`); rec.Code != http.StatusOK {
		t.Fatalf("saldo awal %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/opening/pelunasan", `{"jenis":"piutang","tanggal":"2026-02-10","nominal":3000000}`); rec.Code != http.StatusBadRequest {
		t.Errorf("melebihi saldo awal: %d", rec.Code)
	}
	buat[finance.PelunasanAwal](t, srv, qa, "/api/v1/opening/pelunasan", `{"jenis":"piutang","tanggal":"2026-02-10","nominal":1500000}`)
	// Simpan ulang saldo awal tidak menghapus pelunasan yang sudah dicatat.
	do(srv, qa, http.MethodPut, "/api/v1/opening", `{"tanggal":"2026-01-01","kas":5000000,"piutang":2000000,"utang":800000}`)

	bs := getJSON[finance.BalanceSheet](t, srv, qa, "/api/v1/reports/balance-sheet?period=2026-02")
	cf := getJSON[finance.CashFlow](t, srv, qa, "/api/v1/reports/cashflow?period=2026-02")
	if bs.Assets.Receivables != 500_000 || bs.Assets.Cash != 6_500_000 || !bs.IsBalanced || cf.CashIn != 1_500_000 {
		t.Errorf("setelah pelunasan saldo awal: piutang %d kas %d kas masuk %d", bs.Assets.Receivables, bs.Assets.Cash, cf.CashIn)
	}
	var c coaRingkas
	_ = json.NewDecoder(do(srv, qa, http.MethodGet, "/api/v1/reports/coa?period=2026-02", "").Body).Decode(&c)
	if !c.BalanceSheet.Seimbang {
		t.Error("CoA tidak seimbang setelah pelunasan saldo awal")
	}
	d := getJSON[finance.DashboardSnapshot](t, srv, qa, "/api/v1/dashboard?period=2026-02")
	if d.KPIs.Piutang.UnpaidTotal != bs.Assets.Receivables {
		t.Errorf("kartu piutang dashboard %d ≠ neraca %d", d.KPIs.Piutang.UnpaidTotal, bs.Assets.Receivables)
	}
	langkah = getJSON[[]langkahMulai](t, srv, qa, "/api/v1/mulai")
	if !langkah[1].Selesai {
		t.Errorf("langkah saldo awal belum selesai: %+v", langkah[1])
	}
}
