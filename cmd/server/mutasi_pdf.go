package main

import (
	"bytes"
	"errors"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/ledongthuc/pdf"
	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	pdfmodel "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

	ukurpdf "pajak/internal/pdf"
)

// pdfcpu menulis berkas konfigurasi di folder home, dan menghentikan program (os.Exit) bila
// gagal; image scratch tidak punya folder home. Dimatikan sebelum dipakai.
func init() { pdfapi.DisableConfigDir() }

var (
	errPDFTerkunci   = errors.New("PDF tidak bisa dibuka; berkasnya rusak atau bukan PDF biasa")
	errPDFPerluSandi = errors.New("PDF ini dikunci kata sandi. Isi kata sandi PDF (biasanya yang diberikan bank, mis. tanggal lahir), lalu pilih berkasnya lagi")
	errPDFSandiSalah = errors.New("kata sandi PDF salah. Periksa lagi kata sandi dari bank Anda")
	errPDFGambar     = errors.New("PDF ini tidak berisi teks (hasil scan / foto), jadi belum bisa dibaca. Unduh e-statement asli dari internet banking, atau pakai mutasi .xlsx / .csv")
)

// potongAwalPDF membuang sampah sebelum tanda "%PDF-". Beberapa internet banking menyisipkan
// keluaran HTML (mis. "<br><br>") di depan berkas yang diunduh; tanpa dipotong berkasnya tidak
// dikenali sebagai PDF dan posisi objek di dalamnya bergeser.
//
// Berkas seperti itu biasanya juga terpotong di ujung sebanyak sampah di depannya (panjang
// unduhan dihitung dari PDF aslinya), sehingga "startxref" & "%%EOF" hilang; penutupnya
// disusun ulang dari letak tabel xref terakhir.
func potongAwalPDF(data []byte) []byte {
	if i := bytes.Index(data[:min(len(data), 4096)], []byte("%PDF-")); i > 0 {
		data = data[i:]
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) || bytes.Contains(data[max(0, len(data)-1024):], []byte("%%EOF")) {
		return data
	}
	x := bytes.LastIndex(data, []byte("\nxref"))
	if x < 0 {
		return data
	}
	akhir := len(data)
	if s := bytes.LastIndex(data, []byte("startxref")); s > x {
		akhir = s
	}
	out := append(bytes.Clone(data[:akhir]), []byte("startxref\n"+strconv.Itoa(x+1)+"\n%%EOF\n")...)
	return out
}

// isiPDF: berkas adalah PDF (boleh diawali sampah, lihat potongAwalPDF).
func isiPDF(data []byte) bool { return bytes.HasPrefix(potongAwalPDF(data), []byte("%PDF-")) }

// maxHalamanPDF membatasi rekening koran yang sangat tebal (dan PDF yang sengaja dibuat berat).
const maxHalamanPDF = 60

// barisPDF mengambil teks PDF per baris visual, dengan kolom di satu baris tetap di satu baris
// dan dipisah spasi.
//
// sandi: kata sandi PDF e-statement (boleh kosong). Hanya dipakai membuka berkas di memori
// selama permintaan ini; tidak disimpan dan tidak dicatat di log.
func barisPDF(data []byte, sandi string) (out []string, err error) {
	defer func() { // pustaka PDF bisa panik pada berkas rusak
		if r := recover(); r != nil {
			out, err = nil, errPDFTerkunci
		}
	}()
	data = potongAwalPDF(data)
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil && bytes.Contains(data, []byte("/Encrypt")) {
		var polos []byte
		if polos, err = bukaSandiPDF(data, sandi); err != nil {
			return nil, err
		}
		r, err = pdf.NewReader(bytes.NewReader(polos), int64(len(polos)))
	}
	if err != nil {
		return nil, errPDFTerkunci
	}
	for i := 1; i <= min(r.NumPage(), maxHalamanPDF); i++ {
		p := r.Page(i)
		if p.V.IsNull() || p.V.Key("Contents").Kind() == pdf.Null {
			continue
		}
		out = append(out, barisHalaman(hurufHalaman(p))...)
	}
	if len(out) == 0 {
		return nil, errPDFGambar
	}
	return out, nil
}

// bukaSandiPDF membuka enkripsi PDF (RC4, AES-128, maupun AES-256 yang dipakai e-statement
// bank) dengan pdfcpu; pustaka pembaca teks hanya mengenal enkripsi lama.
func bukaSandiPDF(data []byte, sandi string) (polos []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			polos, err = nil, errPDFTerkunci
		}
	}()
	conf := pdfmodel.NewDefaultConfiguration()
	conf.UserPW, conf.OwnerPW = sandi, sandi
	conf.ValidationMode = pdfmodel.ValidationRelaxed
	var out bytes.Buffer
	if err := pdfapi.Decrypt(bytes.NewReader(data), &out, conf); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "password") {
			if sandi == "" {
				return nil, errPDFPerluSandi
			}
			return nil, errPDFSandiSalah
		}
		return nil, errPDFTerkunci
	}
	return out.Bytes(), nil
}

// --- Pembacaan huruf beserta posisinya ---
//
// Page.Content() dari pustaka tidak tahu lebar huruf font Type0 (font tertanam yang dipakai
// hampir semua e-statement): lebarnya nol, jadi jarak antarkolom tidak bisa diukur. Di sini
// operator teks dibaca sendiri dan lebar diambil dari /W font turunannya.

type matriks [3][3]float64

var identitas = matriks{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}

func (x matriks) kali(y matriks) matriks {
	var z matriks
	for i := range 3 {
		for j := range 3 {
			for k := range 3 {
				z[i][j] += x[i][k] * y[k][j]
			}
		}
	}
	return z
}

// fontPDF: penerjemah kode → teks dan lebar per kode (per 1000 satuan ukuran huruf).
type fontPDF struct {
	enc    pdf.TextEncoding
	duaBit bool // Type0: tiap huruf 2 byte
	lebar  func(kode int) float64
}

func bacaFont(f pdf.Font) fontPDF {
	enc := f.Encoder()
	if enc == nil {
		enc = nopEnc{}
	}
	if f.V.Key("Subtype").Name() != "Type0" {
		return fontPDF{enc: enc, lebar: func(k int) float64 { return f.Width(k) }}
	}
	turunan := f.V.Key("DescendantFonts").Index(0)
	bawaan := 1000.0
	if dw := turunan.Key("DW"); dw.Kind() != pdf.Null {
		bawaan = dw.Float64()
	}
	// /W: [c [w1 w2 …]  c1 c2 w  …]
	peta := map[int]float64{}
	w := turunan.Key("W")
	for i := 0; i < w.Len(); {
		awal := int(w.Index(i).Int64())
		if i+1 < w.Len() && w.Index(i+1).Kind() == pdf.Array {
			arr := w.Index(i + 1)
			for j := 0; j < arr.Len() && j < 65536; j++ {
				peta[awal+j] = arr.Index(j).Float64()
			}
			i += 2
			continue
		}
		if i+2 >= w.Len() {
			break
		}
		akhir, nilai := int(w.Index(i+1).Int64()), w.Index(i+2).Float64()
		for c := awal; c <= akhir && c-awal < 65536; c++ {
			peta[c] = nilai
		}
		i += 3
	}
	return fontPDF{enc: enc, duaBit: true, lebar: func(k int) float64 {
		if v, ok := peta[k]; ok {
			return v
		}
		return bawaan
	}}
}

type nopEnc struct{}

func (nopEnc) Decode(raw string) string { return raw }

type keadaanTeks struct {
	Tc, Tw, Th, Tl, Tfs, Trise float64
	Tm, Tlm, CTM               matriks
	font                       fontPDF
	adaFont                    bool
}

func hurufHalaman(p pdf.Page) []pdf.Text {
	g := keadaanTeks{Th: 1, CTM: identitas, Tm: identitas, Tlm: identitas}
	cache := map[string]fontPDF{}
	var tumpukan []keadaanTeks
	var out []pdf.Text

	tampil := func(raw string) {
		if !g.adaFont {
			return
		}
		lebarKode := 1
		if g.font.duaBit {
			lebarKode = 2
		}
		for i := 0; i+lebarKode <= len(raw); i += lebarKode {
			kodeRaw := raw[i : i+lebarKode]
			kode := int(kodeRaw[0])
			if lebarKode == 2 {
				kode = kode<<8 | int(kodeRaw[1])
			}
			teks := g.font.enc.Decode(kodeRaw)
			w0 := g.font.lebar(kode)
			if w0 <= 0 {
				// Font standar tanpa /Widths (Helvetica dsb.): pakai metrik Helvetica.
				w0 = ukurpdf.Width(teks, ukurpdf.Regular, 1000)
			}
			trm := matriks{{g.Tfs * g.Th, 0, 0}, {0, g.Tfs, 0}, {0, g.Trise, 1}}.kali(g.Tm).kali(g.CTM)
			out = append(out, pdf.Text{FontSize: trm[0][0], X: trm[2][0], Y: trm[2][1], W: w0 / 1000 * trm[0][0], S: teks})
			tx := w0/1000*g.Tfs + g.Tc
			if lebarKode == 1 && kode == ' ' {
				tx += g.Tw
			}
			g.Tm = matriks{{1, 0, 0}, {0, 1, 0}, {tx * g.Th, 0, 1}}.kali(g.Tm)
		}
	}
	geser := func(tx, ty float64) {
		g.Tlm = matriks{{1, 0, 0}, {0, 1, 0}, {tx, ty, 1}}.kali(g.Tlm)
		g.Tm = g.Tlm
	}

	pdf.Interpret(p.V.Key("Contents"), func(stk *pdf.Stack, op string) {
		args := make([]pdf.Value, stk.Len())
		for i := len(args) - 1; i >= 0; i-- {
			args[i] = stk.Pop()
		}
		angka := func(i int) float64 {
			if i < len(args) {
				return args[i].Float64()
			}
			return 0
		}
		switch op {
		case "cm":
			if len(args) == 6 {
				m := matriks{{angka(0), angka(1), 0}, {angka(2), angka(3), 0}, {angka(4), angka(5), 1}}
				g.CTM = m.kali(g.CTM)
			}
		case "q":
			tumpukan = append(tumpukan, g)
		case "Q":
			if n := len(tumpukan); n > 0 {
				g, tumpukan = tumpukan[n-1], tumpukan[:n-1]
			}
		case "BT":
			g.Tm, g.Tlm = identitas, identitas
		case "Tc":
			g.Tc = angka(0)
		case "Tw":
			g.Tw = angka(0)
		case "Tz":
			g.Th = angka(0) / 100
		case "TL":
			g.Tl = angka(0)
		case "Ts":
			g.Trise = angka(0)
		case "Tf":
			if len(args) == 2 {
				nama := args[0].Name()
				f, ok := cache[nama]
				if !ok {
					f = bacaFont(p.Font(nama))
					cache[nama] = f
				}
				g.font, g.adaFont, g.Tfs = f, true, angka(1)
			}
		case "Td":
			geser(angka(0), angka(1))
		case "TD":
			g.Tl = -angka(1)
			geser(angka(0), angka(1))
		case "Tm":
			if len(args) == 6 {
				g.Tlm = matriks{{angka(0), angka(1), 0}, {angka(2), angka(3), 0}, {angka(4), angka(5), 1}}
				g.Tm = g.Tlm
			}
		case "T*":
			geser(0, -g.Tl)
		case "Tj":
			if len(args) == 1 {
				tampil(args[0].RawString())
			}
		case "'":
			geser(0, -g.Tl)
			if len(args) == 1 {
				tampil(args[0].RawString())
			}
		case "\"":
			if len(args) == 3 {
				g.Tw, g.Tc = angka(0), angka(1)
				geser(0, -g.Tl)
				tampil(args[2].RawString())
			}
		case "TJ":
			if len(args) != 1 {
				return
			}
			v := args[0]
			for i := 0; i < v.Len(); i++ {
				x := v.Index(i)
				if x.Kind() == pdf.String {
					tampil(x.RawString())
					continue
				}
				tx := -x.Float64() / 1000 * g.Tfs * g.Th
				g.Tm = matriks{{1, 0, 0}, {0, 1, 0}, {tx, 0, 1}}.kali(g.Tm)
			}
		}
	})
	return out
}

// --- Penyusunan baris ---

// potongan: huruf-huruf yang bersambung di satu baris, dengan posisi awal dan ujungnya.
type potongan struct {
	x, y, ujung, ukuran float64
	teks                strings.Builder
}

func satuBaris(y1, y2, u1, u2 float64) bool { return math.Abs(y1-y2) <= max(u1, u2)*0.35 }

// barisHalaman menyusun huruf menjadi baris: huruf yang bersentuhan disambung, celah lebih dari
// ~0,3 em (lebar spasi) diberi spasi.
func barisHalaman(huruf []pdf.Text) []string {
	var semua []*potongan
	var cur *potongan
	for _, t := range huruf {
		if t.S == "" {
			continue
		}
		u := max(math.Abs(t.FontSize), 1)
		w := t.W
		if cur == nil || !satuBaris(t.Y, cur.y, u, cur.ukuran) || t.X < cur.x || t.X-cur.ujung > u*0.3 || cur.ujung-t.X > u*0.5 {
			cur = &potongan{x: t.X, y: t.Y, ujung: t.X, ukuran: u}
			semua = append(semua, cur)
		}
		cur.teks.WriteString(t.S)
		cur.ujung = max(cur.ujung, t.X+w)
	}
	// Atas ke bawah, lalu kiri ke kanan.
	slices.SortStableFunc(semua, func(a, b *potongan) int {
		if !satuBaris(a.y, b.y, a.ukuran, b.ukuran) {
			if a.y > b.y {
				return -1
			}
			return 1
		}
		switch {
		case a.x < b.x:
			return -1
		case a.x > b.x:
			return 1
		}
		return 0
	})
	var out []string
	var baris []string
	var awal *potongan
	tutup := func() {
		if s := strings.Join(strings.Fields(strings.Join(baris, " ")), " "); s != "" {
			out = append(out, s)
		}
		baris = nil
	}
	for _, p := range semua {
		if awal != nil && !satuBaris(p.y, awal.y, p.ukuran, awal.ukuran) {
			tutup()
		}
		if len(baris) == 0 {
			awal = p
		}
		baris = append(baris, p.teks.String())
	}
	tutup()
	return out
}
