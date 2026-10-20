package ai

import (
	"context"
	"strings"
	"testing"

	"pajak/internal/finance"
	"pajak/internal/llm"
)

func TestKelompokkanMutasi_HanyaKelompokRendahTersamar(t *testing.T) {
	h := finance.MutasiHasil{Rows: []finance.MutasiRow{
		{Keterangan: "TRSF CR 1234567890 BUDI", Arah: "masuk", Nominal: 5_000_000, Jenis: "Penjualan", Keyakinan: finance.KeyakinanRendah},
		{Keterangan: "TRSF CR 5555666677 BUDI", Arah: "masuk", Nominal: 2_000_000, Jenis: "Penjualan", Keyakinan: finance.KeyakinanRendah},
		{Keterangan: "DB TOKO BANGUNAN MAJU", Arah: "keluar", Nominal: 900_000, Jenis: "Biaya Lain", Keyakinan: finance.KeyakinanRendah},
		{Keterangan: "BIAYA ADM", Arah: "keluar", Nominal: 10_000, Jenis: "Biaya Lain", Keyakinan: finance.KeyakinanTinggi},
	}, PerluDiperiksa: 3}
	var diterima string
	c := llmPalsu(t, `{"hasil":[{"no":1,"jenis":"Setor Modal","alasan":"transfer pemilik"},{"no":2,"jenis":"","alasan":"tidak jelas"},{"no":9,"jenis":"Penjualan"}]}`, &diterima)
	kel := h.KelompokRendah()
	saran, err := KelompokkanMutasi(context.Background(), c, "perdagangan · Toko QA", kel)
	if err != nil {
		t.Fatal(err)
	}
	h.TerapkanSaran(saran)
	if h.Rows[0].Jenis != "Setor Modal" || h.Rows[1].Keyakinan != finance.KeyakinanAI || h.Rows[2].Keyakinan != finance.KeyakinanRendah {
		t.Fatalf("hasil = %+v", h.Rows)
	}
	for _, larang := range []string{"1234567890", "5555666677", "BIAYA ADM"} {
		if strings.Contains(diterima, larang) {
			t.Errorf("terkirim ke model: %q", larang)
		}
	}
	if strings.Count(diterima, "TRSF CR # BUDI") != 1 || !strings.Contains(diterima, "Setor Modal") {
		t.Errorf("pesan ke model:\n%s", diterima)
	}
}

func TestKelompokkanMutasi_TanpaKunci(t *testing.T) {
	if _, err := KelompokkanMutasi(context.Background(), llm.New(llm.Config{}), "", []finance.KelompokMutasi{{Kunci: "x"}}); err == nil {
		t.Fatal("tanpa kunci tidak boleh berhasil")
	}
}
