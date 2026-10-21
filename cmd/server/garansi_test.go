package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestStatusGaransi(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	lunas := func(bayar time.Time) order { return order{ID: "LB-2-1", Status: "lunas", DibayarPada: bayar} }
	for _, c := range []struct {
		nama   string
		l      langganan
		alasan string // "" = berlaku
	}{
		{"belum ada pembayaran", langganan{Orders: []order{{Status: "pending"}}}, garansiBelumBayar},
		{"bayar kemarin", langganan{Orders: []order{lunas(now.Add(-24 * time.Hour))}}, ""},
		{"tepat 7 hari", langganan{Orders: []order{lunas(now.Add(-garansiDurasi))}}, garansiLewat},
		{"sudah ekspor", langganan{Orders: []order{lunas(now.Add(-time.Hour))}, EksporPertama: now.Add(-time.Minute)}, garansiSudahEkspor},
		{"sudah diklaim", langganan{Orders: []order{lunas(now.Add(-time.Hour))}, GaransiDiklaim: now.Add(-time.Minute)}, garansiSudahDiklaim},
		{"hanya pembayaran pertama", langganan{Orders: []order{lunas(now.Add(-30 * 24 * time.Hour)), lunas(now.Add(-time.Hour))}}, garansiLewat},
		{"bayar sebelum fitur garansi", langganan{Orders: []order{lunas(time.Time{})}}, garansiLewat},
	} {
		g := statusGaransi(c.l, now)
		if g.Berlaku != (c.alasan == "") || g.Alasan != c.alasan {
			t.Errorf("%s: %+v, want alasan %q", c.nama, g, c.alasan)
		}
		if g.Berlaku && !g.Sampai.Equal(c.l.Orders[0].DibayarPada.Add(garansiDurasi)) {
			t.Errorf("%s: sampai %v", c.nama, g.Sampai)
		}
	}
}

// bayarLewatMidtrans: checkout + notifikasi settlement, kembalikan cookie usaha yang sudah lunas.
func bayarLewatMidtrans(t *testing.T, srv *Server, status map[string]string, body string) *http.Cookie {
	t.Helper()
	c := register(t, srv, body)
	var co struct {
		OrderID string `json:"order_id"`
	}
	_ = json.NewDecoder(do(srv, c, http.MethodPost, "/api/v1/billing/checkout", `{"paket":"starter-bulanan"}`).Body).Decode(&co)
	status[co.OrderID] = "settlement"
	if code := notify(srv, co.OrderID, ""); code != http.StatusOK {
		t.Fatalf("notify: %d", code)
	}
	return c
}

func garansiOf(t *testing.T, srv *Server, c *http.Cookie) meResp {
	t.Helper()
	var me meResp
	_ = json.NewDecoder(do(srv, c, http.MethodGet, "/api/v1/auth/me", "").Body).Decode(&me)
	return me
}

func TestGaransi7Hari(t *testing.T) {
	status := map[string]string{}
	fakeMidtrans(t, status)
	srv := NewServer()

	// Usaha A: tidak mengekspor → klaim berhasil, akses berhenti, klaim kedua ditolak.
	a := bayarLewatMidtrans(t, srv, status, registerBody)
	do(srv, a, http.MethodPut, "/api/v1/business", profileBody)
	if me := garansiOf(t, srv, a); !me.Garansi.Berlaku || me.Langganan.Status != "aktif" {
		t.Fatalf("setelah bayar garansi harus berlaku: %+v %+v", me.Garansi, me.Langganan)
	}
	if rec := do(srv, a, http.MethodGet, "/api/v1/onboarding/template", ""); rec.Code != http.StatusOK {
		t.Fatalf("template: %d", rec.Code)
	}
	if !garansiOf(t, srv, a).Garansi.Berlaku {
		t.Fatal("unduh template kosong bukan ekspor data; garansi tidak boleh gugur")
	}
	if rec := do(srv, a, http.MethodPost, "/api/v1/billing/garansi", `{"alasan":"belum cocok"}`); rec.Code != http.StatusOK {
		t.Fatalf("klaim garansi: %d %s", rec.Code, rec.Body)
	}
	me := garansiOf(t, srv, a)
	if me.Langganan.Status != "habis" || me.Garansi.Berlaku || me.Garansi.Alasan != garansiSudahDiklaim {
		t.Fatalf("setelah klaim akses berhenti & garansi tertutup: %+v %+v", me.Langganan, me.Garansi)
	}
	if rec := do(srv, a, http.MethodPost, "/api/v1/billing/garansi", `{}`); rec.Code != http.StatusConflict {
		t.Fatalf("klaim kedua: want 409, got %d", rec.Code)
	}

	// Usaha B: mengekspor laporan → garansi gugur, klaim 409.
	b := bayarLewatMidtrans(t, srv, status, strings.Replace(registerBody, "qa1@", "qa2@", 1))
	do(srv, b, http.MethodPut, "/api/v1/business", profileBody)
	if rec := do(srv, b, http.MethodGet, "/api/v1/invoice/99/pdf", ""); rec.Code != http.StatusNotFound || !garansiOf(t, srv, b).Garansi.Berlaku {
		t.Fatalf("ekspor yang gagal (404) tidak boleh menggugurkan garansi: %d", rec.Code)
	}
	var inv struct{ ID int }
	_ = json.NewDecoder(do(srv, b, http.MethodPost, "/api/v1/invoice", `{"pelanggan_nama":"Toko Mitra","tanggal":"2026-10-01","jatuh_tempo":"2026-10-31","items":[{"deskripsi":"Kopi","qty":1,"satuan":"kg","harga_satuan":250000}]}`).Body).Decode(&inv)
	if rec := do(srv, b, http.MethodGet, "/api/v1/invoice/"+strconv.Itoa(inv.ID)+"/pdf", ""); rec.Code != http.StatusOK {
		t.Fatalf("ekspor PDF invoice: %d %s", rec.Code, rec.Body)
	}
	if g := garansiOf(t, srv, b).Garansi; g.Berlaku || g.Alasan != garansiSudahEkspor {
		t.Fatalf("setelah ekspor garansi gugur: %+v", g)
	}
	if rec := do(srv, b, http.MethodPost, "/api/v1/billing/garansi", `{}`); rec.Code != http.StatusConflict {
		t.Fatalf("klaim setelah ekspor: want 409, got %d", rec.Code)
	}
	if garansiOf(t, srv, b).Langganan.Status != "aktif" {
		t.Fatal("klaim yang ditolak tidak boleh menghentikan akses")
	}
}
