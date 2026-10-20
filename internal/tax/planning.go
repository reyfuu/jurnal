// Package tax: perencanaan pajak UMKM, diadaptasi dari Tax Planning Calculator V2.1
// (BSV Consulting): sheet "Bentuk Usaha" dan "Perbandingan Tarif". Semua nilai Rupiah per tahun.
package tax

import (
	"cmp"
	"fmt"
	"slices"
)

// PTKP per status (template sheet PPh OP, tabel AS3:AU14).
var PTKP = map[string]int64{
	"TK/0": 54_000_000, "TK/1": 58_500_000, "TK/2": 63_000_000, "TK/3": 67_500_000,
	"K/0": 58_500_000, "K/1": 63_000_000, "K/2": 67_500_000, "K/3": 72_000_000,
	"K/I/0": 112_500_000, "K/I/1": 117_000_000, "K/I/2": 121_500_000, "K/I/3": 126_000_000,
}

// PTKPCodes: urutan tampilan dropdown.
var PTKPCodes = []string{"TK/0", "TK/1", "TK/2", "TK/3", "K/0", "K/1", "K/2", "K/3", "K/I/0", "K/I/1", "K/I/2", "K/I/3"}

const (
	batasBebasOP   = 500_000_000    // omzet bebas PPh final WP OP per tahun
	batasFasilitas = 4_800_000_000  // Pasal 31E: fasilitas penuh sampai omzet 4,8 M
	batas31E       = 50_000_000_000 // di atas 50 M tanpa fasilitas
)

var brackets = []struct{ upTo, bps int64 }{
	{60_000_000, 500}, {250_000_000, 1500}, {500_000_000, 2500}, {5_000_000_000, 3000}, {-1, 3500},
}

// Pasal17: tarif progresif PPh orang pribadi atas penghasilan kena pajak.
func Pasal17(pkp int64) int64 {
	var tax, lower int64
	for _, b := range brackets {
		if pkp <= lower {
			break
		}
		upper := pkp
		if b.upTo > 0 && b.upTo < pkp {
			upper = b.upTo
		}
		tax += (upper - lower) * b.bps / 10_000
		lower = b.upTo
		if b.upTo < 0 {
			break
		}
	}
	return tax
}

// Pasal31E: PPh badan 22% dengan fasilitas 50% untuk porsi PKP dari omzet s.d. 4,8 M.
func Pasal31E(omzet, pkp int64) int64 {
	p := pkp / 1000 * 1000
	if p <= 0 {
		return 0
	}
	fas := p
	switch {
	case omzet > batas31E:
		fas = 0
	case omzet >= batasFasilitas:
		fas = batasFasilitas * p / omzet
	}
	return fas*11/100 + (p-fas)*22/100
}

func ppFinalOP(omzet int64) int64 { return max(0, omzet-batasBebasOP) * 5 / 1000 }

const (
	SkemaPPOP      = "pp_op"
	SkemaPPBadan   = "pp_badan"
	SkemaPasal17OP = "pasal17_op"
	SkemaPasal31E  = "pasal31e"
	SkemaBadan22   = "badan22"
	SkemaNPPN      = "nppn_op"
)

type Skema struct {
	ID            string `json:"id"`
	Nama          string `json:"nama"`
	Keterangan    string `json:"keterangan"`
	Pajak         int64  `json:"pajak"`
	LebihMahalBps int64  `json:"lebih_mahal_bps"` // (pajak - termurah) / pajak
}

type BentukUsahaResult struct {
	Omzet    int64   `json:"omzet"`
	Laba     int64   `json:"laba"`
	PTKP     int64   `json:"ptkp"`
	Skema    []Skema `json:"skema"`
	Termurah string  `json:"termurah"`
}

// CompareBentukUsaha membandingkan 6 skema pajak setahun dan memilih yang termurah
// (seri: urutan pertama menang, sama seperti XLOOKUP di template).
func CompareBentukUsaha(omzet int64, netMarginBps int64, ptkpCode string) (BentukUsahaResult, error) {
	ptkp, ok := PTKP[ptkpCode]
	if !ok {
		return BentukUsahaResult{}, fmt.Errorf("status PTKP %q tidak dikenal", ptkpCode)
	}
	laba := omzet * netMarginBps / 10_000
	r := BentukUsahaResult{Omzet: omzet, Laba: laba, PTKP: ptkp, Skema: []Skema{
		{SkemaPPOP, "PPh Final 0,5% Orang Pribadi", "Omzet di atas Rp500 jt x 0,5%", ppFinalOP(omzet), 0},
		{SkemaPPBadan, "PPh Final 0,5% Badan", "Seluruh omzet x 0,5%", omzet * 5 / 1000, 0},
		{SkemaPasal17OP, "PPh Pasal 17 Orang Pribadi (pembukuan)", "Tarif progresif atas laba - PTKP", Pasal17(laba - ptkp), 0},
		{SkemaPasal31E, "PPh Badan Pasal 31E", "22% dengan fasilitas 50% untuk omzet s.d. Rp4,8 M", Pasal31E(omzet, laba), 0},
		{SkemaBadan22, "PPh Badan Tarif Umum 22%", "Laba x 22%", max(0, laba*22/100), 0},
		{SkemaNPPN, "Norma (NPPN) Orang Pribadi", "Tarif progresif atas 50% omzet - PTKP", Pasal17(omzet/2 - ptkp), 0},
	}}
	if omzet > batasFasilitas { // PP 55/2022 hanya untuk omzet s.d. Rp4,8 M
		r.Skema = slices.DeleteFunc(r.Skema, func(s Skema) bool { return s.ID == SkemaPPOP || s.ID == SkemaPPBadan })
	}
	best := slices.MinFunc(r.Skema, func(a, b Skema) int { return cmp.Compare(a.Pajak, b.Pajak) })
	r.Termurah = best.ID
	for i := range r.Skema {
		if p := r.Skema[i].Pajak; p > 0 {
			r.Skema[i].LebihMahalBps = (p - best.Pajak) * 10_000 / p
		}
	}
	return r, nil
}

type Entitas struct {
	ID        string `json:"id"`
	Nama      string `json:"nama"`
	PPh       int64  `json:"pph"`
	Dividen   int64  `json:"dividen"` // PPh dividen 10% atas laba setelah pajak
	Total     int64  `json:"total"`
	PersenBps int64  `json:"persen_bps"` // total / omzet
}

// CompareEntitas: total beban pajak per bentuk badan (sheet "Perbandingan Tarif").
// costBps = HPP + biaya operasional sebagai bps dari omzet.
func CompareEntitas(omzet, costBps int64) []Entitas {
	laba := omzet * (10_000 - costBps) / 10_000
	rows := []struct {
		id, nama string
		pph      int64
		dividen  bool
		ppFinal  bool
	}{
		{"op", "Orang Pribadi (PPh Final 0,5%)", ppFinalOP(omzet), false, true},
		{"pt_perorangan", "PT Perorangan (PPh Final 0,5%)", omzet * 5 / 1000, true, true},
		{"cv_pp", "CV (PPh Final 0,5%)", omzet * 5 / 1000, false, true},
		{"cv", "CV (Pasal 31E)", Pasal31E(omzet, laba), false, false},
		{"pt", "PT (Pasal 31E)", Pasal31E(omzet, laba), true, false},
		{"badan_umum", "Badan Tarif Umum 22%", max(0, laba*22/100), true, false},
	}
	out := make([]Entitas, 0, len(rows))
	for _, r := range rows {
		if r.ppFinal && omzet > batasFasilitas { // PP 55/2022 hanya untuk omzet s.d. Rp4,8 M
			continue
		}
		e := Entitas{ID: r.id, Nama: r.nama, PPh: r.pph}
		if r.dividen {
			e.Dividen = max(0, laba-r.pph) / 10
		}
		e.Total = e.PPh + e.Dividen
		if omzet > 0 {
			e.PersenBps = e.Total * 10_000 / omzet
		}
		out = append(out, e)
	}
	return out
}

// HasilPKP: posisi omzet setahun terhadap batas pengusaha kecil PPN (Rp4,8 M).
type HasilPKP struct {
	Omzet        int64    `json:"omzet"`
	Batas        int64    `json:"batas"`
	WajibPKP     bool     `json:"wajib_pkp"`
	Sisa         int64    `json:"sisa"`          // ruang omzet sebelum wajib PKP
	PemakaianBps int64    `json:"pemakaian_bps"` // omzet / batas, maks 100%
	Catatan      []string `json:"catatan"`
}

// CekPKP: pengusaha kecil (PMK 197/2013) beromzet s.d. Rp4,8 M setahun buku. Lewat dari itu
// wajib melaporkan usaha untuk dikukuhkan PKP dan mulai memungut PPN.
func CekPKP(omzet int64) HasilPKP {
	omzet = max(0, omzet)
	r := HasilPKP{Omzet: omzet, Batas: batasFasilitas, WajibPKP: omzet > batasFasilitas,
		Sisa: max(0, batasFasilitas-omzet), PemakaianBps: min(10_000, omzet*10_000/batasFasilitas)}
	if r.WajibPKP {
		r.Catatan = []string{
			"Omzet melewati Rp4,8 M: wajib melaporkan usaha untuk dikukuhkan sebagai PKP paling lambat akhir bulan berikutnya setelah bulan omzet terlampaui (PMK 197/2013).",
			"Setelah dikukuhkan, PKP memungut PPN atas penyerahan, menerbitkan e-Faktur, dan melaporkan SPT Masa PPN setiap bulan.",
			"Tahun pajak berikutnya tidak lagi memenuhi syarat PPh Final 0,5% (PP 55/2022); pajak penghasilan memakai tarif umum atau Pasal 31E untuk badan.",
		}
	} else {
		r.Catatan = []string{
			"Masih pengusaha kecil: belum wajib dikukuhkan PKP, tetapi boleh memilih menjadi PKP secara sukarela (misalnya bila pembeli utama adalah PKP yang butuh faktur pajak).",
			"Pantau omzet kumulatif setiap bulan; begitu melewati Rp4,8 M, kewajiban PKP berlaku mulai bulan berikutnya.",
		}
	}
	return r
}
