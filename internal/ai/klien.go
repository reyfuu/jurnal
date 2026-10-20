package ai

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"pajak/internal/llm"
	"pajak/internal/pengetahuan"
)

// Analisa AI per klien di Panel Master. Semua angka dihitung Go (FaktaKlien); model hanya
// menulis penilaian dan WAJIB merujuk sumber dari daftar yang dikirim — artikel terbit &
// pengetahuan terverifikasi (pengetahuan.Potongan) plus benchmark nasional (BM-<KLU>).

type FaktaKlien struct {
	Nama              string   `json:"nama"`
	JenisUsaha        string   `json:"jenis_usaha"`
	BentukUsaha       string   `json:"bentuk_usaha"`
	PKP               bool     `json:"pkp"`
	KLU               string   `json:"klu,omitempty"`
	KLUNama           string   `json:"klu_nama,omitempty"`
	OmzetSetahun      int64    `json:"omzet_setahun"` // rata-rata bulanan × 12
	NPMBps            int64    `json:"npm_bps"`
	BenchmarkNPMBps   int64    `json:"benchmark_npm_bps,omitempty"`
	BenchmarkCTTORBps int64    `json:"benchmark_cttor_bps,omitempty"`
	SumberBenchmark   string   `json:"sumber_benchmark,omitempty"`
	StatusNPM         string   `json:"status_npm"` // kurang_data | di_bawah | wajar
	Risiko            []string `json:"risiko,omitempty"`
}

type PoinAnalisa struct {
	Judul  string   `json:"judul"`
	Isi    string   `json:"isi"`
	Sumber []string `json:"sumber"`
}

type AnalisaKlienHasil struct {
	Tersedia  bool          `json:"tersedia"`
	Alasan    string        `json:"alasan,omitempty"`
	Model     string        `json:"model,omitempty"`
	Penilaian string        `json:"penilaian,omitempty"`
	Poin      []PoinAnalisa `json:"poin,omitempty"`
}

const promptKlien = `Kamu konsultan pajak & pembukuan Master Pajak yang menilai satu klien UMKM untuk tim internal.
Kamu menerima FAKTA (angka yang SUDAH dihitung sistem) dan SUMBER bernomor.

ATURAN MUTLAK:
1. Jangan menghitung atau memperkirakan angka. Salin angka dari FAKTA apa adanya. *_bps adalah basis poin: 3320 = 33,20%.
2. Setiap poin WAJIB merujuk sumber dengan nomornya persis (mis. "A-001", "KB-003", "BM-563"). Pakai HANYA nomor yang ada di daftar SUMBER.
3. Jika tidak ada sumber yang mendukung, jangan buat poinnya.
4. Label dalam kurung menunjukkan jenis sumber. "(Praktik lapangan)" dan "(Pengetahuan umum)" adalah pengalaman tim, bukan aturan resmi: sampaikan sebagai praktik/kebiasaan, jangan sebagai kewajiban hukum.
4. Bahasa Indonesia lugas, untuk konsultan. Bukan nasihat hukum final: hal yang menyangkut SPT diverifikasi tim pajak.

Balas HANYA JSON:
{"penilaian":"2-3 kalimat kondisi klien","poin":[{"judul":"singkat","isi":"1-2 kalimat","sumber":["A-001"]}]}
Maksimal 4 poin, urut dari yang paling penting.`

// QueryKlien: kata kunci pencarian sumber dari profil & risiko klien.
func QueryKlien(f FaktaKlien, risiko []string) string {
	q := []string{f.KLUNama, f.JenisUsaha, "omzet pph final umkm pembukuan"}
	if f.PKP {
		q = append(q, "ppn pkp faktur")
	}
	if f.BentukUsaha != "op" {
		q = append(q, "badan pph 31e")
	}
	return strings.Join(append(q, risiko...), " ")
}

func AnalisaKlien(ctx context.Context, c *llm.Client, f FaktaKlien, sumber []pengetahuan.Potongan) AnalisaKlienHasil {
	if c == nil || !c.Enabled() {
		return AnalisaKlienHasil{Alasan: "Analisa AI belum aktif. Isi LLM_API_KEY (dan LLM_BASE_URL serta LLM_MODEL bila bukan OpenAI) untuk mengaktifkannya."}
	}
	refs := []string{}
	var daftar strings.Builder
	if f.KLU != "" && f.SumberBenchmark != "" {
		ref := "BM-" + f.KLU
		refs = append(refs, ref)
		fmt.Fprintf(&daftar, "[%s] Benchmark nasional KLU %s %s (%s): NPM %d bps, CTTOR %d bps\n", ref, f.KLU, f.KLUNama, f.SumberBenchmark, f.BenchmarkNPMBps, f.BenchmarkCTTORBps)
	}
	for _, p := range sumber {
		if !slices.Contains(refs, p.Ref) {
			refs = append(refs, p.Ref)
		}
		fmt.Fprintf(&daftar, "[%s] (%s) %s: %s\n", p.Ref, cmp.Or(p.Kategori, "Sumber"), p.Judul, p.Teks)
	}
	fakta, err := json.Marshal(f)
	if err != nil {
		return AnalisaKlienHasil{Alasan: "Gagal menyiapkan fakta: " + err.Error()}
	}
	raw, err := c.Complete(ctx, []llm.Message{
		{Role: "system", Content: promptKlien},
		{Role: "user", Content: "FAKTA:\n" + string(fakta) + "\n\nSUMBER:\n" + daftar.String()},
	}, true, 1200)
	if err != nil {
		return AnalisaKlienHasil{Model: c.Model(), Alasan: "Analisa AI gagal: " + err.Error()}
	}
	var out AnalisaKlienHasil
	if err := json.Unmarshal([]byte(llm.ExtractJSON(raw)), &out); err != nil || strings.TrimSpace(out.Penilaian) == "" {
		return AnalisaKlienHasil{Model: c.Model(), Alasan: "Balasan model tidak berbentuk JSON yang diharapkan."}
	}
	out.Tersedia, out.Model, out.Alasan = true, c.Model(), ""
	out.Poin = out.Poin[:min(4, len(out.Poin))]
	for i := range out.Poin {
		// Rujukan karangan dibuang: penilaian tanpa sumber yang bisa dicek tidak boleh tampil
		// seolah berdasar.
		out.Poin[i].Sumber = slices.DeleteFunc(out.Poin[i].Sumber, func(r string) bool { return !slices.Contains(refs, r) })
		if out.Poin[i].Sumber == nil {
			out.Poin[i].Sumber = []string{}
		}
	}
	return out
}
