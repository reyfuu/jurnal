package finance

import (
	"slices"
	"strings"
)

// CashWarning: hari ketika saldo kas akhir hari di bawah nol. Biasanya tanda ada pemasukan
// yang belum dicatat atau pengeluaran yang salah nominal; transaksi tetap disimpan.
type CashWarning struct {
	Tanggal  string `json:"tanggal"`
	Saldo    int64  `json:"saldo"`
	Penyebab []int  `json:"penyebab"` // ID pengeluaran lunas di hari itu
}

// CashWarnings menelusuri saldo kas harian dari uang yang bergerak (tanggal lunas / cicilan).
func CashWarnings(txs []Transaction, openingCash int64) []CashWarning {
	type move struct {
		date string
		tx   Transaction
	}
	var moves []move
	for _, t := range txs {
		for _, g := range t.GerakKas() { // lunas: sekali di tanggal lunas; bercicilan: per cicilan
			x := t
			x.Nominal = g.Nominal
			moves = append(moves, move{g.Tanggal, x})
		}
	}
	slices.SortStableFunc(moves, func(a, b move) int { return strings.Compare(a.date, b.date) })
	var out []CashWarning
	saldo := openingCash
	for i := 0; i < len(moves); {
		day := moves[i].date
		var keluar []int
		for ; i < len(moves) && moves[i].date == day; i++ {
			t := moves[i].tx
			if j, _ := jenisOf(t.Jenis); j.Arah == "masuk" {
				saldo += t.Nominal
			} else {
				saldo -= t.Nominal
				keluar = append(keluar, t.ID)
			}
		}
		if saldo < 0 {
			out = append(out, CashWarning{Tanggal: day, Saldo: saldo, Penyebab: keluar})
		}
	}
	return out
}
