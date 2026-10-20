package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"pajak/internal/auth"
	"pajak/internal/coa"
	"pajak/internal/finance"
	"pajak/internal/katalog"
	"pajak/internal/tax"
)

// pgTest: tes Postgres hanya jalan bila TEST_DATABASE_URL di-set; skema dikosongkan dulu.
func pgTest(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL tidak di-set")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATA_DIR", "")
	t.Setenv("DATABASE_URL", url)
	return url
}

// fill mengisi semua field dengan nilai bukan-nol, supaya field baru tanpa kolom ketahuan.
func fill(v reflect.Value, n int) {
	switch v.Kind() {
	case reflect.Pointer:
		v.Set(reflect.New(v.Type().Elem()))
		fill(v.Elem(), n)
	case reflect.Struct:
		if v.Type() == reflect.TypeFor[time.Time]() {
			v.Set(reflect.ValueOf(time.Date(2026, 10, 20, 8, 30, n, 0, time.UTC)))
			return
		}
		for i := range v.NumField() {
			f := v.Type().Field(i)
			if !f.IsExported() {
				continue
			}
			if f.Type.Kind() == reflect.String && (strings.HasPrefix(f.Name, "Tanggal") || f.Name == "JatuhTempo") {
				v.Field(i).SetString("2026-08-2" + string(rune('0'+n%10)))
				continue
			}
			fill(v.Field(i), n+i)
		}
	case reflect.Slice:
		s := reflect.MakeSlice(v.Type(), 2, 2)
		fill(s.Index(0), n+1)
		fill(s.Index(1), n+2)
		v.Set(s)
	case reflect.String:
		v.SetString("isi-" + string(rune('a'+n%26)))
	case reflect.Int, reflect.Int64:
		v.SetInt(int64(1000 + n))
	case reflect.Bool:
		v.SetBool(true)
	}
}

func filled[T any](n int) T {
	var x T
	fill(reflect.ValueOf(&x).Elem(), n)
	return x
}

func jsonOf(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// normTimes: Postgres menyimpan waktu sampai mikrodetik dan mengembalikan zona sesi.
func normTimes(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			normTimes(v.Elem())
		}
	case reflect.Struct:
		if tm, ok := v.Interface().(time.Time); ok {
			v.Set(reflect.ValueOf(tm.UTC().Round(time.Microsecond)))
			return
		}
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				normTimes(v.Field(i))
			}
		}
	case reflect.Slice:
		for i := range v.Len() {
			normTimes(v.Index(i))
		}
	}
}

// normJSON: salinan v (lewat JSON, data asli tidak disentuh) dengan waktu dinormalkan.
func normJSON[T any](t *testing.T, v T) string {
	t.Helper()
	var c T
	if err := json.Unmarshal([]byte(jsonOf(t, v)), &c); err != nil {
		t.Fatal(err)
	}
	normTimes(reflect.ValueOf(&c).Elem())
	return jsonOf(t, c)
}

type tenantData struct {
	Txs                                                  []finance.Transaction
	Assets                                               []finance.Asset
	Opening                                              *finance.OpeningBalance
	Karyawan                                             []finance.Karyawan
	Invoices                                             []finance.Invoice
	Profile                                              *finance.BusinessProfile
	Langganan                                            langganan
	Tagihan                                              []finance.Tagihan
	Stok                                                 []finance.StokItem
	Produk                                               []finance.Produk
	Supplier                                             []finance.Supplier
	Koreksi                                              []finance.Koreksi
	NextTxID, NextAssetID, NextKaryawanID, NextInvoiceID int
	NextTagihanID, NextStokID                            int
	NextProdukID, NextSupplierID, NextKoreksiID          int
	CoaTemplate                                          int
	Arsip                                                time.Time
	ArsipAlasan                                          string
	Catatan                                              []finance.CatatanKonsultan
	NextCatatanID                                        int
}

// tenantJSON: semua data usaha yang disimpan, dalam bentuk yang bisa dibandingkan.
func tenantJSON(t *testing.T, tn *tenant) string {
	t.Helper()
	tn.mu.RLock()
	d := tenantData{tn.txs, tn.assets, tn.opening, tn.karyawan, tn.invoices, tn.profile, tn.langganan,
		tn.tagihan, tn.stok, tn.produk, tn.supplier, tn.koreksi,
		tn.nextTxID, tn.nextAssetID, tn.nextKaryawanID, tn.nextInvoiceID,
		tn.nextTagihanID, tn.nextStokID, tn.nextProdukID, tn.nextSupplierID, tn.nextKoreksiID, tn.coaTemplateID, tn.arsip, tn.arsipAlasan, tn.catatan, tn.nextCatatanID}
	tn.mu.RUnlock()
	return normJSON(t, d)
}

func authJSON(t *testing.T, srv *Server) string {
	snap := srv.users.Snapshot()
	slices.SortFunc(snap.Sessions, func(a, b auth.SnapshotSession) int { return strings.Compare(a.Token, b.Token) })
	return normJSON(t, snap)
}

func ptr[T any](v T) *T { return &v }

func TestPostgresSimpanMuatSemuaField(t *testing.T) {
	pgTest(t)
	srv := NewServer()
	qa := register(t, srv, registerBody)
	tn := tenantByEmail(t, srv, qa)

	tn.mu.Lock()
	tn.txs = []finance.Transaction{filled[finance.Transaction](1), filled[finance.Transaction](2)}
	tn.txs[0].ID, tn.txs[1].ID = 1, 2
	tn.assets = []finance.Asset{filled[finance.Asset](3)}
	tn.karyawan = []finance.Karyawan{filled[finance.Karyawan](4)}
	tn.invoices = []finance.Invoice{filled[finance.Invoice](5), filled[finance.Invoice](6)}
	tn.invoices[0].ID, tn.invoices[1].ID = 1, 2
	tn.opening = ptr(filled[finance.OpeningBalance](7))
	tn.profile = ptr(filled[finance.BusinessProfile](8))
	tn.langganan = filled[langganan](9)
	tn.langganan.Orders[0].ID, tn.langganan.Orders[1].ID = "LB-2-1", "LB-2-2"
	tn.tagihan = []finance.Tagihan{filled[finance.Tagihan](10), filled[finance.Tagihan](11)}
	tn.tagihan[0].ID, tn.tagihan[1].ID = 1, 2
	tn.tagihan[0].Jenis, tn.tagihan[1].Jenis = finance.TagihanPiutang, finance.TagihanUtang
	tn.stok = []finance.StokItem{filled[finance.StokItem](12)}
	tn.stok[0].ID = 1
	// Produk dengan resep: bahannya tabel anak (produk_bahan), jadi urutan & isinya ikut diuji.
	tn.produk = []finance.Produk{filled[finance.Produk](13)}
	tn.produk[0].ID = 1
	tn.supplier = []finance.Supplier{filled[finance.Supplier](14)}
	tn.supplier[0].ID = 1
	// Usulan koreksi: potret & usulan transaksi disimpan sebagai JSONB.
	tn.koreksi = []finance.Koreksi{filled[finance.Koreksi](15)}
	tn.koreksi[0].ID = 1
	tn.nextKoreksiID = 2
	tn.catatan = []finance.CatatanKonsultan{filled[finance.CatatanKonsultan](16)}
	tn.catatan[0].ID = 1
	tn.nextCatatanID = 2
	tn.coaTemplateID = 1
	tn.nextTxID, tn.nextAssetID, tn.nextKaryawanID, tn.nextInvoiceID = 3, 2, 2, 3
	tn.nextTagihanID, tn.nextStokID, tn.nextProdukID, tn.nextSupplierID = 3, 2, 2, 2
	tn.mu.Unlock()
	srv.saveFor(tn)
	if rec := postBug(srv, qa, map[string]string{"judul": "Bug PG", "deskripsi": "d", "halaman": "dashboard"}, nil); rec.Code != http.StatusCreated {
		t.Fatalf("lapor bug: %d %s", rec.Code, rec.Body)
	}
	// diarsipkan terakhir: usaha yang diarsip menolak permintaan pemiliknya (termasuk lapor bug)
	tn.mu.Lock()
	tn.arsip, tn.arsipAlasan = time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC), "uji arsip"
	tn.mu.Unlock()
	srv.saveFor(tn)

	srv2 := NewServer() // muat ulang dari Postgres
	if got, want := tenantJSON(t, srv2.tenants[tn.id]), tenantJSON(t, tn); got != want {
		t.Fatalf("data usaha berubah setelah simpan-muat:\n got %s\nwant %s", got, want)
	}
	if got, want := tenantJSON(t, srv2.tenants[1]), tenantJSON(t, srv.tenants[1]); got != want {
		t.Fatal("data usaha demo berubah setelah simpan-muat")
	}
	if got, want := authJSON(t, srv2), authJSON(t, srv); got != want {
		t.Fatalf("user & sesi berubah:\n got %s\nwant %s", got, want)
	}
	if got, want := normJSON(t, srv2.bugs.list), normJSON(t, srv.bugs.list); got != want {
		t.Fatalf("laporan bug berubah:\n got %s\nwant %s", got, want)
	}
}

func TestPostgresLewatAPI(t *testing.T) {
	pgTest(t)
	srv := NewServer()
	qa := register(t, srv, registerBody) // tenant dari cookie respons
	if rec := do(srv, qa, http.MethodPut, "/api/v1/business", profileBody); rec.Code != http.StatusOK {
		t.Fatalf("profil: %d", rec.Code)
	}
	aktifkan(t, srv, qa)
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-08-20","jenis":"Penjualan","nominal":5000,"keterangan":"Punya QA","lunas":true}`); rec.Code != http.StatusCreated {
		t.Fatalf("create tx: %d %s", rec.Code, rec.Body)
	}
	owner := login(t, srv, "owner@kedaikopisenja.id")
	if rec := do(srv, owner, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-08-21","jenis":"Penjualan","nominal":7000,"keterangan":"Punya demo","lunas":true}`); rec.Code != http.StatusCreated {
		t.Fatalf("create tx demo: %d", rec.Code)
	}

	srv2 := NewServer()
	body := do(srv2, qa, http.MethodGet, "/api/v1/transactions", "").Body.String()
	if !strings.Contains(body, "Punya QA") || strings.Contains(body, "Punya demo") {
		t.Fatalf("transaksi usaha baru salah setelah restart: %s", body)
	}
	if !strings.Contains(do(srv2, owner, http.MethodGet, "/api/v1/transactions", "").Body.String(), "Punya demo") {
		t.Fatal("transaksi usaha demo hilang setelah restart")
	}
	if !strings.Contains(do(srv2, qa, http.MethodGet, "/api/v1/business", "").Body.String(), "Kedai Kopi Senja") {
		t.Fatal("profil usaha hilang setelah restart")
	}
	if st := tenantByEmail(t, srv2, qa).langgananStatus(time.Now()).Status; st != "aktif" {
		t.Fatalf("langganan harus tetap aktif, got %s", st)
	}
}

func TestPostgresImporDariJSON(t *testing.T) {
	url := pgTest(t)
	dir := t.TempDir()
	t.Setenv("DATA_DIR", dir)
	t.Setenv("DATABASE_URL", "") // versi lama: JSON
	srv := NewServer()
	qa := register(t, srv, registerBody)
	aktifkan(t, srv, qa)
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-08-20","jenis":"Penjualan","nominal":5000,"keterangan":"Dari JSON","lunas":true}`); rec.Code != http.StatusCreated {
		t.Fatalf("create tx: %d", rec.Code)
	}

	t.Setenv("DATABASE_URL", url) // deploy versi Postgres: DB kosong → impor
	srv2 := NewServer()
	if !strings.Contains(do(srv2, qa, http.MethodGet, "/api/v1/transactions", "").Body.String(), "Dari JSON") {
		t.Fatal("transaksi tidak terimpor dari JSON")
	}
	var n int
	if err := srv2.db.QueryRow(`SELECT count(*) FROM transactions WHERE tenant_id = 2`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("baris transaksi usaha 2 di DB: %d %v", n, err)
	}

	t.Setenv("DATA_DIR", t.TempDir()) // JSON tidak dibaca lagi setelah DB terisi
	srv3 := NewServer()
	if !strings.Contains(do(srv3, qa, http.MethodGet, "/api/v1/transactions", "").Body.String(), "Dari JSON") {
		t.Fatal("data harus dimuat dari DB")
	}
}

// Tim, role, audit, dan status/balasan tiket Panel Master tersimpan utuh di Postgres.
func TestPostgresPanelMaster(t *testing.T) {
	pgTest(t)
	srv, m := serverMaster(t)
	qa := register(t, srv, registerBody)
	postBug(srv, qa, map[string]string{"judul": "Bug PG", "deskripsi": "d"}, nil)
	if rec := do(srv, m, http.MethodPost, "/api/v1/master/roles", `{"nama":"Akuntan Senior","izin":["klien_lihat","tiket"]}`); rec.Code != http.StatusCreated {
		t.Fatalf("role: %d %s", rec.Code, rec.Body)
	}
	tambahAnggota(t, srv, m, "budi@masterpajak.id", "akuntan-senior", []int{2})
	if rec := do(srv, m, http.MethodPut, "/api/v1/master/tiket/1", `{"status":"selesai","balasan":"Sudah diperbaiki."}`); rec.Code != http.StatusOK {
		t.Fatalf("balas: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, m, http.MethodPost, "/api/v1/master/klien/2/langganan", `{"aksi":"aktifkan","paket":"bulanan","alasan":"Transfer"}`); rec.Code != http.StatusOK {
		t.Fatalf("langganan: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, m, http.MethodPut, "/api/v1/master/pengingat", `{"aktif":true,"interval_hari":5,"tingkat_min":"penting"}`); rec.Code != http.StatusOK {
		t.Fatalf("pengingat: %d %s", rec.Code, rec.Body)
	}

	srv2 := NewServer()
	if got, want := normJSON(t, srv2.master), normJSON(t, srv.master); got != want {
		t.Fatalf("Panel Master berubah setelah simpan-muat:\n got %s\nwant %s", got, want)
	}
	if got, want := normJSON(t, srv2.bugs.list), normJSON(t, srv.bugs.list); got != want {
		t.Fatalf("tiket berubah:\n got %s\nwant %s", got, want)
	}
	if got, want := authJSON(t, srv2), authJSON(t, srv); got != want {
		t.Fatalf("user berubah (last_seen / akun tim):\n got %s\nwant %s", got, want)
	}
	if a := srv2.master.AturanPengingat(); a.IntervalHari != 5 || a.TingkatMin != "penting" {
		t.Fatalf("pengingat setelah muat ulang = %+v", a)
	}
	if st := tenantByEmail(t, srv2, qa).langgananStatus(time.Now()).Status; st != "aktif" {
		t.Fatalf("aktivasi manual hilang: %s", st)
	}
	// Audit baru setelah muat ulang melanjutkan nomor, tidak menimpa.
	m2 := loginPass(t, srv2, "andi@masterpajak.id", masterPass)
	do(srv2, m2, http.MethodGet, "/api/v1/master/klien/2", "")
	srv3 := NewServer()
	if n, last := len(srv3.master.Audit), srv3.master.Audit[len(srv3.master.Audit)-1]; n != len(srv.master.Audit)+1 || last.ID != n || last.Aksi != "Buka detail klien" {
		t.Fatalf("audit setelah muat ulang: %d baris, terakhir %+v", n, last)
	}
}

// Versi template COA & benchmark buatan tim tersimpan utuh; versi 1 tetap mengikuti kode.
func TestPostgresKatalog(t *testing.T) {
	t.Cleanup(func() { tax.AktifkanBenchmark(tax.KLUBenchmarks, tax.SumberBenchmark) })
	pgTest(t)
	srv, m := serverMaster(t)
	baru := append(slices.Clone(coa.Master), coa.Account{Kode: "6119", Nama: "Beban Asuransi", Subtype: "opex", Normal: coa.Debit, Kelompok: coa.BebanOperasional, Kondisi: coa.Selalu})
	body, _ := json.Marshal(map[string]any{"jenis": "perdagangan", "akun": baru})
	do(srv, m, http.MethodPost, "/api/v1/master/coa/draft", string(body))
	if rec := do(srv, m, http.MethodPost, "/api/v1/master/coa/2/terbitkan", ""); rec.Code != http.StatusOK {
		t.Fatalf("terbit: %d %s", rec.Code, rec.Body)
	}
	unggahXlsx(srv, m, "/api/v1/master/benchmark", katalog.BarisBenchmark(tax.KLUBenchmarks[:5]), map[string]string{"nama": "Uji"})
	if rec := do(srv, m, http.MethodPost, "/api/v1/master/benchmark/2/aktifkan", ""); rec.Code != http.StatusOK {
		t.Fatalf("aktifkan: %d %s", rec.Code, rec.Body)
	}
	pulihkanParameter(t)
	do(srv, m, http.MethodPost, "/api/v1/master/parameter/1/salin", "")
	unggahXlsx(srv, m, "/api/v1/master/parameter/2/norma", [][]string{{"KLU"}, {"47111", "Toko", "20", "15", "-"}}, nil)
	if rec := do(srv, m, http.MethodPost, "/api/v1/master/parameter/2/aktifkan", ""); rec.Code != http.StatusOK {
		t.Fatalf("aktifkan parameter: %d %s", rec.Code, rec.Body)
	}
	tax.AktifkanParameter(tax.ParameterBawaan()) // muat ulang harus memasang lagi versi aktif dari DB
	srv2 := NewServer()
	if got, want := normJSON(t, srv2.katalog), normJSON(t, srv.katalog); got != want {
		t.Fatalf("katalog berubah setelah simpan-muat:\n got %s\nwant %s", got, want)
	}
	if n := len(tax.BenchmarkAktif()); n != 5 {
		t.Errorf("benchmark aktif setelah muat ulang = %d baris", n)
	}
	if n, ok := tax.CariNorma("47111"); !ok || len(tax.P().Norma) != 1 || n.DaerahLainBps != -1 {
		t.Errorf("parameter aktif setelah muat ulang: norma %d baris %+v", len(tax.P().Norma), n)
	}
}

// Artikel & sumber pengetahuan (termasuk status verifikasi & pemakaian AI) tersimpan utuh.
func TestPostgresPengetahuan(t *testing.T) {
	pgTest(t)
	srv, m := serverMaster(t)
	do(srv, m, http.MethodPost, "/api/v1/master/artikel", `{"judul":"PPh Final","kategori":"PPh Final UMKM","tag":["omzet"],"isi":"Tarif 0,5%.","tampil_klien":true,"terbit":true}`)
	do(srv, m, http.MethodPost, "/api/v1/master/pengetahuan", `{"judul":"Batas omzet","isi":"OP sampai Rp500 juta.","topik":["PPh Final"]}`)
	do(srv, m, http.MethodPost, "/api/v1/master/pengetahuan/1/verifikasi", "")
	srv2 := NewServer()
	if got, want := normJSON(t, srv2.pengetahuan), normJSON(t, srv.pengetahuan); got != want || len(srv.pengetahuan.Artikel) != 1 {
		t.Fatalf("pengetahuan berubah setelah simpan-muat:\n got %s\nwant %s", got, want)
	}
}
