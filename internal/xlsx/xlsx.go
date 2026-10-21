// Package xlsx membaca dan menulis workbook .xlsx sederhana (teks, angka, tanggal, rumus), cukup untuk
// template & impor data tanpa dependensi luar.
package xlsx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"time"
)

type Sheet struct {
	Name string
	Rows [][]string
	// Tabel: opsional. Tanpa Tabel, sheet ditulis polos dengan baris pertama tebal.
	Tabel *Tabel
}

// Tabel: sheet sebagai tabel isian. Baris di atas Header adalah judul / ringkasan; baris
// Header adalah judul kolom; di bawahnya data plus BarisKosong baris bergaris yang siap diisi.
// Header -1 = sheet bebas tanpa tabel (mis. Petunjuk): gaya diatur lewat Gaya.
//
// Warna mengikuti legenda template: kuning muda = diisi pengguna (tidak terkunci), abu-abu =
// rumus (terkunci), teks biru = baris contoh. Dengan Kunci, sheet diproteksi (tanpa kata
// sandi): baris boleh ditambah/dihapus, kolom tidak, dan sel di luar kolom tabel tidak bisa
// diisi — supaya bentuk tabel tetap bisa dibaca impor.
type Tabel struct {
	Header      int
	Kolom       int
	Lebar       []float64        // lebar per kolom (satuan karakter Excel)
	Angka       []int            // kolom rupiah: angka berformat ribuan
	Bilangan    []int            // kolom jumlah/qty: angka biasa (boleh desimal)
	Tanggal     []int            // kolom tanggal: ditulis sebagai tanggal Excel (yyyy-mm-dd)
	Contoh      int              // jumlah baris contoh di bawah header (teks biru)
	BarisKosong int              // baris kosong bergaris di bawah data
	Pilihan     map[int][]string // dropdown per kolom (total ≤255 huruf); satu nilai "=Sheet!$A$1:$A$9" = rentang sel
	// Rumus: rumus per kolom untuk SETIAP baris data, termasuk baris kosong; "{r}" diganti nomor
	// baris Excel. Sel yang sudah berisi di Rows tidak ditimpa.
	Rumus map[int]string
	// Gaya: gaya per sel ("C6" → GayaIsiTanggal) yang menimpa aturan bawaan.
	Gaya map[string]string
	// Akhir: >0 = indeks baris (0-based) setelah baris data terakhir; baris sesudahnya ditulis
	// bebas (ringkasan di bawah tabel). Tanpa Akhir, BarisKosong ditambahkan di bawah data.
	Akhir int
	// Referensi: isi tabel adalah data acuan (bergaris, terkunci), bukan isian pengguna.
	Referensi bool
	Kunci     bool
}

// Nama gaya untuk Tabel.Gaya.
const (
	GayaPolos      = "polos"
	GayaTebal      = "tebal"
	GayaJudul      = "judul"
	GayaCatatan    = "catatan"
	GayaHeader     = "header"
	GayaIsi        = "isi"
	GayaIsiAngka   = "isi_angka"
	GayaIsiTanggal = "isi_tanggal"
	GayaRumus      = "rumus"
	GayaRumusTebal = "rumus_tebal"
	GayaGaris      = "garis"
	GayaGarisTebal = "garis_tebal"
	GayaBiru       = "biru"
)

// gaya: satu entri cellXfs. Urutan daftar = indeks s="…" di sel; entri pertama wajib polos.
type gaya struct {
	nama                        string
	numFmt, font, fill, border  int
	terbuka, bungkus, angka, tg bool // terbuka = tidak terkunci saat sheet diproteksi
}

// Font: 0 biasa, 1 tebal, 2 judul, 3 biru (contoh), 4 miring abu (catatan).
// Fill: 2 header, 3 kuning muda (isian), 4 abu-abu (rumus). numFmt 164 = yyyy-mm-dd.
var daftarGaya = []gaya{
	{nama: GayaPolos},
	{nama: GayaTebal, font: 1},
	{nama: GayaJudul, font: 2},
	{nama: GayaCatatan, font: 4},
	{nama: GayaHeader, font: 1, fill: 2, border: 1, bungkus: true},
	{nama: GayaIsi, numFmt: 49, fill: 3, border: 1, terbuka: true},
	{nama: GayaIsiAngka, numFmt: 3, fill: 3, border: 1, terbuka: true, angka: true},
	{nama: "isi_bilangan", fill: 3, border: 1, terbuka: true, angka: true},
	{nama: GayaIsiTanggal, numFmt: 164, fill: 3, border: 1, terbuka: true, tg: true},
	{nama: "isi_c", numFmt: 49, font: 3, fill: 3, border: 1, terbuka: true},
	{nama: "isi_angka_c", numFmt: 3, font: 3, fill: 3, border: 1, terbuka: true, angka: true},
	{nama: "isi_bilangan_c", font: 3, fill: 3, border: 1, terbuka: true, angka: true},
	{nama: "isi_tanggal_c", numFmt: 164, font: 3, fill: 3, border: 1, terbuka: true, tg: true},
	{nama: GayaRumus, numFmt: 3, fill: 4, border: 1, angka: true},
	{nama: GayaRumusTebal, numFmt: 3, font: 1, fill: 4, border: 1, angka: true},
	{nama: GayaGaris, border: 1, bungkus: true, angka: true},
	{nama: GayaGarisTebal, font: 1, border: 1, angka: true},
	{nama: GayaBiru, font: 3},
	{nama: "bebas", terbuka: true},
}

func indeksGaya(nama string) (int, gaya) {
	for i, g := range daftarGaya {
		if g.nama == nama {
			return i, g
		}
	}
	return 0, daftarGaya[0]
}

func stylesXML() string {
	var b strings.Builder
	fmt.Fprintf(&b, `<styleSheet xmlns="%s">`, nsMain)
	b.WriteString(`<numFmts count="1"><numFmt numFmtId="164" formatCode="yyyy\-mm\-dd"/></numFmts>`)
	b.WriteString(`<fonts count="5"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><name val="Calibri"/></font>` +
		`<font><b/><sz val="14"/><name val="Calibri"/></font><font><sz val="11"/><color rgb="FF1F4E9A"/><name val="Calibri"/></font>` +
		`<font><i/><sz val="10"/><color rgb="FF595959"/><name val="Calibri"/></font></fonts>`)
	b.WriteString(`<fills count="5"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill>`)
	for _, rgb := range []string{"FFDCE6F2", "FFFFF2CC", "FFE7E6E6"} {
		fmt.Fprintf(&b, `<fill><patternFill patternType="solid"><fgColor rgb="%s"/><bgColor indexed="64"/></patternFill></fill>`, rgb)
	}
	b.WriteString(`</fills><borders count="2"><border><left/><right/><top/><bottom/><diagonal/></border>` +
		`<border><left style="thin"><color rgb="FF8EA0B8"/></left><right style="thin"><color rgb="FF8EA0B8"/></right><top style="thin"><color rgb="FF8EA0B8"/></top><bottom style="thin"><color rgb="FF8EA0B8"/></bottom><diagonal/></border></borders>` +
		`<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>`)
	fmt.Fprintf(&b, `<cellXfs count="%d">`, len(daftarGaya))
	for _, g := range daftarGaya {
		fmt.Fprintf(&b, `<xf numFmtId="%d" fontId="%d" fillId="%d" borderId="%d" xfId="0" applyNumberFormat="1" applyFont="1" applyFill="1" applyBorder="1" applyAlignment="1" applyProtection="1">`,
			g.numFmt, g.font, g.fill, g.border)
		if g.bungkus {
			b.WriteString(`<alignment vertical="top" wrapText="1"/>`)
		} else {
			b.WriteString(`<alignment vertical="top"/>`)
		}
		if g.terbuka {
			b.WriteString(`<protection locked="0"/>`)
		}
		b.WriteString(`</xf>`)
	}
	b.WriteString(`</cellXfs></styleSheet>`)
	return b.String()
}

// angkaSel: teks yang bisa ditulis sebagai angka (boleh minus & desimal titik).
func angkaSel(v string) bool {
	if v == "" || v == "-" {
		return false
	}
	_, err := strconv.ParseFloat(v, 64)
	return err == nil && !strings.ContainsAny(v, "eEnN") // tolak 1e5, NaN, Inf
}

var epochExcel = time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)

// seriTanggal: "2026-01-01" → nomor seri tanggal Excel ("46023").
func seriTanggal(v string) (string, bool) {
	t, err := time.Parse(time.DateOnly, v)
	if err != nil {
		return "", false
	}
	return strconv.Itoa(int(t.Sub(epochExcel).Hours() / 24)), true
}

// selXML menulis satu sel sesuai gayanya: rumus, angka, tanggal, atau teks.
func selXML(b *strings.Builder, ref, v, namaGaya string) {
	s, g := indeksGaya(namaGaya)
	switch {
	case strings.HasPrefix(v, "="):
		fmt.Fprintf(b, `<c r="%s" s="%d"><f>%s</f></c>`, ref, s, esc(v[1:]))
	case v == "":
		fmt.Fprintf(b, `<c r="%s" s="%d"/>`, ref, s)
	case g.tg:
		if n, ok := seriTanggal(v); ok {
			fmt.Fprintf(b, `<c r="%s" s="%d"><v>%s</v></c>`, ref, s, n)
			return
		}
		fmt.Fprintf(b, `<c r="%s" s="%d" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`, ref, s, esc(v))
	case g.angka && angkaSel(v):
		fmt.Fprintf(b, `<c r="%s" s="%d"><v>%s</v></c>`, ref, s, v)
	default:
		fmt.Fprintf(b, `<c r="%s" s="%d" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`, ref, s, esc(v))
	}
}

// adaIsianDiKanan: ada rumus / sel isian di kanan kolom c — teks di kiri-nya berarti label.
func adaIsianDiKanan(t *Tabel, row []string, r, c int) bool {
	for k := c + 1; k < max(len(row), c+4); k++ {
		if k < len(row) && strings.HasPrefix(row[k], "=") {
			return true
		}
		if g := t.Gaya[fmt.Sprintf("%s%d", colName(k), r+1)]; strings.HasPrefix(g, "isi") {
			return true
		}
	}
	return false
}

// sheetTabel menulis isi worksheet untuk Tabel.
func sheetTabel(sh Sheet) string {
	t := sh.Tabel
	kolom := func(cs []int) map[int]bool {
		m := map[int]bool{}
		for _, c := range cs {
			m[c] = true
		}
		return m
	}
	angka, bilangan, tanggal := kolom(t.Angka), kolom(t.Bilangan), kolom(t.Tanggal)
	var b strings.Builder
	fmt.Fprintf(&b, `<worksheet xmlns="%s">`, nsMain)
	akhirData := len(sh.Rows) + t.BarisKosong
	if t.Akhir > 0 {
		akhirData = t.Akhir
	}
	total := max(len(sh.Rows), akhirData)
	if t.Header >= 0 {
		fmt.Fprintf(&b, `<sheetViews><sheetView workbookViewId="0"><pane ySplit="%d" topLeftCell="A%d" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews>`, t.Header+1, t.Header+2)
	}
	b.WriteString(`<sheetFormatPr defaultRowHeight="15"/><cols>`)
	for c := 0; c < max(t.Kolom, len(t.Lebar)); c++ {
		w := 16.0
		if c < len(t.Lebar) && t.Lebar[c] > 0 {
			w = t.Lebar[c]
		}
		gayaKolom := ""
		if t.Header >= 0 && c < t.Kolom && !t.Referensi {
			// Kolom tabel tidak terkunci: baris baru (sisipan atau di bawah tabel) bisa diisi.
			s, _ := indeksGaya("bebas")
			gayaKolom = fmt.Sprintf(` style="%d"`, s)
		}
		fmt.Fprintf(&b, `<col min="%d" max="%d" width="%.1f"%s customWidth="1"/>`, c+1, c+1, w, gayaKolom)
	}
	b.WriteString(`</cols><sheetData>`)
	for r := 0; r < total; r++ {
		var row []string
		if r < len(sh.Rows) {
			row = sh.Rows[r]
		}
		bebas := t.Header < 0 || r < t.Header || r >= akhirData
		lebar := len(row)
		if !bebas {
			lebar = max(lebar, t.Kolom)
		}
		fmt.Fprintf(&b, `<row r="%d">`, r+1)
		for c := 0; c < lebar; c++ {
			v := ""
			if c < len(row) {
				v = row[c]
			}
			ref := fmt.Sprintf("%s%d", colName(c), r+1)
			dalam := c < t.Kolom
			if !bebas && r > t.Header && dalam && v == "" {
				if f, ok := t.Rumus[c]; ok {
					v = strings.ReplaceAll(f, "{r}", strconv.Itoa(r+1))
				}
			}
			g, ada := t.Gaya[ref]
			switch {
			case ada:
			case bebas || !dalam:
				switch {
				case v == "":
					continue
				case r == 0 && !strings.HasPrefix(v, "="):
					g = GayaJudul
				case r == t.Header:
					g = GayaHeader
				case strings.HasPrefix(v, "="):
					g = GayaRumus
				case r == 1:
					g = GayaCatatan
				case adaIsianDiKanan(t, row, r, c):
					g = GayaTebal
				case !bebas:
					g = GayaCatatan // catatan di samping tabel
				default:
					g = GayaPolos
				}
			case r == t.Header:
				g = GayaHeader
			case strings.HasPrefix(v, "="):
				g = GayaRumus
			case t.Referensi:
				g = GayaGaris
			default:
				switch {
				case tanggal[c]:
					g = GayaIsiTanggal
				case angka[c]:
					g = GayaIsiAngka
				case bilangan[c]:
					g = "isi_bilangan"
				default:
					g = GayaIsi
				}
				if r <= t.Header+t.Contoh {
					g += "_c"
				}
			}
			selXML(&b, ref, v, g)
		}
		b.WriteString(`</row>`)
	}
	b.WriteString(`</sheetData>`)
	if t.Kunci {
		// Atribut bernilai 0 = boleh. insertColumns/deleteColumns tidak disebut = dilarang.
		b.WriteString(`<sheetProtection sheet="1" objects="1" scenarios="1" formatCells="0" formatColumns="0" formatRows="0" insertRows="0" deleteRows="0" sort="0" autoFilter="0"/>`)
	}
	if len(t.Pilihan) > 0 && t.Header >= 0 {
		cols := make([]int, 0, len(t.Pilihan))
		for c := range t.Pilihan {
			cols = append(cols, c)
		}
		sortInts(cols)
		fmt.Fprintf(&b, `<dataValidations count="%d">`, len(cols))
		for _, c := range cols {
			daftar := `"` + strings.Join(t.Pilihan[c], ",") + `"`
			if p := t.Pilihan[c]; len(p) == 1 && strings.HasPrefix(p[0], "=") {
				daftar = p[0][1:]
			}
			fmt.Fprintf(&b, `<dataValidation type="list" allowBlank="1" showErrorMessage="1" errorTitle="Pilihan tidak dikenal" error="Pilih dari daftar." sqref="%s%d:%s%d"><formula1>%s</formula1></dataValidation>`,
				colName(c), t.Header+2, colName(c), maxRows, esc(daftar))
		}
		b.WriteString(`</dataValidations>`)
	}
	b.WriteString(`</worksheet>`)
	return b.String()
}

func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}

const (
	nsMain = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
	nsRel  = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	nsPkg  = "http://schemas.openxmlformats.org/package/2006/relationships"
)

func esc(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// colName: 0 → A, 25 → Z, 26 → AA.
func colName(i int) string {
	s := ""
	for i++; i > 0; i = (i - 1) / 26 {
		s = string(rune('A'+(i-1)%26)) + s
	}
	return s
}

// Write menulis workbook. Sheet polos: semua sel teks (inlineStr), baris pertama tebal. Sheet
// dengan Tabel: lihat Tabel.
func Write(w io.Writer, sheets []Sheet) error {
	zw := zip.NewWriter(w)
	put := func(name, body string) error {
		f, err := zw.Create(name)
		if err == nil {
			_, err = io.WriteString(f, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`+"\n"+body)
		}
		return err
	}
	var ct, wbSheets, wbRels strings.Builder
	for i, sh := range sheets {
		n := i + 1
		fmt.Fprintf(&ct, `<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, n)
		fmt.Fprintf(&wbSheets, `<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, esc(sh.Name), n, n)
		fmt.Fprintf(&wbRels, `<Relationship Id="rId%d" Type="%s/worksheet" Target="worksheets/sheet%d.xml"/>`, n, nsRel, n)

		if sh.Tabel != nil {
			if err := put(fmt.Sprintf("xl/worksheets/sheet%d.xml", n), sheetTabel(sh)); err != nil {
				return err
			}
			continue
		}
		var sd strings.Builder
		for r, row := range sh.Rows {
			fmt.Fprintf(&sd, `<row r="%d">`, r+1)
			for c, v := range row {
				if v == "" {
					continue
				}
				style := ""
				if r == 0 {
					style = ` s="1"`
				}
				fmt.Fprintf(&sd, `<c r="%s%d" t="inlineStr"%s><is><t xml:space="preserve">%s</t></is></c>`, colName(c), r+1, style, esc(v))
			}
			sd.WriteString(`</row>`)
		}
		if err := put(fmt.Sprintf("xl/worksheets/sheet%d.xml", n),
			fmt.Sprintf(`<worksheet xmlns="%s"><sheetData>%s</sheetData></worksheet>`, nsMain, sd.String())); err != nil {
			return err
		}
	}
	styles := len(sheets) + 1
	parts := []struct{ name, body string }{
		{"[Content_Types].xml", `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
			`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
			`<Default Extension="xml" ContentType="application/xml"/>` +
			`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
			`<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>` +
			ct.String() + `</Types>`},
		{"_rels/.rels", fmt.Sprintf(`<Relationships xmlns="%s"><Relationship Id="rId1" Type="%s/officeDocument" Target="xl/workbook.xml"/></Relationships>`, nsPkg, nsRel)},
		// fullCalcOnLoad: rumus template tidak membawa nilai tersimpan, jadi dihitung saat dibuka.
		{"xl/workbook.xml", fmt.Sprintf(`<workbook xmlns="%s" xmlns:r="%s"><sheets>%s</sheets><calcPr calcId="191029" fullCalcOnLoad="1"/></workbook>`, nsMain, nsRel, wbSheets.String())},
		{"xl/_rels/workbook.xml.rels", fmt.Sprintf(`<Relationships xmlns="%s">%s<Relationship Id="rId%d" Type="%s/styles" Target="styles.xml"/></Relationships>`, nsPkg, wbRels.String(), styles, nsRel)},
		{"xl/styles.xml", stylesXML()},
	}
	for _, p := range parts {
		if err := put(p.name, p.body); err != nil {
			return err
		}
	}
	return zw.Close()
}

// ErrNotXLSX: berkas bukan workbook .xlsx yang bisa dibaca.
var ErrNotXLSX = errors.New("berkas bukan Excel .xlsx")

// ErrTooLarge: isi workbook melewati batas baca (mencegah zip bomb & indeks raksasa).
var ErrTooLarge = errors.New("berkas Excel terlalu besar (maks 100.000 baris dan 200 kolom)")

// Batas baca. Berkas unggahan dibatasi 5 MB, tapi isi zip & nomor baris/kolom ditentukan
// pembuat berkas, jadi setiap tahap ekspansi dibatasi sendiri.
const (
	maxEntryBytes = 20 << 20 // isi satu entri zip setelah dekompresi
	maxTotalBytes = 50 << 20 // seluruh entri yang dibaca
	maxRows       = 100_000
	maxCols       = 200
	maxCells      = 1_000_000
	maxCellRunes  = 32_767 // batas isi sel Excel
)

// Read mengembalikan isi sheet pertama sebagai teks.
func Read(data []byte) ([][]string, error) {
	sheets, err := ReadAll(data)
	if err != nil {
		return nil, err
	}
	if len(sheets) == 0 {
		return nil, ErrNotXLSX
	}
	return sheets[0].Rows, nil
}

// ReadAll mengembalikan seluruh sheet beserta namanya, urut seperti di workbook. Baris kosong
// tetap ada (slice kosong) agar nomor baris cocok dengan Excel; angka dikembalikan apa adanya
// (tanggal = nomor seri). Sheet yang gagal dibaca dilewati, bukan menggagalkan seluruh berkas.
func ReadAll(data []byte) ([]Sheet, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, ErrNotXLSX
	}
	total := 0
	open := func(name string) ([]byte, error) {
		f, err := zr.Open(name)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		b, err := io.ReadAll(io.LimitReader(f, maxEntryBytes+1)) // ukuran di header zip bisa bohong
		total += len(b)
		if len(b) > maxEntryBytes || total > maxTotalBytes {
			return nil, ErrTooLarge
		}
		return b, err
	}

	var wb struct {
		Sheets []struct {
			Name string `xml:"name,attr"`
			RID  string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"sheets>sheet"`
	}
	var rels struct {
		Rel []struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
		} `xml:"Relationship"`
	}
	b, err := open("xl/workbook.xml")
	if err != nil || xml.Unmarshal(b, &wb) != nil || len(wb.Sheets) == 0 {
		return nil, ErrNotXLSX
	}
	target := map[string]string{}
	if b, err := open("xl/_rels/workbook.xml.rels"); err == nil && xml.Unmarshal(b, &rels) == nil {
		for _, r := range rels.Rel {
			target[r.ID] = path.Join("xl", strings.TrimPrefix(r.Target, "/xl/"))
		}
	}

	type richText struct {
		T string `xml:"t"`
		R []struct {
			T string `xml:"t"`
		} `xml:"r"`
	}
	text := func(rt richText) string {
		var sb strings.Builder
		sb.WriteString(rt.T)
		for _, r := range rt.R {
			if sb.Len() >= 4*maxCellRunes { // sudah pasti melewati batas rune
				break
			}
			sb.WriteString(r.T)
		}
		return truncate(sb.String())
	}
	var shared []string
	b, err = open("xl/sharedStrings.xml")
	if errors.Is(err, ErrTooLarge) {
		return nil, err
	}
	if err == nil {
		var sst struct {
			SI []richText `xml:"si"`
		}
		if xml.Unmarshal(b, &sst) == nil {
			for _, si := range sst.SI {
				shared = append(shared, text(si))
			}
		}
	}

	out := make([]Sheet, 0, len(wb.Sheets))
	for i, sh := range wb.Sheets {
		sheetPath := target[sh.RID]
		if sheetPath == "" {
			sheetPath = fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1)
		}
		rows, err := readSheet(open, sheetPath, shared, text)
		if errors.Is(err, ErrTooLarge) {
			return nil, err
		}
		if err != nil {
			continue // sheet rusak atau bukan worksheet: lewati, sheet lain tetap bisa dipakai
		}
		out = append(out, Sheet{Name: sh.Name, Rows: rows})
	}
	if len(out) == 0 {
		return nil, ErrNotXLSX
	}
	return out, nil
}

func readSheet[RT any](open func(string) ([]byte, error), sheetPath string, shared []string, text func(RT) string) ([][]string, error) {
	var ws struct {
		Rows []struct {
			R     int `xml:"r,attr"`
			Cells []struct {
				Ref string `xml:"r,attr"`
				T   string `xml:"t,attr"`
				V   string `xml:"v"`
				Is  RT     `xml:"is"`
			} `xml:"c"`
		} `xml:"sheetData>row"`
	}
	b, err := open(sheetPath)
	if errors.Is(err, ErrTooLarge) {
		return nil, err
	}
	if err != nil || xml.Unmarshal(b, &ws) != nil {
		return nil, ErrNotXLSX
	}

	var out [][]string
	nCells := 0
	for _, row := range ws.Rows {
		rn := row.R
		if rn == 0 {
			rn = len(out) + 1
		}
		if rn < 0 || rn > maxRows {
			return nil, ErrTooLarge
		}
		for len(out) < rn-1 {
			out = append(out, []string{})
		}
		cells := []string{}
		for j, c := range row.Cells {
			col := j
			if letters := strings.TrimRight(c.Ref, "0123456789"); letters != "" {
				if len(letters) > 3 {
					return nil, ErrTooLarge
				}
				col = 0
				for _, ch := range letters {
					if ch < 'A' || ch > 'Z' {
						return nil, ErrNotXLSX
					}
					col = col*26 + int(ch-'A'+1)
				}
				col--
			}
			if col >= maxCols {
				return nil, ErrTooLarge
			}
			v := c.V
			switch c.T {
			case "s":
				if k, err := strconv.Atoi(c.V); err == nil && k < len(shared) {
					v = shared[k]
				}
			case "inlineStr":
				v = text(c.Is)
			}
			for len(cells) < col {
				cells = append(cells, "")
			}
			cells = append(cells, truncate(v))
		}
		if nCells += len(cells); nCells > maxCells {
			return nil, ErrTooLarge
		}
		for len(cells) > 0 && strings.TrimSpace(cells[len(cells)-1]) == "" {
			cells = cells[:len(cells)-1]
		}
		out = append(out, cells)
	}
	for len(out) > 0 && len(out[len(out)-1]) == 0 {
		out = out[:len(out)-1]
	}
	return out, nil
}

// truncate memotong teks sel di batas Excel (dihitung per rune agar UTF-8 tidak terpotong).
func truncate(s string) string {
	if len(s) <= maxCellRunes {
		return s
	}
	if r := []rune(s); len(r) > maxCellRunes {
		return string(r[:maxCellRunes])
	}
	return s
}
