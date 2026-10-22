package finance_test

import (
	"errors"
	"reflect"
	"testing"

	"pajak/internal/finance"
)

func txsKoreksi() []finance.Transaction {
	return []finance.Transaction{
		{ID: 1, Tanggal: "2026-10-03", Jenis: "Penjualan", Nominal: 10_000_000, Keterangan: "Penjualan kasir", Lunas: true},
		{ID: 2, Tanggal: "2026-10-13", Jenis: "Perawatan & Perlengkapan", Nominal: 4_500_000, Keterangan: "Beli mesin espresso", Lunas: true},
		{ID: 3, Tanggal: "2026-10-15", Jenis: "Penjualan", Nominal: 1_250_000, Keterangan: "GoFood 15 Okt", Lunas: true},
		{ID: 4, Tanggal: "2026-10-15", Jenis: "Penjualan", Nominal: 1_250_000, Keterangan: "GoFood 15 Okt", Lunas: true},
	}
}

func ptrTx(t finance.Transaction) *finance.Transaction { return &t }

func TestSiapkanKoreksi_Ubah(t *testing.T) {
	txs := txsKoreksi()
	usul := txs[1]
	usul.Jenis = "Beli Alat / Aset"
	k := finance.Koreksi{TransaksiID: 2, Jenis: finance.KoreksiUbah, Usulan: &usul, Alasan: " Dipakai lebih dari setahun "}
	if err := finance.SiapkanKoreksi(&k, txs); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(k.Sebelum, txs[1]) || k.Status != finance.KoreksiMenunggu || k.Alasan != "Dipakai lebih dari setahun" || k.Usulan.ID != 2 {
		t.Fatalf("koreksi = %+v", k)
	}
}

func TestSiapkanKoreksi_Tolak(t *testing.T) {
	txs := txsKoreksi()
	sama := txs[0]
	salahJenis := txs[0]
	salahJenis.Jenis = "Tidak Ada"
	cases := map[string]finance.Koreksi{
		"transaksi tidak ada": {TransaksiID: 99, Jenis: finance.KoreksiHapus, Alasan: "x"},
		"tanpa alasan":        {TransaksiID: 1, Jenis: finance.KoreksiHapus, Alasan: "  "},
		"jenis tak dikenal":   {TransaksiID: 1, Jenis: "ganti", Alasan: "x"},
		"ubah tanpa usulan":   {TransaksiID: 1, Jenis: finance.KoreksiUbah, Alasan: "x"},
		"ubah tidak berubah":  {TransaksiID: 1, Jenis: finance.KoreksiUbah, Usulan: &sama, Alasan: "x"},
		"usulan tidak valid":  {TransaksiID: 1, Jenis: finance.KoreksiUbah, Usulan: &salahJenis, Alasan: "x"},
	}
	for nama, k := range cases {
		if err := finance.SiapkanKoreksi(&k, txs); err == nil {
			t.Errorf("%s: harus ditolak", nama)
		}
	}
}

func TestTerapkanKoreksi_UbahDanHapus(t *testing.T) {
	txs := txsKoreksi()
	usul := txs[1]
	usul.Jenis = "Beli Alat / Aset"
	ubah := finance.Koreksi{TransaksiID: 2, Jenis: finance.KoreksiUbah, Usulan: &usul, Alasan: "aset"}
	_ = finance.SiapkanKoreksi(&ubah, txs)
	got, err := finance.TerapkanKoreksi(txs, ubah)
	if err != nil {
		t.Fatal(err)
	}
	if got[1].Jenis != "Beli Alat / Aset" || got[1].ID != 2 || txs[1].Jenis != "Perawatan & Perlengkapan" {
		t.Fatalf("ubah: %+v (asli %+v)", got[1], txs[1])
	}
	hapus := finance.Koreksi{TransaksiID: 4, Jenis: finance.KoreksiHapus, Alasan: "dobel"}
	_ = finance.SiapkanKoreksi(&hapus, got)
	got2, err := finance.TerapkanKoreksi(got, hapus)
	if err != nil || len(got2) != 3 || got2[2].ID != 3 {
		t.Fatalf("hapus: %+v %v", got2, err)
	}
}

// Transaksi yang sudah diubah pemilik setelah usulan dibuat tidak boleh ditimpa diam-diam.
func TestTerapkanKoreksi_BasiBilaTransaksiBerubah(t *testing.T) {
	txs := txsKoreksi()
	k := finance.Koreksi{TransaksiID: 3, Jenis: finance.KoreksiHapus, Alasan: "dobel"}
	_ = finance.SiapkanKoreksi(&k, txs)
	txs[2].Nominal = 1_300_000
	if _, err := finance.TerapkanKoreksi(txs, k); !errors.Is(err, finance.ErrKoreksiBasi) {
		t.Errorf("transaksi berubah: err = %v", err)
	}
	if _, err := finance.TerapkanKoreksi(txs[:2], k); !errors.Is(err, finance.ErrKoreksiBasi) {
		t.Errorf("transaksi sudah dihapus: err = %v", err)
	}
}

func TestHitungDampak(t *testing.T) {
	txs := txsKoreksi()
	// Biaya 4,5 jt dipindah ke aset: laba Oktober naik 4,5 jt (penyusutan dari register aset terpisah).
	usul := txs[1]
	usul.Jenis = "Beli Alat / Aset"
	d := finance.HitungDampak(txs, finance.Koreksi{TransaksiID: 2, Jenis: finance.KoreksiUbah, Sebelum: txs[1], Usulan: &usul})
	if len(d) != 1 || d[0].Periode != "2026-10" || d[0].LabaSesudah-d[0].LabaSebelum != 4_500_000 {
		t.Fatalf("dampak ubah = %+v", d)
	}
	// Hapus penjualan dobel: laba turun 1,25 jt.
	d = finance.HitungDampak(txs, finance.Koreksi{TransaksiID: 4, Jenis: finance.KoreksiHapus, Sebelum: txs[3]})
	if len(d) != 1 || d[0].LabaSesudah-d[0].LabaSebelum != -1_250_000 {
		t.Fatalf("dampak hapus = %+v", d)
	}
	// Pindah bulan: dua periode terdampak, urut.
	pindah := txs[0]
	pindah.Tanggal = "2026-09-30"
	d = finance.HitungDampak(txs, finance.Koreksi{TransaksiID: 1, Jenis: finance.KoreksiUbah, Sebelum: txs[0], Usulan: &pindah})
	if len(d) != 2 || d[0].Periode != "2026-09" || d[1].Periode != "2026-10" || d[0].LabaSesudah-d[0].LabaSebelum != 10_000_000 {
		t.Fatalf("dampak pindah bulan = %+v", d)
	}
}
