package finance_test

import (
	"errors"
	"testing"
	"time"

	"pajak/internal/finance"
)

func TestCatatan_RapikanDanPengingat(t *testing.T) {
	c := finance.CatatanKonsultan{Isi: "  Saldo kas tidak cocok  ", Tingkat: "perbaiki", Halaman: "laporan-keuangan"}
	if err := finance.RapikanCatatan(&c); err != nil || c.Isi != "Saldo kas tidak cocok" {
		t.Fatalf("%+v %v", c, err)
	}
	for _, x := range []finance.CatatanKonsultan{{Isi: "", Tingkat: "info"}, {Isi: "x", Tingkat: "darurat"}} {
		if err := finance.RapikanCatatan(&x); !errors.Is(err, finance.ErrCatatan) {
			t.Errorf("%+v diterima", x)
		}
	}

	t0 := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	p := finance.PengingatBawaan() // 3 hari, mulai "perbaiki"
	c = finance.CatatanKonsultan{Tingkat: "perbaiki", Status: finance.CatatanTerbuka, Dibuat: t0}
	if c.PerluDiingatkan(p, t0.Add(47*time.Hour)) || !c.PerluDiingatkan(p, t0.Add(72*time.Hour)) {
		t.Error("jeda 3 hari sejak dibuat tidak dipakai")
	}
	c.DiingatkanPada = t0.Add(72 * time.Hour)
	if c.PerluDiingatkan(p, t0.Add(100*time.Hour)) || !c.PerluDiingatkan(p, t0.Add(144*time.Hour)) {
		t.Error("jeda dihitung ulang sejak terakhir diingatkan")
	}
	info := finance.CatatanKonsultan{Tingkat: "info", Status: finance.CatatanTerbuka, Dibuat: t0}
	if info.PerluDiingatkan(p, t0.Add(30*24*time.Hour)) {
		t.Error("tingkat di bawah minimal ikut diingatkan")
	}
	c.Status = finance.CatatanSelesai
	if c.PerluDiingatkan(p, t0.Add(30*24*time.Hour)) {
		t.Error("catatan selesai diingatkan")
	}
	p.Aktif = false
	if (finance.CatatanKonsultan{Tingkat: "penting", Status: "terbuka", Dibuat: t0}).PerluDiingatkan(p, t0.Add(30*24*time.Hour)) {
		t.Error("pengingat nonaktif tetap mengingatkan")
	}
	bad := finance.Pengingat{IntervalHari: 0, TingkatMin: "info"}
	if bad.Validasi() == nil {
		t.Error("interval 0 diterima")
	}
}
