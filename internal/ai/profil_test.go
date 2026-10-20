package ai

import (
	"context"
	"strings"
	"testing"

	"pajak/internal/finance"
	"pajak/internal/llm"
)

func profilContoh() finance.BusinessProfile {
	return finance.BusinessProfile{NamaUsaha: "Toko Uji", JenisUsaha: "perdagangan", BentukUsaha: "op", PTKP: "K/3", KLU: "471",
		OmzetTahunan: 2_400_000_000, LabaSebelumPajak: 168_000_000, Kota: "Kota Surabaya"}
}

func TestFaktaDariProfil(t *testing.T) {
	f := FaktaDariProfil(profilContoh())
	if f.OmzetSetahun != 2_400_000_000 || f.NPMBps != 700 || f.KLU == nil || f.Saran == nil || f.Saran.Hemat != 1_100_000 {
		t.Fatalf("fakta = %+v", f)
	}
}

func TestAnalisaProfil_TanpaKunci(t *testing.T) {
	if a := AnalisaProfil(context.Background(), llm.New(llm.Config{}), FaktaDariProfil(profilContoh())); a.Tersedia {
		t.Fatalf("tanpa kunci = %+v", a)
	}
}

// Model diberi tahu bahwa datanya perkiraan sign-up (terbatas), dan menerima angka dari sistem.
func TestAnalisaProfil_KirimFaktaPerkiraan(t *testing.T) {
	var diterima string
	c := llmPalsu(t, `{"ringkasan":"Analisis awal dari data pendaftaran.","sorotan_positif":["Margin di atas rata-rata."],"rekomendasi":[{"judul":"Mulai catat transaksi","dampak":"tinggi","detail":"x"}]}`, &diterima)
	a := AnalisaProfil(context.Background(), c, FaktaDariProfil(profilContoh()))
	if !a.Tersedia || a.Ringkasan == "" || len(a.Rekomendasi) != 1 {
		t.Fatalf("analisa = %+v", a)
	}
	for _, want := range []string{"perkiraan", `"omzet_setahun":2400000000`, `"npm_bps":700`, "PPh Pasal 17"} {
		if !strings.Contains(diterima, want) {
			t.Errorf("pesan ke model tidak memuat %q", want)
		}
	}
}
