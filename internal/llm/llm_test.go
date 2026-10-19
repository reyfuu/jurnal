package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pajak/internal/llm"
)

func TestConfig_TanpaKunciTidakAktif(t *testing.T) {
	t.Setenv("LLM_API_KEY", "")
	c := llm.New(llm.ConfigFromEnv())
	if c.Enabled() {
		t.Error("tanpa LLM_API_KEY, klien harus tidak aktif")
	}
	if _, err := c.Complete(context.Background(), nil, false, 100); err != llm.ErrDisabled {
		t.Errorf("harus menolak tanpa menghubungi jaringan, got %v", err)
	}
}

func TestConfig_DefaultDanOverride(t *testing.T) {
	t.Setenv("LLM_API_KEY", "k")
	t.Setenv("LLM_BASE_URL", "")
	t.Setenv("LLM_MODEL", "")
	if c := llm.New(llm.ConfigFromEnv()); c.BaseURL() != "https://api.openai.com/v1" {
		t.Errorf("base URL default = %q", c.BaseURL())
	}
	// Pindah penyedia cukup lewat env; trailing slash tidak boleh bikin URL dobel garis miring.
	t.Setenv("LLM_BASE_URL", "https://api.groq.com/openai/v1/")
	t.Setenv("LLM_MODEL", "llama-3.3-70b-versatile")
	c := llm.New(llm.ConfigFromEnv())
	if c.BaseURL() != "https://api.groq.com/openai/v1" {
		t.Errorf("trailing slash harus dibuang, got %q", c.BaseURL())
	}
	if c.Model() != "llama-3.3-70b-versatile" {
		t.Errorf("model = %q", c.Model())
	}
}

// Klien harus bicara dalam bentuk chat completions standar, supaya penyedia mana pun yang
// kompatibel OpenAI bisa dipakai tanpa ubah kode.
func TestComplete_BentukPermintaanStandar(t *testing.T) {
	var got struct {
		Model          string  `json:"model"`
		Temperature    float64 `json:"temperature"`
		ResponseFormat *struct {
			Type string `json:"type"`
		} `json:"response_format"`
		Messages []llm.Message `json:"messages"`
	}
	var auth, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, path = r.Header.Get("Authorization"), r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"ringkasan\":\"oke\"}"}}]}`))
	}))
	defer srv.Close()

	t.Setenv("LLM_API_KEY", "rahasia")
	t.Setenv("LLM_BASE_URL", srv.URL)
	t.Setenv("LLM_MODEL", "model-uji")
	c := llm.New(llm.ConfigFromEnv())

	out, err := c.Complete(context.Background(), []llm.Message{{Role: "user", Content: "halo"}}, true, 500)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if out != `{"ringkasan":"oke"}` {
		t.Errorf("isi balasan = %q", out)
	}
	if path != "/chat/completions" {
		t.Errorf("path = %q, want /chat/completions", path)
	}
	if auth != "Bearer rahasia" {
		t.Errorf("Authorization = %q", auth)
	}
	if got.Model != "model-uji" {
		t.Errorf("model dikirim = %q", got.Model)
	}
	if got.ResponseFormat == nil || got.ResponseFormat.Type != "json_object" {
		t.Error("jsonMode harus meminta response_format json_object")
	}
	// Suhu rendah: analisis laporan keuangan harus stabil, bukan kreatif.
	if got.Temperature > 0.3 {
		t.Errorf("temperature %v terlalu tinggi untuk analisis angka", got.Temperature)
	}
}

func TestComplete_PesanGalatPenyediaDiteruskan(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Incorrect API key provided","type":"invalid_request_error"}}`))
	}))
	defer srv.Close()

	t.Setenv("LLM_API_KEY", "salah")
	t.Setenv("LLM_BASE_URL", srv.URL)
	c := llm.New(llm.ConfigFromEnv())
	_, err := c.Complete(context.Background(), []llm.Message{{Role: "user", Content: "x"}}, false, 10)
	if err == nil || !strings.Contains(err.Error(), "Incorrect API key") {
		t.Errorf("pesan galat penyedia harus diteruskan apa adanya, got %v", err)
	}
}

// Sebagian penyedia tetap membungkus JSON dengan pagar kode meski diminta json_object.
func TestExtractJSON(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{`{"a":1}`, `{"a":1}`},
		{"```json\n{\"a\":1}\n```", `{"a":1}`},
		{"Tentu, ini hasilnya:\n{\"a\":1}\nSemoga membantu.", `{"a":1}`},
		{"```\n{\"a\":{\"b\":2}}\n```", `{"a":{"b":2}}`},
	} {
		if got := llm.ExtractJSON(c.in); got != c.want {
			t.Errorf("ExtractJSON(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
