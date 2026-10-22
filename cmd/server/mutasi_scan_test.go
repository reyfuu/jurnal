package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	pdfmodel "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

	"pajak/internal/ai"
	"pajak/internal/finance"
	"pajak/internal/llm"
)

// scanPDF: PDF tanpa teks berisi satu gambar JPEG per halaman, seperti hasil scan rekening koran.
func scanPDF(t *testing.T, halaman int) []byte {
	t.Helper()
	var imgs []io.Reader
	for range halaman {
		img := image.NewRGBA(image.Rect(0, 0, 120, 170))
		for x := range 120 {
			img.Set(x, 40, color.Black)
		}
		var b bytes.Buffer
		if err := jpeg.Encode(&b, img, nil); err != nil {
			t.Fatal(err)
		}
		imgs = append(imgs, bytes.NewReader(b.Bytes()))
	}
	var out bytes.Buffer
	if err := pdfapi.ImportImages(nil, &out, imgs, nil, nil); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

const jawabanScan = `{"saldo_awal":10000000,"baris":[
	{"tanggal":"2026-09-02","keterangan":"TRSF E-BANKING DARI BUDI SANTOSO","nominal":1500000,"arah":"masuk","saldo":11500000},
	{"tanggal":"2026-09-03","keterangan":"TARIKAN ATM 0938","nominal":500000,"arah":"keluar","saldo":11000000}]}`

// llmScan: model palsu yang membaca setiap halaman scan sebagai dua baris mutasi; panggilan dihitung.
func llmScan(t *testing.T, jawab string, panggil *atomic.Int32) *llm.Client {
	t.Helper()
	palsu := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panggil.Add(1)
		b, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(b), `"image_url"`) {
			t.Errorf("permintaan baca scan tanpa gambar: %.200s", b)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": jawab}}}})
	}))
	t.Cleanup(palsu.Close)
	return llm.New(llm.Config{APIKey: "uji", BaseURL: palsu.URL, Model: "model-uji"})
}

// Jawaban model diubah jadi baris teks gaya rekening koran lalu diurai parser PDF biasa, sehingga
// arah, nominal, dan cek saldo berjalan sama dengan PDF asli.
func TestTeksMutasiScan_DiuraiParserPDF(t *testing.T) {
	var m ai.MutasiScan
	if err := json.Unmarshal([]byte(jawabanScan), &m); err != nil {
		t.Fatal(err)
	}
	m.Baris = append(m.Baris, ai.BarisScan{Tanggal: "bukan tanggal", Keterangan: "sampah", Nominal: 1})
	rows, peringatan := finance.MutasiDariTeksCek(teksMutasiScan(m))
	h := finance.ParseMutasi(rows, 2026, nil)
	if len(h.Rows) != 2 || len(peringatan) != 0 {
		t.Fatalf("baris %+v peringatan %v", h.Rows, peringatan)
	}
	if r := h.Rows[0]; r.Tanggal != "2026-09-02" || r.Arah != "masuk" || r.Nominal != 1_500_000 || !strings.Contains(r.Keterangan, "BUDI") {
		t.Errorf("baris 1: %+v", r)
	}
	if r := h.Rows[1]; r.Tanggal != "2026-09-03" || r.Arah != "keluar" || r.Nominal != 500_000 {
		t.Errorf("baris 2: %+v", r)
	}
}

// Jawaban asli cx/gpt-5.6-terra (9Router produksi, 22 Okt 2026) untuk gambar rekening koran gaya BCA
// berisi keterangan dua baris dan biaya admin; harus terurai lengkap tanpa peringatan saldo.
func TestTeksMutasiScan_JawabanModelNyata(t *testing.T) {
	const nyata = `{"saldo_awal":10000000,"baris":[{"tanggal":"2026-09-02","keterangan":"TRSF E-BANKING CR BUDI SANTOSO","nominal":1500000,"arah":"masuk","saldo":11500000},{"tanggal":"2026-09-03","keterangan":"TARIKAN ATM 0938","nominal":500000,"arah":"keluar","saldo":11000000},{"tanggal":"2026-09-05","keterangan":"KR OTOMATIS LLG-ANTAR BANK","nominal":2000000,"arah":"masuk","saldo":13000000},{"tanggal":"2026-09-08","keterangan":"BIAYA ADM","nominal":15000,"arah":"keluar","saldo":12985000}]}`
	var m ai.MutasiScan
	if err := json.Unmarshal([]byte(nyata), &m); err != nil {
		t.Fatal(err)
	}
	rows, peringatan := finance.MutasiDariTeksCek(teksMutasiScan(m))
	h := finance.ParseMutasi(rows, 2026, nil)
	if len(h.Rows) != 4 || len(peringatan) != 0 || h.TotalMasuk != 3_500_000 || h.TotalKeluar != 515_000 {
		t.Fatalf("baris %d peringatan %v masuk %d keluar %d", len(h.Rows), peringatan, h.TotalMasuk, h.TotalKeluar)
	}
}

// PDF hasil scan (tanpa teks) dibaca AI per halaman; pratinjaunya sama dengan PDF biasa plus
// peringatan agar dicocokkan dengan rekening koran.
func TestMutasi_PDFScanDibacaAI(t *testing.T) {
	srv := NewServer()
	var panggil atomic.Int32
	srv.llmGambar = llmScan(t, jawabanScan, &panggil)
	code, raw, h := unggahBerkasMutasi(srv, login(t, srv, "owner@kedaikopisenja.id"), "scan.pdf", scanPDF(t, 2))
	if code != http.StatusOK || len(h.Rows) != 4 {
		t.Fatalf("pratinjau scan: %d %s", code, raw)
	}
	if panggil.Load() != 2 {
		t.Fatalf("satu panggilan AI per halaman, dapat %d", panggil.Load())
	}
	if !strings.Contains(strings.Join(h.Peringatan, " "), "hasil scan (2 halaman)") {
		t.Fatalf("peringatan scan: %v", h.Peringatan)
	}
}

// Foto halaman mutasi (JPG/PNG) dibaca lewat jalur yang sama.
func TestMutasi_FotoMutasiDibacaAI(t *testing.T) {
	srv := NewServer()
	var panggil atomic.Int32
	srv.llmGambar = llmScan(t, jawabanScan, &panggil)
	code, raw, h := unggahBerkasMutasi(srv, login(t, srv, "owner@kedaikopisenja.id"), "mutasi.png", png1x1)
	if code != http.StatusOK || len(h.Rows) != 2 || panggil.Load() != 1 {
		t.Fatalf("foto mutasi: %d %s (panggilan %d)", code, raw, panggil.Load())
	}
}

// Rekening koran hasil scan dari bank biasanya juga dikunci kata sandi (AES-256).
func TestGambarHalamanPDF_BerkataSandi(t *testing.T) {
	kunci := kunciPDF(t, scanPDF(t, 2), pdfmodel.NewAESConfiguration("010190", "pemilik-bank", 256))
	hal, err := gambarHalamanPDF(kunci, "010190")
	if err != nil || len(hal) != 2 || hal[0].mime != "image/jpeg" {
		t.Fatalf("sandi benar: %v, %d halaman", err, len(hal))
	}
	if _, err := gambarHalamanPDF(kunci, "salah"); err == nil {
		t.Fatal("sandi salah harus gagal")
	}
}

// Gambar besar diperkecil ke sisi maks 2000 px (JPEG) sebelum dikirim ke model.
func TestKecilkanGambar_SisiMaks(t *testing.T) {
	var b bytes.Buffer
	_ = png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 3000, 1000)))
	out, mime := kecilkanGambar(b.Bytes(), "image/png")
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(out))
	if err != nil || mime != "image/jpeg" || cfg.Width != 2000 || cfg.Height != 666 {
		t.Fatalf("hasil %s %dx%d %v", mime, cfg.Width, cfg.Height, err)
	}
}

// Tanpa model pembaca gambar, PDF scan tetap ditolak dengan pesan lama; bila AI gagal, pesannya
// tidak membocorkan alamat penyedia atau nama model.
func TestMutasi_ScanTanpaAIAtauAIGagal(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	srv.llmGambar = llm.New(llm.Config{})
	if code, raw, _ := unggahBerkasMutasi(srv, owner, "scan.pdf", scanPDF(t, 1)); code != http.StatusBadRequest || !strings.Contains(raw, "tidak berisi teks") {
		t.Fatalf("tanpa AI: %d %s", code, raw)
	}

	palsu := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"message":"[rahasia/model-x] upstream 404"}}`))
	}))
	defer palsu.Close()
	srv.llmGambar = llm.New(llm.Config{APIKey: "uji", BaseURL: palsu.URL, Model: "model-x"})
	code, raw, _ := unggahBerkasMutasi(srv, owner, "scan.pdf", scanPDF(t, 1))
	if code != http.StatusBadRequest || !strings.Contains(raw, "AI gagal membaca") {
		t.Fatalf("AI gagal: %d %s", code, raw)
	}
	for _, bocor := range []string{palsu.URL, "model-x", "rahasia"} {
		if strings.Contains(raw, bocor) {
			t.Fatalf("pesan membocorkan %q: %s", bocor, raw)
		}
	}
}
