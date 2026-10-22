package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"pajak/internal/finance"
)

func bayarAPI(t *testing.T, srv *Server, c *http.Cookie, kunci, body string) (int, finance.Pembayaran) {
	t.Helper()
	rec := doKunci(srv, c, http.MethodPost, "/api/v1/pembayaran", body, kunci)
	var p finance.Pembayaran
	_ = json.Unmarshal(rec.Body.Bytes(), &p)
	return rec.Code, p
}

func txTenant(t *testing.T, srv *Server, c *http.Cookie, id int) finance.Transaction {
	t.Helper()
	tn := tenantByEmail(t, srv, c)
	tn.mu.RLock()
	defer tn.mu.RUnlock()
	for _, x := range tn.txs {
		if x.ID == id {
			return x
		}
	}
	t.Fatalf("transaksi #%d tidak ada", id)
	return finance.Transaction{}
}

// Pembayaran sebagian lewat API: idempoten per kunci, tidak boleh melebihi sisa, transaksi yang
// sudah dibayar tidak bisa diubah/dihapus langsung, Lunasi membayar sisanya, batal sekali dengan
// alasan dan hanya di bulan yang belum ditutup.
func TestPembayaran_API(t *testing.T) {
	srv, qa := usahaBaru(t)
	jual := buat[finance.Transaction](t, srv, qa, "/api/v1/transactions", `{"tanggal":"2026-01-05","jenis":"Penjualan","nominal":300000,"pihak":"Toko Maju","cicilan":[{"pembayaran_id":99,"tanggal":"2026-01-05","nominal":300000}]}`)
	if len(jual.Cicilan) != 0 {
		t.Fatalf("cicilan dari klien harus diabaikan: %+v", jual)
	}
	id := strconv.Itoa(jual.ID)
	body := `{"tanggal":"2026-01-20","jenis":"piutang","lawan":"1102","alokasi":[{"transaksi_id":` + id + `,"nominal":200000}]}`
	k1, p1 := bayarAPI(t, srv, qa, "bayar-1", body)
	k2, p2 := bayarAPI(t, srv, qa, "bayar-1", body)
	if k1 != http.StatusCreated || k2 != http.StatusOK || p1.ID == 0 || p2.ID != p1.ID || p1.Total != 200_000 || p1.Pihak != "Toko Maju" {
		t.Fatalf("idempoten: %d %+v / %d %+v", k1, p1, k2, p2)
	}
	if tx := txTenant(t, srv, qa, jual.ID); tx.Lunas || tx.Sisa() != 100_000 {
		t.Fatalf("setelah bayar sebagian: %+v", tx)
	}
	if k, _ := bayarAPI(t, srv, qa, "bayar-lebih", body); k != http.StatusBadRequest { // 200rb > sisa 100rb
		t.Fatalf("melebihi sisa: %d", k)
	}
	if k, _ := bayarAPI(t, srv, qa, "bayar-kas-salah", strings.Replace(body, `"1102"`, `"4101"`, 1)); k != http.StatusBadRequest {
		t.Fatalf("lawan bukan kas/bank: %d", k)
	}
	for _, c := range []struct{ m, b string }{
		{http.MethodPut, `{"tanggal":"2026-01-05","jenis":"Penjualan","nominal":400000}`},
		{http.MethodDelete, ``},
	} {
		if rec := do(srv, qa, c.m, "/api/v1/transactions/"+id, c.b); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "batalkan") {
			t.Errorf("%s transaksi bercicilan: %d %s", c.m, rec.Code, rec.Body)
		}
	}
	// Lunasi = bayar sisanya lewat dokumen Pembayaran.
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions/"+id+"/lunas", `{"tanggal":"2026-02-03"}`); rec.Code != http.StatusOK {
		t.Fatalf("lunasi sisa: %d %s", rec.Code, rec.Body)
	}
	if tx := txTenant(t, srv, qa, jual.ID); !tx.Lunas || len(tx.Cicilan) != 2 || tx.TanggalLunas != "2026-02-03" {
		t.Fatalf("setelah lunasi: %+v", tx)
	}
	daftar := getJSON[[]finance.Pembayaran](t, srv, qa, "/api/v1/pembayaran")
	if len(daftar) != 2 || daftar[0].Total != 100_000 {
		t.Fatalf("riwayat: %+v", daftar)
	}
	batal := "/api/v1/pembayaran/" + strconv.Itoa(daftar[0].ID) + "/batal"
	if rec := do(srv, qa, http.MethodPost, batal, `{"alasan":" "}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("batal tanpa alasan: %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodPost, batal, `{"alasan":"Salah tanggal"}`); rec.Code != http.StatusOK {
		t.Fatalf("batal: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, qa, http.MethodPost, batal, `{"alasan":"lagi"}`); rec.Code != http.StatusConflict {
		t.Fatalf("batal kedua: %d", rec.Code)
	}
	if tx := txTenant(t, srv, qa, jual.ID); tx.Lunas || tx.Sisa() != 100_000 {
		t.Fatalf("setelah batal: %+v", tx)
	}
	// Bulan pembayaran pertama ditutup: pembayarannya tidak bisa dibatalkan lagi.
	if rec := do(srv, qa, http.MethodPost, "/api/v1/tutup-buku", `{"periode":"2026-01"}`); rec.Code != http.StatusOK {
		t.Fatalf("tutup buku: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/pembayaran/"+strconv.Itoa(p1.ID)+"/batal", `{"alasan":"x"}`); rec.Code != http.StatusConflict {
		t.Fatalf("batal di bulan tutup buku: %d", rec.Code)
	}
	if k, _ := bayarAPI(t, srv, qa, "bayar-januari", strings.Replace(body, "200000", "1000", 1)); k != http.StatusConflict {
		t.Fatalf("bayar bertanggal bulan tutup buku: %d", k)
	}
}

// Pembayaran ke baris invoice (DPP + PPN) dibagi ke kedua transaksinya; status invoice mengikuti:
// sebagian = tetap terkirim, penuh = lunas, batal = kembali terkirim. Invoice yang sudah dibayar
// tidak bisa diubah nominalnya atau dihapus.
func TestPembayaran_Invoice(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	pkp := strings.Replace(strings.Replace(profileBody, "manufaktur", "perdagangan", 1), `"pkp":false`, `"pkp":true`, 1)
	if rec := do(srv, owner, http.MethodPut, "/api/v1/business", pkp); rec.Code != http.StatusOK {
		t.Fatalf("profil PKP: %d %s", rec.Code, rec.Body)
	}
	isi := func(harga string) string {
		return `{"nomor":"INV-BAYAR-1","status":"terkirim","tanggal":"2026-08-10","jatuh_tempo":"2026-09-09",` +
			`"pelanggan_nama":"PT Uji Pembeli","items":[{"deskripsi":"Kopi","qty":10,"harga_satuan":` + harga + `}],"diskon_rp":0,"ppn_bps":1100}`
	}
	rec := do(srv, owner, http.MethodPost, "/api/v1/invoice", isi("1000000"))
	var inv finance.Invoice
	_ = json.NewDecoder(rec.Body).Decode(&inv)
	if rec.Code != http.StatusCreated || inv.TransaksiID == 0 || inv.TransaksiPPNID == 0 {
		t.Fatalf("invoice: %d %+v", rec.Code, inv)
	}
	baris := `{"tanggal":"2026-08-20","jenis":"piutang","alokasi":[{"transaksi_id":` + strconv.Itoa(inv.TransaksiID) + `,"nominal":NOMINAL}]}`
	if k, _ := bayarAPI(t, srv, owner, "inv-1", strings.Replace(baris, "NOMINAL", "5000000", 1)); k != http.StatusCreated {
		t.Fatalf("bayar sebagian invoice: %d", k)
	}
	invoiceDari := func() finance.Invoice {
		var out finance.Invoice
		tn := srv.tenants[1]
		tn.mu.RLock()
		defer tn.mu.RUnlock()
		for _, x := range tn.invoices {
			if x.ID == inv.ID {
				out = x
			}
		}
		return out
	}
	if s := invoiceDari().Status; s != finance.StatusTerkirim {
		t.Fatalf("invoice dibayar sebagian: status %s", s)
	}
	umur := getJSON[finance.UmurSaldoResult](t, srv, owner, "/api/v1/reports/umur?per=2026-08-31")
	for _, b := range umur.Baris {
		if b.InvoiceID == inv.ID && (b.Nilai != 6_100_000 || b.Dibayar != 5_000_000) {
			t.Fatalf("baris invoice: %+v", b)
		}
	}
	url := "/api/v1/invoice/" + strconv.Itoa(inv.ID)
	if rec := do(srv, owner, http.MethodPut, url, isi("2000000")); rec.Code != http.StatusConflict {
		t.Fatalf("ubah nominal invoice yang sudah dibayar: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, owner, http.MethodDelete, url, ""); rec.Code != http.StatusConflict {
		t.Fatalf("hapus invoice yang sudah dibayar: %d", rec.Code)
	}
	k, p2 := bayarAPI(t, srv, owner, "inv-2", strings.Replace(baris, "NOMINAL", "6100000", 1))
	if k != http.StatusCreated || len(p2.Alokasi) != 2 {
		t.Fatalf("pelunasan invoice dibagi DPP & PPN: %d %+v", k, p2)
	}
	if got := invoiceDari(); got.Status != finance.StatusLunas || got.TanggalLunas != "2026-08-20" {
		t.Fatalf("invoice lunas: %+v", got)
	}
	if rec := do(srv, owner, http.MethodPost, "/api/v1/pembayaran/"+strconv.Itoa(p2.ID)+"/batal", `{"alasan":"Transfer ditolak bank"}`); rec.Code != http.StatusOK {
		t.Fatalf("batal: %d %s", rec.Code, rec.Body)
	}
	if s := invoiceDari().Status; s != finance.StatusTerkirim {
		t.Fatalf("setelah batal: status %s", s)
	}
}
