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
	// MP: anggota tim Master Pajak. Tidak punya usaha (tenant_id 0) dan hanya membuka Panel
	// Master; role tim & klien yang ditugaskan diatur di internal/master.
	MP Role = "mp"
	// Lihat: peran sementara anggota tim saat "lihat sebagai klien" — hanya baca. Tidak pernah
	// disimpan di user mana pun; server memasangnya per request (lihat cmd/server/lihat.go).
	Lihat Role = "lihat"
)

type RoleInfo struct {
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Views       []string `json:"views"`
}

var financeViews = []string{
	// "arus-kas" dihapus: Arus Kas jadi tab di dalam Laporan Keuangan, bukan menu sendiri.
	"dashboard", "laporan-keuangan", "laporan-penjualan", "transaksi", "piutang-utang", "stok-gudang",
	"produksi-supplier", "invoice", "simulasi-harga", "pajak-umkm", "perencanaan-pajak", "kalkulator-pajak", "karyawan", "master-data", "panduan", "pustaka", "lapor-bug",
	"umur-piutang", "tutup-buku", "buku-besar",
}

// Roles adalah satu-satunya sumber kebenaran hak akses; frontend menerima Views lewat /auth/me.
var Roles = map[Role]RoleInfo{
	Owner: {"Owner", "Akses penuh, termasuk kelola pengguna & akses.", append(slices.Clone(financeViews), "transaksi-kelola", "impor-data", "saldo-awal", "pengguna", "profil-usaha", "langganan", "usulan-koreksi")},
	Admin: {"Admin Keuangan", "Semua laporan & operasional, tanpa kelola pengguna.", append(slices.Clone(financeViews), "transaksi-kelola", "impor-data", "saldo-awal")},
	Kasir: {"Kasir", "Dashboard dan catat transaksi (tanpa ubah/hapus).", []string{"dashboard", "transaksi", "lapor-bug"}},
	MP:    {"Tim Master Pajak", "Panel Master sesuai role tim.", []string{"panel-master"}},
	Lihat: {"Lihat sebagai (tim Master Pajak)", "Membaca data usaha tanpa bisa mengubah.", []string{
		"dashboard", "laporan-keuangan", "laporan-penjualan", "transaksi", "piutang-utang", "stok-gudang",
		"produksi-supplier", "invoice", "pajak-umkm", "karyawan", "master-data", "profil-usaha", "umur-piutang", "tutup-buku",
	}},
}

func (r Role) Can(view string) bool { return slices.Contains(Roles[r].Views, view) }

type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      Role   `json:"role"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"created_at"`
	TenantID  int    `json:"tenant_id"`    // usaha pemilik data; 1 = usaha awal (Kedai Kopi Senja); 0 = tim MP
	WA        string `json:"wa,omitempty"` // 628…; wajib saat daftar mandiri, opsional untuk user tambahan
	// Persetujuan UU PDP (UU 27/2022) saat daftar: waktu (UTC) & versi kebijakan privasi.
	PersetujuanPDP time.Time `json:"persetujuan_pdp,omitzero"`
	VersiPDP       string    `json:"versi_pdp,omitempty"`
	// LastSeen: terakhir memakai aplikasi (dicatat paling sering tiap touchInterval).
	LastSeen time.Time `json:"last_seen,omitzero"`
}

// Input untuk Create/Update. Password kosong saat Update = tidak diubah.
type Input struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     Role   `json:"role"`
	Active   bool   `json:"active"`
	Password string `json:"password"`
	TenantID int    `json:"-"` // diisi server, bukan dari body request; 0 = usaha 1 (role MP: tanpa usaha)
	WA       string `json:"-"` // hanya lewat daftar mandiri; Update tidak mengubahnya
	// Persetujuan UU PDP, diisi server saat daftar mandiri.
	PersetujuanPDP time.Time `json:"-"`
	VersiPDP       string    `json:"-"`
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
		{Name: "Bu Sari (Pemilik)", Email: "owner@kedaikopisenja.id", Role: Owner, Active: true, Password: seedPassword, TenantID: 1},
		{Name: "Dimas (Keuangan)", Email: "admin@kedaikopisenja.id", Role: Admin, Active: true, Password: seedPassword, TenantID: 1},
		{Name: "Rani (Kasir)", Email: "kasir@kedaikopisenja.id", Role: Kasir, Active: true, Password: seedPassword, TenantID: 1},
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

// tenantFor: akun tim MP tidak punya usaha; user lain minimal usaha 1 (data lama tanpa tenant_id).
func tenantFor(role Role, tenantID int) int {
	if role == MP {
		return 0
	}
	return max(tenantID, 1)
}

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

// NormalizeWA mengubah nomor WhatsApp seluler Indonesia (08…, +628…, 628…; spasi, '-' dan
// '.' diabaikan) ke bentuk 628… yang dipakai API WhatsApp. Panjang akhir 10–15 digit.
func NormalizeWA(s string) (string, error) {
	n := strings.NewReplacer(" ", "", "-", "", ".", "", "+", "").Replace(strings.TrimSpace(s))
	if strings.HasPrefix(n, "08") {
		n = "62" + n[1:]
	}
	if !strings.HasPrefix(n, "628") || len(n) < 10 || len(n) > 15 || strings.Trim(n, "0123456789") != "" {
		return "", ErrInvalid
	}
	return n, nil
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
	if in.Role == MP && in.TenantID != 0 || in.Role == Lihat { // Owner usaha tidak bisa membuat akun tim
		return User{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.emailTaken(in.Email, 0) {
		return User{}, ErrEmailTaken
	}
	s.nextID++
	salt := []byte(rand.Text())
	r := &record{
		User: User{ID: s.nextID, Name: in.Name, Email: in.Email, Role: in.Role, Active: in.Active, CreatedAt: time.Now().Format("2006-01-02"), TenantID: tenantFor(in.Role, in.TenantID),
			WA: in.WA, PersetujuanPDP: in.PersetujuanPDP, VersiPDP: in.VersiPDP},
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
	if (in.Role == MP) != (tenantID == 0) || in.Role == Lihat { // user usaha tidak bisa dipindah ke tim, dan sebaliknya
		return User{}, ErrInvalid
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
	if tenantID != 0 && (in.Role != Owner || !in.Active) && s.activeOwners(tenantID, id) == 0 {
		return User{}, ErrLastOwner
	}
	if !in.Active || in.Role != r.Role {
		s.revoke(id) // sesi lama tidak boleh hidup lagi setelah diaktifkan ulang / ganti role
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
	if tenantID != 0 && s.activeOwners(tenantID, id) == 0 {
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
			r.LastSeen = s.Now().UTC().Truncate(time.Second)
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

// touchInterval: last_seen cukup akurat per 5 menit; lebih sering hanya menambah tulisan.
const touchInterval = 5 * time.Minute

// Touch mencatat user masih memakai aplikasi; true bila last_seen berubah.
// ponytail: tidak memicu simpan sendiri — ikut tersimpan pada penulisan berikutnya, jadi
// paling banyak hilang beberapa menit bila server mati.
func (s *Store) Touch(id int, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.users[id]
	if !ok || now.Sub(r.LastSeen) < touchInterval {
		return false
	}
	r.LastSeen = now.UTC().Truncate(time.Second)
	return true
}

// FindByEmail: user dengan email ini (tanpa membedakan huruf besar & spasi tepi).
func (s *Store) FindByEmail(email string) (User, bool) {
	email = normEmail(email)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.users {
		if r.Email == email {
			return r.User, true
		}
	}
	return User{}, false
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
		u.TenantID = tenantFor(u.Role, u.TenantID) // state.json lama belum menyimpan tenant_id
		s.users[u.ID] = &record{u.User, u.Salt, u.Hash}
		s.nextID = max(s.nextID, u.ID)
	}
	for _, sess := range snap.Sessions {
		s.sessions[sess.Token] = session{sess.UserID, sess.Expires}
	}
	return s
}

// --- Panel Master: kelola user semua usaha ---

// Get: user mana pun (semua usaha).
func (s *Store) Get(id int) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.users[id]
	if !ok {
		return User{}, false
	}
	return r.User, true
}

// All: semua user, urut ID.
func (s *Store) All() []User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]User, 0, len(s.users))
	for _, r := range s.users {
		out = append(out, r.User)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// UbahKontak: nama, email, dan nomor WA user usaha (bukan akun tim). Role & password tetap.
func (s *Store) UbahKontak(id int, name, email, wa string) (User, error) {
	in := Input{Name: name, Email: email, Role: Kasir}
	if err := validate(&in, false); err != nil {
		return User{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.users[id]
	if !ok || r.Role == MP {
		return User{}, ErrNotFound
	}
	if s.emailTaken(in.Email, id) {
		return User{}, ErrEmailTaken
	}
	if r.Email != in.Email {
		s.revoke(id) // login lama memakai email lain
	}
	r.Name, r.Email, r.WA = in.Name, in.Email, strings.TrimSpace(wa)
	return r.User, nil
}

// ResetPassword membuat password sementara baru (dikembalikan sekali) dan mengakhiri semua sesi.
func (s *Store) ResetPassword(id int) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.users[id]
	if !ok {
		return "", ErrNotFound
	}
	pw := strings.ToLower(rand.Text()[:12])
	r.hash = hashPassword(pw, r.salt)
	s.revoke(id)
	return pw, nil
}

// HapusUsaha menghapus permanen semua user satu usaha (hard delete usaha). Usaha 0 (tim) ditolak.
func (s *Store) HapusUsaha(tenantID int) int {
	if tenantID == 0 {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for id, r := range s.users {
		if r.TenantID == tenantID {
			delete(s.users, id)
			s.revoke(id)
			n++
		}
	}
	return n
}
