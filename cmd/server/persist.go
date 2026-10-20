package main

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
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
	Tagihan        []finance.Tagihan       `json:"tagihan"`
	NextTagihanID  int                     `json:"next_tagihan_id"`
	Stok           []finance.StokItem      `json:"stok"`
	NextStokID     int                     `json:"next_stok_id"`
	Produk         []finance.Produk        `json:"produk"`
	NextProdukID   int                     `json:"next_produk_id"`
	Supplier       []finance.Supplier      `json:"supplier"`
	NextSupplierID int                     `json:"next_supplier_id"`
	Koreksi        []finance.Koreksi       `json:"koreksi"`
	NextKoreksiID  int                     `json:"next_koreksi_id"`
	CoaTemplate    int                     `json:"coa_template,omitempty"`
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
		// Disalin seperti slice lain di atas: snapshot ditulis ke disk di luar kunci, jadi
		// berbagi array dengan tenant yang masih hidup bisa menulis data yang berubah di tengah.
		Tagihan: slices.Clone(t.tagihan), NextTagihanID: t.nextTagihanID,
		Stok: slices.Clone(t.stok), NextStokID: t.nextStokID,
		Produk: slices.Clone(t.produk), NextProdukID: t.nextProdukID,
		Supplier: slices.Clone(t.supplier), NextSupplierID: t.nextSupplierID,
		Koreksi: slices.Clone(t.koreksi), NextKoreksiID: t.nextKoreksiID, CoaTemplate: t.coaTemplateID,
	}
}

// errSimpan: perubahan sudah di memori tapi gagal disimpan permanen (audit C9).
var errSimpan = errors.New("gagal menyimpan data")

// save menulis semua usaha (user & sesi ikut state.json usaha 1); dipanggil setelah setiap
// perubahan yang berhasil.
//
// ponytail: tulis ulang semua berkas usaha tiap perubahan; cukup untuk puluhan usaha QA/pilot.
// Produksi memakai Postgres (pgstore.go), yang hanya menulis usaha yang berubah.
func (s *Server) save() error {
	if dataPath("") == "" {
		return nil
	}
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	s.tenantsMu.RLock()
	tenants := slices.Collect(maps.Values(s.tenants))
	s.tenantsMu.RUnlock()
	var errs error
	for _, t := range tenants {
		st := t.snapshot()
		if t.id == 1 {
			st.Auth = s.users.Snapshot()
		}
		if err := writeFileAtomic(t.file("state"), st); err != nil {
			log.Printf("GAGAL menyimpan state ke %s: %v", t.file("state"), err)
			errs = errors.Join(errs, err)
		}
	}
	return errs
}

// saveFor menyimpan usaha t (nil = hanya user & sesi) setelah perubahan. Tanpa Postgres,
// semua berkas JSON ditulis ulang seperti sebelumnya.
func (s *Server) saveFor(t *tenant) error {
	var errs error
	if s.db == nil {
		errs = s.save()
	} else {
		s.saveMu.Lock()
		defer s.saveMu.Unlock()
		if t != nil {
			if err := saveTenantDB(s.db, t); err != nil {
				log.Printf("GAGAL menyimpan usaha %d ke Postgres: %v", t.id, err)
				errs = err
			}
		}
		if err := saveAuthDB(s.db, s.users.Snapshot()); err != nil {
			log.Printf("GAGAL menyimpan user & sesi ke Postgres: %v", err)
			errs = errors.Join(errs, err)
		}
	}
	if errs != nil {
		return fmt.Errorf("%w: %w", errSimpan, errs)
	}
	return nil
}

// saveAll menyimpan semua usaha (seed awal atau impor JSON ke Postgres).
func (s *Server) saveAll() {
	if s.db == nil {
		_ = s.save() // gagal sudah dicatat di log; mode JSON tetap jalan seperti sebelumnya
		return
	}
	for _, t := range s.tenants {
		if err := s.saveFor(t); err != nil {
			log.Fatalf("impor usaha %d ke Postgres: %v", t.id, err)
		}
	}
	for _, rep := range s.bugs.list {
		if err := insertBugDB(s.db, rep); err != nil {
			log.Fatalf("impor laporan bug %d: %v", rep.ID, err)
		}
	}
	if err := saveMasterDB(s.db, s.master, s.master.Audit); err != nil {
		log.Fatalf("impor tim Master Pajak: %v", err)
	}
	if err := saveKatalogDB(s.db, s.katalog); err != nil {
		log.Fatalf("impor katalog: %v", err)
	}
	if err := savePengetahuanDB(s.db, s.pengetahuan); err != nil {
		log.Fatalf("impor basis pengetahuan: %v", err)
	}
}

// sessionTenant: usaha milik sesi valid pertama di cookies; ok=false bila tidak ada sesi.
func (s *Server) sessionTenant(cookies []*http.Cookie) (*tenant, bool) {
	for _, c := range cookies {
		if u, ok := s.users.UserBySession(c.Value); ok && c.Name == sessionCookie {
			s.tenantsMu.RLock()
			defer s.tenantsMu.RUnlock()
			return s.tenants[u.TenantID], true
		}
	}
	return nil, false
}

// bufferedResponse menahan respons handler sampai data tersimpan.
type bufferedResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (b *bufferedResponse) Header() http.Header { return b.header }
func (b *bufferedResponse) WriteHeader(code int) {
	if b.status == 0 {
		b.status = code
	}
}
func (b *bufferedResponse) Write(p []byte) (int, error) {
	b.WriteHeader(http.StatusOK)
	return b.body.Write(p)
}

func (b *bufferedResponse) flush(w http.ResponseWriter) {
	maps.Copy(w.Header(), b.header)
	w.WriteHeader(cmp.Or(b.status, http.StatusOK))
	_, _ = w.Write(b.body.Bytes())
}

// persisting menyimpan state setelah handler pengubah data selesai tanpa error. Respons baru
// dikirim setelah simpan berhasil; gagal simpan → 503 supaya klien tidak mengira data aman
// (audit C9). Permintaan tanpa sesi sebelum maupun sesudahnya tidak menulis apa pun (C5).
func (s *Server) persisting(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		before, hadSession := s.sessionTenant(r.Cookies())
		buf := &bufferedResponse{header: http.Header{}}
		h(buf, r)
		after, newSession := s.sessionTenant((&http.Response{Header: buf.header}).Cookies())
		if buf.status < 400 && (hadSession || newSession) {
			if err := s.saveFor(cmp.Or(after, before)); err != nil {
				http.Error(w, "Gagal menyimpan data, coba lagi sebentar lagi.", http.StatusServiceUnavailable)
				return
			}
		}
		buf.flush(w)
	}
}
