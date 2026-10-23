package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"pajak/internal/coa"
	"pajak/internal/finance"
)

// Halaman Catat produksi: beberapa produk sekaligus, tiap produk dengan pemakaian bahan aktual
// dan biaya lain (tenaga harian, gas, ongkir). Bila kartu stok belum dimulai, jumlah & harga
// pokok di daftar barang dipakai sebagai stok awal pada tanggal produksi, jadi produksi pertama
// tidak terhalang. Biaya lain dibayar dari Kas/Bank (transaksi satu grup) dan ikut menjadi nilai
// produk jadi (lihat GerakStok.Biaya).

type biayaProduksi struct {
	Keterangan string `json:"keterangan"`
	Akun       string `json:"akun,omitempty"` // kosong = akun pembelian / persediaan bawaan
	Nominal    int64  `json:"nominal"`
}

type produkCatat struct {
	ProdukID   int                  `json:"produk_id"`
	Hasil      int64                `json:"hasil"` // per seribu satuan
	Bahan      []finance.BarisGerak `json:"bahan"`
	Biaya      []biayaProduksi      `json:"biaya"`
	Keterangan string               `json:"keterangan,omitempty"`
}

type produksiCatatReq struct {
	Tanggal   string        `json:"tanggal"`
	BayarDari string        `json:"bayar_dari,omitempty"` // akun kas/bank untuk biaya lain
	Produk    []produkCatat `json:"produk"`
}

type produksiCatatResp struct {
	Gerak     []finance.GerakStok `json:"gerak"`
	StokAwal  bool                `json:"stok_awal"` // kartu stok baru dimulai otomatis
	Transaksi int                 `json:"transaksi"` // jumlah transaksi biaya lain
}

func (s *tenant) handleCatatProduksi(w http.ResponseWriter, r *http.Request) {
	var req produksiCatatReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "data produksi tidak terbaca", http.StatusBadRequest)
		return
	}
	if !tanggalSah(req.Tanggal) {
		http.Error(w, "tanggal produksi wajib diisi (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	if len(req.Produk) == 0 || len(req.Produk) > 20 {
		http.Error(w, "isi 1–20 produk", http.StatusBadRequest)
		return
	}
	all, _, p := s.akunSaldoAwal()
	peta := coa.PetaDari(all)
	bayar := cmp.Or(peta.Kode(coa.PeranBank), peta.Kode(coa.PeranKas)) // bawaan Bank
	if req.BayarDari != "" {
		a, ok := coa.Find(all, req.BayarDari)
		if !ok || coa.JenisLawan(a, peta) != coa.LawanRekening {
			http.Error(w, "biaya lain hanya bisa dibayar dari akun Kas / Bank", http.StatusBadRequest)
			return
		}
		bayar = a.Kode
	}
	akunBiaya := coa.AutoAccount("Beli Bahan / Stok", p, peta)

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.cekTerbukaLocked(req.Tanggal); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	out := produksiCatatResp{Gerak: []finance.GerakStok{}}
	mulai := slices.ContainsFunc(s.gerak, func(g finance.GerakStok) bool { return g.Jenis == finance.GerakAwal })
	if !mulai {
		if len(s.stok) == 0 {
			http.Error(w, "daftar barang masih kosong; tambah bahan & produk di Stok & Gudang dulu", http.StatusBadRequest)
			return
		}
	} else if err := s.cekSetelahMulai(req.Tanggal); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Susun semua dokumen & transaksi dulu; baru disimpan bila semuanya sah.
	var gerak []finance.GerakStok
	var txs []finance.Transaction
	grup := s.grupBerikutLocked()
	for i, pr := range req.Produk {
		pi := slices.IndexFunc(s.produk, func(x finance.Produk) bool { return x.ID == pr.ProdukID })
		if pi < 0 {
			http.Error(w, fmt.Sprintf("produk ke-%d tidak ditemukan", i+1), http.StatusBadRequest)
			return
		}
		prod := s.produk[pi]
		if pr.Hasil <= 0 {
			http.Error(w, "isi jumlah jadi untuk "+prod.Nama, http.StatusBadRequest)
			return
		}
		batch := int64(1)
		if per := prod.HasilPerBatch * finance.TakaranSkala; per > 0 {
			batch = max(1, (pr.Hasil+per/2)/per)
		}
		g, err := s.gerakProduksiLocked(gerakBody{Tanggal: req.Tanggal, ProdukID: prod.ID, Batch: batch, Bahan: pr.Bahan, Hasil: pr.Hasil,
			Keterangan: strings.TrimSpace(pr.Keterangan)})
		if err != nil {
			http.Error(w, prod.Nama+": "+err.Error(), http.StatusBadRequest)
			return
		}
		for _, b := range pr.Biaya {
			if b.Nominal == 0 {
				continue // baris biaya kosong
			}
			if b.Nominal < 0 {
				http.Error(w, "biaya lain tidak boleh minus", http.StatusBadRequest)
				return
			}
			akun := akunBiaya
			if b.Akun != "" {
				if _, ok := coa.Find(all, b.Akun); !ok {
					http.Error(w, "akun biaya "+b.Akun+" tidak ada di daftar akun", http.StatusBadRequest)
					return
				}
				akun = b.Akun
			}
			t := finance.Transaction{Tanggal: req.Tanggal, Jenis: finance.JenisBeliStok, Nominal: b.Nominal, Akun: akun, Lunas: true, Lawan: bayar,
				Keterangan: "Biaya produksi " + prod.Nama + ": " + strings.TrimSpace(b.Keterangan), Grup: grup}
			if err := finance.Validate(&t); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := s.cekTxLocked(t); err != nil {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			g.Biaya += t.Nominal
			txs = append(txs, t)
		}
		if g.Biaya > 0 {
			g.Grup = grup
			grup++
		}
		gerak = append(gerak, g)
	}

	oleh, kini := currentUser(r).Name, time.Now().UTC()
	if !mulai {
		// Kartu stok belum dimulai: daftar barang hari ini jadi stok awal (titik nol).
		awal := finance.GerakStok{Tanggal: req.Tanggal, Jenis: finance.GerakAwal, Keterangan: "Stok awal otomatis dari daftar barang (produksi pertama)"}
		for _, it := range s.stok {
			awal.Baris = append(awal.Baris, finance.BarisGerak{StokID: it.ID, Fisik: max(0, it.Jumlah) * finance.TakaranSkala, HargaUnit: it.HargaPokok})
		}
		if err := s.kunciMetodeStokLocked(); err != nil {
			http.Error(w, "gagal menyimpan metode nilai stok", http.StatusInternalServerError)
			return
		}
		awal.ID, awal.Oleh, awal.Dibuat = s.nextGerakID, oleh, kini
		s.nextGerakID++
		s.gerak = append(s.gerak, awal)
		out.StokAwal = true
	}
	for _, g := range gerak {
		g.ID, g.Oleh, g.Dibuat = s.nextGerakID, oleh, kini
		s.nextGerakID++
		s.gerak = append(s.gerak, g)
		out.Gerak = append(out.Gerak, g)
	}
	for _, t := range txs {
		t.ID = s.nextTxID
		s.nextTxID++
		s.txs = append(s.txs, t)
	}
	s.beriRefLocked()
	out.Transaksi = len(txs)
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, out)
}
