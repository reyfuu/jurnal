package finance

import (
	"bytes"
	"strings"
	"testing"
)

func contohLabaRugi() IncomeStatement {
	return IncomeStatement{
		Omzet: 100_000, ChannelFees: 10_000, NetSales: 90_000, COGS: 30_000, GrossProfit: 60_000,
		Opex: 20_000, Depreciation: 5_000, TotalOpex: 25_000, OtherIncome: 1_000, NetProfit: 36_000,
		OpexLines: []Line{{"Sewa", 15_000}, {"Listrik", 5_000}},
	}
}

func barisDengan(t *testing.T, b BagianLaporan, kunci string) BarisLaporan {
	t.Helper()
	for _, r := range b.Baris {
		if r.Kunci == kunci {
			return r
		}
	}
	t.Fatalf("baris %q tidak ada di %s", kunci, b.Kode)
	return BarisLaporan{}
}

func TestBagianLabaRugi(t *testing.T) {
	b := BagianLabaRugi(contohLabaRugi())
	if b.Kode != "laba-rugi" {
		t.Fatalf("kode = %q", b.Kode)
	}
	cek := map[string]int64{"omzet": 100_000, "potongan": -10_000, "penjualan_bersih": 90_000, "hpp": -30_000,
		"laba_kotor": 60_000, "beban:Sewa": -15_000, "beban:Listrik": -5_000, "penyusutan": -5_000,
		"total_beban": -25_000, "pendapatan_lain": 1_000, "laba_bersih": 36_000}
	for k, v := range cek {
		if r := barisDengan(t, b, k); r.Nilai != v || r.Agregat != AgregatJumlah {
			t.Errorf("%s = %d (%s), mau %d (jumlah)", k, r.Nilai, r.Agregat, v)
		}
	}
	if r := barisDengan(t, b, "laba_bersih"); r.Gaya != GayaTotal {
		t.Errorf("laba bersih gaya %q", r.Gaya)
	}
}

func TestBagianNeracaPosisiAkhir(t *testing.T) {
	bs := BalanceSheet{
		Assets:      BalanceSheetAssets{Cash: 50, Receivables: 10, Inventory: 5, FixedAssetsCost: 40, AccumDepreciation: 5, FixedAssets: 35, Total: 100},
		Liabilities: BalanceSheetLiabilities{Payables: 20, Loans: 10, Total: 30},
		Equity:      BalanceSheetEquity{OpeningCapital: 50, AddedCapital: 5, CurrentProfit: 15, Total: 70},
	}
	b := BagianNeraca(bs)
	if r := barisDengan(t, b, "total_aset"); r.Nilai != 100 {
		t.Errorf("total aset %d", r.Nilai)
	}
	if r := barisDengan(t, b, "total_pasiva"); r.Nilai != 100 {
		t.Errorf("total pasiva %d", r.Nilai)
	}
	if r := barisDengan(t, b, "akumulasi"); r.Nilai != -5 {
		t.Errorf("akumulasi %d", r.Nilai)
	}
	for _, r := range b.Baris {
		if r.Gaya != GayaJudul && r.Agregat != AgregatAkhir {
			t.Errorf("neraca %s agregat %q, mau akhir", r.Kunci, r.Agregat)
		}
	}
}

func TestBagianArusKasDuaMetode(t *testing.T) {
	cf := CashFlow{OpeningBalance: 1_000, NetCash: 300, ClosingBalance: 1_300,
		OperasiLangsung:      CashFlowSection{Lines: []CashFlowLine{{Label: "Penerimaan penjualan", Amount: 500, Detail: []Line{{"GoFood", 200}}}, {Label: "Penyusutan", NonKas: true}}, Net: 500},
		OperasiTidakLangsung: CashFlowSection{Lines: []CashFlowLine{{Label: "Laba bersih", Amount: 450}, {Label: "Piutang", Amount: 50}}, Net: 500},
		Investasi:            CashFlowSection{Lines: []CashFlowLine{{Label: "Beli alat", Amount: -200}}, Net: -200},
	}
	b := BagianArusKas(cf, false)
	if r := barisDengan(t, b, "saldo_awal"); r.Nilai != 1_000 || r.Agregat != AgregatAwal {
		t.Errorf("saldo awal %+v", r)
	}
	if r := barisDengan(t, b, "saldo_akhir"); r.Nilai != 1_300 || r.Agregat != AgregatAkhir {
		t.Errorf("saldo akhir %+v", r)
	}
	if r := barisDengan(t, b, "operasi:Penerimaan penjualan:GoFood"); r.Gaya != GayaSub || r.Nilai != 200 {
		t.Errorf("rincian %+v", r)
	}
	if r := barisDengan(t, b, "investasi_bersih"); r.Nilai != -200 {
		t.Errorf("investasi %d", r.Nilai)
	}
	for _, r := range b.Baris {
		if r.Kunci == "operasi:Penyusutan" {
			t.Error("baris non-kas ikut tercetak")
		}
	}
	tl := BagianArusKas(cf, true)
	barisDengan(t, tl, "operasi:Laba bersih")
	if !strings.Contains(tl.Judul, "Tidak Langsung") {
		t.Errorf("judul %q", tl.Judul)
	}
}

func TestSusunBanding(t *testing.T) {
	jul := contohLabaRugi()
	agu := contohLabaRugi()
	agu.OpexLines = []Line{{"Sewa", 15_000}, {"Gaji", 7_000}}
	agu.NetProfit = 29_000
	cfJul := CashFlow{OpeningBalance: 100, NetCash: 10, ClosingBalance: 110}
	cfAgu := CashFlow{OpeningBalance: 110, NetCash: 20, ClosingBalance: 130}

	b := SusunBanding([]PeriodeLaporan{
		{Periode: "2026-08", Bagian: []BagianLaporan{BagianLabaRugi(agu), BagianArusKas(cfAgu, false)}},
		{Periode: "2026-07", Bagian: []BagianLaporan{BagianLabaRugi(jul), BagianArusKas(cfJul, false)}},
	})
	if len(b.Kolom) != 2 || b.Kolom[0].Periode != "2026-07" || b.Kolom[0].Label != "Juli 2026" {
		t.Fatalf("kolom harus urut waktu: %+v", b.Kolom)
	}
	lr := b.Bagian[0]
	cari := func(bg BagianBanding, k string) BarisBanding {
		for _, r := range bg.Baris {
			if r.Kunci == k {
				return r
			}
		}
		t.Fatalf("baris %q tidak ada", k)
		return BarisBanding{}
	}
	if r := cari(lr, "laba_bersih"); r.Nilai[0] != 36_000 || r.Nilai[1] != 29_000 || r.Total != 65_000 {
		t.Errorf("laba bersih %+v", r)
	}
	// Baris yang hanya ada di satu periode tetap muncul; periode lain bernilai 0.
	if r := cari(lr, "beban:Gaji"); r.Nilai[0] != 0 || r.Nilai[1] != -7_000 || r.Total != -7_000 {
		t.Errorf("gaji %+v", r)
	}
	if r := cari(lr, "beban:Listrik"); r.Nilai[1] != 0 || r.Total != -5_000 {
		t.Errorf("listrik %+v", r)
	}
	// Urutan: Gaji muncul tepat setelah Sewa, sebelum Listrik/penyusutan tidak tertukar.
	var urut []string
	for _, r := range lr.Baris {
		if strings.HasPrefix(r.Kunci, "beban:") {
			urut = append(urut, r.Kunci)
		}
	}
	if strings.Join(urut, ",") != "beban:Sewa,beban:Listrik,beban:Gaji" && strings.Join(urut, ",") != "beban:Sewa,beban:Gaji,beban:Listrik" {
		t.Errorf("urutan beban %v", urut)
	}
	ak := b.Bagian[1]
	if r := cari(ak, "saldo_awal"); r.Total != 100 {
		t.Errorf("saldo awal gabungan = %d, mau saldo awal periode pertama 100", r.Total)
	}
	if r := cari(ak, "saldo_akhir"); r.Total != 130 {
		t.Errorf("saldo akhir gabungan = %d, mau periode terakhir 130", r.Total)
	}
	if r := cari(ak, "kenaikan_kas"); r.Total != 30 {
		t.Errorf("kenaikan kas gabungan %d", r.Total)
	}
}

func TestValidasiPeriodeBanding(t *testing.T) {
	p, err := ValidasiPeriodeBanding("2026-08, 2026-06,2026-07")
	if err != nil || strings.Join(p, ",") != "2026-06,2026-07,2026-08" {
		t.Fatalf("p=%v err=%v", p, err)
	}
	for _, salah := range []string{"", "2026-13", "2026-8", "2026-07,2026-07", "abc",
		"2025-01,2025-02,2025-03,2025-04,2025-05,2025-06,2025-07,2025-08,2025-09,2025-10,2025-11,2025-12,2026-01"} {
		if _, err := ValidasiPeriodeBanding(salah); err == nil {
			t.Errorf("%q harus ditolak", salah)
		}
	}
	if !PeriodeSah("2026-12") || PeriodeSah("2026-00") {
		t.Error("PeriodeSah salah")
	}
}

func TestRenderLaporanPDF(t *testing.T) {
	var buf bytes.Buffer
	bag := []BagianLaporan{BagianLabaRugi(contohLabaRugi()), BagianNeraca(BalanceSheet{})}
	if err := RenderLaporanPDF(&buf, "Kedai Senja", "Agustus 2026", bag, nil); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, mau := range []string{"%PDF-1.4", "Kedai Senja", "Laporan Laba Rugi", "Neraca", "Agustus 2026", "Laba bersih", "36.000", `\(10.000\)`} {
		if !strings.Contains(s, mau) {
			t.Errorf("PDF tidak memuat %q", mau)
		}
	}
	if n := strings.Count(s, "/Type /Page "); n != 2 {
		t.Errorf("tiap laporan satu halaman sendiri: %d halaman", n)
	}
}

func TestRenderLaporanPDFAsetDanHalamanPanjang(t *testing.T) {
	var aset AssetSummary
	for i := range 80 {
		aset.Assets = append(aset.Assets, AssetReport{Asset: Asset{Nama: "Mesin " + string(rune('A'+i%26)), TanggalPerolehan: "2026-01-05", HargaPerolehan: 1_000_000}, NilaiBuku: 900_000})
	}
	aset.TotalHargaPerolehan = 80_000_000
	var buf bytes.Buffer
	if err := RenderLaporanPDF(&buf, "X", "Agustus 2026", nil, &aset); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if strings.Count(s, "/Type /Page ") < 2 {
		t.Error("80 aset harus berlanjut ke halaman berikutnya")
	}
	for _, mau := range []string{"Register Aset", `\(lanjutan\)`, "80.000.000", "Halaman 2"} {
		if !strings.Contains(s, mau) {
			t.Errorf("PDF aset tidak memuat %q", mau)
		}
	}
}

func TestRenderBandingPDFLanskap(t *testing.T) {
	b := SusunBanding([]PeriodeLaporan{
		{Periode: "2026-07", Bagian: []BagianLaporan{BagianLabaRugi(contohLabaRugi())}},
		{Periode: "2026-08", Bagian: []BagianLaporan{BagianLabaRugi(contohLabaRugi())}},
	})
	var buf bytes.Buffer
	if err := RenderBandingPDF(&buf, "Kedai Senja", b); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, mau := range []string{"MediaBox [0 0 841.89 595.28]", "Perbandingan", "Juli 2026", "Agustus 2026", "TOTAL", "72.000"} {
		if !strings.Contains(s, mau) {
			t.Errorf("PDF banding tidak memuat %q", mau)
		}
	}
}

func TestLabelPeriode(t *testing.T) {
	if LabelPeriode("2026-08") != "Agustus 2026" || labelPeriodePendek("2026-02") != "Feb 2026" || labelPeriodePendek("2026-05") != "Mei 2026" {
		t.Errorf("%q %q %q", LabelPeriode("2026-08"), labelPeriodePendek("2026-02"), labelPeriodePendek("2026-05"))
	}
}
