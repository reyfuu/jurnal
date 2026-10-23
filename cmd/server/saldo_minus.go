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
	// Sejak & Bukti: tanggal dan No. Bukti saat saldo akun terakhir kali berbalik minus ("SA" =
	// sudah minus sejak saldo awal), titik awal mencari jurnal yang salah.
	Sejak string `json:"sejak,omitempty"`
	Bukti string `json:"bukti,omitempty"`
}

// subtypeBolehMinus: laba (rugi) berjalan dan laba (rugi) ditahan boleh minus (rugi) tanpa
// berarti salah catat, jadi tidak diperingatkan. Kecuali 3299 penampung selisih saldo awal:
// subtype-nya juga retained, tetapi saldonya minus berarti isian saldo awal yang salah.
var subtypeBolehMinus = []string{"current_earnings", "retained", "retained_earnings"}

func bolehMinus(subtype, kode string) bool {
	return slices.Contains(subtypeBolehMinus, subtype) && kode != coa.KodeSelisihSaldoAwal
}

func (s *tenant) saldoMinus(period string) []saldoMinus {
	out := []saldoMinus{}
	if j, ok := s.jurnalPeriode(period); ok {
		for _, b := range j.tb {
			if b.Saldo >= 0 || bolehMinus(b.Subtype, b.Kode) {
				continue
			}
			normal, lawan := "debit", "kredit"
			if b.Normal == coa.Kredit {
				normal, lawan = "kredit", "debit"
			}
			m := saldoMinus{Jenis: "akun", Kode: b.Kode, Nama: b.Nama, Normal: normal, Nilai: -b.Saldo,
				Pesan: "Saldo " + b.Kode + " " + b.Nama + " " + finance.FormatRupiahPenuh(-b.Saldo) + " di sisi " + lawan +
					", padahal saldo normalnya " + normal + "."}
			m.Sejak, m.Bukti = s.mulaiMinus(b.Kode, period)
			switch {
			case m.Bukti == "SA":
				m.Pesan += " Sudah minus sejak saldo awal " + finance.FormatTanggalID(m.Sejak) + ": periksa isian saldo awalnya."
			case m.Bukti != "":
				m.Pesan += " Mulai minus " + finance.FormatTanggalID(m.Sejak) + " di No. Bukti " + m.Bukti + ": periksa jurnal itu dan transaksi sebelumnya."
			default:
				m.Pesan += " Periksa transaksi atau saldo awalnya."
			}
			out = append(out, m)
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

// mulaiMinus: tanggal & No. Bukti saat saldo akun terakhir kali berbalik minus, ditelusuri lewat
// Buku Besar dari awal pembukuan sampai akhir periode ("SA" = minus sejak saldo awal).
func (s *tenant) mulaiMinus(kode, period string) (sejak, bukti string) {
	t, err := time.Parse("2006-01", period)
	if err != nil {
		return "", ""
	}
	s.mu.RLock()
	mulai := s.bukuAwalLocked().Tanggal
	s.mu.RUnlock()
	akhir := t.AddDate(0, 1, -1).Format(time.DateOnly)
	if mulai > akhir {
		return "", ""
	}
	bb := s.bukuBesar(mulai, akhir, kode)
	if len(bb.Akun) != 1 {
		return "", ""
	}
	a := bb.Akun[0]
	if a.SaldoAwal < 0 {
		sejak, bukti = mulai, "SA"
	}
	prev := a.SaldoAwal
	for _, b := range a.Baris {
		if b.Saldo < 0 && prev >= 0 {
			sejak, bukti = b.Tanggal, b.Bukti
		}
		prev = b.Saldo
	}
	return sejak, bukti
}

func (s *tenant) handleSaldoMinus(w http.ResponseWriter, r *http.Request) {
	period := periodOrDefault(r)
	if _, err := time.Parse("2006-01", period); err != nil {
		http.Error(w, "periode harus YYYY-MM", http.StatusBadRequest)
		return
	}
	writeJSON(w, s.saldoMinus(period))
}
