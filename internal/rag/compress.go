package rag

import (
	"context"
	"errors"
	"fmt"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type Compression struct {
	File   string `json:"file,omitempty"`
	Before int64  `json:"before"`
	After  int64  `json:"after"`
	Saved  bool   `json:"saved"`
}

func canonicalPath(p string) (string, error) {
	p, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(p)
	if err == nil {
		return resolved, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	parent := filepath.Dir(p)
	if parent == p {
		return "", err
	}
	parent, err = canonicalPath(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(p)), nil
}
func separateDir(root, out string) error {
	a, err := canonicalPath(root)
	if err != nil {
		return err
	}
	b, err := canonicalPath(out)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(a, b)
	if err != nil {
		return err
	}
	if rel == "." || filepath.IsLocal(rel) {
		return errors.New("folder PDF terkompresi wajib di luar folder sumber")
	}
	return nil
}

// Compress publishes only a smaller, readable derivative. Originals are never rewritten.
func Compress(ctx context.Context, source, outDir string) (result Compression, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("kompresi PDF gagal: %v", r)
		}
	}()
	if err = separateDir(filepath.Dir(source), outDir); err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	hash, err := ContentHash(source)
	if err != nil {
		return result, err
	}
	info, err := os.Stat(source)
	if err != nil {
		return result, err
	}
	result.Before = info.Size()
	original, err := Extract(ctx, source)
	if err != nil && !errors.Is(err, ErrOCR) {
		return result, err
	}
	if err = os.MkdirAll(outDir, 0700); err != nil {
		return result, err
	}
	tmp, err := os.CreateTemp(outDir, ".rag-pdf-*.pdf")
	if err != nil {
		return result, err
	}
	name := tmp.Name()
	tmp.Close()
	defer os.Remove(name)
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	if err = api.OptimizeFile(source, name, conf); err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	if err = api.ValidateFile(name, conf); err != nil {
		return result, err
	}
	derivative, err := Extract(ctx, name)
	if err != nil && !errors.Is(err, ErrOCR) {
		return result, err
	}
	normalize := func(pages []Page) []Page {
		out := make([]Page, len(pages))
		for i, p := range pages {
			out[i] = Page{p.Number, strings.Join(strings.Fields(p.Text), " ")}
		}
		return out
	}
	if !reflect.DeepEqual(normalize(original), normalize(derivative)) {
		return result, errors.New("kompresi mengubah jumlah halaman atau teks; hasil ditolak")
	}
	after, err := os.Stat(name)
	if err != nil {
		return result, err
	}
	result.After = after.Size()
	if result.After >= result.Before {
		return result, nil
	}
	if current, e := ContentHash(source); e != nil || current != hash {
		return result, errors.New("naskah berubah saat dikompresi; hasil ditolak")
	}
	target := filepath.Join(outDir, hash+".pdf")
	// Hard-link publication cannot overwrite an existing original or derivative.
	if err = os.Link(name, target); err != nil {
		if !os.IsExist(err) {
			return result, err
		}
		cached, e := Extract(ctx, target)
		if e != nil && !errors.Is(e, ErrOCR) {
			return result, e
		}
		if !reflect.DeepEqual(normalize(original), normalize(cached)) {
			return result, errors.New("salinan kompresi yang ada tidak cocok")
		}
		cachedInfo, e := os.Stat(target)
		if e != nil {
			return result, e
		}
		result.After = cachedInfo.Size()
		if result.After >= result.Before {
			return result, nil
		}
	}
	result.File = target
	result.Saved = true
	return result, nil
}
