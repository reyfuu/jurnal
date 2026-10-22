package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"pajak/internal/finance"
)

// Dua arah: nota dicatat dulu lalu impor mutasi → baris mutasi ditandai sudah tercatat & tidak
// dicentang; utang yang dicatat dulu → baris mutasi menjadi pelunasannya, bukan transaksi baru.
// Sebaliknya, halaman Catat transaksi bisa menanyakan transaksi mirip sebelum menyimpan.
func TestDobel_JurnalDuluLaluMutasi(t *testing.T) {
	srv, qa := usahaBaru(t)
	if rec := do(srv, qa, http.MethodPut, "/api/v1/opening", `{"tanggal":"2026-09-01","rincian":[{"nama":"Kas","kategori":"kas","nominal":5000000},{"nama":"Modal Pemilik","kategori":"modal","nominal":5000000}]}`); rec.Code != http.StatusOK {
		t.Fatalf("saldo awal: %d %s", rec.Code, rec.Body)
	}
	// Dari foto nota: belanja lunas Rp125.000; dan pembelian tempo Rp400.000 (utang).
	for _, body := range []string{
		`{"tanggal":"2026-09-10","sumber":"foto","baris":[{"akun":"6190","debit":125000},{"akun":"1101","kredit":125000}]}`,
		`{"tanggal":"2026-09-05","pihak":"CV Maju","baris":[{"akun":"6190","debit":400000},{"akun":"2101","kredit":400000}]}`,
	} {
		if rec := do(srv, qa, http.MethodPost, "/api/v1/catat", body); rec.Code != http.StatusCreated {
			t.Fatalf("catat: %d %s", rec.Code, rec.Body)
		}
	}
	m := getJSON[[]finance.Mirip](t, srv, qa, "/api/v1/transactions/mirip?tanggal=2026-09-11&nominal=125000&arah=keluar")
	if len(m) != 1 || m[0].Sumber != "foto" {
		t.Fatalf("cek mirip sebelum simpan: %+v", m)
	}

	h := unggahMutasi(t, srv, qa, "Tanggal;Keterangan;Debet;Kredit\n11/09/2026;DEBIT TOKO BANGUNAN;125.000;\n20/09/2026;TRSF KE CV MAJU;400.000;\n22/09/2026;BIAYA ADM;10.000;\n")
	if len(h.Rows) != 3 || !h.Rows[0].Duplikat || h.Rows[0].Pilih || h.Rows[0].Mirip == nil || h.Rows[0].Mirip.Sumber != "foto" {
		t.Fatalf("baris sudah tercatat: %+v", h.Rows[0])
	}
	if h.Rows[1].Lunasi == 0 || !h.Rows[1].Pilih || h.Rows[2].Duplikat || !h.Rows[2].Pilih {
		t.Fatalf("pelunasan & baris baru: %+v / %+v", h.Rows[1], h.Rows[2])
	}
	b, _ := json.Marshal(mutasiCommitReq{Rows: h.Rows})
	rec := do(srv, qa, http.MethodPost, "/api/v1/transactions/mutasi/commit", string(b))
	var hasil mutasiCommitResp
	_ = json.Unmarshal(rec.Body.Bytes(), &hasil)
	if rec.Code != http.StatusOK || hasil.Disimpan != 1 || hasil.Dilunasi != 1 {
		t.Fatalf("commit: %d %+v", rec.Code, hasil)
	}
	um := getJSON[finance.UmurSaldoResult](t, srv, qa, "/api/v1/reports/umur?jenis=utang&per=2026-09-30")
	if um.Total != 0 {
		t.Fatalf("utang CV Maju harus lunas lewat mutasi: %+v", um)
	}
	if p := getJSON[[][2]finance.Transaction](t, srv, qa, "/api/v1/transactions/dobel?period=2026-09"); len(p) != 0 {
		t.Fatalf("tidak boleh ada dobel: %+v", p)
	}
}
