package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"pajak/internal/llm"
	"pajak/internal/tax"
)

// Saran KLU: pemilik usaha menceritakan usahanya dengan bahasa sehari-hari, model memilih
// kategori dari daftar KLU yang dikirim (tidak boleh di luar daftar) dan menjelaskannya.

type SaranKLU struct {
	Kode   string   `json:"kode"`
	Nama   string   `json:"nama"`
	Jenis  string   `json:"jenis"`
	Alasan string   `json:"alasan,omitempty"`
	Contoh []string `json:"contoh"`
}

const promptKLU = `Kamu membantu pemilik UMKM Indonesia memilih kategori bidang usaha (KLU) yang paling cocok. Pemilik menceritakan usahanya dengan bahasa sehari-hari. Daftar KLU yang BOLEH dipilih diberikan (kode | nama | contoh kegiatan).

ATURAN:
1. Pilih 1-3 kode HANYA dari daftar, paling cocok dulu. Jangan mengarang kode.
2. Untuk tiap pilihan tulis "alasan": 1-2 kalimat bahasa awam, jelaskan kategori itu mencakup usaha apa dan kenapa cocok dengan cerita pemilik. Bila ada beda penting antar pilihan (mis. jual di toko vs online, eceran vs grosir), sebutkan.
3. Jangan menyebut lembaga atau sumber data.

Balas HANYA JSON: {"saran":[{"kode":"563","alasan":"..."}]}`

// PilihKLU: saran model untuk deskripsi usaha. Kode di luar daftar dibuang.
func PilihKLU(ctx context.Context, c *llm.Client, deskripsi, jenis string) ([]SaranKLU, error) {
	if c == nil || !c.Enabled() {
		return nil, fmt.Errorf("AI belum aktif")
	}
	klu := tax.BenchmarkAktif()
	var daftar strings.Builder
	for _, k := range klu {
		fmt.Fprintf(&daftar, "%s | %s | %s\n", k.Kode, k.Nama, strings.Join(tax.ContohKLU(k.Kode, 3), "; "))
	}
	user := "Cerita pemilik: " + deskripsi
	if jenis != "" {
		user += "\nJenis usaha yang dipilih pemilik: " + jenis
	}
	raw, err := c.Complete(ctx, []llm.Message{
		{Role: "system", Content: promptKLU},
		{Role: "user", Content: user + "\n\nDaftar KLU:\n" + daftar.String()},
	}, true, 600)
	if err != nil {
		return nil, err
	}
	var out struct {
		Saran []struct{ Kode, Alasan string } `json:"saran"`
	}
	if err := json.Unmarshal([]byte(llm.ExtractJSON(raw)), &out); err != nil {
		return nil, fmt.Errorf("balasan model bukan JSON")
	}
	var hasil []SaranKLU
	for _, s := range out.Saran {
		i := slices.IndexFunc(klu, func(k tax.KLUBenchmark) bool { return k.Kode == strings.TrimSpace(s.Kode) })
		if i < 0 || slices.ContainsFunc(hasil, func(h SaranKLU) bool { return h.Kode == klu[i].Kode }) {
			continue
		}
		hasil = append(hasil, SaranKLU{Kode: klu[i].Kode, Nama: klu[i].Nama, Jenis: tax.JenisKLU(klu[i].Kode),
			Alasan: strings.TrimSpace(s.Alasan), Contoh: tax.ContohKLU(klu[i].Kode, 4)})
		if len(hasil) == 3 {
			break
		}
	}
	if len(hasil) == 0 {
		return nil, fmt.Errorf("model tidak memilih kode dari daftar")
	}
	return hasil, nil
}
