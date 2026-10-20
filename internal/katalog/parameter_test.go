package katalog

import (
	"errors"
	"testing"

	"pajak/internal/tax"
)

func TestParameter_BawaanAktif(t *testing.T) {
	st := NewState()
	a := st.ParameterAktif()
	if a.ID != 1 || a.Status != Aktif || a.Isi.BadanTarifBps != 2200 || a.Dasar != tax.SumberParameterBawaan {
		t.Fatalf("bawaan = %+v", a)
	}
}

func TestParameter_DraftUbahAktifkanArsip(t *testing.T) {
	st := NewState()
	isi := tax.ParameterBawaan()
	isi.PPNTarifBps = 1100
	d, err := st.SimpanParameter(0, "PPN 2027", "PMK baru", isi, "Rina", t0)
	if err != nil || d.Status != Draft || d.ID != 2 {
		t.Fatalf("draft = %+v %v", d, err)
	}
	isi.UMKMTarifBps = 100
	if d, err = st.SimpanParameter(d.ID, "PPN 2027", "PMK baru", isi, "Rina", t0); err != nil || d.Isi.UMKMTarifBps != 100 || len(st.Parameter) != 2 {
		t.Fatalf("ubah draft = %+v %v (versi %d)", d, err, len(st.Parameter))
	}
	if _, err := st.SimpanParameter(1, "x", "", isi, "Rina", t0); !errors.Is(err, ErrTidakValid) {
		t.Errorf("versi aktif ikut diubah: %v", err)
	}
	if err := st.AktifkanParameter(d.ID, "Budi", t0); err != nil {
		t.Fatal(err)
	}
	if a := st.ParameterAktif(); a.ID != d.ID || a.Isi.PPNTarifBps != 1100 || a.Oleh != "Budi" {
		t.Fatalf("aktif = %+v", a)
	}
	if lama, _ := st.ParameterByID(1); lama.Status != Arsip {
		t.Errorf("versi lama = %s, want arsip", lama.Status)
	}
	// kembali ke versi arsip (rollback) diperbolehkan
	if err := st.AktifkanParameter(1, "Budi", t0); err != nil || st.ParameterAktif().ID != 1 {
		t.Fatalf("rollback: %v", err)
	}
}

func TestParameter_TolakIsiTidakValidDanHapusHanyaDraft(t *testing.T) {
	st := NewState()
	isi := tax.ParameterBawaan()
	isi.BadanTarifBps = 20_000
	if _, err := st.SimpanParameter(0, "Rusak", "", isi, "Rina", t0); !errors.Is(err, tax.ErrParameter) {
		t.Errorf("isi tidak valid diterima: %v", err)
	}
	if _, err := st.SimpanParameter(0, " ", "", tax.ParameterBawaan(), "Rina", t0); !errors.Is(err, ErrTidakValid) {
		t.Errorf("nama kosong diterima: %v", err)
	}
	if err := st.HapusDraftParameter(1); !errors.Is(err, ErrTidakValid) {
		t.Errorf("versi aktif terhapus: %v", err)
	}
	d, _ := st.SimpanParameter(0, "Coba", "", tax.ParameterBawaan(), "Rina", t0)
	if err := st.HapusDraftParameter(d.ID); err != nil || len(st.Parameter) != 1 {
		t.Errorf("hapus draft: %v", err)
	}
}

func TestParameter_SalinanTidakBocor(t *testing.T) {
	st := NewState()
	isi := tax.ParameterBawaan()
	d, _ := st.SimpanParameter(0, "A", "", isi, "Rina", t0)
	isi.PTKP["TK/0"] = 1
	if got, _ := st.ParameterByID(d.ID); got.Isi.PTKP["TK/0"] == 1 {
		t.Fatal("draft berbagi map dengan pemanggil")
	}
	c := st.Clone()
	c.Parameter[0].Status = Arsip
	if st.Parameter[0].Status != Aktif {
		t.Fatal("Clone berbagi slice versi")
	}
}

func TestNorma_ExcelBolakBalik(t *testing.T) {
	ns := tax.ParameterBawaan().Norma[:3]
	rows := BarisNorma(ns)
	if rows[0][0] != "KLU" || len(rows) != 4 {
		t.Fatalf("kepala/jumlah: %v", rows[:1])
	}
	got, err := ParseNorma(rows)
	if err != nil || len(got) != 3 || got[2] != ns[2] {
		t.Fatalf("bolak-balik: %+v %v", got, err)
	}
	if _, err := ParseNorma([][]string{rows[0], {"111", "X", "10", "10", "10"}}); !errors.Is(err, tax.ErrParameter) {
		t.Errorf("KLU 3 digit diterima: %v", err)
	}
	got, _ = ParseNorma([][]string{rows[0], {"1111", "X", "-", "", "12,5"}})
	if got[0].KLU != "01111" || got[0].Ibukota10Bps != -1 || got[0].IbukotaLainBps != -1 || got[0].DaerahLainBps != 1250 {
		t.Errorf("KLU tanpa nol depan / sel kosong: %+v", got[0])
	}
}
