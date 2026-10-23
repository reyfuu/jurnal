package main

import (
	"cmp"
	"encoding/json"
	"log"
	"maps"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"pajak/internal/primbon"
)

// Primbon Pajak: katalog peraturan resmi (internal/primbon) untuk usaha yang berlangganan,
// plus bookmark & catatan pribadi per pengguna. Katalog (index.csv + naskah, ±2,5 GB) dibaca
// dari REGULASI_DIR, di luar image Docker; lihat HANDOFF.md untuk cara memperbaruinya.

// maxCatatanPrimbon: panjang maksimal catatan pribadi per peraturan (karakter).
const maxCatatanPrimbon = 10000

type catatanPrimbon struct {
	Isi    string    `json:"isi"`
	Diubah time.Time `json:"diubah"`
}

// primbonPengguna: bookmark & catatan satu pengguna, per UUID peraturan.
type primbonPengguna struct {
	Bookmark map[string]time.Time      `json:"bookmark,omitempty"`
	Catatan  map[string]catatanPrimbon `json:"catatan,omitempty"`
}

// primbonStore: semua pengguna di memori; ditulis ke Postgres (per baris) atau primbon.json.
type primbonStore struct {
	mu    sync.Mutex
	path  string // primbon.json di DATA_DIR; kosong = Postgres atau memori saja
	users map[int]*primbonPengguna
}

// itemPrimbon: peraturan beserta status milik pengguna yang membukanya.
type itemPrimbon struct {
	primbon.Peraturan
	Bookmark   bool            `json:"bookmark"`
	AdaCatatan bool            `json:"ada_catatan"`
	Waktu      time.Time       `json:"waktu,omitzero"` // kapan di-bookmark
	Catatan    *catatanPrimbon `json:"catatan,omitempty"`
}

func (u primbonPengguna) item(p primbon.Peraturan, denganCatatan bool) itemPrimbon {
	it := itemPrimbon{Peraturan: p}
	it.Waktu, it.Bookmark = u.Bookmark[p.UUID]
	c, ada := u.Catatan[p.UUID]
	it.AdaCatatan = ada
	if ada && denganCatatan {
		it.Catatan = &c
	}
	return it
}

// loadPrimbon memuat bookmark & catatan. Data milik user yang sudah dihapus dibuang: ID user
// bisa dipakai lagi oleh pendaftar berikutnya setelah restart (nextID = ID terbesar).
func (s *Server) loadPrimbon() {
	p := &primbonStore{users: map[int]*primbonPengguna{}}
	s.primbonUser = p
	pengguna := func(id int) *primbonPengguna {
		if p.users[id] == nil {
			p.users[id] = &primbonPengguna{Bookmark: map[string]time.Time{}, Catatan: map[string]catatanPrimbon{}}
		}
		return p.users[id]
	}
	if s.db != nil {
		_, err := s.db.Exec(`DELETE FROM primbon_bookmark b WHERE NOT EXISTS (SELECT 1 FROM users u WHERE u.id = b.user_id);
			DELETE FROM primbon_catatan c WHERE NOT EXISTS (SELECT 1 FROM users u WHERE u.id = c.user_id)`)
		var bm []struct {
			UserID int       `json:"user_id"`
			UUID   string    `json:"uuid"`
			Waktu  time.Time `json:"waktu"`
		}
		var ct []struct {
			UserID int    `json:"user_id"`
			UUID   string `json:"uuid"`
			catatanPrimbon
		}
		if err == nil {
			err = selectJSON(s.db, &bm, `SELECT jsonb_agg(to_jsonb(b)) FROM primbon_bookmark b`)
		}
		if err == nil {
			err = selectJSON(s.db, &ct, `SELECT jsonb_agg(to_jsonb(c)) FROM primbon_catatan c`)
		}
		if err != nil {
			log.Fatalf("memuat Primbon dari Postgres: %v", err)
		}
		for _, b := range bm {
			pengguna(b.UserID).Bookmark[b.UUID] = b.Waktu
		}
		for _, c := range ct {
			pengguna(c.UserID).Catatan[c.UUID] = c.catatanPrimbon
		}
		return
	}
	if p.path = dataPath("primbon.json"); p.path == "" {
		return
	}
	if raw, err := os.ReadFile(p.path); err == nil {
		if err := json.Unmarshal(raw, &p.users); err != nil {
			log.Fatalf("primbon.json rusak, perbaiki atau pindahkan dulu: %v", err)
		}
	}
	n := len(p.users)
	maps.DeleteFunc(p.users, func(id int, _ *primbonPengguna) bool { _, ok := s.users.Get(id); return !ok })
	if len(p.users) < n {
		if err := writeFileAtomic(p.path, p.users); err != nil {
			log.Printf("GAGAL membersihkan primbon.json: %v", err)
		}
	}
}

// milik: salinan simpanan satu pengguna (aman dibaca tanpa kunci).
func (p *primbonStore) milik(uid int) primbonPengguna {
	p.mu.Lock()
	defer p.mu.Unlock()
	if u := p.users[uid]; u != nil {
		return primbonPengguna{Bookmark: maps.Clone(u.Bookmark), Catatan: maps.Clone(u.Catatan)}
	}
	return primbonPengguna{}
}

// ubahPrimbon menjalankan f atas simpanan pengguna lalu menyimpannya (query SQL bila Postgres,
// selain itu primbon.json). Gagal simpan = perubahan di memori dibatalkan.
func (s *Server) ubahPrimbon(uid int, f func(u *primbonPengguna), query string, args ...any) error {
	p := s.primbonUser
	p.mu.Lock()
	defer p.mu.Unlock()
	lama := p.users[uid]
	u := &primbonPengguna{Bookmark: map[string]time.Time{}, Catatan: map[string]catatanPrimbon{}}
	if lama != nil {
		maps.Copy(u.Bookmark, lama.Bookmark)
		maps.Copy(u.Catatan, lama.Catatan)
	}
	f(u)
	p.users[uid] = u
	var err error
	switch {
	case s.db != nil:
		_, err = s.db.Exec(query, args...)
	case p.path != "":
		err = writeFileAtomic(p.path, p.users)
	}
	if err != nil {
		log.Printf("GAGAL menyimpan Primbon user %d: %v", uid, err)
		if lama == nil {
			delete(p.users, uid)
		} else {
			p.users[uid] = lama
		}
	}
	return err
}

func (s *Server) primbonRoutes() {
	g := func(h http.HandlerFunc) http.HandlerFunc { return s.guard("primbon-pajak", s.wajibLangganan(h)) }
	m := s.mux
	m.HandleFunc("GET /api/v1/primbon/rag/status", g(s.handleRAGStatus))
	m.HandleFunc("GET /api/v1/primbon/rag/cari", g(s.handleRAGSearch))
	m.HandleFunc("POST /api/v1/primbon/rag/tanya", g(s.handleRAGQuestion))
	m.HandleFunc("GET /api/v1/primbon/peraturan", g(s.handlePrimbonCari))
	m.HandleFunc("GET /api/v1/primbon/peraturan/{uuid}", g(s.handlePrimbonDetail))
	m.HandleFunc("GET /api/v1/primbon/peraturan/{uuid}/berkas", g(s.handlePrimbonBerkas))
	m.HandleFunc("GET /api/v1/primbon/bookmark", g(s.handlePrimbonDaftar))
	m.HandleFunc("PUT /api/v1/primbon/bookmark/{uuid}", g(s.handlePrimbonBookmark))
	m.HandleFunc("DELETE /api/v1/primbon/bookmark/{uuid}", g(s.handlePrimbonBookmark))
	m.HandleFunc("GET /api/v1/primbon/catatan", g(s.handlePrimbonDaftar))
	m.HandleFunc("PUT /api/v1/primbon/catatan/{uuid}", g(s.handlePrimbonCatatan))
}

// wajibLangganan: fitur khusus pelanggan. lockedWrite sengaja tidak mengunci GET, jadi baca
// katalog dan naskah dikunci di sini.
func (s *Server) wajibLangganan(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if t := s.tenantOf(r); t != nil && t.langgananStatus(time.Now()).Status == "habis" {
			http.Error(w, "Primbon Pajak terbuka setelah berlangganan. Pilih paket di menu Langganan.", http.StatusPaymentRequired)
			return
		}
		h(w, r)
	}
}

func (s *Server) primbonData(w http.ResponseWriter) (*primbon.Data, bool) {
	d, err := s.primbon.Data()
	if err != nil {
		http.Error(w, "Katalog peraturan belum tersedia di server.", http.StatusServiceUnavailable)
		return nil, false
	}
	return d, true
}

func (s *Server) handlePrimbonCari(w http.ResponseWriter, r *http.Request) {
	d, ok := s.primbonData(w)
	if !ok {
		return
	}
	q := r.URL.Query()
	angka := func(k string) int { n, _ := strconv.Atoi(q.Get(k)); return n }
	h := d.Cari(primbon.Query{
		Q: clip(q.Get("q"), 200), Kategori: q["kategori"], Jenis: q["jenis"], Status: q["status"],
		Dari: angka("dari"), Sampai: angka("sampai"), Urut: q.Get("urut"), Halaman: angka("hal"),
	})
	u := s.primbonUser.milik(currentUser(r).ID)
	items := make([]itemPrimbon, len(h.Items))
	for i, p := range h.Items {
		items[i] = u.item(p, false)
	}
	writeJSON(w, struct {
		primbon.Hasil
		Items []itemPrimbon `json:"items"`
	}{h, items})
}

func (s *Server) handlePrimbonDetail(w http.ResponseWriter, r *http.Request) {
	d, ok := s.primbonData(w)
	if !ok {
		return
	}
	p, ok := d.Ambil(r.PathValue("uuid"))
	if !ok {
		http.Error(w, "peraturan tidak ditemukan", http.StatusNotFound)
		return
	}
	writeJSON(w, s.primbonUser.milik(currentUser(r).ID).item(p, true))
}

// handlePrimbonDaftar: bookmark (terbaru di-bookmark dulu) atau catatan (terbaru diubah dulu).
func (s *Server) handlePrimbonDaftar(w http.ResponseWriter, r *http.Request) {
	d, ok := s.primbonData(w)
	if !ok {
		return
	}
	u := s.primbonUser.milik(currentUser(r).ID)
	bookmark := strings.HasSuffix(r.URL.Path, "/bookmark")
	uuids := slices.Collect(maps.Keys(u.Catatan))
	if bookmark {
		uuids = slices.Collect(maps.Keys(u.Bookmark))
	}
	out := []itemPrimbon{}
	for _, id := range uuids {
		if p, ok := d.Ambil(id); ok { // peraturan yang keluar dari katalog tidak ditampilkan
			out = append(out, u.item(p, true))
		}
	}
	slices.SortFunc(out, func(a, b itemPrimbon) int {
		if bookmark {
			return b.Waktu.Compare(a.Waktu)
		}
		return b.Catatan.Diubah.Compare(a.Catatan.Diubah)
	})
	writeJSON(w, out)
}

var bukanHurufAngka = regexp.MustCompile(`[^A-Za-z0-9]+`)

// handlePrimbonBerkas menyajikan naskah resmi untuk iframe halaman detail atau diunduh.
func (s *Server) handlePrimbonBerkas(w http.ResponseWriter, r *http.Request) {
	d, ok := s.primbonData(w)
	if !ok {
		return
	}
	uuid, lampiran := r.PathValue("uuid"), r.URL.Query().Has("lampiran")
	path, ok := d.Berkas(uuid, lampiran)
	var f *os.File
	var err error
	if ok {
		f, err = os.Open(path)
	}
	if !ok || err != nil {
		http.Error(w, "Naskah peraturan ini belum tersedia di server.", http.StatusNotFound)
		return
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		http.Error(w, "Naskah peraturan ini belum tersedia di server.", http.StatusNotFound)
		return
	}
	p, _ := d.Ambil(uuid)
	ext := strings.ToLower(filepath.Ext(path))
	h := w.Header()
	h.Set("X-Frame-Options", "SAMEORIGIN")          // tampil di iframe halaman detail (global: DENY)
	h.Set("Cache-Control", "private, max-age=3600") // naskah besar & jarang berubah; hanya cache browser
	if ext == ".pdf" {
		// Tanpa sandbox: Chrome tidak menampilkan PDF di frame sandbox.
		h.Set("Content-Type", "application/pdf")
	} else {
		// Tanpa charset: naskah JDIH ada yang windows-1252, browser membacanya dari <meta>.
		// Sandbox: HTML pihak ketiga tidak boleh menjalankan skrip di origin aplikasi.
		h.Set("Content-Type", "text/html")
		h.Set("Content-Security-Policy", "sandbox")
	}
	nama := strings.Trim(bukanHurufAngka.ReplaceAllString(p.Kode, "-"), "-")
	if lampiran {
		nama += "-Lampiran"
	}
	disp := "inline"
	if r.URL.Query().Has("unduh") {
		disp = "attachment"
	}
	h.Set("Content-Disposition", mime.FormatMediaType(disp, map[string]string{"filename": cmp.Or(nama, "peraturan") + ext}))
	http.ServeContent(w, r, "", fi.ModTime(), f)
}

func waktuPrimbon() time.Time { return time.Now().UTC().Truncate(time.Microsecond) } // presisi Postgres

func (s *Server) handlePrimbonBookmark(w http.ResponseWriter, r *http.Request) {
	d, ok := s.primbonData(w)
	if !ok {
		return
	}
	uuid, uid := r.PathValue("uuid"), currentUser(r).ID
	var err error
	if r.Method == http.MethodDelete {
		err = s.ubahPrimbon(uid, func(u *primbonPengguna) { delete(u.Bookmark, uuid) },
			`DELETE FROM primbon_bookmark WHERE user_id = $1 AND uuid = $2`, uid, uuid)
	} else {
		if _, ok := d.Ambil(uuid); !ok {
			http.Error(w, "peraturan tidak ditemukan", http.StatusNotFound)
			return
		}
		now := waktuPrimbon()
		err = s.ubahPrimbon(uid, func(u *primbonPengguna) {
			if _, ada := u.Bookmark[uuid]; !ada {
				u.Bookmark[uuid] = now
			}
		}, `INSERT INTO primbon_bookmark (user_id, uuid, waktu) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, uid, uuid, now)
	}
	if err != nil {
		http.Error(w, "gagal menyimpan, coba lagi", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handlePrimbonCatatan menyimpan catatan pribadi; isi kosong = catatan dihapus (balasan null).
func (s *Server) handlePrimbonCatatan(w http.ResponseWriter, r *http.Request) {
	d, ok := s.primbonData(w)
	if !ok {
		return
	}
	uuid, uid := r.PathValue("uuid"), currentUser(r).ID
	if _, ok := d.Ambil(uuid); !ok {
		http.Error(w, "peraturan tidak ditemukan", http.StatusNotFound)
		return
	}
	var req struct {
		Isi string `json:"isi"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "catatan tidak valid", http.StatusBadRequest)
		return
	}
	isi := strings.TrimSpace(req.Isi)
	if utf8.RuneCountInString(isi) > maxCatatanPrimbon {
		http.Error(w, "catatan maksimal 10.000 karakter", http.StatusBadRequest)
		return
	}
	var c *catatanPrimbon
	var err error
	if isi == "" {
		err = s.ubahPrimbon(uid, func(u *primbonPengguna) { delete(u.Catatan, uuid) },
			`DELETE FROM primbon_catatan WHERE user_id = $1 AND uuid = $2`, uid, uuid)
	} else {
		c = &catatanPrimbon{Isi: isi, Diubah: waktuPrimbon()}
		err = s.ubahPrimbon(uid, func(u *primbonPengguna) { u.Catatan[uuid] = *c },
			`INSERT INTO primbon_catatan (user_id, uuid, isi, diubah) VALUES ($1, $2, $3, $4)
			 ON CONFLICT (user_id, uuid) DO UPDATE SET isi = EXCLUDED.isi, diubah = EXCLUDED.diubah`, uid, uuid, c.Isi, c.Diubah)
	}
	if err != nil {
		http.Error(w, "gagal menyimpan, coba lagi", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, c)
}
