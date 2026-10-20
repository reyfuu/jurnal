// Package pengetahuan: bahan yang dibaca AI saat menilai klien — artikel pajak (juga dibaca
// klien di Pustaka Pajak) dan sumber pengetahuan (tulis manual, Excel, PDF). AI hanya memakai
// artikel terbit dan sumber yang sudah diverifikasi tim. Pencarian: BM25 di memori.
//
// ponytail: indeks dihitung ulang tiap pencarian dari semua potongan di memori; cukup untuk
// ribuan potongan. Pindah ke full-text search Postgres / pgvector bila melewati ~10 rb potongan
// atau pencarian makna (sinonim) mulai dibutuhkan.
package pengetahuan

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/ledongthuc/pdf"
)

const (
	Draft         = "draft"
	Terbit        = "terbit"
	Menunggu      = "menunggu"
	Terverifikasi = "terverifikasi"
)

var Kategori = []string{"PPh Final UMKM", "PPh 21", "PPh 23", "PPh Badan", "PPN", "Pembukuan", "Administrasi"}

var (
	ErrTidakValid = errors.New("data tidak valid")
	ErrPDFKosong  = errors.New("PDF tidak berisi teks (kemungkinan hasil scan); ketik ringkasannya lewat Tulis manual")
)

type Artikel struct {
	ID           int       `json:"id"`
	Judul        string    `json:"judul"`
	Kategori     string    `json:"kategori"`
	DasarHukum   string    `json:"dasar_hukum,omitempty"`
	BerlakuSejak string    `json:"berlaku_sejak,omitempty"`
	Tag          []string  `json:"tag"`
	Isi          string    `json:"isi"`
	Status       string    `json:"status"`
	TampilKlien  bool      `json:"tampil_klien"`
	Oleh         string    `json:"oleh"`                   // penulis pertama
	Diverifikasi string    `json:"diverifikasi,omitempty"` // yang menerbitkan
	Waktu        time.Time `json:"waktu"`
}

type Sumber struct {
	ID    int    `json:"id"`
	Judul string `json:"judul"`
	Jenis string `json:"jenis"` // manual | excel | pdf (format masukan)
	// Kategori isi: tidak terbatas aturan — pengalaman lapangan tim & pengetahuan umum juga
	// dipakai AI, dengan label supaya model membedakannya dari peraturan.
	Kategori     string    `json:"kategori"`
	Topik        []string  `json:"topik"`
	Isi          string    `json:"isi"`
	Status       string    `json:"status"`
	DipakaiAI    bool      `json:"dipakai_ai"`
	Oleh         string    `json:"oleh"`
	Diverifikasi string    `json:"diverifikasi,omitempty"`
	Waktu        time.Time `json:"waktu"`
}

// KategoriSumber: kategori isi sumber pengetahuan (urutan tampilan).
var KategoriSumber = []struct{ ID, Label string }{
	{"peraturan", "Peraturan & UU"}, {"artikel", "Artikel / teori pajak"}, {"praktik", "Praktik lapangan"},
	{"umum", "Pengetahuan umum"}, {"lainnya", "Lainnya"},
}

// LabelKategori: label kategori; kosong (sumber lama) = Pengetahuan umum.
func LabelKategori(id string) string {
	for _, k := range KategoriSumber {
		if k.ID == id {
			return k.Label
		}
	}
	return "Pengetahuan umum"
}

type State struct {
	Artikel       []Artikel `json:"artikel"`
	Sumber        []Sumber  `json:"sumber"`
	NextArtikelID int       `json:"next_artikel_id"`
	NextSumberID  int       `json:"next_sumber_id"`
}

func NewState() State { return State{NextArtikelID: 1, NextSumberID: 1} }

func (s State) Clone() State {
	c := s
	c.Artikel, c.Sumber = slices.Clone(s.Artikel), slices.Clone(s.Sumber)
	return c
}

func RefArtikel(id int) string { return fmt.Sprintf("A-%03d", id) }
func RefSumber(id int) string  { return fmt.Sprintf("KB-%03d", id) }

func rapikanTag(tag []string) []string {
	out := []string{}
	for _, t := range tag {
		if t = strings.TrimSpace(t); t != "" && !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	return out
}

// SimpanArtikel menambah (ID 0) atau mengubah artikel. terbit=true menerbitkan sekaligus
// mencatat yang memverifikasi; terbit=false menyimpan sebagai draft (artikel terbit ikut turun).
func (s *State) SimpanArtikel(a Artikel, terbit bool, oleh string, now time.Time) (Artikel, error) {
	a.Judul, a.Isi = strings.TrimSpace(a.Judul), strings.TrimSpace(a.Isi)
	a.DasarHukum, a.BerlakuSejak, a.Tag = strings.TrimSpace(a.DasarHukum), strings.TrimSpace(a.BerlakuSejak), rapikanTag(a.Tag)
	if a.Judul == "" || a.Isi == "" || !slices.Contains(Kategori, a.Kategori) || len([]rune(a.Isi)) > 50_000 {
		return Artikel{}, fmt.Errorf("%w: judul, isi (maks 50 rb karakter), dan kategori wajib", ErrTidakValid)
	}
	a.Status, a.Waktu, a.Diverifikasi = Draft, now, ""
	if terbit {
		a.Status, a.Diverifikasi = Terbit, oleh
	}
	if a.ID == 0 {
		a.ID, a.Oleh = s.NextArtikelID, oleh
		s.NextArtikelID++
		s.Artikel = append(s.Artikel, a)
		return a, nil
	}
	i := slices.IndexFunc(s.Artikel, func(x Artikel) bool { return x.ID == a.ID })
	if i < 0 {
		return Artikel{}, fmt.Errorf("%w: artikel tidak ditemukan", ErrTidakValid)
	}
	a.Oleh = s.Artikel[i].Oleh
	s.Artikel[i] = a
	return a, nil
}

// TambahSumber menyimpan sumber baru berstatus menunggu verifikasi (belum dipakai AI).
func (s *State) TambahSumber(src Sumber, oleh string, now time.Time) (Sumber, error) {
	src.Judul, src.Isi, src.Topik = strings.TrimSpace(src.Judul), strings.TrimSpace(src.Isi), rapikanTag(src.Topik)
	if src.Judul == "" || src.Isi == "" || !slices.Contains([]string{"manual", "excel", "pdf"}, src.Jenis) || len(src.Isi) > 2<<20 {
		return Sumber{}, fmt.Errorf("%w: judul & isi wajib (maks 2 MB teks)", ErrTidakValid)
	}
	if src.Kategori = strings.TrimSpace(src.Kategori); src.Kategori == "" {
		src.Kategori = "umum"
	}
	if !slices.ContainsFunc(KategoriSumber, func(k struct{ ID, Label string }) bool { return k.ID == src.Kategori }) {
		return Sumber{}, fmt.Errorf("%w: kategori %q tidak dikenal", ErrTidakValid, src.Kategori)
	}
	src.ID, src.Status, src.DipakaiAI, src.Oleh, src.Diverifikasi, src.Waktu = s.NextSumberID, Menunggu, false, oleh, "", now
	s.NextSumberID++
	s.Sumber = append(s.Sumber, src)
	return src, nil
}

func (s *State) sumber(id int) (*Sumber, error) {
	i := slices.IndexFunc(s.Sumber, func(x Sumber) bool { return x.ID == id })
	if i < 0 {
		return nil, fmt.Errorf("%w: sumber tidak ditemukan", ErrTidakValid)
	}
	return &s.Sumber[i], nil
}

// Verifikasi: sumber diperiksa tim dan mulai dipakai AI.
func (s *State) Verifikasi(id int, oleh string) error {
	src, err := s.sumber(id)
	if err == nil {
		src.Status, src.DipakaiAI, src.Diverifikasi = Terverifikasi, true, oleh
	}
	return err
}

// SetDipakai menyalakan/mematikan pemakaian sumber terverifikasi oleh AI.
func (s *State) SetDipakai(id int, pakai bool) error {
	src, err := s.sumber(id)
	if err != nil {
		return err
	}
	if src.Status != Terverifikasi {
		return fmt.Errorf("%w: verifikasi sumber dulu sebelum dipakai AI", ErrTidakValid)
	}
	src.DipakaiAI = pakai
	return nil
}

func (s *State) HapusSumber(id int) error {
	i := slices.IndexFunc(s.Sumber, func(x Sumber) bool { return x.ID == id })
	if i < 0 {
		return fmt.Errorf("%w: sumber tidak ditemukan", ErrTidakValid)
	}
	s.Sumber = slices.Delete(s.Sumber, i, i+1)
	return nil
}

// --- potongan & pencarian ---

// MaksPotongan: panjang satu potongan teks yang dikirim ke AI.
const MaksPotongan = 900

type Potongan struct {
	Ref   string `json:"ref"`
	Judul string `json:"judul"`
	// Kategori: label jenis isi (Artikel pajak, Peraturan & UU, Praktik lapangan, …).
	Kategori string  `json:"kategori,omitempty"`
	Teks     string  `json:"teks"`
	Skor     float64 `json:"skor,omitempty"`
}

// Potongan: semua potongan yang boleh dipakai AI — artikel terbit & sumber terverifikasi.
func (s State) Potongan() []Potongan {
	var out []Potongan
	for _, a := range s.Artikel {
		if a.Status != Terbit {
			continue
		}
		kepala := strings.Join(slices.DeleteFunc([]string{a.Kategori, a.DasarHukum, strings.Join(a.Tag, ", ")}, func(x string) bool { return x == "" }), " · ")
		for _, b := range Pecah(kepala+"\n\n"+a.Isi, MaksPotongan) {
			out = append(out, Potongan{Ref: RefArtikel(a.ID), Judul: a.Judul, Kategori: "Artikel pajak terbit", Teks: b})
		}
	}
	for _, src := range s.Sumber {
		if src.Status != Terverifikasi || !src.DipakaiAI {
			continue
		}
		for _, b := range Pecah(src.Isi, MaksPotongan) {
			out = append(out, Potongan{Ref: RefSumber(src.ID), Judul: src.Judul, Kategori: LabelKategori(src.Kategori), Teks: b})
		}
	}
	return out
}

// JumlahPotongan per sumber/artikel (untuk tampilan "terindeks").
func JumlahPotongan(isi string) int { return len(Pecah(isi, MaksPotongan)) }

// Pecah memotong teks per paragraf; paragraf pendek digabung sampai maks, paragraf panjang
// dipotong di batas kalimat lalu kata.
func Pecah(isi string, maks int) []string {
	var out []string
	cur := ""
	flush := func() {
		if strings.TrimSpace(cur) != "" {
			out = append(out, strings.TrimSpace(cur))
		}
		cur = ""
	}
	for _, par := range strings.Split(strings.ReplaceAll(isi, "\r\n", "\n"), "\n\n") {
		par = strings.TrimSpace(par)
		if par == "" {
			continue
		}
		for _, bag := range potongPanjang(par, maks) {
			if len([]rune(cur))+len([]rune(bag))+2 > maks {
				flush()
			}
			if cur != "" {
				cur += "\n\n"
			}
			cur += bag
		}
	}
	flush()
	return out
}

func potongPanjang(par string, maks int) []string {
	if len([]rune(par)) <= maks {
		return []string{par}
	}
	var out []string
	cur := ""
	for _, kata := range strings.Fields(par) {
		if len([]rune(cur))+len([]rune(kata))+1 > maks && cur != "" {
			out = append(out, cur)
			cur = ""
		}
		if cur != "" {
			cur += " "
		}
		cur += kata
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

var stopwords = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`yang dan di ke dari untuk dengan atau pada ini itu adalah akan juga dalam tidak bisa
		boleh masih sudah apa apakah bagaimana berapa kalau jika saya kami kita anda kamu ada sebagai oleh karena agar
		atas bagi para per tersebut yaitu harus dapat lebih sampai pakai memakai`) {
		stopwords[w] = true
	}
}

// token: kata huruf kecil tanpa tanda baca; angka tetap (mis. "21", "23", "55").
func token(s string) []string {
	f := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	return slices.DeleteFunc(f, func(w string) bool { return stopwords[w] || len([]rune(w)) < 2 && !unicode.IsDigit([]rune(w)[0]) })
}

// Cari: k potongan paling relevan untuk q (BM25, judul ikut dihitung). nil bila q tanpa kata bermakna.
func Cari(pot []Potongan, q string, k int) []Potongan {
	qt := token(q)
	if len(qt) == 0 || len(pot) == 0 {
		return nil
	}
	const k1, b = 1.2, 0.75
	docs := make([][]string, len(pot))
	df := map[string]int{}
	total := 0
	for i, p := range pot {
		docs[i] = token(p.Judul + " " + p.Teks)
		total += len(docs[i])
		seen := map[string]bool{}
		for _, w := range docs[i] {
			if !seen[w] {
				df[w]++
				seen[w] = true
			}
		}
	}
	avg := float64(total) / float64(len(pot))
	var out []Potongan
	for i, p := range pot {
		tf := map[string]int{}
		for _, w := range docs[i] {
			tf[w]++
		}
		var skor float64
		for _, w := range slices.Compact(slices.Sorted(slices.Values(qt))) {
			if tf[w] == 0 {
				continue
			}
			idf := math.Log(1 + (float64(len(pot))-float64(df[w])+0.5)/(float64(df[w])+0.5))
			f := float64(tf[w])
			skor += idf * f * (k1 + 1) / (f + k1*(1-b+b*float64(len(docs[i]))/avg))
		}
		if skor > 0 {
			p.Skor = math.Round(skor*100) / 100
			out = append(out, p)
		}
	}
	slices.SortStableFunc(out, func(a, b Potongan) int { return cmp.Compare(b.Skor, a.Skor) })
	if len(out) > k {
		out = out[:k]
	}
	return out
}

// --- impor ---

// ParseExcel: kolom Judul, Isi, Topik (pisah koma), Sumber. Tiap baris jadi satu paragraf
// "Judul: Isi (Sumber)" supaya potongannya tetap utuh per baris.
func ParseExcel(rows [][]string) (isi string, topik []string, err error) {
	kol := func(r []string, i int) string {
		if i < len(r) {
			return strings.TrimSpace(r[i])
		}
		return ""
	}
	var par []string
	for i, r := range rows {
		if i == 0 {
			continue
		}
		judul, teks, sumber := kol(r, 0), kol(r, 1), kol(r, 3)
		if teks == "" {
			continue
		}
		p := teks
		if judul != "" {
			p = judul + ": " + teks
		}
		if sumber != "" {
			p += " (" + sumber + ")"
		}
		par = append(par, p)
		topik = append(topik, strings.Split(kol(r, 2), ",")...)
	}
	if len(par) == 0 {
		return "", nil, fmt.Errorf("%w: tidak ada baris berisi kolom Isi", ErrTidakValid)
	}
	return strings.Join(par, "\n\n"), rapikanTag(topik), nil
}

// TeksPDF mengambil teks dari PDF berbasis teks. PDF hasil scan (gambar) → ErrPDFKosong.
func TeksPDF(data []byte) (teks string, err error) {
	defer func() { // pustaka PDF bisa panik pada berkas rusak
		if r := recover(); r != nil {
			teks, err = "", fmt.Errorf("%w: PDF tidak bisa dibaca", ErrTidakValid)
		}
	}()
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("%w: bukan PDF yang bisa dibaca", ErrTidakValid)
	}
	rd, err := r.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrTidakValid, err)
	}
	raw, err := io.ReadAll(io.LimitReader(rd, 2<<20))
	if err != nil {
		return "", err
	}
	teks = strings.TrimSpace(string(raw))
	if teks == "" {
		return "", ErrPDFKosong
	}
	return teks, nil
}
