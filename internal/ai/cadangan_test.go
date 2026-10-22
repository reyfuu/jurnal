package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"pajak/internal/llm"
)

// Model utama mati (argon di 9Router), cadangan menjawab: analisis tetap tampil dan
// "dibaca oleh" menyebut model yang benar-benar menjawab, bukan daftar konfigurasi.
func TestAnalisa_ModelCadanganTercatat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Model string }
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Model != "cadangan" {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"message":"[utama] [404]: Requested entity was not found."}}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{
			"message": map[string]string{"role": "assistant", "content": `{"ringkasan":"Usaha berjalan sehat."}`}}}})
	}))
	t.Cleanup(srv.Close)
	c := llm.New(llm.Config{APIKey: "uji", BaseURL: srv.URL, Model: "utama,cadangan", Timeout: 5e9})

	a := AnalisaProfil(context.Background(), c, FaktaDariProfil(profilContoh()))
	if !a.Tersedia || a.Model != "cadangan" {
		t.Errorf("Tersedia=%v Model=%q Alasan=%q; want analisis dari cadangan", a.Tersedia, a.Model, a.Alasan)
	}
}
