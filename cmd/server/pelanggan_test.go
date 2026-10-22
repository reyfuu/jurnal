package main

import (
	"net/http"
	"strconv"
	"testing"

	"pajak/internal/finance"
)

// Checklist tester No 24: daftar pelanggan (master) seperti daftar supplier, plus ambil nama
// pelanggan dari data lama (transaksi penjualan & invoice) tanpa nama ganda.
func TestPelanggan(t *testing.T) {
	srv, qa := usahaBaru(t)
	p := buat[finance.Pelanggan](t, srv, qa, "/api/v1/pelanggan", `{"nama":"PT Maju","alamat":"Jl. Raya 1","npwp":"01.234","tempo_hari":14}`)
	if p.ID == 0 || p.NPWP != "01.234" {
		t.Fatalf("buat: %+v", p)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/pelanggan", `{"nama":"pt maju "}`); rec.Code != http.StatusConflict {
		t.Fatalf("nama kembar harus ditolak: %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/pelanggan", `{"nama":" "}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("nama kosong harus ditolak: %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodPut, "/api/v1/pelanggan/"+strconv.Itoa(p.ID), `{"nama":"PT Maju Jaya","telepon":"0812"}`); rec.Code != http.StatusOK {
		t.Fatalf("ubah: %d %s", rec.Code, rec.Body)
	}
	for _, tx := range []string{
		`{"tanggal":"2026-10-01","jenis":"Penjualan","nominal":1000,"lunas":true,"pihak":"Toko Sinar"}`,
		`{"tanggal":"2026-10-02","jenis":"Penjualan","nominal":1000,"lunas":true,"pihak":"toko sinar"}`,
		`{"tanggal":"2026-10-03","jenis":"Penjualan","nominal":1000,"lunas":true,"pihak":"PT Maju Jaya"}`,
		`{"tanggal":"2026-10-04","jenis":"Biaya Lain","nominal":1000,"lunas":true,"pihak":"PLN"}`,
	} {
		buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", tx)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/pelanggan/dari-data", ""); rec.Code != http.StatusOK {
		t.Fatalf("dari data: %d %s", rec.Code, rec.Body)
	}
	list := getJSON[[]finance.Pelanggan](t, srv, qa, "/api/v1/pelanggan")
	if len(list) != 2 || list[0].Nama != "PT Maju Jaya" || list[1].Nama != "Toko Sinar" {
		t.Fatalf("daftar: %+v", list)
	}
	if rec := do(srv, qa, http.MethodDelete, "/api/v1/pelanggan/"+strconv.Itoa(p.ID), ""); rec.Code != http.StatusNoContent {
		t.Fatalf("hapus: %d", rec.Code)
	}
}
