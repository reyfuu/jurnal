package finance

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Invoice: faktur penjualan yang diterbitkan ke pelanggan.
//
// Nilai uang disimpan per baris (qty x harga satuan) dan totalnya selalu dihitung ulang oleh
// Totals, tidak pernah disimpan. Yang disimpan hanya yang tidak bisa diturunkan: tarif PPN saat
// terbit, supaya faktur lama tidak ikut berubah ketika tarif di profil usaha diganti.
//
// ponytail: belum ada pelunasan parsial, nota kredit, maupun mata uang selain Rupiah.
type Invoice struct {
	ID         int    `json:"id"`
	Nomor      string `json:"nomor"`
	NoPO       string `json:"no_po,omitempty"` // nomor PO pelanggan, dicetak di faktur bila ada
	Status     string `json:"status"`          // draft | terkirim | lunas
	Tanggal    string `json:"tanggal"`         // YYYY-MM-DD, tanggal terbit
	JatuhTempo string `json:"jatuh_tempo"`     // YYYY-MM-DD

	PelangganNama   string `json:"pelanggan_nama"`
	PelangganAlamat string `json:"pelanggan_alamat,omitempty"`
	PelangganNPWP   string `json:"pelanggan_npwp,omitempty"`

	Items    []InvoiceItem `json:"items"`
	DiskonRp int64         `json:"diskon_rp"`
	// PPNBps: tarif PPN saat faktur terbit, dalam basis poin (1100 = 11%). 0 = tanpa PPN,
	// yang benar untuk usaha non-PKP — mereka memang tidak boleh memungut PPN.
	PPNBps  int    `json:"ppn_bps"`
	Catatan string `json:"catatan,omitempty"`

	// TanggalLunas: tanggal pelanggan membayar (status lunas).
	TanggalLunas string `json:"tanggal_lunas,omitempty"`
	// Dibukukan: invoice ini tercatat di pembukuan — status terkirim/lunas otomatis membuat
	// transaksi Penjualan (DPP) dan PPN Keluaran. Invoice lama (sebelum fitur ini) false sampai
	// pengguna memilih "Bukukan", supaya tidak dobel dengan penjualan yang dicatat manual.
	Dibukukan      bool `json:"dibukukan,omitempty"`
	TransaksiID    int  `json:"transaksi_id,omitempty"`
	TransaksiPPNID int  `json:"transaksi_ppn_id,omitempty"`
}

type InvoiceItem struct {
	Deskripsi   string `json:"deskripsi"`
	Qty         int64  `json:"qty"`
	Satuan      string `json:"satuan,omitempty"` // mis. "pcs", "box", "jam"
	HargaSatuan int64  `json:"harga_satuan"`
}

func (it InvoiceItem) Jumlah() int64 { return it.Qty * it.HargaSatuan }

// InvoiceTotals: seluruh angka kaki faktur, selalu diturunkan dari baris.
type InvoiceTotals struct {
	Subtotal int64 `json:"subtotal"`
	Diskon   int64 `json:"diskon"`
	DPP      int64 `json:"dpp"` // dasar pengenaan pajak = subtotal - diskon
	PPN      int64 `json:"ppn"`
	Total    int64 `json:"total"`
}

func (inv Invoice) Totals() InvoiceTotals {
	var t InvoiceTotals
	for _, it := range inv.Items {
		t.Subtotal += it.Jumlah()
	}
	t.Diskon = min(inv.DiskonRp, t.Subtotal) // diskon tidak boleh membuat DPP negatif
	t.DPP = t.Subtotal - t.Diskon
	t.PPN = t.DPP * int64(inv.PPNBps) / 10_000
	t.Total = t.DPP + t.PPN
	return t
}

// Terlambat: faktur belum lunas dan sudah lewat jatuh tempo pada tanggal acuan.
func (inv Invoice) Terlambat(acuan time.Time) bool {
	if inv.Status == StatusLunas {
		return false
	}
	jt, err := time.Parse(time.DateOnly, inv.JatuhTempo)
	return err == nil && acuan.After(jt)
}

const (
	StatusDraft    = "draft"
	StatusTerkirim = "terkirim"
	StatusLunas    = "lunas"
)

var StatusInvoice = []string{StatusDraft, StatusTerkirim, StatusLunas}

func ValidateInvoice(inv *Invoice) error {
	for _, s := range []*string{
		&inv.Nomor, &inv.NoPO, &inv.Status, &inv.Tanggal, &inv.JatuhTempo,
		&inv.PelangganNama, &inv.PelangganAlamat, &inv.PelangganNPWP, &inv.Catatan,
	} {
		*s = strings.TrimSpace(*s)
	}
	if inv.Status == "" {
		inv.Status = StatusDraft
	}
	if !contains(StatusInvoice, inv.Status) {
		return errors.New("status faktur harus draft, terkirim, atau lunas")
	}
	if inv.PelangganNama == "" {
		return errors.New("nama pelanggan wajib diisi")
	}
	terbit, err := time.Parse(time.DateOnly, inv.Tanggal)
	if err != nil {
		return errors.New("tanggal terbit wajib diisi (format YYYY-MM-DD)")
	}
	jt, err := time.Parse(time.DateOnly, inv.JatuhTempo)
	if err != nil {
		return errors.New("tanggal jatuh tempo wajib diisi (format YYYY-MM-DD)")
	}
	if jt.Before(terbit) {
		return errors.New("jatuh tempo tidak boleh mendahului tanggal terbit")
	}
	if len(inv.Items) == 0 {
		return errors.New("faktur harus punya minimal satu baris rincian")
	}
	inv.TanggalLunas = strings.TrimSpace(inv.TanggalLunas)
	if inv.Status != StatusLunas {
		inv.TanggalLunas = ""
	} else if lunas, err := time.Parse(time.DateOnly, inv.TanggalLunas); err != nil {
		return errors.New("isi tanggal pelanggan membayar untuk faktur lunas (format YYYY-MM-DD)")
	} else if lunas.Before(terbit) {
		return errors.New("tanggal bayar tidak boleh sebelum tanggal terbit faktur")
	}
	for i := range inv.Items {
		it := &inv.Items[i]
		it.Deskripsi = strings.TrimSpace(it.Deskripsi)
		it.Satuan = strings.TrimSpace(it.Satuan)
		if it.Deskripsi == "" {
			return fmt.Errorf("baris %d: deskripsi wajib diisi", i+1)
		}
		if it.Qty <= 0 {
			return fmt.Errorf("baris %d: qty harus lebih dari 0", i+1)
		}
		if it.HargaSatuan < 0 {
			return fmt.Errorf("baris %d: harga satuan tidak boleh negatif", i+1)
		}
	}
	if inv.DiskonRp < 0 {
		return errors.New("diskon tidak boleh negatif")
	}
	if inv.PPNBps < 0 || inv.PPNBps > 10_000 {
		return errors.New("tarif PPN tidak masuk akal")
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// NextInvoiceNumber meneruskan deret yang sudah ada, mis. INV-2026-061 → INV-2026-062.
// Penomoran dipisah per tahun terbit supaya urutannya tetap terbaca saat ganti tahun buku.
func NextInvoiceNumber(existing []Invoice, tanggal string) string {
	tahun := time.Now().Format("2006")
	if t, err := time.Parse(time.DateOnly, tanggal); err == nil {
		tahun = t.Format("2006")
	}
	prefix := "INV-" + tahun + "-"
	urut := 0
	for _, inv := range existing {
		if !strings.HasPrefix(inv.Nomor, prefix) {
			continue
		}
		if n, err := strconv.Atoi(strings.TrimPrefix(inv.Nomor, prefix)); err == nil && n > urut {
			urut = n
		}
	}
	return fmt.Sprintf("%s%03d", prefix, urut+1)
}

// InvoiceSummary: daftar faktur plus rekap yang dipakai halaman Invoice.
type InvoiceSummary struct {
	Invoices        []InvoiceRow `json:"invoices"`
	TotalTagihan    int64        `json:"total_tagihan"` // seluruh faktur terkirim & lunas
	BelumDibayar    int64        `json:"belum_dibayar"` // terkirim, belum lunas
	Terlambat       int64        `json:"terlambat"`     // belum lunas & lewat jatuh tempo
	JumlahTerlambat int          `json:"jumlah_terlambat"`
}

// InvoiceRow menyertakan angka turunan supaya UI tidak menghitung ulang dan berisiko beda.
type InvoiceRow struct {
	Invoice
	Totals        InvoiceTotals `json:"totals"`
	HariTerlambat int           `json:"hari_terlambat"`
}

func SummarizeInvoices(list []Invoice, acuan time.Time) InvoiceSummary {
	s := InvoiceSummary{Invoices: make([]InvoiceRow, 0, len(list))}
	for _, inv := range list {
		t := inv.Totals()
		row := InvoiceRow{Invoice: inv, Totals: t}
		if inv.Terlambat(acuan) {
			jt, _ := time.Parse(time.DateOnly, inv.JatuhTempo)
			row.HariTerlambat = int(acuan.Sub(jt).Hours() / 24)
			s.Terlambat += t.Total
			s.JumlahTerlambat++
		}
		if inv.Status != StatusDraft {
			s.TotalTagihan += t.Total
		}
		if inv.Status == StatusTerkirim {
			s.BelumDibayar += t.Total
		}
		s.Invoices = append(s.Invoices, row)
	}
	return s
}

// SeedInvoices: faktur demo yang sengaja dibuat cocok dengan Obligations di seed dashboard —
// nomor dan nilainya sama, supaya daftar Piutang & Utang dan daftar Invoice tidak bercerita
// berbeda tentang tagihan yang sama.
//
// Nilai di Obligations adalah total tagihan, jadi baris di sini disusun agar totalnya persis
// sama. Dua yang pertama non-PPN (pelanggan demo bukan PKP) supaya subtotal = total.
func SeedInvoices() []Invoice {
	return []Invoice{
		{
			ID: 1, Nomor: "INV-2026-061", NoPO: "PO-KDM-2026-0317", Status: StatusTerkirim,
			Tanggal: "2026-06-10", JatuhTempo: "2026-07-10",
			PelangganNama:   "PT Karya Digital Mandiri",
			PelangganAlamat: "Gedung Menara Rajawali Lt. 8, Jl. DR. Ide Anak Agung Gde Agung, Kuningan, Jakarta Selatan 12950",
			PelangganNPWP:   "02.345.678.9-012.000",
			Catatan:         "Kopi rapat bulanan, pengiriman setiap Senin & Kamis.",
			Items: []InvoiceItem{
				{Deskripsi: "Kopi rapat bulanan - paket 20 cup", Qty: 60, Satuan: "cup", HargaSatuan: 75_000},
				{Deskripsi: "Biaya pengiriman area Jakarta Selatan", Qty: 18, Satuan: "trip", HargaSatuan: 50_000},
			},
		},
		{
			ID: 2, Nomor: "INV-2026-068", Status: StatusTerkirim,
			Tanggal: "2026-06-22", JatuhTempo: "2026-07-22",
			PelangganNama:   "Event Organizer Senayan Fest",
			PelangganAlamat: "Jl. Asia Afrika No. 8, Gelora, Jakarta Pusat 10270",
			Catatan:         "Coffee booth Senayan Fest 2026, 3 hari acara.",
			Items: []InvoiceItem{
				{Deskripsi: "Sewa coffee booth 3 hari (barista + alat)", Qty: 3, Satuan: "hari", HargaSatuan: 1_200_000},
				{Deskripsi: "Es kopi susu Senja 12oz", Qty: 40, Satuan: "cup", HargaSatuan: 15_000},
			},
		},
		{
			ID: 3, Nomor: "INV-2026-089", NoPO: "PO-CWK-0912", Status: StatusTerkirim,
			Tanggal: "2026-08-05", JatuhTempo: "2026-09-05",
			PelangganNama:   "Coworking Space Kemang",
			PelangganAlamat: "Jl. Kemang Raya No. 12, Jakarta Selatan 12730",
			Items: []InvoiceItem{
				{Deskripsi: "Biji kopi Arabica Gayo roasted 1kg", Qty: 4, Satuan: "kg", HargaSatuan: 275_000},
				{Deskripsi: "Dispenser cold brew 5L refill", Qty: 6, Satuan: "galon", HargaSatuan: 50_000},
			},
		},
	}
}
