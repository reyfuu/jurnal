package main

import (
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strconv"
	"time"

	"pajak/internal/finance"
	"pajak/internal/xlsx"
)

// CRUD Piutang & Utang dan Stok & Gudang. Keduanya sebelumnya hanya data contoh di dashboard;
// sekarang jadi data usaha yang bisa diimpor dari Excel lalu disunting per baris.

func (s *tenant) handleListTagihan(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	list := slices.Clone(s.tagihan)
	s.mu.RUnlock()
	writeJSON(w, finance.SummarizeTagihan(list, time.Now()))
}

func decodeTagihan(w http.ResponseWriter, r *http.Request) (finance.Tagihan, bool) {
	var t finance.Tagihan
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "data tagihan tidak valid", http.StatusBadRequest)
		return t, false
	}
	if err := finance.ValidateTagihan(&t); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return t, false
	}
	return t, true
}

// tagihanIndex harus dipanggil dengan s.mu terkunci.
func (s *tenant) tagihanIndex(r *http.Request) int {
	id, _ := strconv.Atoi(r.PathValue("id"))
	return slices.IndexFunc(s.tagihan, func(x finance.Tagihan) bool { return x.ID == id })
}

func (s *tenant) handleUpdateTagihan(w http.ResponseWriter, r *http.Request) {
	t, ok := decodeTagihan(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.tagihanIndex(r)
	if i < 0 {
		http.Error(w, "tagihan tidak ditemukan", http.StatusNotFound)
		return
	}
	t.ID = s.tagihan[i].ID
	s.tagihan[i] = t
	writeJSON(w, t)
}

func (s *tenant) handleDeleteTagihan(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.tagihanIndex(r)
	if i < 0 {
		http.Error(w, "tagihan tidak ditemukan", http.StatusNotFound)
		return
	}
	s.tagihan = slices.Delete(s.tagihan, i, i+1)
	w.WriteHeader(http.StatusNoContent)
}

// --- Stok ---

// handleListStok: posisi sekarang, atau ?pada=YYYY-MM-DD untuk jumlah & nilai per tanggal itu
// dari kartu stok (checklist No 22).
func (s *tenant) handleListStok(w http.ResponseWriter, r *http.Request) {
	k, list := s.kartu()
	if pada := r.URL.Query().Get("pada"); pada != "" {
		if _, err := time.Parse(time.DateOnly, pada); err != nil {
			http.Error(w, "pada harus YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		if k.Mulai == "" || pada < k.Mulai {
			http.Error(w, "stok per tanggal hanya bisa sejak kartu stok dimulai", http.StatusBadRequest)
			return
		}
		k = k.Pada(pada)
	}
	writeJSON(w, finance.RingkasStok(list, k))
}

func decodeStok(w http.ResponseWriter, r *http.Request) (finance.StokItem, bool) {
	var it finance.StokItem
	if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
		http.Error(w, "data stok tidak valid", http.StatusBadRequest)
		return it, false
	}
	if err := finance.ValidateStok(&it); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return it, false
	}
	return it, true
}

func (s *tenant) handleCreateStok(w http.ResponseWriter, r *http.Request) {
	it, ok := decodeStok(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	if err := finance.CekUnik(s.stok, finance.InfoStok, 0, it.SKU, it.Nama, "barang"); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if s.nextStokID == 0 {
		s.nextStokID = 1
	}
	it.ID = s.nextStokID
	s.nextStokID++
	s.stok = append(s.stok, it)
	s.mu.Unlock()
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, it)
}

func (s *tenant) stokIndex(r *http.Request) int {
	id, _ := strconv.Atoi(r.PathValue("id"))
	return slices.IndexFunc(s.stok, func(x finance.StokItem) bool { return x.ID == id })
}

func (s *tenant) handleUpdateStok(w http.ResponseWriter, r *http.Request) {
	it, ok := decodeStok(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.stokIndex(r)
	if i < 0 {
		http.Error(w, "barang tidak ditemukan", http.StatusNotFound)
		return
	}
	if err := finance.CekUnik(s.stok, finance.InfoStok, s.stok[i].ID, it.SKU, it.Nama, "barang"); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	it.ID = s.stok[i].ID
	s.stok[i] = it
	writeJSON(w, it)
}

func (s *tenant) handleDeleteStok(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.stokIndex(r)
	if i < 0 {
		http.Error(w, "barang tidak ditemukan", http.StatusNotFound)
		return
	}
	if s.stokDipakai(s.stok[i].ID) {
		http.Error(w, "barang ini sudah punya riwayat di kartu stok (transaksi, opname, atau produksi); tidak bisa dihapus", http.StatusConflict)
		return
	}
	s.stok = slices.Delete(s.stok, i, i+1)
	w.WriteHeader(http.StatusNoContent)
}

// --- Impor Excel per menu ---

// uploadedSheets membaca satu berkas .xlsx dari field "file". Batas 5 MB sama dengan onboarding.
func uploadedSheets(w http.ResponseWriter, r *http.Request) ([]finance.ImportSheet, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	f, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "unggah berkas .xlsx (maks 5 MB) di field file", http.StatusBadRequest)
		return nil, false
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "gagal membaca berkas", http.StatusBadRequest)
		return nil, false
	}
	raw, err := xlsx.ReadAll(data)
	if err != nil {
		http.Error(w, err.Error()+"; simpan ulang sebagai .xlsx atau pakai template", http.StatusBadRequest)
		return nil, false
	}
	sheets := make([]finance.ImportSheet, 0, len(raw))
	for _, sh := range raw {
		sheets = append(sheets, finance.ImportSheet{Nama: sh.Name, Rows: sh.Rows})
	}
	return sheets, true
}

func writeTemplate(w http.ResponseWriter, nama string, tpl []finance.ImportSheet) {
	sheets := make([]xlsx.Sheet, 0, len(tpl))
	for _, sh := range tpl {
		sheets = append(sheets, xlsx.Sheet{Name: sh.Nama, Rows: sh.Rows})
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="`+nama+`"`)
	_ = xlsx.Write(w, sheets)
}

func (s *Server) handleStokTemplate(w http.ResponseWriter, r *http.Request) {
	writeTemplate(w, "template-stok-gudang.xlsx", finance.StokTemplate())
}

// menuImportResp: pratinjau impor per menu. Tanpa ?commit=1 tidak ada yang disimpan.
type menuImportResp struct {
	Sheet    string               `json:"sheet"`
	Valid    int                  `json:"valid"`
	Errors   []finance.SheetError `json:"errors"`
	Imported bool                 `json:"imported"`
	Diganti  int                  `json:"diganti"` // baris lama yang ditimpa saat mode=ganti
	Stok     []finance.StokItem   `json:"stok,omitempty"`
	Invoice  []finance.Invoice    `json:"invoice,omitempty"`
}

func (s *tenant) handleImportStok(w http.ResponseWriter, r *http.Request) {
	sheets, ok := uploadedSheets(w, r)
	if !ok {
		return
	}
	list, errs, nama := finance.ParseStokFile(sheets)
	// Tambah (bukan ganti): barang yang SKU / namanya sudah ada di Stok & Gudang dilewati.
	if r.URL.Query().Get("mode") != "ganti" {
		s.mu.RLock()
		ada := slices.Clone(s.stok)
		s.mu.RUnlock()
		list = slices.DeleteFunc(list, func(it finance.StokItem) bool {
			if err := finance.CekUnik(ada, finance.InfoStok, 0, it.SKU, it.Nama, "barang"); err != nil {
				errs = append(errs, finance.SheetError{Sheet: nama, Pesan: err.Error() + " (dilewati)"})
				return true
			}
			return false
		})
	}
	resp := menuImportResp{Sheet: nama, Valid: len(list), Errors: errs, Stok: list}
	if r.URL.Query().Get("commit") != "1" {
		writeJSON(w, resp)
		return
	}
	s.mu.Lock()
	if r.URL.Query().Get("mode") == "ganti" {
		resp.Diganti = len(s.stok)
		s.stok = nil
	}
	if s.nextStokID == 0 {
		s.nextStokID = 1
	}
	for _, it := range list {
		it.ID = s.nextStokID
		s.nextStokID++
		s.stok = append(s.stok, it)
	}
	s.mu.Unlock()
	resp.Imported = true
	writeJSON(w, resp)
}
