package rag

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"pajak/internal/primbon"
	"strings"
	"unicode"
)

//go:embed schema.sql
var Schema string

type Store struct{ DB *sql.DB }

func (s Store) EnsureSchema(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, Schema)
	return err
}
func (s Store) Current(ctx context.Context, id, checksum string) (bool, error) {
	var match bool
	err := s.DB.QueryRowContext(ctx, `SELECT state='ready' AND checksum=$2 FROM rag_document WHERE id=$1`, id, checksum).Scan(&match)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return match, err
}

const putDocument = `INSERT INTO rag_document(id,kode,judul,status,berlaku,sumber_url,file,checksum,state,error,metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
 ON CONFLICT(id) DO UPDATE SET kode=EXCLUDED.kode,judul=EXCLUDED.judul,status=EXCLUDED.status,berlaku=EXCLUDED.berlaku,sumber_url=EXCLUDED.sumber_url,file=EXCLUDED.file,checksum=EXCLUDED.checksum,state=EXCLUDED.state,error=EXCLUDED.error,metadata=EXCLUDED.metadata,updated_at=now()`

func docArgs(d Document, hash, state, detail string) []any {
	metadata, _ := json.Marshal(d.Metadata)
	return []any{d.ID, d.Code, d.Title, d.Status, d.Effective, d.URL, d.File, hash, state, detail, string(metadata)}
}

func (s Store) Save(ctx context.Context, d Document, checksum string, pages []Page) error {
	chunks := Split(pages)
	if d.ID == "" || d.Title == "" || checksum == "" || len(chunks) == 0 {
		return errors.New("dokumen atau potongan tidak valid")
	}
	for _, c := range chunks {
		if c.Page < 1 {
			return errors.New("nomor halaman tidak valid")
		}
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Serialize concurrent imports of the same ID before checking the stored fingerprint.
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, d.ID); err != nil {
		return err
	}
	var current bool
	err = tx.QueryRowContext(ctx, `SELECT checksum=$2 AND state='ready' FROM rag_document WHERE id=$1`, d.ID, checksum).Scan(&current)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if current {
		return tx.Commit()
	}
	if _, err = tx.ExecContext(ctx, putDocument, docArgs(d, checksum, "ready", "")...); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM rag_chunk WHERE document_id=$1`, d.ID); err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO rag_chunk(document_id,ordinal,page,text,search) VALUES($1,$2,$3,$4,setweight(to_tsvector('simple',$5),'A') || setweight(to_tsvector('simple',$4),'B'))`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for i, c := range chunks {
		if _, err = stmt.ExecContext(ctx, d.ID, i, c.Page, c.Text, d.Code+" "+d.Title+" "+strings.Join(d.Metadata.Tag, " ")); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s Store) Mark(ctx context.Context, d Document, state, detail string) error {
	if state != "missing" && state != "needs_ocr" && state != "error" {
		return errors.New("invalid failure state")
	}
	_, err := s.DB.ExecContext(ctx, putDocument, docArgs(d, "", state, string([]rune(detail)[:min(len([]rune(detail)), 500)]))...)
	return err
}

type Hit struct {
	Ref        string           `json:"ref"`
	DocumentID string           `json:"document_id"`
	Code       string           `json:"kode"`
	Title      string           `json:"judul"`
	Status     string           `json:"status"`
	Effective  string           `json:"berlaku,omitempty"`
	URL        string           `json:"url,omitempty"`
	Page       int              `json:"halaman"`
	Text       string           `json:"teks"`
	Metadata   primbon.Metadata `json:"metadata,omitzero"`
}

func searchQuery(q string) string {
	stop := map[string]bool{}
	for _, word := range strings.Fields("apa itu dan yang di ke dari untuk dengan tentang saya adalah ini bagaimana berapa apakah bisa jelaskan tolong cara") {
		stop[word] = true
	}
	seen := map[string]bool{}
	var terms []string
	for _, word := range strings.FieldsFunc(strings.ToLower(q), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) }) {
		if !stop[word] && !seen[word] {
			terms = append(terms, word)
			seen[word] = true
		}
		if len(terms) == 64 {
			break
		}
	}
	return strings.Join(terms, " | ")
}

func (s Store) Search(ctx context.Context, q string, limit int) ([]Hit, error) {
	q = strings.TrimSpace(q)
	if q == "" || len([]rune(q)) > 2000 {
		return nil, errors.New("pertanyaan wajib diisi, maksimal 2000 karakter")
	}
	if limit < 1 || limit > 10 {
		limit = 6
	}
	query := searchQuery(q)
	if query == "" {
		return []Hit{}, nil
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT d.id,d.kode,d.judul,d.status,d.berlaku,d.sumber_url,c.page,c.text,d.metadata
 FROM rag_chunk c JOIN rag_document d ON d.id=c.document_id
 WHERE d.state='ready' AND c.search @@ to_tsquery('simple',$1)
 ORDER BY ts_rank_cd(c.search,to_tsquery('simple',$1)) DESC,d.id,c.ordinal LIMIT $2`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	hits := []Hit{}
	for rows.Next() {
		var h Hit
		var metadata []byte
		if err = rows.Scan(&h.DocumentID, &h.Code, &h.Title, &h.Status, &h.Effective, &h.URL, &h.Page, &h.Text, &metadata); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(metadata, &h.Metadata); err != nil {
			return nil, err
		}
		h.Ref = fmt.Sprintf("R%d", len(hits)+1)
		u, e := url.Parse(h.URL)
		if e != nil || u.Scheme != "https" || u.User != nil {
			h.URL = ""
		}
		hits = append(hits, h)
	}
	return hits, rows.Err()
}

type Stats struct {
	Ready    int `json:"ready"`
	Missing  int `json:"missing"`
	NeedsOCR int `json:"needs_ocr"`
	Failed   int `json:"error"`
	Chunks   int `json:"chunks"`
}

func (s Store) Stats(ctx context.Context) (Stats, error) {
	var st Stats
	err := s.DB.QueryRowContext(ctx, `SELECT count(*) FILTER(WHERE state='ready'),count(*) FILTER(WHERE state='missing'),count(*) FILTER(WHERE state='needs_ocr'),count(*) FILTER(WHERE state='error'),(SELECT count(*) FROM rag_chunk c JOIN rag_document d ON c.document_id=d.id WHERE d.state='ready') FROM rag_document`).Scan(&st.Ready, &st.Missing, &st.NeedsOCR, &st.Failed, &st.Chunks)
	return st, err
}
