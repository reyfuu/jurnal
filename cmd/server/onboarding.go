package main

import (
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"pajak/internal/coa"
	"pajak/internal/finance"
	"pajak/internal/xlsx"
)

// Impor data awal onboarding: satu workbook .xlsx berisi saldo awal rinci (Kas dan Bank,
// Piutang, …, Modal), Karyawan, dan Transaksi. Tanpa ?commit=1 hanya pratinjau — tidak ada
// yang disimpan.

func (s *tenant) handleOnboardingTemplate(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	metode := s.metodeUsaha()
	s.mu.RUnlock()
	sheets := make([]xlsx.Sheet, 0, 12)
	for _, sh := range finance.OnboardingTemplate(metode) {
		sheets = append(sheets, xlsx.Sheet{Name: sh.Nama, Rows: sh.Rows, Tabel: tabelXLSX(sh.Format)})
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="template-data-awal.xlsx"`)
	_ = xlsx.Write(w, sheets)
}

// tabelXLSX: format tabel template (paket finance) → xlsx.
func tabelXLSX(f *finance.FormatTabel) *xlsx.Tabel {
	if f == nil {
		return nil
	}
	return &xlsx.Tabel{Header: f.Header, Kolom: f.Kolom, Lebar: f.Lebar, Angka: f.Angka, Bilangan: f.Bilangan, Tanggal: f.Tanggal,
		Contoh: f.Contoh, BarisKosong: f.BarisKosong, Pilihan: f.Pilihan, Rumus: f.Rumus, Gaya: f.Gaya, Akhir: f.Akhir,
		Referensi: f.Referensi, Kunci: f.Kunci}
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
	// NilaiBukuAset: nilai buku aset tetap per tanggal mulai (pembanding baris Aset Tetap).
	NilaiBukuAset int64 `json:"nilai_buku_aset"`
	// BisaDisimpan: false bila saldo awal belum seimbang / ada transaksi sebelum tanggal mulai.
	BisaDisimpan bool `json:"bisa_disimpan"`
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
	// Nilai buku aset awal: dari sheet Aset bila diisi (menggantikan register), selain itu
	// register yang sudah ada. Saldo awal per akun wajib seimbang dan aset tetapnya sama
	// dengan nilai buku itu; transaksi sebelum tanggal mulai tidak boleh ikut (dobel hitung).
	var tolak []string
	if d.Opening != nil {
		as := d.Assets
		if len(as) == 0 {
			s.mu.RLock()
			as = slices.Clone(s.assets)
			s.mu.RUnlock()
		}
		nbv := finance.AssetsForPeriod(as, d.Transaksi, d.Opening.Period()).NilaiBukuAwal
		// Berkas tanpa baris aset tetap (nilai buku template rinci hanya ada setelah disimpan
		// Excel): pakai nilai buku hitungan sistem, lalu tetap wajib seimbang.
		d.Opening.LengkapiAsetTetap(nbv)
		resp.ModalAwal = d.Opening.ModalAwal(nbv)
		resp.SelisihModal = d.Opening.SelisihModal(nbv)
		resp.NilaiBukuAset = nbv
		if len(d.Opening.Rincian) > 0 {
			if err := d.Opening.CekSeimbang(nbv); err != nil {
				tolak = append(tolak, err.Error())
			}
		}
		n := 0
		for _, t := range d.Transaksi {
			if t.Tanggal < d.Opening.Tanggal {
				n++
			}
		}
		if n > 0 {
			tolak = append(tolak, strconv.Itoa(n)+" transaksi bertanggal sebelum tanggal mulai "+d.Opening.Tanggal+"; posisinya sudah tercakup di saldo awal. Hapus barisnya atau mundurkan tanggal mulai")
		}
		for _, p := range tolak {
			resp.Errors = append(resp.Errors, finance.SheetError{Sheet: "Saldo Awal", Pesan: p})
		}
	}
	resp.BisaDisimpan = len(tolak) == 0
	if r.URL.Query().Get("commit") != "1" {
		writeJSON(w, resp)
		return
	}
	if len(tolak) > 0 {
		http.Error(w, "data awal belum bisa disimpan: "+strings.Join(tolak, "; "), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if s.tutupBuku != "" {
		s.mu.Unlock()
		http.Error(w, "impor data awal tidak bisa dipakai setelah ada bulan yang ditutup bukunya", http.StatusConflict)
		return
	}
	if d.Opening != nil {
		o := *d.Opening
		if s.opening != nil {
			o.Pelunasan = s.opening.Pelunasan
		}
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
	// TransaksiPertama: tanggal transaksi paling awal; tanggal saldo awal tidak boleh sesudahnya.
	TransaksiPertama string `json:"transaksi_pertama,omitempty"`
	// NilaiBukuAset: nilai buku register aset per tanggal saldo awal (atau ?tanggal=), untuk
	// baris Aset Tetap yang tidak bisa diketik bebas.
	NilaiBukuAset int64 `json:"nilai_buku_aset"`
	// Kategori & AkunTemplate: pilihan kategori dan baris akun bawaan layar Saldo Awal (dari CoA
	// usaha ini). AsetTetap: baris aset tetap per akun dari daftar aset — tidak diketik bebas.
	Kategori     []finance.KategoriSaldo `json:"kategori"`
	AkunTemplate []finance.SaldoAkun     `json:"akun_template"`
	AsetTetap    []finance.SaldoAkun     `json:"aset_tetap"`
	JumlahAset   int                     `json:"jumlah_aset"`
	// Persediaan: nilai yang seharusnya di saldo awal (daftar barang / hitung fisik pertama).
	Persediaan      int64  `json:"persediaan"`
	LabelPersediaan string `json:"label_persediaan"`
}

// openingSummary: tanggal = tanggal pembanding nilai buku aset ("" = tanggal saldo awal).
func (s *tenant) openingSummary(tanggal string) openingResp {
	all, active, _ := s.akunSaldoAwal()
	peta := coa.PetaDari(all)
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := openingResp{TransaksiPertama: s.transaksiPertamaLocked(), Kategori: finance.KategoriSaldoList, JumlahAset: len(s.assets)}
	out.AkunTemplate = coa.TemplateSaldoAwal(active, peta, finance.TemplateAkunSaldo())
	if len(out.AkunTemplate) == 0 {
		out.AkunTemplate = finance.TemplateAkunSaldo()
	}
	out.Persediaan, out.LabelPersediaan = s.pembandingPersediaanLocked()
	period := s.bukuAwalLocked().Period()
	if s.opening != nil {
		o := *s.opening
		nbv := finance.AssetsForPeriod(s.assets, s.txs, o.Period()).NilaiBukuAwal
		out.Opening, out.ModalAwal, out.SelisihModal, out.Aktif, out.NilaiBukuAset = &o, o.ModalAwal(nbv), o.SelisihModal(nbv), true, nbv
	}
	if len(tanggal) >= 7 {
		period = tanggal[:7]
		out.NilaiBukuAset = finance.AssetsForPeriod(s.assets, s.txs, period).NilaiBukuAwal
	}
	out.AsetTetap = coa.NilaiBukuPerAkun(finance.AssetsForPeriod(s.assets, s.txs, period), period, all, peta)
	if len(out.AsetTetap) == 0 {
		// Belum ada aset: satu baris kosong berkode akun peralatan, isinya dari daftar aset.
		kode := peta.Kode(coa.PeranPeralatan)
		nama := "Aset tetap"
		if a, ok := coa.Find(all, kode); ok {
			nama = a.Nama
		}
		out.AsetTetap = []finance.SaldoAkun{{Kode: kode, Nama: nama, Kategori: finance.KatAsetTetap, Keterangan: "isi dari daftar aset"}}
	}
	return out
}

// transaksiPertamaLocked: tanggal transaksi paling awal ("" bila belum ada). s.mu dipegang.
func (s *tenant) transaksiPertamaLocked() string {
	pertama := ""
	for _, t := range s.txs {
		if pertama == "" || t.Tanggal < pertama {
			pertama = t.Tanggal
		}
	}
	return pertama
}

func (s *tenant) handleGetOpening(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.openingSummary(r.URL.Query().Get("tanggal")))
}

func (s *tenant) handlePutOpening(w http.ResponseWriter, r *http.Request) {
	var o finance.OpeningBalance
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		http.Error(w, "data neraca awal tidak valid", http.StatusBadRequest)
		return
	}
	if err := o.TerapkanRincian(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := finance.ValidateOpeningBalance(&o); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	// Saldo awal per akun wajib seimbang: Harta (termasuk nilai buku register aset) =
	// Kewajiban + Modal. Selisih yang lolos akan diam-diam menjadi modal di semua laporan.
	if len(o.Rincian) > 0 {
		if err := o.CekSeimbang(finance.AssetsForPeriod(s.assets, s.txs, o.Period()).NilaiBukuAwal); err != nil {
			s.mu.Unlock()
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	if s.tutupBuku != "" {
		s.mu.Unlock()
		http.Error(w, "saldo awal tidak bisa diubah setelah ada bulan yang ditutup bukunya", http.StatusConflict)
		return
	}
	// Saldo awal = posisi di pagi hari tanggal mulai. Transaksi sebelum tanggal itu sudah
	// tercermin di saldo awal, jadi kalau tetap dihitung, uangnya terhitung dua kali.
	if p := s.transaksiPertamaLocked(); p != "" && p < o.Tanggal {
		s.mu.Unlock()
		http.Error(w, "sudah ada transaksi tanggal "+p+", sebelum tanggal mulai "+o.Tanggal+". Pilih tanggal mulai paling lambat "+p+" (posisi saldo di pagi hari itu), atau hapus transaksi sebelum tanggal mulai", http.StatusConflict)
		return
	}
	if s.opening != nil {
		o.Pelunasan = s.opening.Pelunasan // pelunasan saldo awal yang sudah dicatat tetap ada
	}
	s.opening = &o
	s.mu.Unlock()
	writeJSON(w, s.openingSummary(""))
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
