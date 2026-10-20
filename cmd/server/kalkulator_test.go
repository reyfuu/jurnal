package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

// Alat gratis: usaha yang belum berlangganan tetap boleh memakai kalkulator & perencanaan
// pajak (hitung saja, tidak menulis data), tetapi menulis data tetap 402.
func TestAlatGratisUntukUsahaBelumBayar(t *testing.T) {
	srv := NewServer()
	qa := register(t, srv, registerBody)
	for path, body := range map[string]string{
		"/api/v1/kalkulator/pkp":       `{"omzet":5000000000}`,
		"/api/v1/kalkulator/potong":    `{"jenis":"pph22","objek":"22-impor-api","nilai_bruto":100000000,"punya_npwp":true}`,
		"/api/v1/kalkulator/pph-badan": `{"peredaran_bruto":3000000000,"penghasilan_kena_pajak":300000000}`,
		"/api/v1/tax/planning":         `{"omzet":1000000000,"net_margin_bps":1500,"ptkp":"TK/0"}`,
	} {
		if rec := do(srv, qa, http.MethodPost, path, body); rec.Code != http.StatusOK {
			t.Errorf("%s: want 200, got %d %s", path, rec.Code, rec.Body)
		}
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-08-20","jenis":"Penjualan","nominal":5000,"lunas":true}`); rec.Code != http.StatusPaymentRequired {
		t.Fatalf("tulis transaksi saat belum bayar: want 402, got %d", rec.Code)
	}

	var pkp struct {
		WajibPKP bool  `json:"wajib_pkp"`
		Batas    int64 `json:"batas"`
	}
	_ = json.NewDecoder(do(srv, qa, http.MethodPost, "/api/v1/kalkulator/pkp", `{"omzet":5000000000}`).Body).Decode(&pkp)
	if !pkp.WajibPKP || pkp.Batas != 4_800_000_000 {
		t.Fatalf("cek PKP 5 M: %+v", pkp)
	}
}
