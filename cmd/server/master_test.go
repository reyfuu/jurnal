package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"pajak/internal/auth"
	"pajak/internal/master"
)

const masterPass = "rahasia-master-1"

// serverMaster: server dengan Master pertama dari env, plus sesi login Master.
func serverMaster(t *testing.T) (*Server, *http.Cookie) {
	t.Helper()
	t.Setenv("MASTER_EMAIL", "andi@masterpajak.id")
	t.Setenv("MASTER_PASSWORD", masterPass)
	srv := NewServer()
	return srv, loginPass(t, srv, "andi@masterpajak.id", masterPass)
}

func loginPass(t *testing.T, srv *Server, email, pass string) *http.Cookie {
	t.Helper()
	rec := do(srv, nil, http.MethodPost, "/api/v1/auth/login", `{"email":"`+email+`","password":"`+pass+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("login %s: %d %s", email, rec.Code, rec.Body)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookie {
			return c
		}
	}
	t.Fatal("tanpa cookie sesi")
	return nil
}

func getJSON[T any](t *testing.T, srv *Server, c *http.Cookie, path string) T {
	t.Helper()
	rec := do(srv, c, http.MethodGet, path, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: %d %s", path, rec.Code, rec.Body)
	}
	var v T
	if err := json.NewDecoder(rec.Body).Decode(&v); err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return v
}

// tambahAnggota: Master menambah anggota tim dan mengembalikan sesi loginnya.
func tambahAnggota(t *testing.T, srv *Server, m *http.Cookie, email, role string, klien []int) *http.Cookie {
	t.Helper()
	k, _ := json.Marshal(klien)
	body := fmt.Sprintf(`{"nama":"Anggota %s","email":%q,"password":"rahasia123","role":%q,"klien":%s}`, role, email, role, k)
	if rec := do(srv, m, http.MethodPost, "/api/v1/master/tim", body); rec.Code != http.StatusCreated {
		t.Fatalf("tambah anggota %s: %d %s", email, rec.Code, rec.Body)
	}
	return loginPass(t, srv, email, "rahasia123")
}

func TestMaster_BootstrapDariEnv(t *testing.T) {
	srv, m := serverMaster(t)
	me := getJSON[meResp](t, srv, m, "/api/v1/auth/me")
	if me.User.Role != auth.MP || me.User.TenantID != 0 || len(me.Views) != 1 || me.Views[0] != "panel-master" {
		t.Fatalf("me master = %+v", me)
	}
	if me.Master == nil || me.Master.Role.ID != master.RoleMaster || len(me.Master.Izin) != len(master.DaftarIzin) {
		t.Fatalf("info master = %+v", me.Master)
	}
	if n := len(srv.users.List(0)); n != 1 {
		t.Errorf("akun tim = %d", n)
	}
}

func TestMaster_TanpaPasswordTidakDibuat(t *testing.T) {
	t.Setenv("MASTER_EMAIL", "andi@masterpajak.id")
	t.Setenv("MASTER_PASSWORD", "")
	srv := NewServer()
	if _, ok := srv.users.FindByEmail("andi@masterpajak.id"); ok {
		t.Error("tanpa MASTER_PASSWORD akun master tidak boleh dibuat")
	}
}

// Akun tim tidak punya usaha: semua endpoint data usaha tertolak, termasuk yang cukup login.
func TestMaster_AkunTimTidakBisaBukaDataUsaha(t *testing.T) {
	srv, m := serverMaster(t)
	for _, p := range []string{"/api/v1/dashboard", "/api/v1/business", "/api/v1/bugs", "/api/v1/coa", "/api/v1/billing", "/api/v1/periods"} {
		if rec := do(srv, m, http.MethodGet, p, ""); rec.Code != http.StatusForbidden {
			t.Errorf("GET %s oleh akun tim: want 403, got %d", p, rec.Code)
		}
	}
	if rec := do(srv, m, http.MethodPost, "/api/v1/ai/chat", `{"question":"halo"}`); rec.Code != http.StatusForbidden {
		t.Errorf("ai chat oleh akun tim: %d", rec.Code)
	}
}

func TestMaster_UserUsahaTidakBisaBukaPanel(t *testing.T) {
	srv, _ := serverMaster(t)
	owner := login(t, srv, "owner@kedaikopisenja.id")
	for _, p := range []string{"/api/v1/master/ringkasan", "/api/v1/master/klien", "/api/v1/master/tiket", "/api/v1/master/audit"} {
		if rec := do(srv, owner, http.MethodGet, p, ""); rec.Code != http.StatusForbidden {
			t.Errorf("GET %s oleh owner: want 403, got %d", p, rec.Code)
		}
	}
	// Owner tidak bisa membuat akun tim lewat menu Pengguna.
	rec := do(srv, owner, http.MethodPost, "/api/v1/users", `{"name":"Susup","email":"susup@x.id","role":"mp","active":true,"password":"rahasia123"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("owner membuat user role mp: want 400, got %d", rec.Code)
	}
}

func TestMaster_PenugasanMembatasiKlien(t *testing.T) {
	srv, m := serverMaster(t)
	register(t, srv, registerBody) // usaha 2
	k := tambahAnggota(t, srv, m, "budi@masterpajak.id", "konsultan", []int{2})

	semua := getJSON[[]klienRow](t, srv, m, "/api/v1/master/klien")
	if len(semua) != 2 {
		t.Fatalf("master melihat %d klien, want 2", len(semua))
	}
	milik := getJSON[[]klienRow](t, srv, k, "/api/v1/master/klien")
	if len(milik) != 1 || milik[0].ID != 2 {
		t.Fatalf("konsultan melihat %+v", milik)
	}
	if milik[0].OwnerEmail != "qa1@x.id" || milik[0].Status != "habis" || milik[0].TerakhirAktif.IsZero() {
		t.Errorf("baris klien = %+v", milik[0])
	}
	if len(milik[0].Penangan) != 1 || milik[0].Penangan[0].Nama != "Anggota konsultan" {
		t.Errorf("penangan = %+v", milik[0].Penangan)
	}
	// Usaha tanpa pembayaran: orders [] (bukan null), supaya layar detail tidak gagal.
	if body := do(srv, k, http.MethodGet, "/api/v1/master/klien/2", "").Body.String(); !strings.Contains(body, `"orders":[]`) || !strings.Contains(body, `"tiket":[]`) {
		t.Errorf("detail klien tanpa order: %s", body)
	}
	if rec := do(srv, k, http.MethodGet, "/api/v1/master/klien/1", ""); rec.Code != http.StatusNotFound {
		t.Errorf("detail klien yang tidak ditugaskan: want 404, got %d", rec.Code)
	}
	for _, c := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/master/audit", ""},
		{http.MethodGet, "/api/v1/master/pembayaran", ""},
		{http.MethodPost, "/api/v1/master/tim", `{"nama":"X","email":"x@mp.id","password":"rahasia123","role":"cs"}`},
		{http.MethodPost, "/api/v1/master/klien/2/langganan", `{"aksi":"trial","hari":7,"alasan":"x"}`},
	} {
		if rec := do(srv, k, c.method, c.path, c.body); rec.Code != http.StatusForbidden {
			t.Errorf("%s %s oleh konsultan: want 403, got %d", c.method, c.path, rec.Code)
		}
	}
	ring := getJSON[ringkasanMaster](t, srv, k, "/api/v1/master/ringkasan")
	if ring.Klien != 1 || ring.MRR != nil || ring.BelumDitugaskan != nil {
		t.Errorf("ringkasan konsultan = %+v", ring)
	}
	ringM := getJSON[ringkasanMaster](t, srv, m, "/api/v1/master/ringkasan")
	if ringM.Klien != 2 || ringM.MRR == nil || len(ringM.BelumDitugaskan) != 1 || ringM.BelumDitugaskan[0].ID != 1 {
		t.Errorf("ringkasan master = %+v", ringM)
	}
}

func TestMaster_TiketBalasanSampaiKePelapor(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv, m := serverMaster(t)
	kasir := login(t, srv, "kasir@kedaikopisenja.id")
	postBug(srv, kasir, map[string]string{"judul": "Usaha satu", "deskripsi": "Keterangan cukup"}, nil)
	qa := register(t, srv, registerBody)
	postBug(srv, qa, map[string]string{"judul": "Unggah mutasi gagal", "deskripsi": "Kolom tanggal tidak ditemukan"}, png1x1)
	cs := tambahAnggota(t, srv, m, "rina@masterpajak.id", "cs", []int{2})

	if got := getJSON[[]tiketRow](t, srv, m, "/api/v1/master/tiket"); len(got) != 2 {
		t.Fatalf("master melihat %d tiket", len(got))
	}
	got := getJSON[[]tiketRow](t, srv, cs, "/api/v1/master/tiket")
	if len(got) != 1 || got[0].ID != 2 || got[0].Nomor != "BUG-0002" || got[0].Status != "baru" || got[0].KlienNama == "" {
		t.Fatalf("tiket cs = %+v", got)
	}
	if rec := do(srv, cs, http.MethodGet, "/api/v1/master/tiket/2/screenshot", ""); rec.Code != http.StatusOK {
		t.Errorf("screenshot tiket klien sendiri: %d", rec.Code)
	}
	if rec := do(srv, cs, http.MethodPut, "/api/v1/master/tiket/1", `{"status":"selesai"}`); rec.Code != http.StatusNotFound {
		t.Errorf("ubah tiket klien lain: want 404, got %d", rec.Code)
	}
	if rec := do(srv, cs, http.MethodPut, "/api/v1/master/tiket/2", `{"status":"ditutup"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("status tak dikenal: want 400, got %d", rec.Code)
	}
	if rec := do(srv, cs, http.MethodPut, "/api/v1/master/tiket/2", `{"status":"dikerjakan","balasan":"Sedang kami cek dengan contoh file Anda."}`); rec.Code != http.StatusOK {
		t.Fatalf("balas tiket: %d %s", rec.Code, rec.Body)
	}
	var milikQA []bugReport
	_ = json.NewDecoder(do(srv, qa, http.MethodGet, "/api/v1/bugs", "").Body).Decode(&milikQA)
	if len(milikQA) != 1 || milikQA[0].Status != "dikerjakan" || len(milikQA[0].Balasan) != 1 || milikQA[0].Balasan[0].Isi != "Sedang kami cek dengan contoh file Anda." {
		t.Fatalf("pelapor melihat %+v", milikQA)
	}
	if b := milikQA[0].Balasan[0]; b.Nama != "Anggota cs" || b.Waktu.IsZero() {
		t.Errorf("balasan = %+v", b)
	}

	// Tersimpan: server baru dari DATA_DIR yang sama membaca status, balasan, tim & audit.
	srv2 := NewServer()
	cs2 := loginPass(t, srv2, "rina@masterpajak.id", "rahasia123")
	got2 := getJSON[[]tiketRow](t, srv2, cs2, "/api/v1/master/tiket")
	if len(got2) != 1 || got2[0].Status != "dikerjakan" || len(got2[0].Balasan) != 1 {
		t.Fatalf("setelah restart: %+v", got2)
	}
	m2 := loginPass(t, srv2, "andi@masterpajak.id", masterPass)
	audit := getJSON[[]master.Catatan](t, srv2, m2, "/api/v1/master/audit")
	if len(audit) < 2 || !strings.Contains(audit[0].Aksi, "BUG-0002") || audit[0].Nama != "Anggota cs" {
		t.Fatalf("audit setelah restart = %+v", audit)
	}
}

func TestMaster_AturLangganan(t *testing.T) {
	srv, m := serverMaster(t)
	qa := register(t, srv, registerBody)
	if rec := do(srv, m, http.MethodPost, "/api/v1/master/klien/2/langganan", `{"aksi":"aktifkan","paket":"bulanan"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("tanpa alasan: want 400, got %d", rec.Code)
	}
	if rec := do(srv, m, http.MethodPost, "/api/v1/master/klien/2/langganan", `{"aksi":"aktifkan","paket":"bulanan","alasan":"Bayar transfer BCA 20 Okt"}`); rec.Code != http.StatusOK {
		t.Fatalf("aktifkan manual: %d %s", rec.Code, rec.Body)
	}
	me := getJSON[meResp](t, srv, qa, "/api/v1/auth/me")
	if me.Langganan.Status != "aktif" || time.Until(me.Langganan.Sampai) < 29*24*time.Hour {
		t.Fatalf("langganan qa = %+v", me.Langganan)
	}
	if me.Garansi.Berlaku {
		t.Error("aktivasi manual tidak boleh membuka garansi Midtrans")
	}
	if rec := do(srv, m, http.MethodPost, "/api/v1/master/klien/2/langganan", `{"aksi":"trial","hari":7,"alasan":"Minta waktu input stok"}`); rec.Code != http.StatusOK {
		t.Fatalf("perpanjang trial: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, m, http.MethodPost, "/api/v1/master/klien/2/langganan", `{"aksi":"trial","hari":400,"alasan":"x"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("trial 400 hari: want 400, got %d", rec.Code)
	}
	audit := getJSON[[]master.Catatan](t, srv, m, "/api/v1/master/audit")
	if len(audit) < 2 || audit[1].Keterangan != "Bayar transfer BCA 20 Okt" || audit[1].KlienID != 2 {
		t.Errorf("audit langganan = %+v", audit)
	}
	pay := getJSON[[]pembayaranRow](t, srv, m, "/api/v1/master/pembayaran")
	if len(pay) != 1 || pay[0].Order.Status != "manual" || pay[0].KlienID != 2 {
		t.Errorf("pembayaran = %+v", pay)
	}
}

func TestMaster_RoleBaruDanMasterTerakhir(t *testing.T) {
	srv, m := serverMaster(t)
	rec := do(srv, m, http.MethodPost, "/api/v1/master/roles", `{"nama":"Akuntan Senior","izin":["klien_lihat","tiket","langganan"]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("role baru: %d %s", rec.Code, rec.Body)
	}
	var r master.Role
	_ = json.NewDecoder(rec.Body).Decode(&r)
	register(t, srv, registerBody)
	a := tambahAnggota(t, srv, m, "sari@masterpajak.id", r.ID, []int{2})
	if rec := do(srv, a, http.MethodPost, "/api/v1/master/klien/2/langganan", `{"aksi":"trial","hari":3,"alasan":"Kompensasi gangguan"}`); rec.Code != http.StatusOK {
		t.Errorf("role baru dengan izin langganan: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, m, http.MethodDelete, "/api/v1/master/roles/"+r.ID, ""); rec.Code != http.StatusConflict {
		t.Errorf("hapus role yang dipakai: want 409, got %d", rec.Code)
	}
	if rec := do(srv, m, http.MethodPut, "/api/v1/master/roles/master", `{"nama":"Master","izin":[]}`); rec.Code != http.StatusBadRequest {
		t.Errorf("ubah role master: want 400, got %d", rec.Code)
	}
	andi, _ := srv.users.FindByEmail("andi@masterpajak.id")
	if rec := do(srv, m, http.MethodPut, fmt.Sprintf("/api/v1/master/tim/%d", andi.ID), `{"role":"cs","klien":[],"aktif":true}`); rec.Code != http.StatusConflict {
		t.Errorf("turunkan master terakhir: want 409, got %d", rec.Code)
	}
	tim := getJSON[timResp](t, srv, m, "/api/v1/master/tim")
	if len(tim.Anggota) != 2 || len(tim.Roles) != 4 || len(tim.Izin) != len(master.DaftarIzin) || len(tim.Klien) != 2 {
		t.Errorf("tim = %d anggota, %d role, %d izin, %d klien", len(tim.Anggota), len(tim.Roles), len(tim.Izin), len(tim.Klien))
	}
	// Anggota dinonaktifkan: sesinya mati.
	sari, _ := srv.users.FindByEmail("sari@masterpajak.id")
	if rec := do(srv, m, http.MethodPut, fmt.Sprintf("/api/v1/master/tim/%d", sari.ID), `{"role":"`+r.ID+`","klien":[2],"aktif":false}`); rec.Code != http.StatusOK {
		t.Fatalf("nonaktifkan anggota: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, a, http.MethodGet, "/api/v1/master/klien", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("sesi anggota nonaktif: want 401, got %d", rec.Code)
	}
}

func TestMaster_LastSeenTercatat(t *testing.T) {
	srv := NewServer()
	before := time.Now().Add(-time.Second)
	c := login(t, srv, "kasir@kedaikopisenja.id")
	do(srv, c, http.MethodGet, "/api/v1/dashboard", "")
	u, _ := srv.users.FindByEmail("kasir@kedaikopisenja.id")
	if u.LastSeen.Before(before.Truncate(time.Second)) {
		t.Errorf("last_seen = %v", u.LastSeen)
	}
}

func slicesAny[T any](s []T, f func(T) bool) bool {
	for _, x := range s {
		if f(x) {
			return true
		}
	}
	return false
}
