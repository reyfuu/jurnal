package tax

import (
	"fmt"
	"slices"
	"strings"
)

// Objek pajak untuk kalkulator potong-pungut. Satu objek = satu baris tarif yang bisa dipilih
// pengguna, meniru susunan kalkulator.pajak.go.id.
//
// === TARIF OBJEK BAWAAN ===
// Daftar ini bawaan versi 1; tarif yang berlaku diubah tim di Panel Master → Parameter Pajak.
// Tarif ditulis dalam basis poin: 1500 = 15%, 250 = 2,5%, 50 = 0,5%.
// DasarBps mengatur berapa persen dari nilai bruto yang jadi dasar pengenaan; 10000 = 100%.
// Kolom Dasar menyebut rujukan aturannya supaya mudah dicek ulang.
//
// Tarif di bawah disusun dari ketentuan yang berlaku umum. SEBELUM DIPAKAI UNTUK SPT, minta tim
// pajak memverifikasi baris yang relevan — tarif bisa berubah oleh PMK baru, dan sebagian objek
// (khususnya PPnBM) bergantung pada klasifikasi barang yang tidak bisa ditebak dari nominal.

type ObjekPajak struct {
	Kode     string `json:"kode"`
	Nama     string `json:"nama"`
	TarifBps int    `json:"tarif_bps"`
	// DasarBps: bagian dari nilai bruto yang jadi DPP. 10000 = seluruh nilai bruto.
	DasarBps int    `json:"dasar_bps"`
	Final    bool   `json:"final"`
	Dasar    string `json:"dasar"`             // rujukan aturan
	Catatan  string `json:"catatan,omitempty"` // hal yang perlu diperhatikan pengguna
}

type JenisPajak struct {
	ID    string       `json:"id"`
	Nama  string       `json:"nama"`
	Ket   string       `json:"keterangan"`
	Objek []ObjekPajak `json:"objek"`
	// TanpaNPWPBps: pengali tarif bila lawan transaksi tidak ber-NPWP. 0 = tidak berlaku.
	TanpaNPWPBps int `json:"tanpa_npwp_bps,omitempty"`
}

func o(kode, nama string, tarifBps int, final bool, dasar, catatan string) ObjekPajak {
	return ObjekPajak{Kode: kode, Nama: nama, TarifBps: tarifBps, DasarBps: 10_000, Final: final, Dasar: dasar, Catatan: catatan}
}

// oDPP: objek yang DPP-nya bukan seluruh nilai bruto.
func oDPP(kode, nama string, tarifBps, dasarBps int, final bool, dasar, catatan string) ObjekPajak {
	x := o(kode, nama, tarifBps, final, dasar, catatan)
	x.DasarBps = dasarBps
	return x
}

var JenisPajakList = []JenisPajak{
	{
		ID: "pph23", Nama: "PPh Pasal 23", TanpaNPWPBps: 20_000,
		Ket: "Pemotongan atas dividen, bunga, royalti, hadiah, sewa selain tanah/bangunan, dan imbalan jasa.",
		Objek: []ObjekPajak{
			o("23-dividen", "Dividen", 1500, false, "UU PPh Pasal 23 ayat (1)", "Dividen ke WP OP dalam negeri dikenakan PPh 4(2), bukan pasal ini."),
			o("23-bunga", "Bunga, premium, diskonto", 1500, false, "UU PPh Pasal 23 ayat (1)", ""),
			o("23-royalti", "Royalti", 1500, false, "UU PPh Pasal 23 ayat (1)", ""),
			o("23-hadiah", "Hadiah & penghargaan", 1500, false, "UU PPh Pasal 23 ayat (1)", "Hadiah undian masuk PPh 4(2) 25%."),
			o("23-sewa", "Sewa harta selain tanah & bangunan", 200, false, "UU PPh Pasal 23 ayat (1)", "Sewa tanah/bangunan memakai PPh 4(2) 10%."),
			o("23-jasa", "Imbalan jasa teknik, manajemen, konsultan & jasa lain", 200, false, "PMK 141/PMK.03/2015", "Berlaku untuk 62 jenis jasa dalam lampiran PMK."),
		},
	},
	{
		ID: "pph4-2", Nama: "PPh Pasal 4 ayat (2)",
		Ket: "Pajak penghasilan yang bersifat final: sewa tanah/bangunan, jasa konstruksi, pengalihan hak atas tanah, bunga, hadiah undian, dan UMKM.",
		Objek: []ObjekPajak{
			o("42-sewa-tb", "Sewa tanah dan/atau bangunan", 1000, true, "PP 34/2017", ""),
			o("42-pengalihan", "Pengalihan hak atas tanah/bangunan", 250, true, "PP 34/2016", "1% untuk rumah sederhana & rumah susun sederhana oleh WP yang usaha pokoknya pengalihan."),
			o("42-pengalihan-rs", "Pengalihan hak atas RS / RSS", 100, true, "PP 34/2016", "Hanya untuk WP yang usaha pokoknya melakukan pengalihan hak."),
			o("42-konstruksi-kecil", "Jasa konstruksi — pelaksanaan, kualifikasi kecil", 175, true, "PP 9/2022", ""),
			o("42-konstruksi-menengah", "Jasa konstruksi — pelaksanaan, kualifikasi menengah/besar", 265, true, "PP 9/2022", ""),
			o("42-konstruksi-tanpa", "Jasa konstruksi — pelaksanaan, tanpa kualifikasi", 400, true, "PP 9/2022", ""),
			o("42-konsultansi", "Jasa konstruksi — perancangan/pengawasan, berkualifikasi", 350, true, "PP 9/2022", ""),
			o("42-konsultansi-tanpa", "Jasa konstruksi — perancangan/pengawasan, tanpa kualifikasi", 600, true, "PP 9/2022", ""),
			o("42-deposito", "Bunga deposito & tabungan", 2000, true, "PP 131/2000", ""),
			o("42-obligasi", "Bunga obligasi", 1000, true, "PP 91/2021", ""),
			o("42-undian", "Hadiah undian", 2500, true, "PP 132/2000", ""),
			o("42-umkm", "Peredaran bruto UMKM", 50, true, "PP 55/2022", "WP OP bebas PPh final atas omzet Rp 500 juta pertama setahun."),
		},
	},
	{
		ID: "pph22", Nama: "PPh Pasal 22", TanpaNPWPBps: 20_000,
		Ket: "Pemungutan atas impor dan pembelian barang tertentu.",
		Objek: []ObjekPajak{
			o("22-impor-api", "Impor barang — menggunakan API", 250, false, "PMK 41/PMK.010/2022", ""),
			o("22-impor-nonapi", "Impor barang — tanpa API", 750, false, "PMK 41/PMK.010/2022", ""),
			o("22-impor-kedelai", "Impor kedelai, gandum, tepung terigu — dengan API", 50, false, "PMK 41/PMK.010/2022", ""),
			o("22-bendahara", "Pembelian barang oleh bendahara pemerintah", 150, false, "PMK 34/PMK.010/2017", ""),
			o("22-bbm-spbu", "Penjualan BBM ke SPBU Pertamina", 25, true, "PMK 34/PMK.010/2017", "Final bagi penyalur/agen."),
			o("22-bbm-swasta", "Penjualan BBM ke SPBU swasta & pihak lain", 30, true, "PMK 34/PMK.010/2017", "Final bagi penyalur/agen."),
			o("22-mewah", "Penjualan barang sangat mewah", 500, false, "PMK 92/PMK.03/2019", ""),
		},
	},
	{
		ID: "pph15", Nama: "PPh Pasal 15",
		Ket: "Norma penghitungan khusus untuk pelayaran, penerbangan, dan kantor perwakilan dagang asing.",
		Objek: []ObjekPajak{
			o("15-pelayaran-dn", "Pelayaran dalam negeri", 120, true, "KMK 416/KMK.04/1996", "1,2% dari peredaran bruto, bersifat final."),
			o("15-penerbangan-dn", "Penerbangan dalam negeri", 180, false, "KMK 475/KMK.04/1996", "1,8% dari peredaran bruto; merupakan kredit pajak, bukan final."),
			o("15-pelayaran-ln", "Pelayaran/penerbangan luar negeri", 264, true, "KMK 417/KMK.04/1996", "2,64% dari peredaran bruto, bersifat final."),
			o("15-kpd", "Kantor perwakilan dagang asing", 44, true, "KEP-667/PJ./2001", "0,44% dari nilai ekspor bruto, bersifat final."),
		},
	},
	{
		ID: "ppn", Nama: "PPN",
		Ket: "Pajak Pertambahan Nilai atas penyerahan barang dan jasa kena pajak.",
		Objek: []ObjekPajak{
			oDPP("ppn-umum", "Penyerahan BKP/JKP — tarif umum", 1200, 9166, false, "UU HPP jo. PMK 131/2024",
				"Tarif 12% dikenakan atas DPP nilai lain 11/12 dari harga jual, sehingga bebannya setara 11%."),
			o("ppn-mewah", "Barang mewah yang terkena PPnBM", 1200, false, "PMK 131/2024", "Dikenakan 12% penuh atas harga jual."),
			o("ppn-membangun", "Kegiatan membangun sendiri", 240, true, "PMK 61/2022", "2,4% dari biaya pembangunan (20% x tarif)."),
		},
	},
	{
		ID: "ppnbm", Nama: "PPnBM",
		Ket: "Pajak Penjualan atas Barang Mewah. Tarif bergantung klasifikasi barang, bukan nominalnya — pilih lapisan sesuai kelompok barang di PMK.",
		Objek: []ObjekPajak{
			o("ppnbm-10", "Lapisan 10%", 1000, false, "PP 61/2020 / PMK 141/2021", ""),
			o("ppnbm-20", "Lapisan 20%", 2000, false, "PP 61/2020 / PMK 141/2021", ""),
			o("ppnbm-30", "Lapisan 30%", 3000, false, "PP 61/2020 / PMK 141/2021", ""),
			o("ppnbm-40", "Lapisan 40%", 4000, false, "PP 61/2020 / PMK 141/2021", ""),
			o("ppnbm-50", "Lapisan 50%", 5000, false, "PP 61/2020 / PMK 141/2021", ""),
			o("ppnbm-60", "Lapisan 60%", 6000, false, "PP 61/2020 / PMK 141/2021", ""),
			o("ppnbm-75", "Lapisan 75%", 7500, false, "PP 61/2020 / PMK 141/2021", ""),
			o("ppnbm-125", "Lapisan 125%", 12500, false, "PMK 141/2021", "Kendaraan bermotor tertentu."),
			o("ppnbm-200", "Lapisan 200%", 20000, false, "PP 61/2020", ""),
		},
	},
}

func JenisPajakOf(id string) (JenisPajak, bool) {
	js := P().Objek
	i := slices.IndexFunc(js, func(j JenisPajak) bool { return j.ID == id })
	if i < 0 {
		return JenisPajak{}, false
	}
	return js[i], true
}

// PotongInput: perhitungan satu pemotongan/pemungutan.
type PotongInput struct {
	Jenis      string `json:"jenis"` // id JenisPajak
	Objek      string `json:"objek"` // kode ObjekPajak
	NilaiBruto int64  `json:"nilai_bruto"`
	PunyaNPWP  bool   `json:"punya_npwp"`
}

type PotongResult struct {
	JenisNama         string    `json:"jenis_nama"`
	ObjekNama         string    `json:"objek_nama"`
	NilaiBruto        int64     `json:"nilai_bruto"`
	DPP               int64     `json:"dpp"`
	TarifBps          int       `json:"tarif_bps"`
	Pajak             int64     `json:"pajak"`
	Diterima          int64     `json:"diterima"` // bruto dikurangi pajak yang dipotong
	Final             bool      `json:"final"`
	Dasar             string    `json:"dasar"`
	Catatan           string    `json:"catatan,omitempty"`
	TambahanTanpaNPWP int64     `json:"tambahan_tanpa_npwp"`
	Langkah           []Langkah `json:"langkah"`
}

func HitungPotong(in PotongInput) (PotongResult, error) {
	j, ok := JenisPajakOf(strings.TrimSpace(in.Jenis))
	if !ok {
		return PotongResult{}, fmt.Errorf("jenis pajak %q tidak dikenal", in.Jenis)
	}
	i := slices.IndexFunc(j.Objek, func(x ObjekPajak) bool { return x.Kode == strings.TrimSpace(in.Objek) })
	if i < 0 {
		return PotongResult{}, fmt.Errorf("objek pajak %q tidak ada pada %s", in.Objek, j.Nama)
	}
	ob := j.Objek[i]
	if in.NilaiBruto < 0 {
		return PotongResult{}, fmt.Errorf("nilai bruto tidak boleh negatif")
	}

	r := PotongResult{
		JenisNama: j.Nama, ObjekNama: ob.Nama, NilaiBruto: in.NilaiBruto,
		TarifBps: ob.TarifBps, Final: ob.Final, Dasar: ob.Dasar, Catatan: ob.Catatan,
	}
	r.DPP = in.NilaiBruto * int64(ob.DasarBps) / 10_000
	r.Pajak = r.DPP * int64(ob.TarifBps) / 10_000

	// Tanpa NPWP dipotong lebih tinggi; hanya berlaku pada jenis yang mengenal aturan ini.
	if !in.PunyaNPWP && j.TanpaNPWPBps > 0 && r.Pajak > 0 {
		normal := r.Pajak
		r.Pajak = normal * int64(j.TanpaNPWPBps) / 10_000
		r.TambahanTanpaNPWP = r.Pajak - normal
		r.TarifBps = ob.TarifBps * j.TanpaNPWPBps / 10_000
	}
	r.Diterima = in.NilaiBruto - r.Pajak

	r.Langkah = []Langkah{{"Nilai bruto", in.NilaiBruto, ""}}
	if ob.DasarBps != 10_000 {
		r.Langkah = append(r.Langkah, Langkah{"Dasar pengenaan pajak", r.DPP,
			fmt.Sprintf("%s dari nilai bruto", persenBps(ob.DasarBps))})
	}
	r.Langkah = append(r.Langkah,
		Langkah{"Tarif " + persenBps(r.TarifBps), r.Pajak, ketTanpaNPWP(in.PunyaNPWP, j)},
		Langkah{"Diterima setelah potong", r.Diterima, ""},
	)
	return r, nil
}

func ketTanpaNPWP(punya bool, j JenisPajak) string {
	if punya || j.TanpaNPWPBps == 0 {
		return j.Nama
	}
	return fmt.Sprintf("tarif %dx lebih tinggi karena tanpa NPWP", j.TanpaNPWPBps/10_000)
}

// persenBps: 1500 → "15%", 175 → "1,75%", 9166 → "91,66%".
func persenBps(bps int) string {
	if bps%100 == 0 {
		return fmt.Sprintf("%d%%", bps/100)
	}
	s := fmt.Sprintf("%.2f", float64(bps)/100)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	return strings.Replace(s, ".", ",", 1) + "%"
}

// ===== PPh Badan =====

type PPhBadanInput struct {
	PeredaranBruto       int64 `json:"peredaran_bruto"` // setahun
	PenghasilanKenaPajak int64 `json:"penghasilan_kena_pajak"`
}

type PPhBadanResult struct {
	PeredaranBruto int64     `json:"peredaran_bruto"`
	PKP            int64     `json:"pkp"`
	TarifBps       int       `json:"tarif_bps"`
	DapatFasilitas bool      `json:"dapat_fasilitas"`
	PPhTerutang    int64     `json:"pph_terutang"`
	Langkah        []Langkah `json:"langkah"`
	Catatan        string    `json:"catatan"`
}

// HitungPPhBadan memakai tarif badan dengan fasilitas Pasal 31E: pengurangan tarif atas bagian
// PKP dari peredaran bruto sampai Batas31EPenuh, selama peredaran bruto setahun tidak melebihi
// Batas31E (bawaan 22%, 50%, Rp4,8 M, Rp50 M).
func HitungPPhBadan(in PPhBadanInput) (PPhBadanResult, error) {
	if in.PenghasilanKenaPajak > in.PeredaranBruto && in.PeredaranBruto >= 0 {
		return PPhBadanResult{}, fmt.Errorf("penghasilan kena pajak (%s) tidak boleh lebih besar dari peredaran bruto (%s): laba tidak mungkin melebihi omzet", rp(in.PenghasilanKenaPajak), rp(in.PeredaranBruto))
	}
	if in.PeredaranBruto < 0 || in.PenghasilanKenaPajak < 0 {
		return PPhBadanResult{}, fmt.Errorf("nilai tidak boleh negatif")
	}
	pr := P()
	r := PPhBadanResult{
		PeredaranBruto: in.PeredaranBruto, PKP: in.PenghasilanKenaPajak,
		TarifBps: int(pr.BadanTarifBps), DapatFasilitas: in.PeredaranBruto <= pr.Batas31E,
	}
	badan, efektif := persenBps(int(pr.BadanTarifBps)), persenBps(int(pr.BadanTarifBps*(10_000-pr.Fasilitas31EBps)/10_000))
	r.PPhTerutang = Pasal31E(in.PeredaranBruto, in.PenghasilanKenaPajak)

	r.Langkah = []Langkah{
		{"Peredaran bruto setahun", in.PeredaranBruto, ""},
		{"Penghasilan kena pajak", in.PenghasilanKenaPajak, ""},
		{"PPh badan terutang", r.PPhTerutang, "tarif " + badan + " (UU HPP Pasal 17)"},
	}
	switch {
	case in.PeredaranBruto > pr.Batas31E:
		r.Catatan = fmt.Sprintf("Peredaran bruto di atas %s: tanpa fasilitas Pasal 31E, seluruh PKP dikenakan %s.", rp(pr.Batas31E), badan)
	case in.PeredaranBruto <= pr.Batas31EPenuh:
		r.Catatan = fmt.Sprintf("Peredaran bruto sampai %s: seluruh PKP mendapat pengurangan tarif %s menjadi efektif %s.", rp(pr.Batas31EPenuh), persenBps(int(pr.Fasilitas31EBps)), efektif)
	default:
		r.Catatan = fmt.Sprintf("Bagian PKP dari peredaran bruto sampai %s mendapat pengurangan %s; sisanya dikenakan %s penuh.", rp(pr.Batas31EPenuh), persenBps(int(pr.Fasilitas31EBps)), badan)
	}
	return r, nil
}
