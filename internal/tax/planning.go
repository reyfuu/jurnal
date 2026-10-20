// Package tax: perencanaan pajak UMKM, diadaptasi dari Tax Planning Calculator V2.1
// (BSV Consulting): sheet "Bentuk Usaha" dan "Perbandingan Tarif". Semua nilai Rupiah per tahun.
package tax

import (
	"cmp"
	"fmt"
	"math/big"
	"slices"
)

// ptkpBawaan per status (template sheet PPh OP, tabel AS3:AU14). Angka yang berlaku: PTKPOf.
var ptkpBawaan = map[string]int64{
	"TK/0": 54_000_000, "TK/1": 58_500_000, "TK/2": 63_000_000, "TK/3": 67_500_000,
	"K/0": 58_500_000, "K/1": 63_000_000, "K/2": 67_500_000, "K/3": 72_000_000,
	"K/I/0": 112_500_000, "K/I/1": 117_000_000, "K/I/2": 121_500_000, "K/I/3": 126_000_000,
}

// PTKPCodes: urutan tampilan dropdown. K/I/* (penghasilan istri digabung) tidak ditawarkan lagi,
// tetapi tetap ada di PTKP supaya profil/karyawan lama yang memakainya masih terhitung.
var PTKPCodes = []string{"TK/0", "TK/1", "TK/2", "TK/3", "K/0", "K/1", "K/2", "K/3"}

// Pasal17: tarif progresif PPh orang pribadi atas penghasilan kena pajak.
func Pasal17(pkp int64) int64 {
	var tax, lower int64
	for _, l := range P().Pasal17 {
		if pkp <= lower {
			break
		}
		upper := pkp
		if l.Sampai > 0 && l.Sampai < pkp {
			upper = l.Sampai
		}
		tax += (upper - lower) * l.TarifBps / 10_000
		if l.Sampai == 0 {
			break
		}
		lower = l.Sampai
	}
	return tax
}

// Pasal31E: PPh badan dengan fasilitas pengurangan tarif untuk porsi PKP dari omzet s.d.
// Batas31EPenuh; omzet di atas Batas31E tanpa fasilitas.
func Pasal31E(omzet, pkp int64) int64 {
	pr := P()
	p := pkp / 1000 * 1000
	if p <= 0 {
		return 0
	}
	fas := p
	switch {
	case omzet > pr.Batas31E:
		fas = 0
	case omzet >= pr.Batas31EPenuh:
		fas = mulDiv(pr.Batas31EPenuh, p, omzet)
	}
	return mulDiv(fas, pr.BadanTarifBps*(10_000-pr.Fasilitas31EBps), 100_000_000) + (p-fas)*pr.BadanTarifBps/10_000
}

func ppFinalOP(omzet int64) int64 {
	return max(0, omzet-P().UMKMBebasOP) * P().UMKMTarifBps / 10_000
}

// mulDiv: a*b/c tanpa luber int64 (omzet puluhan miliar x basis poin).
func mulDiv(a, b, c int64) int64 {
	if c == 0 {
		return 0
	}
	r := new(big.Int).Mul(big.NewInt(a), big.NewInt(b))
	return r.Quo(r, big.NewInt(c)).Int64()
}

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
	return compareLaba(omzet, mulDiv(omzet, netMarginBps, 10_000), ptkpCode)
}

// compareLaba: CompareBentukUsaha dengan laba rupiah langsung (tanpa pembulatan lewat margin).
func compareLaba(omzet, laba int64, ptkpCode string) (BentukUsahaResult, error) {
	ptkp, ok := PTKPOf(ptkpCode)
	if !ok {
		return BentukUsahaResult{}, fmt.Errorf("status PTKP %q tidak dikenal", ptkpCode)
	}
	pr := P()
	umkm, badan := persenBps(int(pr.UMKMTarifBps)), persenBps(int(pr.BadanTarifBps))
	r := BentukUsahaResult{Omzet: omzet, Laba: laba, PTKP: ptkp, Skema: []Skema{
		{SkemaPPOP, "PPh Final " + umkm + " Orang Pribadi", "Omzet di atas " + rp(pr.UMKMBebasOP) + " x " + umkm, ppFinalOP(omzet), 0},
		{SkemaPPBadan, "PPh Final " + umkm + " Badan", "Seluruh omzet x " + umkm, omzet * pr.UMKMTarifBps / 10_000, 0},
		{SkemaPasal17OP, "PPh Pasal 17 Orang Pribadi (pembukuan)", "Tarif progresif atas laba - PTKP", Pasal17(laba - ptkp), 0},
		{SkemaPasal31E, "PPh Badan Pasal 31E", fmt.Sprintf("%s dengan fasilitas %s untuk omzet s.d. %s", badan, persenBps(int(pr.Fasilitas31EBps)), rp(pr.Batas31EPenuh)), Pasal31E(omzet, laba), 0},
		{SkemaBadan22, "PPh Badan Tarif Umum " + badan, "Laba x " + badan, max(0, laba*pr.BadanTarifBps/10_000), 0},
		{SkemaNPPN, "Norma (NPPN) Orang Pribadi", "Tarif progresif atas " + persenBps(int(pr.NormaPerbandinganBps)) + " omzet - PTKP", Pasal17(mulDiv(omzet, pr.NormaPerbandinganBps, 10_000) - ptkp), 0},
	}}
	if omzet > pr.BatasOmzetUMKM { // PPh final UMKM hanya sampai batas omzet
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
	pr := P()
	laba := mulDiv(omzet, 10_000-costBps, 10_000)
	umkm := persenBps(int(pr.UMKMTarifBps))
	rows := []struct {
		id, nama string
		pph      int64
		dividen  bool
		ppFinal  bool
	}{
		{"op", "Orang Pribadi (PPh Final " + umkm + ")", ppFinalOP(omzet), false, true},
		{"pt_perorangan", "PT Perorangan (PPh Final " + umkm + ")", omzet * pr.UMKMTarifBps / 10_000, true, true},
		{"cv_pp", "CV (PPh Final " + umkm + ")", omzet * pr.UMKMTarifBps / 10_000, false, true},
		{"cv", "CV (Pasal 31E)", Pasal31E(omzet, laba), false, false},
		{"pt", "PT (Pasal 31E)", Pasal31E(omzet, laba), true, false},
		{"badan_umum", "Badan Tarif Umum " + persenBps(int(pr.BadanTarifBps)), max(0, laba*pr.BadanTarifBps/10_000), true, false},
	}
	out := make([]Entitas, 0, len(rows))
	for _, r := range rows {
		if r.ppFinal && omzet > pr.BatasOmzetUMKM { // PPh final UMKM hanya sampai batas omzet
			continue
		}
		e := Entitas{ID: r.id, Nama: r.nama, PPh: r.pph}
		if r.dividen {
			e.Dividen = max(0, laba-r.pph) * pr.DividenBps / 10_000
		}
		e.Total = e.PPh + e.Dividen
		if omzet > 0 {
			e.PersenBps = e.Total * 10_000 / omzet
		}
		out = append(out, e)
	}
	return out
}

// HasilPKP: posisi omzet setahun terhadap batas pengusaha kecil PPN (Parameter.BatasPKP).
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
	batas := P().BatasPKP
	r := HasilPKP{Omzet: omzet, Batas: batas, WajibPKP: omzet > batas,
		Sisa: max(0, batas-omzet), PemakaianBps: min(10_000, mulDiv(omzet, 10_000, batas))}
	if r.WajibPKP {
		r.Catatan = []string{
			"Omzet melewati " + rp(batas) + ": wajib melaporkan usaha untuk dikukuhkan sebagai PKP paling lambat akhir bulan berikutnya setelah bulan omzet terlampaui (PMK 197/2013).",
			"Setelah dikukuhkan, PKP memungut PPN atas penyerahan, menerbitkan e-Faktur, dan melaporkan SPT Masa PPN setiap bulan.",
			"Tahun pajak berikutnya tidak lagi memenuhi syarat PPh Final UMKM (" + P().DasarUMKM + "); pajak penghasilan memakai tarif umum atau Pasal 31E untuk badan.",
		}
	} else {
		r.Catatan = []string{
			"Masih pengusaha kecil: belum wajib dikukuhkan PKP, tetapi boleh memilih menjadi PKP secara sukarela (misalnya bila pembeli utama adalah PKP yang butuh faktur pajak).",
			"Pantau omzet kumulatif setiap bulan; begitu melewati " + rp(batas) + ", kewajiban PKP berlaku mulai bulan berikutnya.",
		}
	}
	return r
}
