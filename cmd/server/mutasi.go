package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"path/filepath"
	"slices"
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

var errPDFTanpaBaris = errors.New("tidak ada baris transaksi di PDF ini. Pastikan ini PDF mutasi / rekening koran (baris transaksi diawali tanggal), atau pakai mutasi .xlsx / .csv")

// maxUnggahMutasi: PDF hasil scan jauh lebih besar dari e-statement biasa.
const maxUnggahMutasi = 15 << 20

// bacaTabel membaca .xlsx, .csv, maupun PDF e-statement jadi baris-baris sel.
//
// peringatan: hasil pemeriksaan saldo berjalan untuk PDF (tabel Excel/CSV dibaca apa adanya).
func bacaTabel(nama string, data []byte, sandi string) (rows [][]string, peringatan []string, err error) {
	if isiPDF(data) {
		baris, err := barisPDF(data, sandi)
		if err != nil {
			return nil, nil, err
		}
		rows, peringatan := finance.MutasiDariTeksCek(baris)
		if rows == nil {
			return nil, nil, errPDFTanpaBaris
		}
		return rows, peringatan, nil
	}
	// Excel berkata sandi bukan zip .xlsx, melainkan berkas OLE terenkripsi (sama dengan .xls lama).
	if bytes.HasPrefix(data, []byte{0xD0, 0xCF, 0x11, 0xE0}) {
		return nil, nil, errExcelTerkunci
	}
	if strings.EqualFold(filepath.Ext(nama), ".csv") || !bytes.HasPrefix(data, []byte("PK")) {
		rows, err := bacaCSV(data)
		return rows, nil, err
	}
	sheets, err := xlsx.ReadAll(data)
	if err != nil {
		return nil, nil, err
	}
	// Mutasi bank selalu satu sheet; kalau lebih, yang pertama berisi yang dipakai.
	for _, sh := range sheets {
		if len(sh.Rows) > 0 {
			return sh.Rows, nil, nil
		}
	}
	return nil, nil, errors.New("berkas .xlsx tidak berisi baris apa pun")
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
	var scan bacaScanFunc
	if s.llmGambar != nil && s.llmGambar.Enabled() {
		scan = func(ctx context.Context, data []byte, sandi string, tahun int) ([][]string, []string, error) {
			return s.bacaMutasiScan(ctx, t.id, data, sandi, tahun)
		}
	}
	h, ok := t.mutasiPreview(w, r, scan)
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

// mutasiPreview membaca berkas mutasi jadi pratinjau. scan (boleh nil) membaca PDF hasil scan
// dan foto mutasi dengan AI.
func (s *tenant) mutasiPreview(w http.ResponseWriter, r *http.Request, scan bacaScanFunc) (finance.MutasiHasil, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUnggahMutasi)
	f, fh, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "unggah berkas mutasi PDF, .xlsx, .csv, atau foto mutasi (maks 15 MB) di field file", http.StatusBadRequest)
		return finance.MutasiHasil{}, false
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "gagal membaca berkas", http.StatusBadRequest)
		return finance.MutasiHasil{}, false
	}
	sandi := r.FormValue("sandi")
	tahun, _ := strconv.Atoi(r.FormValue("tahun"))
	rows, peringatanPDF, err := bacaTabel(fh.Filename, data, sandi)
	gambar := isiGambar(data) != ""
	dariScan := false
	if gambar || (isiPDF(data) && (errors.Is(err, errPDFGambar) || errors.Is(err, errPDFTanpaBaris))) {
		switch {
		case scan != nil:
			rows, peringatanPDF, err = scan(r.Context(), data, sandi, tahun)
			dariScan = err == nil
		case gambar:
			rows, err = nil, errFotoMutasiMati
		}
	}
	if err != nil && (isiPDF(data) || gambar || errors.Is(err, errExcelTerkunci)) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return finance.MutasiHasil{}, false
	}
	if err != nil || len(rows) == 0 {
		http.Error(w, "berkas tidak terbaca; unggah hasil unduhan mutasi dari m-banking dalam format PDF, .xlsx, atau .csv", http.StatusBadRequest)
		return finance.MutasiHasil{}, false
	}
	s.mu.RLock()
	existing := append([]finance.Transaction(nil), s.txs...)
	awal, ditutup := "", s.tutupBuku
	if s.opening != nil {
		awal = s.opening.Tanggal
	}
	s.mu.RUnlock()
	h := finance.ParseMutasi(rows, tahun, existing)
	// Baris yang pasti ditolak saat disimpan tidak dicentang, supaya baris lainnya tetap bisa
	// dibukukan (simpan menolak seluruh kiriman bila ada satu baris yang terkunci).
	var nAwal, nTutup int
	for i := range h.Rows {
		switch r := &h.Rows[i]; {
		case awal != "" && r.Tanggal < awal:
			r.Pilih, nAwal = false, nAwal+1
		case ditutup != "" && r.Tanggal[:min(7, len(r.Tanggal))] <= ditutup:
			r.Pilih, nTutup = false, nTutup+1
		}
	}
	if nAwal > 0 {
		h.Peringatan = append(h.Peringatan, fmt.Sprintf("%d baris bertanggal sebelum tanggal saldo awal (%s) dan tidak dicentang. Mundurkan tanggal saldo awal di menu Saldo Awal bila mutasi lama ini memang mau dicatat.", nAwal, awal))
	}
	if nTutup > 0 {
		h.Peringatan = append(h.Peringatan, fmt.Sprintf("%d baris jatuh di bulan yang bukunya sudah ditutup (sampai %s) dan tidak dicentang. Buka kembali bulan itu di menu Tutup Buku bila memang mau dicatat.", nTutup, finance.LabelPeriode(ditutup)))
	}
	if isiPDF(data) || gambar {
		// PDF tidak punya kolom sungguhan; susunannya ditebak dari letak teks (atau dibaca AI).
		h.Peringatan = append(h.Peringatan, peringatanPDF...)
		if !dariScan {
			h.Peringatan = append(h.Peringatan, "Dibaca dari PDF: cocokkan jumlah baris serta total uang masuk dan keluar di atas dengan rekening koran Anda sebelum menyimpan.")
		}
	}
	return h, true
}

type mutasiCommitReq struct {
	Rows []finance.MutasiRow `json:"rows"`
}

type mutasiCommitResp struct {
	Disimpan int `json:"disimpan"`
	// Dilunasi: baris yang dicatat sebagai pelunasan transaksi belum lunas yang sudah ada.
	Dilunasi int                   `json:"dilunasi"`
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
	var lunasi []finance.MutasiRow
	for _, r := range req.Rows {
		if r.Pilih && r.Lunasi != 0 {
			lunasi = append(lunasi, r)
		}
	}
	if len(txs) == 0 && len(lunasi) == 0 {
		// Tidak ada yang bisa dibukukan: laporkan kenapa, jangan balas "berhasil 0".
		http.Error(w, pesanGagalMutasi(errs), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	for _, t := range txs {
		if err := s.cekTxLocked(t); err != nil {
			s.mu.Unlock()
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
	}
	// Baris yang cocok dengan transaksi belum lunas: dicatat sebagai pelunasannya, bukan
	// transaksi baru, supaya penjualan/pembelian yang sama tidak terbukukan dua kali.
	dilunasi := 0
	for _, r := range lunasi {
		m := finance.CariMirip(s.txs, r.Tanggal, r.Nominal, r.Arah, nil)
		if !slices.ContainsFunc(m, func(x finance.Mirip) bool { return x.ID == r.Lunasi && x.BelumLunas }) {
			errs = append(errs, finance.ImportError{Baris: r.Baris, Pesan: "transaksi #" + strconv.Itoa(r.Lunasi) + " tidak cocok lagi untuk dilunasi (nominal/arah beda atau sudah lunas); baris ini dilewati"})
			continue
		}
		if _, _, err := s.lunasiTxLocked(r.Lunasi, r.Tanggal); err != nil {
			errs = append(errs, finance.ImportError{Baris: r.Baris, Pesan: err.Error()})
			continue
		}
		dilunasi++
	}
	for i := range txs {
		s.pakaiRekeningBawaanLocked(&txs[i], "")
		txs[i].ID = s.nextTxID
		s.nextTxID++
		s.txs = append(s.txs, txs[i])
	}
	s.beriNomorJurnalLocked()
	for i := range txs { // contoh di respons memuat nomor jurnalnya
		txs[i].NoJurnal = s.txs[len(s.txs)-len(txs)+i].NoJurnal
	}
	s.mu.Unlock()
	writeJSON(w, mutasiCommitResp{Disimpan: len(txs), Dilunasi: dilunasi, Errors: errs, Contoh: txs[:min(5, len(txs))]})
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
