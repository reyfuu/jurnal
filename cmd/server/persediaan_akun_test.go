package main

import (
	"net/http"
	"testing"

	"pajak/internal/coa"
	"pajak/internal/finance"
)

const manufakturPeriodik = `{"nama_usaha":"Konveksi Senja","jenis_usaha":"manufaktur","bentuk_usaha":"op","ptkp":"K/1","pkp":false,"metode_persediaan":"periodik"}`

// saldoAkun: saldo akun di neraca saldo laporan per akun.
func saldoAkun(c coaReport, kode string) int64 {
	for _, b := range c.TrialBalance {
		if b.Kode == kode {
			return b.Saldo
		}
	}
	return 0
}

// totalPersediaan: jumlah saldo semua akun persediaan (subtype inventory) di neraca saldo.
func totalPersediaan(c coaReport) int64 {
	var n int64
	for _, b := range c.TrialBalance {
		if b.Subtype == "inventory" {
			n += b.Saldo
		}
	}
	return n
}

func neracaSaldoSeimbang(t *testing.T, c coaReport) {
	t.Helper()
	var d, k int64
	for _, b := range c.TrialBalance {
		d, k = d+b.Debit, k+b.Kredit
	}
	if d != k {
		t.Errorf("neraca saldo tidak seimbang: debit %d kredit %d", d, k)
	}
}

func usahaManufaktur(t *testing.T) (*Server, *http.Cookie) {
	t.Helper()
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	qa := register(t, srv, registerBody)
	aktifkan(t, srv, qa)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/business", manufakturPeriodik); rec.Code != http.StatusOK {
		t.Fatalf("profil: %d %s", rec.Code, rec.Body)
	}
	return srv, qa
}

func TestPersediaanAkun_PeriodikDuaAkun(t *testing.T) {
	srv, qa := usahaManufaktur(t)
	kain := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kain","kategori":"Bahan Baku","satuan":"m"}`)
	kaos := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kaos","kategori":"Produk Jadi","satuan":"pcs"}`)
	buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/awal", `{"tanggal":"2026-01-01","baris":[`+
		`{"stok_id":`+itoa(kain.ID)+`,"fisik":10000,"harga_unit":10000},`+
		`{"stok_id":`+itoa(kaos.ID)+`,"fisik":5000,"harga_unit":40000}]}`)
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions",
		`{"tanggal":"2026-01-05","jenis":"Beli Bahan / Stok","nominal":50000,"lunas":true,"stok_id":`+itoa(kain.ID)+`,"qty_stok":5000}`)

	c := getJSON[coaReport](t, srv, qa, "/api/v1/reports/coa?period=2026-01")
	if got := saldoAkun(c, "1301"); got != 150_000 {
		t.Errorf("saldo 1301 = %d, harus 150.000", got)
	}
	if got := saldoAkun(c, "1305"); got != 200_000 {
		t.Errorf("saldo 1305 = %d, harus 200.000", got)
	}
	is := getJSON[finance.IncomeStatement](t, srv, qa, "/api/v1/reports/financial?period=2026-01")
	if c.IncomeStatement == nil || c.IncomeStatement.LabaBersih != is.NetProfit {
		t.Errorf("laba CoA %+v ≠ ringkas %d", c.IncomeStatement, is.NetProfit)
	}
	if c.IncomeStatement != nil && (c.IncomeStatement.PersediaanAwal != 300_000 || c.IncomeStatement.PersediaanAkhir != 350_000) {
		t.Errorf("persediaan awal/akhir CoA %d/%d", c.IncomeStatement.PersediaanAwal, c.IncomeStatement.PersediaanAkhir)
	}
	if c.BalanceSheet == nil || !c.BalanceSheet.Seimbang {
		t.Errorf("neraca CoA tidak seimbang")
	}
	neracaSaldoSeimbang(t, c)

	// Bulan berikutnya: saldo awal per akun dibawa dari kartu, tanpa penyesuaian baru.
	feb := getJSON[coaReport](t, srv, qa, "/api/v1/reports/coa?period=2026-02")
	if saldoAkun(feb, "1301") != 150_000 || saldoAkun(feb, "1305") != 200_000 {
		t.Errorf("Februari 1301 %d 1305 %d", saldoAkun(feb, "1301"), saldoAkun(feb, "1305"))
	}
	neracaSaldoSeimbang(t, feb)
}

func TestPersediaanAkun_DobelTertutup(t *testing.T) {
	srv, qa := usahaManufaktur(t)
	kain := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kain","kategori":"Bahan Baku","satuan":"m"}`)
	buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/awal", `{"tanggal":"2026-01-01","baris":[{"stok_id":`+itoa(kain.ID)+`,"fisik":10000,"harga_unit":10000}]}`)
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions",
		`{"tanggal":"2026-01-10","jenis":"Beli Bahan / Stok","nominal":300000,"lunas":true,"akun":"1302"}`)

	c := getJSON[coaReport](t, srv, qa, "/api/v1/reports/coa?period=2026-01")
	tn := tenantByEmail(t, srv, qa)
	tn.mu.RLock()
	akhir := tn.kartuLocked().Periode("2026-01").PersediaanAkhir
	tn.mu.RUnlock()
	if akhir != 100_000 {
		t.Fatalf("persediaan akhir kartu %d", akhir)
	}
	if got := totalPersediaan(c); got != akhir {
		t.Errorf("total akun persediaan %d, harus = kartu %d", got, akhir)
	}
	if got := saldoAkun(c, "1302"); got != 0 {
		t.Errorf("1302 harus ditutup ke nilai kartu (0), dapat %d", got)
	}
	is := getJSON[finance.IncomeStatement](t, srv, qa, "/api/v1/reports/financial?period=2026-01")
	if c.IncomeStatement == nil || c.IncomeStatement.LabaBersih != is.NetProfit {
		t.Errorf("laba CoA %+v ≠ ringkas %d", c.IncomeStatement, is.NetProfit)
	}
	if c.BalanceSheet == nil || !c.BalanceSheet.Seimbang {
		t.Errorf("neraca CoA tidak seimbang")
	}
	neracaSaldoSeimbang(t, c)
}

// Saldo awal per akun tanpa kartu: baris persediaan saldo awal berkode akun persediaan masuk ke
// akunnya, sisanya ke akun peran; jumlahnya selalu = saldo awal persediaan laporan ringkas.
func TestPersediaanAkun_SaldoAwalRincian(t *testing.T) {
	srv, qa := usahaManufaktur(t)
	tn := tenantByEmail(t, srv, qa)
	ob := finance.OpeningBalance{Tanggal: "2026-01-01", Rincian: []finance.SaldoAkun{
		{Kode: "1102", Nama: "Kas", Kategori: finance.KatKas, Nominal: 1_000_000},
		{Kode: "1301", Nama: "Persediaan Bahan Baku", Kategori: finance.KatPersediaan, Nominal: 50_000},
		{Kode: "1305", Nama: "Persediaan Barang Jadi", Kategori: finance.KatPersediaan, Nominal: 70_000},
		{Nama: "Persediaan lain", Kategori: finance.KatPersediaan, Nominal: 30_000},
	}}
	if err := ob.TerapkanRincian(); err != nil {
		t.Fatal(err)
	}
	tn.mu.Lock()
	tn.opening = &ob
	tn.mu.Unlock()

	c := getJSON[coaReport](t, srv, qa, "/api/v1/reports/coa?period=2026-01")
	if saldoAkun(c, "1301") != 80_000 || saldoAkun(c, "1305") != 70_000 {
		t.Errorf("1301 %d 1305 %d", saldoAkun(c, "1301"), saldoAkun(c, "1305"))
	}
	neracaSaldoSeimbang(t, c)

	all, _, _ := tn.coaAkun()
	peta := coa.PetaDari(all)
	tn.mu.RLock()
	defer tn.mu.RUnlock()
	for _, p := range []string{"2026-01", "2026-02"} {
		var n int64
		for _, v := range tn.persediaanAwalPerAkunLocked(p, all, peta) {
			n += v
		}
		if want := tn.openingFor(p).Inventory; n != want {
			t.Errorf("%s: jumlah saldo awal per akun %d ≠ opening.Inventory %d", p, n, want)
		}
	}
}

// Dagang (satu akun persediaan): bulan sebelum titik nol, bulan titik nol, dan sesudahnya sama
// dengan laporan sebelum persediaan dipecah per akun.
func TestPersediaanAkun_SebelumKartuSama(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	qa := register(t, srv, registerBody)
	aktifkan(t, srv, qa)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/business", dagangCV); rec.Code != http.StatusOK {
		t.Fatalf("profil: %d", rec.Code)
	}
	tn := tenantByEmail(t, srv, qa)
	tn.mu.Lock()
	tn.opening = &finance.OpeningBalance{Tanggal: "2026-01-01", Kas: 10_000_000, Piutang: 1_000_000, Persediaan: 2_000_000, Utang: 500_000, Pinjaman: 3_000_000}
	tn.txs = []finance.Transaction{
		{ID: 1, Tanggal: "2026-01-05", Jenis: "Penjualan", Nominal: 4_000_000, Lunas: true},
		{ID: 2, Tanggal: "2026-01-20", Jenis: "Penjualan", Nominal: 1_500_000, Lunas: true, TanggalLunas: "2026-02-10"},
		{ID: 3, Tanggal: "2026-01-21", Jenis: "Penjualan", Nominal: 800_000},
		{ID: 4, Tanggal: "2026-01-25", Jenis: "Beli Bahan / Stok", Nominal: 1_200_000, Lunas: true, TanggalLunas: "2026-02-03"},
		{ID: 5, Tanggal: "2026-01-28", Jenis: "Gaji & Upah", Nominal: 900_000, Lunas: true},
		{ID: 6, Tanggal: "2026-02-05", Jenis: "Terima Pinjaman", Nominal: 2_000_000, Lunas: true},
		{ID: 7, Tanggal: "2026-02-20", Jenis: "Ambil Uang Pribadi", Nominal: 400_000, Lunas: true},
		{ID: 8, Tanggal: "2026-02-25", Jenis: "Bayar Cicilan", Nominal: 500_000, Lunas: true},
		{ID: 9, Tanggal: "2026-03-02", Jenis: "Setor Modal", Nominal: 1_000_000, Lunas: true},
		{ID: 10, Tanggal: "2026-03-06", Jenis: "Listrik, Air, Internet", Nominal: 300_000, Lunas: false},
	}
	tn.nextTxID = 11
	tn.mu.Unlock()
	kopi := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kopi","kategori":"Barang","satuan":"kg","harga_pokok":300000}`)
	buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/awal", `{"tanggal":"2026-03-01","baris":[{"stok_id":`+itoa(kopi.ID)+`,"fisik":5000,"harga_unit":300000}]}`)
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions",
		`{"tanggal":"2026-04-05","jenis":"Penjualan","nominal":900000,"lunas":true,"stok_id":`+itoa(kopi.ID)+`,"qty_stok":2000}`)

	// Angka dari kode sebelum persediaan dipecah per akun (satu akun 1301).
	harap := map[string]struct{ laba, aset, persediaan, persAwal, persAkhir int64 }{
		"2026-01": {4_200_000, 18_400_000, 2_000_000, 2_000_000, 2_000_000},
		"2026-02": {0, 18_300_000, 2_000_000, 2_000_000, 2_000_000},
		"2026-03": {-300_000, 18_800_000, 1_500_000, 1_500_000, 1_500_000}, // titik nol: persediaan = nilai kartu
		"2026-04": {300_000, 19_100_000, 900_000, 1_500_000, 900_000},
	}
	for _, p := range []string{"2026-01", "2026-02", "2026-03", "2026-04"} {
		c := getJSON[coaReport](t, srv, qa, "/api/v1/reports/coa?period="+p)
		is := getJSON[finance.IncomeStatement](t, srv, qa, "/api/v1/reports/financial?period="+p)
		bs := getJSON[finance.BalanceSheet](t, srv, qa, "/api/v1/reports/balance-sheet?period="+p)
		if c.IncomeStatement == nil || c.BalanceSheet == nil {
			t.Fatalf("%s: laporan CoA kosong", p)
		}
		h := harap[p]
		got := struct{ laba, aset, persediaan, persAwal, persAkhir int64 }{c.IncomeStatement.LabaBersih, c.BalanceSheet.TotalAset,
			saldoAkun(c, "1301"), c.IncomeStatement.PersediaanAwal, c.IncomeStatement.PersediaanAkhir}
		if got != h {
			t.Errorf("%s: dapat %+v, harus %+v", p, got, h)
		}
		if c.IncomeStatement.LabaBersih != is.NetProfit || c.BalanceSheet.TotalAset != bs.Assets.Total || !c.BalanceSheet.Seimbang {
			t.Errorf("%s: CoA laba %d aset %d ≠ ringkas laba %d aset %d", p, c.IncomeStatement.LabaBersih, c.BalanceSheet.TotalAset, is.NetProfit, bs.Assets.Total)
		}
		neracaSaldoSeimbang(t, c)
	}
}
