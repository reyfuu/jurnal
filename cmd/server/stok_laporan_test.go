package main

import (
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
