package finance

import (
	"slices"
	"time"
)

// Daftar periode laporan diturunkan dari data usaha: dari bulan transaksi paling awal sampai
// bulan berjalan. Sebelumnya dipatok Januari–Agustus 2026 (periode data contoh), sehingga
// transaksi bulan lain tidak bisa dilaporkan.

const maksPeriode = 36

type PilihanPeriode struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Bawaan bool   `json:"bawaan,omitempty"` // dibuka pertama kali: bulan terakhir yang punya transaksi
}

// BulanBerjalan: "YYYY-MM" menurut WIB.
func BulanBerjalan() string { return time.Now().In(WIB).Format("2006-01") }

// geserBulan: "2026-01", -1 → "2025-12".
func geserBulan(p string, n int) string {
	t, err := time.Parse("2006-01", p)
	if err != nil {
		return p
	}
	return t.AddDate(0, n, 0).Format("2006-01")
}

// DaftarPeriode: bulan terbaru dulu. ekstra = periode lain yang harus ikut (neraca awal, titik
// nol kartu stok, periode data contoh).
func DaftarPeriode(txs []Transaction, ekstra []string, sekarang string) []PilihanPeriode {
	awal, akhir, terakhirTx := sekarang, sekarang, ""
	for _, t := range txs {
		if len(t.Tanggal) < 7 {
			continue
		}
		b := t.Tanggal[:7]
		awal, akhir = min(awal, b), max(akhir, b)
		terakhirTx = max(terakhirTx, b)
	}
	for _, b := range ekstra {
		if PeriodeSah(b) {
			awal, akhir = min(awal, b), max(akhir, b)
		}
	}
	awal = max(awal, geserBulan(akhir, -(maksPeriode-1)))
	bawaan := terakhirTx
	if bawaan == "" {
		bawaan = sekarang
	}
	var out []PilihanPeriode
	for p := akhir; p >= awal; p = geserBulan(p, -1) {
		out = append(out, PilihanPeriode{ID: p, Label: LabelPeriode(p), Bawaan: p == bawaan})
	}
	return out
}

// bulanData: bulan (YYYY-MM) yang punya transaksi, urut naik.
func bulanData(txs []Transaction) []string {
	var out []string
	for _, t := range txs {
		if len(t.Tanggal) >= 7 && !slices.Contains(out, t.Tanggal[:7]) {
			out = append(out, t.Tanggal[:7])
		}
	}
	slices.Sort(out)
	return out
}
