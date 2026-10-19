package finance

import (
	"strings"
	"testing"
)

func TestBusinessProfileValidate(t *testing.T) {
	ok := BusinessProfile{NamaUsaha: " Kedai Kopi Senja ", JenisUsaha: "jasa", BentukUsaha: "op", PTKP: "K/1"}
	if err := ok.Validate(); err != nil || ok.NamaUsaha != "Kedai Kopi Senja" {
		t.Fatalf("valid profile rejected: %v %+v", err, ok)
	}
	pt := BusinessProfile{NamaUsaha: "PT Maju", JenisUsaha: "manufaktur", BentukUsaha: "pt", PTKP: "K/1"}
	if err := pt.Validate(); err != nil || pt.PTKP != "" {
		t.Fatalf("badan tidak memakai PTKP, harus dikosongkan: %v %+v", err, pt)
	}
	for _, bad := range []BusinessProfile{
		{NamaUsaha: "", JenisUsaha: "jasa", BentukUsaha: "cv"},
		{NamaUsaha: "A", JenisUsaha: "tambang", BentukUsaha: "cv"},
		{NamaUsaha: "A", JenisUsaha: "jasa", BentukUsaha: "koperasi"},
		{NamaUsaha: "A", JenisUsaha: "jasa", BentukUsaha: "op", PTKP: ""},
	} {
		if err := bad.Validate(); err == nil {
			t.Errorf("want error for %+v", bad)
		}
	}
}

func TestBusinessProfileMetodePersediaan(t *testing.T) {
	p := BusinessProfile{NamaUsaha: "Toko", JenisUsaha: "perdagangan", BentukUsaha: "cv"}
	if err := p.Validate(); err != nil || p.MetodePersediaan != "periodik" {
		t.Fatalf("perdagangan tanpa metode → default periodik: %v %+v", err, p)
	}
	j := BusinessProfile{NamaUsaha: "Salon", JenisUsaha: "jasa", BentukUsaha: "cv", MetodePersediaan: "perpetual"}
	if err := j.Validate(); err != nil || j.MetodePersediaan != "" {
		t.Fatalf("jasa tidak memakai metode persediaan: %v %+v", err, j)
	}
	bad := BusinessProfile{NamaUsaha: "Toko", JenisUsaha: "perdagangan", BentukUsaha: "cv", MetodePersediaan: "fifo"}
	if err := bad.Validate(); err == nil {
		t.Fatal("metode tidak dikenal harus ditolak")
	}
}

func TestLokasiUsaha(t *testing.T) {
	p := BusinessProfile{NamaUsaha: "A", JenisUsaha: "jasa", BentukUsaha: "cv", Provinsi: "Atlantis"}
	if p.Validate() == nil {
		t.Fatal("provinsi di luar daftar harus ditolak")
	}
	p.Provinsi = "Jawa Barat"
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	p.Alamat, p.Kecamatan, p.Kota = "Jl. Merdeka 1", "Sukajadi", "Kota Bandung"
	if got := kotaTanggal(p, "2026-08-20"); !strings.HasPrefix(got, "Kota Bandung, ") {
		t.Fatalf("kotaTanggal harus memakai Kota: %q", got)
	}
	if got := identitasPenerbit(p)[0]; got != "Jl. Merdeka 1, Sukajadi, Kota Bandung, Jawa Barat" {
		t.Fatalf("identitas penerbit: %q", got)
	}
}
