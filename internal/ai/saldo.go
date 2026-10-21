package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"pajak/internal/finance"
	"pajak/internal/llm"
)

// Saran kategori akun saldo awal: pemilik mengetik nama akun sendiri ("Tabungan BRI", "Mobil
// box", "Pinjaman dari ipar"), model memilih satu kategori neraca dari daftar yang dikirim.

const promptSaldo = `Kamu membantu pemilik UMKM Indonesia mengisi saldo awal (neraca pembukaan). Pemilik mengetik nama akun sendiri; pilih SATU kategori neraca yang paling cocok dari daftar (id | label).

ATURAN:
1. Pilih id HANYA dari daftar. Jika nama akun terlalu kabur untuk ditebak dengan wajar, isi "kategori" dengan "".
2. Uang di rekening, tabungan, giro, deposito, dompet digital, saldo marketplace = kas.
3. Barang berwujud yang dipakai usaha lebih dari setahun (kendaraan, mesin, gedung, tanah, komputer, peralatan) = aset_tetap.
4. Utang ke pemasok atas pembelian barang = utang; pinjaman uang (bank, koperasi, leasing, keluarga) = pinjaman; utang pajak, gaji, dan kewajiban lain = kewajiban_lain.
5. "alasan": maksimal 12 kata bahasa awam.

Balas HANYA JSON: {"kategori":"kas","alasan":"..."}`

// SaranKategoriSaldo: kategori (id KategoriSaldoList) untuk nama akun; "" bila model ragu.
func SaranKategoriSaldo(ctx context.Context, c *llm.Client, usaha, nama string) (kategori, alasan string, err error) {
	if c == nil || !c.Enabled() {
		return "", "", fmt.Errorf("AI belum aktif")
	}
	var daftar strings.Builder
	for _, k := range finance.KategoriSaldoList {
		fmt.Fprintf(&daftar, "%s | %s\n", k.ID, k.Label)
	}
	raw, err := c.Complete(ctx, []llm.Message{
		{Role: "system", Content: promptSaldo},
		{Role: "user", Content: "Usaha: " + usaha + "\nNama akun: " + nama + "\n\nDaftar kategori:\n" + daftar.String()},
	}, true, 120)
	if err != nil {
		return "", "", err
	}
	var out struct{ Kategori, Alasan string }
	if err := json.Unmarshal([]byte(llm.ExtractJSON(raw)), &out); err != nil {
		return "", "", fmt.Errorf("balasan model bukan JSON")
	}
	for _, k := range finance.KategoriSaldoList {
		if k.ID == strings.TrimSpace(out.Kategori) {
			return k.ID, strings.TrimSpace(out.Alasan), nil
		}
	}
	return "", "", nil
}
