package tax

import (
	"strings"
	"testing"
)

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
		{2_400_000_000, 168_000_000, 18_480_000},     // seluruh PKP dapat fasilitas 50%
		{10_000_000_000, 1_000_000_000, 167_200_000}, // sampel sheet Pasal 31E
		{60_000_000_000, 1_000_000_000, 220_000_000}, // omzet > 50 M: tanpa fasilitas
		{2_400_000_000, 168_000_999, 18_480_000},     // PKP dibulatkan ke bawah ribuan
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
	if cv := byID["cv_pp"]; cv.PPh != 2_000_000 || cv.Dividen != 0 {
		t.Fatalf("CV dengan PPh Final 0,5%% atas seluruh omzet, tanpa dividen: %+v", cv)
	}
	if len(r) != 6 {
		t.Fatalf("want 6 entities, got %d", len(r))
	}
}

// PP 55/2022 hanya untuk omzet s.d. Rp4,8 M: di atasnya skema PPh Final tidak boleh ditawarkan.
func TestPerbandinganTanpaPPFinalDiAtas4_8M(t *testing.T) {
	r, _ := CompareBentukUsaha(4_800_000_001, 1000, "TK/0")
	for _, s := range r.Skema {
		if s.ID == SkemaPPOP || s.ID == SkemaPPBadan {
			t.Fatalf("skema %s tidak boleh muncul di atas Rp4,8 M", s.ID)
		}
	}
	if r.Termurah == SkemaPPOP || r.Termurah == SkemaPPBadan {
		t.Fatalf("termurah tidak boleh PPh Final: %s", r.Termurah)
	}
	if r, _ := CompareBentukUsaha(batasFasilitas, 1000, "TK/0"); len(r.Skema) != 6 {
		t.Fatalf("tepat Rp4,8 M masih boleh PPh Final: %d skema", len(r.Skema))
	}
	for _, e := range CompareEntitas(4_800_000_001, 9000) {
		if e.ID == "op" || e.ID == "pt_perorangan" || e.ID == "cv_pp" {
			t.Fatalf("entitas %s (PPh Final) tidak boleh muncul di atas Rp4,8 M", e.ID)
		}
	}
}

// Rugi: pajak tarif umum 0, bukan negatif.
func TestPerbandinganRugiTidakNegatif(t *testing.T) {
	r, _ := CompareBentukUsaha(1_000_000_000, -500, "TK/0")
	for _, s := range r.Skema {
		if s.Pajak < 0 {
			t.Fatalf("skema %s negatif: %d", s.ID, s.Pajak)
		}
	}
	for _, e := range CompareEntitas(1_000_000_000, 10_500) {
		if e.PPh < 0 || e.Total < 0 {
			t.Fatalf("entitas %s negatif: %+v", e.ID, e)
		}
	}
}

func TestCekPKP(t *testing.T) {
	for _, c := range []struct {
		omzet     int64
		wajib     bool
		sisa, bps int64
	}{
		{0, false, 4_800_000_000, 0},
		{1_200_000_000, false, 3_600_000_000, 2500},
		{4_800_000_000, false, 0, 10_000}, // tepat 4,8 M masih pengusaha kecil
		{4_800_000_001, true, 0, 10_000},
	} {
		r := CekPKP(c.omzet)
		if r.WajibPKP != c.wajib || r.Sisa != c.sisa || r.PemakaianBps != c.bps || r.Batas != batasFasilitas {
			t.Errorf("CekPKP(%d) = %+v; want wajib=%v sisa=%d bps=%d", c.omzet, r, c.wajib, c.sisa, c.bps)
		}
		if len(r.Catatan) == 0 {
			t.Errorf("CekPKP(%d): catatan kosong", c.omzet)
		}
	}
}

// Pilihan PTKP tidak lagi menawarkan K/I/*, tetapi nilainya tetap ada untuk data lama.
func TestPTKPCodesTanpaKI(t *testing.T) {
	if len(PTKPCodes) != 8 {
		t.Fatalf("want 8 pilihan (TK/0–K/3), got %v", PTKPCodes)
	}
	for _, c := range PTKPCodes {
		if _, ok := PTKP[c]; !ok || strings.HasPrefix(c, "K/I/") {
			t.Fatalf("pilihan %q tidak sah", c)
		}
	}
	if _, err := CompareBentukUsaha(1_000_000_000, 1000, "K/I/0"); err != nil {
		t.Fatalf("profil lama K/I/0 harus tetap terhitung: %v", err)
	}
}
