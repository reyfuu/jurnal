package main

import (
	"cmp"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pajak/internal/midtrans"
)

func tenantByEmail(t *testing.T, srv *Server, c *http.Cookie) *tenant {
	t.Helper()
	var me meResp
	_ = json.NewDecoder(do(srv, c, http.MethodGet, "/api/v1/auth/me", "").Body).Decode(&me)
	return srv.tenants[me.User.TenantID]
}

// aktifkan: anggap usaha milik sesi c sudah membayar langganan 30 hari.
func aktifkan(t *testing.T, srv *Server, c *http.Cookie) {
	t.Helper()
	tenantByEmail(t, srv, c).langganan.AktifSampai = time.Now().Add(30 * 24 * time.Hour)
}

func TestBayarDuluSetelahOnboarding(t *testing.T) {
	srv := NewServer()
	qa := register(t, srv, registerBody)
	var me meResp
	_ = json.NewDecoder(do(srv, qa, http.MethodGet, "/api/v1/auth/me", "").Body).Decode(&me)
	if me.Langganan.Status != "habis" {
		t.Fatalf("usaha baru harus bayar dulu (habis), got %+v", me.Langganan)
	}
	// onboarding tetap jalan sebelum bayar
	if rec := do(srv, qa, http.MethodPut, "/api/v1/business", profileBody); rec.Code != http.StatusOK {
		t.Fatalf("belum bayar: simpan profil (onboarding) harus boleh, got %d %s", rec.Code, rec.Body)
	}
	cek := `{"jenis_usaha":"perdagangan","bentuk_usaha":"op","ptkp":"TK/0","omzet_tahunan":6000000000,"laba_sebelum_pajak":60000000}`
	if rec := do(srv, qa, http.MethodPost, "/api/v1/business/cek-awal", cek); rec.Code != http.StatusOK {
		t.Fatalf("belum bayar: cek awal di wizard harus boleh, got %d", rec.Code)
	}
	tx := `{"tanggal":"2026-08-20","jenis":"Penjualan","nominal":5000,"keterangan":"x","lunas":true}`
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions", tx); rec.Code != http.StatusPaymentRequired {
		t.Fatalf("belum bayar: tulis harus 402, got %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodGet, "/api/v1/transactions", ""); rec.Code != http.StatusOK {
		t.Fatalf("belum bayar: baca tetap boleh, got %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/bugs", `{}`); rec.Code == http.StatusPaymentRequired {
		t.Fatal("belum bayar: lapor bug tetap boleh")
	}

	aktifkan(t, srv, qa)
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions", tx); rec.Code != http.StatusCreated {
		t.Fatalf("sudah bayar: boleh mencatat, got %d", rec.Code)
	}
	owner := login(t, srv, "owner@kedaikopisenja.id")
	if rec := do(srv, owner, http.MethodPost, "/api/v1/transactions", tx); rec.Code != http.StatusCreated {
		t.Fatalf("usaha 1 tidak pernah terkunci: %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/auth/logout", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("logout: %d", rec.Code)
	}
}

// fakeMidtrans: Snap & Status API palsu; status order bisa diatur dari test, opsional dengan
// jumlah bayar ("settlement:990000.00"); bawaan harga Starter Bulanan. Mengembalikan jumlah
// panggilan Snap.
func fakeMidtrans(t *testing.T, status map[string]string) *atomic.Int32 {
	t.Helper()
	snap := &atomic.Int32{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/snap/v1/transactions" {
			snap.Add(1)
			_, _ = w.Write([]byte(`{"token":"tok-uji","redirect_url":"https://bayar"}`))
			return
		}
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v2/"), "/status")
		st, ok := status[id]
		if !ok {
			http.NotFound(w, r)
			return
		}
		st, jumlah, _ := strings.Cut(st, ":")
		_ = json.NewEncoder(w).Encode(map[string]string{"order_id": id, "transaction_status": st, "gross_amount": cmp.Or(jumlah, "199000.00"), "status_code": "200"})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("MIDTRANS_SERVER_KEY", "SK-uji")
	t.Setenv("MIDTRANS_SNAP_URL", srv.URL)
	t.Setenv("MIDTRANS_API_URL", srv.URL)
	return snap
}

func checkout(t *testing.T, srv *Server, c *http.Cookie, paket string) string {
	t.Helper()
	var co struct {
		OrderID string `json:"order_id"`
		Token   string `json:"token"`
	}
	rec := do(srv, c, http.MethodPost, "/api/v1/billing/checkout", `{"paket":"`+paket+`"}`)
	_ = json.NewDecoder(rec.Body).Decode(&co)
	if rec.Code != http.StatusOK || co.OrderID == "" || co.Token == "" {
		t.Fatalf("checkout %s: %d %+v", paket, rec.Code, co)
	}
	return co.OrderID
}

// Checkout idempoten: paket sama yang masih pending memakai order & token Snap yang sama.
func TestCheckoutIdempoten(t *testing.T) {
	snap := fakeMidtrans(t, map[string]string{})
	srv := NewServer()
	qa := register(t, srv, registerBody)
	tn := tenantByEmail(t, srv, qa)

	a := checkout(t, srv, qa, "starter-bulanan")
	if b := checkout(t, srv, qa, "starter-bulanan"); b != a || snap.Load() != 1 || len(tn.langganan.Orders) != 1 {
		t.Fatalf("klik ulang: order %s vs %s, snap %d, orders %d", a, b, snap.Load(), len(tn.langganan.Orders))
	}
	if c := checkout(t, srv, qa, "pro-bulanan"); c == a || len(tn.langganan.Orders) != 2 {
		t.Fatalf("paket lain harus order baru: %s, orders %d", c, len(tn.langganan.Orders))
	}
	tn.langganan.Orders[0].Waktu = time.Now().Add(-snapBerlaku - time.Minute)
	if d := checkout(t, srv, qa, "starter-bulanan"); d == a {
		t.Fatal("order pending kedaluwarsa harus diganti order baru")
	}
	tn.langganan.Orders[len(tn.langganan.Orders)-1].Status = "gagal"
	n := len(tn.langganan.Orders)
	checkout(t, srv, qa, "starter-bulanan")
	if len(tn.langganan.Orders) != n+1 {
		t.Fatal("order gagal tidak boleh dipakai ulang")
	}

	var b struct{ Orders []order }
	_ = json.NewDecoder(do(srv, qa, http.MethodGet, "/api/v1/billing", "").Body).Decode(&b)
	for _, o := range b.Orders {
		if o.SnapToken != "" || o.RedirectURL != "" {
			t.Fatalf("token Snap bocor ke GET /billing: %+v", o)
		}
	}
}

// Dua checkout bersamaan (klik ganda / dua tab) tetap satu order.
func TestCheckoutBersamaan(t *testing.T) {
	snap := fakeMidtrans(t, map[string]string{})
	srv := NewServer()
	qa := register(t, srv, registerBody)
	var wg sync.WaitGroup
	ids := make([]string, 5)
	for i := range ids {
		wg.Add(1)
		go func() { defer wg.Done(); ids[i] = checkout(t, srv, qa, "starter-bulanan") }()
	}
	wg.Wait()
	if n := len(tenantByEmail(t, srv, qa).langganan.Orders); n != 1 || snap.Load() != 1 {
		t.Fatalf("checkout bersamaan: %d order, snap %d, ids %v", n, snap.Load(), ids)
	}
}

func notify(srv *Server, orderID, signature string) int {
	if signature == "" {
		signature = midtrans.Signature(orderID, "200", "99000.00", "SK-uji")
	}
	body, _ := json.Marshal(map[string]string{"order_id": orderID, "status_code": "200", "gross_amount": "99000.00", "signature_key": signature, "transaction_status": "settlement"})
	return do(srv, nil, http.MethodPost, "/api/v1/billing/midtrans/notify", string(body)).Code
}

func TestCheckoutDanNotifikasi(t *testing.T) {
	status := map[string]string{}
	fakeMidtrans(t, status)
	srv := NewServer()
	qa := register(t, srv, registerBody)
	if rec := do(srv, qa, http.MethodPost, "/api/v1/billing/checkout", `{"paket":"emas"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("paket asing: want 400, got %d", rec.Code)
	}
	var co struct {
		OrderID string `json:"order_id"`
		Token   string `json:"token"`
	}
	rec := do(srv, qa, http.MethodPost, "/api/v1/billing/checkout", `{"paket":"starter-bulanan"}`)
	_ = json.NewDecoder(rec.Body).Decode(&co)
	if rec.Code != http.StatusOK || co.OrderID == "" {
		t.Fatalf("checkout: %d %+v", rec.Code, co)
	}
	tn := tenantByEmail(t, srv, qa)

	if code := notify(srv, co.OrderID, "palsu"); code != http.StatusForbidden {
		t.Fatalf("signature palsu: want 403, got %d", code)
	}
	status[co.OrderID] = "pending"
	if notify(srv, co.OrderID, ""); tn.langgananStatus(time.Now()).Status != "habis" {
		t.Fatal("pending tidak boleh mengaktifkan")
	}
	status[co.OrderID] = "settlement"
	notify(srv, co.OrderID, "")
	notify(srv, co.OrderID, "") // notifikasi ulang: tidak boleh memperpanjang dua kali
	want := time.Now().Add(30 * 24 * time.Hour)
	if got := tn.langganan.AktifSampai; got.Sub(want).Abs() > time.Second {
		t.Fatalf("aktif sampai %v, want %v (sekarang + 30 hari, sekali)", got, want)
	}
	var b struct {
		Langganan langgananStatus `json:"langganan"`
		Orders    []order         `json:"orders"`
	}
	_ = json.NewDecoder(do(srv, qa, http.MethodGet, "/api/v1/billing", "").Body).Decode(&b)
	if b.Langganan.Status != "aktif" || len(b.Orders) != 1 || b.Orders[0].Status != "lunas" {
		t.Fatalf("billing: %+v", b)
	}
}

func TestCheckoutTanpaKonfigurasi(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", "")
	srv := NewServer()
	qa := register(t, srv, registerBody)
	if rec := do(srv, qa, http.MethodPost, "/api/v1/billing/checkout", `{"paket":"starter-bulanan"}`); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("tanpa key: want 503, got %d", rec.Code)
	}
	if body := do(srv, qa, http.MethodGet, "/api/v1/billing", "").Body.String(); !strings.Contains(body, `"orders":[]`) {
		t.Fatalf("usaha tanpa pembayaran harus orders [] (bukan null): %s", body)
	}
	if rec := do(srv, nil, http.MethodGet, "/api/v1/billing/plans", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "bulanan") {
		t.Fatalf("plans publik: %d", rec.Code)
	}
}
