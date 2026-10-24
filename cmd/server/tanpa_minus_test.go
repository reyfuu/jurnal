package main

import (
	"net/http"
	"strings"
	"testing"

	"pajak/internal/finance"
)

// Task 4 persediaan per akun tahap 2: di perpetual stok tidak boleh minus. Perubahan yang
// membuat (atau memperdalam) minus di kartu stok ditolak 409; minus lama tidak memblokir.

func cekMinus(t *testing.T, code int, body, pesan string) {
	t.Helper()
	if code != http.StatusConflict || strings.TrimSpace(body) != pesan {
		t.Fatalf("harus 409 %q, dapat %d %q", pesan, code, body)
	}
}

// perpetualKaos: usaha perpetual, barang Kaos (pcs) & Kain (m), stok awal 2026-01-01.
func perpetualKaos(t *testing.T, fisikKaos, fisikKain int64) (*Server, *http.Cookie, finance.StokItem, finance.StokItem) {
	t.Helper()
	srv, qa := usahaManufakturPerpetual(t)
	kaos := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kaos","kategori":"Produk Jadi","satuan":"pcs"}`)
	kain := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kain","kategori":"Bahan Baku","satuan":"m"}`)
	buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/awal", `{"tanggal":"2026-01-01","baris":[`+
		`{"stok_id":`+itoa(kaos.ID)+`,"fisik":`+itoa(int(fisikKaos))+`,"harga_unit":50000},`+
		`{"stok_id":`+itoa(kain.ID)+`,"fisik":`+itoa(int(fisikKain))+`,"harga_unit":10000}]}`)
	return srv, qa, kaos, kain
}

func jualBody(tanggal string, stokID int, qty int64) string {
	return `{"tanggal":"` + tanggal + `","jenis":"Penjualan","nominal":100000,"lunas":true,"rincian":[{"stok_id":` + itoa(stokID) + `,"qty_stok":` + itoa(int(qty)) + `}]}`
}

func TestTanpaMinus_PenjualanMelebihiStokDitolak(t *testing.T) {
	srv, qa, kaos, _ := perpetualKaos(t, 5000, 0)
	rec := do(srv, qa, http.MethodPost, "/api/v1/transactions", jualBody("2026-01-10", kaos.ID, 6000))
	cekMinus(t, rec.Code, rec.Body.String(), "stok Kaos tidak cukup pada 10 Jan 2026: tersedia 5 pcs, dibutuhkan 6 pcs")
	if n := len(txsOf(t, srv, qa)); n != 0 {
		t.Fatalf("transaksi tertolak tetap tersimpan (%d)", n)
	}
	// Pas dengan stok boleh; mengubahnya jadi melebihi stok ditolak dan tidak mengubah apa pun.
	tx := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", jualBody("2026-01-10", kaos.ID, 5000))
	rec = do(srv, qa, http.MethodPut, "/api/v1/transactions/"+itoa(tx.ID), jualBody("2026-01-10", kaos.ID, 7500))
	cekMinus(t, rec.Code, rec.Body.String(), "stok Kaos tidak cukup pada 10 Jan 2026: tersedia 5 pcs, dibutuhkan 7,5 pcs")
	if got := txsOf(t, srv, qa); got[0].BarisBarang()[0].QtyStok != 5000 {
		t.Errorf("ubah tertolak tetap mengubah transaksi: %+v", got[0])
	}
}

func TestTanpaMinus_HapusPembelianYangDipakaiDitolak(t *testing.T) {
	srv, qa, kaos, _ := perpetualKaos(t, 0, 0)
	beli := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions",
		`{"tanggal":"2026-01-05","jenis":"Beli Bahan / Stok","nominal":250000,"lunas":true,"rincian":[{"stok_id":`+itoa(kaos.ID)+`,"qty_stok":5000}]}`)
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", jualBody("2026-01-10", kaos.ID, 4000))
	rec := do(srv, qa, http.MethodDelete, "/api/v1/transactions/"+itoa(beli.ID), "")
	cekMinus(t, rec.Code, rec.Body.String(), "stok Kaos tidak cukup pada 10 Jan 2026: tersedia 0 pcs, dibutuhkan 4 pcs")
	if n := len(txsOf(t, srv, qa)); n != 2 {
		t.Errorf("hapus tertolak tetap menghapus (%d transaksi)", n)
	}
	// Opname yang menurunkan stok sebelum penjualan juga ditolak.
	rec = do(srv, qa, http.MethodPost, "/api/v1/stok/opname", `{"tanggal":"2026-01-06","baris":[{"stok_id":`+itoa(kaos.ID)+`,"fisik":3000}]}`)
	cekMinus(t, rec.Code, rec.Body.String(), "stok Kaos tidak cukup pada 10 Jan 2026: tersedia 3 pcs, dibutuhkan 4 pcs")
}

func TestTanpaMinus_ProduksiBahanKurangDitolak(t *testing.T) {
	srv, qa, _, kain := perpetualKaos(t, 0, 2000)
	produk := buat[finance.Produk](t, srv, qa, "/api/v1/produksi/produk",
		`{"sku":"KS-9","nama":"Kemeja","satuan":"pcs","hasil_per_batch":1,"bahan":[{"stok_id":`+itoa(kain.ID)+`,"nama":"Kain","satuan":"m","takaran":1000}]}`)
	tn := tenantByEmail(t, srv, qa)
	jumlahBarang := func() int { tn.mu.RLock(); defer tn.mu.RUnlock(); return len(tn.stok) }
	sebelum := jumlahBarang()
	pesan := "stok Kain tidak cukup pada 06 Jan 2026: tersedia 2 m, dibutuhkan 3 m"
	rec := do(srv, qa, http.MethodPost, "/api/v1/stok/produksi", `{"tanggal":"2026-01-06","produk_id":`+itoa(produk.ID)+`,"batch":3}`)
	cekMinus(t, rec.Code, rec.Body.String(), pesan)
	rec = do(srv, qa, http.MethodPost, "/api/v1/stok/produksi/catat", `{"tanggal":"2026-01-06","produk":[{"produk_id":`+itoa(produk.ID)+`,"hasil":3000}]}`)
	cekMinus(t, rec.Code, rec.Body.String(), pesan)
	if got := jumlahBarang(); got != sebelum {
		t.Errorf("produksi tertolak tetap membuat barang jadi (%d → %d)", sebelum, got)
	}
	// Produksi yang cukup boleh; menghapus pembelian bahan yang dipakainya lalu ditolak lewat hapus dokumen opname.
	buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/opname", `{"tanggal":"2026-01-03","baris":[{"stok_id":`+itoa(kain.ID)+`,"fisik":5000}]}`)
	buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/produksi", `{"tanggal":"2026-01-06","produk_id":`+itoa(produk.ID)+`,"batch":3}`)
	var opnameID int
	tn.mu.RLock()
	for _, g := range tn.gerak {
		if g.Jenis == finance.GerakOpname {
			opnameID = g.ID
		}
	}
	tn.mu.RUnlock()
	rec = do(srv, qa, http.MethodDelete, "/api/v1/stok/gerak/"+itoa(opnameID), "")
	cekMinus(t, rec.Code, rec.Body.String(), pesan)
}

func TestTanpaMinus_MinusLamaTidakMemblokir(t *testing.T) {
	srv, qa, kaos, kain := perpetualKaos(t, 5000, 10000)
	// Data lama (sebelum aturan): Kain dijual 20 m padahal stok 10 m.
	seedTx(t, srv, qa, finance.Transaction{Tanggal: "2026-01-05", Jenis: finance.JenisJual, Nominal: 100000, Lunas: true,
		Rincian: []finance.RincianBarang{{StokID: kain.ID, QtyStok: 20000}}})
	gaji := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-01-08","jenis":"Gaji & Upah","nominal":100000,"lunas":true}`)
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", jualBody("2026-01-10", kaos.ID, 3000))
	if rec := do(srv, qa, http.MethodPut, "/api/v1/transactions/"+itoa(gaji.ID), `{"tanggal":"2026-01-09","jenis":"Gaji & Upah","nominal":120000,"lunas":true}`); rec.Code != http.StatusOK {
		t.Fatalf("ubah transaksi lain saat ada minus lama: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, qa, http.MethodDelete, "/api/v1/transactions/"+itoa(gaji.ID), ""); rec.Code != http.StatusNoContent {
		t.Fatalf("hapus transaksi lain saat ada minus lama: %d %s", rec.Code, rec.Body)
	}
	// Memperdalam minus lama tetap ditolak.
	rec := do(srv, qa, http.MethodPost, "/api/v1/transactions", jualBody("2026-01-12", kain.ID, 1000))
	cekMinus(t, rec.Code, rec.Body.String(), "stok Kain tidak cukup pada 12 Jan 2026: tersedia -10 m, dibutuhkan 1 m")
}

func TestTanpaMinus_PeriodikBoleh(t *testing.T) {
	srv, qa := usahaManufaktur(t)
	kaos := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kaos","kategori":"Produk Jadi","satuan":"pcs"}`)
	buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/awal", `{"tanggal":"2026-01-01","baris":[{"stok_id":`+itoa(kaos.ID)+`,"fisik":5000,"harga_unit":50000}]}`)
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", jualBody("2026-01-10", kaos.ID, 6000))
}

// Kartu belum dimulai: transaksi berbarang pertama memulai kartu dari daftar barang; stok di
// daftar yang kurang tetap ditolak dan stok awal otomatis tidak tersimpan.
func TestTanpaMinus_MulaiOtomatisDicek(t *testing.T) {
	srv, qa := usahaManufakturPerpetual(t)
	kaos := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kaos","kategori":"Produk Jadi","satuan":"pcs","jumlah":2}`)
	rec := do(srv, qa, http.MethodPost, "/api/v1/transactions", jualBody("2026-01-10", kaos.ID, 3000))
	cekMinus(t, rec.Code, rec.Body.String(), "stok Kaos tidak cukup pada 10 Jan 2026: tersedia 2 pcs, dibutuhkan 3 pcs")
	if n := jumlahGerakAwal(t, srv, qa); n != 0 {
		t.Errorf("stok awal otomatis tersimpan padahal transaksi ditolak (%d)", n)
	}
}

// Usulan koreksi yang disetujui pemilik juga tidak boleh membuat stok minus.
func TestTanpaMinus_KoreksiDisetujuiDicek(t *testing.T) {
	srv, _, qa, k := siapKlien(t)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/business", manufakturPerpetual); rec.Code != http.StatusOK {
		t.Fatalf("profil perpetual: %d %s", rec.Code, rec.Body)
	}
	kain := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kain","kategori":"Bahan Baku","satuan":"m"}`)
	usul(t, srv, k, `{"transaksi_id":1,"jenis":"ubah","alasan":"Sebenarnya jual kain","usulan":{"tanggal":"2026-10-13","jenis":"Penjualan","nominal":4500000,"keterangan":"Jual kain","lunas":true,"rincian":[{"stok_id":`+itoa(kain.ID)+`,"qty_stok":10000}]}}`)
	rec := do(srv, qa, http.MethodPost, "/api/v1/koreksi/1/setujui", `{}`)
	cekMinus(t, rec.Code, rec.Body.String(), "stok Kain tidak cukup pada 13 Okt 2026: tersedia 0 m, dibutuhkan 10 m")
	if n := jumlahGerakAwal(t, srv, qa); n != 0 {
		t.Errorf("stok awal otomatis tersimpan padahal usulan ditolak (%d)", n)
	}
}

// Fix round 1: kartu dimulai otomatis menarik transaksi lama ke kartu; minus lama barang lain
// (sudah ada sebelum perubahan ini) tidak boleh memblokir penjualan yang stoknya cukup.
func TestTanpaMinus_MulaiOtomatisMinusLamaBarangLainTidakMemblokir(t *testing.T) {
	srv, qa := usahaManufakturPerpetual(t)
	kaos := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kaos","kategori":"Produk Jadi","satuan":"pcs","jumlah":5}`)
	kain := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kain","kategori":"Bahan Baku","satuan":"m"}`)
	seedTx(t, srv, qa, finance.Transaction{Tanggal: "2026-01-20", Jenis: finance.JenisJual, Nominal: 100000, Lunas: true,
		Rincian: []finance.RincianBarang{{StokID: kain.ID, QtyStok: 1000}}})
	rec := do(srv, qa, http.MethodPost, "/api/v1/transactions", jualBody("2026-01-10", kaos.ID, 1000))
	if rec.Code != http.StatusCreated {
		t.Fatalf("penjualan Kaos dengan minus lama Kain: %d %s", rec.Code, rec.Body)
	}
}

// Fix round 1: memindah penjualan lama yang sudah minus tanpa memperdalam minusnya boleh, baik ke
// hari tanpa baris lain maupun ke hari yang punya baris barang itu; memindahnya ke sebelum
// pembelian (minus lebih dalam) ditolak.
func TestTanpaMinus_PindahPenjualanMinusLama(t *testing.T) {
	srv, qa, _, kain := perpetualKaos(t, 0, 10000)
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions",
		`{"tanggal":"2026-01-03","jenis":"Beli Bahan / Stok","nominal":20000,"lunas":true,"rincian":[{"stok_id":`+itoa(kain.ID)+`,"qty_stok":2000}]}`)
	lama := seedTx(t, srv, qa, finance.Transaction{Tanggal: "2026-01-05", Jenis: finance.JenisJual, Nominal: 100000, Lunas: true,
		Rincian: []finance.RincianBarang{{StokID: kain.ID, QtyStok: 20000}}})
	url := "/api/v1/transactions/" + itoa(lama.ID)
	for _, tgl := range []string{"2026-01-04", "2026-01-03"} {
		if rec := do(srv, qa, http.MethodPut, url, jualBody(tgl, kain.ID, 20000)); rec.Code != http.StatusOK {
			t.Fatalf("pindah penjualan lama ke %s: %d %s", tgl, rec.Code, rec.Body)
		}
	}
	rec := do(srv, qa, http.MethodPut, url, jualBody("2026-01-02", kain.ID, 20000))
	cekMinus(t, rec.Code, rec.Body.String(), "stok Kain tidak cukup pada 02 Jan 2026: tersedia 10 m, dibutuhkan 20 m")
}

// Fix round 1: baris barang masuk yang saldonya turun jadi (lebih) minus memakai pesan saldo.
func TestTanpaMinus_BarisMasukPesanSaldo(t *testing.T) {
	srv, qa, _, kain := perpetualKaos(t, 0, 10000)
	seedTx(t, srv, qa, finance.Transaction{Tanggal: "2026-01-05", Jenis: finance.JenisJual, Nominal: 100000, Lunas: true,
		Rincian: []finance.RincianBarang{{StokID: kain.ID, QtyStok: 20000}}})
	beli := `{"tanggal":"2026-01-08","jenis":"Beli Bahan / Stok","nominal":50000,"lunas":true,"rincian":[{"stok_id":` + itoa(kain.ID) + `,"qty_stok":%d}]}`
	tx := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", strings.Replace(beli, "%d", "5000", 1))
	rec := do(srv, qa, http.MethodPut, "/api/v1/transactions/"+itoa(tx.ID), strings.Replace(beli, "%d", "3000", 1))
	cekMinus(t, rec.Code, rec.Body.String(), "stok Kain jadi minus pada 08 Jan 2026: saldo -7 m")
}

// Fix round 1: barang yang sama di beberapa baris satu transaksi dilaporkan dengan total kebutuhannya.
func TestTanpaMinus_BarisSamaDijumlah(t *testing.T) {
	srv, qa, kaos, _ := perpetualKaos(t, 5000, 0)
	rec := do(srv, qa, http.MethodPost, "/api/v1/transactions",
		`{"tanggal":"2026-01-10","jenis":"Penjualan","nominal":100000,"lunas":true,"rincian":[{"stok_id":`+itoa(kaos.ID)+`,"qty_stok":3000,"nilai":50000},{"stok_id":`+itoa(kaos.ID)+`,"qty_stok":3000,"nilai":50000}]}`)
	cekMinus(t, rec.Code, rec.Body.String(), "stok Kaos tidak cukup pada 10 Jan 2026: tersedia 5 pcs, dibutuhkan 6 pcs")
}
