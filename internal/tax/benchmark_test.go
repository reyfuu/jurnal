package tax

import (
	"regexp"
	"strings"
	"testing"
)

func levels(r CekAwalResult) map[string]string {
	m := map[string]string{}
	for _, x := range r.Risiko {
		m[x.Judul] = x.Level
	}
	return m
}

// Angka diambil dari docs/CTTOR dan NPM Nasional.pdf (baris "Nasional").
func TestKLUData(t *testing.T) {
	if len(KLUBenchmarks) != 222 {
		t.Fatalf("PDF berisi 222 KLU, data %d", len(KLUBenchmarks))
	}
	seen := map[string]bool{}
	for _, k := range KLUBenchmarks {
		if !regexp.MustCompile(`^\d{3}$`).MatchString(k.Kode) || seen[k.Kode] || k.Nama == "" {
			t.Fatalf("baris tidak valid atau ganda: %+v", k)
		}
		seen[k.Kode] = true
	}
	for kode, want := range map[string][2]int64{"471": {47, 343}, "561": {47, -246}, "089": {62, 376}, "962": {46, 617}} {
		k, ok := FindKLU(kode)
		if !ok || k.CTTORBps != want[0] || k.NPMBps != want[1] {
			t.Errorf("KLU %s: %+v, want CTTOR %d NPM %d bps", kode, k, want[0], want[1])
		}
	}
	for kode, want := range map[string]string{"471": "perdagangan", "107": "manufaktur", "011": "manufaktur", "962": "jasa", "561": "jasa"} {
		if got := JenisKLU(kode); got != want {
			t.Errorf("JenisKLU(%s) = %s, want %s", kode, got, want)
		}
	}
}

func TestCekAwal(t *testing.T) {
	if r := CekAwal(CekAwalInput{BentukUsaha: "op", PTKP: "TK/0", KLU: "471"}); r.Status != "kurang_data" {
		t.Fatalf("tanpa omzet: %+v", r)
	}
	if r := CekAwal(CekAwalInput{BentukUsaha: "op", PTKP: "TK/0", Omzet: 300_000_000, Laba: 30_000_000}); r.Status != "kurang_data" || r.KLU != nil {
		t.Fatalf("tanpa KLU: %+v", r)
	}

	// Toko (KLU 471, NPM nasional 3,43%): NPM 2% di bawah, PPh Final -> CTTOR hanya info.
	r := CekAwal(CekAwalInput{BentukUsaha: "op", PTKP: "K/1", KLU: "471", Omzet: 450_000_000, Laba: 9_000_000})
	if r.NPMBps != 200 || r.Status != "di_bawah" || r.KLU.Kode != "471" {
		t.Fatalf("NPM 2%% toko: %+v", r)
	}
	if l := levels(r); l["NPM di bawah rata-rata nasional"] != "tinggi" || l["Mendekati batas bebas pajak"] != "info" || l["CTTOR di bawah rata-rata nasional"] != "" {
		t.Fatalf("risiko: %+v", r.Risiko)
	}
	if r.CTTORPesan == "" {
		t.Fatal("CTTOR tetap ditampilkan untuk PPh Final")
	}

	if r := CekAwal(CekAwalInput{BentukUsaha: "op", PTKP: "TK/0", KLU: "471", Omzet: 450_000_000, Laba: 18_000_000}); r.Status != "wajar" {
		t.Fatalf("NPM 4%% >= 3,43%%: %+v", r)
	}

	// CV (Pasal 31E, non-final) dengan laba tipis: CTTOR di bawah rata-rata dinilai sebagai risiko.
	r = CekAwal(CekAwalInput{BentukUsaha: "cv", KLU: "620", Omzet: 1_000_000_000, Laba: 10_000_000})
	if l := levels(r); l["CTTOR di bawah rata-rata nasional"] != "sedang" {
		t.Fatalf("CTTOR non-final: %+v %s", r.Risiko, r.CTTORPesan)
	}

	r = CekAwal(CekAwalInput{BentukUsaha: "op", PTKP: "TK/0", KLU: "471", Omzet: 6_000_000_000, Laba: 600_000_000})
	if l := levels(r); l["Wajib menjadi PKP"] != "tinggi" || l["PPh Final 0,5% hanya untuk tahun pertama"] != "sedang" {
		t.Fatalf("omzet 6 M OP belum PKP: %+v", r.Risiko)
	}
	if l := levels(CekAwal(CekAwalInput{BentukUsaha: "cv", KLU: "962", Omzet: 1_000_000_000, Laba: -5_000_000})); l["Usaha dilaporkan rugi"] != "tinggi" {
		t.Fatalf("rugi: %+v", l)
	}
}

// ROI tahun pertama = laba / modal, hanya bila modal diisi.
func TestCekAwal_ROI(t *testing.T) {
	if r := CekAwal(CekAwalInput{BentukUsaha: "op", KLU: "471", Omzet: 450_000_000, Laba: 18_000_000}); r.ROIBps != nil {
		t.Fatalf("tanpa modal tidak ada ROI: %+v", r.ROIBps)
	}
	r := CekAwal(CekAwalInput{BentukUsaha: "op", KLU: "471", Omzet: 450_000_000, Laba: 18_000_000, Modal: 120_000_000})
	if r.ROIBps == nil || *r.ROIBps != 1500 || !strings.Contains(r.ROIPesan, "15,00%") || !strings.Contains(r.ROIPesan, "6 tahun 8 bulan") {
		t.Fatalf("ROI = %v %q", r.ROIBps, r.ROIPesan)
	}
	if r := CekAwal(CekAwalInput{BentukUsaha: "op", Omzet: 100_000_000, Laba: -10_000_000, Modal: 50_000_000}); r.ROIBps == nil || *r.ROIBps != -2000 || strings.Contains(r.ROIPesan, "kembali") {
		t.Fatalf("ROI rugi = %v %q", r.ROIBps, r.ROIPesan)
	}
}

func TestAktifkanBenchmark_DipakaiFindKLU(t *testing.T) {
	t.Cleanup(func() { AktifkanBenchmark(KLUBenchmarks, SumberBenchmark) })
	AktifkanBenchmark([]KLUBenchmark{{Kode: "563", Nama: "Penyediaan Minuman", CTTORBps: 45, NPMBps: 610}}, "Nasional 2024")
	if k, ok := FindKLU("563"); !ok || k.NPMBps != 610 {
		t.Fatalf("FindKLU tidak memakai versi aktif: %+v %v", k, ok)
	}
	if _, ok := FindKLU("471"); ok {
		t.Error("KLU di luar versi aktif masih ditemukan")
	}
	if got := CekAwal(CekAwalInput{KLU: "563", Omzet: 100, Laba: 10}).Sumber; got != "Nasional 2024" {
		t.Errorf("sumber = %q", got)
	}
	if n := len(BenchmarkAktif()); n != 1 {
		t.Errorf("BenchmarkAktif = %d baris", n)
	}
}

// Jual telur (peternakan) beromzet 15 M: PPN telur dibebaskan, jadi bukan "wajib PKP" pasti.
func TestCekAwal_KebutuhanPokokBukanPastiWajibPKP(t *testing.T) {
	l := levels(CekAwal(CekAwalInput{BentukUsaha: "op", PTKP: "TK/0", KLU: "014", Omzet: 15_000_000_000, Laba: 900_000_000}))
	if _, ada := l["Wajib menjadi PKP"]; ada || l["Omzet di atas batas PKP: cek fasilitas PPN dibebaskan"] != "sedang" {
		t.Fatalf("%+v", l)
	}
}

// Laba (penghasilan kena pajak) tidak mungkin melebihi omzet: ditolak dengan pesan berangka.
func TestKalkulator_LabaMelebihiOmzetDitolak(t *testing.T) {
	if _, err := HitungPPhBadan(PPhBadanInput{PeredaranBruto: 250_000_000, PenghasilanKenaPajak: 1_500_000_000}); err == nil || !strings.Contains(err.Error(), "tidak boleh lebih besar") {
		t.Fatalf("PPh badan: %v", err)
	}
	laba := int64(1_500_000_000)
	_, err := HitungMarketplace(MarketplaceInput{OmzetMarketplace: 50_000_000, OmzetLain: 300_000_000, NPMBps: 42_857, Laba: &laba, Bentuk: "badan", Skema: "umum"})
	if err == nil || !strings.Contains(err.Error(), "lebih besar dari total omzet") {
		t.Fatalf("marketplace: %v", err)
	}
}
