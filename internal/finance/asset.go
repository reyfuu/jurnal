package finance

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Penyusutan fiskal PMK 72/2023. Yang membedakannya dari penyusutan komersial (PSAK):
// tidak ada nilai residu (aset disusutkan sampai nol), masa manfaat & tarif wajib ikut
// kelompok, dan bangunan hanya boleh garis lurus.
//
//   - Kelompok & tarif harta berwujud: Pasal 2 ayat (3).
//   - Amortisasi harta tak berwujud memakai kelompok & tarif yang sama: Pasal 9 ayat (3).
//   - Penyusutan dimulai bulan dilakukannya pengeluaran: Pasal 5 ayat (1).
//   - Saldo menurun, akhir masa manfaat: sisa nilai buku disusutkan sekaligus: Pasal 2 ayat (2).
//
// ponytail: angka komersial (masa manfaat sendiri + nilai residu) dan koreksi fiskalnya
// belum disimpan; tambahkan kolom komersial bila laporan PSAK dibutuhkan.

const (
	MetodeGarisLurus   = "garis_lurus"
	MetodeSaldoMenurun = "saldo_menurun"
)

// JenisBeliAset: jenis transaksi yang memunculkan draft aset di register.
const JenisBeliAset = "Beli Alat / Aset"

// KelompokAset: masa manfaat dalam bulan, tarif tahunan dalam basis point (2500 = 25%).
// TarifSMBps 0 berarti saldo menurun tidak boleh dipakai (bangunan).
type KelompokAset struct {
	ID          string `json:"id"`
	Nama        string `json:"nama"`
	MasaManfaat int    `json:"masa_manfaat"` // bulan
	TarifGLBps  int    `json:"tarif_gl_bps"`
	TarifSMBps  int    `json:"tarif_sm_bps"`
	Bangunan    bool   `json:"bangunan"`
}

// KelompokAsetList adalah tabel Pasal 2 ayat (3) PMK 72/2023; ditetapkan peraturan, bukan isian user.
var KelompokAsetList = []KelompokAset{
	{"k1", "Bukan bangunan — Kelompok 1", 48, 2500, 5000, false},
	{"k2", "Bukan bangunan — Kelompok 2", 96, 1250, 2500, false},
	{"k3", "Bukan bangunan — Kelompok 3", 192, 625, 1250, false},
	{"k4", "Bukan bangunan — Kelompok 4", 240, 500, 1000, false},
	{"bangunan_permanen", "Bangunan Permanen", 240, 500, 0, true},
	{"bangunan_tidak_permanen", "Bangunan Tidak Permanen", 120, 1000, 0, true},
}

// JenisHarta memetakan nama harta ke kelompoknya supaya user tidak perlu menebak kelompok.
type JenisHarta struct {
	ID          string `json:"id"`
	Nama        string `json:"nama"`
	Kelompok    string `json:"kelompok"`
	TakBerwujud bool   `json:"tak_berwujud"`
	// Lampiran: rujukan tempat memeriksa baris ini di Lampiran PMK 72/2023.
	Lampiran string `json:"lampiran"`
	// Terverifikasi: true hanya setelah baris ini dicocokkan ke Lampiran resmi.
	// Baris yang masih false ditandai di UI dan di template impor.
	Terverifikasi bool `json:"terverifikasi"`
}

// JenisHartaList: subset Lampiran PMK 72/2023 yang relevan untuk UMKM dagang, jasa, dan F&B.
//
// === SATU-SATUNYA TEMPAT MENGOREKSI PEMETAAN KELOMPOK ===
// Cara memperbaiki satu baris:
//  1. Buka Lampiran PMK 72/2023 pada bagian yang disebut kolom Lampiran di bawah.
//  2. Ganti Kelompok ke "k1".."k4" / "bangunan_permanen" / "bangunan_tidak_permanen".
//  3. Set Terverifikasi: true setelah cocok. Jangan ubah ID — ID dipakai aset yang sudah
//     tersimpan, mengubahnya memutus tautan aset lama ke kelompoknya.
//
// Menambah baris baru: cukup tambahkan di sini; template impor, dropdown form, dan sheet
// Panduan ikut terisi otomatis karena semuanya membaca daftar ini.
//
// ponytail: seluruh baris masih Terverifikasi: false — belum dicocokkan baris per baris ke
// Lampiran resmi. UI menandainya dan tetap menyediakan opsi "Lainnya" (kelompok dipilih
// manual) untuk harta yang tidak ada di daftar ini. Jangan dipakai untuk SPT sebelum dicek.
var JenisHartaList = []JenisHarta{
	{"mesin_makanan", "Mesin pengolah makanan & minuman", "k1", false, "Industri makanan & minuman", false},
	{"peralatan_restoran", "Peralatan usaha restoran / rumah makan", "k1", false, "Jenis Usaha Lain-lain", false},
	{"mebel_kayu", "Mebel & peralatan dari kayu atau rotan", "k1", false, "Semua Jenis Usaha", false},
	{"komputer", "Komputer, printer, scanner", "k1", false, "Semua Jenis Usaha", false},
	{"sepeda_motor", "Sepeda motor", "k1", false, "Semua Jenis Usaha", false},
	{"mebel_logam", "Mebel & peralatan dari logam", "k2", false, "Semua Jenis Usaha", false},
	{"kendaraan", "Mobil / kendaraan angkutan", "k2", false, "Semua Jenis Usaha", false},
	{"mesin_produksi", "Mesin & instalasi produksi berat", "k3", false, "Industri pengolahan", false},
	{"bangunan_tidak_permanen", "Bangunan tidak permanen (renovasi ruko sewa)", "bangunan_tidak_permanen", false, "Pasal 2 ayat (3) — bangunan", true},
	{"bangunan_permanen", "Bangunan permanen", "bangunan_permanen", false, "Pasal 2 ayat (3) — bangunan", true},
	{"software", "Harta tak berwujud — software & lisensi", "k1", true, "Pasal 9 ayat (3) — harta tak berwujud", false},
	{"hak_usaha", "Harta tak berwujud — hak usaha, merek, goodwill", "k4", true, "Pasal 9 ayat (3) — harta tak berwujud", false},
}

// JenisHartaBelumTerverifikasi menghitung baris pemetaan yang masih perlu dicek tim pajak.
// Dipakai UI untuk menampilkan peringatan, bukan untuk memblokir pemakaian.
func JenisHartaBelumTerverifikasi() int {
	n := 0
	for _, j := range JenisHartaList {
		if !j.Terverifikasi {
			n++
		}
	}
	return n
}

// Asset adalah satu baris register aset. Kelompok kosong = draft, belum menghasilkan beban.
type Asset struct {
	ID               int    `json:"id"`
	Nama             string `json:"nama"`
	JenisHarta       string `json:"jenis_harta"`       // id JenisHartaList; kosong = kelompok dipilih manual
	Kelompok         string `json:"kelompok"`          // id KelompokAsetList
	Metode           string `json:"metode"`            // garis_lurus | saldo_menurun
	TanggalPerolehan string `json:"tanggal_perolehan"` // YYYY-MM-DD
	HargaPerolehan   int64  `json:"harga_perolehan"`
	TakBerwujud      bool   `json:"tak_berwujud"` // true = amortisasi, bukan penyusutan
	TransaksiID      int    `json:"transaksi_id,omitempty"`
	Keterangan       string `json:"keterangan,omitempty"`
}

func KelompokOf(id string) (KelompokAset, bool) {
	i := slices.IndexFunc(KelompokAsetList, func(k KelompokAset) bool { return k.ID == id })
	if i < 0 {
		return KelompokAset{}, false
	}
	return KelompokAsetList[i], true
}

func jenisHartaOf(id string) (JenisHarta, bool) {
	i := slices.IndexFunc(JenisHartaList, func(j JenisHarta) bool { return j.ID == id })
	if i < 0 {
		return JenisHarta{}, false
	}
	return JenisHartaList[i], true
}

// ValidateAsset merapikan isian dan mengisi kelompok dari jenis harta bila dipilih.
func ValidateAsset(a *Asset) error {
	for _, s := range []*string{&a.Nama, &a.JenisHarta, &a.Kelompok, &a.Metode, &a.TanggalPerolehan, &a.Keterangan} {
		*s = strings.TrimSpace(*s)
	}
	if a.Nama == "" {
		return errors.New("nama aset wajib diisi")
	}
	if _, err := time.Parse(time.DateOnly, a.TanggalPerolehan); err != nil {
		return errors.New("tanggal perolehan wajib diisi (format YYYY-MM-DD)")
	}
	if a.HargaPerolehan <= 0 {
		return errors.New("harga perolehan harus lebih dari 0")
	}
	if a.JenisHarta != "" {
		j, ok := jenisHartaOf(a.JenisHarta)
		if !ok {
			return errors.New("jenis harta tidak dikenal")
		}
		a.Kelompok, a.TakBerwujud = j.Kelompok, j.TakBerwujud
	}
	k, ok := KelompokOf(a.Kelompok)
	if !ok {
		return errors.New("pilih kelompok aset sesuai PMK (Kelompok 1-4 atau bangunan)")
	}
	if a.Metode == "" {
		a.Metode = MetodeGarisLurus
	}
	if a.Metode != MetodeGarisLurus && a.Metode != MetodeSaldoMenurun {
		return errors.New("metode penyusutan harus garis_lurus atau saldo_menurun")
	}
	if a.Metode == MetodeSaldoMenurun && k.TarifSMBps == 0 {
		return errors.New("bangunan hanya boleh disusutkan dengan metode garis lurus")
	}
	if a.TakBerwujud && k.Bangunan {
		return errors.New("harta tak berwujud tidak bisa memakai kelompok bangunan")
	}
	return nil
}

// parseYM membaca "YYYY-MM" atau "YYYY-MM-DD".
func parseYM(s string) (year, month int, ok bool) {
	if len(s) < 7 || s[4] != '-' {
		return 0, 0, false
	}
	y, err1 := strconv.Atoi(s[:4])
	m, err2 := strconv.Atoi(s[5:7])
	if err1 != nil || err2 != nil || m < 1 || m > 12 {
		return 0, 0, false
	}
	return y, m, true
}

// monthsElapsed: jumlah bulan penyusutan dari bulan perolehan sampai akhir period, inklusif.
// 0 bila period jatuh sebelum bulan perolehan.
func monthsElapsed(tanggalPerolehan, period string) int {
	ay, am, ok1 := parseYM(tanggalPerolehan)
	by, bm, ok2 := parseYM(period)
	if !ok1 || !ok2 {
		return 0
	}
	return max(0, (by-ay)*12+(bm-am)+1)
}

func prevPeriod(period string) string {
	y, m, ok := parseYM(period)
	if !ok {
		return period
	}
	if m == 1 {
		y, m = y-1, 12
	} else {
		m--
	}
	return fmt.Sprintf("%04d-%02d", y, m)
}

// spread membagi amount rata ke n bulan; sisa pembulatan jatuh di bulan terakhir
// supaya jumlah seluruh jadwal tetap sama dengan harga perolehan.
func spread(out []int64, amount int64, n int) []int64 {
	per := amount / int64(n)
	for range n - 1 {
		out = append(out, per)
	}
	return append(out, amount-per*int64(n-1))
}

// AssetSchedule mengembalikan beban penyusutan per bulan sejak bulan perolehan,
// sepanjang masa manfaat kelompoknya. Aset draft (kelompok kosong) tidak punya jadwal.
func AssetSchedule(a Asset) []int64 {
	k, ok := KelompokOf(a.Kelompok)
	if !ok || a.HargaPerolehan <= 0 {
		return nil
	}
	n := k.MasaManfaat
	if a.Metode != MetodeSaldoMenurun || k.TarifSMBps == 0 {
		return spread(make([]int64, 0, n), a.HargaPerolehan, n)
	}
	// Saldo menurun: tarif tahunan atas sisa nilai buku. Dihitung per tahun kalender karena
	// penyusutan dimulai bulan pengeluaran, jadi tahun pertama sering tidak genap 12 bulan.
	_, bulanAwal, ok := parseYM(a.TanggalPerolehan)
	if !ok {
		return nil
	}
	out := make([]int64, 0, n)
	sisa, done, bulan := a.HargaPerolehan, 0, 13-bulanAwal
	for done < n {
		bulan = min(bulan, n-done)
		amount := sisa * int64(k.TarifSMBps) * int64(bulan) / (10_000 * 12)
		if done+bulan >= n {
			amount = sisa // akhir masa manfaat: sisa nilai buku disusutkan sekaligus
		}
		out = spread(out, amount, bulan)
		sisa -= amount
		done += bulan
		bulan = 12
	}
	return out
}

func sumSchedule(sched []int64, months int) int64 {
	var total int64
	for _, v := range sched[:min(max(months, 0), len(sched))] {
		total += v
	}
	return total
}

// AssetYear adalah satu baris jadwal penyusutan per tahun kalender.
type AssetYear struct {
	Tahun      int   `json:"tahun"`
	Bulan      int   `json:"bulan"` // bulan yang disusutkan di tahun itu
	DasarAwal  int64 `json:"dasar_awal"`
	Penyusutan int64 `json:"penyusutan"`
	SisaAkhir  int64 `json:"sisa_akhir"`
}

func jadwalTahunan(a Asset, sched []int64) []AssetYear {
	tahun, bulanAwal, ok := parseYM(a.TanggalPerolehan)
	if !ok || len(sched) == 0 {
		return nil
	}
	var out []AssetYear
	sisa, i, bulan := a.HargaPerolehan, 0, 13-bulanAwal
	for i < len(sched) {
		bulan = min(bulan, len(sched)-i)
		var amount int64
		for _, v := range sched[i : i+bulan] {
			amount += v
		}
		out = append(out, AssetYear{Tahun: tahun, Bulan: bulan, DasarAwal: sisa, Penyusutan: amount, SisaAkhir: sisa - amount})
		sisa -= amount
		i += bulan
		tahun++
		bulan = 12
	}
	return out
}

// AssetReport adalah satu aset beserta hitungan penyusutannya pada satu periode.
type AssetReport struct {
	Asset
	KelompokNama  string      `json:"kelompok_nama"`
	MasaManfaat   int         `json:"masa_manfaat"` // bulan
	TarifBps      int         `json:"tarif_bps"`    // tarif tahunan sesuai metode yang dipakai
	BebanBulanIni int64       `json:"beban_bulan_ini"`
	Akumulasi     int64       `json:"akumulasi"`
	NilaiBuku     int64       `json:"nilai_buku"`
	NilaiBukuAwal int64       `json:"nilai_buku_awal"` // awal periode; 0 bila baru diperoleh periode ini
	Draft         bool        `json:"draft"`
	Jadwal        []AssetYear `json:"jadwal,omitempty"`
}

// AssetSummary adalah register aset satu periode plus total yang dipakai laporan lain.
type AssetSummary struct {
	Assets              []AssetReport  `json:"assets"`
	Kelompok            []KelompokAset `json:"kelompok"`
	JenisHarta          []JenisHarta   `json:"jenis_harta"`
	TotalHargaPerolehan int64          `json:"total_harga_perolehan"`
	TotalAkumulasi      int64          `json:"total_akumulasi"`
	TotalNilaiBuku      int64          `json:"total_nilai_buku"`
	NilaiBukuAwal       int64          `json:"nilai_buku_awal"`
	BebanPenyusutan     int64          `json:"beban_penyusutan"` // aset berwujud, periode ini
	BebanAmortisasi     int64          `json:"beban_amortisasi"` // harta tak berwujud, periode ini
	BebanTotal          int64          `json:"beban_total"`
	// TambahanTanpaKas: aset yang masuk periode ini tanpa transaksi kas; dibukukan sebagai
	// setoran modal berupa aset agar neraca tetap seimbang.
	TambahanTanpaKas int64 `json:"tambahan_tanpa_kas"`
	DraftCount       int   `json:"draft_count"`
	// PemetaanBelumDicek: jumlah baris JenisHartaList yang belum diverifikasi ke Lampiran PMK.
	PemetaanBelumDicek int `json:"pemetaan_belum_dicek"`
}

// draftsFromTx memunculkan aset draft dari transaksi "Beli Alat / Aset" yang belum terdaftar,
// supaya pembelian aset tidak hilang dari neraca sebelum user melengkapi kelompok & metodenya.
func draftsFromTx(txs []Transaction, assets []Asset) []Asset {
	registered := make(map[int]bool, len(assets))
	for _, a := range assets {
		if a.TransaksiID != 0 {
			registered[a.TransaksiID] = true
		}
	}
	var out []Asset
	for _, t := range txs {
		if t.Jenis != JenisBeliAset || registered[t.ID] {
			continue
		}
		out = append(out, Asset{
			Nama:             cmp.Or(t.Keterangan, t.Item, JenisBeliAset),
			TanggalPerolehan: t.Tanggal,
			HargaPerolehan:   t.Nominal,
			TransaksiID:      t.ID,
		})
	}
	return out
}

// AssetsForPeriod menghitung penyusutan & amortisasi seluruh aset sampai akhir period (YYYY-MM).
// txs adalah seluruh transaksi (bukan hanya periode itu), dipakai untuk memunculkan draft.
func AssetsForPeriod(assets []Asset, txs []Transaction, period string) AssetSummary {
	all := append(slices.Clone(assets), draftsFromTx(txs, assets)...)
	prev := prevPeriod(period)
	sum := AssetSummary{Kelompok: KelompokAsetList, JenisHarta: JenisHartaList, PemetaanBelumDicek: JenisHartaBelumTerverifikasi()}
	for _, a := range all {
		bulanAkhir := monthsElapsed(a.TanggalPerolehan, period)
		if bulanAkhir <= 0 {
			continue // belum diperoleh sampai akhir periode ini
		}
		bulanAwal := monthsElapsed(a.TanggalPerolehan, prev)
		sched := AssetSchedule(a)
		akumAwal := sumSchedule(sched, bulanAwal)

		r := AssetReport{Asset: a, Draft: a.Kelompok == ""}
		r.Akumulasi = sumSchedule(sched, bulanAkhir)
		r.BebanBulanIni = r.Akumulasi - akumAwal
		r.NilaiBuku = a.HargaPerolehan - r.Akumulasi
		if bulanAwal > 0 {
			r.NilaiBukuAwal = a.HargaPerolehan - akumAwal
		} else if a.TransaksiID == 0 && !r.Draft {
			// Masuk periode ini tanpa transaksi kas: dibukukan sebagai setoran modal berupa aset.
			// Draft selalu berasal dari transaksi, jadi tidak pernah masuk hitungan ini.
			sum.TambahanTanpaKas += a.HargaPerolehan
		}
		if k, ok := KelompokOf(a.Kelompok); ok {
			r.KelompokNama, r.MasaManfaat, r.TarifBps = k.Nama, k.MasaManfaat, k.TarifGLBps
			if a.Metode == MetodeSaldoMenurun && k.TarifSMBps > 0 {
				r.TarifBps = k.TarifSMBps
			}
			r.Jadwal = jadwalTahunan(a, sched)
		}

		switch {
		case r.Draft:
			sum.DraftCount++
		case a.TakBerwujud:
			sum.BebanAmortisasi += r.BebanBulanIni
		default:
			sum.BebanPenyusutan += r.BebanBulanIni
		}
		sum.TotalHargaPerolehan += a.HargaPerolehan
		sum.TotalAkumulasi += r.Akumulasi
		sum.TotalNilaiBuku += r.NilaiBuku
		sum.NilaiBukuAwal += r.NilaiBukuAwal
		sum.Assets = append(sum.Assets, r)
	}
	sum.BebanTotal = sum.BebanPenyusutan + sum.BebanAmortisasi
	slices.SortStableFunc(sum.Assets, func(x, y AssetReport) int {
		return cmp.Or(strings.Compare(x.TanggalPerolehan, y.TanggalPerolehan), x.ID-y.ID)
	})
	return sum
}
