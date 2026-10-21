package katalog

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"pajak/internal/coa"
	"pajak/internal/tax"
)

var t0 = time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)

func asuransi() coa.Account {
	return coa.Account{Kode: "6119", Nama: "Beban Asuransi", Subtype: "opex", Normal: coa.Debit, Kelompok: coa.BebanOperasional, Kondisi: coa.Selalu}
}

func TestNewState_BawaanTerbitDanAktif(t *testing.T) {
	st := NewState()
	tm, ok := st.TemplateAktif("perdagangan")
	if !ok || tm.Versi != 1 || !slices.Equal(tm.Akun, coa.Master) {
		t.Fatalf("template perdagangan bawaan = %+v %v", tm.Versi, ok)
	}
	if _, ok := st.TemplateAktif("jasa"); ok {
		t.Error("jasa belum punya template")
	}
	if b := st.BenchmarkAktif(); len(b.Baris) != len(tax.KLUBenchmarks) || b.Sumber != tax.SumberBenchmark {
		t.Errorf("benchmark bawaan = %d baris %q", len(b.Baris), b.Sumber)
	}
}

func TestTemplate_DraftTerbitArsip(t *testing.T) {
	st := NewState()
	baru := append(slices.Clone(coa.Master), asuransi())
	d, err := st.SimpanDraft("perdagangan", baru, "Tambah asuransi", "Andi", t0)
	if err != nil || d.Status != Draft || d.Versi != 0 {
		t.Fatalf("draft = %+v %v", d, err)
	}
	// Draft kedua untuk jenis yang sama menggantikan draft pertama.
	d2, _ := st.SimpanDraft("perdagangan", baru, "Tambah asuransi (revisi)", "Andi", t0)
	if d2.ID != d.ID || len(st.Template) != 3 { // dagang + manufaktur bawaan + 1 draft
		t.Fatalf("draft ganda: %d template", len(st.Template))
	}
	terbit, lama, err := st.Terbitkan(d.ID, "Andi", t0)
	if err != nil || terbit.Versi != 2 || terbit.Status != Terbit || lama == nil || lama.Versi != 1 {
		t.Fatalf("terbit = %+v lama %+v err %v", terbit, lama, err)
	}
	if aktif, _ := st.TemplateAktif("perdagangan"); aktif.ID != terbit.ID {
		t.Error("versi aktif bukan yang baru terbit")
	}
	if v1, _ := st.TemplateByID(lama.ID); v1.Status != Arsip {
		t.Errorf("versi lama harus arsip, dapat %s", v1.Status)
	}
	if _, _, err := st.Terbitkan(terbit.ID, "Andi", t0); err == nil {
		t.Error("menerbitkan ulang versi terbit harus ditolak")
	}
}

func TestTemplate_TerbitDitolakBilaTidakValid(t *testing.T) {
	st := NewState()
	tanpaKas := slices.DeleteFunc(slices.Clone(coa.Master), func(a coa.Account) bool { return a.Kode == "1101" })
	d, _ := st.SimpanDraft("jasa", tanpaKas, "", "Sari", t0)
	if _, _, err := st.Terbitkan(d.ID, "Sari", t0); !errors.Is(err, coa.ErrTemplate) {
		t.Fatalf("terbit tanpa kas: err = %v", err)
	}
	if _, ok := st.TemplateAktif("jasa"); ok {
		t.Error("draft tidak valid ikut aktif")
	}
	if _, err := st.SimpanDraft("kuliner", coa.Master, "", "Sari", t0); !errors.Is(err, ErrTidakValid) {
		t.Errorf("jenis usaha tak dikenal: err = %v", err)
	}
}

func TestBandingkan(t *testing.T) {
	baru := slices.Clone(coa.Master)
	baru[0].Nama = "Kas Toko"
	baru = append(baru, asuransi())
	baru = slices.DeleteFunc(baru, func(a coa.Account) bool { return a.Kode == "1602" || a.Kode == "6116" })
	d := Bandingkan(coa.Master, baru)
	if len(d.Tambah) != 1 || d.Tambah[0].Kode != "6119" || len(d.Ubah) != 1 || d.Ubah[0].Kode != "1101" || len(d.Hapus) != 1 || d.Hapus[0].Kode != "6116" {
		t.Errorf("diff = %+v", d)
	}
}

func TestBenchmark_SimpanAktifkan(t *testing.T) {
	st := NewState()
	baris := []tax.KLUBenchmark{{Kode: "563", Nama: "Penyediaan Minuman", CTTORBps: 45, NPMBps: 610}, {Kode: "471", Nama: "Eceran", CTTORBps: 47, NPMBps: 343}}
	b, err := st.SimpanBenchmark("Nasional 2024", "Benchmark nasional 2024", baris, "Andi", t0)
	if err != nil || b.Status != Draft {
		t.Fatalf("simpan = %+v %v", b, err)
	}
	// KLU yang dipakai klien wajib tetap ada di versi baru.
	err = st.AktifkanBenchmark(b.ID, []string{"563", "962"})
	var hilang ErrKLUHilang
	if !errors.As(err, &hilang) || !slices.Equal(hilang.Kode, []string{"962"}) {
		t.Fatalf("aktifkan dengan KLU hilang: err = %v", err)
	}
	if err := st.AktifkanBenchmark(b.ID, []string{"563", "471"}); err != nil {
		t.Fatal(err)
	}
	if a := st.BenchmarkAktif(); a.ID != b.ID {
		t.Error("versi aktif tidak berganti")
	}
	if lama, _ := st.BenchmarkByID(1); lama.Status != Arsip {
		t.Errorf("versi lama = %s", lama.Status)
	}
	dobel := append(slices.Clone(baris), baris[0])
	if _, err := st.SimpanBenchmark("X", "", dobel, "Andi", t0); !errors.Is(err, ErrTidakValid) {
		t.Errorf("KLU dobel: err = %v", err)
	}
}

func TestParseBenchmark(t *testing.T) {
	rows := [][]string{
		{"KLU", "Nama", "NPM (%)", "CTTOR (%)"},
		{"563", "Penyediaan Minuman", "-5,02", "0,39"},
		{"471", "Perdagangan Eceran", "3.43", "0.47"},
		{"", "", "", ""},
	}
	got, err := ParseBenchmark(rows)
	if err != nil || len(got) != 2 || got[0].NPMBps != -502 || got[0].CTTORBps != 39 || got[1].NPMBps != 343 {
		t.Fatalf("parse = %+v %v", got, err)
	}
	if _, err := ParseBenchmark([][]string{{"KLU", "Nama", "NPM (%)", "CTTOR (%)"}, {"56", "X", "1", "1"}}); err == nil || !strings.Contains(err.Error(), "baris 2") {
		t.Errorf("KLU 2 digit: err = %v", err)
	}
	// Bolak-balik ekspor → impor menghasilkan data yang sama.
	back, err := ParseBenchmark(BarisBenchmark(tax.KLUBenchmarks))
	if err != nil || !slices.Equal(back, tax.KLUBenchmarks) {
		t.Fatalf("ekspor-impor benchmark tidak sama (%v)", err)
	}
}

func TestParseTemplate_BolakBalik(t *testing.T) {
	back, err := ParseTemplate(BarisTemplate(coa.Master))
	if err != nil || !slices.Equal(back, coa.Master) {
		t.Fatalf("ekspor-impor template tidak sama (%v)", err)
	}
	if _, err := ParseTemplate([][]string{BarisTemplate(nil)[0], {"6119", "Asuransi", "opex", "kiri", "Beban Operasional", "selalu", ""}}); err == nil {
		t.Error("saldo normal 'kiri' harus ditolak")
	}
}
