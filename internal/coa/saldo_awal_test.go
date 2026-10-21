package coa

import (
	"testing"

	"pajak/internal/finance"
)

// CoA manufaktur: baris saldo awal memakai kode & nama akunnya sendiri, tanpa akun pengurang,
// dan kode akun baru mengikuti kelompoknya.
func TestTemplateSaldoAwal_Manufaktur(t *testing.T) {
	p := finance.BusinessProfile{BentukUsaha: "op"}
	active := ActiveFrom(Manufaktur, p)
	peta := PetaDari(Manufaktur)
	kat := map[string]string{}
	for _, b := range TemplateSaldoAwal(active, peta, nil) {
		kat[b.Kode] = b.Kategori
	}
	for kode, mau := range map[string]string{"1102": "kas", "1111": "kas", "1201": "piutang", "1203": "aset_lain", "1301": "persediaan",
		"1305": "persediaan", "2101": "utang", "2131": "pinjaman", "2501": "pinjaman", "2105": "kewajiban_lain", "3103": "modal", "3201": "laba_ditahan"} {
		if kat[kode] != mau {
			t.Errorf("akun %s: kategori %q, mau %q", kode, kat[kode], mau)
		}
	}
	for _, kode := range []string{"1309", "1503", "1552", "3203", "3204", "3901"} {
		if _, ada := kat[kode]; ada {
			t.Errorf("akun %s tidak diisi di saldo awal (aset tetap dari daftar aset, pengurang, penampung)", kode)
		}
	}
	if k := KodeBaruSaldo(finance.KatKas, active, peta, nil); k != "1116" {
		t.Errorf("kode kas baru = %s, mau 1116", k)
	}
	if a, persis, ok := CocokAkunSaldo("utang usaha", active, peta); !ok || !persis || a.Kode != "2101" {
		t.Errorf("cocok utang usaha: %+v %v %v", a, persis, ok)
	}
}

func TestTebakKategoriNama(t *testing.T) {
	for nama, mau := range map[string]string{"Tabungan Mandiri": "kas", "Mobil pickup": "aset_tetap", "Utang ke Bank Mandiri": "pinjaman",
		"KUR BRI": "pinjaman", "Stok bahan kue": "persediaan", "Uang muka sewa": "aset_lain", "Utang pajak": "kewajiban_lain"} {
		if got := TebakKategoriNama(nama); got != mau {
			t.Errorf("%q → %q, mau %q", nama, got, mau)
		}
	}
}
