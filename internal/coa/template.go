package coa

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"pajak/internal/finance"
)

// Template COA yang dikelola tim Master Pajak (per jenis usaha, berversi) harus tetap bisa dipakai
// jurnal otomatis: semua kode yang ditulis langsung di jurnal & laporan wajib ada.

var ErrTemplate = errors.New("template COA tidak valid")

// kodeSistem: kode yang dipakai langsung di jurnal & laporan (selain pemetaan jenis transaksi);
// TestKodeWajib_MencakupKodeDiKode menjaga daftar ini tetap lengkap.
var kodeSistem = []string{
	"1101", "1102", "1104", "1201", "1301", "1501", "1502", "1503", "1504", "1505", "1506", "1507", "1508",
	"2101", "2102", "3101", "3102", "3201", "4101", "5102", "5104", "6113", "6118", "8101", "9103",
}

// KodeWajib: kode akun yang harus ada di setiap template terbit, urut.
func KodeWajib() []string {
	out := slices.Clone(kodeSistem)
	for _, j := range finance.JenisList {
		for _, p := range []finance.BusinessProfile{
			{BentukUsaha: "op", MetodePersediaan: "periodik"}, {BentukUsaha: "pt", MetodePersediaan: "perpetual"},
		} {
			out = append(out, AutoAccount(j.Nama, p))
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

var kondisiValid = []Kondisi{Selalu, JikaAda, JikaPT, JikaNonPT, JikaPerpetual, JikaPeriodik}

// ValidateTemplate: kode 4 digit unik, isian lengkap & dikenal, dan semua KodeWajib ada.
func ValidateTemplate(accs []Account) error {
	ada := map[string]bool{}
	for i, a := range accs {
		baris := fmt.Sprintf("baris %d (%s)", i+1, a.Kode)
		switch {
		case len(a.Kode) != 4 || strings.Trim(a.Kode, "0123456789") != "":
			return fmt.Errorf("%w: %s: kode harus 4 angka", ErrTemplate, baris)
		case ada[a.Kode]:
			return fmt.Errorf("%w: kode %s dobel", ErrTemplate, a.Kode)
		case strings.TrimSpace(a.Nama) == "" || strings.TrimSpace(a.Subtype) == "":
			return fmt.Errorf("%w: %s: nama dan subtype wajib", ErrTemplate, baris)
		case !slices.Contains(Kelompoks, a.Kelompok):
			return fmt.Errorf("%w: %s: kelompok %q tidak dikenal", ErrTemplate, baris, a.Kelompok)
		case a.Normal != Debit && a.Normal != Kredit:
			return fmt.Errorf("%w: %s: saldo normal harus debit atau kredit", ErrTemplate, baris)
		case !slices.Contains(kondisiValid, a.Kondisi):
			return fmt.Errorf("%w: %s: kondisi %q tidak dikenal", ErrTemplate, baris, a.Kondisi)
		}
		ada[a.Kode] = true
	}
	var hilang []string
	for _, k := range KodeWajib() {
		if !ada[k] {
			hilang = append(hilang, k)
		}
	}
	if len(hilang) > 0 {
		return fmt.Errorf("%w: akun wajib untuk jurnal otomatis tidak ada: %s", ErrTemplate, strings.Join(hilang, ", "))
	}
	return nil
}
