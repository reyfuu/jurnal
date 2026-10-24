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
// Aturan "minus baru": hitung kartu sekarang (s.txs, s.gerak) dan kartu usulan. Untuk tiap baris
// kartu usulan yang minus (barang B, tanggal D, saldo q < 0), batas lamanya adalah saldo B
// terendah di kartu sekarang pada tanggal D — minimum SaldoQty baris B bertanggal D, atau bila
// tidak ada baris di tanggal itu, saldo B yang terbawa sampai akhir D (0 bila kartu belum
// dimulai). Baris usulan ditolak bila q lebih rendah dari batas itu: barang yang tidak minus jadi
// minus, atau minus lama menjadi lebih dalam / lebih awal. Baris minus lama yang tidak berubah
// punya q sama dengan batasnya, jadi lolos. Yang dilaporkan baris pertama (urutan kartu).

func (s *tenant) cekTanpaMinusLocked(txs []finance.Transaction, gerak []finance.GerakStok) error {
	if !s.perpetualLocked() {
		return nil
	}
	m, _ := s.metodeStokLocked()
	usulan := finance.HitungKartuMetode(m, s.stok, txs, gerak)
	if !adaMinus(usulan) {
		return nil
	}
	batas := batasSaldoLama(s.kartuLocked())
	for _, b := range usulan.Baris {
		if b.Minus && b.SaldoQty < batas(b.StokID, b.Tanggal) {
			return s.galatStokKurang(b)
		}
	}
	return nil
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

// galatStokKurang: "stok <nama> tidak cukup pada <tanggal>: tersedia <qty> <satuan>, dibutuhkan <qty> <satuan>".
// tersedia = saldo sebelum baris itu; dibutuhkan = jumlah yang dikeluarkan baris itu.
func (s *tenant) galatStokKurang(b finance.BarisKartu) error {
	it, _ := s.stokByID(b.StokID)
	nama := strings.TrimSpace(it.Nama)
	if nama == "" {
		nama = fmt.Sprintf("#%d", b.StokID)
	}
	satuan := ""
	if sat := strings.TrimSpace(it.Satuan); sat != "" {
		satuan = " " + sat
	}
	return fmt.Errorf("stok %s tidak cukup pada %s: tersedia %s%s, dibutuhkan %s%s", nama, finance.FormatTanggalID(b.Tanggal),
		finance.FormatQty(b.SaldoQty-b.Qty), satuan, finance.FormatQty(max(0, -b.Qty)), satuan)
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
