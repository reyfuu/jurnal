package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"pajak/internal/auth"
)

// Regresi temuan security audit run-1 (~/security-audit-skill/pajak/run-1/REPORT.md).

// post mengirim POST anonim dengan header tambahan; kembalikan recorder.
func post(srv *Server, path, body string, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func loginFrom(srv *Server, ip, email, password string) int {
	return post(srv, "/api/v1/auth/login", `{"email":"`+email+`","password":"`+password+`"}`, map[string]string{"X-Real-IP": ip}).Code
}

// C2: permintaan login serentak tidak boleh lolos melebihi batas gagal.
func TestLimiterLoginSerentak(t *testing.T) {
	srv := NewServer()
	start := make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	codes := map[int]int{}
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			c := loginFrom(srv, "198.51.100.7", "owner@kedaikopisenja.id", "salah")
			mu.Lock()
			codes[c]++
			mu.Unlock()
		}()
	}
	close(start)
	wg.Wait()
	if codes[http.StatusUnauthorized] > maxLoginFailures {
		t.Fatalf("%d tebakan dievaluasi, batas %d: %v", codes[http.StatusUnauthorized], maxLoginFailures, codes)
	}
}

// C4: login sukses milik penyerang tidak boleh mengosongkan jatah gagal per IP.
func TestLoginSuksesTidakMeresetLimiter(t *testing.T) {
	srv := NewServer()
	ip := "198.51.100.8"
	for i := range maxLoginFailures - 1 {
		if c := loginFrom(srv, ip, "owner@kedaikopisenja.id", "salah"); c != http.StatusUnauthorized {
			t.Fatalf("tebakan %d: %d", i+1, c)
		}
	}
	if c := loginFrom(srv, ip, "kasir@kedaikopisenja.id", auth.DemoPassword); c != http.StatusOK {
		t.Fatalf("login sah: %d", c)
	}
	if c := loginFrom(srv, ip, "owner@kedaikopisenja.id", "salah"); c != http.StatusUnauthorized {
		t.Fatalf("tebakan ke-5: %d", c)
	}
	if c := loginFrom(srv, ip, "owner@kedaikopisenja.id", "salah"); c != http.StatusTooManyRequests {
		t.Fatalf("tebakan ke-6 harus 429 walau ada login sukses di tengah, got %d", c)
	}
}

// N6 / C4: pendaftaran dari satu IP dibatasi.
func TestDaftarDibatasiPerIP(t *testing.T) {
	srv := NewServer()
	for i := range maxLoginFailures + 1 {
		body := `{"name":"QA","email":"qa` + strconv.Itoa(i) + `@x.id","password":"rahasia123","wa":"081234567890","setuju_pdp":true}`
		c := post(srv, "/api/v1/auth/register", body, map[string]string{"X-Real-IP": "198.51.100.9"}).Code
		if want := http.StatusCreated; i == maxLoginFailures {
			if c != http.StatusTooManyRequests {
				t.Fatalf("daftar ke-%d dari IP sama harus 429, got %d", i+1, c)
			}
		} else if c != want {
			t.Fatalf("daftar ke-%d: %d", i+1, c)
		}
	}
}

// C8: form lintas situs tidak boleh login/daftar atau mengubah data.
func TestTolakPermintaanLintasSitus(t *testing.T) {
	srv := NewServer()
	login := `{"email":"owner@kedaikopisenja.id","password":"` + auth.DemoPassword + `"}`
	for _, tc := range []struct {
		name string
		path string
		hdr  map[string]string
		want int
	}{
		{"fetch metadata cross-site", "/api/v1/auth/login", map[string]string{"Sec-Fetch-Site": "cross-site", "Content-Type": "application/json"}, http.StatusForbidden},
		{"fetch metadata same-site", "/api/v1/auth/login", map[string]string{"Sec-Fetch-Site": "same-site", "Content-Type": "application/json"}, http.StatusForbidden},
		{"form text/plain", "/api/v1/auth/login", map[string]string{"Content-Type": "text/plain"}, http.StatusUnsupportedMediaType},
		{"form urlencoded daftar", "/api/v1/auth/register", map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, http.StatusUnsupportedMediaType},
		{"same-origin JSON", "/api/v1/auth/login", map[string]string{"Sec-Fetch-Site": "same-origin", "Content-Type": "application/json"}, http.StatusOK},
	} {
		if got := post(srv, tc.path, login, tc.hdr).Code; got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
	owner := login0(t, srv)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(`{"tanggal":"2026-08-20","jenis":"Penjualan","nominal":1,"lunas":true}`))
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.AddCookie(owner)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("tulis lintas situs dengan cookie korban harus 403, got %d", rec.Code)
	}
	get := httptest.NewRequest(http.MethodGet, "/api/v1/billing/plans", nil)
	get.Header.Set("Sec-Fetch-Site", "cross-site")
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, get)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET lintas situs tetap boleh, got %d", rec.Code)
	}
}

func login0(t *testing.T, srv *Server) *http.Cookie { return login(t, srv, "owner@kedaikopisenja.id") }

// C1: isi laporan bug dibatasi dan jumlahnya dikuota per usaha.
func TestLaporanBugDibatasi(t *testing.T) {
	srv := NewServer()
	kasir := login(t, srv, "kasir@kedaikopisenja.id")
	rec := postBug(srv, kasir, map[string]string{"judul": "a", "deskripsi": "b", "halaman": strings.Repeat("<", 5000)}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("lapor bug: %d", rec.Code)
	}
	if got := utf8.RuneCountInString(srv.bugs.list[0].Halaman); got > 200 {
		t.Fatalf("halaman disimpan %d rune, maks 200", got)
	}
	for i := 1; i < maxBugsPerDay; i++ {
		if rec := postBug(srv, kasir, map[string]string{"judul": "a", "deskripsi": "b"}, nil); rec.Code != http.StatusCreated {
			t.Fatalf("laporan ke-%d: %d", i+1, rec.Code)
		}
	}
	if rec := postBug(srv, kasir, map[string]string{"judul": "a", "deskripsi": "b"}, nil); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("laporan ke-%d dalam 24 jam harus 429, got %d", maxBugsPerDay+1, rec.Code)
	}
}

// C5: permintaan anonim tidak memicu penulisan state; body JSON dibatasi.
func TestAnonimTidakMenulisState(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DATA_DIR", dir)
	srv := NewServer()
	if err := os.Remove(filepath.Join(dir, "state.json")); err != nil {
		t.Fatal(err)
	}
	if rec := post(srv, "/api/v1/auth/logout", "", nil); rec.Code != http.StatusNoContent {
		t.Fatalf("logout: %d", rec.Code)
	}
	if _, err := os.Stat(filepath.Join(dir, "state.json")); !os.IsNotExist(err) {
		t.Fatal("logout tanpa sesi tidak boleh menulis state")
	}
	big := `{"tanggal":"2026-08-20","jenis":"Penjualan","nominal":1,"lunas":true,"keterangan":"` + strings.Repeat("x", 2<<20) + `"}`
	if rec := do(srv, login0(t, srv), http.MethodPost, "/api/v1/transactions", big); rec.Code < 400 {
		t.Fatalf("body JSON 2 MiB harus ditolak, got %d", rec.Code)
	}
}

// C9: kalau simpan gagal, klien harus tahu (503), bukan 201.
func TestSimpanGagalBukanSukses(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DATA_DIR", dir)
	srv := NewServer()
	owner := login0(t, srv)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })
	rec := do(srv, owner, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-08-20","jenis":"Penjualan","nominal":1,"lunas":true}`)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("simpan gagal harus 503, got %d %s", rec.Code, rec.Body)
	}
}

// C3: ID usaha yang masih dipakai user tidak boleh diberikan ke pendaftar baru.
func TestIDUsahaYatimTidakDipakaiUlang(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DATA_DIR", dir)
	srv := NewServer()
	register(t, srv, registerBody) // usaha 2
	if err := os.Remove(filepath.Join(dir, "state-2.json")); err != nil {
		t.Fatal(err)
	}
	srv2 := NewServer()
	b := register(t, srv2, `{"name":"Korban","email":"korban@x.id","password":"rahasia123","wa":"081234567891","setuju_pdp":true}`)
	var me meResp
	_ = json.NewDecoder(do(srv2, b, http.MethodGet, "/api/v1/auth/me", "").Body).Decode(&me)
	if me.User.TenantID == 2 {
		t.Fatal("pendaftar baru mendapat ID usaha yang masih dipakai user lama")
	}
}

// Hardening: respons API tidak boleh di-cache; nonaktifkan user mencabut sesinya.
func TestAPINoStoreDanNonaktifCabutSesi(t *testing.T) {
	srv := NewServer()
	owner := login0(t, srv)
	if cc := do(srv, owner, http.MethodGet, "/api/v1/auth/me", "").Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", cc)
	}
	rec := do(srv, owner, http.MethodPost, "/api/v1/users", `{"name":"Tono","email":"tono@x.id","role":"kasir","active":true,"password":"rahasia123"}`)
	var u auth.User
	_ = json.NewDecoder(rec.Body).Decode(&u)
	tono := post(srv, "/api/v1/auth/login", `{"email":"tono@x.id","password":"rahasia123"}`, nil).Result().Cookies()[0]
	path := "/api/v1/users/" + strconv.Itoa(u.ID)
	do(srv, owner, http.MethodPut, path, `{"name":"Tono","email":"tono@x.id","role":"kasir","active":false}`)
	do(srv, owner, http.MethodPut, path, `{"name":"Tono","email":"tono@x.id","role":"kasir","active":true}`)
	if c := do(srv, tono, http.MethodGet, "/api/v1/auth/me", "").Code; c != http.StatusUnauthorized {
		t.Fatalf("sesi lama setelah dinonaktifkan harus dicabut, got %d", c)
	}
	if rec := do(srv, owner, http.MethodGet, `/api/v1/reports/coa.xlsx?period=2026-08%22x%3Dy`, ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("period tidak valid harus 400, got %d %s", rec.Code, rec.Body)
	}
}
