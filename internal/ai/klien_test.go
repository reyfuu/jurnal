package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pajak/internal/llm"
	"pajak/internal/pengetahuan"
)

func faktaContoh() FaktaKlien {
	return FaktaKlien{Nama: "Kedai Kopi Senja", JenisUsaha: "perdagangan", BentukUsaha: "op", KLU: "563", KLUNama: "Penyediaan Minuman",
		OmzetSetahun: 312_400_000, NPMBps: 3320, BenchmarkNPMBps: -502, BenchmarkCTTORBps: 39, SumberBenchmark: "DJP 2021", StatusNPM: "wajar"}
}

var sumberContoh = []pengetahuan.Potongan{
	{Ref: "A-001", Judul: "PPh Final UMKM", Teks: "Tarif 0,5% dari omzet sampai Rp4,8 miliar."},
	{Ref: "KB-001", Judul: "Batas omzet", Teks: "OP tidak kena PPh final atas omzet sampai Rp500 juta."},
}

// llmPalsu membalas isi sebagai pesan asisten dan menyimpan pesan user terakhir.
func llmPalsu(t *testing.T, isi string, diterima *string) *llm.Client {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []llm.Message `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		*diterima = req.Messages[len(req.Messages)-1].Content
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": isi}}}})
	}))
	t.Cleanup(srv.Close)
	return llm.New(llm.Config{APIKey: "uji", BaseURL: srv.URL, Model: "palsu", Timeout: 5e9})
}

func TestAnalisaKlien_TanpaKunci(t *testing.T) {
	a := AnalisaKlien(context.Background(), llm.New(llm.Config{}), faktaContoh(), sumberContoh)
	if a.Tersedia || !strings.Contains(a.Alasan, "LLM_API_KEY") {
		t.Fatalf("tanpa kunci = %+v", a)
	}
}

// Model hanya boleh merujuk sumber yang dikirim; rujukan karangan dibuang.
func TestAnalisaKlien_RujukanDisaring(t *testing.T) {
	var diterima string
	c := llmPalsu(t, `{"penilaian":"Margin jauh di atas rata-rata usaha sejenis [BM-563].","poin":[
		{"judul":"Omzet masih di bawah Rp500 juta","isi":"Sebagian besar omzet belum kena PPh final.","sumber":["KB-001","A-999"]},
		{"judul":"Tetap catat semua transaksi","isi":"x","sumber":[]}]}`, &diterima)
	a := AnalisaKlien(context.Background(), c, faktaContoh(), sumberContoh)
	if !a.Tersedia || a.Model != "palsu" || len(a.Poin) != 2 {
		t.Fatalf("analisa = %+v", a)
	}
	if got := a.Poin[0].Sumber; len(got) != 1 || got[0] != "KB-001" {
		t.Errorf("rujukan setelah disaring = %v", got)
	}
	// Fakta & sumber benar-benar dikirim ke model, termasuk benchmark sebagai rujukan BM-<KLU>.
	for _, want := range []string{`"npm_bps":3320`, "KB-001", "BM-563", "Batas omzet"} {
		if !strings.Contains(diterima, want) {
			t.Errorf("pesan ke model tidak memuat %q", want)
		}
	}
}

func TestAnalisaKlien_BalasanRusak(t *testing.T) {
	var diterima string
	a := AnalisaKlien(context.Background(), llmPalsu(t, "maaf saya tidak bisa", &diterima), faktaContoh(), sumberContoh)
	if a.Tersedia || a.Alasan == "" {
		t.Errorf("balasan bukan JSON = %+v", a)
	}
}

func TestQueryKlien(t *testing.T) {
	q := QueryKlien(faktaContoh(), []string{"NPM di bawah rata-rata nasional"})
	for _, want := range []string{"Penyediaan Minuman", "pph final", "NPM di bawah"} {
		if !strings.Contains(strings.ToLower(q), strings.ToLower(want)) {
			t.Errorf("query %q tidak memuat %q", q, want)
		}
	}
}
