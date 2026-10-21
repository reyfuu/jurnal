package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	"pajak/internal/ai"
)

func tanyaAI(t *testing.T, srv *Server, c *http.Cookie, q string) ai.QueryResponse {
	t.Helper()
	b, _ := json.Marshal(map[string]string{"query": q})
	rec := do(srv, c, http.MethodPost, "/api/v1/ai/chat", string(b))
	var r ai.QueryResponse
	if err := json.NewDecoder(rec.Body).Decode(&r); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("%q: %d %v", q, rec.Code, err)
	}
	return r
}

func TestTanyaAI_PanduanMenuDenganTautan(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	kasir := login(t, srv, "kasir@kedaikopisenja.id")

	r := tanyaAI(t, srv, owner, "dmn saya bisa isi laporan keuangan")
	var views []string
	for _, l := range r.Tautan {
		views = append(views, l.View)
	}
	if r.Category != "panduan" || !slices.Contains(views, "transaksi") || !slices.Contains(views, "laporan-keuangan") || !slices.Contains(views, "saldo-awal") {
		t.Fatalf("owner: kategori %q tautan %v", r.Category, views)
	}

	// Kasir tidak punya Laporan Keuangan & Impor Data Awal: tidak boleh diberi tautannya.
	r = tanyaAI(t, srv, kasir, "dmn saya bisa isi laporan keuangan")
	for _, l := range r.Tautan {
		if l.View != "transaksi" {
			t.Errorf("kasir diberi tautan %s", l.View)
		}
	}

	// Pertanyaan analisis tetap dijawab mesin biasa, plus satu tautan pelengkap.
	r = tanyaAI(t, srv, owner, "Bagaimana kondisi arus kas dan piutang saya?")
	if r.Category != "cashflow" || len(r.Tautan) != 1 {
		t.Errorf("analisis: %q %+v", r.Category, r.Tautan)
	}
	// Salah ketik tetap dikenali; pertanyaan menu yang tidak dikenali tidak dijawab ringkasan laporan.
	if r = tanyaAI(t, srv, owner, "kalau mau input trasaksi dmn ya ?"); r.Category != "panduan" || len(r.Tautan) == 0 || r.Tautan[0].View != "transaksi" {
		t.Errorf("salah ketik: %q %+v", r.Category, r.Tautan)
	}
	if r = tanyaAI(t, srv, owner, "dimana menu qwerty"); r.Category != "panduan" || len(r.Tautan) == 0 {
		t.Errorf("menu tak dikenal: %q %+v", r.Category, r.Tautan)
	}
	if r = tanyaAI(t, srv, owner, "Ringkas laporan bulan ini"); r.Category != "summary" || len(r.Tautan) != 0 {
		t.Errorf("ringkasan: %q %+v", r.Category, r.Tautan)
	}
}
