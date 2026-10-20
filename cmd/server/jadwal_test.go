package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"pajak/internal/finance"
)

func TestJadwalPengingat_MasterAturKlienLihat(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv, m, qa, k := siapKlien(t) // qa = Owner usaha 2 (punya profil), k = konsultan

	var awal struct {
		Item []finance.JadwalPengingat `json:"item"`
	}
	rec := do(srv, k, http.MethodGet, "/api/v1/master/jadwal", "")
	_ = json.NewDecoder(rec.Body).Decode(&awal)
	if rec.Code != http.StatusOK || len(awal.Item) != len(finance.JadwalBawaan().Item) {
		t.Fatalf("konsultan lihat jadwal bawaan: %d %d", rec.Code, len(awal.Item))
	}

	hari := time.Now().In(finance.WIB).Day()
	body := fmt.Sprintf(`{"item":[{"judul":"Kirim rekap ke konsultan","frekuensi":"bulanan","tanggal":%d,"hari_sebelum":0,"untuk":"semua","aktif":true},
		{"judul":"Nonaktif","frekuensi":"bulanan","tanggal":%d,"hari_sebelum":0,"untuk":"semua","aktif":false}]}`, hari, hari)
	if rec := do(srv, k, http.MethodPut, "/api/v1/master/jadwal", body); rec.Code != http.StatusForbidden {
		t.Errorf("konsultan ubah jadwal: %d", rec.Code)
	}
	if rec := do(srv, m, http.MethodPut, "/api/v1/master/jadwal", `{"item":[{"judul":"","frekuensi":"bulanan","tanggal":1,"untuk":"semua"}]}`); rec.Code != http.StatusBadRequest {
		t.Errorf("jadwal tanpa judul: %d", rec.Code)
	}
	if rec := do(srv, m, http.MethodPut, "/api/v1/master/jadwal", body); rec.Code != http.StatusOK {
		t.Fatalf("master simpan jadwal: %d %s", rec.Code, rec.Body)
	}

	me := getJSON[meResp](t, srv, qa, "/api/v1/auth/me")
	if len(me.Jadwal) != 1 || me.Jadwal[0].Judul != "Kirim rekap ke konsultan" || me.Jadwal[0].SisaHari != 0 {
		t.Fatalf("klien me.jadwal = %+v", me.Jadwal)
	}
	riwayat := do(srv, m, http.MethodGet, "/api/v1/master/audit", "").Body.String()
	if !strings.Contains(riwayat, "Ubah jadwal pengingat") {
		t.Error("perubahan jadwal tidak tercatat di audit log")
	}

	// Tersimpan & dimuat ulang (mode berkas).
	srv2 := NewServer()
	var lagi struct {
		Item []finance.JadwalPengingat `json:"item"`
	}
	_ = json.NewDecoder(do(srv2, loginPass(t, srv2, "andi@masterpajak.id", masterPass), http.MethodGet, "/api/v1/master/jadwal", "").Body).Decode(&lagi)
	if len(lagi.Item) != 2 || lagi.Item[0].ID == 0 {
		t.Errorf("setelah muat ulang: %+v", lagi.Item)
	}
}
