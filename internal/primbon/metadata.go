package primbon

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Metadata is a small optional overlay; document files remain in REGULASI_DIR.
type Metadata struct {
	Tag              []string `json:"tag,omitempty"`
	Status           string   `json:"status,omitempty"`
	Perubahan        string   `json:"perubahan,omitempty"`
	MulaiBerlaku     string   `json:"mulai_berlaku,omitempty"`
	StatusPada       string   `json:"status_pada,omitempty"`
	DiverifikasiPada string   `json:"diverifikasi_pada,omitempty"`
	SumberVerifikasi string   `json:"sumber_verifikasi,omitempty"`
}

func ReadMetadata(dir string) (map[string]Metadata, error) {
	f, err := os.Open(filepath.Join(dir, "metadata.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := json.NewDecoder(io.LimitReader(f, 8<<20))
	dec.DisallowUnknownFields()
	var entries map[string]Metadata
	if err = dec.Decode(&entries); err != nil {
		return nil, fmt.Errorf("metadata.json: %w", err)
	}
	var trailing any
	if err = dec.Decode(&trailing); err != io.EOF {
		return nil, fmt.Errorf("metadata.json: data tambahan tidak valid")
	}
	for id, m := range entries {
		for _, date := range []string{m.StatusPada, m.DiverifikasiPada, m.MulaiBerlaku} {
			if date != "" && tanggal(date) == "" {
				return nil, fmt.Errorf("metadata.json %s: tanggal tidak valid", id)
			}
		}
		if m.SumberVerifikasi != "" {
			u, e := url.Parse(m.SumberVerifikasi)
			if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
				return nil, fmt.Errorf("metadata.json %s: sumber verifikasi wajib URL HTTPS", id)
			}
		}
		if len(m.Tag) > 20 {
			return nil, fmt.Errorf("metadata.json %s: maksimal 20 tag", id)
		}
		seen := map[string]bool{}
		var tags []string
		for _, tag := range m.Tag {
			tag = strings.TrimSpace(tag)
			if len([]rune(tag)) > 80 {
				return nil, fmt.Errorf("metadata.json %s: tag maksimal 80 karakter", id)
			}
			key := strings.ToLower(tag)
			if tag != "" && !seen[key] {
				tags = append(tags, tag)
				seen[key] = true
			}
		}
		m.Tag = tags
		m.StatusPada = tanggal(m.StatusPada)
		m.DiverifikasiPada = tanggal(m.DiverifikasiPada)
		m.MulaiBerlaku = tanggal(m.MulaiBerlaku)
		m.Status = statusRapi(m.Status)
		m.Perubahan = strings.TrimSpace(m.Perubahan)
		entries[id] = m
	}
	return entries, nil
}
