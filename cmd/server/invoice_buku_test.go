package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"pajak/internal/finance"
)

func txByID(srv *Server, id int) (finance.Transaction, bool) {
	t := srv.tenants[1]
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, x := range t.txs {
		if x.ID == id {
			return x, true
		}
	}
	return finance.Transaction{}, false
}

// Invoice terkirim langsung menjadi omzet (DPP), utang PPN, dan SATU baris piutang sebesar total
// invoice dengan jatuh temponya; pelunasan dari Piutang & Utang melunasi invoice & kedua
// transaksinya; transaksi invoice tidak bisa diubah dari menu Transaksi; hapus invoice = hapus
// transaksinya.
func TestInvoice_TersambungKePembukuan(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	pkp := strings.Replace(strings.Replace(profileBody, "manufaktur", "perdagangan", 1), `"pkp":false`, `"pkp":true`, 1)
	if rec := do(srv, owner, http.MethodPut, "/api/v1/business", pkp); rec.Code != http.StatusOK {
		t.Fatalf("profil PKP: %d %s", rec.Code, rec.Body)
	}
	body := func(status, lunas string) string {
		return `{"nomor":"INV-UJI-001","status":"` + status + `","tanggal":"2026-08-10","jatuh_tempo":"2026-09-09","tanggal_lunas":"` + lunas + `",` +
			`"pelanggan_nama":"PT Uji Pembeli","items":[{"deskripsi":"Kopi","qty":10,"harga_satuan":1000000}],"diskon_rp":0,"ppn_bps":1100}`
	}
	rec := do(srv, owner, http.MethodPost, "/api/v1/invoice", body("draft", ""))
	var inv finance.Invoice
	_ = json.NewDecoder(rec.Body).Decode(&inv)
	if rec.Code != http.StatusCreated || !inv.Dibukukan || inv.TransaksiID != 0 {
		t.Fatalf("draft: %d %+v", rec.Code, inv)
	}
	url := "/api/v1/invoice/" + strconv.Itoa(inv.ID)
	rec = do(srv, owner, http.MethodPut, url, body("terkirim", ""))
	_ = json.NewDecoder(rec.Body).Decode(&inv)
	jual, ok1 := txByID(srv, inv.TransaksiID)
	ppn, ok2 := txByID(srv, inv.TransaksiPPNID)
	if rec.Code != http.StatusOK || !ok1 || !ok2 || jual.Nominal != 10_000_000 || ppn.Nominal != 1_100_000 || jual.Lunas || jual.Jenis != "Penjualan" || ppn.Jenis != "PPN Keluaran" {
		t.Fatalf("terkirim: %d %+v | %+v | %+v", rec.Code, inv, jual, ppn)
	}
	umur := getJSON[finance.UmurSaldoResult](t, srv, owner, "/api/v1/reports/umur?per=2026-08-31")
	var baris *finance.BarisUmur
	for i := range umur.Baris {
		if umur.Baris[i].InvoiceID == inv.ID {
			if baris != nil {
				t.Fatal("invoice muncul lebih dari satu baris piutang")
			}
			baris = &umur.Baris[i]
		}
	}
	if baris == nil || baris.Nilai != 11_100_000 || baris.JatuhTempo != "2026-09-09" {
		t.Fatalf("baris piutang invoice: %+v", baris)
	}
	if bs := getJSON[finance.BalanceSheet](t, srv, owner, "/api/v1/reports/balance-sheet?period=2026-08"); !bs.IsBalanced || bs.Liabilities.Other < 1_100_000 {
		t.Fatalf("neraca: seimbang=%v kewajiban lain %d", bs.IsBalanced, bs.Liabilities.Other)
	}
	// Transaksi invoice tidak bisa diubah / dihapus dari menu Transaksi.
	if rec := do(srv, owner, http.MethodDelete, "/api/v1/transactions/"+strconv.Itoa(jual.ID), ""); rec.Code != http.StatusConflict {
		t.Fatalf("hapus transaksi invoice: %d", rec.Code)
	}
	// Dilunasi dari Piutang & Utang → invoice lunas, DPP & PPN lunas.
	if rec := do(srv, owner, http.MethodPost, "/api/v1/transactions/"+strconv.Itoa(baris.TransaksiID)+"/lunas", `{"tanggal":"2026-09-05"}`); rec.Code != http.StatusOK {
		t.Fatalf("lunasi: %d %s", rec.Code, rec.Body)
	}
	jual, _ = txByID(srv, inv.TransaksiID)
	ppn, _ = txByID(srv, inv.TransaksiPPNID)
	var lunas finance.Invoice
	tn := srv.tenants[1]
	tn.mu.RLock()
	for _, x := range tn.invoices {
		if x.ID == inv.ID {
			lunas = x
		}
	}
	tn.mu.RUnlock()
	if !jual.Lunas || !ppn.Lunas || jual.TanggalLunas != "2026-09-05" || lunas.Status != finance.StatusLunas || lunas.TanggalLunas != "2026-09-05" {
		t.Fatalf("pelunasan: %+v | %+v | %+v", jual, ppn, lunas)
	}
	// Hapus invoice → transaksinya ikut hilang.
	if rec := do(srv, owner, http.MethodDelete, url, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("hapus invoice: %d %s", rec.Code, rec.Body)
	}
	if _, ada := txByID(srv, jual.ID); ada {
		t.Fatal("transaksi penjualan invoice yang dihapus masih ada")
	}
}

// Invoice lama (sebelum fitur ini) tidak dibukukan sampai pengguna memilih Bukukan.
func TestInvoice_LamaDibukukanSetelahDiminta(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	tn := srv.tenants[1]
	tn.mu.Lock()
	tn.invoices = append(tn.invoices, finance.Invoice{ID: 900, Nomor: "INV-LAMA", Status: finance.StatusTerkirim, Tanggal: "2026-08-01", JatuhTempo: "2026-08-31",
		PelangganNama: "CV Lama", Items: []finance.InvoiceItem{{Deskripsi: "X", Qty: 1, HargaSatuan: 500_000}}})
	tn.mu.Unlock()
	body := `{"nomor":"INV-LAMA","status":"terkirim","tanggal":"2026-08-01","jatuh_tempo":"2026-08-31","pelanggan_nama":"CV Lama","items":[{"deskripsi":"X","qty":1,"harga_satuan":500000}]}`
	var inv finance.Invoice
	_ = json.NewDecoder(do(srv, owner, http.MethodPut, "/api/v1/invoice/900", body).Body).Decode(&inv)
	if inv.Dibukukan || inv.TransaksiID != 0 {
		t.Fatalf("invoice lama dibukukan diam-diam: %+v", inv)
	}
	_ = json.NewDecoder(do(srv, owner, http.MethodPut, "/api/v1/invoice/900", strings.Replace(body, `"status"`, `"dibukukan":true,"status"`, 1)).Body).Decode(&inv)
	if tx, ok := txByID(srv, inv.TransaksiID); !inv.Dibukukan || !ok || tx.Nominal != 500_000 {
		t.Fatalf("bukukan: %+v %+v", inv, tx)
	}
}

// "Bukukan sekarang": invoice lama terkirim/lunas langsung jadi transaksi (piutang bila belum
// lunas), draft hanya ditandai dan tercatat begitu dikirim.
func TestInvoice_BukukanSemuaInvoiceLama(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	tn := srv.tenants[1]
	item := []finance.InvoiceItem{{Deskripsi: "X", Qty: 1, HargaSatuan: 100_000}}
	tn.mu.Lock()
	tn.invoices = append(tn.invoices,
		finance.Invoice{ID: 901, Nomor: "INV-L1", Status: finance.StatusTerkirim, Tanggal: "2026-10-21", JatuhTempo: "2026-11-20", PelangganNama: "Bayu", Items: item},
		finance.Invoice{ID: 902, Nomor: "INV-L2", Status: finance.StatusDraft, Tanggal: "2026-10-21", JatuhTempo: "2026-11-20", PelangganNama: "Agung", Items: item})
	tn.mu.Unlock()
	var hasil struct {
		Dibukukan int
		Gagal     []string
	}
	if err := json.NewDecoder(do(srv, owner, http.MethodPost, "/api/v1/invoice/bukukan", "").Body).Decode(&hasil); err != nil || hasil.Dibukukan < 1 || len(hasil.Gagal) != 0 {
		t.Fatalf("bukukan: %+v %v", hasil, err)
	}
	tn.mu.RLock()
	l1, l2 := tn.invoices[len(tn.invoices)-2], tn.invoices[len(tn.invoices)-1]
	tn.mu.RUnlock()
	if tx, ok := txByID(srv, l1.TransaksiID); !l1.Dibukukan || !ok || tx.Lunas || tx.Pihak != "Bayu" {
		t.Fatalf("invoice terkirim harus jadi piutang: %+v %+v", l1, tx)
	}
	if !l2.Dibukukan || l2.TransaksiID != 0 {
		t.Fatalf("draft hanya ditandai: %+v", l2)
	}
	body := `{"nomor":"INV-L2","status":"terkirim","tanggal":"2026-10-21","jatuh_tempo":"2026-11-20","pelanggan_nama":"Agung","items":[{"deskripsi":"X","qty":1,"harga_satuan":100000}]}`
	var inv finance.Invoice
	_ = json.NewDecoder(do(srv, owner, http.MethodPut, "/api/v1/invoice/902", body).Body).Decode(&inv)
	if inv.TransaksiID == 0 {
		t.Fatalf("draft yang dikirim harus langsung tercatat: %+v", inv)
	}
}

func TestInvoice_DraftLamaDibukukanSaatDikirim(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	tn := srv.tenants[1]
	tn.mu.Lock()
	tn.invoices = append(tn.invoices, finance.Invoice{ID: 903, Nomor: "INV-D", Status: finance.StatusDraft, Tanggal: "2026-10-21", JatuhTempo: "2026-11-20",
		PelangganNama: "Agung", Items: []finance.InvoiceItem{{Deskripsi: "X", Qty: 1, HargaSatuan: 100_100}}})
	tn.mu.Unlock()
	body := `{"nomor":"INV-D","status":"terkirim","tanggal":"2026-10-21","jatuh_tempo":"2026-11-20","pelanggan_nama":"Agung","items":[{"deskripsi":"X","qty":1,"harga_satuan":100100}]}`
	var inv finance.Invoice
	_ = json.NewDecoder(do(srv, owner, http.MethodPut, "/api/v1/invoice/903", body).Body).Decode(&inv)
	if tx, ok := txByID(srv, inv.TransaksiID); !inv.Dibukukan || !ok || tx.Lunas || tx.Nominal != 100_100 {
		t.Fatalf("draft lama yang dikirim harus jadi piutang: %+v %+v", inv, tx)
	}
}

// Impor invoice dari Excel: pratinjau tidak mengubah apa pun; commit menerbitkan & membukukan
// invoice yang sah (nomor otomatis bila kosong), nomor ganda dan bulan tutup buku dilewati.
func TestImporInvoice_PratinjauLaluCommit(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	pkp := strings.Replace(strings.Replace(profileBody, "manufaktur", "perdagangan", 1), `"pkp":false`, `"pkp":true`, 1)
	do(srv, owner, http.MethodPut, "/api/v1/business", pkp)
	ten := srv.tenants[1]
	ten.tutupBuku = "2026-08"
	nInv, nTx := len(ten.invoices), len(ten.txs)
	lama := ten.invoices[0].Nomor
	rows := [][]string{{"Nomor", "Tanggal", "Jatuh Tempo", "Pelanggan", "Status", "Tanggal Lunas", "Deskripsi", "Qty", "Harga Satuan", "PPN (%)"},
		{"INV-IMP-1", "2026-09-01", "2026-09-30", "PT Impor Satu", "", "", "Kopi", "10", "100000", ""},
		{"INV-IMP-1", "", "", "", "", "", "Gula", "2", "50000", ""},
		{"", "2026-09-02", "2026-09-02", "CV Tunai", "Lunas", "2026-09-02", "Katering", "1", "500000", "0"},
		{lama, "2026-09-03", "2026-09-30", "PT Ganda", "", "", "Kopi", "1", "1000", ""},
		{"INV-IMP-4", "2026-08-15", "2026-09-15", "PT Agustus", "", "", "Kopi", "1", "1000", ""},
	}
	kirim := func(q string) menuImportResp {
		body, ctype := workbookUpload(t, []finance.ImportSheet{{Nama: "Invoice", Rows: rows}})
		rec := upload(srv, owner, "/api/v1/invoice/import"+q, body, ctype)
		var res menuImportResp
		if err := json.NewDecoder(rec.Body).Decode(&res); err != nil || rec.Code != http.StatusOK {
			t.Fatalf("impor%s: %d %s", q, rec.Code, rec.Body)
		}
		return res
	}
	pr := kirim("")
	if pr.Valid != 2 || len(pr.Errors) != 2 || pr.Imported || len(ten.invoices) != nInv || len(ten.txs) != nTx {
		t.Fatalf("pratinjau: valid %d errors %+v imported %v; invoice %d→%d, tx %d→%d", pr.Valid, pr.Errors, pr.Imported, nInv, len(ten.invoices), nTx, len(ten.txs))
	}
	res := kirim("?commit=1")
	if !res.Imported || res.Valid != 2 || len(ten.invoices) != nInv+2 {
		t.Fatalf("commit: %+v", res)
	}
	// INV-IMP-1: DPP 1.100.000 + PPN 11% = dua transaksi; CV Tunai tanpa PPN, lunas = satu.
	if got := len(ten.txs) - nTx; got != 3 {
		t.Errorf("transaksi baru: %d, mau 3", got)
	}
	baru := ten.invoices[len(ten.invoices)-1]
	if baru.PelangganNama != "CV Tunai" || !strings.HasPrefix(baru.Nomor, "INV-2026-") || baru.Status != finance.StatusLunas || !baru.Dibukukan {
		t.Errorf("invoice bernomor otomatis: %+v", baru)
	}
	for _, e := range res.Errors {
		if !strings.Contains(e.Pesan, lama) && !strings.Contains(e.Pesan, "INV-IMP-4") {
			t.Errorf("galat tak terduga: %+v", e)
		}
	}
}
