package main

import (
	"net/http"
	"strings"
	"time"

	"pajak/internal/auth"
	"pajak/internal/master"
)

// Lihat sebagai klien: anggota tim membuka data satu usaha apa adanya, HANYA BACA, selama
// lihatDurasi. Alasan wajib dan tercatat di audit log. Selama aktif, permintaan akun tim ke
// endpoint data usaha dijalankan sebagai user sementara ber-role auth.Lihat atas usaha itu
// (lihat guard); permintaan pengubah dan ekspor ditolak.

const lihatDurasi = 2 * time.Hour

type lihatGrant struct {
	KlienID int
	Alasan  string
	Sampai  time.Time
	// dibuka: halaman yang sudah dicatat di audit selama sesi ini (sekali per halaman).
	dibuka map[string]bool
}

// lihatInfo menyertai /auth/me selama mode lihat-sebagai.
type lihatInfo struct {
	KlienID int       `json:"klien_id"`
	Nama    string    `json:"nama"`
	Alasan  string    `json:"alasan"`
	Sampai  time.Time `json:"sampai"`
}

type lihatKey struct{}

// sedangLihat: request ini berjalan dalam mode lihat-sebagai.
func sedangLihat(r *http.Request) bool { return r.Context().Value(lihatKey{}) != nil }

// lihatUntuk: izin lihat-sebagai milik user yang masih berlaku. Penugasan & izin dicek ulang di
// setiap request, jadi mencabutnya di Tim & Akses langsung menghentikan akses.
func (s *Server) lihatUntuk(userID int, now time.Time) (lihatGrant, bool) {
	s.masterMu.Lock()
	defer s.masterMu.Unlock()
	g, ok := s.lihat[userID]
	if !ok {
		return g, false
	}
	ak, anggota := s.master.Akses(userID)
	if !now.Before(g.Sampai) || !anggota || !ak.Bisa(master.LihatSebagai) || !ak.BolehKlien(g.KlienID) {
		delete(s.lihat, userID)
		return g, false
	}
	return g, true
}

// sebagaiKlien: user sementara untuk endpoint data usaha selama lihat-sebagai.
func sebagaiKlien(u auth.User, g lihatGrant) auth.User {
	return auth.User{ID: u.ID, Name: u.Name, Email: u.Email, Role: auth.Lihat, Active: true, TenantID: g.KlienID}
}

func (s *Server) lihatInfoFor(u auth.User) *lihatInfo {
	g, ok := s.lihatUntuk(u.ID, time.Now())
	if !ok {
		return nil
	}
	s.tenantsMu.RLock()
	t := s.tenants[g.KlienID]
	s.tenantsMu.RUnlock()
	nama := ""
	if t != nil {
		t.mu.RLock()
		if t.profile != nil {
			nama = t.profile.NamaUsaha
		}
		nama = namaKlien(t, nama)
		t.mu.RUnlock()
	}
	return &lihatInfo{g.KlienID, nama, g.Alasan, g.Sampai}
}

func (s *Server) handleMulaiLihat(w http.ResponseWriter, r *http.Request) {
	t := s.klienDari(w, r)
	if t == nil {
		return
	}
	var req struct{ Alasan string }
	if !decodeBody(w, r, &req) {
		return
	}
	req.Alasan = clip(strings.TrimSpace(req.Alasan), 300)
	if req.Alasan == "" {
		http.Error(w, "alasan wajib diisi", http.StatusBadRequest)
		return
	}
	if err := s.catat(r, "Lihat sebagai klien", t.id, req.Alasan); err != nil {
		writeMasterErr(w, err)
		return
	}
	g := lihatGrant{t.id, req.Alasan, time.Now().Add(lihatDurasi), map[string]bool{}}
	s.masterMu.Lock()
	s.lihat[currentUser(r).ID] = g
	s.masterMu.Unlock()
	writeJSON(w, s.lihatInfoFor(currentUser(r)))
}

// namaHalaman: label halaman untuk audit lihat-sebagai (sama dengan menu di frontend).
var namaHalaman = map[string]string{
	"dashboard": "Dashboard", "laporan-keuangan": "Laporan Keuangan", "laporan-penjualan": "Laporan Penjualan",
	"transaksi": "Transaksi", "piutang-utang": "Piutang & Utang", "stok-gudang": "Stok & Gudang",
	"produksi-supplier": "Produksi & Supplier", "invoice": "Invoice", "simulasi-harga": "Simulasi & Harga",
	"pajak-umkm": "Pajak UMKM", "kalkulator-pajak": "Kalkulator Pajak", "perencanaan-pajak": "Perencanaan Pajak",
	"karyawan": "Karyawan", "master-data": "Daftar Akun", "profil-usaha": "Profil Usaha", "pengguna": "Pengguna & Akses",
	"langganan": "Langganan", "impor-data": "Impor Data Awal", "saldo-awal": "Saldo Awal", "buku-besar": "Buku Besar", "jurnal-umum": "Jurnal Umum", "laporan-transaksi": "Laporan Transaksi", "pustaka": "Artikel Pajak",
}

// catatBukaLihat: setiap halaman data klien yang dibuka selama lihat-sebagai tercatat sekali
// per sesi, atas nama anggota tim (bukan klien), dan terlihat juga oleh klien (Riwayat akses tim).
func (s *Server) catatBukaLihat(u auth.User, view string) {
	s.masterMu.Lock()
	g, ok := s.lihat[u.ID]
	if !ok || g.dibuka == nil || g.dibuka[view] {
		s.masterMu.Unlock()
		return
	}
	g.dibuka[view] = true
	peran := "Tim Master Pajak"
	if ak, ok := s.master.Akses(u.ID); ok {
		peran = ak.Role.Nama
	}
	s.masterMu.Unlock()
	label := namaHalaman[view]
	if label == "" {
		label = view
	}
	_ = s.ubahMaster(func(st *master.State) error {
		st.Catat(master.Catatan{UserID: u.ID, Nama: u.Name, Peran: peran, Aksi: "Membuka " + label + " (lihat sebagai klien, hanya baca)", KlienID: g.KlienID})
		return nil
	})
}

// handleRiwayatAksesTim: untuk pemilik usaha — semua catatan audit tim Master Pajak atas usaha ini
// (lihat-sebagai, halaman yang dibuka, usulan koreksi, perubahan data oleh Master), terbaru dulu.
func (s *Server) handleRiwayatAksesTim(w http.ResponseWriter, r *http.Request) {
	id := s.tenantOf(r).id
	s.masterMu.Lock()
	a := s.master.Audit
	out := []master.Catatan{}
	for i := len(a) - 1; i >= 0 && len(out) < 200; i-- {
		if a[i].KlienID == id {
			out = append(out, a[i])
		}
	}
	s.masterMu.Unlock()
	writeJSON(w, out)
}

func (s *Server) handleSelesaiLihat(w http.ResponseWriter, r *http.Request) {
	s.masterMu.Lock()
	delete(s.lihat, currentUser(r).ID)
	s.masterMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}
