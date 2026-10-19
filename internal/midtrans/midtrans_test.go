package midtrans

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignature(t *testing.T) {
	// want dihitung terpisah: python3 hashlib.sha512(order_id+status_code+gross_amount+server_key)
	got := Signature("LB-2-1792000000", "200", "99000.00", "SB-Mid-server-uji")
	want := "7b7f66d95c102628f7eed8919cdacc199c09c06fd7d54cbefa56ac9e6db084d8c995193bcfcc1bda22e8f439e63a007c1666921357c2621484f18ea9ee657a42"
	if got != want {
		t.Fatalf("signature\n got %s\nwant %s", got, want)
	}
	c := &Client{ServerKey: "SB-Mid-server-uji"}
	n := Notification{OrderID: "LB-2-1792000000", StatusCode: "200", GrossAmount: "99000.00", SignatureKey: want}
	if !c.VerifySignature(n) {
		t.Fatal("signature valid ditolak")
	}
	n.GrossAmount = "1.00"
	if c.VerifySignature(n) {
		t.Fatal("signature palsu diterima")
	}
}

func TestCreateSnapAndStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, _, _ := r.BasicAuth(); u != "SK" {
			http.Error(w, "unauthorized", 401)
			return
		}
		switch r.URL.Path {
		case "/snap/v1/transactions":
			var body map[string]map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["transaction_details"]["order_id"] != "LB-1" || body["transaction_details"]["gross_amount"] != float64(99000) {
				http.Error(w, "bad body", 400)
				return
			}
			_, _ = w.Write([]byte(`{"token":"tok","redirect_url":"https://pay"}`))
		case "/v2/LB-1/status":
			_, _ = w.Write([]byte(`{"order_id":"LB-1","transaction_status":"settlement","gross_amount":"99000.00","status_code":"200"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := &Client{ServerKey: "SK", SnapURL: srv.URL, APIURL: srv.URL}
	res, err := c.CreateSnap(context.Background(), SnapRequest{OrderID: "LB-1", Amount: 99000})
	if err != nil || res.Token != "tok" {
		t.Fatalf("snap: %+v %v", res, err)
	}
	st, err := c.Status(context.Background(), "LB-1")
	if err != nil || !st.Paid() {
		t.Fatalf("status: %+v %v", st, err)
	}
	if _, err := (&Client{ServerKey: "salah", SnapURL: srv.URL}).CreateSnap(context.Background(), SnapRequest{OrderID: "LB-1"}); err == nil {
		t.Fatal("key salah harus error")
	}
}
