package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"pajak/internal/finance"
)

// Checklist tester No 31: nomor jurnal JU-YYYYMM-NNNN per bulan transaksi, tidak berubah saat
// diedit, tidak dipakai ulang setelah dihapus; satu catatan multi-akun berbagi satu nomor.
func TestNomorJurnal(t *testing.T) {
	srv, qa := usahaBaru(t)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/opening", `{"tanggal":"2026-10-01","rincian":[{"nama":"Kas","kategori":"kas","nominal":1000000},{"nama":"Modal Pemilik","kategori":"modal","nominal":1000000}]}`); rec.Code != http.StatusOK {
		t.Fatalf("saldo awal: %d %s", rec.Code, rec.Body)
	}
	a := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-10-05","jenis":"Penjualan","nominal":100000,"lunas":true}`)
	b := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-10-06","jenis":"Biaya Lain","nominal":20000,"lunas":true}`)
	nomor := func() map[int]string {
		var res struct {
			Transactions []finance.Transaction `json:"transactions"`
		}
		_ = json.Unmarshal(do(srv, qa, http.MethodGet, "/api/v1/transactions", "").Body.Bytes(), &res)
		out := map[int]string{}
		for _, x := range res.Transactions {
			out[x.ID] = x.NoJurnal
		}
		return out
	}
	if n := nomor(); n[a.ID] != "JU-202610-0001" || n[b.ID] != "JU-202610-0002" {
		t.Fatalf("nomor awal: %v", n)
	}
	if rec := do(srv, qa, http.MethodDelete, "/api/v1/transactions/"+strconv.Itoa(a.ID), ""); rec.Code >= 300 {
		t.Fatalf("hapus: %d", rec.Code)
	}
	c := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-10-07","jenis":"Penjualan","nominal":5000,"lunas":true}`)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/transactions/"+strconv.Itoa(b.ID), `{"tanggal":"2026-11-01","jenis":"Biaya Lain","nominal":20000,"lunas":true}`); rec.Code != http.StatusOK {
		t.Fatalf("ubah: %d %s", rec.Code, rec.Body)
	}
	rec := do(srv, qa, http.MethodPost, "/api/v1/catat", `{"tanggal":"2026-11-03","keterangan":"Belanja","baris":[{"akun":"6106","debit":1000},{"akun":"6190","debit":2000},{"akun":"1101","kredit":3000}]}`)
	var cat catatResp
	_ = json.Unmarshal(rec.Body.Bytes(), &cat)
	n := nomor()
	if n[c.ID] != "JU-202610-0003" || n[b.ID] != "JU-202610-0002" {
		t.Fatalf("tidak dipakai ulang / tetap saat diedit: %v", n)
	}
	if len(cat.Transaksi) != 2 || n[cat.Transaksi[0]] != "JU-202611-0001" || n[cat.Transaksi[1]] != "JU-202611-0001" {
		t.Fatalf("satu catatan satu nomor: %v %v", cat.Transaksi, n)
	}
}
