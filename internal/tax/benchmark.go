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
// pajak / peredaran usaha) per kelompok KLU 3 digit. Data di klu_data.go, dibuat dari PDF sumber
// oleh scripts/gen_klu_benchmark.py.
type KLUBenchmark struct {
	Kode     string `json:"kode"`
	Nama     string `json:"nama"`
	CTTORBps int64  `json:"cttor_bps"`
	NPMBps   int64  `json:"npm_bps"`
}

const SumberBenchmark = "Rata-rata nasional per KLU, data tahun pajak 2020"

type benchmarkSet struct {
	baris  []KLUBenchmark
	sumber string
}

// aktif: versi benchmark yang dipakai seluruh aplikasi; diganti dari Panel Master
// (AktifkanBenchmark). Bawaan = data yang dibangkitkan di klu_data.go.
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
	Modal                  int64 // modal yang ditanam pemilik; 0 = belum diisi (ROI tidak dihitung)
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
	// ROI tahun pertama = laba / modal; dihitung hanya bila modal diisi.
	ModalAwal int64  `json:"modal_awal,omitempty"`
	ROIBps    *int64 `json:"roi_bps,omitempty"`
	ROIPesan  string `json:"roi_pesan,omitempty"`
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
func skemaSaatIni(bentuk string, omzet int64, tahunPertama bool) string {
	finalOK := finalBoleh(omzet, tahunPertama)
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

// kluKebutuhanPokok: bidang usaha yang umumnya menjual barang kebutuhan pokok yang PPN-nya
// dibebaskan (PP 49/2022 Pasal 7 & Lampiran: beras, gabah, jagung, sagu, kedelai, garam,
// daging, telur, susu, buah-buahan, sayur-sayuran, gula konsumsi kristal putih dari tebu),
// sesuai kriteria lampiran (umumnya segar / belum diolah). Perlu diverifikasi tim pajak.
var kluKebutuhanPokok = map[string]bool{
	"011": true, // pertanian tanaman semusim (padi, jagung, sayur)
	"012": true, // tanaman tahunan (buah)
	"014": true, // peternakan (telur, daging, susu)
	"106": true, // penggilingan padi (beras)
	"462": true, // perdagangan besar hasil pertanian & hewan hidup
	"463": true, // perdagangan besar makanan (beras, gula, telur grosir)
	"472": true, // eceran khusus makanan di toko
	"478": true, // kaki lima & los pasar
}

// CekAwal: NPM & CTTOR usaha vs rata-rata nasional KLU-nya, plus risiko pemeriksaan / bentuk
// usaha, untuk ringkasan onboarding dan Perencanaan Pajak.
func CekAwal(in CekAwalInput) CekAwalResult {
	r := CekAwalResult{Sumber: aktif.Load().sumber, Risiko: []Risiko{}, Saran: SaranBentukUsaha(in.BentukUsaha, in.PTKP, in.Omzet, in.Laba)}
	if k, ok := FindKLU(in.KLU); ok {
		r.KLU = &k
	}
	if in.Modal > 0 {
		roi := mulDiv(in.Laba, 10_000, in.Modal)
		r.ModalAwal, r.ROIBps = in.Modal, &roi
		r.ROIPesan = fmt.Sprintf("ROI tahun pertama %s: laba %s dari modal %s.", pct(roi), rp(in.Laba), rp(in.Modal))
		if in.Laba > 0 {
			r.ROIPesan += fmt.Sprintf(" Dengan laba setahun yang sama, modal kembali dalam sekitar %s.", lamaBalikModal(in.Modal, in.Laba))
		}
	}
	switch {
	case in.Omzet <= 0:
		r.Status = "kurang_data"
		r.Pesan = "Isi perkiraan omzet dan laba per tahun untuk membandingkan dengan rata-rata nasional."
		return r
	case r.KLU == nil:
		r.Status = "kurang_data"
		r.Pesan = "Pilih bidang usaha (KLU) di Profil Usaha untuk membandingkan dengan rata-rata nasional usaha sejenis."
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
		add("tinggi", "NPM di bawah rata-rata nasional", fmt.Sprintf("NPM %s di bawah rata-rata nasional %s. Margin yang jauh di bawah rata-rata sektor berpotensi diminta klarifikasi (SP2DK) atau diperiksa.", pct(r.NPMBps), pct(k.NPMBps)))
	}
	pr := P()
	if in.Omzet > pr.BatasPKP && !in.PKP {
		if kluKebutuhanPokok[in.KLU] {
			// Bahan kebutuhan pokok (beras, telur, daging, susu, sayur, buah, gula konsumsi, …)
			// mendapat fasilitas PPN dibebaskan (PP 49/2022), jadi pengusaha yang hanya menjual
			// barang itu tidak memungut PPN. Bidang usaha ini bisa juga menjual barang lain yang
			// kena PPN, jadi ini peringatan untuk diperiksa, bukan kepastian wajib PKP.
			add("sedang", "Omzet di atas batas PKP: cek fasilitas PPN dibebaskan",
				"Omzet di atas "+rp(pr.BatasPKP)+" per tahun. Barang kebutuhan pokok (beras, gabah, jagung, sagu, kedelai, garam, daging, telur, susu, buah-buahan, sayur-sayuran, dan gula konsumsi) dibebaskan PPN berdasarkan PP 49/2022 sesuai kriteria lampirannya (umumnya segar / belum diolah), jadi tidak ada PPN yang dipungut dari pembeli. Bila ada penjualan barang lain yang kena PPN (mis. barang olahan), usaha wajib dikukuhkan sebagai PKP. Pastikan bersama konsultan pajak.")
		} else {
			add("tinggi", "Wajib menjadi PKP", "Omzet di atas "+rp(pr.BatasPKP)+" per tahun wajib dikukuhkan sebagai PKP dan memungut PPN, kecuali seluruh penjualannya barang / jasa yang tidak dikenai atau dibebaskan PPN.")
		}
	}
	umkm := persenBps(int(pr.UMKMTarifBps))
	if in.Omzet > pr.BatasOmzetUMKM {
		add("sedang", "PPh Final "+umkm+" hanya untuk tahun pertama", "Omzet di atas "+rp(pr.BatasOmzetUMKM)+": PPh Final UMKM masih berlaku sampai akhir tahun pajak saat omzet pertama kali melewati batas. Mulai tahun berikutnya wajib pembukuan dan tarif umum (Pasal 17 / Pasal 31E).")
	}
	if in.BentukUsaha == "op" && in.Omzet >= pr.UMKMBebasOP*8/10 && in.Omzet <= pr.UMKMBebasOP {
		add("info", "Mendekati batas bebas pajak", "Omzet di atas "+rp(pr.UMKMBebasOP)+" per tahun mulai dikenai PPh Final "+umkm+" atas kelebihannya.")
	}

	ptkp := cmp.Or(in.PTKP, "TK/0")
	if in.BentukUsaha != "op" {
		ptkp = "TK/0"
	}
	// perkiraan onboarding = tahun pertama: PPh final tetap boleh walau omzet melewati batas
	bu, err := compareLaba(in.Omzet, mulDiv(in.Omzet, r.NPMBps, 10_000), ptkp, true)
	if err != nil {
		return r
	}
	skema := bu.Skema
	now := skema[slices.IndexFunc(skema, func(s Skema) bool { return s.ID == skemaSaatIni(in.BentukUsaha, in.Omzet, true) })]

	// CTTOR: pajak skema sekarang / omzet. Untuk PPh Final, pajaknya ditetapkan dari omzet oleh
	// aturan, jadi angkanya hanya informasi, bukan sinyal risiko.
	r.CTTORBps = now.Pajak * 10_000 / in.Omzet
	final := now.ID == SkemaPPOP || now.ID == SkemaPPBadan
	switch {
	case final:
		r.CTTORPesan = fmt.Sprintf("CTTOR Anda %s dengan PPh Final %s; rata-rata nasional %s. Untuk PPh Final, besarnya pajak ditetapkan dari omzet, jadi perbandingan ini hanya informasi.", pct(r.CTTORBps), persenBps(int(pr.UMKMTarifBps)), pct(k.CTTORBps))
	case r.CTTORBps < k.CTTORBps:
		r.CTTORPesan = fmt.Sprintf("CTTOR Anda %s, di bawah rata-rata nasional %s.", pct(r.CTTORBps), pct(k.CTTORBps))
		add("sedang", "CTTOR di bawah rata-rata nasional", fmt.Sprintf("Perbandingan pajak terutang dengan omzet (%s) lebih rendah dari rata-rata nasional KLU %s (%s). Pastikan semua penghasilan tercatat.", pct(r.CTTORBps), k.Kode, pct(k.CTTORBps)))
	default:
		r.CTTORPesan = fmt.Sprintf("CTTOR Anda %s, setara atau di atas rata-rata nasional %s.", pct(r.CTTORBps), pct(k.CTTORBps))
	}
	return r
}

// lamaBalikModal: perkiraan waktu modal kembali (modal / laba setahun), dibulatkan ke atas per bulan.
func lamaBalikModal(modal, labaSetahun int64) string {
	bulan := (modal*12 + labaSetahun - 1) / labaSetahun
	switch {
	case bulan < 12:
		return fmt.Sprintf("%d bulan", bulan)
	case bulan%12 == 0:
		return fmt.Sprintf("%d tahun", bulan/12)
	default:
		return fmt.Sprintf("%d tahun %d bulan", bulan/12, bulan%12)
	}
}
