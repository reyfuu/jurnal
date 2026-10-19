// Package pdf menulis dokumen PDF satu atau beberapa halaman berisi teks, garis, dan kotak —
// cukup untuk mencetak invoice tanpa dependensi luar, sejalan dengan internal/xlsx.
//
// Hanya font bawaan PDF (Helvetica) yang dipakai, jadi tidak ada font yang perlu disertakan.
// Font bawaan memakai encoding WinAnsi, yang menampung seluruh karakter Latin-1; teks di luar
// itu (mis. aksara non-Latin) tidak didukung dan diganti '?' oleh winAnsi.
package pdf

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// Satuan di seluruh paket ini adalah titik PDF (1/72 inci). A4 = 595,28 x 841,89 pt.
const (
	A4Width  = 595.28
	A4Height = 841.89
)

type Font string

const (
	Regular Font = "F1" // Helvetica
	Bold    Font = "F2" // Helvetica-Bold
)

type Align int

const (
	Left Align = iota
	Right
	Center
)

// RGB bernilai 0..1 per kanal, sesuai operator warna PDF.
type RGB struct{ R, G, B float64 }

var (
	Black = RGB{0, 0, 0}
	Gray  = RGB{0.45, 0.48, 0.52}
	Line  = RGB{0.82, 0.84, 0.87}
)

// Doc mengumpulkan perintah gambar per halaman lalu merangkainya jadi PDF.
type Doc struct {
	pages []*bytes.Buffer
	cur   *bytes.Buffer
}

func New() *Doc {
	d := &Doc{}
	d.AddPage()
	return d
}

func (d *Doc) AddPage() {
	b := &bytes.Buffer{}
	d.pages = append(d.pages, b)
	d.cur = b
}

func (d *Doc) PageCount() int { return len(d.pages) }

// Text menggambar satu baris teks dengan titik jangkar (x, y); y dihitung dari atas halaman,
// kebalikan dari sumbu PDF, supaya pemanggil bisa berpikir "turun ke bawah".
func (d *Doc) Text(x, y float64, s string, f Font, size float64, c RGB, a Align) {
	if s == "" {
		return
	}
	switch a {
	case Right:
		x -= Width(s, f, size)
	case Center:
		x -= Width(s, f, size) / 2
	}
	fmt.Fprintf(d.cur, "BT /%s %.2f Tf %.3f %.3f %.3f rg %.2f %.2f Td (%s) Tj ET\n",
		f, size, c.R, c.G, c.B, x, A4Height-y, escape(winAnsi(s)))
}

// Rect menggambar kotak terisi; dipakai untuk latar baris header tabel.
func (d *Doc) Rect(x, y, w, h float64, c RGB) {
	fmt.Fprintf(d.cur, "%.3f %.3f %.3f rg %.2f %.2f %.2f %.2f re f\n",
		c.R, c.G, c.B, x, A4Height-y-h, w, h)
}

// HLine menggambar garis mendatar setebal thickness.
func (d *Doc) HLine(x1, x2, y, thickness float64, c RGB) {
	fmt.Fprintf(d.cur, "%.3f %.3f %.3f RG %.2f w %.2f %.2f m %.2f %.2f l S\n",
		c.R, c.G, c.B, thickness, x1, A4Height-y, x2, A4Height-y)
}

// Write merangkai seluruh halaman jadi satu berkas PDF.
func (d *Doc) Write(w io.Writer) error {
	// objek 1 Catalog, 2 Pages, 3..n Page, lalu Contents tiap halaman, lalu 2 font.
	n := len(d.pages)
	objs := make([]string, 0, 3+2*n+2)

	kids := make([]string, n)
	for i := range d.pages {
		kids[i] = fmt.Sprintf("%d 0 R", 3+i)
	}
	fontRef := fmt.Sprintf("<< /F1 %d 0 R /F2 %d 0 R >>", 3+2*n, 4+2*n)

	objs = append(objs,
		"<< /Type /Catalog /Pages 2 0 R >>",
		fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), n))
	for i := range d.pages {
		objs = append(objs, fmt.Sprintf(
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Resources << /Font %s >> /Contents %d 0 R >>",
			A4Width, A4Height, fontRef, 3+n+i))
	}
	for _, p := range d.pages {
		objs = append(objs, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", p.Len()+1, p.String()))
	}
	objs = append(objs,
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>")

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs)+1)
	for i, o := range objs {
		offsets[i+1] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	start := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for i := 1; i <= len(objs); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, start)

	_, err := w.Write(buf.Bytes())
	return err
}

// escape melindungi karakter yang punya arti khusus di dalam literal string PDF.
func escape(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`)
	return r.Replace(s)
}

// winAnsi memetakan teks Go (UTF-8) ke byte WinAnsi yang dipakai font bawaan. Rune di luar
// Latin-1 jadi '?' — lebih baik satu karakter salah daripada seluruh berkas rusak.
func winAnsi(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '\t':
			b.WriteByte(' ')
		case r < 0x20:
			// karakter kendali dibuang
		case r < 0x100:
			b.WriteByte(byte(r))
		case r == '’': // tanda kutip tunggal melengkung
			b.WriteByte('\'')
		case r == '–', r == '—': // en dash, em dash
			b.WriteByte('-')
		case r == '·', r == '•': // middle dot, bullet
			b.WriteByte('-')
		default:
			b.WriteByte('?')
		}
	}
	return b.String()
}
