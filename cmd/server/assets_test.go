package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"pajak/internal/finance"
)

func assetsOf(t *testing.T, srv *Server, c *http.Cookie) finance.AssetSummary {
	t.Helper()
	var as finance.AssetSummary
	rec := do(srv, c, http.MethodGet, "/api/v1/assets?period=2026-08", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list assets: %d %s", rec.Code, rec.Body)
	}
	if err := json.NewDecoder(rec.Body).Decode(&as); err != nil {
		t.Fatalf("decode assets: %v", err)
	}
	return as
}

func incomeOf(t *testing.T, srv *Server, c *http.Cookie) finance.IncomeStatement {
	t.Helper()
	var is finance.IncomeStatement
	_ = json.NewDecoder(do(srv, c, http.MethodGet, "/api/v1/reports/financial?period=2026-08", "").Body).Decode(&is)
	return is
}

func TestAssetsEndpoint_MasterDanRegister(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	as := assetsOf(t, srv, owner)

	if len(as.Assets) != 6 || as.BebanTotal != 6_550_000 {
		t.Fatalf("register seed: %d aset, beban %d", len(as.Assets), as.BebanTotal)
	}
	// Master kelompok PMK ikut dikirim supaya form tidak perlu menebak tarif.
	if len(as.Kelompok) != 6 || len(as.JenisHarta) == 0 {
		t.Fatalf("master: %d kelompok, %d jenis harta", len(as.Kelompok), len(as.JenisHarta))
	}
	k, ok := finance.KelompokOf("k1")
	if !ok || k.MasaManfaat != 48 || k.TarifGLBps != 2500 || k.TarifSMBps != 5000 {
		t.Errorf("Kelompok 1 PMK: %+v", k)
	}
}

func TestAssetsCRUD_UpdatesLaporan(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	base := incomeOf(t, srv, owner)

	// Oven Kelompok 1 garis lurus: 24jt / 48 bulan = 500rb per bulan.
	rec := do(srv, owner, http.MethodPost, "/api/v1/assets",
		`{"nama":"Oven Deck 2 Pintu","jenis_harta":"peralatan_restoran","metode":"garis_lurus","tanggal_perolehan":"2026-02-01","harga_perolehan":24000000}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	var created finance.Asset
	_ = json.NewDecoder(rec.Body).Decode(&created)
	if created.Kelompok != "k1" {
		t.Fatalf("kelompok harus terisi dari jenis harta: %+v", created)
	}

	if got := incomeOf(t, srv, owner); got.Depreciation != base.Depreciation+500_000 {
		t.Errorf("penyusutan setelah tambah aset: want %d, got %d", base.Depreciation+500_000, got.Depreciation)
	}
	if got := incomeOf(t, srv, owner); got.NetProfit != base.NetProfit-500_000 {
		t.Errorf("laba bersih harus turun sebesar penyusutan: want %d, got %d", base.NetProfit-500_000, got.NetProfit)
	}

	// Ganti ke saldo menurun 50%: 24jt x 50% x 11/12 = 11jt untuk 2026 → 1jt per bulan.
	path := "/api/v1/assets/" + strconv.Itoa(created.ID)
	rec = do(srv, owner, http.MethodPut, path,
		`{"nama":"Oven Deck 2 Pintu","jenis_harta":"peralatan_restoran","metode":"saldo_menurun","tanggal_perolehan":"2026-02-01","harga_perolehan":24000000}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec.Code, rec.Body)
	}
	if got := incomeOf(t, srv, owner); got.Depreciation != base.Depreciation+1_000_000 {
		t.Errorf("penyusutan saldo menurun: want %d, got %d", base.Depreciation+1_000_000, got.Depreciation)
	}

	if rec := do(srv, owner, http.MethodDelete, path, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}
	if got := incomeOf(t, srv, owner); got.NetProfit != base.NetProfit {
		t.Errorf("laba kembali seperti semula setelah aset dihapus: want %d, got %d", base.NetProfit, got.NetProfit)
	}
	if rec := do(srv, owner, http.MethodDelete, path, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("delete dua kali: want 404, got %d", rec.Code)
	}
}

func TestAssets_Validation(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	cases := []struct{ name, body string }{
		{"nama kosong", `{"nama":"","kelompok":"k1","tanggal_perolehan":"2026-02-01","harga_perolehan":1000000}`},
		{"tanggal salah", `{"nama":"Oven","kelompok":"k1","tanggal_perolehan":"01-02-2026","harga_perolehan":1000000}`},
		{"harga nol", `{"nama":"Oven","kelompok":"k1","tanggal_perolehan":"2026-02-01","harga_perolehan":0}`},
		{"kelompok tak dikenal", `{"nama":"Oven","kelompok":"k9","tanggal_perolehan":"2026-02-01","harga_perolehan":1000000}`},
		{"bangunan saldo menurun", `{"nama":"Ruko","kelompok":"bangunan_permanen","metode":"saldo_menurun","tanggal_perolehan":"2026-02-01","harga_perolehan":1000000}`},
	}
	for _, c := range cases {
		if rec := do(srv, owner, http.MethodPost, "/api/v1/assets", c.body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: want 400, got %d %s", c.name, rec.Code, rec.Body)
		}
	}
}

// Register aset ikut hak akses "laporan-keuangan"; Kasir tidak boleh melihat atau mengubahnya.
func TestAssets_RoleAccess(t *testing.T) {
	srv := NewServer()
	kasir := login(t, srv, "kasir@kedaikopisenja.id")
	admin := login(t, srv, "admin@kedaikopisenja.id")

	for _, tc := range []struct {
		c    *http.Cookie
		want int
	}{{kasir, http.StatusForbidden}, {admin, http.StatusOK}} {
		if rec := do(srv, tc.c, http.MethodGet, "/api/v1/assets", ""); rec.Code != tc.want {
			t.Errorf("GET /assets: want %d, got %d", tc.want, rec.Code)
		}
	}
	if rec := do(srv, kasir, http.MethodPost, "/api/v1/assets", `{"nama":"X","kelompok":"k1","tanggal_perolehan":"2026-02-01","harga_perolehan":1}`); rec.Code != http.StatusForbidden {
		t.Errorf("POST /assets sebagai kasir: want 403, got %d", rec.Code)
	}
}

// Transaksi "Beli Alat / Aset" langsung muncul sebagai draft yang tinggal dilengkapi.
func TestAssets_DraftDariTransaksi(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")

	rec := do(srv, owner, http.MethodPost, "/api/v1/transactions",
		`{"tanggal":"2026-08-12","jenis":"Beli Alat / Aset","nominal":12000000,"keterangan":"Mesin Es Batu","lunas":true}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create tx: %d %s", rec.Code, rec.Body)
	}
	as := assetsOf(t, srv, owner)
	if as.DraftCount != 1 {
		t.Fatalf("want 1 draft aset, got %d", as.DraftCount)
	}
	if as.TotalNilaiBuku != 200_525_000+12_000_000 {
		t.Errorf("draft harus masuk nilai buku: got %d", as.TotalNilaiBuku)
	}

	var bs finance.BalanceSheet
	_ = json.NewDecoder(do(srv, owner, http.MethodGet, "/api/v1/reports/balance-sheet?period=2026-08", "").Body).Decode(&bs)
	if !bs.IsBalanced {
		t.Errorf("neraca harus tetap seimbang dengan draft aset: %+v", bs)
	}
}
