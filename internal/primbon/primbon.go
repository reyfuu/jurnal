// Package primbon memuat katalog peraturan perpajakan resmi (index.csv hasil skrip
// scripts/peraturan) beserta naskahnya di disk, dan menyediakan pencarian + filter untuk
// menu Primbon Pajak. Hanya metadata faktual dan naskah resmi pemerintah; tidak ada
// rangkuman atau konten berhak cipta pihak lain.
package primbon

import (
	"cmp"
	"encoding/csv"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

// PerHalaman: jumlah peraturan per halaman katalog.
const PerHalaman = 20

type Peraturan struct {
	UUID       string   `json:"uuid"`
	Kode       string   `json:"kode"`
	Judul      string   `json:"judul"`
	Jenis      string   `json:"jenis"` // UU | PP | PERPRES | PMK | KMK | PERDIRJEN | SE_DJP | SEMA | LAIN
	JenisLabel string   `json:"jenis_label"`
	Kategori   []string `json:"kategori"`
	// KategoriLabel: label tiap kategori, urutan sama dengan Kategori.
	KategoriLabel []string `json:"kategori_label"`
	Status        string   `json:"status,omitempty"`
	Tahun         int      `json:"tahun,omitempty"`
	Ditetapkan    string   `json:"ditetapkan,omitempty"` // YYYY-MM-DD
	Diundangkan   string   `json:"diundangkan,omitempty"`
	Berlaku       string   `json:"berlaku,omitempty"`
	SumberLN      string   `json:"sumber_ln,omitempty"`
	Halaman       int      `json:"halaman,omitempty"`
	// Format naskah di server: pdf | htm; kosong = belum tersedia (tim masih mengunduh).
	Format   string `json:"format,omitempty"`
	Lampiran bool   `json:"lampiran,omitempty"` // naskah HTML punya lampiran PDF terpisah
	Abstrak  bool   `json:"abstrak,omitempty"`  // JDIH hanya memuat abstrak, bukan naskah lengkap
	// SumberURL: halaman resmi (JDIH Kemenkeu / pajak.go.id / JDIH BPK).
	SumberURL string `json:"sumber_url,omitempty"`

	file string // relatif terhadap folder katalog
	teks string // huruf kecil, untuk pencarian
}

type Opsi struct {
	Kode   string `json:"kode"`
	Label  string `json:"label"`
	Jumlah int    `json:"jumlah"`
}

// Facet: pilihan filter beserta jumlahnya, dihitung dari seluruh katalog.
type Facet struct {
	Kategori []Opsi `json:"kategori"`
	Jenis    []Opsi `json:"jenis"`
	Status   []Opsi `json:"status"`
	TahunMin int    `json:"tahun_min"`
	TahunMax int    `json:"tahun_max"`
}

type Query struct {
	Q                       string
	Kategori, Jenis, Status []string // OR di dalam grup, AND antar grup
	Dari, Sampai            int      // tahun; 0 = tanpa batas
	Urut                    string   // terbaru (bawaan) | terlama | abjad
	Halaman                 int
}

type Hasil struct {
	Items        []Peraturan `json:"items"`
	Total        int         `json:"total"`
	Halaman      int         `json:"halaman"`
	TotalHalaman int         `json:"total_halaman"`
	Facet        Facet       `json:"facet"`
}

// Data: satu muatan katalog, tidak berubah setelah dimuat (aman dibaca bersamaan).
type Data struct {
	dir   string
	list  []Peraturan // urut terbaru
	index map[string]int
	facet Facet
}

// Urutan dan label baku, mengikuti pengelompokan katalog sumber.
var (
	kategoriBaku = []Opsi{
		{Kode: "PPN", Label: "Pajak Pertambahan Nilai"},
		{Kode: "PPnBM", Label: "Pajak Penjualan atas Barang Mewah"},
		{Kode: "PPh", Label: "Pajak Penghasilan"},
		{Kode: "KUP", Label: "Ketentuan Umum Perpajakan"},
		{Kode: "PBB", Label: "Pajak Bumi & Bangunan"},
		{Kode: "BEA_MATERAI", Label: "Bea Meterai"},
		{Kode: "BEA_CUKAI", Label: "Bea & Cukai"},
		{Kode: "PAJAK_DAERAH", Label: "Pajak Daerah"},
		{Kode: "UMKM", Label: "UMKM & PPh Final"},
		{Kode: "FAKTUR", Label: "Faktur Pajak"},
		{Kode: "NPWP", Label: "NPWP & Pendaftaran"},
		{Kode: "PPS", Label: "Pengungkapan Sukarela"},
		{Kode: "CARBON_TAX", Label: "Pajak Karbon"},
		{Kode: "TRANSFER_PRICING", Label: "Transfer Pricing"},
		{Kode: "TINDAK_PIDANA", Label: "Tindak Pidana Perpajakan"},
		{Kode: "SENGKETA_PAJAK", Label: "Sengketa Pajak"},
		{Kode: "KURS", Label: "Kurs KMK"},
		{Kode: "KEUNEG", Label: "Keuangan Negara"},
		{Kode: "PENG_SP_DJP", Label: "Pengumuman & Siaran Pers DJP"},
		{Kode: "TANPA_KATEGORI", Label: "Lainnya"},
	}
	jenisBaku = []Opsi{
		{Kode: "UU", Label: "Undang-Undang"},
		{Kode: "PP", Label: "Peraturan Pemerintah"},
		{Kode: "PERPRES", Label: "Peraturan Presiden"},
		{Kode: "PMK", Label: "Peraturan Menteri Keuangan"},
		{Kode: "KMK", Label: "Keputusan Menteri Keuangan"},
		{Kode: "PERDIRJEN", Label: "Peraturan Direktur Jenderal"},
		{Kode: "SE_DJP", Label: "Surat Edaran DJP"},
		{Kode: "SEMA", Label: "Surat Edaran Mahkamah Agung"},
		{Kode: "LAIN", Label: "Lainnya"},
	}
	statusBaku = []Opsi{{Kode: "Berlaku"}, {Kode: "Diubah"}, {Kode: "Dicabut sebagian"}, {Kode: "Dicabut"}, {Kode: "Tidak berlaku"}}

	// Kolom jenis dari JDIH (nama panjang) atau manifest (kode).
	jenisDariNama = map[string]string{
		"undang-undang": "UU", "uu": "UU",
		"peraturan pemerintah": "PP", "pp": "PP",
		"peraturan presiden": "PERPRES", "perpres": "PERPRES",
		"peraturan menteri": "PMK", "peraturan menteri keuangan": "PMK", "pmk": "PMK",
		"keputusan menteri": "KMK", "keputusan menteri keuangan": "KMK", "kmk": "KMK",
		"peraturan unit eselon i": "PERDIRJEN", "perdirjen": "PERDIRJEN", "peraturan direktur jenderal pajak": "PERDIRJEN",
		"se_djp": "SE_DJP", "sema": "SEMA",
	}
	reTahun = regexp.MustCompile(`(19|20)\d{2}`)
)

func label(baku []Opsi, kode string) string {
	if i := slices.IndexFunc(baku, func(o Opsi) bool { return o.Kode == kode }); i >= 0 && baku[i].Label != "" {
		return baku[i].Label
	}
	return kode
}

// jenisDari: kode jenis dari kolom jenis, atau dari awalan nomor bila kolomnya kosong.
func jenisDari(jenis, kode string) string {
	if j, ok := jenisDariNama[strings.ToLower(strings.TrimSpace(jenis))]; ok {
		return j
	}
	k := strings.ToUpper(kode)
	for _, p := range []struct{ awalan, jenis string }{
		{"UU", "UU"}, {"PP ", "PP"}, {"PERPRES", "PERPRES"}, {"PMK", "PMK"}, {"KMK", "KMK"},
		{"PER-", "PERDIRJEN"}, {"KEP-", "PERDIRJEN"}, {"PENG-", "PERDIRJEN"}, {"SE-", "SE_DJP"}, {"SEMA", "SEMA"},
	} {
		if strings.HasPrefix(k, p.awalan) {
			return p.jenis
		}
	}
	return "LAIN"
}

// statusRapi: "berlaku" → "Berlaku".
func statusRapi(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + strings.ToLower(s[n:])
}

// tanggal: YYYY-MM-DD yang sah; 1900-01-01 adalah pengisi kosong dari JDIH.
func tanggal(s string) string {
	s = strings.TrimSpace(s)
	if t, err := time.Parse(time.DateOnly, s); err != nil || t.Year() <= 1900 {
		return ""
	}
	return s
}

func ada(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}

// Muat membaca dir/index.csv dan memeriksa naskah mana yang sudah ada di disk.
func Muat(dir string) (*Data, error) {
	f, err := os.Open(filepath.Join(dir, "index.csv"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	head, err := r.Read()
	if err != nil {
		return nil, err
	}
	kol := map[string]int{}
	for i, h := range head {
		kol[strings.TrimSpace(strings.TrimPrefix(h, "\ufeff"))] = i
	}
	if _, ok := kol["uuid"]; !ok {
		return nil, errors.New("index.csv tanpa kolom uuid")
	}
	d := &Data{dir: dir, index: map[string]int{}}
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			continue // baris rusak dilewati; baris lain tetap termuat
		}
		get := func(name string) string {
			if i, ok := kol[name]; ok && i < len(rec) {
				return strings.TrimSpace(rec[i])
			}
			return ""
		}
		p := Peraturan{
			UUID: get("uuid"), Kode: get("kode"), Judul: get("judul"),
			Status:      statusRapi(get("status")),
			Ditetapkan:  tanggal(get("tanggal_penetapan")),
			Diundangkan: tanggal(get("tanggal_pengundangan")),
			Berlaku:     tanggal(get("berlaku")),
			SumberLN:    get("sumber_ln"),
			Abstrak:     strings.Contains(get("catatan"), "abstrak"),
			SumberURL:   cmp.Or(get("jdih_url"), get("sumber_url")),
		}
		if _, dup := d.index[p.UUID]; p.UUID == "" || dup {
			continue
		}
		p.Jenis = jenisDari(get("jenis"), p.Kode)
		for k := range strings.SplitSeq(get("kategori"), ";") {
			if k = strings.TrimSpace(k); k != "" {
				p.Kategori = append(p.Kategori, k)
			}
		}
		if p.Kategori == nil {
			p.Kategori = []string{"TANPA_KATEGORI"}
		}
		p.Tahun, _ = strconv.Atoi(get("tahun"))
		if p.Tahun == 0 && p.Ditetapkan != "" {
			p.Tahun, _ = strconv.Atoi(p.Ditetapkan[:4])
		}
		if m := reTahun.FindAllString(p.Kode, -1); p.Tahun == 0 && m != nil {
			p.Tahun, _ = strconv.Atoi(m[len(m)-1])
		}
		p.Halaman, _ = strconv.Atoi(get("halaman"))
		if p.SumberLN == "tanpa slug JDIH" || p.SumberLN == "-" { // catatan skrip unduh / kosong
			p.SumberLN = ""
		}
		file := get("file")
		ext := strings.ToLower(filepath.Ext(file))
		if (ext == ".pdf" || ext == ".htm" || ext == ".html") && filepath.IsLocal(file) && ada(filepath.Join(dir, file)) {
			p.file, p.Format = file, strings.TrimPrefix(ext, ".")[:3]
			p.Lampiran = p.Format == "htm" && ada(filepath.Join(dir, lampiranDari(file)))
		}
		p.JenisLabel = label(jenisBaku, p.Jenis)
		teks := []string{p.Kode, p.Judul, p.Jenis, p.JenisLabel, strconv.Itoa(p.Tahun)}
		for _, k := range p.Kategori { // "pph 21" menemukan judul "Pajak Penghasilan Pasal 21"
			p.KategoriLabel = append(p.KategoriLabel, label(kategoriBaku, k))
			teks = append(teks, k, label(kategoriBaku, k))
		}
		p.teks = strings.ToLower(strings.Join(teks, " "))
		d.index[p.UUID] = len(d.list)
		d.list = append(d.list, p)
	}
	slices.SortStableFunc(d.list, terbaru)
	for i, p := range d.list {
		d.index[p.UUID] = i
	}
	d.facet = hitungFacet(d.list)
	return d, nil
}

func lampiranDari(file string) string {
	return strings.TrimSuffix(file, filepath.Ext(file)) + "-lampiran.pdf"
}

// tanggalUrut: tanggal penetapan, atau pengundangan bila kosong.
func tanggalUrut(p Peraturan) string { return cmp.Or(p.Ditetapkan, p.Diundangkan) }

// tanpaTahun: 1 bila tahun tidak diketahui, supaya selalu di akhir daftar.
func tanpaTahun(p Peraturan) int {
	if p.Tahun == 0 {
		return 1
	}
	return 0
}

// terbaru: tahun lalu tanggal menurun.
func terbaru(a, b Peraturan) int {
	return cmp.Or(cmp.Compare(tanpaTahun(a), tanpaTahun(b)), cmp.Compare(b.Tahun, a.Tahun),
		cmp.Compare(tanggalUrut(b), tanggalUrut(a)), cmp.Compare(a.Kode, b.Kode))
}

func terlama(a, b Peraturan) int {
	return cmp.Or(cmp.Compare(tanpaTahun(a), tanpaTahun(b)), cmp.Compare(a.Tahun, b.Tahun),
		cmp.Compare(tanggalUrut(a), tanggalUrut(b)), cmp.Compare(a.Kode, b.Kode))
}

func abjad(a, b Peraturan) int {
	return cmp.Or(cmp.Compare(strings.ToLower(a.Judul), strings.ToLower(b.Judul)), cmp.Compare(a.Kode, b.Kode))
}

func hitungFacet(list []Peraturan) Facet {
	kat, jen, sta := map[string]int{}, map[string]int{}, map[string]int{}
	var f Facet
	for _, p := range list {
		for _, k := range p.Kategori {
			kat[k]++
		}
		jen[p.Jenis]++
		if p.Status != "" {
			sta[p.Status]++
		}
		if p.Tahun > 0 {
			if f.TahunMin == 0 || p.Tahun < f.TahunMin {
				f.TahunMin = p.Tahun
			}
			f.TahunMax = max(f.TahunMax, p.Tahun)
		}
	}
	f.Kategori, f.Jenis, f.Status = opsiDari(kategoriBaku, kat), opsiDari(jenisBaku, jen), opsiDari(statusBaku, sta)
	return f
}

// opsiDari: opsi baku yang muncul di data (urutan baku), lalu kode lain urut abjad.
func opsiDari(baku []Opsi, jumlah map[string]int) []Opsi {
	out := []Opsi{}
	for _, o := range baku {
		if n := jumlah[o.Kode]; n > 0 {
			out = append(out, Opsi{Kode: o.Kode, Label: cmp.Or(o.Label, o.Kode), Jumlah: n})
			delete(jumlah, o.Kode)
		}
	}
	lain := []Opsi{}
	for k, n := range jumlah {
		lain = append(lain, Opsi{Kode: k, Label: k, Jumlah: n})
	}
	slices.SortFunc(lain, func(a, b Opsi) int { return cmp.Compare(a.Kode, b.Kode) })
	return append(out, lain...)
}

func (d *Data) Total() int { return len(d.list) }

func (d *Data) Ambil(uuid string) (Peraturan, bool) {
	i, ok := d.index[uuid]
	if !ok {
		return Peraturan{}, false
	}
	return d.list[i], true
}

// Berkas: path naskah (atau lampirannya) di disk; ok=false bila belum tersedia.
func (d *Data) Berkas(uuid string, lampiran bool) (string, bool) {
	p, ok := d.Ambil(uuid)
	switch {
	case !ok || p.Format == "":
		return "", false
	case lampiran && !p.Lampiran:
		return "", false
	case lampiran:
		return filepath.Join(d.dir, lampiranDari(p.file)), true
	}
	return filepath.Join(d.dir, p.file), true
}

func (d *Data) Cari(q Query) Hasil {
	kata := strings.Fields(strings.ToLower(q.Q))
	items := []Peraturan{}
	for _, p := range d.list {
		switch {
		case len(q.Kategori) > 0 && !slices.ContainsFunc(p.Kategori, func(k string) bool { return slices.Contains(q.Kategori, k) }),
			len(q.Jenis) > 0 && !slices.Contains(q.Jenis, p.Jenis),
			len(q.Status) > 0 && !slices.Contains(q.Status, p.Status),
			q.Dari > 0 && p.Tahun < q.Dari,
			q.Sampai > 0 && (p.Tahun == 0 || p.Tahun > q.Sampai),
			!semuaKata(p.teks, kata):
			continue
		}
		items = append(items, p)
	}
	switch q.Urut {
	case "terlama":
		slices.SortStableFunc(items, terlama)
	case "abjad":
		slices.SortStableFunc(items, abjad)
	}
	h := Hasil{Total: len(items), TotalHalaman: max(1, (len(items)+PerHalaman-1)/PerHalaman), Facet: d.facet}
	h.Halaman = min(max(q.Halaman, 1), h.TotalHalaman)
	h.Items = items[min((h.Halaman-1)*PerHalaman, len(items)):min(h.Halaman*PerHalaman, len(items))]
	return h
}

func semuaKata(teks string, kata []string) bool {
	for _, k := range kata {
		if !cocok(teks, k) {
			return false
		}
	}
	return true
}

// cocok: kata ada di teks; kata berupa angka harus utuh, supaya "10" tidak menemukan
// "105", "010", atau "2010".
func cocok(teks, kata string) bool {
	if strings.TrimFunc(kata, unicode.IsDigit) != "" {
		return strings.Contains(teks, kata)
	}
	for i := 0; ; {
		j := strings.Index(teks[i:], kata)
		if j < 0 {
			return false
		}
		a, b := i+j, i+j+len(kata)
		if (a == 0 || !digit(teks[a-1])) && (b == len(teks) || !digit(teks[b])) {
			return true
		}
		i = a + 1
	}
}

func digit(c byte) bool { return '0' <= c && c <= '9' }

// Katalog: folder katalog yang dimuat ulang otomatis bila index.csv berubah, supaya tim
// cukup menyalin naskah + index.csv baru ke server tanpa restart.
type Katalog struct {
	dir  string
	mu   sync.Mutex
	mod  time.Time
	data *Data
}

func Buka(dir string) *Katalog { return &Katalog{dir: dir} }

// Data: katalog terkini. Bila muat ulang gagal (mis. index.csv sedang ditulis), data lama
// tetap dipakai.
func (k *Katalog) Data() (*Data, error) {
	fi, err := os.Stat(filepath.Join(k.dir, "index.csv"))
	k.mu.Lock()
	defer k.mu.Unlock()
	if err != nil {
		if k.data != nil {
			return k.data, nil
		}
		return nil, err
	}
	if k.data == nil || !fi.ModTime().Equal(k.mod) {
		d, err := Muat(k.dir)
		k.mod = fi.ModTime()
		if err != nil {
			if k.data != nil {
				return k.data, nil
			}
			return nil, err
		}
		k.data = d
	}
	return k.data, nil
}
