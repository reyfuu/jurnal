package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"pajak/internal/finance"
)

// Mutasi "sama" — pindah akun bayar: baris mutasi yang cocok dengan catatan manual dibuktikan
// bergerak di rekening yang diimpor, jadi Lawan catatan itu dipindah ke rekening tersebut tanpa
// membuat transaksi baru.

func commitMutasiRekening(t *testing.T, srv *Server, qa *http.Cookie, rekening string, rows ...finance.MutasiRow) (int, mutasiCommitResp, string) {
	t.Helper()
	b, _ := json.Marshal(mutasiCommitReq{Rows: rows, Rekening: rekening})
	rec := do(srv, qa, http.MethodPost, "/api/v1/transactions/mutasi/commit", string(b))
	var res mutasiCommitResp
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	return rec.Code, res, strings.TrimSpace(rec.Body.String())
}

func saldoPerAkun(t *testing.T, srv *Server, qa *http.Cookie, period string) map[string]int64 {
	t.Helper()
	var rep struct {
		TrialBalance []struct {
			Kode  string `json:"kode"`
			Saldo int64  `json:"saldo"`
		} `json:"trial_balance"`
	}
	_ = json.Unmarshal(do(srv, qa, http.MethodGet, "/api/v1/reports/coa?period="+period, "").Body.Bytes(), &rep)
	out := map[string]int64{}
	for _, b := range rep.TrialBalance {
		out[b.Kode] = b.Saldo
	}
	return out
}

// catatLama: transaksi manual lama yang dibayar dari Kas (Lawan kosong, dicatat sebelum rekening
// bawaan Bank; formulir Transaksi tidak bisa memilih rekening dan Catat selalu ber-Grup).
func catatLama(t *testing.T, srv *Server, qa *http.Cookie, body string) finance.Transaction {
	t.Helper()
	x := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", body)
	tn := tenantByEmail(t, srv, qa)
	tn.mu.Lock()
	defer tn.mu.Unlock()
	for i := range tn.txs {
		if tn.txs[i].ID == x.ID {
			tn.txs[i].Lawan = ""
			return tn.txs[i]
		}
	}
	t.Fatalf("transaksi #%d tidak ada", x.ID)
	return x
}

const csvListrik = "Tanggal;Keterangan;Debet;Kredit\n11/01/2026;BIAYA LISTRIK PLN;500.000;\n"

// Catatan manual lama dibayar dari Kas Besar (1102, Lawan kosong); mutasi Bank 1111 yang dijawab "sama — pindah"
// memindah Lawan-nya ke 1111: jumlah transaksi tetap, saldo kas naik & bank turun.
func TestMutasi_SamaPindahAkunBayar(t *testing.T) {
	srv, qa := usahaManufaktur(t)
	manual := catatLama(t, srv, qa, `{"tanggal":"2026-01-10","jenis":"Biaya Lain","nominal":500000,"lunas":true}`)
	sebelum := saldoPerAkun(t, srv, qa, "2026-01")
	h := unggahMutasi(t, srv, qa, csvListrik)
	if len(h.Rows) != 1 || !h.Rows[0].Duplikat || h.Rows[0].Mirip == nil || h.Rows[0].Mirip.ID != manual.ID || h.Rows[0].Mirip.Lawan != "1102" {
		t.Fatalf("baris harus cocok dengan catatan manual (lawan 1102): %+v", h.Rows)
	}
	r := h.Rows[0]
	r.Pilih, r.Pindah = true, manual.ID
	// Rekening harus akun kas/bank.
	if code, _, body := commitMutasiRekening(t, srv, qa, "5101", r); code != http.StatusBadRequest {
		t.Fatalf("rekening bukan kas/bank: %d %s", code, body)
	}
	code, res, body := commitMutasiRekening(t, srv, qa, "1111", r)
	if code != http.StatusOK || res.Dipindah != 1 || res.Disimpan != 0 || len(res.Errors) != 0 {
		t.Fatalf("commit: %d %s", code, body)
	}
	got := txsOf(t, srv, qa)
	if len(got) != 1 || got[0].ID != manual.ID || got[0].Lawan != "1111" || got[0].Nominal != 500000 || got[0].Jenis != "Biaya Lain" {
		t.Fatalf("transaksi sesudah pindah: %+v", got)
	}
	sesudah := saldoPerAkun(t, srv, qa, "2026-01")
	if sesudah["1102"]-sebelum["1102"] != 500000 || sesudah["1111"]-sebelum["1111"] != -500000 {
		t.Errorf("saldo kas %d→%d, bank %d→%d", sebelum["1102"], sesudah["1102"], sebelum["1111"], sesudah["1111"])
	}
	// Pratinjau ulang: catatan itu kini sudah di rekening 1111.
	if h := unggahMutasi(t, srv, qa, csvListrik); h.Rows[0].Mirip == nil || h.Rows[0].Mirip.Lawan != "1111" {
		t.Errorf("pratinjau sesudah pindah: %+v", h.Rows[0].Mirip)
	}
}

// Transaksi invoice tidak dipindah akun bayarnya dari impor mutasi (ubah lewat menu Invoice).
func TestMutasi_PindahTransaksiInvoiceDitolak(t *testing.T) {
	srv, qa := usahaManufaktur(t)
	rec := do(srv, qa, http.MethodPost, "/api/v1/invoice", `{"nomor":"INV-1","status":"lunas","tanggal":"2026-01-10","jatuh_tempo":"2026-01-10","tanggal_lunas":"2026-01-10",`+
		`"pelanggan_nama":"PT Uji","items":[{"deskripsi":"Jasa","qty":1,"harga_satuan":500000}],"diskon_rp":0,"ppn_bps":0}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("invoice: %d %s", rec.Code, rec.Body)
	}
	txs := txsOf(t, srv, qa)
	if len(txs) != 1 || txs[0].InvoiceID == 0 || !txs[0].Lunas {
		t.Fatalf("transaksi invoice: %+v", txs)
	}
	inv := txs[0]
	h := unggahMutasi(t, srv, qa, "Tanggal;Keterangan;Debet;Kredit\n11/01/2026;TRSF CR PT UJI;;500.000\n")
	if len(h.Rows) != 1 || h.Rows[0].Mirip == nil || h.Rows[0].Mirip.ID != inv.ID || h.Rows[0].Mirip.InvoiceID == 0 {
		t.Fatalf("baris harus cocok dengan transaksi invoice: %+v", h.Rows)
	}
	r := h.Rows[0]
	r.Pilih, r.Pindah = true, inv.ID
	code, res, body := commitMutasiRekening(t, srv, qa, "1111", r)
	if code != http.StatusBadRequest || !strings.Contains(body, "baris 2: "+errTxInvoice.Error()) {
		t.Fatalf("pindah transaksi invoice: %d %s %+v", code, body, res)
	}
	if got := txsOf(t, srv, qa); len(got) != 1 || got[0].Lawan != inv.Lawan {
		t.Errorf("transaksi invoice berubah: %+v", got)
	}
}

// Bulan transaksi manual sudah ditutup: pindah ditolak per baris, baris lain tetap dibukukan.
func TestMutasi_PindahBulanTutupDitolak(t *testing.T) {
	srv, qa := usahaManufaktur(t)
	manual := catatLama(t, srv, qa, `{"tanggal":"2026-01-30","jenis":"Biaya Lain","nominal":500000,"lunas":true}`)
	if rec := do(srv, qa, http.MethodPost, "/api/v1/tutup-buku", `{"periode":"2026-01"}`); rec.Code != http.StatusOK {
		t.Fatalf("tutup: %d %s", rec.Code, rec.Body)
	}
	h := unggahMutasi(t, srv, qa, "Tanggal;Keterangan;Debet;Kredit\n01/02/2026;BIAYA LISTRIK PLN;500.000;\n02/02/2026;BIAYA ADM;10.000;\n")
	if len(h.Rows) != 2 || h.Rows[0].Mirip == nil || h.Rows[0].Mirip.ID != manual.ID {
		t.Fatalf("baris harus cocok dengan catatan Januari: %+v", h.Rows)
	}
	r := h.Rows[0]
	r.Pilih, r.Pindah = true, manual.ID
	code, res, body := commitMutasiRekening(t, srv, qa, "1111", r, h.Rows[1])
	if code != http.StatusOK || res.Disimpan != 1 || res.Dipindah != 0 || len(res.Errors) != 1 || res.Errors[0].Baris != 2 || !strings.Contains(res.Errors[0].Pesan, "sudah ditutup") {
		t.Fatalf("commit: %d %s", code, body)
	}
	for _, tx := range txsOf(t, srv, qa) {
		if tx.ID == manual.ID && tx.Lawan != "" {
			t.Errorf("catatan di bulan tutup ikut dipindah: %+v", tx)
		}
	}
}

// Transaksi baru dari mutasi memakai rekening yang dipilih; tanpa pilihan = rekening bawaan.
func TestMutasi_TransaksiBaruPakaiRekeningTerpilih(t *testing.T) {
	srv, qa := usahaManufaktur(t)
	adm := finance.MutasiRow{Baris: 2, Tanggal: "2026-01-12", Jenis: "Biaya Lain", Nominal: 10000, Arah: "keluar", Pilih: true}
	if code, res, body := commitMutasiRekening(t, srv, qa, "1112", adm); code != http.StatusOK || res.Disimpan != 1 || res.Contoh[0].Lawan != "1112" {
		t.Fatalf("rekening terpilih: %d %s", code, body)
	}
	adm.Baris = 3
	if code, res, body := commitMutasiRekening(t, srv, qa, "", adm); code != http.StatusOK || res.Disimpan != 1 || res.Contoh[0].Lawan != "1111" {
		t.Fatalf("rekening bawaan: %d %s", code, body)
	}
	if s := saldoPerAkun(t, srv, qa, "2026-01"); s["1112"] != -10000 || s["1111"] != -10000 {
		t.Errorf("saldo payroll %d operasional %d", s["1112"], s["1111"])
	}
}
