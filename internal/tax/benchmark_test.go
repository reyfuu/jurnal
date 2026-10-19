package tax

import "testing"

func levels(r CekAwalResult) map[string]string {
	m := map[string]string{}
	for _, x := range r.Risiko {
		m[x.Judul] = x.Level
	}
	return m
}

func TestCekAwal(t *testing.T) {
	if r := CekAwal(CekAwalInput{JenisUsaha: "jasa", BentukUsaha: "op", PTKP: "TK/0"}); r.Status != "kurang_data" {
		t.Fatalf("tanpa omzet: %+v", r)
	}

	r := CekAwal(CekAwalInput{JenisUsaha: "perdagangan", BentukUsaha: "op", PTKP: "K/1", Omzet: 450_000_000, Laba: 9_000_000})
	if r.NPMBps != 200 || r.Status != "di_bawah" {
		t.Fatalf("NPM 2%% dagang harus di bawah benchmark: %+v", r)
	}
	if l := levels(r); l["Margin di bawah benchmark DJP"] != "tinggi" || l["Mendekati batas bebas pajak"] != "info" {
		t.Fatalf("risiko: %+v", r.Risiko)
	}

	r = CekAwal(CekAwalInput{JenisUsaha: "jasa", BentukUsaha: "op", PTKP: "TK/0", Omzet: 300_000_000, Laba: 60_000_000})
	if r.Status != "wajar" || len(r.Risiko) != 0 {
		t.Fatalf("jasa 20%% di bawah 500 jt: wajar tanpa risiko, got %+v", r)
	}

	r = CekAwal(CekAwalInput{JenisUsaha: "perdagangan", BentukUsaha: "op", PTKP: "TK/0", Omzet: 6_000_000_000, Laba: 600_000_000})
	l := levels(r)
	if l["Wajib menjadi PKP"] != "tinggi" || l["Tidak bisa memakai PPh Final 0,5%"] != "tinggi" || l["Bentuk usaha belum paling hemat"] != "sedang" {
		t.Fatalf("omzet 6 M OP belum PKP: %+v", r.Risiko)
	}

	if l := levels(CekAwal(CekAwalInput{JenisUsaha: "jasa", BentukUsaha: "cv", Omzet: 1_000_000_000, Laba: -5_000_000})); l["Usaha dilaporkan rugi"] != "tinggi" {
		t.Fatalf("rugi: %+v", l)
	}
}
