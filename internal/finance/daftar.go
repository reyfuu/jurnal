package finance

import (
	"cmp"
	"io"
	"slices"
	"strconv"
	"strings"

	"pajak/internal/pdf"
)

// Daftar: satu daftar untuk "Unduh semua daftar" (satu sheet Excel / satu bagian PDF). Sel
// berisi teks apa adanya; kolom Rupiah berisi angka bulat tanpa titik supaya di Excel tetap
// angka, dan di PDF ditulis bertitik rata kanan. Baris terakhir berawalan "Total" dicetak tebal.
type Daftar struct {
	Judul  string
	Kolom  []string
	Rupiah []int
	Baris  [][]string
}

// RekapPihak: satu pelanggan dengan penjualannya dalam rentang.
type RekapPihak struct {
	Nama      string
	Jumlah    int
	Penjualan int64
}

// RekapPelanggan: pelanggan = pihak transaksi penjualan (omzet) dalam rentang, nama tanpa beda
// huruf besar/kecil, urut penjualan terbesar.
func RekapPelanggan(txs []Transaction, dari, sampai string) []RekapPihak {
	per := map[string]*RekapPihak{}
	for _, t := range txs {
		j, _ := LookupJenis(t.Jenis)
		nama := strings.TrimSpace(t.Pihak)
		if j.Pos != "omzet" || nama == "" || t.Tanggal < dari || t.Tanggal > sampai {
			continue
		}
		k := strings.ToLower(nama)
		if per[k] == nil {
			per[k] = &RekapPihak{Nama: nama}
		}
		per[k].Jumlah++
		per[k].Penjualan += t.Nominal
	}
	out := make([]RekapPihak, 0, len(per))
	for _, r := range per {
		out = append(out, *r)
	}
	slices.SortFunc(out, func(a, b RekapPihak) int {
		return cmp.Or(cmp.Compare(b.Penjualan, a.Penjualan), cmp.Compare(a.Nama, b.Nama))
	})
	return out
}

// RenderDaftarPDF: setiap daftar dimulai di halaman baru (A4 rebah). Kolom rupiah selebar satu
// bagian, kolom teks dua bagian.
func RenderDaftarPDF(w io.Writer, usaha, sub string, daftar []Daftar) error {
	c := cetakBaru(pdf.NewLandscape(), usaha)
	c.ukuran = 8
	isi := c.kanan - c.kiri
	for _, d := range daftar {
		c.halaman("Daftar "+d.Judul, sub)
		bobot := 0.0
		for i := range d.Kolom {
			bobot += lebarRelatif(d, i)
		}
		kolom := make([]kolomPDF, len(d.Kolom))
		for i, k := range d.Kolom {
			kolom[i] = kolomPDF{judul: strings.ToUpper(k), lebar: isi * lebarRelatif(d, i) / bobot, kanan: slices.Contains(d.Rupiah, i)}
		}
		var baris []barisPDF
		for _, r := range d.Baris {
			b := barisPDF{gaya: GayaBaris, sel: slices.Clone(r)}
			if len(r) > 0 && r[0] == "Total" {
				b.gaya = GayaTotal
			}
			for _, i := range d.Rupiah {
				if i < len(b.sel) {
					if n, err := strconv.ParseInt(b.sel[i], 10, 64); err == nil {
						b.sel[i] = akuntansi(n)
					}
				}
			}
			baris = append(baris, b)
		}
		if len(baris) == 0 {
			baris = append(baris, barisPDF{gaya: GayaSub, sel: []string{"Tidak ada data."}})
		}
		c.tabel(kolom, baris)
	}
	return c.selesai(w)
}

func lebarRelatif(d Daftar, i int) float64 {
	if slices.Contains(d.Rupiah, i) {
		return 1
	}
	return 2
}
