package main

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png" // dekoder foto mutasi & gambar PNG di PDF
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	pdfmodel "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	"pajak/internal/ai"
	"pajak/internal/finance"
)

// PDF mutasi hasil scan dan foto mutasi: tiap halaman (gambar terbesar per halaman PDF, atau
// gambar yang diunggah) dibaca model pembaca gambar, jawabannya diubah menjadi baris teks gaya
// rekening koran, lalu diurai parser PDF biasa (finance.MutasiDariTeksCek). Arah dan cek saldo
// berjalan jadi sama dengan PDF asli. Image produksi tanpa poppler, jadi gambarnya diambil
// langsung dari PDF dengan pdfcpu.

const (
	maxHalamanScan = 15
	maxScanMutasi  = 60 // halaman dibaca AI per usaha per 24 jam
	paralelScan    = 3
	batasWaktuScan = 80 * time.Second // di bawah batas 100 detik Cloudflare
	sisiMaksScan   = 2000             // px; cukup untuk teks rekening koran, hemat kuota
)

var (
	errScanTanpaGambar = errors.New("format gambar di PDF ini belum bisa dibaca. Foto halaman mutasi dengan HP lalu unggah sebagai JPG, atau pakai mutasi .xlsx / .csv")
	errScanGagal       = errors.New("AI gagal membaca hasil scan. Coba lagi, atau unggah mutasi .xlsx / .csv dari m-banking")
	errScanKosong      = errors.New("AI tidak menemukan baris transaksi di hasil scan ini. Pastikan gambarnya halaman mutasi yang tegak dan terang")
	errScanKuota       = errors.New("kuota baca hasil scan sudah habis; coba lagi dalam 24 jam, atau unggah mutasi .xlsx / .csv")
	errFotoMutasiMati  = errors.New("foto mutasi belum bisa dibaca saat ini. Unggah mutasi PDF, .xlsx, atau .csv dari m-banking")
)

// bacaScanFunc: pembaca hasil scan untuk pratinjau mutasi; nil = model pembaca gambar tidak aktif.
type bacaScanFunc func(ctx context.Context, data []byte, sandi string, tahun int) ([][]string, []string, error)

type halamanScan struct {
	mime string
	data []byte
}

// isiGambar: jenis foto mutasi yang diterima, dari isi berkas (bukan nama); "" = bukan gambar.
func isiGambar(data []byte) string {
	switch m := http.DetectContentType(data); m {
	case "image/jpeg", "image/png", "image/webp":
		return m
	}
	return ""
}

// gambarHalamanPDF mengambil gambar terbesar tiap halaman (hasil scan = satu gambar per halaman).
// JBIG2/JPEG 2000 dan sejenisnya dilewati karena tidak bisa dibaca model.
func gambarHalamanPDF(data []byte, sandi string) (out []halamanScan, err error) {
	defer func() {
		if r := recover(); r != nil {
			out, err = nil, errPDFTerkunci
		}
	}()
	data = potongAwalPDF(data)
	if bytes.Contains(data, []byte("/Encrypt")) {
		// ExtractImagesRaw tidak membuka enkripsi sendiri; pakai jalur sandi yang sama dengan PDF teks
		if data, err = bukaSandiPDF(data, sandi); err != nil {
			return nil, err
		}
	}
	conf := pdfmodel.NewDefaultConfiguration()
	conf.ValidationMode = pdfmodel.ValidationRelaxed
	halaman, err := pdfapi.ExtractImagesRaw(bytes.NewReader(data), nil, conf)
	if err != nil {
		return nil, errPDFTerkunci
	}
	for _, gambar := range halaman {
		var pilih *pdfmodel.Image
		for _, g := range gambar {
			if g.Reader != nil && (g.FileType == "jpg" || g.FileType == "png") && (pilih == nil || g.Width*g.Height > pilih.Width*pilih.Height) {
				pilih = &g
			}
		}
		if pilih == nil {
			continue
		}
		b, err := io.ReadAll(pilih.Reader)
		if err == nil && len(b) > 0 {
			out = append(out, halamanScan{map[string]string{"jpg": "image/jpeg", "png": "image/png"}[pilih.FileType], b})
		}
	}
	return out, nil
}

// kecilkanGambar: JPEG dengan sisi panjang maks sisiMaksScan di atas latar putih. Gagal dibaca =
// kirim apa adanya, biar model yang mencoba.
func kecilkanGambar(data []byte, mime string) ([]byte, string) {
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil && mime == "image/jpeg" && max(cfg.Width, cfg.Height) <= sisiMaksScan {
		return data, mime
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data, mime
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if s := max(w, h); s > sisiMaksScan {
		w, h = max(1, w*sisiMaksScan/s), max(1, h*sisiMaksScan/s)
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	var out bytes.Buffer
	if jpeg.Encode(&out, dst, &jpeg.Options{Quality: 85}) != nil {
		return data, mime
	}
	return out.Bytes(), "image/jpeg"
}

// bacaMutasiScan membaca halaman scan / foto mutasi dengan AI (paralel, dengan tenggat) menjadi
// tabel mutasi + peringatan. Halaman yang gagal tidak menggagalkan halaman lain.
func (s *Server) bacaMutasiScan(ctx context.Context, tenantID int, data []byte, sandi string, tahun int) ([][]string, []string, error) {
	var hal []halamanScan
	if m := isiGambar(data); m != "" {
		hal = []halamanScan{{m, data}}
	} else {
		var err error
		if hal, err = gambarHalamanPDF(data, sandi); err != nil {
			return nil, nil, err
		}
	}
	if len(hal) == 0 {
		return nil, nil, errScanTanpaGambar
	}
	var peringatan []string
	if len(hal) > maxHalamanScan {
		peringatan = append(peringatan, fmt.Sprintf("Hanya %d halaman pertama yang dibaca; unggah sisanya terpisah.", maxHalamanScan))
		hal = hal[:maxHalamanScan]
	}
	if tahun == 0 {
		tahun = time.Now().In(finance.WIB).Year()
	}
	ctx, cancel := context.WithTimeout(ctx, batasWaktuScan)
	defer cancel()
	hasil := make([]ai.MutasiScan, len(hal))
	gagal := make([]error, len(hal))
	var wg sync.WaitGroup
	antre := make(chan struct{}, paralelScan)
	for i, h := range hal {
		if !s.jatahAI("mutasi-scan|"+strconv.Itoa(tenantID), maxScanMutasi) {
			gagal[i] = errScanKuota
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			antre <- struct{}{}
			defer func() { <-antre }()
			img, mime := kecilkanGambar(h.data, h.mime)
			hasil[i], gagal[i] = ai.BacaMutasiScan(ctx, s.llmGambar, mime, img, tahun)
		}()
	}
	wg.Wait()

	var m ai.MutasiScan
	dibaca := 0
	for i, err := range gagal {
		switch {
		case errors.Is(err, errScanKuota):
			peringatan = append(peringatan, fmt.Sprintf("Halaman %d tidak dibaca: kuota baca hasil scan habis.", i+1))
		case err != nil:
			// detail (alamat penyedia, nama model) hanya untuk log server
			log.Printf("baca scan mutasi usaha %d halaman %d: %v", tenantID, i+1, err)
			peringatan = append(peringatan, fmt.Sprintf("Halaman %d tidak terbaca AI; periksa baris halaman itu sendiri.", i+1))
		default:
			dibaca++
			if i == 0 {
				m.SaldoAwal = hasil[0].SaldoAwal
			}
			m.Baris = append(m.Baris, hasil[i].Baris...)
		}
	}
	if dibaca == 0 {
		if errors.Is(gagal[0], errScanKuota) {
			return nil, nil, errScanKuota
		}
		return nil, nil, errScanGagal
	}
	rows, cek := finance.MutasiDariTeksCek(teksMutasiScan(m))
	if rows == nil {
		return nil, nil, errScanKosong
	}
	peringatan = append(cek, peringatan...)
	peringatan = append(peringatan, fmt.Sprintf("Dibaca AI dari hasil scan (%d halaman). Cocokkan jumlah baris, total uang masuk dan keluar, serta saldo akhir dengan rekening koran sebelum menyimpan.", len(hal)))
	return rows, peringatan, nil
}

// polaUangKet: angka uang di dalam keterangan; dibuang supaya parser tidak mengiranya nominal.
var polaUangKet = regexp.MustCompile(`\d{1,3}(?:[.,]\d{3})+(?:[.,]\d{2})?`)

// rupiahTeks: 1500000 → "1,500,000.00" (gaya rekening koran yang dikenali parser PDF).
func rupiahTeks(n int64) string {
	if n < 0 {
		return "-" + rupiahTeks(-n)
	}
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s + ".00"
}

// teksMutasiScan menyusun jawaban model menjadi baris rekening koran:
// "2026-09-02 KETERANGAN 1,500,000.00 CR 11,500,000.00". Baris tanpa tanggal/nominal sah dibuang.
func teksMutasiScan(m ai.MutasiScan) []string {
	var out []string
	for _, b := range m.Baris {
		if _, err := time.Parse(time.DateOnly, b.Tanggal); err != nil || b.Nominal <= 0 {
			continue
		}
		if len(out) == 0 && m.SaldoAwal != nil {
			out = append(out, b.Tanggal+" SALDO AWAL "+rupiahTeks(*m.SaldoAwal))
		}
		ket := strings.Join(strings.Fields(polaUangKet.ReplaceAllString(b.Keterangan, "")), " ")
		baris := b.Tanggal + " " + cmp.Or(ket, "TRANSAKSI") + " " + rupiahTeks(b.Nominal)
		switch b.Arah {
		case "masuk":
			baris += " CR"
		case "keluar":
			baris += " DB"
		}
		if b.Saldo != nil {
			baris += " " + rupiahTeks(*b.Saldo)
		}
		out = append(out, baris)
	}
	return out
}
