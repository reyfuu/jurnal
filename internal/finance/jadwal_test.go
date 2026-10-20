package finance

import (
	"testing"
	"time"
)

func tgl(s string) time.Time {
	t, _ := time.ParseInLocation("2006-01-02 15:04", s, WIB)
	return t
}

func TestJatuhTempoBulanan(t *testing.T) {
	j := JadwalPengingat{Frekuensi: "bulanan", Tanggal: 15}
	for _, c := range []struct{ now, mau string }{
		{"2026-10-10 09:00", "2026-10-15"},
		{"2026-10-15 23:00", "2026-10-15"}, // hari H masih bulan ini
		{"2026-10-16 00:30", "2026-11-15"},
	} {
		if got := j.JatuhTempo(tgl(c.now)).Format(time.DateOnly); got != c.mau {
			t.Errorf("%s: %s, mau %s", c.now, got, c.mau)
		}
	}
	akhir := JadwalPengingat{Frekuensi: "bulanan", Tanggal: 31}
	if got := akhir.JatuhTempo(tgl("2026-02-10 08:00")).Format(time.DateOnly); got != "2026-02-28" {
		t.Errorf("tanggal 31 di Februari = %s, mau hari terakhir 2026-02-28", got)
	}
}

func TestJatuhTempoTahunan(t *testing.T) {
	j := JadwalPengingat{Frekuensi: "tahunan", Tanggal: 31, Bulan: 3}
	if got := j.JatuhTempo(tgl("2026-03-01 08:00")).Format(time.DateOnly); got != "2026-03-31" {
		t.Errorf("%s", got)
	}
	if got := j.JatuhTempo(tgl("2026-04-01 08:00")).Format(time.DateOnly); got != "2027-03-31" {
		t.Errorf("sesudah lewat pindah ke tahun depan: %s", got)
	}
}

func TestJadwalAktifSesuaiProfilDanJendela(t *testing.T) {
	d := DaftarJadwal{Item: []JadwalPengingat{
		{ID: 1, Judul: "Setor", Frekuensi: "bulanan", Tanggal: 15, HariSebelum: 3, Untuk: UntukSemua, Aktif: true},
		{ID: 2, Judul: "PPN", Frekuensi: "bulanan", Tanggal: 31, HariSebelum: 30, Untuk: UntukPKP, Aktif: true},
		{ID: 3, Judul: "Badan", Frekuensi: "bulanan", Tanggal: 14, HariSebelum: 5, Untuk: UntukBadan, Aktif: true},
		{ID: 4, Judul: "Mati", Frekuensi: "bulanan", Tanggal: 13, HariSebelum: 5, Untuk: UntukSemua, Aktif: false},
		{ID: 5, Judul: "OP", Frekuensi: "bulanan", Tanggal: 15, HariSebelum: 5, Untuk: UntukOP, Aktif: true},
	}}
	now := tgl("2026-10-12 10:00")
	op := &BusinessProfile{BentukUsaha: "op"}
	got := d.Aktif(op, now)
	if len(got) != 2 || got[0].ID != 1 || got[1].ID != 5 || got[0].SisaHari != 3 || got[0].JatuhTempo != "2026-10-15" {
		t.Fatalf("OP non-PKP: %+v", got)
	}
	cv := &BusinessProfile{BentukUsaha: "cv", PKP: true}
	got = d.Aktif(cv, now)
	ids := []int{}
	for _, g := range got {
		ids = append(ids, g.ID)
	}
	// Urut jatuh tempo terdekat: Badan (14), Setor (15), PPN (31).
	if len(ids) != 3 || ids[0] != 3 || ids[1] != 1 || ids[2] != 2 {
		t.Errorf("CV PKP: %v", ids)
	}
	if got := d.Aktif(nil, now); len(got) != 1 || got[0].ID != 1 {
		t.Errorf("tanpa profil hanya untuk semua: %+v", got)
	}
	// H-4: jadwal H-3 (ID 1) belum tampil, jadwal H-5 (ID 5) sudah.
	if got := d.Aktif(op, tgl("2026-10-11 23:00")); len(got) != 1 || got[0].ID != 5 || got[0].SisaHari != 4 {
		t.Errorf("jendela H-n: %+v", got)
	}
}

func TestRapikanJadwal(t *testing.T) {
	d := JadwalBawaan()
	if len(d.Item) == 0 || d.NextID != len(d.Item)+1 {
		t.Fatalf("bawaan: %d item, next %d", len(d.Item), d.NextID)
	}
	for _, j := range d.Item {
		if err := j.Validasi(); err != nil {
			t.Errorf("bawaan %q tidak valid: %v", j.Judul, err)
		}
	}
	lama := d.Item[0]
	lama.Judul = "  Tutup buku  "
	baru := JadwalPengingat{Judul: "Rapat bulanan", Frekuensi: "bulanan", Tanggal: 1, HariSebelum: 0, Untuk: UntukSemua, Aktif: true}
	if err := d.Ganti([]JadwalPengingat{lama, baru}); err != nil {
		t.Fatal(err)
	}
	if len(d.Item) != 2 || d.Item[0].ID != lama.ID || d.Item[0].Judul != "Tutup buku" || d.Item[1].ID == 0 || d.Item[1].ID == lama.ID {
		t.Errorf("ganti: %+v", d.Item)
	}
	salah := []JadwalPengingat{
		{Judul: "", Frekuensi: "bulanan", Tanggal: 1, Untuk: UntukSemua},
		{Judul: "x", Frekuensi: "mingguan", Tanggal: 1, Untuk: UntukSemua},
		{Judul: "x", Frekuensi: "bulanan", Tanggal: 0, Untuk: UntukSemua},
		{Judul: "x", Frekuensi: "tahunan", Tanggal: 1, Bulan: 13, Untuk: UntukSemua},
		{Judul: "x", Frekuensi: "tahunan", Tanggal: 30, Bulan: 2, Untuk: UntukSemua},
		{Judul: "x", Frekuensi: "bulanan", Tanggal: 1, HariSebelum: 61, Untuk: UntukSemua},
		{Judul: "x", Frekuensi: "bulanan", Tanggal: 1, Untuk: "pt"},
	}
	for _, s := range salah {
		if err := s.Validasi(); err == nil {
			t.Errorf("harus ditolak: %+v", s)
		}
	}
	sebelum := len(d.Item)
	if err := d.Ganti([]JadwalPengingat{salah[0]}); err == nil || len(d.Item) != sebelum {
		t.Error("ganti dengan item tidak valid harus gagal tanpa mengubah daftar")
	}
	banyak := make([]JadwalPengingat, MaksJadwal+1)
	for i := range banyak {
		banyak[i] = baru
	}
	if err := d.Ganti(banyak); err == nil {
		t.Error("lebih dari MaksJadwal harus ditolak")
	}
}
