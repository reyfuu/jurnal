package finance

import (
	"slices"
	"testing"
)

func TestDataWilayahLengkap(t *testing.T) {
	nKota, nKec := 0, 0
	for _, p := range ProvinsiOptions {
		kota := KotaDi(p)
		if len(kota) == 0 {
			t.Errorf("provinsi %s tanpa kabupaten/kota", p)
		}
		nKota += len(kota)
		for _, k := range kota {
			nKec += len(k.Kecamatan)
		}
	}
	if nKota != 514 || nKec != 7285 {
		t.Fatalf("Kepmendagri 2025: 514 kab/kota & 7.285 kecamatan, dapat %d & %d", nKota, nKec)
	}
	if KotaDi("Atlantis") != nil {
		t.Fatal("provinsi tak dikenal harus nil")
	}
	i := slices.IndexFunc(KotaDi("Jawa Barat"), func(k Kota) bool { return k.Nama == "Kota Bandung" })
	if i < 0 || !slices.Contains(KotaDi("Jawa Barat")[i].Kecamatan, "Sukasari") {
		t.Fatal("Kota Bandung / Sukasari harus ada di Jawa Barat")
	}
}

func TestValidateWilayah(t *testing.T) {
	ok := BusinessProfile{Provinsi: "Jawa Barat", Kota: "Kota Bandung", Kecamatan: "Sukasari"}
	for name, p := range map[string]BusinessProfile{
		"lengkap":    ok,
		"kosong":     {},
		"hanya prov": {Provinsi: "Bali"},
		"tanpa kec":  {Provinsi: "Jawa Barat", Kota: "Kota Bandung"},
	} {
		if err := p.ValidateWilayah(); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, p := range map[string]BusinessProfile{
		"kota provinsi lain":   {Provinsi: "Bali", Kota: "Kota Bandung"},
		"kecamatan kota lain":  {Provinsi: "Jawa Barat", Kota: "Kota Bandung", Kecamatan: "Kebayoran Baru"},
		"kota karangan":        {Provinsi: "Jawa Barat", Kota: "Bandung"},
		"kecamatan tanpa kota": {Provinsi: "Jawa Barat", Kecamatan: "Sukasari"},
	} {
		if p.ValidateWilayah() == nil {
			t.Errorf("%s: harus ditolak", name)
		}
	}
}
