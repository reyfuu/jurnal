package ai

import (
	"context"
	"encoding/json"
	"strings"

	"pajak/internal/finance"
	"pajak/internal/llm"
)

// Analysis adalah hasil pembacaan model atas seluruh laporan satu periode.
//
// Model HANYA menulis narasi. Seluruh angka pada Facts dihitung Go dan dikirim apa adanya;
// prompt melarang model berhitung sendiri. Di aplikasi pembukuan & pajak, angka karangan jauh
// lebih berbahaya daripada tidak ada analisis sama sekali.
type Analysis struct {
	// Tersedia=false berarti fitur belum dikonfigurasi atau panggilan gagal. Halaman tetap
	// menampilkan rincian laporan; hanya bagian rekomendasi yang kosong.
	Tersedia bool   `json:"tersedia"`
	Alasan   string `json:"alasan,omitempty"`
	Model    string `json:"model,omitempty"`

	Ringkasan         string        `json:"ringkasan,omitempty"`
	SorotanPositif    []string      `json:"sorotan_positif,omitempty"`
	PerluDiperhatikan []string      `json:"perlu_diperhatikan,omitempty"`
	Rekomendasi       []Rekomendasi `json:"rekomendasi,omitempty"`
}

type Rekomendasi struct {
	Judul  string `json:"judul"`
	Dampak string `json:"dampak"` // tinggi | sedang | rendah
	Detail string `json:"detail"`
}

const systemPrompt = `Kamu analis keuangan untuk UMKM Indonesia. Kamu menerima laporan keuangan satu periode dalam JSON yang SUDAH dihitung oleh sistem.

ATURAN MUTLAK:
1. JANGAN menghitung, menjumlah, membagi, atau memperkirakan angka apa pun. Pakai HANYA angka yang ada di JSON, salin persis.
2. Jika sebuah angka tidak ada di JSON, jangan sebutkan. Lebih baik tidak membahas daripada mengarang.
3. Nilai uang dalam Rupiah bilangan bulat. Tulis lengkap tanpa singkatan juta / jt, mis. "Rp18.450.000" untuk 18450000.
4. margin_bps adalah basis poin: 2005 berarti 20,05%.
5. Bahasa Indonesia yang lugas untuk pemilik usaha kecil, bukan bahasa akuntan. Hindari jargon.
6. Jangan memberi nasihat hukum atau pajak yang pasti; untuk urusan SPT arahkan memverifikasi ke konsultan pajak.

Balas HANYA objek JSON dengan bentuk:
{
  "ringkasan": "2-3 kalimat kondisi usaha periode ini",
  "sorotan_positif": ["maksimal 4 butir, masing-masing satu kalimat"],
  "perlu_diperhatikan": ["maksimal 4 butir, sebutkan risikonya"],
  "rekomendasi": [{"judul":"aksi singkat","dampak":"tinggi|sedang|rendah","detail":"1-2 kalimat langkah konkret"}]
}
Maksimal 4 rekomendasi, diurutkan dari dampak terbesar.`

// Analyze meminta model membaca fakta laporan. Tanpa LLM_API_KEY, fungsi ini tidak menghubungi
// jaringan dan mengembalikan Tersedia=false — halaman pemanggil tetap menampilkan laporannya.
func Analyze(ctx context.Context, c *llm.Client, f finance.BusinessFacts) Analysis {
	if c == nil || !c.Enabled() {
		return Analysis{
			Tersedia: false,
			Alasan:   "Analisis AI belum aktif. Isi LLM_API_KEY (dan LLM_BASE_URL serta LLM_MODEL bila bukan OpenAI) untuk mengaktifkannya.",
		}
	}

	fakta, err := json.Marshal(f)
	if err != nil {
		return Analysis{Tersedia: false, Alasan: "Gagal menyiapkan data laporan: " + err.Error()}
	}

	raw, err := c.Complete(ctx, []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: "Laporan keuangan periode " + f.PeriodeLabel + ":\n\n" + string(fakta)},
	}, true, 1600)
	if err != nil {
		return Analysis{Tersedia: false, Model: c.Model(), Alasan: "Analisis AI gagal: " + err.Error()}
	}

	var out Analysis
	if err := json.Unmarshal([]byte(llm.ExtractJSON(raw)), &out); err != nil {
		return Analysis{Tersedia: false, Model: c.Model(), Alasan: "Balasan model tidak berbentuk JSON yang diharapkan."}
	}
	if strings.TrimSpace(out.Ringkasan) == "" && len(out.Rekomendasi) == 0 {
		return Analysis{Tersedia: false, Model: c.Model(), Alasan: "Model tidak mengembalikan analisis."}
	}

	out.Tersedia = true
	out.Model = c.Model()
	out.rapikan()
	return out
}

// rapikan membatasi panjang daftar dan menormalkan nilai dampak, supaya tata letak halaman
// tidak rusak kalau model mengabaikan batasan di prompt.
func (a *Analysis) rapikan() {
	a.SorotanPositif = potong(a.SorotanPositif, 4)
	a.PerluDiperhatikan = potong(a.PerluDiperhatikan, 4)
	if len(a.Rekomendasi) > 4 {
		a.Rekomendasi = a.Rekomendasi[:4]
	}
	for i := range a.Rekomendasi {
		d := strings.ToLower(strings.TrimSpace(a.Rekomendasi[i].Dampak))
		if d != "tinggi" && d != "sedang" && d != "rendah" {
			d = "sedang"
		}
		a.Rekomendasi[i].Dampak = d
	}
}

func potong(list []string, n int) []string {
	var out []string
	for _, s := range list {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
		if len(out) == n {
			break
		}
	}
	return out
}

// StatusLLM dipakai UI untuk menjelaskan apa yang perlu diisi agar analisis aktif.
type StatusLLM struct {
	Aktif    bool   `json:"aktif"`
	Model    string `json:"model,omitempty"`
	BaseURL  string `json:"base_url,omitempty"`
	Petunjuk string `json:"petunjuk,omitempty"`
}

func Status(c *llm.Client) StatusLLM {
	if c == nil || !c.Enabled() {
		return StatusLLM{
			Aktif: false,
			Petunjuk: "Setel LLM_API_KEY lalu jalankan ulang server. Bebas penyedia selama antarmukanya kompatibel OpenAI — " +
				"OpenAI, Groq, OpenRouter, DeepSeek, Together, xAI, atau model lokal lewat Ollama/LM Studio. " +
				"Atur LLM_BASE_URL dan LLM_MODEL sesuai penyedia yang dipakai.",
		}
	}
	return StatusLLM{Aktif: true, Model: c.Model(), BaseURL: c.BaseURL()}
}
