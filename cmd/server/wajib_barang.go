package main

import (
	"errors"
	"time"

	"pajak/internal/finance"
)

// Wajib barang (persediaan per akun tahap 2): di metode perpetual setiap Penjualan dan
// Beli Bahan / Stok membawa barang + jumlahnya, atau ditandai Bukan barang (jasa), supaya kartu
// stok dan jurnal perpetual tidak bolong. Transaksi bertanggal sebelum kartu stok dimulai
// dikecualikan (tercakup stok awal). Kartu yang belum dimulai dimulai otomatis oleh transaksi
// berbarang pertama dari daftar barang, sama seperti Catat Produksi pertama.

var errWajibBarang = errors.New("pilih barang & jumlahnya, atau tandai Bukan barang (jasa)")

const ketStokAwalTransaksi = "Stok awal otomatis dari daftar barang (transaksi pertama)"

func (s *tenant) perpetualLocked() bool {
	return s.profile != nil && s.profile.MetodePersediaan == "perpetual"
}

// mulaiKartuStokLocked: tanggal titik nol kartu stok (GerakAwal paling awal, sama dengan
// HitungKartu); kosong = belum dimulai. Tanpa menghitung ulang seluruh kartu.
func (s *tenant) mulaiKartuStokLocked() string {
	mulai := ""
	for _, g := range s.gerak {
		if g.Jenis == finance.GerakAwal && (mulai == "" || g.Tanggal < mulai) {
			mulai = g.Tanggal
		}
	}
	return mulai
}

var errBarangTidakAda = errors.New("barang yang dipilih tidak ada di Stok & Gudang")

// cekAdaBarangLocked: setiap barang di rincian masih ada di daftar barang. Dicek di bawah kunci
// tepat sebelum simpan, karena barang bisa dihapus sesudah cek awal (formulir, Catat, usulan
// koreksi yang menunggu persetujuan). s.mu dipegang.
func (s *tenant) cekAdaBarangLocked(t finance.Transaction) error {
	for _, b := range t.BarisBarang() {
		if _, ok := s.stokByID(b.StokID); !ok {
			return errBarangTidakAda
		}
	}
	return nil
}

// cekBarangLocked: barang di rincian harus ada (semua metode), lalu aturan wajib barang untuk
// satu transaksi usulan. s.mu dipegang.
// Berlaku juga bila daftar barang masih kosong: barang ditambah dulu, atau tandai jasa.
func (s *tenant) cekBarangLocked(t finance.Transaction) error {
	if err := s.cekAdaBarangLocked(t); err != nil {
		return err
	}
	if !s.perpetualLocked() || (t.Jenis != finance.JenisJual && t.Jenis != finance.JenisBeliStok) {
		return nil
	}
	if t.BukanBarang || len(t.BarisBarang()) > 0 {
		return nil // berbarang: bila kartu belum dimulai, mulaiKartuOtomatisLocked memulainya
	}
	if mulai := s.mulaiKartuStokLocked(); mulai != "" && t.Tanggal < mulai {
		return nil
	}
	return errWajibBarang
}

// stokAwalDariDaftarLocked: GerakAwal dari jumlah & harga pokok di daftar barang (titik nol
// kartu stok yang dimulai otomatis). Belum diberi ID dan belum disimpan.
func (s *tenant) stokAwalDariDaftarLocked(tanggal, keterangan string) finance.GerakStok {
	awal := finance.GerakStok{Tanggal: tanggal, Jenis: finance.GerakAwal, Keterangan: keterangan}
	for _, it := range s.stok {
		awal.Baris = append(awal.Baris, finance.BarisGerak{StokID: it.ID, Fisik: max(0, it.Jumlah) * finance.TakaranSkala, HargaUnit: it.HargaPokok})
	}
	return awal
}

// mulaiKartuOtomatisLocked: usulan stok awal bila perpetual dan kartu belum dimulai; nil bila
// tidak perlu. Pemanggil memastikan transaksinya berbarang, lalu menyimpannya lewat
// simpanStokAwalLocked bersama transaksinya (gagal → tidak ada yang tersimpan).
func (s *tenant) mulaiKartuOtomatisLocked(tanggal string) *finance.GerakStok {
	if !s.perpetualLocked() || len(s.stok) == 0 || s.mulaiKartuStokLocked() != "" {
		return nil
	}
	g := s.stokAwalDariDaftarLocked(tanggal, ketStokAwalTransaksi)
	return &g
}

// simpanStokAwalLocked: kunci metode nilai stok, beri ID, dan simpan stok awal. s.mu dipegang.
func (s *tenant) simpanStokAwalLocked(g *finance.GerakStok, oleh string, kini time.Time) error {
	if err := s.kunciMetodeStokLocked(); err != nil {
		return err
	}
	g.ID, g.Oleh, g.Dibuat = s.nextGerakID, oleh, kini
	s.nextGerakID++
	s.gerak = append(s.gerak, *g)
	return nil
}

// stokAwalTxLocked: usulan stok awal untuk transaksi berbarang yang akan disimpan; nil bila tidak perlu.
func (s *tenant) stokAwalTxLocked(t finance.Transaction) *finance.GerakStok {
	if len(t.BarisBarang()) == 0 {
		return nil
	}
	return s.mulaiKartuOtomatisLocked(t.Tanggal)
}
