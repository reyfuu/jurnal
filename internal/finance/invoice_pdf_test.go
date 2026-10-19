package finance_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"pajak/internal/finance"
	"pajak/internal/pdf"
)

func contohProfil() finance.BusinessProfile {
	return finance.BusinessProfile{
		NamaUsaha: "Kedai Kopi Senja", JenisUsaha: "perdagangan", BentukUsaha: "cv", PKP: true,
		Alamat:  "Jl. Senopati Raya No. 45, Kebayoran Baru, Jakarta Selatan 12190",
		NPWP:    "01.234.567.8-012.000",
		Telepon: "021-7123456", Email: "halo@kedaikopisenja.id",
		BankNama: "BCA", BankRekening: "123-456-7890", BankAtasNama: "CV Kedai Kopi Senja",
		PenandatanganNama: "Sari Wulandari", PenandatanganJabatan: "Pemilik",
		PPNBps: 1100,
	}
}

func contohInvoice() finance.Invoice {
	return finance.Invoice{
		Nomor: "INV-2026-061", NoPO: "PO-KDM-2026-0317", Status: finance.StatusTerkirim,
		Tanggal: "2026-06-10", JatuhTempo: "2026-07-10",
		PelangganNama:   "PT Karya Digital Mandiri",
		PelangganAlamat: "Gedung Menara Rajawali Lt. 8, Jl. DR. Ide Anak Agung Gde Agung, Kuningan, Jakarta Selatan 12950",
		PelangganNPWP:   "02.345.678.9-012.000",
		PPNBps:          1100,
		DiskonRp:        200_000,
		Catatan:         "Pembayaran mohon dilakukan sebelum jatuh tempo.",
		Items: []finance.InvoiceItem{
			{Deskripsi: "Kopi rapat bulanan - paket 20 cup", Qty: 60, Satuan: "cup", HargaSatuan: 75_000},
			{Deskripsi: "Snack box isi 4 (risol, pastel, lemper, kue lapis)", Qty: 30, Satuan: "box", HargaSatuan: 30_000},
			{Deskripsi: "Biaya pengiriman area Jakarta Selatan", Qty: 4, Satuan: "trip", HargaSatuan: 50_000},
		},
	}
}

func TestInvoiceTotals_DiskonSebelumPPN(t *testing.T) {
	got := contohInvoice().Totals()
	// 60x75.000 + 30x30.000 + 4x50.000 = 4.500.000 + 900.000 + 200.000 = 5.600.000
	want := finance.InvoiceTotals{Subtotal: 5_600_000, Diskon: 200_000, DPP: 5_400_000, PPN: 594_000, Total: 5_994_000}
	if got != want {
		t.Errorf("Totals() = %+v, want %+v", got, want)
	}
}

// PPN harus dihitung dari DPP (setelah diskon), bukan dari subtotal — salah di sini membuat
// faktur memungut pajak lebih besar dari seharusnya.
func TestInvoiceTotals_PPNDariDPPBukanSubtotal(t *testing.T) {
	inv := contohInvoice()
	inv.DiskonRp = 1_600_000
	got := inv.Totals()
	if got.DPP != 4_000_000 {
		t.Fatalf("DPP = %d, want 4.000.000", got.DPP)
	}
	if got.PPN != 440_000 {
		t.Errorf("PPN = %d, want 440.000 (11%% dari DPP)", got.PPN)
	}
}

func TestInvoiceTotals_DiskonTidakBikinDPPNegatif(t *testing.T) {
	inv := contohInvoice()
	inv.DiskonRp = 99_000_000
	got := inv.Totals()
	if got.DPP != 0 || got.Total != 0 {
		t.Errorf("diskon melebihi subtotal: DPP %d total %d, keduanya harus 0", got.DPP, got.Total)
	}
}

func TestInvoiceTotals_NonPKPTanpaPPN(t *testing.T) {
	inv := contohInvoice()
	inv.PPNBps = 0
	got := inv.Totals()
	if got.PPN != 0 {
		t.Errorf("tanpa tarif PPN harus 0, got %d", got.PPN)
	}
	if got.Total != got.DPP {
		t.Errorf("total %d harus sama dengan DPP %d", got.Total, got.DPP)
	}
}

func TestTarifPPNBps_NonPKPSelaluNol(t *testing.T) {
	p := contohProfil()
	p.PKP = false
	if got := p.TarifPPNBps(); got != 0 {
		t.Errorf("non-PKP tidak boleh memungut PPN, got %d bps", got)
	}
	p.PKP, p.PPNBps = true, 0
	if got := p.TarifPPNBps(); got != finance.DefaultPPNBps {
		t.Errorf("PKP tanpa setelan harus pakai default %d, got %d", finance.DefaultPPNBps, got)
	}
	p.PPNBps = 1200
	if got := p.TarifPPNBps(); got != 1200 {
		t.Errorf("tarif dari profil harus dipakai, got %d", got)
	}
}

func TestRenderInvoicePDF_BerkasSahDanMemuatIsiPenting(t *testing.T) {
	var buf bytes.Buffer
	if err := finance.RenderInvoicePDF(&buf, contohInvoice(), contohProfil()); err != nil {
		t.Fatalf("RenderInvoicePDF: %v", err)
	}
	s := buf.String()
	if !strings.HasPrefix(s, "%PDF-1.4") {
		t.Fatal("bukan berkas PDF")
	}
	for _, want := range []string{
		"INVOICE", "INV-2026-061",
		"PO-KDM-2026-0317", // nomor PO pelanggan wajib tercetak bila ada
		"PT Karya Digital Mandiri",
		"10 Juni 2026", "10 Juli 2026", // tanggal dalam bahasa Indonesia
		"5.600.000", "594.000", "5.994.000", // subtotal, PPN, total
		"PPN 11%",
		"lima juta sembilan ratus sembilan puluh empat ribu rupiah", // terbilang
		"BCA", "Sari Wulandari",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("faktur tidak memuat %q", want)
		}
	}
}

// Tanpa nomor PO, labelnya tidak boleh ikut tercetak sebagai baris kosong.
func TestRenderInvoicePDF_TanpaPOTidakMencetakLabelnya(t *testing.T) {
	inv := contohInvoice()
	inv.NoPO = ""
	var buf bytes.Buffer
	if err := finance.RenderInvoicePDF(&buf, inv, contohProfil()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "No. PO") {
		t.Error("label No. PO tercetak padahal nomornya kosong")
	}
}

// Faktur dengan banyak baris harus pindah halaman, bukan menimpa blok total.
func TestRenderInvoicePDF_BanyakBarisPindahHalaman(t *testing.T) {
	inv := contohInvoice()
	inv.Items = nil
	for i := 0; i < 40; i++ {
		inv.Items = append(inv.Items, finance.InvoiceItem{
			Deskripsi: "Barang contoh nomor sekian", Qty: 1, Satuan: "pcs", HargaSatuan: 10_000,
		})
	}
	var buf bytes.Buffer
	if err := finance.RenderInvoicePDF(&buf, inv, contohProfil()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "/Count 2") {
		t.Error("40 baris harus memakai 2 halaman")
	}
}

// Profil minimal (baru onboarding, belum isi alamat/bank) tidak boleh bikin render gagal.
func TestRenderInvoicePDF_ProfilMinimalTetapJalan(t *testing.T) {
	var buf bytes.Buffer
	p := finance.BusinessProfile{NamaUsaha: "Warung Bu Sari"}
	inv := finance.Invoice{
		Nomor: "INV-2026-001", Status: finance.StatusDraft,
		Tanggal: "2026-01-05", JatuhTempo: "2026-02-05", PelangganNama: "Pak Budi",
		Items: []finance.InvoiceItem{{Deskripsi: "Nasi kotak", Qty: 10, HargaSatuan: 25_000}},
	}
	if err := finance.RenderInvoicePDF(&buf, inv, p); err != nil {
		t.Fatalf("profil minimal harus tetap bisa dicetak: %v", err)
	}
	if !strings.Contains(buf.String(), "250.000") {
		t.Error("total tidak tercetak")
	}
}

// Simpan satu contoh untuk diperiksa mata; hanya jalan bila diminta lewat env.
func TestRenderInvoicePDF_SimpanContoh(t *testing.T) {
	out := os.Getenv("INVOICE_PDF_OUT")
	if out == "" {
		t.Skip("set INVOICE_PDF_OUT untuk menyimpan contoh faktur")
	}
	var buf bytes.Buffer
	if err := finance.RenderInvoicePDF(&buf, contohInvoice(), contohProfil()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("contoh faktur ditulis ke %s", out)
}

// Faktur demo harus cocok dengan daftar Piutang & Utang di dashboard: dua tampilan tentang
// tagihan yang sama tidak boleh menyebut angka berbeda.
func TestSeedInvoices_CocokDenganObligations(t *testing.T) {
	piutang := map[string]int64{}
	for _, o := range finance.SeedKedaiKopiSenja("2026-08").Obligations {
		if o.Kind == "RECEIVABLE" {
			piutang[o.InvoiceNo] = o.Amount
		}
	}
	for _, inv := range finance.SeedInvoices() {
		want, ok := piutang[inv.Nomor]
		if !ok {
			t.Errorf("faktur %s tidak ada di daftar piutang dashboard", inv.Nomor)
			continue
		}
		if got := inv.Totals().Total; got != want {
			t.Errorf("faktur %s total %d, di daftar piutang %d", inv.Nomor, got, want)
		}
	}
}

func TestNextInvoiceNumber_MeneruskanDeret(t *testing.T) {
	list := finance.SeedInvoices()
	if got := finance.NextInvoiceNumber(list, "2026-09-01"); got != "INV-2026-090" {
		t.Errorf("nomor berikutnya = %q, want INV-2026-090", got)
	}
	// Ganti tahun: deret dimulai ulang supaya urutannya tetap terbaca.
	if got := finance.NextInvoiceNumber(list, "2027-01-03"); got != "INV-2027-001" {
		t.Errorf("tahun baru = %q, want INV-2027-001", got)
	}
	if got := finance.NextInvoiceNumber(nil, "2026-01-01"); got != "INV-2026-001" {
		t.Errorf("tanpa faktur sebelumnya = %q, want INV-2026-001", got)
	}
}

// Regresi: blok TERBILANG pernah dipatok mundur 70 pt dari akhir blok total, angka yang hanya
// pas ketika ada diskon DAN PPN. Pada faktur tanpa keduanya blok totalnya lebih pendek,
// sehingga terbilang naik 34 pt dan menabrak baris terakhir tabel rincian.
//
// Teksnya tetap ada saat itu, hanya posisinya yang salah — jadi yang diuji di sini posisinya.
func TestRenderInvoicePDF_TerbilangTidakMenabrakTabel(t *testing.T) {
	// Tiga bentuk kaki yang berbeda tingginya; ketiganya tidak boleh tumpang tindih.
	for _, c := range []struct {
		nama string
		ubah func(*finance.Invoice)
	}{
		{"tanpa diskon & tanpa PPN", func(inv *finance.Invoice) { inv.DiskonRp, inv.PPNBps = 0, 0 }},
		{"PPN saja", func(inv *finance.Invoice) { inv.DiskonRp = 0 }},
		{"diskon & PPN", func(inv *finance.Invoice) {}},
	} {
		inv := contohInvoice()
		c.ubah(&inv)

		var buf bytes.Buffer
		if err := finance.RenderInvoicePDF(&buf, inv, contohProfil()); err != nil {
			t.Fatalf("%s: %v", c.nama, err)
		}
		items := pdf.TextItemsFrom(buf.Bytes())

		terbilang, ok := pdf.Find(items, "TERBILANG")
		if !ok {
			t.Fatalf("%s: label TERBILANG tidak ditemukan", c.nama)
		}
		// Baris terakhir tabel rincian; deskripsinya diawali nomor urut.
		barisAkhir, ok := pdf.Find(items, "3. Biaya pengiriman")
		if !ok {
			t.Fatalf("%s: baris tabel terakhir tidak ditemukan", c.nama)
		}
		// Y dihitung dari atas: blok terbilang harus berada DI BAWAH baris tabel terakhir.
		if terbilang.Y <= barisAkhir.Y {
			t.Errorf("%s: TERBILANG di y=%.0f menabrak baris tabel di y=%.0f",
				c.nama, terbilang.Y, barisAkhir.Y)
		}
		// Dan harus sejajar baris pertama blok total, bukan melayang jauh di bawahnya.
		subtotal, ok := pdf.Find(items, "Subtotal")
		if !ok {
			t.Fatalf("%s: baris Subtotal tidak ditemukan", c.nama)
		}
		if beda := terbilang.Y - subtotal.Y; beda < -2 || beda > 2 {
			t.Errorf("%s: TERBILANG (y=%.0f) harus sejajar Subtotal (y=%.0f), selisih %.0f",
				c.nama, terbilang.Y, subtotal.Y, beda)
		}
	}
}

// Kolom angka harus rata kanan pada batas yang sama, bukan sekadar "ada".
func TestRenderInvoicePDF_KolomJumlahRataKanan(t *testing.T) {
	var buf bytes.Buffer
	if err := finance.RenderInvoicePDF(&buf, contohInvoice(), contohProfil()); err != nil {
		t.Fatal(err)
	}
	items := pdf.TextItemsFrom(buf.Bytes())
	// Tepi kanan = X + lebar teks, jadi font & ukurannya harus sesuai yang dipakai saat
	// menggambar: baris tabel dan subtotal Regular 9,5; baris TOTAL Bold 11.
	var kanan []float64
	for _, c := range []struct {
		angka string
		font  pdf.Font
		size  float64
	}{
		{"4.500.000", pdf.Regular, 9.5},
		{"900.000", pdf.Regular, 9.5},
		{"200.000", pdf.Regular, 9.5},
		{"5.600.000", pdf.Regular, 9.5},
		{"5.994.000", pdf.Bold, 11},
	} {
		it, ok := pdf.Find(items, c.angka)
		if !ok {
			t.Fatalf("angka %s tidak tercetak", c.angka)
		}
		kanan = append(kanan, it.X+pdf.Width(it.Text, c.font, c.size))
	}
	for i, x := range kanan {
		if beda := x - kanan[0]; beda < -3 || beda > 3 {
			t.Errorf("angka ke-%d berakhir di x=%.0f, kolom pertama di x=%.0f", i+1, x, kanan[0])
		}
	}
}
