package finance_test

import (
	"testing"

	"pajak/internal/finance"
)

func TestTerbilang(t *testing.T) {
	for _, c := range []struct{ n int64; want string }{
		{0, "nol rupiah"},
		{1, "satu rupiah"},
		{10, "sepuluh rupiah"},
		{11, "sebelas rupiah"},
		{15, "lima belas rupiah"},
		{20, "dua puluh rupiah"},
		{21, "dua puluh satu rupiah"},
		{100, "seratus rupiah"},
		{101, "seratus satu rupiah"},
		{250, "dua ratus lima puluh rupiah"},
		{1_000, "seribu rupiah"},
		{1_500, "seribu lima ratus rupiah"},
		{2_000, "dua ribu rupiah"},
		{75_000, "tujuh puluh lima ribu rupiah"},
		{1_000_000, "satu juta rupiah"},
		{5_400_000, "lima juta empat ratus ribu rupiah"},
		{5_994_000, "lima juta sembilan ratus sembilan puluh empat ribu rupiah"},
		{316_775_000, "tiga ratus enam belas juta tujuh ratus tujuh puluh lima ribu rupiah"},
		{1_000_000_000, "satu miliar rupiah"},
		{-2_500, "minus dua ribu lima ratus rupiah"},
	} {
		if got := finance.Terbilang(c.n); got != c.want {
			t.Errorf("Terbilang(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}
