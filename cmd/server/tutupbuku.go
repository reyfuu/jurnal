package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"pajak/internal/coa"
	"slices"
	"time"

	"pajak/internal/auth"
	"pajak/internal/finance"
)

// Tutup buku bulanan: daftar cek akhir bulan, lalu bulan itu (dan semua sebelumnya) dikunci
// supaya laporan yang sudah dipakai (pajak, bank) tidak berubah diam-diam. Hanya Owner yang
// bisa membuka kembali.

// cekTerbukaLocked: error bila salah satu tanggal jatuh di bulan yang sudah ditutup.
// Pemanggil memegang s.mu.
func (s *tenant) cekTerbukaLocked(tanggal ...string) error {
	if s.tutupBuku == "" {
		return nil
	}
	for _, t := range tanggal {
		if len(t) >= 7 && t[:7] <= s.tutupBuku {
			return fmt.Errorf("buku bulan %s sudah ditutup; buka kembali di menu Tutup Buku bila memang perlu diubah", finance.LabelPeriode(t[:7]))
		}
	}
	return nil
}

// cekTxLocked: transaksi baru / diubah harus di bulan yang belum ditutup dan tidak sebelum
// tanggal saldo awal yang diisi pengguna (posisinya sudah tercakup saldo awal; dicatat lagi =
// terhitung dua kali). Tanpa saldo awal isian, tanggal mulai ikut transaksi paling awal.
func (s *tenant) cekTxLocked(t finance.Transaction) error {
	if s.opening != nil && t.Tanggal < s.opening.Tanggal {
		return fmt.Errorf("tanggal %s sebelum tanggal saldo awal (%s); posisi sebelum tanggal itu sudah tercakup di saldo awal. Bila memang mau mencatat transaksi lama, mundurkan tanggal saldo awal dulu di menu Saldo Awal", t.Tanggal, s.opening.Tanggal)
	}
	return s.cekTerbukaLocked(tanggalTx(t)...)
}

func tanggalTx(t finance.Transaction) []string {
	out := []string{t.Tanggal}
	if t.Lunas && t.TanggalLunas != "" {
		out = append(out, t.TanggalLunas)
	}
	return out
}

type cekTutup struct {
	Kode   string `json:"kode"`
	Judul  string `json:"judul"`
	Status string `json:"status"` // ok | peringatan | info
	Detail string `json:"detail"`
	View   string `json:"view,omitempty"`
}

type tutupBukuResp struct {
	Periode     string     `json:"periode"`
	Label       string     `json:"label"`
	TutupSampai string     `json:"tutup_sampai"`
	Terkunci    bool       `json:"terkunci"`
	BisaTutup   bool       `json:"bisa_tutup"` // bulan sebelumnya sudah ditutup (atau ini bulan pertama)
	BisaBuka    bool       `json:"bisa_buka"`
	Cek         []cekTutup `json:"cek"`
	Laba        int64      `json:"laba"`
	EBITDA      int64      `json:"ebitda"`
	Penyusutan  int64      `json:"penyusutan"`
	KasAkhir    int64      `json:"kas_akhir"`
}

func (s *tenant) ringkasTutup(period string, u auth.User) tutupBukuResp {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rep := s.reportLocked(period)
	k := s.kartuLocked()
	akhir := period + "-31"
	if t, err := time.Parse("2006-01", period); err == nil {
		akhir = t.AddDate(0, 1, -1).Format(time.DateOnly)
	}
	ob := s.bukuAwalLocked()
	out := tutupBukuResp{Periode: period, Label: finance.LabelPeriode(period), TutupSampai: s.tutupBuku,
		Terkunci: s.tutupBuku != "" && period <= s.tutupBuku, Laba: rep.Income.NetProfit, EBITDA: rep.Income.EBITDA,
		Penyusutan: rep.Income.Depreciation + rep.Income.Amortization, KasAkhir: rep.CashFlow.ClosingBalance}
	// Menutup satu bulan sekaligus mengunci semua bulan sebelumnya.
	out.BisaTutup = !out.Terkunci && period <= finance.BulanBerjalan()
	out.BisaBuka = out.Terkunci && period == s.tutupBuku && u.Role == auth.Owner

	n := len(finance.ForPeriod(s.txs, period))
	c := cekTutup{Kode: "transaksi", Judul: "Semua transaksi bulan ini sudah dicatat", View: "transaksi", Status: "ok",
		Detail: fmt.Sprintf("%d transaksi tercatat. Pastikan penjualan, belanja, dan biaya sudah lengkap (cek mutasi rekening).", n)}
	if n == 0 {
		c.Status, c.Detail = "peringatan", "Belum ada transaksi di bulan ini."
	}
	out.Cek = append(out.Cek, c)

	// Transaksi yang kemungkinan tercatat dua kali (mis. dari foto nota lalu ikut impor mutasi).
	c = cekTutup{Kode: "dobel", Judul: "Tidak ada transaksi dobel", View: "transaksi", Status: "ok", Detail: "Tidak ada pasangan transaksi dengan arah, nominal, dan tanggal yang sama."}
	if p := finance.PasanganMirip(s.txs, period); len(p) > 0 {
		c.Status, c.Detail = "peringatan", fmt.Sprintf("%d pasang transaksi mirip (arah & nominal sama, tanggal berdekatan) — cek di Transaksi apakah ada yang tercatat dua kali.", len(p))
	}
	out.Cek = append(out.Cek, c)

	for _, piutang := range []bool{true, false} {
		um := finance.UmurSaldo(ob, s.txs, akhir, piutang)
		lama := um.Kelompok[1].Nilai + um.Kelompok[2].Nilai + um.Kelompok[3].Nilai
		c := cekTutup{Kode: "piutang", Judul: "Piutang lewat 30 hari ditindaklanjuti", View: "piutang-utang", Status: "ok",
			Detail: fmt.Sprintf("Piutang belum lunas %s.", finance.FormatRupiahPenuh(um.Total))}
		if !piutang {
			c = cekTutup{Kode: "utang", Judul: "Utang lewat 30 hari dibayar / dijadwalkan", View: "piutang-utang", Status: "ok",
				Detail: fmt.Sprintf("Utang belum dibayar %s.", finance.FormatRupiahPenuh(um.Total))}
		}
		if lama > 0 {
			c.Status = "peringatan"
			c.Detail += fmt.Sprintf(" %s sudah lebih dari 30 hari — catat pelunasannya bila sudah dibayar.", finance.FormatRupiahPenuh(lama))
		}
		out.Cek = append(out.Cek, c)
	}

	if k.Mulai != "" && k.Mulai <= akhir {
		opname := slices.ContainsFunc(s.gerak, func(g finance.GerakStok) bool {
			return g.Jenis == finance.GerakOpname && g.Tanggal >= period+"-01" && g.Tanggal <= akhir
		})
		c := cekTutup{Kode: "opname", Judul: "Stok opname akhir bulan", View: "stok-gudang", Status: "ok", Detail: "Stok opname bulan ini sudah dicatat."}
		if !opname {
			c.Status, c.Detail = "peringatan", "Belum ada stok opname di bulan ini. Hitung fisik barang supaya selisih stok tercatat."
		}
		out.Cek = append(out.Cek, c)
		minus := 0
		for _, it := range s.stok {
			if k.PosisiPada(it.ID, akhir).Qty < 0 {
				minus++
			}
		}
		c = cekTutup{Kode: "minus", Judul: "Tidak ada stok minus", View: "stok-gudang", Status: "ok", Detail: "Semua stok tidak minus di akhir bulan."}
		if minus > 0 {
			c.Status, c.Detail = "peringatan", fmt.Sprintf("%d barang minus di akhir bulan — ada pembelian/produksi yang belum dicatat.", minus)
		}
		out.Cek = append(out.Cek, c)
	} else {
		out.Cek = append(out.Cek, cekTutup{Kode: "opname", Judul: "Kartu stok", View: "stok-gudang", Status: "info",
			Detail: "Kartu stok belum dimulai; persediaan memakai angka saldo awal."})
	}

	out.Cek = append(out.Cek, cekTutup{Kode: "penyusutan", Judul: "Penyusutan & amortisasi dicatat", View: "laporan-keuangan", Status: "ok",
		Detail: fmt.Sprintf("Otomatis dari register aset: %s bulan ini.", finance.FormatRupiahPenuh(out.Penyusutan))})

	kasMinus := len(finance.CashWarnings(finance.ForPeriod(s.txs, period), rep.CashFlow.OpeningBalance))
	c = cekTutup{Kode: "kas", Judul: "Saldo kas tidak pernah minus", View: "transaksi", Status: "ok", Detail: fmt.Sprintf("Kas akhir bulan %s.", finance.FormatRupiahPenuh(out.KasAkhir))}
	if kasMinus > 0 {
		c.Status, c.Detail = "peringatan", fmt.Sprintf("Kas sempat minus di %d hari — biasanya ada pemasukan yang belum dicatat.", kasMinus)
	}
	out.Cek = append(out.Cek, c)

	for _, k := range s.cekSaldoAwalLocked() {
		if k.Status == "selisih" {
			out.Cek = append(out.Cek, cekTutup{Kode: "saldo_" + k.Kode, Judul: "Saldo awal cocok: " + k.Judul, View: k.View, Status: "peringatan", Detail: k.Pesan})
		}
	}

	c = cekTutup{Kode: "neraca", Judul: "Neraca seimbang", View: "laporan-keuangan", Status: "ok", Detail: "Aktiva = Pasiva."}
	if !rep.BalanceSheet.IsBalanced {
		c.Status, c.Detail = "peringatan", "Aktiva ≠ Pasiva — hubungi tim MasterPajak AI."
	}
	out.Cek = append(out.Cek, c)
	return out
}

func (s *tenant) handleTutupBuku(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if !finance.PeriodeSah(period) {
		http.Error(w, "periode tidak valid", http.StatusBadRequest)
		return
	}
	writeJSON(w, s.ringkasTutup(period, currentUser(r)))
}

func (s *tenant) handleTutupBulan(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Periode string `json:"periode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || !finance.PeriodeSah(b.Periode) {
		http.Error(w, "periode tidak valid", http.StatusBadRequest)
		return
	}
	if b.Periode > finance.BulanBerjalan() {
		http.Error(w, "bulan yang belum berjalan tidak bisa ditutup", http.StatusBadRequest)
		return
	}
	if !s.ringkasTutup(b.Periode, currentUser(r)).BisaTutup {
		http.Error(w, "bulan ini sudah ditutup", http.StatusConflict)
		return
	}
	s.mu.Lock()
	s.tutupBuku = b.Periode
	s.mu.Unlock()
	writeJSON(w, s.ringkasTutup(b.Periode, currentUser(r)))
}

// handleBukaBulan: Owner membuka kembali bulan terakhir yang ditutup.
func (s *tenant) handleBukaBulan(w http.ResponseWriter, r *http.Request) {
	if currentUser(r).Role != auth.Owner {
		http.Error(w, "hanya Owner yang bisa membuka kembali buku yang sudah ditutup", http.StatusForbidden)
		return
	}
	s.mu.Lock()
	if s.tutupBuku == "" {
		s.mu.Unlock()
		http.Error(w, "belum ada bulan yang ditutup", http.StatusConflict)
		return
	}
	dibuka := s.tutupBuku
	t, _ := time.Parse("2006-01", s.tutupBuku)
	s.tutupBuku = t.AddDate(0, -1, 0).Format("2006-01")
	if s.tutupBuku < s.bukuAwalLocked().Period() {
		s.tutupBuku = ""
	}
	s.mu.Unlock()
	writeJSON(w, s.ringkasTutup(dibuka, currentUser(r)))
}

// --- Pelunasan saldo awal ---

func (s *tenant) handlePelunasanAwal(w http.ResponseWriter, r *http.Request) {
	var p finance.PelunasanAwal
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "data tidak terbaca", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.opening == nil {
		http.Error(w, "saldo awal belum diisi", http.StatusConflict)
		return
	}
	if err := s.cekTerbukaLocked(p.Tanggal); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if p.Akun == "" {
		p.Akun = s.rekeningBawaanLocked() // bawaan Bank
	} else if a, ok := coa.Find(s.coaTemplateLocked(), p.Akun); !ok || coa.JenisLawan(a, coa.PetaDari(s.coaTemplateLocked())) != coa.LawanRekening {
		http.Error(w, "pelunasan hanya bisa lewat akun Kas / Bank", http.StatusBadRequest)
		return
	}
	ob := *s.opening
	ob.Pelunasan = slices.Clone(ob.Pelunasan)
	if err := ob.TambahPelunasan(p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.opening = &ob
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, p)
}

// --- Langkah memulai ---

type langkahMulai struct {
	Kode       string `json:"kode"`
	Judul      string `json:"judul"`
	Penjelasan string `json:"penjelasan"`
	Selesai    bool   `json:"selesai"`
	View       string `json:"view"`
	Aksi       string `json:"aksi,omitempty"`
	// Peringatan: selisih saldo awal dengan rincian langkah ini (lihat cek_saldo_awal.go).
	Peringatan string `json:"peringatan,omitempty"`
}

func (s *tenant) handleLangkahMulai(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dagang := s.profile == nil || s.profile.JenisUsaha != "jasa"
	out := []langkahMulai{
		{Kode: "profil", Judul: "Lengkapi profil usaha", Penjelasan: "Nama, bentuk usaha, bidang usaha (KLU), dan status PKP — menentukan pajak yang berlaku.",
			Selesai: s.profile != nil && s.profile.KLU != "", View: "profil-usaha"},
		// Aset tetap & barang diisi sebelum saldo awal: nilai buku aset dan nilai persediaannya
		// yang mengisi baris saldo awal, bukan diketik dua kali.
		{Kode: "aset", Judul: "Daftarkan aset tetap", Penjelasan: "Kendaraan, mesin, peralatan, bangunan yang sudah dimiliki — nilai bukunya otomatis masuk saldo awal. Lewati bila tidak punya.",
			Selesai: len(s.assets) > 0 || s.opening != nil, View: "saldo-awal"},
	}
	if dagang {
		out = append(out, langkahMulai{Kode: "barang", Judul: "Daftarkan barang & bahan", Penjelasan: "Isi daftar barang di Stok & Gudang (bisa impor Excel); nilainya jadi persediaan di saldo awal.",
			Selesai: len(s.stok) > 0, View: "stok-gudang"})
	}
	out = append(out, langkahMulai{Kode: "saldo", Judul: "Isi saldo awal", Penjelasan: "Kas, piutang, utang, dan pinjaman pada hari pertama memakai aplikasi.",
		Selesai: s.opening != nil, View: "saldo-awal"})
	if dagang {
		out = append(out,
			langkahMulai{Kode: "kartu", Judul: "Mulai kartu stok", Penjelasan: "Hitung fisik pertama supaya stok & HPP dihitung otomatis.",
				Selesai: s.kartuLocked().Mulai != "", View: "stok-gudang", Aksi: "mulai"},
		)
	}
	out = append(out,
		langkahMulai{Kode: "transaksi", Judul: "Catat transaksi pertama", Penjelasan: "Penjualan, belanja, dan biaya harian — atau impor mutasi rekening.",
			Selesai: len(s.txs) > 0, View: "transaksi", Aksi: "baru"},
		langkahMulai{Kode: "tutup", Judul: "Tutup buku bulan pertama", Penjelasan: "Cek akhir bulan lalu kunci bulannya supaya laporan tidak berubah.",
			Selesai: s.tutupBuku != "", View: "tutup-buku"},
	)
	for _, c := range s.cekSaldoAwalLocked() {
		if c.Status != "selisih" {
			continue
		}
		for i := range out {
			if out[i].Kode == c.Langkah {
				out[i].Peringatan = c.Pesan
			}
		}
	}
	writeJSON(w, out)
}

func (s *Server) alurKerjaRoutes() {
	m := s.mux
	m.HandleFunc("GET /api/v1/tutup-buku", s.guard("tutup-buku", s.on((*tenant).handleTutupBuku)))
	m.HandleFunc("POST /api/v1/tutup-buku", s.persisting(s.guard("tutup-buku", s.on((*tenant).handleTutupBulan))))
	m.HandleFunc("DELETE /api/v1/tutup-buku", s.persisting(s.guard("tutup-buku", s.on((*tenant).handleBukaBulan))))
	m.HandleFunc("POST /api/v1/opening/pelunasan", s.persisting(s.guard("transaksi-kelola", s.on((*tenant).handlePelunasanAwal))))
	m.HandleFunc("GET /api/v1/mulai", s.guard("", s.on((*tenant).handleLangkahMulai)))
	m.HandleFunc("GET /api/v1/saldo-minus", s.guard("", s.on((*tenant).handleSaldoMinus)))
	m.HandleFunc("GET /api/v1/saldo-awal/cek", s.guard("", s.on((*tenant).handleCekSaldoAwal)))
	m.HandleFunc("POST /api/v1/saldo-awal/saran", s.guard("saldo-awal", s.handleSaranSaldo))
}
