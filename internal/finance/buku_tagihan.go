package finance

import (
	"strconv"
	"time"
)

// TagihanDariBuku: piutang & utang belum lunas dari buku besar dalam bentuk Tagihan, supaya
// dashboard (kartu piutang, daftar kewajiban) dan utang per supplier memakai angka yang sama
// dengan Neraca. Jatuh tempo dianggap 30 hari setelah tanggal transaksi.
func TagihanDariBuku(piutang, utang UmurSaldoResult) []Tagihan {
	var out []Tagihan
	for _, u := range []UmurSaldoResult{piutang, utang} {
		jenis := TagihanUtang
		if u.Piutang {
			jenis = TagihanPiutang
		}
		for _, b := range u.Baris {
			jt := b.Tanggal
			if t, err := time.Parse(time.DateOnly, b.Tanggal); err == nil {
				jt = t.AddDate(0, 0, 30).Format(time.DateOnly)
			}
			nomor := "Saldo awal"
			if b.TransaksiID > 0 {
				nomor = "TRX-" + strconv.Itoa(b.TransaksiID)
			}
			out = append(out, Tagihan{ID: b.TransaksiID, Nomor: nomor, Pihak: b.Pihak, Jenis: jenis, JatuhTempo: jt, Nominal: b.Nilai, Keterangan: b.Keterangan})
		}
	}
	return out
}
