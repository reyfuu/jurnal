package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"pajak/internal/finance"
	"pajak/internal/xlsx"
)

// Kartu stok: stok awal (titik nol), stok opname, produksi, kartu per barang, dan laporan
// mutasi stok. Beli & jual barang tercatat lewat transaksi yang memilih barang
// (finance.Transaction.StokID), jadi tidak ada endpoint terpisah untuk keduanya.

// kartuLocked: s.mu harus terkunci (baca).
func (s *tenant) kartuLocked() finance.KartuStok {
	return finance.HitungKartu(s.stok, s.txs, s.gerak)
}

func (s *tenant) kartu() (finance.KartuStok, []finance.StokItem) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.kartuLocked(), slices.Clone(s.stok)
}

func (s *tenant) stokByID(id int) (finance.StokItem, bool) {
	i := slices.IndexFunc(s.stok, func(x finance.StokItem) bool { return x.ID == id })
	if i < 0 {
		return finance.StokItem{}, false
	}
	return s.stok[i], true
}

// validStok: barang yang dipilih di transaksi harus ada. s.mu dipegang pemanggil atau tidak
// sama sekali — cukup baca singkat.
func (s *tenant) validStok(t finance.Transaction) bool {
	if t.StokID == 0 {
		return true
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.stokByID(t.StokID)
	return ok
}

// stokDipakai: barang yang sudah punya riwayat tidak boleh dihapus (kartunya jadi yatim).
func (s *tenant) stokDipakai(id int) bool {
	if slices.ContainsFunc(s.txs, func(t finance.Transaction) bool { return t.StokID == id }) {
		return true
	}
	return slices.ContainsFunc(s.gerak, func(g finance.GerakStok) bool {
		return slices.ContainsFunc(g.Baris, func(b finance.BarisGerak) bool { return b.StokID == id })
	})
}

func tanggalSah(s string) bool {
	_, err := time.Parse(time.DateOnly, s)
	return err == nil
}

type gerakBody struct {
	Tanggal    string               `json:"tanggal"`
	Keterangan string               `json:"keterangan"`
	Baris      []finance.BarisGerak `json:"baris"`
	ProdukID   int                  `json:"produk_id"`
	Batch      int64                `json:"batch"`
}

func bacaGerak(w http.ResponseWriter, r *http.Request) (gerakBody, bool) {
	var b gerakBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "data tidak terbaca", http.StatusBadRequest)
		return b, false
	}
	b.Keterangan = strings.TrimSpace(b.Keterangan)
	if !tanggalSah(b.Tanggal) {
		http.Error(w, "tanggal wajib diisi (YYYY-MM-DD)", http.StatusBadRequest)
		return b, false
	}
	if len([]rune(b.Keterangan)) > 300 {
		http.Error(w, "keterangan maksimal 300 karakter", http.StatusBadRequest)
		return b, false
	}
	return b, true
}

// cekBarisFisik: tiap barang sekali, ada di daftar stok, jumlah & harga tidak negatif.
func (s *tenant) cekBarisFisik(baris []finance.BarisGerak) error {
	if len(baris) == 0 {
		return fmt.Errorf("isi minimal satu barang")
	}
	seen := map[int]bool{}
	for _, b := range baris {
		it, ok := s.stokByID(b.StokID)
		if !ok {
			return fmt.Errorf("barang #%d tidak ada di Stok & Gudang", b.StokID)
		}
		if seen[b.StokID] {
			return fmt.Errorf("%s diisi dua kali", it.Nama)
		}
		seen[b.StokID] = true
		if b.Fisik < 0 || b.HargaUnit < 0 {
			return fmt.Errorf("jumlah dan harga %s tidak boleh negatif", it.Nama)
		}
	}
	return nil
}

func (s *tenant) simpanGerak(w http.ResponseWriter, r *http.Request, g finance.GerakStok) {
	g.ID = s.nextGerakID
	s.nextGerakID++
	g.Oleh, g.Dibuat = currentUser(r).Name, time.Now().UTC()
	s.gerak = append(s.gerak, g)
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, g)
}

// handleStokAwal: titik nol kartu stok — hitung fisik pertama beserta harga pokoknya.
func (s *tenant) handleStokAwal(w http.ResponseWriter, r *http.Request) {
	b, ok := bacaGerak(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if slices.ContainsFunc(s.gerak, func(g finance.GerakStok) bool { return g.Jenis == finance.GerakAwal }) {
		http.Error(w, "stok awal sudah dicatat; pakai stok opname untuk menyesuaikan jumlah", http.StatusConflict)
		return
	}
	for i := range b.Baris {
		b.Baris[i].Qty = 0
	}
	if err := s.cekBarisFisik(b.Baris); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.simpanGerak(w, r, finance.GerakStok{Tanggal: b.Tanggal, Jenis: finance.GerakAwal, Baris: b.Baris, Keterangan: b.Keterangan})
}

// mulaiLocked: tanggal titik nol, atau error bila belum ada / tanggal sebelum titik nol.
func (s *tenant) cekSetelahMulai(tanggal string) error {
	i := slices.IndexFunc(s.gerak, func(g finance.GerakStok) bool { return g.Jenis == finance.GerakAwal })
	if i < 0 {
		return fmt.Errorf("mulai kartu stok dulu dengan mencatat stok awal")
	}
	if tanggal < s.gerak[i].Tanggal {
		return fmt.Errorf("tanggal tidak boleh sebelum titik nol stok (%s)", s.gerak[i].Tanggal)
	}
	return nil
}

func (s *tenant) handleStokOpname(w http.ResponseWriter, r *http.Request) {
	b, ok := bacaGerak(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.cekSetelahMulai(b.Tanggal); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for i := range b.Baris {
		b.Baris[i].Qty, b.Baris[i].HargaUnit = 0, 0
	}
	if err := s.cekBarisFisik(b.Baris); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.simpanGerak(w, r, finance.GerakStok{Tanggal: b.Tanggal, Jenis: finance.GerakOpname, Baris: b.Baris, Keterangan: b.Keterangan})
}

// handleStokProduksi: jalankan resep sebanyak batch — bahan dari stok berkurang, barang jadi
// (StokItem dengan SKU/nama produk; dibuat bila belum ada) bertambah.
func (s *tenant) handleStokProduksi(w http.ResponseWriter, r *http.Request) {
	b, ok := bacaGerak(w, r)
	if !ok {
		return
	}
	if b.Batch < 1 || b.Batch > 100_000 {
		http.Error(w, "jumlah batch 1–100.000", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.cekSetelahMulai(b.Tanggal); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	pi := slices.IndexFunc(s.produk, func(p finance.Produk) bool { return p.ID == b.ProdukID })
	if pi < 0 {
		http.Error(w, "produk tidak ditemukan", http.StatusNotFound)
		return
	}
	p := s.produk[pi]
	var baris []finance.BarisGerak
	for _, bh := range p.Bahan {
		if bh.StokID == 0 {
			continue // bahan manual tidak dilacak per gudang
		}
		if _, ok := s.stokByID(bh.StokID); !ok {
			http.Error(w, "bahan "+bh.Nama+" sudah tidak ada di Stok & Gudang; perbarui resepnya", http.StatusBadRequest)
			return
		}
		baris = append(baris, finance.BarisGerak{StokID: bh.StokID, Qty: -bh.Takaran * b.Batch})
	}
	jadi := slices.IndexFunc(s.stok, func(x finance.StokItem) bool {
		if p.SKU != "" {
			return strings.EqualFold(x.SKU, p.SKU)
		}
		return strings.EqualFold(x.Nama, p.Nama)
	})
	if jadi < 0 {
		it := finance.StokItem{ID: s.nextStokID, SKU: p.SKU, Nama: p.Nama, Kategori: "Produk Jadi", Satuan: p.Satuan, HargaJual: p.HargaJual}
		s.nextStokID++
		s.stok = append(s.stok, it)
		jadi = len(s.stok) - 1
	}
	baris = append(baris, finance.BarisGerak{StokID: s.stok[jadi].ID, Qty: p.HasilPerBatch * b.Batch * finance.TakaranSkala})
	ket := cmp.Or(b.Keterangan, fmt.Sprintf("Produksi %s × %d batch", p.Nama, b.Batch))
	s.simpanGerak(w, r, finance.GerakStok{Tanggal: b.Tanggal, Jenis: finance.GerakProduksi, Baris: baris, ProdukID: p.ID, Batch: b.Batch, Keterangan: ket})
}

func (s *tenant) handleListGerak(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	out := slices.Clone(s.gerak)
	s.mu.RUnlock()
	slices.SortStableFunc(out, func(a, b finance.GerakStok) int {
		return cmp.Or(cmp.Compare(b.Tanggal, a.Tanggal), cmp.Compare(b.ID, a.ID))
	})
	writeJSON(w, nonNil(out))
}

func (s *tenant) handleDeleteGerak(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.gerak, func(g finance.GerakStok) bool { return g.ID == id })
	if i < 0 {
		http.Error(w, "dokumen stok tidak ditemukan", http.StatusNotFound)
		return
	}
	if s.gerak[i].Jenis == finance.GerakAwal && len(s.gerak) > 1 {
		http.Error(w, "hapus dulu stok opname & produksi sebelum menghapus stok awal", http.StatusConflict)
		return
	}
	s.gerak = slices.Delete(s.gerak, i, i+1)
	w.WriteHeader(http.StatusNoContent)
}

type kartuBarangResp struct {
	Barang finance.StokItem     `json:"barang"`
	Mulai  string               `json:"mulai"`
	Awal   finance.PosisiStok   `json:"awal"` // saldo sebelum 'dari'
	Baris  []finance.BarisKartu `json:"baris"`
}

// rentang: ?dari & ?sampai (YYYY-MM-DD); bawaan bulan berjalan.
func rentang(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	now := time.Now().In(finance.WIB)
	dari := cmp.Or(r.URL.Query().Get("dari"), now.Format("2006-01")+"-01")
	sampai := cmp.Or(r.URL.Query().Get("sampai"), now.Format(time.DateOnly))
	if !tanggalSah(dari) || !tanggalSah(sampai) || sampai < dari {
		http.Error(w, "rentang tanggal tidak valid", http.StatusBadRequest)
		return "", "", false
	}
	return dari, sampai, true
}

func (s *tenant) handleKartuBarang(w http.ResponseWriter, r *http.Request) {
	dari, sampai, ok := rentang(w, r)
	if !ok {
		return
	}
	id, _ := strconv.Atoi(r.PathValue("id"))
	s.mu.RLock()
	it, ada := s.stokByID(id)
	k := s.kartuLocked()
	s.mu.RUnlock()
	if !ada {
		http.Error(w, "barang tidak ditemukan", http.StatusNotFound)
		return
	}
	hari, _ := time.Parse(time.DateOnly, dari)
	out := kartuBarangResp{Barang: it, Mulai: k.Mulai, Awal: k.PosisiPada(id, hari.AddDate(0, 0, -1).Format(time.DateOnly)), Baris: []finance.BarisKartu{}}
	for _, b := range k.Baris {
		if b.StokID == id && b.Tanggal >= dari && b.Tanggal <= sampai {
			out.Baris = append(out.Baris, b)
		}
	}
	writeJSON(w, out)
}

func (s *tenant) handleMutasiStok(w http.ResponseWriter, r *http.Request) {
	dari, sampai, ok := rentang(w, r)
	if !ok {
		return
	}
	k, stok := s.kartu()
	writeJSON(w, k.Mutasi(stok, dari, sampai))
}

func (s *tenant) handleMutasiStokXlsx(w http.ResponseWriter, r *http.Request) {
	dari, sampai, ok := rentang(w, r)
	if !ok {
		return
	}
	k, stok := s.kartu()
	l := k.Mutasi(stok, dari, sampai)
	rows := [][]string{
		{"Laporan Mutasi Stok", dari + " s.d. " + sampai},
		{"Jumlah dalam satuan barang; nilai dalam rupiah (rata-rata bergerak)."},
		{},
		{"SKU", "Barang", "Satuan", "Awal", "Beli", "Hasil produksi", "Terjual", "Dipakai produksi", "Selisih opname", "Akhir",
			"Nilai awal", "Nilai beli", "Nilai produksi", "HPP terjual", "Nilai dipakai", "Nilai selisih opname", "Nilai akhir"},
	}
	baris := func(m finance.MutasiItem) []string {
		q, n := finance.FormatQty, func(v int64) string { return strconv.FormatInt(v, 10) }
		return []string{m.SKU, m.Nama, m.Satuan, q(m.AwalQty), q(m.BeliQty), q(m.ProduksiQty), q(m.JualQty), q(m.PakaiQty), q(m.OpnameQty), q(m.AkhirQty),
			n(m.AwalNilai), n(m.BeliNilai), n(m.ProduksiNilai), n(m.JualNilai), n(m.PakaiNilai), n(m.OpnameNilai), n(m.AkhirNilai)}
	}
	for _, m := range l.Items {
		rows = append(rows, baris(m))
	}
	tot := baris(l.Total)
	for i := 3; i <= 9; i++ {
		tot[i] = "" // jumlah antarbarang beda satuan tidak bermakna
	}
	rows = append(rows, tot)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="mutasi-stok-%s-%s.xlsx"`, dari, sampai))
	_ = xlsx.Write(w, []xlsx.Sheet{{Name: "Mutasi Stok", Rows: rows}})
}

// pilihanBarang: daftar ringkas untuk memilih barang di formulir transaksi; dibuka untuk
// semua yang boleh mencatat transaksi (termasuk Kasir yang tidak melihat Stok & Gudang).
type pilihanBarang struct {
	ID         int    `json:"id"`
	SKU        string `json:"sku"`
	Nama       string `json:"nama"`
	Kategori   string `json:"kategori"`
	Satuan     string `json:"satuan"`
	JumlahMili int64  `json:"jumlah_mili"`
	HargaJual  int64  `json:"harga_jual"`
}

func (s *tenant) handlePilihanBarang(w http.ResponseWriter, _ *http.Request) {
	k, stok := s.kartu()
	out := struct {
		Mulai  string          `json:"mulai"`
		Barang []pilihanBarang `json:"barang"`
	}{Mulai: k.Mulai, Barang: []pilihanBarang{}}
	for _, it := range stok {
		q := it.Jumlah * finance.TakaranSkala
		if k.Mulai != "" {
			q = k.Posisi[it.ID].Qty
		}
		out.Barang = append(out.Barang, pilihanBarang{it.ID, it.SKU, it.Nama, it.Kategori, it.Satuan, q, it.HargaJual})
	}
	writeJSON(w, out)
}

func (s *Server) kartuStokRoutes() {
	s.mux.HandleFunc("GET /api/v1/stok/pilihan", s.guard("transaksi", s.on((*tenant).handlePilihanBarang)))
	g := func(h http.HandlerFunc) http.HandlerFunc { return s.guard("stok-gudang", h) }
	m := s.mux
	m.HandleFunc("POST /api/v1/stok/awal", s.persisting(g(s.on((*tenant).handleStokAwal))))
	m.HandleFunc("POST /api/v1/stok/opname", s.persisting(g(s.on((*tenant).handleStokOpname))))
	m.HandleFunc("POST /api/v1/stok/produksi", s.persisting(g(s.on((*tenant).handleStokProduksi))))
	m.HandleFunc("GET /api/v1/stok/gerak", g(s.on((*tenant).handleListGerak)))
	m.HandleFunc("DELETE /api/v1/stok/gerak/{id}", s.persisting(g(s.on((*tenant).handleDeleteGerak))))
	m.HandleFunc("GET /api/v1/stok/kartu/{id}", g(s.on((*tenant).handleKartuBarang)))
	m.HandleFunc("GET /api/v1/stok/mutasi", g(s.on((*tenant).handleMutasiStok)))
	m.HandleFunc("GET /api/v1/stok/mutasi.xlsx", g(s.exporting(s.on((*tenant).handleMutasiStokXlsx))))
}
