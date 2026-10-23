package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"pajak/internal/finance"
)

// No. Referensi: urut per bulan tanggal transaksi, satu catatan satu nomor, backdate ikut
// bulannya; hapus = batal (nomor tidak dipakai ulang), Owner bisa memulihkan.
func TestNoReferensiDanBatal(t *testing.T) {
	srv, qa := usahaBaru(t)
	a := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-10-01","jenis":"Penjualan","nominal":100000,"lunas":true}`)
	b := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-10-02","jenis":"Biaya Lain","nominal":50000,"lunas":true}`)
	if a.Ref != "TRX-2610-0001" || b.Ref != "TRX-2610-0002" {
		t.Fatalf("nomor urut: %q %q", a.Ref, b.Ref)
	}
	var c struct{ Grup int }
	rec := do(srv, qa, http.MethodPost, "/api/v1/catat", `{"tanggal":"2026-10-03","baris":[{"akun":"6190","debit":30000},{"akun":"6104","debit":20000},{"akun":"1101","kredit":50000}]}`)
	_ = json.NewDecoder(rec.Body).Decode(&c)
	if rec.Code != http.StatusCreated {
		t.Fatalf("catat: %d %s", rec.Code, rec.Body)
	}
	mundur := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-09-28","jenis":"Biaya Lain","nominal":10000,"lunas":true}`)
	if mundur.Ref != "TRX-2609-0001" {
		t.Errorf("backdate ikut bulan transaksi: %q", mundur.Ref)
	}
	var list struct{ Transactions []finance.Transaction }
	_ = json.NewDecoder(do(srv, qa, http.MethodGet, "/api/v1/transactions", "").Body).Decode(&list)
	ref := map[string]int{}
	for _, x := range list.Transactions {
		if x.Grup == c.Grup {
			ref[x.Ref]++
		}
	}
	if len(ref) != 1 || ref["TRX-2610-0003"] < 2 {
		t.Errorf("satu catatan harus satu nomor: %v", ref)
	}
	// ubah: nomor tetap
	if rec := do(srv, qa, http.MethodPut, "/api/v1/transactions/"+strconv.Itoa(b.ID), `{"tanggal":"2026-10-05","jenis":"Biaya Lain","nominal":60000,"lunas":true}`); rec.Code != http.StatusOK {
		t.Fatalf("ubah: %d", rec.Code)
	}
	// batal
	if rec := do(srv, qa, http.MethodDelete, "/api/v1/transactions/"+strconv.Itoa(b.ID), ""); rec.Code != http.StatusBadRequest {
		t.Errorf("batal tanpa alasan harus ditolak: %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodDelete, "/api/v1/transactions/"+strconv.Itoa(b.ID)+"?alasan=input+dobel", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("batal: %d %s", rec.Code, rec.Body)
	}
	batal := getJSON[[]finance.Transaction](t, srv, qa, "/api/v1/transactions/batal")
	if len(batal) != 1 || batal[0].Ref != "TRX-2610-0002" || batal[0].Batal == nil || batal[0].Batal.Alasan != "input dobel" || batal[0].Nominal != 60000 {
		t.Fatalf("riwayat batal: %+v", batal)
	}
	if d := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-10-09","jenis":"Penjualan","nominal":1000,"lunas":true}`); d.Ref != "TRX-2610-0004" {
		t.Errorf("nomor batal tidak dipakai ulang: %q", d.Ref)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions/"+strconv.Itoa(b.ID)+"/pulihkan", ""); rec.Code != http.StatusOK {
		t.Fatalf("pulihkan: %d %s", rec.Code, rec.Body)
	}
	_ = json.NewDecoder(do(srv, qa, http.MethodGet, "/api/v1/transactions", "").Body).Decode(&list)
	pulih := false
	for _, x := range list.Transactions {
		pulih = pulih || (x.ID == b.ID && x.Ref == "TRX-2610-0002" && x.Batal == nil)
	}
	if !pulih || len(getJSON[[]finance.Transaction](t, srv, qa, "/api/v1/transactions/batal")) != 0 {
		t.Error("transaksi harus kembali dengan nomor yang sama")
	}
}
