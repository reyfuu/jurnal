// Package xlsx membaca dan menulis workbook .xlsx sederhana (hanya teks), cukup untuk
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
)

type Sheet struct {
	Name string
	Rows [][]string
	// Tabel: opsional. Tanpa Tabel, sheet ditulis polos dengan baris pertama tebal.
	Tabel *Tabel
}

// Tabel: sheet sebagai tabel isian. Baris di atas Header adalah judul / ringkasan; baris
// Header adalah judul kolom; di bawahnya data plus BarisKosong baris bergaris yang siap diisi.
// Dengan Kunci, sheet diproteksi (tanpa kata sandi): baris boleh ditambah, tetapi kolom tidak,
// dan sel di luar kolom tabel tidak bisa diisi — supaya bentuk tabel tetap bisa dibaca impor.
type Tabel struct {
	Header      int
	Kolom       int
	Lebar       []float64        // lebar per kolom (satuan karakter Excel)
	Angka       []int            // kolom angka: disimpan sebagai angka berformat ribuan
	BarisKosong int              // baris kosong bergaris di bawah data
	Pilihan     map[int][]string // dropdown per kolom (total ≤255 huruf)
	Kunci       bool
}

// Gaya sel (indeks cellXfs di styles.xml).
const (
	gayaPolos = iota
	gayaTebal
	gayaHeader
	gayaIsi
	gayaIsiAngka
	gayaJudul
	gayaRumus
	gayaBebas
)

const stylesXML = `<styleSheet xmlns="%s">` +
	`<fonts count="3"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="14"/><name val="Calibri"/></font></fonts>` +
	`<fills count="3"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill>` +
	`<fill><patternFill patternType="solid"><fgColor rgb="FFDCE6F2"/><bgColor indexed="64"/></patternFill></fill></fills>` +
	`<borders count="2"><border><left/><right/><top/><bottom/><diagonal/></border>` +
	`<border><left style="thin"><color rgb="FF8EA0B8"/></left><right style="thin"><color rgb="FF8EA0B8"/></right><top style="thin"><color rgb="FF8EA0B8"/></top><bottom style="thin"><color rgb="FF8EA0B8"/></bottom><diagonal/></border></borders>` +
	`<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>` +
	`<cellXfs count="8">` +
	`<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>` +
	`<xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0" applyFont="1"/>` +
	`<xf numFmtId="0" fontId="1" fillId="2" borderId="1" xfId="0" applyFont="1" applyFill="1" applyBorder="1" applyAlignment="1"><alignment vertical="center" wrapText="1"/></xf>` +
	`<xf numFmtId="49" fontId="0" fillId="0" borderId="1" xfId="0" applyNumberFormat="1" applyBorder="1" applyProtection="1"><protection locked="0"/></xf>` +
	`<xf numFmtId="3" fontId="0" fillId="0" borderId="1" xfId="0" applyNumberFormat="1" applyBorder="1" applyProtection="1"><protection locked="0"/></xf>` +
	`<xf numFmtId="0" fontId="2" fillId="0" borderId="0" xfId="0" applyFont="1"/>` +
	`<xf numFmtId="3" fontId="1" fillId="2" borderId="1" xfId="0" applyNumberFormat="1" applyFont="1" applyFill="1" applyBorder="1"/>` +
	`<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0" applyProtection="1"><protection locked="0"/></xf>` +
	`</cellXfs></styleSheet>`

// angkaSel: teks yang ditulis sebagai angka (bilangan bulat, boleh minus).
func angkaSel(v string) bool {
	if v == "" || v == "-" {
		return false
	}
	_, err := strconv.ParseInt(v, 10, 64)
	return err == nil
}

// sheetTabel menulis isi worksheet untuk Tabel.
func sheetTabel(sh Sheet) string {
	t := sh.Tabel
	angka := map[int]bool{}
	for _, c := range t.Angka {
		angka[c] = true
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<worksheet xmlns="%s">`, nsMain)
	akhir := len(sh.Rows) + t.BarisKosong
	fmt.Fprintf(&b, `<sheetViews><sheetView workbookViewId="0"><pane ySplit="%d" topLeftCell="A%d" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews>`, t.Header+1, t.Header+2)
	b.WriteString(`<sheetFormatPr defaultRowHeight="15"/><cols>`)
	for c := 0; c < t.Kolom; c++ {
		w := 16.0
		if c < len(t.Lebar) && t.Lebar[c] > 0 {
			w = t.Lebar[c]
		}
		// Kolom tabel tidak terkunci: baris baru (sisipan atau di bawah tabel) bisa diisi.
		fmt.Fprintf(&b, `<col min="%d" max="%d" width="%.1f" style="%d" customWidth="1"/>`, c+1, c+1, w, gayaBebas)
	}
	b.WriteString(`</cols><sheetData>`)
	for r := 0; r < akhir; r++ {
		var row []string
		if r < len(sh.Rows) {
			row = sh.Rows[r]
		}
		fmt.Fprintf(&b, `<row r="%d">`, r+1)
		lebar := t.Kolom
		if r < t.Header {
			lebar = len(row)
		}
		for c := 0; c < lebar; c++ {
			v := ""
			if c < len(row) {
				v = row[c]
			}
			ref := fmt.Sprintf("%s%d", colName(c), r+1)
			gaya := gayaIsi
			switch {
			case r < t.Header && r == 0 && c == 0:
				gaya = gayaJudul
			case r < t.Header && c == 0:
				gaya = gayaTebal
			case r < t.Header && c >= 2 && !strings.HasPrefix(v, "="):
				gaya = gayaPolos // catatan di samping isian ringkasan
			case r == t.Header:
				gaya = gayaHeader
			case strings.HasPrefix(v, "="):
				gaya = gayaRumus
			case angka[c] && r > t.Header:
				gaya = gayaIsiAngka
			case r < t.Header && v == "":
				continue
			}
			switch {
			case strings.HasPrefix(v, "="):
				fmt.Fprintf(&b, `<c r="%s" s="%d"><f>%s</f></c>`, ref, gaya, esc(v[1:]))
			case v == "":
				fmt.Fprintf(&b, `<c r="%s" s="%d"/>`, ref, gaya)
			case gaya == gayaIsiAngka && angkaSel(v):
				fmt.Fprintf(&b, `<c r="%s" s="%d"><v>%s</v></c>`, ref, gaya, v)
			default:
				fmt.Fprintf(&b, `<c r="%s" s="%d" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`, ref, gaya, esc(v))
			}
		}
		b.WriteString(`</row>`)
	}
	b.WriteString(`</sheetData>`)
	if t.Kunci {
		// Atribut bernilai 0 = boleh. insertColumns/deleteColumns tidak disebut = dilarang.
		b.WriteString(`<sheetProtection sheet="1" objects="1" scenarios="1" formatCells="0" formatColumns="0" formatRows="0" insertRows="0" deleteRows="0" sort="0" autoFilter="0"/>`)
	}
	if len(t.Pilihan) > 0 {
		cols := make([]int, 0, len(t.Pilihan))
		for c := range t.Pilihan {
			cols = append(cols, c)
		}
		sortInts(cols)
		fmt.Fprintf(&b, `<dataValidations count="%d">`, len(cols))
		for _, c := range cols {
			fmt.Fprintf(&b, `<dataValidation type="list" allowBlank="1" showErrorMessage="1" errorTitle="Pilihan tidak dikenal" error="Pilih dari daftar." sqref="%s%d:%s%d"><formula1>"%s"</formula1></dataValidation>`,
				colName(c), t.Header+2, colName(c), maxRows, esc(strings.Join(t.Pilihan[c], ",")))
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
		{"xl/workbook.xml", fmt.Sprintf(`<workbook xmlns="%s" xmlns:r="%s"><sheets>%s</sheets></workbook>`, nsMain, nsRel, wbSheets.String())},
		{"xl/_rels/workbook.xml.rels", fmt.Sprintf(`<Relationships xmlns="%s">%s<Relationship Id="rId%d" Type="%s/styles" Target="styles.xml"/></Relationships>`, nsPkg, wbRels.String(), styles, nsRel)},
		{"xl/styles.xml", fmt.Sprintf(stylesXML, nsMain)},
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
