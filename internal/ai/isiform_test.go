package ai

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"pajak/internal/llm"
)

func TestDeteksiIsiForm(t *testing.T) {
	for _, c := range []struct{ q, want string }{
		{"catat penjualan kopi 2 juta tunai hari ini", "transaksi"},
		{"tolong catatkan biaya listrik 450rb kemarin", "transaksi"},
		{"input pembelian bahan 1,2 jt belum lunas", "transaksi"},
		{"catat penjulan 500 ribu", "transaksi"}, // salah ketik
		{"bayar supplier CV Sumber 3 juta", ""},  // tanpa kata kerja isi form
		{"catat bayar supplier CV Sumber 3 juta", "transaksi"},
		{"buatkan invoice untuk PT Maju 10 kaos @50rb", "invoice"},
		{"buat invoice penjualan kaos ke PT Maju", "invoice"}, // invoice menang atas "penjualan"
		{"tambah barang kopi arabika 5 kg harga jual 120rb", "stok"},
		{"tambah pemasok CV Sumber Rejeki telepon 0812", "pemasok"},
		{"tambahkan supplier baru PT Bahan Jaya", "pemasok"},
		// Pertanyaan cara / tempat tetap dijawab panduan, bukan diisi.
		{"bagaimana cara catat penjualan?", ""},
		{"di mana saya mencatat biaya?", ""},
		{"menu untuk buat invoice di mana", ""},
		{"berapa laba bulan ini", ""},
		{"ringkas laporan bulan ini", ""},
	} {
		if got := DeteksiIsiForm(c.q); got != c.want {
			t.Errorf("DeteksiIsiForm(%q) = %q, want %q", c.q, got, c.want)
		}
	}
}

func isian(t *testing.T, h *IsiFormHasil, v any) {
	t.Helper()
	if err := json.Unmarshal(h.Isian, v); err != nil {
		t.Fatalf("isian %s: %v", h.Isian, err)
	}
}

// Model hanya membaca kalimat; server yang menjaga isian tetap sah sebelum sampai ke form.
func TestIsiForm_TransaksiDinormalisasi(t *testing.T) {
	var prompt string
	c := llmPalsu(t, `{"isian":{"tanggal":"","jenis":"Penjualan","nominal":2000000,"keterangan":" Penjualan kopi ","lunas":true,"asing":"x"},"catatan":["Cek nama pelanggan"]}`, &prompt)

	h, err := IsiForm(context.Background(), c, "transaksi", "catat penjualan kopi 2 juta tunai", "2026-10-22")
	if err != nil {
		t.Fatalf("IsiForm: %v", err)
	}
	if h.Form != "transaksi" || h.View != "transaksi" {
		t.Errorf("form/view = %q/%q", h.Form, h.View)
	}
	var tx map[string]any
	isian(t, h, &tx)
	if tx["tanggal"] != "2026-10-22" || tx["jenis"] != "Penjualan" || tx["nominal"] != float64(2000000) ||
		tx["keterangan"] != "Penjualan kopi" || tx["lunas"] != true {
		t.Errorf("isian = %v", tx)
	}
	if _, ok := tx["asing"]; ok {
		t.Error("kolom yang tidak ada di form harus dibuang")
	}
	if !strings.Contains(strings.Join(h.Catatan, "|"), "hari ini") || !strings.Contains(strings.Join(h.Catatan, "|"), "Cek nama pelanggan") {
		t.Errorf("catatan = %v", h.Catatan)
	}
	if !strings.Contains(prompt, "catat penjualan kopi 2 juta tunai") {
		t.Errorf("kalimat pengguna harus dikirim ke model, got %q", prompt)
	}
}

func TestIsiForm_TransaksiJenisDanNominalTidakSah(t *testing.T) {
	var prompt string
	c := llmPalsu(t, `{"isian":{"tanggal":"2026-13-40","jenis":"Jualan Online","nominal":-5000},"catatan":[]}`, &prompt)

	h, err := IsiForm(context.Background(), c, "transaksi", "catat jualan online", "2026-10-22")
	if err != nil {
		t.Fatalf("IsiForm: %v", err)
	}
	var tx map[string]any
	isian(t, h, &tx)
	if _, ok := tx["jenis"]; ok {
		t.Errorf("jenis di luar daftar harus dibuang, isian %v", tx)
	}
	if _, ok := tx["nominal"]; ok {
		t.Errorf("nominal negatif harus dibuang, isian %v", tx)
	}
	if tx["tanggal"] != "2026-10-22" {
		t.Errorf("tanggal tidak sah harus jadi hari ini, got %v", tx["tanggal"])
	}
	cat := strings.Join(h.Catatan, "|")
	for _, w := range []string{"Jenis", "Nominal"} {
		if !strings.Contains(cat, w) {
			t.Errorf("catatan harus menyebut %s, got %v", w, h.Catatan)
		}
	}
}

func TestIsiForm_InvoiceTempoHariJadiTanggal(t *testing.T) {
	var prompt string
	c := llmPalsu(t, `{"isian":{"tanggal":"2026-10-22","tempo_hari":30,"pelanggan_nama":"PT Maju","items":[{"deskripsi":"Kaos","qty":10,"satuan":"pcs","harga_satuan":50000},{"deskripsi":"","qty":1,"harga_satuan":1}],"catatan_invoice":"Terima kasih"},"catatan":[]}`, &prompt)

	h, err := IsiForm(context.Background(), c, "invoice", "buatkan invoice PT Maju 10 kaos @50rb tempo 30 hari", "2026-10-22")
	if err != nil {
		t.Fatalf("IsiForm: %v", err)
	}
	var inv struct {
		Tanggal, JatuhTempo string
		PelangganNama       string `json:"pelanggan_nama"`
		Items               []struct {
			Deskripsi   string
			Qty         int64
			Satuan      string
			HargaSatuan int64 `json:"harga_satuan"`
		}
		Catatan string
	}
	isian(t, h, &inv)
	if h.View != "invoice" || inv.PelangganNama != "PT Maju" || inv.Catatan != "Terima kasih" {
		t.Errorf("isian = %s", h.Isian)
	}
	// Tanggal jatuh tempo dihitung server, bukan model.
	var jt struct {
		JatuhTempo string `json:"jatuh_tempo"`
	}
	isian(t, h, &jt)
	if jt.JatuhTempo != "2026-11-21" {
		t.Errorf("jatuh tempo = %q, want 2026-11-21", jt.JatuhTempo)
	}
	if len(inv.Items) != 1 || inv.Items[0].Deskripsi != "Kaos" || inv.Items[0].Qty != 10 || inv.Items[0].HargaSatuan != 50000 {
		t.Errorf("item tanpa deskripsi harus dibuang, items %+v", inv.Items)
	}
}

func TestIsiForm_StokJumlahBulat(t *testing.T) {
	var prompt string
	c := llmPalsu(t, `{"isian":{"nama":"Kopi Arabika","satuan":"kg","jumlah":2.5,"stok_min":1,"harga_pokok":"85000","harga_jual":120000},"catatan":[]}`, &prompt)

	h, err := IsiForm(context.Background(), c, "stok", "tambah barang kopi arabika 2,5 kg", "2026-10-22")
	if err != nil {
		t.Fatalf("IsiForm: %v", err)
	}
	var s map[string]any
	isian(t, h, &s)
	// Jumlah di daftar stok bilangan bulat dalam satuannya (nilai = jumlah × harga pokok);
	// pecahan dibulatkan dan pengguna diberi tahu.
	if h.View != "stok-gudang" || s["nama"] != "Kopi Arabika" || s["jumlah"] != float64(3) || s["stok_min"] != float64(1) ||
		s["harga_pokok"] != float64(85000) || s["harga_jual"] != float64(120000) {
		t.Errorf("isian = %v", s)
	}
	if !strings.Contains(strings.Join(h.Catatan, "|"), "dibulatkan") {
		t.Errorf("catatan = %v", h.Catatan)
	}
}

func TestIsiForm_Pemasok(t *testing.T) {
	var prompt string
	c := llmPalsu(t, `{"isian":{"nama":"CV Sumber Rejeki","telepon":"0812 3456","tempo_hari":14},"catatan":[]}`, &prompt)

	h, err := IsiForm(context.Background(), c, "pemasok", "tambah pemasok CV Sumber Rejeki 0812 3456 tempo 14 hari", "2026-10-22")
	if err != nil {
		t.Fatalf("IsiForm: %v", err)
	}
	var p map[string]any
	isian(t, h, &p)
	if h.View != "produksi-supplier" || p["nama"] != "CV Sumber Rejeki" || p["telepon"] != "0812 3456" || p["tempo_hari"] != float64(14) {
		t.Errorf("isian = %v", p)
	}
}

func TestIsiForm_TanpaIsianGagal(t *testing.T) {
	var prompt string
	c := llmPalsu(t, `{"isian":{},"catatan":[]}`, &prompt)
	if _, err := IsiForm(context.Background(), c, "pemasok", "tambah pemasok", "2026-10-22"); err == nil {
		t.Error("isian kosong harus gagal supaya pengguna diarahkan mengisi sendiri")
	}
	if _, err := IsiForm(context.Background(), llm.New(llm.Config{}), "transaksi", "catat penjualan", "2026-10-22"); err == nil {
		t.Error("tanpa LLM harus gagal")
	}
}
