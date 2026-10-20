package katalog

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"pajak/internal/tax"
)

// ParameterVersi: satu versi angka aturan pajak. Hanya draft yang bisa diubah; versi aktif
// dipakai seluruh kalkulator. Versi arsip bisa diaktifkan lagi (rollback).
type ParameterVersi struct {
	ID     int           `json:"id"`
	Nama   string        `json:"nama"`
	Dasar  string        `json:"dasar"` // aturan yang mendasari perubahan, misalnya "PMK 81/2024"
	Status string        `json:"status"`
	Isi    tax.Parameter `json:"isi"`
	Oleh   string        `json:"oleh,omitempty"`
	Waktu  time.Time     `json:"waktu"`
}

func (s *State) ParameterByID(id int) (ParameterVersi, bool) {
	i := slices.IndexFunc(s.Parameter, func(p ParameterVersi) bool { return p.ID == id })
	if i < 0 {
		return ParameterVersi{}, false
	}
	return s.Parameter[i], true
}

func (s *State) ParameterAktif() ParameterVersi {
	i := slices.IndexFunc(s.Parameter, func(p ParameterVersi) bool { return p.Status == Aktif })
	if i < 0 { // tidak terjadi lewat API; jaga-jaga data rusak
		return NewState().Parameter[0]
	}
	return s.Parameter[i]
}

// SimpanParameter membuat draft baru (id 0) atau mengganti isi draft id. Isi divalidasi penuh
// di sini, karena formulir di Panel Master selalu mengirim parameter lengkap.
func (s *State) SimpanParameter(id int, nama, dasar string, isi tax.Parameter, oleh string, now time.Time) (ParameterVersi, error) {
	nama = strings.TrimSpace(nama)
	if nama == "" {
		return ParameterVersi{}, fmt.Errorf("%w: nama versi wajib", ErrTidakValid)
	}
	if err := isi.Validasi(); err != nil {
		return ParameterVersi{}, err
	}
	v := ParameterVersi{ID: id, Nama: nama, Dasar: strings.TrimSpace(dasar), Status: Draft, Isi: isi.Salin(), Oleh: oleh, Waktu: now}
	v.Isi.Selaraskan()
	if id == 0 {
		v.ID = s.NextParameterID
		s.NextParameterID++
		s.Parameter = append(s.Parameter, v)
		return v, nil
	}
	i := slices.IndexFunc(s.Parameter, func(p ParameterVersi) bool { return p.ID == id && p.Status == Draft })
	if i < 0 {
		return ParameterVersi{}, fmt.Errorf("%w: hanya draft yang bisa diubah; buat draft baru dari versi ini", ErrTidakValid)
	}
	s.Parameter[i] = v
	return v, nil
}

func (s *State) HapusDraftParameter(id int) error {
	i := slices.IndexFunc(s.Parameter, func(p ParameterVersi) bool { return p.ID == id && p.Status == Draft })
	if i < 0 {
		return fmt.Errorf("%w: hanya draft yang bisa dihapus", ErrTidakValid)
	}
	s.Parameter = slices.Delete(s.Parameter, i, i+1)
	return nil
}

// AktifkanParameter: draft atau arsip jadi aktif; versi aktif sebelumnya jadi arsip.
func (s *State) AktifkanParameter(id int, oleh string, now time.Time) error {
	i := slices.IndexFunc(s.Parameter, func(p ParameterVersi) bool { return p.ID == id })
	if i < 0 || s.Parameter[i].Status == Aktif {
		return fmt.Errorf("%w: versi tidak ditemukan atau sudah aktif", ErrTidakValid)
	}
	if err := s.Parameter[i].Isi.Validasi(); err != nil { // arsip lama bisa tertinggal aturan validasi baru
		return err
	}
	for j := range s.Parameter {
		if s.Parameter[j].Status == Aktif {
			s.Parameter[j].Status = Arsip
		}
	}
	s.Parameter[i].Status, s.Parameter[i].Oleh, s.Parameter[i].Waktu = Aktif, oleh, now
	return nil
}

// --- Excel norma NPPN ---

var kepalaNorma = []string{"KLU", "Uraian", "10 ibukota provinsi (%)", "Ibukota provinsi lainnya (%)", "Daerah lainnya (%)"}

func normaPersen(bps int64) string {
	if bps < 0 {
		return "-"
	}
	return bpsPersen(bps)
}

func BarisNorma(ns []tax.Norma) [][]string {
	out := [][]string{kepalaNorma}
	for _, n := range ns {
		out = append(out, []string{n.KLU, n.Uraian, normaPersen(n.Ibukota10Bps), normaPersen(n.IbukotaLainBps), normaPersen(n.DaerahLainBps)})
	}
	return out
}

// ParseNorma: sel kosong atau "-" = tidak tersedia. KLU 4 angka dianggap kehilangan nol depan
// (Excel menyimpan 01111 sebagai angka 1111).
func ParseNorma(rows [][]string) ([]tax.Norma, error) {
	var out []tax.Norma
	for i, r := range rows {
		if i == 0 || kosong(r) {
			continue
		}
		n := tax.Norma{KLU: kolom(r, 0), Uraian: kolom(r, 1)}
		if len(n.KLU) == 4 {
			n.KLU = "0" + n.KLU
		}
		for j, dst := range []*int64{&n.Ibukota10Bps, &n.IbukotaLainBps, &n.DaerahLainBps} {
			v := kolom(r, 2+j)
			if v == "" || v == "-" {
				*dst = -1
				continue
			}
			b, err := persen(v)
			if err != nil {
				return nil, fmt.Errorf("%w: baris %d: norma harus angka persen", tax.ErrParameter, i+1)
			}
			*dst = b
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: tidak ada baris norma", tax.ErrParameter)
	}
	slices.SortStableFunc(out, func(a, b tax.Norma) int { return cmp.Compare(a.KLU, b.KLU) })
	p := tax.ParameterBawaan()
	p.Norma = out
	if err := p.Validasi(); err != nil {
		return nil, err
	}
	return out, nil
}
