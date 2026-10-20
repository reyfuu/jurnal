package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"pajak/internal/finance"
	"pajak/internal/xlsx"
)

// Impor mutasi rekening bank. Dua langkah, dan langkah pertama tidak menyimpan apa pun:
// unggah → tebakan jenis per baris ditampilkan untuk dikoreksi → baris yang sudah benar
// dikirim balik untuk dibukukan.
//
// Commit menerima baris hasil koreksi dari klien, bukan membaca ulang berkasnya. Kalau berkas
// dibaca ulang, koreksi pengguna akan hilang dan tebakan mesin yang dibukukan — padahal
// seluruh gunanya langkah pratinjau justru supaya tebakan itu bisa diperbaiki.

// bacaTabel membaca .xlsx maupun .csv jadi baris-baris sel.
func bacaTabel(nama string, data []byte) ([][]string, error) {
	if strings.EqualFold(filepath.Ext(nama), ".csv") || !bytes.HasPrefix(data, []byte("PK")) {
		return bacaCSV(data)
	}
	sheets, err := xlsx.ReadAll(data)
	if err != nil {
		return nil, err
	}
	// Mutasi bank selalu satu sheet; kalau lebih, yang pertama berisi yang dipakai.
	for _, sh := range sheets {
		if len(sh.Rows) > 0 {
			return sh.Rows, nil
		}
	}
	return nil, errors.New("berkas .xlsx tidak berisi baris apa pun")
}

// bacaCSV menebak pemisah kolomnya. Excel berbahasa Indonesia menyimpan CSV dengan titik
// koma, karena koma sudah dipakai sebagai pemisah desimal.
func bacaCSV(data []byte) ([][]string, error) {
	s := string(data)
	if i := strings.IndexByte(s, '\n'); i > 0 {
		baris := s[:i]
		if strings.Count(baris, ";") > strings.Count(baris, ",") {
			return csvRows(s, ';')
		}
		if strings.Count(baris, "\t") > strings.Count(baris, ",") {
			return csvRows(s, '\t')
		}
	}
	return csvRows(s, ',')
}

func csvRows(s string, sep rune) ([][]string, error) {
	r := csv.NewReader(strings.NewReader(s))
	r.Comma = sep
	r.FieldsPerRecord = -1 // cetakan bank sering punya jumlah kolom tidak sama per baris
	r.LazyQuotes = true
	return r.ReadAll()
}

func (s *tenant) handleMutasiPreview(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	f, fh, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "unggah berkas mutasi .xlsx atau .csv (maks 5 MB) di field file", http.StatusBadRequest)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "gagal membaca berkas", http.StatusBadRequest)
		return
	}
	rows, err := bacaTabel(fh.Filename, data)
	if err != nil || len(rows) == 0 {
		http.Error(w, "berkas tidak terbaca; unggah hasil unduhan mutasi dari m-banking dalam format .xlsx atau .csv", http.StatusBadRequest)
		return
	}
	tahun, _ := strconv.Atoi(r.FormValue("tahun"))
	s.mu.RLock()
	existing := append([]finance.Transaction(nil), s.txs...)
	s.mu.RUnlock()
	writeJSON(w, finance.ParseMutasi(rows, tahun, existing))
}

type mutasiCommitReq struct {
	Rows []finance.MutasiRow `json:"rows"`
}

type mutasiCommitResp struct {
	Disimpan int                   `json:"disimpan"`
	Errors   []finance.ImportError `json:"errors"`
	Contoh   []finance.Transaction `json:"contoh"`
}

func (s *tenant) handleMutasiCommit(w http.ResponseWriter, r *http.Request) {
	var req mutasiCommitReq
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 5<<20)).Decode(&req); err != nil {
		http.Error(w, "data baris mutasi tidak valid", http.StatusBadRequest)
		return
	}
	txs, errs := finance.MutasiKeTransaksi(req.Rows)
	if len(txs) == 0 {
		// Tidak ada yang bisa dibukukan: laporkan kenapa, jangan balas "berhasil 0".
		http.Error(w, pesanGagalMutasi(errs), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	for i := range txs {
		txs[i].ID = s.nextTxID
		s.nextTxID++
		s.txs = append(s.txs, txs[i])
	}
	s.mu.Unlock()
	writeJSON(w, mutasiCommitResp{Disimpan: len(txs), Errors: errs, Contoh: txs[:min(5, len(txs))]})
}

func pesanGagalMutasi(errs []finance.ImportError) string {
	if len(errs) == 0 {
		return "tidak ada baris yang dipilih untuk dibukukan"
	}
	pesan := make([]string, 0, len(errs))
	for _, e := range errs[:min(3, len(errs))] {
		pesan = append(pesan, "baris "+strconv.Itoa(e.Baris)+": "+e.Pesan)
	}
	return "tidak ada baris yang bisa dibukukan — " + strings.Join(pesan, "; ")
}
