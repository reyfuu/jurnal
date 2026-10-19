package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestTrialLaluTerkunci(t *testing.T) {
	srv := NewServer()
	qa := register(t, srv, registerBody)
	var me meResp
	_ = json.NewDecoder(do(srv, qa, http.MethodGet, "/api/v1/auth/me", "").Body).Decode(&me)
	if me.Langganan.Status != "trial" || time.Until(me.Langganan.Sampai) < 71*time.Hour {
		t.Fatalf("usaha baru harus trial 3 hari: %+v", me.Langganan)
	}
	tx := `{"tanggal":"2026-08-20","jenis":"Penjualan","nominal":5000,"keterangan":"x","lunas":true}`
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions", tx); rec.Code != http.StatusCreated {
		t.Fatalf("trial boleh mencatat: %d", rec.Code)
	}

	tenantByEmail(t, srv, qa).langganan.TrialSampai = time.Now().Add(-time.Minute)
	if rec := do(srv, qa, http.MethodPost, "/api/v1/transactions", tx); rec.Code != http.StatusPaymentRequired {
		t.Fatalf("habis: tulis harus 402, got %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodGet, "/api/v1/transactions", ""); rec.Code != http.StatusOK {
		t.Fatalf("habis: baca tetap boleh, got %d", rec.Code)
	}
	if rec := do(srv, qa, http.MethodPost, "/api/v1/auth/logout", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("habis: logout tetap boleh, got %d", rec.Code)
	}
	owner := login(t, srv, "owner@kedaikopisenja.id")
	if rec := do(srv, owner, http.MethodPost, "/api/v1/transactions", tx); rec.Code != http.StatusCreated {
		t.Fatalf("usaha 1 tidak pernah terkunci: %d", rec.Code)
	}
}

// fakeMidtrans: Snap & Status API palsu; status order bisa diatur dari test.
func fakeMidtrans(t *testing.T, status map[string]string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/snap/v1/transactions" {
			_, _ = w.Write([]byte(`{"token":"tok-uji","redirect_url":"https://bayar"}`))
			return
		}
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v2/"), "/status")
		st, ok := status[id]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"order_id": id, "transaction_status": st, "gross_amount": "99000.00", "status_code": "200"})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("MIDTRANS_SERVER_KEY", "SK-uji")
	t.Setenv("MIDTRANS_SNAP_URL", srv.URL)
	t.Setenv("MIDTRANS_API_URL", srv.URL)
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
	rec := do(srv, qa, http.MethodPost, "/api/v1/billing/checkout", `{"paket":"bulanan"}`)
	_ = json.NewDecoder(rec.Body).Decode(&co)
	if rec.Code != http.StatusOK || co.OrderID == "" {
		t.Fatalf("checkout: %d %+v", rec.Code, co)
	}
	tn := tenantByEmail(t, srv, qa)
	trialSampai := tn.langganan.TrialSampai

	if code := notify(srv, co.OrderID, "palsu"); code != http.StatusForbidden {
		t.Fatalf("signature palsu: want 403, got %d", code)
	}
	status[co.OrderID] = "pending"
	if notify(srv, co.OrderID, ""); tn.langgananStatus(time.Now()).Status != "trial" {
		t.Fatal("pending tidak boleh mengaktifkan")
	}
	status[co.OrderID] = "settlement"
	notify(srv, co.OrderID, "")
	notify(srv, co.OrderID, "") // notifikasi ulang: tidak boleh memperpanjang dua kali
	want := trialSampai.Add(30 * 24 * time.Hour)
	if got := tn.langganan.AktifSampai; got.Sub(want).Abs() > time.Second {
		t.Fatalf("aktif sampai %v, want %v (trial + 30 hari, sekali)", got, want)
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
	if rec := do(srv, qa, http.MethodPost, "/api/v1/billing/checkout", `{"paket":"bulanan"}`); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("tanpa key: want 503, got %d", rec.Code)
	}
	if body := do(srv, qa, http.MethodGet, "/api/v1/billing", "").Body.String(); !strings.Contains(body, `"orders":[]`) {
		t.Fatalf("usaha tanpa pembayaran harus orders [] (bukan null): %s", body)
	}
	if rec := do(srv, nil, http.MethodGet, "/api/v1/billing/plans", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "bulanan") {
		t.Fatalf("plans publik: %d", rec.Code)
	}
}
