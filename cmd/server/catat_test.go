package main

import (
	"encoding/json"
	"net/http"
	"pajak/internal/finance"
	"strconv"
	"strings"
	"testing"

	"pajak/internal/coa"
)

// Catatan jurnal multi-akun tersimpan sebagai satu grup transaksi: masuk neraca saldo per akun
// persis, bisa dibuka ulang, tidak bisa diubah satu per satu, dan terhapus bersama.
func TestCatat_JurnalMultiAkun(t *testing.T) {
	srv, qa := usahaBaru(t)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/opening", `{"tanggal":"2026-01-01","rincian":[{"nama":"Kas","kategori":"kas","nominal":1000000},{"nama":"Modal Pemilik","kategori":"modal","nominal":1000000}]}`); rec.Code != http.StatusOK {
		t.Fatalf("saldo awal: %d %s", rec.Code, rec.Body)
	}
	body := `{"tanggal":"2026-01-10","pihak":"Toko Sumber Rejeki","no_bukti":"SR-1","keterangan":"Belanja","baris":[
		{"akun":"6106","keterangan":"Ongkir","debit":10500},
		{"akun":"6190","keterangan":"Lain-lain","debit":30000},
		{"akun":"1101","kredit":25000},
		{"akun":"2101","kredit":15500,"jatuh_tempo":"2026-02-10"}]}`
	rec := do(srv, qa, http.MethodPost, "/api/v1/catat", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("catat: %d %s", rec.Code, rec.Body)
	}
	var c catatResp
	_ = json.Unmarshal(rec.Body.Bytes(), &c)
	if c.Grup == 0 || len(c.Transaksi) != 3 {
		t.Fatalf("hasil: %+v", c)
	}
	var rep struct {
		TrialBalance []coa.Balance `json:"trial_balance"`
	}
	_ = json.Unmarshal(do(srv, qa, http.MethodGet, "/api/v1/reports/coa?period=2026-01", "").Body.Bytes(), &rep)
	saldo := map[string]int64{}
	for _, b := range rep.TrialBalance {
		saldo[b.Kode] = b.Saldo
	}
	if saldo["6106"] != 10_500 || saldo["6190"] != 30_000 || saldo["1101"] != 975_000 || saldo["2101"] != 15_500 {
		t.Fatalf("neraca saldo: 6106=%d 6190=%d 1101=%d 2101=%d", saldo["6106"], saldo["6190"], saldo["1101"], saldo["2101"])
	}
	url := "/api/v1/catat/" + strconv.Itoa(c.Grup)
	g := getJSON[catatResp](t, srv, qa, url)
	if g.Pihak != "Toko Sumber Rejeki" || g.NoBukti != "SR-1" || len(g.Baris) != 4 {
		t.Fatalf("buka ulang: %+v", g)
	}
	if rec := do(srv, qa, http.MethodPut, "/api/v1/transactions/"+strconv.Itoa(c.Transaksi[0]), `{"tanggal":"2026-01-10","jenis":"Biaya Lain","nominal":1,"lunas":true}`); rec.Code != http.StatusConflict {
		t.Fatalf("ubah satu baris grup harus ditolak: %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/catat", `{"tanggal":"2026-01-10","baris":[{"akun":"6190","debit":10},{"akun":"1101","kredit":9}]}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("tidak seimbang harus ditolak: %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodDelete, url, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("hapus: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, qa, http.MethodGet, url, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("catatan terhapus harus hilang: %d", rec.Code)
	}
}

// Checklist tester No 27: halaman Catat hanya punya satu keterangan (kepala). Membuka catatan
// harus mengembalikan keterangan itu supaya simpan ulang tidak mengosongkannya; catatan lama
// dengan keterangan per baris digabung jadi satu.
func TestCatat_KeteranganKepalaBertahanSaatDiubah(t *testing.T) {
	srv, qa := usahaBaru(t)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/opening", `{"tanggal":"2026-01-01","rincian":[{"nama":"Kas","kategori":"kas","nominal":1000000},{"nama":"Modal Pemilik","kategori":"modal","nominal":1000000}]}`); rec.Code != http.StatusOK {
		t.Fatalf("saldo awal: %d %s", rec.Code, rec.Body)
	}
	buatCatat := func(body string) string {
		rec := do(srv, qa, http.MethodPost, "/api/v1/catat", body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("catat: %d %s", rec.Code, rec.Body)
		}
		var c catatResp
		_ = json.Unmarshal(rec.Body.Bytes(), &c)
		return "/api/v1/catat/" + strconv.Itoa(c.Grup)
	}
	url := buatCatat(`{"tanggal":"2026-01-10","keterangan":"Belanja bahan","baris":[{"akun":"6106","debit":10000},{"akun":"6190","debit":5000},{"akun":"1101","kredit":15000}]}`)
	if g := getJSON[catatResp](t, srv, qa, url); g.Keterangan != "Belanja bahan" {
		t.Fatalf("keterangan kepala = %q", g.Keterangan)
	}
	// simpan ulang seperti klien baru: keterangan baris kosong, kepala dari GET
	if rec := do(srv, qa, http.MethodPut, url, `{"tanggal":"2026-01-10","keterangan":"Belanja bahan","baris":[{"akun":"6106","debit":12000},{"akun":"6190","debit":5000},{"akun":"1101","kredit":17000}]}`); rec.Code != http.StatusOK {
		t.Fatalf("ubah: %d %s", rec.Code, rec.Body)
	}
	if g := getJSON[catatResp](t, srv, qa, url); g.Keterangan != "Belanja bahan" {
		t.Fatalf("keterangan setelah diubah = %q", g.Keterangan)
	}
	lama := buatCatat(`{"tanggal":"2026-01-11","baris":[{"akun":"6106","keterangan":"Ongkir","debit":10000},{"akun":"6190","keterangan":"Lain-lain","debit":5000},{"akun":"1101","kredit":15000}]}`)
	if g := getJSON[catatResp](t, srv, qa, lama); g.Keterangan != "Ongkir; Lain-lain" {
		t.Fatalf("keterangan per baris lama digabung = %q", g.Keterangan)
	}
}

// Checklist tester No 20: pindah dana Bank → Kas dan Kas → Bank menggeser saldo per rekening,
// tidak mengubah laba, dan saldonya terbawa ke bulan berikutnya.
func TestCatat_PindahDanaAntarRekening(t *testing.T) {
	srv, qa := usahaBaru(t)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/opening", `{"tanggal":"2026-01-01","rincian":[{"nama":"Kas","kategori":"kas","nominal":100000},{"nama":"Bank","kategori":"kas","nominal":1000000},{"nama":"Modal Pemilik","kategori":"modal","nominal":1100000}]}`); rec.Code != http.StatusOK {
		t.Fatalf("saldo awal: %d %s", rec.Code, rec.Body)
	}
	for _, body := range []string{
		`{"tanggal":"2026-01-10","keterangan":"Tarik tunai","baris":[{"akun":"1101","debit":300000},{"akun":"1102","kredit":300000}]}`,
		`{"tanggal":"2026-01-20","keterangan":"Setor ke bank","baris":[{"akun":"1102","debit":50000},{"akun":"9102","debit":2500},{"akun":"1101","kredit":52500}]}`,
	} {
		if rec := do(srv, qa, http.MethodPost, "/api/v1/catat", body); rec.Code != http.StatusCreated {
			t.Fatalf("catat: %d %s", rec.Code, rec.Body)
		}
	}
	saldo := func(period string) map[string]int64 {
		var rep struct {
			TrialBalance []coa.Balance `json:"trial_balance"`
		}
		_ = json.Unmarshal(do(srv, qa, http.MethodGet, "/api/v1/reports/coa?period="+period, "").Body.Bytes(), &rep)
		out := map[string]int64{}
		for _, b := range rep.TrialBalance {
			out[b.Kode] = b.Saldo
		}
		return out
	}
	for _, period := range []string{"2026-01", "2026-02"} {
		s := saldo(period)
		if s["1101"] != 347_500 || s["1102"] != 750_000 {
			t.Fatalf("%s: kas=%d bank=%d, mau 347.500 & 750.000", period, s["1101"], s["1102"])
		}
	}
	// form transaksi lama tidak bisa memilih rekening asal: harus lewat Catat
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-01-25","jenis":"Pindah Dana","nominal":1000,"lunas":true,"akun":"1101","lawan":"1102"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("form pindah dana harus ditolak: %d", rec.Code)
	}
	is := getJSON[finance.IncomeStatement](t, srv, qa, "/api/v1/reports/financial?period=2026-01")
	bs := getJSON[finance.BalanceSheet](t, srv, qa, "/api/v1/reports/balance-sheet?period=2026-01")
	if is.NetProfit != -2_500 || bs.Assets.Cash != 1_097_500 {
		t.Fatalf("laba bersih %d (mau -2.500, hanya biaya admin), kas & bank %d", is.NetProfit, bs.Assets.Cash)
	}
}

// --- Persediaan per akun tahap 2 (Task 5): rincian barang di Catat cepat ---

// akunCatatBawaan: akun pendapatan, persediaan (Beli Bahan / Stok), kas, dan piutang dari /catat/opsi.
func akunCatatBawaan(t *testing.T, srv *Server, qa *http.Cookie) (jual, beli, kas, piutang string) {
	t.Helper()
	o := getJSON[struct {
		Bawaan map[string]string `json:"bawaan"`
	}](t, srv, qa, "/api/v1/catat/opsi")
	return o.Bawaan["Penjualan"], o.Bawaan["Beli Bahan / Stok"], o.Bawaan["kas"], o.Bawaan["piutang"]
}

// catatJual: catatan penjualan tunai 100.000 dengan tambahan field (mis. rincian) di ekor JSON.
func catatJual(tanggal, jual, kas, ekor string) string {
	return `{"tanggal":"` + tanggal + `","baris":[{"akun":"` + kas + `","debit":100000},{"akun":"` + jual + `","kredit":100000}]` + ekor + `}`
}

func saldoKartu(t *testing.T, srv *Server, qa *http.Cookie, stokID int) int64 {
	t.Helper()
	tn := tenantByEmail(t, srv, qa)
	tn.mu.RLock()
	defer tn.mu.RUnlock()
	m, _ := tn.metodeStokLocked()
	saldo := int64(0)
	for _, b := range finance.HitungKartuMetode(m, tn.stok, tn.txs, tn.gerak).Baris {
		if b.StokID == stokID {
			saldo = b.SaldoQty
		}
	}
	return saldo
}

func TestCatat_PerpetualPenjualanWajibBarang(t *testing.T) {
	srv, qa, kaos, _ := perpetualKaos(t, 5000, 0)
	jual, _, kas, _ := akunCatatBawaan(t, srv, qa)
	rec := do(srv, qa, http.MethodPost, "/api/v1/catat", catatJual("2026-01-10", jual, kas, ""))
	harusDitolakWajibBarang(t, rec.Code, rec.Body.String())
	if n := len(txsOf(t, srv, qa)); n != 0 {
		t.Fatalf("catatan tertolak tetap tersimpan (%d transaksi)", n)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/catat", catatJual("2026-01-10", jual, kas, `,"bukan_barang":true`)); rec.Code != http.StatusCreated {
		t.Fatalf("jasa: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/catat", catatJual("2026-01-11", jual, kas, `,"rincian":[{"stok_id":`+itoa(kaos.ID)+`,"qty_stok":2000}]`)); rec.Code != http.StatusCreated {
		t.Fatalf("berbarang: %d %s", rec.Code, rec.Body)
	}
	got := txsOf(t, srv, qa)
	if len(got) != 2 {
		t.Fatalf("transaksi tersimpan: %+v", got)
	}
	jasa, barang := got[0], got[1]
	if barang.BukanBarang {
		jasa, barang = barang, jasa
	}
	if !jasa.BukanBarang || len(barang.BarisBarang()) != 1 || barang.BarisBarang()[0].QtyStok != 2000 {
		t.Fatalf("transaksi tersimpan: %+v", got)
	}
	if s := saldoKartu(t, srv, qa, kaos.ID); s != 3000 {
		t.Fatalf("saldo Kaos sesudah jual 2 pcs = %d, mau 3000", s)
	}
}

func TestCatat_RincianTersimpanDanTampilSaatUbah(t *testing.T) {
	srv, qa, _, kain := perpetualKaos(t, 0, 0)
	_, beli, kas, _ := akunCatatBawaan(t, srv, qa)
	body := func(qty int) string {
		return `{"tanggal":"2026-01-10","baris":[{"akun":"` + beli + `","debit":40000},{"akun":"` + kas + `","kredit":40000}],` +
			`"rincian":[{"stok_id":` + itoa(kain.ID) + `,"qty_stok":` + itoa(qty) + `,"nilai":35000}]}`
	}
	c := buat[catatResp](t, srv, qa, "/api/v1/catat", body(3000))
	url := "/api/v1/catat/" + itoa(c.Grup)
	g := getJSON[catatResp](t, srv, qa, url)
	if len(g.Rincian) != 1 || g.Rincian[0] != (finance.RincianBarang{StokID: kain.ID, QtyStok: 3000, Nilai: 35000}) || g.BukanBarang {
		t.Fatalf("buka ulang rincian: %+v", g.Rincian)
	}
	if rec := do(srv, qa, http.MethodPut, url, body(4500)); rec.Code != http.StatusOK {
		t.Fatalf("ubah: %d %s", rec.Code, rec.Body)
	}
	if g := getJSON[catatResp](t, srv, qa, url); len(g.Rincian) != 1 || g.Rincian[0].QtyStok != 4500 {
		t.Fatalf("rincian sesudah diubah: %+v", g.Rincian)
	}
	if got := txsOf(t, srv, qa); len(got) != 1 || got[0].BarisBarang()[0].QtyStok != 4500 {
		t.Fatalf("transaksi sesudah diubah: %+v", got)
	}
	if s := saldoKartu(t, srv, qa, kain.ID); s != 4500 {
		t.Fatalf("saldo Kain sesudah diubah = %d, mau 4500", s)
	}
	// Jasa juga tampil saat ubah.
	j := buat[catatResp](t, srv, qa, "/api/v1/catat", `{"tanggal":"2026-01-11","bukan_barang":true,"baris":[{"akun":"`+beli+`","debit":5000},{"akun":"`+kas+`","kredit":5000}]}`)
	if g := getJSON[catatResp](t, srv, qa, "/api/v1/catat/"+itoa(j.Grup)); !g.BukanBarang || len(g.Rincian) != 0 {
		t.Fatalf("buka ulang jasa: %+v", g)
	}
}

func TestCatat_DuaTransaksiBerbarangDitolak(t *testing.T) {
	const pesan = "catatan ini berisi lebih dari satu penjualan/pembelian barang; pecah jadi beberapa catatan"
	srv, qa, kaos, _ := perpetualKaos(t, 5000, 0)
	jual, _, kas, piutang := akunCatatBawaan(t, srv, qa)
	rincian := `"rincian":[{"stok_id":` + itoa(kaos.ID) + `,"qty_stok":1000}]`
	// sebagian tunai, sisanya piutang = dua transaksi Penjualan
	body := `{"tanggal":"2026-01-10","baris":[{"akun":"` + kas + `","debit":60000},{"akun":"` + piutang + `","debit":40000,"jatuh_tempo":"2026-02-10"},` +
		`{"akun":"` + jual + `","kredit":100000}],` + rincian + `}`
	rec := do(srv, qa, http.MethodPost, "/api/v1/catat", body)
	if rec.Code != http.StatusBadRequest || strings.TrimSpace(rec.Body.String()) != pesan {
		t.Fatalf("harus 400 %q, dapat %d %q", pesan, rec.Code, rec.Body)
	}
	// tanpa rincian di perpetual: pesannya juga minta dipecah (rincian tidak bisa ditempel)
	rec = do(srv, qa, http.MethodPost, "/api/v1/catat", strings.Replace(body, ","+rincian, "", 1))
	if rec.Code != http.StatusBadRequest || strings.TrimSpace(rec.Body.String()) != pesan {
		t.Fatalf("tanpa rincian harus 400 %q, dapat %d %q", pesan, rec.Code, rec.Body)
	}
	if n := len(txsOf(t, srv, qa)); n != 0 {
		t.Fatalf("catatan tertolak tetap tersimpan (%d)", n)
	}
	// jasa boleh dua transaksi
	if rec := do(srv, qa, http.MethodPost, "/api/v1/catat", strings.Replace(body, rincian, `"bukan_barang":true`, 1)); rec.Code != http.StatusCreated {
		t.Fatalf("jasa dua transaksi: %d %s", rec.Code, rec.Body)
	}
}

// Penjualan lewat Catat cepat melebihi stok ditolak 409; mengubah catatan itu memakai stok yang
// dibebaskan transaksi lamanya (tidak dihitung dua kali); menghapus pembelian yang dipakai ditolak.
func TestCatat_PenjualanMelebihiStokDitolak(t *testing.T) {
	srv, qa, kaos, kain := perpetualKaos(t, 5000, 0)
	jual, beli, kas, _ := akunCatatBawaan(t, srv, qa)
	rincian := func(id, qty int) string {
		return `,"rincian":[{"stok_id":` + itoa(id) + `,"qty_stok":` + itoa(qty) + `}]`
	}
	rec := do(srv, qa, http.MethodPost, "/api/v1/catat", catatJual("2026-01-10", jual, kas, rincian(kaos.ID, 6000)))
	cekMinus(t, rec.Code, rec.Body.String(), "stok Kaos tidak cukup pada 10 Jan 2026: tersedia 5 pcs, dibutuhkan 6 pcs")
	c := buat[catatResp](t, srv, qa, "/api/v1/catat", catatJual("2026-01-10", jual, kas, rincian(kaos.ID, 5000)))
	url := "/api/v1/catat/" + itoa(c.Grup)
	if rec := do(srv, qa, http.MethodPut, url, catatJual("2026-01-10", jual, kas, rincian(kaos.ID, 5000))); rec.Code != http.StatusOK {
		t.Fatalf("simpan ulang tanpa perubahan: %d %s", rec.Code, rec.Body)
	}
	rec = do(srv, qa, http.MethodPut, url, catatJual("2026-01-10", jual, kas, rincian(kaos.ID, 5500)))
	cekMinus(t, rec.Code, rec.Body.String(), "stok Kaos tidak cukup pada 10 Jan 2026: tersedia 5 pcs, dibutuhkan 5,5 pcs")
	// beli Kain lewat catat lalu jual Kain lewat form; hapus catatan pembelian → minus → ditolak
	b := buat[catatResp](t, srv, qa, "/api/v1/catat", `{"tanggal":"2026-01-11","baris":[{"akun":"`+beli+`","debit":20000},{"akun":"`+kas+`","kredit":20000}]`+rincian(kain.ID, 2000)+`}`)
	buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", jualBody("2026-01-12", kain.ID, 2000))
	rec = do(srv, qa, http.MethodDelete, "/api/v1/catat/"+itoa(b.Grup), "")
	cekMinus(t, rec.Code, rec.Body.String(), "stok Kain tidak cukup pada 12 Jan 2026: tersedia 0 m, dibutuhkan 2 m")
}

// Kartu belum dimulai: catatan berbarang pertama memulainya dari daftar barang; bila ditolak
// (stok kurang), stok awal otomatis tidak tersimpan.
func TestCatat_MulaiKartuOtomatis(t *testing.T) {
	srv, qa := usahaManufakturPerpetual(t)
	kaos := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kaos","kategori":"Produk Jadi","satuan":"pcs","jumlah":5}`)
	jual, _, kas, _ := akunCatatBawaan(t, srv, qa)
	rec := do(srv, qa, http.MethodPost, "/api/v1/catat", catatJual("2026-01-10", jual, kas, `,"rincian":[{"stok_id":`+itoa(kaos.ID)+`,"qty_stok":9000}]`))
	cekMinus(t, rec.Code, rec.Body.String(), "stok Kaos tidak cukup pada 10 Jan 2026: tersedia 5 pcs, dibutuhkan 9 pcs")
	if n := jumlahGerakAwal(t, srv, qa); n != 0 {
		t.Fatalf("stok awal otomatis tersimpan padahal catatan ditolak (%d)", n)
	}
	buat[catatResp](t, srv, qa, "/api/v1/catat", catatJual("2026-01-10", jual, kas, `,"rincian":[{"stok_id":`+itoa(kaos.ID)+`,"qty_stok":2000}]`))
	if n := jumlahGerakAwal(t, srv, qa); n != 1 {
		t.Fatalf("catatan berbarang pertama harus memulai kartu stok (gerak awal %d)", n)
	}
	if s := saldoKartu(t, srv, qa, kaos.ID); s != 3000 {
		t.Fatalf("saldo Kaos = %d, mau 3000", s)
	}
}

// /catat/opsi menandai akun yang membawa barang: persediaan di debit (Beli) dan pendapatan usaha
// di kredit (Penjualan), supaya layar Catat cepat menampilkan pemilih barang.
func TestCatat_OpsiTandaiAkunBerbarang(t *testing.T) {
	srv, qa := usahaManufakturPerpetual(t)
	jual, beli, kas, _ := akunCatatBawaan(t, srv, qa)
	o := getJSON[struct {
		Akun []akunCatat `json:"akun"`
	}](t, srv, qa, "/api/v1/catat/opsi")
	tanda := map[string]akunCatat{}
	for _, a := range o.Akun {
		tanda[a.Kode] = a
	}
	if !tanda[beli].BeliBarang || !tanda[jual].JualBarang || tanda[kas].BeliBarang || tanda[kas].JualBarang || tanda[jual].BeliBarang {
		t.Fatalf("tanda akun: beli %+v, jual %+v, kas %+v", tanda[beli], tanda[jual], tanda[kas])
	}
}

// Fix round 1 (R5): pembelian dua barang beda akun persediaan lewat Catat cepat (akun baris =
// akun persediaan bawaan) didebit per akun barangnya; tidak ada penyesuaian akhir bulan untuknya.
func TestCatat_BeliDuaBarangBedaAkunPersediaan(t *testing.T) {
	srv, qa, kain, benang := kainBenangPerpetual(t)
	_, beli, kas, _ := akunCatatBawaan(t, srv, qa)
	c := buat[catatResp](t, srv, qa, "/api/v1/catat", `{"tanggal":"2026-01-05","baris":[{"akun":"`+beli+`","debit":40000},{"akun":"`+kas+`","kredit":40000}],`+
		`"rincian":[{"stok_id":`+itoa(kain.ID)+`,"qty_stok":3000,"nilai":30000},{"stok_id":`+itoa(benang.ID)+`,"qty_stok":2000,"nilai":10000}],"item":"Kain, Benang"}`)
	if got := txsOf(t, srv, qa); len(got) != 1 || got[0].Item != "Kain, Benang" {
		t.Errorf("item transaksi catat = %+v, mau nama barang terpilih", got)
	}
	r := getJSON[coaReport](t, srv, qa, "/api/v1/reports/coa?period=2026-01")
	if got := saldoAkun(r, "1301"); got != 10_000+30_000 {
		t.Errorf("saldo 1301 = %d, harus 40.000", got)
	}
	if got := saldoAkun(r, "1302"); got != 5_000+10_000 {
		t.Errorf("saldo 1302 = %d, harus 15.000", got)
	}
	tn := tenantByEmail(t, srv, qa)
	j, _ := tn.jurnalPeriode("2026-01")
	dr := map[string]int64{}
	for _, e := range j.entri {
		if e.Ket == ketPenyesuaian && (e.Kode == "1301" || e.Kode == "1302") {
			t.Errorf("penyesuaian akhir bulan untuk %s: %+v", e.Kode, e.Line)
		}
		if e.TxID == c.Transaksi[0] && e.Debit > 0 {
			dr[e.Kode] += e.Debit
		}
	}
	if dr["1301"] != 30_000 || dr["1302"] != 10_000 || len(dr) != 2 {
		t.Errorf("debit pembelian per akun %+v, mau 1301 30.000 & 1302 10.000", dr)
	}
}
