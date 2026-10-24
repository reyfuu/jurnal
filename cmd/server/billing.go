package main

import (
	"cmp"
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
	"pajak/internal/midtrans"
)

// Langganan aplikasi per usaha, dibayar lewat Midtrans Snap. Usaha baru wajib bayar dulu
// setelah onboarding: tanpa langganan aktif hanya bisa mengisi profil, melihat data, dan
// membayar (lihat lockedWrite). TrialSampai tetap dibaca untuk usaha lama yang masih masa coba.

type paket struct {
	ID    string `json:"id"`
	Nama  string `json:"nama"`
	Harga int64  `json:"harga"`
	Hari  int    `json:"hari"`
	Hemat string `json:"hemat,omitempty"`
	// Tingkat: starter | pro | premium (kosong = paket lama satu tingkat).
	Tingkat string `json:"tingkat,omitempty"`
	// HargaNormal: harga sebelum promo, ditampilkan dicoret di halaman harga.
	HargaNormal int64 `json:"harga_normal,omitempty"`
}

// paketList: paket yang dijual (3 tingkat × bulanan/tahunan, sama dengan landing page). Starter
// membuka seluruh aplikasi kecuali fitur Pro (lihat pro); Pro & Premium membuka semuanya.
// Ubah harga di sini saja.
var paketList = []paket{
	{"starter-bulanan", "Starter Bulanan", 199_000, 30, "", "starter", 399_000},
	{"starter-tahunan", "Starter Tahunan", 1_990_000, 365, "Hemat 2 bulan", "starter", 4_790_000},
	{"pro-bulanan", "Pro Bulanan", 499_000, 30, "", "pro", 999_000},
	{"pro-tahunan", "Pro Tahunan", 5_000_000, 365, "Hemat 2 bulan", "pro", 11_990_000},
	{"premium-bulanan", "Premium Master Bulanan", 1_490_000, 30, "", "premium", 2_990_000},
	{"premium-tahunan", "Premium Master Tahunan", 14_900_000, 365, "Hemat 2 bulan", "premium", 35_900_000},
}

// paketLama: tidak dijual lagi, tetap dikenali untuk order yang sudah ada (notifikasi Midtrans
// yang terlambat, riwayat, MRR di Panel Master).
var paketLama = []paket{
	{ID: "bulanan", Nama: "Bulanan", Harga: 99_000, Hari: 30},
	{ID: "tahunan", Nama: "Tahunan", Harga: 990_000, Hari: 365},
}

// paketDijual: hanya paket yang sedang dijual (checkout, aktivasi manual).
func paketDijual(id string) (paket, bool) {
	i := slices.IndexFunc(paketList, func(p paket) bool { return p.ID == id })
	if i < 0 {
		return paket{}, false
	}
	return paketList[i], true
}

// findPaket: paket dijual atau paket lama, untuk order yang sudah tercatat.
func findPaket(id string) (paket, bool) {
	if p, ok := paketDijual(id); ok {
		return p, true
	}
	i := slices.IndexFunc(paketLama, func(p paket) bool { return p.ID == id })
	if i < 0 {
		return paket{}, false
	}
	return paketLama[i], true
}

type order struct {
	ID          string    `json:"id"`
	Paket       string    `json:"paket"`
	Jumlah      int64     `json:"jumlah"`
	Status      string    `json:"status"` // pending | lunas | gagal
	Waktu       time.Time `json:"waktu"`
	DibayarPada time.Time `json:"dibayar_pada,omitzero"` // saat lunas; awal masa garansi
	// Token Snap disimpan supaya checkout ulang paket yang sama memakai transaksi yang sama
	// (lihat handleCheckout). Tidak dikirim ke GET /billing.
	SnapToken   string `json:"snap_token,omitempty"`
	RedirectURL string `json:"redirect_url,omitempty"`
}

// snapBerlaku: masa berlaku token Snap bawaan Midtrans; order pending lebih tua dibuat ulang.
const snapBerlaku = 24 * time.Hour

type langganan struct {
	TrialSampai time.Time `json:"trial_sampai"`
	AktifSampai time.Time `json:"aktif_sampai,omitzero"`
	Orders      []order   `json:"orders"`
	// Garansi 7 hari uang kembali (lihat statusGaransi).
	EksporPertama  time.Time `json:"ekspor_pertama,omitzero"` // ekspor data pertama: garansi gugur
	GaransiDiklaim time.Time `json:"garansi_diklaim,omitzero"`
	GaransiAlasan  string    `json:"garansi_alasan,omitempty"`
}

// Garansi 7 hari uang kembali: berlaku untuk pembayaran PERTAMA usaha, 7×24 jam sejak lunas,
// dan gugur begitu usaha mengekspor datanya (laporan Excel, PDF invoice). Klaim menghentikan
// akses saat itu juga; operator mengembalikan dana manual lewat dashboard Midtrans.
const garansiDurasi = 7 * 24 * time.Hour

const (
	garansiBelumBayar   = "belum ada pembayaran"
	garansiLewat        = "sudah lewat 7 hari sejak pembayaran pertama"
	garansiSudahEkspor  = "data sudah pernah diekspor"
	garansiSudahDiklaim = "garansi sudah diklaim"
)

type garansiStatus struct {
	Berlaku bool      `json:"berlaku"`
	Sampai  time.Time `json:"sampai,omitzero"`
	Alasan  string    `json:"alasan,omitempty"` // kenapa tidak berlaku
	OrderID string    `json:"order_id,omitempty"`
}

func statusGaransi(l langganan, now time.Time) garansiStatus {
	i := slices.IndexFunc(l.Orders, func(o order) bool { return o.Status == "lunas" })
	if i < 0 {
		return garansiStatus{Alasan: garansiBelumBayar}
	}
	o := l.Orders[i] // pesanan tersimpan urut waktu: lunas pertama = pembayaran pertama
	g := garansiStatus{Sampai: o.DibayarPada.Add(garansiDurasi), OrderID: o.ID}
	switch {
	case !l.GaransiDiklaim.IsZero():
		g.Alasan = garansiSudahDiklaim
	case !l.EksporPertama.IsZero():
		g.Alasan = garansiSudahEkspor
	case o.DibayarPada.IsZero() || !now.Before(g.Sampai): // nol = lunas sebelum fitur garansi ada
		g.Alasan = garansiLewat
	default:
		g.Berlaku = true
	}
	return g
}

// handleKlaimGaransi: Owner mengajukan uang kembali. Akses berhenti saat itu juga.
func (s *Server) handleKlaimGaransi(w http.ResponseWriter, r *http.Request) {
	var req struct{ Alasan string }
	if !decodeBody(w, r, &req) {
		return
	}
	t, now := s.tenantOf(r), time.Now()
	t.mu.Lock()
	g := statusGaransi(t.langganan, now)
	if !g.Berlaku {
		t.mu.Unlock()
		http.Error(w, "Garansi tidak berlaku: "+g.Alasan, http.StatusConflict)
		return
	}
	l := &t.langganan
	l.GaransiDiklaim, l.GaransiAlasan = now, clip(strings.TrimSpace(req.Alasan), 500)
	l.AktifSampai = now
	if l.TrialSampai.After(now) {
		l.TrialSampai = now
	}
	t.mu.Unlock()
	log.Printf("GARANSI DIKLAIM usaha %d order %s (%s): kembalikan dana manual lewat dashboard Midtrans", t.id, g.OrderID, currentUser(r).Email)
	writeJSON(w, s.newMe(currentUser(r)))
}

// exporting: ekspor data pertama yang berhasil menggugurkan garansi. Respons ditahan sampai
// tanda itu tersimpan, supaya berkas tidak pernah sampai tanpa garansinya ikut gugur.
func (s *Server) exporting(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Tim yang sedang lihat-sebagai tidak boleh menggugurkan garansi klien lewat ekspor.
		if sedangLihat(r) {
			http.Error(w, "ekspor tidak tersedia di mode lihat sebagai klien", http.StatusForbidden)
			return
		}
		buf := &bufferedResponse{header: http.Header{}}
		h(buf, r)
		if buf.status == 0 || buf.status == http.StatusOK {
			t := s.tenantOf(r)
			t.mu.Lock()
			pertama := t.langganan.EksporPertama.IsZero()
			if pertama {
				t.langganan.EksporPertama = time.Now()
			}
			t.mu.Unlock()
			if pertama {
				if err := s.saveFor(t); err != nil {
					http.Error(w, "gagal menyimpan, coba lagi", http.StatusServiceUnavailable)
					return
				}
			}
		}
		buf.flush(w)
	}
}

type langgananStatus struct {
	Status string    `json:"status"` // trial | aktif | habis
	Sampai time.Time `json:"sampai,omitzero"`
	// Tingkat: tingkat paket terbayar terakhir saat aktif (starter | pro | premium; kosong =
	// masa coba, habis, atau paket lama). Pro: fitur Pro terbuka (masa coba, Pro, Premium, paket lama).
	Tingkat string `json:"tingkat,omitempty"`
	Pro     bool   `json:"pro"`
}

// status: usaha 1 (milik operator) selalu aktif.
func (t *tenant) langgananStatus(now time.Time) langgananStatus {
	if t.id == 1 {
		return langgananStatus{Status: "aktif", Pro: true}
	}
	t.mu.RLock()
	l := t.langganan
	t.mu.RUnlock()
	switch {
	case now.Before(l.AktifSampai):
		tingkat := ""
		for _, o := range slices.Backward(l.Orders) { // pesanan urut waktu: terbayar terakhir menentukan
			if o.Status == "lunas" || o.Status == "manual" {
				p, _ := findPaket(o.Paket)
				tingkat = p.Tingkat
				break
			}
		}
		return langgananStatus{Status: "aktif", Sampai: l.AktifSampai, Tingkat: tingkat, Pro: tingkat != "starter"}
	case now.Before(l.TrialSampai):
		return langgananStatus{Status: "trial", Sampai: l.TrialSampai, Pro: true}
	}
	return langgananStatus{Status: "habis", Sampai: cmp.Or(l.AktifSampai, l.TrialSampai)}
}

// pro: fitur khusus paket Pro/Premium. Paket lain mendapat 402 berisi ajakan upgrade. Tim
// Master Pajak (lihat-sebagai) selalu boleh. Dipasang di dalam guard.
func (s *Server) pro(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if currentUser(r).Role != auth.MP && !s.tenantOf(r).langgananStatus(time.Now()).Pro {
			http.Error(w, "Fitur ini khusus paket Pro. Upgrade di menu Langganan untuk membukanya.", http.StatusPaymentRequired)
			return
		}
		h(w, r)
	}
}

// lockedWrite: langganan belum/tidak aktif → hanya baca. Pembayaran, sesi, lapor bug,
// profil usaha (onboarding + cek awal), serta alat gratis (kalkulator & perencanaan pajak:
// POST yang hanya menghitung, tidak menyimpan) tetap jalan.
// ponytail: baca (GET) tidak dikunci di server; layar bayar dipaksa di frontend (App.tsx).
func (s *Server) lockedWrite(r *http.Request, tenantID int) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return false
	}
	for _, p := range []string{"/api/v1/billing", "/api/v1/auth/", "/api/v1/bugs", "/api/v1/business", "/api/v1/kalkulator/", "/api/v1/tax/planning"} {
		if strings.HasPrefix(r.URL.Path, p) {
			return false
		}
	}
	s.tenantsMu.RLock()
	t := s.tenants[tenantID]
	s.tenantsMu.RUnlock()
	return t != nil && t.langgananStatus(time.Now()).Status == "habis"
}

func (s *Server) midtransClient() *midtrans.Client {
	key := os.Getenv("MIDTRANS_SERVER_KEY")
	if key == "" {
		return nil
	}
	return &midtrans.Client{ServerKey: key, Production: os.Getenv("MIDTRANS_PRODUCTION") == "true", SnapURL: os.Getenv("MIDTRANS_SNAP_URL"), APIURL: os.Getenv("MIDTRANS_API_URL")}
}

func (s *Server) handlePlans(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"paket": paketList})
}

func (s *Server) handleBilling(w http.ResponseWriter, r *http.Request) {
	t := s.tenantOf(r)
	t.mu.RLock()
	orders := append([]order{}, t.langganan.Orders...) // bukan nil: JSON [] supaya frontend aman
	garansi := statusGaransi(t.langganan, time.Now())
	t.mu.RUnlock()
	for i := range orders {
		orders[i].SnapToken, orders[i].RedirectURL = "", ""
	}
	slices.Reverse(orders) // terbaru dulu
	writeJSON(w, map[string]any{
		"langganan":  t.langgananStatus(time.Now()),
		"garansi":    garansi,
		"paket":      paketList,
		"orders":     orders,
		"client_key": os.Getenv("MIDTRANS_CLIENT_KEY"),
		"snap_js":    midtrans.SnapJS(os.Getenv("MIDTRANS_PRODUCTION") == "true"),
		"aktif":      s.midtransClient() != nil,
	})
}

func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	var req struct{ Paket string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	p, ok := paketDijual(req.Paket)
	if !ok {
		http.Error(w, "paket tidak dikenal", http.StatusBadRequest)
		return
	}
	mt := s.midtransClient()
	if mt == nil {
		http.Error(w, "pembayaran belum dikonfigurasi", http.StatusServiceUnavailable)
		return
	}
	u, t := currentUser(r), s.tenantOf(r)
	// Idempoten: klik ganda, retry, atau tab lain untuk paket yang sama memakai order pending
	// yang masih berlaku, supaya tidak ada dua transaksi yang bisa dibayar dua kali.
	t.checkoutMu.Lock()
	defer t.checkoutMu.Unlock()
	t.mu.RLock()
	i := slices.IndexFunc(t.langganan.Orders, func(o order) bool {
		return o.Status == "pending" && o.Paket == p.ID && o.SnapToken != "" && time.Since(o.Waktu) < snapBerlaku
	})
	var lama order
	if i >= 0 {
		lama = t.langganan.Orders[i]
	}
	t.mu.RUnlock()
	if i >= 0 {
		writeJSON(w, map[string]any{"order_id": lama.ID, "token": lama.SnapToken, "redirect_url": lama.RedirectURL})
		return
	}
	o := order{ID: fmt.Sprintf("LB-%d-%d", t.id, time.Now().UnixNano()), Paket: p.ID, Jumlah: p.Harga, Status: "pending", Waktu: time.Now()}
	snap, err := mt.CreateSnap(r.Context(), midtrans.SnapRequest{
		OrderID: o.ID, Amount: p.Harga, Name: u.Name, Email: u.Email,
		Items: []midtrans.Item{{ID: p.ID, Price: p.Harga, Quantity: 1, Name: "MasterPajak AI " + p.Nama}},
	})
	if err != nil {
		log.Printf("midtrans checkout %s: %v", o.ID, err)
		http.Error(w, "gagal menghubungi Midtrans, coba lagi", http.StatusBadGateway)
		return
	}
	o.SnapToken, o.RedirectURL = snap.Token, snap.RedirectURL
	t.mu.Lock()
	t.langganan.Orders = append(t.langganan.Orders, o)
	t.mu.Unlock()
	writeJSON(w, map[string]any{"order_id": o.ID, "token": snap.Token, "redirect_url": snap.RedirectURL})
}

// handleMidtransNotify: webhook Midtrans (tanpa login). Signature diverifikasi, lalu status
// dikonfirmasi langsung ke Midtrans sebelum langganan diperpanjang; idempoten per order.
func (s *Server) handleMidtransNotify(w http.ResponseWriter, r *http.Request) {
	mt := s.midtransClient()
	if mt == nil {
		http.Error(w, "pembayaran belum dikonfigurasi", http.StatusServiceUnavailable)
		return
	}
	var n midtrans.Notification
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&n); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !mt.VerifySignature(n) {
		http.Error(w, "signature tidak valid", http.StatusForbidden)
		return
	}
	st, err := mt.Status(r.Context(), n.OrderID)
	if err != nil {
		log.Printf("midtrans status %s: %v", n.OrderID, err)
		http.Error(w, "gagal konfirmasi status", http.StatusBadGateway) // Midtrans akan mengirim ulang
		return
	}
	if err := s.applyPayment(st, time.Now()); err != nil {
		log.Printf("midtrans notify %s: %v", n.OrderID, err)
		if errors.Is(err, errSimpan) {
			http.Error(w, "gagal menyimpan", http.StatusServiceUnavailable) // Midtrans akan mengirim ulang
			return
		}
	}
	w.WriteHeader(http.StatusOK) // order tak dikenal tetap 200 supaya Midtrans tidak mengulang terus
}

// applyPayment mencatat hasil Status API ke order; lunas memperpanjang langganan sekali saja.
func (s *Server) applyPayment(st midtrans.Notification, now time.Time) error {
	var tid int
	if _, err := fmt.Sscanf(st.OrderID, "LB-%d-", &tid); err != nil {
		return fmt.Errorf("order_id asing: %q", st.OrderID)
	}
	s.tenantsMu.RLock()
	t := s.tenants[tid]
	s.tenantsMu.RUnlock()
	if t == nil {
		return fmt.Errorf("usaha %d tidak ada", tid)
	}
	t.mu.Lock()
	i := slices.IndexFunc(t.langganan.Orders, func(o order) bool { return o.ID == st.OrderID })
	if i < 0 {
		t.mu.Unlock()
		return fmt.Errorf("order tidak dikenal")
	}
	o := &t.langganan.Orders[i]
	switch {
	case o.Status == "lunas":
		// sudah diproses
	case st.Paid():
		bayar, _ := strconv.ParseFloat(st.GrossAmount, 64)
		p, ok := findPaket(o.Paket)
		if !ok || int64(bayar) != o.Jumlah {
			t.mu.Unlock()
			return fmt.Errorf("jumlah %s tidak cocok dengan order %d", st.GrossAmount, o.Jumlah)
		}
		o.Status, o.DibayarPada = "lunas", now
		l := &t.langganan
		mulai := latest(now, l.AktifSampai, l.TrialSampai)
		l.AktifSampai = mulai.Add(time.Duration(p.Hari) * 24 * time.Hour)
	case slices.Contains([]string{"deny", "cancel", "expire", "failure"}, st.TransactionStatus):
		o.Status = "gagal"
	}
	t.mu.Unlock()
	return s.saveFor(t)
}

// latest: waktu paling akhir (builtin max tidak menerima time.Time).
func latest(ts ...time.Time) time.Time {
	return slices.MaxFunc(ts, func(a, b time.Time) int { return a.Compare(b) })
}

// handleBillingRefresh: tombol "cek status" bila notifikasi belum sampai (mis. webhook belum diatur).
func (s *Server) handleBillingRefresh(w http.ResponseWriter, r *http.Request) {
	mt := s.midtransClient()
	if mt == nil {
		http.Error(w, "pembayaran belum dikonfigurasi", http.StatusServiceUnavailable)
		return
	}
	t := s.tenantOf(r)
	t.mu.RLock()
	var pending []string
	for _, o := range t.langganan.Orders {
		if o.Status == "pending" {
			pending = append(pending, o.ID)
		}
	}
	t.mu.RUnlock()
	for _, id := range pending {
		if st, err := mt.Status(r.Context(), id); err == nil {
			_ = s.applyPayment(st, time.Now())
		}
	}
	s.handleBilling(w, r)
}
