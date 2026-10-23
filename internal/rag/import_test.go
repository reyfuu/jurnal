package rag

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type memoryIndex struct {
	fingerprints map[string]string
	states       map[string]string
	saves        int
}

func (m *memoryIndex) Current(_ context.Context, id, hash string) (bool, error) {
	return m.fingerprints[id] == hash && m.states[id] == "ready", nil
}
func (m *memoryIndex) Save(_ context.Context, d Document, hash string, p []Page) error {
	m.saves++
	m.fingerprints[d.ID] = hash
	m.states[d.ID] = "ready"
	return nil
}
func (m *memoryIndex) Mark(_ context.Context, d Document, state, detail string) error {
	m.states[d.ID] = state
	return nil
}

func TestImportMissingAndResumeWithoutRewriting(t *testing.T) {
	root := t.TempDir()
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("index.csv", "uuid,judul,file\na,Pajak,a.htm\nb,Lain,b.htm\n")
	write("a.htm", "<p>Pasal 1 Pajak UMKM</p>")
	st := &memoryIndex{fingerprints: map[string]string{}, states: map[string]string{}}
	summary, err := Import(context.Background(), st, ImportOptions{Dir: root}, nil)
	if err != nil || st.saves != 0 || len(st.states) != 0 || summary.Planned != 1 || summary.Missing != 1 {
		t.Fatalf("dry run: %+v %v %+v", summary, err, st)
	}
	summary, err = Import(context.Background(), st, ImportOptions{Dir: root, Apply: true}, nil)
	if err != nil || st.saves != 1 || st.states["b"] != "missing" {
		t.Fatalf("import: %+v %v %+v", summary, err, st)
	}
	write("b.htm", "<p>Pasal 2 Ketentuan lain</p>")
	summary, err = Import(context.Background(), st, ImportOptions{Dir: root, Apply: true}, nil)
	if err != nil || st.saves != 2 || summary.Unchanged != 1 {
		t.Fatalf("resume: %+v %v %+v", summary, err, st)
	}
	write("a.htm", "<p>Pasal 3 Pengganti</p>")
	_, err = Import(context.Background(), st, ImportOptions{Dir: root, Apply: true}, nil)
	if err != nil || st.saves != 3 {
		t.Fatal("changed doc not reindexed", err)
	}
}

func TestCompressionRejectsSourceDirectory(t *testing.T) {
	root := t.TempDir()
	if _, err := Compress(context.Background(), filepath.Join(root, "a.pdf"), root); err == nil {
		t.Fatal("accepted source directory")
	}
	if _, err := Import(context.Background(), nil, ImportOptions{Dir: root, Apply: true}, nil); err == nil {
		t.Fatal("apply without DB")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Import(ctx, nil, ImportOptions{Dir: root}, nil); !errors.Is(err, context.Canceled) {
		t.Fatal("cancel ignored", err)
	}
}

func TestImportScanStillCompresses(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "index.csv"), []byte("uuid,judul,file\nscan,Dokumen scan,a.pdf\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "a.pdf"), samplePDF(""), 0600); err != nil {
		t.Fatal(err)
	}
	st := &memoryIndex{fingerprints: map[string]string{}, states: map[string]string{}}
	summary, err := Import(context.Background(), st, ImportOptions{Dir: source, Apply: true, CompressedDir: filepath.Join(root, "compressed")}, nil)
	if err != nil || summary.NeedsOCR != 1 || summary.Compressed != 1 || st.saves != 0 || st.states["scan"] != "needs_ocr" {
		t.Fatalf("scan compression: %+v %v %+v", summary, err, st)
	}
}
