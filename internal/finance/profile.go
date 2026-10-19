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
	if _, ok := tax.PTKP[p.PTKP]; !ok {
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
