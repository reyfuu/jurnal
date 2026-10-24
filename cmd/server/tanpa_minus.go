package main

import (
	"fmt"
	"sort"
	"strings"

	"pajak/internal/finance"
)

// Stok tidak boleh minus (persediaan per akun tahap 2, Task 4): di metode perpetual setiap
// perubahan yang mengubah kartu stok (transaksi, opname, produksi, hapus dokumen, koreksi, dan
// jalur lain yang memanggil cekTanpaMinusLocked) dihitung dulu pada keadaan usulan dan ditolak
// (HTTP 409) bila membuat stok minus. Data lama yang sudah minus tidak memblokir perubahan lain.
//
// Aturan "minus baru": bandingkan kartu usulan dengan kartu dasar (biasanya keadaan sekarang;
// lihat cekTanpaMinusDariLocked). Baris dikenali dari sumbernya: (TransaksiID, GerakID, StokID,
// urutan baris barang itu di sumbernya). Untuk tiap baris usulan yang minus (saldo q < 0):
//   - sumbernya ada di kartu dasar → ditolak bila q lebih rendah dari saldo baris yang sama di
//     kartu dasar. Baris minus lama yang tidak berubah (juga bila transaksinya dipindah tanggal
//     tanpa memperdalam minus) lolos.
//   - baris baru (sumber belum ada di kartu dasar) → pembanding per tanggal: saldo barang itu
//     terendah pada tanggal yang sama di kartu dasar (minimum SaldoQty baris bertanggal itu, atau
//     saldo terbawa sampai akhir tanggal itu; 0 bila belum ada). Pembelian yang mengurangi minus
//     lama lolos; penjualan yang memperdalam minus atau membuat minus baru ditolak.
// Yang dilaporkan baris pertama yang melanggar (urutan kartu).

func (s *tenant) cekTanpaMinusLocked(txs []finance.Transaction, gerak []finance.GerakStok) error {
	return s.cekTanpaMinusDariLocked(s.txs, s.gerak, txs, gerak)
}

// cekTanpaMinusDariLocked: seperti cekTanpaMinusLocked dengan kartu dasar dari dasarTxs &
// dasarGerak. Dipakai saat kartu dimulai otomatis: dasarnya keadaan sekarang + stok awal otomatis
// (tanpa transaksi/dokumen yang diusulkan), supaya minus lama yang ikut tertarik ke kartu oleh stok
// awal itu tidak dianggap minus baru. s.mu dipegang.
func (s *tenant) cekTanpaMinusDariLocked(dasarTxs []finance.Transaction, dasarGerak []finance.GerakStok, txs []finance.Transaction, gerak []finance.GerakStok) error {
	if !s.perpetualLocked() {
		return nil
	}
	m, _ := s.metodeStokLocked()
	usulan := finance.HitungKartuMetode(m, s.stok, txs, gerak)
	if !adaMinus(usulan) {
		return nil
	}
	dasar := finance.HitungKartuMetode(m, s.stok, dasarTxs, dasarGerak)
	lama := barisPerSumber(dasar)
	batas := batasSaldoLama(dasar)
	for i, kunci := range kunciBaris(usulan) {
		b := usulan.Baris[i]
		if !b.Minus {
			continue
		}
		pembanding, ada := lama[kunci]
		if !ada {
			pembanding = batas(b.StokID, b.Tanggal)
		}
		if b.SaldoQty < pembanding {
			return s.galatStokKurang(usulan, i)
		}
	}
	return nil
}

// sumberBaris: identitas baris kartu menurut asal datanya.
type sumberBaris struct{ tx, gerak, stok, ke int }

// kunciBaris: kunci sumber tiap baris k.Baris (sejajar indeks).
func kunciBaris(k finance.KartuStok) []sumberBaris {
	hitung := map[sumberBaris]int{}
	out := make([]sumberBaris, len(k.Baris))
	for i, b := range k.Baris {
		x := sumberBaris{tx: b.TransaksiID, gerak: b.GerakID, stok: b.StokID}
		x.ke = hitung[x]
		hitung[x]++
		out[i] = x
	}
	return out
}

// barisPerSumber: saldo tiap baris kartu dasar menurut kuncinya.
func barisPerSumber(k finance.KartuStok) map[sumberBaris]int64 {
	out := map[sumberBaris]int64{}
	for i, x := range kunciBaris(k) {
		out[x] = k.Baris[i].SaldoQty
	}
	return out
}

func adaMinus(k finance.KartuStok) bool {
	for _, b := range k.Baris {
		if b.Minus {
			return true
		}
	}
	return false
}

// batasSaldoLama: fungsi (barang, tanggal) → saldo terendah barang itu pada tanggal itu di kartu k.
func batasSaldoLama(k finance.KartuStok) func(id int, tanggal string) int64 {
	type hari struct {
		tanggal    string
		min, akhir int64
	}
	per := map[int][]hari{} // urut tanggal, karena k.Baris urut tanggal
	for _, b := range k.Baris {
		h := per[b.StokID]
		if n := len(h); n > 0 && h[n-1].tanggal == b.Tanggal {
			h[n-1].min, h[n-1].akhir = min(h[n-1].min, b.SaldoQty), b.SaldoQty
		} else {
			h = append(h, hari{b.Tanggal, b.SaldoQty, b.SaldoQty})
		}
		per[b.StokID] = h
	}
	return func(id int, tanggal string) int64 {
		h := per[id]
		i := sort.Search(len(h), func(i int) bool { return h[i].tanggal >= tanggal })
		if i < len(h) && h[i].tanggal == tanggal {
			return h[i].min
		}
		if i == 0 {
			return 0
		}
		return h[i-1].akhir
	}
}

// galatStokKurang: pesan untuk baris k.Baris[i] yang membuat minus.
//   - baris keluar: "stok <nama> tidak cukup pada <tanggal>: tersedia <qty> <satuan>, dibutuhkan
//     <qty> <satuan>". Baris keluar barang yang sama dari sumber yang sama di tanggal itu
//     (mis. dua baris rincian satu penjualan) dijumlah; tersedia = saldo sebelum baris pertamanya.
//   - baris masuk (saldo tetap minus): "stok <nama> jadi minus pada <tanggal>: saldo <qty> <satuan>".
func (s *tenant) galatStokKurang(k finance.KartuStok, i int) error {
	b := k.Baris[i]
	it, _ := s.stokByID(b.StokID)
	nama := strings.TrimSpace(it.Nama)
	if nama == "" {
		nama = fmt.Sprintf("#%d", b.StokID)
	}
	satuan := ""
	if sat := strings.TrimSpace(it.Satuan); sat != "" {
		satuan = " " + sat
	}
	tgl := finance.FormatTanggalID(b.Tanggal)
	if b.Qty >= 0 {
		return fmt.Errorf("stok %s jadi minus pada %s: saldo %s%s", nama, tgl, finance.FormatQty(b.SaldoQty), satuan)
	}
	tersedia, butuh, pertama := int64(0), int64(0), true
	for _, x := range k.Baris {
		if x.Tanggal != b.Tanggal || x.StokID != b.StokID || x.TransaksiID != b.TransaksiID || x.GerakID != b.GerakID || x.Qty >= 0 {
			continue
		}
		if pertama {
			tersedia, pertama = x.SaldoQty-x.Qty, false
		}
		butuh -= x.Qty
	}
	return fmt.Errorf("stok %s tidak cukup pada %s: tersedia %s%s, dibutuhkan %s%s", nama, tgl,
		finance.FormatQty(tersedia), satuan, finance.FormatQty(butuh), satuan)
}

// tambahGerakUsulan: gerak sekarang + dokumen usulan dengan ID sementara sesuai urutan simpan
// (ID ikut menentukan urutan kartu di hari yang sama).
func (s *tenant) tambahGerakUsulan(baru ...*finance.GerakStok) []finance.GerakStok {
	out := append([]finance.GerakStok(nil), s.gerak...)
	id := s.nextGerakID
	for _, g := range baru {
		if g == nil {
			continue
		}
		x := *g
		x.ID = id
		id++
		out = append(out, x)
	}
	return out
}
