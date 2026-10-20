package tax

import (
	"regexp"
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
	if l := levels(r); l["NPM di bawah rata-rata DJP"] != "tinggi" || l["Mendekati batas bebas pajak"] != "info" || l["CTTOR di bawah rata-rata DJP"] != "" {
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
	if l := levels(r); l["CTTOR di bawah rata-rata DJP"] != "sedang" {
		t.Fatalf("CTTOR non-final: %+v %s", r.Risiko, r.CTTORPesan)
	}

	r = CekAwal(CekAwalInput{BentukUsaha: "op", PTKP: "TK/0", KLU: "471", Omzet: 6_000_000_000, Laba: 600_000_000})
	if l := levels(r); l["Wajib menjadi PKP"] != "tinggi" || l["Tidak bisa memakai PPh Final 0,5%"] != "tinggi" {
		t.Fatalf("omzet 6 M OP belum PKP: %+v", r.Risiko)
	}
	if l := levels(CekAwal(CekAwalInput{BentukUsaha: "cv", KLU: "962", Omzet: 1_000_000_000, Laba: -5_000_000})); l["Usaha dilaporkan rugi"] != "tinggi" {
		t.Fatalf("rugi: %+v", l)
	}
}
