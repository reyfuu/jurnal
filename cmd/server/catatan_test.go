package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"pajak/internal/finance"
)

func TestCatatan_TulisTindaklanjutPengingat(t *testing.T) {
	srv, m, qa, k := siapKlien(t) // k = konsultan usaha 2
	do(srv, k, http.MethodPost, "/api/v1/master/klien/2/lihat", `{"alasan":"Cek laporan"}`)
	rec := do(srv, k, http.MethodPost, "/api/v1/master/klien/2/catatan", `{"halaman":"laporan-keuangan","periode":"2026-08","tingkat":"perbaiki","isi":"Saldo kas tidak cocok dengan rekening koran"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("tulis catatan: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, k, http.MethodPost, "/api/v1/master/klien/2/catatan", `{"tingkat":"darurat","isi":"x"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("tingkat ngawur: %d", rec.Code)
	}
	if rec := do(srv, k, http.MethodPost, "/api/v1/master/klien/1/catatan", `{"tingkat":"info","isi":"x"}`); rec.Code != http.StatusNotFound {
		t.Errorf("klien tidak ditugaskan: %d", rec.Code)
	}

	// klien: banner + daftar; belum jatuh tempo pengingat (bawaan 3 hari)
	me := getJSON[meResp](t, srv, qa, "/api/v1/auth/me")
	if me.CatatanTerbuka != 1 || me.CatatanIngat != 0 {
		t.Fatalf("me = terbuka %d ingat %d", me.CatatanTerbuka, me.CatatanIngat)
	}
	var list catatanKlienResp
	_ = json.NewDecoder(do(srv, qa, http.MethodGet, "/api/v1/catatan", "").Body).Decode(&list)
	if len(list.Catatan) != 1 || list.Catatan[0].Oleh != "Anggota konsultan" || list.Catatan[0].Peran != "Konsultan" || len(list.Pengingat) != 0 {
		t.Fatalf("daftar klien = %+v", list)
	}

	// 4 hari kemudian → jatuh tempo pengingat; setelah ditampilkan, jeda dihitung ulang
	tn := srv.tenantByID(2)
	tn.mu.Lock()
	tn.catatan[0].Dibuat = time.Now().Add(-4 * 24 * time.Hour)
	tn.mu.Unlock()
	_ = json.NewDecoder(do(srv, qa, http.MethodGet, "/api/v1/catatan", "").Body).Decode(&list)
	if !slices.Equal(list.Pengingat, []int{1}) {
		t.Fatalf("pengingat = %v", list.Pengingat)
	}
	do(srv, qa, http.MethodPost, "/api/v1/catatan/diingatkan", `{"ids":[1]}`)
	if me := getJSON[meResp](t, srv, qa, "/api/v1/auth/me"); me.CatatanIngat != 0 || tn.catatan[0].JumlahPengingat != 1 {
		t.Fatalf("setelah diingatkan: ingat %d, jumlah %d", me.CatatanIngat, tn.catatan[0].JumlahPengingat)
	}

	// Master mengubah aturan: interval 1 hari & semua tingkat
	if rec := do(srv, k, http.MethodPut, "/api/v1/master/pengingat", `{"aktif":true,"interval_hari":1,"tingkat_min":"info"}`); rec.Code != http.StatusForbidden {
		t.Errorf("konsultan ubah pengaturan: %d", rec.Code)
	}
	if rec := do(srv, m, http.MethodPut, "/api/v1/master/pengingat", `{"aktif":true,"interval_hari":0,"tingkat_min":"info"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("interval 0: %d", rec.Code)
	}
	if rec := do(srv, m, http.MethodPut, "/api/v1/master/pengingat", `{"aktif":true,"interval_hari":1,"tingkat_min":"info","kanal":["aplikasi","whatsapp"]}`); rec.Code != http.StatusOK {
		t.Fatalf("atur pengingat: %d %s", rec.Code, rec.Body)
	}
	if a := srv.aturanPengingat(); a.IntervalHari != 1 || !slices.Equal(a.Kanal, []string{"aplikasi"}) {
		t.Errorf("aturan = %+v (WA belum tersambung, harus aplikasi saja)", a)
	}

	// tindak lanjut: kasir tidak boleh; owner boleh, tercatat di riwayat akses
	if rec := do(srv, qa, http.MethodPost, "/api/v1/catatan/1/selesai", ""); rec.Code != http.StatusOK {
		t.Fatalf("selesai: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/catatan/1/selesai", ""); rec.Code != http.StatusConflict {
		t.Errorf("selesai dua kali: %d", rec.Code)
	}
	if me := getJSON[meResp](t, srv, qa, "/api/v1/auth/me"); me.CatatanTerbuka != 0 {
		t.Errorf("masih terbuka %d", me.CatatanTerbuka)
	}
	riwayat := do(srv, qa, http.MethodGet, "/api/v1/akses-tim", "").Body.String()
	if !strings.Contains(riwayat, "Tulis catatan konsultan (perlu diperbaiki) di Laporan Keuangan") || !strings.Contains(riwayat, "sudah diperbaiki") {
		t.Errorf("riwayat akses: %s", riwayat)
	}
	// Master melihat semua catatan + statusnya
	var mc struct {
		Catatan []catatanRow `json:"catatan"`
	}
	_ = json.NewDecoder(do(srv, m, http.MethodGet, "/api/v1/master/catatan", "").Body).Decode(&mc)
	if len(mc.Catatan) != 1 || mc.Catatan[0].Status != finance.CatatanSelesai || mc.Catatan[0].KlienID != 2 {
		t.Fatalf("catatan master = %+v", mc.Catatan)
	}
}
