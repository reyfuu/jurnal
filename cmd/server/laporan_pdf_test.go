package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"pajak/internal/finance"
)

func TestLaporanKeuanganPDF(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")

	rec := do(srv, owner, http.MethodGet, "/api/v1/reports/pdf?period=2026-08&bagian=semua", "")
	body := rec.Body.String()
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/pdf" {
		t.Fatalf("semua: %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "Laporan-Keuangan-2026-08.pdf") {
		t.Errorf("nama berkas %q", cd)
	}
	for _, mau := range []string{"Laporan Laba Rugi", "Posisi Keuangan", "Metode Langsung", "Agustus 2026"} {
		if !strings.Contains(body, mau) {
			t.Errorf("PDF semua tidak memuat %q", mau)
		}
	}
	if strings.Contains(body, "Register Aset") {
		t.Error("tarik semua = laba rugi, neraca, arus kas saja")
	}

	cek := func(q, mau string) {
		t.Helper()
		rec := do(srv, owner, http.MethodGet, "/api/v1/reports/pdf?"+q, "")
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), mau) {
			t.Errorf("%s: %d, mau memuat %q", q, rec.Code, mau)
		}
	}
	cek("period=2026-08&bagian=laba-rugi", "Laporan Laba Rugi")
	cek("period=2026-08&bagian=neraca", "Neraca")
	cek("period=2026-08&bagian=arus-kas&metode=tidak-langsung", "Metode Tidak Langsung")
	cek("period=2026-08&bagian=aset", "Register Aset")

	for _, q := range []string{"period=2026-08&bagian=pajak", "period=2026-13&bagian=semua", "period=2026-08&bagian=arus-kas&metode=aneh"} {
		if rec := do(srv, owner, http.MethodGet, "/api/v1/reports/pdf?"+q, ""); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d, mau 400", q, rec.Code)
		}
	}
}

func TestBandingMultiPeriode(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")

	laba := map[string]int64{}
	for _, p := range []string{"2026-06", "2026-07", "2026-08"} {
		var is finance.IncomeStatement
		_ = json.NewDecoder(do(srv, owner, http.MethodGet, "/api/v1/reports/financial?period="+p, "").Body).Decode(&is)
		laba[p] = is.NetProfit
	}

	rec := do(srv, owner, http.MethodGet, "/api/v1/reports/banding?periode=2026-08,2026-06,2026-07", "")
	var b finance.Banding
	if err := json.NewDecoder(rec.Body).Decode(&b); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("banding: %d %v", rec.Code, err)
	}
	if len(b.Kolom) != 3 || b.Kolom[0].Periode != "2026-06" || len(b.Bagian) != 3 {
		t.Fatalf("kolom/bagian: %+v %d", b.Kolom, len(b.Bagian))
	}
	var lb finance.BarisBanding
	for _, r := range b.Bagian[0].Baris {
		if r.Kunci == "laba_bersih" {
			lb = r
		}
	}
	if lb.Nilai[0] != laba["2026-06"] || lb.Nilai[2] != laba["2026-08"] || lb.Total != laba["2026-06"]+laba["2026-07"]+laba["2026-08"] {
		t.Errorf("laba bersih banding %+v, mau %v", lb, laba)
	}
	if laba["2026-08"] == 0 {
		t.Error("data demo tanpa laba; tes tidak menguji apa pun")
	}

	rec = do(srv, owner, http.MethodGet, "/api/v1/reports/banding.pdf?periode=2026-07,2026-08", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "MediaBox [0 0 841.89 595.28]") ||
		!strings.Contains(rec.Header().Get("Content-Disposition"), "Perbandingan-2026-07-sd-2026-08.pdf") {
		t.Errorf("banding pdf: %d %q", rec.Code, rec.Header().Get("Content-Disposition"))
	}
	for _, q := range []string{"", "periode=2026-08,2026-08", "periode=juli"} {
		if rec := do(srv, owner, http.MethodGet, "/api/v1/reports/banding?"+q, ""); rec.Code != http.StatusBadRequest {
			t.Errorf("banding?%s: %d, mau 400", q, rec.Code)
		}
	}
}

// Neraca per tanggal bebas: API dan PDF ("Per 15 Agustus 2026").
func TestNeracaPerTanggal(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")

	var bs finance.BalanceSheet
	rec := do(srv, owner, http.MethodGet, "/api/v1/reports/balance-sheet?tanggal=2026-08-15", "")
	_ = json.NewDecoder(rec.Body).Decode(&bs)
	if rec.Code != http.StatusOK || bs.Tanggal != "2026-08-15" || !bs.IsBalanced {
		t.Fatalf("neraca 15 Agustus: %d %+v", rec.Code, bs)
	}
	if rec := do(srv, owner, http.MethodGet, "/api/v1/reports/balance-sheet?tanggal=15-08-2026", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("format tanggal salah harus ditolak: %d", rec.Code)
	}
	rec = do(srv, owner, http.MethodGet, "/api/v1/reports/pdf?period=2026-08&bagian=neraca&tanggal=2026-08-15", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Per 15 Agustus 2026") || !strings.Contains(rec.Header().Get("Content-Disposition"), "Neraca-2026-08-15.pdf") {
		t.Errorf("PDF neraca per tanggal: %d %q", rec.Code, rec.Header().Get("Content-Disposition"))
	}
}
