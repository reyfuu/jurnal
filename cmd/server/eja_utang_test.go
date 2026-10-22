package main

import (
	"testing"

	"pajak/internal/coa"
	"pajak/internal/finance"
	"pajak/internal/katalog"
)

// Bugslist 22 Okt #2: nama "Hutang …" yang sudah tersimpan (saldo awal tenant, template CoA
// terbitan Panel Master) ikut tampil sebagai "Utang …" setelah dimuat.
func TestEjaUtang_DataTersimpan(t *testing.T) {
	tn := newTenant(99)
	tn.restore(&state{Opening: &finance.OpeningBalance{Tanggal: "2026-01-01", Rincian: []finance.SaldoAkun{
		{Kode: "2102", Nama: "Hutang Bank", Kategori: finance.KatPinjaman, Nominal: 1},
	}}})
	if got := tn.opening.Rincian[0].Nama; got != "Utang Bank" {
		t.Fatalf("saldo awal tersimpan: %q", got)
	}

	st := katalog.NewState()
	terbit := st.Template[0]
	terbit.ID, terbit.Versi = 7, 2
	terbit.Akun = []coa.Account{{Kode: "2101", Nama: "Hutang Usaha"}}
	st.Template = append(st.Template, terbit)
	if !segarkanBawaan(&st) {
		t.Fatal("katalog berubah harus ditandai untuk disimpan")
	}
	if got := st.Template[len(st.Template)-1].Akun[0].Nama; got != "Utang Usaha" {
		t.Fatalf("template terbitan: %q", got)
	}
	for _, a := range st.Template[0].Akun {
		if finance.EjaUtang(a.Nama) != a.Nama {
			t.Errorf("CoA bawaan masih %q", a.Nama)
		}
	}
}
