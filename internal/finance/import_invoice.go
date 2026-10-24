package finance

import (
	"cmp"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Impor invoice dari Excel: satu baris per item. Baris dengan nomor yang sama menjadi satu
// invoice; kolom invoice (tanggal, pelanggan, status, …) cukup diisi di baris pertamanya.

const sheetInvoice = "Invoice"

var invoiceHeaders = []string{"Nomor", "Tanggal", "Jatuh Tempo", "Pelanggan", "Alamat", "NPWP", "No. PO", "Status", "Tanggal Lunas",
	"Deskripsi", "Qty", "Satuan", "Harga Satuan", "Diskon (Rp)", "PPN (%)", "Catatan", "Barang"}

// InvoiceImpor: invoice hasil baca berkas beserta nomor baris Excel pertamanya (untuk pesan).
// Barang: isi kolom Barang tiap item (sejajar Invoice.Items; "" = kosong atau "jasa", yang sudah
// menjadi BukanBarang), dicocokkan ke daftar barang oleh server.
type InvoiceImpor struct {
	Baris   int      `json:"baris"`
	Invoice Invoice  `json:"invoice"`
	Barang  []string `json:"-"`
}

// InvoiceTemplate: template impor menu Invoice beserta panduannya.
func InvoiceTemplate() []ImportSheet {
	rows := [][]string{invoiceHeaders,
		{"INV-2026-101", "2026-09-01", "2026-09-30", "PT Karya Digital Mandiri", "Jl. Sudirman 10, Jakarta", "", "PO-778", "Terkirim", "", "Kopi literan 1L", "10", "botol", "120000", "", "", "Rapat bulanan", ""},
		{"INV-2026-101", "", "", "", "", "", "", "", "", "Croissant butter", "20", "pcs", "25000", "", "", "", ""},
		{"", "2026-09-05", "2026-09-05", "Coworking Space Kemang", "", "", "", "Lunas", "2026-09-05", "Katering rapat", "1", "paket", "1500000", "100000", "", "", "jasa"},
	}
	panduan := [][]string{
		{"Kolom", "Wajib?", "Catatan"},
		{"Nomor", "Opsional", "Baris bernomor sama digabung jadi satu invoice. Kosong + Pelanggan diisi = invoice baru bernomor otomatis (INV-tahun-urut); kosong + Pelanggan kosong = item tambahan invoice di atasnya."},
		{"Tanggal, Jatuh Tempo", "Wajib", "Di baris pertama invoice. Format 2026-09-01 atau 01/09/2026."},
		{"Pelanggan", "Wajib", "Di baris pertama invoice. Alamat, NPWP, No. PO, dan Catatan opsional."},
		{"Status", "Opsional", "Terkirim (bawaan bila kosong) = langsung masuk Penjualan & Piutang. Lunas = isi Tanggal Lunas. Draft = belum dibukukan."},
		{"Deskripsi, Qty, Harga Satuan", "Wajib", "Satu baris per item. Qty bilangan bulat; harga angka rupiah tanpa titik."},
		{"Diskon (Rp)", "Opsional", "Potongan untuk seluruh invoice, di baris pertama."},
		{"PPN (%)", "Opsional", "Kosong = tarif PPN usaha (0 bila non-PKP). Isi 0 bila invoice ini tanpa PPN."},
		{"Barang", "Opsional", "Per item: SKU atau nama barang persis seperti di Stok & Gudang (huruf besar/kecil bebas), atau jasa untuk item bukan barang. Metode perpetual: wajib diisi untuk invoice yang dikirim/lunas."},
		{"", "", ""},
		{"Nomor yang sudah ada di menu Invoice dan invoice di bulan yang sudah tutup buku dilewati; semuanya ditampilkan dulu sebelum diimpor.", "", ""},
	}
	return []ImportSheet{{Nama: sheetInvoice, Rows: rows}, {Nama: sheetPanduan, Rows: panduan}}
}

func kolomInvoice(h string) string {
	// Kolom barang hanya dari header persis "Barang" (sama dengan impor transaksi), supaya header
	// lama seperti "Nama Barang" tidak berubah arti.
	if strings.EqualFold(strings.TrimSpace(h), "barang") {
		return "barang"
	}
	n := norm(h)
	switch {
	case strings.Contains(n, "jatuh"):
		return "jt"
	case strings.Contains(n, "lunas"):
		return "tgl_lunas"
	case strings.Contains(n, "tanggal") || strings.Contains(n, "tgl"):
		return "tgl"
	case strings.Contains(n, "no po") || n == "po":
		return "po"
	case strings.Contains(n, "nomor") || n == "no" || strings.Contains(n, "no invoice") || strings.Contains(n, "no faktur"):
		return "nomor"
	case strings.Contains(n, "pelanggan") || strings.Contains(n, "customer"):
		return "pelanggan"
	case strings.Contains(n, "alamat"):
		return "alamat"
	case strings.Contains(n, "npwp"):
		return "npwp"
	case strings.Contains(n, "status"):
		return "status"
	case strings.Contains(n, "deskripsi") || strings.Contains(n, "uraian") || strings.Contains(n, "item"):
		return "deskripsi"
	case strings.Contains(n, "qty") || strings.Contains(n, "kuantitas"):
		return "qty"
	case strings.Contains(n, "harga"):
		return "harga"
	case strings.Contains(n, "satuan"):
		return "satuan"
	case strings.Contains(n, "diskon") || strings.Contains(n, "potongan"):
		return "diskon"
	case strings.Contains(n, "ppn"):
		return "ppn"
	case strings.Contains(n, "catatan") || strings.Contains(n, "keterangan"):
		return "catatan"
	}
	return ""
}

// ParseInvoiceFile membaca sheet invoice. ppnBawaan: tarif PPN usaha (basis poin) untuk kolom
// PPN yang kosong. Invoice dengan satu baris bermasalah dilewati utuh (dilaporkan), supaya tidak
// ada invoice yang masuk dengan item kurang.
func ParseInvoiceFile(sheets []ImportSheet, ppnBawaan int) ([]InvoiceImpor, []SheetError, string) {
	sh, ok := pilihSheet(sheets, sheetInvoice)
	if !ok {
		return nil, sheetTidakDitemukan(sheetInvoice), ""
	}
	list, errs := parseInvoiceRows(sh.Rows, ppnBawaan)
	return list, tag(sh.Nama, errs), sh.Nama
}

func parseInvoiceRows(rows [][]string, ppnBawaan int) ([]InvoiceImpor, []ImportError) {
	b, ok := bacaRinci(rows, kolomInvoice, "pelanggan", "deskripsi", "qty", "harga")
	if !ok {
		return nil, []ImportError{{0, "header tidak ditemukan: butuh kolom Pelanggan, Deskripsi, Qty, dan Harga Satuan"}}
	}
	var out []InvoiceImpor
	var errs gagal
	rusak := map[int]bool{}
	for i := b.header + 1; i < len(rows); i++ {
		if b.kosong(i, "nomor", "tgl", "jt", "pelanggan", "deskripsi", "qty", "harga") {
			continue
		}
		nomor, pelanggan := b.get(i, "nomor"), b.get(i, "pelanggan")
		cur := len(out) - 1
		if cur < 0 || (nomor != "" && nomor != out[cur].Invoice.Nomor) || (nomor == "" && pelanggan != "") {
			inv, pesan := kepalaInvoice(b, i, ppnBawaan)
			out = append(out, InvoiceImpor{Baris: i + 1, Invoice: inv})
			cur = len(out) - 1
			if pesan != "" {
				errs.add(i, "%s", pesan)
				rusak[cur] = true
			}
		} else if pelanggan != "" && !strings.EqualFold(pelanggan, out[cur].Invoice.PelangganNama) {
			errs.add(i, "pelanggan %q berbeda dengan baris pertama invoice ini (%q); kolom invoice cukup diisi di baris pertama", pelanggan, out[cur].Invoice.PelangganNama)
			rusak[cur] = true
			continue
		}
		it, pesan := itemInvoice(b, i)
		if pesan != "" {
			errs.add(i, "%s", pesan)
			rusak[cur] = true
			continue
		}
		teks := b.get(i, "barang")
		if JasaTeks(teks) {
			it.BukanBarang, teks = true, ""
		}
		out[cur].Invoice.Items = append(out[cur].Invoice.Items, it)
		out[cur].Barang = append(out[cur].Barang, teks)
	}
	hasil := out[:0]
	for k, x := range out {
		if rusak[k] {
			continue
		}
		if err := ValidateInvoice(&x.Invoice); err != nil {
			errs.add(x.Baris-1, "invoice %s: %v", cmp.Or(x.Invoice.Nomor, x.Invoice.PelangganNama), err)
			continue
		}
		hasil = append(hasil, x)
	}
	return hasil, errs
}

func kepalaInvoice(b barisRinci, i, ppnBawaan int) (Invoice, string) {
	inv := Invoice{Nomor: b.get(i, "nomor"), NoPO: b.get(i, "po"), PelangganNama: b.get(i, "pelanggan"),
		PelangganAlamat: b.get(i, "alamat"), PelangganNPWP: b.get(i, "npwp"), Catatan: b.get(i, "catatan"), PPNBps: ppnBawaan}
	var ok bool
	if inv.Tanggal, ok = parseTanggal(b.get(i, "tgl")); !ok {
		return inv, fmt.Sprintf("Tanggal %q tidak dikenali (isi tanggal terbit, mis. 2026-09-01)", b.get(i, "tgl"))
	}
	if inv.JatuhTempo, ok = parseTanggal(b.get(i, "jt")); !ok {
		return inv, fmt.Sprintf("Jatuh Tempo %q tidak dikenali (mis. 2026-09-30)", b.get(i, "jt"))
	}
	switch s := norm(b.get(i, "status")); {
	case s == "" || strings.Contains(s, "belum") || strings.Contains(s, "kirim"):
		inv.Status = StatusTerkirim
	case strings.Contains(s, "draf"):
		inv.Status = StatusDraft
	case strings.Contains(s, "lunas"):
		inv.Status = StatusLunas
		if inv.TanggalLunas, ok = parseTanggal(b.get(i, "tgl_lunas")); !ok {
			return inv, "invoice lunas: isi Tanggal Lunas (tanggal pelanggan membayar)"
		}
	default:
		return inv, fmt.Sprintf("Status %q tidak dikenal; isi Terkirim, Lunas, atau Draft", b.get(i, "status"))
	}
	if v := b.get(i, "diskon"); v != "" {
		n, ok := parseRupiah(v)
		if !ok || n < 0 {
			return inv, fmt.Sprintf("Diskon %q bukan angka rupiah", v)
		}
		inv.DiskonRp = n
	}
	if v := b.get(i, "ppn"); v != "" {
		f, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(v), "%")), ",", "."), 64)
		if f > 0 && f < 1 && !strings.HasSuffix(strings.TrimSpace(v), "%") {
			f *= 100 // sel Excel berformat persen tersimpan sebagai pecahan (0,11 = 11%)
		}
		if err != nil || f < 0 || f > 100 {
			return inv, fmt.Sprintf("PPN %q bukan persen (mis. 11 atau 0)", v)
		}
		inv.PPNBps = int(math.Round(f * 100))
	}
	return inv, ""
}

func itemInvoice(b barisRinci, i int) (InvoiceItem, string) {
	it := InvoiceItem{Deskripsi: b.get(i, "deskripsi"), Satuan: b.get(i, "satuan")}
	if it.Deskripsi == "" {
		return it, "Deskripsi item wajib diisi"
	}
	q, err := strconv.ParseFloat(strings.ReplaceAll(b.get(i, "qty"), ",", "."), 64)
	if err != nil || q <= 0 || q != math.Trunc(q) {
		return it, fmt.Sprintf("jumlah (qty) %q harus bilangan bulat lebih dari 0", b.get(i, "qty"))
	}
	it.Qty = int64(q)
	h, ok := parseRupiah(b.get(i, "harga"))
	if !ok || h < 0 {
		return it, fmt.Sprintf("Harga Satuan %q bukan angka rupiah", b.get(i, "harga"))
	}
	it.HargaSatuan = h
	return it, ""
}
