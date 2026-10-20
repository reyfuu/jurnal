package tax

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync/atomic"
)

// Parameter: semua angka aturan pajak yang dipakai kalkulator, perencanaan pajak, dan laporan
// PPh final. Versi aktif dikelola tim Master Pajak di Panel Master (katalog, draft → aktif →
// arsip); bawaan = ParameterBawaan(). Nilai Rupiah bulat, tarif dalam basis poin.
type Parameter struct {
	PTKP    map[string]int64 `json:"ptkp"`
	Pasal17 []Lapis          `json:"pasal17"`

	BiayaJabatanBps   int64 `json:"biaya_jabatan_bps"`
	BiayaJabatanMaks  int64 `json:"biaya_jabatan_maks"`   // setahun
	TanpaNPWPPPh21Bps int64 `json:"tanpa_npwp_pph21_bps"` // 12000 = 120% dari tarif normal
	// DasarBukanPegawaiBps: PPh 21 bukan pegawai dikenakan atas 50% penghasilan bruto.
	DasarBukanPegawaiBps int64    `json:"dasar_bukan_pegawai_bps"`
	TER                  TERParam `json:"ter"`

	UMKMTarifBps   int64  `json:"umkm_tarif_bps"`
	UMKMBebasOP    int64  `json:"umkm_bebas_op"`    // omzet setahun yang tidak dikenai PPh final, WP OP
	BatasOmzetUMKM int64  `json:"batas_omzet_umkm"` // omzet setahun maksimal untuk PPh final
	DasarUMKM      string `json:"dasar_umkm"`       // rujukan aturan yang tampil di layar

	BadanTarifBps   int64 `json:"badan_tarif_bps"`
	Fasilitas31EBps int64 `json:"fasilitas_31e_bps"` // pengurangan tarif Pasal 31E
	Batas31EPenuh   int64 `json:"batas_31e_penuh"`   // omzet yang porsinya mendapat fasilitas
	Batas31E        int64 `json:"batas_31e"`         // di atas ini tanpa fasilitas
	DividenBps      int64 `json:"dividen_bps"`

	BatasPKP        int64 `json:"batas_pkp"`
	PPNTarifBps     int64 `json:"ppn_tarif_bps"`
	PPNDPPPembilang int64 `json:"ppn_dpp_pembilang"` // DPP nilai lain 11/12
	PPNDPPPenyebut  int64 `json:"ppn_dpp_penyebut"`

	BatasNPPN int64 `json:"batas_nppn"` // omzet setahun di bawah ini boleh memakai norma (NPPN)
	// NormaPerbandinganBps: norma untuk skema NPPN di perbandingan bentuk usaha, yang tidak
	// tahu KLU 5 digit usaha. Kalkulator NPPN memakai Norma per KLU.
	NormaPerbandinganBps int64        `json:"norma_perbandingan_bps"`
	Objek                []JenisPajak `json:"objek"`
	Norma                []Norma      `json:"norma"`
	// PerluVerifikasi: hal yang belum dipastikan tim pajak; tampil di kalkulator.
	PerluVerifikasi []string `json:"perlu_verifikasi"`
}

// Lapis: satu lapisan tarif progresif. Sampai 0 = tanpa batas (hanya lapisan terakhir).
type Lapis struct {
	Sampai   int64 `json:"sampai"`
	TarifBps int64 `json:"tarif_bps"`
}

type TERParam struct {
	Kategori map[string]string     `json:"kategori"`
	Lapisan  map[string][]TERBaris `json:"lapisan"`
}

const (
	WilayahIbukota10   = "ibukota10"    // Medan, Palembang, Jakarta, Bandung, Semarang, Surabaya, Denpasar, Manado, Makassar, Pontianak
	WilayahIbukotaLain = "ibukota_lain" // ibukota provinsi lainnya
	WilayahDaerahLain  = "daerah_lain"
)

// Norma penghitungan penghasilan neto per KLU 5 digit dan wilayah. -1 = tidak tersedia.
type Norma struct {
	KLU            string `json:"klu"`
	Uraian         string `json:"uraian"`
	Ibukota10Bps   int64  `json:"ibukota10_bps"`
	IbukotaLainBps int64  `json:"ibukota_lain_bps"`
	DaerahLainBps  int64  `json:"daerah_lain_bps"`
}

func (n Norma) Bps(wilayah string) int64 {
	switch wilayah {
	case WilayahIbukota10:
		return n.Ibukota10Bps
	case WilayahIbukotaLain:
		return n.IbukotaLainBps
	case WilayahDaerahLain:
		return n.DaerahLainBps
	}
	return -1
}

var pasal17Bawaan = []Lapis{{60_000_000, 500}, {250_000_000, 1500}, {500_000_000, 2500}, {5_000_000_000, 3000}, {0, 3500}}

// SumberParameterBawaan: dasar versi 1 di Panel Master.
const SumberParameterBawaan = "UU HPP, PP 55/2022, PMK 168/2023, PMK 131/2024, PER-17/PJ/2015"

// ParameterBawaan: angka di kode, salinan baru setiap dipanggil.
func ParameterBawaan() Parameter {
	return Parameter{
		PTKP: ptkpBawaan, Pasal17: pasal17Bawaan,
		BiayaJabatanBps: 500, BiayaJabatanMaks: 6_000_000, TanpaNPWPPPh21Bps: 12_000, DasarBukanPegawaiBps: 5000,
		TER:          TERParam{Kategori: TERKategori, Lapisan: TERLapisan},
		UMKMTarifBps: 50, UMKMBebasOP: 500_000_000, BatasOmzetUMKM: 4_800_000_000, DasarUMKM: "PP 55/2022",
		BadanTarifBps: 2200, Fasilitas31EBps: 5000, Batas31EPenuh: 4_800_000_000, Batas31E: 50_000_000_000, DividenBps: 1000,
		BatasPKP: 4_800_000_000, PPNTarifBps: 1200, PPNDPPPembilang: 11, PPNDPPPenyebut: 12,
		BatasNPPN: 4_800_000_000, NormaPerbandinganBps: 5000, Objek: JenisPajakList, Norma: normaBawaan,
		PerluVerifikasi: []string{
			"Tax Planning Calculator V2.1 menyebut skema PPh final UMKM sebagai \"PP 20/2026\"; aturan itu belum diverifikasi tim, aplikasi masih merujuk PP 55/2022.",
			"Lapisan TER bulanan (PMK 168/2023) belum lengkap; lengkapi dari Lampiran PMK sebelum TER dipakai.",
			"Tabel norma disalin dari workbook (PER-17/PJ/2015); KLU 01623 dan 43299 kembar di sumber, baris kedua dibuang.",
		},
	}.Salin()
}

// Salin: salinan dalam (map & slice baru) supaya versi yang diubah tidak bocor ke versi lain.
func (p Parameter) Salin() Parameter {
	c := p
	c.PTKP = maps.Clone(p.PTKP)
	c.Pasal17 = slices.Clone(p.Pasal17)
	c.TER.Kategori = maps.Clone(p.TER.Kategori)
	c.TER.Lapisan = make(map[string][]TERBaris, len(p.TER.Lapisan))
	for k, v := range p.TER.Lapisan {
		c.TER.Lapisan[k] = slices.Clone(v)
	}
	c.Objek = slices.Clone(p.Objek)
	for i := range c.Objek {
		c.Objek[i].Objek = slices.Clone(c.Objek[i].Objek)
	}
	c.Norma = slices.Clone(p.Norma)
	c.PerluVerifikasi = slices.Clone(p.PerluVerifikasi)
	return c
}

// ObjekTerikat: objek potong-pungut yang tarifnya disalin dari field utama oleh Selaraskan,
// supaya satu angka aturan tidak hidup di dua tempat. Panel Master menguncinya.
var ObjekTerikat = []string{"42-umkm", "ppn-umum", "ppn-mewah"}

// Selaraskan menyalin tarif UMKM & PPN ke objek terikat; dipanggil sebelum simpan & aktifkan.
func (p *Parameter) Selaraskan() {
	for i := range p.Objek {
		for j := range p.Objek[i].Objek {
			o := &p.Objek[i].Objek[j]
			switch o.Kode {
			case "42-umkm":
				o.TarifBps = int(p.UMKMTarifBps)
			case "ppn-umum":
				o.TarifBps = int(p.PPNTarifBps)
				if p.PPNDPPPenyebut > 0 {
					o.DasarBps = int(p.PPNDPPPembilang * 10_000 / p.PPNDPPPenyebut)
				}
			case "ppn-mewah":
				o.TarifBps = int(p.PPNTarifBps)
			}
		}
	}
}

// PPNEfektifBps: tarif x DPP nilai lain (12% x 11/12 = 11%).
func (p *Parameter) PPNEfektifBps() int64 {
	return p.PPNTarifBps * p.PPNDPPPembilang / p.PPNDPPPenyebut
}

// PPN atas harga jual (sebelum PPN), dihitung dalam satu pembagian supaya pembulatannya tunggal.
func (p *Parameter) PPN(hargaJual int64) int64 {
	return mulDiv(hargaJual, p.PPNTarifBps*p.PPNDPPPembilang, 10_000*p.PPNDPPPenyebut)
}

var ErrParameter = errors.New("parameter pajak tidak valid")

func salah(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrParameter, fmt.Sprintf(format, a...))
}

func bpsWajar(nama string, v, maks int64) error {
	if v < 0 || v > maks {
		return salah("%s %d bps di luar 0–%d", nama, v, maks)
	}
	return nil
}

func (p Parameter) Validasi() error {
	for _, kode := range append(slices.Clone(PTKPCodes), "K/I/0", "K/I/1", "K/I/2", "K/I/3") {
		if p.PTKP[kode] <= 0 {
			return salah("PTKP %s wajib diisi", kode)
		}
	}
	if len(p.Pasal17) == 0 {
		return salah("lapisan Pasal 17 kosong")
	}
	for i, l := range p.Pasal17 {
		terakhir := i == len(p.Pasal17)-1
		switch {
		case terakhir && l.Sampai != 0:
			return salah("lapisan Pasal 17 terakhir harus tanpa batas")
		case !terakhir && (l.Sampai <= 0 || (i > 0 && l.Sampai <= p.Pasal17[i-1].Sampai)):
			return salah("batas lapisan Pasal 17 ke-%d harus lebih besar dari lapisan sebelumnya", i+1)
		}
		if err := bpsWajar(fmt.Sprintf("tarif lapisan Pasal 17 ke-%d", i+1), l.TarifBps, 10_000); err != nil {
			return err
		}
	}
	for _, c := range []struct {
		nama   string
		v, max int64
	}{
		{"biaya jabatan", p.BiayaJabatanBps, 10_000}, {"pengali tanpa NPWP PPh 21", p.TanpaNPWPPPh21Bps, 30_000},
		{"dasar PPh 21 bukan pegawai", p.DasarBukanPegawaiBps, 10_000}, {"tarif UMKM", p.UMKMTarifBps, 10_000},
		{"tarif badan", p.BadanTarifBps, 10_000}, {"fasilitas Pasal 31E", p.Fasilitas31EBps, 10_000},
		{"tarif dividen", p.DividenBps, 10_000}, {"tarif PPN", p.PPNTarifBps, 10_000},
		{"norma perbandingan", p.NormaPerbandinganBps, 10_000},
	} {
		if err := bpsWajar(c.nama, c.v, c.max); err != nil {
			return err
		}
	}
	switch {
	case p.TanpaNPWPPPh21Bps < 10_000:
		return salah("pengali tanpa NPWP tidak boleh di bawah 100%%")
	case p.DasarBukanPegawaiBps == 0 || p.NormaPerbandinganBps == 0:
		return salah("dasar bukan pegawai dan norma perbandingan wajib lebih dari 0")
	case p.BiayaJabatanMaks < 0 || p.UMKMBebasOP < 0:
		return salah("batas biaya jabatan dan omzet bebas UMKM tidak boleh negatif")
	case p.BatasOmzetUMKM <= 0 || p.BatasPKP <= 0 || p.BatasNPPN <= 0:
		return salah("batas omzet UMKM, PKP, dan NPPN wajib diisi")
	case p.Batas31EPenuh <= 0 || p.Batas31EPenuh > p.Batas31E:
		return salah("batas fasilitas Pasal 31E harus di antara 0 dan batas omzet Pasal 31E")
	case p.PPNDPPPenyebut <= 0 || p.PPNDPPPembilang <= 0 || p.PPNDPPPembilang > p.PPNDPPPenyebut:
		return salah("DPP PPN harus pecahan antara 0 dan 1, misalnya 11/12")
	case strings.TrimSpace(p.DasarUMKM) == "":
		return salah("rujukan aturan UMKM wajib diisi")
	}
	if err := validasiTER(p.TER); err != nil {
		return err
	}
	if err := validasiObjek(p.Objek); err != nil {
		return err
	}
	return validasiNorma(p.Norma)
}

func validasiTER(t TERParam) error {
	for kode, kat := range t.Kategori {
		if _, ok := t.Lapisan[kat]; !ok {
			return salah("TER: kategori %s untuk %s tidak punya lapisan", kat, kode)
		}
	}
	for kat, ls := range t.Lapisan {
		for i, b := range ls {
			switch {
			case i == 0 && b.BatasBawah != 0:
				return salah("TER %s: lapisan pertama harus mulai dari 0", kat)
			case i > 0 && (ls[i-1].BatasAtas < 0 || b.BatasBawah <= ls[i-1].BatasAtas):
				return salah("TER %s: lapisan ke-%d harus mulai setelah lapisan sebelumnya", kat, i+1)
			case b.BatasAtas >= 0 && b.BatasAtas < b.BatasBawah:
				return salah("TER %s: batas atas lapisan ke-%d di bawah batas bawahnya", kat, i+1)
			case b.TarifBps < 0 || b.TarifBps > 10_000:
				return salah("TER %s: tarif lapisan ke-%d di luar 0–100%%", kat, i+1)
			}
		}
	}
	return nil
}

func validasiObjek(js []JenisPajak) error {
	jenis, kode := map[string]bool{}, map[string]bool{}
	for _, j := range js {
		if j.ID == "" || jenis[j.ID] || strings.TrimSpace(j.Nama) == "" {
			return salah("jenis pajak %q kosong atau kembar", j.ID)
		}
		jenis[j.ID] = true
		if j.TanpaNPWPBps != 0 && (j.TanpaNPWPBps < 10_000 || j.TanpaNPWPBps > 30_000) {
			return salah("%s: pengali tanpa NPWP harus 100%%–300%%", j.Nama)
		}
		for _, o := range j.Objek {
			switch {
			case o.Kode == "" || kode[o.Kode] || strings.TrimSpace(o.Nama) == "":
				return salah("objek pajak %q kosong atau kembar", o.Kode)
			case o.TarifBps < 0 || o.TarifBps > 30_000:
				return salah("%s: tarif di luar 0–300%%", o.Nama)
			case o.DasarBps <= 0 || o.DasarBps > 10_000:
				return salah("%s: dasar pengenaan harus 0–100%%", o.Nama)
			}
			kode[o.Kode] = true
		}
	}
	return nil
}

func validasiNorma(ns []Norma) error {
	ada := map[string]bool{}
	for i, n := range ns {
		if len(n.KLU) != 5 || strings.Trim(n.KLU, "0123456789") != "" {
			return salah("norma baris %d: KLU harus 5 angka", i+1)
		}
		if ada[n.KLU] {
			return salah("norma: KLU %s kembar", n.KLU)
		}
		ada[n.KLU] = true
		for _, v := range []int64{n.Ibukota10Bps, n.IbukotaLainBps, n.DaerahLainBps} {
			if v < -1 || v > 10_000 {
				return salah("norma KLU %s di luar 0–100%%", n.KLU)
			}
		}
	}
	return nil
}

// aktif: versi parameter yang dipakai seluruh aplikasi. Hanya dibaca — jangan ubah isi P().
var paramAktif atomic.Pointer[Parameter]

func init() { AktifkanParameter(ParameterBawaan()) }

// AktifkanParameter memasang salinan p (dipanggil saat boot & saat tim mengaktifkan versi).
func AktifkanParameter(p Parameter) {
	c := p.Salin()
	c.Selaraskan()
	paramAktif.Store(&c)
}

// P: parameter aktif, hanya untuk dibaca.
func P() *Parameter { return paramAktif.Load() }

// PTKPOf: PTKP setahun untuk status, menurut parameter aktif.
func PTKPOf(kode string) (int64, bool) {
	v, ok := P().PTKP[strings.TrimSpace(kode)]
	return v, ok
}

func CariNorma(klu string) (Norma, bool) {
	ns := P().Norma
	i := slices.IndexFunc(ns, func(n Norma) bool { return n.KLU == klu })
	if i < 0 {
		return Norma{}, false
	}
	return ns[i], true
}
