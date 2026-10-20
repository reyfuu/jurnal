package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"pajak/internal/auth"
)

func TestKelolaKlien_UbahArsipPulihkanHapusPermanen(t *testing.T) {
	srv, m := serverMaster(t)
	qa := register(t, srv, registerBody) // usaha 2
	do(srv, qa, http.MethodPut, "/api/v1/business", profilDagang)

	if rec := do(srv, m, http.MethodPut, "/api/v1/master/klien/2", `{"nama_usaha":"Toko QA Baru","kota":"","provinsi":"","telepon":"0274-1"}`); rec.Code != http.StatusOK {
		t.Fatalf("ubah klien: %d %s", rec.Code, rec.Body)
	}
	if p := srv.tenantByID(2).currentProfile(); p.NamaUsaha != "Toko QA Baru" || p.Telepon != "0274-1" || p.JenisUsaha != "perdagangan" {
		t.Fatalf("profil setelah diubah Master: %+v", p)
	}

	// hapus permanen ditolak sebelum diarsipkan
	if rec := do(srv, m, http.MethodDelete, "/api/v1/master/klien/2", `{"konfirmasi":"Toko QA Baru"}`); rec.Code != http.StatusConflict {
		t.Errorf("hapus sebelum arsip: %d", rec.Code)
	}
	if rec := do(srv, m, http.MethodPost, "/api/v1/master/klien/2/arsip", `{"alasan":""}`); rec.Code != http.StatusBadRequest {
		t.Errorf("arsip tanpa alasan: %d", rec.Code)
	}
	if rec := do(srv, m, http.MethodPost, "/api/v1/master/klien/2/arsip", `{"alasan":"permintaan pemilik"}`); rec.Code != http.StatusNoContent {
		t.Fatalf("arsip: %d %s", rec.Code, rec.Body)
	}
	// diarsipkan: sesi & login pemilik ditolak
	if rec := do(srv, qa, http.MethodGet, "/api/v1/business", ""); rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "diarsipkan") {
		t.Errorf("sesi usaha diarsip: %d %s", rec.Code, rec.Body)
	}
	if rec := do(srv, nil, http.MethodPost, "/api/v1/auth/login", `{"email":"qa1@x.id","password":"rahasia123"}`); rec.Code != http.StatusForbidden {
		t.Errorf("login usaha diarsip: %d", rec.Code)
	}
	rows := getJSON[[]klienRow](t, srv, m, "/api/v1/master/klien")
	for _, r := range rows {
		if r.ID == 2 && (r.Arsip.IsZero() || r.ArsipAlasan != "permintaan pemilik" || r.PerluPerhatian) {
			t.Errorf("baris klien diarsip = %+v", r)
		}
	}
	// pulihkan → bisa masuk lagi
	if rec := do(srv, m, http.MethodDelete, "/api/v1/master/klien/2/arsip", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("pulihkan: %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodGet, "/api/v1/business", ""); rec.Code != http.StatusOK {
		t.Errorf("setelah dipulihkan: %d", rec.Code)
	}

	do(srv, m, http.MethodPost, "/api/v1/master/klien/2/arsip", `{"alasan":"tutup usaha"}`)
	if rec := do(srv, m, http.MethodDelete, "/api/v1/master/klien/2", `{"konfirmasi":"salah"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("konfirmasi salah: %d", rec.Code)
	}
	k := tambahAnggota(t, srv, m, "budi@masterpajak.id", "konsultan", []int{2})
	if rec := do(srv, k, http.MethodDelete, "/api/v1/master/klien/2", `{"konfirmasi":"toko qa baru"}`); rec.Code != http.StatusForbidden {
		t.Errorf("konsultan hapus permanen: %d", rec.Code)
	}
	if rec := do(srv, m, http.MethodDelete, "/api/v1/master/klien/2", `{"konfirmasi":" toko qa baru "}`); rec.Code != http.StatusNoContent {
		t.Fatalf("hapus permanen: %d %s", rec.Code, rec.Body)
	}
	if srv.tenantByID(2) != nil || len(srv.users.List(2)) != 0 {
		t.Fatal("usaha / user masih ada setelah hapus permanen")
	}
	if rec := do(srv, m, http.MethodDelete, "/api/v1/master/klien/1", `{"konfirmasi":"x"}`); rec.Code != http.StatusConflict {
		t.Errorf("usaha contoh dihapus: %d", rec.Code)
	}
}

func TestKelolaPengguna_UbahNonaktifResetHapus(t *testing.T) {
	srv, m := serverMaster(t)
	register(t, srv, registerBody) // usaha 2, owner qa1@x.id
	owner := srv.users.List(2)[0]
	kasir, err := srv.users.Create(auth.Input{Name: "Kasir QA", Email: "kasir@qa.id", Role: auth.Kasir, Active: true, Password: "rahasia123", TenantID: 2})
	if err != nil {
		t.Fatal(err)
	}

	list := getJSON[[]penggunaRow](t, srv, m, "/api/v1/master/pengguna")
	var dapat bool
	for _, p := range list {
		if p.Role == auth.MP {
			t.Fatal("akun tim ikut di daftar pengguna klien")
		}
		if p.ID == owner.ID && p.WA != "" && p.Usaha != "" {
			dapat = true
		}
	}
	if !dapat {
		t.Fatalf("owner usaha 2 tidak lengkap di daftar: %+v", list)
	}

	if rec := do(srv, m, http.MethodPut, "/api/v1/master/pengguna/"+itoa(kasir.ID), `{"name":"Kasir Baru","email":"kasir2@qa.id","wa":"0899","aktif":true}`); rec.Code != http.StatusOK {
		t.Fatalf("ubah pengguna: %d %s", rec.Code, rec.Body)
	}
	if u, _ := srv.users.Get(kasir.ID); u.Email != "kasir2@qa.id" || u.WA != "0899" || u.Role != auth.Kasir {
		t.Fatalf("setelah ubah: %+v", u)
	}
	// owner satu-satunya tidak bisa dinonaktifkan
	if rec := do(srv, m, http.MethodPut, "/api/v1/master/pengguna/"+itoa(owner.ID), `{"name":"QA","email":"qa1@x.id","aktif":false}`); rec.Code != http.StatusConflict {
		t.Errorf("nonaktif owner terakhir: %d", rec.Code)
	}

	var res struct{ Password string }
	rec := do(srv, m, http.MethodPost, "/api/v1/master/pengguna/"+itoa(kasir.ID)+"/reset-password", "")
	_ = json.NewDecoder(rec.Body).Decode(&res)
	if rec.Code != http.StatusOK || len(res.Password) != 12 {
		t.Fatalf("reset: %d %+v", rec.Code, res)
	}
	if loginPass(t, srv, "kasir2@qa.id", res.Password) == nil {
		t.Fatal("password sementara tidak bisa dipakai")
	}

	if rec := do(srv, m, http.MethodDelete, "/api/v1/master/pengguna/"+itoa(kasir.ID), `{"konfirmasi":"kasir2@qa.id"}`); rec.Code != http.StatusConflict {
		t.Errorf("hapus pengguna aktif: %d", rec.Code)
	}
	do(srv, m, http.MethodPut, "/api/v1/master/pengguna/"+itoa(kasir.ID), `{"name":"Kasir Baru","email":"kasir2@qa.id","aktif":false}`)
	if rec := do(srv, m, http.MethodDelete, "/api/v1/master/pengguna/"+itoa(kasir.ID), `{"konfirmasi":"kasir2@qa.id"}`); rec.Code != http.StatusNoContent {
		t.Fatalf("hapus pengguna: %d %s", rec.Code, rec.Body)
	}
	if _, ok := srv.users.Get(kasir.ID); ok {
		t.Fatal("pengguna masih ada")
	}

	k := tambahAnggota(t, srv, m, "cs@masterpajak.id", "cs", []int{2})
	if rec := do(srv, k, http.MethodGet, "/api/v1/master/pengguna", ""); rec.Code != http.StatusForbidden {
		t.Errorf("CS melihat pengguna klien: %d", rec.Code)
	}
}

func TestKelolaTim_HapusPermanen(t *testing.T) {
	srv, m := serverMaster(t)
	tambahAnggota(t, srv, m, "budi@masterpajak.id", "konsultan", nil)
	budi, _ := srv.users.FindByEmail("budi@masterpajak.id")
	if rec := do(srv, m, http.MethodDelete, "/api/v1/master/tim/"+itoa(budi.ID), `{"konfirmasi":"budi@masterpajak.id"}`); rec.Code != http.StatusConflict {
		t.Errorf("hapus anggota aktif: %d", rec.Code)
	}
	do(srv, m, http.MethodPut, "/api/v1/master/tim/"+itoa(budi.ID), `{"role":"konsultan","klien":[],"aktif":false}`)
	if rec := do(srv, m, http.MethodDelete, "/api/v1/master/tim/"+itoa(budi.ID), `{"konfirmasi":"budi@masterpajak.id"}`); rec.Code != http.StatusNoContent {
		t.Fatalf("hapus anggota: %d %s", rec.Code, rec.Body)
	}
	if _, ok := srv.users.Get(budi.ID); ok {
		t.Fatal("anggota masih ada")
	}
	me := currentMaster(t, srv, m)
	if rec := do(srv, m, http.MethodDelete, "/api/v1/master/tim/"+itoa(me), `{"konfirmasi":"x"}`); rec.Code != http.StatusConflict {
		t.Errorf("hapus diri sendiri: %d", rec.Code)
	}
}

func currentMaster(t *testing.T, srv *Server, m *http.Cookie) int {
	t.Helper()
	return getJSON[meResp](t, srv, m, "/api/v1/auth/me").User.ID
}

// Lihat-sebagai: halaman yang dibuka tercatat sekali per sesi atas nama anggota tim, dan pemilik
// usaha bisa membaca riwayat akses tim atas usahanya.
func TestLihat_HalamanTercatatAtasNamaTimDanTerlihatKlien(t *testing.T) {
	srv, _, qa, k := siapKlien(t)
	do(srv, k, http.MethodPost, "/api/v1/master/klien/2/lihat", `{"alasan":"Cek transaksi"}`)
	for range 3 {
		do(srv, k, http.MethodGet, "/api/v1/transactions", "")
	}
	riwayat := getJSON[[]struct {
		Nama, Peran, Aksi, Keterangan string
		KlienID                       int `json:"klien_id"`
	}](t, srv, qa, "/api/v1/akses-tim")
	buka, mulai := 0, false
	for _, c := range riwayat {
		if c.KlienID != 2 {
			t.Fatalf("riwayat usaha lain bocor: %+v", c)
		}
		if strings.HasPrefix(c.Aksi, "Membuka Transaksi") {
			buka++
			if c.Nama != "Anggota konsultan" || c.Peran != "Konsultan" {
				t.Errorf("tercatat atas nama %q (%q), bukan anggota tim", c.Nama, c.Peran)
			}
		}
		if c.Aksi == "Lihat sebagai klien" && c.Keterangan == "Cek transaksi" {
			mulai = true
		}
	}
	if buka != 1 || !mulai {
		t.Fatalf("riwayat = %+v", riwayat)
	}
	// anggota tim yang sedang lihat-sebagai tidak bisa membaca riwayat ini
	if rec := do(srv, k, http.MethodGet, "/api/v1/akses-tim", ""); rec.Code != http.StatusForbidden {
		t.Errorf("tim membuka riwayat akses saat lihat: %d", rec.Code)
	}
}

// Usaha baru tanpa transaksi: daftar dikirim [] (bukan null) supaya halaman Transaksi tidak crash.
func TestTransaksi_KosongBukanNull(t *testing.T) {
	srv := NewServer()
	qa := register(t, srv, registerBody)
	aktifkan(t, srv, qa)
	if body := do(srv, qa, http.MethodGet, "/api/v1/transactions", "").Body.String(); !strings.Contains(body, `"transactions":[]`) {
		t.Fatalf("daftar transaksi kosong = %s", body[:min(len(body), 80)])
	}
}
