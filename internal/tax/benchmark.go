package tax

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
)

// KLUBenchmark: rata-rata nasional CTTOR (PPh terutang / peredaran usaha) dan NPM (laba sebelum
// pajak / peredaran usaha) per kelompok KLU 3 digit. Data di klu_data.go, dibuat dari PDF DJP
// oleh scripts/gen_klu_benchmark.py.
type KLUBenchmark struct {
	Kode     string `json:"kode"`
	Nama     string `json:"nama"`
	CTTORBps int64  `json:"cttor_bps"`
	NPMBps   int64  `json:"npm_bps"`
}

const SumberBenchmark = "Buku sektoral DJP 2021 (SPT tahun pajak 2020), angka nasional"

type benchmarkSet struct {
	baris  []KLUBenchmark
	sumber string
}

// aktif: versi benchmark yang dipakai seluruh aplikasi; diganti dari Panel Master
// (AktifkanBenchmark). Bawaan = data DJP yang dibangkitkan di klu_data.go.
var aktif atomic.Pointer[benchmarkSet]

func init() { AktifkanBenchmark(KLUBenchmarks, SumberBenchmark) }

// AktifkanBenchmark mengganti versi benchmark yang dipakai FindKLU & CekAwal. baris disalin.
func AktifkanBenchmark(baris []KLUBenchmark, sumber string) {
	aktif.Store(&benchmarkSet{slices.Clone(baris), sumber})
}

// BenchmarkAktif: salinan baris versi aktif.
func BenchmarkAktif() []KLUBenchmark { return slices.Clone(aktif.Load().baris) }

func FindKLU(kode string) (KLUBenchmark, bool) {
	b := aktif.Load().baris
	i := slices.IndexFunc(b, func(k KLUBenchmark) bool { return k.Kode == kode })
	if i < 0 {
		return KLUBenchmark{}, false
	}
	return b[i], true
}

// JenisKLU: jenis usaha aplikasi untuk menyaring daftar KLU di onboarding.
// Golongan 45-47 = perdagangan, 01-33 = pertanian, tambang & industri (produksi), sisanya jasa.
func JenisKLU(kode string) string {
	gol, _ := strconv.Atoi(kode[:min(2, len(kode))])
	switch {
	case gol >= 45 && gol <= 47:
		return "perdagangan"
	case gol >= 1 && gol <= 33:
		return "manufaktur"
	}
	return "jasa"
}

// CekAwalInput: data profil usaha yang dibutuhkan (finance mengimpor tax, bukan sebaliknya).
type CekAwalInput struct {
	BentukUsaha, PTKP, KLU string
	PKP                    bool
	Omzet, Laba            int64 // per tahun; laba sebelum pajak (boleh negatif)
}

type Risiko struct {
	Level string `json:"level"` // tinggi | sedang | info
	Judul string `json:"judul"`
	Isi   string `json:"isi"`
}

type CekAwalResult struct {
	KLU        *KLUBenchmark `json:"klu"`
	NPMBps     int64         `json:"npm_bps"`
	CTTORBps   int64         `json:"cttor_bps"`
	Status     string        `json:"status"` // kurang_data | di_bawah | wajar
	Pesan      string        `json:"pesan"`
	CTTORPesan string        `json:"cttor_pesan,omitempty"`
	Sumber     string        `json:"sumber"`
	Risiko     []Risiko      `json:"risiko"`
	Saran      *SaranBentuk  `json:"saran"` // skema pajak paling hemat; nil bila omzet kosong
}

func pct(bps int64) string {
	return strings.Replace(fmt.Sprintf("%.2f%%", float64(bps)/100), ".", ",", 1)
}

func rp(n int64) string {
	s := fmt.Sprint(max(n, -n))
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "." + s[i:]
	}
	if n < 0 {
		return "-Rp" + s
	}
	return "Rp" + s
}

// skemaSaatIni: skema PPh yang berlaku untuk bentuk usaha sekarang (sama dengan ringkasan onboarding).
func skemaSaatIni(bentuk string, omzet int64) string {
	finalOK := omzet <= P().BatasOmzetUMKM
	switch {
	case bentuk == "op" && finalOK:
		return SkemaPPOP
	case bentuk == "op":
		return SkemaPasal17OP
	case bentuk == "pt_perorangan" && finalOK:
		return SkemaPPBadan
	default:
		return SkemaPasal31E
	}
}

// CekAwal: NPM & CTTOR usaha vs rata-rata nasional KLU-nya, plus risiko pemeriksaan / bentuk
// usaha, untuk ringkasan onboarding dan Perencanaan Pajak.
func CekAwal(in CekAwalInput) CekAwalResult {
	r := CekAwalResult{Sumber: aktif.Load().sumber, Risiko: []Risiko{}, Saran: SaranBentukUsaha(in.BentukUsaha, in.PTKP, in.Omzet, in.Laba)}
	if k, ok := FindKLU(in.KLU); ok {
		r.KLU = &k
	}
	switch {
	case in.Omzet <= 0:
		r.Status = "kurang_data"
		r.Pesan = "Isi perkiraan omzet dan laba per tahun untuk membandingkan dengan rata-rata DJP."
		return r
	case r.KLU == nil:
		r.Status = "kurang_data"
		r.Pesan = "Pilih bidang usaha (KLU) di Profil Usaha untuk membandingkan dengan rata-rata DJP usaha sejenis."
		return r
	}
	k := *r.KLU
	r.NPMBps = in.Laba * 10_000 / in.Omzet
	if r.NPMBps < k.NPMBps {
		r.Status = "di_bawah"
		r.Pesan = fmt.Sprintf("NPM Anda %s, di bawah rata-rata nasional %s untuk KLU %s (%s).", pct(r.NPMBps), pct(k.NPMBps), k.Kode, k.Nama)
	} else {
		r.Status = "wajar"
		r.Pesan = fmt.Sprintf("NPM Anda %s, setara atau di atas rata-rata nasional %s untuk KLU %s (%s).", pct(r.NPMBps), pct(k.NPMBps), k.Kode, k.Nama)
	}

	add := func(level, judul, isi string) { r.Risiko = append(r.Risiko, Risiko{level, judul, isi}) }
	if in.Laba <= 0 {
		add("tinggi", "Usaha dilaporkan rugi", "Rugi terus-menerus sering memicu SP2DK. Siapkan bukti biaya dan pembukuan yang rapi.")
	} else if r.Status == "di_bawah" {
		add("tinggi", "NPM di bawah rata-rata DJP", fmt.Sprintf("DJP membandingkan SPT dengan rasio sektor. NPM %s di bawah rata-rata nasional %s berpotensi diminta klarifikasi (SP2DK) atau diperiksa.", pct(r.NPMBps), pct(k.NPMBps)))
	}
	pr := P()
	if in.Omzet > pr.BatasPKP && !in.PKP {
		add("tinggi", "Wajib menjadi PKP", "Omzet di atas "+rp(pr.BatasPKP)+" per tahun wajib dikukuhkan sebagai PKP dan memungut PPN.")
	}
	umkm := persenBps(int(pr.UMKMTarifBps))
	if in.Omzet > pr.BatasOmzetUMKM {
		add("tinggi", "Tidak bisa memakai PPh Final "+umkm, "Omzet di atas "+rp(pr.BatasOmzetUMKM)+" tidak memenuhi syarat PPh Final UMKM; wajib pembukuan dan tarif umum.")
	}
	if in.BentukUsaha == "op" && in.Omzet >= pr.UMKMBebasOP*8/10 && in.Omzet <= pr.UMKMBebasOP {
		add("info", "Mendekati batas bebas pajak", "Omzet di atas "+rp(pr.UMKMBebasOP)+" per tahun mulai dikenai PPh Final "+umkm+" atas kelebihannya.")
	}

	ptkp := cmp.Or(in.PTKP, "TK/0")
	if in.BentukUsaha != "op" {
		ptkp = "TK/0"
	}
	bu, err := CompareBentukUsaha(in.Omzet, r.NPMBps, ptkp)
	if err != nil {
		return r
	}
	skema := bu.Skema
	if in.Omzet > pr.BatasOmzetUMKM { // PPh final hanya sampai batas omzet UMKM
		skema = slices.DeleteFunc(slices.Clone(skema), func(s Skema) bool { return s.ID == SkemaPPOP || s.ID == SkemaPPBadan })
	}
	now := skema[slices.IndexFunc(skema, func(s Skema) bool { return s.ID == skemaSaatIni(in.BentukUsaha, in.Omzet) })]

	// CTTOR: pajak skema sekarang / omzet. Untuk PPh Final, pajaknya ditetapkan dari omzet oleh
	// aturan, jadi angkanya hanya informasi, bukan sinyal risiko.
	r.CTTORBps = now.Pajak * 10_000 / in.Omzet
	final := now.ID == SkemaPPOP || now.ID == SkemaPPBadan
	switch {
	case final:
		r.CTTORPesan = fmt.Sprintf("CTTOR Anda %s dengan PPh Final %s; rata-rata nasional %s. Untuk PPh Final, besarnya pajak ditetapkan dari omzet, jadi perbandingan ini hanya informasi.", pct(r.CTTORBps), persenBps(int(pr.UMKMTarifBps)), pct(k.CTTORBps))
	case r.CTTORBps < k.CTTORBps:
		r.CTTORPesan = fmt.Sprintf("CTTOR Anda %s, di bawah rata-rata nasional %s.", pct(r.CTTORBps), pct(k.CTTORBps))
		add("sedang", "CTTOR di bawah rata-rata DJP", fmt.Sprintf("Perbandingan pajak terutang dengan omzet (%s) lebih rendah dari rata-rata nasional KLU %s (%s). Pastikan semua penghasilan tercatat.", pct(r.CTTORBps), k.Kode, pct(k.CTTORBps)))
	default:
		r.CTTORPesan = fmt.Sprintf("CTTOR Anda %s, setara atau di atas rata-rata nasional %s.", pct(r.CTTORBps), pct(k.CTTORBps))
	}
	return r
}
