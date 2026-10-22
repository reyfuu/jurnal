package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"pajak/internal/llm"
)

// Baca mutasi rekening dari gambar (halaman rekening koran hasil scan, atau foto layar m-banking).
// Model hanya menyalin baris yang terlihat; arah akhir, cek saldo berjalan, dan jenis transaksi
// dihitung aplikasi seperti PDF biasa.

type BarisScan struct {
	Tanggal    string `json:"tanggal"`
	Keterangan string `json:"keterangan"`
	Nominal    int64  `json:"nominal"`
	Arah       string `json:"arah"`  // masuk | keluar | "" (tidak jelas)
	Saldo      *int64 `json:"saldo"` // nil = tidak tertera
}

type MutasiScan struct {
	SaldoAwal *int64      `json:"saldo_awal"`
	Baris     []BarisScan `json:"baris"`
}

const promptMutasiScan = `Kamu menyalin mutasi rekening bank Indonesia dari gambar (halaman rekening koran hasil scan atau foto layar m-banking).

ATURAN:
1. Salin hanya baris transaksi yang terlihat, urut dari atas. Jangan mengarang; baris yang tidak terbaca dilewati.
2. "tanggal": YYYY-MM-DD. Bila tahun tidak tertulis, pakai tahun %d.
3. "keterangan": teks keterangan transaksi apa adanya (gabungkan bila terpotong ke baris berikutnya), tanpa angka nominal dan saldo.
4. "nominal": jumlah mutasi dalam rupiah, bilangan bulat positif tanpa pemisah ribuan dan tanpa sen (1.500.000,00 menjadi 1500000).
5. "arah": "masuk" untuk kredit / CR / uang masuk, "keluar" untuk debet / DB / uang keluar, "" bila tidak jelas.
6. "saldo": saldo setelah transaksi bila tertera, selain itu null.
7. "saldo_awal": saldo awal / saldo sebelumnya bila tertera di gambar, selain itu null. Baris saldo awal, saldo akhir, dan total bukan transaksi.

Balas HANYA JSON: {"saldo_awal":null,"baris":[{"tanggal":"","keterangan":"","nominal":0,"arah":"","saldo":null}]}`

// BacaMutasiScan membaca satu gambar halaman mutasi. tahun dipakai bila tahun tidak tertulis.
func BacaMutasiScan(ctx context.Context, c *llm.Client, mime string, gambar []byte, tahun int) (MutasiScan, error) {
	if c == nil || !c.Enabled() {
		return MutasiScan{}, fmt.Errorf("AI belum aktif")
	}
	// satu halaman rekening koran bisa berisi puluhan baris
	raw, _, err := c.CompleteGambar(ctx, fmt.Sprintf(promptMutasiScan, tahun), "Salin mutasi di gambar ini.", mime, gambar, true, 4000)
	if err != nil {
		return MutasiScan{}, err
	}
	var m MutasiScan
	if err := json.Unmarshal([]byte(llm.ExtractJSON(raw)), &m); err != nil {
		return MutasiScan{}, fmt.Errorf("balasan model bukan JSON")
	}
	return m, nil
}
