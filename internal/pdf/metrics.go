package pdf

import "strings"

// Lebar karakter Helvetica & Helvetica-Bold dari AFM Adobe, dalam 1/1000 em, untuk kode 32..126.
// Hanya rentang ini yang didaftar: teks invoice berbahasa Indonesia praktis seluruhnya ASCII,
// dan karakter di luarnya memakai widthFallback.
var (
	helvetica = [...]int16{
		278, 278, 355, 556, 556, 889, 667, 191, 333, 333, 389, 584, 278, 333, 278, 278, // 32..47
		556, 556, 556, 556, 556, 556, 556, 556, 556, 556, 278, 278, 584, 584, 584, 556, // 48..63
		1015, 667, 667, 722, 722, 667, 611, 778, 722, 278, 500, 667, 556, 833, 722, 778, // 64..79
		667, 778, 722, 667, 611, 722, 667, 944, 667, 667, 611, 278, 278, 278, 469, 556, // 80..95
		333, 556, 556, 500, 556, 556, 278, 556, 556, 222, 222, 500, 222, 833, 556, 556, // 96..111
		556, 556, 333, 500, 278, 556, 500, 722, 500, 500, 500, 334, 260, 334, 584, // 112..126
	}
	helveticaBold = [...]int16{
		278, 333, 474, 556, 556, 889, 722, 238, 333, 333, 389, 584, 278, 333, 278, 278,
		556, 556, 556, 556, 556, 556, 556, 556, 556, 556, 333, 333, 584, 584, 584, 611,
		975, 722, 722, 722, 722, 667, 611, 778, 722, 278, 556, 722, 611, 833, 722, 778,
		667, 778, 722, 667, 611, 722, 667, 944, 667, 667, 611, 333, 278, 333, 584, 556,
		333, 556, 611, 556, 611, 556, 333, 611, 611, 278, 278, 556, 278, 889, 611, 611,
		611, 611, 389, 556, 333, 611, 556, 778, 556, 556, 500, 389, 280, 389, 584,
	}
)

// widthFallback: lebar untuk karakter di luar 32..126 (mis. huruf beraksen), diambil dari
// rata-rata huruf kecil supaya penempatan rata kanan tidak melenceng jauh.
const widthFallback = 556

func charWidth(r rune, f Font) int16 {
	tbl := helvetica[:]
	fb := int16(widthFallback)
	if f == Bold {
		tbl = helveticaBold[:]
		fb = 611
	}
	if r < 32 || int(r)-32 >= len(tbl) {
		return fb
	}
	return tbl[int(r)-32]
}

// Width mengukur lebar teks dalam titik PDF pada ukuran font tertentu.
func Width(s string, f Font, size float64) float64 {
	var w int
	for _, r := range s {
		w += int(charWidth(r, f))
	}
	return float64(w) * size / 1000
}

// Truncate memotong teks agar muat dalam max titik, menambahkan elipsis bila terpotong.
// Dipakai untuk deskripsi baris invoice yang kepanjangan supaya tidak menabrak kolom harga.
func Truncate(s string, f Font, size, max float64) string {
	if Width(s, f, size) <= max {
		return s
	}
	const ell = "..."
	limit := max - Width(ell, f, size)
	if limit <= 0 {
		return ""
	}
	var b strings.Builder
	var w float64
	for _, r := range s {
		cw := float64(charWidth(r, f)) * size / 1000
		if w+cw > limit {
			break
		}
		b.WriteRune(r)
		w += cw
	}
	return strings.TrimRight(b.String(), " ") + ell
}

// WrapText memecah teks jadi beberapa baris yang masing-masing muat dalam max titik.
// Kata yang sendirian lebih panjang dari max dipotong paksa agar tidak meluber.
func WrapText(s string, f Font, size, max float64, maxLines int) []string {
	var out []string
	for _, para := range strings.Split(s, "\n") {
		line := ""
		for _, word := range strings.Fields(para) {
			try := word
			if line != "" {
				try = line + " " + word
			}
			if Width(try, f, size) <= max {
				line = try
				continue
			}
			if line != "" {
				out = append(out, line)
			}
			line = word
			if Width(line, f, size) > max {
				out = append(out, Truncate(line, f, size, max))
				line = ""
			}
		}
		if line != "" {
			out = append(out, line)
		}
	}
	if maxLines > 0 && len(out) > maxLines {
		out = out[:maxLines]
	}
	return out
}
