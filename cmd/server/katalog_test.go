package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"pajak/internal/coa"
	"pajak/internal/katalog"
	"pajak/internal/master"
	"pajak/internal/tax"
	"pajak/internal/xlsx"
)

const profilDagang = `{"nama_usaha":"Toko QA","jenis_usaha":"perdagangan","bentuk_usaha":"op","ptkp":"TK/0","pkp":false,"metode_persediaan":"periodik","klu":"471"}`

// unggahXlsx mengirim rows sebagai berkas .xlsx di field "file" plus field lain.
func unggahXlsx(srv *Server, c *http.Cookie, path string, rows [][]string, fields map[string]string) *httptest.ResponseRecorder {
	var file bytes.Buffer
	_ = xlsx.Write(&file, []xlsx.Sheet{{Name: "Data", Rows: rows}})
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	fw, _ := mw.CreateFormFile("file", "data.xlsx")
	_, _ = fw.Write(file.Bytes())
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(c)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

type coaResp struct {
	Supported bool          `json:"supported"`
	Accounts  []coa.Account `json:"accounts"`
	Template  *struct {
		ID, Versi int
	} `json:"template"`
	Pembaruan *struct {
		ID, Versi int
		Diff      katalog.DiffAkun
	} `json:"pembaruan"`
}

func TestKatalog_TemplateBaruTidakMengubahKlienLama(t *testing.T) {
	t.Cleanup(func() { tax.AktifkanBenchmark(tax.KLUBenchmarks, tax.SumberBenchmark) })
	t.Setenv("DATA_DIR", t.TempDir())
	srv, m := serverMaster(t)
	lama := register(t, srv, registerBody)
	aktifkan(t, srv, lama)
	if rec := do(srv, lama, http.MethodPut, "/api/v1/business", profilDagang); rec.Code != http.StatusOK {
		t.Fatalf("profil: %d %s", rec.Code, rec.Body)
	}
	if got := getJSON[coaResp](t, srv, lama, "/api/v1/coa"); got.Template == nil || got.Template.Versi != 1 || got.Pembaruan != nil {
		t.Fatalf("coa awal = %+v", got)
	}

	// Draft tanpa akun kas ditolak saat terbit, dengan pesan yang menyebut akunnya.
	tanpaKas := slices.DeleteFunc(slices.Clone(coa.Master), func(a coa.Account) bool { return a.Kode == "1101" })
	rec := unggahXlsx(srv, m, "/api/v1/master/coa/draft", katalog.BarisTemplate(tanpaKas), map[string]string{"jenis": "perdagangan"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("draft dari Excel: %d %s", rec.Code, rec.Body)
	}
	var d katalog.Template
	_ = json.NewDecoder(rec.Body).Decode(&d)
	if rec := do(srv, m, http.MethodPost, "/api/v1/master/coa/"+itoa(d.ID)+"/terbitkan", ""); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "kas") {
		t.Fatalf("terbit tanpa kas: %d %s", rec.Code, rec.Body)
	}

	baru := append(slices.Clone(coa.Master), coa.Account{Kode: "6119", Nama: "Beban Asuransi", Subtype: "opex", Normal: coa.Debit, Kelompok: coa.BebanOperasional, Kondisi: coa.Selalu})
	body, _ := json.Marshal(map[string]any{"jenis": "perdagangan", "akun": baru, "catatan": "Tambah asuransi"})
	if rec := do(srv, m, http.MethodPost, "/api/v1/master/coa/draft", string(body)); rec.Code != http.StatusCreated {
		t.Fatalf("draft JSON: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, m, http.MethodPost, "/api/v1/master/coa/"+itoa(d.ID)+"/terbitkan", ""); rec.Code != http.StatusOK {
		t.Fatalf("terbitkan: %d %s", rec.Code, rec.Body)
	}

	// Klien lama tetap di versi 1 dan mendapat tawaran pembaruan.
	got := getJSON[coaResp](t, srv, lama, "/api/v1/coa")
	if got.Template.Versi != 1 || got.Pembaruan == nil || got.Pembaruan.Versi != 2 || len(got.Pembaruan.Diff.Tambah) != 1 {
		t.Fatalf("klien lama setelah terbit = %+v", got)
	}
	if _, ok := coa.Find(got.Accounts, "6119"); ok {
		t.Fatal("akun baru masuk ke klien lama sebelum diperbarui")
	}
	if rec := do(srv, lama, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-10-14","jenis":"Biaya Lain","nominal":100,"keterangan":"x","lunas":true,"akun":"6119"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("akun versi baru dipakai sebelum diperbarui: %d", rec.Code)
	}
	// Klien baru langsung memakai versi 2.
	baruC := register(t, srv, strings.Replace(registerBody, "qa1@x.id", "qa2@x.id", 1))
	do(srv, baruC, http.MethodPut, "/api/v1/business", profilDagang)
	if g := getJSON[coaResp](t, srv, baruC, "/api/v1/coa"); g.Template.Versi != 2 || g.Pembaruan != nil {
		t.Fatalf("klien baru = %+v", g.Template)
	}
	// Owner memperbarui: akun baru bisa dipakai.
	if rec := do(srv, lama, http.MethodPost, "/api/v1/coa/perbarui", ""); rec.Code != http.StatusOK {
		t.Fatalf("perbarui: %d %s", rec.Code, rec.Body)
	}
	if g := getJSON[coaResp](t, srv, lama, "/api/v1/coa"); g.Template.Versi != 2 || g.Pembaruan != nil {
		t.Fatalf("setelah perbarui = %+v", g)
	}
	if rec := do(srv, lama, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-10-14","jenis":"Biaya Lain","nominal":100,"keterangan":"x","lunas":true,"akun":"6119"}`); rec.Code != http.StatusCreated {
		t.Errorf("akun baru setelah perbarui: %d %s", rec.Code, rec.Body)
	}

	// Tersimpan: restart membaca versi, patokan klien, dan audit.
	srv2 := NewServer()
	if g := getJSON[coaResp](t, srv2, baruC, "/api/v1/coa"); g.Template.Versi != 2 {
		t.Errorf("setelah restart = %+v", g.Template)
	}
	audit := getJSON[[]master.Catatan](t, srv2, loginPass(t, srv2, "andi@masterpajak.id", masterPass), "/api/v1/master/audit")
	if !slicesAny(audit, func(c master.Catatan) bool { return c.Aksi == "Terbitkan template COA perdagangan v2" }) {
		t.Errorf("audit terbit tidak ada: %+v", audit)
	}
}

func TestKatalog_BenchmarkVersiBaru(t *testing.T) {
	t.Cleanup(func() { tax.AktifkanBenchmark(tax.KLUBenchmarks, tax.SumberBenchmark) })
	srv, m := serverMaster(t)
	qa := register(t, srv, registerBody)
	do(srv, qa, http.MethodPut, "/api/v1/business", profilDagang) // KLU 471

	tanpa471 := slices.DeleteFunc(slices.Clone(tax.KLUBenchmarks), func(k tax.KLUBenchmark) bool { return k.Kode == "471" })
	rec := unggahXlsx(srv, m, "/api/v1/master/benchmark", katalog.BarisBenchmark(tanpa471), map[string]string{"nama": "Nasional 2024 (uji)"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("impor benchmark: %d %s", rec.Code, rec.Body)
	}
	var b katalog.Benchmark
	_ = json.NewDecoder(rec.Body).Decode(&b)
	if rec := do(srv, m, http.MethodPost, "/api/v1/master/benchmark/"+itoa(b.ID)+"/aktifkan", ""); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "471") {
		t.Fatalf("aktifkan tanpa KLU klien: %d %s", rec.Code, rec.Body)
	}

	naik := slices.Clone(tax.KLUBenchmarks)
	i := slices.IndexFunc(naik, func(k tax.KLUBenchmark) bool { return k.Kode == "471" })
	naik[i].NPMBps = 777
	rec = unggahXlsx(srv, m, "/api/v1/master/benchmark", katalog.BarisBenchmark(naik), map[string]string{"nama": "Nasional 2024", "sumber": "Benchmark nasional 2024"})
	_ = json.NewDecoder(rec.Body).Decode(&b)
	if rec := do(srv, m, http.MethodPost, "/api/v1/master/benchmark/"+itoa(b.ID)+"/aktifkan", ""); rec.Code != http.StatusOK {
		t.Fatalf("aktifkan: %d %s", rec.Code, rec.Body)
	}
	if k, _ := tax.FindKLU("471"); k.NPMBps != 777 {
		t.Errorf("benchmark aktif belum berganti: %+v", k)
	}
	var cek tax.CekAwalResult
	_ = json.NewDecoder(do(srv, qa, http.MethodPost, "/api/v1/business/cek-awal", `{"klu":"471","omzet_tahunan":1000000000,"laba_sebelum_pajak":100000000,"bentuk_usaha":"op"}`).Body).Decode(&cek)
	if cek.Sumber != "Benchmark nasional 2024" || cek.KLU == nil || cek.KLU.NPMBps != 777 {
		t.Errorf("cek awal klien = %+v", cek)
	}
	if g := getJSON[katalogBenchmarkResp](t, srv, m, "/api/v1/master/benchmark"); g.KlienPerKLU["471"] != 1 || len(g.Versi) != 3 {
		t.Errorf("daftar benchmark = %d versi, klien 471 = %d", len(g.Versi), g.KlienPerKLU["471"])
	}
	k := tambahAnggota(t, srv, m, "budi@masterpajak.id", "konsultan", []int{2})
	if rec := do(srv, k, http.MethodGet, "/api/v1/master/benchmark", ""); rec.Code != http.StatusForbidden {
		t.Errorf("konsultan buka benchmark: %d", rec.Code)
	}
}

// Versi 1 benchmark yang tersimpan di DB sebelum penggantian nama ikut memakai nama & sumber dari kode.
func TestSegarkanBawaan_NamaBenchmarkIkutKode(t *testing.T) {
	st := katalog.NewState()
	st.Benchmark[0].Nama, st.Benchmark[0].Sumber = "Buku sektoral lama", "sumber lama"
	segarkanBawaan(&st)
	if b := st.Benchmark[0]; b.Nama != "Benchmark nasional · data 2020" || b.Sumber != tax.SumberBenchmark || strings.Contains(b.Nama+b.Sumber, "DJP") {
		t.Fatalf("benchmark bawaan = %q / %q", b.Nama, b.Sumber)
	}
}
