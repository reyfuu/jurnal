package main

import (
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strconv"

	"pajak/internal/finance"
	"pajak/internal/xlsx"
)

// Impor data awal onboarding: satu workbook .xlsx berisi sheet Saldo Awal, Aset, Karyawan,
// dan Transaksi. Tanpa ?commit=1 hanya pratinjau — tidak ada yang disimpan.

func (s *tenant) handleOnboardingTemplate(w http.ResponseWriter, r *http.Request) {
	sheets := make([]xlsx.Sheet, 0, 5)
	for _, sh := range finance.OnboardingTemplate() {
		sheets = append(sheets, xlsx.Sheet{Name: sh.Nama, Rows: sh.Rows})
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="template-data-awal.xlsx"`)
	_ = xlsx.Write(w, sheets)
}

// onboardingResp: ringkasan per sheet + contoh baris, supaya user bisa memeriksa sebelum commit.
type onboardingResp struct {
	Opening        *finance.OpeningBalance `json:"opening"`
	ModalAwal      int64                   `json:"modal_awal"`
	SelisihModal   int64                   `json:"selisih_modal"`
	Assets         []finance.Asset         `json:"assets"`
	Karyawan       []finance.Karyawan      `json:"karyawan"`
	TransaksiCount int                     `json:"transaksi_count"`
	TransaksiCeki  []finance.Transaction   `json:"transaksi_sample"`
	Errors         []finance.SheetError    `json:"errors"`
	Terbaca        []string                `json:"terbaca"`
	Imported       bool                    `json:"imported"`
	Tagihan        []finance.Tagihan       `json:"tagihan"`
	Stok           []finance.StokItem      `json:"stok"`
	// Replaced menandai data lama yang ditimpa saat commit.
	ReplacedAssets   int `json:"replaced_assets"`
	ReplacedKaryawan int `json:"replaced_karyawan"`
	ReplacedTagihan  int `json:"replaced_tagihan"`
	ReplacedStok     int `json:"replaced_stok"`
}

func (s *tenant) handleOnboardingImport(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	f, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "unggah berkas .xlsx (maks 5 MB) di field file", http.StatusBadRequest)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "gagal membaca berkas", http.StatusBadRequest)
		return
	}
	raw, err := xlsx.ReadAll(data)
	if err != nil {
		http.Error(w, err.Error()+"; simpan ulang sebagai .xlsx atau pakai template", http.StatusBadRequest)
		return
	}
	sheets := make([]finance.ImportSheet, 0, len(raw))
	for _, sh := range raw {
		sheets = append(sheets, finance.ImportSheet{Nama: sh.Name, Rows: sh.Rows})
	}
	d := finance.ParseOnboarding(sheets)
	s.mu.RLock()
	finance.TerapkanMetode(d.Assets, s.metodeUsaha()) // kolom Metode di Excel kalah oleh metode usaha
	s.mu.RUnlock()

	resp := onboardingResp{
		Opening: d.Opening, Assets: d.Assets, Karyawan: d.Karyawan,
		Tagihan: d.Tagihan, Stok: d.Stok,
		TransaksiCount: len(d.Transaksi), TransaksiCeki: d.Transaksi[:min(5, len(d.Transaksi))],
		Errors: d.Errors, Terbaca: d.Terbaca,
	}
	// Nilai buku aset awal dipakai untuk mencocokkan modal disetor yang diisi user.
	if d.Opening != nil {
		nbv := finance.AssetsForPeriod(d.Assets, d.Transaksi, d.Opening.Period()).NilaiBukuAwal
		resp.ModalAwal = d.Opening.ModalAwal(nbv)
		resp.SelisihModal = d.Opening.SelisihModal(nbv)
	}
	if r.URL.Query().Get("commit") != "1" {
		writeJSON(w, resp)
		return
	}

	s.mu.Lock()
	if d.Opening != nil {
		o := *d.Opening
		s.opening = &o
	}
	// Aset & karyawan hasil impor MENGGANTI register lama (onboarding, bukan tambah data);
	// transaksi ditambahkan karena bisa diimpor bertahap per periode.
	if len(d.Assets) > 0 {
		resp.ReplacedAssets = len(s.assets)
		s.assets = nil
		s.nextAssetID = 1
		for _, a := range d.Assets {
			a.ID = s.nextAssetID
			s.nextAssetID++
			s.assets = append(s.assets, a)
		}
	}
	if len(d.Karyawan) > 0 {
		resp.ReplacedKaryawan = len(s.karyawan)
		s.karyawan = nil
		s.nextKaryawanID = 1
		for _, k := range d.Karyawan {
			k.ID = s.nextKaryawanID
			s.nextKaryawanID++
			s.karyawan = append(s.karyawan, k)
		}
	}
	if len(d.Tagihan) > 0 {
		resp.ReplacedTagihan = len(s.tagihan)
		s.tagihan = nil
		s.nextTagihanID = 1
		for _, t := range d.Tagihan {
			t.ID = s.nextTagihanID
			s.nextTagihanID++
			s.tagihan = append(s.tagihan, t)
		}
	}
	if len(d.Stok) > 0 {
		resp.ReplacedStok = len(s.stok)
		s.stok = nil
		s.nextStokID = 1
		for _, it := range d.Stok {
			it.ID = s.nextStokID
			s.nextStokID++
			s.stok = append(s.stok, it)
		}
	}
	for _, t := range d.Transaksi {
		t.ID = s.nextTxID
		s.nextTxID++
		s.txs = append(s.txs, t)
	}
	s.mu.Unlock()
	resp.Imported = true
	writeJSON(w, resp)
}

// --- Neraca awal ---

type openingResp struct {
	Opening      *finance.OpeningBalance `json:"opening"`
	ModalAwal    int64                   `json:"modal_awal"`
	SelisihModal int64                   `json:"selisih_modal"`
	// Aktif false = laporan masih memakai saldo awal data demo.
	Aktif bool `json:"aktif"`
}

func (s *tenant) openingSummary() openingResp {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.opening == nil {
		return openingResp{}
	}
	o := *s.opening
	nbv := finance.AssetsForPeriod(s.assets, s.txs, o.Period()).NilaiBukuAwal
	return openingResp{Opening: &o, ModalAwal: o.ModalAwal(nbv), SelisihModal: o.SelisihModal(nbv), Aktif: true}
}

func (s *tenant) handleGetOpening(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.openingSummary())
}

func (s *tenant) handlePutOpening(w http.ResponseWriter, r *http.Request) {
	var o finance.OpeningBalance
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		http.Error(w, "data neraca awal tidak valid", http.StatusBadRequest)
		return
	}
	if err := finance.ValidateOpeningBalance(&o); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	s.opening = &o
	s.mu.Unlock()
	writeJSON(w, s.openingSummary())
}

// --- Karyawan ---

func (s *tenant) handleListKaryawan(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	list := slices.Clone(s.karyawan)
	s.mu.RUnlock()
	writeJSON(w, finance.SummarizeKaryawan(list))
}

func decodeKaryawan(w http.ResponseWriter, r *http.Request) (finance.Karyawan, bool) {
	var k finance.Karyawan
	if err := json.NewDecoder(r.Body).Decode(&k); err != nil {
		http.Error(w, "data karyawan tidak valid", http.StatusBadRequest)
		return k, false
	}
	if err := finance.ValidateKaryawan(&k); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return k, false
	}
	return k, true
}

func (s *tenant) handleCreateKaryawan(w http.ResponseWriter, r *http.Request) {
	k, ok := decodeKaryawan(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	k.ID = s.nextKaryawanID
	s.nextKaryawanID++
	s.karyawan = append(s.karyawan, k)
	s.mu.Unlock()
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, k)
}

// karyawanIndex harus dipanggil dengan s.mu terkunci.
func (s *tenant) karyawanIndex(r *http.Request) int {
	id, _ := strconv.Atoi(r.PathValue("id"))
	return slices.IndexFunc(s.karyawan, func(k finance.Karyawan) bool { return k.ID == id })
}

func (s *tenant) handleUpdateKaryawan(w http.ResponseWriter, r *http.Request) {
	k, ok := decodeKaryawan(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.karyawanIndex(r)
	if i < 0 {
		http.Error(w, "karyawan tidak ditemukan", http.StatusNotFound)
		return
	}
	k.ID = s.karyawan[i].ID
	s.karyawan[i] = k
	writeJSON(w, k)
}

func (s *tenant) handleDeleteKaryawan(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.karyawanIndex(r)
	if i < 0 {
		http.Error(w, "karyawan tidak ditemukan", http.StatusNotFound)
		return
	}
	s.karyawan = slices.Delete(s.karyawan, i, i+1)
	w.WriteHeader(http.StatusNoContent)
}
