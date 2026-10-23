package rag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestChunksKeepPagesAndUnicode(t *testing.T) {
	pages := []Page{{Number: 1, Text: "Pasal 1\n" + strings.Repeat("Pajak café Indonesia. ", 200)}, {Number: 3, Text: "Pasal 2 Ketentuan lain"}}
	chunks := Split(pages)
	if len(chunks) < 3 {
		t.Fatal("expected bounded chunks")
	}
	var reconstructed strings.Builder
	for _, c := range chunks {
		if !utf8.ValidString(c.Text) || len([]rune(c.Text)) > ChunkRunes {
			t.Fatalf("invalid chunk: %+v", c)
		}
		if c.Page == 1 {
			reconstructed.WriteString(c.Text)
		}
	}
	if !strings.Contains(reconstructed.String(), "café") || chunks[len(chunks)-1].Page != 3 {
		t.Fatal("text or page lost")
	}
	if Split([]Page{{Number: 1, Text: "  "}}) != nil {
		t.Fatal("empty page indexed")
	}
}

func TestCatalogAndSafePath(t *testing.T) {
	root := t.TempDir()
	csv := "uuid,kode,judul,file,status,jdih_url\nid-1,PP 55/2022,Pajak UMKM,umkm/a.htm,Berlaku,https://jdih.kemenkeu.go.id/a\nid-2,PMK 2,Lainnya,,Diubah,\n"
	if err := os.WriteFile(filepath.Join(root, "index.csv"), []byte(csv), 0600); err != nil {
		t.Fatal(err)
	}
	docs, err := ReadCatalog(root)
	if err != nil || len(docs) != 2 {
		t.Fatalf("catalog: %+v %v", docs, err)
	}
	for _, p := range []string{"../secret.pdf", "/etc/passwd", `C:\secret.pdf`, ""} {
		if _, err := SourcePath(root, p); err == nil {
			t.Fatalf("accepted %q", p)
		}
	}
	if err := os.Mkdir(filepath.Join(root, "umkm"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "umkm", "a.htm"), []byte("<html><head><style>hide</style></head><body><p>Pasal 1 &amp; UMKM</p><script>ignore()</script><p>Tarif <b>pajak</b></p></body></html>"), 0600); err != nil {
		t.Fatal(err)
	}
	p, err := SourcePath(root, docs[0].File)
	if err != nil {
		t.Fatal(err)
	}
	pages, err := Extract(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 1 || !strings.Contains(pages[0].Text, "Pasal 1 & UMKM") || strings.Contains(pages[0].Text, "ignore") {
		t.Fatalf("HTML: %+v", pages)
	}
}

func TestFingerprintIncludesMetadata(t *testing.T) {
	a := Document{ID: "a", Title: "PP 55", Status: "Berlaku"}
	b := a
	b.Status = "Dicabut"
	if Fingerprint(a, "abc") == Fingerprint(b, "abc") || Fingerprint(a, "abc") == Fingerprint(a, "def") {
		t.Fatal("metadata or content changes ignored")
	}
}

func TestCatalogOverlayAffectsRAGFingerprint(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.csv"), []byte("uuid,judul,file,status\na,Aturan,a.pdf,Berlaku\n"), 0600); err != nil {
		t.Fatal(err)
	}
	docs, err := ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	before := Fingerprint(docs[0], "same-file")
	if err = os.WriteFile(filepath.Join(root, "metadata.json"), []byte(`{"a":{"status":"Dicabut","tag":["Payroll"]}}`), 0600); err != nil {
		t.Fatal(err)
	}
	docs, err = ReadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if docs[0].Status != "Dicabut" || Fingerprint(docs[0], "same-file") == before {
		t.Fatal("RAG ignored updated catalog metadata")
	}
}

func samplePDF(text string) []byte {
	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	stream := "BT /F1 12 Tf 20 50 Td (" + text + ") Tj ET"
	objects := []string{`<< /Type /Catalog /Pages 2 0 R >>`, `<< /Type /Pages /Kids [3 0 R] /Count 1 >>`, `<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 100] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>`, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream), `<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>`, fmt.Sprintf("<< /Length 60000 >>\nstream\n%s\nendstream", strings.Repeat("x", 60000))}
	for i, o := range objects {
		offsets = append(offsets, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for _, o := range offsets[1:] {
		fmt.Fprintf(&b, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xref)
	return []byte(b.String())
}
func TestPDFCompressionPreservesOriginalAndEvidence(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	if err := os.Mkdir(sourceDir, 0700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(sourceDir, "a.pdf")
	raw := samplePDF("Pasal 1 Pajak UMKM")
	if err := os.WriteFile(source, raw, 0600); err != nil {
		t.Fatal(err)
	}
	pages, err := Extract(context.Background(), source)
	if err != nil || len(pages) != 1 || !strings.Contains(pages[0].Text, "Pajak UMKM") {
		t.Fatalf("extract: %+v %v", pages, err)
	}
	out, err := Compress(context.Background(), source, filepath.Join(root, "compressed"))
	if err != nil {
		t.Fatal(err)
	}
	if !out.Saved || out.After >= out.Before {
		t.Fatalf("not smaller: %+v", out)
	}
	after, err := os.ReadFile(source)
	if err != nil || string(after) != string(raw) {
		t.Fatal("original modified", err)
	}
	if _, err = Compress(context.Background(), source, filepath.Join(root, "compressed")); err != nil {
		t.Fatalf("repeat compression: %v", err)
	}
}
