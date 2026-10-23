// Package rag indexes the shared regulation corpus, separately from tenant data.
package rag

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/ledongthuc/pdf"
	"golang.org/x/net/html"
	"pajak/internal/primbon"
)

const ChunkRunes = 1600
const MaxFileBytes = 128 << 20
const MaxTextBytes = 16 << 20
const extractorVersion = "page-v1"

var ErrOCR = errors.New("PDF tidak memiliki teks; perlu OCR sebelum diindeks")

type Document struct {
	ID        string           `json:"id"`
	Code      string           `json:"kode"`
	Title     string           `json:"judul"`
	Status    string           `json:"status"`
	Effective string           `json:"berlaku"`
	URL       string           `json:"url"`
	File      string           `json:"file"`
	Metadata  primbon.Metadata `json:"metadata,omitzero"`
}
type Page struct {
	Number int
	Text   string
}
type Chunk struct {
	Page int
	Text string
}

func Split(pages []Page) []Chunk {
	var out []Chunk
	for _, p := range pages {
		text := []rune(strings.TrimSpace(p.Text))
		for len(text) > 0 {
			end := min(ChunkRunes, len(text))
			if end < len(text) {
				for i := end; i > ChunkRunes/2; i-- {
					if unicode.IsSpace(text[i-1]) {
						end = i
						break
					}
				}
			}
			if part := strings.TrimSpace(string(text[:end])); part != "" {
				out = append(out, Chunk{p.Number, part})
			}
			text = text[end:]
		}
	}
	return out
}

func ReadCatalog(root string) ([]Document, error) {
	metadata, err := primbon.ReadMetadata(root)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(filepath.Join(root, "index.csv"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	head, err := r.Read()
	if err != nil {
		return nil, err
	}
	columns := map[string]int{}
	for i, h := range head {
		columns[strings.TrimPrefix(strings.TrimSpace(h), "\ufeff")] = i
	}
	for _, required := range []string{"uuid", "judul", "file"} {
		if _, ok := columns[required]; !ok {
			return nil, fmt.Errorf("index.csv tanpa kolom %s", required)
		}
	}
	seen := map[string]bool{}
	var docs []Document
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("index.csv: %w", err)
		}
		get := func(k string) string {
			i, ok := columns[k]
			if !ok || i >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[i])
		}
		d := Document{ID: get("uuid"), Code: get("kode"), Title: get("judul"), File: get("file"), Status: get("status"), Effective: get("berlaku"), URL: get("jdih_url")}
		d.Metadata = metadata[d.ID]
		if d.Metadata.MulaiBerlaku != "" {
			d.Effective = d.Metadata.MulaiBerlaku
		}
		if d.Metadata.Status != "" {
			d.Status = d.Metadata.Status
		}
		if d.URL == "" {
			d.URL = get("sumber_url")
		}
		if d.ID == "" || d.Title == "" || seen[d.ID] {
			return nil, fmt.Errorf("uuid/judul kosong atau uuid duplikat: %q", d.ID)
		}
		seen[d.ID] = true
		docs = append(docs, d)
	}
	return docs, nil
}

// SourcePath checks both lexical traversal and symlinks before opening a corpus file.
func SourcePath(root, name string) (string, error) {
	if !filepath.IsLocal(name) || strings.Contains(name, ":") || strings.Contains(name, `\`) {
		return "", fmt.Errorf("path naskah tidak aman: %q", name)
	}
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", err
	}
	p, err := filepath.EvalSymlinks(filepath.Join(root, name))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, p)
	if err != nil || !filepath.IsLocal(rel) {
		return "", errors.New("naskah berada di luar folder katalog")
	}
	return p, nil
}

func ContentHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return "", err
	}
	if !st.Mode().IsRegular() || st.Size() > MaxFileBytes {
		return "", errors.New("naskah bukan berkas biasa atau melebihi 128 MiB")
	}
	h := sha256.New()
	if _, err = io.Copy(h, io.LimitReader(f, MaxFileBytes+1)); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func Fingerprint(d Document, content string) string {
	raw, _ := json.Marshal(d)
	sum := sha256.Sum256(append(raw, []byte("\n"+extractorVersion+"\n"+content)...))
	return hex.EncodeToString(sum[:])
}

func Extract(ctx context.Context, path string) (pages []Page, err error) {
	defer func() {
		if r := recover(); r != nil {
			pages = nil
			err = fmt.Errorf("naskah gagal diurai: %v", r)
		}
	}()
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxFileBytes {
		return nil, errors.New("naskah melebihi 128 MiB")
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".htm", ".html":
		z := html.NewTokenizer(bytes.NewReader(data))
		var text strings.Builder
		skip := 0
		for {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			tt := z.Next()
			switch tt {
			case html.ErrorToken:
				if z.Err() != io.EOF {
					return nil, z.Err()
				}
				s := strings.TrimSpace(text.String())
				if s == "" {
					return nil, errors.New("HTML tanpa teks")
				}
				return []Page{{1, s}}, nil
			case html.StartTagToken:
				name, _ := z.TagName()
				tag := string(name)
				if tag == "script" || tag == "style" || tag == "head" {
					skip++
				}
				if skip == 0 && (tag == "p" || tag == "br" || tag == "div" || tag == "tr" || tag == "li") {
					text.WriteByte('\n')
				}
			case html.EndTagToken:
				name, _ := z.TagName()
				tag := string(name)
				if skip > 0 && (tag == "script" || tag == "style" || tag == "head") {
					skip--
				}
				if skip == 0 {
					text.WriteByte(' ')
				}
			case html.TextToken:
				if skip == 0 {
					text.Write(z.Text())
					if text.Len() > MaxTextBytes {
						return nil, errors.New("teks melebihi 16 MiB")
					}
				}
			}
		}
	case ".pdf":
		rd, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, err
		}
		if rd.NumPage() > 10000 {
			return nil, errors.New("PDF melebihi 10000 halaman")
		}
		size := 0
		for n := 1; n <= rd.NumPage(); n++ {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			text, err := rd.Page(n).GetPlainText(nil)
			if err != nil {
				return nil, err
			}
			size += len(text)
			if size > MaxTextBytes {
				return nil, errors.New("teks melebihi 16 MiB")
			}
			pages = append(pages, Page{n, strings.TrimSpace(text)})
		}
		if len(Split(pages)) == 0 {
			return pages, ErrOCR
		}
		return pages, nil
	default:
		return nil, errors.New("format naskah harus PDF atau HTML")
	}
}
