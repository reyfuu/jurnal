package finance

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"pajak/internal/pdf"
)

// Cetak PDF Laporan Keuangan: per periode (A4 tegak, tiap laporan satu halaman baru) dan
// perbandingan multi-periode (A4 rebah). Keduanya memakai tabel yang sama: header kolom
// diulang dan kop diberi "(lanjutan)" bila baris melewati satu halaman.

var (
	warnaJudul = pdf.RGB{R: 0.94, G: 0.95, B: 0.97}
	warnaMerah = pdf.RGB{R: 0.75, G: 0.15, B: 0.15}
)

type kolomPDF struct {
	judul string
	lebar float64
	kanan bool
}

type barisPDF struct {
	sel  []string
	gaya string
	// negatif per sel angka, untuk mewarnai angka minus merah.
	minus []bool
}

type cetak struct {
	d              *pdf.Doc
	usaha          string
	kiri, kanan    float64
	bawah          float64
	y              float64
	judul, sub     string
	ukuran         float64 // ukuran huruf isi tabel
	dicetak        string
	halamanDipakai bool
}

func cetakBaru(d *pdf.Doc, usaha string) *cetak {
	if usaha == "" {
		usaha = "(Nama usaha belum diisi di Profil Usaha)"
	}
	return &cetak{d: d, usaha: usaha, kiri: 40, kanan: d.Width() - 40, bawah: d.Height() - 56, ukuran: 9,
		dicetak: tanggalID(time.Now().Format(time.DateOnly))}
}

// halaman memulai halaman baru (kecuali halaman pertama yang masih kosong) dengan kop.
func (c *cetak) halaman(judul, sub string) {
	if c.halamanDipakai {
		c.kaki()
		c.d.AddPage()
	}
	c.halamanDipakai = true
	c.judul, c.sub = judul, sub
	c.kop(judul)
}

func (c *cetak) kop(judul string) {
	c.d.Text(c.kiri, 48, pdf.Truncate(c.usaha, pdf.Bold, 13, c.kanan-c.kiri-160), pdf.Bold, 13, pdf.Black, pdf.Left)
	c.d.Text(c.kiri, 66, judul, pdf.Bold, 11, pdf.Black, pdf.Left)
	c.d.Text(c.kiri, 80, c.sub, pdf.Regular, 8.5, pdf.Gray, pdf.Left)
	c.d.Text(c.kanan, 48, "Master Pajak", pdf.Bold, 9, pdf.Gray, pdf.Right)
	c.d.Text(c.kanan, 60, "Dicetak "+c.dicetak, pdf.Regular, 8, pdf.Gray, pdf.Right)
	c.d.HLine(c.kiri, c.kanan, 90, 1, pdf.Black)
	c.y = 108
}

func (c *cetak) kaki() {
	y := c.d.Height() - 30
	c.d.Text(c.kiri, y, "Disusun dari data yang dicatat pengguna di aplikasi Master Pajak.", pdf.Regular, 7.5, pdf.Gray, pdf.Left)
	c.d.Text(c.kanan, y, fmt.Sprintf("Halaman %d", c.d.PageCount()), pdf.Regular, 7.5, pdf.Gray, pdf.Right)
}

func (c *cetak) tinggiBaris() float64 { return c.ukuran + 7 }

func (c *cetak) kepalaTabel(kolom []kolomPDF) {
	h := c.tinggiBaris() + 2
	c.d.Rect(c.kiri, c.y, c.kanan-c.kiri, h, warnaJudul)
	x := c.kiri
	for _, k := range kolom {
		c.teksSel(x, k, k.judul, pdf.Bold, pdf.Gray, c.y+h-5)
		x += k.lebar
	}
	c.y += h + 2
}

func (c *cetak) teksSel(x float64, k kolomPDF, s string, f pdf.Font, w pdf.RGB, baseline float64) {
	s = pdf.Truncate(s, f, c.ukuran, k.lebar-6)
	if k.kanan {
		c.d.Text(x+k.lebar-3, baseline, s, f, c.ukuran, w, pdf.Right)
	} else {
		c.d.Text(x+3, baseline, s, f, c.ukuran, w, pdf.Left)
	}
}

func (c *cetak) tabel(kolom []kolomPDF, baris []barisPDF) {
	c.kepalaTabel(kolom)
	for _, b := range baris {
		h := c.tinggiBaris()
		if b.gaya == GayaJudul {
			h += 4
		}
		if c.y+h > c.bawah {
			c.kaki()
			c.d.AddPage()
			c.kop(c.judul + " (lanjutan)")
			c.kepalaTabel(kolom)
		}
		font, warna := pdf.Regular, pdf.Black
		switch b.gaya {
		case GayaJudul:
			c.d.Rect(c.kiri, c.y+1, c.kanan-c.kiri, h-2, warnaJudul)
			font = pdf.Bold
		case GayaTotal:
			c.d.HLine(c.kiri, c.kanan, c.y+0.5, 0.6, pdf.Line)
			font = pdf.Bold
		case GayaSub:
			warna = pdf.Gray
		}
		baseline := c.y + h - 5
		x := c.kiri
		for i, k := range kolom {
			if i < len(b.sel) {
				w := warna
				if i < len(b.minus) && b.minus[i] {
					w = warnaMerah
				}
				teks, kk := b.sel[i], k
				if i == 0 && b.gaya == GayaSub {
					kk.lebar -= 12
					x += 12
					c.teksSel(x, kk, teks, font, w, baseline)
					x -= 12
				} else {
					c.teksSel(x, kk, teks, font, w, baseline)
				}
			}
			x += k.lebar
		}
		c.y += h
	}
	c.y += 10
}

func (c *cetak) selesai(w io.Writer) error {
	c.kaki()
	return c.d.Write(w)
}

// akuntansi: angka minus ditulis dalam kurung, kebiasaan laporan keuangan.
func akuntansi(n int64) string {
	if n < 0 {
		return "(" + rupiah(-n) + ")"
	}
	return rupiah(n)
}

// RenderLaporanPDF mencetak laporan satu periode; aset (boleh nil) dicetak sebagai register.
// labelNeraca: periode posisi neraca ("" = sama dengan labelPeriode); berbeda bila laba rugi &
// arus kas memakai rentang tanggal ("15 Januari 2026 s.d. 15 Desember 2026").
func RenderLaporanPDF(w io.Writer, usaha, labelPeriode, labelNeraca string, bagian []BagianLaporan, aset *AssetSummary) error {
	c := cetakBaru(pdf.New(), usaha)
	isi := c.kanan - c.kiri
	if labelNeraca == "" {
		labelNeraca = labelPeriode
	}
	for _, b := range bagian {
		sub := "Periode " + labelPeriode + " - dalam Rupiah"
		if b.Kode == "neraca" {
			sub = "Posisi per akhir " + labelNeraca + " - dalam Rupiah"
		}
		c.halaman(b.Judul, sub)
		kolom := []kolomPDF{{judul: "KETERANGAN", lebar: isi - 140}, {judul: "JUMLAH (RP)", lebar: 140, kanan: true}}
		var baris []barisPDF
		for _, r := range b.Baris {
			bp := barisPDF{sel: []string{r.Label}, gaya: r.Gaya}
			if r.Gaya != GayaJudul {
				bp.sel = append(bp.sel, akuntansi(r.Nilai))
				bp.minus = []bool{false, r.Nilai < 0}
			}
			baris = append(baris, bp)
		}
		c.tabel(kolom, baris)
	}
	if aset != nil {
		c.halaman("Register Aset & Penyusutan", "Periode "+labelPeriode+" - dalam Rupiah")
		c.ukuran = 7.5
		angka := (isi - 295) / 4
		kolom := []kolomPDF{{judul: "NAMA ASET", lebar: 125}, {judul: "KELOMPOK", lebar: 122}, {judul: "DIPEROLEH", lebar: 48},
			{judul: "PEROLEHAN", lebar: angka, kanan: true}, {judul: "AKUMULASI", lebar: angka, kanan: true},
			{judul: "NILAI BUKU", lebar: angka, kanan: true}, {judul: "BEBAN", lebar: angka, kanan: true}}
		var baris []barisPDF
		for _, a := range aset.Assets {
			nama := a.Nama
			if a.Draft {
				nama += " (draft)"
			}
			kel := a.KelompokNama
			if kel == "" {
				kel = a.Kelompok
			}
			baris = append(baris, barisPDF{gaya: GayaBaris, sel: []string{nama, kel, a.TanggalPerolehan,
				rupiah(a.HargaPerolehan), rupiah(a.Akumulasi), rupiah(a.NilaiBuku), rupiah(a.BebanBulanIni)}})
		}
		if len(aset.Assets) == 0 {
			baris = append(baris, barisPDF{gaya: GayaSub, sel: []string{"Belum ada aset terdaftar."}})
		}
		baris = append(baris, barisPDF{gaya: GayaTotal, sel: []string{"Total", "", "",
			rupiah(aset.TotalHargaPerolehan), rupiah(aset.TotalAkumulasi), rupiah(aset.TotalNilaiBuku), rupiah(aset.BebanTotal)}})
		c.tabel(kolom, baris)
		c.ukuran = 9
	}
	return c.selesai(w)
}

// RenderBandingPDF mencetak perbandingan multi-periode; huruf mengecil bila kolomnya banyak.
func RenderBandingPDF(w io.Writer, usaha string, b Banding) error {
	c := cetakBaru(pdf.NewLandscape(), usaha)
	n := len(b.Kolom)
	label := 200.0
	angka := (c.kanan - c.kiri - label) / float64(n+1)
	// Huruf terbesar yang masih memuat angka terlebar di kolomnya.
	terlebar := "(000.000.000)"
	for _, bg := range b.Bagian {
		for _, r := range bg.Baris {
			for _, v := range append(slices.Clone(r.Nilai), r.Total) {
				if s := akuntansi(v); len(s) > len(terlebar) {
					terlebar = s
				}
			}
		}
	}
	c.ukuran = 9
	for c.ukuran > 5.5 && pdf.Width(terlebar, pdf.Bold, c.ukuran) > angka-6 {
		c.ukuran -= 0.5
	}
	rentang := ""
	if n > 0 {
		rentang = b.Kolom[0].Label
		if n > 1 {
			rentang += " s.d. " + b.Kolom[n-1].Label
		}
	}
	for _, bg := range b.Bagian {
		c.halaman("Perbandingan "+bg.Judul, fmt.Sprintf("%d bulan: %s - dalam Rupiah. Kolom Total: laba rugi & arus kas dijumlahkan; neraca dan saldo kas akhir memakai posisi bulan terakhir.", n, rentang))
		kolom := []kolomPDF{{judul: "AKUN", lebar: label}}
		for _, k := range b.Kolom {
			judul := k.Label
			if pdf.Width(judul, pdf.Bold, c.ukuran) > angka-6 {
				judul = labelPeriodePendek(k.Periode) // "Agu 2026" supaya tidak terpotong
			}
			kolom = append(kolom, kolomPDF{judul: judul, lebar: angka, kanan: true})
		}
		judulTotal := "TOTAL"
		if bg.Kode == "neraca" {
			judulTotal = "POSISI AKHIR"
		}
		kolom = append(kolom, kolomPDF{judul: judulTotal, lebar: angka, kanan: true})
		var baris []barisPDF
		for _, r := range bg.Baris {
			bp := barisPDF{sel: []string{r.Label}, gaya: r.Gaya, minus: []bool{false}}
			if r.Gaya != GayaJudul {
				for _, v := range append(slices.Clone(r.Nilai), r.Total) {
					bp.sel = append(bp.sel, akuntansi(v))
					bp.minus = append(bp.minus, v < 0)
				}
			}
			baris = append(baris, bp)
		}
		c.tabel(kolom, baris)
	}
	return c.selesai(w)
}

// labelPeriodePendek: "2026-08" → "Agu 2026".
func labelPeriodePendek(p string) string {
	l := LabelPeriode(p)
	if i := strings.IndexByte(l, ' '); i > 3 {
		return l[:3] + l[i:]
	}
	return l
}
