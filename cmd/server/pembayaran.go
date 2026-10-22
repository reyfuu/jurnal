package main

import (
	"cmp"
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"pajak/internal/coa"
	"pajak/internal/finance"
)

// Pembayaran piutang/utang (Accurate M1): satu dokumen, bisa sebagian, ke beberapa tagihan.
// Idempotency-Key mencegah pembayaran tercatat dua kali saat klik ganda / retry; batal
// mencabut cicilannya dengan alasan, hanya sekali, dan hanya di bulan yang belum ditutup.

func (s *Server) pembayaranRoutes() {
	s.mux.HandleFunc("GET /api/v1/pembayaran", s.guard("umur-piutang", s.on((*tenant).handleListPembayaran)))
	s.mux.HandleFunc("POST /api/v1/pembayaran", s.persisting(s.guard("transaksi-kelola", s.on((*tenant).handleBuatPembayaran))))
	s.mux.HandleFunc("POST /api/v1/pembayaran/{id}/batal", s.persisting(s.guard("transaksi-kelola", s.on((*tenant).handleBatalPembayaran))))
}

func (s *tenant) handleListPembayaran(w http.ResponseWriter, r *http.Request) {
	jenis := r.URL.Query().Get("jenis")
	s.mu.RLock()
	out := slices.DeleteFunc(slices.Clone(s.pembayaran), func(p finance.Pembayaran) bool { return jenis != "" && p.Jenis != jenis })
	s.mu.RUnlock()
	slices.Reverse(out) // terbaru dulu
	writeJSON(w, nonNil(out))
}

func (s *tenant) handleBuatPembayaran(w http.ResponseWriter, r *http.Request) {
	kunci, err := kunciIdempoten(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var p finance.Pembayaran
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "data pembayaran tidak valid", http.StatusBadRequest)
		return
	}
	if err := finance.ValidatePembayaran(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if p.Lawan != "" {
		all, _, _ := s.akunSaldoAwal()
		if a, ok := coa.Find(all, p.Lawan); !ok || coa.JenisLawan(a, coa.PetaDari(all)) != coa.LawanRekening {
			http.Error(w, "pembayaran hanya bisa lewat akun Kas / Bank", http.StatusBadRequest)
			return
		}
	}
	p.Kunci, p.Oleh, p.Dibuat, p.Batal = kunci, currentUser(r).Name, time.Now().UTC().Truncate(time.Second), nil
	s.mu.Lock()
	defer s.mu.Unlock()
	// Kunci sama = permintaan yang sama diulang: kembalikan dokumen yang sudah tercatat.
	if i := slices.IndexFunc(s.pembayaran, func(x finance.Pembayaran) bool { return kunci != "" && x.Kunci == kunci }); i >= 0 {
		writeJSON(w, s.pembayaran[i])
		return
	}
	hasil, kode, err := s.catatPembayaranLocked(p)
	if err != nil {
		http.Error(w, err.Error(), kode)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, hasil)
}

// catatPembayaranLocked memvalidasi lalu menerapkan pembayaran (alokasi invoice dipecah ke
// transaksi DPP lalu PPN-nya). kode = status HTTP bila gagal. s.mu dipegang.
func (s *tenant) catatPembayaranLocked(p finance.Pembayaran) (finance.Pembayaran, int, error) {
	if err := s.cekTerbukaLocked(p.Tanggal); err != nil {
		return p, http.StatusConflict, err
	}
	p.Alokasi = s.pecahAlokasiLocked(p.Alokasi)
	p.ID = s.nextPembayaranID
	txs, err := finance.Bayar(s.txs, p)
	if err != nil {
		return p, http.StatusBadRequest, err
	}
	if err := finance.ValidatePembayaran(&p); err != nil { // total ulang setelah dipecah
		return p, http.StatusBadRequest, err
	}
	if i := slices.IndexFunc(s.txs, func(t finance.Transaction) bool { return t.ID == p.Alokasi[0].TransaksiID }); i >= 0 {
		p.Pihak = s.txs[i].Pihak
	}
	s.txs = txs
	s.nextPembayaranID++
	s.pembayaran = append(slices.Clone(s.pembayaran), p)
	s.segarkanInvoiceLocked(p.Alokasi)
	return p, 0, nil
}

// pecahAlokasiLocked: alokasi ke satu baris invoice (sebesar DPP + PPN) dibagi ke transaksi DPP
// lalu PPN-nya sebesar sisa masing-masing; alokasi ke transaksi yang sama digabung. Kelebihan
// ditinggal di transaksi terakhir supaya ditolak Bayar ("melebihi sisa").
func (s *tenant) pecahAlokasiLocked(in []finance.Alokasi) []finance.Alokasi {
	cari := func(id int) (finance.Transaction, bool) {
		i := slices.IndexFunc(s.txs, func(t finance.Transaction) bool { return t.ID == id })
		if i < 0 {
			return finance.Transaction{}, false
		}
		return s.txs[i], true
	}
	var out []finance.Alokasi
	tambah := func(id int, n int64) {
		if i := slices.IndexFunc(out, func(a finance.Alokasi) bool { return a.TransaksiID == id }); i >= 0 {
			out[i].Nominal += n
			return
		}
		out = append(out, finance.Alokasi{TransaksiID: id, Nominal: n})
	}
	terpakai := func(id int) int64 {
		for _, a := range out {
			if a.TransaksiID == id {
				return a.Nominal
			}
		}
		return 0
	}
	for _, a := range in {
		ids := []int{a.TransaksiID}
		if t, ok := cari(a.TransaksiID); ok && t.InvoiceID != 0 {
			if inv := slices.IndexFunc(s.invoices, func(x finance.Invoice) bool { return x.ID == t.InvoiceID }); inv >= 0 {
				for _, id := range []int{s.invoices[inv].TransaksiID, s.invoices[inv].TransaksiPPNID} {
					if id != 0 && id != a.TransaksiID {
						ids = append(ids, id)
					}
				}
			}
		}
		n := a.Nominal
		for k, id := range ids {
			ambil := n
			if t, ok := cari(id); ok && k < len(ids)-1 {
				ambil = min(n, max(0, t.Sisa()-terpakai(id)))
			}
			if ambil > 0 {
				tambah(id, ambil)
				n -= ambil
			}
		}
	}
	return out
}

// segarkanInvoiceLocked: status invoice yang transaksinya terkena pembayaran mengikuti
// transaksinya — lunas bila semua lunas, kembali terkirim bila pembayaran dibatalkan.
func (s *tenant) segarkanInvoiceLocked(alokasi []finance.Alokasi) {
	kena := map[int]bool{}
	for _, a := range alokasi {
		if i := slices.IndexFunc(s.txs, func(t finance.Transaction) bool { return t.ID == a.TransaksiID }); i >= 0 && s.txs[i].InvoiceID != 0 {
			kena[s.txs[i].InvoiceID] = true
		}
	}
	for i, inv := range s.invoices {
		if !kena[inv.ID] {
			continue
		}
		lunas, tanggal := true, ""
		for _, id := range []int{inv.TransaksiID, inv.TransaksiPPNID} {
			j := slices.IndexFunc(s.txs, func(t finance.Transaction) bool { return id != 0 && t.ID == id })
			if j < 0 {
				continue
			}
			lunas = lunas && s.txs[j].Lunas
			tanggal = max(tanggal, cmp.Or(s.txs[j].TanggalLunas, s.txs[j].Tanggal))
		}
		switch {
		case lunas:
			inv.Status, inv.TanggalLunas = finance.StatusLunas, tanggal
		case inv.Status == finance.StatusLunas:
			inv.Status, inv.TanggalLunas = finance.StatusTerkirim, ""
		}
		s.invoices[i] = inv
	}
}

func (s *tenant) handleBatalPembayaran(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Alasan string `json:"alasan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "data tidak valid", http.StatusBadRequest)
		return
	}
	req.Alasan = clip(strings.TrimSpace(req.Alasan), 300)
	if req.Alasan == "" {
		http.Error(w, "alasan pembatalan wajib diisi", http.StatusBadRequest)
		return
	}
	id, _ := strconv.Atoi(r.PathValue("id"))
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.pembayaran, func(p finance.Pembayaran) bool { return p.ID == id })
	if i < 0 {
		http.Error(w, "pembayaran tidak ditemukan", http.StatusNotFound)
		return
	}
	p := s.pembayaran[i]
	if p.Batal != nil {
		http.Error(w, "pembayaran ini sudah dibatalkan", http.StatusConflict)
		return
	}
	if err := s.cekTerbukaLocked(p.Tanggal); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	s.txs = finance.BatalBayar(s.txs, p.ID)
	p.Batal = &finance.BatalPembayaran{Alasan: req.Alasan, Oleh: currentUser(r).Name, Pada: time.Now().UTC().Truncate(time.Second)}
	s.pembayaran = slices.Clone(s.pembayaran)
	s.pembayaran[i] = p
	s.segarkanInvoiceLocked(p.Alokasi)
	writeJSON(w, p)
}


// adaCicilan: salah satu transaksi itu sudah punya pembayaran.
func adaCicilan(txs ...finance.Transaction) bool {
	return slices.ContainsFunc(txs, func(t finance.Transaction) bool { return len(t.Cicilan) > 0 })
}

