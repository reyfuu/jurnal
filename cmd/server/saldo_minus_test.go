package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"pajak/internal/coa"
	"pajak/internal/finance"
)

func usahaBaru(t *testing.T) (*Server, *http.Cookie) {
	t.Helper()
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	qa := register(t, srv, registerBody)
	aktifkan(t, srv, qa)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/business", dagangCV); rec.Code != http.StatusOK {
		t.Fatalf("profil: %d %s", rec.Code, rec.Body)
	}
	return srv, qa
}

// Saldo awal yang belum seimbang bisa disimpan setelah dikonfirmasi; selisihnya masuk akun
// Laba Ditahan - Selisih Saldo Awal (3299), diperingatkan, dan hilang begitu disimpan seimbang.
func TestSaldoAwal_PaksaSimpanKeAkunSelisih(t *testing.T) {
	srv, qa := usahaBaru(t)
	timpang := `{"tanggal":"2026-01-01","rincian":[{"nama":"Kas","kategori":"kas","nominal":10000000},{"nama":"Modal Pemilik","kategori":"modal","nominal":7000000}]}`
	if rec := do(srv, qa, http.MethodPut, "/api/v1/opening", timpang); rec.Code != http.StatusBadRequest {
		t.Fatalf("tanpa paksa harus ditolak: %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodPut, "/api/v1/opening?paksa=1", timpang); rec.Code != http.StatusOK {
		t.Fatalf("paksa: %d %s", rec.Code, rec.Body)
	}
	o := getJSON[openingResp](t, srv, qa, "/api/v1/opening")
	var selisih *finance.SaldoAkun
	for i, r := range o.Opening.Rincian {
		if r.Kode == coa.KodeSelisihSaldoAwal {
			selisih = &o.Opening.Rincian[i]
		}
	}
	if selisih == nil || selisih.Nominal != 3_000_000 || selisih.Kategori != finance.KatLabaDitahan || o.SelisihModal != 0 {
		t.Fatalf("baris selisih: %+v (opening %+v)", selisih, o.Opening.Rincian)
	}
	ada := false
	for _, c := range getJSON[[]cekKecocokan](t, srv, qa, "/api/v1/saldo-awal/cek") {
		ada = ada || (c.Kode == "paksa" && c.Status == "selisih")
	}
	if !ada {
		t.Fatal("selisih yang disimpan paksa harus diperingatkan")
	}
	var rep struct {
		TrialBalance []coa.Balance `json:"trial_balance"`
	}
	_ = json.Unmarshal(do(srv, qa, http.MethodGet, "/api/v1/reports/coa?period=2026-01", "").Body.Bytes(), &rep)
	saldo := map[string]int64{}
	for _, b := range rep.TrialBalance {
		saldo[b.Kode] = b.Saldo
	}
	if saldo[coa.KodeSelisihSaldoAwal] != 3_000_000 || saldo["3201"] != 0 {
		t.Fatalf("neraca saldo: 3299=%d 3201=%d", saldo[coa.KodeSelisihSaldoAwal], saldo["3201"])
	}
	seimbang := strings.Replace(timpang, `"nominal":7000000`, `"nominal":10000000`, 1)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/opening", seimbang); rec.Code != http.StatusOK {
		t.Fatalf("simpan seimbang: %d %s", rec.Code, rec.Body)
	}
	for _, r := range getJSON[openingResp](t, srv, qa, "/api/v1/opening").Opening.Rincian {
		if r.Kode == coa.KodeSelisihSaldoAwal {
			t.Fatalf("baris selisih harus hilang setelah seimbang: %+v", r)
		}
	}
}

// Kas yang dibelanjakan melebihi saldonya = saldo kas di kredit (minus, PSAK). Penjualan yang
// bersaldo kredit itu normal, jadi tidak ikut diperingatkan.
func TestSaldoMinus_HanyaYangBerlawananDenganSaldoNormal(t *testing.T) {
	srv, qa := usahaBaru(t)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/opening", `{"tanggal":"2026-01-01","rincian":[{"nama":"Kas","kategori":"kas","nominal":1000000},{"nama":"Modal Pemilik","kategori":"modal","nominal":1000000}]}`); rec.Code != http.StatusOK {
		t.Fatalf("saldo awal: %d %s", rec.Code, rec.Body)
	}
	for _, tx := range []string{
		`{"tanggal":"2026-01-05","jenis":"Penjualan","nominal":500000,"lunas":true}`,
		`{"tanggal":"2026-01-06","jenis":"Sewa Tempat","nominal":4000000,"lunas":true}`,
	} {
		buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", tx)
	}
	minus := getJSON[[]saldoMinus](t, srv, qa, "/api/v1/saldo-minus?period=2026-01")
	if len(minus) != 1 || minus[0].Kode != "1101" || minus[0].Nilai != 2_500_000 || minus[0].Normal != "debit" {
		t.Fatalf("peringatan: %+v", minus)
	}
}
