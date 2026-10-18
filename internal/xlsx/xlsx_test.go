package xlsx_test

import (
	"archive/zip"
	"bytes"
	"reflect"
	"testing"

	"pajak/internal/xlsx"
)

func TestWriteThenRead_FirstSheet(t *testing.T) {
	var buf bytes.Buffer
	sheets := []xlsx.Sheet{
		{Name: "Transaksi", Rows: [][]string{
			{"Tanggal", "Jenis", "Nominal"},
			{"2026-08-01", "Penjualan & <Promo>", "1250000"},
			{"", "", ""},
			{"2026-08-02", "Gaji & Upah"},
		}},
		{Name: "Daftar Jenis", Rows: [][]string{{"Penjualan"}}},
	}
	if err := xlsx.Write(&buf, sheets); err != nil {
		t.Fatal(err)
	}
	rows, err := xlsx.Read(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"Tanggal", "Jenis", "Nominal"},
		{"2026-08-01", "Penjualan & <Promo>", "1250000"},
		{},
		{"2026-08-02", "Gaji & Upah"},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("roundtrip:\nwant %q\ngot  %q", want, rows)
	}
}

// Berkas dari Excel/LibreOffice memakai sharedStrings, sel angka, rich text, dan kolom yang dilompati.
func TestRead_SharedStringsAndGaps(t *testing.T) {
	files := map[string]string{
		"xl/workbook.xml": `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Data" sheetId="1" r:id="rId7"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId7" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/data.xml"/></Relationships>`,
		"xl/sharedStrings.xml": `<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><si><t>Tanggal</t></si><si><r><t>Nomi</t></r><r><t>nal</t></r></si></sst>`,
		"xl/worksheets/data.xml": `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>
			<row r="1"><c r="A1" t="s"><v>0</v></c><c r="C1" t="s"><v>1</v></c></row>
			<row r="3"><c r="A3"><v>46234</v></c><c r="C3"><v>1500000.5</v></c><c r="D3" t="b"><v>1</v></c></row>
		</sheetData></worksheet>`,
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		f, _ := zw.Create(name)
		_, _ = f.Write([]byte(body))
	}
	_ = zw.Close()

	rows, err := xlsx.Read(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"Tanggal", "", "Nominal"}, {}, {"46234", "", "1500000.5", "1"}}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("want %q\ngot  %q", want, rows)
	}
}

func TestRead_NotXLSX(t *testing.T) {
	if _, err := xlsx.Read([]byte("tanggal,jenis\n")); err == nil {
		t.Fatal("want error for non-xlsx input")
	}
}
