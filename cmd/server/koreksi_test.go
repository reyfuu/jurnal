package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"pajak/internal/finance"
	"pajak/internal/master"
)

// siapKlien: Master, usaha QA (id 2, aktif) dengan satu transaksi, dan konsultan yang
// ditugaskan ke usaha itu.
func siapKlien(t *testing.T) (srv *Server, m, qa, k *http.Cookie) {
	t.Helper()
	srv, m = serverMaster(t)
	qa = register(t, srv, registerBody)
	aktifkan(t, srv, qa)
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-10-13","jenis":"Perawatan & Perlengkapan","nominal":4500000,"keterangan":"Beli mesin espresso","lunas":true}`); rec.Code != http.StatusCreated {
		t.Fatalf("tx: %d %s", rec.Code, rec.Body)
	}
	k = tambahAnggota(t, srv, m, "budi@masterpajak.id", "konsultan", []int{2})
	return
}

func TestLihat_HanyaBacaDanTercatat(t *testing.T) {
	srv, m, _, k := siapKlien(t)
	if rec := do(srv, k, http.MethodGet, "/api/v1/transactions", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("tanpa lihat-sebagai: want 403, got %d", rec.Code)
	}
	if rec := do(srv, k, http.MethodPost, "/api/v1/master/klien/2/lihat", `{"alasan":"  "}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("tanpa alasan: want 400, got %d", rec.Code)
	}
	if rec := do(srv, k, http.MethodPost, "/api/v1/master/klien/1/lihat", `{"alasan":"x"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("klien tidak ditugaskan: want 404, got %d", rec.Code)
	}
	if rec := do(srv, k, http.MethodPost, "/api/v1/master/klien/2/lihat", `{"alasan":"Cek kategori pembelian mesin"}`); rec.Code != http.StatusOK {
		t.Fatalf("mulai lihat: %d %s", rec.Code, rec.Body)
	}

	var list txListResp
	rec := do(srv, k, http.MethodGet, "/api/v1/transactions", "")
	_ = json.NewDecoder(rec.Body).Decode(&list)
	if rec.Code != http.StatusOK || len(list.Transactions) != 1 || list.CanEdit {
		t.Fatalf("baca transaksi klien: %d %+v", rec.Code, list)
	}
	me := getJSON[meResp](t, srv, k, "/api/v1/auth/me")
	if me.Lihat == nil || me.Lihat.KlienID != 2 || me.Lihat.Alasan != "Cek kategori pembelian mesin" || me.Langganan.Status != "aktif" {
		t.Fatalf("me saat lihat = %+v", me)
	}
	for _, c := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-10-14","jenis":"Penjualan","nominal":1,"keterangan":"x","lunas":true}`},
		{http.MethodDelete, "/api/v1/transactions/1", ""},
		{http.MethodGet, "/api/v1/reports/coa.xlsx?period=2026-10", ""}, // ekspor menggugurkan garansi klien
		{http.MethodGet, "/api/v1/users", ""},
	} {
		if rec := do(srv, k, c.method, c.path, c.body); rec.Code != http.StatusForbidden {
			t.Errorf("%s %s saat lihat: want 403, got %d", c.method, c.path, rec.Code)
		}
	}
	if tn := srv.tenants[2]; !tn.langganan.EksporPertama.IsZero() {
		t.Error("lihat-sebagai menggugurkan garansi klien")
	}
	if rec := do(srv, k, http.MethodGet, "/api/v1/master/klien", ""); rec.Code != http.StatusOK {
		t.Errorf("Panel Master saat lihat: %d", rec.Code)
	}
	// Penugasan dicabut: akses berhenti saat itu juga.
	budi, _ := srv.users.FindByEmail("budi@masterpajak.id")
	if rec := do(srv, m, http.MethodPut, "/api/v1/master/tim/"+itoa(budi.ID), `{"role":"konsultan","klien":[],"aktif":true}`); rec.Code != http.StatusOK {
		t.Fatalf("cabut penugasan: %d", rec.Code)
	}
	if rec := do(srv, k, http.MethodGet, "/api/v1/transactions", ""); rec.Code != http.StatusForbidden {
		t.Errorf("setelah penugasan dicabut: want 403, got %d", rec.Code)
	}
	audit := getJSON[[]master.Catatan](t, srv, m, "/api/v1/master/audit")
	if !slicesAny(audit, func(c master.Catatan) bool {
		return c.Aksi == "Lihat sebagai klien" && c.Keterangan == "Cek kategori pembelian mesin" && c.KlienID == 2
	}) {
		t.Errorf("audit lihat-sebagai tidak ada: %+v", audit)
	}
}

func TestLihat_KeluarDanTanpaIzin(t *testing.T) {
	srv, m, _, k := siapKlien(t)
	cs := tambahAnggota(t, srv, m, "rina@masterpajak.id", "cs", []int{2})
	if rec := do(srv, cs, http.MethodPost, "/api/v1/master/klien/2/lihat", `{"alasan":"x"}`); rec.Code != http.StatusForbidden {
		t.Errorf("cs tanpa izin lihat: want 403, got %d", rec.Code)
	}
	do(srv, k, http.MethodPost, "/api/v1/master/klien/2/lihat", `{"alasan":"Tiket BUG-0001"}`)
	if rec := do(srv, k, http.MethodDelete, "/api/v1/master/lihat", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("keluar: %d", rec.Code)
	}
	if rec := do(srv, k, http.MethodGet, "/api/v1/transactions", ""); rec.Code != http.StatusForbidden {
		t.Errorf("setelah keluar: want 403, got %d", rec.Code)
	}
	if me := getJSON[meResp](t, srv, k, "/api/v1/auth/me"); me.Lihat != nil || me.Views[0] != "panel-master" {
		t.Errorf("me setelah keluar = %+v", me)
	}
}

func txsOf(t *testing.T, srv *Server, c *http.Cookie) []finance.Transaction {
	t.Helper()
	return getJSON[txListResp](t, srv, c, "/api/v1/transactions").Transactions
}

func usul(t *testing.T, srv *Server, c *http.Cookie, body string) koreksiRow {
	t.Helper()
	rec := do(srv, c, http.MethodPost, "/api/v1/master/klien/2/koreksi", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("usulkan koreksi: %d %s", rec.Code, rec.Body)
	}
	var k koreksiRow
	_ = json.NewDecoder(rec.Body).Decode(&k)
	return k
}

const usulAset = `{"transaksi_id":1,"jenis":"ubah","alasan":"Mesin dipakai lebih dari setahun, dicatat sebagai aset","usulan":{"tanggal":"2026-10-13","jenis":"Beli Alat / Aset","nominal":4500000,"keterangan":"Beli mesin espresso","lunas":true}}`

func TestKoreksi_DisetujuiPemilik(t *testing.T) {
	srv, m, qa, k := siapKlien(t)
	kr := usul(t, srv, k, usulAset)
	if kr.Nomor != "KR-001" || kr.Status != finance.KoreksiMenunggu || len(kr.Dampak) != 1 || kr.Dampak[0].LabaSesudah-kr.Dampak[0].LabaSebelum != 4_500_000 || kr.Oleh != "Anggota konsultan" {
		t.Fatalf("usulan = %+v", kr)
	}
	// Belum disetujui: pembukuan klien tidak berubah.
	if got := txsOf(t, srv, qa); got[0].Jenis != "Perawatan & Perlengkapan" {
		t.Fatalf("usulan sudah mengubah pembukuan sebelum disetujui: %+v", got[0])
	}
	if me := getJSON[meResp](t, srv, qa, "/api/v1/auth/me"); me.KoreksiMenunggu != 1 {
		t.Errorf("koreksi_menunggu = %d", me.KoreksiMenunggu)
	}
	milik := getJSON[[]koreksiRow](t, srv, qa, "/api/v1/koreksi")
	if len(milik) != 1 || milik[0].Usulan.Jenis != "Beli Alat / Aset" {
		t.Fatalf("daftar usulan klien = %+v", milik)
	}
	// Usaha lain tidak melihat & tidak bisa memutuskan.
	owner1 := login(t, srv, "owner@kedaikopisenja.id")
	if got := getJSON[[]koreksiRow](t, srv, owner1, "/api/v1/koreksi"); len(got) != 0 {
		t.Errorf("usaha lain melihat %d usulan", len(got))
	}
	if rec := do(srv, owner1, http.MethodPost, "/api/v1/koreksi/1/setujui", `{}`); rec.Code != http.StatusNotFound {
		t.Errorf("usaha lain menyetujui: want 404, got %d", rec.Code)
	}
	kasir1 := login(t, srv, "kasir@kedaikopisenja.id")
	if rec := do(srv, kasir1, http.MethodGet, "/api/v1/koreksi", ""); rec.Code != http.StatusForbidden {
		t.Errorf("kasir membuka usulan: want 403, got %d", rec.Code)
	}

	if rec := do(srv, qa, http.MethodPost, "/api/v1/koreksi/1/setujui", `{"catatan":"Oke, betul"}`); rec.Code != http.StatusOK {
		t.Fatalf("setujui: %d %s", rec.Code, rec.Body)
	}
	if got := txsOf(t, srv, qa); got[0].Jenis != "Beli Alat / Aset" || got[0].ID != 1 {
		t.Fatalf("setelah disetujui transaksi belum berubah: %+v", got[0])
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/koreksi/1/setujui", `{}`); rec.Code != http.StatusConflict {
		t.Errorf("setujui dua kali: want 409, got %d", rec.Code)
	}
	tim := getJSON[[]koreksiRow](t, srv, k, "/api/v1/master/koreksi")
	if len(tim) != 1 || tim[0].Status != finance.KoreksiDisetujui || tim[0].Catatan != "Oke, betul" || tim[0].DiputuskanOleh != "QA Satu" || tim[0].KlienNama == "" {
		t.Fatalf("usulan di Panel Master = %+v", tim)
	}
	audit := getJSON[[]master.Catatan](t, srv, m, "/api/v1/master/audit")
	if audit[0].Aksi != "Setujui koreksi KR-001" || audit[0].Nama != "QA Satu" || audit[0].Peran != "Owner klien" {
		t.Errorf("audit keputusan klien = %+v", audit[0])
	}
}

func TestKoreksi_BasiDitolakDanValidasi(t *testing.T) {
	srv, m, qa, k := siapKlien(t)
	usul(t, srv, k, `{"transaksi_id":1,"jenis":"hapus","alasan":"Tercatat dua kali"}`)
	// Pemilik mengubah transaksinya dulu: usulan basi, tidak menimpa perubahan pemilik.
	if rec := do(srv, qa, http.MethodPut, "/api/v1/transactions/1", `{"tanggal":"2026-10-13","jenis":"Perawatan & Perlengkapan","nominal":4600000,"keterangan":"Beli mesin espresso","lunas":true}`); rec.Code != http.StatusOK {
		t.Fatalf("ubah tx: %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/koreksi/1/setujui", `{}`); rec.Code != http.StatusConflict {
		t.Fatalf("setujui usulan basi: want 409, got %d", rec.Code)
	}
	if got := getJSON[[]koreksiRow](t, srv, qa, "/api/v1/koreksi"); got[0].Status != finance.KoreksiKedaluwarsa {
		t.Errorf("status usulan basi = %s", got[0].Status)
	}
	if got := txsOf(t, srv, qa); len(got) != 1 || got[0].Nominal != 4_600_000 {
		t.Errorf("transaksi pemilik tertimpa usulan basi: %+v", got)
	}

	usul(t, srv, k, `{"transaksi_id":1,"jenis":"hapus","alasan":"Tercatat dua kali"}`)
	if rec := do(srv, qa, http.MethodPost, "/api/v1/koreksi/2/tolak", `{"catatan":"Bukan dobel, memang beli dua"}`); rec.Code != http.StatusOK {
		t.Fatalf("tolak: %d %s", rec.Code, rec.Body)
	}
	if got := getJSON[[]koreksiRow](t, srv, qa, "/api/v1/koreksi"); got[0].ID != 2 || got[0].Status != finance.KoreksiDitolak {
		t.Errorf("setelah tolak = %+v", got[0])
	}
	if got := txsOf(t, srv, qa); len(got) != 1 {
		t.Error("tolak tetap menghapus transaksi")
	}

	cs := tambahAnggota(t, srv, m, "rina@masterpajak.id", "cs", []int{2})
	for _, c := range []struct {
		c    *http.Cookie
		path string
		body string
		want int
	}{
		{cs, "/api/v1/master/klien/2/koreksi", usulAset, http.StatusForbidden},
		{k, "/api/v1/master/klien/1/koreksi", usulAset, http.StatusNotFound},
		{k, "/api/v1/master/klien/2/koreksi", `{"transaksi_id":1,"jenis":"ubah","alasan":"x","usulan":{"tanggal":"2026-10-13","jenis":"Penjualan","nominal":1,"keterangan":"x","lunas":true,"akun":"9999"}}`, http.StatusBadRequest},
		{k, "/api/v1/master/klien/2/koreksi", `{"transaksi_id":99,"jenis":"hapus","alasan":"x"}`, http.StatusBadRequest},
	} {
		if rec := do(srv, c.c, http.MethodPost, c.path, c.body); rec.Code != c.want {
			t.Errorf("POST %s %s: want %d, got %d %s", c.path, c.body[:20], c.want, rec.Code, rec.Body)
		}
	}
}
