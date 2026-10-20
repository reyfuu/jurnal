package xlsx_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"pajak/internal/xlsx"
)

// workbook: xlsx minimal satu sheet dengan isi sheet1.xml & sharedStrings.xml pilihan.
func workbook(t *testing.T, sheet, shared string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"xl/workbook.xml":          `<workbook xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="S" r:id="rId1"/></sheets></workbook>`,
		"xl/worksheets/sheet1.xml": sheet,
		"xl/sharedStrings.xml":     shared,
	} {
		f, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// Audit C7: nomor baris raksasa dari berkas ~500 byte tidak boleh dikembangkan jadi jutaan baris.
func TestRowIndexBesarDitolak(t *testing.T) {
	data := workbook(t, `<worksheet><sheetData><row r="2000000"><c r="A2000000"><v>1</v></c></row></sheetData></worksheet>`, `<sst/>`)
	start := time.Now()
	if _, err := xlsx.Read(data); !errors.Is(err, xlsx.ErrTooLarge) {
		t.Fatalf("want ErrTooLarge, got %v", err)
	}
	if d := time.Since(start); d > 200*time.Millisecond {
		t.Fatalf("penolakan harus cepat, butuh %v", d)
	}
}

// Audit C7: indeks kolom raksasa (XFD... atau huruf asing) tidak boleh mengalokasikan sel kosong.
func TestKolomBesarDitolak(t *testing.T) {
	for _, ref := range []string{"ZZZ1", "AAAA1"} {
		data := workbook(t, `<worksheet><sheetData><row r="1"><c r="`+ref+`"><v>1</v></c></row></sheetData></worksheet>`, `<sst/>`)
		if _, err := xlsx.Read(data); err == nil {
			t.Fatalf("%s: kolom di luar batas harus ditolak", ref)
		}
	}
}

// Audit C7: entri zip yang mengembang melewati batas ditolak (zip bomb).
func TestEntriTerlaluBesarDitolak(t *testing.T) {
	big := `<worksheet><sheetData><row r="1"><c r="A1"><v>` + strings.Repeat("9", 21<<20) + `</v></c></row></sheetData></worksheet>`
	if _, err := xlsx.Read(workbook(t, big, `<sst/>`)); !errors.Is(err, xlsx.ErrTooLarge) {
		t.Fatalf("want ErrTooLarge, got %v", err)
	}
}

// Audit C6: banyak run rich-text digabung linear dan dipotong di batas sel Excel.
func TestRichTextLinearDanDipotong(t *testing.T) {
	run := `<r><t>` + strings.Repeat("x", 100) + `</t></r>`
	shared := `<sst><si>` + strings.Repeat(run, 50_000) + `</si></sst>`
	data := workbook(t, `<worksheet><sheetData><row r="1"><c r="A1" t="s"><v>0</v></c></row></sheetData></worksheet>`, shared)
	start := time.Now()
	rows, err := xlsx.Read(data)
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 10*time.Second { // versi kuadratik butuh menit
		t.Fatalf("gabungan rich-text harus linear, butuh %v", d)
	}
	if got := len(rows[0][0]); got != 32_767 {
		t.Fatalf("panjang sel = %d, want 32767 (batas Excel)", got)
	}
}
