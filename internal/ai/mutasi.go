package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"pajak/internal/finance"
	"pajak/internal/llm"
)

// Pengelompokan mutasi bank: hanya kelompok baris yang tidak dikenali aturan kata kunci yang
// dikirim, dengan nomor rekening/referensi sudah disamarkan (finance.SamarkanAngka).

// MaksKelompokPerPanggilan: batas kelompok per permintaan model (sekitar 2–3 rb token).
const MaksKelompokPerPanggilan = 60

const promptMutasi = `Kamu membantu pemilik UMKM Indonesia mengelompokkan baris mutasi rekening bank ke jenis transaksi pembukuan. Setiap baris berisi nomor, arah uang (masuk/keluar), keterangan dari bank (nomor rekening disamarkan "#"), dan nominal.

ATURAN:
1. Pilih jenis HANYA dari daftar yang sesuai arahnya. Jangan membuat jenis baru.
2. Jika keterangan tidak cukup untuk menebak dengan wajar, isi "jenis" dengan "" (kosong) — pemilik akan memilih sendiri.
3. "alasan": maksimal 12 kata bahasa awam, kenapa jenis itu.
4. Hati-hati: transfer masuk dari nama orang belum tentu penjualan (bisa setoran modal/pinjaman); tarik tunai belum tentu uang pribadi.

Balas HANYA JSON: {"hasil":[{"no":1,"jenis":"...","alasan":"..."}]}`

// KelompokkanMutasi: saran jenis per kelompok (kunci = KelompokMutasi.Kunci). usaha = konteks
// singkat (jenis & nama usaha) supaya tebakan sesuai bidangnya.
func KelompokkanMutasi(ctx context.Context, c *llm.Client, usaha string, kel []finance.KelompokMutasi) (map[string]finance.SaranJenis, error) {
	if c == nil || !c.Enabled() {
		return nil, fmt.Errorf("AI belum aktif")
	}
	if len(kel) == 0 {
		return map[string]finance.SaranJenis{}, nil
	}
	var masuk, keluar []string
	for _, j := range finance.JenisList {
		if j.Pos == finance.PosPindah { // perlu rekening asal & tujuan; hanya lewat Catat
			continue
		}
		if j.Arah == "masuk" {
			masuk = append(masuk, j.Nama)
		} else {
			keluar = append(keluar, j.Nama)
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Usaha: %s\nJenis untuk uang MASUK: %s\nJenis untuk uang KELUAR: %s\n\nBaris:\n", usaha, strings.Join(masuk, "; "), strings.Join(keluar, "; "))
	for i, k := range kel {
		fmt.Fprintf(&b, "%d | %s | %s | Rp%d\n", i+1, k.Arah, k.Keterangan, k.Nominal)
	}
	raw, err := c.Complete(ctx, []llm.Message{{Role: "system", Content: promptMutasi}, {Role: "user", Content: b.String()}}, true, 40+len(kel)*30)
	if err != nil {
		return nil, err
	}
	var out struct {
		Hasil []struct {
			No            int
			Jenis, Alasan string
		} `json:"hasil"`
	}
	if err := json.Unmarshal([]byte(llm.ExtractJSON(raw)), &out); err != nil {
		return nil, fmt.Errorf("balasan model bukan JSON")
	}
	hasil := map[string]finance.SaranJenis{}
	for _, h := range out.Hasil {
		if h.No < 1 || h.No > len(kel) || strings.TrimSpace(h.Jenis) == "" {
			continue
		}
		hasil[kel[h.No-1].Kunci] = finance.SaranJenis{Jenis: strings.TrimSpace(h.Jenis), Alasan: h.Alasan}
	}
	return hasil, nil
}
