// Package katalog: data acuan yang dikelola tim Master Pajak di Panel Master — template COA per
// jenis usaha, benchmark NPM/CTTOR nasional, dan parameter pajak (tarif, PTKP, batas, norma) —
// masing-masing berversi (draft → terbit/aktif → arsip). Hanya data & aturan; HTTP dan penyimpanan ada di cmd/server.
package katalog

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"pajak/internal/coa"
	"pajak/internal/tax"
)

const (
	Draft  = "draft"
	Terbit = "terbit" // template COA yang berlaku untuk klien baru
	Aktif  = "aktif"  // versi benchmark yang dipakai aplikasi
	Arsip  = "arsip"  // versi lama; template arsip tetap dipakai klien yang belum memperbarui
)

// JenisUsaha: jenis usaha di profil (finance.BusinessProfile.JenisUsaha) yang bisa punya template.
var JenisUsaha = []string{"perdagangan", "jasa", "manufaktur"}

var ErrTidakValid = errors.New("data tidak valid")

type Template struct {
	ID      int           `json:"id"`
	Jenis   string        `json:"jenis"`
	Versi   int           `json:"versi"` // 0 = draft
	Status  string        `json:"status"`
	Akun    []coa.Account `json:"akun"`
	Catatan string        `json:"catatan,omitempty"`
	Oleh    string        `json:"oleh,omitempty"`
	Waktu   time.Time     `json:"waktu"`
}

type Benchmark struct {
	ID     int                `json:"id"`
	Nama   string             `json:"nama"`
	Sumber string             `json:"sumber"`
	Status string             `json:"status"`
	Baris  []tax.KLUBenchmark `json:"baris"`
	Oleh   string             `json:"oleh,omitempty"`
	Waktu  time.Time          `json:"waktu"`
}

type State struct {
	Template        []Template       `json:"template"`
	Benchmark       []Benchmark      `json:"benchmark"`
	Parameter       []ParameterVersi `json:"parameter"`
	NextTemplateID  int              `json:"next_template_id"`
	NextBenchmarkID int              `json:"next_benchmark_id"`
	NextParameterID int              `json:"next_parameter_id"`
}

// NewState: data bawaan aplikasi — template perdagangan dari "Daftar COA.xlsx" dan benchmark
// nasional di internal/tax — sebagai versi 1.
func NewState() State {
	return State{
		Template:        []Template{{ID: 1, Jenis: "perdagangan", Versi: 1, Status: Terbit, Akun: slices.Clone(coa.Master), Catatan: "Daftar COA.xlsx, review akuntan Master Pajak"}},
		Benchmark:       []Benchmark{{ID: 1, Nama: "Benchmark nasional · data 2020", Sumber: tax.SumberBenchmark, Status: Aktif, Baris: slices.Clone(tax.KLUBenchmarks)}},
		Parameter:       []ParameterVersi{{ID: 1, Nama: "Bawaan aplikasi", Dasar: tax.SumberParameterBawaan, Status: Aktif, Isi: tax.ParameterBawaan()}},
		NextTemplateID:  2,
		NextBenchmarkID: 2,
		NextParameterID: 2,
	}
}

// Clone: salinan untuk diubah lalu disimpan (gagal simpan = state lama tetap).
func (s State) Clone() State {
	c := s
	c.Template, c.Benchmark, c.Parameter = slices.Clone(s.Template), slices.Clone(s.Benchmark), slices.Clone(s.Parameter)
	return c
}

// --- template COA ---

func (s *State) TemplateByID(id int) (Template, bool) {
	i := slices.IndexFunc(s.Template, func(t Template) bool { return t.ID == id })
	if i < 0 {
		return Template{}, false
	}
	return s.Template[i], true
}

// TemplateAktif: versi terbit terbaru untuk jenis usaha.
func (s *State) TemplateAktif(jenis string) (Template, bool) {
	var out Template
	for _, t := range s.Template {
		if t.Jenis == jenis && t.Status == Terbit && t.Versi > out.Versi {
			out = t
		}
	}
	return out, out.ID != 0
}

// SimpanDraft membuat atau mengganti draft jenis usaha itu (satu draft per jenis). Validasi
// penuh baru di Terbitkan, supaya draft yang belum lengkap tetap bisa disimpan.
func (s *State) SimpanDraft(jenis string, akun []coa.Account, catatan, oleh string, now time.Time) (Template, error) {
	if !slices.Contains(JenisUsaha, jenis) || len(akun) == 0 {
		return Template{}, fmt.Errorf("%w: jenis usaha %q atau daftar akun kosong", ErrTidakValid, jenis)
	}
	d := Template{Jenis: jenis, Status: Draft, Akun: slices.Clone(akun), Catatan: strings.TrimSpace(catatan), Oleh: oleh, Waktu: now}
	if i := slices.IndexFunc(s.Template, func(t Template) bool { return t.Jenis == jenis && t.Status == Draft }); i >= 0 {
		d.ID = s.Template[i].ID
		s.Template[i] = d
		return d, nil
	}
	d.ID = s.NextTemplateID
	s.NextTemplateID++
	s.Template = append(s.Template, d)
	return d, nil
}

// HapusDraft membuang draft; versi terbit/arsip tidak bisa dihapus.
func (s *State) HapusDraft(id int) error {
	i := slices.IndexFunc(s.Template, func(t Template) bool { return t.ID == id && t.Status == Draft })
	if i < 0 {
		return ErrTidakValid
	}
	s.Template = slices.Delete(s.Template, i, i+1)
	return nil
}

// Terbitkan menjadikan draft versi aktif jenis usahanya; versi aktif sebelumnya (lama) jadi arsip.
func (s *State) Terbitkan(id int, oleh string, now time.Time) (Template, *Template, error) {
	i := slices.IndexFunc(s.Template, func(t Template) bool { return t.ID == id && t.Status == Draft })
	if i < 0 {
		return Template{}, nil, fmt.Errorf("%w: hanya draft yang bisa diterbitkan", ErrTidakValid)
	}
	if err := coa.ValidateTemplate(s.Template[i].Akun); err != nil {
		return Template{}, nil, err
	}
	var lama *Template
	if a, ok := s.TemplateAktif(s.Template[i].Jenis); ok {
		j := slices.IndexFunc(s.Template, func(t Template) bool { return t.ID == a.ID })
		s.Template[j].Status = Arsip
		lama = &s.Template[j]
		s.Template[i].Versi = a.Versi + 1
	} else {
		s.Template[i].Versi = 1
	}
	s.Template[i].Status, s.Template[i].Oleh, s.Template[i].Waktu = Terbit, oleh, now
	return s.Template[i], lama, nil
}

// DiffAkun: perubahan dari template lama ke baru, untuk tawaran pembaruan ke klien.
type DiffAkun struct {
	Tambah []coa.Account `json:"tambah"`
	Hapus  []coa.Account `json:"hapus"`
	Ubah   []coa.Account `json:"ubah"` // isian versi baru
}

func Bandingkan(lama, baru []coa.Account) DiffAkun {
	d := DiffAkun{Tambah: []coa.Account{}, Hapus: []coa.Account{}, Ubah: []coa.Account{}}
	for _, b := range baru {
		switch l, ok := coa.Find(lama, b.Kode); {
		case !ok:
			d.Tambah = append(d.Tambah, b)
		case l != b:
			d.Ubah = append(d.Ubah, b)
		}
	}
	for _, l := range lama {
		if _, ok := coa.Find(baru, l.Kode); !ok {
			d.Hapus = append(d.Hapus, l)
		}
	}
	return d
}

// --- benchmark ---

func (s *State) BenchmarkByID(id int) (Benchmark, bool) {
	i := slices.IndexFunc(s.Benchmark, func(b Benchmark) bool { return b.ID == id })
	if i < 0 {
		return Benchmark{}, false
	}
	return s.Benchmark[i], true
}

func (s *State) BenchmarkAktif() Benchmark {
	i := slices.IndexFunc(s.Benchmark, func(b Benchmark) bool { return b.Status == Aktif })
	if i < 0 { // tidak terjadi lewat API; jaga-jaga data rusak
		return NewState().Benchmark[0]
	}
	return s.Benchmark[i]
}

func validasiBenchmark(baris []tax.KLUBenchmark) error {
	if len(baris) == 0 {
		return fmt.Errorf("%w: benchmark kosong", ErrTidakValid)
	}
	ada := map[string]bool{}
	for i, k := range baris {
		switch {
		case len(k.Kode) != 3 || strings.Trim(k.Kode, "0123456789") != "":
			return fmt.Errorf("%w: baris %d: KLU harus 3 angka", ErrTidakValid, i+1)
		case ada[k.Kode]:
			return fmt.Errorf("%w: KLU %s dobel", ErrTidakValid, k.Kode)
		case strings.TrimSpace(k.Nama) == "":
			return fmt.Errorf("%w: baris %d: nama wajib", ErrTidakValid, i+1)
		case k.NPMBps < -10_000 || k.NPMBps > 10_000 || k.CTTORBps < 0 || k.CTTORBps > 10_000:
			return fmt.Errorf("%w: baris %d: NPM/CTTOR di luar rentang wajar", ErrTidakValid, i+1)
		}
		ada[k.Kode] = true
	}
	return nil
}

// SimpanBenchmark menyimpan versi baru sebagai draft.
func (s *State) SimpanBenchmark(nama, sumber string, baris []tax.KLUBenchmark, oleh string, now time.Time) (Benchmark, error) {
	nama = strings.TrimSpace(nama)
	if nama == "" {
		return Benchmark{}, fmt.Errorf("%w: nama versi wajib", ErrTidakValid)
	}
	if err := validasiBenchmark(baris); err != nil {
		return Benchmark{}, err
	}
	b := Benchmark{ID: s.NextBenchmarkID, Nama: nama, Sumber: cmp.Or(strings.TrimSpace(sumber), nama), Status: Draft,
		Baris: slices.Clone(baris), Oleh: oleh, Waktu: now}
	s.NextBenchmarkID++
	s.Benchmark = append(s.Benchmark, b)
	return b, nil
}

// ErrKLUHilang: KLU yang sedang dipakai profil klien tidak ada di versi yang akan diaktifkan.
// Mengaktifkannya membuat profil klien itu gagal divalidasi saat disimpan berikutnya.
type ErrKLUHilang struct{ Kode []string }

func (e ErrKLUHilang) Error() string {
	return "KLU yang dipakai klien tidak ada di versi ini: " + strings.Join(e.Kode, ", ")
}

// AktifkanBenchmark menjadikan versi id aktif; versi aktif sebelumnya jadi arsip.
func (s *State) AktifkanBenchmark(id int, kluDipakai []string) error {
	i := slices.IndexFunc(s.Benchmark, func(b Benchmark) bool { return b.ID == id })
	if i < 0 || s.Benchmark[i].Status == Aktif {
		return fmt.Errorf("%w: versi tidak ditemukan atau sudah aktif", ErrTidakValid)
	}
	var hilang []string
	for _, k := range kluDipakai {
		if !slices.ContainsFunc(s.Benchmark[i].Baris, func(b tax.KLUBenchmark) bool { return b.Kode == k }) && !slices.Contains(hilang, k) {
			hilang = append(hilang, k)
		}
	}
	if len(hilang) > 0 {
		slices.Sort(hilang)
		return ErrKLUHilang{hilang}
	}
	for j := range s.Benchmark {
		if s.Benchmark[j].Status == Aktif {
			s.Benchmark[j].Status = Arsip
		}
	}
	s.Benchmark[i].Status = Aktif
	return nil
}

// --- Excel ---

var kepalaBenchmark = []string{"KLU", "Nama", "NPM (%)", "CTTOR (%)"}

// persen: "5,05" / "5.05" / "-2,41" → basis poin.
func persen(s string) (int64, error) {
	f, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "%")), ",", "."), 64)
	if err != nil {
		return 0, err
	}
	if f < 0 {
		return -int64(-f*100 + 0.5), nil
	}
	return int64(f*100 + 0.5), nil
}

func bpsPersen(bps int64) string {
	return strings.Replace(strconv.FormatFloat(float64(bps)/100, 'f', 2, 64), ".", ",", 1)
}

// BarisBenchmark: isi Excel (kepala + data); dipakai untuk unduh versi aktif sekaligus template impor.
func BarisBenchmark(b []tax.KLUBenchmark) [][]string {
	out := [][]string{kepalaBenchmark}
	for _, k := range b {
		out = append(out, []string{k.Kode, k.Nama, bpsPersen(k.NPMBps), bpsPersen(k.CTTORBps)})
	}
	return out
}

func kosong(r []string) bool {
	return !slices.ContainsFunc(r, func(c string) bool { return strings.TrimSpace(c) != "" })
}

func kolom(r []string, i int) string {
	if i < len(r) {
		return strings.TrimSpace(r[i])
	}
	return ""
}

func ParseBenchmark(rows [][]string) ([]tax.KLUBenchmark, error) {
	var out []tax.KLUBenchmark
	for i, r := range rows {
		if i == 0 || kosong(r) {
			continue
		}
		npm, e1 := persen(kolom(r, 2))
		cttor, e2 := persen(kolom(r, 3))
		if e1 != nil || e2 != nil {
			return nil, fmt.Errorf("%w: baris %d: NPM dan CTTOR harus angka persen", ErrTidakValid, i+1)
		}
		out = append(out, tax.KLUBenchmark{Kode: kolom(r, 0), Nama: kolom(r, 1), NPMBps: npm, CTTORBps: cttor})
		if err := validasiBenchmark(out[len(out)-1:]); err != nil {
			return nil, fmt.Errorf("baris %d: %w", i+1, err)
		}
	}
	return out, validasiBenchmark(out)
}

var kepalaTemplate = []string{"Kode", "Nama", "Subtype", "Saldo normal", "Kelompok", "Kondisi", "Catatan"}

func BarisTemplate(akun []coa.Account) [][]string {
	out := [][]string{kepalaTemplate}
	for _, a := range akun {
		out = append(out, []string{a.Kode, a.Nama, a.Subtype, string(a.Normal), string(a.Kelompok), string(a.Kondisi), a.Catatan})
	}
	return out
}

// ParseTemplate membaca daftar akun; kelengkapan (akun wajib) dicek saat terbit.
func ParseTemplate(rows [][]string) ([]coa.Account, error) {
	var out []coa.Account
	for i, r := range rows {
		if i == 0 || kosong(r) {
			continue
		}
		a := coa.Account{Kode: kolom(r, 0), Nama: kolom(r, 1), Subtype: kolom(r, 2), Normal: coa.Normal(strings.ToLower(kolom(r, 3))),
			Kelompok: coa.Kelompok(kolom(r, 4)), Kondisi: coa.Kondisi(strings.ToLower(cmp.Or(kolom(r, 5), "selalu"))), Catatan: kolom(r, 6)}
		if a.Normal != coa.Debit && a.Normal != coa.Kredit {
			return nil, fmt.Errorf("%w: baris %d: saldo normal harus debit atau kredit", ErrTidakValid, i+1)
		}
		out = append(out, a)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: tidak ada akun", ErrTidakValid)
	}
	return out, nil
}
