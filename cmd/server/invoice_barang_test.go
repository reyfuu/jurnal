package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"pajak/internal/finance"
)

// Invoice per baris (persediaan per akun tahap 2, Task 9): di perpetual tiap baris item invoice
// yang dikirim membawa barang atau ditandai Bukan barang (jasa). Barangnya menjadi rincian
// transaksi Penjualan invoice (nilai = jumlah baris dikurangi bagian diskon), jadi kartu stok
// ikut bergerak bersama invoice.

// invoiceBody: invoice tanggal 2026-01-10 dengan baris items (JSON array) dan diskon.
func invoiceBody(nomor, status, items string, diskon int) string {
	return `{"nomor":"` + nomor + `","status":"` + status + `","tanggal":"2026-01-10","jatuh_tempo":"2026-02-09",` +
		`"pelanggan_nama":"PT Pembeli","items":` + items + `,"diskon_rp":` + itoa(diskon) + `,"ppn_bps":0}`
}

func kirimInvoice(t *testing.T, srv *Server, qa *http.Cookie, method, url, body string, kode int) finance.Invoice {
	t.Helper()
	rec := do(srv, qa, method, url, body)
	if rec.Code != kode {
		t.Fatalf("%s %s: %d %s (harus %d)", method, url, rec.Code, rec.Body, kode)
	}
	var inv finance.Invoice
	_ = json.NewDecoder(rec.Body).Decode(&inv)
	return inv
}

func txTenant(t *testing.T, srv *Server, qa *http.Cookie, id int) finance.Transaction {
	t.Helper()
	tn := tenantByEmail(t, srv, qa)
	tn.mu.RLock()
	defer tn.mu.RUnlock()
	i := slices.IndexFunc(tn.txs, func(x finance.Transaction) bool { return x.ID == id })
	if i < 0 {
		t.Fatalf("transaksi %d tidak ada", id)
	}
	return tn.txs[i]
}

// perpetualDuaBarang: perpetual, Kain (m) & Kaos (pcs) masing-masing 10, kartu mulai 2026-01-01.
func perpetualDuaBarang(t *testing.T) (*Server, *http.Cookie, finance.StokItem, finance.StokItem) {
	t.Helper()
	srv, qa := usahaManufakturPerpetual(t)
	kain := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kain","kategori":"Bahan Baku","satuan":"m"}`)
	kaos := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kaos","kategori":"Produk Jadi","satuan":"pcs","sku":"KS-01"}`)
	buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/awal", `{"tanggal":"2026-01-01","baris":[{"stok_id":`+itoa(kain.ID)+`,"fisik":10000,"harga_unit":10000},{"stok_id":`+itoa(kaos.ID)+`,"fisik":10000,"harga_unit":30000}]}`)
	return srv, qa, kain, kaos
}

func TestInvoice_PerpetualItemWajibBarang(t *testing.T) {
	srv, qa, kain := perpetualKartuMulai(t, "2026-01-01")
	items := `[{"deskripsi":"Kain","qty":2,"harga_satuan":50000,"stok_id":` + itoa(kain.ID) + `},{"deskripsi":"Ongkir","qty":1,"harga_satuan":10000}]`
	rec := do(srv, qa, http.MethodPost, "/api/v1/invoice", invoiceBody("INV-W-1", "terkirim", items, 0))
	if want := "baris 2: pilih barang atau tandai Bukan barang (jasa)"; rec.Code != http.StatusBadRequest || strings.TrimSpace(rec.Body.String()) != want {
		t.Fatalf("harus 400 %q, dapat %d %q", want, rec.Code, rec.Body)
	}
	// Draft belum dibukukan: boleh tanpa barang, tetapi saat dikirim aturan berlaku.
	draft := kirimInvoice(t, srv, qa, http.MethodPost, "/api/v1/invoice", invoiceBody("INV-W-1", "draft", items, 0), http.StatusCreated)
	url := "/api/v1/invoice/" + itoa(draft.ID)
	if rec := do(srv, qa, http.MethodPut, url, invoiceBody("INV-W-1", "terkirim", items, 0)); rec.Code != http.StatusBadRequest || !strings.HasPrefix(rec.Body.String(), "baris 2:") {
		t.Fatalf("draft → terkirim tanpa barang: %d %s", rec.Code, rec.Body)
	}
	// Barang & bukan barang bersamaan, atau barang yang tidak ada, ditolak.
	if rec := do(srv, qa, http.MethodPut, url, invoiceBody("INV-W-1", "terkirim", `[{"deskripsi":"Kain","qty":1,"harga_satuan":1,"stok_id":`+itoa(kain.ID)+`,"bukan_barang":true}]`, 0)); rec.Code != http.StatusBadRequest || !strings.HasPrefix(rec.Body.String(), "baris 1:") {
		t.Fatalf("barang + bukan barang: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, qa, http.MethodPut, url, invoiceBody("INV-W-1", "terkirim", `[{"deskripsi":"X","qty":1,"harga_satuan":1,"stok_id":999}]`, 0)); rec.Code != http.StatusBadRequest || !strings.HasPrefix(rec.Body.String(), "baris 1:") {
		t.Fatalf("barang tidak ada: %d %s", rec.Code, rec.Body)
	}
	jasa := strings.Replace(items, `"harga_satuan":10000}`, `"harga_satuan":10000,"bukan_barang":true}`, 1)
	inv := kirimInvoice(t, srv, qa, http.MethodPut, url, invoiceBody("INV-W-1", "terkirim", jasa, 0), http.StatusOK)
	if inv.TransaksiID == 0 || !inv.Items[1].BukanBarang || inv.Items[0].StokID != kain.ID {
		t.Fatalf("invoice terkirim: %+v", inv)
	}
	// Pilihan barang untuk form invoice (dijaga menu Invoice).
	if rec := do(srv, qa, http.MethodGet, "/api/v1/invoice/pilihan-barang", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"nama":"Kain"`) {
		t.Errorf("pilihan barang invoice: %d %s", rec.Code, rec.Body)
	}
	// Periodik: tetap boleh tanpa barang.
	srvP, qaP := usahaManufaktur(t)
	kirimInvoice(t, srvP, qaP, http.MethodPost, "/api/v1/invoice", invoiceBody("INV-P-1", "terkirim", `[{"deskripsi":"Ongkir","qty":1,"harga_satuan":10000}]`, 0), http.StatusCreated)
}

func TestInvoice_RincianIkutItemDanDiskon(t *testing.T) {
	srv, qa, kain, kaos := perpetualDuaBarang(t)
	items := `[{"deskripsi":"Kain","qty":3,"harga_satuan":20000,"stok_id":` + itoa(kain.ID) + `},` +
		`{"deskripsi":"Kaos","qty":2,"harga_satuan":50000,"stok_id":` + itoa(kaos.ID) + `},` +
		`{"deskripsi":"Jasa jahit","qty":1,"harga_satuan":50000,"bukan_barang":true}]`
	inv := kirimInvoice(t, srv, qa, http.MethodPost, "/api/v1/invoice", invoiceBody("INV-R-1", "terkirim", items, 10_000), http.StatusCreated)
	tx := txTenant(t, srv, qa, inv.TransaksiID)
	// Subtotal 210.000, diskon 10.000 → bagian 2.857 / 4.761 (+2 sisa ke baris terbesar) / 2.380.
	want := []finance.RincianBarang{{StokID: kain.ID, QtyStok: 3000, Nilai: 57_143}, {StokID: kaos.ID, QtyStok: 2000, Nilai: 95_237}}
	if tx.Nominal != 200_000 || tx.BukanBarang || !slices.Equal(tx.Rincian, want) {
		t.Fatalf("transaksi invoice: nominal %d bukan barang %v rincian %+v (harus %+v)", tx.Nominal, tx.BukanBarang, tx.Rincian, want)
	}
	if sum := tx.Rincian[0].Nilai + tx.Rincian[1].Nilai; sum != 200_000-47_620 {
		t.Errorf("Σ nilai rincian %d, harus DPP − bagian jasa = %d", sum, 200_000-47_620)
	}
	if a, b := saldoKartu(t, srv, qa, kain.ID), saldoKartu(t, srv, qa, kaos.ID); a != 7000 || b != 8000 {
		t.Errorf("kartu Kain %d, Kaos %d (harus 7 & 8)", a, b)
	}
	// Semua baris jasa → transaksi bertanda bukan barang tanpa rincian.
	semuaJasa := kirimInvoice(t, srv, qa, http.MethodPost, "/api/v1/invoice", invoiceBody("INV-R-2", "terkirim", `[{"deskripsi":"Jasa","qty":1,"harga_satuan":50000,"bukan_barang":true}]`, 0), http.StatusCreated)
	if tx := txTenant(t, srv, qa, semuaJasa.TransaksiID); !tx.BukanBarang || len(tx.Rincian) != 0 {
		t.Errorf("invoice jasa: %+v", tx)
	}
}

func TestInvoice_UbahItemKartuIkut(t *testing.T) {
	srv, qa, kain, kaos := perpetualDuaBarang(t)
	baris := func(qty int) string {
		return `[{"deskripsi":"Kain","qty":` + itoa(qty) + `,"harga_satuan":20000,"stok_id":` + itoa(kain.ID) + `}]`
	}
	inv := kirimInvoice(t, srv, qa, http.MethodPost, "/api/v1/invoice", invoiceBody("INV-U-1", "terkirim", baris(3), 0), http.StatusCreated)
	url := "/api/v1/invoice/" + itoa(inv.ID)
	if q := saldoKartu(t, srv, qa, kain.ID); q != 7000 {
		t.Fatalf("kartu Kain setelah kirim: %d", q)
	}
	kirimInvoice(t, srv, qa, http.MethodPut, url, invoiceBody("INV-U-1", "terkirim", baris(5), 0), http.StatusOK)
	if q := saldoKartu(t, srv, qa, kain.ID); q != 5000 {
		t.Errorf("kartu Kain setelah qty 5: %d", q)
	}
	// Ganti barang: Kain kembali penuh, Kaos berkurang.
	kaosLunas := strings.Replace(invoiceBody("INV-U-1", "lunas", `[{"deskripsi":"Kaos","qty":4,"harga_satuan":50000,"stok_id":`+itoa(kaos.ID)+`}]`, 0),
		`"ppn_bps":0`, `"ppn_bps":0,"tanggal_lunas":"2026-01-20"`, 1)
	kirimInvoice(t, srv, qa, http.MethodPut, url, kaosLunas, http.StatusOK)
	if a, b := saldoKartu(t, srv, qa, kain.ID), saldoKartu(t, srv, qa, kaos.ID); a != 10_000 || b != 6000 {
		t.Errorf("setelah ganti barang: Kain %d, Kaos %d (harus 10 & 6)", a, b)
	}
	// Kembali ke draft → transaksinya hilang, kartu pulih.
	kirimInvoice(t, srv, qa, http.MethodPut, url, invoiceBody("INV-U-1", "draft", baris(5), 0), http.StatusOK)
	if b := saldoKartu(t, srv, qa, kaos.ID); b != 10_000 {
		t.Errorf("kembali draft: Kaos %d", b)
	}
	kirimInvoice(t, srv, qa, http.MethodPut, url, invoiceBody("INV-U-1", "terkirim", baris(2), 0), http.StatusOK)
	if rec := do(srv, qa, http.MethodDelete, url, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("hapus: %d %s", rec.Code, rec.Body)
	}
	if q := saldoKartu(t, srv, qa, kain.ID); q != 10_000 {
		t.Errorf("hapus invoice: Kain %d", q)
	}
}

func TestInvoice_MelebihiStokDitolak(t *testing.T) {
	srv, qa, kain := perpetualKartuMulai(t, "2026-01-01")
	tn := tenantByEmail(t, srv, qa)
	nTx := len(tn.txs)
	items := `[{"deskripsi":"Kain","qty":11,"harga_satuan":20000,"stok_id":` + itoa(kain.ID) + `}]`
	rec := do(srv, qa, http.MethodPost, "/api/v1/invoice", invoiceBody("INV-M-1", "terkirim", items, 0))
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "stok Kain tidak cukup") {
		t.Fatalf("melebihi stok: %d %s", rec.Code, rec.Body)
	}
	if len(tn.txs) != nTx || len(tn.invoices) != 0 {
		t.Fatalf("invoice ditolak tidak boleh meninggalkan data: tx %d→%d, invoice %d", nTx, len(tn.txs), len(tn.invoices))
	}
	// Draft boleh; mengirimnya ditolak dengan alasan yang sama.
	draft := kirimInvoice(t, srv, qa, http.MethodPost, "/api/v1/invoice", invoiceBody("INV-M-1", "draft", items, 0), http.StatusCreated)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/invoice/"+itoa(draft.ID), invoiceBody("INV-M-1", "terkirim", items, 0)); rec.Code != http.StatusConflict {
		t.Fatalf("kirim draft melebihi stok: %d %s", rec.Code, rec.Body)
	}
	if len(tn.txs) != nTx {
		t.Errorf("transaksi tertinggal setelah ditolak: %d→%d", nTx, len(tn.txs))
	}
}

func TestInvoice_BarangPertamaMemulaiKartu(t *testing.T) {
	srv, qa := usahaManufakturPerpetual(t)
	kain := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kain","kategori":"Bahan Baku","satuan":"m","jumlah":10,"harga_pokok":10000}`)
	kirimInvoice(t, srv, qa, http.MethodPost, "/api/v1/invoice", invoiceBody("INV-A-1", "terkirim", `[{"deskripsi":"Kain","qty":2,"harga_satuan":20000,"stok_id":`+itoa(kain.ID)+`}]`, 0), http.StatusCreated)
	tn := tenantByEmail(t, srv, qa)
	tn.mu.RLock()
	defer tn.mu.RUnlock()
	k := tn.kartuLocked()
	if k.Mulai != "2026-01-10" || k.Posisi[kain.ID].Qty != 8000 {
		t.Errorf("kartu mulai %q, saldo Kain %d (harus 2026-01-10 & 8 m)", k.Mulai, k.Posisi[kain.ID].Qty)
	}
}

// Invoice lama (dibukukan sebelum aturan wajib barang) tetap sinkron: dilunasi dari form atau dari
// Piutang & Utang, dan dibukukan lewat "Bukukan", tanpa harus memilih barang. Mengubah barisnya
// baru menerapkan aturan.
func TestInvoice_LamaTanpaBarangTetapSinkron(t *testing.T) {
	srv, qa := usahaManufaktur(t)
	kain := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kain","kategori":"Bahan Baku","satuan":"m"}`)
	buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/awal", `{"tanggal":"2026-01-01","baris":[{"stok_id":`+itoa(kain.ID)+`,"fisik":10000,"harga_unit":10000}]}`)
	items := `[{"deskripsi":"Kain lama","qty":2,"harga_satuan":20000}]`
	lama := kirimInvoice(t, srv, qa, http.MethodPost, "/api/v1/invoice", invoiceBody("INV-L-1", "terkirim", items, 0), http.StatusCreated)
	belum := kirimInvoice(t, srv, qa, http.MethodPost, "/api/v1/invoice", invoiceBody("INV-L-2", "terkirim", items, 0), http.StatusCreated)
	tn := tenantByEmail(t, srv, qa)
	tn.mu.Lock() // INV-L-2: invoice sebelum invoice tersambung ke pembukuan
	for i := range tn.invoices {
		if tn.invoices[i].ID == belum.ID {
			tn.txs = slices.DeleteFunc(tn.txs, func(x finance.Transaction) bool { return x.InvoiceID == belum.ID })
			tn.invoices[i].Dibukukan, tn.invoices[i].TransaksiID = false, 0
		}
	}
	tn.mu.Unlock()
	if rec := do(srv, qa, http.MethodPut, "/api/v1/business", manufakturPerpetual); rec.Code != http.StatusOK {
		t.Fatalf("pindah perpetual: %d %s", rec.Code, rec.Body)
	}
	url := "/api/v1/invoice/" + itoa(lama.ID)
	lunas := strings.Replace(invoiceBody("INV-L-1", "lunas", items, 0), `"ppn_bps":0`, `"ppn_bps":0,"tanggal_lunas":"2026-01-20"`, 1)
	if inv := kirimInvoice(t, srv, qa, http.MethodPut, url, lunas, http.StatusOK); !txTenant(t, srv, qa, inv.TransaksiID).Lunas {
		t.Errorf("invoice lama dilunasi lewat form: transaksi belum lunas")
	}
	kirimInvoice(t, srv, qa, http.MethodPut, url, invoiceBody("INV-L-1", "terkirim", items, 0), http.StatusOK)
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions/"+itoa(lama.TransaksiID)+"/lunas", `{"tanggal":"2026-01-25"}`); rec.Code != http.StatusOK {
		t.Fatalf("lunasi invoice lama dari Piutang & Utang: %d %s", rec.Code, rec.Body)
	}
	rec := do(srv, qa, http.MethodPost, "/api/v1/invoice/bukukan", "")
	var out struct {
		Dibukukan int      `json:"dibukukan"`
		Gagal     []string `json:"gagal"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&out)
	if rec.Code != http.StatusOK || out.Dibukukan != 1 || len(out.Gagal) != 0 {
		t.Fatalf("bukukan invoice lama: %d %+v", rec.Code, out)
	}
	// Mengubah baris invoice lama → aturan wajib barang berlaku.
	if rec := do(srv, qa, http.MethodPut, url, invoiceBody("INV-L-1", "terkirim", `[{"deskripsi":"Kain lama","qty":3,"harga_satuan":20000}]`, 0)); rec.Code != http.StatusBadRequest {
		t.Fatalf("ubah baris invoice lama tanpa barang: %d %s", rec.Code, rec.Body)
	}
}

// Impor invoice: kolom Barang (SKU / nama persis, tanpa beda huruf besar; "jasa" = bukan barang).
func TestImportInvoice_KolomBarang(t *testing.T) {
	srv, qa, kain, kaos := perpetualDuaBarang(t)
	h := []string{"Nomor", "Tanggal", "Jatuh Tempo", "Pelanggan", "Deskripsi", "Qty", "Harga Satuan", "Barang"}
	rows := [][]string{h,
		{"INV-I-1", "2026-01-10", "2026-02-10", "PT Satu", "Kain", "2", "20000", "kain"},
		{"INV-I-1", "", "", "", "Kaos", "1", "50000", "ks-01"},
		{"INV-I-1", "", "", "", "Ongkir", "1", "10000", "Jasa"},
		{"INV-I-2", "2026-01-10", "2026-02-10", "PT Dua", "Topi", "1", "10000", "Topi"},
		{"INV-I-3", "2026-01-10", "2026-02-10", "PT Tiga", "Ongkir", "1", "10000", ""},
	}
	body, ctype := workbookUpload(t, []finance.ImportSheet{{Nama: "Invoice", Rows: rows}})
	rec := upload(srv, qa, "/api/v1/invoice/import?commit=1", body, ctype)
	var res menuImportResp
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("impor: %d %s", rec.Code, rec.Body)
	}
	if res.Valid != 1 || len(res.Errors) != 2 {
		t.Fatalf("impor: valid %d errors %+v", res.Valid, res.Errors)
	}
	pesan := res.Errors[0].Pesan + " | " + res.Errors[1].Pesan
	if !strings.Contains(pesan, `barang "Topi" tidak ada`) || !strings.Contains(pesan, "baris 1: pilih barang atau tandai Bukan barang (jasa)") {
		t.Errorf("pesan galat: %s", pesan)
	}
	it := res.Invoice[0].Items
	if it[0].StokID != kain.ID || it[1].StokID != kaos.ID || !it[2].BukanBarang {
		t.Fatalf("barang hasil impor: %+v", it)
	}
	if a, b := saldoKartu(t, srv, qa, kain.ID), saldoKartu(t, srv, qa, kaos.ID); a != 8000 || b != 9000 {
		t.Errorf("kartu setelah impor: Kain %d, Kaos %d", a, b)
	}
}

// Pratinjau impor menjalankan langkah simpan lalu memulihkan semuanya, termasuk kartu stok yang
// dimulai otomatis oleh invoice berbarang pertama dan kunci metode nilai stoknya.
func TestImportInvoice_PratinjauTidakMemulaiKartu(t *testing.T) {
	srv, qa := usahaManufakturPerpetual(t)
	buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kain","kategori":"Bahan Baku","satuan":"m","jumlah":10,"harga_pokok":10000}`)
	rows := [][]string{{"Nomor", "Tanggal", "Jatuh Tempo", "Pelanggan", "Deskripsi", "Qty", "Harga Satuan", "Barang"},
		{"INV-P-1", "2026-01-10", "2026-02-10", "PT Satu", "Kain", "2", "20000", "Kain"}}
	kirim := func(q string) menuImportResp {
		body, ctype := workbookUpload(t, []finance.ImportSheet{{Nama: "Invoice", Rows: rows}})
		rec := upload(srv, qa, "/api/v1/invoice/import"+q, body, ctype)
		var res menuImportResp
		if err := json.NewDecoder(rec.Body).Decode(&res); err != nil || rec.Code != http.StatusOK || res.Valid != 1 {
			t.Fatalf("impor%s: %d %s", q, rec.Code, rec.Body)
		}
		return res
	}
	kirim("")
	tn := tenantByEmail(t, srv, qa)
	tn.mu.RLock()
	mulai, metode := tn.mulaiKartuStokLocked(), tn.profile.MetodeNilaiStok
	tn.mu.RUnlock()
	if mulai != "" || metode != "" {
		t.Fatalf("pratinjau memulai kartu %q / mengunci metode %q", mulai, metode)
	}
	kirim("?commit=1")
	if n := jumlahGerakAwal(t, srv, qa); n != 1 {
		t.Errorf("impor memulai kartu: %d stok awal", n)
	}
}
