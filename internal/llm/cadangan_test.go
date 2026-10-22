package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"pajak/internal/llm"
)

// penyediaUji menjawab per model sesuai peta balasan dan mencatat urutan model yang dicoba.
// Model tanpa entri di peta menjawab 503 seperti 9Router saat penyedia hulunya mati.
func penyediaUji(t *testing.T, models string, balasan map[string]string) (*llm.Client, *[]string) {
	t.Helper()
	var dicoba []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Model string }
		_ = json.NewDecoder(r.Body).Decode(&req)
		dicoba = append(dicoba, req.Model)
		b, ok := balasan[req.Model]
		if !ok {
			w.WriteHeader(http.StatusServiceUnavailable)
			b = `{"error":{"message":"[` + req.Model + `] [404]: Requested entity was not found."}}`
		}
		_, _ = w.Write([]byte(b))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("LLM_API_KEY", "k")
	t.Setenv("LLM_BASE_URL", srv.URL)
	t.Setenv("LLM_MODEL", models)
	return llm.New(llm.ConfigFromEnv()), &dicoba
}

const jawabanOke = `{"choices":[{"message":{"role":"assistant","content":"{\"ringkasan\":\"oke\"}"},"finish_reason":"stop"}]}`

// LLM_MODEL berisi daftar: model utama yang mati di penyedia (mis. argon di 9Router) tidak
// boleh mematikan Laporan AI selama ada cadangan yang menjawab.
func TestCompleteModel_CadanganSaatModelUtamaGagal(t *testing.T) {
	c, dicoba := penyediaUji(t, "utama, cadangan", map[string]string{"cadangan": jawabanOke})

	isi, model, err := c.CompleteModel(context.Background(), []llm.Message{{Role: "user", Content: "x"}}, true, 120)
	if err != nil {
		t.Fatalf("CompleteModel: %v", err)
	}
	if isi != `{"ringkasan":"oke"}` || model != "cadangan" {
		t.Errorf("isi, model = %q, %q; want jawaban cadangan", isi, model)
	}
	if !slices.Equal(*dicoba, []string{"utama", "cadangan"}) {
		t.Errorf("urutan dicoba = %v", *dicoba)
	}
}

func TestCompleteModel_UtamaMenjawabCadanganTidakDipanggil(t *testing.T) {
	c, dicoba := penyediaUji(t, "utama,cadangan", map[string]string{"utama": jawabanOke, "cadangan": jawabanOke})

	if _, model, err := c.CompleteModel(context.Background(), nil, false, 120); err != nil || model != "utama" {
		t.Fatalf("model, err = %q, %v; want utama", model, err)
	}
	if !slices.Equal(*dicoba, []string{"utama"}) {
		t.Errorf("cadangan tidak boleh dipanggil bila utama menjawab, dicoba = %v", *dicoba)
	}
}

// Model penalaran bisa menghabiskan max_tokens untuk berpikir: jawaban kosong atau terpotong
// dianggap gagal supaya cadangan dipakai, bukan diteruskan sebagai JSON rusak.
func TestCompleteModel_IsiKosongAtauTerpotongPakaiCadangan(t *testing.T) {
	for nama, b := range map[string]string{
		"kosong":    `{"choices":[{"message":{"role":"assistant","content":"  "},"finish_reason":"stop"}]}`,
		"terpotong": `{"choices":[{"message":{"role":"assistant","content":"{\"ringkasan\":\"ok"},"finish_reason":"length"}]}`,
	} {
		t.Run(nama, func(t *testing.T) {
			c, _ := penyediaUji(t, "utama,cadangan", map[string]string{"utama": b, "cadangan": jawabanOke})
			if _, model, err := c.CompleteModel(context.Background(), nil, true, 120); err != nil || model != "cadangan" {
				t.Errorf("model, err = %q, %v; want cadangan", model, err)
			}
		})
	}
}

func TestCompleteModel_SemuaGagalGalatSemuaModel(t *testing.T) {
	c, _ := penyediaUji(t, "utama,cadangan", nil)

	_, _, err := c.CompleteModel(context.Background(), nil, false, 120)
	if err == nil || !strings.Contains(err.Error(), "[utama]") || !strings.Contains(err.Error(), "[cadangan]") {
		t.Errorf("galat harus memuat penyebab tiap model, got %v", err)
	}
	// Satu model (perilaku lama): Complete tetap meneruskan galat penyedia.
	c1, _ := penyediaUji(t, "utama", nil)
	if _, err := c1.Complete(context.Background(), nil, false, 120); err == nil || !strings.Contains(err.Error(), "Requested entity") {
		t.Errorf("Complete satu model, got %v", err)
	}
}

func TestModel_DaftarDariEnv(t *testing.T) {
	c, _ := penyediaUji(t, " ag/gemini-4-argon-high , cx/gpt-5.6-terra,", nil)
	if got := c.Model(); got != "ag/gemini-4-argon-high, cx/gpt-5.6-terra" {
		t.Errorf("Model() = %q", got)
	}
}
