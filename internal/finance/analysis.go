package finance

import (
	"fmt"
	"time"
)

// BusinessFacts merangkum SELURUH laporan satu periode jadi satu struktur datar.
//
// Dua pemakainya sengaja dibuat berbagi struktur yang sama: halaman Laporan AI menampilkannya
// sebagai rincian, dan model bahasa menerimanya sebagai bahan analisis. Dengan begitu mustahil
// model menilai angka yang berbeda dari yang dilihat pengguna di layar.
//
// Semua angka di sini sudah dihitung Go. Model tidak pernah diminta berhitung.
type BusinessFacts struct {
	Periode      string `json:"periode"`
	PeriodeLabel string `json:"periode_label"`
	NamaUsaha    string `json:"nama_usaha"`
	BentukUsaha  string `json:"bentuk_usaha,omitempty"`
	JenisUsaha   string `json:"jenis_usaha,omitempty"`

	LabaRugi   FactsLabaRugi `json:"laba_rugi"`
	ArusKas    FactsArusKas  `json:"arus_kas"`
	Neraca     FactsNeraca   `json:"neraca"`
	Aset       FactsAset     `json:"aset"`
	Pajak      FactsPajak    `json:"pajak"`
	Tagihan    FactsTagihan  `json:"tagihan"`
	Operasi    FactsOperasi  `json:"operasi"`
	Tren       []FactsTren   `json:"tren"`
	Peringatan []string      `json:"peringatan,omitempty"`
}

type FactsLabaRugi struct {
	Omzet            int64 `json:"omzet"`
	HPP              int64 `json:"hpp"`
	LabaKotor        int64 `json:"laba_kotor"`
	BebanOperasional int64 `json:"beban_operasional"`
	Penyusutan       int64 `json:"penyusutan"`
	Amortisasi       int64 `json:"amortisasi"`
	TotalBiaya       int64 `json:"total_biaya"`
	LabaBersih       int64 `json:"laba_bersih"`
	MarginBps        int   `json:"margin_bps"`
	// Perbandingan ke periode sebelumnya; nil bila tidak ada data pembanding.
	OmzetSebelumnya *int64 `json:"omzet_sebelumnya,omitempty"`
	LabaSebelumnya  *int64 `json:"laba_sebelumnya,omitempty"`
}

type FactsArusKas struct {
	SaldoAwal          int64 `json:"saldo_awal"`
	KasMasuk           int64 `json:"kas_masuk"`
	KasKeluar          int64 `json:"kas_keluar"`
	KasBersih          int64 `json:"kas_bersih"`
	SaldoAkhir         int64 `json:"saldo_akhir"`
	OperasiLangsung    int64 `json:"operasi_langsung"`
	OperasiTakLangsung int64 `json:"operasi_tidak_langsung"`
	Investasi          int64 `json:"investasi"`
	Pendanaan          int64 `json:"pendanaan"`
}

type FactsNeraca struct {
	TotalAktiva int64 `json:"total_aktiva"`
	TotalPasiva int64 `json:"total_pasiva"`
	Kas         int64 `json:"kas"`
	Piutang     int64 `json:"piutang"`
	Persediaan  int64 `json:"persediaan"`
	Utang       int64 `json:"utang"`
	Ekuitas     int64 `json:"ekuitas"`
	Seimbang    bool  `json:"seimbang"`
}

type FactsAset struct {
	Jumlah             int   `json:"jumlah"`
	HargaPerolehan     int64 `json:"harga_perolehan"`
	Akumulasi          int64 `json:"akumulasi"`
	NilaiBuku          int64 `json:"nilai_buku"`
	BebanBulanIni      int64 `json:"beban_bulan_ini"`
	Draft              int   `json:"draft"`
	PemetaanBelumDicek int   `json:"pemetaan_belum_dicek"`
}

type FactsPajak struct {
	BentukBadan     string `json:"bentuk_badan"`
	OmzetKumulatif  int64  `json:"omzet_kumulatif"`
	BatasBebasPajak int64  `json:"batas_bebas_pajak"`
	DasarKenaPajak  int64  `json:"dasar_kena_pajak"`
	PajakTerutang   int64  `json:"pajak_terutang"`
	SisaBatasBebas  int64  `json:"sisa_batas_bebas"`
}

type FactsTagihan struct {
	PiutangTotal        int64 `json:"piutang_total"`
	UtangTotal          int64 `json:"utang_total"`
	PiutangLewatTempo   int64 `json:"piutang_lewat_tempo"`
	JumlahFakturTelat   int   `json:"jumlah_faktur_telat"`
	InvoiceBelumDibayar int64 `json:"invoice_belum_dibayar"`
}

type FactsOperasi struct {
	JumlahTransaksi int           `json:"jumlah_transaksi"`
	BiayaAplikasi   int64         `json:"biaya_aplikasi"`
	Channel         []ChannelPerf `json:"channel,omitempty"`
	ProdukTeratas   []ProductPerf `json:"produk_teratas,omitempty"`
	StokMenipis     []string      `json:"stok_menipis,omitempty"`
	JumlahKaryawan  int           `json:"jumlah_karyawan"`
	BebanGaji       int64         `json:"beban_gaji_bulanan"`
}

type FactsTren struct {
	Bulan      string `json:"bulan"`
	Omzet      int64  `json:"omzet"`
	LabaBersih int64  `json:"laba_bersih"`
	SaldoKas   int64  `json:"saldo_kas"`
}

// CollectFacts menyusun seluruh angka laporan satu periode. Sumbernya laporan yang sudah
// dihitung (rep) ditambah snapshot dashboard untuk hal yang belum diturunkan dari transaksi
// (channel, produk, stok, pajak).
func CollectFacts(
	rep PeriodReport, snap DashboardSnapshot, p BusinessProfile,
	invoices []Invoice, karyawan []Karyawan, riwayat []PeriodSummary,
) BusinessFacts {
	is, cf, bs, as := rep.Income, rep.CashFlow, rep.BalanceSheet, rep.Assets

	f := BusinessFacts{
		Periode:      rep.Period,
		PeriodeLabel: snap.PeriodLabel,
		NamaUsaha:    cmpOr(p.NamaUsaha, snap.Tenant.Name),
		BentukUsaha:  p.BentukUsaha,
		JenisUsaha:   p.JenisUsaha,
		LabaRugi: FactsLabaRugi{
			Omzet: is.Omzet, HPP: is.COGS, LabaKotor: is.GrossProfit,
			BebanOperasional: is.Opex,
			Penyusutan:       is.Depreciation, Amortisasi: is.Amortization,
			TotalBiaya: is.COGS + is.TotalOpex, LabaBersih: is.NetProfit,
			MarginBps: is.MarginBps,
		},
		ArusKas: FactsArusKas{
			SaldoAwal: cf.OpeningBalance, KasMasuk: cf.CashIn, KasKeluar: cf.CashOut,
			KasBersih: cf.NetCash, SaldoAkhir: cf.ClosingBalance,
			OperasiLangsung: cf.OperasiLangsung.Net, OperasiTakLangsung: cf.OperasiTidakLangsung.Net,
			Investasi: cf.Investasi.Net, Pendanaan: cf.Pendanaan.Net,
		},
		Neraca: FactsNeraca{
			TotalAktiva: bs.Assets.Total, TotalPasiva: bs.Liabilities.Total + bs.Equity.Total,
			Kas: bs.Assets.Cash, Piutang: bs.Assets.Receivables, Persediaan: bs.Assets.Inventory,
			Utang: bs.Liabilities.Total, Ekuitas: bs.Equity.Total, Seimbang: bs.IsBalanced,
		},
		Aset: FactsAset{
			Jumlah: len(as.Assets), HargaPerolehan: as.TotalHargaPerolehan,
			Akumulasi: as.TotalAkumulasi, NilaiBuku: as.TotalNilaiBuku,
			BebanBulanIni: as.BebanTotal, Draft: as.DraftCount,
			PemetaanBelumDicek: as.PemetaanBelumDicek,
		},
		Pajak: FactsPajak{
			BentukBadan: snap.TaxReport.EntityType, OmzetKumulatif: snap.TaxReport.TotalGross,
			BatasBebasPajak: snap.TaxReport.YearlyExemption, DasarKenaPajak: snap.TaxReport.TotalTaxable,
			PajakTerutang:  snap.TaxReport.TotalTaxDue,
			SisaBatasBebas: maxI64(snap.TaxReport.YearlyExemption-snap.TaxReport.TotalGross, 0),
		},
		Operasi: FactsOperasi{
			JumlahTransaksi: snap.Tenant.TotalTransactions, BiayaAplikasi: snap.TotalAppFees,
			Channel: snap.Channels, ProdukTeratas: topN(snap.TopProducts, 5),
			JumlahKaryawan: len(karyawan),
		},
	}

	for _, k := range karyawan {
		if k.Aktif {
			f.Operasi.BebanGaji += k.PenghasilanBulanan()
		}
	}
	for _, it := range snap.Inventory {
		if it.Status == "Stok Menipis" || it.Status == "Habis" {
			f.Operasi.StokMenipis = append(f.Operasi.StokMenipis, it.Name+" ("+it.Stock+")")
		}
	}
	for _, o := range snap.Obligations {
		switch o.Kind {
		case "RECEIVABLE":
			f.Tagihan.PiutangTotal += o.Amount
		case "PAYABLE":
			f.Tagihan.UtangTotal += o.Amount
		}
	}
	inv := SummarizeInvoices(invoices, time.Now())
	f.Tagihan.PiutangLewatTempo = inv.Terlambat
	f.Tagihan.JumlahFakturTelat = inv.JumlahTerlambat
	f.Tagihan.InvoiceBelumDibayar = inv.BelumDibayar

	for _, h := range riwayat {
		f.Tren = append(f.Tren, FactsTren{Bulan: h.Month, Omzet: h.Omzet, LabaBersih: h.LabaBersih, SaldoKas: h.EndingCash})
	}
	// Pembanding periode sebelumnya diambil dari riwayat, bukan dihitung ulang, supaya sama
	// dengan grafik tren di Dashboard.
	if n := len(riwayat); n >= 2 {
		prev := riwayat[n-2]
		f.LabaRugi.OmzetSebelumnya = &prev.Omzet
		f.LabaRugi.LabaSebelumnya = &prev.LabaBersih
	}

	f.Peringatan = kumpulkanPeringatan(f, snap)
	return f
}

// kumpulkanPeringatan menandai kondisi yang objektif bermasalah. Ini bukan "analisis" — hanya
// fakta yang pantas disorot, supaya model (atau pembaca) tidak melewatkannya.
func kumpulkanPeringatan(f BusinessFacts, snap DashboardSnapshot) []string {
	var out []string
	if !f.Neraca.Seimbang {
		out = append(out, "Neraca tidak seimbang: aktiva dan pasiva berbeda. Periksa data sebelum memakai laporan ini.")
	}
	if f.LabaRugi.LabaBersih < 0 {
		out = append(out, fmt.Sprintf("Rugi bersih %s pada periode ini.", formatRpID(-f.LabaRugi.LabaBersih)))
	}
	if f.ArusKas.SaldoAkhir < 0 {
		out = append(out, "Saldo kas akhir negatif.")
	}
	if f.ArusKas.KasBersih < 0 && f.LabaRugi.LabaBersih > 0 {
		out = append(out, "Laba positif tapi kas berkurang; periksa piutang, persediaan, dan cicilan.")
	}
	if f.Tagihan.JumlahFakturTelat > 0 {
		out = append(out, fmt.Sprintf("%d faktur lewat jatuh tempo senilai %s.",
			f.Tagihan.JumlahFakturTelat, formatRpID(f.Tagihan.PiutangLewatTempo)))
	}
	if f.Aset.Draft > 0 {
		out = append(out, fmt.Sprintf("%d aset masih draft; kelompok dan metode penyusutannya belum dilengkapi.", f.Aset.Draft))
	}
	if f.Aset.PemetaanBelumDicek > 0 {
		out = append(out, fmt.Sprintf("%d jenis harta belum diverifikasi ke Lampiran PMK 72/2023; jangan dipakai untuk SPT sebelum dicek tim pajak.", f.Aset.PemetaanBelumDicek))
	}
	if len(f.Operasi.StokMenipis) > 0 {
		out = append(out, "Stok menipis: "+joinMax(f.Operasi.StokMenipis, 3))
	}
	if f.Pajak.SisaBatasBebas == 0 && f.Pajak.PajakTerutang > 0 {
		out = append(out, fmt.Sprintf("Batas omzet bebas pajak sudah terlampaui; PPh final terutang %s.", formatRpID(f.Pajak.PajakTerutang)))
	}
	if n := len(snap.CashWarnings); n > 0 {
		w := snap.CashWarnings[0]
		out = append(out, fmt.Sprintf("Kas diperkirakan minus %s pada %s (%d hari bermasalah).",
			formatRpID(-w.Saldo), w.Tanggal, n))
	}
	return out
}

func topN[T any](list []T, n int) []T {
	if len(list) > n {
		return list[:n]
	}
	return list
}

func joinMax(list []string, n int) string {
	s := ""
	for i, v := range list {
		if i >= n {
			return s + fmt.Sprintf(", dan %d lainnya", len(list)-n)
		}
		if i > 0 {
			s += ", "
		}
		s += v
	}
	return s
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func maxI64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// formatRpID: 18450000 → "Rp 18.450.000".
func formatRpID(n int64) string { return "Rp " + rupiah(n) }
