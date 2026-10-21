package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"pajak/internal/ai"
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

// bacaTabel membaca .xlsx, .csv, maupun PDF e-statement jadi baris-baris sel.
func bacaTabel(nama string, data []byte, sandi string) ([][]string, error) {
	if bytes.HasPrefix(data, []byte("%PDF")) {
		baris, err := barisPDF(data, sandi)
		if err != nil {
			return nil, err
		}
		rows := finance.MutasiDariTeks(baris)
		if rows == nil {
			return nil, errors.New("tidak ada baris transaksi di PDF ini. Pastikan ini PDF mutasi / rekening koran (baris transaksi diawali tanggal), atau pakai mutasi .xlsx / .csv")
		}
		return rows, nil
	}
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

// maxMutasiAI: panggilan AI pengelompokan mutasi per usaha per 24 jam; maxPanggilanPerUnggah
// membatasi satu berkas besar (60 kelompok per panggilan).
const (
	maxMutasiAI           = 30
	maxPanggilanPerUnggah = 3
)

// handleMutasiPreview: tebakan kata kunci, lalu baris yang masih "rendah" dibantu AI (bila aktif
// & kuota ada). Kegagalan AI tidak menggagalkan pratinjau — baris tetap bisa dipilih manual.
func (s *Server) handleMutasiPreview(w http.ResponseWriter, r *http.Request) {
	t := s.tenantOf(r)
	h, ok := t.mutasiPreview(w, r)
	if !ok {
		return
	}
	kel := h.KelompokRendah()
	switch {
	case len(kel) == 0:
	case s.llm == nil || !s.llm.Enabled():
		h.AIPesan = "Bantuan AI belum aktif; pilih jenis baris bertanda Periksa sendiri."
	default:
		p := t.currentProfile()
		usaha := strings.TrimSpace(p.JenisUsaha + " · " + p.NamaUsaha)
		saran := map[string]finance.SaranJenis{}
		for i := 0; i < len(kel) && i/ai.MaksKelompokPerPanggilan < maxPanggilanPerUnggah; i += ai.MaksKelompokPerPanggilan {
			if !s.jatahAI("mutasi|"+strconv.Itoa(t.id), maxMutasiAI) {
				h.AIPesan = "Kuota bantuan AI hari ini habis; sisa baris bertanda Periksa dipilih sendiri."
				break
			}
			hasil, err := ai.KelompokkanMutasi(r.Context(), s.llm, usaha, kel[i:min(i+ai.MaksKelompokPerPanggilan, len(kel))])
			if err != nil {
				h.AIPesan = "Bantuan AI gagal (" + err.Error() + "); pilih jenis baris bertanda Periksa sendiri."
				break
			}
			maps.Copy(saran, hasil)
		}
		if len(kel) > ai.MaksKelompokPerPanggilan*maxPanggilanPerUnggah && h.AIPesan == "" {
			h.AIPesan = "Sebagian baris terlalu banyak untuk dibantu AI sekaligus; sisanya dipilih sendiri."
		}
		h.TerapkanSaran(saran)
	}
	writeJSON(w, h)
}

func (s *tenant) mutasiPreview(w http.ResponseWriter, r *http.Request) (finance.MutasiHasil, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	f, fh, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "unggah berkas mutasi PDF, .xlsx, atau .csv (maks 5 MB) di field file", http.StatusBadRequest)
		return finance.MutasiHasil{}, false
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "gagal membaca berkas", http.StatusBadRequest)
		return finance.MutasiHasil{}, false
	}
	rows, err := bacaTabel(fh.Filename, data, r.FormValue("sandi"))
	if err != nil && bytes.HasPrefix(data, []byte("%PDF")) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return finance.MutasiHasil{}, false
	}
	if err != nil || len(rows) == 0 {
		http.Error(w, "berkas tidak terbaca; unggah hasil unduhan mutasi dari m-banking dalam format PDF, .xlsx, atau .csv", http.StatusBadRequest)
		return finance.MutasiHasil{}, false
	}
	tahun, _ := strconv.Atoi(r.FormValue("tahun"))
	s.mu.RLock()
	existing := append([]finance.Transaction(nil), s.txs...)
	s.mu.RUnlock()
	h := finance.ParseMutasi(rows, tahun, existing)
	if bytes.HasPrefix(data, []byte("%PDF")) {
		// PDF tidak punya kolom sungguhan; susunannya ditebak dari letak teks.
		h.Peringatan = append(h.Peringatan, "Dibaca dari PDF: cocokkan jumlah baris serta total uang masuk dan keluar di atas dengan rekening koran Anda sebelum menyimpan.")
	}
	return h, true
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
	for _, t := range txs {
		if err := s.cekTerbukaLocked(tanggalTx(t)...); err != nil {
			s.mu.Unlock()
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
	}
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
