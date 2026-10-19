// Package support menyimpan laporan bug dari pengguna supaya sampai ke pengembang.
package support

import (
	"errors"
	"slices"
	"strings"
	"sync"
	"time"
)

// Laporan bug disimpan di server lebih dulu, baru (kalau dikonfigurasi) diteruskan ke
// webhook pengembang. Urutannya begitu dan bukan sebaliknya: penerusan bisa gagal karena
// jaringan atau URL salah, dan laporan yang hilang karena itu akan hilang selamanya —
// pengguna sudah merasa melapor, padahal tidak ada jejaknya.

const (
	StatusBaru    = "baru"
	StatusDibaca  = "dibaca"
	StatusSelesai = "selesai"
	StatusDitolak = "bukan bug"
)

var StatusList = []string{StatusBaru, StatusDibaca, StatusSelesai, StatusDitolak}

const (
	Ringan      = "ringan"     // mengganggu tapi masih bisa dipakai
	Mengganggu  = "mengganggu" // bikin salah angka atau harus diakali
	Parah       = "parah"      // tidak bisa dipakai / data hilang
	KeparahanTl = "pertanyaan" // bukan bug, cuma bertanya
)

var KeparahanList = []string{Ringan, Mengganggu, Parah, KeparahanTl}

// Laporan: satu laporan bug beserta konteks yang ikut terkirim.
type Laporan struct {
	ID        int    `json:"id"`
	Dibuat    string `json:"dibuat"` // RFC3339
	Judul     string `json:"judul"`
	Deskripsi string `json:"deskripsi"`
	// Langkah: cara memunculkan masalahnya. Paling menentukan apakah bug bisa diperbaiki.
	Langkah   string `json:"langkah"`
	Keparahan string `json:"keparahan"`
	// Halaman: menu tempat masalah ditemui, diisi otomatis dari menu yang sedang dibuka.
	Halaman string `json:"halaman"`

	// Konteks otomatis. Ditampilkan apa adanya ke pelapor sebelum ia mengirim, supaya tidak
	// ada yang terkirim tanpa ia tahu.
	Pelapor   string `json:"pelapor"`
	Email     string `json:"email"`
	Peran     string `json:"peran"`
	TenantID  int    `json:"tenant_id"`
	UserAgent string `json:"user_agent"`
	Versi     string `json:"versi"`

	Status string `json:"status"`
	// Balasan dari pengembang; hanya bisa diisi ID Master.
	Balasan string `json:"balasan,omitempty"`
	// Diteruskan menandai apakah laporan sudah sampai ke webhook pengembang.
	Diteruskan bool   `json:"diteruskan"`
	GagalKirim string `json:"gagal_kirim,omitempty"`
}

const (
	maxJudul     = 150
	maxDeskripsi = 4000
	maxLangkah   = 2000
)

func Validate(l *Laporan) error {
	for _, s := range []*string{&l.Judul, &l.Deskripsi, &l.Langkah, &l.Halaman, &l.Keparahan} {
		*s = strings.TrimSpace(*s)
	}
	if l.Judul == "" {
		return errors.New("judul laporan wajib diisi")
	}
	if len([]rune(l.Judul)) > maxJudul {
		return errors.New("judul terlalu panjang (maks 150 huruf)")
	}
	if len([]rune(l.Deskripsi)) < 10 {
		return errors.New("ceritakan masalahnya sedikit lebih lengkap (minimal 10 huruf) supaya bisa ditelusuri")
	}
	if len([]rune(l.Deskripsi)) > maxDeskripsi {
		return errors.New("penjelasan terlalu panjang (maks 4000 huruf)")
	}
	if len([]rune(l.Langkah)) > maxLangkah {
		return errors.New("langkah terlalu panjang (maks 2000 huruf)")
	}
	if !slices.Contains(KeparahanList, l.Keparahan) {
		l.Keparahan = Mengganggu
	}
	return nil
}

// Store menyimpan laporan seluruh usaha dalam satu daftar: pengembang perlu melihat semuanya,
// bukan per usaha.
type Store struct {
	mu     sync.RWMutex
	list   []Laporan
	nextID int
	Now    func() time.Time // bisa diganti di test
}

func NewStore() *Store { return &Store{nextID: 1, Now: time.Now} }

type Snapshot struct {
	Laporan []Laporan `json:"laporan"`
	NextID  int       `json:"next_id"`
}

func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Snapshot{Laporan: slices.Clone(s.list), NextID: s.nextID}
}

func (s *Store) Restore(snap Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.list = snap.Laporan
	s.nextID = max(snap.NextID, 1)
}

// Create menyimpan laporan dan mengembalikan salinannya yang sudah berisi ID & waktu.
func (s *Store) Create(l Laporan) (Laporan, error) {
	if err := Validate(&l); err != nil {
		return Laporan{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	l.ID = s.nextID
	s.nextID++
	l.Dibuat = s.Now().Format(time.RFC3339)
	l.Status = StatusBaru
	s.list = append(s.list, l)
	return l, nil
}

// List mengembalikan laporan terbaru lebih dulu. tenantID 0 = semua usaha (ID Master).
func (s *Store) List(tenantID int) []Laporan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Laporan, 0, len(s.list))
	for i := len(s.list) - 1; i >= 0; i-- {
		if tenantID == 0 || s.list[i].TenantID == tenantID {
			out = append(out, s.list[i])
		}
	}
	return out
}

var ErrNotFound = errors.New("laporan tidak ditemukan")

// Tanggapi mengubah status & balasan; hanya dipakai ID Master.
func (s *Store) Tanggapi(id int, status, balasan string) (Laporan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.list, func(l Laporan) bool { return l.ID == id })
	if i < 0 {
		return Laporan{}, ErrNotFound
	}
	if status != "" {
		if !slices.Contains(StatusList, status) {
			return Laporan{}, errors.New("status tidak dikenal")
		}
		s.list[i].Status = status
	}
	s.list[i].Balasan = strings.TrimSpace(balasan)
	return s.list[i], nil
}

// TandaiTerkirim mencatat hasil penerusan ke webhook pengembang.
func (s *Store) TandaiTerkirim(id int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.list, func(l Laporan) bool { return l.ID == id })
	if i < 0 {
		return
	}
	if err != nil {
		s.list[i].GagalKirim = err.Error()
		return
	}
	s.list[i].Diteruskan = true
	s.list[i].GagalKirim = ""
}

// Rekap untuk ditampilkan di halaman laporan.
type Rekap struct {
	Total     int            `json:"total"`
	PerStatus map[string]int `json:"per_status"`
}

func (s *Store) Rekap(tenantID int) Rekap {
	r := Rekap{PerStatus: map[string]int{}}
	for _, l := range s.List(tenantID) {
		r.Total++
		r.PerStatus[l.Status]++
	}
	return r
}
