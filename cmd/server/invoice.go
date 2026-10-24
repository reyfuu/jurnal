package main

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"pajak/internal/finance"
)

func (s *tenant) handleListInvoice(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	list := slices.Clone(s.invoices)
	s.mu.RUnlock()
	sum := finance.SummarizeInvoices(list, time.Now())
	writeJSON(w, map[string]any{
		"summary": sum,
		"status":  finance.StatusInvoice,
		// nomor_berikutnya & ppn_bps dipakai form untuk mengisi faktur baru tanpa menebak.
		"nomor_berikutnya": finance.NextInvoiceNumber(list, time.Now().Format(time.DateOnly)),
		"ppn_bps":          s.currentProfile().TarifPPNBps(),
		"profil_lengkap":   profilFakturLengkap(s.currentProfile()),
	})
}

// profilFakturLengkap menandai data penerbit yang masih kosong, supaya UI bisa mengingatkan
// sebelum faktur terlanjur dikirim ke pelanggan tanpa alamat atau rekening.
func profilFakturLengkap(p finance.BusinessProfile) []string {
	var kurang []string
	for _, c := range []struct {
		ada  bool
		nama string
	}{
		{p.Alamat != "", "alamat usaha"},
		{p.NPWP != "", "NPWP"},
		{p.BankNama != "" || p.BankRekening != "", "rekening pembayaran"},
		{p.PenandatanganNama != "", "nama penandatangan"},
	} {
		if !c.ada {
			kurang = append(kurang, c.nama)
		}
	}
	return kurang
}

func (s *tenant) decodeInvoice(w http.ResponseWriter, r *http.Request) (finance.Invoice, bool) {
	var inv finance.Invoice
	if err := json.NewDecoder(r.Body).Decode(&inv); err != nil {
		http.Error(w, "data faktur tidak valid", http.StatusBadRequest)
		return inv, false
	}
	if err := finance.ValidateInvoice(&inv); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return inv, false
	}
	return inv, true
}

// galatBarangInvoice: baris invoice melanggar aturan barang (HTTP 400, bukan 409).
type galatBarangInvoice struct{ error }

// cekBarangInvoiceLocked: barang tiap baris harus ada di daftar barang; di perpetual, invoice yang
// dikirim/lunas bertanggal sejak kartu stok dimulai (atau kartu belum dimulai) wajib memilih
// barang atau menandai Bukan barang (jasa) di tiap baris. Dipanggil saat invoice dibuat atau
// barisnya / tanggal / statusnya dari draft berubah; invoice lama yang tidak diubah tidak dicek.
func (s *tenant) cekBarangInvoiceLocked(inv finance.Invoice) error {
	for i, it := range inv.Items {
		if _, ok := s.stokByID(it.StokID); it.StokID != 0 && !ok {
			return galatBarangInvoice{fmt.Errorf("baris %d: barang yang dipilih tidak ada di Stok & Gudang", i+1)}
		}
	}
	if !s.perpetualLocked() || inv.Status == finance.StatusDraft {
		return nil
	}
	if mulai := s.mulaiKartuStokLocked(); mulai != "" && inv.Tanggal < mulai {
		return nil
	}
	for i, it := range inv.Items {
		if it.StokID == 0 && !it.BukanBarang {
			return galatBarangInvoice{fmt.Errorf("baris %d: pilih barang atau tandai Bukan barang (jasa)", i+1)}
		}
	}
	return nil
}

// kodeGalatInvoice: 400 untuk galat baris barang, selain itu 409.
func kodeGalatInvoice(err error) int {
	if errors.As(err, new(galatBarangInvoice)) {
		return http.StatusBadRequest
	}
	return http.StatusConflict
}

func (s *tenant) handleCreateInvoice(w http.ResponseWriter, r *http.Request) {
	inv, ok := s.decodeInvoice(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	err := s.tambahInvoiceLocked(&inv)
	s.mu.Unlock()
	if err != nil {
		http.Error(w, err.Error(), kodeGalatInvoice(err))
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, inv)
}

// tambahInvoiceLocked menerbitkan invoice baru yang sudah divalidasi: nomor otomatis bila kosong,
// nomor ganda ditolak, dan langsung dibukukan (lihat invoice_buku.go). Gagal = tidak ada yang
// berubah. Dipanggil dengan s.mu terkunci.
func (s *tenant) tambahInvoiceLocked(inv *finance.Invoice) error {
	if inv.Nomor == "" {
		inv.Nomor = finance.NextInvoiceNumber(s.invoices, inv.Tanggal)
	}
	if slices.ContainsFunc(s.invoices, func(x finance.Invoice) bool { return x.Nomor == inv.Nomor }) {
		return errors.New("nomor faktur " + inv.Nomor + " sudah dipakai")
	}
	if err := s.cekBarangInvoiceLocked(*inv); err != nil {
		return err
	}
	inv.ID = s.nextInvoiceID
	inv.Dibukukan = true
	inv.TransaksiID, inv.TransaksiPPNID, inv.NoJurnal = 0, 0, ""
	txs, nextTx := s.txs, s.nextTxID
	if err := s.sinkronInvoiceLocked(inv); err != nil {
		s.txs, s.nextTxID = txs, nextTx // transaksi DPP yang sempat dibuat ikut dibatalkan
		return err
	}
	s.nextInvoiceID++
	s.invoices = append(s.invoices, *inv)
	s.beriNomorJurnalLocked()
	inv.NoJurnal = s.invoices[len(s.invoices)-1].NoJurnal
	return nil
}

func (s *Server) handleInvoiceTemplate(w http.ResponseWriter, r *http.Request) {
	writeTemplate(w, "template-invoice.xlsx", finance.InvoiceTemplate())
}

// handleImportInvoice: impor invoice dari Excel. Pratinjau menjalankan langkah yang sama dengan
// commit lalu mengembalikan semuanya, jadi nomor ganda & bulan tutup buku sudah terlihat sebelum
// menyimpan. Invoice yang gagal dilewati; sisanya tetap masuk.
func (s *tenant) handleImportInvoice(w http.ResponseWriter, r *http.Request) {
	sheets, ok := uploadedSheets(w, r)
	if !ok {
		return
	}
	list, errs, nama := finance.ParseInvoiceFile(sheets, s.currentProfile().TarifPPNBps())
	commit := r.URL.Query().Get("commit") == "1"
	resp := menuImportResp{Sheet: nama, Errors: errs, Invoice: []finance.Invoice{}}
	s.mu.Lock()
	txs, nextTx, invs, nextInv := s.txs, s.nextTxID, s.invoices, s.nextInvoiceID
	gerak, nextGerak, prof := s.gerak, s.nextGerakID, s.profile // kartu stok yang dimulai otomatis
	for _, x := range list {
		inv := x.Invoice
		err := s.barangImporInvoiceLocked(&inv, x.Barang)
		if err == nil {
			err = s.tambahInvoiceLocked(&inv)
		}
		if err != nil {
			resp.Errors = append(resp.Errors, finance.SheetError{Sheet: nama, Baris: x.Baris, Pesan: "invoice " + cmp.Or(inv.Nomor, inv.PelangganNama) + ": " + err.Error()})
			continue
		}
		resp.Invoice = append(resp.Invoice, inv)
	}
	if !commit {
		s.txs, s.nextTxID, s.invoices, s.nextInvoiceID = txs, nextTx, invs, nextInv
		s.gerak, s.nextGerakID = gerak, nextGerak
		if s.profile != prof && prof != nil { // metode nilai stok sempat dikunci: pulihkan juga berkasnya
			s.profile = prof
			if err := saveProfile(s.file("business"), *prof); err != nil {
				log.Printf("pulihkan profil setelah pratinjau impor invoice: %v", err)
			}
		}
	}
	s.mu.Unlock()
	resp.Valid = len(resp.Invoice)
	if commit && resp.Valid == 0 {
		http.Error(w, "tidak ada invoice yang bisa diimpor; periksa pesan di pratinjau", http.StatusBadRequest)
		return
	}
	resp.Imported = commit
	writeJSON(w, resp)
}

// barangImporInvoiceLocked: cocokkan teks kolom Barang tiap item (sejajar inv.Items) ke daftar
// barang, aturan sama dengan impor transaksi (cariBarangImporLocked). Periodik: barang opsional,
// jadi yang tak cocok dibiarkan kosong. s.mu dipegang.
func (s *tenant) barangImporInvoiceLocked(inv *finance.Invoice, barang []string) error {
	for k, teks := range barang {
		if teks == "" || k >= len(inv.Items) {
			continue
		}
		id, err := s.cariBarangImporLocked(teks)
		if err != nil {
			if !s.perpetualLocked() {
				continue
			}
			return fmt.Errorf("baris %d: %w", k+1, err)
		}
		inv.Items[k].StokID = id
	}
	return nil
}

// invoiceIndex harus dipanggil dengan s.mu terkunci.
func (s *tenant) invoiceIndex(r *http.Request) int {
	id, _ := strconv.Atoi(r.PathValue("id"))
	return slices.IndexFunc(s.invoices, func(x finance.Invoice) bool { return x.ID == id })
}

func (s *tenant) handleUpdateInvoice(w http.ResponseWriter, r *http.Request) {
	inv, ok := s.decodeInvoice(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.invoiceIndex(r)
	if i < 0 {
		http.Error(w, "faktur tidak ditemukan", http.StatusNotFound)
		return
	}
	// Nomor faktur tidak boleh bentrok dengan faktur lain.
	bentrok := slices.ContainsFunc(s.invoices, func(x finance.Invoice) bool {
		return x.Nomor == inv.Nomor && x.ID != s.invoices[i].ID
	})
	if bentrok {
		http.Error(w, "nomor faktur "+inv.Nomor+" sudah dipakai", http.StatusConflict)
		return
	}
	lama := s.invoices[i]
	inv.ID = lama.ID
	if inv.Nomor == "" {
		inv.Nomor = lama.Nomor
	}
	// Tautan pembukuan diatur server; "bukukan" dari klien hanya bisa menyalakan, bukan mematikan.
	// Draft belum pernah terbit, jadi belum mungkin dicatat manual: begitu dikirim langsung dibukukan.
	inv.Dibukukan = lama.Dibukukan || inv.Dibukukan || lama.Status == finance.StatusDraft
	inv.TransaksiID, inv.TransaksiPPNID, inv.NoJurnal = lama.TransaksiID, lama.TransaksiPPNID, lama.NoJurnal
	// Invoice lama yang hanya dilunasi / dikirim ulang tanpa mengubah baris & tanggal tidak dicek.
	berubah := lama.Status == finance.StatusDraft || lama.Tanggal != inv.Tanggal || !slices.Equal(lama.Items, inv.Items)
	if berubah && inv.Dibukukan {
		if err := s.cekBarangInvoiceLocked(inv); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	txs, nextTx := slices.Clone(s.txs), s.nextTxID
	if err := s.sinkronInvoiceLocked(&inv); err != nil {
		s.txs, s.nextTxID = txs, nextTx // transaksi DPP yang sempat diubah ikut dipulihkan
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	s.invoices[i] = inv
	s.beriNomorJurnalLocked()
	writeJSON(w, s.invoices[i])
}

func (s *tenant) handleDeleteInvoice(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.invoiceIndex(r)
	if i < 0 {
		http.Error(w, "faktur tidak ditemukan", http.StatusNotFound)
		return
	}
	// Transaksi pembukuannya ikut terhapus (ditolak bila bulannya sudah tutup buku).
	hapus := s.invoices[i]
	hapus.Status = finance.StatusDraft
	if err := s.sinkronInvoiceLocked(&hapus); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	s.invoices = slices.Delete(s.invoices, i, i+1)
	w.WriteHeader(http.StatusNoContent)
}

// handleInvoicePDF mencetak satu faktur. Nama berkasnya memakai nomor faktur supaya berkas
// yang diunduh langsung terbaca tanpa dibuka.
func (s *tenant) handleInvoicePDF(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	i := s.invoiceIndex(r)
	var inv finance.Invoice
	if i >= 0 {
		inv = s.invoices[i]
	}
	s.mu.RUnlock()
	if i < 0 {
		http.Error(w, "faktur tidak ditemukan", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, namaBerkasPDF(inv)))
	if err := finance.RenderInvoicePDF(w, inv, s.currentProfile()); err != nil {
		// Header sudah terkirim, jadi tidak bisa lagi mengubah status; cukup catat di log.
		log.Printf("render faktur %s gagal: %v", inv.Nomor, err)
	}
}

// namaBerkasPDF: "INV-2026-061.pdf", dibersihkan dari karakter yang bermasalah di nama berkas.
func namaBerkasPDF(inv finance.Invoice) string {
	nama := inv.Nomor
	if nama == "" {
		nama = "invoice"
	}
	nama = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`/\:*?"<>|`, r) {
			return '-'
		}
		return r
	}, nama)
	return nama + ".pdf"
}
