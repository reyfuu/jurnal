package tax_test

import (
	"errors"
	"testing"

	"pajak/internal/tax"
)

func TestCekLabaOmzet(t *testing.T) {
	cases := []struct {
		laba, omzet int64
		want        error
	}{
		{100, 1000, nil},
		{1000, 1000, nil},  // margin 100%: batas atas sah
		{-1000, 1000, nil}, // margin -100%: batas bawah sah
		{0, 0, nil},
		{1001, 1000, tax.ErrLabaMelebihiOmzet},
		{-1001, 1000, tax.ErrRugiMelebihiOmzet},
		{1, 0, tax.ErrLabaMelebihiOmzet},
	}
	for _, c := range cases {
		if err := tax.CekLabaOmzet(c.laba, c.omzet); !errors.Is(err, c.want) {
			t.Errorf("laba %d omzet %d: %v, mau %v", c.laba, c.omzet, err, c.want)
		}
	}
}

func TestCekMarginBps(t *testing.T) {
	for _, bps := range []int64{0, 10_000, -10_000, 550} {
		if err := tax.CekMarginBps(bps); err != nil {
			t.Errorf("%d bps sah: %v", bps, err)
		}
	}
	for _, bps := range []int64{10_001, -10_001, 15_000} {
		if err := tax.CekMarginBps(bps); !errors.Is(err, tax.ErrMargin) {
			t.Errorf("%d bps harus ditolak: %v", bps, err)
		}
	}
}
