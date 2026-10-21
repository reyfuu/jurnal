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
	// Format: bentuk tabel di Excel (garis, header, kolom angka, dropdown, kunci kolom).
	// Hanya untuk template yang diunduh; diabaikan saat membaca berkas unggahan.
	Format *FormatTabel
}

// FormatTabel dicerminkan server ke xlsx.Tabel (paket finance tidak mengenal xlsx).
type FormatTabel struct {
	Header, Kolom, BarisKosong int
	Lebar                      []float64
	Angka                      []int
	Pilihan                    map[int][]string
	Kunci                      bool
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
	Tagihan   []Tagihan       `json:"tagihan"`
	Stok      []StokItem      `json:"stok"`
	Errors    []SheetError    `json:"errors"`
	// Terbaca mencatat sheet mana yang dikenali, supaya user tahu kalau ada sheet terlewat.
	Terbaca []string `json:"terbaca"`
}

const (
	sheetSaldoAwal = "Saldo Awal"
	sheetAset      = "Aset"
	sheetKaryawan  = "Karyawan"
	sheetTransaksi = "Transaksi"
	sheetTagihan   = "Piutang & Utang"
	sheetStok      = "Stok"
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
func OnboardingTemplate(metode string) []ImportSheet {
	// Contoh aset dibeli sebelum tanggal mulai, jadi nilai bukunya masuk saldo awal.
	aset := [][]string{asetHeaders,
		{"Mesin Espresso 2 Group", "Mesin pengolah makanan & minuman", "", "Garis Lurus", "2025-02-01", "90000000"},
		{"Kulkas & Chiller Display", "Peralatan usaha restoran / rumah makan", "", "Garis Lurus", "2025-03-01", "18000000"},
		{"Renovasi Ruko Sewa", "Bangunan tidak permanen (renovasi ruko sewa)", "", "Garis Lurus", "2025-02-01", "36000000"},
		{"Software Kasir & Lisensi", "Harta tak berwujud — software & lisensi", "", "Garis Lurus", "2025-02-01", "18000000"},
	}
	saldo := saldoAwalTemplateRows(aset, metode)

	karyawan := [][]string{karyawanHeaders,
		{"Dewi Lestari", "Kepala Barista", "2026-01-05", "5500000", "750000", "K/1", "", "Aktif"},
		{"Bagus Pratama", "Barista", "2026-02-01", "4200000", "500000", "TK/0", "", "Aktif"},
		{"Sari Andini", "Kasir", "2026-02-15", "4000000", "400000", "TK/0", "", "Aktif"},
	}

	// Barisnya dipakai juga oleh template per-menu (lihat import_menu.go), jadi satu sumber saja.
	tagihan := tagihanTemplateRows()
	stok := stokTemplateRows()

	panduan := [][]string{
		{"Sheet", "Wajib?", "Catatan"},
		{sheetSaldoAwal, "Disarankan", "Posisi keuangan di hari pertama pakai aplikasi. Neraca & arus kas dihitung dari sini + transaksi."},
		{sheetAset, "Opsional", "Isi Jenis Harta saja; Kelompok PMK & tarif terisi otomatis. Bangunan hanya boleh Garis Lurus."},
		{sheetKaryawan, "Opsional", "Status PTKP memakai kode seperti TK/0, K/1, K/3."},
		{sheetTransaksi, "Opsional", "Format sama dengan template transaksi biasa."},
		{sheetTagihan, "Opsional", "Jenis diisi Piutang atau Utang. Tanpa kolom Jenis, nomor berawalan PO dianggap utang."},
		{sheetStok, "Opsional", "Stok Minimum menentukan peringatan stok menipis; kosongkan bila tidak dipakai."},
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

	jenisTx := make([]string, 0, len(JenisList))
	for _, j := range JenisList {
		jenisTx = append(jenisTx, j.Nama)
	}
	tabel := func(kolom int, lebar []float64, angka []int, pilihan map[int][]string) *FormatTabel {
		return &FormatTabel{Kolom: kolom, Lebar: lebar, Angka: angka, Pilihan: pilihan, BarisKosong: 50, Kunci: true}
	}
	kategori := make([]string, 0, len(KategoriSaldoList))
	for _, k := range KategoriSaldoList {
		kategori = append(kategori, k.Label)
	}
	return []ImportSheet{
		{sheetSaldoAwal, saldo, &FormatTabel{Header: saldoHeaderRow, Kolom: 5, Lebar: []float64{12, 40, 32, 20, 48}, Angka: []int{3},
			Pilihan: map[int][]string{2: kategori}, BarisKosong: 30, Kunci: true}},
		{sheetAset, aset, tabel(6, []float64{32, 46, 16, 16, 18, 18}, []int{5}, map[int][]string{3: {"Garis Lurus", "Saldo Menurun"}})},
		{sheetKaryawan, karyawan, tabel(8, []float64{26, 20, 16, 16, 14, 13, 22, 12}, []int{3, 4}, map[int][]string{7: {"Aktif", "Nonaktif"}})},
		{sheetTagihan, tagihan, tabel(7, []float64{12, 18, 34, 16, 18, 14, 34}, []int{4}, map[int][]string{0: {"Piutang", "Utang"}, 5: {"Belum Lunas", "Lunas"}})},
		{sheetStok, stok, tabel(8, []float64{12, 36, 16, 12, 10, 14, 16, 16}, []int{3, 5, 6, 7}, nil)},
		{sheetTransaksi, TemplateRows(), tabel(10, []float64{14, 24, 16, 28, 22, 10, 14, 24, 14, 14}, []int{2, 5}, map[int][]string{1: jenisTx, 8: {"Lunas", "Belum Lunas"}})},
		{sheetPanduan, panduan, &FormatTabel{Kolom: 5, Lebar: []float64{44, 30, 18, 40, 18}}},
	}
}

// saldoHeaderRow: baris judul kolom di sheet Saldo Awal; di atasnya tanggal mulai & ringkasan.
const saldoHeaderRow = 7

// templateAkunSaldo: akun neraca CoA di sheet Saldo Awal (nama sama dengan CoA Master; dijaga
// tes di paket coa). Aset tetap per jenis tidak dirinci di sini: nilai bukunya satu baris yang
// harus sama dengan sheet Aset.
var templateAkunSaldo = []SaldoAkun{
	{Kode: "1101", Nama: "Kas", Kategori: KatKas, Keterangan: "uang tunai di laci / kasir"},
	{Kode: "1102", Nama: "Bank", Kategori: KatKas, Keterangan: "saldo semua rekening bank usaha; tambah baris per rekening bila perlu"},
	{Kode: "1103", Nama: "E-Wallet (QRIS/OVO/GoPay/Dana)", Kategori: KatKas},
	{Kode: "1104", Nama: "Saldo Marketplace", Kategori: KatKas, Keterangan: "saldo penjual yang belum ditarik"},
	{Kode: "1201", Nama: "Piutang Usaha", Kategori: KatPiutang, Keterangan: "penjualan yang belum dibayar pelanggan"},
	{Kode: "1202", Nama: "Piutang Pihak Berelasi", Kategori: KatAsetLain},
	{Kode: "1301", Nama: "Persediaan Barang Dagang", Kategori: KatPersediaan, Keterangan: "stok dihitung dengan harga beli"},
	{Kode: "1302", Nama: "Perlengkapan", Kategori: KatAsetLain},
	{Kode: "1401", Nama: "Biaya Dibayar Dimuka", Kategori: KatAsetLain, Keterangan: "mis. sewa tempat setahun yang sudah dibayar"},
	{Kode: "1402", Nama: "PPN Masukan", Kategori: KatAsetLain},
	{Kode: "1403", Nama: "Pajak Dibayar Dimuka (PPh)", Kategori: KatAsetLain},
	{Kode: "1404", Nama: "Uang Muka Pembelian", Kategori: KatAsetLain},
	{Kode: "1405", Nama: "Uang Muka Karyawan", Kategori: KatAsetLain},
	{Kode: "1407", Nama: "Piutang Karyawan", Kategori: KatAsetLain},
	{Kode: "", Nama: "Aset tetap (nilai buku, sesuai sheet Aset)", Kategori: KatAsetTetap, Keterangan: "harus sama dengan total nilai buku daftar aset per tanggal mulai"},
	{Kode: "2101", Nama: "Hutang Usaha", Kategori: KatUtang, Keterangan: "tagihan supplier yang belum dibayar"},
	{Kode: "2102", Nama: "Hutang Bank", Kategori: KatPinjaman, Keterangan: "sisa pokok pinjaman bank / koperasi / leasing"},
	{Kode: "2104", Nama: "Beban Yang Masih Harus Dibayar", Kategori: KatKewajibanLain},
	{Kode: "2105", Nama: "Hutang Lain-lain", Kategori: KatKewajibanLain},
	{Kode: "2107", Nama: "Utang PPh 21", Kategori: KatKewajibanLain},
	{Kode: "2110", Nama: "Utang PPh Final", Kategori: KatKewajibanLain},
	{Kode: "2111", Nama: "PPN Kurang Bayar", Kategori: KatKewajibanLain},
	{Kode: "2112", Nama: "Utang BPJS", Kategori: KatKewajibanLain},
	{Kode: "2114", Nama: "Utang Pihak Berelasi", Kategori: KatKewajibanLain},
	{Kode: "3101", Nama: "Modal Pemilik", Kategori: KatModal, Keterangan: "modal yang disetor pemilik sampai tanggal mulai"},
	{Kode: "3201", Nama: "Laba Ditahan", Kategori: KatLabaDitahan, Keterangan: "laba tahun-tahun lalu yang belum diambil; minus bila rugi"},
}

// TemplateAkunSaldo: salinan daftar akun template (dipakai layar Saldo Awal & tes CoA).
func TemplateAkunSaldo() []SaldoAkun { return append([]SaldoAkun(nil), templateAkunSaldo...) }

// labelKategori: label tampilan kategori saldo.
func labelKategori(id string) string {
	for _, k := range KategoriSaldoList {
		if k.ID == id {
			return k.Label
		}
	}
	return id
}

// saldoAwalTemplateRows: sheet Saldo Awal contoh yang SEIMBANG — nilai buku aset dihitung dari
// contoh sheet Aset (dengan metode usaha), modal disetor menjadi penyeimbangnya.
func saldoAwalTemplateRows(asetRows [][]string, metode string) [][]string {
	const mulai = "2026-01-01"
	as, _ := parseAsetRows(asetRows)
	TerapkanMetode(as, metode)
	nbv := AssetsForPeriod(as, nil, mulai[:7]).NilaiBukuAwal
	contoh := map[string]int64{"1101": 5_000_000, "1102": 40_000_000, "1201": 8_500_000, "1301": 8_250_000, "1401": 6_000_000,
		"2101": 5_200_000, "2102": 20_000_000, "2110": 450_000, "3201": 10_000_000}
	o := OpeningBalance{Tanggal: mulai}
	for _, a := range templateAkunSaldo {
		a.Nominal = contoh[a.Kode]
		if a.Kategori == KatAsetTetap {
			a.Nominal = nbv
		}
		o.Rincian = append(o.Rincian, a)
	}
	_ = o.TerapkanRincian()
	modal := o.Harta(nbv) - o.Kewajiban() - o.LabaDitahan

	rows := [][]string{
		{"Saldo Awal (Neraca Pembukaan)"},
		{"Tanggal mulai pembukuan", mulai, "Format TAHUN-BULAN-TANGGAL. Isi posisi usaha di pagi hari tanggal ini."},
		{"Total Harta", `=SUMIF(C:C,"Harta*",D:D)`},
		{"Total Kewajiban", `=SUMIF(C:C,"Kewajiban*",D:D)`},
		{"Total Modal", `=SUMIF(C:C,"Modal*",D:D)`},
		{"Selisih (harus 0)", "=B3-B4-B5", "Harta harus sama dengan Kewajiban + Modal. Bila belum 0, saldo awal ditolak saat impor."},
		{"", "", "Tambah baris untuk akun yang belum ada (mis. rekening bank kedua); pilih Kategori Akun dari daftar."},
		{"Kode Akun", "Nama Akun", "Kategori Akun", "Nominal (Rp)", "Keterangan"},
	}
	for _, a := range templateAkunSaldo {
		n := contoh[a.Kode]
		switch a.Kategori {
		case KatAsetTetap:
			n = nbv
		case KatModal:
			n = modal
		}
		nominal := ""
		if n != 0 {
			nominal = fmt.Sprint(n)
		}
		rows = append(rows, []string{a.Kode, a.Nama, labelKategori(a.Kategori), nominal, a.Keterangan})
	}
	return rows
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
	case strings.Contains(n, "piutang") || strings.Contains(n, "utang") || strings.Contains(n, "hutang") || strings.Contains(n, "tagihan"):
		return sheetTagihan
	case strings.Contains(n, "stok") || strings.Contains(n, "gudang") || strings.Contains(n, "persediaan") || strings.Contains(n, "inventory") || strings.Contains(n, "barang"):
		return sheetStok
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
		case sheetTagihan:
			ts, errs := parseTagihanRows(sh.Rows)
			d.Tagihan = append(d.Tagihan, ts...)
			d.Errors = append(d.Errors, tag(sh.Nama, errs)...)
		case sheetStok:
			st, errs := parseStokRows(sh.Rows)
			d.Stok = append(d.Stok, st...)
			d.Errors = append(d.Errors, tag(sh.Nama, errs)...)
		}
	}
	if len(d.Terbaca) == 0 {
		d.Errors = append(d.Errors, SheetError{Pesan: "tidak ada sheet yang dikenali; pakai template (Saldo Awal, Aset, Karyawan, Piutang & Utang, Stok, Transaksi)"})
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

// parseSaldoAwal: format per akun (kolom Kategori Akun & Nominal) bila ada, selain itu format
// lama dua kolom (label, nilai).
func parseSaldoAwal(rows [][]string) (*OpeningBalance, []ImportError) {
	if o, errs, ok := parseSaldoAwalAkun(rows); ok {
		return o, errs
	}
	return parseSaldoAwalLama(rows)
}

// parseSaldoAwalAkun membaca sheet Saldo Awal per akun. ok=false bila sheet bukan format ini.
func parseSaldoAwalAkun(rows [][]string) (*OpeningBalance, []ImportError, bool) {
	header, cols := -1, map[string]int{}
	for i, row := range rows {
		m := map[string]int{}
		for c, h := range row {
			n := norm(h)
			switch {
			case strings.Contains(n, "kategori"):
				m["kategori"] = c
			case strings.Contains(n, "nominal") || strings.Contains(n, "saldo") || strings.Contains(n, "jumlah") || n == "nilai" || strings.Contains(n, "nilai rp"):
				m["nominal"] = c
			case strings.Contains(n, "kode"):
				m["kode"] = c
			case strings.Contains(n, "nama") || n == "akun":
				m["nama"] = c
			case strings.Contains(n, "keterangan") || strings.Contains(n, "catatan"):
				m["ket"] = c
			}
		}
		_, k := m["kategori"]
		_, n := m["nominal"]
		if k && n {
			header, cols = i, m
			break
		}
	}
	if header < 0 {
		return nil, nil, false
	}
	get := func(row []string, k string) string {
		c, ok := cols[k]
		if !ok || c >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[c])
	}
	var o OpeningBalance
	var errs []ImportError
	for i, row := range rows[:header] {
		if len(row) >= 2 && strings.Contains(norm(row[0]), "tanggal") {
			if t, ok := parseTanggal(strings.TrimSpace(row[1])); ok {
				o.Tanggal = t
			} else {
				errs = append(errs, ImportError{i + 1, fmt.Sprintf("tanggal mulai %q tidak dikenali (pakai YYYY-MM-DD)", row[1])})
			}
		}
	}
	for i := header + 1; i < len(rows); i++ {
		row := rows[i]
		nama, kat, raw := get(row, "nama"), get(row, "kategori"), get(row, "nominal")
		if nama == "" && kat == "" && raw == "" {
			continue
		}
		var n int64
		if raw != "" {
			neg := strings.HasPrefix(raw, "-") || (strings.HasPrefix(raw, "(") && strings.HasSuffix(raw, ")"))
			v, ok := parseRupiah(strings.Trim(raw, "-() "))
			if !ok {
				errs = append(errs, ImportError{i + 1, fmt.Sprintf("nominal %q bukan angka", raw)})
				continue
			}
			if neg {
				v = -v
			}
			n = v
		}
		if n == 0 && raw == "" && kat != "" && nama != "" {
			// Akun template yang tidak diisi: lewati tanpa dianggap salah.
			continue
		}
		o.Rincian = append(o.Rincian, SaldoAkun{Kode: get(row, "kode"), Nama: nama, Kategori: kat, Nominal: n, Keterangan: get(row, "ket")})
	}
	if o.Tanggal == "" {
		errs = append(errs, ImportError{0, "tanggal mulai pembukuan belum diisi (baris \"Tanggal mulai pembukuan\" di atas tabel)"})
		return nil, errs, true
	}
	if err := o.TerapkanRincian(); err != nil {
		return nil, append(errs, ImportError{0, err.Error()}), true
	}
	if err := ValidateOpeningBalance(&o); err != nil {
		return nil, append(errs, ImportError{0, err.Error()}), true
	}
	return &o, errs, true
}

func parseSaldoAwalLama(rows [][]string) (*OpeningBalance, []ImportError) {
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

// --- Piutang & Utang ---

var tagihanHeaders = []string{"Jenis", "Nomor", "Pelanggan / Supplier", "Jatuh Tempo", "Nominal", "Status", "Keterangan"}

func tagihanColumn(h string) string {
	n := norm(h)
	switch {
	case strings.Contains(n, "jenis") || strings.Contains(n, "tipe"):
		return "jenis"
	case strings.Contains(n, "nomor") || strings.Contains(n, "no ") || n == "no" || strings.Contains(n, "invoice") || strings.Contains(n, "faktur"):
		return "nomor"
	case strings.Contains(n, "pelanggan") || strings.Contains(n, "supplier") || strings.Contains(n, "pihak") || strings.Contains(n, "nama"):
		return "pihak"
	case strings.Contains(n, "jatuh") || strings.Contains(n, "tempo") || strings.Contains(n, "tanggal"):
		return "jatuh_tempo"
	case strings.Contains(n, "nominal") || strings.Contains(n, "jumlah") || strings.Contains(n, "nilai"):
		return "nominal"
	case strings.Contains(n, "status") || strings.Contains(n, "lunas"):
		return "status"
	case strings.Contains(n, "keterangan") || strings.Contains(n, "catatan"):
		return "keterangan"
	}
	return ""
}

// jenisTagihanDari menebak arah tagihan dari teks bebas. Dibuat longgar karena istilahnya
// beragam di lapangan: "Piutang", "Receivable", "Tagihan Masuk", "Utang", "Hutang".
//
// Piutang WAJIB diperiksa lebih dulu: kata "piutang" memuat "utang", sehingga urutan terbalik
// membuat seluruh piutang salah dibaca sebagai utang.
func jenisTagihanDari(s string) string {
	n := norm(s)
	switch {
	case strings.Contains(n, "piutang") || strings.Contains(n, "receivable") || strings.Contains(n, "pelanggan") || strings.Contains(n, "masuk"):
		return TagihanPiutang
	case strings.Contains(n, "utang") || strings.Contains(n, "hutang") || strings.Contains(n, "payable") || strings.Contains(n, "supplier") || strings.Contains(n, "keluar"):
		return TagihanUtang
	}
	return ""
}

func parseTagihanRows(rows [][]string) ([]Tagihan, []ImportError) {
	headerAt, cols, ok := rowReader(rows, tagihanColumn, []string{"pihak", "jatuh_tempo", "nominal"})
	if !ok {
		return nil, []ImportError{{0, "header tidak ditemukan: butuh kolom Pelanggan/Supplier, Jatuh Tempo, dan Nominal"}}
	}
	var out []Tagihan
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

		t := Tagihan{Nomor: get("nomor"), Pihak: get("pihak"), Keterangan: get("keterangan")}
		t.Jenis = jenisTagihanDari(get("jenis"))
		if t.Jenis == "" {
			// Tanpa kolom Jenis, nomor berawalan PO menandakan utang ke supplier.
			if strings.HasPrefix(strings.ToUpper(t.Nomor), "PO") {
				t.Jenis = TagihanUtang
			} else {
				t.Jenis = TagihanPiutang
			}
		}
		jt, okt := parseTanggal(get("jatuh_tempo"))
		if !okt {
			fail("jatuh tempo %q tidak dikenali (pakai YYYY-MM-DD atau DD/MM/YYYY)", get("jatuh_tempo"))
			continue
		}
		t.JatuhTempo = jt
		n, okn := parseRupiah(get("nominal"))
		if !okn {
			fail("nominal %q bukan angka", get("nominal"))
			continue
		}
		t.Nominal = n
		t.Lunas = strings.Contains(norm(get("status")), "lunas") && !strings.Contains(norm(get("status")), "belum")
		if err := ValidateTagihan(&t); err != nil {
			fail("%s", err)
			continue
		}
		out = append(out, t)
	}
	return out, errs
}

// --- Stok & Gudang ---

var stokHeaders = []string{"SKU", "Nama Barang", "Kategori", "Jumlah", "Satuan", "Stok Minimum", "Harga Pokok", "Harga Jual"}

func stokColumn(h string) string {
	n := norm(h)
	switch {
	case strings.Contains(n, "sku") || strings.Contains(n, "kode"):
		return "sku"
	case strings.Contains(n, "nama") || strings.Contains(n, "barang") || strings.Contains(n, "produk"):
		return "nama"
	case strings.Contains(n, "kategori") || strings.Contains(n, "jenis") || strings.Contains(n, "tipe"):
		return "kategori"
	// Stok minimum diperiksa lebih dulu: "Stok Minimum" juga mengandung kata "stok".
	case strings.Contains(n, "minimum") || strings.Contains(n, "min"):
		return "stok_min"
	case strings.Contains(n, "jumlah") || strings.Contains(n, "stok") || strings.Contains(n, "qty") || strings.Contains(n, "kuantitas"):
		return "jumlah"
	case strings.Contains(n, "satuan") || strings.Contains(n, "unit"):
		return "satuan"
	case strings.Contains(n, "pokok") || strings.Contains(n, "hpp") || strings.Contains(n, "modal") || strings.Contains(n, "beli"):
		return "harga_pokok"
	case strings.Contains(n, "jual"):
		return "harga_jual"
	}
	return ""
}

func parseStokRows(rows [][]string) ([]StokItem, []ImportError) {
	headerAt, cols, ok := rowReader(rows, stokColumn, []string{"nama", "jumlah"})
	if !ok {
		return nil, []ImportError{{0, "header tidak ditemukan: butuh kolom Nama Barang dan Jumlah"}}
	}
	var out []StokItem
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

		s := StokItem{SKU: get("sku"), Nama: get("nama"), Kategori: get("kategori"), Satuan: get("satuan")}
		j, okj := parseRupiah(get("jumlah"))
		if !okj {
			fail("jumlah %q bukan angka", get("jumlah"))
			continue
		}
		s.Jumlah = j
		// Kolom berikut opsional: kosong berarti 0, bukan baris yang gagal.
		for _, f := range []struct {
			kolom string
			ke    *int64
		}{{"stok_min", &s.StokMin}, {"harga_pokok", &s.HargaPokok}, {"harga_jual", &s.HargaJual}} {
			if v := get(f.kolom); v != "" {
				n, okn := parseRupiah(v)
				if !okn {
					fail("%s %q bukan angka", strings.ReplaceAll(f.kolom, "_", " "), v)
					continue
				}
				*f.ke = n
			}
		}
		if err := ValidateStok(&s); err != nil {
			fail("%s", err)
			continue
		}
		out = append(out, s)
	}
	return out, errs
}
