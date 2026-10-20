package finance

import (
	"strings"
	"testing"
)

func idsDari(p []PilihanPeriode) string {
	var s []string
	for _, x := range p {
		s = append(s, x.ID)
	}
	return strings.Join(s, ",")
}

func TestDaftarPeriode(t *testing.T) {
	txs := []Transaction{{Tanggal: "2026-08-03"}, {Tanggal: "2026-10-02"}}
	p := DaftarPeriode(txs, nil, "2026-10")
	if idsDari(p) != "2026-10,2026-09,2026-08" || p[0].Label != "Oktober 2026" || !p[0].Bawaan {
		t.Fatalf("%+v", p)
	}
	// Bulan berjalan tanpa transaksi tetap ada; bawaan = bulan terakhir yang punya transaksi.
	p = DaftarPeriode([]Transaction{{Tanggal: "2026-08-03"}}, nil, "2026-10")
	if idsDari(p) != "2026-10,2026-09,2026-08" || p[0].Bawaan || !p[2].Bawaan {
		t.Errorf("bawaan %+v", p)
	}
	// Ekstra (mis. periode neraca awal / titik nol) ikut memperpanjang ke belakang.
	if p := DaftarPeriode(nil, []string{"2026-07"}, "2026-08"); idsDari(p) != "2026-08,2026-07" || !p[0].Bawaan {
		t.Errorf("ekstra %+v", p)
	}
	// Usaha baru tanpa data: bulan berjalan saja.
	if p := DaftarPeriode(nil, nil, "2026-10"); idsDari(p) != "2026-10" || !p[0].Bawaan {
		t.Errorf("kosong %+v", p)
	}
	// Maksimal 36 bulan ke belakang.
	if p := DaftarPeriode([]Transaction{{Tanggal: "2020-01-01"}}, nil, "2026-10"); len(p) != 36 || p[35].ID != "2023-11" {
		t.Errorf("batas %d %s", len(p), p[len(p)-1].ID)
	}
}

func TestTrendHistoryMengikutiData(t *testing.T) {
	txs := []Transaction{
		{Tanggal: "2026-09-05", Jenis: "Penjualan", Nominal: 100, Lunas: true},
		{Tanggal: "2026-10-05", Jenis: "Penjualan", Nominal: 200, Lunas: true},
	}
	h := TrendHistory(txs, nil, "2026-10", func(string) ReportOpening { return ReportOpening{} }, nil)
	if len(h) != 2 || h[1].Omzet != 200 {
		t.Errorf("tren Oktober harus memuat Sep & Okt: %+v", h)
	}
}

func TestEmptySnapshotMemakaiPeriodeDiminta(t *testing.T) {
	s := EmptySnapshot("2026-10")
	if s.Period != "2026-10" || s.PeriodLabel != "Oktober 2026" {
		t.Errorf("%s %s", s.Period, s.PeriodLabel)
	}
}
