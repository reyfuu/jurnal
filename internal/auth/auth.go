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
	// "arus-kas" dihapus: Arus Kas jadi tab di dalam Laporan Keuangan, bukan menu sendiri.
	"dashboard", "laporan-keuangan", "laporan-penjualan", "transaksi", "piutang-utang", "stok-gudang",
	"produksi-supplier", "invoice", "simulasi-harga", "pajak-umkm", "perencanaan-pajak", "kalkulator-pajak", "karyawan", "master-data", "panduan",
	"lapor-bug",
}

// Roles adalah satu-satunya sumber kebenaran hak akses; frontend menerima Views lewat /auth/me.
var Roles = map[Role]RoleInfo{
	Owner: {"Owner", "Akses penuh, termasuk kelola pengguna & akses.", append(slices.Clone(financeViews), "transaksi-kelola", "impor-data", "pengguna", "profil-usaha")},
	Admin: {"Admin Keuangan", "Semua laporan & operasional, tanpa kelola pengguna.", append(slices.Clone(financeViews), "transaksi-kelola", "impor-data")},
	// Kasir ikut dapat menu Lapor Bug: dia yang paling sering menemui masalah saat memakai aplikasi.
	Kasir: {"Kasir", "Dashboard dan catat transaksi (tanpa ubah/hapus).", []string{"dashboard", "transaksi", "lapor-bug"}},
}

func (r Role) Can(view string) bool { return slices.Contains(Roles[r].Views, view) }

type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      Role   `json:"role"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"created_at"`
	TenantID  int    `json:"tenant_id"` // usaha pemilik data; 1 = usaha awal (Kedai Kopi Senja)
}

// Input untuk Create/Update. Password kosong saat Update = tidak diubah.
type Input struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     Role   `json:"role"`
	Active   bool   `json:"active"`
	Password string `json:"password"`
	TenantID int    `json:"-"` // diisi server, bukan dari body request; 0 = usaha 1
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

// SessionTTL: sesi login berlaku 12 jam (satu hari kerja), lalu wajib login ulang.
const SessionTTL = 12 * time.Hour

type session struct {
	userID  int
	expires time.Time
}

type Store struct {
	mu       sync.RWMutex
	users    map[int]*record
	nextID   int
	sessions map[string]session
	Now      func() time.Time // bisa diganti di test
}

func NewStore() *Store { return NewStoreWithPassword(DemoPassword) }

// NewStoreWithPassword membuat akun seed dengan password tertentu (produksi: env SEED_PASSWORD).
func NewStoreWithPassword(seedPassword string) *Store {
	s := &Store{users: map[int]*record{}, sessions: map[string]session{}, Now: time.Now}
	for _, in := range []Input{
		{"Bu Sari (Pemilik)", "owner@kedaikopisenja.id", Owner, true, seedPassword, 1},
		{"Dimas (Keuangan)", "admin@kedaikopisenja.id", Admin, true, seedPassword, 1},
		{"Rani (Kasir)", "kasir@kedaikopisenja.id", Kasir, true, seedPassword, 1},
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

// activeOwners menghitung Owner aktif satu usaha, mengabaikan user dengan ID except.
func (s *Store) activeOwners(tenantID, except int) int {
	n := 0
	for id, r := range s.users {
		if id != except && r.TenantID == tenantID && r.Role == Owner && r.Active {
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
		User: User{ID: s.nextID, Name: in.Name, Email: in.Email, Role: in.Role, Active: in.Active, CreatedAt: time.Now().Format("2006-01-02"), TenantID: max(in.TenantID, 1)},
		salt: salt,
		hash: hashPassword(in.Password, salt),
	}
	s.users[r.ID] = r
	return r.User, nil
}

// Update mengubah user milik usaha tenantID; user usaha lain dianggap tidak ada.
func (s *Store) Update(tenantID, id int, in Input) (User, error) {
	if err := validate(&in, false); err != nil {
		return User{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.users[id]
	if !ok || r.TenantID != tenantID {
		return User{}, ErrNotFound
	}
	if s.emailTaken(in.Email, id) {
		return User{}, ErrEmailTaken
	}
	if (in.Role != Owner || !in.Active) && s.activeOwners(tenantID, id) == 0 {
		return User{}, ErrLastOwner
	}
	r.Name, r.Email, r.Role, r.Active = in.Name, in.Email, in.Role, in.Active
	if in.Password != "" {
		r.hash = hashPassword(in.Password, r.salt)
		s.revoke(id)
	}
	return r.User, nil
}

func (s *Store) Delete(tenantID, id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.users[id]; !ok || r.TenantID != tenantID {
		return ErrNotFound
	}
	if s.activeOwners(tenantID, id) == 0 {
		return ErrLastOwner
	}
	delete(s.users, id)
	s.revoke(id)
	return nil
}

// List mengembalikan user satu usaha.
func (s *Store) List(tenantID int) []User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]User, 0, len(s.users))
	for _, r := range s.users {
		if r.TenantID == tenantID {
			out = append(out, r.User)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Store) Login(email, password string) (string, User, error) {
	email = normEmail(email)
	s.mu.Lock()
	defer s.mu.Unlock()
	for t, sess := range s.sessions {
		if !s.Now().Before(sess.expires) {
			delete(s.sessions, t)
		}
	}
	for _, r := range s.users {
		if r.Email == email && r.Active && subtle.ConstantTimeCompare(hashPassword(password, r.salt), r.hash) == 1 {
			token := rand.Text()
			s.sessions[token] = session{r.ID, s.Now().Add(SessionTTL)}
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
	sess, ok := s.sessions[token]
	if !ok || !s.Now().Before(sess.expires) {
		return User{}, false
	}
	r, ok := s.users[sess.userID]
	if !ok || !r.Active {
		return User{}, false
	}
	return r.User, true
}

// revoke menghapus semua sesi milik user; pemanggil memegang s.mu.
func (s *Store) revoke(userID int) {
	for t, sess := range s.sessions {
		if sess.userID == userID {
			delete(s.sessions, t)
		}
	}
}

// StoreSnapshot: bentuk serial Store untuk disimpan ke disk (hash password, bukan password).
type StoreSnapshot struct {
	Users []SnapshotUser `json:"users"`
	// Sessions: token → sesi; ikut disimpan agar user tidak logout setiap deploy.
	Sessions []SnapshotSession `json:"sessions"`
}

type SnapshotUser struct {
	User
	Salt []byte `json:"salt"`
	Hash []byte `json:"hash"`
}

type SnapshotSession struct {
	Token   string    `json:"token"`
	UserID  int       `json:"user_id"`
	Expires time.Time `json:"expires"`
}

func (s *Store) Snapshot() StoreSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap := StoreSnapshot{}
	for _, r := range s.users {
		snap.Users = append(snap.Users, SnapshotUser{r.User, r.salt, r.hash})
	}
	sort.Slice(snap.Users, func(i, j int) bool { return snap.Users[i].ID < snap.Users[j].ID })
	now := s.Now()
	for t, sess := range s.sessions {
		if now.Before(sess.expires) {
			snap.Sessions = append(snap.Sessions, SnapshotSession{t, sess.userID, sess.expires})
		}
	}
	return snap
}

// Restore membangun Store dari snapshot; akun seed tidak dibuat ulang.
func Restore(snap StoreSnapshot) *Store {
	s := &Store{users: map[int]*record{}, sessions: map[string]session{}, Now: time.Now}
	for _, u := range snap.Users {
		u.TenantID = max(u.TenantID, 1) // state.json lama belum menyimpan tenant_id
		s.users[u.ID] = &record{u.User, u.Salt, u.Hash}
		s.nextID = max(s.nextID, u.ID)
	}
	for _, sess := range snap.Sessions {
		s.sessions[sess.Token] = session{sess.UserID, sess.Expires}
	}
	return s
}
