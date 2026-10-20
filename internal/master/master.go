// Package master: hak akses tim Master Pajak di Panel Master — daftar izin, role yang bisa
// ditambah, penugasan klien ke anggota tim, dan audit log. Hanya data & aturan; HTTP dan
// penyimpanan ada di cmd/server.
package master

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

type Izin string

const (
	KlienSemua Izin = "klien_semua"
	KlienLihat Izin = "klien_lihat"
	Tiket      Izin = "tiket"
	// LihatSebagai: membaca laporan klien apa adanya (hanya baca, wajib alasan, tercatat).
	LihatSebagai Izin = "lihat_sebagai"
	UsulKoreksi  Izin = "usul_koreksi"
	COATemplate  Izin = "coa_template"
	Benchmark    Izin = "benchmark"
	// ParameterPajak: tarif, PTKP, batas omzet, norma NPPN yang dipakai semua kalkulator klien.
	ParameterPajak Izin = "parameter_pajak"
	Pengetahuan    Izin = "pengetahuan"
	Artikel        Izin = "artikel"
	ArtikelTerbit  Izin = "artikel_terbit"
	Langganan      Izin = "langganan"
	Pembayaran     Izin = "pembayaran"
	Tim            Izin = "tim"
	Audit          Izin = "audit"
)

type IzinInfo struct {
	ID         Izin   `json:"id"`
	Label      string `json:"label"`
	Keterangan string `json:"keterangan,omitempty"`
}

// DaftarIzin: urutan tampilan di layar Role & izin. Izin baru ditambah di sini bersama
// endpoint yang memeriksanya; izin tanpa endpoint hanya membingungkan Master.
var DaftarIzin = []IzinInfo{
	{KlienSemua, "Lihat semua klien", "Tanpa perlu ditugaskan"},
	{KlienLihat, "Lihat klien yang ditugaskan", "Ringkasan, daftar & detail klien, tanpa angka keuangan"},
	{Tiket, "Lihat & balas tiket", "Laporan bug dari klien yang bisa dilihat"},
	{LihatSebagai, "Lihat sebagai klien", "Membaca laporan klien, hanya baca; wajib alasan dan tercatat"},
	{UsulKoreksi, "Usulkan koreksi pembukuan", "Berlaku setelah pemilik usaha menyetujui"},
	{Langganan, "Atur langganan klien", "Perpanjang trial, aktifkan paket manual"},
	{COATemplate, "Kelola template COA", "Draft, impor Excel, dan terbitkan versi template per jenis usaha"},
	{Benchmark, "Kelola benchmark NPM nasional", "Impor versi data baru dan aktifkan"},
	{ParameterPajak, "Kelola parameter pajak", "Tarif, PTKP, batas omzet, norma NPPN; aktifkan versi baru"},
	{Pengetahuan, "Kelola basis pengetahuan AI", "Tambah sumber (manual, Excel, PDF), verifikasi, pilih yang dipakai AI"},
	{Artikel, "Tulis artikel pajak", "Simpan draft; terbit setelah diverifikasi"},
	{ArtikelTerbit, "Verifikasi & terbitkan artikel", ""},
	{Pembayaran, "Lihat pembayaran", "Order Midtrans semua klien yang bisa dilihat"},
	{Tim, "Kelola tim & akses", "Tambah anggota, role, dan penugasan klien"},
	{Audit, "Lihat audit log", ""},
}

func izinDikenal(i Izin) bool {
	return slices.ContainsFunc(DaftarIzin, func(x IzinInfo) bool { return x.ID == i })
}

// RoleMaster: role terkunci yang selalu punya semua izin, termasuk izin yang ditambahkan
// kemudian, supaya Panel Master tidak pernah terkunci dari dirinya sendiri.
const RoleMaster = "master"

type Role struct {
	ID    string `json:"id"`
	Nama  string `json:"nama"`
	Kunci bool   `json:"kunci"`
	Izin  []Izin `json:"izin"`
}

func DefaultRoles() []Role {
	semua := make([]Izin, len(DaftarIzin))
	for i, x := range DaftarIzin {
		semua[i] = x.ID
	}
	return []Role{
		{ID: RoleMaster, Nama: "Master", Kunci: true, Izin: semua},
		{ID: "konsultan", Nama: "Konsultan", Izin: []Izin{KlienLihat, Tiket, LihatSebagai, UsulKoreksi, Artikel}},
		{ID: "cs", Nama: "CS / Support", Izin: []Izin{KlienLihat, Tiket}},
	}
}

// Anggota: user tim (auth role "mp") beserta role tim dan usaha yang ditugaskan kepadanya.
type Anggota struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	Klien  []int  `json:"klien"`
}

// Catatan: satu baris audit log. Nama & Peran disalin saat kejadian supaya log tetap terbaca
// walau anggotanya kemudian dihapus atau ganti role.
type Catatan struct {
	ID         int       `json:"id"`
	Waktu      time.Time `json:"waktu"`
	UserID     int       `json:"user_id"`
	Nama       string    `json:"nama"`
	Peran      string    `json:"peran"`
	Aksi       string    `json:"aksi"`
	KlienID    int       `json:"klien_id,omitempty"`
	Keterangan string    `json:"keterangan,omitempty"`
}

type State struct {
	Roles       []Role    `json:"roles"`
	Anggota     []Anggota `json:"anggota"`
	Audit       []Catatan `json:"audit"`
	NextAuditID int       `json:"next_audit_id"`
}

func NewState() State { return State{Roles: DefaultRoles(), NextAuditID: 1} }

// Clone: salinan dalam supaya perubahan bisa dibatalkan bila penyimpanan gagal.
func (s State) Clone() State {
	c := State{Roles: slices.Clone(s.Roles), Anggota: slices.Clone(s.Anggota), Audit: slices.Clone(s.Audit), NextAuditID: s.NextAuditID}
	for i := range c.Roles {
		c.Roles[i].Izin = slices.Clone(c.Roles[i].Izin)
	}
	for i := range c.Anggota {
		c.Anggota[i].Klien = slices.Clone(c.Anggota[i].Klien)
	}
	return c
}

var (
	ErrTidakValid     = errors.New("data tidak valid")
	ErrRoleDipakai    = errors.New("role masih dipakai anggota tim")
	ErrMasterTerakhir = errors.New("minimal harus ada satu Master")
)

// Akses: hak efektif satu anggota.
type Akses struct {
	UserID int   `json:"user_id"`
	Role   Role  `json:"role"`
	Klien  []int `json:"klien"`
}

func (a Akses) Bisa(i Izin) bool { return a.Role.ID == RoleMaster || slices.Contains(a.Role.Izin, i) }

// BolehKlien: anggota boleh melihat usaha ini (semua klien, atau usaha yang ditugaskan).
func (a Akses) BolehKlien(id int) bool { return a.Bisa(KlienSemua) || slices.Contains(a.Klien, id) }

func (s *State) role(id string) (Role, bool) {
	i := slices.IndexFunc(s.Roles, func(r Role) bool { return r.ID == id })
	if i < 0 {
		return Role{}, false
	}
	return s.Roles[i], true
}

// Akses: ok=false bila user bukan anggota tim (atau role-nya sudah tidak ada).
func (s *State) Akses(userID int) (Akses, bool) {
	i := slices.IndexFunc(s.Anggota, func(a Anggota) bool { return a.UserID == userID })
	if i < 0 {
		return Akses{}, false
	}
	a := s.Anggota[i]
	r, ok := s.role(a.Role)
	if !ok {
		return Akses{}, false
	}
	return Akses{UserID: userID, Role: r, Klien: a.Klien}, true
}

// SimpanRole menambah role (ID kosong) atau mengubah role yang ada. Role Master tidak bisa diubah.
func (s *State) SimpanRole(r Role) (Role, error) {
	r.Nama = strings.TrimSpace(r.Nama)
	if r.Nama == "" || r.ID == RoleMaster {
		return Role{}, ErrTidakValid
	}
	izin := []Izin{}
	for _, i := range r.Izin {
		if !izinDikenal(i) {
			return Role{}, fmt.Errorf("%w: izin %q", ErrTidakValid, i)
		}
		if !slices.Contains(izin, i) {
			izin = append(izin, i)
		}
	}
	r.Izin, r.Kunci = izin, false
	if r.ID == "" {
		r.ID = s.idBaru(r.Nama)
		s.Roles = append(s.Roles, r)
		return r, nil
	}
	i := slices.IndexFunc(s.Roles, func(x Role) bool { return x.ID == r.ID })
	if i < 0 {
		return Role{}, ErrTidakValid
	}
	s.Roles[i] = r
	return r, nil
}

func (s *State) idBaru(nama string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(nama) {
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			b.WriteRune(c)
		case b.Len() > 0 && !strings.HasSuffix(b.String(), "-"):
			b.WriteByte('-')
		}
	}
	base := strings.Trim(b.String(), "-")
	if base == "" {
		base = "role"
	}
	id := base
	for n := 2; ; n++ {
		if _, ada := s.role(id); !ada {
			return id
		}
		id = fmt.Sprintf("%s-%d", base, n)
	}
}

func (s *State) HapusRole(id string) error {
	if id == RoleMaster {
		return ErrTidakValid
	}
	if slices.ContainsFunc(s.Anggota, func(a Anggota) bool { return a.Role == id }) {
		return ErrRoleDipakai
	}
	i := slices.IndexFunc(s.Roles, func(r Role) bool { return r.ID == id })
	if i < 0 {
		return ErrTidakValid
	}
	s.Roles = slices.Delete(s.Roles, i, i+1)
	return nil
}

// MasterLain: ada anggota lain (selain except) yang ber-role Master.
func (s *State) MasterLain(except int) bool { return s.masterLain(except) }

func (s *State) masterLain(except int) bool {
	return slices.ContainsFunc(s.Anggota, func(a Anggota) bool { return a.UserID != except && a.Role == RoleMaster })
}

// SetAnggota menambah atau mengubah keanggotaan. Klien disimpan unik & urut.
func (s *State) SetAnggota(a Anggota) error {
	if _, ok := s.role(a.Role); !ok || a.UserID <= 0 {
		return ErrTidakValid
	}
	klien := slices.Clone(a.Klien)
	slices.Sort(klien)
	a.Klien = slices.Compact(klien)
	i := slices.IndexFunc(s.Anggota, func(x Anggota) bool { return x.UserID == a.UserID })
	if i < 0 {
		s.Anggota = append(s.Anggota, a)
		return nil
	}
	if s.Anggota[i].Role == RoleMaster && a.Role != RoleMaster && !s.masterLain(a.UserID) {
		return ErrMasterTerakhir
	}
	s.Anggota[i] = a
	return nil
}

func (s *State) LepasAnggota(userID int) error {
	i := slices.IndexFunc(s.Anggota, func(x Anggota) bool { return x.UserID == userID })
	if i < 0 {
		return nil
	}
	if s.Anggota[i].Role == RoleMaster && !s.masterLain(userID) {
		return ErrMasterTerakhir
	}
	s.Anggota = slices.Delete(s.Anggota, i, i+1)
	return nil
}

// Penangan: anggota yang ditugaskan langsung ke usaha ini (tidak termasuk yang melihat semua
// klien). Kosong = belum ditugaskan; tiketnya hanya terlihat oleh yang melihat semua klien.
func (s *State) Penangan(klienID int) []int {
	out := []int{}
	for _, a := range s.Anggota {
		if ak, ok := s.Akses(a.UserID); ok && !ak.Bisa(KlienSemua) && slices.Contains(a.Klien, klienID) {
			out = append(out, a.UserID)
		}
	}
	return out
}

// Catat menambah baris audit. Audit hanya bertambah: tidak ada fungsi ubah atau hapus.
func (s *State) Catat(c Catatan) Catatan {
	c.ID = s.NextAuditID
	if c.Waktu.IsZero() {
		c.Waktu = time.Now().UTC().Truncate(time.Second)
	}
	s.NextAuditID++
	s.Audit = append(s.Audit, c)
	return c
}

// Batas "perlu perhatian" di Ringkasan Panel Master.
const (
	BatasTidakAktif = 14 * 24 * time.Hour
	BatasMauHabis   = 3 * 24 * time.Hour
)

// PerluPerhatian: usaha yang langganannya habis, trial-nya habis dalam 3 hari, atau tidak
// dibuka siapa pun selama 14 hari. terakhir nol = belum pernah tercatat (bukan berarti lalai).
func PerluPerhatian(status string, sampai, terakhir, now time.Time) (bool, string) {
	switch {
	case status == "habis":
		return true, "Langganan habis"
	case status == "trial" && sampai.Sub(now) <= BatasMauHabis:
		return true, "Masa coba segera habis"
	case !terakhir.IsZero() && now.Sub(terakhir) >= BatasTidakAktif:
		return true, fmt.Sprintf("Tidak aktif %d hari", int(now.Sub(terakhir).Hours()/24))
	}
	return false, ""
}
