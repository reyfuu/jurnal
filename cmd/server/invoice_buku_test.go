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
