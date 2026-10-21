package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"slices"
	"strings"
	"testing"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	pdfmodel "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

	"pajak/internal/finance"
	"pajak/internal/pdf"
)

// rekeningKoranPDF: PDF berteks dengan kolom di posisi berbeda, seperti e-statement bank.
func rekeningKoranPDF(t *testing.T) []byte {
	t.Helper()
	d := pdf.New()
	d.AddPage()
	hitam := pdf.RGB{}
	baris := [][]string{
		{"TANGGAL", "KETERANGAN", "MUTASI", "SALDO"},
		{"01/09", "SALDO AWAL", "", "10,000,000.00"},
		{"02/09", "TRSF E-BANKING CR BUDI", "1,500,000.00", "11,500,000.00"},
		{"03/09", "BIAYA ADM", "15,000.00 DB", "11,485,000.00"},
	}
	for i, b := range baris {
		y := 100 + float64(i)*16
		d.Text(40, y, b[0], pdf.Regular, 9, hitam, pdf.Left)
		d.Text(90, y, b[1], pdf.Regular, 9, hitam, pdf.Left)
		d.Text(420, y, b[2], pdf.Regular, 9, hitam, pdf.Right)
		d.Text(540, y, b[3], pdf.Regular, 9, hitam, pdf.Right)
	}
	var buf bytes.Buffer
	if err := d.Write(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func unggahBerkasMutasi(srv *Server, c *http.Cookie, nama string, isi []byte) (int, string, finance.MutasiHasil) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", nama)
	_, _ = fw.Write(isi)
	_ = mw.WriteField("tahun", "2026")
	_ = mw.Close()
	rec := upload(srv, c, "/api/v1/transactions/mutasi", &body, mw.FormDataContentType())
	var h finance.MutasiHasil
	raw := rec.Body.String()
	_ = json.Unmarshal([]byte(raw), &h)
	return rec.Code, raw, h
}

func TestMutasi_PDFRekeningKoran(t *testing.T) {
	srv := NewServer()
	code, raw, h := unggahBerkasMutasi(srv, login(t, srv, "owner@kedaikopisenja.id"), "estatement.pdf", rekeningKoranPDF(t))
	if code != http.StatusOK || len(h.Rows) != 2 || len(h.Errors) != 0 {
		t.Fatalf("pratinjau PDF: %d %s", code, raw)
	}
	if r := h.Rows[0]; r.Tanggal != "2026-09-02" || r.Arah != "masuk" || r.Nominal != 1_500_000 || !strings.Contains(r.Keterangan, "BUDI") {
		t.Errorf("baris 1: %+v", r)
	}
	if r := h.Rows[1]; r.Tanggal != "2026-09-03" || r.Arah != "keluar" || r.Nominal != 15_000 {
		t.Errorf("baris 2: %+v", r)
	}
}

func TestMutasi_PDFRusakBerpesanJelas(t *testing.T) {
	srv := NewServer()
	code, raw, _ := unggahBerkasMutasi(srv, login(t, srv, "owner@kedaikopisenja.id"), "rusak.pdf", []byte("%PDF-1.4\nbukan pdf"))
	if code != http.StatusBadRequest || !strings.Contains(raw, "PDF") {
		t.Fatalf("PDF rusak: %d %s", code, raw)
	}
}

// pdfType0: PDF minimal dengan font Type0 (2 byte per huruf, lebar dari /W) seperti font
// tertanam di e-statement. Kolomnya berjarak hanya ~1 em, jadi tanpa lebar huruf yang benar
// kolom-kolomnya menempel.
func pdfType0(baris [][2]any) []byte {
	hex := func(s string) string {
		var b strings.Builder
		for _, r := range s {
			fmt.Fprintf(&b, "%04X", r)
		}
		return "<" + b.String() + ">"
	}
	var isi strings.Builder
	for i, b := range baris {
		y := 700 - i*14
		for _, sel := range b[1].([]struct {
			x float64
			s string
		}) {
			fmt.Fprintf(&isi, "BT /F1 10 Tf 1 0 0 1 %.1f %d Tm %s Tj ET\n", sel.x, y, hex(sel.s))
		}
	}
	cmap := "/CIDInit /ProcSet findresource begin 12 dict begin begincmap /CMapName /Uji def 1 begincodespacerange <0000> <FFFF> endcodespacerange 1 beginbfrange <0020> <007E> <0020> endbfrange endcmap CMapName currentdict /CMap defineresource pop end end"
	obj := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", isi.Len(), isi.String()),
		"<< /Type /Font /Subtype /Type0 /BaseFont /Uji /Encoding /Identity-H /DescendantFonts [6 0 R] /ToUnicode 7 0 R >>",
		"<< /Type /Font /Subtype /CIDFontType2 /BaseFont /Uji /CIDSystemInfo << /Registry (Adobe) /Ordering (Identity) /Supplement 0 >> /DW 500 /W [32 [250] 44 46 250] >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(cmap), cmap),
	}
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offs := make([]int, len(obj))
	for i, o := range obj {
		offs[i] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(obj)+1)
	for _, o := range offs {
		fmt.Fprintf(&out, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(obj)+1, xref)
	return out.Bytes()
}

func TestBarisPDF_FontType0KolomRapat(t *testing.T) {
	type sel = struct {
		x float64
		s string
	}
	// Lebar huruf 500/1000 × 10 pt = 5 pt; "02/09" berakhir di 65, kolom berikut mulai 75 (1 em).
	data := pdfType0([][2]any{
		{0, []sel{{40, "Tanggal"}, {120, "Keterangan"}, {300, "Debet"}, {360, "Kredit"}, {430, "Saldo"}}},
		{0, []sel{{40, "02/09/2026"}, {100, "SETORAN TUNAI"}, {175, "0.00"}, {205, "1,500,000.00"}, {275, "2,500,000.00"}}},
	})
	baris, err := barisPDF(data, "")
	if err != nil {
		t.Fatal(err)
	}
	want := "02/09/2026 SETORAN TUNAI 0.00 1,500,000.00 2,500,000.00"
	if len(baris) != 2 || baris[1] != want {
		t.Fatalf("baris = %q; want baris kedua %q", baris, want)
	}
	h := finance.ParseMutasi(finance.MutasiDariTeks(baris), 2026, nil)
	if len(h.Rows) != 1 || h.Rows[0].Arah != "masuk" || h.Rows[0].Nominal != 1_500_000 {
		t.Fatalf("hasil: %+v %v", h.Rows, h.Errors)
	}
}

func kunciPDF(t *testing.T, data []byte, conf *pdfmodel.Configuration) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := pdfapi.Encrypt(bytes.NewReader(data), &out, conf); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// E-statement bank biasanya dikunci kata sandi; AES-256 tidak dikenal pustaka pembaca teks.
func TestBarisPDF_BerkataSandi(t *testing.T) {
	polos := rekeningKoranPDF(t)
	for nama, conf := range map[string]*pdfmodel.Configuration{
		"AES-256": pdfmodel.NewAESConfiguration("010190", "pemilik-bank", 256),
		"AES-128": pdfmodel.NewAESConfiguration("010190", "pemilik-bank", 128),
		"RC4-128": pdfmodel.NewRC4Configuration("010190", "pemilik-bank", 128),
	} {
		data := kunciPDF(t, polos, conf)
		if _, err := barisPDF(data, ""); !errors.Is(err, errPDFPerluSandi) {
			t.Errorf("%s tanpa sandi: %v", nama, err)
		}
		if _, err := barisPDF(data, "salah"); !errors.Is(err, errPDFSandiSalah) {
			t.Errorf("%s sandi salah: %v", nama, err)
		}
		baris, err := barisPDF(data, "010190")
		if err != nil || !slices.Contains(baris, "02/09 TRSF E-BANKING CR BUDI 1,500,000.00 11,500,000.00") {
			t.Errorf("%s sandi benar: %v %q", nama, err, baris)
		}
	}
}

func TestMutasi_PDFBerkataSandiLewatAPI(t *testing.T) {
	srv := NewServer()
	c := login(t, srv, "owner@kedaikopisenja.id")
	data := kunciPDF(t, rekeningKoranPDF(t), pdfmodel.NewAESConfiguration("010190", "pemilik", 256))
	if code, raw, _ := unggahBerkasMutasi(srv, c, "estatement.pdf", data); code != http.StatusBadRequest || !strings.Contains(raw, "kata sandi") {
		t.Fatalf("tanpa sandi: %d %s", code, raw)
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "estatement.pdf")
	_, _ = fw.Write(data)
	_ = mw.WriteField("tahun", "2026")
	_ = mw.WriteField("sandi", "010190")
	_ = mw.Close()
	rec := upload(srv, c, "/api/v1/transactions/mutasi", &body, mw.FormDataContentType())
	var h finance.MutasiHasil
	_ = json.NewDecoder(rec.Body).Decode(&h)
	if rec.Code != http.StatusOK || len(h.Rows) != 2 {
		t.Fatalf("dengan sandi: %d %+v", rec.Code, h)
	}
}
