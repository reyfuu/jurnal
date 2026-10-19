package coa

import (
	"slices"

	"pajak/internal/finance"
)

// autoMap: akun bawaan per jenis transaksi (perdagangan). Akun yang bergantung profil
// ditangani di AutoAccount.
var autoMap = map[string]string{
	"Penjualan": "4101", "Pendapatan Lain": "8101", "Setor Modal": "3101", "Terima Pinjaman": "2102",
	"Kemasan": "5104", "Potongan Aplikasi": "6108", "Ongkir & Transport": "6106", "Gaji & Upah": "6101",
	"Sewa Tempat": "6104", "Listrik, Air, Internet": "6105", "Iklan & Promosi": "6102",
	"Perawatan & Perlengkapan": "6112", "Pajak & Izin": "9103", "Biaya Lain": "6190",
	"Beli Alat / Aset": "1501", "Bayar Cicilan": "2102",
}

// AutoAccount: "CoA otomatis" — akun untuk jenis transaksi sesuai profil usaha.
func AutoAccount(jenis string, p finance.BusinessProfile) string {
	switch jenis {
	case "Beli Bahan / Stok":
		if perpetual(p) {
			return "1301" // perpetual: pembelian langsung menambah persediaan
		}
		return "5102"
	case "Ambil Uang Pribadi":
		if isPT(p) {
			return "3201" // PT: pengambilan pemilik = pembagian laba (mengurangi laba ditahan)
		}
		return "3102"
	}
	return autoMap[jenis]
}

// AccountFor: akun koreksi user bila ada, selain itu otomatis.
func AccountFor(t finance.Transaction, p finance.BusinessProfile) string {
	if t.Akun != "" {
		return t.Akun
	}
	return AutoAccount(t.Jenis, p)
}

var marketplace = []string{"GoFood", "GrabFood", "ShopeeFood"}

// CounterAccount: sisi kas/bank/piutang/utang dari transaksi.
func CounterAccount(t finance.Transaction) string {
	j, _ := finance.LookupJenis(t.Jenis)
	switch {
	case !t.Lunas && j.Arah == "masuk":
		return "1201"
	case !t.Lunas:
		return "2101"
	case slices.Contains(marketplace, t.Channel):
		return "1104"
	case t.Channel == "BCA Bisnis" || t.Channel == "B2B Tempo":
		return "1102"
	}
	return "1101"
}

type Line struct {
	TxID    int    `json:"tx_id"`
	Tanggal string `json:"tanggal"`
	Kode    string `json:"kode"`
	Debit   int64  `json:"debit"`
	Kredit  int64  `json:"kredit"`
}

// Journal: jurnal berpasangan (double entry) dari transaksi; 2 baris per transaksi.
func Journal(txs []finance.Transaction, p finance.BusinessProfile) []Line {
	out := make([]Line, 0, 2*len(txs))
	for _, t := range txs {
		j, _ := finance.LookupJenis(t.Jenis)
		akun, lawan := AccountFor(t, p), CounterAccount(t)
		if j.Arah == "masuk" {
			out = append(out, Line{t.ID, t.Tanggal, lawan, t.Nominal, 0}, Line{t.ID, t.Tanggal, akun, 0, t.Nominal})
		} else {
			out = append(out, Line{t.ID, t.Tanggal, akun, t.Nominal, 0}, Line{t.ID, t.Tanggal, lawan, 0, t.Nominal})
		}
	}
	return out
}

// Opening: saldo awal periode (posisi debit positif). Modal awal = kas awal + persediaan awal,
// sama dengan neraca ringkas yang lama.
func Opening(cash, inventory int64) map[string]int64 {
	return map[string]int64{"1101": cash, "1301": inventory, "3101": -(cash + inventory)}
}

type Balance struct {
	Account
	Debit  int64 `json:"debit"`  // posisi neraca saldo
	Kredit int64 `json:"kredit"` //
	Saldo  int64 `json:"saldo"`  // menurut saldo normal akun (negatif = berlawanan)
}

// TrialBalance: neraca saldo dari saldo awal + jurnal, urut sesuai CoA.
func TrialBalance(active []Account, opening map[string]int64, lines []Line) []Balance {
	net := map[string]int64{}
	for k, v := range opening {
		net[k] += v
	}
	for _, l := range lines {
		net[l.Kode] += l.Debit - l.Kredit
	}
	var out []Balance
	for _, acc := range Master { // urutan Master; akun nonaktif tetap muncul bila bersaldo
		n, ok := net[acc.Kode]
		if !ok {
			continue
		}
		if _, aktif := Find(active, acc.Kode); !aktif && n == 0 {
			continue
		}
		b := Balance{Account: acc}
		if n >= 0 {
			b.Debit = n
		} else {
			b.Kredit = -n
		}
		b.Saldo = n
		if acc.Normal == Kredit {
			b.Saldo = -n
		}
		out = append(out, b)
	}
	return out
}
