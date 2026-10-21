package ai

import (
	"context"
	"slices"
	"strings"
	"testing"
)

func semuaBoleh(string) bool { return true }

func viewsDari(t []Tautan) []string {
	var v []string
	for _, x := range t {
		v = append(v, x.View+"#"+x.Tab)
	}
	return v
}

func TestCariPanduan_IsiLaporanKeuangan(t *testing.T) {
	for _, q := range []string{"dmn saya bisa isi laporan keuangan", "Di mana saya bisa mengisi laporan keuangan?", "gimana cara input laporan keuangan"} {
		p, ok := CariPanduan(q, semuaBoleh)
		if !ok || !p.Kuat {
			t.Fatalf("%q: tidak dikenali sebagai pertanyaan menu (%+v)", q, p)
		}
		v := viewsDari(p.Tautan)
		if !slices.Contains(v, "transaksi#") || !slices.Contains(v, "laporan-keuangan#") {
			t.Errorf("%q: tautan %v, mau Transaksi & Laporan Keuangan", q, v)
		}
		if !strings.Contains(p.Jawaban, "otomatis") {
			t.Errorf("%q: jawaban harus menjelaskan laporan tersusun otomatis: %s", q, p.Jawaban)
		}
	}
}

func TestCariPanduan_TopikLain(t *testing.T) {
	cek := map[string]string{
		"cara impor mutasi bank BCA":         "transaksi#",
		"dimana saya isi saldo awal":         "saldo-awal#",
		"mau download laporan keuangan pdf":  "laporan-keuangan#",
		"bandingkan laporan beberapa bulan":  "laporan-keuangan#banding",
		"di mana catat piutang pelanggan":    "piutang-utang#",
		"cara bikin invoice":                 "invoice#",
		"tambah aset mesin espresso di mana": "laporan-keuangan#aset",
		"cara tambah kasir baru":             "pengguna#",
		"dimana ganti nama usaha":            "profil-usaha#",
		"gimana cara berlangganan":           "langganan#",
		"hitung harga jual gofood dimana":    "simulasi-harga#",
		"dimana lihat stok barang":           "stok-gudang#",
	}
	for q, mau := range cek {
		p, ok := CariPanduan(q, semuaBoleh)
		if !ok || len(p.Tautan) == 0 || viewsDari(p.Tautan)[0] != mau {
			t.Errorf("%q: %v, mau tautan pertama %s", q, viewsDari(p.Tautan), mau)
		}
	}
}

func TestCariPanduan_BukanPertanyaanMenu(t *testing.T) {
	for _, q := range []string{"Ringkas laporan bulan ini", "halo", "berapa laba saya"} {
		if p, ok := CariPanduan(q, semuaBoleh); ok && p.Kuat {
			t.Errorf("%q dianggap pertanyaan menu: %+v", q, p)
		}
	}
	// Pertanyaan pengetahuan tetap dijawab mesin biasa, tapi boleh diberi tautan terkait.
	p, ok := CariPanduan("Bagaimana aturan pajak UMKM PP 55/2022?", semuaBoleh)
	if !ok || p.Kuat || len(p.Tautan) == 0 || p.Tautan[0].View != "pajak-umkm" {
		t.Errorf("aturan pajak: %+v", p)
	}
}

func TestCariPanduan_HormatiHakAkses(t *testing.T) {
	kasir := func(v string) bool { return v == "dashboard" || v == "transaksi" || v == "lapor-bug" }
	p, ok := CariPanduan("dmn saya bisa isi laporan keuangan", kasir)
	if !ok {
		t.Fatal("tidak dikenali")
	}
	for _, x := range p.Tautan {
		if !kasir(x.View) {
			t.Errorf("kasir diberi tautan ke %s", x.View)
		}
	}
	p, ok = CariPanduan("cara tambah pengguna baru", kasir)
	if !ok || len(p.Tautan) != 0 || !strings.Contains(p.Jawaban, "Owner") {
		t.Errorf("menu tanpa akses harus dijelaskan, bukan ditautkan: %+v", p)
	}
}

func TestPilihPanduanAI(t *testing.T) {
	var diterima string
	c := llmPalsu(t, `{"topik":["piutang","tidak-ada"]}`, &diterima)
	p, err := PilihPanduan(context.Background(), c, "pelanggan saya ngutang, nyatetnya dimana ya", semuaBoleh)
	if err != nil || len(p.Tautan) == 0 || p.Tautan[0].View != "piutang-utang" || !p.AI {
		t.Fatalf("p=%+v err=%v", p, err)
	}
	if !strings.Contains(diterima, "piutang") || !strings.Contains(diterima, "pelanggan saya ngutang") {
		t.Error("prompt tidak memuat daftar topik / pertanyaan")
	}
	var buang string
	kosong := llmPalsu(t, `{"topik":["ngarang"]}`, &buang)
	if _, err := PilihPanduan(context.Background(), kosong, "x", semuaBoleh); err == nil {
		t.Error("topik di luar daftar harus ditolak")
	}
	if _, err := PilihPanduan(context.Background(), nil, "x", semuaBoleh); err == nil {
		t.Error("tanpa LLM harus error")
	}
}

func TestCariPanduan_LemahSatuTautan(t *testing.T) {
	p, ok := CariPanduan("Bagaimana kondisi arus kas dan piutang saya?", semuaBoleh)
	if !ok || p.Kuat || len(p.Tautan) != 1 || p.Tautan[0].View != "laporan-keuangan" {
		t.Errorf("%+v", p)
	}
}

func TestCariPanduan_TahanSalahKetik(t *testing.T) {
	cek := map[string]string{
		"kalau mau input trasaksi dmn ya ?": "transaksi#",
		"dimana bikin invoce":               "invoice#",
		"cara impor mutsai bank":            "transaksi#",
		"dmn isi saldo awla":                "saldo-awal#",
	}
	for q, mau := range cek {
		p, ok := CariPanduan(q, semuaBoleh)
		if !ok || !p.Kuat || len(p.Tautan) == 0 || viewsDari(p.Tautan)[0] != mau {
			t.Errorf("%q: %+v, mau %s", q, viewsDari(p.Tautan), mau)
		}
	}
	// Kata pendek tidak dicocokkan longgar: "stok" ≠ "stop".
	if p, ok := CariPanduan("dimana tombol stop", semuaBoleh); ok && p.Kode == "stok" {
		t.Error("kata pendek tidak boleh dicocokkan longgar")
	}
}

func TestPanduanUmum(t *testing.T) {
	kasir := func(v string) bool { return v == "dashboard" || v == "transaksi" || v == "lapor-bug" }
	p := PanduanUmum(kasir)
	if !p.Kuat || len(p.Tautan) == 0 {
		t.Fatalf("%+v", p)
	}
	for _, l := range p.Tautan {
		if !kasir(l.View) {
			t.Errorf("tautan %s tidak boleh untuk kasir", l.View)
		}
	}
}
