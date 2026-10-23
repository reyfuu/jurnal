// rag imports only a trusted local regulation catalog. It never downloads source URLs.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"os"
	"os/signal"
	"pajak/internal/rag"
)

func run(ctx context.Context) error {
	api.DisableConfigDir()
	dir := flag.String("dir", os.Getenv("REGULASI_DIR"), "folder index.csv dan naskah")
	out := flag.String("compressed-dir", "", "folder salinan PDF, wajib di luar folder sumber")
	apply := flag.Bool("apply", false, "simpan indeks dan hasil kompresi; bawaan hanya pratinjau")
	limit := flag.Int("limit", 0, "batas entri katalog untuk uji awal (0 = semua)")
	flag.Parse()
	if *dir == "" {
		return fmt.Errorf("isi --dir atau REGULASI_DIR")
	}
	var index rag.Indexer
	if *apply {
		url := os.Getenv("DATABASE_URL")
		if url == "" {
			return fmt.Errorf("DATABASE_URL wajib untuk --apply")
		}
		db, err := sql.Open("pgx", url)
		if err != nil {
			return err
		}
		defer db.Close()
		db.SetMaxOpenConns(1)
		store := rag.Store{DB: db}
		if err = store.EnsureSchema(ctx); err != nil {
			return fmt.Errorf("skema RAG: %w", err)
		}
		index = store
	}
	enc := json.NewEncoder(os.Stdout)
	summary, err := rag.Import(ctx, index, rag.ImportOptions{Dir: *dir, CompressedDir: *out, Apply: *apply, Limit: *limit}, func(e rag.Event) { enc.Encode(e) })
	enc.Encode(struct {
		Summary rag.Summary `json:"summary"`
	}{summary})
	if err != nil {
		return err
	}
	if summary.Failed > 0 || summary.NeedsOCR > 0 || summary.CompressionFailed > 0 {
		return fmt.Errorf("ada dokumen gagal/perlu OCR atau kompresi gagal; lihat laporan JSON")
	}
	return nil
}
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
