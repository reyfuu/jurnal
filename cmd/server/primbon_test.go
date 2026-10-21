package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"pajak/internal/primbon"
)

// primbonServer: server dengan katalog contoh internal/primbon/testdata.
func primbonServer(t *testing.T) *Server {
	t.Helper()
	t.Setenv("REGULASI_DIR", filepath.Join("..", "..", "internal", "primbon", "testdata", "katalog"))
	return NewServer()
}

type primbonHasil struct {
	Items []itemPrimbon `json:"items"`
	Total int           `json:"total"`
	Facet primbon.Facet `json:"facet"`
}

func uuidPrimbon(items []itemPrimbon) []string {
	var out []string
	for _, p := range items {
		out = append(out, p.UUID)
	}
	return out
}

func cariPrimbon(t *testing.T, srv *Server, c *http.Cookie, query string) primbonHasil {
	t.Helper()
	rec := do(srv, c, http.MethodGet, "/api/v1/primbon/peraturan?"+query, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("cari %q: %d %s", query, rec.Code, rec.Body)
	}
	var h primbonHasil
	if err := json.NewDecoder(rec.Body).Decode(&h); err != nil {
		t.Fatal(err)
	}
	return h
}

func TestPrimbonHanyaPelangganBerbayar(t *testing.T) {
	srv := primbonServer(t)
	if rec := do(srv, login(t, srv, "kasir@kedaikopisenja.id"), http.MethodGet, "/api/v1/primbon/peraturan", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("kasir: want 403, got %d", rec.Code)
	}
	baru := register(t, srv, registerBody) // usaha baru belum berlangganan
	for _, r := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/primbon/peraturan", ""},
		{http.MethodGet, "/api/v1/primbon/peraturan/a1", ""},
		{http.MethodGet, "/api/v1/primbon/peraturan/a1/berkas", ""},
		{http.MethodGet, "/api/v1/primbon/bookmark", ""},
		{http.MethodGet, "/api/v1/primbon/catatan", ""},
		{http.MethodPut, "/api/v1/primbon/bookmark/a1", ""},
		{http.MethodPut, "/api/v1/primbon/catatan/a1", `{"isi":"x"}`},
	} {
		if rec := do(srv, baru, r.method, r.path, r.body); rec.Code != http.StatusPaymentRequired {
			t.Errorf("belum berlangganan %s %s: want 402, got %d", r.method, r.path, rec.Code)
		}
	}
	if h := cariPrimbon(t, srv, login(t, srv, "admin@kedaikopisenja.id"), ""); h.Total != 8 {
		t.Fatalf("admin usaha aktif: total %d", h.Total)
	}
}

func TestPrimbonCariDanDetail(t *testing.T) {
	srv := primbonServer(t)
	owner := login(t, srv, "owner@kedaikopisenja.id")
	cases := []struct {
		query string
		want  []string
	}{
		{"kategori=PPN&kategori=KURS", []string{"a7", "a2"}},
		{"q=faktur&status=Diubah", []string{"a2"}},
		{"jenis=PMK&dari=2024&sampai=2024&urut=abjad", []string{"a5", "a1"}},
		{"kategori=PPh&urut=terlama&hal=1", []string{"a3", "a6", "a1"}},
	}
	for _, c := range cases {
		if got := uuidPrimbon(cariPrimbon(t, srv, owner, c.query).Items); !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.query, got, c.want)
		}
	}
	if f := cariPrimbon(t, srv, owner, "kategori=KURS").Facet; len(f.Kategori) == 0 || f.TahunMax != 2026 {
		t.Errorf("facet ikut dikirim: %+v", f)
	}
	var p itemPrimbon
	rec := do(srv, owner, http.MethodGet, "/api/v1/primbon/peraturan/a2", "")
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil || p.Format != "htm" || !p.Lampiran || p.Kode != "PER-03/PJ/2022" {
		t.Fatalf("detail a2: %d %+v %v", rec.Code, p, err)
	}
	if rec := do(srv, owner, http.MethodGet, "/api/v1/primbon/peraturan/tidak-ada", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("uuid tak dikenal: want 404, got %d", rec.Code)
	}
}

func TestPrimbonBerkasNaskahResmi(t *testing.T) {
	srv := primbonServer(t)
	owner := login(t, srv, "owner@kedaikopisenja.id")

	rec := do(srv, owner, http.MethodGet, "/api/v1/primbon/peraturan/a1/berkas", "")
	h := rec.Header()
	if rec.Code != http.StatusOK || h.Get("Content-Type") != "application/pdf" || !strings.HasPrefix(rec.Body.String(), "%PDF") {
		t.Fatalf("PDF: %d %q", rec.Code, h.Get("Content-Type"))
	}
	if h.Get("X-Frame-Options") != "SAMEORIGIN" || h.Get("Content-Security-Policy") != "" {
		t.Errorf("PDF harus bisa tampil di iframe aplikasi, tanpa sandbox: %q %q", h.Get("X-Frame-Options"), h.Get("Content-Security-Policy"))
	}
	if cd := h.Get("Content-Disposition"); !strings.HasPrefix(cd, "inline") || !strings.Contains(cd, "PMK-Nomor-10-Tahun-2024.pdf") {
		t.Errorf("tampil di halaman dengan nama ramah: %q", cd)
	}
	if cd := do(srv, owner, http.MethodGet, "/api/v1/primbon/peraturan/a1/berkas?unduh=1", "").Header().Get("Content-Disposition"); !strings.HasPrefix(cd, "attachment") {
		t.Errorf("unduh: %q", cd)
	}

	rec = do(srv, owner, http.MethodGet, "/api/v1/primbon/peraturan/a2/berkas", "")
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "text/html" || !strings.Contains(rec.Header().Get("Content-Security-Policy"), "sandbox") {
		t.Fatalf("HTML JDIH: %d %q %q", rec.Code, rec.Header().Get("Content-Type"), rec.Header().Get("Content-Security-Policy"))
	}
	if rec := do(srv, owner, http.MethodGet, "/api/v1/primbon/peraturan/a2/berkas?lampiran=1", ""); rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/pdf" {
		t.Fatalf("lampiran: %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	for _, uuid := range []string{"a3", "a4", "a5", "tidak-ada"} { // belum diunduh, tanpa-pdf, ../, tak dikenal
		if rec := do(srv, owner, http.MethodGet, "/api/v1/primbon/peraturan/"+uuid+"/berkas", ""); rec.Code != http.StatusNotFound {
			t.Errorf("%s: want 404, got %d", uuid, rec.Code)
		}
	}
	if xfo := do(srv, owner, http.MethodGet, "/api/v1/primbon/peraturan", "").Header().Get("X-Frame-Options"); xfo != "DENY" {
		t.Errorf("rute lain tetap DENY: %q", xfo)
	}
}

func TestPrimbonKatalogBelumAda(t *testing.T) {
	t.Setenv("REGULASI_DIR", t.TempDir())
	srv := NewServer()
	if rec := do(srv, login(t, srv, "owner@kedaikopisenja.id"), http.MethodGet, "/api/v1/primbon/peraturan", ""); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("katalog belum diunggah: want 503, got %d", rec.Code)
	}
}

func TestPrimbonBookmarkCatatanPerPengguna(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := primbonServer(t)
	owner, admin := login(t, srv, "owner@kedaikopisenja.id"), login(t, srv, "admin@kedaikopisenja.id")

	if rec := do(srv, owner, http.MethodPut, "/api/v1/primbon/bookmark/a1", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("bookmark: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, owner, http.MethodPut, "/api/v1/primbon/catatan/a2", `{"isi":"  cek lampiran tabel  "}`); rec.Code != http.StatusOK {
		t.Fatalf("catatan: %d %s", rec.Code, rec.Body)
	}
	for _, r := range []struct{ path, body string }{{"/api/v1/primbon/bookmark/zz", ""}, {"/api/v1/primbon/catatan/zz", `{"isi":"x"}`}} {
		if rec := do(srv, owner, http.MethodPut, r.path, r.body); rec.Code != http.StatusNotFound {
			t.Errorf("%s: want 404, got %d", r.path, rec.Code)
		}
	}
	if rec := do(srv, owner, http.MethodPut, "/api/v1/primbon/catatan/a1", `{"isi":"`+strings.Repeat("a", 10001)+`"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("catatan terlalu panjang: want 400, got %d", rec.Code)
	}

	cek := func(srv *Server, c *http.Cookie, siapa string, adaBookmark, adaCatatan bool) {
		t.Helper()
		h := cariPrimbon(t, srv, c, "")
		i := slices.IndexFunc(h.Items, func(p itemPrimbon) bool { return p.UUID == "a1" })
		j := slices.IndexFunc(h.Items, func(p itemPrimbon) bool { return p.UUID == "a2" })
		if h.Items[i].Bookmark != adaBookmark || h.Items[j].AdaCatatan != adaCatatan {
			t.Errorf("%s: bookmark a1 %v, catatan a2 %v", siapa, h.Items[i].Bookmark, h.Items[j].AdaCatatan)
		}
		var bm, ct []itemPrimbon
		_ = json.NewDecoder(do(srv, c, http.MethodGet, "/api/v1/primbon/bookmark", "").Body).Decode(&bm)
		_ = json.NewDecoder(do(srv, c, http.MethodGet, "/api/v1/primbon/catatan", "").Body).Decode(&ct)
		if (len(bm) == 1) != adaBookmark || (len(ct) == 1) != adaCatatan {
			t.Errorf("%s: daftar bookmark %v, catatan %v", siapa, uuidPrimbon(bm), uuidPrimbon(ct))
		}
		if adaCatatan && (ct[0].Catatan == nil || ct[0].Catatan.Isi != "cek lampiran tabel" || ct[0].Catatan.Diubah.IsZero()) {
			t.Errorf("%s: isi catatan %+v", siapa, ct[0].Catatan)
		}
	}
	cek(srv, owner, "owner", true, true)
	cek(srv, admin, "admin usaha yang sama", false, false)

	srv2 := primbonServer(t) // bertahan setelah restart (DATA_DIR)
	owner2 := login(t, srv2, "owner@kedaikopisenja.id")
	cek(srv2, owner2, "owner setelah restart", true, true)

	if rec := do(srv2, owner2, http.MethodDelete, "/api/v1/primbon/bookmark/a1", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("hapus bookmark: %d", rec.Code)
	}
	if rec := do(srv2, owner2, http.MethodPut, "/api/v1/primbon/catatan/a2", `{"isi":"   "}`); rec.Code != http.StatusOK {
		t.Fatalf("kosongkan catatan: %d", rec.Code)
	}
	srv3 := primbonServer(t)
	cek(srv3, login(t, srv3, "owner@kedaikopisenja.id"), "owner setelah hapus", false, false)
}

func TestPrimbonDataPenggunaTerhapusDibuang(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := primbonServer(t)
	admin := login(t, srv, "admin@kedaikopisenja.id")
	if rec := do(srv, admin, http.MethodPut, "/api/v1/primbon/bookmark/a1", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("bookmark admin: %d", rec.Code)
	}
	var me meResp
	_ = json.NewDecoder(do(srv, admin, http.MethodGet, "/api/v1/auth/me", "").Body).Decode(&me)
	if rec := do(srv, login(t, srv, "owner@kedaikopisenja.id"), http.MethodDelete, "/api/v1/users/"+strconv.Itoa(me.User.ID), ""); rec.Code != http.StatusNoContent {
		t.Fatalf("hapus admin: %d %s", rec.Code, rec.Body)
	}
	// ID user bisa dipakai ulang setelah restart, jadi data Primbon milik user yang sudah
	// tidak ada dibuang saat boot.
	srv2 := primbonServer(t)
	if _, ok := srv2.primbonUser.users[me.User.ID]; ok {
		t.Fatal("data Primbon user terhapus masih termuat")
	}
}

func TestPostgresPrimbon(t *testing.T) {
	pgTest(t)
	srv := primbonServer(t)
	owner, admin := login(t, srv, "owner@kedaikopisenja.id"), login(t, srv, "admin@kedaikopisenja.id")
	for _, r := range []struct {
		c          *http.Cookie
		path, body string
	}{
		{owner, "/api/v1/primbon/bookmark/a1", ""},
		{owner, "/api/v1/primbon/bookmark/a1", ""}, // bookmark ulang: tetap satu baris
		{owner, "/api/v1/primbon/catatan/a2", `{"isi":"draf"}`},
		{owner, "/api/v1/primbon/catatan/a2", `{"isi":"cek lampiran"}`}, // upsert
		{admin, "/api/v1/primbon/bookmark/a2", ""},
	} {
		if rec := do(srv, r.c, http.MethodPut, r.path, r.body); rec.Code >= 300 {
			t.Fatalf("PUT %s: %d %s", r.path, rec.Code, rec.Body)
		}
	}
	var me meResp
	_ = json.NewDecoder(do(srv, admin, http.MethodGet, "/api/v1/auth/me", "").Body).Decode(&me)
	if rec := do(srv, owner, http.MethodDelete, "/api/v1/users/"+strconv.Itoa(me.User.ID), ""); rec.Code != http.StatusNoContent {
		t.Fatalf("hapus admin: %d", rec.Code)
	}

	srv2 := primbonServer(t)
	owner2 := login(t, srv2, "owner@kedaikopisenja.id")
	var bm, ct []itemPrimbon
	_ = json.NewDecoder(do(srv2, owner2, http.MethodGet, "/api/v1/primbon/bookmark", "").Body).Decode(&bm)
	_ = json.NewDecoder(do(srv2, owner2, http.MethodGet, "/api/v1/primbon/catatan", "").Body).Decode(&ct)
	if !slices.Equal(uuidPrimbon(bm), []string{"a1"}) || bm[0].Waktu.IsZero() || len(ct) != 1 || ct[0].Catatan.Isi != "cek lampiran" {
		t.Fatalf("setelah restart: bookmark %+v, catatan %+v", bm, ct)
	}
	if _, ok := srv2.primbonUser.users[me.User.ID]; ok {
		t.Fatal("baris Primbon user terhapus masih termuat dari Postgres")
	}
}
