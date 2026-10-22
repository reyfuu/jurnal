package main

import (
	"net/http"
	"slices"
	"time"

	"pajak/internal/coa"
	"pajak/internal/finance"
)

// Peringatan saldo minus (PSAK): saldo akun yang berlawanan dengan saldo normalnya — kas, bank,
// atau persediaan bersaldo kredit, utang bersaldo debit, dan seterusnya. Penjualan di kredit
// atau beban di debit justru normal, jadi tidak diperingatkan. Ditambah stok barang minus.
// Hanya peringatan: transaksi tetap tersimpan.

type saldoMinus struct {
	Jenis  string `json:"jenis"` // akun | stok
	Kode   string `json:"kode,omitempty"`
	Nama   string `json:"nama"`
	Normal string `json:"normal,omitempty"` // saldo normal akun: debit | kredit
	Nilai  int64  `json:"nilai"`            // rupiah (akun) atau jumlah per seribu (stok), positif
	Satuan string `json:"satuan,omitempty"`
	Pesan  string `json:"pesan"`
}

// subtypeBolehMinus: akun penampung laba berjalan bisa rugi tanpa berarti salah catat.
var subtypeBolehMinus = []string{"current_earnings"}

func (s *tenant) saldoMinus(period string) []saldoMinus {
	out := []saldoMinus{}
	if j, ok := s.jurnalPeriode(period); ok {
		for _, b := range j.tb {
			if b.Saldo >= 0 || slices.Contains(subtypeBolehMinus, b.Subtype) {
				continue
			}
			normal, lawan := "debit", "kredit"
			if b.Normal == coa.Kredit {
				normal, lawan = "kredit", "debit"
			}
			out = append(out, saldoMinus{Jenis: "akun", Kode: b.Kode, Nama: b.Nama, Normal: normal, Nilai: -b.Saldo,
				Pesan: "Saldo " + b.Kode + " " + b.Nama + " " + finance.FormatRupiahPenuh(-b.Saldo) + " di sisi " + lawan +
					", padahal saldo normalnya " + normal + ". Periksa transaksi atau saldo awalnya."})
		}
	}
	k, stok := s.kartu()
	for _, it := range stok {
		if p, ok := k.Posisi[it.ID]; ok && p.Qty < 0 {
			out = append(out, saldoMinus{Jenis: "stok", Nama: it.Nama, Nilai: -p.Qty, Satuan: it.Satuan,
				Pesan: "Stok " + it.Nama + " minus " + finance.FormatQty(-p.Qty) + " " + it.Satuan + ": barang keluar melebihi yang tercatat masuk. Catat pembeliannya atau lakukan stok opname."})
		}
	}
	return out
}

func (s *tenant) handleSaldoMinus(w http.ResponseWriter, r *http.Request) {
	period := periodOrDefault(r)
	if _, err := time.Parse("2006-01", period); err != nil {
		http.Error(w, "periode harus YYYY-MM", http.StatusBadRequest)
		return
	}
	writeJSON(w, s.saldoMinus(period))
}
