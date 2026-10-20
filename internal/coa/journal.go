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

// PenyesuaianPersediaan: jurnal akhir periode supaya 1301 Persediaan = nilai kartu stok.
// Lawannya akun HPP: 5102 (periodik) atau 5101 (perpetual).
func PenyesuaianPersediaan(tanggal string, selisih int64, p finance.BusinessProfile) []Line {
	hpp := "5102"
	if perpetual(p) {
		hpp = "5101"
	}
	if selisih >= 0 {
		return []Line{{Tanggal: tanggal, Kode: "1301", Debit: selisih}, {Tanggal: tanggal, Kode: hpp, Kredit: selisih}}
	}
	return []Line{{Tanggal: tanggal, Kode: hpp, Debit: -selisih}, {Tanggal: tanggal, Kode: "1301", Kredit: -selisih}}
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

// akunAset: akun harta, akumulasi, dan beban untuk satu aset. Dipilih dari jenis hartanya
// supaya peralatan, gedung, kendaraan, dan harta tak berwujud tidak menumpuk di satu akun.
func akunAset(a finance.Asset) (harta, akumulasi, beban string) {
	switch a.JenisHarta {
	case "kendaraan", "sepeda_motor":
		return "1505", "1506", "6113"
	case "bangunan_permanen", "bangunan_tidak_permanen":
		return "1503", "1504", "6113"
	}
	if a.TakBerwujud {
		return "1507", "1508", "6118"
	}
	return "1501", "1502", "6113"
}

// diperolehPadaPeriode: aset baru masuk periode ini, jadi belum punya nilai buku awal.
func diperolehPadaPeriode(r finance.AssetReport, period string) bool {
	return len(r.TanggalPerolehan) >= 7 && r.TanggalPerolehan[:7] == period
}

// DepreciationJournal: beban penyusutan & amortisasi periode ini dari register aset, plus aset
// yang masuk tanpa transaksi kas (mis. hasil impor data awal).
//
// Tanpa jurnal ini laporan per akun tidak mengenal penyusutan sama sekali, sehingga laba
// bersihnya berbeda dengan tab Laba Rugi untuk periode yang sama.
func DepreciationJournal(as finance.AssetSummary, period string) []Line {
	tanggal := period + "-01"
	out := make([]Line, 0, 2*len(as.Assets))
	for _, r := range as.Assets {
		harta, akumulasi, beban := akunAset(r.Asset)
		if r.BebanBulanIni != 0 {
			out = append(out,
				Line{0, tanggal, beban, r.BebanBulanIni, 0},
				Line{0, tanggal, akumulasi, 0, r.BebanBulanIni},
			)
		}
		// Aset tanpa transaksi kas belum punya sisi debit di jurnal transaksi; dibukukan
		// sebagai setoran modal berupa aset, sama dengan perlakuannya di neraca ringkas.
		if r.TransaksiID == 0 && !r.Draft && diperolehPadaPeriode(r, period) {
			out = append(out,
				Line{0, tanggal, harta, r.HargaPerolehan, 0},
				Line{0, tanggal, "3101", 0, r.HargaPerolehan},
			)
		}
	}
	return out
}

// OpeningAssets: posisi aset awal periode per akun — harga perolehan di akun harta, penyusutan
// periode sebelumnya di akun akumulasi (negatif karena peta ini berposisi debit). Dipisah,
// bukan diringkas jadi nilai buku, supaya neraca per akun menyajikan kontra seperti biasa.
// Tanpa ini aset yang sudah dimiliki sebelum periode hilang dari neraca per akun.
func OpeningAssets(as finance.AssetSummary, period string) map[string]int64 {
	out := map[string]int64{}
	for _, r := range as.Assets {
		if diperolehPadaPeriode(r, period) {
			continue // perolehan periode ini masuk lewat jurnal, bukan saldo awal
		}
		harta, akumulasi, _ := akunAset(r.Asset)
		out[harta] += r.HargaPerolehan
		out[akumulasi] -= r.HargaPerolehan - r.NilaiBukuAwal
	}
	return out
}

// Opening: saldo awal periode (posisi debit positif). Modal awal = kas + persediaan + nilai
// buku aset awal, sama dengan neraca ringkas. assetAwal dari OpeningAssets; nil bila tanpa aset.
func Opening(cash, inventory int64, assetAwal map[string]int64) map[string]int64 {
	out := map[string]int64{"1101": cash, "1301": inventory}
	total := cash + inventory
	for kode, v := range assetAwal {
		out[kode] += v
		total += v
	}
	out["3101"] = -total
	return out
}

type Balance struct {
	Account
	Debit  int64 `json:"debit"`  // posisi neraca saldo
	Kredit int64 `json:"kredit"` //
	Saldo  int64 `json:"saldo"`  // menurut saldo normal akun (negatif = berlawanan)
}

// TrialBalance: neraca saldo dari saldo awal + jurnal, urut sesuai CoA Master.
func TrialBalance(active []Account, opening map[string]int64, lines []Line) []Balance {
	return TrialBalanceFrom(Master, active, opening, lines)
}

// TrialBalanceFrom: neraca saldo urut sesuai template; akun nonaktif tetap muncul bila bersaldo.
func TrialBalanceFrom(template, active []Account, opening map[string]int64, lines []Line) []Balance {
	net := map[string]int64{}
	for k, v := range opening {
		net[k] += v
	}
	for _, l := range lines {
		net[l.Kode] += l.Debit - l.Kredit
	}
	var out []Balance
	for _, acc := range template {
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
