package coa

import (
	"slices"
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

// Akun yang ditambahkan sendiri di Saldo Awal (kas, aset lancar lain, kewajiban lain) muncul
// sebagai akun CoA tersendiri, urut kode, tanpa mengubah template bersama.
func TestAkunSaldoAwal_AkunBaruMasukCoA(t *testing.T) {
	peta := PetaDari(Master)
	rincian := []finance.SaldoAkun{
		{Kode: "1105", Nama: "Bank BCA", Kategori: finance.KatKas},
		{Kode: "1410", Nama: "Uang Jaminan", Kategori: finance.KatAsetLain},
		{Kode: "2120", Nama: "Utang Pemegang Saham", Kategori: finance.KatKewajibanLain},
		{Kode: "1102", Nama: "Bank", Kategori: finance.KatKas},          // akun CoA sendiri
		{Kode: "1202", Nama: "Piutang X", Kategori: finance.KatPiutang}, // piutang tetap di akun perannya
		{Kode: "1105", Nama: "Bank BCA", Kategori: finance.KatKas},      // dobel: satu akun
	}
	n := len(Master)
	all := AkunSaldoAwal(rincian, Master, peta)
	if len(Master) != n || len(all) != n+3 {
		t.Fatalf("akun tambahan: %d (template %d → %d)", len(all)-n, n, len(Master))
	}
	for kode, kat := range map[string]string{"1105": finance.KatKas, "1410": finance.KatAsetLain, "2120": finance.KatKewajibanLain} {
		a, ok := Find(all, kode)
		if !ok || a.Catatan != CatatanSaldoAwal || KategoriSaldoAkun(a, peta) != kat {
			t.Errorf("%s: %+v kategori %q", kode, a, KategoriSaldoAkun(a, peta))
		}
	}
	if i := slices.IndexFunc(all, func(a Account) bool { return a.Kode == "1105" }); all[i-1].Kode != "1104" || all[i+1].Kode != "1201" {
		t.Errorf("1105 harus di antara 1104 dan 1201: %s, %s", all[i-1].Kode, all[i+1].Kode)
	}
}

// Saldo awal kas dengan akun sendiri (akun CoA bernama sama atau akun tambahan) dipisah dari
// akun Kas; kas lain yang kodenya tidak cocok tetap di Kas.
func TestKasAwalLain(t *testing.T) {
	peta := PetaDari(Master)
	ob := finance.OpeningBalance{Rincian: []finance.SaldoAkun{
		{Kode: "1101", Nama: "Kas", Kategori: finance.KatKas, Nominal: 5_000_000},
		{Kode: "1102", Nama: "Bank", Kategori: finance.KatKas, Nominal: 20_000_000},
		{Kode: "1105", Nama: "Bank BCA", Kategori: finance.KatKas, Nominal: 10_000_000},
		{Kode: "1103", Nama: "OVO", Kategori: finance.KatKas, Nominal: 1_000_000}, // 1103 = E-Wallet (nama lain)
	}}
	got := KasAwalLain(ob, AkunSaldoAwal(ob.Rincian, Master, peta), peta)
	if len(got) != 2 || got["1102"] != 20_000_000 || got["1105"] != 10_000_000 {
		t.Fatalf("kas per akun: %v", got)
	}
}

// Baris tanpa kode mendapat kode: nama sama dengan akun CoA → akun itu; nama sama dengan baris
// lain → kode baris itu; selain itu kode baru. Kategori berakun peran mendapat kode akun perannya.
func TestIsiKodeSaldo(t *testing.T) {
	active := ActiveFrom(Master, finance.BusinessProfile{BentukUsaha: "op"})
	rincian := []finance.SaldoAkun{
		{Kode: "1105", Nama: "Bank BCA", Kategori: finance.KatKas},
		{Nama: "Deposito", Kategori: finance.KatKas},
		{Nama: "deposito", Kategori: finance.KatKas},
		{Nama: "Bank", Kategori: finance.KatKas},
		{Nama: "Kredit Koperasi", Kategori: finance.KatPinjaman},
		{Nama: "Biaya Dibayar di Muka", Kategori: finance.KatAsetLain},
		{Nama: "Bank BCA", Kategori: finance.KatKas},
	}
	IsiKodeSaldo(rincian, active, PetaDari(Master))
	want := []string{"1105", "1106", "1106", "1102", "2102", "1401", "1105"}
	for i, r := range rincian {
		if r.Kode != want[i] {
			t.Errorf("%s: kode %q, mau %q", r.Nama, r.Kode, want[i])
		}
	}
}
