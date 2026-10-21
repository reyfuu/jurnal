package xlsx_test

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"

	"pajak/internal/xlsx"
)

// Tabel isian: tanggal jadi sel tanggal Excel, rupiah jadi angka, rumus per baris ikut ke baris
// kosong, sel isian tidak terkunci & rumus terkunci, dan semua bagian XML well-formed.
func TestWriteTabel_TanggalAngkaRumusKunci(t *testing.T) {
	sheets := []xlsx.Sheet{
		{Name: "Bebas", Rows: [][]string{{"", "Judul"}, {"", "Tanggal", "2026-01-01"}}, Tabel: &xlsx.Tabel{Header: -1, Kolom: 3,
			Gaya: map[string]string{"C2": xlsx.GayaIsiTanggal}, Kunci: true}},
		{Name: "Data", Rows: [][]string{
			{"Judul"}, {"Penjelasan"}, {"Total", "=SUM(C4:C100)"},
			{"Nama", "Tgl", "Nilai", "Sisa"},
			{"Contoh", "2025-12-31", "1500000", ""},
		}, Tabel: &xlsx.Tabel{Header: 3, Kolom: 4, Tanggal: []int{1}, Angka: []int{2, 3}, Contoh: 1, BarisKosong: 2,
			Rumus: map[int]string{3: `=IF(C{r}="","",C{r}*2)`}, Pilihan: map[int][]string{0: {"=Bebas!$B$1:$B$2"}}, Kunci: true}},
	}
	var buf bytes.Buffer
	if err := xlsx.Write(&buf, sheets); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	isi := map[string]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		_ = rc.Close()
		isi[f.Name] = string(b)
		dec := xml.NewDecoder(bytes.NewReader(b))
		for {
			if _, err := dec.Token(); err == io.EOF {
				break
			} else if err != nil {
				t.Fatalf("%s tidak well-formed: %v", f.Name, err)
			}
		}
	}
	data := isi["xl/worksheets/sheet2.xml"]
	for _, want := range []string{
		`<c r="B5" s="12"><v>46022</v></c>`,                             // tanggal contoh: seri Excel, teks biru
		`<c r="C5" s="10"><v>1500000</v></c>`,                           // rupiah contoh: angka
		`<c r="D7" s="13"><f>IF(C7=&#34;&#34;,&#34;&#34;,C7*2)</f></c>`, // rumus ikut ke baris kosong
		`<c r="B6" s="8"/>`,                                             // baris kosong: isian tanggal
		`<formula1>Bebas!$B$1:$B$2</formula1>`,
		`insertRows="0"`,
	} {
		if !strings.Contains(data, want) {
			t.Errorf("sheet Data tidak memuat %s", want)
		}
	}
	if !strings.Contains(isi["xl/worksheets/sheet1.xml"], `<c r="C2" s="8"><v>46023</v></c>`) {
		t.Errorf("sel tanggal sheet bebas: %s", isi["xl/worksheets/sheet1.xml"])
	}
	styles := isi["xl/styles.xml"]
	for _, want := range []string{`numFmtId="164" formatCode="yyyy\-mm\-dd"`, `FFFFF2CC`, `FFE7E6E6`, `FF1F4E9A`} {
		if !strings.Contains(styles, want) {
			t.Errorf("styles tidak memuat %s", want)
		}
	}
	// Dibaca balik: tanggal = nomor seri (parser impor menerimanya), rumus tanpa nilai = kosong.
	got, err := xlsx.ReadAll(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if row := got[1].Rows[4]; row[1] != "46022" || row[2] != "1500000" {
		t.Errorf("baris contoh dibaca: %q", row)
	}
}

// Dropdown "saran" (Bebas) tetap menerima ketikan di luar daftar; sel Status diwarnai merah
// bila memuat BELUM dan hijau bila SEIMBANG, lewat conditional formatting.
func TestTabel_PilihanBebasDanStatus(t *testing.T) {
	sheets := []xlsx.Sheet{{Name: "Data", Rows: [][]string{
		{"Judul"}, {"Status", `=IF(1=1,"SEIMBANG","BELUM SEIMBANG")`}, {"Jenis", "Nilai"}, {"Kas", "1"},
	}, Tabel: &xlsx.Tabel{Header: 2, Kolom: 2, Pilihan: map[int][]string{0: {"Kas", "Bank"}}, Bebas: []int{0},
		Status: []string{"B2"}, Kunci: true}}}
	var buf bytes.Buffer
	if err := xlsx.Write(&buf, sheets); err != nil {
		t.Fatal(err)
	}
	zr, _ := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	isi := map[string]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		_ = rc.Close()
		isi[f.Name] = string(b)
	}
	data := isi["xl/worksheets/sheet1.xml"]
	for _, want := range []string{`showErrorMessage="0"`, `<conditionalFormatting sqref="B2">`, `SEARCH(&#34;BELUM&#34;,B2)`} {
		if !strings.Contains(data, want) {
			t.Errorf("sheet tidak memuat %s:\n%s", want, data)
		}
	}
	// Urutan elemen worksheet wajib: sheetProtection → conditionalFormatting → dataValidations.
	if p, c, v := strings.Index(data, "<sheetProtection"), strings.Index(data, "<conditionalFormatting"), strings.Index(data, "<dataValidations"); !(p < c && c < v) {
		t.Errorf("urutan elemen salah: proteksi %d, format %d, validasi %d", p, c, v)
	}
	if !strings.Contains(isi["xl/styles.xml"], `<dxfs count="2">`) {
		t.Error("styles tanpa dxfs")
	}
}
