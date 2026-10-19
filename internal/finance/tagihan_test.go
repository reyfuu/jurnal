package finance_test

import (
	"testing"
	"time"

	"pajak/internal/finance"
)

func acuan(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestValidateTagihan_MenormalkanJenis(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"", finance.TagihanPiutang}, {"Piutang", finance.TagihanPiutang},
		{"utang", finance.TagihanUtang}, {"Hutang", finance.TagihanUtang},
		{"PAYABLE", finance.TagihanUtang}, {"RECEIVABLE", finance.TagihanPiutang},
	} {
		tg := finance.Tagihan{Jenis: c.in, Pihak: "A", JatuhTempo: "2026-09-01", Nominal: 1000}
		if err := finance.ValidateTagihan(&tg); err != nil {
			t.Errorf("%q ditolak: %v", c.in, err)
			continue
		}
		if tg.Jenis != c.want {
			t.Errorf("jenis %q jadi %q, want %q", c.in, tg.Jenis, c.want)
		}
	}
	tg := finance.Tagihan{Jenis: "entah", Pihak: "A", JatuhTempo: "2026-09-01", Nominal: 1000}
	if err := finance.ValidateTagihan(&tg); err == nil {
		t.Error("jenis ngawur harus ditolak")
	}
}

// Hari terlambat dihitung saat dibaca, bukan disimpan — kalau disimpan, angkanya basi
// begitu tanggal berganti.
func TestTagihan_HariTerlambatDihitung(t *testing.T) {
	tg := finance.Tagihan{JatuhTempo: "2026-07-10", Nominal: 1000}
	if got := tg.HariTerlambat(acuan(t, "2026-07-01")); got != 0 {
		t.Errorf("belum jatuh tempo harus 0, got %d", got)
	}
	if got := tg.HariTerlambat(acuan(t, "2026-08-10")); got != 31 {
		t.Errorf("telat = %d, want 31", got)
	}
	tg.Lunas = true
	if got := tg.HariTerlambat(acuan(t, "2026-08-10")); got != 0 {
		t.Errorf("yang sudah lunas tidak terlambat, got %d", got)
	}
}

// Daftar Piutang & Utang hanya menampilkan yang belum selesai.
func TestToObligations_LunasTidakIkut(t *testing.T) {
	list := []finance.Tagihan{
		{Nomor: "INV-1", Pihak: "A", Jenis: finance.TagihanPiutang, JatuhTempo: "2026-07-10", Nominal: 5_000_000},
		{Nomor: "INV-2", Pihak: "B", Jenis: finance.TagihanPiutang, JatuhTempo: "2026-07-20", Nominal: 2_000_000, Lunas: true},
	}
	out := finance.ToObligations(list, acuan(t, "2026-08-01"))
	if len(out) != 1 {
		t.Fatalf("want 1 baris, got %d", len(out))
	}
	if out[0].DueDate != "10 Jul 2026" {
		t.Errorf("tanggal tampilan = %q, want 10 Jul 2026", out[0].DueDate)
	}
	if out[0].DaysOverdue != 22 {
		t.Errorf("telat = %d, want 22", out[0].DaysOverdue)
	}
}

func TestStokItem_StatusDariStokMinimum(t *testing.T) {
	for _, c := range []struct {
		jumlah, min int64
		want        string
	}{
		{0, 10, "Habis"}, {5, 10, "Stok Menipis"}, {10, 10, "Stok Menipis"},
		{11, 10, "Aman"}, {5, 0, "Aman"}, // tanpa batas minimum tidak pernah "menipis"
	} {
		s := finance.StokItem{Jumlah: c.jumlah, StokMin: c.min}
		if got := s.Status(); got != c.want {
			t.Errorf("jumlah %d min %d = %q, want %q", c.jumlah, c.min, got, c.want)
		}
	}
}

func TestSummarizeStok_NilaiPersediaan(t *testing.T) {
	s := finance.SummarizeStok([]finance.StokItem{
		{Nama: "A", Jumlah: 10, StokMin: 15, HargaPokok: 100_000},
		{Nama: "B", Jumlah: 0, StokMin: 5, HargaPokok: 50_000},
		{Nama: "C", Jumlah: 20, HargaPokok: 25_000},
	})
	if s.NilaiPersediaan != 10*100_000+0+20*25_000 {
		t.Errorf("nilai persediaan = %d", s.NilaiPersediaan)
	}
	if s.Menipis != 1 || s.Habis != 1 {
		t.Errorf("menipis %d habis %d, want 1 dan 1", s.Menipis, s.Habis)
	}
}

func TestSummarizeTagihan_PisahPiutangUtang(t *testing.T) {
	s := finance.SummarizeTagihan([]finance.Tagihan{
		{Pihak: "A", Jenis: finance.TagihanPiutang, JatuhTempo: "2026-07-10", Nominal: 5_000_000},
		{Pihak: "B", Jenis: finance.TagihanPiutang, JatuhTempo: "2026-12-10", Nominal: 2_000_000},
		{Pihak: "C", Jenis: finance.TagihanUtang, JatuhTempo: "2026-09-08", Nominal: 4_800_000},
		{Pihak: "D", Jenis: finance.TagihanPiutang, JatuhTempo: "2026-01-01", Nominal: 9_000_000, Lunas: true},
	}, acuan(t, "2026-08-01"))

	if s.TotalPiutang != 7_000_000 {
		t.Errorf("total piutang = %d, want 7.000.000 (yang lunas tidak dihitung)", s.TotalPiutang)
	}
	if s.TotalUtang != 4_800_000 {
		t.Errorf("total utang = %d", s.TotalUtang)
	}
	if s.PiutangTerlambat != 5_000_000 || s.JumlahTerlambat != 1 {
		t.Errorf("terlambat %d / %d faktur", s.PiutangTerlambat, s.JumlahTerlambat)
	}
}
