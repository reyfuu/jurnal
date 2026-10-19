package tax

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// Benchmark: rentang Net Profit Margin (laba sebelum pajak / omzet) yang dianggap wajar
// per jenis usaha. Nilai INDIKATIF — sesuaikan dengan rasio benchmark DJP per KLU terbaru;
// cukup ubah tabel ini.
type Benchmark struct {
	MinBps int64 `json:"min_bps"`
	MaxBps int64 `json:"max_bps"`
}

var BenchmarkNPM = map[string]Benchmark{
	"perdagangan": {300, 1_000},
	"jasa":        {1_000, 3_000},
	"manufaktur":  {500, 1_500},
}

// CekAwalInput: data profil usaha yang dibutuhkan (finance mengimpor tax, bukan sebaliknya).
type CekAwalInput struct {
	JenisUsaha, BentukUsaha, PTKP string
	PKP                           bool
	Omzet, Laba                   int64 // per tahun; laba sebelum pajak (boleh negatif)
}

type Risiko struct {
	Level string `json:"level"` // tinggi | sedang | info
	Judul string `json:"judul"`
	Isi   string `json:"isi"`
}

type CekAwalResult struct {
	NPMBps    int64     `json:"npm_bps"`
	Benchmark Benchmark `json:"benchmark"`
	Status    string    `json:"status"` // kurang_data | di_bawah | wajar | di_atas
	Pesan     string    `json:"pesan"`
	Risiko    []Risiko  `json:"risiko"`
}

func pct(bps int64) string { return strings.Replace(fmt.Sprintf("%.1f%%", float64(bps)/100), ".", ",", 1) }

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
	finalOK := omzet <= batasFasilitas
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

// CekAwal: NPM vs benchmark dan risiko pemeriksaan / bentuk usaha, untuk ringkasan onboarding
// dan Perencanaan Pajak.
func CekAwal(in CekAwalInput) CekAwalResult {
	r := CekAwalResult{Benchmark: BenchmarkNPM[in.JenisUsaha], Risiko: []Risiko{}}
	if in.Omzet <= 0 {
		r.Status = "kurang_data"
		r.Pesan = "Isi perkiraan omzet dan laba per tahun untuk membandingkan dengan benchmark DJP."
		return r
	}
	r.NPMBps = in.Laba * 10_000 / in.Omzet
	rentang := pct(r.Benchmark.MinBps) + "–" + pct(r.Benchmark.MaxBps)
	switch {
	case r.NPMBps < r.Benchmark.MinBps:
		r.Status = "di_bawah"
		r.Pesan = fmt.Sprintf("NPM Anda %s, di bawah benchmark usaha sejenis (%s). Belum wajar menurut rasio DJP.", pct(r.NPMBps), rentang)
	case r.NPMBps > r.Benchmark.MaxBps:
		r.Status = "di_atas"
		r.Pesan = fmt.Sprintf("NPM Anda %s, di atas benchmark usaha sejenis (%s). Wajar dari sisi pajak; pastikan semua biaya sudah tercatat.", pct(r.NPMBps), rentang)
	default:
		r.Status = "wajar"
		r.Pesan = fmt.Sprintf("NPM Anda %s, sesuai benchmark usaha sejenis (%s).", pct(r.NPMBps), rentang)
	}

	add := func(level, judul, isi string) { r.Risiko = append(r.Risiko, Risiko{level, judul, isi}) }
	if in.Laba <= 0 {
		add("tinggi", "Usaha dilaporkan rugi", "Rugi terus-menerus sering memicu SP2DK. Siapkan bukti biaya dan pembukuan yang rapi.")
	} else if r.Status == "di_bawah" {
		add("tinggi", "Margin di bawah benchmark DJP", "DJP membandingkan SPT dengan rasio sektor; margin yang jauh lebih rendah berpotensi diminta klarifikasi (SP2DK) atau diperiksa.")
	}
	if in.Omzet > batasFasilitas {
		if !in.PKP {
			add("tinggi", "Wajib menjadi PKP", "Omzet di atas Rp4,8 miliar per tahun wajib dikukuhkan sebagai PKP dan memungut PPN.")
		}
		add("tinggi", "Tidak bisa memakai PPh Final 0,5%", "Omzet di atas Rp4,8 miliar tidak memenuhi syarat PPh Final UMKM; wajib pembukuan dan tarif umum.")
	}
	if in.BentukUsaha == "op" && in.Omzet >= batasBebasOP*8/10 && in.Omzet <= batasBebasOP {
		add("info", "Mendekati batas bebas pajak", "Omzet di atas Rp500 juta per tahun mulai dikenai PPh Final 0,5% atas kelebihannya.")
	}

	ptkp := cmp.Or(in.PTKP, "TK/0")
	if in.BentukUsaha != "op" {
		ptkp = "TK/0"
	}
	if bu, err := CompareBentukUsaha(in.Omzet, r.NPMBps, ptkp); err == nil {
		skema := bu.Skema
		if in.Omzet > batasFasilitas { // PPh Final hanya untuk omzet s.d. 4,8 M
			skema = slices.DeleteFunc(slices.Clone(skema), func(s Skema) bool { return s.ID == SkemaPPOP || s.ID == SkemaPPBadan })
		}
		now := skema[slices.IndexFunc(skema, func(s Skema) bool { return s.ID == skemaSaatIni(in.BentukUsaha, in.Omzet) })]
		best := slices.MinFunc(skema, func(a, b Skema) int { return cmp.Compare(a.Pajak, b.Pajak) })
		if hemat := now.Pajak - best.Pajak; hemat >= 1_000_000 && hemat*10 >= now.Pajak {
			add("sedang", "Bentuk usaha belum paling hemat",
				fmt.Sprintf("Skema sekarang (%s) sekitar %s per tahun. %s sekitar %s, hemat %s. Bandingkan di menu Perencanaan Pajak.",
					now.Nama, rp(now.Pajak), best.Nama, rp(best.Pajak), rp(hemat)))
		}
	}
	return r
}
