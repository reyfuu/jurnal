package main

import (
	"encoding/json"
	"net/http"
	"pajak/internal/finance"
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

// Checklist tester No 20: pindah dana Bank → Kas dan Kas → Bank menggeser saldo per rekening,
// tidak mengubah laba, dan saldonya terbawa ke bulan berikutnya.
func TestCatat_PindahDanaAntarRekening(t *testing.T) {
	srv, qa := usahaBaru(t)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/opening", `{"tanggal":"2026-01-01","rincian":[{"nama":"Kas","kategori":"kas","nominal":100000},{"nama":"Bank","kategori":"kas","nominal":1000000},{"nama":"Modal Pemilik","kategori":"modal","nominal":1100000}]}`); rec.Code != http.StatusOK {
		t.Fatalf("saldo awal: %d %s", rec.Code, rec.Body)
	}
	for _, body := range []string{
		`{"tanggal":"2026-01-10","keterangan":"Tarik tunai","baris":[{"akun":"1101","debit":300000},{"akun":"1102","kredit":300000}]}`,
		`{"tanggal":"2026-01-20","keterangan":"Setor ke bank","baris":[{"akun":"1102","debit":50000},{"akun":"9102","debit":2500},{"akun":"1101","kredit":52500}]}`,
	} {
		if rec := do(srv, qa, http.MethodPost, "/api/v1/catat", body); rec.Code != http.StatusCreated {
			t.Fatalf("catat: %d %s", rec.Code, rec.Body)
		}
	}
	saldo := func(period string) map[string]int64 {
		var rep struct {
			TrialBalance []coa.Balance `json:"trial_balance"`
		}
		_ = json.Unmarshal(do(srv, qa, http.MethodGet, "/api/v1/reports/coa?period="+period, "").Body.Bytes(), &rep)
		out := map[string]int64{}
		for _, b := range rep.TrialBalance {
			out[b.Kode] = b.Saldo
		}
		return out
	}
	for _, period := range []string{"2026-01", "2026-02"} {
		s := saldo(period)
		if s["1101"] != 347_500 || s["1102"] != 750_000 {
			t.Fatalf("%s: kas=%d bank=%d, mau 347.500 & 750.000", period, s["1101"], s["1102"])
		}
	}
	// form transaksi lama tidak bisa memilih rekening asal: harus lewat Catat
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-01-25","jenis":"Pindah Dana","nominal":1000,"lunas":true,"akun":"1101","lawan":"1102"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("form pindah dana harus ditolak: %d", rec.Code)
	}
	is := getJSON[finance.IncomeStatement](t, srv, qa, "/api/v1/reports/financial?period=2026-01")
	bs := getJSON[finance.BalanceSheet](t, srv, qa, "/api/v1/reports/balance-sheet?period=2026-01")
	if is.NetProfit != -2_500 || bs.Assets.Cash != 1_097_500 {
		t.Fatalf("laba bersih %d (mau -2.500, hanya biaya admin), kas & bank %d", is.NetProfit, bs.Assets.Cash)
	}
}
