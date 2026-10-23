package rag

import (
	"context"
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"os"
	"pajak/internal/primbon"
	"strings"
	"testing"
)

func testStore(t *testing.T) Store {
	t.Helper()
	url := os.Getenv("RAG_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("RAG_TEST_DATABASE_URL not set (dedicated disposable database only)")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err = db.Exec(`DROP TABLE IF EXISTS rag_chunk, rag_document CASCADE`); err != nil {
		t.Fatal(err)
	}
	s := Store{DB: db}
	if err = s.EnsureSchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestStoreAtomicIncrementalRetrieval(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	d := Document{ID: "id-1", Code: "PP 55/2022", Title: "PPh Final UMKM", Status: "Diubah", URL: "https://jdih.kemenkeu.go.id/a", File: "a.pdf"}
	d.Metadata = primbon.Metadata{Tag: []string{"Penggajian"}, StatusPada: "2026-10-23"}
	hash := Fingerprint(d, "content-v1")
	pages := []Page{{Number: 2, Text: "Pasal 4 Pajak UMKM omzet penghasilan"}}
	if err := s.Save(ctx, d, hash, pages); err != nil {
		t.Fatal(err)
	}
	ready, err := s.Current(ctx, d.ID, hash)
	if err != nil || !ready {
		t.Fatal(ready, err)
	}
	hits, err := s.Search(ctx, "UMKM", 5)
	if err != nil || len(hits) != 1 || hits[0].Page != 2 || hits[0].Status != "Diubah" {
		t.Fatalf("hits: %+v %v", hits, err)
	}
	tagHits, err := s.Search(ctx, "Penggajian", 5)
	if err != nil || len(tagHits) != 1 || tagHits[0].Metadata.StatusPada != "2026-10-23" {
		t.Fatalf("metadata/tag retrieval: %+v %v", tagHits, err)
	}
	if _, err = s.DB.Exec(`CREATE FUNCTION rag_reject() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'do not rewrite'; END $$; CREATE TRIGGER reject_rewrite BEFORE DELETE ON rag_chunk FOR EACH ROW EXECUTE FUNCTION rag_reject()`); err != nil {
		t.Fatal(err)
	}
	if err = s.Save(ctx, d, hash, pages); err != nil {
		t.Fatalf("unchanged document rewritten: %v", err)
	}
	if err = s.Save(ctx, d, "new-hash", []Page{{Number: 1, Text: "Pengganti"}}); err == nil {
		t.Fatal("expected rollback")
	}
	hits, err = s.Search(ctx, "omzet", 5)
	if err != nil || len(hits) != 1 {
		t.Fatalf("old evidence lost on rollback: %+v %v", hits, err)
	}
	if _, err = s.DB.Exec(`DROP TRIGGER reject_rewrite ON rag_chunk; DROP FUNCTION rag_reject()`); err != nil {
		t.Fatal(err)
	}
	if err = s.Mark(ctx, d, "needs_ocr", "scan has no text"); err != nil {
		t.Fatal(err)
	}
	hits, err = s.Search(ctx, "omzet", 5)
	if err != nil || len(hits) != 0 {
		t.Fatalf("non-ready evidence leaked: %+v %v", hits, err)
	}
	if err = s.Save(ctx, d, "new-hash", []Page{{Number: 1, Text: "Ketentuan pengganti"}}); err != nil {
		t.Fatal(err)
	}
	hits, err = s.Search(ctx, "omzet", 5)
	if err != nil || len(hits) != 0 {
		t.Fatal("old chunks retained", err)
	}
	hits, err = s.Search(ctx, "pengganti", 5)
	if err != nil || len(hits) != 1 {
		t.Fatalf("updated evidence missing: %+v %v", hits, err)
	}
	if _, err = s.Search(ctx, strings.Repeat("a", 2001), 5); err == nil {
		t.Fatal("unbounded query")
	}
	stats, err := s.Stats(ctx)
	if err != nil || stats.Ready != 1 || stats.Chunks != 1 {
		t.Fatal(stats, err)
	}
}

func BenchmarkSplit1512Documents(b *testing.B) {
	pages := []Page{{Number: 1, Text: strings.Repeat("Pasal 1 penghasilan pajak UMKM. ", 400)}}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 1512; j++ {
			Split(pages)
		}
	}
}
