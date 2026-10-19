package finance

import (
	"fmt"
	"slices"
	"strings"
)

// Impor data awal (onboarding): satu workbook berisi beberapa sheet supaya usaha yang sudah
// jalan bisa langsung memakai aplikasi tanpa input manual.
//
// Yang diimpor dari laporan keuangan adalah POSISI AWAL (neraca awal), bukan neraca & arus kas
// jadi: kedua laporan itu diturunkan dari transaksi + register aset, jadi angka jadi akan
// langsung bertabrakan dengan hasil hitung begitu ada transaksi baru.

// ImportSheet adalah satu sheet mentah; server yang mengubahnya dari/ke xlsx.Sheet.
type ImportSheet struct {
	Nama string
	Rows [][]string
}

// SheetError menunjuk sheet dan baris Excel (1-based) yang gagal; Baris 0 = masalah header.
type SheetError struct {
	Sheet string `json:"sheet"`
	Baris int    `json:"baris"`
	Pesan string `json:"pesan"`
}

// OnboardingData adalah hasil parse workbook; dipakai untuk pratinjau maupun commit.
type OnboardingData struct {
	Opening   *OpeningBalance `json:"opening"`
	Assets    []Asset         `json:"assets"`
	Karyawan  []Karyawan      `json:"karyawan"`
	Transaksi []Transaction   `json:"transaksi"`
	Errors    []SheetError    `json:"errors"`
	// Terbaca mencatat sheet mana yang dikenali, supaya user tahu kalau ada sheet terlewat.
	Terbaca []string `json:"terbaca"`
}

const (
	sheetSaldoAwal = "Saldo Awal"
	sheetAset      = "Aset"
	sheetKaryawan  = "Karyawan"
	sheetTransaksi = "Transaksi"
	sheetPanduan   = "Panduan"
)

var saldoAwalLabels = []struct {
	Label string
	Hint  string
}{
	{"Tanggal Mulai Pembukuan", "hari pertama pakai aplikasi; transaksi sebelum tanggal ini tidak perlu diimpor"},
	{"Kas & Bank", "saldo tunai + seluruh rekening bank usaha"},
	{"Piutang Usaha", "total tagihan pelanggan yang belum dibayar"},
	{"Persediaan", "nilai stok bahan baku & barang dagang"},
	{"Utang Usaha", "total tagihan supplier & biaya yang belum dibayar"},
	{"Pinjaman", "sisa pokok pinjaman bank / koperasi"},
	{"Modal Disetor", "opsional: untuk dicocokkan; kosongkan bila tidak tahu"},
}

var asetHeaders = []string{"Nama Aset", "Jenis Harta", "Kelompok PMK", "Metode", "Tanggal Perolehan", "Harga Perolehan"}

var karyawanHeaders = []string{"Nama", "Jabatan", "Tanggal Masuk", "Gaji Pokok", "Tunjangan", "Status PTKP", "NPWP", "Status"}

// OnboardingTemplate: workbook contoh yang isinya lolos ParseOnboarding apa adanya.
func OnboardingTemplate() []ImportSheet {
	saldo := [][]string{{"Keterangan", "Nilai", "Penjelasan"}}
	for _, l := range saldoAwalLabels {
		contoh := ""
		switch l.Label {
		case "Tanggal Mulai Pembukuan":
			contoh = "2026-01-01"
		case "Kas & Bank":
			contoh = "45000000"
		case "Piutang Usaha":
			contoh = "8500000"
		case "Persediaan":
			contoh = "8250000"
		case "Utang Usaha":
			contoh = "5200000"
		case "Pinjaman":
			contoh = "0"
		}
		saldo = append(saldo, []string{l.Label, contoh, l.Hint})
	}

	aset := [][]string{asetHeaders,
		{"Mesin Espresso 2 Group", "Mesin pengolah makanan & minuman", "", "Saldo Menurun", "2026-02-01", "90000000"},
		{"Kulkas & Chiller Display", "Peralatan usaha restoran / rumah makan", "", "Garis Lurus", "2026-03-01", "18000000"},
		{"Renovasi Ruko Sewa", "Bangunan tidak permanen (renovasi ruko sewa)", "", "Garis Lurus", "2026-02-01", "36000000"},
		{"Software Kasir & Lisensi", "Harta tak berwujud — software & lisensi", "", "Garis Lurus", "2026-02-01", "18000000"},
	}

	karyawan := [][]string{karyawanHeaders,
		{"Dewi Lestari", "Kepala Barista", "2026-01-05", "5500000", "750000", "K/1", "", "Aktif"},
		{"Bagus Pratama", "Barista", "2026-02-01", "4200000", "500000", "TK/0", "", "Aktif"},
		{"Sari Andini", "Kasir", "2026-02-15", "4000000", "400000", "TK/0", "", "Aktif"},
	}

	panduan := [][]string{
		{"Sheet", "Wajib?", "Catatan"},
		{sheetSaldoAwal, "Disarankan", "Posisi keuangan di hari pertama pakai aplikasi. Neraca & arus kas dihitung dari sini + transaksi."},
		{sheetAset, "Opsional", "Isi Jenis Harta saja; Kelompok PMK & tarif terisi otomatis. Bangunan hanya boleh Garis Lurus."},
		{sheetKaryawan, "Opsional", "Status PTKP memakai kode seperti TK/0, K/1, K/3."},
		{sheetTransaksi, "Opsional", "Format sama dengan template transaksi biasa."},
		{"", "", ""},
		{"Daftar Jenis Harta", "Kelompok", "Masa Manfaat", "Rujukan Lampiran PMK", "Status Pemetaan"},
	}
	for _, j := range JenisHartaList {
		k, _ := KelompokOf(j.Kelompok)
		jenis := j.Nama
		if j.TakBerwujud {
			jenis += " (amortisasi)"
		}
		status := "PERLU DICEK"
		if j.Terverifikasi {
			status = "Terverifikasi"
		}
		panduan = append(panduan, []string{jenis, k.Nama, fmt.Sprintf("%d bulan", k.MasaManfaat), j.Lampiran, status})
	}
	panduan = append(panduan, []string{"", "", ""}, []string{"Daftar Jenis Transaksi", "Arah uang", ""})
	for _, j := range JenisList {
		panduan = append(panduan, []string{j.Nama, j.Arah, ""})
	}

	return []ImportSheet{
		{sheetSaldoAwal, saldo},
		{sheetAset, aset},
		{sheetKaryawan, karyawan},
		{sheetTransaksi, TemplateRows()},
		{sheetPanduan, panduan},
	}
}

// kindOfSheet mengenali sheet dari namanya; nama bebas selama mengandung kata kuncinya.
func kindOfSheet(nama string) string {
	n := norm(nama)
	switch {
	case strings.Contains(n, "saldo") || strings.Contains(n, "neraca awal") || strings.Contains(n, "awal"):
		return sheetSaldoAwal
	case strings.Contains(n, "aset") || strings.Contains(n, "harta") || strings.Contains(n, "aktiva tetap"):
		return sheetAset
	case strings.Contains(n, "karyawan") || strings.Contains(n, "pegawai") || strings.Contains(n, "staf") || strings.Contains(n, "staff"):
		return sheetKaryawan
	case strings.Contains(n, "transaksi") || strings.Contains(n, "mutasi") || strings.Contains(n, "jurnal"):
		return sheetTransaksi
	}
	return ""
}

// ParseOnboarding membaca seluruh sheet yang dikenali. Sheet yang tidak dikenali diabaikan,
// baris bermasalah dilaporkan dan tidak ikut diimpor.
func ParseOnboarding(sheets []ImportSheet) OnboardingData {
	var d OnboardingData
	for _, sh := range sheets {
		kind := kindOfSheet(sh.Nama)
		if kind == "" || len(sh.Rows) == 0 {
			continue
		}
		d.Terbaca = append(d.Terbaca, sh.Nama)
		switch kind {
		case sheetSaldoAwal:
			o, errs := parseSaldoAwal(sh.Rows)
			d.Opening = o
			d.Errors = append(d.Errors, tag(sh.Nama, errs)...)
		case sheetAset:
			as, errs := parseAsetRows(sh.Rows)
			d.Assets = append(d.Assets, as...)
			d.Errors = append(d.Errors, tag(sh.Nama, errs)...)
		case sheetKaryawan:
			ks, errs := parseKaryawanRows(sh.Rows)
			d.Karyawan = append(d.Karyawan, ks...)
			d.Errors = append(d.Errors, tag(sh.Nama, errs)...)
		case sheetTransaksi:
			txs, errs := ParseImport(sh.Rows)
			d.Transaksi = append(d.Transaksi, txs...)
			d.Errors = append(d.Errors, tag(sh.Nama, errs)...)
		}
	}
	if len(d.Terbaca) == 0 {
		d.Errors = append(d.Errors, SheetError{Pesan: "tidak ada sheet yang dikenali; pakai template (Saldo Awal, Aset, Karyawan, Transaksi)"})
	}
	return d
}

func tag(sheet string, errs []ImportError) []SheetError {
	out := make([]SheetError, 0, len(errs))
	for _, e := range errs {
		out = append(out, SheetError{Sheet: sheet, Baris: e.Baris, Pesan: e.Pesan})
	}
	return out
}

// --- Saldo awal: sheet dua kolom (label, nilai) ---

func saldoField(label string) string {
	h := norm(label)
	switch {
	case strings.Contains(h, "tanggal"):
		return "tanggal"
	case strings.Contains(h, "pinjaman") || strings.Contains(h, "kredit bank") || strings.Contains(h, "leasing"):
		return "pinjaman"
	case strings.Contains(h, "piutang"):
		return "piutang"
	case strings.Contains(h, "persediaan") || strings.Contains(h, "stok") || strings.Contains(h, "inventory"):
		return "persediaan"
	case strings.Contains(h, "utang") || strings.Contains(h, "hutang"):
		return "utang"
	case strings.Contains(h, "modal") || strings.Contains(h, "ekuitas"):
		return "modal"
	case strings.Contains(h, "kas") || strings.Contains(h, "bank") || strings.Contains(h, "tunai"):
		return "kas"
	}
	return ""
}

func parseSaldoAwal(rows [][]string) (*OpeningBalance, []ImportError) {
	var o OpeningBalance
	var errs []ImportError
	found := false
	for i, row := range rows {
		if len(row) < 2 {
			continue
		}
		field := saldoField(row[0])
		if field == "" {
			continue
		}
		raw := strings.TrimSpace(row[1])
		if raw == "" {
			continue // baris sengaja dikosongkan: pakai 0
		}
		found = true
		if field == "tanggal" {
			t, ok := parseTanggal(raw)
			if !ok {
				errs = append(errs, ImportError{i + 1, fmt.Sprintf("tanggal mulai %q tidak dikenali (pakai YYYY-MM-DD)", raw)})
				continue
			}
			o.Tanggal = t
			continue
		}
		n, ok := parseRupiah(raw)
		if !ok {
			errs = append(errs, ImportError{i + 1, fmt.Sprintf("nilai %q bukan angka", raw)})
			continue
		}
		switch field {
		case "kas":
			o.Kas = n
		case "piutang":
			o.Piutang = n
		case "persediaan":
			o.Persediaan = n
		case "utang":
			o.Utang = n
		case "pinjaman":
			o.Pinjaman = n
		case "modal":
			o.ModalDisetor = n
		}
	}
	if !found {
		return nil, append(errs, ImportError{0, "sheet Saldo Awal tidak punya baris yang dikenali (Tanggal Mulai, Kas & Bank, Piutang, Persediaan, Utang, Pinjaman)"})
	}
	if err := ValidateOpeningBalance(&o); err != nil {
		return nil, append(errs, ImportError{0, err.Error()})
	}
	return &o, errs
}

// --- Aset ---

func asetColumn(h string) string {
	n := norm(h)
	switch {
	case strings.Contains(n, "jenis") || strings.Contains(n, "golongan harta"):
		return "jenis_harta"
	case strings.Contains(n, "kelompok"):
		return "kelompok"
	case strings.Contains(n, "metode") || strings.Contains(n, "penyusutan"):
		return "metode"
	case strings.Contains(n, "harga") || strings.Contains(n, "nilai") || strings.Contains(n, "rp"):
		return "harga"
	case strings.Contains(n, "tanggal") || strings.Contains(n, "perolehan"):
		return "tanggal"
	case strings.Contains(n, "nama") || strings.Contains(n, "aset") || strings.Contains(n, "keterangan"):
		return "nama"
	}
	return ""
}

func findJenisHarta(s string) (JenisHarta, bool) {
	q := norm(s)
	if q == "" {
		return JenisHarta{}, false
	}
	// Buang keterangan dalam tanda kurung pada template, mis. "... (amortisasi)".
	if i := strings.Index(q, "("); i > 0 {
		q = strings.TrimSpace(q[:i])
	}
	if i := slices.IndexFunc(JenisHartaList, func(j JenisHarta) bool { return norm(j.Nama) == q || j.ID == q }); i >= 0 {
		return JenisHartaList[i], true
	}
	if i := slices.IndexFunc(JenisHartaList, func(j JenisHarta) bool {
		n := norm(j.Nama)
		return strings.HasPrefix(n, q) || strings.HasPrefix(q, n)
	}); i >= 0 {
		return JenisHartaList[i], true
	}
	return JenisHarta{}, false
}

func findKelompok(s string) (KelompokAset, bool) {
	q := norm(s)
	if q == "" {
		return KelompokAset{}, false
	}
	if k, ok := KelompokOf(strings.ReplaceAll(q, " ", "_")); ok {
		return k, ok
	}
	bangunan := strings.Contains(q, "bangunan")
	switch {
	case bangunan && (strings.Contains(q, "tidak") || strings.Contains(q, "non")):
		return KelompokOf("bangunan_tidak_permanen")
	case bangunan:
		return KelompokOf("bangunan_permanen")
	}
	for _, n := range []string{"1", "2", "3", "4"} {
		if strings.HasSuffix(q, n) || strings.Contains(q, "kelompok "+n) {
			return KelompokOf("k" + n)
		}
	}
	return KelompokAset{}, false
}

func findMetode(s string) string {
	n := norm(s)
	if strings.Contains(n, "saldo") || strings.Contains(n, "menurun") || n == "sm" || strings.Contains(n, "declin") {
		return MetodeSaldoMenurun
	}
	return MetodeGarisLurus // kosong atau "garis lurus" → default paling umum
}

// rowReader mencari baris header lalu mengembalikan pembaca kolom per baris.
func rowReader(rows [][]string, colOf func(string) string, wajib []string) (int, map[string]int, bool) {
	for i := 0; i < len(rows) && i < 10; i++ {
		m := map[string]int{}
		for c, h := range rows[i] {
			if k := colOf(h); k != "" {
				if _, dup := m[k]; !dup {
					m[k] = c
				}
			}
		}
		if !slices.ContainsFunc(wajib, func(k string) bool { _, ok := m[k]; return !ok }) {
			return i, m, true
		}
	}
	return -1, nil, false
}

func parseAsetRows(rows [][]string) ([]Asset, []ImportError) {
	headerAt, cols, ok := rowReader(rows, asetColumn, []string{"nama", "tanggal", "harga"})
	if !ok {
		return nil, []ImportError{{0, "header tidak ditemukan: butuh kolom Nama Aset, Tanggal Perolehan, dan Harga Perolehan"}}
	}
	var out []Asset
	var errs []ImportError
	for i := headerAt + 1; i < len(rows); i++ {
		row := rows[i]
		if strings.TrimSpace(strings.Join(row, "")) == "" {
			continue
		}
		get := func(k string) string {
			if c, okc := cols[k]; okc && c < len(row) {
				return strings.TrimSpace(row[c])
			}
			return ""
		}
		fail := func(format string, a ...any) { errs = append(errs, ImportError{i + 1, fmt.Sprintf(format, a...)}) }

		a := Asset{Nama: get("nama"), Metode: findMetode(get("metode"))}
		if j, okj := findJenisHarta(get("jenis_harta")); okj {
			a.JenisHarta = j.ID
		} else if k, okk := findKelompok(get("kelompok")); okk {
			a.Kelompok = k.ID
		} else {
			fail("jenis harta %q dan kelompok %q tidak dikenali; lihat sheet Panduan", get("jenis_harta"), get("kelompok"))
			continue
		}
		t, okt := parseTanggal(get("tanggal"))
		if !okt {
			fail("tanggal perolehan %q tidak dikenali (pakai YYYY-MM-DD)", get("tanggal"))
			continue
		}
		a.TanggalPerolehan = t
		n, okn := parseRupiah(get("harga"))
		if !okn {
			fail("harga perolehan %q bukan angka", get("harga"))
			continue
		}
		a.HargaPerolehan = n
		if err := ValidateAsset(&a); err != nil {
			fail("%s", err)
			continue
		}
		out = append(out, a)
	}
	return out, errs
}

// --- Karyawan ---

func karyawanColumn(h string) string {
	n := norm(h)
	switch {
	case strings.Contains(n, "gaji") || strings.Contains(n, "upah pokok"):
		return "gaji"
	case strings.Contains(n, "tunjangan") || strings.Contains(n, "allowance"):
		return "tunjangan"
	case strings.Contains(n, "ptkp"):
		return "ptkp"
	case strings.Contains(n, "npwp"):
		return "npwp"
	case strings.Contains(n, "tanggal") || strings.Contains(n, "masuk") || strings.Contains(n, "mulai kerja"):
		return "tanggal"
	case strings.Contains(n, "jabatan") || strings.Contains(n, "posisi") || strings.Contains(n, "divisi"):
		return "jabatan"
	case strings.Contains(n, "status") || strings.Contains(n, "aktif"):
		return "status"
	case strings.Contains(n, "nama") || strings.Contains(n, "karyawan") || strings.Contains(n, "pegawai"):
		return "nama"
	}
	return ""
}

func parseKaryawanRows(rows [][]string) ([]Karyawan, []ImportError) {
	headerAt, cols, ok := rowReader(rows, karyawanColumn, []string{"nama", "tanggal", "gaji"})
	if !ok {
		return nil, []ImportError{{0, "header tidak ditemukan: butuh kolom Nama, Tanggal Masuk, dan Gaji Pokok"}}
	}
	var out []Karyawan
	var errs []ImportError
	for i := headerAt + 1; i < len(rows); i++ {
		row := rows[i]
		if strings.TrimSpace(strings.Join(row, "")) == "" {
			continue
		}
		get := func(k string) string {
			if c, okc := cols[k]; okc && c < len(row) {
				return strings.TrimSpace(row[c])
			}
			return ""
		}
		fail := func(format string, a ...any) { errs = append(errs, ImportError{i + 1, fmt.Sprintf(format, a...)}) }

		k := Karyawan{Nama: get("nama"), Jabatan: get("jabatan"), PTKP: strings.ToUpper(get("ptkp")), NPWP: get("npwp")}
		k.Aktif = !strings.Contains(norm(get("status")), "resign") && !strings.Contains(norm(get("status")), "tidak aktif")
		t, okt := parseTanggal(get("tanggal"))
		if !okt {
			fail("tanggal masuk %q tidak dikenali (pakai YYYY-MM-DD)", get("tanggal"))
			continue
		}
		k.TanggalMasuk = t
		gaji, okg := parseRupiah(get("gaji"))
		if !okg {
			fail("gaji pokok %q bukan angka", get("gaji"))
			continue
		}
		k.GajiPokok = gaji
		if v := get("tunjangan"); v != "" {
			n, okn := parseRupiah(v)
			if !okn {
				fail("tunjangan %q bukan angka", v)
				continue
			}
			k.Tunjangan = n
		}
		if err := ValidateKaryawan(&k); err != nil {
			fail("%s", err)
			continue
		}
		out = append(out, k)
	}
	return out, errs
}
