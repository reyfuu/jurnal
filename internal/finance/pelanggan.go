package finance

import (
	"cmp"
	"errors"
	"strings"
)

// Pelanggan: master pelanggan (checklist tester No 24), pola sama dengan Supplier. Invoice dan
// transaksi tetap menyimpan nama pelanggan sebagai teks; daftar ini mengisi alamat & NPWP saat
// membuat invoice dan menjadi pilihan nama.
type Pelanggan struct {
	ID        int    `json:"id"`
	Nama      string `json:"nama"`
	Kontak    string `json:"kontak,omitempty"`
	Telepon   string `json:"telepon,omitempty"`
	Email     string `json:"email,omitempty"`
	Alamat    string `json:"alamat,omitempty"`
	NPWP      string `json:"npwp,omitempty"`
	TempoHari int    `json:"tempo_hari"`
	Catatan   string `json:"catatan,omitempty"`
}

func ValidatePelanggan(p *Pelanggan) error {
	for _, s := range []*string{&p.Nama, &p.Kontak, &p.Telepon, &p.Email, &p.Alamat, &p.NPWP, &p.Catatan} {
		*s = strings.TrimSpace(*s)
	}
	if p.Nama == "" {
		return errors.New("nama pelanggan wajib diisi")
	}
	if p.TempoHari < 0 {
		return errors.New("tempo pembayaran tidak boleh negatif")
	}
	return nil
}

func InfoPelanggan(p Pelanggan) (int, string, string) { return p.ID, "", p.Nama }

// PelangganDariData: pelanggan dari data lama — pihak transaksi penjualan dan pelanggan invoice
// (alamat & NPWP ikut dari invoice) — tanpa nama ganda (huruf besar/kecil & spasi diabaikan).
func PelangganDariData(txs []Transaction, inv []Invoice) []Pelanggan {
	var out []Pelanggan
	idx := map[string]int{}
	tambah := func(p Pelanggan) {
		k := KunciNama(p.Nama)
		if k == "" {
			return
		}
		if i, ok := idx[k]; ok {
			out[i].Alamat, out[i].NPWP = cmp.Or(out[i].Alamat, p.Alamat), cmp.Or(out[i].NPWP, p.NPWP)
			return
		}
		idx[k] = len(out)
		out = append(out, p)
	}
	for _, v := range inv {
		tambah(Pelanggan{Nama: strings.TrimSpace(v.PelangganNama), Alamat: v.PelangganAlamat, NPWP: v.PelangganNPWP})
	}
	for _, t := range txs {
		if j, ok := jenisOf(t.Jenis); ok && j.Pos == "omzet" {
			tambah(Pelanggan{Nama: strings.TrimSpace(t.Pihak)})
		}
	}
	return out
}
