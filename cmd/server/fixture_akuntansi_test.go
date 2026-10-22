package main

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"pajak/internal/finance"
)

// Fixture penerimaan handoff platform akuntansi (docs/Handoff_Accounting_Platform_Coding_Agent
// §11): usaha dagang baru, saldo awal nol, tanpa pajak, persediaan perpetual rata-rata bergerak.
//  1. Setor modal ke bank 10.000.000
//  2. Beli 10 unit A @100.000, lunas dari bank
//  3. Jual kredit 2 unit A @150.000
//  4. Terima 200.000 dari pelanggan, dialokasikan ke invoice itu (Pembayaran sebagian)
//  5. Biaya operasional 50.000 dari bank
//
// Plus fixture costing: beli 10 @100.000 lalu 10 @120.000, jual 2 → HPP 220.000, sisa 18 unit
// bernilai 1.980.000.
func TestFixtureAkuntansi_Handoff(t *testing.T) {
	srv, qa := usahaBaru(t)
	perpetual := strings.Replace(dagangCV, `"periodik"`, `"perpetual"`, 1)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/business", perpetual); rec.Code != http.StatusOK {
		t.Fatalf("profil perpetual: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/stok/metode", `{"metode":"rata_rata"}`); rec.Code != http.StatusOK {
		t.Fatalf("metode rata-rata: %d %s", rec.Code, rec.Body)
	}
	a := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"sku":"A","nama":"Barang A","kategori":"Barang Dagang","satuan":"unit","harga_pokok":100000}`)
	b := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"sku":"B","nama":"Barang B","kategori":"Barang Dagang","satuan":"unit","harga_pokok":100000}`)
	ida, idb := strconv.Itoa(a.ID), strconv.Itoa(b.ID)
	buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/awal", `{"tanggal":"2026-01-01","baris":[{"stok_id":`+ida+`,"fisik":0,"harga_unit":0},{"stok_id":`+idb+`,"fisik":0,"harga_unit":0}]}`)

	for _, body := range []string{
		`{"tanggal":"2026-01-02","jenis":"Setor Modal","nominal":10000000,"lunas":true,"channel":"BCA Bisnis"}`,
		`{"tanggal":"2026-01-03","jenis":"Beli Bahan / Stok","nominal":1000000,"lunas":true,"channel":"BCA Bisnis","stok_id":` + ida + `,"qty_stok":10000}`,
	} {
		buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", body)
	}
	jual := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions",
		`{"tanggal":"2026-01-05","jenis":"Penjualan","nominal":300000,"pihak":"Pelanggan A","jatuh_tempo":"2026-02-04","stok_id":`+ida+`,"qty_stok":2000}`)
	if k, _ := bayarAPI(t, srv, qa, "fixture-bayar", `{"tanggal":"2026-01-10","jenis":"piutang","lawan":"1102","alokasi":[{"transaksi_id":`+strconv.Itoa(jual.ID)+`,"nominal":200000}]}`); k != http.StatusCreated {
		t.Fatalf("terima pembayaran: %d", k)
	}
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-01-12","jenis":"Biaya Lain","nominal":50000,"lunas":true,"channel":"BCA Bisnis","keterangan":"Biaya operasional"}`)

	bs := getJSON[finance.BalanceSheet](t, srv, qa, "/api/v1/reports/balance-sheet?period=2026-01")
	is := getJSON[finance.IncomeStatement](t, srv, qa, "/api/v1/reports/financial?period=2026-01")
	for nama, c := range map[string][2]int64{
		"Bank":             {bs.Assets.Cash, 9_150_000},
		"Piutang":          {bs.Assets.Receivables, 100_000},
		"Nilai persediaan": {bs.Assets.Inventory, 800_000},
		"Utang":            {bs.Liabilities.Payables, 0},
		"Pendapatan":       {is.Omzet, 300_000},
		"HPP":              {is.COGS, 200_000},
		"Beban operasi":    {is.Opex, 50_000},
		"Laba":             {is.NetProfit, 50_000},
		"Total aset":       {bs.Assets.Total, 10_050_000},
		"Modal + laba":     {bs.Liabilities.Total + bs.Equity.Total, 10_050_000},
	} {
		if c[0] != c[1] {
			t.Errorf("%s = %d, mau %d", nama, c[0], c[1])
		}
	}
	if !bs.IsBalanced {
		t.Error("neraca tidak seimbang")
	}
	stok := getJSON[finance.StokSummary](t, srv, qa, "/api/v1/stok")
	for _, r := range stok.Items {
		if r.ID == a.ID && r.JumlahMili != 8_000 {
			t.Errorf("stok A = %d mili, mau 8 unit", r.JumlahMili)
		}
	}
	// Laporan historis: per tanggal sebelum pembayaran, piutang masih 300.000.
	if u := getJSON[finance.UmurSaldoResult](t, srv, qa, "/api/v1/reports/umur?per=2026-01-09"); u.Total != 300_000 {
		t.Errorf("piutang per 9 Jan = %d, mau 300.000", u.Total)
	}
	if u := getJSON[finance.UmurSaldoResult](t, srv, qa, "/api/v1/reports/umur?per=2026-01-31"); u.Total != 100_000 {
		t.Errorf("piutang per 31 Jan = %d, mau 100.000", u.Total)
	}
	// Neraca saldo per akun: seimbang, piutang & persediaan cocok dengan sub-buku, bank menerima
	// pembayaran pelanggan di tanggal bayarnya.
	saldo := map[string]int64{}
	var d, k int64
	for _, x := range getJSON[coaReport](t, srv, qa, "/api/v1/reports/coa?period=2026-01").TrialBalance {
		d, k = d+x.Debit, k+x.Kredit
		saldo[x.Kode] = x.Saldo
	}
	if d != k || saldo["1201"] != 100_000 || saldo["1301"] != 800_000 || saldo["1102"]+saldo["1101"] != 9_150_000 {
		t.Errorf("neraca saldo: debit %d kredit %d, 1201 %d, 1301 %d, kas+bank %d", d, k, saldo["1201"], saldo["1301"], saldo["1101"]+saldo["1102"])
	}

	// Costing rata-rata bergerak (Februari, barang B).
	for _, body := range []string{
		`{"tanggal":"2026-02-02","jenis":"Beli Bahan / Stok","nominal":1000000,"lunas":true,"channel":"BCA Bisnis","stok_id":` + idb + `,"qty_stok":10000}`,
		`{"tanggal":"2026-02-03","jenis":"Beli Bahan / Stok","nominal":1200000,"lunas":true,"channel":"BCA Bisnis","stok_id":` + idb + `,"qty_stok":10000}`,
		`{"tanggal":"2026-02-04","jenis":"Penjualan","nominal":400000,"lunas":true,"channel":"BCA Bisnis","stok_id":` + idb + `,"qty_stok":2000}`,
	} {
		buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", body)
	}
	feb := getJSON[finance.IncomeStatement](t, srv, qa, "/api/v1/reports/financial?period=2026-02")
	febBS := getJSON[finance.BalanceSheet](t, srv, qa, "/api/v1/reports/balance-sheet?period=2026-02")
	if feb.COGS != 220_000 || febBS.Assets.Inventory != 800_000+1_980_000 || !febBS.IsBalanced {
		t.Errorf("costing: HPP %d (mau 220.000), persediaan %d (mau 2.780.000), seimbang %v", feb.COGS, febBS.Assets.Inventory, febBS.IsBalanced)
	}
}
