package tax

import (
	"slices"
	"strings"
	"testing"
)

func adaSaran(r MarketplaceResult, kata string) bool {
	for _, s := range r.Rekomendasi {
		if strings.Contains(s.Judul+" "+s.Isi, kata) {
			return true
		}
	}
	return false
}

func TestMarketplace_NPMKecilLebihBayarDanSaranSKB(t *testing.T) {
	// CV pembukuan, omzet 2 M semua lewat marketplace, NPM 3%: dipungut 10 jt (0,5%),
	// terutang Pasal 31E 60 jt x 11% = 6,6 jt → lebih bayar 3,4 jt.
	r, err := HitungMarketplace(MarketplaceInput{OmzetMarketplace: 2_000_000_000, NPMBps: 300, Bentuk: "badan", Skema: "umum"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Dipungut != 10_000_000 || r.Terutang != 6_600_000 || r.LebihBayar != 3_400_000 || r.Status != "lebih_bayar" {
		t.Fatalf("got %+v", r)
	}
	// ambang: 11% x laba = 0,5% x omzet → NPM 4,55%
	if r.NPMImpasBps != 455 {
		t.Errorf("NPM impas %d, want 455", r.NPMImpasBps)
	}
	if !adaSaran(r, "SKB") || !slices.ContainsFunc(r.Rekomendasi, func(s SaranPajak) bool { return s.Langganan }) {
		t.Errorf("saran SKB + ajakan berlangganan tidak ada: %+v", r.Rekomendasi)
	}
}

func TestMarketplace_NPMBesarKurangBayar(t *testing.T) {
	r, _ := HitungMarketplace(MarketplaceInput{OmzetMarketplace: 2_000_000_000, NPMBps: 1000, Bentuk: "badan", Skema: "umum"})
	// laba 200 jt x 11% = 22 jt; dikreditkan 10 jt → kurang 12 jt
	if r.Status != "kurang_bayar" || r.KurangBayar != 12_000_000 || r.LebihBayar != 0 || adaSaran(r, "SKB") {
		t.Fatalf("got %+v", r)
	}
}

func TestMarketplace_OPPembukuanPakaiPTKP(t *testing.T) {
	r, _ := HitungMarketplace(MarketplaceInput{OmzetMarketplace: 1_000_000_000, NPMBps: 500, Bentuk: "op", Skema: "umum", PTKP: "K/1"})
	// laba 50 jt < PTKP 63 jt → terutang 0, seluruh 5 jt lebih bayar
	if r.Terutang != 0 || r.LebihBayar != 5_000_000 || !adaSaran(r, "SKB") {
		t.Fatalf("got %+v", r)
	}
}

func TestMarketplace_FinalBagianPelunasan(t *testing.T) {
	// OP PP 55: omzet marketplace 800 jt + toko 400 jt = 1,2 M; terutang (1,2 M - 500 jt) x 0,5% = 3,5 jt,
	// dipungut 4 jt → kelebihan 0,5 jt (bagian bebas Rp500 jt ikut terpungut)
	r, _ := HitungMarketplace(MarketplaceInput{OmzetMarketplace: 800_000_000, OmzetLain: 400_000_000, NPMBps: 1000, Bentuk: "op", Skema: "final", PTKP: "TK/0"})
	if !r.Final || r.Terutang != 3_500_000 || r.Dipungut != 4_000_000 || r.LebihBayar != 500_000 {
		t.Fatalf("got %+v", r)
	}
	// pembanding pembukuan: laba 120 jt - 54 jt = 66 jt → 3 jt + 6 jt x 15% = 3,9 jt
	if r.TerutangUmum != 3_900_000 {
		t.Errorf("pembanding umum %d", r.TerutangUmum)
	}
}

func TestMarketplace_OPDiBawah500JutBebas(t *testing.T) {
	r, _ := HitungMarketplace(MarketplaceInput{OmzetMarketplace: 300_000_000, OmzetLain: 100_000_000, NPMBps: 1000, Bentuk: "op", Skema: "final", PTKP: "TK/0"})
	if r.Status != "bebas" || r.Dipungut != 0 || !adaSaran(r, "surat pernyataan") {
		t.Fatalf("got %+v", r)
	}
}

func TestMarketplace_FinalDiAtasBatasJadiUmum(t *testing.T) {
	r, err := HitungMarketplace(MarketplaceInput{OmzetMarketplace: 6_000_000_000, NPMBps: 500, Bentuk: "badan", Skema: "final"})
	if err != nil || r.Final || !strings.Contains(strings.Join(r.Catatan, " "), "Rp4.800.000.000") {
		t.Fatalf("got %+v %v", r, err)
	}
}

func TestMarketplace_TolakInputNgawur(t *testing.T) {
	for _, in := range []MarketplaceInput{
		{OmzetMarketplace: -1, Bentuk: "op", Skema: "final", PTKP: "TK/0"},
		{OmzetMarketplace: 1, Bentuk: "pt", Skema: "final"},
		{OmzetMarketplace: 1, Bentuk: "badan", Skema: "lain"},
		{OmzetMarketplace: 1, Bentuk: "op", Skema: "umum", PTKP: "X"},
		{OmzetMarketplace: 1, NPMBps: 20_000, Bentuk: "badan", Skema: "umum"},
	} {
		if _, err := HitungMarketplace(in); err == nil {
			t.Errorf("%+v diterima", in)
		}
	}
}

func TestMarketplace_TarifDariParameter(t *testing.T) {
	pakaiParameter(t, func(p *Parameter) { p.MarketplaceTarifBps = 100 })
	r, _ := HitungMarketplace(MarketplaceInput{OmzetMarketplace: 1_000_000_000, NPMBps: 1000, Bentuk: "badan", Skema: "umum"})
	if r.Dipungut != 10_000_000 {
		t.Fatalf("dipungut %d", r.Dipungut)
	}
}

func TestParameter_FieldMarketplaceKosongDiisiBawaan(t *testing.T) {
	p := ParameterBawaan()
	p.MarketplaceTarifBps, p.MarketplaceBebasOP, p.DasarMarketplace = 0, 0, "" // versi lama sebelum field ini
	p.Selaraskan()
	if p.MarketplaceTarifBps != 50 || p.MarketplaceBebasOP != 500_000_000 || p.DasarMarketplace != "PMK 37/2025" {
		t.Fatalf("got %d %d %q", p.MarketplaceTarifBps, p.MarketplaceBebasOP, p.DasarMarketplace)
	}
}

func TestMarketplace_LabaRupiahDipakaiPersis(t *testing.T) {
	// Laba 50.333.333 dari omzet 900 jt (5,5926%) — tidak boleh dibulatkan ke basis poin dulu.
	laba := int64(50_333_333)
	r, err := HitungMarketplace(MarketplaceInput{OmzetMarketplace: 900_000_000, Laba: &laba, Bentuk: "badan", Skema: "umum"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Laba != laba {
		t.Errorf("laba %d, mau %d persis", r.Laba, laba)
	}
	if r.Terutang != Pasal31E(900_000_000, laba) {
		t.Errorf("terutang %d tidak dihitung dari laba rupiah", r.Terutang)
	}
	lebih := int64(1_000_000_001)
	if _, err := HitungMarketplace(MarketplaceInput{OmzetMarketplace: 1_000_000_000, Laba: &lebih, Bentuk: "badan", Skema: "umum"}); err == nil {
		t.Error("laba melebihi omzet harus ditolak")
	}
}
