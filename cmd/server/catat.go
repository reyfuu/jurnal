package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"pajak/internal/coa"
	"pajak/internal/finance"
)

// Catat cepat multi-akun (halaman Catat transaksi): baris jurnal debit/kredit disimpan sebagai
// satu grup transaksi (lihat coa.PecahCatat). Transaksi grup hanya diubah lewat halaman ini.

var errTxGrup = errors.New("transaksi ini bagian dari catatan jurnal multi-akun; ubah atau hapus lewat halaman Catat transaksi supaya semua barisnya tetap seimbang")

type catatReq struct {
	Tanggal    string           `json:"tanggal"`
	Pihak      string           `json:"pihak,omitempty"`
	NoBukti    string           `json:"no_bukti,omitempty"`
	Keterangan string           `json:"keterangan,omitempty"`
	Baris      []coa.BarisCatat `json:"baris"`
	// Sumber: "foto" bila baris diisi AI dari foto nota (dipakai pengecekan transaksi dobel).
	Sumber string `json:"sumber,omitempty"`
	// Bukti: token dari baca foto nota; fotonya disimpan sebagai bukti catatan baru (bukti_nota.go).
	Bukti string `json:"bukti,omitempty"`
}

type catatResp struct {
	catatReq
	Grup      int           `json:"grup"`
	Transaksi []int         `json:"transaksi"`
	FotoNota  *fotoNotaInfo `json:"foto_nota,omitempty"`
}

type akunCatat struct {
	Kode     string `json:"kode"`
	Nama     string `json:"nama"`
	Kelompok string `json:"kelompok"`
	Lawan    string `json:"lawan,omitempty"` // rekening | piutang | utang
}

// handleOpsiCatat: daftar akun untuk baris jurnal dan akun bawaan tiap jenis Catat cepat.
func (s *tenant) handleOpsiCatat(w http.ResponseWriter, _ *http.Request) {
	all, active, p := s.akunSaldoAwal()
	peta := coa.PetaDari(all)
	out := struct {
		Akun   []akunCatat       `json:"akun"`
		Bawaan map[string]string `json:"bawaan"`
	}{Akun: []akunCatat{}, Bawaan: map[string]string{
		"kas": peta.Kode(coa.PeranKas), "piutang": peta.Kode(coa.PeranPiutang), "utang": peta.Kode(coa.PeranUtangUsaha)}}
	for _, a := range active {
		out.Akun = append(out.Akun, akunCatat{a.Kode, a.Nama, string(a.Kelompok), coa.JenisLawan(a, peta)})
	}
	for _, j := range finance.JenisList {
		out.Bawaan[j.Nama] = coa.AutoAccount(j.Nama, p, peta)
	}
	writeJSON(w, out)
}

func bacaCatat(w http.ResponseWriter, r *http.Request) (catatReq, bool) {
	var c catatReq
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "data catatan tidak valid", http.StatusBadRequest)
		return c, false
	}
	c.Pihak, c.NoBukti, c.Keterangan = strings.TrimSpace(c.Pihak), strings.TrimSpace(c.NoBukti), strings.TrimSpace(c.Keterangan)
	if len(c.Baris) > 100 {
		http.Error(w, "paling banyak 100 baris per catatan", http.StatusBadRequest)
		return c, false
	}
	return c, true
}

// susunTxCatat: transaksi siap simpan (tanpa ID) dari catatan; err = pesan untuk pengguna.
func (s *tenant) susunTxCatat(c catatReq, grup int) ([]finance.Transaction, error) {
	all, _, p := s.akunSaldoAwal()
	txs, err := coa.PecahCatat(c.Baris, all, p, coa.PetaDari(all))
	if err != nil {
		return nil, err
	}
	for i := range txs {
		t := &txs[i]
		t.Tanggal, t.Pihak, t.NoBukti, t.Grup = c.Tanggal, c.Pihak, c.NoBukti, grup
		if c.Sumber == "foto" {
			t.Sumber = "foto"
		}
		if t.Keterangan == "" {
			t.Keterangan = c.Keterangan
		}
		if err := finance.Validate(t); err != nil {
			return nil, err
		}
	}
	return txs, nil
}

// simpanCatatLocked mengganti transaksi grup (lama boleh kosong) dengan yang baru. s.mu dipegang.
func (s *tenant) simpanCatatLocked(grup int, baru []finance.Transaction) ([]int, error) {
	if slices.ContainsFunc(s.gerak, func(g finance.GerakStok) bool { return g.Grup == grup }) {
		return nil, errors.New("catatan ini biaya lain produksi; ubah atau hapus lewat Produksi & Supplier → riwayat produksi")
	}
	for _, t := range s.txs {
		if t.Grup == grup {
			if err := s.cekTerbukaLocked(tanggalTx(t)...); err != nil {
				return nil, err
			}
		}
	}
	for _, t := range baru {
		if err := s.cekTxLocked(t); err != nil {
			return nil, err
		}
	}
	s.txs = slices.DeleteFunc(s.txs, func(t finance.Transaction) bool { return t.Grup == grup })
	var ids []int
	for _, t := range baru {
		t.ID = s.nextTxID
		s.nextTxID++
		s.txs = append(s.txs, t)
		ids = append(ids, t.ID)
	}
	return ids, nil
}

func (s *tenant) grupBerikutLocked() int {
	g := 0
	for _, t := range s.txs {
		g = max(g, t.Grup)
	}
	return g + 1
}

func (s *tenant) handleCreateCatat(w http.ResponseWriter, r *http.Request) {
	c, ok := bacaCatat(w, r)
	if !ok {
		return
	}
	txs, err := s.susunTxCatat(c, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	grup := s.grupBerikutLocked()
	for i := range txs {
		txs[i].Grup = grup
	}
	ids, err := s.simpanCatatLocked(grup, txs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	foto := s.lampirkanBuktiLocked(r, grup, c.Bukti)
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, catatResp{c, grup, ids, foto})
}

func (s *tenant) handleGetCatat(w http.ResponseWriter, r *http.Request) {
	grup, _ := strconv.Atoi(r.PathValue("grup"))
	all, _, p := s.akunSaldoAwal()
	s.mu.RLock()
	var txs []finance.Transaction
	for _, t := range s.txs {
		if grup > 0 && t.Grup == grup {
			txs = append(txs, t)
		}
	}
	s.mu.RUnlock()
	if len(txs) == 0 {
		http.Error(w, "catatan tidak ditemukan", http.StatusNotFound)
		return
	}
	out := catatResp{Grup: grup, catatReq: catatReq{Tanggal: txs[0].Tanggal, Pihak: txs[0].Pihak, NoBukti: txs[0].NoBukti,
		Baris: coa.SusunCatat(txs, p, coa.PetaDari(all))}}
	for _, t := range txs {
		out.Transaksi = append(out.Transaksi, t.ID)
	}
	s.mu.RLock()
	if m, ok := s.buktiLocked(grup); ok {
		out.FotoNota = m.info()
	}
	s.mu.RUnlock()
	writeJSON(w, out)
}

func (s *tenant) handleUpdateCatat(w http.ResponseWriter, r *http.Request) {
	grup, _ := strconv.Atoi(r.PathValue("grup"))
	c, ok := bacaCatat(w, r)
	if !ok {
		return
	}
	txs, err := s.susunTxCatat(c, grup)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if grup <= 0 || !slices.ContainsFunc(s.txs, func(t finance.Transaction) bool { return t.Grup == grup }) {
		http.Error(w, "catatan tidak ditemukan", http.StatusNotFound)
		return
	}
	ids, err := s.simpanCatatLocked(grup, txs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	out := catatResp{catatReq: c, Grup: grup, Transaksi: ids}
	if m, ok := s.buktiLocked(grup); ok {
		out.FotoNota = m.info()
	}
	writeJSON(w, out)
}

func (s *tenant) handleDeleteCatat(w http.ResponseWriter, r *http.Request) {
	grup, _ := strconv.Atoi(r.PathValue("grup"))
	s.mu.Lock()
	defer s.mu.Unlock()
	if grup <= 0 || !slices.ContainsFunc(s.txs, func(t finance.Transaction) bool { return t.Grup == grup }) {
		http.Error(w, "catatan tidak ditemukan", http.StatusNotFound)
		return
	}
	if _, err := s.simpanCatatLocked(grup, nil); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	s.hapusBuktiLocked(grup)
	w.WriteHeader(http.StatusNoContent)
}
