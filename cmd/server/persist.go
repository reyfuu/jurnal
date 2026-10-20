package main

import (
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"

	"pajak/internal/auth"
	"pajak/internal/finance"
)

// ponytail: snapshot JSON atomik (tulis temp lalu rename) untuk 1 usaha & ratusan user;
// ganti ke database saat multi-tenant (roadmap) — tanpa dependensi, image scratch tetap.

type state struct {
	Transactions []finance.Transaction `json:"transactions"`
	NextTxID     int                   `json:"next_tx_id"`
	Assets       []finance.Asset       `json:"assets"`
	NextAssetID  int                   `json:"next_asset_id"`
	// Opening: neraca awal onboarding; nil = belum diisi.
	Opening        *finance.OpeningBalance `json:"opening"`
	Karyawan       []finance.Karyawan      `json:"karyawan"`
	NextKaryawanID int                     `json:"next_karyawan_id"`
	Invoices       []finance.Invoice       `json:"invoices"`
	NextInvoiceID  int                     `json:"next_invoice_id"`
	Auth           auth.StoreSnapshot      `json:"auth"`
	Langganan      *langganan              `json:"langganan,omitempty"`
}

// dataPath: berkas di $DATA_DIR; kosong = mode in-memory (test, dev lokal).
func dataPath(name string) string {
	if dir := os.Getenv("DATA_DIR"); dir != "" {
		return filepath.Join(dir, name)
	}
	return ""
}

func writeFileAtomic(path string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// file: berkas data usaha. Usaha 1 memakai nama lama (state.json, business.json) supaya data
// produksi tetap terbaca; usaha lain state-<id>.json, business-<id>.json.
func (t *tenant) file(name string) string {
	if t.id == 1 {
		return dataPath(name + ".json")
	}
	return dataPath(fmt.Sprintf("%s-%d.json", name, t.id))
}

// loadTenants memuat semua usaha hasil register (state-<id>.json).
func loadTenants() (map[int]*tenant, error) {
	out := map[int]*tenant{}
	if dataPath("") == "" {
		return out, nil
	}
	paths, _ := filepath.Glob(dataPath("state-*.json"))
	for _, p := range paths {
		var id int
		if _, err := fmt.Sscanf(filepath.Base(p), "state-%d.json", &id); err != nil || id < 2 {
			continue
		}
		st, err := loadState(p)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		t := newTenant(id)
		t.restore(st)
		t.profile = loadProfile(t.file("business"))
		out[id] = t
	}
	return out, nil
}

func loadState(p string) (*state, error) {
	if p == "" {
		return nil, nil
	}
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var st state
	return &st, json.Unmarshal(b, &st)
}

// snapshot menyalin data usaha ke bentuk berkas; Auth diisi pemanggil (hanya usaha 1).
// Slice disalin di bawah kunci: handler mengubah elemen di tempat (s.txs[i] = ...).
func (t *tenant) snapshot() state {
	t.mu.RLock()
	defer t.mu.RUnlock()
	l := t.langganan
	l.Orders = slices.Clone(l.Orders)
	return state{
		Transactions: slices.Clone(t.txs), NextTxID: t.nextTxID,
		Assets: slices.Clone(t.assets), NextAssetID: t.nextAssetID,
		Opening: t.opening, Karyawan: slices.Clone(t.karyawan), NextKaryawanID: t.nextKaryawanID,
		Invoices: slices.Clone(t.invoices), NextInvoiceID: t.nextInvoiceID,
		Langganan: &l,
	}
}

// save menulis semua usaha (user & sesi ikut state.json usaha 1); dipanggil setelah setiap
// perubahan yang berhasil.
//
// ponytail: tulis ulang semua berkas usaha tiap perubahan; cukup untuk puluhan usaha QA/pilot,
// ganti ke database (roadmap multi-tenant) sebelum ratusan usaha.
func (s *Server) save() {
	if dataPath("") == "" {
		return
	}
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	s.tenantsMu.RLock()
	tenants := slices.Collect(maps.Values(s.tenants))
	s.tenantsMu.RUnlock()
	for _, t := range tenants {
		st := t.snapshot()
		if t.id == 1 {
			st.Auth = s.users.Snapshot()
		}
		if err := writeFileAtomic(t.file("state"), st); err != nil {
			log.Printf("GAGAL menyimpan state ke %s: %v", t.file("state"), err)
		}
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// saveFor menyimpan usaha t (nil = hanya user & sesi) setelah perubahan. Tanpa Postgres,
// semua berkas JSON ditulis ulang seperti sebelumnya.
func (s *Server) saveFor(t *tenant) {
	if s.db == nil {
		s.save()
		return
	}
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	if t != nil {
		if err := saveTenantDB(s.db, t); err != nil {
			log.Printf("GAGAL menyimpan usaha %d ke Postgres: %v", t.id, err)
		}
	}
	if err := saveAuthDB(s.db, s.users.Snapshot()); err != nil {
		log.Printf("GAGAL menyimpan user & sesi ke Postgres: %v", err)
	}
}

// saveAll menyimpan semua usaha (seed awal atau impor JSON ke Postgres).
func (s *Server) saveAll() {
	if s.db == nil {
		s.save()
		return
	}
	for _, t := range s.tenants {
		s.saveFor(t)
	}
	for _, rep := range s.bugs.list {
		if err := insertBugDB(s.db, rep); err != nil {
			log.Fatalf("impor laporan bug %d: %v", rep.ID, err)
		}
	}
}

// changedTenant: usaha milik sesi baru di respons (login, daftar) atau sesi request.
func (s *Server) changedTenant(r *http.Request, h http.Header) *tenant {
	for _, c := range append((&http.Response{Header: h}).Cookies(), r.Cookies()...) {
		if u, ok := s.users.UserBySession(c.Value); ok && c.Name == sessionCookie {
			s.tenantsMu.RLock()
			defer s.tenantsMu.RUnlock()
			return s.tenants[u.TenantID]
		}
	}
	return nil
}

// persisting menyimpan state setelah handler pengubah data selesai tanpa error.
func (s *Server) persisting(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		h(rec, r)
		if rec.status < 400 {
			s.saveFor(s.changedTenant(r, rec.Header()))
		}
	}
}
