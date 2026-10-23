package main

import (
	"maps"
	"net/http"
	"strings"
	"testing"
)

// Jurnal Umum: semua entri jurnal per bukti, tiap bukti seimbang, totalnya sama dengan mutasi
// Buku Besar rentang yang sama, dan tiap transaksi bulan itu muncul tepat sekali.
func TestJurnalUmum_SeimbangDanCocokBukuBesar(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	do(srv, owner, http.MethodPut, "/api/v1/business", strings.Replace(profileBody, "manufaktur", "perdagangan", 1))
	ju := getJSON[jurnalUmumResp](t, srv, owner, "/api/v1/reports/jurnal?period=2026-08")
	bb := getJSON[bukuBesarResp](t, srv, owner, "/api/v1/reports/buku-besar?period=2026-08")
	if !ju.Supported || !ju.Seimbang || ju.TotalDebit == 0 || ju.TotalDebit != bb.TotalDebit {
		t.Fatalf("jurnal umum: supported %v seimbang %v debit %d (buku besar %d)", ju.Supported, ju.Seimbang, ju.TotalDebit, bb.TotalDebit)
	}
	tx := map[string]int{}
	for _, e := range ju.Entri {
		var d, k int64
		for _, b := range e.Baris {
			d, k = d+b.Debit, k+b.Kredit
			if b.Nama == "" {
				t.Errorf("%s: akun %s tanpa nama", e.Bukti, b.Kode)
			}
		}
		if d != k || d != e.Debit || e.Tanggal < "2026-08-01" || e.Tanggal > "2026-08-31" {
			t.Fatalf("bukti %s %s tidak seimbang / di luar rentang: %+v", e.Bukti, e.Tanggal, e)
		}
		if strings.HasPrefix(e.Bukti, "JU-") { // nomor jurnal tetap per transaksi (checklist No 31)
			tx[e.Bukti]++
		}
	}
	n := 0
	for _, x := range srv.tenants[1].txs {
		if strings.HasPrefix(x.Tanggal, "2026-08") {
			n++
		}
	}
	if len(tx) != n {
		t.Errorf("transaksi Agustus %d, bukti JU %d", n, len(tx))
	}
	for b, c := range tx {
		if c != 1 {
			t.Errorf("%s muncul %d kali", b, c)
		}
	}
}

// Rincian Buku Besar mencantumkan No. Bukti yang sama dengan Jurnal Umum (JU-, SIS-, BYR-),
// supaya mutasi yang salah bisa ditelusuri ke transaksinya.
func TestBukuBesar_NoBuktiSamaDenganJurnalUmum(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	do(srv, owner, http.MethodPut, "/api/v1/business", strings.Replace(profileBody, "manufaktur", "perdagangan", 1))
	ju := getJSON[jurnalUmumResp](t, srv, owner, "/api/v1/reports/jurnal?period=2026-08")
	bb := getJSON[struct {
		Akun []struct {
			Kode  string
			Baris []struct{ Bukti string }
		}
	}](t, srv, owner, "/api/v1/reports/buku-besar?period=2026-08")
	dariJU, dariBB := map[string]bool{}, map[string]bool{}
	for _, e := range ju.Entri {
		for _, b := range e.Baris {
			dariJU[e.Bukti+" "+b.Kode] = true
		}
	}
	for _, a := range bb.Akun {
		for _, b := range a.Baris {
			if b.Bukti == "" {
				t.Fatalf("akun %s: baris buku besar tanpa no. bukti", a.Kode)
			}
			dariBB[b.Bukti+" "+a.Kode] = true
		}
	}
	if len(dariJU) == 0 || !maps.Equal(dariJU, dariBB) {
		t.Errorf("no. bukti per akun beda: jurnal umum %d, buku besar %d", len(dariJU), len(dariBB))
	}
	sis := false
	for k := range dariBB {
		sis = sis || strings.HasPrefix(k, "SIS-")
	}
	if !sis {
		t.Error("entri sistem (penyusutan) tidak bernomor SIS- di buku besar")
	}
}

// Saldo awal tampil sebagai bukti pertama bila tanggal mulai buku masuk rentang; filter akun &
// kata kunci disaring di server (tampilan = ekspor Excel).
func TestJurnalUmum_SaldoAwalPertamaDanFilter(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	do(srv, owner, http.MethodPut, "/api/v1/business", strings.Replace(profileBody, "manufaktur", "perdagangan", 1))
	ju := getJSON[jurnalUmumResp](t, srv, owner, "/api/v1/reports/jurnal?dari=2026-01-01&sampai=2026-01-31")
	if len(ju.Entri) == 0 || ju.Entri[0].Bukti != "SA" || ju.Entri[0].Debit == 0 || ju.Entri[0].Debit != ju.Entri[0].Kredit || !ju.Seimbang {
		t.Fatalf("bukti pertama harus saldo awal seimbang: %+v", ju.Entri[:min(1, len(ju.Entri))])
	}
	kas := getJSON[jurnalUmumResp](t, srv, owner, "/api/v1/reports/jurnal?period=2026-08&akun=1101")
	for _, e := range kas.Entri {
		ada := false
		for _, b := range e.Baris {
			ada = ada || b.Kode == "1101"
		}
		if !ada {
			t.Fatalf("filter akun 1101 memuat %s tanpa akun itu", e.Bukti)
		}
	}
	cari := getJSON[jurnalUmumResp](t, srv, owner, "/api/v1/reports/jurnal?period=2026-08&q=penyusutan")
	if len(cari.Entri) == 0 || !strings.Contains(strings.ToLower(cari.Entri[0].Keterangan), "penyusutan") {
		t.Fatalf("cari penyusutan: %+v", cari.Entri)
	}
	if rec := do(srv, owner, http.MethodGet, "/api/v1/reports/jurnal.xlsx?period=2026-08", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Disposition"), "jurnal-umum") {
		t.Fatalf("ekspor: %d %v", rec.Code, rec.Header())
	}
	kasir := login(t, srv, "kasir@kedaikopisenja.id")
	if rec := do(srv, kasir, http.MethodGet, "/api/v1/reports/jurnal?period=2026-08", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("kasir: %d", rec.Code)
	}
}
