// Package xlsx membaca sheet pertama dan menulis workbook .xlsx sederhana (hanya teks),
// cukup untuk template & impor transaksi tanpa dependensi luar.
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

// Write menulis workbook; semua sel disimpan sebagai teks (inlineStr), baris pertama tebal.
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
		{"xl/styles.xml", fmt.Sprintf(`<styleSheet xmlns="%s"><fonts count="2"><font/><font><b/></font></fonts>`+
			`<fills count="1"><fill><patternFill patternType="none"/></fill></fills><borders count="1"><border/></borders>`+
			`<cellStyleXfs count="1"><xf/></cellStyleXfs><cellXfs count="2"><xf/><xf fontId="1" applyFont="1"/></cellXfs></styleSheet>`, nsMain)},
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

// Read mengembalikan isi sheet pertama sebagai teks. Baris kosong tetap ada (slice kosong)
// agar nomor baris cocok dengan Excel; angka dikembalikan apa adanya (tanggal = nomor seri).
func Read(data []byte) ([][]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, ErrNotXLSX
	}
	open := func(name string) ([]byte, error) {
		f, err := zr.Open(name)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return io.ReadAll(f)
	}

	var wb struct {
		Sheets []struct {
			RID string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
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
	sheetPath := "xl/worksheets/sheet1.xml"
	if b, err := open("xl/_rels/workbook.xml.rels"); err == nil && xml.Unmarshal(b, &rels) == nil {
		for _, r := range rels.Rel {
			if r.ID == wb.Sheets[0].RID {
				sheetPath = path.Join("xl", strings.TrimPrefix(r.Target, "/xl/"))
			}
		}
	}

	type richText struct {
		T string `xml:"t"`
		R []struct {
			T string `xml:"t"`
		} `xml:"r"`
	}
	text := func(rt richText) string {
		s := rt.T
		for _, r := range rt.R {
			s += r.T
		}
		return s
	}
	var shared []string
	if b, err := open("xl/sharedStrings.xml"); err == nil {
		var sst struct {
			SI []richText `xml:"si"`
		}
		if xml.Unmarshal(b, &sst) == nil {
			for _, si := range sst.SI {
				shared = append(shared, text(si))
			}
		}
	}

	var ws struct {
		Rows []struct {
			R     int `xml:"r,attr"`
			Cells []struct {
				Ref string   `xml:"r,attr"`
				T   string   `xml:"t,attr"`
				V   string   `xml:"v"`
				Is  richText `xml:"is"`
			} `xml:"c"`
		} `xml:"sheetData>row"`
	}
	if b, err = open(sheetPath); err != nil || xml.Unmarshal(b, &ws) != nil {
		return nil, ErrNotXLSX
	}

	var out [][]string
	for _, row := range ws.Rows {
		rn := row.R
		if rn == 0 {
			rn = len(out) + 1
		}
		for len(out) < rn-1 {
			out = append(out, []string{})
		}
		cells := []string{}
		for j, c := range row.Cells {
			col := j
			if letters := strings.TrimRight(c.Ref, "0123456789"); letters != "" {
				col = 0
				for _, ch := range letters {
					col = col*26 + int(ch-'A'+1)
				}
				col--
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
			cells = append(cells, v)
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
