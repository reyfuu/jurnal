package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

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
	// Rincian / BukanBarang: barang untuk satu-satunya transaksi Penjualan / Beli Bahan / Stok
	// yang dihasilkan catatan ini (persediaan per akun tahap 2; wajib di perpetual).
	Rincian     []finance.RincianBarang `json:"rincian,omitempty"`
	BukanBarang bool                    `json:"bukan_barang,omitempty"`
}

var errPecahCatat = errors.New("catatan ini berisi lebih dari satu penjualan/pembelian barang; pecah jadi beberapa catatan")

// txBerbarang: indeks transaksi yang boleh membawa barang (Penjualan, Beli Bahan / Stok).
func txBerbarang(txs []finance.Transaction) []int {
	var out []int
	for i, t := range txs {
		if t.Jenis == finance.JenisJual || t.Jenis == finance.JenisBeliStok {
			out = append(out, i)
		}
	}
	return out
}

type catatResp struct {
	catatReq
	Grup      int           `json:"grup"`
	Transaksi []int         `json:"transaksi"`
	NoJurnal  string        `json:"no_jurnal,omitempty"`
	FotoNota  *fotoNotaInfo `json:"foto_nota,omitempty"`
}

type akunCatat struct {
	Kode     string `json:"kode"`
	Nama     string `json:"nama"`
	Kelompok string `json:"kelompok"`
	Lawan    string `json:"lawan,omitempty"` // rekening | piutang | utang
	// BeliBarang / JualBarang: akun ini di debit menjadi Beli Bahan / Stok, di kredit menjadi
	// Penjualan; layar Catat cepat lalu menampilkan pemilih barang.
	BeliBarang bool `json:"beli_barang,omitempty"`
	JualBarang bool `json:"jual_barang,omitempty"`
}

// handleOpsiCatat: daftar akun untuk baris jurnal dan akun bawaan tiap jenis Catat cepat.
func (s *tenant) handleOpsiCatat(w http.ResponseWriter, _ *http.Request) {
	all, active, p := s.akunSaldoAwal()
	peta := coa.PetaDari(all)
	out := struct {
		Akun   []akunCatat       `json:"akun"`
		Bawaan map[string]string `json:"bawaan"`
	}{Akun: []akunCatat{}, Bawaan: map[string]string{
		"kas": peta.Kode(coa.PeranKas), "bank": peta.Kode(coa.PeranBank), "piutang": peta.Kode(coa.PeranPiutang), "utang": peta.Kode(coa.PeranUtangUsaha)}}
	for _, a := range active {
		x := akunCatat{Kode: a.Kode, Nama: a.Nama, Kelompok: string(a.Kelompok), Lawan: coa.JenisLawan(a, peta)}
		if x.Lawan == "" {
			jd, _ := coa.JenisUntukAkun(a, false, p, peta)
			jk, _ := coa.JenisUntukAkun(a, true, p, peta)
			x.BeliBarang, x.JualBarang = jd == finance.JenisBeliStok, jk == finance.JenisJual
		}
		out.Akun = append(out.Akun, x)
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
	barang := txBerbarang(txs)
	if len(c.Rincian) > 0 {
		switch len(barang) {
		case 0:
			return nil, errors.New("barang hanya untuk catatan penjualan atau pembelian bahan / stok")
		case 1:
			txs[barang[0]].Rincian = slices.Clone(c.Rincian)
		default:
			return nil, errPecahCatat
		}
	}
	for _, i := range barang {
		txs[i].BukanBarang = c.BukanBarang
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
		if !s.validStok(*t) {
			return nil, errors.New("barang yang dipilih tidak ada di Stok & Gudang")
		}
		finance.NormalkanBarang(t)
	}
	return txs, nil
}

// simpanCatatLocked mengganti transaksi grup (lama boleh kosong) dengan yang baru. Galat: kode
// HTTP + pesan (periode 409, wajib barang 400, stok minus 409). oleh: pencatat stok awal otomatis.
// s.mu dipegang.
func (s *tenant) simpanCatatLocked(grup int, baru []finance.Transaction, oleh string) ([]int, int, error) {
	if slices.ContainsFunc(s.gerak, func(g finance.GerakStok) bool { return g.Grup == grup }) {
		return nil, http.StatusConflict, errors.New("catatan ini biaya lain produksi; ubah atau hapus lewat Produksi & Supplier → riwayat produksi")
	}
	for _, t := range s.txs {
		if t.Grup == grup {
			if err := s.cekTerbukaLocked(tanggalTx(t)...); err != nil {
				return nil, http.StatusConflict, err
			}
		}
	}
	for _, t := range baru {
		if err := s.cekTxLocked(t); err != nil {
			return nil, http.StatusConflict, err
		}
	}
	awal, kode, err := s.cekBarangCatatLocked(grup, baru)
	if err != nil {
		return nil, kode, err
	}
	if awal != nil {
		if err := s.simpanStokAwalLocked(awal, oleh, time.Now().UTC()); err != nil {
			return nil, http.StatusInternalServerError, errors.New("gagal menyimpan metode nilai stok")
		}
	}
	nomor := "" // catatan yang diubah tetap memakai nomor jurnalnya
	if i := slices.IndexFunc(s.txs, func(t finance.Transaction) bool { return t.Grup == grup }); i >= 0 {
		nomor = s.txs[i].NoJurnal
	}
	s.txs = slices.DeleteFunc(s.txs, func(t finance.Transaction) bool { return t.Grup == grup })
	var ids []int
	for _, t := range baru {
		t.NoJurnal = nomor
		t.ID = s.nextTxID
		s.nextTxID++
		s.txs = append(s.txs, t)
		ids = append(ids, t.ID)
	}
	return ids, 0, nil
}

// cekBarangCatatLocked: wajib barang per transaksi (400) lalu stok tidak minus (409) pada keadaan
// usulan: transaksi grup lama diganti yang baru. Dasar pembanding = keadaan sekarang + stok awal
// otomatis (bila kartu dimulai oleh catatan ini), tanpa transaksi baru (lihat cekSimpanTxLocked).
// Mengembalikan usulan stok awal otomatis (nil bila tidak perlu). s.mu dipegang.
func (s *tenant) cekBarangCatatLocked(grup int, baru []finance.Transaction) (*finance.GerakStok, int, error) {
	var awal *finance.GerakStok
	for _, t := range baru {
		if err := s.cekBarangLocked(t); err != nil {
			if len(txBerbarang(baru)) > 1 { // rincian hanya bisa ditempel ke satu transaksi
				return nil, http.StatusBadRequest, errPecahCatat
			}
			return nil, http.StatusBadRequest, err
		}
		if awal == nil {
			awal = s.stokAwalTxLocked(t)
		}
	}
	txs := slices.DeleteFunc(slices.Clone(s.txs), func(t finance.Transaction) bool { return t.Grup == grup })
	id := s.nextTxID
	for _, t := range baru {
		t.ID = id
		id++
		txs = append(txs, t)
	}
	gerak := s.tambahGerakUsulan(awal)
	if err := s.cekTanpaMinusDariLocked(s.txs, gerak, txs, gerak); err != nil {
		return nil, http.StatusConflict, err
	}
	return awal, 0, nil
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
	ids, kode, err := s.simpanCatatLocked(grup, txs, currentUser(r).Name)
	if err != nil {
		http.Error(w, err.Error(), kode)
		return
	}
	foto := s.lampirkanBuktiLocked(r, grup, c.Bukti)
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, catatResp{c, grup, ids, s.nomorTxLocked(ids[0]), foto})
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
	// Satu keterangan per catatan: gabungan keterangan berbeda (catatan lama punya per baris).
	var ket []string
	for _, t := range txs {
		if k := strings.TrimSpace(t.Keterangan); k != "" && !slices.Contains(ket, k) {
			ket = append(ket, k)
		}
	}
	out := catatResp{Grup: grup, catatReq: catatReq{Tanggal: txs[0].Tanggal, Pihak: txs[0].Pihak, NoBukti: txs[0].NoBukti,
		Keterangan: strings.Join(ket, "; "), Baris: coa.SusunCatat(txs, p, coa.PetaDari(all))}, NoJurnal: txs[0].NoJurnal}
	for _, t := range txs {
		out.Transaksi = append(out.Transaksi, t.ID)
	}
	out.Rincian, out.BukanBarang = barangCatat(txs)
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
	ids, kode, err := s.simpanCatatLocked(grup, txs, currentUser(r).Name)
	if err != nil {
		http.Error(w, err.Error(), kode)
		return
	}
	out := catatResp{catatReq: c, Grup: grup, Transaksi: ids, NoJurnal: s.nomorTxLocked(ids[0])}
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
	if _, kode, err := s.simpanCatatLocked(grup, nil, currentUser(r).Name); err != nil {
		http.Error(w, err.Error(), kode)
		return
	}
	s.hapusBuktiLocked(grup)
	w.WriteHeader(http.StatusNoContent)
}

// barangCatat: rincian & tanda jasa catatan untuk layar ubah (transaksi berbarang grup itu).
// Data lama stok_id tunggal tampil sebagai rincian satu baris.
func barangCatat(txs []finance.Transaction) ([]finance.RincianBarang, bool) {
	barang := txBerbarang(txs)
	jasa := len(barang) > 0
	for _, i := range barang {
		if b := txs[i].BarisBarang(); len(b) > 0 {
			return b, false
		}
		jasa = jasa && txs[i].BukanBarang
	}
	return nil, jasa
}
