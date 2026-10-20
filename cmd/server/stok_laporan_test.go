package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"pajak/internal/finance"
)

type coaRingkas struct {
	IncomeStatement struct {
		LabaBersih      int64 `json:"laba_bersih"`
		HPP             int64 `json:"hpp"`
		PersediaanAkhir int64 `json:"persediaan_akhir"`
	} `json:"income_statement"`
	BalanceSheet struct {
		Seimbang bool `json:"seimbang"`
	} `json:"balance_sheet"`
}

func TestKartuStok_MasukLaporanKeuangan(t *testing.T) {
	for _, metode := range []string{"periodik", "perpetual"} {
		t.Run(metode, func(t *testing.T) {
			t.Setenv("DATA_DIR", t.TempDir())
			srv := NewServer()
			qa := register(t, srv, registerBody)
			aktifkan(t, srv, qa)
			if rec := do(srv, qa, http.MethodPut, "/api/v1/business", strings.Replace(dagangCV, "periodik", metode, 1)); rec.Code != http.StatusOK {
				t.Fatalf("profil: %d %s", rec.Code, rec.Body)
			}
			kopi := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kopi","kategori":"Barang","satuan":"kg","harga_pokok":200000}`)
			id := itoa(kopi.ID)
			buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/awal", `{"tanggal":"2026-01-01","baris":[{"stok_id":`+id+`,"fisik":10000,"harga_unit":200000}]}`)
			for _, tx := range []string{
				`{"tanggal":"2026-01-05","jenis":"Beli Bahan / Stok","nominal":2500000,"lunas":true,"stok_id":` + id + `,"qty_stok":10000}`,
				`{"tanggal":"2026-01-06","jenis":"Beli Bahan / Stok","nominal":300000,"lunas":false}`,
				`{"tanggal":"2026-01-10","jenis":"Penjualan","nominal":3000000,"lunas":true,"stok_id":` + id + `,"qty_stok":2500}`,
			} {
				buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", tx)
			}

			is := getJSON[finance.IncomeStatement](t, srv, qa, "/api/v1/reports/financial?period=2026-01")
			// HPP = modal 2,5 kg @225rb (562.500) + belanja tanpa barang 300rb; pembelian 2,5 jt masuk persediaan.
			if is.COGS != 862_500 || is.HPPStok == nil || is.NetProfit != 3_000_000-862_500 {
				t.Fatalf("laba rugi: hpp %d laba %d rincian %+v", is.COGS, is.NetProfit, is.HPPStok)
			}
			bs := getJSON[finance.BalanceSheet](t, srv, qa, "/api/v1/reports/balance-sheet?period=2026-01")
			if bs.Assets.Inventory != 3_937_500 || !bs.IsBalanced {
				t.Errorf("neraca %+v", bs)
			}
			cf := getJSON[finance.CashFlow](t, srv, qa, "/api/v1/reports/cashflow?period=2026-01")
			if cf.OperasiLangsung.Net != cf.OperasiTidakLangsung.Net || cf.KenaikanPersediaan != 1_937_500 {
				t.Errorf("arus kas langsung %d tidak langsung %d", cf.OperasiLangsung.Net, cf.OperasiTidakLangsung.Net)
			}
			c := getJSON[coaRingkas](t, srv, qa, "/api/v1/reports/coa?period=2026-01")
			if c.IncomeStatement.LabaBersih != is.NetProfit || c.IncomeStatement.HPP != is.COGS || !c.BalanceSheet.Seimbang {
				t.Errorf("laporan CoA %s: %+v (laba sederhana %d)", metode, c, is.NetProfit)
			}
			if metode == "periodik" && c.IncomeStatement.PersediaanAkhir != 3_937_500 {
				t.Errorf("persediaan akhir CoA %d", c.IncomeStatement.PersediaanAkhir)
			}
		})
	}
}

// Buku berjalan tiga bulan: laporan CoA (per akun) harus sama dengan laporan ringkas.
func TestBukuBerjalan_CoASamaDenganRingkas(t *testing.T) {
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

	for _, p := range []string{"2026-01", "2026-02", "2026-03"} {
		bs := getJSON[finance.BalanceSheet](t, srv, qa, "/api/v1/reports/balance-sheet?period="+p)
		is := getJSON[finance.IncomeStatement](t, srv, qa, "/api/v1/reports/financial?period="+p)
		var c struct {
			IncomeStatement struct {
				LabaBersih int64 `json:"laba_bersih"`
			} `json:"income_statement"`
			BalanceSheet struct {
				TotalAset int64 `json:"total_aset"`
				Seimbang  bool  `json:"seimbang"`
			} `json:"balance_sheet"`
		}
		_ = json.NewDecoder(do(srv, qa, http.MethodGet, "/api/v1/reports/coa?period="+p, "").Body).Decode(&c)
		if !bs.IsBalanced || !c.BalanceSheet.Seimbang {
			t.Errorf("%s seimbang: ringkas %v CoA %v", p, bs.IsBalanced, c.BalanceSheet.Seimbang)
		}
		if c.IncomeStatement.LabaBersih != is.NetProfit || c.BalanceSheet.TotalAset != bs.Assets.Total {
			t.Errorf("%s CoA laba %d aset %d ≠ ringkas laba %d aset %d", p, c.IncomeStatement.LabaBersih, c.BalanceSheet.TotalAset, is.NetProfit, bs.Assets.Total)
		}
	}
}

func TestUmurPiutang_DanPelunasan(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	qa := register(t, srv, registerBody)
	aktifkan(t, srv, qa)
	jual := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-01-10","jenis":"Penjualan","nominal":2000000,"lunas":false,"pihak":"Toko Maju"}`)
	u := getJSON[finance.UmurSaldoResult](t, srv, qa, "/api/v1/reports/umur?jenis=piutang&per=2026-03-01")
	bs := getJSON[finance.BalanceSheet](t, srv, qa, "/api/v1/reports/balance-sheet?period=2026-02")
	if u.Total != 2_000_000 || u.Total != bs.Assets.Receivables || u.Kelompok[1].Nilai != 2_000_000 || u.Pihak[0].Nama != "Toko Maju" {
		t.Fatalf("umur %+v, piutang neraca %d", u, bs.Assets.Receivables)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions/"+itoa(jual.ID)+"/lunas", `{"tanggal":"2026-01-01"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("lunas sebelum transaksi: %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions/"+itoa(jual.ID)+"/lunas", `{"tanggal":"2026-03-05"}`); rec.Code != http.StatusOK {
		t.Fatalf("lunasi: %d %s", rec.Code, rec.Body)
	}
	// Kas masuk Maret (bukan Januari); piutang Februari tetap ada, Maret sudah nol.
	if feb := getJSON[finance.BalanceSheet](t, srv, qa, "/api/v1/reports/balance-sheet?period=2026-02"); feb.Assets.Receivables != 2_000_000 {
		t.Errorf("piutang Februari %d", feb.Assets.Receivables)
	}
	mar := getJSON[finance.CashFlow](t, srv, qa, "/api/v1/reports/cashflow?period=2026-03")
	jan := getJSON[finance.CashFlow](t, srv, qa, "/api/v1/reports/cashflow?period=2026-01")
	if mar.CashIn != 2_000_000 || jan.CashIn != 0 {
		t.Errorf("kas masuk Januari %d Maret %d", jan.CashIn, mar.CashIn)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions/"+itoa(jual.ID)+"/lunas", `{"tanggal":"2026-03-06"}`); rec.Code != http.StatusConflict {
		t.Errorf("lunas dua kali: %d", rec.Code)
	}
}
