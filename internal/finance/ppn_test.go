package finance

import "testing"

// PPN keluaran adalah titipan negara: tidak menambah omzet/laba, menambah utang PPN di neraca,
// dan arus kas operasi metode langsung = tidak langsung — saat invoice belum dibayar, sesudah
// dibayar, dan sesudah PPN disetor.
func TestPPNKeluaran_BukanOmzetTapiUtangPajak(t *testing.T) {
	ob := OpeningBalance{Tanggal: "2026-08-01", Kas: 10_000_000, ModalDisetor: 10_000_000}
	txs := []Transaction{
		{ID: 1, Tanggal: "2026-08-05", Jenis: "Penjualan", Nominal: 10_000_000, Pihak: "PT A"},
		{ID: 2, Tanggal: "2026-08-05", Jenis: "PPN Keluaran", Nominal: 1_100_000, Pihak: "PT A"},
	}
	cek := func(nama string, txs []Transaction, period string, utang, piutang int64) {
		t.Helper()
		r := BuildLaporanBerjalan(txs, nil, period, ob, KartuStok{})
		bs, cf := r.BalanceSheet, r.CashFlow
		if !bs.IsBalanced {
			t.Errorf("%s: neraca tidak seimbang %+v", nama, bs)
		}
		if bs.Liabilities.Other != utang || bs.Assets.Receivables != piutang {
			t.Errorf("%s: utang PPN %d (want %d), piutang %d (want %d)", nama, bs.Liabilities.Other, utang, bs.Assets.Receivables, piutang)
		}
		if cf.OperasiLangsung.Net != cf.OperasiTidakLangsung.Net {
			t.Errorf("%s: kas operasi langsung %d ≠ tidak langsung %d", nama, cf.OperasiLangsung.Net, cf.OperasiTidakLangsung.Net)
		}
	}
	r := BuildLaporanBerjalan(txs, nil, "2026-08", ob, KartuStok{})
	if r.Income.Omzet != 10_000_000 {
		t.Fatalf("omzet harus tanpa PPN: %d", r.Income.Omzet)
	}
	cek("belum dibayar", txs, "2026-08", 1_100_000, 11_100_000)

	lunas := []Transaction{txs[0], txs[1]}
	lunas[0].Lunas, lunas[0].TanggalLunas = true, "2026-09-10"
	lunas[1].Lunas, lunas[1].TanggalLunas = true, "2026-09-10"
	cek("dibayar bulan berikutnya", lunas, "2026-09", 1_100_000, 0)

	setor := append(lunas, Transaction{ID: 3, Tanggal: "2026-10-15", Jenis: "Setor PPN", Nominal: 1_100_000, Lunas: true})
	cek("sesudah disetor", setor, "2026-10", 0, 0)
}
