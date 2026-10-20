package ai

import (
	"context"
	"strings"
	"testing"

	"pajak/internal/llm"
)

func TestPilihKLU_HanyaKodeDariDaftar(t *testing.T) {
	var diterima string
	c := llmPalsu(t, `{"saran":[{"kode":"563","alasan":"Kedai kopi termasuk penyediaan minuman."},{"kode":"999","alasan":"karangan"},{"kode":"563","alasan":"dobel"},{"kode":"472","alasan":"Bila menjual biji kopi."}]}`, &diterima)
	s, err := PilihKLU(context.Background(), c, "jual kopi susu kekinian", "perdagangan")
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != 2 || s[0].Kode != "563" || s[1].Kode != "472" || s[0].Nama != "Penyediaan Minuman" || len(s[0].Contoh) == 0 || s[0].Alasan == "" {
		t.Fatalf("saran = %+v", s)
	}
	for _, want := range []string{"jual kopi susu kekinian", "563 | Penyediaan Minuman |", "Rumah minum/kafe", "perdagangan"} {
		if !strings.Contains(diterima, want) {
			t.Errorf("pesan ke model tidak memuat %q", want)
		}
	}
}

func TestPilihKLU_TanpaKunciAtauKodeKarangan(t *testing.T) {
	if _, err := PilihKLU(context.Background(), llm.New(llm.Config{}), "laundry", ""); err == nil {
		t.Error("tanpa kunci tidak boleh berhasil")
	}
	var d string
	if _, err := PilihKLU(context.Background(), llmPalsu(t, `{"saran":[{"kode":"000","alasan":"x"}]}`, &d), "laundry", ""); err == nil {
		t.Error("kode karangan semua harus gagal (dipakai pencarian biasa)")
	}
}
