package coa

import (
	"slices"

	"pajak/internal/finance"
)

// peranJenis: peran akun per jenis transaksi. Jenis yang bergantung profil ditangani di
// AutoAccount.
var peranJenis = map[string]string{
	"Penjualan": PeranPenjualan, "Pendapatan Lain": PeranPendapatanLain, "Setor Modal": PeranModal, "Terima Pinjaman": PeranPinjaman,
	"Kemasan": PeranKemasan, "Potongan Aplikasi": PeranPotonganAplikasi, "Ongkir & Transport": PeranOngkir, "Gaji & Upah": PeranGaji,
	"Sewa Tempat": PeranSewa, "Listrik, Air, Internet": PeranListrik, "Iklan & Promosi": PeranIklan,
	"Perawatan & Perlengkapan": PeranPerawatan, "Pajak & Izin": PeranBebanPajak, "Biaya Lain": PeranBiayaLain,
	"Beli Alat / Aset": PeranPeralatan, "Bayar Cicilan": PeranPinjaman, "Pindah Dana": PeranBank, // tujuan bawaan; Catat selalu mengisi Akun
	"PPN Keluaran": PeranPPNKeluaran, "Setor PPN": PeranPPNKeluaran,
}

// modalKode: akun modal; template boleh memisahkan modal PT (modal disetor) dari modal pemilik
// usaha perorangan/CV lewat peran opsional "modal_pt".
func modalKode(peta Peta, p finance.BusinessProfile) string {
	if isPT(p) && peta["modal_pt"] != "" {
		return peta["modal_pt"]
	}
	return peta.Kode(PeranModal)
}

// AutoAccount: "CoA otomatis" — akun untuk jenis transaksi sesuai profil usaha dan template.
func AutoAccount(jenis string, p finance.BusinessProfile, peta Peta) string {
	switch jenis {
	case "Beli Bahan / Stok":
		if perpetual(p) {
			return peta.Kode(PeranPersediaan) // perpetual: pembelian langsung menambah persediaan
		}
		return peta.Kode(PeranPembelian)
	case "Ambil Uang Pribadi":
		if isPT(p) {
			return peta.Kode(PeranLabaDitahan) // PT: pengambilan pemilik = pembagian laba
		}
		return peta.Kode(PeranPrive)
	case "Setor Modal":
		return modalKode(peta, p)
	}
	return peta.Kode(peranJenis[jenis])
}

// AccountFor: akun koreksi user bila ada, selain itu otomatis.
func AccountFor(t finance.Transaction, p finance.BusinessProfile, peta Peta) string {
	if t.Akun != "" {
		return t.Akun
	}
	return AutoAccount(t.Jenis, p, peta)
}

var marketplace = []string{"GoFood", "GrabFood", "ShopeeFood"}

// ChannelKhusus: channel yang uangnya selalu masuk ke akun tertentu (saldo marketplace / bank
// B2B), jadi tidak diberi rekening bawaan.
func ChannelKhusus(channel string) bool {
	return slices.Contains(marketplace, channel) || channel == "BCA Bisnis" || channel == "B2B Tempo"
}

// AkunBayar: rekening tempat uang transaksi bergerak saat dibayar (Lawan pilihan pengguna,
// rekening bawaan, atau aturan channel).
func AkunBayar(t finance.Transaction, peta Peta) string {
	return CounterAccount(finance.Transaction{Jenis: t.Jenis, Lunas: true, Channel: t.Channel, Lawan: t.Lawan}, peta)
}

// CounterAccount: sisi kas/bank/piutang/utang dari transaksi.
func CounterAccount(t finance.Transaction, peta Peta) string {
	j, _ := finance.LookupJenis(t.Jenis)
	switch {
	case t.Lunas && t.Lawan != "":
		return t.Lawan // catatan jurnal: rekening kas/bank yang dipilih pengguna
	case !t.Lunas && j.Arah == "masuk":
		return peta.Kode(PeranPiutang)
	case !t.Lunas:
		return peta.Kode(PeranUtangUsaha)
	case slices.Contains(marketplace, t.Channel):
		return peta.Kode(PeranMarketplace)
	case t.Channel == "BCA Bisnis" || t.Channel == "B2B Tempo":
		return peta.Kode(PeranBank)
	}
	return peta.Kode(PeranKas)
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
func PenyesuaianPersediaan(tanggal string, selisih int64, p finance.BusinessProfile, peta Peta) []Line {
	hpp, persediaan := peta.Kode(PeranPembelian), peta.Kode(PeranPersediaan)
	if perpetual(p) {
		hpp = peta.Kode(PeranHPP)
	}
	if selisih >= 0 {
		return []Line{{Tanggal: tanggal, Kode: persediaan, Debit: selisih}, {Tanggal: tanggal, Kode: hpp, Kredit: selisih}}
	}
	return []Line{{Tanggal: tanggal, Kode: hpp, Debit: -selisih}, {Tanggal: tanggal, Kode: persediaan, Kredit: -selisih}}
}

// Journal: jurnal berpasangan (double entry) dari transaksi; 2 baris per transaksi.
func Journal(txs []finance.Transaction, p finance.BusinessProfile, peta Peta) []Line {
	out := make([]Line, 0, 2*len(txs))
	for _, t := range txs {
		j, _ := finance.LookupJenis(t.Jenis)
		akun, lawan := AccountFor(t, p, peta), CounterAccount(t, peta)
		if j.Arah == "masuk" {
			out = append(out, Line{t.ID, t.Tanggal, lawan, t.Nominal, 0}, Line{t.ID, t.Tanggal, akun, 0, t.Nominal})
		} else {
			out = append(out, Line{t.ID, t.Tanggal, akun, t.Nominal, 0}, Line{t.ID, t.Tanggal, lawan, 0, t.Nominal})
		}
	}
	return out
}

// JournalBerjalan: jurnal satu periode untuk buku berjalan. Transaksi bulan ini dijurnal ke
// kas bila sudah lunas sampai akhir bulan, selain itu ke piutang (1201) / utang (2101);
// transaksi bulan lalu yang dilunasi bulan ini mendapat jurnal pelunasan (Dr Kas / Cr Piutang,
// atau Dr Utang / Cr Kas) di tanggal bayar.
func JournalBerjalan(txs []finance.Transaction, period string, p finance.BusinessProfile, peta Peta) []Line {
	piutang, utang := peta.Kode(PeranPiutang), peta.Kode(PeranUtangUsaha)
	akhir := period + "-31"
	var out []Line
	for _, t := range txs {
		if len(t.Tanggal) < 7 {
			continue
		}
		j, _ := finance.LookupJenis(t.Jenis)
		masuk := j.Arah == "masuk"
		bayar := t.TanggalLunas
		if bayar == "" {
			bayar = t.Tanggal
		}
		kas := AkunBayar(t, peta)
		switch {
		case t.Tanggal[:7] == period:
			lawan := kas
			if !t.Lunas || bayar > akhir {
				lawan = utang
				if masuk {
					lawan = piutang
				}
			}
			akun := AccountFor(t, p, peta)
			if masuk {
				out = append(out, Line{t.ID, t.Tanggal, lawan, t.Nominal, 0}, Line{t.ID, t.Tanggal, akun, 0, t.Nominal})
			} else {
				out = append(out, Line{t.ID, t.Tanggal, akun, t.Nominal, 0}, Line{t.ID, t.Tanggal, lawan, 0, t.Nominal})
			}
		case t.Tanggal[:7] < period && t.Lunas && len(bayar) >= 7 && bayar[:7] == period:
			if masuk {
				out = append(out, Line{t.ID, bayar, kas, t.Nominal, 0}, Line{t.ID, bayar, piutang, 0, t.Nominal})
			} else {
				out = append(out, Line{t.ID, bayar, utang, t.Nominal, 0}, Line{t.ID, bayar, kas, 0, t.Nominal})
			}
		}
	}
	return out
}

// OpeningBerjalan: saldo awal akun pada awal periode dari posisi buku berjalan. Modal (3101)
// adalah selisih aset − kewajiban − laba ditahan (3201). lain: saldo awal akun lain per kode
// (posisi debit positif, kewajiban negatif) — aset lancar lain & kewajiban lain saldo awal.
func OpeningBerjalan(kas, piutang, persediaan, utang, pinjaman, labaDitahan int64, assetAwal, lain map[string]int64, peta Peta, p finance.BusinessProfile) map[string]int64 {
	out := map[string]int64{}
	for kode, v := range map[string]int64{peta.Kode(PeranKas): kas, peta.Kode(PeranPiutang): piutang, peta.Kode(PeranPersediaan): persediaan,
		peta.Kode(PeranUtangUsaha): -utang, peta.Kode(PeranPinjaman): -pinjaman, peta.Kode(PeranLabaDitahan): -labaDitahan} {
		out[kode] += v
	}
	bersih := kas + piutang + persediaan - utang - pinjaman
	for kode, v := range assetAwal {
		out[kode] += v
		bersih += v
	}
	for kode, v := range lain {
		out[kode] += v
		bersih += v
	}
	out[modalKode(peta, p)] += -(bersih - labaDitahan)
	return out
}

// akunAset: akun harta, akumulasi, dan beban untuk satu aset. Dipilih dari jenis hartanya
// supaya peralatan, gedung, kendaraan, dan harta tak berwujud tidak menumpuk di satu akun.
func akunAset(a finance.Asset, peta Peta) (harta, akumulasi, beban string) {
	susut := peta.Kode(PeranBebanPenyusutan)
	switch a.JenisHarta {
	case "kendaraan", "sepeda_motor":
		return peta.Kode(PeranKendaraan), peta.Kode(PeranAkumKendaraan), susut
	case "bangunan_permanen", "bangunan_tidak_permanen":
		return peta.Kode(PeranGedung), peta.Kode(PeranAkumGedung), susut
	}
	if a.TakBerwujud {
		return peta.Kode(PeranTakBerwujud), peta.Kode(PeranAkumAmortisasi), peta.Kode(PeranBebanAmortisasi)
	}
	return peta.Kode(PeranPeralatan), peta.Kode(PeranAkumPeralatan), susut
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
func DepreciationJournal(as finance.AssetSummary, period string, peta Peta, p finance.BusinessProfile) []Line {
	tanggal := period + "-01"
	out := make([]Line, 0, 2*len(as.Assets))
	for _, r := range as.Assets {
		harta, akumulasi, beban := akunAset(r.Asset, peta)
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
				Line{0, tanggal, modalKode(peta, p), 0, r.HargaPerolehan},
			)
		}
	}
	return out
}

// OpeningAssets: posisi aset awal periode per akun — harga perolehan di akun harta, penyusutan
// periode sebelumnya di akun akumulasi (negatif karena peta ini berposisi debit). Dipisah,
// bukan diringkas jadi nilai buku, supaya neraca per akun menyajikan kontra seperti biasa.
// Tanpa ini aset yang sudah dimiliki sebelum periode hilang dari neraca per akun.
func OpeningAssets(as finance.AssetSummary, period string, peta Peta) map[string]int64 {
	out := map[string]int64{}
	for _, r := range as.Assets {
		if diperolehPadaPeriode(r, period) {
			continue // perolehan periode ini masuk lewat jurnal, bukan saldo awal
		}
		harta, akumulasi, _ := akunAset(r.Asset, peta)
		out[harta] += r.HargaPerolehan
		out[akumulasi] -= r.HargaPerolehan - r.NilaiBukuAwal
	}
	return out
}

// Opening: saldo awal periode (posisi debit positif). Modal awal = kas + persediaan + nilai
// buku aset awal, sama dengan neraca ringkas. assetAwal dari OpeningAssets; nil bila tanpa aset.
func Opening(cash, inventory int64, assetAwal map[string]int64, peta Peta) map[string]int64 {
	out := map[string]int64{peta.Kode(PeranKas): cash}
	out[peta.Kode(PeranPersediaan)] += inventory
	total := cash + inventory
	for kode, v := range assetAwal {
		out[kode] += v
		total += v
	}
	out[peta.Kode(PeranModal)] += -total
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

// SaldoAwalLain: akun aset lancar lain & kewajiban lain dari saldo awal (posisi debit positif).
// Kode dari rincian dipakai bila ada di daftar akun; selain itu 1401 Biaya Dibayar Dimuka dan
// 2105 Utang Lain-lain menampung totalnya.
func SaldoAwalLain(ob finance.OpeningBalance, akun []Account, peta Peta) map[string]int64 {
	out := map[string]int64{}
	asetLain, kewLain := peta.Kode(PeranDibayarDimuka), peta.Kode(PeranUtangLain)
	if len(ob.Rincian) == 0 {
		if ob.AsetLain != 0 {
			out[asetLain] = ob.AsetLain
		}
		if ob.KewajibanLain != 0 {
			out[kewLain] = -ob.KewajibanLain
		}
		return out
	}
	for _, r := range ob.Rincian {
		kode, tanda := "", int64(1)
		switch r.Kategori {
		case finance.KatAsetLain:
			kode = asetLain
		case finance.KatKewajibanLain:
			kode, tanda = kewLain, -1
		case finance.KatLabaDitahan:
			// Selisih saldo awal yang disimpan paksa: tetap di akun penampungnya, bukan di Laba
			// Ditahan biasa. Pos laba ditahan (OpeningBerjalan) sudah memuatnya, jadi dikembalikan.
			if r.Kode == KodeSelisihSaldoAwal && r.Kode != peta.Kode(PeranLabaDitahan) {
				if _, ok := Find(akun, r.Kode); ok {
					out[r.Kode] -= r.Nominal
					out[peta.Kode(PeranLabaDitahan)] += r.Nominal
				}
			}
			continue
		default:
			continue
		}
		if kodeSendiri(r, akun) {
			kode = r.Kode
		}
		out[kode] += tanda * r.Nominal
	}
	return out
}

// SaldoBawaan: saldo akun yang dibawa utuh dari bulan ke bulan di saldo awal per akun, selain
// pos ringkas (kas, piutang, utang, …): kas di akun selain Kas (bank, saldo marketplace) dan
// prive kumulatif. Tanpa ini saldo awal bulanan menggabungkan semua kas ke akun Kas dan prive
// ke Modal, sehingga riwayat Buku Besar per akun terputus di setiap awal bulan (totalnya sama).
// sebelum: tanggal awal periode (YYYY-MM-DD); posisi debit positif.
func SaldoBawaan(txs []finance.Transaction, sebelum string, p finance.BusinessProfile, peta Peta) map[string]int64 {
	out := map[string]int64{}
	kas, prive := peta.Kode(PeranKas), peta.Kode(PeranPrive)
	for _, t := range txs {
		j, ok := finance.LookupJenis(t.Jenis)
		if !ok {
			continue
		}
		tanda := int64(1)
		if j.Arah != "masuk" {
			tanda = -1
		}
		bayar := t.TanggalLunas
		if bayar == "" {
			bayar = t.Tanggal
		}
		if t.Lunas && bayar < sebelum {
			if k := AkunBayar(t, peta); k != kas {
				out[k] += tanda * t.Nominal
			}
			// pindah dana: rekening tujuan bertambah (Kas tidak dicatat: sisa dari total kas)
			if j.Pos == finance.PosPindah {
				if k := AccountFor(t, p, peta); k != kas {
					out[k] += t.Nominal
				}
			}
		}
		if t.Tanggal < sebelum && !isPT(p) && AccountFor(t, p, peta) == prive {
			out[prive] += t.Nominal
		}
	}
	return out
}
