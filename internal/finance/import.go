package finance

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ImportError menunjuk baris Excel (1-based) yang gagal; Baris 0 = masalah header.
type ImportError struct {
	Baris int    `json:"baris"`
	Pesan string `json:"pesan"`
}

var importHeaders = []string{"Tanggal", "Jenis", "Nominal", "Keterangan", "Produk / Bahan", "Jumlah", "Channel", "Pelanggan / Supplier", "Status Bayar", "Tanggal Lunas", "Barang", "jumlah_barang"}

// TemplateRows: header + contoh untuk template Excel; contohnya harus lolos ParseImport.
func TemplateRows() [][]string {
	return [][]string{
		importHeaders,
		{"2026-08-01", "Penjualan", "1250000", "Rekap kasir", "Es Kopi Susu", "50", "Dine-in", "", "Lunas", "", "Es Kopi Susu", "50"},
		{"2026-08-02", "Beli Bahan / Stok", "850000", "Belanja susu", "Fresh Milk 1L", "40", "", "Toko Susu Segar", "Belum Lunas", "", "Fresh Milk 1L", "40"},
		{"2026-08-28", "Gaji & Upah", "12500000", "Gaji barista", "", "", "", "", "Lunas", "", "", ""},
	}
}

// JenisSheetRows: daftar jenis untuk sheet bantuan di template.
func JenisSheetRows() [][]string {
	rows := [][]string{{"Jenis", "Arah uang", "Contoh kata yang juga dikenali"}}
	for _, j := range JenisList {
		if j.Pos == PosPindah { // hanya lewat Catat (perlu rekening asal & tujuan)
			continue
		}
		rows = append(rows, []string{j.Nama, j.Arah, strings.Join(jenisAlias[j.Nama], ", ")})
	}
	return rows
}

var jenisAlias = map[string][]string{
	"Penjualan":                {"jual", "jualan", "omzet", "omset", "pemasukan", "pendapatan usaha"},
	"Pendapatan Lain":          {"pendapatan lain lain", "bunga bank", "jual barang bekas"},
	"Setor Modal":              {"modal", "tambah modal"},
	"Terima Pinjaman":          {"pinjaman", "utang masuk", "kur"},
	"Beli Bahan / Stok":        {"beli bahan", "bahan baku", "beli stok", "belanja bahan", "kulakan", "hpp"},
	"Kemasan":                  {"packaging", "plastik", "cup"},
	"Potongan Aplikasi":        {"komisi", "potongan", "fee aplikasi", "admin marketplace"},
	"Ongkir & Transport":       {"ongkir", "transport", "bensin", "kurir"},
	"Gaji & Upah":              {"gaji", "upah", "thr", "bonus", "lembur"},
	"Sewa Tempat":              {"sewa", "kontrakan"},
	"Listrik, Air, Internet":   {"listrik", "air", "internet", "wifi", "pulsa"},
	"Iklan & Promosi":          {"iklan", "promosi", "ads", "marketing"},
	"Perawatan & Perlengkapan": {"perawatan", "servis", "perlengkapan", "atk"},
	"Pajak & Izin":             {"pajak", "izin", "retribusi"},
	JenisSetorPPhFinal:         {"pph final umkm", "setor pph final"},
	"Biaya Lain":               {"lain lain", "lainnya"},
	"Beli Alat / Aset":         {"beli alat", "aset", "peralatan"},
	"Bayar Cicilan":            {"cicilan", "angsuran", "bayar utang"},
	"Ambil Uang Pribadi":       {"prive", "ambil pribadi", "uang pribadi"},
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func norm(s string) string {
	return strings.TrimSpace(nonAlnum.ReplaceAllString(strings.ToLower(s), " "))
}

func findJenis(s string) (string, bool) {
	q := norm(s)
	if q == "" {
		return "", false
	}
	for _, j := range JenisList {
		if norm(j.Nama) == q {
			return j.Nama, true
		}
	}
	for nama, aliases := range jenisAlias {
		for _, a := range aliases {
			if a == q {
				return nama, true
			}
		}
	}
	for _, j := range JenisList {
		if n := norm(j.Nama); strings.HasPrefix(n, q) || strings.HasPrefix(q, n) {
			return j.Nama, true
		}
	}
	return "", false
}

// column menebak arti header; urutan cek penting ("tanggal lunas" sebelum "tanggal", "nominal" sebelum "jumlah").
func column(h string) string {
	// Kolom barang hanya dikenali dari header persis template ("Barang", "jumlah_barang") supaya
	// berkas lama dengan header seperti "Nama Barang" / "Jumlah Barang" tetap terbaca seperti dulu.
	switch strings.ToLower(strings.TrimSpace(h)) {
	case "barang":
		return "barang"
	case "jumlah_barang":
		return "jumlah_barang"
	}
	h = norm(h)
	switch {
	case strings.Contains(h, "tanggal") && strings.Contains(h, "lunas"):
		return "tanggal_lunas"
	case strings.Contains(h, "tanggal") || h == "tgl":
		return "tanggal"
	case strings.Contains(h, "jenis") || strings.Contains(h, "kategori"):
		return "jenis"
	case strings.Contains(h, "nominal") || strings.Contains(h, "rp") || strings.Contains(h, "jumlah uang"):
		return "nominal"
	case strings.Contains(h, "keterangan") || strings.Contains(h, "catatan"):
		return "keterangan"
	case strings.Contains(h, "produk") || strings.Contains(h, "bahan") || strings.Contains(h, "item"):
		return "item"
	case strings.Contains(h, "jumlah") || strings.Contains(h, "qty"):
		return "qty"
	case strings.Contains(h, "channel"):
		return "channel"
	case strings.Contains(h, "pelanggan") || strings.Contains(h, "supplier") || strings.Contains(h, "pihak"):
		return "pihak"
	case strings.Contains(h, "status"):
		return "status"
	}
	return ""
}

var excelEpoch = time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)

func parseTanggal(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil && f > 1 && f < 2_958_466 { // nomor seri tanggal Excel
		return excelEpoch.AddDate(0, 0, int(f)).Format(time.DateOnly), true
	}
	for _, layout := range []string{time.DateOnly, "2006/01/02", "02/01/2006", "2/1/2006", "02-01-2006", "2-1-2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format(time.DateOnly), true
		}
	}
	return "", false
}

var plainNumber = regexp.MustCompile(`^\d+(\.\d+)?$`)

// parseRupiah menerima "1250000", "Rp 1.250.000", "1.250.000,00", dan angka sel Excel "1250000.5".
func parseRupiah(s string) (int64, bool) {
	s = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "Rp"))
	s = strings.ReplaceAll(s, " ", "")
	if plainNumber.MatchString(s) && !regexp.MustCompile(`\.\d{3}$`).MatchString(s) {
		f, err := strconv.ParseFloat(s, 64)
		return int64(math.Round(f)), err == nil
	}
	if i := strings.LastIndex(s, ","); i >= 0 && len(s)-i <= 3 { // desimal Indonesia ",00"
		s = s[:i]
	}
	s = strings.NewReplacer(".", "", ",", "").Replace(s)
	n, err := strconv.ParseInt(s, 10, 64)
	return n, err == nil && s != ""
}

// ImportRow: satu baris sheet yang lolos baca, dengan teks kolom barang apa adanya. Pencocokan ke
// daftar barang dilakukan server. JumlahBarang per seribu satuan (TakaranSkala); AdaJumlah = sel
// jumlah barang terisi.
type ImportRow struct {
	Baris        int // nomor baris Excel (1-based), sama dengan ImportError.Baris
	Tx           Transaction
	BarangTeks   string
	JumlahBarang int64
	AdaJumlah    bool
}

// JasaTeks: isi kolom barang yang berarti transaksi ini bukan barang persediaan (BukanBarang).
func JasaTeks(s string) bool { return strings.EqualFold(strings.TrimSpace(s), "jasa") }

var (
	ribuanTitik = regexp.MustCompile(`^\d{1,3}(\.\d{3})+$`)
	desimalKoma = regexp.MustCompile(`^(\d+|\d{1,3}(\.\d{3})+),(\d{1,3})$`)
	desimalTtk  = regexp.MustCompile(`^\d+\.\d{1,3}$`)
	hanyaAngka  = regexp.MustCompile(`^\d+$`)
)

// parseJumlahBarang membaca jumlah desimal menjadi per seribu (TakaranSkala), aturan sama dengan
// parseRupiah: bila ada "," maka "," adalah desimal dan "." pemisah ribuan ("1.250,5" = 1250,5);
// bila hanya "." maka ribuan jika berpola 1-3 angka lalu kelompok tiga angka ("1.250" = 1250),
// selain itu desimal ("2.5", "1.25" dari sel angka Excel). Campuran gaya "1,250.5" ditolak.
// Maksimum tiga angka desimal; harus lebih dari 0.
func parseJumlahBarang(s string) (int64, bool) {
	s = strings.ReplaceAll(strings.TrimSpace(s), " ", "")
	switch {
	case hanyaAngka.MatchString(s):
	case ribuanTitik.MatchString(s):
		s = strings.ReplaceAll(s, ".", "")
	case desimalKoma.MatchString(s):
		s = strings.Replace(strings.ReplaceAll(s, ".", ""), ",", ".", 1)
	case desimalTtk.MatchString(s):
	default:
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f <= 0 || f > 1e12 {
		return 0, false
	}
	q := int64(math.Round(f * TakaranSkala))
	return q, q > 0
}

// ParseImport mengubah baris sheet menjadi transaksi valid; baris bermasalah dilaporkan, tidak ikut.
// Kolom barang (opsional) diabaikan di sini; pakai ParseImportBaris untuk membawanya.
func ParseImport(rows [][]string) ([]Transaction, []ImportError) {
	baris, errs := ParseImportBaris(rows)
	var txs []Transaction
	for _, r := range baris {
		txs = append(txs, r.Tx)
	}
	return txs, errs
}

// ParseImportBaris: seperti ParseImport tetapi membawa nomor baris dan teks kolom barang &
// jumlah barang (opsional) untuk dicocokkan server.
func ParseImportBaris(rows [][]string) ([]ImportRow, []ImportError) {
	headerAt, cols := -1, map[string]int{}
	for i := 0; i < len(rows) && i < 10 && headerAt < 0; i++ {
		m := map[string]int{}
		for c, h := range rows[i] {
			if k := column(h); k != "" {
				if _, dup := m[k]; !dup {
					m[k] = c
				}
			}
		}
		if _, a := m["tanggal"]; a {
			if _, b := m["jenis"]; b {
				if _, c := m["nominal"]; c {
					headerAt, cols = i, m
				}
			}
		}
	}
	if headerAt < 0 {
		return nil, []ImportError{{0, "header tidak ditemukan: butuh kolom Tanggal, Jenis, dan Nominal (pakai template)"}}
	}

	var out []ImportRow
	var errs []ImportError
	for i := headerAt + 1; i < len(rows); i++ {
		row := rows[i]
		get := func(k string) string {
			if c, ok := cols[k]; ok && c < len(row) {
				return strings.TrimSpace(row[c])
			}
			return ""
		}
		if strings.TrimSpace(strings.Join(row, "")) == "" {
			continue
		}
		fail := func(format string, a ...any) { errs = append(errs, ImportError{i + 1, fmt.Sprintf(format, a...)}) }

		t := Transaction{Keterangan: get("keterangan"), Item: get("item"), Channel: get("channel"), Pihak: get("pihak")}
		var ok bool
		if t.Tanggal, ok = parseTanggal(get("tanggal")); !ok {
			fail("tanggal %q tidak dikenali (pakai YYYY-MM-DD atau DD/MM/YYYY)", get("tanggal"))
			continue
		}
		if t.Jenis, ok = findJenis(get("jenis")); !ok {
			fail("jenis %q tidak dikenal; lihat sheet Daftar Jenis", get("jenis"))
			continue
		}
		if t.Nominal, ok = parseRupiah(get("nominal")); !ok {
			fail("nominal %q bukan angka", get("nominal"))
			continue
		}
		if q := get("qty"); q != "" {
			n, ok := parseRupiah(q)
			if !ok {
				fail("jumlah %q bukan angka", q)
				continue
			}
			t.Qty = int(n)
		}
		st := norm(get("status"))
		t.Lunas = !(strings.Contains(st, "belum") || strings.Contains(st, "tempo") || strings.Contains(st, "utang"))
		if tl := get("tanggal_lunas"); tl != "" {
			if t.TanggalLunas, ok = parseTanggal(tl); !ok {
				fail("tanggal lunas %q tidak dikenali", tl)
				continue
			}
		}
		r := ImportRow{Baris: i + 1, BarangTeks: get("barang")}
		if q := get("jumlah_barang"); q != "" {
			if r.JumlahBarang, ok = parseJumlahBarang(q); !ok {
				fail("jumlah barang %q bukan angka lebih dari 0 (maksimal 3 desimal)", q)
				continue
			}
			r.AdaJumlah = true
		}
		if err := Validate(&t); err != nil {
			fail("%s", err)
			continue
		}
		r.Tx = t
		out = append(out, r)
	}
	return out, errs
}
