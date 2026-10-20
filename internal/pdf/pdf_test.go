package pdf_test

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"pajak/internal/pdf"
)

func render(t *testing.T) []byte {
	t.Helper()
	d := pdf.New()
	d.Text(40, 60, "INVOICE", pdf.Bold, 20, pdf.Black, pdf.Left)
	d.Rect(40, 100, 515, 18, pdf.Line)
	d.HLine(40, 555, 130, 0.7, pdf.Line)
	d.Text(555, 150, "Rp 5.400.000", pdf.Regular, 10, pdf.Black, pdf.Right)
	d.AddPage()
	d.Text(297, 400, "halaman 2", pdf.Regular, 10, pdf.Gray, pdf.Center)
	var buf bytes.Buffer
	if err := d.Write(&buf); err != nil {
		t.Fatalf("Write: %v", err)
	}
	return buf.Bytes()
}

func TestWrite_StrukturBerkas(t *testing.T) {
	b := render(t)
	s := string(b)
	if !strings.HasPrefix(s, "%PDF-1.4\n") {
		t.Error("berkas harus diawali header %PDF-1.4")
	}
	if !strings.HasSuffix(s, "%%EOF\n") {
		t.Error("berkas harus diakhiri penanda EOF")
	}
	for _, want := range []string{"/Type /Catalog", "/Type /Pages", "/Count 2", "/Type /Page ", "trailer", "startxref"} {
		if !strings.Contains(s, want) {
			t.Errorf("objek %q tidak ditemukan", want)
		}
	}
}

// Tabel xref harus menunjuk ke posisi byte objek yang benar, kalau tidak pembaca PDF menolak
// berkasnya. Ini bagian yang paling mudah salah saat menulis PDF sendiri.
func TestWrite_XrefMenunjukObjekYangBenar(t *testing.T) {
	b := render(t)
	i := bytes.LastIndex(b, []byte("startxref\n"))
	if i < 0 {
		t.Fatal("startxref tidak ada")
	}
	start, ok := leadingInt(b[i+len("startxref\n"):])
	if !ok {
		t.Fatal("startxref tidak terbaca")
	}
	if !bytes.HasPrefix(b[start:], []byte("xref\n")) {
		t.Fatalf("startxref menunjuk ke %d, di sana bukan tabel xref", start)
	}
	// Setiap offset di tabel xref harus mendarat tepat di awal "<n> 0 obj".
	// Tata letak: lines[0]="xref", lines[1]="0 N", lines[2]=entri bebas, lalu objek 1 dst.
	lines := bytes.Split(b[start:], []byte("\n"))
	// lines[1] berbunyi "0 N": N = jumlah entri, termasuk entri bebas objek 0.
	count, ok := leadingIntAfterSpace(lines[1])
	if !ok || count < 2 {
		t.Fatalf("header xref tidak terbaca: %q", lines[1])
	}
	for n := 1; n < count && n+2 < len(lines); n++ {
		off, ok := leadingInt(lines[n+2])
		if !ok || off == 0 {
			continue
		}
		want := []byte(strconv.Itoa(n) + " 0 obj")
		if !bytes.HasPrefix(b[off:], want) {
			t.Errorf("offset objek %d = %d, di sana bukan %q", n, off, want)
		}
	}
}

func TestEscape_KarakterKhususTidakMerusakStringPDF(t *testing.T) {
	d := pdf.New()
	d.Text(40, 40, `Toko (Senja) \ "Kopi"`, pdf.Regular, 10, pdf.Black, pdf.Left)
	var buf bytes.Buffer
	if err := d.Write(&buf); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, `\(Senja\)`) {
		t.Error("tanda kurung harus di-escape agar literal string PDF tidak terpotong")
	}
	if !strings.Contains(s, `\\`) {
		t.Error("backslash harus di-escape")
	}
}

func TestWidth_RataKananDanPemotongan(t *testing.T) {
	// "WWW" jauh lebih lebar dari "iii" pada Helvetica; kalau sama, tabel lebar tidak terpakai.
	if pdf.Width("WWW", pdf.Regular, 10) <= pdf.Width("iii", pdf.Regular, 10) {
		t.Error("lebar karakter tidak dibedakan")
	}
	if got := pdf.Width("", pdf.Regular, 10); got != 0 {
		t.Errorf("teks kosong lebarnya harus 0, got %v", got)
	}
	long := "Kopi rapat bulanan untuk divisi pemasaran dan penjualan"
	cut := pdf.Truncate(long, pdf.Regular, 9, 120)
	if pdf.Width(cut, pdf.Regular, 9) > 120 {
		t.Errorf("hasil potong %q masih melebihi 120 pt", cut)
	}
	if !strings.HasSuffix(cut, "...") {
		t.Errorf("teks terpotong harus berakhiran elipsis, got %q", cut)
	}
	if short := pdf.Truncate("Kopi", pdf.Regular, 9, 120); short != "Kopi" {
		t.Errorf("teks yang muat tidak boleh diubah, got %q", short)
	}
}

func TestWrapText_SemuaBarisMuat(t *testing.T) {
	s := "Jl. Senopati Raya No. 45, Kebayoran Baru, Jakarta Selatan 12190"
	lines := pdf.WrapText(s, pdf.Regular, 9, 150, 0)
	if len(lines) < 2 {
		t.Fatalf("teks panjang harus dipecah, got %d baris", len(lines))
	}
	for _, l := range lines {
		if w := pdf.Width(l, pdf.Regular, 9); w > 150 {
			t.Errorf("baris %q lebarnya %v, melebihi 150", l, w)
		}
	}
	if got := pdf.WrapText(s, pdf.Regular, 9, 150, 1); len(got) != 1 {
		t.Errorf("maxLines harus membatasi jumlah baris, got %d", len(got))
	}
}

// winAnsi: rune di luar Latin-1 tidak boleh menghasilkan byte multi-karakter yang membuat
// panjang stream tidak cocok dengan /Length.
func TestWinAnsi_PanjangStreamCocok(t *testing.T) {
	d := pdf.New()
	d.Text(40, 40, "Kopi — “Senja” • 日本", pdf.Regular, 10, pdf.Black, pdf.Left)
	var buf bytes.Buffer
	if err := d.Write(&buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("Kopi - ")) {
		t.Error("em dash harus diganti tanda hubung biasa")
	}
	assertStreamLengths(t, buf.Bytes())
}

func TestWrite_LengthStreamAkurat(t *testing.T) { assertStreamLengths(t, render(t)) }

// assertStreamLengths memastikan /Length tiap stream sama dengan byte sebenarnya; kalau
// meleset, pembaca PDF membaca melewati batas dan berkasnya dianggap rusak.
func assertStreamLengths(t *testing.T, b []byte) {
	t.Helper()
	rest := b
	for {
		i := bytes.Index(rest, []byte("<< /Length "))
		if i < 0 {
			return
		}
		rest = rest[i+len("<< /Length "):]
		n, ok := leadingInt(rest)
		if !ok {
			t.Fatal("/Length tidak terbaca")
		}
		s := bytes.Index(rest, []byte("stream\n"))
		e := bytes.Index(rest, []byte("\nendstream"))
		if s < 0 || e < 0 {
			t.Fatal("stream tidak lengkap")
		}
		if got := e - (s + len("stream\n")) + 1; got != n {
			t.Errorf("/Length = %d, panjang stream sebenarnya %d", n, got)
		}
		rest = rest[e:]
	}
}

// leadingIntAfterSpace membaca bilangan kedua pada baris "0 N".
func leadingIntAfterSpace(b []byte) (int, bool) {
	i := bytes.IndexByte(b, ' ')
	if i < 0 {
		return 0, false
	}
	return leadingInt(b[i+1:])
}

// leadingInt membaca bilangan desimal di awal b. Dipakai untuk membaca startxref, offset,
// dan /Length langsung dari byte berkas tanpa mengurai PDF sepenuhnya.
func leadingInt(b []byte) (int, bool) {
	i := 0
	for i < len(b) && b[i] >= '0' && b[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, false
	}
	n, err := strconv.Atoi(string(b[:i]))
	return n, err == nil
}

func TestLandscapeMediaBoxDanKoordinat(t *testing.T) {
	d := pdf.NewLandscape()
	if d.Width() != pdf.A4Height || d.Height() != pdf.A4Width {
		t.Fatalf("ukuran %vx%v", d.Width(), d.Height())
	}
	d.Text(10, 20, "Hai", pdf.Regular, 10, pdf.Black, pdf.Left)
	var buf bytes.Buffer
	if err := d.Write(&buf); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "/MediaBox [0 0 841.89 595.28]") {
		t.Error("MediaBox bukan lanskap")
	}
	// y dihitung dari atas halaman lanskap: 595.28 - 20.
	if !strings.Contains(s, "10.00 575.28 Td") {
		t.Error("koordinat y tidak memakai tinggi halaman lanskap")
	}
}
