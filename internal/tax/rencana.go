package tax

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Kalkulator perencanaan pajak dari Tax Planning Calculator V2.1 (BSV Consulting): NPPN, gross
// up nilai kontrak, PPN & harga jual, beli vs sewa aset. Semua angka aturan dari P().

// ===== NPPN =====

type NPPNInput struct {
	Omzet   int64  `json:"omzet"` // setahun
	KLU     string `json:"klu"`   // 5 digit; kosong = pakai NormaBps
	Wilayah string `json:"wilayah"`
	// NormaBps: norma manual bila KLU kosong (misalnya dari surat DJP).
	NormaBps int64  `json:"norma_bps"`
	PTKP     string `json:"ptkp"`
}

type NPPNResult struct {
	Norma    *Norma    `json:"norma"`
	NormaBps int64     `json:"norma_bps"`
	Neto     int64     `json:"neto"`
	PTKP     int64     `json:"ptkp"`
	PKP      int64     `json:"pkp"`
	PPh      int64     `json:"pph"`
	PPhFinal int64     `json:"pph_final"` // pembanding: PPh final UMKM WP OP atas omzet yang sama
	Boleh    bool      `json:"boleh"`     // omzet di bawah batas NPPN
	Catatan  []string  `json:"catatan"`
	Langkah  []Langkah `json:"langkah"`
}

var ErrNormaTidakAda = errors.New("norma untuk KLU dan wilayah ini tidak tersedia; isi persentase norma secara manual")

// HitungNPPN: penghasilan neto = omzet x norma (PER-17/PJ/2015), lalu Pasal 17 atas neto - PTKP.
func HitungNPPN(in NPPNInput) (NPPNResult, error) {
	pr := P()
	ptkp, ok := PTKPOf(in.PTKP)
	if !ok {
		return NPPNResult{}, fmt.Errorf("status PTKP %q tidak dikenal", in.PTKP)
	}
	if in.Omzet < 0 {
		return NPPNResult{}, errors.New("omzet tidak boleh negatif")
	}
	r := NPPNResult{PTKP: ptkp, NormaBps: in.NormaBps}
	if k := strings.TrimSpace(in.KLU); k != "" {
		n, ok := CariNorma(k)
		if !ok || n.Bps(in.Wilayah) < 0 {
			return NPPNResult{}, ErrNormaTidakAda
		}
		r.Norma, r.NormaBps = &n, n.Bps(in.Wilayah)
	}
	if r.NormaBps <= 0 || r.NormaBps > 10_000 {
		return NPPNResult{}, errors.New("persentase norma harus di antara 0 dan 100%")
	}
	r.Neto = mulDiv(in.Omzet, r.NormaBps, 10_000)
	r.PKP = max(0, r.Neto-ptkp) / 1000 * 1000
	r.PPh = Pasal17(r.PKP)
	r.PPhFinal = ppFinalOP(in.Omzet)
	r.Boleh = in.Omzet < pr.BatasNPPN
	r.Langkah = []Langkah{
		{"Peredaran bruto setahun", in.Omzet, ""},
		{"Penghasilan neto (norma " + persenBps(int(r.NormaBps)) + ")", r.Neto, "omzet x norma"},
		{"PTKP " + in.PTKP, -ptkp, ""},
		{"Penghasilan kena pajak", r.PKP, "dibulatkan ke bawah ke ribuan penuh"},
		{"PPh terutang setahun", r.PPh, "tarif progresif Pasal 17"},
	}
	r.Catatan = []string{"Norma wajib diberitahukan ke DJP dalam 3 bulan pertama tahun pajak; tanpa pemberitahuan dianggap memilih pembukuan."}
	if !r.Boleh {
		r.Catatan = append(r.Catatan, "Omzet "+rp(in.Omzet)+" tidak di bawah "+rp(pr.BatasNPPN)+": wajib pembukuan, norma tidak bisa dipakai.")
	}
	if in.Omzet <= pr.BatasOmzetUMKM {
		r.Catatan = append(r.Catatan, fmt.Sprintf("Pembanding: PPh final UMKM (%s) atas omzet yang sama %s.", pr.DasarUMKM, rp(r.PPhFinal)))
	}
	return r, nil
}

// ===== Gross up =====

// JenisPPh21BukanPegawai: gross up honorarium orang pribadi bukan pegawai (DPP 50% bruto).
const JenisPPh21BukanPegawai = "pph21-bukan-pegawai"

type GrossUpInput struct {
	Jenis     string `json:"jenis"` // id JenisPajak pemotongan, atau JenisPPh21BukanPegawai
	Objek     string `json:"objek"`
	Diterima  int64  `json:"diterima"` // yang harus diterima bersih oleh lawan transaksi
	PunyaNPWP bool   `json:"punya_npwp"`
}

type GrossUpResult struct {
	NilaiKontrak    int64     `json:"nilai_kontrak"`
	DPP             int64     `json:"dpp"`
	Pajak           int64     `json:"pajak"`
	Diterima        int64     `json:"diterima"`
	TarifEfektifBps int64     `json:"tarif_efektif_bps"` // pajak / nilai kontrak
	Final           bool      `json:"final"`
	Langkah         []Langkah `json:"langkah"`
}

// pajakGrossUp: pajak yang dipotong dari nilai kontrak k.
func pajakGrossUp(in GrossUpInput, k int64) (int64, error) {
	if in.Jenis == JenisPPh21BukanPegawai {
		pr := P()
		pph := Pasal17(mulDiv(k, pr.DasarBukanPegawaiBps, 10_000))
		if !in.PunyaNPWP {
			pph = pph * pr.TanpaNPWPPPh21Bps / 10_000
		}
		return pph, nil
	}
	r, err := HitungPotong(PotongInput{Jenis: in.Jenis, Objek: in.Objek, NilaiBruto: k, PunyaNPWP: in.PunyaNPWP})
	return r.Pajak, err
}

// HitungGrossUp mencari nilai kontrak terkecil yang membuat lawan transaksi tetap menerima
// Diterima setelah dipotong pajak. Dicari dengan bagi dua, bukan rumus x/(1-tarif), supaya
// tarif progresif (PPh 21) dan pembulatan ke bawah ikut terhitung persis.
func HitungGrossUp(in GrossUpInput) (GrossUpResult, error) {
	switch {
	case in.Diterima < 0:
		return GrossUpResult{}, errors.New("nilai yang diterima tidak boleh negatif")
	case in.Jenis == "ppn" || in.Jenis == "ppnbm":
		return GrossUpResult{}, errors.New("PPN dan PPnBM dipungut di atas harga, bukan dipotong; pakai kalkulator PPN & harga jual")
	}
	if _, err := pajakGrossUp(in, 0); err != nil {
		return GrossUpResult{}, err
	}
	bersih := func(k int64) int64 { p, _ := pajakGrossUp(in, k); return k - p }
	// batas atas: tarif efektif tertinggi yang mungkin (35% x 120%, 15% x 200%) < 50%
	lo, hi := in.Diterima, in.Diterima*2+1000
	for bersih(hi) < in.Diterima {
		hi *= 2
	}
	for lo < hi {
		mid := lo + (hi-lo)/2
		if bersih(mid) >= in.Diterima {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	r := GrossUpResult{NilaiKontrak: lo}
	r.Pajak, _ = pajakGrossUp(in, lo)
	r.Diterima = lo - r.Pajak
	if lo > 0 {
		r.TarifEfektifBps = mulDiv(r.Pajak, 10_000, lo)
	}
	nama := "PPh 21 bukan pegawai"
	if in.Jenis == JenisPPh21BukanPegawai {
		r.DPP = mulDiv(lo, P().DasarBukanPegawaiBps, 10_000)
	} else {
		p, _ := HitungPotong(PotongInput{Jenis: in.Jenis, Objek: in.Objek, NilaiBruto: lo, PunyaNPWP: in.PunyaNPWP})
		r.DPP, r.Final, nama = p.DPP, p.Final, p.JenisNama+" — "+p.ObjekNama
	}
	r.Langkah = []Langkah{
		{"Yang harus diterima bersih", in.Diterima, ""},
		{"Nilai kontrak / invoice", r.NilaiKontrak, "nilai terkecil yang menutup pajaknya sendiri"},
		{"Dasar pengenaan pajak", r.DPP, ""},
		{nama, -r.Pajak, "tarif efektif " + persenBps(int(r.TarifEfektifBps))},
		{"Diterima setelah dipotong", r.Diterima, ""},
	}
	return r, nil
}

// ===== PPN & harga jual =====

const (
	SkemaNonPKP        = "non_pkp"
	SkemaPKPDibebankan = "pkp_dibebankan" // PPN ditambahkan di atas harga
	SkemaPKPDitanggung = "pkp_ditanggung" // harga ke pembeli sudah termasuk PPN
)

type PPNHargaInput struct {
	Pembelian       int64 `json:"pembelian"`
	TermasukPPN     bool  `json:"termasuk_ppn"`       // nilai pembelian sudah termasuk PPN
	PorsiDariPKPBps int64 `json:"porsi_dari_pkp_bps"` // bagian pembelian dari PKP (ada PPN masukan)
	MarkupBps       int64 `json:"markup_bps"`
	OpexBps         int64 `json:"opex_bps"`     // persen dari penjualan (DPP)
	OpexNominal     int64 `json:"opex_nominal"` // dipakai bila > 0, menggantikan OpexBps
}

type SkemaPPN struct {
	ID           string `json:"id"`
	Nama         string `json:"nama"`
	Keterangan   string `json:"keterangan"`
	HargaPembeli int64  `json:"harga_pembeli"` // yang dibayar pembeli
	DPP          int64  `json:"dpp"`           // penjualan bagi penjual
	PPNKeluaran  int64  `json:"ppn_keluaran"`
	PPNMasukan   int64  `json:"ppn_masukan"` // yang bisa dikreditkan
	PPNSetor     int64  `json:"ppn_setor"`   // negatif = lebih bayar
	HPP          int64  `json:"hpp"`
	Opex         int64  `json:"opex"`
	Laba         int64  `json:"laba"`
}

type PPNHargaResult struct {
	TarifEfektifBps int64      `json:"tarif_efektif_bps"`
	Skema           []SkemaPPN `json:"skema"`
	// KenaikanHargaBps: kenaikan harga yang dibayar pembeli bila penjual jadi PKP dan
	// membebankan PPN, dibanding saat belum PKP.
	KenaikanHargaBps int64    `json:"kenaikan_harga_bps"`
	LabaTertinggi    string   `json:"laba_tertinggi"`
	Catatan          []string `json:"catatan"`
}

// HitungPPNHarga menggabungkan sheet "PPN Dibebankan vs Ditanggung" dan "Simulasi Pricing PPN":
// harga & laba bila belum PKP, PKP yang menambahkan PPN di atas harga, dan PKP yang menanggung
// PPN di dalam harga yang sama.
func HitungPPNHarga(in PPNHargaInput) (PPNHargaResult, error) {
	if in.Pembelian < 0 || in.MarkupBps < 0 || in.OpexBps < 0 || in.OpexNominal < 0 || in.PorsiDariPKPBps < 0 || in.PorsiDariPKPBps > 10_000 {
		return PPNHargaResult{}, errors.New("nilai tidak boleh negatif dan porsi dari PKP 0–100%")
	}
	pr := P()
	pemb, peny := pr.PPNTarifBps*pr.PPNDPPPembilang, 10_000*pr.PPNDPPPenyebut
	dariPKP := mulDiv(in.Pembelian, in.PorsiDariPKPBps, 10_000)
	var masukan, biaya int64 // biaya = pembelian tanpa PPN
	if in.TermasukPPN {
		masukan = mulDiv(dariPKP, pemb, peny+pemb)
		biaya = in.Pembelian - masukan
	} else {
		masukan = mulDiv(dariPKP, pemb, peny)
		biaya = in.Pembelian
	}
	opex := func(penjualan int64) int64 {
		if in.OpexNominal > 0 {
			return in.OpexNominal
		}
		return mulDiv(penjualan, in.OpexBps, 10_000)
	}
	naik := func(x int64) int64 { return x + mulDiv(x, in.MarkupBps, 10_000) }

	non := SkemaPPN{ID: SkemaNonPKP, Nama: "Belum PKP", Keterangan: "Tidak memungut PPN; PPN masukan tidak bisa dikreditkan, jadi ikut menjadi HPP.",
		HPP: biaya + masukan}
	non.DPP = naik(non.HPP)
	non.HargaPembeli = non.DPP

	beban := SkemaPPN{ID: SkemaPKPDibebankan, Nama: "PKP, PPN dibebankan ke pembeli", Keterangan: "Harga belum termasuk PPN; PPN ditambahkan di faktur.",
		HPP: biaya, PPNMasukan: masukan}
	beban.DPP = naik(biaya)
	beban.PPNKeluaran = pr.PPN(beban.DPP)
	beban.HargaPembeli = beban.DPP + beban.PPNKeluaran

	tanggung := SkemaPPN{ID: SkemaPKPDitanggung, Nama: "PKP, PPN ditanggung penjual", Keterangan: "Harga ke pembeli tetap, sudah termasuk PPN.",
		HPP: biaya, PPNMasukan: masukan, HargaPembeli: naik(biaya)}
	tanggung.DPP = mulDiv(tanggung.HargaPembeli, peny, peny+pemb)
	tanggung.PPNKeluaran = tanggung.HargaPembeli - tanggung.DPP

	r := PPNHargaResult{TarifEfektifBps: pr.PPNEfektifBps()}
	for _, s := range []*SkemaPPN{&non, &beban, &tanggung} {
		if s.ID != SkemaNonPKP {
			s.PPNSetor = s.PPNKeluaran - s.PPNMasukan
		}
		s.Opex = opex(s.DPP)
		s.Laba = s.DPP - s.HPP - s.Opex
		r.Skema = append(r.Skema, *s)
	}
	if non.HargaPembeli > 0 {
		r.KenaikanHargaBps = mulDiv(beban.HargaPembeli-non.HargaPembeli, 10_000, non.HargaPembeli)
	}
	r.LabaTertinggi = slices.MaxFunc(r.Skema, func(a, b SkemaPPN) int { return int(a.Laba - b.Laba) }).ID
	r.Catatan = []string{
		fmt.Sprintf("PPN %s x DPP %d/%d = efektif %s dari harga jual.", persenBps(int(pr.PPNTarifBps)), pr.PPNDPPPembilang, pr.PPNDPPPenyebut, persenBps(int(r.TarifEfektifBps))),
		"Omzet di atas " + rp(pr.BatasPKP) + " setahun wajib PKP; skema belum PKP hanya berlaku di bawah batas itu.",
	}
	return r, nil
}

// ===== Beli vs sewa aset =====

const (
	MetodeGarisLurus   = "garis_lurus"
	MetodeSaldoMenurun = "saldo_menurun"
)

type BeliSewaInput struct {
	Bentuk       string `json:"bentuk"` // badan | op
	PTKP         string `json:"ptkp"`   // untuk op
	Omzet        int64  `json:"omzet"`
	BiayaLain    int64  `json:"biaya_lain"` // HPP + biaya operasional di luar penyusutan / sewa, setahun
	HargaAset    int64  `json:"harga_aset"`
	UmurTahun    int    `json:"umur_tahun"`
	Metode       string `json:"metode"`
	Bangunan     bool   `json:"bangunan"`
	SewaPerTahun int64  `json:"sewa_per_tahun"`
}

type TahunAset struct {
	Tahun int   `json:"tahun"`
	Beban int64 `json:"beban"` // penyusutan atau sewa
	Laba  int64 `json:"laba"`
	PPh   int64 `json:"pph"`
}

type OpsiAset struct {
	Tahun    []TahunAset `json:"tahun"`
	TotalPPh int64       `json:"total_pph"`
	// TotalKas: uang keluar selama umur aset — harga aset atau sewa — ditambah PPh.
	TotalKas int64 `json:"total_kas"`
}

type BeliSewaResult struct {
	Metode      string   `json:"metode"` // bisa berubah jadi garis lurus untuk bangunan
	Beli        OpsiAset `json:"beli"`
	Sewa        OpsiAset `json:"sewa"`
	PotongSewa  int64    `json:"potong_sewa"` // PPh 23 / 4(2) atas sewa setahun
	Rekomendasi string   `json:"rekomendasi"` // beli | sewa
	Selisih     int64    `json:"selisih"`
	Catatan     []string `json:"catatan"`
}

// HitungBeliSewa membandingkan beli aset (disusutkan) dengan menyewa selama umur ekonomisnya.
// Workbook hanya membandingkan PPh tahun pertama; di sini ditambah total kas keluar, karena
// pajak yang sedikit lebih rendah belum tentu sepadan dengan harga aset atau sewanya.
func HitungBeliSewa(in BeliSewaInput) (BeliSewaResult, error) {
	switch {
	case in.UmurTahun <= 0 || in.UmurTahun > 50:
		return BeliSewaResult{}, errors.New("umur ekonomis 1–50 tahun")
	case in.Omzet < 0 || in.BiayaLain < 0 || in.HargaAset < 0 || in.SewaPerTahun < 0:
		return BeliSewaResult{}, errors.New("nilai tidak boleh negatif")
	case in.Metode != MetodeGarisLurus && in.Metode != MetodeSaldoMenurun:
		return BeliSewaResult{}, errors.New("metode penyusutan harus garis_lurus atau saldo_menurun")
	case in.Bentuk != "badan" && in.Bentuk != "op":
		return BeliSewaResult{}, errors.New("bentuk harus badan atau op")
	}
	ptkp := int64(0)
	if in.Bentuk == "op" {
		var ok bool
		if ptkp, ok = PTKPOf(in.PTKP); !ok {
			return BeliSewaResult{}, fmt.Errorf("status PTKP %q tidak dikenal", in.PTKP)
		}
	}
	pph := func(laba int64) int64 {
		if in.Bentuk == "badan" {
			return Pasal31E(in.Omzet, laba)
		}
		return Pasal17(max(0, laba-ptkp) / 1000 * 1000)
	}
	r := BeliSewaResult{Metode: in.Metode}
	if in.Bangunan {
		r.Metode = MetodeGarisLurus
	}
	sisa := in.HargaAset
	for y := 1; y <= in.UmurTahun; y++ {
		var d int64
		switch {
		case r.Metode == MetodeGarisLurus && y == in.UmurTahun:
			d = sisa
		case r.Metode == MetodeGarisLurus:
			d = in.HargaAset / int64(in.UmurTahun)
		case y == in.UmurTahun: // saldo menurun: sisa nilai buku habis di tahun terakhir
			d = sisa
		default:
			d = sisa * 2 / int64(in.UmurTahun)
		}
		sisa -= d
		for _, o := range []struct {
			opsi  *OpsiAset
			beban int64
		}{{&r.Beli, d}, {&r.Sewa, in.SewaPerTahun}} {
			laba := in.Omzet - in.BiayaLain - o.beban
			t := TahunAset{Tahun: y, Beban: o.beban, Laba: laba, PPh: pph(laba)}
			o.opsi.Tahun = append(o.opsi.Tahun, t)
			o.opsi.TotalPPh += t.PPh
		}
	}
	r.Beli.TotalKas = in.HargaAset + r.Beli.TotalPPh
	r.Sewa.TotalKas = in.SewaPerTahun*int64(in.UmurTahun) + r.Sewa.TotalPPh

	objek := "23-sewa"
	if in.Bangunan {
		objek = "42-sewa-tb"
	}
	jenis := map[string]string{"23-sewa": "pph23", "42-sewa-tb": "pph4-2"}[objek]
	if p, err := HitungPotong(PotongInput{Jenis: jenis, Objek: objek, NilaiBruto: in.SewaPerTahun, PunyaNPWP: true}); err == nil {
		r.PotongSewa = p.Pajak
	}
	r.Rekomendasi, r.Selisih = "beli", r.Sewa.TotalKas-r.Beli.TotalKas
	if r.Sewa.TotalKas < r.Beli.TotalKas {
		r.Rekomendasi, r.Selisih = "sewa", r.Beli.TotalKas-r.Sewa.TotalKas
	}
	r.Catatan = []string{
		"PPh atas sewa (" + rp(r.PotongSewa) + " setahun) dipotong dari pembayaran ke pemilik aset dan disetor penyewa; bukan biaya tambahan, kecuali disepakati ditanggung penyewa (gross up).",
		"Perbandingan tanpa nilai waktu uang, nilai sisa aset, biaya perawatan, dan bunga pinjaman.",
	}
	if in.Bangunan && in.Metode == MetodeSaldoMenurun {
		r.Catatan = append(r.Catatan, "Bangunan hanya boleh disusutkan dengan metode garis lurus; perhitungan memakai garis lurus.")
	}
	return r, nil
}
