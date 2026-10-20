package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"pajak/internal/finance"
	"pajak/internal/llm"
)

func unggahMutasi(t *testing.T, srv *Server, c *http.Cookie, csv string) finance.MutasiHasil {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "mutasi.csv")
	_, _ = fw.Write([]byte(csv))
	_ = mw.WriteField("tahun", "2026")
	_ = mw.Close()
	rec := upload(srv, c, "/api/v1/transactions/mutasi", &body, mw.FormDataContentType())
	if rec.Code != http.StatusOK {
		t.Fatalf("pratinjau mutasi: %d %s", rec.Code, rec.Body)
	}
	var h finance.MutasiHasil
	_ = json.NewDecoder(rec.Body).Decode(&h)
	return h
}

const csvMutasi = "Tanggal;Keterangan;Debet;Kredit\n" +
	"01/09/2026;BIAYA ADM;10.000;\n" +
	"02/09/2026;TRSF CR 1234567890 BUDI;;5.000.000\n" +
	"03/09/2026;TRSF CR 9988776655 BUDI;;2.000.000\n"

func TestMutasi_TanpaAIBerpesan(t *testing.T) {
	srv := NewServer()
	h := unggahMutasi(t, srv, login(t, srv, "owner@kedaikopisenja.id"), csvMutasi)
	if h.DibantuAI != 0 || !strings.Contains(h.AIPesan, "belum aktif") || h.PerluDiperiksa != 2 {
		t.Fatalf("tanpa AI: %+v", h)
	}
}

func TestMutasi_BarisRendahDibantuAISekaliPerKelompok(t *testing.T) {
	var panggilan atomic.Int32
	var pesan string
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panggilan.Add(1)
		var req struct{ Messages []llm.Message }
		_ = json.NewDecoder(r.Body).Decode(&req)
		pesan = req.Messages[len(req.Messages)-1].Content
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"message": map[string]string{"role": "assistant",
			"content": `{"hasil":[{"no":1,"jenis":"Setor Modal","alasan":"transfer dari pemilik"}]}`}}}})
	}))
	defer fake.Close()
	srv := NewServer()
	srv.llm = llm.New(llm.Config{APIKey: "uji", BaseURL: fake.URL, Model: "palsu", Timeout: 5e9})
	owner := login(t, srv, "owner@kedaikopisenja.id")

	h := unggahMutasi(t, srv, owner, csvMutasi)
	if panggilan.Load() != 1 || h.DibantuAI != 2 || h.PerluDiperiksa != 0 {
		t.Fatalf("panggilan %d, hasil %+v", panggilan.Load(), h)
	}
	for _, r := range h.Rows {
		if strings.Contains(r.Keterangan, "BUDI") && (r.Jenis != "Setor Modal" || r.Keyakinan != finance.KeyakinanAI) {
			t.Errorf("baris %+v", r)
		}
	}
	if strings.Contains(pesan, "1234567890") || strings.Contains(pesan, "BIAYA ADM") || strings.Count(pesan, "BUDI") != 1 {
		t.Errorf("dikirim ke model:\n%s", pesan)
	}

	// kuota habis → tetap jalan tanpa AI
	for range maxMutasiAI {
		srv.jatahAI("mutasi|1", maxMutasiAI)
	}
	if h := unggahMutasi(t, srv, owner, csvMutasi); h.DibantuAI != 0 || !strings.Contains(h.AIPesan, "Kuota") {
		t.Fatalf("kuota habis: %+v", h)
	}
}
