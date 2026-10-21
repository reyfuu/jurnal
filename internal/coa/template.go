package coa

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Template COA yang dikelola tim Master Pajak (per jenis usaha, berversi) harus tetap bisa dipakai
// jurnal otomatis: semua peran akun (peran.go) wajib dipegang akun di template.

var ErrTemplate = errors.New("template COA tidak valid")

// Jurnal otomatis tidak lagi memakai kode tetap: tiap peran (peran.go) harus dipegang satu akun
// di template, lewat kolom Peran atau — untuk template lama — kode bawaan CoA dagang.

// PeranWajib: peran yang harus terisi di setiap template terbit, beserta kode bawaannya.
func PeranWajib() []PeranInfo { return slices.Clone(DaftarPeran) }

var kondisiValid = []Kondisi{Selalu, JikaAda, JikaPT, JikaNonPT, JikaPerpetual, JikaPeriodik}

// ValidateTemplate: kode 4 digit unik, isian lengkap & dikenal, dan semua peran wajib terisi.
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
	peranDipakai := map[string]string{}
	for _, a := range accs {
		if a.Peran == "" {
			continue
		}
		if !slices.ContainsFunc(DaftarPeran, func(r PeranInfo) bool { return r.ID == a.Peran }) && a.Peran != "modal_pt" {
			return fmt.Errorf("%w: akun %s: peran %q tidak dikenal", ErrTemplate, a.Kode, a.Peran)
		}
		if lain, dobel := peranDipakai[a.Peran]; dobel {
			return fmt.Errorf("%w: peran %q dipegang dua akun (%s dan %s)", ErrTemplate, a.Peran, lain, a.Kode)
		}
		peranDipakai[a.Peran] = a.Kode
	}
	if kurang := PeranKurang(accs); len(kurang) > 0 {
		return fmt.Errorf("%w: peran untuk jurnal otomatis belum dipegang akun mana pun: %s", ErrTemplate, strings.Join(kurang, ", "))
	}
	return nil
}
