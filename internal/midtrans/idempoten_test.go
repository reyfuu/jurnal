package midtrans

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// CreateSnap mengirim Idempotency-Key = order_id, supaya retry order yang sama (mis. timeout)
// mendapat respons yang sama dari Midtrans, bukan error order_id dobel atau transaksi kedua.
// Kunci > 46 karakter diabaikan Midtrans, jadi tidak dikirim.
func TestCreateSnap_IdempotencyKey(t *testing.T) {
	var kunci []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		kunci = append(kunci, r.Header.Get("Idempotency-Key"))
		_, _ = w.Write([]byte(`{"token":"tok","redirect_url":"https://pay"}`))
	}))
	defer srv.Close()
	c := &Client{ServerKey: "SK", SnapURL: srv.URL}
	for _, id := range []string{"LB-7-1792000000000000000", "LB-7-" + strings.Repeat("9", 60)} {
		if _, err := c.CreateSnap(context.Background(), SnapRequest{OrderID: id, Amount: 1}); err != nil {
			t.Fatal(err)
		}
	}
	if kunci[0] != "LB-7-1792000000000000000" || kunci[1] != "" {
		t.Fatalf("Idempotency-Key = %q", kunci)
	}
}

// Order id masuk path Status API dalam bentuk ter-escape.
func TestStatus_OrderIDDiEscape(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.EscapedPath()
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	if _, err := (&Client{ServerKey: "SK", APIURL: srv.URL}).Status(context.Background(), "LB-1/../x"); err != nil {
		t.Fatal(err)
	}
	if path != "/v2/LB-1%2F..%2Fx/status" {
		t.Fatalf("path %q", path)
	}
}
