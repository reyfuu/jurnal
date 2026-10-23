package rag

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Indexer interface {
	Current(context.Context, string, string) (bool, error)
	Save(context.Context, Document, string, []Page) error
	Mark(context.Context, Document, string, string) error
}
type ImportOptions struct {
	Dir           string
	CompressedDir string
	Apply         bool
	Limit         int
}
type Summary struct {
	Indexed           int `json:"indexed"`
	Unchanged         int `json:"unchanged"`
	Missing           int `json:"missing"`
	NeedsOCR          int `json:"needs_ocr"`
	Failed            int `json:"failed"`
	Planned           int `json:"planned"`
	Compressed        int `json:"compressed"`
	CompressionFailed int `json:"compression_failed"`
}
type Event struct {
	ID          string       `json:"id"`
	State       string       `json:"state"`
	Detail      string       `json:"detail,omitempty"`
	Compression *Compression `json:"compression,omitempty"`
}

// Import is deliberately sequential: later runs pick up newly available or changed files.
func Import(ctx context.Context, index Indexer, opts ImportOptions, emit func(Event)) (Summary, error) {
	var sum Summary
	if err := ctx.Err(); err != nil {
		return sum, err
	}
	if opts.Apply && index == nil {
		return sum, errors.New("impor --apply membutuhkan database")
	}
	if opts.Limit < 0 {
		return sum, errors.New("limit tidak boleh negatif")
	}
	if opts.CompressedDir != "" {
		if err := separateDir(opts.Dir, opts.CompressedDir); err != nil {
			return sum, err
		}
	}
	docs, err := ReadCatalog(opts.Dir)
	if err != nil {
		return sum, err
	}
	if opts.Limit > 0 {
		docs = docs[:min(len(docs), opts.Limit)]
	}
	report := func(e Event) {
		if emit != nil {
			emit(e)
		}
	}
	for _, d := range docs {
		if err = ctx.Err(); err != nil {
			return sum, err
		}
		process := func() error {
			docCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()
			mark := func(state string, cause error) error {
				report(Event{ID: d.ID, State: state, Detail: cause.Error()})
				if opts.Apply {
					return index.Mark(docCtx, d, state, cause.Error())
				}
				return nil
			}
			path, e := SourcePath(opts.Dir, d.File)
			if e != nil {
				if d.File == "" || os.IsNotExist(e) {
					sum.Missing++
					return mark("missing", fmt.Errorf("naskah belum tersedia: %s", d.File))
				}
				sum.Failed++
				return mark("error", e)
			}
			hash, e := ContentHash(path)
			if e != nil {
				sum.Failed++
				return mark("error", e)
			}
			fingerprint := Fingerprint(d, hash)
			current := false
			if index != nil {
				current, e = index.Current(docCtx, d.ID, fingerprint)
				if e != nil {
					return e
				}
			}
			if current {
				sum.Unchanged++
				report(Event{ID: d.ID, State: "unchanged"})
			} else if !opts.Apply {
				sum.Planned++
				report(Event{ID: d.ID, State: "planned"})
			} else {
				pages, e := Extract(docCtx, path)
				switch {
				case errors.Is(e, ErrOCR):
					sum.NeedsOCR++
					if e = mark("needs_ocr", e); e != nil {
						return e
					}
				case e != nil:
					sum.Failed++
					return mark("error", e)
				default:
					// A writer replacing a source during extraction must not attach old hashes to new text.
					now, e := ContentHash(path)
					if e != nil || now != hash {
						sum.Failed++
						return mark("error", errors.New("naskah berubah saat diindeks; ulangi impor"))
					}
					if e = index.Save(docCtx, d, fingerprint, pages); e != nil {
						return e
					}
					sum.Indexed++
					report(Event{ID: d.ID, State: "indexed"})
				}
			}
			if opts.Apply && opts.CompressedDir != "" && strings.EqualFold(filepath.Ext(path), ".pdf") {
				result, e := Compress(docCtx, path, opts.CompressedDir)
				if e != nil {
					sum.CompressionFailed++
					report(Event{ID: d.ID, State: "compression_error", Detail: e.Error()})
				} else {
					if result.Saved {
						sum.Compressed++
					}
					report(Event{ID: d.ID, State: "compression_checked", Compression: &result})
				}
			}
			return nil
		}
		if err = process(); err != nil {
			return sum, fmt.Errorf("impor %s: %w", d.ID, err)
		}
	}
	return sum, nil
}
