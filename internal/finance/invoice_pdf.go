package finance

import (
	"fmt"
	"io"
	"strings"
	"time"

	"pajak/internal/pdf"
)

// Tata letak faktur, dalam titik PDF. Margin 40 pt (~1,4 cm) di kiri-kanan A4.
const (
	mLeft    = 40.0
	mRight   = pdf.A4Width - 40.0
	isiLebar = mRight - mLeft

	// Kolom tabel rincian, diukur dari tepi kanan supaya angka selalu rata kanan rapi.
	colJumlah = mRight       // rata kanan
	colHarga  = mRight - 95  // rata kanan
	colQty    = mRight - 190 // rata kanan
	colDesk   = mLeft + 6    // rata kiri
	deskLebar = colQty - 60 - colDesk

	barisTinggi = 17.0
	// kakiTinggi: ruang yang harus disisakan di bawah tabel untuk total, terbilang,
	// pembayaran, dan tanda tangan. Baris yang tidak muat pindah ke halaman berikutnya.
	kakiTinggi = 250.0
)

// RenderInvoicePDF menulis faktur sebagai PDF A4.
//
// Seluruh angka diambil dari inv.Totals(), bukan dihitung ulang di sini, supaya tampilan
// cetak tidak mungkin berbeda dari angka yang dipakai aplikasi.
func RenderInvoicePDF(w io.Writer, inv Invoice, p BusinessProfile) error {
	d := pdf.New()
	t := inv.Totals()

	y := kop(d, inv, p)
	y = pihak(d, inv, p, y)
	y = rincian(d, inv, y)
	kaki(d, inv, p, t, y)
	nomorHalaman(d, inv)

	return d.Write(w)
}

// kop menggambar identitas penerbit di kiri dan judul INVOICE di kanan.
func kop(d *pdf.Doc, inv Invoice, p BusinessProfile) float64 {
	nama := p.NamaUsaha
	if nama == "" {
		nama = "(Nama usaha belum diisi di Profil Usaha)"
	}
	d.Text(mLeft, 58, nama, pdf.Bold, 15, pdf.Black, pdf.Left)

	y := 74.0
	for _, baris := range identitasPenerbit(p) {
		for _, l := range pdf.WrapText(baris, pdf.Regular, 8.5, 250, 2) {
			d.Text(mLeft, y, l, pdf.Regular, 8.5, pdf.Gray, pdf.Left)
			y += 11
		}
	}

	d.Text(mRight, 58, "INVOICE", pdf.Bold, 22, pdf.Black, pdf.Right)

	// Label dan nilai sama-sama rata kanan pada dua kolom. Batas kolom label dihitung dari
	// nilai terpanjang, bukan dipatok, supaya nomor PO yang panjang tidak menabrak labelnya.
	baris := [][2]string{
		{"No. Invoice", inv.Nomor},
		{"No. PO", inv.NoPO}, // hanya tercetak bila pelanggan memberi nomor PO
		{"Tanggal", tanggalID(inv.Tanggal)},
		{"Jatuh Tempo", tanggalID(inv.JatuhTempo)},
	}
	nilaiLebar := 0.0
	for _, b := range baris {
		nilaiLebar = maxF(nilaiLebar, pdf.Width(b[1], pdf.Bold, 9))
	}
	labelKanan := mRight - nilaiLebar - 10

	ky := 76.0
	for _, b := range baris {
		if b[1] == "" {
			continue
		}
		d.Text(labelKanan, ky, b[0], pdf.Regular, 8.5, pdf.Gray, pdf.Right)
		d.Text(mRight, ky, b[1], pdf.Bold, 9, pdf.Black, pdf.Right)
		ky += 13
	}

	y = maxF(y, ky) + 8
	d.HLine(mLeft, mRight, y, 1.2, pdf.Black)
	return y + 22
}

func identitasPenerbit(p BusinessProfile) []string {
	var out []string
	if p.Alamat != "" {
		out = append(out, p.Alamat)
	}
	if kontak := gabungTakKosong(" · ", p.Telepon, p.Email); kontak != "" {
		out = append(out, kontak)
	}
	if p.NPWP != "" {
		out = append(out, "NPWP "+p.NPWP)
	}
	return out
}

// pihak menggambar blok "Tagihan Kepada" dan status faktur.
func pihak(d *pdf.Doc, inv Invoice, p BusinessProfile, y float64) float64 {
	d.Text(mLeft, y, "TAGIHAN KEPADA", pdf.Bold, 8, pdf.Gray, pdf.Left)
	ty := y + 15
	d.Text(mLeft, ty, inv.PelangganNama, pdf.Bold, 10.5, pdf.Black, pdf.Left)
	ty += 13
	if inv.PelangganAlamat != "" {
		for _, l := range pdf.WrapText(inv.PelangganAlamat, pdf.Regular, 9, 250, 3) {
			d.Text(mLeft, ty, l, pdf.Regular, 9, pdf.Gray, pdf.Left)
			ty += 11
		}
	}
	if inv.PelangganNPWP != "" {
		d.Text(mLeft, ty, "NPWP "+inv.PelangganNPWP, pdf.Regular, 9, pdf.Gray, pdf.Left)
		ty += 11
	}

	// Status di kanan: hanya yang perlu perhatian yang ditonjolkan.
	d.Text(mRight, y+6, strings.ToUpper(statusLabel(inv)), pdf.Bold, 10, warnaStatus(inv), pdf.Right)
	if p.PKP && inv.PPNBps > 0 {
		d.Text(mRight, y+21, "Pengusaha Kena Pajak", pdf.Regular, 8.5, pdf.Gray, pdf.Right)
	}
	return maxF(ty, y+34) + 14
}

func statusLabel(inv Invoice) string {
	if inv.Terlambat(time.Now()) {
		return "Lewat Jatuh Tempo"
	}
	switch inv.Status {
	case StatusLunas:
		return "Lunas"
	case StatusTerkirim:
		return "Belum Dibayar"
	}
	return "Draft"
}

func warnaStatus(inv Invoice) pdf.RGB {
	if inv.Terlambat(time.Now()) {
		return pdf.RGB{R: 0.80, G: 0.20, B: 0.25}
	}
	if inv.Status == StatusLunas {
		return pdf.RGB{R: 0.10, G: 0.50, B: 0.35}
	}
	return pdf.Gray
}

// rincian menggambar tabel baris faktur, pindah halaman bila tidak muat.
func rincian(d *pdf.Doc, inv Invoice, y float64) float64 {
	y = headerTabel(d, y)
	for i, it := range inv.Items {
		if y+barisTinggi > pdf.A4Height-kakiTinggi {
			d.AddPage()
			y = headerTabel(d, 60)
		}
		baris(d, i+1, it, y)
		y += barisTinggi
	}
	d.HLine(mLeft, mRight, y+2, 0.8, pdf.Black)
	return y + 2
}

func headerTabel(d *pdf.Doc, y float64) float64 {
	d.Rect(mLeft, y-11, isiLebar, 19, pdf.RGB{R: 0.94, G: 0.95, B: 0.96})
	d.Text(colDesk, y+2, "DESKRIPSI", pdf.Bold, 8, pdf.Gray, pdf.Left)
	d.Text(colQty, y+2, "QTY", pdf.Bold, 8, pdf.Gray, pdf.Right)
	d.Text(colHarga, y+2, "HARGA SATUAN", pdf.Bold, 8, pdf.Gray, pdf.Right)
	d.Text(colJumlah-6, y+2, "JUMLAH", pdf.Bold, 8, pdf.Gray, pdf.Right)
	return y + 23
}

func baris(d *pdf.Doc, no int, it InvoiceItem, y float64) {
	desk := fmt.Sprintf("%d. %s", no, it.Deskripsi)
	d.Text(colDesk, y, pdf.Truncate(desk, pdf.Regular, 9.5, deskLebar), pdf.Regular, 9.5, pdf.Black, pdf.Left)

	qty := fmt.Sprintf("%d", it.Qty)
	if it.Satuan != "" {
		qty += " " + it.Satuan
	}
	d.Text(colQty, y, qty, pdf.Regular, 9.5, pdf.Black, pdf.Right)
	d.Text(colHarga, y, rupiah(it.HargaSatuan), pdf.Regular, 9.5, pdf.Black, pdf.Right)
	d.Text(colJumlah-6, y, rupiah(it.Jumlah()), pdf.Regular, 9.5, pdf.Black, pdf.Right)
	d.HLine(mLeft, mRight, y+5, 0.4, pdf.Line)
}

// kaki menggambar total, terbilang, info pembayaran, catatan, dan tanda tangan.
func kaki(d *pdf.Doc, inv Invoice, p BusinessProfile, t InvoiceTotals, y float64) {
	y += 16
	kiriBatas := mRight - 210

	total := func(label, nilai string, tebal bool) {
		f, size := pdf.Regular, 9.5
		if tebal {
			f, size = pdf.Bold, 11
		}
		d.Text(kiriBatas, y, label, f, size, pdf.Black, pdf.Left)
		d.Text(colJumlah-6, y, nilai, f, size, pdf.Black, pdf.Right)
		y += 16
	}
	total("Subtotal", rupiah(t.Subtotal), false)
	if t.Diskon > 0 {
		total("Diskon", "-"+rupiah(t.Diskon), false)
		total("Dasar Pengenaan Pajak", rupiah(t.DPP), false)
	}
	if inv.PPNBps > 0 {
		total(fmt.Sprintf("PPN %s", persen(inv.PPNBps)), rupiah(t.PPN), false)
	}
	d.HLine(kiriBatas, mRight, y-4, 0.8, pdf.Black)
	y += 4
	total("TOTAL TAGIHAN", rupiah(t.Total), true)

	// Terbilang di kiri, sejajar blok total.
	ty := y - 70
	d.Text(mLeft, ty, "TERBILANG", pdf.Bold, 8, pdf.Gray, pdf.Left)
	ty += 13
	for _, l := range pdf.WrapText("# "+Terbilang(t.Total)+" #", pdf.Regular, 9, kiriBatas-mLeft-20, 3) {
		d.Text(mLeft, ty, l, pdf.Regular, 9, pdf.Black, pdf.Left)
		ty += 12
	}

	y = maxF(y, ty) + 14
	d.HLine(mLeft, mRight, y, 0.4, pdf.Line)
	y += 18

	if bayar := infoPembayaran(p); len(bayar) > 0 {
		d.Text(mLeft, y, "PEMBAYARAN", pdf.Bold, 8, pdf.Gray, pdf.Left)
		by := y + 13
		for _, l := range bayar {
			d.Text(mLeft, by, l, pdf.Regular, 9, pdf.Black, pdf.Left)
			by += 12
		}
		y = maxF(y, by)
	}

	if inv.Catatan != "" {
		d.Text(mLeft, y+6, "CATATAN", pdf.Bold, 8, pdf.Gray, pdf.Left)
		cy := y + 19
		for _, l := range pdf.WrapText(inv.Catatan, pdf.Regular, 9, 300, 4) {
			d.Text(mLeft, cy, l, pdf.Regular, 9, pdf.Gray, pdf.Left)
			cy += 12
		}
		y = maxF(y, cy)
	}

	tandaTangan(d, inv, p)
}

func infoPembayaran(p BusinessProfile) []string {
	if p.BankNama == "" && p.BankRekening == "" {
		return nil
	}
	var out []string
	if s := gabungTakKosong(" ", p.BankNama, p.BankRekening); s != "" {
		out = append(out, s)
	}
	if p.BankAtasNama != "" {
		out = append(out, "a.n. "+p.BankAtasNama)
	}
	return out
}

// tandaTangan ditempel di kanan bawah halaman terakhir, bukan mengalir bersama isi, supaya
// posisinya tetap sama di semua faktur.
func tandaTangan(d *pdf.Doc, inv Invoice, p BusinessProfile) {
	x := mRight - 80
	y := pdf.A4Height - 130
	d.Text(x, y, kotaTanggal(p, inv.Tanggal), pdf.Regular, 9, pdf.Black, pdf.Center)
	d.Text(x, y+13, "Hormat kami,", pdf.Regular, 9, pdf.Black, pdf.Center)
	d.HLine(x-75, x+75, y+62, 0.4, pdf.Line)
	nama := p.PenandatanganNama
	if nama == "" {
		nama = p.NamaUsaha
	}
	d.Text(x, y+74, nama, pdf.Bold, 9, pdf.Black, pdf.Center)
	if p.PenandatanganJabatan != "" {
		d.Text(x, y+86, p.PenandatanganJabatan, pdf.Regular, 8.5, pdf.Gray, pdf.Center)
	}
}

// kotaTanggal: "Jakarta, 10 Juni 2026". Kota ditebak dari baris terakhir alamat; bila alamat
// kosong, tanggalnya saja — lebih baik tanpa kota daripada menebak yang salah.
func kotaTanggal(p BusinessProfile, tanggal string) string {
	tgl := tanggalID(tanggal)
	if p.Alamat == "" {
		return tgl
	}
	bagian := strings.Split(p.Alamat, ",")
	kota := strings.TrimSpace(bagian[len(bagian)-1])
	// Buang kode pos di ujung, mis. "Jakarta Selatan 12190" → "Jakarta Selatan".
	kata := strings.Fields(kota)
	if n := len(kata); n > 1 && isDigits(kata[n-1]) {
		kota = strings.Join(kata[:n-1], " ")
	}
	if kota == "" {
		return tgl
	}
	return kota + ", " + tgl
}

func nomorHalaman(d *pdf.Doc, inv Invoice) {
	if d.PageCount() < 2 {
		return
	}
	// ponytail: nomor halaman hanya di halaman terakhir — Doc belum bisa menggambar ulang ke
	// halaman sebelumnya setelah berpindah. Cukup untuk faktur yang hampir selalu satu halaman.
	d.Text(mLeft, pdf.A4Height-40, inv.Nomor, pdf.Regular, 8, pdf.Gray, pdf.Left)
	d.Text(mRight, pdf.A4Height-40, fmt.Sprintf("Halaman %d", d.PageCount()), pdf.Regular, 8, pdf.Gray, pdf.Right)
}

var bulanID = [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember"}

// tanggalID: "2026-06-10" → "10 Juni 2026". Tanggal tak terbaca dikembalikan apa adanya.
func tanggalID(s string) string {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return s
	}
	return fmt.Sprintf("%d %s %d", t.Day(), bulanID[t.Month()-1], t.Year())
}

// rupiah memformat bilangan bulat dengan pemisah ribuan titik, tanpa awalan "Rp" supaya
// kolom angka tetap rapat; satuan mata uang sudah jelas dari konteks faktur.
func rupiah(n int64) string {
	neg := n < 0
	s := fmt.Sprintf("%d", abs(n))
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// persen: 1100 → "11%", 1050 → "10,5%".
func persen(bps int) string {
	if bps%100 == 0 {
		return fmt.Sprintf("%d%%", bps/100)
	}
	return strings.Replace(fmt.Sprintf("%.1f%%", float64(bps)/100), ".", ",", 1)
}

func gabungTakKosong(sep string, bagian ...string) string {
	var out []string
	for _, s := range bagian {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return strings.Join(out, sep)
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
