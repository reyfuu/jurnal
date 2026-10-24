package main

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"pajak/internal/finance"
)

// Tahap 2 persediaan per akun: di perpetual, Penjualan & Beli Bahan / Stok wajib membawa barang
// (atau ditandai Bukan barang (jasa)). Kartu stok yang belum dimulai dimulai otomatis oleh
// transaksi berbarang pertama.

const pesanWajibBarang = "pilih barang & jumlahnya, atau tandai Bukan barang (jasa)"

// perpetualKartuMulai: usaha perpetual dengan barang Kain dan kartu stok mulai 2026-01-01.
func perpetualKartuMulai(t *testing.T, mulai string) (*Server, *http.Cookie, finance.StokItem) {
	t.Helper()
	srv, qa := usahaManufakturPerpetual(t)
	kain := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kain","kategori":"Bahan Baku","satuan":"m"}`)
	buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/awal", `{"tanggal":"`+mulai+`","baris":[{"stok_id":`+itoa(kain.ID)+`,"fisik":10000,"harga_unit":10000}]}`)
	return srv, qa, kain
}

func harusDitolakWajibBarang(t *testing.T, code int, body string) {
	t.Helper()
	if code != http.StatusBadRequest || strings.TrimSpace(body) != pesanWajibBarang {
		t.Fatalf("harus 400 %q, dapat %d %q", pesanWajibBarang, code, body)
	}
}

// seedTx menaruh transaksi langsung di penyimpanan (data lama sebelum aturan wajib barang).
func seedTx(t *testing.T, srv *Server, qa *http.Cookie, tx finance.Transaction) finance.Transaction {
	t.Helper()
	tn := tenantByEmail(t, srv, qa)
	tn.mu.Lock()
	defer tn.mu.Unlock()
	tx.ID = tn.nextTxID
	tn.nextTxID++
	tn.txs = append(tn.txs, tx)
	return tx
}

func jumlahGerakAwal(t *testing.T, srv *Server, qa *http.Cookie) int {
	t.Helper()
	tn := tenantByEmail(t, srv, qa)
	tn.mu.RLock()
	defer tn.mu.RUnlock()
	n := 0
	for _, g := range tn.gerak {
		if g.Jenis == finance.GerakAwal {
			n++
		}
	}
	return n
}

func TestWajibBarang_PerpetualTolakTanpaBarang(t *testing.T) {
	srv, qa, _ := perpetualKartuMulai(t, "2026-01-01")
	for _, jenis := range []string{"Penjualan", "Beli Bahan / Stok"} {
		rec := do(srv, qa, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-01-10","jenis":"`+jenis+`","nominal":100000,"lunas":true}`)
		harusDitolakWajibBarang(t, rec.Code, rec.Body.String())
	}
	// Jenis lain tidak terkena aturan.
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-01-10","jenis":"Gaji & Upah","nominal":100000,"lunas":true}`)
}

func TestWajibBarang_BukanBarangBoleh(t *testing.T) {
	srv, qa, _ := perpetualKartuMulai(t, "2026-01-01")
	tx := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-01-10","jenis":"Penjualan","nominal":100000,"lunas":true,"bukan_barang":true}`)
	if !tx.BukanBarang {
		t.Errorf("tanda bukan barang hilang: %+v", tx)
	}
}

func TestWajibBarang_PeriodikBoleh(t *testing.T) {
	srv, qa := usahaManufaktur(t)
	kain := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kain","kategori":"Bahan Baku","satuan":"m"}`)
	buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/awal", `{"tanggal":"2026-01-01","baris":[{"stok_id":`+itoa(kain.ID)+`,"fisik":10000,"harga_unit":10000}]}`)
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-01-10","jenis":"Penjualan","nominal":100000,"lunas":true}`)
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-01-10","jenis":"Beli Bahan / Stok","nominal":100000,"lunas":true}`)
}

func TestWajibBarang_UsahaBaruBeliPertamaMemulaiKartu(t *testing.T) {
	srv, qa := usahaManufakturPerpetual(t)
	kain := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kain","kategori":"Bahan Baku","satuan":"m","jumlah":0}`)
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions",
		`{"tanggal":"2026-02-05","jenis":"Beli Bahan / Stok","nominal":100000,"lunas":true,"rincian":[{"stok_id":`+itoa(kain.ID)+`,"qty_stok":10000}]}`)

	tn := tenantByEmail(t, srv, qa)
	tn.mu.RLock()
	defer tn.mu.RUnlock()
	i := slices.IndexFunc(tn.gerak, func(g finance.GerakStok) bool { return g.Jenis == finance.GerakAwal })
	if i < 0 {
		t.Fatal("kartu stok tidak dimulai otomatis")
	}
	if g := tn.gerak[i]; g.Tanggal != "2026-02-05" || g.Keterangan != "Stok awal otomatis dari daftar barang (transaksi pertama)" || g.ID == 0 {
		t.Errorf("stok awal otomatis: %+v", g)
	}
	k := tn.kartuLocked()
	if k.Mulai != "2026-02-05" || k.Posisi[kain.ID].Qty != 10000 {
		t.Errorf("kartu mulai %q, saldo Kain %d (harus 10 m)", k.Mulai, k.Posisi[kain.ID].Qty)
	}
	if m, kunci := tn.metodeStokLocked(); !kunci || m == "" {
		t.Errorf("metode nilai stok harus terkunci saat kartu dimulai: %q %v", m, kunci)
	}
}

func TestWajibBarang_KartuBelumDimulaiTanpaBarangDitolak(t *testing.T) {
	srv, qa := usahaManufakturPerpetual(t)
	buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kain","kategori":"Bahan Baku","satuan":"m"}`)
	rec := do(srv, qa, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-02-05","jenis":"Beli Bahan / Stok","nominal":100000,"lunas":true}`)
	harusDitolakWajibBarang(t, rec.Code, rec.Body.String())
	if n := jumlahGerakAwal(t, srv, qa); n != 0 {
		t.Errorf("transaksi ditolak tidak boleh memulai kartu stok (%d)", n)
	}
	// Bukan barang (jasa) boleh dan tidak memulai kartu.
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-02-05","jenis":"Penjualan","nominal":100000,"lunas":true,"bukan_barang":true}`)
	if n := jumlahGerakAwal(t, srv, qa); n != 0 {
		t.Errorf("transaksi jasa tidak boleh memulai kartu stok (%d)", n)
	}
}

// Transaksi berbarang yang gagal sesudah pemeriksaan barang tidak meninggalkan stok awal.
func TestWajibBarang_GagalSimpanTidakMemulaiKartu(t *testing.T) {
	srv, qa := usahaManufakturPerpetual(t)
	kain := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kain","kategori":"Bahan Baku","satuan":"m"}`)
	tn := tenantByEmail(t, srv, qa)
	tn.mu.Lock()
	tn.opening = &finance.OpeningBalance{Tanggal: "2026-03-01"}
	tn.mu.Unlock()
	rec := do(srv, qa, http.MethodPost, "/api/v1/transactions",
		`{"tanggal":"2026-02-05","jenis":"Beli Bahan / Stok","nominal":100000,"lunas":true,"rincian":[{"stok_id":`+itoa(kain.ID)+`,"qty_stok":10000}]}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("sebelum saldo awal harus 409, dapat %d %s", rec.Code, rec.Body)
	}
	if n := jumlahGerakAwal(t, srv, qa); n != 0 {
		t.Errorf("transaksi gagal tidak boleh memulai kartu stok (%d)", n)
	}
}

func TestWajibBarang_TanpaDaftarBarangTidakBerubah(t *testing.T) {
	srv, qa := usahaManufakturPerpetual(t)
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-02-05","jenis":"Penjualan","nominal":100000,"lunas":true}`)
}

func TestWajibBarang_SebelumMulaiKartuBoleh(t *testing.T) {
	srv, qa, _ := perpetualKartuMulai(t, "2026-03-01")
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-02-10","jenis":"Penjualan","nominal":100000,"lunas":true}`)
	rec := do(srv, qa, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-03-01","jenis":"Penjualan","nominal":100000,"lunas":true}`)
	harusDitolakWajibBarang(t, rec.Code, rec.Body.String())
}

// Review Focus 2: pelunasan penjualan perpetual lama tanpa barang tetap berhasil.
func TestWajibBarang_PelunasanTransaksiLamaTetapBoleh(t *testing.T) {
	srv, qa, _ := perpetualKartuMulai(t, "2026-01-01")
	lama := seedTx(t, srv, qa, finance.Transaction{Tanggal: "2026-01-10", Jenis: finance.JenisJual, Nominal: 100000, Lunas: false})
	rec := do(srv, qa, http.MethodPost, "/api/v1/transactions/"+itoa(lama.ID)+"/lunas", `{"tanggal":"2026-01-25"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("pelunasan transaksi lama: %d %s", rec.Code, rec.Body)
	}
}

// Keputusan 5: transaksi lama tanpa barang wajib dilengkapi saat disimpan ulang.
func TestWajibBarang_EditLamaWajibLengkap(t *testing.T) {
	srv, qa, kain := perpetualKartuMulai(t, "2026-01-01")
	lama := seedTx(t, srv, qa, finance.Transaction{Tanggal: "2026-01-10", Jenis: finance.JenisBeliStok, Nominal: 100000, Lunas: true})
	url := "/api/v1/transactions/" + itoa(lama.ID)
	rec := do(srv, qa, http.MethodPut, url, `{"tanggal":"2026-01-10","jenis":"Beli Bahan / Stok","nominal":100000,"lunas":true,"keterangan":"ubah"}`)
	harusDitolakWajibBarang(t, rec.Code, rec.Body.String())
	rec = do(srv, qa, http.MethodPut, url, `{"tanggal":"2026-01-10","jenis":"Beli Bahan / Stok","nominal":100000,"lunas":true,"rincian":[{"stok_id":`+itoa(kain.ID)+`,"qty_stok":5000}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("dilengkapi barang harus 200: %d %s", rec.Code, rec.Body)
	}
	// Data lama stok_id tunggal tersimpan sebagai rincian satu baris.
	rec = do(srv, qa, http.MethodPut, url, `{"tanggal":"2026-01-10","jenis":"Beli Bahan / Stok","nominal":100000,"lunas":true,"stok_id":`+itoa(kain.ID)+`,"qty_stok":5000}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("stok_id tunggal harus 200: %d %s", rec.Code, rec.Body)
	}
	tn := tenantByEmail(t, srv, qa)
	tn.mu.RLock()
	defer tn.mu.RUnlock()
	i := slices.IndexFunc(tn.txs, func(x finance.Transaction) bool { return x.ID == lama.ID })
	if got := tn.txs[i]; got.StokID != 0 || len(got.Rincian) != 1 || got.Rincian[0].StokID != kain.ID || got.Rincian[0].QtyStok != 5000 {
		t.Errorf("harus dinormalkan ke rincian: %+v", got)
	}
}
