package coa

import (
	"testing"

	"pajak/internal/finance"
)

func profile(bentuk, metode string) finance.BusinessProfile {
	return finance.BusinessProfile{NamaUsaha: "Toko", JenisUsaha: "perdagangan", BentukUsaha: bentuk, MetodePersediaan: metode}
}

func codes(accs []Account) map[string]Account {
	m := map[string]Account{}
	for _, a := range accs {
		m[a.Kode] = a
	}
	return m
}

func TestMasterFromFile(t *testing.T) {
	if len(Master) != 72 {
		t.Fatalf("Daftar COA.xlsx berisi 72 akun, got %d", len(Master))
	}
	m := codes(Master)
	if len(m) != len(Master) {
		t.Fatal("kode akun harus unik")
	}
	if _, ok := m["2113"]; ok {
		t.Fatal("2113 Utang Karyawan diganti 1407 Piutang Karyawan")
	}
	if a := m["1407"]; a.Normal != Debit || a.Kelompok != AsetLancar {
		t.Fatalf("1407 Piutang Karyawan harus aset lancar debit, got %+v", a)
	}
	kontra := map[string]bool{"1502": true, "1504": true, "1506": true, "4103": true, "4104": true, "5105": true, "3102": true}
	for _, a := range Master {
		want := a.Kelompok.Normal()
		if kontra[a.Kode] {
			want = opposite(want)
		}
		if a.Normal != want {
			t.Errorf("%s %s saldo normal %s, want %s", a.Kode, a.Nama, a.Normal, want)
		}
	}
}

func TestActive(t *testing.T) {
	pt := codes(Active(profile("pt", "periodik")))
	cv := codes(Active(profile("cv", "perpetual")))
	if _, ok := pt["2115"]; !ok {
		t.Error("PT harus punya 2115 Utang Dividen")
	}
	if _, ok := pt["3102"]; ok {
		t.Error("PT tidak memakai 3102 Prive")
	}
	if _, ok := cv["2115"]; ok {
		t.Error("CV tidak punya utang dividen")
	}
	if _, ok := cv["3102"]; !ok {
		t.Error("CV memakai 3102 Prive")
	}
	if _, ok := pt["5102"]; !ok || pt["5101"].Kode != "" {
		t.Error("periodik: 5102 aktif, 5101 tidak")
	}
	if _, ok := cv["5101"]; !ok || cv["5102"].Kode != "" {
		t.Error("perpetual: 5101 aktif, 5102 tidak")
	}
	if !pt["1503"].Opsional {
		t.Error("Gedung (jika ada) ditandai opsional")
	}
	jasa := finance.BusinessProfile{JenisUsaha: "jasa", BentukUsaha: "op"}
	if Active(jasa) != nil {
		t.Error("CoA jasa belum tersedia")
	}
}
