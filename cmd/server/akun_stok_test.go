package main

import (
	"net/http"
	"strings"
	"testing"

	"pajak/internal/finance"
)

type stokAkunResp struct {
	Items []struct {
		ID          int    `json:"id"`
		AkunEfektif string `json:"akun_efektif"`
	} `json:"items"`
	PilihanAkun []struct {
		Kode string `json:"kode"`
		Nama string `json:"nama"`
	} `json:"pilihan_akun"`
}

func TestStok_AkunPersediaan(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	if rec := do(srv, owner, http.MethodPut, "/api/v1/business", profileBody); rec.Code != http.StatusOK {
		t.Fatalf("profil manufaktur: %d %s", rec.Code, rec.Body)
	}
	efektif := func(id int) string {
		for _, it := range getJSON[stokAkunResp](t, srv, owner, "/api/v1/stok").Items {
			if it.ID == id {
				return it.AkunEfektif
			}
		}
		t.Fatalf("barang %d tidak ada di daftar", id)
		return ""
	}

	it := buat[finance.StokItem](t, srv, owner, "/api/v1/stok", `{"nama":"Kain Uji Akun","kategori":"Bahan Baku"}`)
	if got := efektif(it.ID); got != "1301" {
		t.Fatalf("tebakan dari kategori: %q", got)
	}

	if rec := do(srv, owner, http.MethodPut, "/api/v1/stok/"+itoa(it.ID), `{"nama":"Kain Uji Akun","kategori":"Bahan Baku","akun":"1303"}`); rec.Code != http.StatusOK {
		t.Fatalf("ubah: %d %s", rec.Code, rec.Body)
	}
	if got := efektif(it.ID); got != "1303" {
		t.Fatalf("akun pilihan: %q", got)
	}

	rec := do(srv, owner, http.MethodPost, "/api/v1/stok", `{"nama":"Salah Akun","akun":"4101"}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "akun persediaan tidak dikenal") {
		t.Fatalf("akun bukan persediaan: %d %s", rec.Code, rec.Body)
	}

	// Akun yang sudah tidak ada lagi: jatuh ke tebakan kategori, tidak error.
	tn := srv.tenants[1]
	tn.mu.Lock()
	for i := range tn.stok {
		if tn.stok[i].ID == it.ID {
			tn.stok[i].Akun = "1399"
		}
	}
	tn.mu.Unlock()
	if got := efektif(it.ID); got != "1301" {
		t.Fatalf("akun hilang harus jatuh ke tebakan: %q", got)
	}

	if p := getJSON[stokAkunResp](t, srv, owner, "/api/v1/stok").PilihanAkun; len(p) < 2 || p[0].Kode == "" || p[0].Nama == "" {
		t.Fatalf("pilihan_akun: %+v", p)
	}
}
