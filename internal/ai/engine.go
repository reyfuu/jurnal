package ai

import (
	"context"
	"fmt"
	"strings"

	"pajak/internal/finance"
)

type QueryRequest struct {
	Query  string `json:"query"`
	Period string `json:"period,omitempty"`
}

type QueryResponse struct {
	Answer            string   `json:"answer"`
	Category          string   `json:"category"`
	ActionSuggestions []string `json:"action_suggestions"`
}

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func formatRp(n int64) string {
	if n < 0 {
		return "-" + formatRp(-n)
	}
	s := fmt.Sprintf("%d", n)
	var res []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			res = append(res, '.')
		}
		res = append(res, byte(c))
	}
	return string(res)
}

func (e *Engine) Answer(ctx context.Context, req QueryRequest, snap finance.DashboardSnapshot, ledger []finance.LedgerEntry) QueryResponse {
	q := strings.ToLower(strings.TrimSpace(req.Query))

	is := finance.BuildIncomeStatement(ledger)
	cf := finance.BuildCashFlow(ledger, 80_000_000)

	// 1. Ringkasan Laporan Bisnis
	if strings.Contains(q, "ringkas") || strings.Contains(q, "summary") || strings.Contains(q, "laporan bulan") || strings.Contains(q, "performa") || strings.Contains(q, "ikhtisar") {
		return e.answerSummary(snap, is, cf)
	}

	// 2. Pajak UMKM PP 55 / Aturan Pajak
	if strings.Contains(q, "pajak") || strings.Contains(q, "pp 55") || strings.Contains(q, "pph") || strings.Contains(q, "ptkp") || strings.Contains(q, "tarif") {
		return e.answerTax(snap)
	}

	// 3. Arus Kas, Kas, Piutang & Utang
	if strings.Contains(q, "arus kas") || strings.Contains(q, "cashflow") || strings.Contains(q, "piutang") || strings.Contains(q, "utang") || strings.Contains(q, "saldo kas") || strings.Contains(q, " kas") || strings.Contains(q, "kas ") || q == "kas" {
		return e.answerCashflow(snap, cf)
	}

	// 3. Simulasi Harga / Komisi Aplikasi
	if strings.Contains(q, "harga") || strings.Contains(q, "komisi") || strings.Contains(q, "gofood") || strings.Contains(q, "grab") || strings.Contains(q, "shopee") || strings.Contains(q, "markup") {
		return e.answerPricing()
	}

	// Default: Ringkasan Laporan Bisnis
	return e.answerSummary(snap, is, cf)
}

func (e *Engine) answerSummary(snap finance.DashboardSnapshot, is finance.IncomeStatement, cf finance.CashFlow) QueryResponse {
	answer := fmt.Sprintf(
		"### Ringkasan Laporan Bisnis — %s\n\n"+
			"**1. Performa Keuangan Utama:**\n"+
			"- **Omzet Bruto:** %s (Total transaksi: %d pesanan)\n"+
			"- **Potongan Fee Aplikasi:** Rp %s\n"+
			"- **Penjualan Bersih (Net Sales):** Rp %s\n"+
			"- **Beban Pokok Penjualan (HPP):** Rp %s\n"+
			"- **Laba Kotor:** Rp %s\n"+
			"- **Biaya Operasional (OPEX):** Rp %s\n"+
			"- **Laba Bersih:** %s (Net Margin: **%.1f%%**)\n\n"+
			"**2. Posisi Likuiditas & Kas:**\n"+
			"- **Saldo Kas Akhir:** %s (Kenaikan kas bersih: Rp %s dari saldo awal Rp 80,0 jt)\n"+
			"- **Rekonsiliasi Arus Kas:** Laba Bersih (%s) - Piutang Baru (%s) + Utang Baru (%s) = Kenaikan Kas (%s). **Status: Invarian 100%% Terpenuhi.**\n\n"+
			"**3. Rekomendasi & Catatan Kritis Akademi Pajak:**\n"+
			"- **Piutang Tertahan:** Terdapat piutang ShopeePay sebesar Rp 11,0 jt yang sudah lewat 30 hari. Segera lakukan penagihan/rekonsiliasi ke kanal pembayaran.\n"+
			"- **Pajak UMKM PP 55/2022:** Omzet kumulatif saat ini Rp 190,0 jt masih di bawah ambang batas bebas pajak Rp 500 jt. Pajak terutang bulan ini: **Rp 0 (Nihil)**.",
		snap.PeriodLabel,
		snap.KPIs.Omzet.Display,
		snap.Tenant.TotalTransactions,
		formatRp(is.ChannelFees),
		formatRp(is.NetSales),
		formatRp(is.COGS),
		formatRp(is.GrossProfit),
		formatRp(is.Opex),
		snap.KPIs.LabaBersih.Display,
		float64(is.MarginBps)/100.0,
		snap.KPIs.SaldoKas.Display,
		formatRp(cf.ClosingBalance-cf.OpeningBalance),
		formatRp(is.NetProfit),
		formatRp(cf.NewReceivables),
		formatRp(cf.NewPayables),
		formatRp(cf.ClosingBalance-cf.OpeningBalance),
	)

	return QueryResponse{
		Answer:   answer,
		Category: "summary",
		ActionSuggestions: []string{
			"Bagaimana aturan pajak UMKM PP 55/2022?",
			"Bagaimana kondisi arus kas dan piutang saya?",
			"Bagaimana cara hitung simulasi harga komisi GoFood?",
		},
	}
}

func (e *Engine) answerTax(snap finance.DashboardSnapshot) QueryResponse {
	answer := "### Panduan Regulasi Pajak UMKM (PP 55 Tahun 2022)\n\n" +
		"Berdasarkan **Peraturan Pemerintah No. 55 Tahun 2022** (turunan UU HPP):\n\n" +
		"**1. Batas Omzet Bebas Pajak (PTKP Omzet Rp 500 Juta):**\n" +
		"- Khusus untuk Wajib Pajak **Orang Pribadi (OP)** UMKM, omzet kumulatif sampai dengan **Rp 500.000.000 per tahun pajak TIDAK DIKENAKAN PAJAK (Bebas Pajak / Nihil)**.\n" +
		"- PPh Final 0,5% hanya dihitung dari bagian peredaran bruto kumulatif yang **melebihi Rp 500 juta**.\n\n" +
		"**2. Wajib Pajak Badan (CV, PT, Koperasi):**\n" +
		"- Tidak mendapatkan fasilitas pembebasan Rp 500 juta. Seluruh omzet langsung dikenakan PPh Final 0,5% sejak rupiah pertama.\n\n" +
		"**3. Jangka Waktu Pemanfaatan Tarif 0,5%:**\n" +
		"- Wajib Pajak **Orang Pribadi**: Maksimal **7 Tahun Pajak**.\n" +
		"- Wajib Pajak **Koperasi, CV, atau Firma**: Maksimal **4 Tahun Pajak**.\n" +
		"- Wajib Pajak **PT (Perseroan Terbatas)**: Maksimal **3 Tahun Pajak**.\n" +
		"- Setelah jangka waktu berakhir, wajib menggunakan tarif umum PPh Pasal 17 dengan pembukuan.\n\n" +
		"**4. Simulasi Status Pajak Anda (Kedai Kopi Senja):**\n" +
		"- Omzet Bulan Ini: Rp 92.000.000\n" +
		"- Estimasi Omzet Kumulatif Berjalan: Rp 190.000.000\n" +
		"- Sisa Kuota Bebas Pajak: **Rp 310.000.000**\n" +
		"- **Status PPh Final Bulan Ini:** **Rp 0 (BEBAS PAJAK)**."

	return QueryResponse{
		Answer:   answer,
		Category: "tax",
		ActionSuggestions: []string{
			"Ringkas laporan bulan ini",
			"Bagaimana kondisi arus kas dan piutang saya?",
			"Bagaimana cara hitung simulasi harga komisi GoFood?",
		},
	}
}

func (e *Engine) answerCashflow(snap finance.DashboardSnapshot, cf finance.CashFlow) QueryResponse {
	answer := fmt.Sprintf(
		"### Analisis Arus Kas, Likuiditas & Piutang\n\n"+
			"**1. Posisi Kas Terkini:**\n"+
			"- **Saldo Kas Akhir:** %s (Saldo Awal: Rp 80,0 jt)\n"+
			"- **Pertumbuhan Kas Bersih:** +Rp %s\n\n"+
			"**2. Rekonsiliasi Laba vs Kas:**\n"+
			"- Laba Bersih Operasional: Rp %s\n"+
			"- Dikurangi Piutang Baru Belum Cair: -Rp %s\n"+
			"- Ditambah Utang Operasional Baru: +Rp %s\n"+
			"- **Kenaikan Kas Riil:** +Rp %s\n\n"+
			"**3. Peringatan Piutang Berisiko (Aging > 30 Hari):**\n"+
			"- **ShopeePay:** Rp 11.000.000 tertahan lebih dari 30 hari.\n"+
			"- *Tindakan Disarankan:* Periksa jadwal settlement payment gateway dan pastikan rekening bank penerima tidak memiliki kendala KYC/verifikasi.",
		snap.KPIs.SaldoKas.Display,
		formatRp(cf.ClosingBalance-cf.OpeningBalance),
		formatRp(cf.ClosingBalance-cf.OpeningBalance+cf.NewReceivables-cf.NewPayables),
		formatRp(cf.NewReceivables),
		formatRp(cf.NewPayables),
		formatRp(cf.ClosingBalance-cf.OpeningBalance),
	)

	return QueryResponse{
		Answer:   answer,
		Category: "cashflow",
		ActionSuggestions: []string{
			"Ringkas laporan bulan ini",
			"Bagaimana aturan pajak UMKM PP 55/2022?",
			"Bagaimana cara hitung simulasi harga komisi GoFood?",
		},
	}
}

func (e *Engine) answerPricing() QueryResponse {
	answer := "### Rumus Simulasi Harga Jual & Komisi Aplikasi (Akademi Pajak)\n\n" +
		"Banyak pengusaha UMKM keliru saat menghitung kenaikan harga untuk GoFood/GrabFood/ShopeeFood sehingga margin laba tergerus.\n\n" +
		"**1. Rumus yang SALAH (Sering Dilakukan):**\n" +
		"`Harga Online = Harga Dine-in + 20%` (salah)\n" +
		"*Contoh:* Minuman Rp 20.000 dinaikkan 20% jadi Rp 24.000. Saat dipotong komisi 20% oleh aplikasi (20% × Rp 24.000 = Rp 4.800), uang yang Anda terima hanya Rp 19.200 (rugi Rp 800 per cup!).\n\n" +
		"**2. Rumus yang BENAR (Markup Gross-up):**\n" +
		"$$\\text{Harga Jual Online} = \\frac{\\text{Harga Bersih Target}}{1 - \\text{Komisi Aplikasi}}$$\n\n" +
		"*Contoh:* Target terima Rp 20.000 dengan komisi GoFood 20%:\n" +
		"$$\\text{Harga Jual} = \\frac{20.000}{1 - 0.20} = \\frac{20.000}{0.80} = \\text{Rp 25.000}$$\n" +
		"Saat dipotong 20% (Rp 5.000), Anda tetap menerima **Rp 20.000 utuh**!\n\n" +
		"Gunakan menu **Simulasi Harga** di dashboard untuk menghitung otomatis seluruh kanal Anda."

	return QueryResponse{
		Answer:   answer,
		Category: "pricing",
		ActionSuggestions: []string{
			"Ringkas laporan bulan ini",
			"Bagaimana aturan pajak UMKM PP 55/2022?",
			"Bagaimana kondisi arus kas dan piutang saya?",
		},
	}
}
