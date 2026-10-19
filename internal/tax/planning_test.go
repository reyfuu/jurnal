package tax

import "testing"

func TestPasal17(t *testing.T) {
	for _, c := range []struct{ pkp, want int64 }{
		{0, 0}, {-5, 0},
		{60_000_000, 3_000_000},
		{96_000_000, 8_400_000},
		{250_000_000, 31_500_000},
		{500_000_000, 94_000_000},
		{1_128_000_000, 282_400_000},
		{5_000_000_000, 1_444_000_000},
		{6_000_000_000, 1_794_000_000},
	} {
		if got := Pasal17(c.pkp); got != c.want {
			t.Errorf("Pasal17(%d) = %d, want %d", c.pkp, got, c.want)
		}
	}
}

func TestPasal31E(t *testing.T) {
	for _, c := range []struct{ omzet, pkp, want int64 }{
		{2_400_000_000, 168_000_000, 18_480_000},    // seluruh PKP dapat fasilitas 50%
		{10_000_000_000, 1_000_000_000, 167_200_000}, // sampel sheet Pasal 31E
		{60_000_000_000, 1_000_000_000, 220_000_000}, // omzet > 50 M: tanpa fasilitas
		{2_400_000_000, 168_000_999, 18_480_000},    // PKP dibulatkan ke bawah ribuan
	} {
		if got := Pasal31E(c.omzet, c.pkp); got != c.want {
			t.Errorf("Pasal31E(%d, %d) = %d, want %d", c.omzet, c.pkp, got, c.want)
		}
	}
}

func TestCompareBentukUsaha_TemplateSample(t *testing.T) {
	r, err := CompareBentukUsaha(2_400_000_000, 700, "K/3")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int64{
		SkemaPPOP: 9_500_000, SkemaPPBadan: 12_000_000, SkemaPasal17OP: 8_400_000,
		SkemaNPPN: 282_400_000, SkemaBadan22: 36_960_000, SkemaPasal31E: 18_480_000,
	}
	if len(r.Skema) != 6 {
		t.Fatalf("want 6 schemes, got %d", len(r.Skema))
	}
	for _, s := range r.Skema {
		if s.Pajak != want[s.ID] {
			t.Errorf("%s = %d, want %d", s.ID, s.Pajak, want[s.ID])
		}
	}
	if r.Termurah != SkemaPasal17OP || r.Laba != 168_000_000 || r.PTKP != 72_000_000 {
		t.Fatalf("got termurah %s laba %d ptkp %d", r.Termurah, r.Laba, r.PTKP)
	}
	// selisih PP Badan vs termurah: (12 - 8,4) / 12 = 30%
	for _, s := range r.Skema {
		if s.ID == SkemaPPBadan && s.LebihMahalBps != 3000 {
			t.Errorf("selisih PP Badan = %d bps, want 3000", s.LebihMahalBps)
		}
	}
}

func TestCompareBentukUsaha_UnknownPTKP(t *testing.T) {
	if _, err := CompareBentukUsaha(1, 1, "X/9"); err == nil {
		t.Fatal("want error for unknown PTKP")
	}
}

func TestCompareEntitas(t *testing.T) {
	// omzet 400 jt < 500 jt: PP OP harus 0, bukan negatif seperti di template
	r := CompareEntitas(400_000_000, 9_800)
	byID := map[string]Entitas{}
	for _, e := range r {
		byID[e.ID] = e
	}
	if op := byID["op"]; op.PPh != 0 || op.Total != 0 {
		t.Fatalf("OP below 500 jt should be 0, got %+v", op)
	}
	pt := byID["pt_perorangan"] // PP Badan 2 jt; laba 8 jt; dividen 10% x (8 - 2) jt
	if pt.PPh != 2_000_000 || pt.Dividen != 600_000 || pt.Total != 2_600_000 {
		t.Fatalf("PT Perorangan %+v", pt)
	}
	if cv := byID["cv"]; cv.Dividen != 0 {
		t.Fatalf("CV has no dividend tax, got %+v", cv)
	}
	if len(r) != 5 {
		t.Fatalf("want 5 entities, got %d", len(r))
	}
}
