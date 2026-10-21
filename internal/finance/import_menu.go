package finance

// Impor per menu. Halaman Stok & Gudang dan Invoice punya template satu sheet
// masing-masing, terpisah dari template onboarding (saldo awal rinci + Karyawan + Transaksi).
//
// Dipisah karena tujuannya berbeda: onboarding mengisi seluruh posisi awal sekali jalan dan
// MENGGANTI register yang ada, sementara impor per menu dipakai berulang untuk daftar yang
// hidup — menambah barang baru atau menyetor ulang daftar stok terkoreksi. Karena itu modenya
// dipilih pengguna (tambah / ganti), tidak dipaksa seperti onboarding.

func stokTemplateRows() [][]string {
	return [][]string{stokHeaders,
		{"RM-001", "Biji Kopi Arabica Gayo (Roasted)", "Bahan Baku", "42", "kg", "15", "185000", "0"},
		{"RM-002", "Fresh Milk Full Cream 1L", "Bahan Baku", "18", "liter", "24", "21500", "0"},
		{"FG-001", "Es Kopi Susu Senja (12oz)", "Produk Jadi", "120", "cup", "30", "6800", "18000"},
	}
}

// StokTemplate: template halaman Stok & Gudang.
func StokTemplate() []ImportSheet {
	panduan := [][]string{
		{"Kolom", "Wajib?", "Catatan"},
		{"SKU", "Opsional", "Kode barang, mis. RM-001."},
		{"Nama Barang", "Wajib", "Nama yang tampil di daftar stok."},
		{"Kategori", "Opsional", "Mis. Bahan Baku atau Produk Jadi. Kosong dianggap \"Barang\"."},
		{"Jumlah", "Wajib", "Stok saat ini dalam satuan di kolom berikutnya."},
		{"Satuan", "Opsional", "kg, liter, pcs, cup, dan sejenisnya."},
		{"Stok Minimum", "Opsional", "Batas peringatan stok menipis. Kosong atau 0 = tanpa peringatan."},
		{"Harga Pokok", "Disarankan", "HPP per satuan; dipakai menghitung nilai persediaan."},
		{"Harga Jual", "Opsional", "Isi untuk produk jadi; dipakai menghitung margin."},
		{"", "", ""},
		{"Nama sheet bebas selama memuat kata \"stok\", \"gudang\", \"persediaan\", atau \"barang\".", "", ""},
		{"Baris yang bermasalah dilewati dan ditampilkan sebelum impor; sisanya tetap masuk.", "", ""},
	}
	return []ImportSheet{{Nama: sheetStok, Rows: stokTemplateRows()}, {Nama: sheetPanduan, Rows: panduan}}
}

// pilihSheet mencari sheet yang cocok dengan satu jenis saja.
//
// Bila tidak ada yang namanya cocok tapi berkasnya hanya punya satu sheet, sheet itu yang
// dipakai: orang sering menyimpan ulang lewat Google Sheets atau "Save As" yang menamai
// sheet-nya "Sheet1", dan menolak berkasnya hanya karena namanya akan membingungkan.
func pilihSheet(sheets []ImportSheet, jenis string) (ImportSheet, bool) {
	for _, sh := range sheets {
		if kindOfSheet(sh.Nama) == jenis && len(sh.Rows) > 0 {
			return sh, true
		}
	}
	var isi []ImportSheet
	for _, sh := range sheets {
		if len(sh.Rows) > 0 {
			isi = append(isi, sh)
		}
	}
	if len(isi) == 1 {
		return isi[0], true
	}
	return ImportSheet{}, false
}

func sheetTidakDitemukan(jenis string) []SheetError {
	return []SheetError{{Pesan: "tidak ada sheet " + jenis + " yang dikenali; pakai template atau beri nama sheet-nya \"" + jenis + "\""}}
}

func ParseStokFile(sheets []ImportSheet) ([]StokItem, []SheetError, string) {
	sh, ok := pilihSheet(sheets, sheetStok)
	if !ok {
		return nil, sheetTidakDitemukan(sheetStok), ""
	}
	list, errs := parseStokRows(sh.Rows)
	return list, tag(sh.Nama, errs), sh.Nama
}
