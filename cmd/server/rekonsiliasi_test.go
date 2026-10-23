package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"pajak/internal/finance"
)

// Rekonsiliasi lewat API: omzet dari pembukuan, setoran per masa, unduhan Excel & PDF, kunci
// paket Pro, dashboard & Tutup Buku memakai hitungan yang sama.
func TestRekonsiliasiAPI(t *testing.T) {
	srv, qa := usahaBaru(t) // CV: terutang sejak rupiah pertama
	if rec := do(srv, qa, http.MethodPut, "/api/v1/opening", `{"tanggal":"2026-01-01","rincian":[{"nama":"Kas","kategori":"kas","nominal":10000000},{"nama":"Modal Pemilik","kategori":"modal","nominal":10000000}]}`); rec.Code != http.StatusOK {
		t.Fatalf("saldo awal: %d %s", rec.Code, rec.Body)
	}
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-03-05","jenis":"Penjualan","nominal":20000000,"lunas":true}`)
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-04-05","jenis":"Penjualan","nominal":10000000,"lunas":true}`)
	s := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-04-10","jenis":"Setor PPh Final","nominal":100000,"lunas":true}`)
	if s.MasaPajak != "2026-03" {
		t.Fatalf("masa bawaan setoran: %q", s.MasaPajak)
	}
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-05-10","jenis":"Setor PPh Final","nominal":30000,"lunas":true,"masa_pajak":"2026-04"}`)

	var rk finance.Rekonsiliasi
	rec := do(srv, qa, http.MethodGet, "/api/v1/reports/rekonsiliasi?tahun=2026", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &rk); rec.Code != http.StatusOK || err != nil {
		t.Fatalf("rekonsiliasi: %d %s", rec.Code, rec.Body)
	}
	if b := rk.Bulan[2]; b.Omzet != 20_000_000 || b.PPhTerutang != 100_000 || b.Status != "cocok" {
		t.Errorf("Maret: %+v", b)
	}
	if b := rk.Bulan[3]; b.PPhTerutang != 50_000 || b.PPhDisetor != 30_000 || b.Status != "kurang" {
		t.Errorf("April: %+v", b)
	}
	if rk.Subjek != "Badan" || rk.KurangBayar != 20_000 || len(rk.Peringatan) == 0 {
		t.Errorf("ringkasan: %+v", rk)
	}
	if rec := do(srv, qa, http.MethodGet, "/api/v1/reports/rekonsiliasi?tahun=abc", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("tahun salah: %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodGet, "/api/v1/reports/rekonsiliasi.xlsx?tahun=2026", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Disposition"), "rekonsiliasi-pph-final-2026.xlsx") {
		t.Errorf("xlsx: %d %v", rec.Code, rec.Header())
	}
	if rec := do(srv, qa, http.MethodGet, "/api/v1/reports/rekonsiliasi.pdf?tahun=2026", ""); rec.Code != http.StatusOK || !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF")) {
		t.Errorf("pdf: %d", rec.Code)
	}

	// Dashboard: rekap PPh Final dari pembukuan Januari sampai bulan terpilih.
	var dash struct {
		TaxReport finance.UMKMTaxReport `json:"tax_report"`
	}
	_ = json.Unmarshal(do(srv, qa, http.MethodGet, "/api/v1/dashboard?period=2026-04", "").Body.Bytes(), &dash)
	if dash.TaxReport.TotalGross != 30_000_000 || dash.TaxReport.TotalTaxDue != 150_000 || len(dash.TaxReport.Months) != 4 || dash.TaxReport.EntityType != "Badan" {
		t.Errorf("dashboard: %+v", dash.TaxReport)
	}

	// Tutup Buku April: kurang bayar diperingatkan.
	var tb tutupBukuResp
	_ = json.Unmarshal(do(srv, qa, http.MethodGet, "/api/v1/tutup-buku?period=2026-04", "").Body.Bytes(), &tb)
	var cek *cekTutup
	for i := range tb.Cek {
		if tb.Cek[i].Kode == "pph_final" {
			cek = &tb.Cek[i]
		}
	}
	if cek == nil || cek.Status != "peringatan" || !strings.Contains(cek.Detail, "Kurang bayar Rp20.000") {
		t.Errorf("tutup buku: %+v", cek)
	}

	// Starter: rekonsiliasi & unduhan dikunci, Tutup Buku tanpa butir PPh Final.
	tn := tenantByEmail(t, srv, qa)
	tn.mu.Lock()
	tn.langganan.Orders = []order{{ID: "o", Paket: "starter-bulanan", Status: "lunas", Waktu: time.Now()}}
	tn.mu.Unlock()
	for _, u := range []string{"/api/v1/reports/rekonsiliasi", "/api/v1/reports/rekonsiliasi.xlsx", "/api/v1/reports/rekonsiliasi.pdf"} {
		if rec := do(srv, qa, http.MethodGet, u+"?tahun=2026", ""); rec.Code != http.StatusPaymentRequired {
			t.Errorf("starter %s: %d", u, rec.Code)
		}
	}
	tb = tutupBukuResp{}
	_ = json.Unmarshal(do(srv, qa, http.MethodGet, "/api/v1/tutup-buku?period=2026-04", "").Body.Bytes(), &tb)
	for _, c := range tb.Cek {
		if c.Kode == "pph_final" {
			t.Error("starter tidak mendapat butir PPh Final")
		}
	}
}
