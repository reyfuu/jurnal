package coa

import (
	"slices"

	"pajak/internal/finance"
)

// LineKet: baris jurnal kartu stok beserta keterangannya untuk Jurnal Umum / Buku Besar.
type LineKet struct {
	Line
	Ket string
}

// JurnalKartu: jurnal perpetual per kejadian kartu stok bertanggal [dari, sampai], di tanggal
// kejadiannya dan ke akun persediaan tiap barang:
//   - penjualan: Dr HPP / Cr akun barang sebesar nilai keluar (TxID = transaksi penjualan);
//   - stok opname: selisih lebih Dr akun barang / Cr HPP, selisih kurang kebalikannya;
//   - produksi, per dokumen: bahan terpakai Cr akun bahan, hasil Dr akun hasil sebesar nilai
//     tanpa biaya lain (biaya lain dijurnal oleh transaksi biayanya sendiri).
//
// Stok awal & pembelian tidak dijurnal di sini (sudah lewat saldo awal / transaksi), begitu pula
// koreksi nilai stok minus (disapu penyesuaian sisa akhir bulan). akunStok nil = akun peran
// persediaan. Hanya untuk metode perpetual; pemanggil yang memeriksa metodenya.
func JurnalKartu(baris []finance.BarisKartu, dari, sampai string, akunStok func(int) string, peta Peta) []Line {
	lk := JurnalKartuKet(baris, dari, sampai, akunStok, peta)
	out := make([]Line, len(lk))
	for i, l := range lk {
		out[i] = l.Line
	}
	return out
}

// JurnalKartuKet: JurnalKartu dengan keterangan per baris — "Produksi — <keterangan dokumen>"
// (atau "Produksi"), "HPP — <keterangan>" (atau "HPP"), dan "Selisih stok opname".
func JurnalKartuKet(baris []finance.BarisKartu, dari, sampai string, akunStok func(int) string, peta Peta) []LineKet {
	akun := func(id int) string {
		if akunStok != nil {
			if k := akunStok(id); k != "" {
				return k
			}
		}
		return peta.Kode(PeranPersediaan)
	}
	hpp := peta.Kode(PeranHPP)
	var out []LineKet
	gerak := map[int][]int{} // dokumen produksi → indeks baris out
	var urutGerak []int
	for _, b := range baris {
		if b.Tanggal < dari || b.Tanggal > sampai {
			continue
		}
		switch b.Jenis {
		case finance.KartuJual:
			if b.Nilai == 0 {
				continue
			}
			ket := "HPP"
			if b.Keterangan != "" {
				ket += " — " + b.Keterangan
			}
			out = append(out,
				LineKet{Line{TxID: b.TransaksiID, Tanggal: b.Tanggal, Kode: hpp, Debit: -b.Nilai}, ket},
				LineKet{Line{TxID: b.TransaksiID, Tanggal: b.Tanggal, Kode: akun(b.StokID), Kredit: -b.Nilai}, ket})
		case finance.KartuOpname:
			const ket = "Selisih stok opname"
			switch {
			case b.Nilai > 0:
				out = append(out,
					LineKet{Line{Tanggal: b.Tanggal, Kode: akun(b.StokID), Debit: b.Nilai}, ket},
					LineKet{Line{Tanggal: b.Tanggal, Kode: hpp, Kredit: b.Nilai}, ket})
			case b.Nilai < 0:
				out = append(out,
					LineKet{Line{Tanggal: b.Tanggal, Kode: hpp, Debit: -b.Nilai}, ket},
					LineKet{Line{Tanggal: b.Tanggal, Kode: akun(b.StokID), Kredit: -b.Nilai}, ket})
			}
		case finance.KartuProduksiPakai, finance.KartuProduksiMasuk:
			ket := "Produksi"
			if b.Keterangan != "" {
				ket += " — " + b.Keterangan
			}
			l := Line{Tanggal: b.Tanggal, Kode: akun(b.StokID)}
			if b.Jenis == finance.KartuProduksiPakai {
				l.Kredit = -b.Nilai
			} else {
				l.Debit = b.Nilai - b.Biaya
			}
			if l.Debit == 0 && l.Kredit == 0 {
				continue
			}
			if _, ada := gerak[b.GerakID]; !ada {
				urutGerak = append(urutGerak, b.GerakID)
			}
			gerak[b.GerakID] = append(gerak[b.GerakID], len(out))
			out = append(out, LineKet{l, ket})
		}
	}
	// Pembulatan pembagian nilai hasil: selisih debit − kredit satu dokumen produksi ditambahkan
	// ke baris debit terbesar (seri → yang pertama), supaya jurnalnya tetap seimbang.
	for _, g := range urutGerak {
		idx := gerak[g]
		var selisih int64
		for _, i := range idx {
			selisih += out[i].Kredit - out[i].Debit
		}
		if selisih == 0 {
			continue
		}
		terbesar := -1
		for _, i := range idx {
			if out[i].Debit > 0 && (terbesar < 0 || out[i].Debit > out[terbesar].Debit) {
				terbesar = i
			}
		}
		if terbesar < 0 || out[terbesar].Debit+selisih < 0 {
			// tanpa baris hasil: lawannya HPP supaya tetap seimbang
			l := out[slices.Max(idx)]
			lawan := Line{Tanggal: l.Tanggal, Kode: hpp}
			if selisih > 0 {
				lawan.Debit = selisih
			} else {
				lawan.Kredit = -selisih
			}
			out = append(out, LineKet{lawan, l.Ket})
			continue
		}
		out[terbesar].Debit += selisih
	}
	return out
}
