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
	"time"

	"pajak/internal/ai"
	"pajak/internal/coa"
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

var errBiayaProduksi = errors.New("catatan ini biaya lain produksi; ubah lewat Produksi & Supplier → riwayat produksi")

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
	peta := coa.PetaDari(s.akunLocked())
	produksi := map[int]bool{}
	for _, g := range s.gerak {
		if g.Grup != 0 {
			produksi[g.Grup] = true
		}
	}
	s.mu.RUnlock()
	h := finance.ParseMutasi(rows, tahun, existing)
	// Akun bayar efektif transaksi yang cocok (Lawan kosong = aturan channel / Kas), supaya klien
	// bisa menawarkan "pindah akun bayar" hanya bila berbeda dari rekening mutasi ini.
	for i := range h.Rows {
		if m := h.Rows[i].Mirip; m != nil && !m.BelumLunas {
			if j := slices.IndexFunc(existing, func(t finance.Transaction) bool { return t.ID == m.ID }); j >= 0 {
				m.Lawan = coa.AkunBayar(existing[j], peta)
				m.Produksi = existing[j].Grup != 0 && produksi[existing[j].Grup]
				m.Khusus = coa.ChannelKhusus(existing[j].Channel)
			}
		}
	}
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
	// Rekening: kode akun kas/bank mutasi ini (kosong = rekening bawaan). Transaksi baru dicatat
	// di rekening ini, dan baris "sama — pindah" memindah akun bayar transaksinya ke sini.
	Rekening string `json:"rekening,omitempty"`
}

type mutasiCommitResp struct {
	Disimpan int `json:"disimpan"`
	// Dilunasi: baris yang dicatat sebagai pelunasan transaksi belum lunas yang sudah ada.
	Dilunasi int `json:"dilunasi"`
	// Dipindah: transaksi lama yang akun bayarnya dipindah ke rekening mutasi ini (MutasiRow.Pindah).
	Dipindah int                   `json:"dipindah"`
	Errors   []finance.ImportError `json:"errors"`
	Contoh   []finance.Transaction `json:"contoh"`
}

func (s *tenant) handleMutasiCommit(w http.ResponseWriter, r *http.Request) {
	var req mutasiCommitReq
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 5<<20)).Decode(&req); err != nil {
		http.Error(w, "data baris mutasi tidak valid", http.StatusBadRequest)
		return
	}
	txs, baris, errs := finance.MutasiKeTransaksiBaris(req.Rows)
	var lunasi, pindah []finance.MutasiRow
	for _, r := range req.Rows {
		switch {
		case r.Pilih && r.Lunasi != 0:
			lunasi = append(lunasi, r)
		case r.Pilih && r.Pindah != 0:
			pindah = append(pindah, r)
		}
	}
	s.mu.Lock()
	rekening, err := s.rekeningMutasiLocked(req.Rekening)
	if err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, t := range txs {
		if err := s.cekTxLocked(t); err != nil {
			s.mu.Unlock()
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
	}
	// Barang harus ada & wajib barang: galat per baris, baris lain tetap dibukukan.
	txs, baris, errs = s.saringBarangMutasi(txs, baris, errs)
	// Pindah akun bayar diperiksa dulu (tanpa mengubah apa pun), diterapkan sesudah cek stok.
	var dipindah []int
	for _, r := range pindah {
		if slices.Contains(dipindah, r.Pindah) {
			errs = append(errs, finance.ImportError{Baris: r.Baris, Pesan: "transaksi #" + strconv.Itoa(r.Pindah) + " sudah dipindah oleh baris lain; baris ini dilewati"})
			continue
		}
		if err := s.cekPindahMutasiLocked(r, rekening); err != nil {
			errs = append(errs, finance.ImportError{Baris: r.Baris, Pesan: err.Error()})
			continue
		}
		dipindah = append(dipindah, r.Pindah)
	}
	urutGalat(errs)
	if len(txs) == 0 && len(lunasi) == 0 && len(dipindah) == 0 {
		s.mu.Unlock()
		// Tidak ada yang bisa dibukukan: laporkan kenapa, jangan balas "berhasil 0".
		http.Error(w, pesanGagalMutasi(errs), http.StatusBadRequest)
		return
	}
	// Stok tidak minus dicek atas seluruh transaksi baru sekaligus (kartu yang belum dimulai
	// dimulai di tanggal baris berbarang paling awal). Gagal = 409 dan tidak ada yang disimpan,
	// termasuk baris tanpa barang dan pelunasan: menyimpan sebagian akan bergantung urutan baris.
	// Pindah akun bayar tidak mengubah kartu stok, jadi tidak ikut cek ini.
	awal, err := s.cekStokImporLocked(txs)
	if err != nil {
		pesan := err.Error()
		// cekStokImporLocked memberi txs ID sementara nextTxID+i: tunjuk baris berkasnya.
		var gm *galatMinus
		if errors.As(err, &gm) && gm.txID >= s.nextTxID && gm.txID < s.nextTxID+len(txs) {
			pesan += " (baris " + strconv.Itoa(baris[gm.txID-s.nextTxID]) + ")"
		}
		s.mu.Unlock()
		http.Error(w, pesan, http.StatusConflict)
		return
	}
	if awal != nil { // hanya bila ada baris berbarang yang akan tersimpan
		if err := s.simpanStokAwalLocked(awal, currentUser(r).Name, time.Now().UTC()); err != nil {
			s.mu.Unlock()
			http.Error(w, "gagal menyimpan metode nilai stok", http.StatusInternalServerError)
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
		if _, _, err := s.lunasiTxLocked(r.Lunasi, r.Tanggal, rekening); err != nil {
			errs = append(errs, finance.ImportError{Baris: r.Baris, Pesan: err.Error()})
			continue
		}
		dilunasi++
	}
	for _, id := range dipindah { // dicari ulang: pelunasan di atas bisa menggeser s.txs
		if i := slices.IndexFunc(s.txs, func(t finance.Transaction) bool { return t.ID == id }); i >= 0 {
			s.txs[i].Lawan = rekening
		}
	}
	for i := range txs {
		txs[i].Lawan = rekening // kosong (CoA tanpa bank) → rekening bawaan / aturan channel
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
	urutGalat(errs) // galat pelunasan menyusul sesudah galat barang
	writeJSON(w, mutasiCommitResp{Disimpan: len(txs), Dilunasi: dilunasi, Dipindah: len(dipindah), Errors: errs, Contoh: txs[:min(5, len(txs))]})
}

// rekeningMutasiLocked: kode rekening mutasi yang dipilih, divalidasi seperti BayarDari produksi
// (harus akun kas/bank/e-wallet); kosong = rekening bawaan. s.mu dipegang.
func (s *tenant) rekeningMutasiLocked(kode string) (string, error) {
	if kode == "" {
		return s.rekeningBawaanLocked(), nil
	}
	all := s.akunLocked()
	if a, ok := coa.Find(all, kode); !ok || coa.JenisLawan(a, coa.PetaDari(all)) != coa.LawanRekening {
		return "", errors.New("rekening mutasi harus akun Kas / Bank")
	}
	return kode, nil
}

// cekPindahMutasiLocked: nil bila akun bayar transaksi r.Pindah boleh dipindah ke rekening:
// masih cocok (CariMirip, sudah lunas), bukan transaksi invoice / biaya lain produksi / Pindah
// Dana / ber-channel khusus, dan bulannya belum ditutup. Catatan Catat cepat (Grup) boleh: baris
// rekeningnya ikut berganti saat catatan dibuka ulang. Tidak mengubah apa pun. s.mu dipegang.
func (s *tenant) cekPindahMutasiLocked(r finance.MutasiRow, rekening string) error {
	id := strconv.Itoa(r.Pindah)
	if rekening == "" {
		return errors.New("usaha belum punya akun Bank; pilih rekening mutasi dulu")
	}
	m := finance.CariMirip(s.txs, r.Tanggal, r.Nominal, r.Arah, nil)
	if !slices.ContainsFunc(m, func(x finance.Mirip) bool { return x.ID == r.Pindah && !x.BelumLunas }) {
		return errors.New("transaksi #" + id + " tidak cocok lagi dengan baris ini (nominal/arah/tanggal beda atau belum lunas); baris ini dilewati")
	}
	i := slices.IndexFunc(s.txs, func(t finance.Transaction) bool { return t.ID == r.Pindah })
	if i < 0 {
		return errors.New("transaksi #" + id + " tidak ditemukan; baris ini dilewati")
	}
	t := s.txs[i]
	switch {
	case t.InvoiceID != 0:
		return errTxInvoice
	case t.Grup != 0 && slices.ContainsFunc(s.gerak, func(g finance.GerakStok) bool { return g.Grup == t.Grup }):
		return errBiayaProduksi
	case t.Jenis == finance.JenisPindahDana:
		return errors.New("transaksi #" + id + " adalah Pindah Dana; ubah rekeningnya lewat menu Transaksi")
	case coa.ChannelKhusus(t.Channel):
		// pakaiRekeningBawaanLocked mengosongkan Lawan channel ini saat diubah: pindah akan hilang.
		return errors.New("transaksi #" + id + " lewat channel " + t.Channel + "; uangnya tercatat di akun channel itu, bukan di rekening mutasi")
	}
	if err := s.cekTerbukaLocked(tanggalTx(t)...); err != nil {
		return err
	}
	return nil
}

func urutGalat(errs []finance.ImportError) {
	slices.SortStableFunc(errs, func(a, b finance.ImportError) int { return a.Baris - b.Baris })
}

// saringBarangMutasi: rapikan barang tiap transaksi mutasi, lalu buang yang barangnya tidak ada
// di daftar atau melanggar wajib barang (galatnya masuk errs dengan nomor baris berkasnya).
// baris sejajar txs. s.mu dipegang.
func (s *tenant) saringBarangMutasi(txs []finance.Transaction, baris []int, errs []finance.ImportError) ([]finance.Transaction, []int, []finance.ImportError) {
	var okTx []finance.Transaction
	var okBaris []int
	cek := func(t finance.Transaction) error {
		for _, b := range t.BarisBarang() {
			if _, ok := s.stokByID(b.StokID); !ok {
				return errors.New("barang yang dipilih tidak ada di Stok & Gudang")
			}
		}
		return s.cekBarangLocked(t)
	}
	for i, t := range txs {
		finance.NormalkanBarang(&t)
		if err := cek(t); err != nil {
			errs = append(errs, finance.ImportError{Baris: baris[i], Pesan: err.Error()})
			continue
		}
		okTx, okBaris = append(okTx, t), append(okBaris, baris[i])
	}
	return okTx, okBaris, errs
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
