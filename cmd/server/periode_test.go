package main

import (
	"testing"

	"pajak/internal/finance"
)

func TestPeriode_DariDataSampaiBulanBerjalan(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	qa := register(t, srv, registerBody)
	aktifkan(t, srv, qa)
	sekarang := finance.BulanBerjalan()
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"`+sekarang+`-02","jenis":"Penjualan","nominal":1500000,"lunas":true}`)

	p := getJSON[[]finance.PilihanPeriode](t, srv, qa, "/api/v1/periods")
	if len(p) == 0 || p[0].ID != sekarang || !p[0].Bawaan {
		t.Fatalf("periode %+v, mau dimulai bulan berjalan %s", p, sekarang)
	}

	d := getJSON[finance.DashboardSnapshot](t, srv, qa, "/api/v1/dashboard?period="+sekarang)
	if d.Period != sekarang || d.PeriodLabel != finance.LabelPeriode(sekarang) {
		t.Errorf("dashboard pindah periode: %s %s", d.Period, d.PeriodLabel)
	}
	is := getJSON[finance.IncomeStatement](t, srv, qa, "/api/v1/reports/financial?period="+sekarang)
	if is.Omzet != 1_500_000 {
		t.Errorf("laporan bulan berjalan omzet %d", is.Omzet)
	}
}

func TestPeriode_DemoTetapPunyaPeriodeContoh(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	p := getJSON[[]finance.PilihanPeriode](t, srv, owner, "/api/v1/periods")
	ada := map[string]bool{}
	bawaan := ""
	for _, x := range p {
		ada[x.ID] = true
		if x.Bawaan {
			bawaan = x.ID
		}
	}
	if !ada["2026-01"] || !ada["2026-08"] || !ada[finance.BulanBerjalan()] || bawaan != "2026-08" {
		t.Errorf("demo: %+v (bawaan %s)", p, bawaan)
	}
}
