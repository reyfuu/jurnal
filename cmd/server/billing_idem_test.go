package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// snapUji: Snap & Status API palsu yang menghitung panggilan Snap, merekam Idempotency-Key,
// bisa ditahan (tahan != nil) untuk uji serentak, dan bisa digagalkan sekali (gagal).
type snapUji struct {
	mu      sync.Mutex
	panggil int
	kunci   []string
	gagal   bool
	masuk   chan struct{} // diberi tahu saat Snap dipanggil (bila tahan aktif)
	tahan   chan struct{} // Snap menunggu sampai channel ini ditutup
	status  map[string]string
}

func fakeSnap(t *testing.T) *snapUji {
	t.Helper()
	f := &snapUji{status: map[string]string{}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/snap/v1/transactions" {
			f.mu.Lock()
			f.panggil++
			f.kunci = append(f.kunci, r.Header.Get("Idempotency-Key"))
			gagal, tahan, masuk := f.gagal, f.tahan, f.masuk
			f.gagal = false
			f.mu.Unlock()
			if tahan != nil {
				masuk <- struct{}{}
				<-tahan
			}
			if gagal {
				http.Error(w, "gangguan", http.StatusInternalServerError)
				return
			}
			_, _ = w.Write([]byte(`{"token":"tok-` + time.Now().Format("150405.000000000") + `","redirect_url":"https://bayar"}`))
			return
		}
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v2/"), "/status")
		f.mu.Lock()
		st, ok := f.status[id]
		f.mu.Unlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"order_id": id, "transaction_status": st, "gross_amount": "199000.00", "status_code": "200"})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("MIDTRANS_SERVER_KEY", "SK-uji")
	t.Setenv("MIDTRANS_SNAP_URL", srv.URL)
	t.Setenv("MIDTRANS_API_URL", srv.URL)
	return f
}

func doKunci(srv *Server, c *http.Cookie, method, path, body, kunci string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if c != nil {
		req.AddCookie(c)
	}
	if kunci != "" {
		req.Header.Set("Idempotency-Key", kunci)
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

type checkoutResp struct {
	OrderID string `json:"order_id"`
	Token   string `json:"token"`
}

func checkout(t *testing.T, srv *Server, c *http.Cookie, kunci string) (int, checkoutResp) {
	t.Helper()
	rec := doKunci(srv, c, http.MethodPost, "/api/v1/billing/checkout", `{"paket":"starter-bulanan"}`, kunci)
	var co checkoutResp
	_ = json.Unmarshal(rec.Body.Bytes(), &co)
	return rec.Code, co
}

// Klik ganda / retry dengan kunci yang sama: satu order, satu transaksi Snap, token sama.
// Kunci lain = niat bayar baru. Midtrans menerima Idempotency-Key = order_id.
func TestCheckout_KunciSamaSatuOrder(t *testing.T) {
	f := fakeSnap(t)
	srv := NewServer()
	qa := register(t, srv, registerBody)
	k1, a := checkout(t, srv, qa, "niat-1")
	k2, b := checkout(t, srv, qa, "niat-1")
	if k1 != http.StatusOK || k2 != http.StatusOK || a.OrderID == "" || a != b {
		t.Fatalf("kunci sama: %d %+v / %d %+v", k1, a, k2, b)
	}
	if _, c := checkout(t, srv, qa, "niat-2"); c.OrderID == a.OrderID {
		t.Fatal("kunci lain harus order baru")
	}
	if n := len(tenantByEmail(t, srv, qa).langganan.Orders); n != 2 || f.panggil != 2 {
		t.Fatalf("order %d, panggilan Snap %d; mau 2 & 2", n, f.panggil)
	}
	if f.kunci[0] != a.OrderID {
		t.Fatalf("Idempotency-Key ke Midtrans = %q, mau order_id %q", f.kunci[0], a.OrderID)
	}
	if code, _ := checkout(t, srv, qa, "kunci tidak sah!"); code != http.StatusBadRequest {
		t.Fatalf("kunci tidak sah: %d", code)
	}
}

// Dua permintaan serentak dengan kunci sama: yang kedua tidak membuat transaksi Snap kedua.
func TestCheckout_KunciSamaSerentak(t *testing.T) {
	f := fakeSnap(t)
	tahan := make(chan struct{})
	f.masuk, f.tahan = make(chan struct{}, 1), tahan
	srv := NewServer()
	qa := register(t, srv, registerBody)
	var wg sync.WaitGroup
	var pertama checkoutResp
	var kodePertama int
	wg.Add(1)
	go func() {
		defer wg.Done()
		kodePertama, pertama = checkout(t, srv, qa, "niat-serentak")
	}()
	<-f.masuk // permintaan pertama sedang menunggu Snap
	if code, _ := checkout(t, srv, qa, "niat-serentak"); code != http.StatusConflict {
		t.Fatalf("permintaan kedua saat yang pertama diproses: %d, mau 409", code)
	}
	f.mu.Lock()
	f.tahan = nil // panggilan Snap berikutnya tidak ditahan
	f.mu.Unlock()
	close(tahan)
	wg.Wait()
	ulang, kedua := checkout(t, srv, qa, "niat-serentak")
	if kodePertama != http.StatusOK || ulang != http.StatusOK || kedua != pertama || f.panggil != 1 {
		t.Fatalf("pertama %d %+v, ulang %d %+v, panggilan Snap %d", kodePertama, pertama, ulang, kedua, f.panggil)
	}
}

// Snap gagal: reservasi dilepas, retry dengan kunci sama berhasil dan tetap satu order.
func TestCheckout_SnapGagalBisaDiulang(t *testing.T) {
	f := fakeSnap(t)
	f.gagal = true
	srv := NewServer()
	qa := register(t, srv, registerBody)
	if code, _ := checkout(t, srv, qa, "niat-gagal"); code != http.StatusBadGateway {
		t.Fatalf("Snap gagal: %d", code)
	}
	if n := len(tenantByEmail(t, srv, qa).langganan.Orders); n != 0 {
		t.Fatalf("order tertinggal setelah Snap gagal: %d", n)
	}
	if code, co := checkout(t, srv, qa, "niat-gagal"); code != http.StatusOK || co.Token == "" {
		t.Fatalf("retry: %d %+v", code, co)
	}
	if n := len(tenantByEmail(t, srv, qa).langganan.Orders); n != 1 {
		t.Fatalf("order setelah retry: %d", n)
	}
}

// Aktivasi manual Panel Master yang terkirim dua kali (klik ganda) menambah hari sekali saja.
func TestMasterLangganan_KunciSama(t *testing.T) {
	srv, m := serverMaster(t)
	qa := register(t, srv, registerBody)
	body := `{"aksi":"aktifkan","paket":"starter-bulanan","alasan":"Transfer BCA"}`
	for range 2 {
		if rec := doKunci(srv, m, http.MethodPost, "/api/v1/master/klien/2/langganan", body, "manual-1"); rec.Code != http.StatusOK {
			t.Fatalf("aktifkan: %d %s", rec.Code, rec.Body)
		}
	}
	tn := tenantByEmail(t, srv, qa)
	if n := len(tn.langganan.Orders); n != 1 || time.Until(tn.langganan.AktifSampai) > 31*24*time.Hour {
		t.Fatalf("order %d, aktif sampai %v; mau sekali 30 hari", n, tn.langganan.AktifSampai)
	}
}

// Cek status yang gagal menyimpan tidak boleh melapor sukses (Midtrans/klien mengulang).
func TestBillingRefresh_SimpanGagal503(t *testing.T) {
	f := fakeSnap(t)
	dir := t.TempDir()
	t.Setenv("DATA_DIR", dir)
	srv := NewServer()
	qa := register(t, srv, registerBody)
	_, co := checkout(t, srv, qa, "niat-refresh")
	f.mu.Lock()
	f.status[co.OrderID] = "settlement"
	f.mu.Unlock()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })
	if rec := do(srv, qa, http.MethodPost, "/api/v1/billing/refresh", ""); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("refresh dengan simpan gagal: %d, mau 503", rec.Code)
	}
}
