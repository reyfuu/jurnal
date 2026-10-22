package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"time"

	"pajak/internal/finance"
)

// Register aset: sumber tunggal penyusutan & amortisasi untuk Laba Rugi, Neraca, dan Arus Kas.
// Hak akses mengikuti "laporan-keuangan" karena tab Aset ada di dalam menu itu.

// asetListResp: ringkasan aset + jumlah aset yang sumber dananya belum dipilih.
type asetListResp struct {
	finance.AssetSummary
	// SumberBelumDipilih: aset tanpa transaksi, diperoleh sejak saldo awal, yang belum dipilih
	// sumber dananya. Sementara dicatat sebagai tambahan modal supaya neraca tetap seimbang.
	SumberBelumDipilih int    `json:"sumber_belum_dipilih"`
	TanggalSaldoAwal   string `json:"tanggal_saldo_awal"`
}

func (s *tenant) handleListAssets(w http.ResponseWriter, r *http.Request) {
	out := asetListResp{AssetSummary: s.report(periodOrDefault(r)).Assets}
	if pada := r.URL.Query().Get("pada"); pada != "" { // per tanggal bebas (checklist No 22)
		if _, err := time.Parse(time.DateOnly, pada); err != nil {
			http.Error(w, "pada harus YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		s.mu.RLock()
		out.AssetSummary = finance.AssetsPada(s.assets, s.txs, pada)
		s.mu.RUnlock()
	}
	s.mu.RLock()
	out.TanggalSaldoAwal = s.bukuAwalLocked().Tanggal
	for _, a := range s.assets {
		if a.TransaksiID == 0 && a.SumberDana == "" && a.TanggalPerolehan >= out.TanggalSaldoAwal {
			out.SumberBelumDipilih++
		}
	}
	s.mu.RUnlock()
	writeJSON(w, out)
}

// sinkronTxAset menjaga transaksi "Beli Alat / Aset" milik aset yang dibayar kas / utang.
// lama = keadaan sebelum diubah (nil saat membuat). Dipanggil dengan s.mu terkunci.
//
// Aset yang didaftarkan dari transaksi pengguna (draft "Lengkapi") tetap memakai transaksi itu.
// Aset yang diperoleh sebelum tanggal saldo awal sudah termasuk saldo awal, jadi tanpa sumber.
func (s *tenant) sinkronTxAset(a *finance.Asset, lama *finance.Asset) error {
	milikAset := lama != nil && lama.TransaksiID != 0 && finance.SumberBerTransaksi(lama.SumberDana)
	if lama != nil {
		a.TransaksiID = lama.TransaksiID // tautan transaksi hanya diatur server
	}
	if a.TransaksiID != 0 && !milikAset {
		a.SumberDana = "" // dari transaksi pengguna: kas/utangnya sudah tercatat di sana
		return nil
	}
	if a.TanggalPerolehan < s.bukuAwalLocked().Tanggal {
		a.SumberDana = ""
	}
	i := slices.IndexFunc(s.txs, func(t finance.Transaction) bool { return milikAset && t.ID == lama.TransaksiID })
	if !finance.SumberBerTransaksi(a.SumberDana) {
		if i >= 0 {
			s.txs = slices.Delete(s.txs, i, i+1)
		}
		a.TransaksiID = 0
		return nil
	}
	t := finance.Transaction{Tanggal: a.TanggalPerolehan, Jenis: finance.JenisBeliAset, Nominal: a.HargaPerolehan,
		Keterangan: "Aset: " + a.Nama, Lunas: a.SumberDana == finance.SumberKas}
	if i >= 0 && lama.SumberDana == a.SumberDana {
		// Status lunas tidak disentuh: utang aset bisa sudah dilunasi dari menu Piutang & Utang.
		t.ID, t.Lunas, t.TanggalLunas, t.Channel, t.Pihak = s.txs[i].ID, s.txs[i].Lunas, s.txs[i].TanggalLunas, s.txs[i].Channel, s.txs[i].Pihak
		if t.TanggalLunas != "" && t.TanggalLunas < t.Tanggal {
			t.TanggalLunas = t.Tanggal
		}
	}
	if err := finance.Validate(&t); err != nil {
		return err
	}
	if err := s.cekTerbukaLocked(t.Tanggal); err != nil {
		return err
	}
	lawanLama := ""
	if i >= 0 {
		lawanLama = s.txs[i].Lawan
	}
	s.pakaiRekeningBawaanLocked(&t, lawanLama)
	if i >= 0 {
		t.ID = s.txs[i].ID
		t.NoJurnal = s.txs[i].NoJurnal
		s.txs[i] = t
	} else {
		t.ID = s.nextTxID
		s.nextTxID++
		s.txs = append(s.txs, t)
	}
	a.TransaksiID = t.ID
	return nil
}

// metodeUsaha: metode penyusutan profil; "" = belum dipilih. Dipanggil dengan s.mu terkunci
// (baca atau tulis) — profile hanya diganti di bawah kunci tulis.
func (s *tenant) metodeUsaha() string {
	if s.profile == nil {
		return ""
	}
	return s.profile.MetodePenyusutan
}

// decodeAsset membaca dan memvalidasi body; menulis 400 dan mengembalikan false bila gagal.
// Bila usaha sudah memilih metode penyusutan, metode di body diabaikan.
func (s *tenant) decodeAsset(w http.ResponseWriter, r *http.Request) (finance.Asset, bool) {
	var a finance.Asset
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, "data aset tidak valid", http.StatusBadRequest)
		return a, false
	}
	s.mu.RLock()
	m := s.metodeUsaha()
	s.mu.RUnlock()
	if m != "" {
		a.Metode = ""
	}
	if err := finance.ValidateAsset(&a); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return a, false
	}
	a.Metode = finance.MetodeUntuk(m, a)
	return a, true
}

// handlePilihMetode: Owner memilih metode penyusutan usaha satu kali; semua aset dihitung ulang.
func (s *tenant) handlePilihMetode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Metode string `json:"metode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "data tidak valid", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case s.profile == nil:
		http.Error(w, "lengkapi profil usaha dulu", http.StatusConflict)
		return
	case s.profile.MetodePenyusutan != "":
		http.Error(w, errMetodeTerkunci, http.StatusConflict)
		return
	}
	prof := *s.profile
	prof.MetodePenyusutan = req.Metode
	if req.Metode == "" {
		prof.MetodePenyusutan = "-" // kosong bukan pilihan
	}
	if err := prof.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := saveProfile(s.file("business"), prof); err != nil {
		http.Error(w, "gagal menyimpan profil usaha", http.StatusInternalServerError)
		return
	}
	s.profile = &prof
	finance.TerapkanMetode(s.assets, prof.MetodePenyusutan)
	writeJSON(w, prof)
}

const errMetodeTerkunci = "Metode penyusutan sudah dipilih dan dikunci untuk semua aset. Perubahan metode memerlukan persetujuan DJP; hubungi tim MasterPajak AI."

func (s *tenant) handleCreateAsset(w http.ResponseWriter, r *http.Request) {
	a, ok := s.decodeAsset(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	if err := s.cekTerbukaLocked(a.TanggalPerolehan); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if a.TransaksiID != 0 && slices.ContainsFunc(s.assets, func(x finance.Asset) bool { return x.TransaksiID == a.TransaksiID }) {
		s.mu.Unlock()
		http.Error(w, "transaksi ini sudah terdaftar sebagai aset", http.StatusConflict)
		return
	}
	if err := s.sinkronTxAset(&a, nil); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	a.ID = s.nextAssetID
	s.nextAssetID++
	s.assets = append(s.assets, a)
	s.mu.Unlock()
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, a)
}

// assetIndex harus dipanggil dengan s.mu terkunci.
func (s *tenant) assetIndex(r *http.Request) int {
	id, _ := strconv.Atoi(r.PathValue("id"))
	return slices.IndexFunc(s.assets, func(a finance.Asset) bool { return a.ID == id })
}

func (s *tenant) handleUpdateAsset(w http.ResponseWriter, r *http.Request) {
	a, ok := s.decodeAsset(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.assetIndex(r)
	if i < 0 {
		http.Error(w, "aset tidak ditemukan", http.StatusNotFound)
		return
	}
	if err := s.cekTerbukaLocked(s.assets[i].TanggalPerolehan, a.TanggalPerolehan); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	lama := s.assets[i]
	if err := s.sinkronTxAset(&a, &lama); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	a.ID = lama.ID
	s.assets[i] = a
	writeJSON(w, a)
}

func (s *tenant) handleDeleteAsset(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.assetIndex(r)
	if i < 0 {
		http.Error(w, "aset tidak ditemukan", http.StatusNotFound)
		return
	}
	if err := s.cekTerbukaLocked(s.assets[i].TanggalPerolehan); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	// Transaksi yang dibuat otomatis untuk aset ini ikut dihapus; transaksi milik pengguna tetap.
	lama := s.assets[i]
	kosong := lama
	kosong.SumberDana = ""
	if err := s.sinkronTxAset(&kosong, &lama); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	s.assets = slices.Delete(s.assets, i, i+1)
	w.WriteHeader(http.StatusNoContent)
}
