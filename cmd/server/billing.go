package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

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
}

// Paket: harga contoh, ubah di sini saja.
var paketList = []paket{
	{"bulanan", "Bulanan", 99_000, 30, ""},
	{"tahunan", "Tahunan", 990_000, 365, "Hemat 2 bulan"},
}

func findPaket(id string) (paket, bool) {
	i := slices.IndexFunc(paketList, func(p paket) bool { return p.ID == id })
	if i < 0 {
		return paket{}, false
	}
	return paketList[i], true
}

type order struct {
	ID     string    `json:"id"`
	Paket  string    `json:"paket"`
	Jumlah int64     `json:"jumlah"`
	Status string    `json:"status"` // pending | lunas | gagal
	Waktu  time.Time `json:"waktu"`
}

type langganan struct {
	TrialSampai time.Time `json:"trial_sampai"`
	AktifSampai time.Time `json:"aktif_sampai,omitzero"`
	Orders      []order   `json:"orders"`
}

type langgananStatus struct {
	Status string    `json:"status"` // trial | aktif | habis
	Sampai time.Time `json:"sampai,omitzero"`
}

// status: usaha 1 (milik operator) selalu aktif.
func (t *tenant) langgananStatus(now time.Time) langgananStatus {
	if t.id == 1 {
		return langgananStatus{Status: "aktif"}
	}
	t.mu.RLock()
	l := t.langganan
	t.mu.RUnlock()
	switch {
	case now.Before(l.AktifSampai):
		return langgananStatus{"aktif", l.AktifSampai}
	case now.Before(l.TrialSampai):
		return langgananStatus{"trial", l.TrialSampai}
	}
	return langgananStatus{"habis", cmp.Or(l.AktifSampai, l.TrialSampai)}
}

// lockedWrite: langganan belum/tidak aktif → hanya baca. Pembayaran, sesi, lapor bug, dan
// profil usaha (onboarding + cek awal) tetap jalan.
// ponytail: baca (GET) tidak dikunci di server; layar bayar dipaksa di frontend (App.tsx).
func (s *Server) lockedWrite(r *http.Request, tenantID int) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return false
	}
	for _, p := range []string{"/api/v1/billing", "/api/v1/auth/", "/api/v1/bugs", "/api/v1/business"} {
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
	t.mu.RUnlock()
	slices.Reverse(orders) // terbaru dulu
	writeJSON(w, map[string]any{
		"langganan":  t.langgananStatus(time.Now()),
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
	p, ok := findPaket(req.Paket)
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
	o := order{ID: fmt.Sprintf("LB-%d-%d", t.id, time.Now().UnixNano()), Paket: p.ID, Jumlah: p.Harga, Status: "pending", Waktu: time.Now()}
	snap, err := mt.CreateSnap(r.Context(), midtrans.SnapRequest{
		OrderID: o.ID, Amount: p.Harga, Name: u.Name, Email: u.Email,
		Items: []midtrans.Item{{ID: p.ID, Price: p.Harga, Quantity: 1, Name: "Laporan Bisnis " + p.Nama}},
	})
	if err != nil {
		log.Printf("midtrans checkout %s: %v", o.ID, err)
		http.Error(w, "gagal menghubungi Midtrans, coba lagi", http.StatusBadGateway)
		return
	}
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
		o.Status = "lunas"
		l := &t.langganan
		mulai := latest(now, l.AktifSampai, l.TrialSampai)
		l.AktifSampai = mulai.Add(time.Duration(p.Hari) * 24 * time.Hour)
	case slices.Contains([]string{"deny", "cancel", "expire", "failure"}, st.TransactionStatus):
		o.Status = "gagal"
	}
	t.mu.Unlock()
	s.save()
	return nil
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
