package main

import (
	"net/http"
	"strings"
	"testing"
)

func postSimulasi(t *testing.T, srv *Server, body string) simulasiResp {
	t.Helper()
	var r simulasiResp
	doJSON(t, srv, login(t, srv, "owner@kedaikopisenja.id"), http.MethodPost, "/api/v1/simulasi?period=2026-08", strings.NewReader(body), http.StatusOK, &r)
	return r
}

func TestSimulasi_DasarDariTigaBulanTerakhir(t *testing.T) {
	srv := NewServer()
	r := postSimulasi(t, srv, `{}`)
	d := r.Dasar
	if len(d.Periode) != 3 || d.Periode[2] != "2026-08" || d.Omzet <= 0 || d.HariBuka <= 0 {
		t.Fatalf("dasar = %+v", d)
	}
	if r.Hasil.Laba != d.Laba || r.Hasil.Omzet != d.Omzet {
		t.Errorf("tanpa skenario hasil harus sama dengan dasar: %+v vs %+v", r.Hasil, d)
	}
	if !r.TitikImpas.Ada || r.TitikImpas.PerBulan <= 0 || r.TitikImpas.PerHari <= 0 {
		t.Errorf("titik impas = %+v", r.TitikImpas)
	}
	// Produk resep punya modal dari HPP resep (bukan "–").
	var resep bool
	for _, p := range r.Produk {
		if p.AdaModal && p.Modal == 8_300 && p.Harga > 0 {
			resep = true
		}
	}
	if !resep {
		t.Errorf("produk resep FG-001 (HPP 8.300) tidak ada: %+v", r.Produk)
	}

	naik := postSimulasi(t, srv, `{"harga_bps":500}`)
	if naik.Hasil.Omzet <= d.Omzet || naik.Hasil.Laba <= d.Laba || naik.Dasar.Omzet != d.Omzet {
		t.Errorf("naik harga 5%%: %+v", naik.Hasil)
	}
}

func TestSimulasi_KasirTidakBoleh(t *testing.T) {
	srv := NewServer()
	if rec := do(srv, login(t, srv, "kasir@kedaikopisenja.id"), http.MethodPost, "/api/v1/simulasi", `{}`); rec.Code != http.StatusForbidden {
		t.Errorf("kasir: want 403, got %d", rec.Code)
	}
}

func TestPriceSimulation_MarginPersen(t *testing.T) {
	srv := NewServer()
	var r simulateResp
	doJSON(t, srv, login(t, srv, "owner@kedaikopisenja.id"), http.MethodPost, "/api/v1/pricing/simulate?period=2026-08",
		strings.NewReader(`{"cogs":8000,"fee_bps":0,"margin_bps":6000,"monthly_volume":1000}`), http.StatusOK, &r)
	if r.RecommendedPrice != 20_000 || r.NetTakeHome-r.COGS != 12_000 {
		t.Fatalf("margin 60%% = %+v", r.PriceSimulationResult)
	}
	if rec := do(srv, login(t, srv, "owner@kedaikopisenja.id"), http.MethodPost, "/api/v1/pricing/simulate", `{"cogs":8000,"fee_bps":5000,"margin_bps":6000}`); rec.Code != http.StatusBadRequest {
		t.Errorf("komisi + margin ≥ 100%%: want 400, got %d", rec.Code)
	}
}
