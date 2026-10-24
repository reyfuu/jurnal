package main

import (
	"fmt"
	"slices"
	"strings"

	"pajak/internal/finance"
)

// Impor Excel dengan barang (persediaan per akun tahap 2): satu baris Excel = satu barang.
// Kolom barang berisi SKU atau nama persis (tanpa beda huruf besar), atau "jasa" = Bukan barang;
// jumlah_barang desimal dibaca jadi per seribu satuan. Baris yang gagal dicocokkan dilaporkan
// per baris dan dilewati, sama dengan galat baris lain; stok minus menolak seluruh impor.

func rapikanNama(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }

// cariBarangImporLocked: SKU persis dulu, lalu nama persis. Nama kembar → galat. s.mu dipegang.
func (s *tenant) cariBarangImporLocked(teks string) (int, error) {
	q := rapikanNama(teks)
	for _, it := range s.stok {
		if it.SKU != "" && rapikanNama(it.SKU) == q {
			return it.ID, nil
		}
	}
	id, n := 0, 0
	for _, it := range s.stok {
		if rapikanNama(it.Nama) == q {
			id, n = it.ID, n+1
		}
	}
	switch {
	case n == 0:
		return 0, fmt.Errorf("barang %q tidak ada di daftar barang", teks)
	case n > 1:
		return 0, fmt.Errorf("barang %q ada lebih dari satu di daftar barang; pakai SKU-nya", teks)
	}
	return id, nil
}

// terapkanBarangImporLocked: tempel barang tiap baris, validasi ulang, dan terapkan wajib barang.
// Baris bermasalah masuk errs (tidak ikut). s.mu dipegang.
func (s *tenant) terapkanBarangImporLocked(baris []finance.ImportRow) ([]finance.Transaction, []finance.ImportError) {
	var txs []finance.Transaction
	var errs []finance.ImportError
	for _, r := range baris {
		t := r.Tx
		gagal := func(err error) { errs = append(errs, finance.ImportError{Baris: r.Baris, Pesan: err.Error()}) }
		// Periodik: barang opsional, jadi yang tak bisa dicocokkan dilewati tanpa galat.
		longgar := !s.perpetualLocked()
		bisaBarang := t.Jenis == finance.JenisJual || t.Jenis == finance.JenisBeliStok
		switch {
		case longgar && !bisaBarang:
		case finance.JasaTeks(r.BarangTeks):
			t.BukanBarang = true
		case r.BarangTeks != "":
			id, err := s.cariBarangImporLocked(r.BarangTeks)
			if err != nil {
				if longgar {
					break
				}
				gagal(err)
				continue
			}
			if !r.AdaJumlah && longgar {
				break
			}
			t.Rincian = []finance.RincianBarang{{StokID: id, QtyStok: r.JumlahBarang, Nilai: t.Nominal}}
		case r.AdaJumlah && !longgar:
			gagal(fmt.Errorf("jumlah barang diisi tetapi kolom barang kosong"))
			continue
		}
		if err := finance.Validate(&t); err != nil {
			gagal(err)
			continue
		}
		if err := s.cekBarangLocked(t); err != nil {
			gagal(err)
			continue
		}
		txs = append(txs, t)
	}
	return txs, errs
}

// cekStokImporLocked: stok tidak minus pada seluruh kumpulan impor sekaligus. Kartu yang belum
// dimulai dimulai otomatis di tanggal transaksi berbarang paling awal (dikembalikan; nil bila
// tidak perlu). Dasar pembanding = sekarang + stok awal itu, tanpa transaksi impor. s.mu dipegang.
func (s *tenant) cekStokImporLocked(baru []finance.Transaction) (*finance.GerakStok, error) {
	awalTgl := ""
	for _, t := range baru {
		if len(t.BarisBarang()) > 0 && (awalTgl == "" || t.Tanggal < awalTgl) {
			awalTgl = t.Tanggal
		}
	}
	var awal *finance.GerakStok
	if awalTgl != "" {
		awal = s.mulaiKartuOtomatisLocked(awalTgl)
	}
	txs := slices.Clone(s.txs)
	id := s.nextTxID
	for _, t := range baru {
		t.ID = id
		id++
		txs = append(txs, t)
	}
	gerak := s.tambahGerakUsulan(awal)
	return awal, s.cekTanpaMinusDariLocked(s.txs, gerak, txs, gerak)
}
