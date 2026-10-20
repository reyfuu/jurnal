package pengetahuan

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)

func artikelPPh() Artikel {
	return Artikel{Judul: "PPh Final 0,5% untuk UMKM", Kategori: "PPh Final UMKM", DasarHukum: "PP 55/2022",
		Tag: []string{"omzet 4,8 M", "OP"}, Isi: "Tarif PPh final 0,5% dari omzet berlaku untuk wajib pajak dengan peredaran bruto sampai Rp4,8 miliar setahun.", TampilKlien: true}
}

func TestSimpanArtikel_DraftLaluTerbit(t *testing.T) {
	st := NewState()
	a, err := st.SimpanArtikel(artikelPPh(), false, "Budi", t0)
	if err != nil || a.ID != 1 || a.Status != Draft || a.Diverifikasi != "" {
		t.Fatalf("draft = %+v %v", a, err)
	}
	a.Judul = "PPh Final 0,5% untuk UMKM: siapa yang boleh"
	b, err := st.SimpanArtikel(a, true, "Andi", t0)
	if err != nil || b.ID != 1 || b.Status != Terbit || b.Diverifikasi != "Andi" || len(st.Artikel) != 1 || b.Oleh != "Budi" {
		t.Fatalf("terbit = %+v %v", b, err)
	}
	for nama, x := range map[string]Artikel{
		"tanpa judul":    {Kategori: "PPN", Isi: "x"},
		"tanpa isi":      {Judul: "x", Kategori: "PPN"},
		"kategori asing": {Judul: "x", Kategori: "Zakat", Isi: "x"},
		"id tak ada":     {ID: 9, Judul: "x", Kategori: "PPN", Isi: "x"},
	} {
		if _, err := st.SimpanArtikel(x, false, "Budi", t0); !errors.Is(err, ErrTidakValid) {
			t.Errorf("%s: err = %v", nama, err)
		}
	}
}

func TestSumber_HanyaTerverifikasiDipakai(t *testing.T) {
	st := NewState()
	s, err := st.TambahSumber(Sumber{Judul: "Batas omzet", Jenis: "manual", Isi: "Orang pribadi tidak kena PPh final atas omzet sampai Rp500 juta setahun."}, "Andi", t0)
	if err != nil || s.Status != Menunggu || s.DipakaiAI {
		t.Fatalf("sumber baru = %+v %v", s, err)
	}
	if err := st.SetDipakai(s.ID, true); !errors.Is(err, ErrTidakValid) {
		t.Errorf("pakai sebelum verifikasi: err = %v", err)
	}
	if len(st.Potongan()) != 0 {
		t.Error("sumber menunggu ikut dipakai AI")
	}
	if err := st.Verifikasi(s.ID, "Sari"); err != nil {
		t.Fatal(err)
	}
	p := st.Potongan()
	if len(p) != 1 || p[0].Ref != "KB-001" {
		t.Fatalf("potongan = %+v", p)
	}
	_ = st.SetDipakai(s.ID, false)
	if len(st.Potongan()) != 0 {
		t.Error("sumber yang dilepas tetap dipakai")
	}
}

func TestPotongan_ArtikelTerbitSaja(t *testing.T) {
	st := NewState()
	st.SimpanArtikel(artikelPPh(), false, "Budi", t0)
	if len(st.Potongan()) != 0 {
		t.Fatal("draft artikel ikut dipakai AI")
	}
	a := st.Artikel[0]
	st.SimpanArtikel(a, true, "Andi", t0)
	p := st.Potongan()
	if len(p) != 1 || p[0].Ref != "A-001" || !strings.Contains(p[0].Teks, "PP 55/2022") {
		t.Fatalf("potongan artikel = %+v", p)
	}
}

func TestPecah(t *testing.T) {
	isi := "Paragraf satu pendek.\n\nParagraf dua juga pendek.\n\n" + strings.Repeat("Kalimat panjang tentang pajak. ", 60)
	bag := Pecah(isi, 300)
	if len(bag) < 3 {
		t.Fatalf("bagian = %d", len(bag))
	}
	for _, b := range bag {
		if len([]rune(b)) > 300 || strings.TrimSpace(b) == "" {
			t.Errorf("bagian %d karakter: %q", len([]rune(b)), b[:min(40, len(b))])
		}
	}
	if !strings.Contains(bag[0], "Paragraf satu") || !strings.Contains(bag[0], "Paragraf dua") {
		t.Errorf("paragraf pendek harus digabung: %q", bag[0])
	}
}

func TestCari_RelevanDulu(t *testing.T) {
	pot := []Potongan{
		{Ref: "A-001", Judul: "PPh 21 karyawan", Teks: "Pemotongan PPh 21 bulanan memakai tarif efektif rata-rata."},
		{Ref: "A-002", Judul: "PPh Final UMKM", Teks: "Tarif PPh final 0,5% dari omzet untuk UMKM dengan omzet sampai 4,8 miliar."},
		{Ref: "KB-001", Judul: "Batas omzet", Teks: "Orang pribadi tidak kena PPh final atas omzet sampai Rp500 juta."},
	}
	got := Cari(pot, "warung kopi omzet 3 miliar masih boleh pakai pph final?", 2)
	if len(got) != 2 || got[0].Ref != "A-002" || got[0].Skor <= 0 {
		t.Fatalf("hasil = %+v", got)
	}
	if Cari(pot, "yang dan di", 3) != nil {
		t.Error("kata umum saja harus tanpa hasil")
	}
}

func TestParseExcel(t *testing.T) {
	isi, topik, err := ParseExcel([][]string{
		{"Judul", "Isi", "Topik", "Sumber"},
		{"Transfer ke pemilik", "Catat sebagai prive, bukan biaya.", "Pembukuan", "Kebijakan internal"},
		{"", "", "", ""},
		{"Komisi GoFood", "Catat sebagai potongan aplikasi.", "Pembukuan, Marketplace", ""},
	})
	if err != nil || strings.Count(isi, "\n\n") != 1 || !strings.Contains(isi, "Transfer ke pemilik: Catat sebagai prive") || len(topik) != 2 {
		t.Fatalf("isi=%q topik=%v err=%v", isi, topik, err)
	}
	if _, _, err := ParseExcel([][]string{{"Judul", "Isi"}}); err == nil {
		t.Error("tanpa baris data harus ditolak")
	}
}

// pdfMini: PDF satu halaman dengan teks t (font standar, stream tanpa kompresi).
func pdfMini(t string) []byte {
	stream := fmt.Sprintf("BT /F1 12 Tf 72 712 Td (%s) Tj ET", t)
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	offs := make([]int, len(objs))
	for i, o := range objs {
		offs[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, o := range offs {
		fmt.Fprintf(&b, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	return []byte(b.String())
}

func TestTeksPDF(t *testing.T) {
	got, err := TeksPDF(pdfMini("Batas omzet PPh final"))
	if err != nil || !strings.Contains(got, "Batas omzet PPh final") {
		t.Fatalf("teks = %q err = %v", got, err)
	}
	if _, err := TeksPDF([]byte("bukan pdf")); err == nil {
		t.Error("berkas bukan PDF harus ditolak")
	}
	if _, err := TeksPDF(pdfMini("")); !errors.Is(err, ErrPDFKosong) {
		t.Errorf("PDF tanpa teks (hasil scan): err = %v", err)
	}
}

func TestSumber_KategoriIsi(t *testing.T) {
	st := NewState()
	s, err := st.TambahSumber(Sumber{Judul: "Tips SP2DK", Isi: "Siapkan rekap omzet per bulan.", Jenis: "manual", Kategori: "praktik"}, "Budi", time.Now())
	if err != nil || s.Kategori != "praktik" {
		t.Fatalf("praktik: %+v %v", s, err)
	}
	if s, _ := st.TambahSumber(Sumber{Judul: "x", Isi: "y", Jenis: "manual"}, "Budi", time.Now()); s.Kategori != "umum" {
		t.Errorf("kategori kosong = %q, want umum", s.Kategori)
	}
	if _, err := st.TambahSumber(Sumber{Judul: "x", Isi: "y", Jenis: "manual", Kategori: "gosip"}, "Budi", time.Now()); err == nil {
		t.Error("kategori ngawur diterima")
	}
	st.Sumber[0].Status, st.Sumber[0].DipakaiAI = Terverifikasi, true
	if p := st.Potongan(); len(p) == 0 || p[0].Kategori != "Praktik lapangan" {
		t.Errorf("potongan untuk AI tanpa label kategori: %+v", p)
	}
}
