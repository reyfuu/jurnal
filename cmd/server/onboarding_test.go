package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"pajak/internal/finance"
	"pajak/internal/xlsx"
)

// workbookUpload membungkus beberapa sheet jadi berkas .xlsx yang siap diunggah.
func workbookUpload(t *testing.T, sheets []finance.ImportSheet) (*bytes.Buffer, string) {
	t.Helper()
	out := make([]xlsx.Sheet, 0, len(sheets))
	for _, sh := range sheets {
		out = append(out, xlsx.Sheet{Name: sh.Nama, Rows: sh.Rows})
	}
	var file bytes.Buffer
	if err := xlsx.Write(&file, out); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "data-awal.xlsx")
	_, _ = fw.Write(file.Bytes())
	_ = mw.Close()
	return &body, mw.FormDataContentType()
}

func TestOnboardingTemplateEndpoint(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	rec := do(srv, owner, http.MethodGet, "/api/v1/onboarding/template", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Disposition"), ".xlsx") {
		t.Fatalf("template: %d %v", rec.Code, rec.Header())
	}
	sheets, err := xlsx.ReadAll(rec.Body.Bytes())
	if err != nil {
		t.Fatalf("template tidak bisa dibaca: %v", err)
	}
	names := make([]string, 0, len(sheets))
	for _, sh := range sheets {
		names = append(names, sh.Name)
	}
	for _, want := range []string{"Saldo Awal", "Aset", "Karyawan", "Transaksi", "Panduan"} {
		if !strings.Contains(strings.Join(names, "|"), want) {
			t.Errorf("sheet %q tidak ada di template; isi: %v", want, names)
		}
	}
}

// Pratinjau tidak boleh menyimpan apa pun; commit mengganti register aset & karyawan
// dan menambahkan transaksi.
func TestOnboardingImport_PratinjauLaluCommit(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	baseIncome := incomeOf(t, srv, owner)

	sheets := []finance.ImportSheet{
		{Nama: "Saldo Awal", Rows: [][]string{
			{"Keterangan", "Nilai"},
			{"Tanggal Mulai Pembukuan", "2026-01-01"},
			{"Kas & Bank", "Rp 45.000.000"},
			{"Piutang Usaha", "8.500.000"},
			{"Persediaan", "8.250.000"},
			{"Utang Usaha", "5.200.000"},
		}},
		{Nama: "Aset", Rows: [][]string{
			{"Nama Aset", "Jenis Harta", "Metode", "Tanggal Perolehan", "Harga Perolehan"},
			{"Oven Deck 2 Pintu", "Peralatan usaha restoran / rumah makan", "Garis Lurus", "2026-02-01", "24000000"},
			{"Pesawat Jet", "Pesawat terbang", "Garis Lurus", "2026-02-01", "1000000"},
		}},
		{Nama: "Karyawan", Rows: [][]string{
			{"Nama", "Jabatan", "Tanggal Masuk", "Gaji Pokok", "Tunjangan", "Status PTKP"},
			{"Dewi Lestari", "Kepala Barista", "2026-01-05", "5500000", "750000", "K/1"},
			{"Bagus Pratama", "Barista", "2026-02-01", "4200000", "500000", "TK/0"},
		}},
		{Nama: "Transaksi", Rows: [][]string{
			{"Tanggal", "Jenis", "Nominal", "Keterangan"},
			{"2026-08-29", "Penjualan", "3.000.000", "Impor onboarding"},
		}},
	}

	body, ctype := workbookUpload(t, sheets)
	rec := upload(srv, owner, "/api/v1/onboarding/import", body, ctype)
	var res onboardingResp
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rec.Code != http.StatusOK || res.Imported {
		t.Fatalf("pratinjau: %d imported=%v", rec.Code, res.Imported)
	}
	if res.Opening == nil || res.Opening.Kas != 45_000_000 {
		t.Fatalf("saldo awal pratinjau: %+v", res.Opening)
	}
	if len(res.Assets) != 1 || len(res.Karyawan) != 2 || res.TransaksiCount != 1 {
		t.Fatalf("isi pratinjau: %d aset, %d karyawan, %d transaksi", len(res.Assets), len(res.Karyawan), res.TransaksiCount)
	}
	if len(res.Errors) != 1 || res.Errors[0].Sheet != "Aset" || res.Errors[0].Baris != 3 {
		t.Fatalf("baris aset tidak dikenal harus dilaporkan: %+v", res.Errors)
	}
	if got := incomeOf(t, srv, owner); got.NetProfit != baseIncome.NetProfit {
		t.Fatal("pratinjau tidak boleh mengubah laporan")
	}

	body, ctype = workbookUpload(t, sheets)
	rec = upload(srv, owner, "/api/v1/onboarding/import?commit=1", body, ctype)
	res = onboardingResp{}
	_ = json.NewDecoder(rec.Body).Decode(&res)
	if rec.Code != http.StatusOK || !res.Imported {
		t.Fatalf("commit: %d %+v", rec.Code, res)
	}
	// Register aset seed (6 baris) diganti oleh 1 aset hasil impor.
	if res.ReplacedAssets != 6 {
		t.Errorf("aset lama yang diganti: want 6, got %d", res.ReplacedAssets)
	}
	as := assetsOf(t, srv, owner)
	if len(as.Assets) != 1 || as.Assets[0].Nama != "Oven Deck 2 Pintu" {
		t.Fatalf("register setelah impor: %+v", as.Assets)
	}
	if as.BebanPenyusutan != 500_000 {
		t.Errorf("penyusutan oven 24jt/48 bulan: want 500_000, got %d", as.BebanPenyusutan)
	}

	var ks finance.KaryawanSummary
	_ = json.NewDecoder(do(srv, owner, http.MethodGet, "/api/v1/karyawan", "").Body).Decode(&ks)
	if ks.TotalAktif != 2 || ks.BebanGajiBulanan != 10_950_000 {
		t.Errorf("karyawan: %d aktif, beban %d", ks.TotalAktif, ks.BebanGajiBulanan)
	}

	// Saldo awal onboarding menggantikan saldo demo, dan neraca tetap seimbang.
	var op openingResp
	_ = json.NewDecoder(do(srv, owner, http.MethodGet, "/api/v1/opening", "").Body).Decode(&op)
	if !op.Aktif || op.Opening.Kas != 45_000_000 {
		t.Fatalf("neraca awal tidak aktif: %+v", op)
	}
	var bs finance.BalanceSheet
	_ = json.NewDecoder(do(srv, owner, http.MethodGet, "/api/v1/reports/balance-sheet?period=2026-08", "").Body).Decode(&bs)
	if !bs.IsBalanced {
		t.Errorf("neraca tidak seimbang setelah onboarding: %+v", bs)
	}
	if bs.Assets.Inventory != 8_250_000 {
		t.Errorf("persediaan dari neraca awal: got %d", bs.Assets.Inventory)
	}
}

func TestOpeningEndpoint_ManualDanSelisihModal(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")

	var op openingResp
	_ = json.NewDecoder(do(srv, owner, http.MethodGet, "/api/v1/opening", "").Body).Decode(&op)
	if op.Aktif {
		t.Fatal("sebelum diisi, neraca awal harus belum aktif")
	}

	// ModalDisetor sengaja salah: harus dilaporkan sebagai selisih, bukan diterima diam-diam.
	rec := do(srv, owner, http.MethodPut, "/api/v1/opening",
		`{"tanggal":"2026-01-01","kas":45000000,"piutang":8500000,"persediaan":8250000,"utang":5200000,"modal_disetor":10000000}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("put opening: %d %s", rec.Code, rec.Body)
	}
	_ = json.NewDecoder(rec.Body).Decode(&op)
	if !op.Aktif || op.SelisihModal == 0 {
		t.Fatalf("selisih modal harus dilaporkan: %+v", op)
	}
	// Aset seed sudah ada, jadi modal awal ikut memuat nilai bukunya; yang penting selisihnya terlaporkan.
	if op.ModalAwal <= 0 {
		t.Errorf("modal awal harus terhitung: %d", op.ModalAwal)
	}

	if rec := do(srv, owner, http.MethodPut, "/api/v1/opening", `{"tanggal":"01-01-2026"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("tanggal salah format: want 400, got %d", rec.Code)
	}
	if rec := do(srv, owner, http.MethodPut, "/api/v1/opening", `{"tanggal":"2026-01-01","kas":-5}`); rec.Code != http.StatusBadRequest {
		t.Errorf("kas negatif: want 400, got %d", rec.Code)
	}
}

func TestKaryawanCRUD(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")

	rec := do(srv, owner, http.MethodPost, "/api/v1/karyawan",
		`{"nama":"Dewi Lestari","jabatan":"Barista","tanggal_masuk":"2026-01-05","gaji_pokok":5500000,"tunjangan":750000,"ptkp":"K/1","aktif":true}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	var k finance.Karyawan
	_ = json.NewDecoder(rec.Body).Decode(&k)

	path := "/api/v1/karyawan/" + strconv.Itoa(k.ID)
	rec = do(srv, owner, http.MethodPut, path,
		`{"nama":"Dewi Lestari","jabatan":"Kepala Barista","tanggal_masuk":"2026-01-05","gaji_pokok":6000000,"tunjangan":750000,"ptkp":"K/1","aktif":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec.Code, rec.Body)
	}
	var ks finance.KaryawanSummary
	_ = json.NewDecoder(do(srv, owner, http.MethodGet, "/api/v1/karyawan", "").Body).Decode(&ks)
	if ks.BebanGajiBulanan != 6_750_000 {
		t.Errorf("beban gaji setelah update: want 6_750_000, got %d", ks.BebanGajiBulanan)
	}

	if rec := do(srv, owner, http.MethodDelete, path, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}
	if rec := do(srv, owner, http.MethodDelete, path, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("delete dua kali: want 404, got %d", rec.Code)
	}
	if rec := do(srv, owner, http.MethodPost, "/api/v1/karyawan", `{"nama":"","tanggal_masuk":"2026-01-05","gaji_pokok":1}`); rec.Code != http.StatusBadRequest {
		t.Errorf("nama kosong: want 400, got %d", rec.Code)
	}
}

// Impor data awal & neraca awal hanya untuk yang boleh mengubah data; Kasir tertutup.
func TestOnboarding_RoleAccess(t *testing.T) {
	srv := NewServer()
	kasir := login(t, srv, "kasir@kedaikopisenja.id")
	admin := login(t, srv, "admin@kedaikopisenja.id")

	cases := []struct {
		c      *http.Cookie
		method string
		path   string
		want   int
	}{
		{kasir, http.MethodGet, "/api/v1/onboarding/template", http.StatusForbidden},
		{kasir, http.MethodGet, "/api/v1/karyawan", http.StatusForbidden},
		{kasir, http.MethodGet, "/api/v1/opening", http.StatusForbidden},
		{admin, http.MethodGet, "/api/v1/onboarding/template", http.StatusOK},
		{admin, http.MethodGet, "/api/v1/karyawan", http.StatusOK},
		{admin, http.MethodGet, "/api/v1/opening", http.StatusOK},
	}
	for _, tc := range cases {
		if rec := do(srv, tc.c, tc.method, tc.path, ""); rec.Code != tc.want {
			t.Errorf("%s %s: want %d, got %d", tc.method, tc.path, tc.want, rec.Code)
		}
	}
}

// Tanggal saldo awal tidak boleh setelah transaksi yang sudah dicatat: transaksi itu sudah
// tercermin di saldo awal, jadi akan terhitung dua kali.
func TestSaldoAwal_TidakBolehSetelahTransaksiPertama(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	info := getJSON[openingResp](t, srv, owner, "/api/v1/opening")
	if info.TransaksiPertama == "" {
		t.Fatal("demo punya transaksi; tanggal pertama harus terisi")
	}
	telat := `{"tanggal":"2099-01-01","kas":1000}`
	if rec := do(srv, owner, http.MethodPut, "/api/v1/opening", telat); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), info.TransaksiPertama) {
		t.Fatalf("tanggal sesudah transaksi pertama: %d %s", rec.Code, rec.Body)
	}
	pas := `{"tanggal":"` + info.TransaksiPertama + `","kas":1000}`
	if rec := do(srv, owner, http.MethodPut, "/api/v1/opening", pas); rec.Code != http.StatusOK {
		t.Fatalf("tanggal = transaksi pertama: %d %s", rec.Code, rec.Body)
	}
}

// Template yang diunduh (tabel bergaris, rumus ringkasan, kunci kolom) diunggah apa adanya:
// harus lolos, saldo awalnya seimbang, dan neraca bulan pertama memuat semua posnya.
func TestTemplateDataAwal_UnduhUnggahNeracaSeimbang(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	rec := do(srv, owner, http.MethodGet, "/api/v1/onboarding/template", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("template: %d", rec.Code)
	}
	berkas := rec.Body.Bytes()
	for _, isi := range []string{"sheetProtection", "dataValidation", "SUMIF"} {
		if !bytes.Contains(unzipSemua(t, berkas), []byte(isi)) {
			t.Errorf("template tidak memuat %s", isi)
		}
	}
	kirim := func(commit bool) (int, onboardingResp, string) {
		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		fw, _ := mw.CreateFormFile("file", "template-data-awal.xlsx")
		_, _ = fw.Write(berkas)
		_ = mw.Close()
		url := "/api/v1/onboarding/import"
		if commit {
			url += "?commit=1"
		}
		r := upload(srv, owner, url, &body, mw.FormDataContentType())
		var res onboardingResp
		raw := r.Body.String()
		_ = json.Unmarshal([]byte(raw), &res)
		return r.Code, res, raw
	}
	if code, res, raw := kirim(false); code != http.StatusOK || !res.BisaDisimpan || len(res.Errors) != 0 || res.NilaiBukuAset == 0 {
		t.Fatalf("pratinjau: %d %s", code, raw)
	}
	if code, _, raw := kirim(true); code != http.StatusOK {
		t.Fatalf("commit: %d %s", code, raw)
	}
	bs := getJSON[finance.BalanceSheet](t, srv, owner, "/api/v1/reports/balance-sheet?period=2026-01")
	if !bs.IsBalanced || bs.Assets.OtherAssets != 6_000_000 || bs.Liabilities.Other != 450_000 || bs.Equity.RetainedEarnings != 10_000_000 {
		t.Fatalf("neraca Januari: %+v", bs)
	}
	info := getJSON[openingResp](t, srv, owner, "/api/v1/opening")
	if info.Opening == nil || bs.Equity.OpeningCapital != info.Opening.ModalDisetor || info.SelisihModal != 0 {
		t.Fatalf("modal awal %d, modal disetor %+v", bs.Equity.OpeningCapital, info)
	}
}

// Saldo awal per akun lewat menu Saldo Awal: tidak seimbang ditolak, seimbang diterima.
func TestSaldoAwalPerAkun_HarusSeimbang(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	info := getJSON[openingResp](t, srv, owner, "/api/v1/opening?tanggal=2026-01-01")
	nbv := strconv.FormatInt(info.NilaiBukuAset, 10)
	baris := func(modal string) string {
		return `{"tanggal":"2026-01-01","rincian":[` +
			`{"nama":"Bank","kategori":"kas","nominal":20000000},` +
			`{"nama":"Aset tetap","kategori":"aset_tetap","nominal":` + nbv + `},` +
			`{"nama":"Hutang Bank","kategori":"pinjaman","nominal":5000000},` +
			`{"nama":"Modal Pemilik","kategori":"modal","nominal":` + modal + `}]}`
	}
	if rec := do(srv, owner, http.MethodPut, "/api/v1/opening", baris("1")); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "belum seimbang") {
		t.Fatalf("tidak seimbang: %d %s", rec.Code, rec.Body)
	}
	modal := strconv.FormatInt(20_000_000+info.NilaiBukuAset-5_000_000, 10)
	if rec := do(srv, owner, http.MethodPut, "/api/v1/opening", baris(modal)); rec.Code != http.StatusOK {
		t.Fatalf("seimbang: %d %s", rec.Code, rec.Body)
	}
	if bs := getJSON[finance.BalanceSheet](t, srv, owner, "/api/v1/reports/balance-sheet?period=2026-01"); !bs.IsBalanced {
		t.Fatalf("neraca: %+v", bs)
	}
}

func unzipSemua(t *testing.T, b []byte) []byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	for _, f := range zr.File {
		rc, _ := f.Open()
		_, _ = io.Copy(&out, rc)
		_ = rc.Close()
	}
	return out.Bytes()
}
