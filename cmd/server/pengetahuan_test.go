package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pajak/internal/ai"
	"pajak/internal/master"
	"pajak/internal/pengetahuan"
)

const artikelBody = `{"judul":"PPh Final 0,5%% untuk UMKM","kategori":"PPh Final UMKM","dasar_hukum":"PP 55/2022","tag":["omzet"],"isi":"Tarif PPh final 0,5%% dari omzet untuk usaha dengan omzet sampai Rp4,8 miliar setahun.","tampil_klien":%v,"terbit":%v}`

func TestArtikel_DraftTerbitDanPustaka(t *testing.T) {
	srv, m := serverMaster(t)
	qa := register(t, srv, registerBody)
	aktifkan(t, srv, qa)
	k := tambahAnggota(t, srv, m, "budi@masterpajak.id", "konsultan", []int{2})

	if rec := do(srv, k, http.MethodPost, "/api/v1/master/artikel", fmt.Sprintf(artikelBody, true, true)); rec.Code != http.StatusForbidden {
		t.Fatalf("konsultan menerbitkan: want 403, got %d", rec.Code)
	}
	rec := do(srv, k, http.MethodPost, "/api/v1/master/artikel", fmt.Sprintf(artikelBody, true, false))
	if rec.Code != http.StatusCreated {
		t.Fatalf("draft: %d %s", rec.Code, rec.Body)
	}
	if got := getJSON[[]pengetahuan.Artikel](t, srv, qa, "/api/v1/pustaka"); len(got) != 0 {
		t.Fatalf("draft tampil di pustaka klien: %+v", got)
	}
	if rec := do(srv, m, http.MethodPut, "/api/v1/master/artikel/1", fmt.Sprintf(artikelBody, true, true)); rec.Code != http.StatusOK {
		t.Fatalf("terbitkan: %d %s", rec.Code, rec.Body)
	}
	if got := getJSON[[]pengetahuan.Artikel](t, srv, qa, "/api/v1/pustaka"); len(got) != 1 || got[0].Diverifikasi != "Master" {
		t.Fatalf("pustaka = %+v", got)
	}
	if rec := do(srv, k, http.MethodPut, "/api/v1/master/artikel/1", fmt.Sprintf(artikelBody, true, false)); rec.Code != http.StatusForbidden {
		t.Errorf("konsultan mengubah artikel terbit: want 403, got %d", rec.Code)
	}
	// Artikel internal (tidak tampil ke klien) tetap terbit untuk AI tapi tidak di pustaka.
	do(srv, m, http.MethodPost, "/api/v1/master/artikel", fmt.Sprintf(artikelBody, false, true))
	if got := getJSON[[]pengetahuan.Artikel](t, srv, qa, "/api/v1/pustaka"); len(got) != 1 {
		t.Errorf("artikel internal tampil ke klien: %d", len(got))
	}
}

func unggahBerkas(srv *Server, c *http.Cookie, path, nama string, isi []byte, fields map[string]string) *httptest.ResponseRecorder {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	fw, _ := mw.CreateFormFile("file", nama)
	_, _ = fw.Write(isi)
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(c)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func TestPengetahuan_VerifikasiDanCari(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv, m := serverMaster(t)
	if rec := do(srv, m, http.MethodPost, "/api/v1/master/pengetahuan", `{"judul":"Batas omzet OP","isi":"Orang pribadi tidak kena PPh final atas omzet sampai Rp500 juta setahun.","topik":["PPh Final"]}`); rec.Code != http.StatusCreated {
		t.Fatalf("tulis manual: %d %s", rec.Code, rec.Body)
	}
	if got := getJSON[[]pengetahuan.Potongan](t, srv, m, "/api/v1/master/pengetahuan/cari?q=omzet+pph+final"); len(got) != 0 {
		t.Fatalf("sumber belum diverifikasi ikut ditemukan: %+v", got)
	}
	if rec := do(srv, m, http.MethodPost, "/api/v1/master/pengetahuan/1/verifikasi", ""); rec.Code != http.StatusOK {
		t.Fatalf("verifikasi: %d", rec.Code)
	}
	if got := getJSON[[]pengetahuan.Potongan](t, srv, m, "/api/v1/master/pengetahuan/cari?q=omzet+pph+final"); len(got) != 1 || got[0].Ref != "KB-001" {
		t.Fatalf("cari = %+v", got)
	}
	// Excel: satu baris satu potongan.
	if rec := unggahXlsx(srv, m, "/api/v1/master/pengetahuan", [][]string{{"Judul", "Isi", "Topik", "Sumber"}, {"Prive", "Ambil uang pribadi bukan biaya usaha.", "Pembukuan", ""}}, map[string]string{"judul": "Kamus kesalahan input"}); rec.Code != http.StatusCreated {
		t.Fatalf("impor excel: %d %s", rec.Code, rec.Body)
	}
	// PDF tanpa teks (hasil scan) ditolak dengan pesan yang jelas.
	if rec := unggahBerkas(srv, m, "/api/v1/master/pengetahuan", "scan.pdf", []byte("%PDF-1.4 rusak"), map[string]string{"judul": "PMK"}); rec.Code != http.StatusBadRequest {
		t.Errorf("pdf rusak: want 400, got %d", rec.Code)
	}
	list := getJSON[pengetahuanResp](t, srv, m, "/api/v1/master/pengetahuan")
	if len(list.Sumber) != 2 || list.Sumber[1].Jenis != "excel" || list.Sumber[1].Potongan != 1 || list.Sumber[0].Status != pengetahuan.Terverifikasi {
		t.Fatalf("daftar sumber = %+v", list.Sumber)
	}
	k := tambahAnggota(t, srv, m, "budi@masterpajak.id", "konsultan", nil)
	if rec := do(srv, k, http.MethodGet, "/api/v1/master/pengetahuan", ""); rec.Code != http.StatusForbidden {
		t.Errorf("konsultan buka basis pengetahuan: %d", rec.Code)
	}
	srv2 := NewServer()
	if got := len(srv2.pengetahuan.Sumber); got != 2 {
		t.Errorf("setelah restart: %d sumber", got)
	}
}

type analisaResp struct {
	Fakta   ai.FaktaKlien          `json:"fakta"`
	Sumber  []pengetahuan.Potongan `json:"sumber"`
	Analisa ai.AnalisaKlienHasil   `json:"analisa"`
}

func TestAnalisaKlien_FaktaTanpaKunciLLM(t *testing.T) {
	t.Setenv("LLM_API_KEY", "")
	srv, m, qa, k := siapKlien(t)
	do(srv, qa, http.MethodPut, "/api/v1/business", profilDagang) // KLU 471
	do(srv, qa, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-08-03","jenis":"Penjualan","nominal":20000000,"keterangan":"Penjualan","lunas":true}`)
	do(srv, m, http.MethodPost, "/api/v1/master/pengetahuan", `{"judul":"Batas omzet OP","isi":"Orang pribadi tidak kena PPh final atas omzet sampai Rp500 juta setahun.","topik":["PPh Final"]}`)
	do(srv, m, http.MethodPost, "/api/v1/master/pengetahuan/1/verifikasi", "")

	got := getJSON[analisaResp](t, srv, k, "/api/v1/master/klien/2/analisa")
	if got.Fakta.KLU != "471" || got.Fakta.OmzetSetahun <= 0 || got.Fakta.BenchmarkNPMBps == 0 || got.Fakta.StatusNPM == "" {
		t.Fatalf("fakta = %+v", got.Fakta)
	}
	if len(got.Sumber) == 0 || got.Sumber[0].Ref != "KB-001" {
		t.Errorf("sumber = %+v", got.Sumber)
	}
	if got.Analisa.Tersedia || !strings.Contains(got.Analisa.Alasan, "LLM_API_KEY") {
		t.Errorf("analisa tanpa kunci = %+v", got.Analisa)
	}
	cs := tambahAnggota(t, srv, m, "rina@masterpajak.id", "cs", []int{2})
	if rec := do(srv, cs, http.MethodGet, "/api/v1/master/klien/2/analisa", ""); rec.Code != http.StatusForbidden {
		t.Errorf("cs tanpa izin lihat: %d", rec.Code)
	}
	audit := getJSON[[]master.Catatan](t, srv, m, "/api/v1/master/audit")
	if !slicesAny(audit, func(c master.Catatan) bool { return c.Aksi == "Analisa AI klien" && c.KlienID == 2 }) {
		t.Error("analisa tidak tercatat di audit")
	}
	_ = json.Marshal
}
