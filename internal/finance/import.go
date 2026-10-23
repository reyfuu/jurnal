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

var importHeaders = []string{"Tanggal", "Jenis", "Nominal", "Keterangan", "Produk / Bahan", "Jumlah", "Channel", "Pelanggan / Supplier", "Status Bayar", "Tanggal Lunas"}

// TemplateRows: header + contoh untuk template Excel; contohnya harus lolos ParseImport.
func TemplateRows() [][]string {
	return [][]string{
		importHeaders,
		{"2026-08-01", "Penjualan", "1250000", "Rekap kasir", "Es Kopi Susu", "50", "Dine-in", "", "Lunas", ""},
		{"2026-08-02", "Beli Bahan / Stok", "850000", "Belanja susu", "Fresh Milk 1L", "40", "", "Toko Susu Segar", "Belum Lunas", ""},
		{"2026-08-28", "Gaji & Upah", "12500000", "Gaji barista", "", "", "", "", "Lunas", ""},
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

// ParseImport mengubah baris sheet menjadi transaksi valid; baris bermasalah dilaporkan, tidak ikut.
func ParseImport(rows [][]string) ([]Transaction, []ImportError) {
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

	var txs []Transaction
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
		if err := Validate(&t); err != nil {
			fail("%s", err)
			continue
		}
		txs = append(txs, t)
	}
	return txs, errs
}
