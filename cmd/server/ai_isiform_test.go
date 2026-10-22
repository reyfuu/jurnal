package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"pajak/internal/llm"
)

// llmIsiForm: model palsu yang selalu membalas isian transaksi; hanya panggilan berprompt isi
// form yang dihitung (panduan menu juga boleh memakai model).
func llmIsiForm(t *testing.T, srv *Server) *atomic.Int32 {
	t.Helper()
	var panggilan atomic.Int32
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Messages []llm.Message }
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Messages) > 0 && strings.Contains(req.Messages[0].Content, "mengisi form aplikasi") {
			panggilan.Add(1)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"message": map[string]string{"role": "assistant",
			"content": `{"isian":{"tanggal":"2026-08-20","jenis":"Penjualan","nominal":2000000,"keterangan":"Penjualan kopi","lunas":true},"catatan":[]}`}}}})
	}))
	t.Cleanup(fake.Close)
	srv.llm = llm.New(llm.Config{APIKey: "uji", BaseURL: fake.URL, Model: "palsu", Timeout: 5e9})
	return &panggilan
}

func jumlahTx(t *testing.T, srv *Server, c *http.Cookie) int {
	t.Helper()
	var r txListResp
	if err := json.NewDecoder(do(srv, c, http.MethodGet, "/api/v1/transactions", "").Body).Decode(&r); err != nil {
		t.Fatal(err)
	}
	return len(r.Transactions)
}

// Perintah mengisi form dijawab usulan isian; tidak ada yang tersimpan sebelum pengguna klik Simpan.
func TestTanyaAI_IsiFormTransaksiTanpaMenyimpan(t *testing.T) {
	srv := NewServer()
	panggilan := llmIsiForm(t, srv)
	owner := login(t, srv, "owner@kedaikopisenja.id")
	sebelum := jumlahTx(t, srv, owner)

	r := tanyaAI(t, srv, owner, "catat penjualan kopi 2 juta tunai tanggal 20 agustus")
	if r.Category != "isi-form" || r.Aksi == nil || r.Aksi.Form != "transaksi" || r.Aksi.View != "transaksi" {
		t.Fatalf("jawaban = %+v", r)
	}
	if !strings.Contains(string(r.Aksi.Isian), `"nominal":2000000`) {
		t.Errorf("isian = %s", r.Aksi.Isian)
	}
	if panggilan.Load() != 1 {
		t.Errorf("model dipanggil %d kali", panggilan.Load())
	}
	if n := jumlahTx(t, srv, owner); n != sebelum {
		t.Errorf("jumlah transaksi %d -> %d: AI tidak boleh menyimpan", sebelum, n)
	}
}

// Pertanyaan cara tetap panduan menu, tidak memanggil model isi form.
func TestTanyaAI_PertanyaanCaraTetapPanduan(t *testing.T) {
	srv := NewServer()
	panggilan := llmIsiForm(t, srv)
	owner := login(t, srv, "owner@kedaikopisenja.id")

	r := tanyaAI(t, srv, owner, "bagaimana cara catat penjualan?")
	if r.Aksi != nil || r.Category == "isi-form" {
		t.Errorf("jawaban = %+v", r)
	}
	if panggilan.Load() != 0 {
		t.Errorf("model isi form tidak boleh dipanggil, %d kali", panggilan.Load())
	}
}

// Kasir tidak bisa membuka Invoice: dijelaskan, tanpa isian dan tanpa memanggil model.
func TestTanyaAI_IsiFormHormatiHakAkses(t *testing.T) {
	srv := NewServer()
	panggilan := llmIsiForm(t, srv)
	kasir := login(t, srv, "kasir@kedaikopisenja.id")

	r := tanyaAI(t, srv, kasir, "buatkan invoice untuk PT Maju 10 kaos @50rb")
	if r.Aksi != nil || !strings.Contains(r.Answer, "tidak tersedia untuk akun Anda") {
		t.Errorf("jawaban = %+v", r)
	}
	if panggilan.Load() != 0 {
		t.Errorf("model dipanggil %d kali", panggilan.Load())
	}
	// Form yang memang boleh: Kasir bisa mencatat transaksi.
	if r := tanyaAI(t, srv, kasir, "catat penjualan kopi 2 juta"); r.Aksi == nil {
		t.Errorf("kasir boleh catat transaksi, jawaban = %+v", r)
	}
}

// Tanpa AI aktif: arahkan ke form supaya diisi sendiri.
func TestTanyaAI_IsiFormTanpaLLM(t *testing.T) {
	t.Setenv("LLM_API_KEY", "")
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")

	r := tanyaAI(t, srv, owner, "tambah pemasok CV Sumber Rejeki")
	if r.Aksi != nil || len(r.Tautan) != 1 || r.Tautan[0].View != "produksi-supplier" {
		t.Errorf("jawaban = %+v", r)
	}
}
