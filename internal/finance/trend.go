package finance

import (
	"fmt"
	"slices"
	"strings"
)

// Grafik tren & sparkline dashboard dihitung dari laporan tiap periode, bukan dari angka seed,
// supaya tidak bertentangan dengan KPI yang juga dihitung ulang. Sebelumnya tren memakai angka
// seed sementara KPI dihitung dari transaksi, jadi penyusutan hanya terlihat di KPI.

// PeriodSummary adalah satu titik grafik tren, hasil hitung periode itu.
type PeriodSummary struct {
	Period     string `json:"period"`
	Month      string `json:"month"` // label pendek, mis. "Agu 26"
	Omzet      int64  `json:"omzet"`
	TotalBiaya int64  `json:"total_biaya"` // potongan channel + HPP + seluruh beban operasional
	LabaBersih int64  `json:"laba_bersih"`
	EndingCash int64  `json:"ending_cash"`
}

// shortMonth: "2026-08" → "Agu 26".
func shortMonth(period string) string {
	y, m, ok := parseYM(period)
	if !ok {
		return period
	}
	return fmt.Sprintf("%s %02d", bulanSingkat[m-1], y%100)
}

// SummarizePeriod meringkas satu laporan periode untuk grafik tren.
func SummarizePeriod(rep PeriodReport) PeriodSummary {
	is := rep.Income
	return PeriodSummary{
		Period:     rep.Period,
		Month:      shortMonth(rep.Period),
		Omzet:      is.Omzet,
		TotalBiaya: is.ChannelFees + is.COGS + is.TotalOpex,
		LabaBersih: is.NetProfit,
		EndingCash: rep.CashFlow.ClosingBalance,
	}
}

// TrendHistory menghitung ringkasan seluruh periode sampai period (inklusif), urut kronologis.
// opening menyediakan saldo awal tiap periode agar fungsi ini tidak perlu tahu asal datanya.
//
// ponytail: menghitung ulang semua periode setiap request dashboard. Untuk 8 periode dan
// ribuan transaksi masih sub-milidetik; pasang cache per tenant bila periode sudah bertahun-tahun.
// laporan menghitung laporan satu periode (laporan berjalan usaha itu).
func TrendHistory(txs []Transaction, period string, laporan func(string) PeriodReport) []PeriodSummary {
	// Sampai 12 bulan terakhir, mulai dari bulan data paling awal (tanpa bulan kosong di depan).
	ids := []string{period}
	data := bulanData(txs)
	for i := 1; i < 12; i++ {
		p := geserBulan(period, -i)
		if len(data) == 0 || p < data[0] {
			break
		}
		ids = append(ids, p)
	}
	slices.Sort(ids)
	out := make([]PeriodSummary, 0, len(ids))
	for _, id := range ids {
		out = append(out, SummarizePeriod(laporan(id)))
	}
	return out
}

func jt(rp int64) float64 { return float64(rp) / 1_000_000 }

// deltaBadge: "▲ 4% vs bulan lalu" / "▼ 12% vs bulan lalu" / "Stabil vs bulan lalu".
func deltaBadge(now, before int64) (text, trend string) {
	if before == 0 {
		if now == 0 {
			return "Stabil vs bulan lalu", "flat"
		}
		return "Baru bulan ini", "up"
	}
	pct := (now - before) * 100 / abs64(before)
	switch {
	case pct > 0:
		return fmt.Sprintf("▲ %d%% vs bulan lalu", pct), "up"
	case pct < 0:
		return fmt.Sprintf("▼ %d%% vs bulan lalu", -pct), "down"
	}
	return "Stabil vs bulan lalu", "flat"
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// marginBps menghitung margin dalam basis poin tanpa bagi nol.
func marginBps(laba, omzet int64) int {
	if omzet <= 0 {
		return 0
	}
	return int(laba * 10_000 / omzet)
}

// ApplyTrend menimpa grafik tren, sparkline, dan badge perbandingan dengan hasil hitung.
// history harus urut kronologis dan berakhir di periode yang sedang dilihat; paling banyak
// 11 titik terakhir yang dipakai (sesuai lebar grafik di dashboard).
func ApplyTrend(snap *DashboardSnapshot, history []PeriodSummary) {
	if len(history) == 0 {
		return
	}
	if len(history) > 11 {
		history = history[len(history)-11:]
	}
	trend := make([]MonthlyTrendPoint, 0, len(history))
	omzet := make([]float64, 0, len(history))
	laba := make([]float64, 0, len(history))
	kas := make([]float64, 0, len(history))
	margin := make([]float64, 0, len(history))
	for _, h := range history {
		trend = append(trend, MonthlyTrendPoint{
			Month: h.Month, Omzet: h.Omzet, TotalBiaya: h.TotalBiaya,
			LabaBersih: h.LabaBersih, EndingCash: h.EndingCash,
		})
		omzet = append(omzet, jt(h.Omzet))
		laba = append(laba, jt(h.LabaBersih))
		kas = append(kas, jt(h.EndingCash))
		margin = append(margin, float64(marginBps(h.LabaBersih, h.Omzet))/100)
	}
	snap.MonthlyTrend11M = trend

	k := &snap.KPIs
	k.Omzet.Sparkline, k.LabaBersih.Sparkline = omzet, laba
	k.SaldoKas.Sparkline, k.MarginLaba.Sparkline = kas, margin
	if len(history) < 2 {
		return
	}
	now, before := history[len(history)-1], history[len(history)-2]
	k.Omzet.BadgeText, k.Omzet.Trend = deltaBadge(now.Omzet, before.Omzet)
	k.LabaBersih.BadgeText, k.LabaBersih.Trend = deltaBadge(now.LabaBersih, before.LabaBersih)
	k.SaldoKas.BadgeText, k.SaldoKas.Trend = deltaBadge(now.EndingCash, before.EndingCash)

	// Margin dibandingkan dalam poin persen, bukan persen dari persen.
	poin := float64(marginBps(now.LabaBersih, now.Omzet)-marginBps(before.LabaBersih, before.Omzet)) / 100
	k.MarginLaba.Trend = "up"
	if poin < 0 {
		k.MarginLaba.Trend = "down"
	}
	k.MarginLaba.BadgeText = strings.ReplaceAll(fmt.Sprintf("%+.1f poin vs bulan lalu", poin), ".", ",")
}
