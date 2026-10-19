package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"pajak/internal/auth"
	"pajak/internal/finance"
)

// ponytail: snapshot JSON atomik (tulis temp lalu rename) untuk 1 usaha & ratusan user;
// ganti ke database saat multi-tenant (roadmap) — tanpa dependensi, image scratch tetap.

type state struct {
	Transactions []finance.Transaction `json:"transactions"`
	NextTxID     int                   `json:"next_tx_id"`
	Auth         auth.StoreSnapshot    `json:"auth"`
}

// dataPath: berkas di $DATA_DIR; kosong = mode in-memory (test, dev lokal).
func dataPath(name string) string {
	if dir := os.Getenv("DATA_DIR"); dir != "" {
		return filepath.Join(dir, name)
	}
	return ""
}

func writeFileAtomic(path string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func loadState() (*state, error) {
	p := dataPath("state.json")
	if p == "" {
		return nil, nil
	}
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var st state
	return &st, json.Unmarshal(b, &st)
}

// save menulis seluruh state; dipanggil setelah setiap perubahan yang berhasil.
func (s *Server) save() {
	p := dataPath("state.json")
	if p == "" {
		return
	}
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	s.mu.RLock()
	st := state{Transactions: s.txs, NextTxID: s.nextTxID, Auth: s.users.Snapshot()}
	err := writeFileAtomic(p, st)
	s.mu.RUnlock()
	if err != nil {
		log.Printf("GAGAL menyimpan state ke %s: %v", p, err)
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// persisting menyimpan state setelah handler pengubah data selesai tanpa error.
func (s *Server) persisting(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		h(rec, r)
		if rec.status < 400 {
			s.save()
		}
	}
}
