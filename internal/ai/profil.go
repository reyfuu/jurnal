package ai

import (
	"context"
	"encoding/json"
	"strings"

	"pajak/internal/finance"
	"pajak/internal/llm"
	"pajak/internal/tax"
)

// Laporan AI untuk usaha yang belum punya transaksi (baru daftar / belum berlangganan): satu-
// satunya data adalah perkiraan di form sign-up, jadi analisisnya sengaja terbatas — posisi
// margin terhadap DJP, risiko pajak, dan skema pajak paling hemat.

type SaranRingkas struct {
	Judul  string   `json:"judul"`
	Hemat  int64    `json:"hemat_setahun"`
	Alasan []string `json:"alasan"`
}

type FaktaProfil struct {
	NamaUsaha    string            `json:"nama_usaha"`
	JenisUsaha   string            `json:"jenis_usaha"`
	BentukUsaha  string            `json:"bentuk_usaha"`
	Kota         string            `json:"kota,omitempty"`
	PKP          bool              `json:"pkp"`
	OmzetSetahun int64             `json:"omzet_setahun"` // perkiraan dari form sign-up
	LabaSetahun  int64             `json:"laba_setahun"`
	NPMBps       int64             `json:"npm_bps"`
	KLU          *tax.KLUBenchmark `json:"benchmark_klu,omitempty"`
	StatusNPM    string            `json:"status_npm"` // kurang_data | di_bawah | wajar
	CTTORBps     int64             `json:"cttor_bps,omitempty"`
	Risiko       []tax.Risiko      `json:"risiko"`
	Saran        *SaranRingkas     `json:"saran_skema_pajak,omitempty"`
}

func FaktaDariProfil(p finance.BusinessProfile) FaktaProfil {
	c := p.CekAwal()
	f := FaktaProfil{NamaUsaha: p.NamaUsaha, JenisUsaha: p.JenisUsaha, BentukUsaha: p.BentukUsaha, Kota: p.Kota, PKP: p.PKP,
		OmzetSetahun: p.OmzetTahunan, LabaSetahun: p.LabaSebelumPajak, KLU: c.KLU, StatusNPM: c.Status, CTTORBps: c.CTTORBps, Risiko: c.Risiko}
	if p.OmzetTahunan > 0 {
		f.NPMBps = p.LabaSebelumPajak * 10_000 / p.OmzetTahunan
	}
	if s := c.Saran; s != nil {
		f.Saran = &SaranRingkas{Judul: s.Judul, Hemat: s.Hemat, Alasan: s.Alasan}
	}
	return f
}

const promptProfil = `Kamu analis keuangan untuk UMKM Indonesia. Usaha ini BARU mendaftar dan BELUM mencatat transaksi. Satu-satunya data adalah perkiraan omzet dan laba setahun yang diisi pemilik di form pendaftaran, ditambah hasil hitungan sistem (benchmark NPM/CTTOR DJP, risiko pajak, saran skema pajak) dalam JSON.

ATURAN MUTLAK:
1. JANGAN menghitung atau mengarang angka. Pakai HANYA angka di JSON, salin persis.
2. Sampaikan di ringkasan bahwa ini analisis awal dari perkiraan pendaftaran, jadi terbatas; analisis lengkap (laba rugi, arus kas, neraca) tersedia setelah transaksi dicatat.
3. npm_bps dan cttor_bps adalah basis poin: 700 berarti 7,00%.
4. Bahasa Indonesia lugas untuk pemilik usaha kecil. Jangan memberi nasihat pajak yang pasti; arahkan verifikasi ke konsultan.
5. Bahas: posisi margin terhadap rata-rata DJP (bila ada), risiko pajak, dan skema pajak yang disarankan sistem beserta alasannya.

Balas HANYA objek JSON:
{
  "ringkasan": "2-3 kalimat",
  "sorotan_positif": ["maksimal 4 butir"],
  "perlu_diperhatikan": ["maksimal 4 butir"],
  "rekomendasi": [{"judul":"aksi singkat","dampak":"tinggi|sedang|rendah","detail":"1-2 kalimat"}]
}
Maksimal 4 rekomendasi; salah satunya mulai mencatat transaksi supaya analisis lengkap.`

// AnalisaProfil: narasi model atas FaktaProfil. Tanpa LLM_API_KEY tidak menghubungi jaringan.
func AnalisaProfil(ctx context.Context, c *llm.Client, f FaktaProfil) Analysis {
	if c == nil || !c.Enabled() {
		return Analysis{Tersedia: false, Alasan: "Analisis AI belum aktif. Isi LLM_API_KEY (dan LLM_BASE_URL serta LLM_MODEL bila bukan OpenAI) untuk mengaktifkannya."}
	}
	fakta, err := json.Marshal(f)
	if err != nil {
		return Analysis{Tersedia: false, Alasan: "Gagal menyiapkan data: " + err.Error()}
	}
	raw, err := c.Complete(ctx, []llm.Message{
		{Role: "system", Content: promptProfil},
		{Role: "user", Content: "Data perkiraan dari form pendaftaran:\n\n" + string(fakta)},
	}, true, 1200)
	if err != nil {
		return Analysis{Tersedia: false, Model: c.Model(), Alasan: "Analisis AI gagal: " + err.Error()}
	}
	var out Analysis
	if err := json.Unmarshal([]byte(llm.ExtractJSON(raw)), &out); err != nil || (strings.TrimSpace(out.Ringkasan) == "" && len(out.Rekomendasi) == 0) {
		return Analysis{Tersedia: false, Model: c.Model(), Alasan: "Model tidak mengembalikan analisis."}
	}
	out.Tersedia, out.Model = true, c.Model()
	out.rapikan()
	return out
}
