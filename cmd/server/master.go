package main

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"pajak/internal/auth"
	"pajak/internal/master"
)

// Panel Master: konsol tim Master Pajak untuk memantau klien (usaha). Akun tim adalah user auth
// ber-role "mp" tanpa usaha; role tim, izin, dan klien yang ditugaskan diatur di
// internal/master. Panel ini hanya membaca METADATA usaha (status, aktivitas, jumlah transaksi)
// — angka keuangan klien tidak dikirim lewat endpoint mana pun di sini.

// masterInfo menyertai /auth/me untuk akun tim.
type masterInfo struct {
	Role  master.Role   `json:"role"`
	Izin  []master.Izin `json:"izin"`
	Klien int           `json:"klien"` // jumlah klien yang ditugaskan; -1 = semua
}

func (s *Server) masterInfoFor(u auth.User) *masterInfo {
	s.masterMu.Lock()
	ak, ok := s.master.Akses(u.ID)
	s.masterMu.Unlock()
	if !ok {
		return nil
	}
	in := &masterInfo{Role: ak.Role, Izin: []master.Izin{}, Klien: len(ak.Klien)}
	for _, i := range master.DaftarIzin {
		if ak.Bisa(i.ID) {
			in.Izin = append(in.Izin, i.ID)
		}
	}
	if ak.Bisa(master.KlienSemua) {
		in.Klien = -1
	}
	return in
}

// --- muat, simpan, bootstrap ---

func masterFile() string { return dataPath("master.json") }

func loadMasterFile() master.State {
	st := master.NewState()
	if p := masterFile(); p != "" {
		if raw, err := os.ReadFile(p); err == nil {
			if err := json.Unmarshal(raw, &st); err != nil {
				log.Fatalf("master.json rusak, perbaiki atau pindahkan dulu: %v", err)
			}
		}
	}
	return st
}

// saveMaster menulis state tim; baru = baris audit yang belum tersimpan (Postgres hanya
// menambahkan baris audit, tidak pernah menulis ulang).
func (s *Server) saveMaster(st master.State, baru []master.Catatan) error {
	if s.db != nil {
		return saveMasterDB(s.db, st, baru)
	}
	if p := masterFile(); p != "" {
		return writeFileAtomic(p, st)
	}
	return nil
}

// ubahMaster menjalankan f atas salinan state. Salinan disimpan dulu, baru dipasang: gagal
// simpan berarti state di memori tetap yang lama.
//
// ponytail: Clone menyalin seluruh audit tiap perubahan; cukup untuk puluhan ribu baris.
// Pisahkan audit dari State bila log melewati ~100 rb baris.
func (s *Server) ubahMaster(f func(st *master.State) error) error {
	s.masterMu.Lock()
	defer s.masterMu.Unlock()
	st := s.master.Clone()
	n := len(st.Audit)
	if err := f(&st); err != nil {
		return err
	}
	if err := s.saveMaster(st, st.Audit[n:]); err != nil {
		log.Printf("GAGAL menyimpan data Panel Master: %v", err)
		return fmt.Errorf("%w: %w", errSimpan, err)
	}
	s.master = st
	return nil
}

// masterEmail: email Master pertama. Bersama MASTER_PASSWORD dipakai sekali saat belum ada
// anggota ber-role Master; setelah itu tim dikelola dari Panel Master.
func masterEmail() string { return strings.ToLower(strings.TrimSpace(os.Getenv("MASTER_EMAIL"))) }

func (s *Server) bootstrapMaster() {
	email := masterEmail()
	if email == "" {
		return
	}
	s.masterMu.Lock()
	ada := s.master.MasterLain(0)
	s.masterMu.Unlock()
	if ada {
		return
	}
	u, ok := s.users.FindByEmail(email)
	switch {
	case ok && u.Role != auth.MP:
		log.Printf("PERINGATAN: MASTER_EMAIL %s dipakai akun usaha; pakai email lain untuk Master", email)
		return
	case !ok:
		var err error
		if u, err = s.users.Create(auth.Input{Name: "Master", Email: email, Password: os.Getenv("MASTER_PASSWORD"), Role: auth.MP, Active: true}); err != nil {
			log.Printf("PERINGATAN: Master pertama tidak dibuat (MASTER_PASSWORD minimal 8 karakter): %v", err)
			return
		}
	}
	err := s.ubahMaster(func(st *master.State) error {
		if err := st.SetAnggota(master.Anggota{UserID: u.ID, Role: master.RoleMaster}); err != nil {
			return err
		}
		st.Catat(master.Catatan{UserID: u.ID, Nama: u.Name, Peran: "Master", Aksi: "Master pertama dibuat dari MASTER_EMAIL"})
		return nil
	})
	if err == nil {
		err = s.saveFor(nil)
	}
	if err != nil {
		log.Printf("PERINGATAN: gagal menyimpan Master pertama: %v", err)
	}
}

// --- guard ---

type aksesKey struct{}

// guardMP: hanya akun tim yang terdaftar di Panel Master, dan (bila izin diberikan) yang punya
// salah satu izin itu.
func (s *Server) guardMP(h http.HandlerFunc, izin ...master.Izin) http.HandlerFunc {
	return s.guard("panel-master", func(w http.ResponseWriter, r *http.Request) {
		s.masterMu.Lock()
		ak, ok := s.master.Akses(currentUser(r).ID)
		s.masterMu.Unlock()
		if !ok || len(izin) > 0 && !slices.ContainsFunc(izin, ak.Bisa) {
			http.Error(w, "akses ditolak untuk role tim ini", http.StatusForbidden)
			return
		}
		h(w, r.WithContext(context.WithValue(r.Context(), aksesKey{}, ak)))
	})
}

func aksesOf(r *http.Request) master.Akses { return r.Context().Value(aksesKey{}).(master.Akses) }

// izinKlien: membuka ringkasan & daftar klien.
var izinKlien = []master.Izin{master.KlienLihat, master.KlienSemua}

// catat menambah baris audit atas nama anggota yang login.
func (s *Server) catat(r *http.Request, aksi string, klienID int, ket string) error {
	u, ak := currentUser(r), aksesOf(r)
	return s.ubahMaster(func(st *master.State) error {
		st.Catat(master.Catatan{UserID: u.ID, Nama: u.Name, Peran: ak.Role.Nama, Aksi: aksi, KlienID: klienID, Keterangan: ket})
		return nil
	})
}

func writeMasterErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch {
	case errors.Is(err, master.ErrTidakValid):
		code = http.StatusBadRequest
	case errors.Is(err, master.ErrRoleDipakai), errors.Is(err, master.ErrMasterTerakhir):
		code = http.StatusConflict
	case errors.Is(err, errSimpan):
		code = http.StatusServiceUnavailable
	}
	http.Error(w, err.Error(), code)
}

// --- klien ---

type orang struct {
	ID   int    `json:"id"`
	Nama string `json:"nama"`
}

type klienRow struct {
	ID                int       `json:"id"`
	Nama              string    `json:"nama"`
	Jenis             string    `json:"jenis,omitempty"`
	Kota              string    `json:"kota,omitempty"`
	KLU               string    `json:"klu,omitempty"`
	PKP               bool      `json:"pkp"`
	Demo              bool      `json:"demo,omitempty"`
	OwnerNama         string    `json:"owner_nama,omitempty"`
	OwnerEmail        string    `json:"owner_email,omitempty"`
	OwnerWA           string    `json:"owner_wa,omitempty"`
	User              int       `json:"user"`
	Status            string    `json:"status"`
	Sampai            time.Time `json:"sampai,omitzero"`
	Paket             string    `json:"paket,omitempty"`
	TerakhirAktif     time.Time `json:"terakhir_aktif,omitzero"`
	TransaksiBulanIni int       `json:"transaksi_bulan_ini"`
	Onboarding        bool      `json:"onboarding"`
	Daftar            string    `json:"daftar,omitempty"`
	Penangan          []orang   `json:"penangan"`
	PerluPerhatian    bool      `json:"perlu_perhatian"`
	Alasan            string    `json:"alasan,omitempty"`
	Arsip             time.Time `json:"arsip,omitzero"` // diarsipkan (soft delete)
	ArsipAlasan       string    `json:"arsip_alasan,omitempty"`
	mrr               int64     // pendapatan per bulan dari usaha ini (tidak dikirim)
}

// paketTerakhir: order lunas/manual terbaru, dasar nama paket & MRR.
func paketTerakhir(orders []order) (paket, order, bool) {
	for i := len(orders) - 1; i >= 0; i-- {
		if o := orders[i]; o.Status == "lunas" || o.Status == "manual" {
			p, ok := findPaket(o.Paket)
			return p, o, ok
		}
	}
	return paket{}, order{}, false
}

// namaKlien: nama usaha di Panel Master; profil kosong tetap punya nama yang bisa dikenali.
// Pemanggil memegang t.mu (nama = profile.NamaUsaha yang sudah dibaca).
func namaKlien(t *tenant, nama string) string {
	switch {
	case nama != "":
		return nama
	case t.demo:
		return "Kedai Kopi Senja (data contoh)"
	}
	return fmt.Sprintf("Usaha #%d (belum isi profil)", t.id)
}

// klienRows: usaha yang boleh dilihat anggota, urut ID.
//
// hitungTx memindai seluruh transaksi tiap usaha, jadi hanya dinyalakan oleh endpoint yang
// benar-benar menampilkan kolom "Transaksi bulan ini". Pada 1.000 usaha × 10.000 transaksi
// setahun, memindai tanpa syarat berarti 10 juta baris per request (terukur ~0,5–1,3 detik
// di BenchmarkKlienRows).
func (s *Server) klienRows(ak master.Akses, now time.Time, hitungTx bool) []klienRow {
	return s.klienRowsFilter(ak, now, hitungTx, 0)
}

// klienRowsFilter: sama seperti klienRows, tetapi hanyaID > 0 membatasi hasil ke satu usaha
// supaya halaman detail tidak perlu membangun baris seluruh klien.
func (s *Server) klienRowsFilter(ak master.Akses, now time.Time, hitungTx bool, hanyaID int) []klienRow {
	users := map[int][]auth.User{}
	nama := map[int]string{}
	for _, u := range s.users.Snapshot().Users {
		nama[u.ID] = u.Name
		if u.Role != auth.MP {
			users[u.TenantID] = append(users[u.TenantID], u.User)
		}
	}
	s.masterMu.Lock()
	penangan := map[int][]orang{}
	s.tenantsMu.RLock()
	ts := make([]*tenant, 0, len(s.tenants))
	for id, t := range s.tenants {
		if hanyaID != 0 && id != hanyaID {
			continue
		}
		if ak.BolehKlien(id) {
			ts = append(ts, t)
			for _, uid := range s.master.Penangan(id) {
				penangan[id] = append(penangan[id], orang{uid, nama[uid]})
			}
		}
	}
	s.tenantsMu.RUnlock()
	s.masterMu.Unlock()
	slices.SortFunc(ts, func(a, b *tenant) int { return a.id - b.id })

	bulan := now.Format("2006-01")
	out := make([]klienRow, 0, len(ts))
	for _, t := range ts {
		st := t.langgananStatus(now)
		row := klienRow{ID: t.id, Demo: t.demo, Status: st.Status, Sampai: st.Sampai, Penangan: nonNil(penangan[t.id])}
		t.mu.RLock()
		row.Arsip, row.ArsipAlasan = t.arsip, t.arsipAlasan
		if p := t.profile; p != nil {
			row.Nama, row.Jenis, row.Kota, row.KLU, row.PKP, row.Onboarding = p.NamaUsaha, p.JenisUsaha, p.Kota, p.KLU, p.PKP, true
		}
		if hitungTx {
			for _, tx := range t.txs {
				if strings.HasPrefix(tx.Tanggal, bulan) {
					row.TransaksiBulanIni++
				}
			}
		}
		if p, o, ok := paketTerakhir(t.langganan.Orders); ok {
			row.Paket = p.Nama
			if st.Status == "aktif" && p.Hari > 0 {
				row.mrr = o.Jumlah * 30 / int64(p.Hari)
			}
		}
		t.mu.RUnlock()
		row.Nama = namaKlien(t, row.Nama)
		for _, u := range users[t.id] {
			row.User++
			if u.LastSeen.After(row.TerakhirAktif) {
				row.TerakhirAktif = u.LastSeen
			}
			if row.Daftar == "" || u.CreatedAt < row.Daftar {
				row.Daftar = u.CreatedAt
			}
			if u.Role == auth.Owner && row.OwnerEmail == "" {
				row.OwnerNama, row.OwnerEmail, row.OwnerWA = u.Name, u.Email, u.WA
			}
		}
		if !t.demo && row.Arsip.IsZero() { // usaha contoh milik operator selalu aktif; tidak perlu diawasi
			row.PerluPerhatian, row.Alasan = master.PerluPerhatian(row.Status, row.Sampai, row.TerakhirAktif, now)
			if row.Status == "habis" && row.Paket == "" {
				row.Alasan = "Belum berlangganan"
			}
		}
		out = append(out, row)
	}
	return out
}

func (s *Server) handleMasterKlien(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.klienRows(aksesOf(r), time.Now(), true)) // daftar klien menampilkan kolom transaksi bulan ini
}

type ringkasanMaster struct {
	Klien           int        `json:"klien"`
	Aktif           int        `json:"aktif"`
	Trial           int        `json:"trial"`
	Habis           int        `json:"habis"`
	MRR             *int64     `json:"mrr,omitempty"` // hanya untuk izin pembayaran
	TiketTerbuka    int        `json:"tiket_terbuka"`
	PerluPerhatian  []klienRow `json:"perlu_perhatian"`
	BelumDitugaskan []klienRow `json:"belum_ditugaskan,omitempty"` // hanya yang melihat semua klien
	TerakhirAktif   []klienRow `json:"terakhir_aktif"`
}

func (s *Server) handleMasterRingkasan(w http.ResponseWriter, r *http.Request) {
	ak := aksesOf(r)
	rows := s.klienRows(ak, time.Now(), false) // ringkasan hanya memakai status, MRR, dan keaktifan
	out := ringkasanMaster{Klien: len(rows), PerluPerhatian: []klienRow{}}
	var mrr int64
	for _, k := range rows {
		switch k.Status {
		case "aktif":
			out.Aktif++
		case "trial":
			out.Trial++
		case "habis":
			out.Habis++
		}
		mrr += k.mrr
		if k.PerluPerhatian {
			out.PerluPerhatian = append(out.PerluPerhatian, k)
		}
		if ak.Bisa(master.KlienSemua) && len(k.Penangan) == 0 {
			out.BelumDitugaskan = append(out.BelumDitugaskan, k)
		}
	}
	if ak.Bisa(master.Pembayaran) {
		out.MRR = &mrr
	}
	for _, t := range s.tiketRows(ak) {
		if t.Status != "selesai" {
			out.TiketTerbuka++
		}
	}
	aktif := slices.Clone(rows)
	slices.SortStableFunc(aktif, func(a, b klienRow) int { return b.TerakhirAktif.Compare(a.TerakhirAktif) })
	out.TerakhirAktif = aktif[:min(5, len(aktif))]
	writeJSON(w, out)
}

// klienDari: usaha di path {id} yang boleh dilihat anggota; nil + 404 bila tidak.
func (s *Server) klienDari(w http.ResponseWriter, r *http.Request) *tenant {
	id, _ := strconv.Atoi(r.PathValue("id"))
	s.tenantsMu.RLock()
	t := s.tenants[id]
	s.tenantsMu.RUnlock()
	if t == nil || !aksesOf(r).BolehKlien(id) {
		http.Error(w, "klien tidak ditemukan", http.StatusNotFound)
		return nil
	}
	return t
}

type klienDetail struct {
	Klien   klienRow     `json:"klien"`
	Users   []auth.User  `json:"users"`
	Orders  []order      `json:"orders"`
	Tiket   []tiketRow   `json:"tiket"`
	Koreksi []koreksiRow `json:"koreksi"`
	// Identitas usaha untuk diedit tim (tanpa angka keuangan); nil = profil belum diisi.
	Identitas *identitasUsaha `json:"identitas"`
}

type identitasUsaha struct {
	NamaUsaha string `json:"nama_usaha"`
	Alamat    string `json:"alamat,omitempty"`
	Kecamatan string `json:"kecamatan,omitempty"`
	Kota      string `json:"kota,omitempty"`
	Provinsi  string `json:"provinsi,omitempty"`
	NPWP      string `json:"npwp,omitempty"`
	Telepon   string `json:"telepon,omitempty"`
	Email     string `json:"email,omitempty"`
}

func (s *Server) handleMasterKlienDetail(w http.ResponseWriter, r *http.Request) {
	t := s.klienDari(w, r)
	if t == nil {
		return
	}
	ak := aksesOf(r)
	var d klienDetail
	for _, k := range s.klienRowsFilter(ak, time.Now(), true, t.id) { // detail menampilkan transaksi bulan ini, satu usaha saja
		if k.ID == t.id {
			d.Klien = k
		}
	}
	d.Users = nonNil(s.users.List(t.id))
	t.mu.RLock()
	d.Orders = nonNil(slices.Clone(t.langganan.Orders))
	if p := t.profile; p != nil {
		d.Identitas = &identitasUsaha{p.NamaUsaha, p.Alamat, p.Kecamatan, p.Kota, p.Provinsi, p.NPWP, p.Telepon, p.Email}
	}
	t.mu.RUnlock()
	d.Tiket = []tiketRow{}
	for _, x := range s.tiketRows(ak) {
		if x.TenantID == t.id {
			d.Tiket = append(d.Tiket, x)
		}
	}
	d.Koreksi = koreksiRows(t)
	if err := s.catat(r, "Buka detail klien", t.id, ""); err != nil {
		writeMasterErr(w, err)
		return
	}
	writeJSON(w, d)
}

// handleMasterLangganan: perpanjang masa coba atau aktifkan paket manual (bayar transfer,
// promo, kompensasi). Alasan wajib dan masuk audit log.
func (s *Server) handleMasterLangganan(w http.ResponseWriter, r *http.Request) {
	t := s.klienDari(w, r)
	if t == nil {
		return
	}
	var req struct {
		Aksi, Paket, Alasan string
		Hari                int
	}
	if !decodeBody(w, r, &req) {
		return
	}
	req.Alasan = clip(strings.TrimSpace(req.Alasan), 300)
	if req.Alasan == "" {
		http.Error(w, "alasan wajib diisi", http.StatusBadRequest)
		return
	}
	now := time.Now().UTC().Truncate(time.Second)
	t.mu.Lock()
	lama := t.langganan
	l := &t.langganan
	var aksi string
	switch req.Aksi {
	case "trial":
		if req.Hari < 1 || req.Hari > 90 {
			t.mu.Unlock()
			http.Error(w, "perpanjangan masa coba 1–90 hari", http.StatusBadRequest)
			return
		}
		l.TrialSampai = maxTime(now, l.TrialSampai).AddDate(0, 0, req.Hari)
		aksi = fmt.Sprintf("Perpanjang masa coba +%d hari", req.Hari)
	case "aktifkan":
		p, ok := paketDijual(req.Paket)
		if !ok {
			t.mu.Unlock()
			http.Error(w, "paket tidak dikenal", http.StatusBadRequest)
			return
		}
		l.AktifSampai = maxTime(now, l.AktifSampai).AddDate(0, 0, p.Hari)
		// Status "manual", bukan "lunas": tidak lewat Midtrans, jadi tidak membuka garansi 7 hari.
		l.Orders = append(slices.Clone(l.Orders), order{ID: fmt.Sprintf("MANUAL-%d-%d", t.id, now.Unix()), Paket: p.ID, Jumlah: p.Harga, Status: "manual", Waktu: now})
		aksi = "Aktifkan paket " + p.Nama + " (manual)"
	default:
		t.mu.Unlock()
		http.Error(w, "aksi harus trial atau aktifkan", http.StatusBadRequest)
		return
	}
	t.mu.Unlock()
	if err := s.saveFor(t); err != nil {
		t.mu.Lock()
		t.langganan = lama
		t.mu.Unlock()
		writeMasterErr(w, err)
		return
	}
	if err := s.catat(r, aksi, t.id, req.Alasan); err != nil {
		writeMasterErr(w, err)
		return
	}
	writeJSON(w, t.langgananStatus(time.Now()))
}

// nonNil: slice kosong dikirim sebagai [] (bukan null) supaya frontend tidak perlu menjaga null.
func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func maxTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

// --- pembayaran ---

type pembayaranRow struct {
	KlienID   int    `json:"klien_id"`
	KlienNama string `json:"klien_nama"`
	Order     order  `json:"order"`
	// Garansi: hanya pada order lunas pertama usaha itu (dasar garansi 7 hari).
	Garansi *garansiStatus `json:"garansi,omitempty"`
}

func (s *Server) handleMasterPembayaran(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	out := []pembayaranRow{}
	for _, k := range s.klienRows(aksesOf(r), now, false) { // pembayaran hanya butuh nama & order
		s.tenantsMu.RLock()
		t := s.tenants[k.ID]
		s.tenantsMu.RUnlock()
		t.mu.RLock()
		l := t.langganan
		t.mu.RUnlock()
		pertama := slices.IndexFunc(l.Orders, func(o order) bool { return o.Status == "lunas" })
		for i, o := range l.Orders {
			row := pembayaranRow{KlienID: k.ID, KlienNama: k.Nama, Order: o}
			if i == pertama {
				g := statusGaransi(l, now)
				row.Garansi = &g
			}
			out = append(out, row)
		}
	}
	slices.SortStableFunc(out, func(a, b pembayaranRow) int { return b.Order.Waktu.Compare(a.Order.Waktu) })
	writeJSON(w, out)
}

// --- tiket (laporan bug) ---

var statusTiket = []string{"baru", "dikerjakan", "selesai"}

type tiketRow struct {
	bugReport
	Nomor     string `json:"nomor"`
	KlienNama string `json:"klien_nama"`
}

func nomorTiket(id int) string { return fmt.Sprintf("BUG-%04d", id) }

// tiketRows: laporan bug dari klien yang boleh dilihat anggota, terbaru dulu.
func (s *Server) tiketRows(ak master.Akses) []tiketRow {
	nama := map[int]string{}
	s.tenantsMu.RLock()
	for id, t := range s.tenants {
		t.mu.RLock()
		if t.profile != nil {
			nama[id] = t.profile.NamaUsaha
		}
		nama[id] = namaKlien(t, nama[id])
		t.mu.RUnlock()
	}
	s.tenantsMu.RUnlock()
	b := s.bugs
	b.mu.Lock()
	defer b.mu.Unlock()
	out := []tiketRow{}
	for i := len(b.list) - 1; i >= 0; i-- {
		rep := b.list[i]
		if !ak.BolehKlien(rep.TenantID) {
			continue
		}
		rep.Status = cmp.Or(rep.Status, "baru")
		out = append(out, tiketRow{rep, nomorTiket(rep.ID), cmp.Or(nama[rep.TenantID], fmt.Sprintf("Usaha #%d", rep.TenantID))}) // usaha sudah dihapus
	}
	return out
}

func (s *Server) handleMasterTiket(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.tiketRows(aksesOf(r)))
}

func (s *Server) handleMasterTiketScreenshot(w http.ResponseWriter, r *http.Request) {
	ak := aksesOf(r)
	s.serveBugScreenshot(w, r, func(rep bugReport) bool { return ak.BolehKlien(rep.TenantID) })
}

// handleMasterUbahTiket: ganti status dan/atau kirim balasan yang terbaca pelapor.
func (s *Server) handleMasterUbahTiket(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	var req struct{ Status, Balasan string }
	if !decodeBody(w, r, &req) {
		return
	}
	req.Balasan = strings.TrimSpace(req.Balasan)
	if req.Status != "" && !slices.Contains(statusTiket, req.Status) || req.Status == "" && req.Balasan == "" || len([]rune(req.Balasan)) > 2000 {
		http.Error(w, "status harus baru, dikerjakan, atau selesai; balasan maks 2000 karakter", http.StatusBadRequest)
		return
	}
	u, ak := currentUser(r), aksesOf(r)
	b := s.bugs
	b.mu.Lock()
	if id < 1 || id > len(b.list) || !ak.BolehKlien(b.list[id-1].TenantID) {
		b.mu.Unlock()
		http.Error(w, "tiket tidak ditemukan", http.StatusNotFound)
		return
	}
	lama := b.list[id-1]
	rep := lama
	if req.Status != "" {
		rep.Status = req.Status
	}
	if req.Balasan != "" {
		rep.Balasan = append(slices.Clone(rep.Balasan), bugBalasan{Nama: u.Name, Isi: req.Balasan, Waktu: time.Now().UTC().Truncate(time.Second)})
	}
	b.list[id-1] = rep
	var err error
	switch {
	case s.db != nil:
		err = updateBugDB(s.db, rep)
	case b.path != "":
		err = writeFileAtomic(b.path, b.list)
	}
	if err != nil {
		b.list[id-1] = lama
	}
	b.mu.Unlock()
	if err != nil {
		writeMasterErr(w, fmt.Errorf("%w: %w", errSimpan, err))
		return
	}
	aksi := "Ubah status tiket " + nomorTiket(id) + " → " + rep.Status
	if req.Balasan != "" {
		aksi = "Balas tiket " + nomorTiket(id)
	}
	if err := s.catat(r, aksi, rep.TenantID, ""); err != nil {
		writeMasterErr(w, err)
		return
	}
	rep.Status = cmp.Or(rep.Status, "baru")
	writeJSON(w, rep)
}

// --- tim & role ---

type anggotaRow struct {
	User  auth.User `json:"user"`
	Role  string    `json:"role"`
	Klien []int     `json:"klien"`
}

type timResp struct {
	Anggota []anggotaRow      `json:"anggota"`
	Roles   []master.Role     `json:"roles"`
	Izin    []master.IzinInfo `json:"izin"`
	Klien   []orang           `json:"klien"`
}

func (s *Server) handleMasterTim(w http.ResponseWriter, r *http.Request) {
	out := timResp{Anggota: []anggotaRow{}, Izin: master.DaftarIzin, Klien: []orang{}}
	s.masterMu.Lock()
	st := s.master.Clone()
	s.masterMu.Unlock()
	out.Roles = st.Roles
	for _, u := range s.users.List(0) {
		row := anggotaRow{User: u, Klien: []int{}}
		if i := slices.IndexFunc(st.Anggota, func(a master.Anggota) bool { return a.UserID == u.ID }); i >= 0 {
			row.Role, row.Klien = st.Anggota[i].Role, nonNil(st.Anggota[i].Klien)
		}
		out.Anggota = append(out.Anggota, row)
	}
	for _, k := range s.klienRows(master.Akses{Role: master.Role{ID: master.RoleMaster}}, time.Now(), false) { // tim hanya butuh id & nama
		out.Klien = append(out.Klien, orang{k.ID, k.Nama})
	}
	writeJSON(w, out)
}

func (s *Server) handleMasterTambahAnggota(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Nama, Email, Password, Role string
		Klien                       []int
	}
	if !decodeBody(w, r, &req) {
		return
	}
	u, err := s.users.Create(auth.Input{Name: req.Nama, Email: req.Email, Password: req.Password, Role: auth.MP, Active: true})
	if err != nil {
		writeAuthErr(w, err)
		return
	}
	pelaku, ak := currentUser(r), aksesOf(r)
	err = s.ubahMaster(func(st *master.State) error {
		if err := st.SetAnggota(master.Anggota{UserID: u.ID, Role: req.Role, Klien: req.Klien}); err != nil {
			return err
		}
		st.Catat(master.Catatan{UserID: pelaku.ID, Nama: pelaku.Name, Peran: ak.Role.Nama, Aksi: "Tambah anggota tim " + u.Email, Keterangan: "role " + req.Role})
		return nil
	})
	if err != nil {
		_ = s.users.Delete(0, u.ID)
		writeMasterErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, anggotaRow{User: u, Role: req.Role, Klien: nonNil(req.Klien)})
}

func (s *Server) handleMasterUbahAnggota(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	var req struct {
		Role  string
		Klien []int
		Aktif bool
	}
	if !decodeBody(w, r, &req) {
		return
	}
	i := slices.IndexFunc(s.users.List(0), func(u auth.User) bool { return u.ID == id })
	if i < 0 {
		http.Error(w, "anggota tidak ditemukan", http.StatusNotFound)
		return
	}
	u := s.users.List(0)[i]
	pelaku, ak := currentUser(r), aksesOf(r)
	err := s.ubahMaster(func(st *master.State) error {
		if !req.Aktif && !st.MasterLain(id) {
			if a, ok := st.Akses(id); ok && a.Role.ID == master.RoleMaster {
				return master.ErrMasterTerakhir
			}
		}
		if err := st.SetAnggota(master.Anggota{UserID: id, Role: req.Role, Klien: req.Klien}); err != nil {
			return err
		}
		ket := "role " + req.Role
		if !req.Aktif {
			ket += ", dinonaktifkan"
		}
		st.Catat(master.Catatan{UserID: pelaku.ID, Nama: pelaku.Name, Peran: ak.Role.Nama, Aksi: "Ubah anggota tim " + u.Email, Keterangan: ket})
		return nil
	})
	if err != nil {
		writeMasterErr(w, err)
		return
	}
	if u.Active != req.Aktif {
		if _, err := s.users.Update(0, id, auth.Input{Name: u.Name, Email: u.Email, Role: auth.MP, Active: req.Aktif}); err != nil {
			writeAuthErr(w, err)
			return
		}
	}
	writeJSON(w, anggotaRow{User: u, Role: req.Role, Klien: nonNil(req.Klien)})
}

func (s *Server) handleMasterSimpanRole(w http.ResponseWriter, r *http.Request) {
	var req master.Role
	if !decodeBody(w, r, &req) {
		return
	}
	req.ID = r.PathValue("id") // kosong pada POST = role baru
	pelaku, ak := currentUser(r), aksesOf(r)
	var hasil master.Role
	err := s.ubahMaster(func(st *master.State) error {
		var err error
		if hasil, err = st.SimpanRole(req); err != nil {
			return err
		}
		aksi := "Ubah role " + hasil.Nama
		if req.ID == "" {
			aksi = "Buat role " + hasil.Nama
		}
		st.Catat(master.Catatan{UserID: pelaku.ID, Nama: pelaku.Name, Peran: ak.Role.Nama, Aksi: aksi, Keterangan: fmt.Sprint(hasil.Izin)})
		return nil
	})
	if err != nil {
		writeMasterErr(w, err)
		return
	}
	if req.ID == "" {
		w.WriteHeader(http.StatusCreated)
	}
	writeJSON(w, hasil)
}

func (s *Server) handleMasterHapusRole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	pelaku, ak := currentUser(r), aksesOf(r)
	err := s.ubahMaster(func(st *master.State) error {
		if err := st.HapusRole(id); err != nil {
			return err
		}
		st.Catat(master.Catatan{UserID: pelaku.ID, Nama: pelaku.Name, Peran: ak.Role.Nama, Aksi: "Hapus role " + id})
		return nil
	})
	if err != nil {
		writeMasterErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- audit ---

// maxAuditTampil: audit log terbaru yang dikirim ke layar.
const maxAuditTampil = 500

func (s *Server) handleMasterAudit(w http.ResponseWriter, r *http.Request) {
	s.masterMu.Lock()
	a := s.master.Audit
	out := make([]master.Catatan, 0, min(len(a), maxAuditTampil))
	for i := len(a) - 1; i >= 0 && len(out) < maxAuditTampil; i-- {
		out = append(out, a[i])
	}
	s.masterMu.Unlock()
	writeJSON(w, out)
}

func (s *Server) masterRoutes() {
	m := s.mux
	m.HandleFunc("GET /api/v1/master/ringkasan", s.guardMP(s.handleMasterRingkasan, izinKlien...))
	m.HandleFunc("GET /api/v1/master/klien", s.guardMP(s.handleMasterKlien, izinKlien...))
	m.HandleFunc("GET /api/v1/master/klien/{id}", s.guardMP(s.handleMasterKlienDetail, izinKlien...))
	m.HandleFunc("POST /api/v1/master/klien/{id}/langganan", s.guardMP(s.handleMasterLangganan, master.Langganan))
	m.HandleFunc("GET /api/v1/master/tiket", s.guardMP(s.handleMasterTiket, master.Tiket))
	m.HandleFunc("PUT /api/v1/master/tiket/{id}", s.guardMP(s.handleMasterUbahTiket, master.Tiket))
	m.HandleFunc("GET /api/v1/master/tiket/{id}/screenshot", s.guardMP(s.handleMasterTiketScreenshot, master.Tiket))
	m.HandleFunc("GET /api/v1/master/pembayaran", s.guardMP(s.handleMasterPembayaran, master.Pembayaran))
	m.HandleFunc("GET /api/v1/master/tim", s.guardMP(s.handleMasterTim, master.Tim))
	m.HandleFunc("POST /api/v1/master/tim", s.persisting(s.guardMP(s.handleMasterTambahAnggota, master.Tim)))
	m.HandleFunc("PUT /api/v1/master/tim/{id}", s.persisting(s.guardMP(s.handleMasterUbahAnggota, master.Tim)))
	m.HandleFunc("POST /api/v1/master/roles", s.guardMP(s.handleMasterSimpanRole, master.Tim))
	m.HandleFunc("PUT /api/v1/master/roles/{id}", s.guardMP(s.handleMasterSimpanRole, master.Tim))
	m.HandleFunc("DELETE /api/v1/master/roles/{id}", s.guardMP(s.handleMasterHapusRole, master.Tim))
	m.HandleFunc("GET /api/v1/master/audit", s.guardMP(s.handleMasterAudit, master.Audit))
	s.kelolaRoutes()
	m.HandleFunc("POST /api/v1/master/klien/{id}/lihat", s.guardMP(s.handleMulaiLihat, master.LihatSebagai))
	m.HandleFunc("DELETE /api/v1/master/lihat", s.guardMP(s.handleSelesaiLihat))
	m.HandleFunc("POST /api/v1/master/klien/{id}/koreksi", s.guardMP(s.handleUsulKoreksi, master.UsulKoreksi))
	m.HandleFunc("GET /api/v1/master/koreksi", s.guardMP(s.handleMasterKoreksi, master.UsulKoreksi, master.KlienSemua))
}
