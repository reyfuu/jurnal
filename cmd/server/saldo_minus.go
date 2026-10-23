package main

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
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
	// Titik: setiap saat saldo akun berbalik minus sejak awal pembukuan, urut tanggal — jurnal
	// yang perlu diperiksa, semuanya sekaligus. Bukti "SA" = sudah minus sejak saldo awal.
	Titik []titikMinus `json:"titik,omitempty"`
}

type titikMinus struct {
	Tanggal string `json:"tanggal"`
	Bukti   string `json:"bukti"`
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
			m.Titik = s.titikMinus(b.Kode, period)
			if len(m.Titik) == 0 {
				m.Pesan += " Periksa transaksi atau saldo awalnya."
			} else {
				daftar := make([]string, len(m.Titik))
				for i, p := range m.Titik {
					daftar[i] = finance.FormatTanggalID(p.Tanggal) + " (" + p.Bukti + ")"
					if p.Bukti == "SA" {
						daftar[i] = finance.FormatTanggalID(p.Tanggal) + " (saldo awal)"
					}
				}
				m.Pesan += " Saldo berbalik minus di " + strconv.Itoa(len(m.Titik)) + " titik: " + strings.Join(daftar, ", ") + ". Periksa jurnal-jurnal itu."
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

// titikMinus: setiap saat saldo akun berbalik dari tidak minus menjadi minus, ditelusuri lewat
// Buku Besar dari awal pembukuan sampai akhir periode ("SA" = minus sejak saldo awal).
func (s *tenant) titikMinus(kode, period string) []titikMinus {
	t, err := time.Parse("2006-01", period)
	if err != nil {
		return nil
	}
	s.mu.RLock()
	mulai := s.bukuAwalLocked().Tanggal
	s.mu.RUnlock()
	akhir := t.AddDate(0, 1, -1).Format(time.DateOnly)
	if mulai > akhir {
		return nil
	}
	bb := s.bukuBesar(mulai, akhir, kode)
	if len(bb.Akun) != 1 {
		return nil
	}
	a := bb.Akun[0]
	var out []titikMinus
	if a.SaldoAwal < 0 {
		out = append(out, titikMinus{mulai, "SA"})
	}
	prev := a.SaldoAwal
	for _, b := range a.Baris {
		if b.Saldo < 0 && prev >= 0 {
			out = append(out, titikMinus{b.Tanggal, b.Bukti})
		}
		prev = b.Saldo
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
