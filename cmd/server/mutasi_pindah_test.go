package main

import (
	"encoding/json"
	"net/http"
	"strconv"
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

// Catat cepat dibayar dari Kas Besar: akun bayarnya boleh dipindah; catatan yang dibuka ulang
// menunjukkan baris rekening 1111.
func TestMutasi_PindahCatatCepatKas(t *testing.T) {
	srv, qa := usahaManufaktur(t)
	rec := do(srv, qa, http.MethodPost, "/api/v1/catat", `{"tanggal":"2026-01-10","keterangan":"Listrik","baris":[{"akun":"6190","debit":500000},{"akun":"1102","kredit":500000}]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("catat: %d %s", rec.Code, rec.Body)
	}
	var c catatResp
	_ = json.Unmarshal(rec.Body.Bytes(), &c)
	sebelum := saldoPerAkun(t, srv, qa, "2026-01")
	h := unggahMutasi(t, srv, qa, csvListrik)
	m := h.Rows[0].Mirip
	if m == nil || m.Grup != c.Grup || m.Lawan != "1102" || m.Produksi || m.Khusus {
		t.Fatalf("baris harus cocok dengan catatan (lawan 1102): %+v", m)
	}
	r := h.Rows[0]
	r.Pilih, r.Pindah = true, m.ID
	if code, res, body := commitMutasiRekening(t, srv, qa, "1111", r); code != http.StatusOK || res.Dipindah != 1 || len(res.Errors) != 0 {
		t.Fatalf("commit: %d %s", code, body)
	}
	if got := txsOf(t, srv, qa); len(got) != 1 || got[0].Lawan != "1111" || got[0].Grup != c.Grup {
		t.Fatalf("transaksi sesudah pindah: %+v", got)
	}
	g := getJSON[catatResp](t, srv, qa, "/api/v1/catat/"+strconv.Itoa(c.Grup))
	ada := false
	for _, b := range g.Baris {
		ada = ada || (b.Akun == "1111" && b.Kredit == 500000)
		if b.Akun == "1102" {
			t.Errorf("baris Kas Besar masih ada: %+v", g.Baris)
		}
	}
	if !ada {
		t.Errorf("catatan tanpa baris 1111: %+v", g.Baris)
	}
	sesudah := saldoPerAkun(t, srv, qa, "2026-01")
	if sesudah["1102"]-sebelum["1102"] != 500000 || sesudah["1111"]-sebelum["1111"] != -500000 {
		t.Errorf("saldo kas %d→%d, bank %d→%d", sebelum["1102"], sesudah["1102"], sebelum["1111"], sesudah["1111"])
	}
}

// Biaya lain produksi diubah lewat riwayat produksi, bukan dari impor mutasi.
func TestMutasi_PindahBiayaProduksiDitolak(t *testing.T) {
	srv, qa, kain, pKaos := siapProduksiKaos(t)
	body := `{"tanggal":"2026-01-05","bayar_dari":"1102","produk":[{"produk_id":` + itoa(pKaos.ID) + `,"hasil":10000,"bahan":[{"stok_id":` + itoa(kain.ID) + `,"qty":10000}],` +
		`"biaya":[{"keterangan":"Upah jahit","nominal":20000}]}]}`
	if rec := do(srv, qa, http.MethodPost, "/api/v1/stok/produksi/catat", body); rec.Code != http.StatusCreated {
		t.Fatalf("catat produksi: %d %s", rec.Code, rec.Body)
	}
	biaya := txBiayaProduksi(t, srv, qa)
	if len(biaya) != 1 || biaya[0].Lawan != "1102" {
		t.Fatalf("biaya produksi: %+v", biaya)
	}
	h := unggahMutasi(t, srv, qa, "Tanggal;Keterangan;Debet;Kredit\n06/01/2026;TRSF UPAH JAHIT;20.000;\n")
	m := h.Rows[0].Mirip
	if m == nil || m.ID != biaya[0].ID || !m.Produksi {
		t.Fatalf("baris harus cocok dengan biaya produksi: %+v", m)
	}
	r := h.Rows[0]
	r.Pilih, r.Pindah = true, m.ID
	code, _, resp := commitMutasiRekening(t, srv, qa, "1111", r)
	if code != http.StatusBadRequest || !strings.Contains(resp, errBiayaProduksi.Error()) {
		t.Fatalf("pindah biaya produksi: %d %s", code, resp)
	}
	if got := txBiayaProduksi(t, srv, qa); got[0].Lawan != "1102" {
		t.Errorf("biaya produksi ikut dipindah: %+v", got)
	}
}

// Channel marketplace / B2B: uangnya di akun channel, pindah ditolak (dan disembunyikan di UI).
func TestMutasi_PindahChannelKhususDitolak(t *testing.T) {
	srv, qa := usahaManufaktur(t)
	jual := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-01-10","jenis":"Pendapatan Lain","nominal":300000,"lunas":true,"channel":"GoFood"}`)
	h := unggahMutasi(t, srv, qa, "Tanggal;Keterangan;Debet;Kredit\n11/01/2026;TRSF CR GOFOOD;;300.000\n")
	m := h.Rows[0].Mirip
	if m == nil || m.ID != jual.ID || !m.Khusus {
		t.Fatalf("baris harus cocok dengan transaksi GoFood: %+v", m)
	}
	r := h.Rows[0]
	r.Pilih, r.Pindah = true, m.ID
	code, _, resp := commitMutasiRekening(t, srv, qa, "1111", r)
	if code != http.StatusBadRequest || !strings.Contains(resp, "lewat channel GoFood") {
		t.Fatalf("pindah channel khusus: %d %s", code, resp)
	}
}

// Pelunasan lewat mutasi memakai rekening mutasi; ID yang sama di dua baris hanya dipindah sekali.
func TestMutasi_LunasiPakaiRekeningTerpilih(t *testing.T) {
	srv, qa := usahaManufaktur(t)
	utang := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-01-10","jenis":"Biaya Lain","nominal":500000,"lunas":false}`)
	h := unggahMutasi(t, srv, qa, "Tanggal;Keterangan;Debet;Kredit\n15/01/2026;BAYAR TAGIHAN;500.000;\n")
	if m := h.Rows[0].Mirip; m == nil || m.ID != utang.ID || !m.BelumLunas {
		t.Fatalf("baris harus cocok dengan utang: %+v", m)
	}
	r := h.Rows[0]
	r.Pilih, r.Lunasi = true, utang.ID
	if code, res, body := commitMutasiRekening(t, srv, qa, "1112", r); code != http.StatusOK || res.Dilunasi != 1 {
		t.Fatalf("lunasi: %d %s", code, body)
	}
	if got := txsOf(t, srv, qa); len(got) != 1 || !got[0].Lunas || got[0].Lawan != "1112" {
		t.Fatalf("pelunasan: %+v", got)
	}
	if s := saldoPerAkun(t, srv, qa, "2026-01"); s["1112"] != -500000 || s["1111"] != 0 {
		t.Errorf("saldo payroll %d operasional %d", s["1112"], s["1111"])
	}

	manual := catatLama(t, srv, qa, `{"tanggal":"2026-01-20","jenis":"Biaya Lain","nominal":70000,"lunas":true}`)
	a := finance.MutasiRow{Baris: 2, Tanggal: "2026-01-21", Jenis: "Biaya Lain", Nominal: 70000, Arah: "keluar", Pilih: true, Pindah: manual.ID}
	b := a
	b.Baris = 3
	code, res, body := commitMutasiRekening(t, srv, qa, "1111", a, b)
	if code != http.StatusOK || res.Dipindah != 1 || len(res.Errors) != 1 || res.Errors[0].Baris != 3 {
		t.Fatalf("pindah dobel: %d %s", code, body)
	}
}
