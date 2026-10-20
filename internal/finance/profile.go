package finance

import (
	"errors"
	"slices"
	"strings"

	"pajak/internal/tax"
)

// BusinessProfile ditanyakan ke Owner saat pertama login; menentukan template CoA (Fase C)
// dan default Perencanaan Pajak.
type BusinessProfile struct {
	NamaUsaha   string `json:"nama_usaha"`
	JenisUsaha  string `json:"jenis_usaha"`  // perdagangan | jasa | manufaktur
	BentukUsaha string `json:"bentuk_usaha"` // op | pt_perorangan | cv | pt
	PTKP        string `json:"ptkp"`         // hanya untuk op, mis. "K/1"
	PKP         bool   `json:"pkp"`          // Pengusaha Kena Pajak (wajib pungut PPN)
	// MetodePersediaan: periodik | perpetual (perdagangan & manufaktur); menentukan akun HPP aktif.
	MetodePersediaan string `json:"metode_persediaan"`
	// KLU: kelompok KLU 3 digit (tax.KLUBenchmarks); dasar pembanding NPM & CTTOR DJP.
	KLU string `json:"klu,omitempty"`

	// Perkiraan setahun dari onboarding: dipakai cek benchmark DJP & Perencanaan Pajak
	// selama belum ada data transaksi. Laba sebelum pajak boleh negatif (rugi).
	OmzetTahunan     int64 `json:"omzet_tahunan,omitempty"`
	LabaSebelumPajak int64 `json:"laba_sebelum_pajak,omitempty"`

	// Identitas penerbit faktur. Semuanya opsional saat onboarding supaya wizard tidak jadi
	// panjang, tapi dicetak di invoice begitu diisi — faktur tanpa alamat & NPWP penerbit
	// tidak layak dikirim ke pelanggan badan.
	Alamat string `json:"alamat,omitempty"`
	// Lokasi usaha dari onboarding; dicetak di invoice bersama alamat.
	Kecamatan string `json:"kecamatan,omitempty"`
	Kota      string `json:"kota,omitempty"`
	Provinsi  string `json:"provinsi,omitempty"`
	NPWP      string `json:"npwp,omitempty"`
	Telepon   string `json:"telepon,omitempty"`
	Email     string `json:"email,omitempty"`

	// Rekening tujuan pembayaran, dicetak di kaki faktur.
	BankNama     string `json:"bank_nama,omitempty"`
	BankRekening string `json:"bank_rekening,omitempty"`
	BankAtasNama string `json:"bank_atas_nama,omitempty"`

	// PPNBps: tarif PPN dalam basis poin (1100 = 11%). Dibuat field, bukan konstanta, supaya
	// perubahan tarif tidak perlu ubah kode. Hanya dipakai bila PKP.
	PPNBps int `json:"ppn_bps,omitempty"`

	// PenandatanganNama & PenandatanganJabatan: dicetak di atas garis tanda tangan faktur.
	PenandatanganNama    string `json:"penandatangan_nama,omitempty"`
	PenandatanganJabatan string `json:"penandatangan_jabatan,omitempty"`
}

// DefaultPPNBps dipakai bila profil PKP belum menyetel tarif sendiri.
const DefaultPPNBps = 1100

// TarifPPNBps: tarif yang berlaku untuk faktur baru. Non-PKP selalu 0 — mereka memang tidak
// boleh memungut PPN, jadi ini dijaga di satu tempat, bukan diserahkan ke pemanggil.
func (p BusinessProfile) TarifPPNBps() int {
	if !p.PKP {
		return 0
	}
	if p.PPNBps > 0 {
		return p.PPNBps
	}
	return DefaultPPNBps
}

type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Hint  string `json:"hint,omitempty"`
}

var JenisUsahaOptions = []Option{
	{"perdagangan", "Perdagangan", "Beli lalu jual kembali: toko, distributor, reseller, minimarket"},
	{"jasa", "Jasa", "Menjual keahlian atau layanan: salon, bengkel, konsultan, laundry, agensi"},
	{"manufaktur", "Manufaktur / Produksi", "Mengolah bahan baku jadi produk: konveksi, catering, roastery, furnitur"},
}

var MetodePersediaanOptions = []Option{
	{"periodik", "Periodik", "HPP dihitung di akhir periode dari stok opname; cocok bila belum ada kartu stok per barang"},
	{"perpetual", "Perpetual", "Stok dan HPP tercatat setiap transaksi; butuh kartu stok per barang"},
}

// ProvinsiOptions: 38 provinsi Indonesia.
var ProvinsiOptions = []string{
	"Aceh", "Sumatera Utara", "Sumatera Barat", "Riau", "Kepulauan Riau", "Jambi", "Bengkulu", "Sumatera Selatan",
	"Kepulauan Bangka Belitung", "Lampung", "Banten", "DKI Jakarta", "Jawa Barat", "Jawa Tengah", "DI Yogyakarta",
	"Jawa Timur", "Bali", "Nusa Tenggara Barat", "Nusa Tenggara Timur", "Kalimantan Barat", "Kalimantan Tengah",
	"Kalimantan Selatan", "Kalimantan Timur", "Kalimantan Utara", "Sulawesi Utara", "Gorontalo", "Sulawesi Tengah",
	"Sulawesi Barat", "Sulawesi Selatan", "Sulawesi Tenggara", "Maluku", "Maluku Utara", "Papua", "Papua Barat",
	"Papua Barat Daya", "Papua Tengah", "Papua Pegunungan", "Papua Selatan",
}

var BentukUsahaOptions = []Option{
	{"op", "Orang Pribadi", "Usaha atas nama pribadi (NPWP pemilik)"},
	{"pt_perorangan", "PT Perorangan", "Badan hukum satu pemilik"},
	{"cv", "CV", "Persekutuan komanditer"},
	{"pt", "PT", "Perseroan terbatas"},
}

func hasOption(opts []Option, v string) bool {
	return slices.ContainsFunc(opts, func(o Option) bool { return o.Value == v })
}

func (p *BusinessProfile) Validate() error {
	p.NamaUsaha = strings.TrimSpace(p.NamaUsaha)
	if p.NamaUsaha == "" {
		return errors.New("nama usaha wajib diisi")
	}
	if !hasOption(JenisUsahaOptions, p.JenisUsaha) {
		return errors.New("pilih jenis usaha: perdagangan, jasa, atau manufaktur")
	}
	if !hasOption(BentukUsahaOptions, p.BentukUsaha) {
		return errors.New("pilih bentuk usaha")
	}
	if _, ok := tax.FindKLU(p.KLU); p.KLU != "" && !ok {
		return errors.New("bidang usaha (KLU) tidak dikenal")
	}
	if p.Provinsi != "" && !slices.Contains(ProvinsiOptions, p.Provinsi) {
		return errors.New("pilih provinsi dari daftar")
	}
	if p.OmzetTahunan < 0 {
		return errors.New("omzet per tahun tidak boleh negatif")
	}
	switch {
	case p.JenisUsaha == "jasa":
		p.MetodePersediaan = ""
	case p.MetodePersediaan == "":
		p.MetodePersediaan = "periodik" // profil lama (sebelum Fase C) & default UMKM
	case !hasOption(MetodePersediaanOptions, p.MetodePersediaan):
		return errors.New("metode persediaan harus periodik atau perpetual")
	}
	if p.BentukUsaha != "op" {
		p.PTKP = "" // PTKP hanya berlaku untuk wajib pajak orang pribadi
		return nil
	}
	if _, ok := tax.PTKPOf(p.PTKP); !ok {
		return errors.New("pilih status PTKP pemilik (mis. TK/0, K/1)")
	}
	return nil
}

func (p BusinessProfile) JenisLabel() string {
	for _, o := range JenisUsahaOptions {
		if o.Value == p.JenisUsaha {
			return o.Label
		}
	}
	return ""
}

// CekAwal: NPM vs benchmark DJP dan risiko pajak dari perkiraan omzet & laba profil.
func (p BusinessProfile) CekAwal() tax.CekAwalResult {
	return tax.CekAwal(tax.CekAwalInput{
		BentukUsaha: p.BentukUsaha, PTKP: p.PTKP, KLU: p.KLU, PKP: p.PKP,
		Omzet: p.OmzetTahunan, Laba: p.LabaSebelumPajak,
	})
}
