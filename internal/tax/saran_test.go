package tax

import (
	"strings"
	"testing"
)

func TestSaranBentukUsaha_SampelWorkbook(t *testing.T) {
	// Sheet "Bentuk Usaha": omzet 2,4 M, NPM 7%, K/3 → PPh Pasal 17 OP 8,4 jt paling hemat;
	// skema sekarang (OP, PPh final) 9,5 jt.
	s := SaranBentukUsaha("op", "K/3", 2_400_000_000, 168_000_000)
	if s == nil || s.Termurah != SkemaPasal17OP || s.Sekarang != SkemaPPOP || s.Hemat != 1_100_000 {
		t.Fatalf("saran = %+v", s)
	}
	if !strings.Contains(s.Judul, "PPh Pasal 17") || len(s.Alasan) < 2 || len(s.Skema) != 6 {
		t.Fatalf("narasi: %q %v", s.Judul, s.Alasan)
	}
	gabung := strings.Join(s.Alasan, " ")
	// 1,1 jt / 9,5 jt = 11,57% lebih kecil
	if s.HematBps != 1157 || s.Omzet != 2_400_000_000 {
		t.Errorf("hemat %d bps, omzet %d", s.HematBps, s.Omzet)
	}
	for _, kata := range []string{"Rp8.400.000", "Rp9.500.000", "Rp1.100.000", "11,57% lebih kecil", "pembukuan"} {
		if !strings.Contains(gabung, kata) {
			t.Errorf("alasan tidak menyebut %s: %s", kata, gabung)
		}
	}
}

func TestSaranBentukUsaha_SudahPalingHemat(t *testing.T) {
	// omzet 400 jt OP: di bawah Rp500 jt, PPh final 0 → pertahankan
	s := SaranBentukUsaha("op", "TK/0", 400_000_000, 40_000_000)
	if s.Termurah != SkemaPPOP || s.Sekarang != SkemaPPOP || s.Hemat != 0 || !strings.Contains(s.Judul, "sudah") {
		t.Fatalf("saran = %+v", s)
	}
	if !strings.Contains(strings.Join(s.Alasan, " "), "Rp500.000.000") {
		t.Errorf("alasan: %v", s.Alasan)
	}
	if !strings.Contains(s.Asumsi, "tahun pertama") || strings.Contains(s.Asumsi, "Rp4.800.000.000") {
		t.Errorf("asumsi: %q", s.Asumsi)
	}
	// dasar hukum berhenti di "jo. PP 20/2026", tanpa rincian 7/4/3 tahun
	if g := strings.Join(s.Alasan, " "); !strings.Contains(g, "batas waktu pemakaian (PP 55/2022 jo. PP 20/2026).") || strings.Contains(g, "7 tahun") {
		t.Errorf("dasar PPh final: %s", g)
	}
}

func TestSaranBentukUsaha_BadanDanTanpaOmzet(t *testing.T) {
	if s := SaranBentukUsaha("pt", "", 0, 0); s != nil {
		t.Fatal("tanpa omzet tidak ada saran")
	}
	// PT beromzet 6 M: Pasal 31E; badan memakai PTKP TK/0 untuk skema OP. PPh final tetap
	// dibandingkan sebagai tahun pertama melewati batas omzet.
	s := SaranBentukUsaha("pt", "", 6_000_000_000, 300_000_000)
	if s.Sekarang != SkemaPasal31E || s.PTKP != 54_000_000 {
		t.Fatalf("saran = %+v", s)
	}
	for _, k := range s.Skema {
		if (k.ID == SkemaPPOP || k.ID == SkemaPPBadan) && !strings.Contains(k.Nama, "tahun pertama") {
			t.Fatalf("PPh final di atas batas tanpa tanda tahun pertama: %q", k.Nama)
		}
	}
	if !strings.Contains(s.Asumsi, "tahun pertama") || !strings.Contains(s.Asumsi, "Rp4.800.000.000") {
		t.Errorf("asumsi omzet di atas batas: %q", s.Asumsi)
	}
	if !strings.Contains(strings.Join(s.Catatan, " "), "TK/0") {
		t.Errorf("asumsi PTKP badan tidak disebut: %v", s.Catatan)
	}
}

func TestCekAwal_MemuatSaranTanpaKLU(t *testing.T) {
	r := CekAwal(CekAwalInput{BentukUsaha: "op", PTKP: "K/3", Omzet: 2_400_000_000, Laba: 168_000_000})
	if r.Status != "kurang_data" || r.Saran == nil || r.Saran.Termurah != SkemaPasal17OP {
		t.Fatalf("saran harus ada walau KLU belum dipilih: %+v", r.Saran)
	}
	for _, x := range r.Risiko {
		if x.Judul == "Bentuk usaha belum paling hemat" {
			t.Error("risiko bentuk usaha kini digantikan saran")
		}
	}
}
