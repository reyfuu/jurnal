package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"pajak/internal/finance"
)

// Nomor jurnal tetap per transaksi (checklist tester No 31): JU-YYYYMM-NNNN menurut bulan
// tanggal transaksi. Diberikan di satu tempat setelah setiap perubahan data (persisting) dan saat
// data dimuat, sehingga semua jalur pembuat transaksi (form, Catat, impor, mutasi, invoice, aset)
// tercakup. Satu catatan (grup) atau satu invoice berbagi nomor. Nomor tidak berubah saat diedit
// di bulan yang sama; pindah ke bulan lain → nomor baru di bulan itu. Penghitung tidak pernah
// turun, jadi nomor terhapus atau ditinggalkan tidak dipakai ulang. Invoice menyimpan nomornya
// sehingga invoice yang kembali ke draft lalu dikirim lagi memakai nomor yang sama.

func (s *tenant) beriNomorJurnal() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.beriNomorJurnalLocked()
}

// beriNomorJurnalLocked: s.mu dipegang.
func (s *tenant) beriNomorJurnalLocked() {
	if s.nomorJurnal == nil {
		s.nomorJurnal = map[string]int{}
	}
	perGrup, perInvoice := map[int]string{}, map[int]string{}
	for _, inv := range s.invoices {
		if b, n, ok := urai(inv.NoJurnal); ok {
			s.nomorJurnal[b] = max(s.nomorJurnal[b], n)
			if bulanTx(inv.Tanggal) == b {
				perInvoice[inv.ID] = inv.NoJurnal
			}
		}
	}
	var kosong []int
	for i := range s.txs {
		t := &s.txs[i]
		b, n, ok := urai(t.NoJurnal)
		if ok {
			// penghitung minimal setinggi nomor yang sudah ada (data lama / penghitung hilang)
			s.nomorJurnal[b] = max(s.nomorJurnal[b], n)
			if b != bulanTx(t.Tanggal) {
				t.NoJurnal = "" // dipindah ke bulan lain: nomor baru di bulan itu
			}
		}
		if t.NoJurnal == "" {
			kosong = append(kosong, i)
			continue
		}
		if t.Grup > 0 {
			perGrup[t.Grup] = t.NoJurnal
		}
		if t.InvoiceID > 0 {
			perInvoice[t.InvoiceID] = t.NoJurnal
		}
	}
	slices.SortFunc(kosong, func(a, b int) int { return s.txs[a].ID - s.txs[b].ID })
	for _, i := range kosong {
		t := &s.txs[i]
		switch {
		case t.Grup > 0 && perGrup[t.Grup] != "":
			t.NoJurnal = perGrup[t.Grup]
		case t.InvoiceID > 0 && perInvoice[t.InvoiceID] != "":
			t.NoJurnal = perInvoice[t.InvoiceID]
		case bulanTx(t.Tanggal) != "":
			b := bulanTx(t.Tanggal)
			s.nomorJurnal[b]++
			t.NoJurnal = fmt.Sprintf("JU-%s-%04d", b, s.nomorJurnal[b])
		}
		if t.Grup > 0 {
			perGrup[t.Grup] = t.NoJurnal
		}
		if t.InvoiceID > 0 {
			perInvoice[t.InvoiceID] = t.NoJurnal
		}
	}
	for i := range s.invoices {
		if no := perInvoice[s.invoices[i].ID]; no != "" {
			s.invoices[i].NoJurnal = no
		}
	}
}

// nomorTxLocked memberi nomor yang belum ada lalu mengembalikan nomor transaksi id, supaya
// respons simpan langsung memuat nomornya. s.mu dipegang.
func (s *tenant) nomorTxLocked(id int) string {
	s.beriNomorJurnalLocked()
	if i := slices.IndexFunc(s.txs, func(t finance.Transaction) bool { return t.ID == id }); i >= 0 {
		return s.txs[i].NoJurnal
	}
	return ""
}

// bulanTx: "2026-10-05" → "202610"; "" bila tanggal tidak lengkap.
func bulanTx(tanggal string) string {
	if len(tanggal) < 7 {
		return ""
	}
	return tanggal[:4] + tanggal[5:7]
}

// urai: "JU-202610-0012" → ("202610", 12).
func urai(no string) (string, int, bool) {
	b, n, ok := strings.Cut(strings.TrimPrefix(no, "JU-"), "-")
	if !ok || len(b) != 6 {
		return "", 0, false
	}
	k, err := strconv.Atoi(n)
	return b, k, err == nil
}
