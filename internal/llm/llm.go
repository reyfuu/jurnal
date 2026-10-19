// Package llm memanggil model bahasa lewat antarmuka chat completions yang kompatibel OpenAI.
//
// Antarmuka itu dipilih karena sudah jadi standar de facto: satu kode ini jalan untuk OpenAI,
// Groq, OpenRouter, DeepSeek, Together, Mistral, xAI, Anthropic (endpoint kompatibelnya), serta
// model lokal lewat Ollama atau LM Studio. Pindah penyedia cukup mengganti variabel lingkungan,
// tanpa ubah kode dan tanpa menambah dependensi.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config dibaca dari variabel lingkungan:
//
//	LLM_API_KEY   kunci penyedia (kosong = fitur AI mati, aplikasi tetap jalan)
//	LLM_BASE_URL  mis. https://api.openai.com/v1, https://api.groq.com/openai/v1,
//	              https://openrouter.ai/api/v1, http://localhost:11434/v1 (Ollama)
//	LLM_MODEL     nama model di penyedia tersebut
//	LLM_TIMEOUT   detik, default 45
type Config struct {
	APIKey  string
	BaseURL string
	Model   string
	Timeout time.Duration
}

const (
	defaultBaseURL = "https://api.openai.com/v1"
	defaultModel   = "gpt-4o-mini"
	defaultTimeout = 45 * time.Second
)

func ConfigFromEnv() Config {
	c := Config{
		APIKey:  strings.TrimSpace(os.Getenv("LLM_API_KEY")),
		BaseURL: strings.TrimRight(strings.TrimSpace(os.Getenv("LLM_BASE_URL")), "/"),
		Model:   strings.TrimSpace(os.Getenv("LLM_MODEL")),
		Timeout: defaultTimeout,
	}
	if c.BaseURL == "" {
		c.BaseURL = defaultBaseURL
	}
	if c.Model == "" {
		c.Model = defaultModel
	}
	if s := os.Getenv("LLM_TIMEOUT"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			c.Timeout = time.Duration(n) * time.Second
		}
	}
	return c
}

// Enabled: tanpa kunci, pemanggil memakai jalur cadangan dan tidak menghubungi jaringan.
// Model lokal (Ollama/LM Studio) tidak butuh kunci sungguhan; isi apa saja, mis. "local".
func (c Config) Enabled() bool { return c.APIKey != "" }

type Client struct {
	cfg  Config
	http *http.Client
}

func New(cfg Config) *Client {
	return &Client{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}
}

func (c *Client) Model() string   { return c.cfg.Model }
func (c *Client) BaseURL() string { return c.cfg.BaseURL }
func (c *Client) Enabled() bool   { return c.cfg.Enabled() }

type Message struct {
	Role    string `json:"role"` // system | user | assistant
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string    `json:"model"`
	Messages       []Message `json:"messages"`
	Temperature    float64   `json:"temperature"`
	MaxTokens      int       `json:"max_tokens,omitempty"`
	ResponseFormat *struct {
		Type string `json:"type"`
	} `json:"response_format,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

var ErrDisabled = errors.New("LLM_API_KEY belum diisi")

// Complete mengirim percakapan dan mengembalikan isi balasan pertama.
// jsonMode meminta penyedia membalas JSON; penyedia yang tidak mendukung akan mengabaikannya,
// karena itu pemanggil tetap harus tahan terhadap balasan yang dibungkus teks.
func (c *Client) Complete(ctx context.Context, msgs []Message, jsonMode bool, maxTokens int) (string, error) {
	if !c.cfg.Enabled() {
		return "", ErrDisabled
	}
	req := chatRequest{Model: c.cfg.Model, Messages: msgs, Temperature: 0.2, MaxTokens: maxTokens}
	if jsonMode {
		req.ResponseFormat = &struct {
			Type string `json:"type"`
		}{Type: "json_object"}
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	res, err := c.http.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("menghubungi %s: %w", c.cfg.BaseURL, err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return "", err
	}
	var out chatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("balasan %s tidak terbaca (status %d)", c.cfg.BaseURL, res.StatusCode)
	}
	if out.Error != nil {
		return "", fmt.Errorf("%s menolak: %s", c.cfg.Model, out.Error.Message)
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s menjawab status %d", c.cfg.BaseURL, res.StatusCode)
	}
	if len(out.Choices) == 0 {
		return "", errors.New("model tidak mengembalikan jawaban")
	}
	return out.Choices[0].Message.Content, nil
}

// ExtractJSON mengambil objek JSON dari balasan model. Sebagian penyedia membungkusnya dengan
// pagar kode atau kalimat pengantar meski sudah diminta JSON, jadi isinya dipotong dari kurung
// kurawal terluar daripada dipercaya mentah-mentah.
func ExtractJSON(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "```"); i >= 0 {
		rest := s[i+3:]
		if j := strings.IndexByte(rest, '\n'); j >= 0 {
			rest = rest[j+1:]
		}
		if k := strings.Index(rest, "```"); k >= 0 {
			s = strings.TrimSpace(rest[:k])
		}
	}
	a, b := strings.IndexByte(s, '{'), strings.LastIndexByte(s, '}')
	if a >= 0 && b > a {
		return s[a : b+1]
	}
	return s
}
