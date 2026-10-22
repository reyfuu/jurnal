package coa

import (
	"testing"

	"pajak/internal/finance"
)

// Bugslist 22 Okt #1: di CoA manufaktur pinjaman bank dibukukan ke 2131 (Utang Bank Jangka
// Pendek), sedangkan 2102 adalah GRNI. Baris saldo awal "2102 Hutang Bank" (kode bawaan
// penomoran dagang) harus tampil & tersimpan dengan kode akun tempat saldonya dibukukan.
func TestIsiKodeSaldo_KategoriPeranIkutCoAUsaha(t *testing.T) {
	p := finance.BusinessProfile{JenisUsaha: "manufaktur", BentukUsaha: "op", MetodePersediaan: "periodik"}
	active, peta := ActiveFrom(Manufaktur, p), PetaDari(Manufaktur)
	rincian := []finance.SaldoAkun{
		{Kode: "2102", Nama: "Hutang Bank", Kategori: finance.KatPinjaman, Nominal: 5_000_000},
		{Kode: "2101", Nama: "Hutang Usaha", Kategori: finance.KatUtang, Nominal: 1_000_000},
		{Kode: "1201", Nama: "Piutang Usaha", Kategori: finance.KatPiutang, Nominal: 2_000_000},
		{Kode: "", Nama: "Persediaan", Kategori: finance.KatPersediaan, Nominal: 3_000_000},
		{Kode: "3299", Nama: "Laba Ditahan - Selisih Saldo Awal", Kategori: finance.KatLabaDitahan, Nominal: 1},
	}
	IsiKodeSaldo(rincian, active, peta)
	for i, mau := range []string{peta.Kode(PeranPinjaman), peta.Kode(PeranUtangUsaha), peta.Kode(PeranPiutang), peta.Kode(PeranPersediaan), "3299"} {
		if rincian[i].Kode != mau {
			t.Errorf("%s: kode %q, mau %q", rincian[i].Nama, rincian[i].Kode, mau)
		}
	}
	if rincian[0].Kode != "2131" {
		t.Fatalf("pinjaman manufaktur harus 2131, dapat %q", rincian[0].Kode)
	}
}

// Kode tampilan saldo awal tersimpan lama juga dikoreksi tanpa mengubah slice asal.
func TestKodePeranSaldo_TidakMengubahAsal(t *testing.T) {
	asal := []finance.SaldoAkun{{Kode: "2102", Nama: "Hutang Bank", Kategori: finance.KatPinjaman}}
	out := KodePeranSaldo(asal, PetaDari(Manufaktur))
	if out[0].Kode != "2131" || asal[0].Kode != "2102" {
		t.Fatalf("out %q asal %q", out[0].Kode, asal[0].Kode)
	}
}
