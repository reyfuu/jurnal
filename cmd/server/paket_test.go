package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// Paket dijual: 3 tingkat × bulanan/tahunan (sesuai landing page). ID lama tetap dikenali
// untuk order yang sudah ada, tetapi tidak bisa dibeli lagi.
func TestPaketTigaTingkat(t *testing.T) {
	srv := NewServer()
	var r struct{ Paket []paket }
	_ = json.NewDecoder(do(srv, nil, http.MethodGet, "/api/v1/billing/plans", "").Body).Decode(&r)
	mau := map[string]struct {
		tingkat     string
		harga, hari int64
	}{
		"starter-bulanan": {"starter", 199_000, 30}, "starter-tahunan": {"starter", 1_990_000, 365},
		"pro-bulanan": {"pro", 499_000, 30}, "pro-tahunan": {"pro", 5_000_000, 365},
		"premium-bulanan": {"premium", 1_490_000, 30}, "premium-tahunan": {"premium", 14_900_000, 365},
	}
	if len(r.Paket) != len(mau) {
		t.Fatalf("paket dijual: %d, mau %d", len(r.Paket), len(mau))
	}
	for _, p := range r.Paket {
		m, ok := mau[p.ID]
		if !ok || p.Tingkat != m.tingkat || p.Harga != m.harga || int64(p.Hari) != m.hari || p.HargaNormal <= p.Harga {
			t.Errorf("paket %+v tidak sesuai", p)
		}
	}

	qa := register(t, srv, registerBody)
	if rec := do(srv, qa, http.MethodPost, "/api/v1/billing/checkout", `{"paket":"bulanan"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("paket lama tidak dijual lagi: want 400, got %d", rec.Code)
	}

	// order lama (paket "tahunan" Rp990.000) yang lunas setelah harga berubah tetap mengaktifkan.
	status := map[string]string{}
	fakeMidtrans(t, status)
	tn := tenantByEmail(t, srv, qa)
	id := fmt.Sprintf("LB-%d-1", tn.id)
	tn.langganan.Orders = append(tn.langganan.Orders, order{ID: id, Paket: "tahunan", Jumlah: 990_000, Status: "pending", Waktu: time.Now()})
	status[id] = "settlement:990000.00"
	notify(srv, id, "")
	if got := tn.langganan.AktifSampai; time.Until(got) < 364*24*time.Hour {
		t.Fatalf("order lama tahunan harus mengaktifkan 365 hari, aktif sampai %v", got)
	}
	if p, _, ok := paketTerakhir(tn.langganan.Orders); !ok || p.Nama != "Tahunan" {
		t.Errorf("nama paket order lama di Panel Master: %+v %v", p, ok)
	}
}
