package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Nomor jurnal tetap per transaksi (checklist tester No 31): JU-YYYYMM-NNNN menurut bulan
// tanggal transaksi saat pertama diberi nomor. Diberikan di satu tempat setelah setiap perubahan
// data (persisting) dan saat data dimuat, sehingga semua jalur pembuat transaksi (form, Catat,
// impor, mutasi, invoice, aset) tercakup. Satu catatan (grup) atau satu invoice berbagi nomor.
// Nomor tidak berubah saat diedit dan penghitung tidak pernah turun (nomor terhapus tidak dipakai ulang).

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
	var kosong []int
	for i, t := range s.txs {
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
		// penghitung minimal setinggi nomor yang sudah ada (data lama / penghitung hilang)
		if b, n, ok := urai(t.NoJurnal); ok && n > s.nomorJurnal[b] {
			s.nomorJurnal[b] = n
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
		case len(t.Tanggal) >= 7:
			b := t.Tanggal[:4] + t.Tanggal[5:7]
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
