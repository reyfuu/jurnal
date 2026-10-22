package main

import (
	"net/http"
	"strings"
	"testing"

	"pajak/internal/finance"
)

func TestAsetPada_MetadataMetodeSamaDenganBulanan(t *testing.T) {
	srv := NewServer()
	c := login(t, srv, "owner@kedaikopisenja.id")
	if rec := do(srv, c, http.MethodPut, "/api/v1/business", profileBody); rec.Code != http.StatusOK {
		t.Fatalf("profil: %d %s", rec.Code, rec.Body)
	}
	for _, metode := range []string{"", finance.MetodeSaldoMenurun} {
		if metode != "" {
			if rec := do(srv, c, http.MethodPost, "/api/v1/business/metode-penyusutan", `{"metode":"`+metode+`"}`); rec.Code != http.StatusOK {
				t.Fatalf("pilih metode: %d %s", rec.Code, rec.Body)
			}
		}
		bulanan := getJSON[finance.AssetSummary](t, srv, c, "/api/v1/assets?period=2026-08")
		harian := getJSON[finance.AssetSummary](t, srv, c, "/api/v1/assets?pada=2026-08-15")
		if harian.MetodeUsaha != bulanan.MetodeUsaha || harian.MetodeSaran != bulanan.MetodeSaran || harian.MetodeCampur != bulanan.MetodeCampur {
			t.Errorf("metode %q: bulanan (%q, %q, %v), per tanggal (%q, %q, %v)", metode,
				bulanan.MetodeUsaha, bulanan.MetodeSaran, bulanan.MetodeCampur,
				harian.MetodeUsaha, harian.MetodeSaran, harian.MetodeCampur)
		}
	}
}

func TestPDFAset_MengikutiTanggalPilihan(t *testing.T) {
	srv := NewServer()
	c := login(t, srv, "owner@kedaikopisenja.id")
	buat[finance.Transaction](t, srv, c, "/api/v1/transactions", `{"tanggal":"2026-08-20","jenis":"Beli Alat / Aset","nominal":7000000,"lunas":true,"keterangan":"MesinTanggalDuaPuluh"}`)
	for _, tc := range []struct {
		tanggal string
		ada     bool
	}{{"2026-08-15", false}, {"2026-08-31", true}, {"2026-07-15", false}} {
		rec := do(srv, c, http.MethodGet, "/api/v1/reports/pdf?period=2026-08&bagian=aset&tanggal="+tc.tanggal, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", tc.tanggal, rec.Code, rec.Body)
		}
		if ada := strings.Contains(rec.Body.String(), "MesinTanggalDuaPuluh"); ada != tc.ada {
			t.Errorf("%s: aset dibeli 20 Agustus muncul=%v, mau %v", tc.tanggal, ada, tc.ada)
		}
		if !strings.Contains(rec.Body.String(), "Per "+finance.TanggalPanjang(tc.tanggal)) {
			t.Errorf("%s: label tanggal PDF tidak sesuai", tc.tanggal)
		}
		if !strings.Contains(rec.Header().Get("Content-Disposition"), "Register-Aset-"+tc.tanggal+".pdf") {
			t.Errorf("%s: nama berkas %q", tc.tanggal, rec.Header().Get("Content-Disposition"))
		}
	}
	if rec := do(srv, c, http.MethodGet, "/api/v1/reports/pdf?period=2026-08&bagian=aset&tanggal=2026-13-01", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("tanggal salah: %d, mau 400", rec.Code)
	}
}
