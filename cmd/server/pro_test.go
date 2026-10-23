package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Fitur Pro (rekonsiliasi pajak) terbuka untuk masa coba, paket Pro/Premium, dan paket lama
// sebelum ada tingkat; tertutup untuk Starter dan langganan habis.
func TestLanggananPro(t *testing.T) {
	now := time.Now()
	nanti, tadi := now.Add(48*time.Hour), now.Add(-48*time.Hour)
	bayar := func(paket, status string) []order {
		return []order{{ID: "o", Paket: paket, Status: status, Waktu: tadi}}
	}
	kasus := []struct {
		nama    string
		l       langganan
		tingkat string
		pro     bool
	}{
		{"trial", langganan{TrialSampai: nanti}, "", true},
		{"starter", langganan{AktifSampai: nanti, Orders: bayar("starter-bulanan", "lunas")}, "starter", false},
		{"pro", langganan{AktifSampai: nanti, Orders: bayar("pro-tahunan", "lunas")}, "pro", true},
		{"premium manual", langganan{AktifSampai: nanti, Orders: bayar("premium-bulanan", "manual")}, "premium", true},
		{"paket lama", langganan{AktifSampai: nanti, Orders: bayar("bulanan", "lunas")}, "", true},
		{"pro lalu starter: terakhir menentukan", langganan{AktifSampai: nanti, Orders: append(bayar("pro-bulanan", "lunas"),
			order{ID: "p", Paket: "starter-bulanan", Status: "lunas", Waktu: now}, order{ID: "q", Paket: "pro-bulanan", Status: "pending", Waktu: now})}, "starter", false},
		{"habis", langganan{AktifSampai: tadi, Orders: bayar("pro-bulanan", "lunas")}, "", false},
	}
	for _, k := range kasus {
		tn := newTenant(7)
		tn.langganan = k.l
		if st := tn.langgananStatus(now); st.Tingkat != k.tingkat || st.Pro != k.pro {
			t.Errorf("%s: %+v, mau tingkat %q pro %v", k.nama, st, k.tingkat, k.pro)
		}
	}
	if st := newTenant(1).langgananStatus(now); !st.Pro {
		t.Errorf("usaha operator selalu Pro: %+v", st)
	}
}

func TestPro_TolakStarter(t *testing.T) {
	srv, qa := usahaBaru(t)
	tn := tenantByEmail(t, srv, qa)
	tn.mu.Lock()
	tn.langganan = langganan{AktifSampai: time.Now().Add(time.Hour), Orders: []order{{ID: "o", Paket: "starter-bulanan", Status: "lunas"}}}
	tn.mu.Unlock()
	h := srv.guard("", srv.pro(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	cek := func() int {
		rec, req := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil)
		req.AddCookie(qa)
		h(rec, req)
		return rec.Code
	}
	if c := cek(); c != http.StatusPaymentRequired {
		t.Fatalf("starter: %d, mau 402", c)
	}
	tn.mu.Lock()
	tn.langganan.Orders[0].Paket = "pro-bulanan"
	tn.mu.Unlock()
	if c := cek(); c != http.StatusOK {
		t.Fatalf("pro: %d, mau 200", c)
	}
}
