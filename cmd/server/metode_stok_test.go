package main

import (
	"net/http"
	"strings"
	"testing"

	"pajak/internal/finance"
)

// Metode nilai stok mengikuti pola metode penyusutan: bawaan FIFO, dipilih sekali lalu dikunci.
func TestMetodeStok_DipilihSekaliLaluDikunci(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	qa := register(t, srv, registerBody)
	aktifkan(t, srv, qa)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/business", dagangCV); rec.Code != http.StatusOK {
		t.Fatalf("profil: %d %s", rec.Code, rec.Body)
	}
	if m := getJSON[metodeStokResp](t, srv, qa, "/api/v1/stok/metode"); m.Metode != finance.MetodeFIFO || m.Terkunci {
		t.Fatalf("bawaan: %+v", m)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/stok/metode", `{"metode":"rata_rata"}`); rec.Code != http.StatusOK {
		t.Fatalf("pilih: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/stok/metode", `{"metode":"fifo"}`); rec.Code != http.StatusConflict {
		t.Fatalf("ganti lewat pilih harus ditolak: %d", rec.Code)
	}
	ganti := strings.Replace(dagangCV, "{", `{"metode_nilai_stok":"fifo",`, 1)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/business", ganti); rec.Code != http.StatusConflict {
		t.Fatalf("ganti lewat profil harus ditolak: %d %s", rec.Code, rec.Body)
	}
	// Formulir profil yang tidak mengirim metode tetap aman dan metodenya tidak hilang.
	if rec := do(srv, qa, http.MethodPut, "/api/v1/business", dagangCV); rec.Code != http.StatusOK {
		t.Fatalf("simpan profil tanpa metode: %d %s", rec.Code, rec.Body)
	}
	if m := getJSON[metodeStokResp](t, srv, qa, "/api/v1/stok/metode"); m.Metode != finance.MetodeRataRata || !m.Terkunci {
		t.Fatalf("terkunci: %+v", m)
	}
}

// Kartu stok dimulai tanpa memilih: metode bawaan (FIFO) tercatat dan dikunci sejak titik nol.
func TestMetodeStok_MulaiKartuMengunciBawaan(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	qa := register(t, srv, registerBody)
	aktifkan(t, srv, qa)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/business", dagangCV); rec.Code != http.StatusOK {
		t.Fatalf("profil: %d %s", rec.Code, rec.Body)
	}
	kopi := buat[finance.StokItem](t, srv, qa, "/api/v1/stok", `{"nama":"Kopi","kategori":"Barang","satuan":"kg","harga_pokok":200000}`)
	buat[finance.GerakStok](t, srv, qa, "/api/v1/stok/awal", `{"tanggal":"2026-01-01","baris":[{"stok_id":`+itoa(kopi.ID)+`,"fisik":1000,"harga_unit":200000}]}`)
	if m := getJSON[metodeStokResp](t, srv, qa, "/api/v1/stok/metode"); m.Metode != finance.MetodeFIFO || !m.Terkunci {
		t.Fatalf("setelah mulai kartu: %+v", m)
	}
	type bisnis struct {
		Profile finance.BusinessProfile `json:"profile"`
	}
	if b := getJSON[bisnis](t, srv, qa, "/api/v1/business"); b.Profile.MetodeNilaiStok != finance.MetodeFIFO {
		t.Fatalf("metode bawaan harus tersimpan di profil: %+v", b.Profile)
	}
}

// Usaha yang memulai kartu stok sebelum ada pilihan metode tetap rata-rata (angka lama tidak berubah).
func TestMetodeStok_KartuLamaTetapRataRata(t *testing.T) {
	srv := NewServer()
	tn := srv.tenants[1]
	tn.mu.Lock()
	tn.gerak = append(tn.gerak, finance.GerakStok{ID: 99, Tanggal: "2026-01-01", Jenis: finance.GerakAwal})
	if tn.profile != nil {
		tn.profile.MetodeNilaiStok = ""
	}
	m, kunci := tn.metodeStokLocked()
	tn.mu.Unlock()
	if m != finance.MetodeRataRata || !kunci {
		t.Fatalf("kartu lama: %s terkunci=%v", m, kunci)
	}
}
