package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"pajak/internal/katalog"
	"pajak/internal/tax"
)

func pulihkanParameter(t *testing.T) {
	t.Cleanup(func() { tax.AktifkanParameter(tax.ParameterBawaan()) })
}

func TestParameter_DraftUbahAktifkanDipakaiKalkulator(t *testing.T) {
	pulihkanParameter(t)
	srv, m := serverMaster(t)
	qa := register(t, srv, registerBody)

	rec := do(srv, m, http.MethodPost, "/api/v1/master/parameter/1/salin", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("salin: %d %s", rec.Code, rec.Body)
	}
	var d parameterRingkas
	_ = json.NewDecoder(rec.Body).Decode(&d)
	if d.Status != katalog.Draft || d.Isi.Norma != nil || d.JumlahNorma < 1000 {
		t.Fatalf("draft = status %s, norma %d / %d", d.Status, len(d.Isi.Norma), d.JumlahNorma)
	}

	d.Isi.BatasPKP = 1_000_000_000
	body, _ := json.Marshal(map[string]any{"nama": "Uji batas PKP", "dasar": "PMK uji", "isi": d.Isi})
	if rec := do(srv, m, http.MethodPut, "/api/v1/master/parameter/"+itoa(d.ID), string(body)); rec.Code != http.StatusOK {
		t.Fatalf("ubah: %d %s", rec.Code, rec.Body)
	}
	var cek tax.HasilPKP
	_ = json.NewDecoder(do(srv, qa, http.MethodPost, "/api/v1/kalkulator/pkp", `{"omzet":2000000000}`).Body).Decode(&cek)
	if cek.WajibPKP {
		t.Fatal("draft sudah dipakai klien sebelum diaktifkan")
	}
	if rec := do(srv, m, http.MethodPost, "/api/v1/master/parameter/"+itoa(d.ID)+"/aktifkan", ""); rec.Code != http.StatusOK {
		t.Fatalf("aktifkan: %d %s", rec.Code, rec.Body)
	}
	_ = json.NewDecoder(do(srv, qa, http.MethodPost, "/api/v1/kalkulator/pkp", `{"omzet":2000000000}`).Body).Decode(&cek)
	if !cek.WajibPKP || cek.Batas != 1_000_000_000 {
		t.Fatalf("batas PKP versi aktif tidak dipakai: %+v", cek)
	}

	g := getJSON[struct{ Versi []parameterRingkas }](t, srv, m, "/api/v1/master/parameter")
	if len(g.Versi) != 2 || g.Versi[0].Status != katalog.Arsip || g.Versi[1].JumlahNorma != d.JumlahNorma {
		t.Fatalf("daftar = %+v", g.Versi)
	}
	// versi aktif tidak bisa diubah; angka ngawur ditolak
	if rec := do(srv, m, http.MethodPut, "/api/v1/master/parameter/"+itoa(d.ID), string(body)); rec.Code != http.StatusBadRequest {
		t.Errorf("ubah versi aktif: %d", rec.Code)
	}
	d.Isi.BadanTarifBps = 50_000
	salah, _ := json.Marshal(map[string]any{"nama": "Rusak", "isi": d.Isi})
	do(srv, m, http.MethodPost, "/api/v1/master/parameter/1/salin", "")
	if rec := do(srv, m, http.MethodPut, "/api/v1/master/parameter/3", string(salah)); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "tarif badan") {
		t.Errorf("isi ngawur: %d %s", rec.Code, rec.Body)
	}

	k := tambahAnggota(t, srv, m, "budi@masterpajak.id", "konsultan", []int{2})
	if rec := do(srv, k, http.MethodGet, "/api/v1/master/parameter", ""); rec.Code != http.StatusForbidden {
		t.Errorf("konsultan buka parameter: %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodGet, "/api/v1/master/parameter", ""); rec.Code != http.StatusForbidden {
		t.Errorf("klien buka parameter: %d", rec.Code)
	}
}

func TestParameter_ImporNormaDipakaiNPPN(t *testing.T) {
	pulihkanParameter(t)
	srv, m := serverMaster(t)
	qa := register(t, srv, registerBody)
	do(srv, m, http.MethodPost, "/api/v1/master/parameter/1/salin", "")
	rows := [][]string{{"KLU", "Uraian", "a", "b", "c"}, {"47111", "Toko kelontong", "20", "15", "12,5"}, {"1111", "Jagung", "-", "", "10"}}
	rec := unggahXlsx(srv, m, "/api/v1/master/parameter/2/norma", rows, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("impor norma: %d %s", rec.Code, rec.Body)
	}
	var d parameterRingkas
	_ = json.NewDecoder(rec.Body).Decode(&d)
	if d.JumlahNorma != 2 {
		t.Fatalf("norma %d baris", d.JumlahNorma)
	}
	do(srv, m, http.MethodPost, "/api/v1/master/parameter/2/aktifkan", "")

	var r tax.NPPNResult
	rec = do(srv, qa, http.MethodPost, "/api/v1/kalkulator/nppn", `{"omzet":1000000000,"klu":"47111","wilayah":"daerah_lain","ptkp":"TK/0"}`)
	_ = json.NewDecoder(rec.Body).Decode(&r)
	if rec.Code != http.StatusOK || r.NormaBps != 1250 || r.Neto != 125_000_000 {
		t.Fatalf("NPPN: %d %+v", rec.Code, r)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/kalkulator/nppn", `{"omzet":1000000000,"klu":"01111","wilayah":"ibukota10","ptkp":"TK/0"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("norma tidak tersedia: %d", rec.Code)
	}
	var cari []tax.Norma
	_ = json.NewDecoder(do(srv, qa, http.MethodGet, "/api/v1/kalkulator/norma?q=kelontong", "").Body).Decode(&cari)
	if len(cari) != 1 || cari[0].KLU != "47111" {
		t.Errorf("cari norma = %+v", cari)
	}
	if rec := do(srv, m, http.MethodGet, "/api/v1/master/parameter/2/norma.xlsx", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Disposition"), "norma-nppn-2") {
		t.Errorf("unduh norma: %d", rec.Code)
	}
}

func TestKalkulatorPerencanaan_Endpoint(t *testing.T) {
	srv := NewServer()
	qa := register(t, srv, registerBody)
	for path, body := range map[string]string{
		"/api/v1/kalkulator/gross-up":  `{"jenis":"pph23","objek":"23-jasa","diterima":200000000,"punya_npwp":true}`,
		"/api/v1/kalkulator/ppn-harga": `{"pembelian":1000000000,"porsi_dari_pkp_bps":10000,"markup_bps":1000,"opex_bps":200}`,
		"/api/v1/kalkulator/pph22-marketplace": `{"omzet_marketplace":2000000000,"npm_bps":300,"bentuk":"badan","skema":"umum"}`,
		"/api/v1/kalkulator/beli-sewa": `{"bentuk":"badan","omzet":1000000000,"biaya_lain":400000000,"harga_aset":10000000,"umur_tahun":4,"metode":"garis_lurus","sewa_per_tahun":4000000}`,
	} {
		if rec := do(srv, qa, http.MethodPost, path, body); rec.Code != http.StatusOK {
			t.Errorf("%s: %d %s", path, rec.Code, rec.Body)
		}
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/kalkulator/gross-up", `{"jenis":"ppn","objek":"ppn-umum","diterima":1}`); rec.Code != http.StatusBadRequest {
		t.Errorf("gross up PPN: %d", rec.Code)
	}
	var km struct {
		Parameter struct {
			PPNEfektifBps int64 `json:"ppn_efektif_bps"`
		} `json:"parameter"`
	}
	_ = json.NewDecoder(do(srv, qa, http.MethodGet, "/api/v1/kalkulator", "").Body).Decode(&km)
	if km.Parameter.PPNEfektifBps != 1100 {
		t.Errorf("parameter kalkulator = %+v", km)
	}
}
