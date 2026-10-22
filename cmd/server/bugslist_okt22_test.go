package main

import (
	"net/http"
	"strings"
	"testing"
)

// Bugslist 22 Okt #7: setelah Tutup Buku, layar Saldo Awal harus tahu bahwa saldo awal
// terkunci (bukan menampilkan tombol Ubah lalu gagal 409 saat disimpan).
func TestSaldoAwal_TerkunciSetelahTutupBuku(t *testing.T) {
	srv, qa := usahaBaru(t)
	seimbang := `{"tanggal":"2026-01-01","rincian":[{"nama":"Kas","kategori":"kas","nominal":5000000},{"nama":"Modal Pemilik","kategori":"modal","nominal":5000000}]}`
	if rec := do(srv, qa, http.MethodPut, "/api/v1/opening", seimbang); rec.Code != http.StatusOK {
		t.Fatalf("simpan saldo awal: %d %s", rec.Code, rec.Body)
	}
	if o := getJSON[openingResp](t, srv, qa, "/api/v1/opening"); o.Terkunci != "" {
		t.Fatalf("sebelum tutup buku tidak boleh terkunci: %q", o.Terkunci)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/tutup-buku", `{"periode":"2026-01"}`); rec.Code != http.StatusOK {
		t.Fatalf("tutup buku: %d %s", rec.Code, rec.Body)
	}
	o := getJSON[openingResp](t, srv, qa, "/api/v1/opening")
	if !strings.Contains(o.Terkunci, "Januari 2026") || !strings.Contains(o.Terkunci, "Tutup Buku") {
		t.Fatalf("terkunci = %q, mau menyebut bulan & menu Tutup Buku", o.Terkunci)
	}
	rec := do(srv, qa, http.MethodPut, "/api/v1/opening", seimbang)
	if rec.Code != http.StatusConflict || strings.TrimSpace(rec.Body.String()) != o.Terkunci {
		t.Fatalf("simpan setelah tutup buku: %d %q, mau 409 dengan pesan yang sama", rec.Code, rec.Body)
	}
}

// Panic di handler tidak boleh memutus koneksi (proxy lalu menampilkan halaman 5xx HTML):
// server membalas 500 teks polos.
func TestPanicDibalas500(t *testing.T) {
	srv := NewServer()
	srv.mux.HandleFunc("GET /api/v1/uji-panic", func(http.ResponseWriter, *http.Request) { panic("uji") })
	rec := do(srv, nil, http.MethodGet, "/api/v1/uji-panic", "")
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "<") {
		t.Fatalf("panic: %d %q", rec.Code, rec.Body)
	}
}
