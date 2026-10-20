package tax

import (
	"slices"
	"testing"
)

func kodeKandidat(ks []KandidatKLU) []string {
	out := make([]string, len(ks))
	for i, k := range ks {
		out[i] = k.Kode
	}
	return out
}

func TestContohKLU_DariUraianLimaDigit(t *testing.T) {
	c := ContohKLU("563", 3)
	if len(c) != 3 || c[0] != "Bar" || c[1] != "Rumah minum/kafe" {
		t.Fatalf("contoh 563 = %v", c)
	}
	if len(ContohKLU("999", 3)) != 0 {
		t.Error("KLU tak dikenal punya contoh")
	}
}

func TestCariKLU_BahasaSehariHari(t *testing.T) {
	for _, c := range []struct{ teks, jenis, want string }{
		{"jual kopi susu di kafe kecil", "", "563"},
		{"laundry kiloan", "jasa", "962"},
		{"jualan baju di shopee", "perdagangan", "479"},
		{"catering nasi kotak untuk acara", "", "562"},
		{"bengkel mobil", "jasa", "452"},
		{"barbershop", "", "961"},
		{"saya jual skincare di marketplace", "perdagangan", "479"},
		{"jual kopi susu di kafe kecil", "perdagangan", "563"},
		{"distributor kopi ke kafe-kafe", "perdagangan", "463"},
		{"jual skincare di toko sendiri", "perdagangan", "477"},
		{"bikin kue kering untuk dijual", "manufaktur", "107"},
	} {
		got := kodeKandidat(CariKLU(c.teks, c.jenis, 5))
		if !slices.Contains(got[:min(3, len(got))], c.want) {
			t.Errorf("%q → %v, want %s di 3 teratas", c.teks, got, c.want)
		}
	}
	if got := kodeKandidat(CariKLU("laundry kiloan dekat kampus, bisa antar jemput", "jasa", 5)); len(got) != 1 || got[0] != "962" {
		t.Errorf("kecocokan kebetulan ikut tampil: %v", got)
	}
	for _, x := range kodeKandidat(CariKLU("saya jual skincare di marketplace", "perdagangan", 5)) {
		if x == "182" || x == "268" {
			t.Errorf("jual online tidak boleh jatuh ke reproduksi/industri media: %s", x)
		}
	}
	if got := CariKLU("di dan yang", "", 5); len(got) != 0 {
		t.Errorf("kata umum saja memberi hasil %v", kodeKandidat(got))
	}
}
