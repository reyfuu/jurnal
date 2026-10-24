package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

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
	// Kode akun Neraca Saldo Awal (template memakai kode CoA dagang) diganti kode CoA usaha: lewat
	// peran akun, selain itu nama akun yang sama persis; tidak ada akun yang cocok = kosong (lebih
	// baik daripada kode yang salah, mis. 1102 di CoA manufaktur adalah Kas Besar).
	all, active, _ := s.akunSaldoAwal()
	peta := coa.PetaDari(all)
	kode := func(bawaan, nama string) string {
		for _, pi := range coa.DaftarPeran {
			if pi.Bawaan == bawaan {
				return peta.Kode(pi.ID)
			}
		}
		if a, persis, ok := coa.CocokAkunSaldo(nama, active, peta); ok && persis {
			return a.Kode
		}
		return ""
	}
	sheets := make([]xlsx.Sheet, 0, 12)
	for _, sh := range finance.OnboardingTemplate(metode) {
		if sh.Nama == "Neraca Saldo Awal" {
			for _, row := range sh.Rows {
				if len(row) > 1 && len(row[0]) == 4 && row[0][0] >= '1' && row[0][0] <= '9' {
					row[0] = kode(row[0], row[1])
				}
			}
		}
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
		Contoh: f.Contoh, BarisKosong: f.BarisKosong, Pilihan: f.Pilihan, Bebas: f.Bebas, Status: f.Status, Rumus: f.Rumus, Gaya: f.Gaya,
		Akhir: f.Akhir, Referensi: f.Referensi, Kunci: f.Kunci}
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
	// Transaksi data awal tunduk pada aturan impor transaksi Excel: barang dicocokkan, wajib
	// barang per baris (baris gagal dilaporkan & dilewati), stok tidak minus untuk seluruh berkas
	// (menolak simpan). Dihitung ulang di bawah kunci saat commit.
	stokBaru := stokDataAwal(d.Stok)
	s.mu.Lock()
	txsAwal, errBarang, _, errStok := s.barangDataAwalLocked(d.BarisTransaksi, stokBaru)
	s.mu.Unlock()
	d.Transaksi = txsAwal
	d.Errors = append(d.Errors, errBarang...)
	if errStok != nil {
		d.Errors = append(d.Errors, finance.SheetError{Sheet: sheetTransaksiAwal(d.BarisTransaksi), Pesan: errStok.Error()})
	}
	var akun []coa.Account
	if d.Opening != nil {
		s.kodePersediaanImpor(d.Opening.Rincian)
		akun = s.isiKodeSaldoAwal(d.Opening)
	}
	s.mu.RLock()
	finance.TerapkanMetode(d.Assets, s.metodeUsaha()) // kolom Metode di Excel kalah oleh metode usaha
	awal, simpanan := s.bukuAwalLocked().Period(), s.opening
	s.mu.RUnlock()
	if d.Opening != nil {
		awal = d.Opening.Period()
	} else if simpanan == nil {
		// Tanggal mulai turunan transaksi ikut mundur bila transaksi impor lebih tua.
		for _, t := range d.Transaksi {
			if len(t.Tanggal) >= 7 && t.Tanggal[:7] < awal {
				awal = t.Tanggal[:7]
			}
		}
	}
	// Aset impor = aset per saldo awal. Yang diperoleh pada/sesudah bulan mulai tidak punya
	// transaksi kas, jadi laporan membacanya sebagai "Tambahan modal": daftarkan lewat menu Aset.
	n := 0
	for _, a := range d.Assets {
		if a.TanggalPerolehan[:7] >= awal {
			d.Errors = append(d.Errors, finance.SheetError{Sheet: "Aset", Pesan: a.Nama + " diperoleh " + a.TanggalPerolehan +
				", tidak sebelum bulan mulai pembukuan (" + awal + "). Daftarkan lewat menu Aset dan pilih sumber dananya (kas, utang, atau modal) supaya tidak terbaca sebagai tambahan modal"})
			continue
		}
		d.Assets[n] = a
		n++
	}
	d.Assets = d.Assets[:n]
	// Daftar aset baru tanpa saldo awal di berkas: saldo awal tersimpan bisa jadi tidak cocok
	// (sama seperti menambah aset lama lewat menu Aset, cukup diperingatkan).
	if d.Opening == nil && simpanan != nil && len(simpanan.Rincian) > 0 && len(d.Assets) > 0 {
		s.mu.RLock()
		nbv := finance.AssetsForPeriod(d.Assets, s.txs, simpanan.Period()).NilaiBukuAwal
		s.mu.RUnlock()
		if nbv != simpanan.AsetTetap {
			d.Errors = append(d.Errors, finance.SheetError{Sheet: "Aset", Pesan: "nilai buku daftar aset ini per saldo awal " + finance.FormatRupiahPenuh(nbv) +
				", sedangkan Saldo Awal mencatat aset tetap " + finance.FormatRupiahPenuh(simpanan.AsetTetap) + ". Sesuaikan menu Saldo Awal sesudah impor"})
		}
	}

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
	if errStok != nil {
		tolak = append(tolak, errStok.Error())
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
		if err := s.akunHilangDipakaiLocked(d.Opening.Rincian, akun); err != nil {
			s.mu.Unlock()
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
	}
	// Ulangi cek barang di bawah kunci commit (daftar barang / kartu bisa berubah sejak
	// pratinjau di atas), sebelum apa pun diubah.
	txsAwal, _, awalStok, errStok := s.barangDataAwalLocked(d.BarisTransaksi, stokBaru)
	if errStok != nil {
		s.mu.Unlock()
		http.Error(w, errStok.Error(), http.StatusConflict)
		return
	}
	if awalStok != nil { // transaksi berbarang pertama memulai kartu stok
		if err := s.simpanStokAwalLocked(awalStok, currentUser(r).Name, time.Now().UTC()); err != nil {
			s.mu.Unlock()
			http.Error(w, "gagal menyimpan metode nilai stok", http.StatusInternalServerError)
			return
		}
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
		s.stok = stokBaru // ID sama dengan yang dipakai mencocokkan barang transaksi di atas
		s.nextStokID = len(stokBaru) + 1
	}
	for _, t := range txsAwal {
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
	// PersediaanAkun: pembanding persediaan per akun persediaan (urut kode); kosong bila belum
	// ada barang bernilai. Form Saldo Awal mengisi baris persediaan dari sini.
	PersediaanAkun []finance.SaldoAkun `json:"persediaan_akun"`
	// Terkunci: alasan saldo awal tidak bisa diubah lagi (ada bulan yang sudah tutup buku).
	Terkunci string `json:"terkunci,omitempty"`
}

// kunciSaldoAwalLocked: alasan saldo awal terkunci, "" bila masih bisa diubah. s.mu dipegang.
func (s *tenant) kunciSaldoAwalLocked() string {
	if s.tutupBuku == "" {
		return ""
	}
	return "Saldo awal terkunci karena buku sampai " + finance.LabelPeriode(s.tutupBuku) + " sudah ditutup. Buka kembali di menu Tutup Buku bila memang perlu diubah."
}

// openingSummary: tanggal = tanggal pembanding nilai buku aset ("" = tanggal saldo awal).
func (s *tenant) openingSummary(tanggal string) openingResp {
	all, active, _ := s.akunSaldoAwal()
	peta := coa.PetaDari(all)
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := openingResp{TransaksiPertama: s.transaksiPertamaLocked(), Kategori: finance.KategoriSaldoList, JumlahAset: len(s.assets), Terkunci: s.kunciSaldoAwalLocked()}
	out.AkunTemplate = coa.TemplateSaldoAwal(active, peta, finance.TemplateAkunSaldo())
	if len(out.AkunTemplate) == 0 {
		out.AkunTemplate = finance.TemplateAkunSaldo()
	}
	out.Persediaan, out.LabelPersediaan = s.pembandingPersediaanLocked()
	out.PersediaanAkun = barisPersediaanAkun(s.pembandingPersediaanPerAkunLocked(all, peta), all, out.LabelPersediaan)
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
	// ?paksa=1: pengguna sudah memeriksa dan tetap menyimpan saldo awal yang belum seimbang;
	// selisihnya masuk akun Laba Ditahan - Selisih Saldo Awal (dihitung ulang tiap simpan).
	paksa := r.URL.Query().Get("paksa") == "1"
	o.Rincian = slices.DeleteFunc(o.Rincian, barisSelisihSaldoAwal)
	if err := o.TerapkanRincian(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := finance.ValidateOpeningBalance(&o); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	akun := s.isiKodeSaldoAwal(&o)
	s.mu.Lock()
	// Saldo awal per akun wajib seimbang: Harta (termasuk nilai buku register aset) =
	// Kewajiban + Modal. Selisih yang lolos akan diam-diam menjadi modal di semua laporan.
	if nbv := finance.AssetsForPeriod(s.assets, s.txs, o.Period()).NilaiBukuAwal; paksa && len(o.Rincian) > 0 && o.AsetTetap == nbv {
		if d := o.Harta(nbv) - o.Kewajiban() - o.Modal(); d != 0 {
			a := coa.AkunSelisihSaldoAwal(akun, coa.PetaDari(akun))
			o.Rincian = append(o.Rincian, finance.SaldoAkun{Kode: a.Kode, Nama: a.Nama, Kategori: finance.KatLabaDitahan, Nominal: d, Keterangan: ketSelisihSaldoAwal})
			if err := o.TerapkanRincian(); err != nil {
				s.mu.Unlock()
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
	}
	if len(o.Rincian) > 0 {
		if err := o.CekSeimbang(finance.AssetsForPeriod(s.assets, s.txs, o.Period()).NilaiBukuAwal); err != nil {
			s.mu.Unlock()
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	if kunci := s.kunciSaldoAwalLocked(); kunci != "" {
		s.mu.Unlock()
		http.Error(w, kunci, http.StatusConflict)
		return
	}
	if err := s.akunHilangDipakaiLocked(o.Rincian, akun); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusConflict)
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

// isiKodeSaldoAwal memberi kode akun pada baris saldo awal tanpa kode (dari CoA usaha ini) dan
// mengembalikan daftar akun saat ini. Jangan dipanggil dengan s.mu terkunci.
func (s *tenant) isiKodeSaldoAwal(o *finance.OpeningBalance) []coa.Account {
	all, active, _ := s.akunSaldoAwal()
	coa.IsiKodeSaldo(o.Rincian, active, coa.PetaDari(all))
	return all
}

// kodePersediaanImpor: baris persediaan impor ("Persediaan — <kategori>") masuk akun persediaan
// kategorinya (bahan baku, barang jadi, …), sama dengan tebakan akun barangnya di Stok & Gudang.
// Jangan dipanggil dengan s.mu terkunci.
func (s *tenant) kodePersediaanImpor(rows []finance.SaldoAkun) {
	all, _, _ := s.akunSaldoAwal()
	peta := coa.PetaDari(all)
	for i, r := range rows {
		if kat, ok := strings.CutPrefix(r.Nama, "Persediaan — "); ok && r.Kategori == finance.KatPersediaan {
			rows[i].Kode = coa.AkunPersediaanBarang(kat, all, peta)
		}
	}
}

// akunHilangDipakaiLocked: akun tambahan Saldo Awal yang hilang dari saldo awal baru tetapi
// masih dipakai transaksi (koreksi akun). Akun itu ikut hilang dari daftar akun, jadi
// transaksinya tidak punya akun lagi di neraca saldo.
func (s *tenant) akunHilangDipakaiLocked(baru []finance.SaldoAkun, akun []coa.Account) error {
	if s.opening == nil {
		return nil
	}
	for _, r := range s.opening.Rincian {
		a, ok := coa.Find(akun, r.Kode)
		if !ok || a.Catatan != coa.CatatanSaldoAwal || slices.ContainsFunc(baru, func(b finance.SaldoAkun) bool { return b.Kode == r.Kode }) {
			continue
		}
		n := 0
		for _, t := range s.txs {
			if t.Akun == r.Kode {
				n++
			}
		}
		if n > 0 {
			return fmt.Errorf("akun %s %s dipakai %d transaksi; pindahkan transaksinya ke akun lain dulu, baru hapus barisnya", r.Kode, a.Nama, n)
		}
	}
	return nil
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

// ketSelisihSaldoAwal menandai baris penyeimbang saldo awal yang disimpan paksa.
const ketSelisihSaldoAwal = "selisih saldo awal yang disimpan walau belum seimbang"

func barisSelisihSaldoAwal(r finance.SaldoAkun) bool {
	return r.Kategori == finance.KatLabaDitahan && r.Keterangan == ketSelisihSaldoAwal
}

// stokDataAwal: daftar barang dari berkas data awal dengan ID yang akan dipakai saat commit
// (1, 2, …; daftar lama diganti). nil bila berkas tidak berisi daftar barang.
func stokDataAwal(stok []finance.StokItem) []finance.StokItem {
	if len(stok) == 0 {
		return nil
	}
	out := slices.Clone(stok)
	for i := range out {
		out[i].ID = i + 1
	}
	return out
}

// sheetTransaksiAwal: nama sheet transaksi untuk galat seluruh berkas (stok minus).
func sheetTransaksiAwal(baris []finance.BarisAwal) string {
	if len(baris) > 0 {
		return baris[0].Sheet
	}
	return "Transaksi"
}

// barangDataAwalLocked: baris transaksi impor data awal lewat aturan impor transaksi Excel —
// barang dicocokkan & wajib barang per baris (terapkanBarangImporLocked; perpetual ketat,
// periodik longgar; baris sebelum kartu stok dimulai dikecualikan), lalu stok tidak minus untuk
// seluruh kumpulan (cekStokImporLocked, mengembalikan stok awal otomatis bila kartu belum
// dimulai). stokBaru: daftar barang dari berkas yang sama, yang menggantikan daftar sekarang saat
// commit; nil = daftar sekarang. s.mu dipegang (tulis): s.stok dipinjamkan ke stokBaru lalu
// dikembalikan sebelum fungsi kembali.
func (s *tenant) barangDataAwalLocked(baris []finance.BarisAwal, stokBaru []finance.StokItem) ([]finance.Transaction, []finance.SheetError, *finance.GerakStok, error) {
	if stokBaru != nil {
		lama := s.stok
		s.stok = stokBaru
		defer func() { s.stok = lama }()
	}
	var txs []finance.Transaction
	var errs []finance.SheetError
	for i := 0; i < len(baris); { // per sheet: nomor baris galat milik sheet-nya
		j := i
		var grup []finance.ImportRow
		for ; j < len(baris) && baris[j].Sheet == baris[i].Sheet; j++ {
			grup = append(grup, baris[j].ImportRow)
		}
		ok, galat := s.terapkanBarangImporLocked(grup)
		txs = append(txs, ok...)
		for _, e := range galat {
			errs = append(errs, finance.SheetError{Sheet: baris[i].Sheet, Baris: e.Baris, Pesan: e.Pesan})
		}
		i = j
	}
	awal, err := s.cekStokImporLocked(txs)
	return txs, errs, awal, err
}
