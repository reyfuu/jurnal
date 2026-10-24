package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"pajak/internal/finance"
)

// Impor mutasi di perpetual: baris Penjualan / Beli Bahan / Stok membawa barang atau tanda jasa.

func commitMutasiRows(t *testing.T, srv *Server, qa *http.Cookie, rows ...finance.MutasiRow) (int, mutasiCommitResp, string) {
	t.Helper()
	b, _ := json.Marshal(mutasiCommitReq{Rows: rows})
	rec := do(srv, qa, http.MethodPost, "/api/v1/transactions/mutasi/commit", string(b))
	var res mutasiCommitResp
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	return rec.Code, res, strings.TrimSpace(rec.Body.String())
}

func barisJual(baris int, tanggal string, nominal int64, rincian ...finance.RincianBarang) finance.MutasiRow {
	return finance.MutasiRow{Baris: baris, Tanggal: tanggal, Jenis: "Penjualan", Nominal: nominal, Arah: "masuk", Pilih: true, Rincian: rincian}
}

func TestMutasi_PerpetualTanpaBarangGalatPerBaris(t *testing.T) {
	srv, qa, kaos, kain := perpetualKaos(t, 10000, 0)
	jasa := barisJual(4, "2026-01-12", 30000)
	jasa.BukanBarang = true
	code, res, body := commitMutasiRows(t, srv, qa,
		barisJual(2, "2026-01-10", 100000),
		barisJual(3, "2026-01-11", 100000, finance.RincianBarang{StokID: kaos.ID, QtyStok: 2000}),
		jasa,
		finance.MutasiRow{Baris: 5, Tanggal: "2026-01-12", Jenis: "Beli Bahan / Stok", Nominal: 50000, Arah: "keluar", Pilih: true,
			Rincian: []finance.RincianBarang{{StokID: kain.ID, QtyStok: 5000}}},
		finance.MutasiRow{Baris: 6, Tanggal: "2026-01-13", Jenis: "Biaya Lain", Nominal: 10000, Arah: "keluar", Pilih: true},
		barisJual(7, "2026-01-13", 1000, finance.RincianBarang{StokID: 999, QtyStok: 1000}),
	)
	if code != http.StatusOK || res.Disimpan != 4 || len(res.Errors) != 2 {
		t.Fatalf("commit: %d %s", code, body)
	}
	if e := res.Errors[0]; e.Baris != 2 || e.Pesan != errWajibBarang.Error() {
		t.Errorf("galat baris tanpa barang: %+v", e)
	}
	if e := res.Errors[1]; e.Baris != 7 || e.Pesan != "barang yang dipilih tidak ada di Stok & Gudang" {
		t.Errorf("galat barang tak dikenal: %+v", e)
	}
	got := txsOf(t, srv, qa)
	if len(got) != 4 {
		t.Fatalf("tersimpan %d", len(got))
	}
	for _, tx := range got {
		if tx.Jenis == "Penjualan" && tx.Nominal == 100000 && (len(tx.BarisBarang()) != 1 || tx.BarisBarang()[0].StokID != kaos.ID) {
			t.Errorf("penjualan berbarang: %+v", tx)
		}
		if tx.Jenis == "Penjualan" && tx.Nominal == 30000 && !tx.BukanBarang {
			t.Errorf("jasa: %+v", tx)
		}
	}
	kartu := getJSON[kartuBarangResp](t, srv, qa, "/api/v1/stok/kartu/"+itoa(kaos.ID)+"?dari=2026-01-01&sampai=2026-01-31")
	if n := len(kartu.Baris); n == 0 || kartu.Baris[n-1].SaldoQty != 8000 {
		t.Errorf("saldo kaos sesudah jual 2: %+v", kartu.Baris)
	}
	// Semua baris gagal: 400 dengan alasannya, bukan "berhasil 0".
	code, _, body = commitMutasiRows(t, srv, qa, barisJual(9, "2026-01-20", 5000))
	if code != http.StatusBadRequest || !strings.Contains(body, "baris 9: "+errWajibBarang.Error()) {
		t.Errorf("semua gagal: %d %s", code, body)
	}
}

// Seluruh kiriman dicek stok tidak minus sekaligus; gagal = 409 dan tidak ada yang tersimpan
// (termasuk baris tanpa barang dan pelunasan), supaya urutan per baris tidak membingungkan.
func TestMutasi_MelebihiStokDitolak(t *testing.T) {
	srv, qa, kaos, _ := perpetualKaos(t, 5000, 0)
	code, _, body := commitMutasiRows(t, srv, qa,
		barisJual(2, "2026-01-10", 100000, finance.RincianBarang{StokID: kaos.ID, QtyStok: 3000}),
		barisJual(3, "2026-01-11", 100000, finance.RincianBarang{StokID: kaos.ID, QtyStok: 3000}),
		finance.MutasiRow{Baris: 4, Tanggal: "2026-01-12", Jenis: "Biaya Lain", Nominal: 10000, Arah: "keluar", Pilih: true},
	)
	cekMinus(t, code, body, "stok Kaos tidak cukup pada 11 Jan 2026: tersedia 2 pcs, dibutuhkan 3 pcs (baris 3)")
	if n := len(txsOf(t, srv, qa)); n != 0 {
		t.Fatalf("commit tertolak tetap menyimpan %d transaksi", n)
	}
}

// Kartu belum dimulai: baris berbarang paling awal memulainya dari daftar barang. Stok awal
// otomatis hanya tersimpan bila ada baris berbarang yang tersimpan.
func TestMutasi_KartuDimulaiOtomatis(t *testing.T) {
	srv, qa := usahaManufakturPerpetual(t)
	kaos := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kaos","kategori":"Produk Jadi","satuan":"pcs","jumlah":10,"harga_pokok":50000}`)
	jasa := barisJual(2, "2026-02-01", 30000)
	jasa.BukanBarang = true
	if code, res, body := commitMutasiRows(t, srv, qa, jasa); code != http.StatusOK || res.Disimpan != 1 {
		t.Fatalf("jasa: %d %s", code, body)
	}
	if n := jumlahGerakAwal(t, srv, qa); n != 0 {
		t.Fatalf("baris jasa saja tidak boleh memulai kartu (%d)", n)
	}
	code, res, body := commitMutasiRows(t, srv, qa,
		barisJual(3, "2026-02-10", 100000, finance.RincianBarang{StokID: kaos.ID, QtyStok: 4000}),
		barisJual(4, "2026-02-05", 100000, finance.RincianBarang{StokID: kaos.ID, QtyStok: 5000}),
	)
	if code != http.StatusOK || res.Disimpan != 2 {
		t.Fatalf("commit: %d %s", code, body)
	}
	gerak := getJSON[[]finance.GerakStok](t, srv, qa, "/api/v1/stok/gerak")
	if len(gerak) != 1 || gerak[0].Jenis != finance.GerakAwal || gerak[0].Tanggal != "2026-02-05" {
		t.Errorf("kartu harus mulai otomatis di tanggal baris berbarang paling awal: %+v", gerak)
	}
	// Daftar barang hanya 10: kartu baru dengan baris yang melebihi stok ditolak, stok awal tidak tersimpan.
	srv2, qa2 := usahaManufakturPerpetual(t)
	kaos2 := buat[finance.StokItem](t, srv2, qa2, "/api/v1/stok", `{"nama":"Kaos","kategori":"Produk Jadi","satuan":"pcs","jumlah":2}`)
	code, _, body = commitMutasiRows(t, srv2, qa2, barisJual(2, "2026-01-10", 100000, finance.RincianBarang{StokID: kaos2.ID, QtyStok: 3000}))
	cekMinus(t, code, body, "stok Kaos tidak cukup pada 10 Jan 2026: tersedia 2 pcs, dibutuhkan 3 pcs (baris 2)")
	if n := jumlahGerakAwal(t, srv2, qa2); n != 0 {
		t.Errorf("stok awal otomatis tersimpan padahal commit ditolak (%d)", n)
	}
}

// Baris yang diduga sudah tercatat dan dilewati tidak dibukukan dan tidak mengubah kartu stok;
// bila dipaksa "tetap catat sebagai transaksi baru" ia juga wajib barang.
func TestMutasi_DuplikatLewatiTidakMengubahKartu(t *testing.T) {
	srv, qa, kaos, _ := perpetualKaos(t, 10000, 0)
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", jualBody("2026-01-10", kaos.ID, 3000))
	sebelum := getJSON[kartuBarangResp](t, srv, qa, "/api/v1/stok/kartu/"+itoa(kaos.ID)+"?dari=2026-01-01&sampai=2026-01-31")
	h := unggahMutasi(t, srv, qa, "Tanggal;Keterangan;Debet;Kredit\n10/01/2026;TRSF CR TOKO A;;100.000\n12/01/2026;BIAYA ADM;10.000;\n")
	if len(h.Rows) != 2 || !h.Rows[0].Duplikat || h.Rows[0].Pilih {
		t.Fatalf("baris pertama harus diduga sudah tercatat: %+v", h.Rows)
	}
	code, res, body := commitMutasiRows(t, srv, qa, h.Rows...)
	if code != http.StatusOK || res.Disimpan != 1 || len(res.Errors) != 0 {
		t.Fatalf("commit: %d %s", code, body)
	}
	sesudah := getJSON[kartuBarangResp](t, srv, qa, "/api/v1/stok/kartu/"+itoa(kaos.ID)+"?dari=2026-01-01&sampai=2026-01-31")
	if len(sesudah.Baris) != len(sebelum.Baris) || sesudah.Baris[len(sesudah.Baris)-1].SaldoQty != 7000 {
		t.Errorf("kartu berubah: %+v → %+v", sebelum.Baris, sesudah.Baris)
	}
	if n := len(txsOf(t, srv, qa)); n != 2 {
		t.Errorf("tersimpan %d transaksi, harus 2", n)
	}
	baru := h.Rows[0]
	baru.Pilih = true
	code, _, body = commitMutasiRows(t, srv, qa, baru)
	if code != http.StatusBadRequest || !strings.Contains(body, errWajibBarang.Error()) {
		t.Errorf("duplikat dicatat baru tanpa barang: %d %s", code, body)
	}
}

// Galat dari pelunasan (diproses sesudah wajib barang) tetap terurut per baris di respons.
func TestMutasi_GalatUrutPerBaris(t *testing.T) {
	srv, qa, _, _ := perpetualKaos(t, 10000, 0)
	code, res, body := commitMutasiRows(t, srv, qa,
		finance.MutasiRow{Baris: 2, Tanggal: "2026-01-10", Jenis: "Penjualan", Nominal: 5000, Arah: "masuk", Pilih: true, Lunasi: 99},
		barisJual(3, "2026-01-11", 1000),
		finance.MutasiRow{Baris: 4, Tanggal: "2026-01-12", Jenis: "Biaya Lain", Nominal: 10000, Arah: "keluar", Pilih: true},
	)
	if code != http.StatusOK || res.Disimpan != 1 || len(res.Errors) != 2 || res.Errors[0].Baris != 2 || res.Errors[1].Baris != 3 {
		t.Fatalf("commit: %d %s", code, body)
	}
}
