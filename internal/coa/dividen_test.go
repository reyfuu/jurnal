package coa

import (
	"testing"

	"pajak/internal/finance"
)

// Dividen PT: Dr Laba Ditahan / Cr Utang Dividen (bukan utang usaha), di CoA dagang & manufaktur.
func TestDividen_AkunUtangDividen(t *testing.T) {
	pt := finance.BusinessProfile{BentukUsaha: "pt", MetodePersediaan: "periodik"}
	for _, c := range []struct {
		nama         string
		akun         []Account
		utangDividen string
	}{{"dagang", Master, "2115"}, {"manufaktur", Manufaktur, "2123"}} {
		peta := PetaDari(c.akun)
		tx := finance.Transaction{ID: 1, Tanggal: "2026-02-15", Jenis: finance.JenisDividen, Nominal: 2_000_000}
		if got := AccountFor(tx, pt, peta); got != peta.Kode(PeranLabaDitahan) {
			t.Errorf("%s: akun dividen %s, mau laba ditahan", c.nama, got)
		}
		if got := CounterAccount(tx, peta); got != c.utangDividen {
			t.Errorf("%s: lawan dividen belum dibayar %s, mau %s", c.nama, got, c.utangDividen)
		}
		tx.Lunas, tx.TanggalLunas = true, "2026-03-10"
		var dr string
		for _, l := range JournalBerjalan([]finance.Transaction{tx}, "2026-03", pt, peta) {
			if l.Debit > 0 {
				dr = l.Kode
			}
		}
		if dr != c.utangDividen {
			t.Errorf("%s: pembayaran dividen mendebit %s, mau %s", c.nama, dr, c.utangDividen)
		}
	}
}
