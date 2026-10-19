package finance

import "testing"

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
