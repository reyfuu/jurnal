package finance_test

import (
	"testing"

	"pajak/internal/finance"
)

func seedHistory(period string) []finance.PeriodSummary {
	return finance.TrendHistory(finance.SeedTransactions(), finance.SeedAssets(), period, finance.SeedOpening)
}

// Inti perbaikan: grafik tren dan KPI harus memakai angka yang sama. Sebelumnya tren memakai
// angka seed sementara KPI dihitung ulang, sehingga penyusutan hanya muncul di KPI.
func TestApplyTrend_KonsistenDenganKPI(t *testing.T) {
	const period = "2026-08"
	snap := finance.SeedKedaiKopiSenja(period)
	rep := finance.BuildPeriodReport(finance.SeedTransactions(), finance.SeedAssets(), period, finance.SeedOpening(period))
	txs := finance.ForPeriod(finance.SeedTransactions(), period)

	finance.ApplyKPIs(&snap, rep.Income, rep.CashFlow, txs, len(txs))
	finance.ApplyTrend(&snap, seedHistory(period))

	last := snap.MonthlyTrend11M[len(snap.MonthlyTrend11M)-1]
	if last.Month != "Agu 26" {
		t.Errorf("titik terakhir harus periode yang dilihat: got %q", last.Month)
	}
	if last.LabaBersih != rep.Income.NetProfit {
		t.Errorf("laba di grafik %d != laba hasil hitung %d", last.LabaBersih, rep.Income.NetProfit)
	}
	if last.LabaBersih != snap.KPIs.LabaBersih.Value {
		t.Errorf("laba di grafik %d != KPI laba %d", last.LabaBersih, snap.KPIs.LabaBersih.Value)
	}
	if last.Omzet != snap.KPIs.Omzet.Value {
		t.Errorf("omzet di grafik %d != KPI omzet %d", last.Omzet, snap.KPIs.Omzet.Value)
	}
	if last.EndingCash != snap.KPIs.SaldoKas.Value {
		t.Errorf("saldo kas di grafik %d != KPI saldo kas %d", last.EndingCash, snap.KPIs.SaldoKas.Value)
	}
	// Penyusutan harus ikut terlihat di grafik: laba Agustus bukan lagi angka seed Rp25 jt.
	if last.LabaBersih != 18_450_000 {
		t.Errorf("laba di grafik harus sudah dikurangi penyusutan: want 18_450_000, got %d", last.LabaBersih)
	}
	// Omzet − total biaya harus sama dengan laba (seed lama tidak memenuhi ini).
	if last.Omzet-last.TotalBiaya != last.LabaBersih {
		t.Errorf("omzet %d − biaya %d != laba %d", last.Omzet, last.TotalBiaya, last.LabaBersih)
	}
}

// Sparkline tiap KPI harus sepanjang riwayat dan berakhir di nilai periode itu.
func TestApplyTrend_SparklineIkutHasilHitung(t *testing.T) {
	const period = "2026-08"
	snap := finance.SeedKedaiKopiSenja(period)
	history := seedHistory(period)
	finance.ApplyTrend(&snap, history)

	k := snap.KPIs
	for _, c := range []struct {
		name string
		s    []float64
	}{
		{"omzet", k.Omzet.Sparkline}, {"laba", k.LabaBersih.Sparkline},
		{"kas", k.SaldoKas.Sparkline}, {"margin", k.MarginLaba.Sparkline},
	} {
		if len(c.s) != len(history) {
			t.Errorf("sparkline %s: want %d titik, got %d", c.name, len(history), len(c.s))
		}
	}
	if got := k.LabaBersih.Sparkline[len(k.LabaBersih.Sparkline)-1]; got != 18.45 {
		t.Errorf("titik terakhir sparkline laba (juta): want 18.45, got %v", got)
	}
	if got := k.SaldoKas.Sparkline[len(k.SaldoKas.Sparkline)-1]; got != 97 {
		t.Errorf("titik terakhir sparkline kas (juta): want 97, got %v", got)
	}
}

// Badge perbandingan dihitung dari periode sebelumnya, bukan teks tetap seperti "Stabil".
func TestApplyTrend_BadgeDariPeriodeSebelumnya(t *testing.T) {
	snap := finance.SeedKedaiKopiSenja("2026-08")
	finance.ApplyTrend(&snap, []finance.PeriodSummary{
		{Period: "2026-07", Month: "Jul 26", Omzet: 100_000_000, LabaBersih: 20_000_000, EndingCash: 50_000_000},
		{Period: "2026-08", Month: "Agu 26", Omzet: 110_000_000, LabaBersih: 11_000_000, EndingCash: 50_000_000},
	})

	k := snap.KPIs
	if k.Omzet.BadgeText != "▲ 10% vs bulan lalu" || k.Omzet.Trend != "up" {
		t.Errorf("badge omzet: %q (%s)", k.Omzet.BadgeText, k.Omzet.Trend)
	}
	if k.LabaBersih.BadgeText != "▼ 45% vs bulan lalu" || k.LabaBersih.Trend != "down" {
		t.Errorf("badge laba: %q (%s)", k.LabaBersih.BadgeText, k.LabaBersih.Trend)
	}
	if k.SaldoKas.BadgeText != "Stabil vs bulan lalu" {
		t.Errorf("badge kas: %q", k.SaldoKas.BadgeText)
	}
	// Margin 20% → 10% = turun 10 poin.
	if k.MarginLaba.BadgeText != "-10,0 poin vs bulan lalu" || k.MarginLaba.Trend != "down" {
		t.Errorf("badge margin: %q (%s)", k.MarginLaba.BadgeText, k.MarginLaba.Trend)
	}
}

// Riwayat hanya sampai periode yang dilihat, supaya grafik tidak memuat masa depan.
func TestTrendHistory_BerhentiDiPeriodeYangDilihat(t *testing.T) {
	h := seedHistory("2026-03")
	if len(h) != 3 {
		t.Fatalf("Januari s/d Maret: want 3 titik, got %d", len(h))
	}
	if h[0].Period != "2026-01" || h[2].Period != "2026-03" {
		t.Errorf("urutan kronologis rusak: %s .. %s", h[0].Period, h[2].Period)
	}
	// Aset seed diperoleh Februari, jadi Januari belum kena penyusutan.
	jan := finance.BuildPeriodReport(finance.SeedTransactions(), finance.SeedAssets(), "2026-01", finance.SeedOpening("2026-01"))
	if jan.Income.Depreciation != 0 {
		t.Errorf("Januari belum punya aset: penyusutan %d", jan.Income.Depreciation)
	}
	if h[0].LabaBersih != jan.Income.NetProfit {
		t.Errorf("titik Januari %d != laba Januari %d", h[0].LabaBersih, jan.Income.NetProfit)
	}
}

func TestApplyTrend_RiwayatKosongTidakMengubahApaPun(t *testing.T) {
	snap := finance.SeedKedaiKopiSenja("2026-08")
	before := len(snap.MonthlyTrend11M)
	finance.ApplyTrend(&snap, nil)
	if len(snap.MonthlyTrend11M) != before {
		t.Errorf("riwayat kosong tidak boleh menghapus grafik: %d → %d", before, len(snap.MonthlyTrend11M))
	}
}
