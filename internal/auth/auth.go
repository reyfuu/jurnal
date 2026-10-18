// Package auth: user, sesi login, dan role tetap (Owner / Admin Keuangan / Kasir).
package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

// ponytail: users & sessions in-memory like the finance seed; move to Postgres `users`/`sessions` tables with tenant_id once the DB lands.

type Role string

const (
	Owner Role = "owner"
	Admin Role = "admin"
	Kasir Role = "kasir"
)

type RoleInfo struct {
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Views       []string `json:"views"`
}

var financeViews = []string{
	"dashboard", "laporan-keuangan", "arus-kas", "transaksi", "piutang-utang", "stok-gudang",
	"produksi-supplier", "invoice", "simulasi-harga", "pajak-umkm", "master-data", "panduan",
}

// Roles adalah satu-satunya sumber kebenaran hak akses; frontend menerima Views lewat /auth/me.
var Roles = map[Role]RoleInfo{
	Owner: {"Owner", "Akses penuh, termasuk kelola pengguna & akses.", append(slices.Clone(financeViews), "transaksi-kelola", "pengguna")},
	Admin: {"Admin Keuangan", "Semua laporan & operasional, tanpa kelola pengguna.", append(slices.Clone(financeViews), "transaksi-kelola")},
	Kasir: {"Kasir", "Dashboard dan catat transaksi (tanpa ubah/hapus).", []string{"dashboard", "transaksi"}},
}

func (r Role) Can(view string) bool { return slices.Contains(Roles[r].Views, view) }

type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      Role   `json:"role"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"created_at"`
}

// Input untuk Create/Update. Password kosong saat Update = tidak diubah.
type Input struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     Role   `json:"role"`
	Active   bool   `json:"active"`
	Password string `json:"password"`
}

var (
	ErrInvalid        = errors.New("data pengguna tidak valid")
	ErrEmailTaken     = errors.New("email sudah dipakai")
	ErrNotFound       = errors.New("pengguna tidak ditemukan")
	ErrLastOwner      = errors.New("minimal harus ada satu Owner aktif")
	ErrBadCredentials = errors.New("email atau password salah")
)

// DemoPassword dipakai semua akun seed; ditampilkan di halaman login mode demo.
const DemoPassword = "senja2026"

type record struct {
	User
	salt, hash []byte
}

type Store struct {
	mu       sync.RWMutex
	users    map[int]*record
	nextID   int
	sessions map[string]int
}

func NewStore() *Store {
	s := &Store{users: map[int]*record{}, sessions: map[string]int{}}
	for _, in := range []Input{
		{"Bu Sari (Pemilik)", "owner@kedaikopisenja.id", Owner, true, DemoPassword},
		{"Dimas (Keuangan)", "admin@kedaikopisenja.id", Admin, true, DemoPassword},
		{"Rani (Kasir)", "kasir@kedaikopisenja.id", Kasir, true, DemoPassword},
	} {
		if _, err := s.Create(in); err != nil {
			panic(err)
		}
	}
	return s
}

func hashPassword(password string, salt []byte) []byte {
	key, err := pbkdf2.Key(sha256.New, password, salt, 210_000, 32)
	if err != nil {
		panic(err) // only fails for invalid key length
	}
	return key
}

func normEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }

func validate(in *Input, requirePassword bool) error {
	in.Name, in.Email = strings.TrimSpace(in.Name), normEmail(in.Email)
	_, roleOK := Roles[in.Role]
	if in.Name == "" || !strings.Contains(in.Email, "@") || !roleOK {
		return ErrInvalid
	}
	if (requirePassword || in.Password != "") && len(in.Password) < 8 {
		return ErrInvalid
	}
	return nil
}

// activeOwners menghitung Owner aktif, mengabaikan user dengan ID except.
func (s *Store) activeOwners(except int) int {
	n := 0
	for id, r := range s.users {
		if id != except && r.Role == Owner && r.Active {
			n++
		}
	}
	return n
}

func (s *Store) emailTaken(email string, except int) bool {
	for id, r := range s.users {
		if id != except && r.Email == email {
			return true
		}
	}
	return false
}

func (s *Store) Create(in Input) (User, error) {
	if err := validate(&in, true); err != nil {
		return User{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.emailTaken(in.Email, 0) {
		return User{}, ErrEmailTaken
	}
	s.nextID++
	salt := []byte(rand.Text())
	r := &record{
		User: User{ID: s.nextID, Name: in.Name, Email: in.Email, Role: in.Role, Active: in.Active, CreatedAt: time.Now().Format("2006-01-02")},
		salt: salt,
		hash: hashPassword(in.Password, salt),
	}
	s.users[r.ID] = r
	return r.User, nil
}

func (s *Store) Update(id int, in Input) (User, error) {
	if err := validate(&in, false); err != nil {
		return User{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	if s.emailTaken(in.Email, id) {
		return User{}, ErrEmailTaken
	}
	if (in.Role != Owner || !in.Active) && s.activeOwners(id) == 0 {
		return User{}, ErrLastOwner
	}
	r.Name, r.Email, r.Role, r.Active = in.Name, in.Email, in.Role, in.Active
	if in.Password != "" {
		r.hash = hashPassword(in.Password, r.salt)
	}
	return r.User, nil
}

func (s *Store) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[id]; !ok {
		return ErrNotFound
	}
	if s.activeOwners(id) == 0 {
		return ErrLastOwner
	}
	delete(s.users, id)
	return nil
}

func (s *Store) List() []User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]User, 0, len(s.users))
	for _, r := range s.users {
		out = append(out, r.User)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Store) Login(email, password string) (string, User, error) {
	email = normEmail(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.users {
		if r.Email == email && r.Active && subtle.ConstantTimeCompare(hashPassword(password, r.salt), r.hash) == 1 {
			token := rand.Text()
			s.sessions[token] = r.ID
			return token, r.User, nil
		}
	}
	return "", User{}, ErrBadCredentials
}

func (s *Store) Logout(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

// UserBySession mengembalikan user aktif pemilik token; user yang dinonaktifkan/dihapus otomatis tertolak.
func (s *Store) UserBySession(token string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.users[s.sessions[token]]
	if !ok || !r.Active {
		return User{}, false
	}
	return r.User, true
}
