package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"pajak/internal/coa"
)

// Catatan jurnal multi-akun tersimpan sebagai satu grup transaksi: masuk neraca saldo per akun
// persis, bisa dibuka ulang, tidak bisa diubah satu per satu, dan terhapus bersama.
func TestCatat_JurnalMultiAkun(t *testing.T) {
	srv, qa := usahaBaru(t)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/opening", `{"tanggal":"2026-01-01","rincian":[{"nama":"Kas","kategori":"kas","nominal":1000000},{"nama":"Modal Pemilik","kategori":"modal","nominal":1000000}]}`); rec.Code != http.StatusOK {
		t.Fatalf("saldo awal: %d %s", rec.Code, rec.Body)
	}
	body := `{"tanggal":"2026-01-10","pihak":"Toko Sumber Rejeki","no_bukti":"SR-1","keterangan":"Belanja","baris":[
		{"akun":"6106","keterangan":"Ongkir","debit":10500},
		{"akun":"6190","keterangan":"Lain-lain","debit":30000},
		{"akun":"1101","kredit":25000},
		{"akun":"2101","kredit":15500,"jatuh_tempo":"2026-02-10"}]}`
	rec := do(srv, qa, http.MethodPost, "/api/v1/catat", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("catat: %d %s", rec.Code, rec.Body)
	}
	var c catatResp
	_ = json.Unmarshal(rec.Body.Bytes(), &c)
	if c.Grup == 0 || len(c.Transaksi) != 3 {
		t.Fatalf("hasil: %+v", c)
	}
	var rep struct {
		TrialBalance []coa.Balance `json:"trial_balance"`
	}
	_ = json.Unmarshal(do(srv, qa, http.MethodGet, "/api/v1/reports/coa?period=2026-01", "").Body.Bytes(), &rep)
	saldo := map[string]int64{}
	for _, b := range rep.TrialBalance {
		saldo[b.Kode] = b.Saldo
	}
	if saldo["6106"] != 10_500 || saldo["6190"] != 30_000 || saldo["1101"] != 975_000 || saldo["2101"] != 15_500 {
		t.Fatalf("neraca saldo: 6106=%d 6190=%d 1101=%d 2101=%d", saldo["6106"], saldo["6190"], saldo["1101"], saldo["2101"])
	}
	url := "/api/v1/catat/" + strconv.Itoa(c.Grup)
	g := getJSON[catatResp](t, srv, qa, url)
	if g.Pihak != "Toko Sumber Rejeki" || g.NoBukti != "SR-1" || len(g.Baris) != 4 {
		t.Fatalf("buka ulang: %+v", g)
	}
	if rec := do(srv, qa, http.MethodPut, "/api/v1/transactions/"+strconv.Itoa(c.Transaksi[0]), `{"tanggal":"2026-01-10","jenis":"Biaya Lain","nominal":1,"lunas":true}`); rec.Code != http.StatusConflict {
		t.Fatalf("ubah satu baris grup harus ditolak: %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/catat", `{"tanggal":"2026-01-10","baris":[{"akun":"6190","debit":10},{"akun":"1101","kredit":9}]}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("tidak seimbang harus ditolak: %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodDelete, url, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("hapus: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, qa, http.MethodGet, url, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("catatan terhapus harus hilang: %d", rec.Code)
	}
}

// Checklist tester No 27: halaman Catat hanya punya satu keterangan (kepala). Membuka catatan
// harus mengembalikan keterangan itu supaya simpan ulang tidak mengosongkannya; catatan lama
// dengan keterangan per baris digabung jadi satu.
func TestCatat_KeteranganKepalaBertahanSaatDiubah(t *testing.T) {
	srv, qa := usahaBaru(t)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/opening", `{"tanggal":"2026-01-01","rincian":[{"nama":"Kas","kategori":"kas","nominal":1000000},{"nama":"Modal Pemilik","kategori":"modal","nominal":1000000}]}`); rec.Code != http.StatusOK {
		t.Fatalf("saldo awal: %d %s", rec.Code, rec.Body)
	}
	buatCatat := func(body string) string {
		rec := do(srv, qa, http.MethodPost, "/api/v1/catat", body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("catat: %d %s", rec.Code, rec.Body)
		}
		var c catatResp
		_ = json.Unmarshal(rec.Body.Bytes(), &c)
		return "/api/v1/catat/" + strconv.Itoa(c.Grup)
	}
	url := buatCatat(`{"tanggal":"2026-01-10","keterangan":"Belanja bahan","baris":[{"akun":"6106","debit":10000},{"akun":"6190","debit":5000},{"akun":"1101","kredit":15000}]}`)
	if g := getJSON[catatResp](t, srv, qa, url); g.Keterangan != "Belanja bahan" {
		t.Fatalf("keterangan kepala = %q", g.Keterangan)
	}
	// simpan ulang seperti klien baru: keterangan baris kosong, kepala dari GET
	if rec := do(srv, qa, http.MethodPut, url, `{"tanggal":"2026-01-10","keterangan":"Belanja bahan","baris":[{"akun":"6106","debit":12000},{"akun":"6190","debit":5000},{"akun":"1101","kredit":17000}]}`); rec.Code != http.StatusOK {
		t.Fatalf("ubah: %d %s", rec.Code, rec.Body)
	}
	if g := getJSON[catatResp](t, srv, qa, url); g.Keterangan != "Belanja bahan" {
		t.Fatalf("keterangan setelah diubah = %q", g.Keterangan)
	}
	lama := buatCatat(`{"tanggal":"2026-01-11","baris":[{"akun":"6106","keterangan":"Ongkir","debit":10000},{"akun":"6190","keterangan":"Lain-lain","debit":5000},{"akun":"1101","kredit":15000}]}`)
	if g := getJSON[catatResp](t, srv, qa, lama); g.Keterangan != "Ongkir; Lain-lain" {
		t.Fatalf("keterangan per baris lama digabung = %q", g.Keterangan)
	}
}
