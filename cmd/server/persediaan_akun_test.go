package main

import (
	"net/http"
	"strings"
	"testing"

	"pajak/internal/coa"
	"pajak/internal/finance"
	"pajak/internal/xlsx"
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

	// Ekspor Excel: persediaan awal = jumlah semua akun, jadi tanpa kode akun peran.
	rec := do(srv, qa, http.MethodGet, "/api/v1/reports/coa.xlsx?period=2026-01", "")
	rows, err := xlsx.Read(rec.Body.Bytes())
	if rec.Code != http.StatusOK || err != nil {
		t.Fatalf("ekspor xlsx: %d %v", rec.Code, err)
	}
	var awalXlsx []string
	for _, r := range rows {
		if len(r) >= 3 && strings.HasPrefix(r[1], "Persediaan Awal") {
			awalXlsx = r
		}
	}
	if len(awalXlsx) < 3 || awalXlsx[0] != "" || awalXlsx[1] != "Persediaan Awal (semua akun)" || awalXlsx[2] != "300000" {
		t.Errorf("baris persediaan awal xlsx: %q", awalXlsx)
	}

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
	// Rincian HPP menunjukkan persediaan awal & pembelian sebenarnya.
	if is := c.IncomeStatement; is != nil {
		var pembelian int64
		for _, l := range is.HPPLines {
			if l.Kode == "5102" {
				pembelian = l.Jumlah
			}
		}
		if is.PersediaanAwal != 100_000 || is.PersediaanAkhir != 100_000 || pembelian != 300_000 || is.HPP != 300_000 {
			t.Errorf("rincian HPP: awal %d pembelian %d akhir %d hpp %d", is.PersediaanAwal, pembelian, is.PersediaanAkhir, is.HPP)
		}
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

// Penyesuaian persediaan bertanggal hari terakhir bulan dan tampil paling akhir di Jurnal Umum.
func TestPersediaanAkun_PenyesuaianAkhirBulan(t *testing.T) {
	srv, qa := usahaManufaktur(t)
	kain := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kain","kategori":"Bahan Baku","satuan":"m"}`)
	buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/awal", `{"tanggal":"2026-01-01","baris":[{"stok_id":`+itoa(kain.ID)+`,"fisik":10000,"harga_unit":10000}]}`)
	for _, tgl := range []string{"2026-01-05", "2026-02-05"} {
		buat[finance.Transaction](t, srv, qa, "/api/v1/transactions",
			`{"tanggal":"`+tgl+`","jenis":"Beli Bahan / Stok","nominal":50000,"lunas":true,"stok_id":`+itoa(kain.ID)+`,"qty_stok":5000}`)
	}
	tn := tenantByEmail(t, srv, qa)
	for _, k := range []struct{ period, dari, akhir string }{{"2026-01", "2026-01-01", "2026-01-31"}, {"2026-02", "2026-02-01", "2026-02-28"}} {
		j, ok := tn.jurnalPeriode(k.period)
		if !ok {
			t.Fatalf("%s: jurnal tidak ada", k.period)
		}
		var ada bool
		for _, e := range j.entri {
			if e.Ket == "Penyesuaian persediaan ke nilai kartu stok" {
				ada = true
				if e.Tanggal != k.akhir {
					t.Errorf("%s: tanggal penyesuaian %s, harus %s", k.period, e.Tanggal, k.akhir)
				}
			}
		}
		if !ada {
			t.Fatalf("%s: tidak ada jurnal penyesuaian", k.period)
		}
		ju := tn.jurnalUmum(k.dari, k.akhir, "", "")
		if n := len(ju.Entri); n == 0 || ju.Entri[n-1].Keterangan != "Penyesuaian persediaan ke nilai kartu stok" || ju.Entri[n-1].Tanggal != k.akhir {
			t.Errorf("%s: penyesuaian harus entri terakhir Jurnal Umum: %+v", k.period, ju.Entri)
		}
	}
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

const manufakturPerpetual = `{"nama_usaha":"Konveksi Senja","jenis_usaha":"manufaktur","bentuk_usaha":"op","ptkp":"K/1","pkp":false,"metode_persediaan":"perpetual"}`

func usahaManufakturPerpetual(t *testing.T) (*Server, *http.Cookie) {
	t.Helper()
	srv, qa := usahaManufaktur(t)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/business", manufakturPerpetual); rec.Code != http.StatusOK {
		t.Fatalf("profil perpetual: %d %s", rec.Code, rec.Body)
	}
	return srv, qa
}

const ketPenyesuaian = "Penyesuaian persediaan ke nilai kartu stok"

// Perpetual: pembelian, produksi bertahap, dan penjualan dijurnal di tanggal kejadiannya ke akun
// barangnya, jadi penyesuaian sisa akhir bulan nol untuk semua akun.
func TestPersediaanAkun_PerpetualAlurManufaktur(t *testing.T) {
	srv, qa := usahaManufakturPerpetual(t)
	kain := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kain","kategori":"Bahan Baku","satuan":"m"}`)
	potongan := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Potongan","kategori":"Barang Setengah Jadi","satuan":"pcs"}`)
	kaos := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kaos","kategori":"Produk Jadi","satuan":"pcs"}`)
	buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/awal", `{"tanggal":"2026-01-01","baris":[`+
		`{"stok_id":`+itoa(kain.ID)+`,"fisik":1000,"harga_unit":10000},`+
		`{"stok_id":`+itoa(potongan.ID)+`,"fisik":0,"harga_unit":0},`+
		`{"stok_id":`+itoa(kaos.ID)+`,"fisik":0,"harga_unit":0}]}`)
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions",
		`{"tanggal":"2026-01-03","jenis":"Beli Bahan / Stok","nominal":100000,"lunas":true,"stok_id":`+itoa(kain.ID)+`,"qty_stok":10000}`)
	pPotongan := buat[finance.Produk](t, srv, qa, "/api/v1/produksi/produk", `{"nama":"Potongan","satuan":"pcs","hasil_per_batch":10,"bahan":[{"stok_id":`+itoa(kain.ID)+`,"takaran":6000}]}`)
	pKaos := buat[finance.Produk](t, srv, qa, "/api/v1/produksi/produk", `{"nama":"Kaos","satuan":"pcs","hasil_per_batch":10,"bahan":[{"stok_id":`+itoa(potongan.ID)+`,"takaran":10000}]}`)
	// Biaya lain diberi akun barang hasil (1303) seperti Task 6 kelak membuatnya otomatis.
	for _, body := range []string{
		`{"tanggal":"2026-01-05","produk":[{"produk_id":` + itoa(pPotongan.ID) + `,"hasil":10000,"bahan":[{"stok_id":` + itoa(kain.ID) + `,"qty":6000}],` +
			`"biaya":[{"keterangan":"Upah potong","akun":"1303","nominal":20000}]}]}`,
		`{"tanggal":"2026-01-08","produk":[{"produk_id":` + itoa(pKaos.ID) + `,"hasil":10000,"bahan":[{"stok_id":` + itoa(potongan.ID) + `,"qty":10000}]}]}`,
	} {
		if rec := do(srv, qa, http.MethodPost, "/api/v1/stok/produksi/catat", body); rec.Code != http.StatusCreated {
			t.Fatalf("catat produksi: %d %s", rec.Code, rec.Body)
		}
	}
	jual := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions",
		`{"tanggal":"2026-01-15","jenis":"Penjualan","nominal":100000,"lunas":true,"keterangan":"Kaos ke toko","stok_id":`+itoa(kaos.ID)+`,"qty_stok":4000}`)

	c := getJSON[coaReport](t, srv, qa, "/api/v1/reports/coa?period=2026-01")
	tn := tenantByEmail(t, srv, qa)
	all, _, _ := tn.coaAkun()
	tn.mu.RLock()
	target := tn.kartuLocked().NilaiPerAkun("2026-01-31", tn.akunStokLocked(all, coa.PetaDari(all)))
	tn.mu.RUnlock()
	for kode, mau := range map[string]int64{"1301": 50_000, "1303": 0, "1305": 48_000} {
		if target[kode] != mau {
			t.Errorf("kartu %s = %d, harus %d", kode, target[kode], mau)
		}
		if got := saldoAkun(c, kode); got != target[kode] {
			t.Errorf("saldo %s = %d, harus = kartu %d", kode, got, target[kode])
		}
	}
	j, _ := tn.jurnalPeriode("2026-01")
	for _, e := range j.entri {
		if e.Ket == ketPenyesuaian {
			t.Errorf("penyesuaian sisa harus 0: %+v", e)
		}
	}
	var hpp bool
	for _, e := range tn.jurnalUmum("2026-01-01", "2026-01-31", "", "").Entri {
		for _, b := range e.Baris {
			if b.Kode == "5101" && b.Debit == 32_000 {
				hpp = true
				if e.Tanggal != "2026-01-15" || e.TxID != jual.ID || !strings.HasPrefix(e.Keterangan, "HPP — Penjualan") {
					t.Errorf("entri HPP: tanggal %s tx %d ket %q", e.Tanggal, e.TxID, e.Keterangan)
				}
			}
		}
		if strings.HasPrefix(e.Keterangan, "Produksi") && (e.Debit != e.Kredit || !strings.HasPrefix(e.Bukti, "SIS-")) {
			t.Errorf("entri produksi %+v", e)
		}
	}
	if !hpp {
		t.Errorf("Jurnal Umum tanpa baris HPP 32.000")
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

// Perpetual: pembelian tanpa barang masuk akun peran persediaan, lalu disapu penyesuaian sisa ke
// HPP; total persediaan tetap = kartu (Review Focus 4).
func TestPersediaanAkun_PerpetualBeliTanpaBarang(t *testing.T) {
	srv, qa := usahaManufakturPerpetual(t)
	kain := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kain","kategori":"Bahan Baku","satuan":"m"}`)
	buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/awal", `{"tanggal":"2026-01-01","baris":[{"stok_id":`+itoa(kain.ID)+`,"fisik":1000,"harga_unit":10000}]}`)
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions",
		`{"tanggal":"2026-01-10","jenis":"Beli Bahan / Stok","nominal":100000,"lunas":true}`)

	c := getJSON[coaReport](t, srv, qa, "/api/v1/reports/coa?period=2026-01")
	tn := tenantByEmail(t, srv, qa)
	tn.mu.RLock()
	akhir := tn.kartuLocked().Periode("2026-01").PersediaanAkhir
	tn.mu.RUnlock()
	if got := totalPersediaan(c); got != akhir || akhir != 10_000 {
		t.Errorf("total akun persediaan %d, kartu %d (harus 10.000)", got, akhir)
	}
	if got := saldoAkun(c, "5101"); got != 100_000 {
		t.Errorf("HPP 5101 = %d, harus 100.000 (pembelian tanpa barang disapu)", got)
	}
	is := getJSON[finance.IncomeStatement](t, srv, qa, "/api/v1/reports/financial?period=2026-01")
	if c.IncomeStatement == nil || c.IncomeStatement.LabaBersih != is.NetProfit {
		t.Errorf("laba CoA %+v ≠ ringkas %d", c.IncomeStatement, is.NetProfit)
	}
	neracaSaldoSeimbang(t, c)
}
