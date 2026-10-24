package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"pajak/internal/coa"
	"pajak/internal/finance"
)

// Saldo awal yang tidak cocok dengan rinciannya diperingatkan (bukan diam-diam jadi modal):
// aset tetap vs daftar aset, persediaan vs daftar barang; muncul juga di Langkah memulai.
func TestCekSaldoAwal_SelisihDiperingatkan(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	saldo := `{"tanggal":"2026-01-01","rincian":[
		{"nama":"Bank","kategori":"kas","nominal":20000000},
		{"nama":"Persediaan","kategori":"persediaan","nominal":5000000},
		{"nama":"Modal Pemilik","kategori":"modal","nominal":25000000}]}`
	if rec := do(srv, owner, http.MethodPut, "/api/v1/opening", saldo); rec.Code != http.StatusOK {
		t.Fatalf("saldo awal: %d %s", rec.Code, rec.Body)
	}
	cek := func() map[string]cekKecocokan {
		out := map[string]cekKecocokan{}
		for _, c := range getJSON[[]cekKecocokan](t, srv, owner, "/api/v1/saldo-awal/cek") {
			out[c.Kode] = c
		}
		return out
	}
	c := cek()
	if c["aset"].Status != "ok" {
		t.Fatalf("aset awal harus cocok (0 = 0): %+v", c["aset"])
	}
	nilaiStok := finance.SummarizeStok(srv.tenants[1].stok).NilaiPersediaan
	if p := c["persediaan"]; p.SaldoAwal != 5_000_000 || (p.Status == "selisih") != (p.Pembanding != 5_000_000) {
		t.Fatalf("persediaan: %+v (nilai daftar barang %d)", p, nilaiStok)
	}
	// Aset dibeli sebelum tanggal saldo awal ditambahkan belakangan: saldo awal tidak lagi cocok.
	if rec := do(srv, owner, http.MethodPost, "/api/v1/assets", `{"nama":"Laptop lama","kelompok":"k1","tanggal_perolehan":"2025-01-01","harga_perolehan":15000000}`); rec.Code != http.StatusCreated {
		t.Fatalf("aset: %d %s", rec.Code, rec.Body)
	}
	c = cek()
	if a := c["aset"]; a.Status != "selisih" || a.Pembanding != 11_250_000 || a.Selisih != -11_250_000 || !strings.Contains(a.Pesan, "Rp11.250.000") {
		t.Fatalf("aset selisih: %+v", a)
	}
	if s := c["seimbang"]; s.Status != "selisih" {
		t.Fatalf("harta = kewajiban + modal harus ikut diperingatkan: %+v", s)
	}
	var adaDiLangkah bool
	for _, l := range getJSON[[]langkahMulai](t, srv, owner, "/api/v1/mulai") {
		if l.Kode == "aset" && l.Peringatan != "" {
			adaDiLangkah = true
		}
	}
	if !adaDiLangkah {
		t.Fatal("langkah Daftarkan aset tetap harus memuat peringatan selisih")
	}
}

// Nama akun yang diketik langsung mendapat kategori & kode: akun CoA yang sama memakai kodenya,
// akun baru ("Bank BCA") mendapat kode berikutnya di kelompoknya; aset tetap datang per akun
// dari daftar aset.
func TestSaranSaldo_KategoriDanKodeOtomatis(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	saran := func(body string) saranSaldo {
		rec := do(srv, owner, http.MethodPost, "/api/v1/saldo-awal/saran", body)
		if rec.Code != http.StatusOK {
			t.Fatalf("saran: %d %s", rec.Code, rec.Body)
		}
		var s saranSaldo
		if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	if s := saran(`{"nama":"Hutang Bank"}`); s.Kategori != "pinjaman" || s.Kode != "2102" || s.Baru {
		t.Fatalf("akun CoA yang sama: %+v", s)
	}
	if s := saran(`{"nama":"Bank BCA","dipakai":["1105"]}`); s.Kategori != "kas" || s.Kode != "1106" || !s.Baru || s.Sumber != "coa" {
		t.Fatalf("rekening baru: %+v", s)
	}
	if s := saran(`{"nama":"Tabungan pensiun","kategori":"aset_lain"}`); s.Kategori != "aset_lain" || s.Kode != "1408" {
		t.Fatalf("kategori dipilih sendiri: %+v", s)
	}
	if s := saran(`{"nama":"Pinjaman koperasi"}`); s.Kategori != "pinjaman" || s.Kode == "" {
		t.Fatalf("kata kunci (AI tidak aktif di tes): %+v", s)
	}

	if rec := do(srv, owner, http.MethodPost, "/api/v1/assets", `{"nama":"Motor box","jenis_harta":"kendaraan","kelompok":"k1","tanggal_perolehan":"2025-01-01","harga_perolehan":12000000}`); rec.Code != http.StatusCreated {
		t.Fatalf("aset: %d %s", rec.Code, rec.Body)
	}
	o := getJSON[openingResp](t, srv, owner, "/api/v1/opening?tanggal=2026-01-01")
	if len(o.AsetTetap) == 0 || o.AsetTetap[0].Kode != "1505" || o.AsetTetap[0].Nominal != o.NilaiBukuAset-sumLain(o.AsetTetap[1:]) {
		t.Fatalf("aset tetap per akun: %+v (nilai buku %d)", o.AsetTetap, o.NilaiBukuAset)
	}
	for _, b := range o.AkunTemplate {
		if b.Kategori == "aset_tetap" || b.Kode == "" {
			t.Fatalf("template saldo awal: baris %+v", b)
		}
	}
}

func sumLain(bs []finance.SaldoAkun) (n int64) {
	for _, b := range bs {
		n += b.Nominal
	}
	return n
}

// Akun yang ditambahkan sendiri di Saldo Awal (rekening bank lain, uang jaminan) menjadi akun
// tersendiri di Daftar Akun dan Buku Besar, saldo kas Kas berkurang sebesar itu (total tetap),
// neraca saldo tetap seimbang, dan akun itu tidak bisa dihapus selama dipakai transaksi.
func TestSaldoAwal_AkunSendiriDiDaftarAkunDanBukuBesar(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	if rec := do(srv, owner, http.MethodPut, "/api/v1/business", strings.Replace(profileBody, "manufaktur", "perdagangan", 1)); rec.Code != http.StatusOK {
		t.Fatalf("profil: %d %s", rec.Code, rec.Body)
	}
	saldo := `{"tanggal":"2026-01-01","rincian":[
		{"nama":"Kas","kategori":"kas","nominal":5000000},
		{"nama":"Bank BCA","kategori":"kas","nominal":10000000},
		{"nama":"Uang Jaminan","kategori":"aset_lain","nominal":2000000},
		{"nama":"Modal Pemilik","kategori":"modal","nominal":17000000}]}`
	if rec := do(srv, owner, http.MethodPut, "/api/v1/opening", saldo); rec.Code != http.StatusOK {
		t.Fatalf("saldo awal: %d %s", rec.Code, rec.Body)
	}
	op := getJSON[openingResp](t, srv, owner, "/api/v1/opening")
	kode := map[string]string{}
	for _, r := range op.Opening.Rincian {
		kode[r.Nama] = r.Kode
	}
	if kode["Kas"] != "1101" || kode["Bank BCA"] != "1105" || kode["Uang Jaminan"] != "1408" {
		t.Fatalf("kode akun saldo awal: %v", kode)
	}
	daftar := getJSON[struct {
		Accounts []coa.Account `json:"accounts"`
	}](t, srv, owner, "/api/v1/coa")
	for _, k := range []string{"1105", "1408"} {
		if a, ok := coa.Find(daftar.Accounts, k); !ok || a.Catatan != coa.CatatanSaldoAwal {
			t.Errorf("Daftar Akun harus memuat %s: %+v", k, a)
		}
	}
	bb := getJSON[bukuBesarResp](t, srv, owner, "/api/v1/reports/buku-besar?dari=2026-01-01&sampai=2026-01-31")
	awal := map[string]int64{}
	for _, a := range bb.Akun {
		awal[a.Kode] = a.SaldoAwal
	}
	if awal["1105"] != 10_000_000 || awal["1101"] != 5_000_000 || awal["1408"] != 2_000_000 || !bb.Seimbang {
		t.Fatalf("saldo awal Buku Besar: 1101 %d, 1105 %d, 1408 %d, seimbang %v", awal["1101"], awal["1105"], awal["1408"], bb.Seimbang)
	}
	for _, p := range []string{"2026-01", "2026-08"} {
		var d, k int64
		for _, b := range getJSON[coaReport](t, srv, owner, "/api/v1/reports/coa?period="+p).TrialBalance {
			d, k = d+b.Debit, k+b.Kredit
		}
		if d != k {
			t.Errorf("%s: neraca saldo tidak seimbang %d vs %d", p, d, k)
		}
	}
	srv.tenants[1].txs[0].Akun = "1105" // transaksi dikoreksi ke rekening itu
	tanpaBCA := strings.Replace(saldo, `{"nama":"Bank BCA","kategori":"kas","nominal":10000000},`, `{"nama":"Bank","kategori":"kas","nominal":10000000},`, 1)
	if rec := do(srv, owner, http.MethodPut, "/api/v1/opening", tanpaBCA); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "Bank BCA") {
		t.Fatalf("akun yang dipakai transaksi tidak boleh hilang: %d %s", rec.Code, rec.Body)
	}
}

// Akun persediaan tambahan klien (turunan 1301) ikut dikenali pemeriksaan per akun: saldo awal
// yang sama dengan daftar barang per akun tidak memunculkan selisih palsu.
func TestCekSaldoAwal_PersediaanAkunTambahanKlien(t *testing.T) {
	srv, qa := usahaManufaktur(t)
	rec := do(srv, qa, http.MethodPost, "/api/v1/coa/akun", `{"nama":"Persediaan Kemasan","induk":"1301"}`)
	var akun finance.AkunTambahan
	if err := json.Unmarshal(rec.Body.Bytes(), &akun); err != nil || rec.Code != http.StatusOK || akun.Kode == "" {
		t.Fatalf("akun tambahan: %d %s", rec.Code, rec.Body)
	}
	buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kain","kategori":"Bahan Baku","satuan":"m","jumlah":10,"harga_pokok":10000}`)
	buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Plastik","kategori":"Bahan Penolong","satuan":"pcs","jumlah":4,"harga_pokok":5000,"akun":"`+akun.Kode+`"}`)
	saldo := `{"tanggal":"2026-01-01","rincian":[
		{"nama":"Kas","kategori":"kas","nominal":880000},
		{"kode":"1301","nama":"Persediaan Bahan Baku","kategori":"persediaan","nominal":100000},
		{"kode":"` + akun.Kode + `","nama":"Persediaan Kemasan","kategori":"persediaan","nominal":20000},
		{"nama":"Modal Pemilik","kategori":"modal","nominal":1000000}]}`
	if rec := do(srv, qa, http.MethodPut, "/api/v1/opening", saldo); rec.Code != http.StatusOK {
		t.Fatalf("saldo awal: %d %s", rec.Code, rec.Body)
	}
	var persediaan []cekKecocokan
	for _, c := range getJSON[[]cekKecocokan](t, srv, qa, "/api/v1/saldo-awal/cek") {
		if strings.HasPrefix(c.Kode, "persediaan") {
			persediaan = append(persediaan, c)
		}
	}
	if len(persediaan) != 1 || persediaan[0].Kode != "persediaan" || persediaan[0].Status != "ok" {
		t.Fatalf("persediaan harus satu entri ok: %+v", persediaan)
	}
}
