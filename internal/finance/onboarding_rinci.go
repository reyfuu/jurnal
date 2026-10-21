package finance

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

// Template saldo awal rinci: satu sheet per kelompok akun (Kas dan Bank, Piutang, …, Modal),
// Neraca Saldo Awal terbentuk sendiri lewat rumus dan menunjukkan apakah sudah seimbang.
// Tiap baris = satu rekening / faktur / barang / aset, supaya saldo awal bisa dicocokkan ke
// dokumen sumbernya. Saat impor, nilai turunan (sisa, nilai persediaan, akumulasi) dihitung
// ulang dari kolom isian — nilai rumus di berkas hanya ada bila berkas sudah disimpan Excel.

const (
	sheetPetunjuk      = "Petunjuk"
	sheetNeraca        = "Neraca Saldo Awal"
	sheetKasBank       = "Kas dan Bank"
	sheetPiutang       = "Piutang"
	sheetPersediaan    = "Persediaan"
	sheetPerlengkapan  = "Perlengkapan"
	sheetAsetLancar    = "Aset Lancar Lain"
	sheetAsetTetap     = "Aset Tetap"
	sheetUtang         = "Utang"
	sheetModal         = "Modal"
	tanggalContoh      = "2026-01-01"
	barisHeaderRinci   = 5    // baris judul kolom (0-based) di sheet rincian = baris 6 Excel
	barisAkhirRumus    = 1000 // batas rentang rumus SUM/SUMIFS ke sheet rincian
	selTanggalSaldo    = "Petunjuk!$C$6"
	barisKosongRinci   = 50
	labelTanggalSaldo  = "Tanggal Saldo Awal"
	labelSaranSaldoLab = "Saran Saldo Laba agar seimbang"
)

// Pilihan dropdown; teksnya juga dipakai rumus SUMIFS di Neraca Saldo Awal, jadi satu sumber.
var (
	jenisAsetLancarList = []string{"Biaya Dibayar di Muka", "Pajak Dibayar di Muka", "Uang Muka Pembelian", "Piutang Lain-lain"}
	jenisUtangList      = []string{"Utang Usaha", "Utang Pajak", "Biaya Masih Harus Dibayar", "Pendapatan Diterima di Muka", "Utang Lain-lain", "Utang Bank / Pinjaman"}
	jenisEkuitasList    = []string{"Modal Disetor", "Tambahan Modal Disetor", "Saldo Laba", "Prive", "Dividen"}
	kategoriBarangList  = []string{"Barang Dagangan", "Bahan Baku", "Barang Dalam Proses", "Barang Jadi", "Kemasan"}
)

// labelKelompokFiskal: "Kelompok 1 (4 th)", "Bangunan Permanen (20 th)", …
func labelKelompokFiskal(k KelompokAset) string {
	nama := k.Nama
	if strings.HasPrefix(k.ID, "k") {
		nama = "Kelompok " + k.ID[1:]
	}
	return fmt.Sprintf("%s (%d th)", nama, k.MasaManfaat/12)
}

// kindRinci mengenali sheet template rinci dari namanya ("" = bukan sheet rinci).
func kindRinci(nama string) string {
	n := norm(nama)
	has := func(w ...string) bool {
		return slices.ContainsFunc(w, func(x string) bool { return strings.Contains(n, x) })
	}
	switch {
	case has("petunjuk"):
		return sheetPetunjuk
	case has("neraca saldo"): // bukan "Neraca Awal" format lama (sheet saldo awal dua kolom)
		return sheetNeraca
	case has("kas") && (has("bank") || n == "kas"):
		return sheetKasBank
	case has("piutang"):
		return sheetPiutang
	case has("persediaan"):
		return sheetPersediaan
	case has("perlengkapan"):
		return sheetPerlengkapan
	case has("aset lancar"):
		return sheetAsetLancar
	case has("aset tetap", "aktiva tetap"):
		return sheetAsetTetap
	case has("utang", "hutang", "kewajiban", "liabilitas"):
		return sheetUtang
	case has("modal", "ekuitas"):
		return sheetModal
	}
	return ""
}

// formatRinci: workbook memakai template rinci bila ada sheet khas template ini.
func formatRinci(sheets []ImportSheet) bool {
	return slices.ContainsFunc(sheets, func(sh ImportSheet) bool {
		switch kindRinci(sh.Nama) {
		case sheetKasBank, sheetModal, sheetNeraca, sheetAsetLancar:
			return true
		}
		return false
	})
}

// ---------------------------------------------------------------------------------------
// Template
// ---------------------------------------------------------------------------------------

// rentang: rentang kolom sheet rincian untuk rumus lintas sheet, mis. 'Kas dan Bank'!$F$7:$F$1000.
func rentang(sheet, kol string) string {
	s := sheet
	if strings.ContainsAny(s, " &-") {
		s = "'" + s + "'"
	}
	return fmt.Sprintf("%s!$%s$%d:$%s$%d", s, kol, barisHeaderRinci+2, kol, barisAkhirRumus)
}

// rinciDef: satu sheet rincian. Baris 1 judul, 2 penjelasan, 3 "Per tanggal", 4 total,
// 5 kosong, 6 judul kolom, 7 dst. data.
type rinciDef struct {
	nama, judul, sub         string
	header                   []string
	total                    []string // baris 4
	contoh                   [][]string
	lebar                    []float64
	angka, bilangan, tanggal []int
	rumus                    map[int]string
	pilihan                  map[int][]string
	bebas                    []int // kolom pilihan yang hanya saran
	gaya                     map[string]string
}

// rumusPerTanggal: "Per tanggal: 2026-01-01 | Nama usaha" tanpa TEXT() (kode format TEXT
// bergantung bahasa Excel).
const rumusPerTanggal = `="Per tanggal: "&IF(Petunjuk!$C$6="","(isi Tanggal Saldo Awal di sheet Petunjuk)",YEAR(Petunjuk!$C$6)&"-"&RIGHT("0"&MONTH(Petunjuk!$C$6),2)&"-"&RIGHT("0"&DAY(Petunjuk!$C$6),2))&IF(Petunjuk!$C$4="","","   |   "&Petunjuk!$C$4)`

func (r rinciDef) sheet() ImportSheet {
	rows := [][]string{{r.judul}, {r.sub}, {rumusPerTanggal}, r.total, {}, r.header}
	rows = append(rows, r.contoh...)
	rumus := map[int]string{0: `=IF(B{r}="","",ROW()-6)`} // nomor urut
	for c, f := range r.rumus {
		rumus[c] = f
	}
	gaya := map[string]string{"A3": "catatan"}
	for k, v := range r.gaya {
		gaya[k] = v
	}
	return ImportSheet{Nama: r.nama, Rows: rows, Format: &FormatTabel{
		Header: barisHeaderRinci, Kolom: len(r.header), Lebar: r.lebar, Angka: r.angka, Bilangan: r.bilangan, Tanggal: r.tanggal,
		Contoh: len(r.contoh), BarisKosong: barisKosongRinci, Pilihan: r.pilihan, Bebas: r.bebas, Rumus: rumus, Gaya: gaya, Kunci: true,
	}}
}

// asetContoh: contoh aset tetap; semuanya diperoleh sebelum tanggal saldo awal.
var asetContoh = []struct {
	nama, jenis, tanggal string
	harga                int64
	lokasi               string
}{
	{"Laptop Kasir", "komputer", "2025-01-01", 15_000_000, "Meja kasir"},
	{"Mesin Espresso 2 Group", "mesin_makanan", "2025-02-01", 90_000_000, "Bar"},
	{"Kulkas & Chiller Display", "peralatan_restoran", "2025-03-01", 18_000_000, "Bar"},
	{"Renovasi Ruko Sewa", "bangunan_tidak_permanen", "2025-02-01", 36_000_000, "Ruko Jl. Melati"},
	{"Software Kasir & Lisensi", "software", "2025-02-01", 18_000_000, "Kasir"},
}

// OnboardingTemplate: workbook contoh yang isinya lolos ParseOnboarding apa adanya dan
// SEIMBANG untuk metode penyusutan usaha mana pun: akumulasi contoh aset dihitung sistem
// dengan metode usaha, lalu Modal Disetor menjadi penyeimbangnya. Kode akun Neraca Saldo Awal
// mengikuti CoA dagang; server menggantinya dengan kode CoA usaha (lihat handleOnboardingTemplate).
func OnboardingTemplate(metode string) []ImportSheet {
	petunjuk, jh, kf := petunjukSheet()
	// Tabel acuan di Petunjuk: Jenis Harta → Kelompok Fiskal, Kelompok Fiskal → masa manfaat.
	jhRentang := fmt.Sprintf("Petunjuk!$B$%d:$C$%d", jh[0], jh[1])
	jhDaftar := fmt.Sprintf("Petunjuk!$B$%d:$B$%d", jh[0], jh[1])
	kfRentang := fmt.Sprintf("Petunjuk!$B$%d:$C$%d", kf[0], kf[1])

	// Aset tetap contoh. Akumulasi Dihitung di Excel = garis lurus fiskal; untuk saldo menurun
	// angka sistem ditulis di Akumulasi Manual supaya Excel & aplikasi sama.
	var as []Asset
	for _, c := range asetContoh {
		j, _ := jenisHartaOf(c.jenis)
		as = append(as, Asset{Nama: c.nama, JenisHarta: j.ID, Kelompok: j.Kelompok, TakBerwujud: j.TakBerwujud,
			Metode: MetodeGarisLurus, TanggalPerolehan: c.tanggal, HargaPerolehan: c.harga})
	}
	TerapkanMetode(as, metode)
	akum := map[string]int64{}
	for _, r := range AssetsForPeriod(as, nil, tanggalContoh[:7]).Assets {
		akum[r.Nama] = r.HargaPerolehan - r.NilaiBukuAwal
	}
	var asetRows [][]string
	for i, c := range asetContoh {
		j, _ := jenisHartaOf(c.jenis)
		k, _ := KelompokOf(j.Kelompok)
		manual, ket := "", "Faktur pembelian"
		if as[i].Metode == MetodeSaldoMenurun {
			manual, ket = strconv.FormatInt(akum[c.nama], 10), "Akumulasi manual = saldo menurun (dihitung aplikasi)"
		}
		asetRows = append(asetRows, []string{"", c.nama, j.Nama, labelKelompokFiskal(k), c.tanggal, strconv.FormatInt(c.harga, 10),
			"", "", manual, "", "", c.lokasi, ket})
	}

	kelompok := make([]string, 0, len(KelompokAsetList))
	for _, k := range KelompokAsetList {
		kelompok = append(kelompok, labelKelompokFiskal(k))
	}
	t := selTanggalSaldo
	bulanJalan := fmt.Sprintf("MAX(0,(YEAR(%s)-YEAR(E{r}))*12+MONTH(%s)-MONTH(E{r}))", t, t)

	sheets := []ImportSheet{
		petunjuk,
		{}, // Neraca Saldo Awal, diisi di bawah
		rinciDef{nama: sheetKasBank, judul: "RINCIAN KAS DAN BANK",
			sub:    "Satu baris per kas fisik / rekening bank / e-wallet. Bank: pakai saldo rekening koran per tanggal saldo awal. Jenis boleh diketik sendiri (mis. Deposito): jadi akun tersendiri.",
			header: []string{"No", "Nama Kas / Bank", "Jenis", "No. Rekening", "Atas Nama", "Saldo (Rp)", "Dasar Saldo", "Keterangan"},
			total: []string{"", "Total Kas", fmt.Sprintf(`=SUMIFS(F$7:F$%d,C$7:C$%d,"Kas")`, barisAkhirRumus, barisAkhirRumus),
				"Total Bank", fmt.Sprintf(`=SUMIFS(F$7:F$%d,C$7:C$%d,"Bank")`, barisAkhirRumus, barisAkhirRumus)},
			contoh: [][]string{
				{"", "Kas Kecil Kasir", "Kas", "", "", "5000000", "Hitung fisik", "Cash opname 31 Des 2025"},
				{"", "Bank BCA Operasional", "Bank", "1234567890", "Kedai Kopi Senja", "40000000", "Rekening koran", "Saldo akhir Des 2025"},
			},
			lebar: []float64{5, 30, 10, 18, 24, 18, 18, 32}, angka: []int{5},
			pilihan: map[int][]string{2: {"Kas", "Bank"}}, bebas: []int{2},
		}.sheet(),
		rinciDef{nama: sheetPiutang, judul: "RINCIAN PIUTANG USAHA",
			sub:    "Satu baris per faktur yang belum lunas. Umur dihitung otomatis dari jatuh tempo ke tanggal saldo awal.",
			header: []string{"No", "Nama Pelanggan", "No. Faktur / Invoice", "Tgl Faktur", "Jatuh Tempo", "Nilai Faktur (Rp)", "Sudah Dibayar (Rp)", "Sisa Piutang (Rp)", "Lewat Jatuh Tempo (hari)", "Kategori Umur", "Keterangan"},
			total: []string{"", "", "", "", "", "Total Piutang", fmt.Sprintf("=SUM(F$7:F$%d)-SUM(G$7:G$%d)", barisAkhirRumus, barisAkhirRumus),
				"Cadangan Kerugian", "", "← isi bila ada piutang yang kemungkinan tak tertagih (mengurangi piutang)"},
			contoh: [][]string{
				{"", "PT Karya Digital Mandiri", "INV-2025-118", "2025-12-10", "2026-01-09", "10000000", "1500000", "", "", "", "Termin 2 belum dibayar"},
			},
			lebar: []float64{5, 28, 18, 13, 13, 16, 16, 16, 13, 18, 30}, angka: []int{5, 6, 7}, tanggal: []int{3, 4},
			rumus: map[int]string{
				7: `=IF(F{r}="","",F{r}-N(G{r}))`,
				8: `=IF(OR(E{r}="",H{r}=""),"",MAX(0,` + t + `-E{r}))`,
				9: `=IF(I{r}="","",IF(I{r}=0,"Belum jatuh tempo",IF(I{r}<=30,"1-30 hari",IF(I{r}<=60,"31-60 hari",IF(I{r}<=90,"61-90 hari","Lebih dari 90 hari")))))`,
			},
			gaya: map[string]string{"I4": "isi_angka", "J4": "catatan"},
		}.sheet(),
		rinciDef{nama: sheetPersediaan, judul: "RINCIAN PERSEDIAAN",
			sub:    "Dari stock opname per tanggal saldo awal. Harga = harga pokok / beli per satuan, bukan harga jual. Ikut mengisi menu Stok & Gudang.",
			header: []string{"No", "Nama Barang", "Kode Barang", "Kategori", "Satuan", "Jumlah (Qty)", "Harga Pokok / Satuan (Rp)", "Nilai Persediaan (Rp)", "Lokasi / Gudang", "Keterangan"},
			total: []string{"", "", "", "", "", "", "Total Persediaan",
				fmt.Sprintf("=SUMPRODUCT(F$7:F$%d,G$7:G$%d)", barisAkhirRumus, barisAkhirRumus)},
			contoh: [][]string{
				{"", "Biji Kopi Arabica Gayo (Roasted)", "RM-001", "Bahan Baku", "kg", "30", "185000", "", "Gudang belakang", "Stock opname 31 Des 2025"},
				{"", "Fresh Milk Full Cream 1L", "RM-002", "Bahan Baku", "liter", "100", "21500", "", "Chiller", ""},
				{"", "Gula Aren Cair", "RM-003", "Bahan Baku", "liter", "20", "27500", "", "Gudang belakang", ""},
			},
			lebar: []float64{5, 32, 14, 18, 10, 13, 18, 18, 18, 26}, angka: []int{6, 7}, bilangan: []int{5},
			rumus:   map[int]string{7: `=IF(OR(F{r}="",G{r}=""),"",F{r}*G{r})`},
			pilihan: map[int][]string{3: kategoriBarangList},
		}.sheet(),
		rinciDef{nama: sheetPerlengkapan, judul: "RINCIAN PERLENGKAPAN",
			sub:    "Barang habis pakai untuk operasional (ATK, kemasan, bahan kebersihan) yang masih tersisa — bukan untuk dijual.",
			header: []string{"No", "Nama Perlengkapan", "Satuan", "Jumlah (Qty)", "Harga / Satuan (Rp)", "Nilai (Rp)", "Keterangan"},
			total: []string{"", "", "", "", "Total Perlengkapan",
				fmt.Sprintf("=SUMPRODUCT(D$7:D$%d,E$7:E$%d)", barisAkhirRumus, barisAkhirRumus)},
			contoh: [][]string{
				{"", "Cup plastik 12oz + tutup", "pcs", "1000", "850", "", "Sisa stok kemasan"},
			},
			lebar: []float64{5, 32, 10, 13, 18, 18, 30}, angka: []int{4, 5}, bilangan: []int{3},
			rumus: map[int]string{5: `=IF(OR(D{r}="",E{r}=""),"",D{r}*E{r})`},
		}.sheet(),
		rinciDef{nama: sheetAsetLancar, judul: "RINCIAN ASET LANCAR LAIN",
			sub:    "Biaya dibayar di muka, pajak dibayar di muka, uang muka pembelian, dan piutang lain-lain. Jenis lain boleh diketik sendiri (mis. Uang Jaminan): jadi akun tersendiri.",
			header: []string{"No", "Uraian", "Jenis", "Pihak Terkait", "Tgl Bayar / Dokumen", "Nilai Awal (Rp)", "Sudah Dibebankan / Dikreditkan (Rp)", "Sisa per Tgl Saldo Awal (Rp)", "Keterangan"},
			total:  []string{"", "", "", "", "", "", "Total", fmt.Sprintf("=SUM(F$7:F$%d)-SUM(G$7:G$%d)", barisAkhirRumus, barisAkhirRumus)},
			contoh: [][]string{
				{"", "Sewa ruko Jan–Jun 2026", "Biaya Dibayar di Muka", "CV Graha Ruko", "2025-12-20", "6000000", "0", "", "Dibebankan Rp1.000.000 / bulan"},
				{"", "PPh 23 dipotong pelanggan (bukti potong)", "Pajak Dibayar di Muka", "PT Karya Digital Mandiri", "2025-12-10", "200000", "", "", "Dikreditkan di SPT Tahunan"},
			},
			lebar: []float64{5, 34, 22, 24, 14, 16, 20, 18, 30}, angka: []int{5, 6, 7}, tanggal: []int{4},
			rumus:   map[int]string{7: `=IF(F{r}="","",F{r}-N(G{r}))`},
			pilihan: map[int][]string{2: jenisAsetLancarList}, bebas: []int{2},
		}.sheet(),
		rinciDef{nama: sheetAsetTetap, judul: "RINCIAN ASET TETAP (TERMASUK PERALATAN & HARTA TAK BERWUJUD)",
			sub: "Satu baris per aset. Akumulasi Dihitung = garis lurus fiskal per bulan (sama dengan aplikasi). Sudah punya angka dari pembukuan lama atau memakai saldo menurun? Isi Akumulasi Manual.",
			header: []string{"No", "Nama Aset", "Jenis Harta (opsional)", "Kelompok Fiskal", "Tgl Perolehan", "Harga Perolehan (Rp)", "Masa Manfaat (bulan)",
				"Akumulasi Dihitung (Rp)", "Akumulasi Manual (Rp) – opsional", "Akumulasi Dipakai (Rp)", "Nilai Buku (Rp)", "Lokasi / Pemakai", "Keterangan"},
			total: []string{"", "", "", "", "Harga Perolehan", fmt.Sprintf("=SUM(F$7:F$%d)", barisAkhirRumus),
				"Akum. Penyusutan", fmt.Sprintf("=SUM(J$7:J$%d)", barisAkhirRumus), "Nilai Buku", fmt.Sprintf("=SUM(K$7:K$%d)", barisAkhirRumus)},
			contoh: asetRows,
			lebar:  []float64{5, 28, 34, 26, 13, 17, 12, 17, 18, 17, 17, 18, 30}, angka: []int{5, 7, 8, 9, 10}, tanggal: []int{4},
			rumus: map[int]string{
				// Kelompok Fiskal diutamakan; kosong = ikut Jenis Harta (tabel di Petunjuk).
				6: `=IF(F{r}="","",IFERROR(VLOOKUP(IF(D{r}<>"",D{r},VLOOKUP(C{r},` + jhRentang + `,2,FALSE)),` + kfRentang + `,2,FALSE),""))`,
				// Sama dengan AssetSchedule garis lurus: INT(harga/n) per bulan, sisa pembulatan di
				// bulan terakhir; dihitung dari bulan perolehan s.d. bulan sebelum tanggal saldo awal.
				7:  `=IF(OR(E{r}="",F{r}="",G{r}=""),"",IF(` + bulanJalan + `>=G{r},F{r},INT(F{r}/G{r})*` + bulanJalan + `))`,
				9:  `=IF(I{r}<>"",I{r},H{r})`,
				10: `=IF(OR(F{r}="",J{r}=""),"",F{r}-J{r})`,
			},
			pilihan: map[int][]string{2: {"=" + jhDaftar}, 3: kelompok},
		}.sheet(),
		rinciDef{nama: sheetUtang, judul: "DAFTAR UTANG",
			sub:    "Semua kewajiban yang belum dibayar per tanggal saldo awal: ke supplier, pajak, bank, gaji, dan DP pelanggan. Jenis lain boleh diketik sendiri (mis. Utang ke Pemegang Saham).",
			header: []string{"No", "Uraian", "Jenis Utang", "Nama Kreditur / Pihak", "No. Dokumen", "Tgl Dokumen", "Jatuh Tempo", "Nilai Awal (Rp)", "Sudah Dibayar (Rp)", "Sisa Utang (Rp)", "Keterangan"},
			total:  []string{"", "", "", "", "", "", "", "", "Total Utang", fmt.Sprintf("=SUM(H$7:H$%d)-SUM(I$7:I$%d)", barisAkhirRumus, barisAkhirRumus)},
			contoh: [][]string{
				{"", "Pembelian biji kopi Desember", "Utang Usaha", "Roastery Gayo Lestari", "PO-SUP-098", "2025-12-15", "2026-01-14", "5200000", "", "", ""},
				{"", "PPh Final UMKM Desember 2025", "Utang Pajak", "Kas Negara", "", "2025-12-31", "2026-01-15", "450000", "", "", "Disetor Januari 2026"},
				{"", "Pinjaman KUR", "Utang Bank / Pinjaman", "Bank BRI", "KUR-2025-0601", "2025-06-01", "2027-06-01", "30000000", "10000000", "", "Sisa pokok"},
			},
			lebar: []float64{5, 32, 26, 24, 16, 13, 13, 16, 16, 16, 28}, angka: []int{7, 8, 9}, tanggal: []int{5, 6},
			rumus:   map[int]string{9: `=IF(H{r}="","",H{r}-N(I{r}))`},
			pilihan: map[int][]string{2: jenisUtangList}, bebas: []int{2},
		}.sheet(),
	}

	// Modal: Modal Disetor contoh = penyeimbang, dihitung dari sheet lain lewat parser sendiri
	// supaya angka Excel dan angka impor pasti sama.
	modalRows := func(modal string) [][]string {
		return [][]string{
			{"", "Pemilik (setoran modal)", "Modal Disetor", "Bukti setor / akta", modal, "", ""},
			{"", "Laba ditahan s.d. 31 Des 2025", "Saldo Laba", "Laporan keuangan 2025", "10000000", "", ""},
		}
	}
	neraca, barisTotal := neracaSheet()
	sheets[1] = neraca
	modal := rinciDef{nama: sheetModal, judul: "RINCIAN MODAL / EKUITAS",
		sub:    "Modal disetor sesuai akta / bukti setor, saldo laba tahun-tahun sebelumnya, dan prive / dividen (isi angka positif; otomatis mengurangi ekuitas).",
		header: []string{"No", "Nama Pemilik / Uraian", "Jenis Ekuitas", "Dasar / Dokumen", "Nilai (Rp)", "Keterangan"},
		total: []string{"", "", "Total Ekuitas (bersih)", fmt.Sprintf(`=SUM(E$7:E$%d)-2*SUMIFS(E$7:E$%d,C$7:C$%d,"Prive")-2*SUMIFS(E$7:E$%d,C$7:C$%d,"Dividen")`,
			barisAkhirRumus, barisAkhirRumus, barisAkhirRumus, barisAkhirRumus, barisAkhirRumus)},
		lebar: []float64{5, 30, 24, 26, 18, 40, 30}, angka: []int{4},
		pilihan: map[int][]string{2: jenisEkuitasList}, bebas: []int{2},
	}
	modal.contoh = modalRows("0")
	draft := append(slices.Clone(sheets), modal.sheet())
	d := ParseOnboarding(draft)
	if d.Opening != nil {
		TerapkanMetode(d.Assets, metode)
		nbv := AssetsForPeriod(d.Assets, nil, d.Opening.Period()).NilaiBukuAwal
		d.Opening.LengkapiAsetTetap(nbv)
		modal.contoh = modalRows(strconv.FormatInt(d.Opening.Harta(nbv)-d.Opening.Kewajiban()-d.Opening.Modal(), 10))
	}
	sh := modal.sheet()
	// Kolom saran (G) di luar tabel isian: rumus penyeimbang + penjelasannya.
	sh.Rows[barisHeaderRinci] = append(sh.Rows[barisHeaderRinci], labelSaranSaldoLab)
	sh.Rows[barisHeaderRinci+1][6] = fmt.Sprintf(`=SUMIFS(E$7:E$%d,C$7:C$%d,"Saldo Laba")+('%s'!$D$%d-'%s'!$E$%d)`,
		barisAkhirRumus, barisAkhirRumus, sheetNeraca, barisTotal, sheetNeraca, barisTotal)
	sh.Rows[barisHeaderRinci+2][6] = "Saldo laba yang membuat Neraca Saldo Awal seimbang. Pastikan semua aset & utang sudah lengkap sebelum memakainya."
	sheets = append(sheets, sh)

	karyawan := [][]string{karyawanHeaders,
		{"Dewi Lestari", "Kepala Barista", "2026-01-05", "5500000", "750000", "K/1", "", "Aktif"},
		{"Bagus Pratama", "Barista", "2026-02-01", "4200000", "500000", "TK/0", "", "Aktif"},
		{"Sari Andini", "Kasir", "2026-02-15", "4000000", "400000", "TK/0", "", "Aktif"},
	}
	jenisTx := make([]string, 0, len(JenisList))
	for _, j := range JenisList {
		jenisTx = append(jenisTx, j.Nama)
	}
	return append(sheets,
		ImportSheet{Nama: sheetKaryawan, Rows: karyawan, Format: &FormatTabel{Kolom: 8, Lebar: []float64{26, 20, 16, 16, 14, 13, 22, 12},
			Angka: []int{3, 4}, Tanggal: []int{2}, Contoh: len(karyawan) - 1, BarisKosong: 50, Kunci: true,
			Pilihan: map[int][]string{7: {"Aktif", "Nonaktif"}}}},
		ImportSheet{Nama: sheetTransaksi, Rows: TemplateRows(), Format: &FormatTabel{Kolom: 10, Lebar: []float64{14, 24, 16, 28, 22, 10, 14, 24, 14, 14},
			Angka: []int{2}, Bilangan: []int{5}, Tanggal: []int{0, 9}, Contoh: len(TemplateRows()) - 1, BarisKosong: 50, Kunci: true,
			Pilihan: map[int][]string{1: jenisTx, 8: {"Lunas", "Belum Lunas"}}}},
	)
}

// petunjukSheet: pengaturan (tanggal saldo awal di C6), legenda warna, langkah pengisian,
// dan tabel acuan yang dipakai rumus & dropdown sheet Aset Tetap. Mengembalikan baris Excel
// awal–akhir tabel Jenis Harta (Jenis → Kelompok Fiskal) dan Kelompok Fiskal (Kelompok → bulan).
func petunjukSheet() (ImportSheet, [2]int, [2]int) {
	gaya := map[string]string{"B1": "judul", "B3": "tebal", "C4": "isi", "C5": "isi", "C6": "isi_tanggal",
		"D6": "catatan", "D7": "catatan", "B8": "tebal", "C8": "rumus_tebal", "D8": "catatan", "B9": "tebal", "B10": "isi", "B11": "rumus", "B12": "biru", "B14": "tebal"}
	rows := [][]string{
		{"", "TEMPLATE PENGISIAN SALDO AWAL"},
		{"", "Isi rincian di tiap sheet — Neraca Saldo Awal terbentuk otomatis dan dicek keseimbangannya."},
		{"", "PENGATURAN"},
		{"", "Nama Perusahaan", ""},
		{"", "NPWP", ""},
		{"", labelTanggalSaldo, tanggalContoh, "Posisi di pagi hari tanggal ini (= saldo akhir hari sebelumnya), biasanya tanggal 1. Dipakai menghitung umur piutang & penyusutan; transaksi mulai tanggal ini."},
		{"", "Mata Uang", "Rupiah (Rp)", "Isi angka tanpa titik / koma / Rp, mis. 1500000."},
		{"", "Status Neraca Saldo Awal", "='" + sheetNeraca + "'!$D$4", "Terisi otomatis. Harus SEIMBANG sebelum diunggah; bila belum, lihat Selisih di sheet Neraca Saldo Awal."},
		{"", "ARTI WARNA"},
		{"", "Kuning muda", "Kolom yang DIISI pengguna"},
		{"", "Abu-abu", "Terisi otomatis oleh rumus — jangan diketik"},
		{"", "Teks biru", "Contoh pengisian — timpa atau hapus dengan data Anda"},
		{},
		{"", "LANGKAH PENGISIAN"},
		{"", "1. Isi Pengaturan di atas, terutama Tanggal Saldo Awal."},
		{"", "2. Siapkan dokumen: rekening koran, hasil stock opname, daftar faktur belum lunas, daftar aset, akta / bukti setor modal."},
		{"", "3. Isi sheet Kas dan Bank → Piutang → Persediaan → Perlengkapan → Aset Lancar Lain → Aset Tetap → Utang → Modal."},
		{"", "4. Satu baris = satu rekening / faktur / barang / aset. Jangan gabungkan beberapa item dalam satu baris."},
		{"", "5. Timpa atau hapus baris contoh (teks biru) dengan data Anda. Butuh baris lebih? Sisipkan baris di tengah tabel."},
		{"", "6. Buka sheet Neraca Saldo Awal: Status harus SEIMBANG. Bila belum, lihat Selisih-nya."},
		{"", "7. Selisih yang tersisa biasanya Saldo Laba (laba ditahan) — sheet Modal menampilkan angka yang disarankan."},
		{"", "8. Aset Tetap: hanya aset yang dibeli SEBELUM Tanggal Saldo Awal (yang sesudahnya dicatat di menu Aset beserta sumber dananya). Aset yang belum lunas: sisa utangnya diisi di sheet Utang, bukan dianggap modal. Akumulasi Dihitung = garis lurus fiskal seperti di aplikasi; usaha dengan saldo menurun, atau yang sudah punya angka dari pembukuan lama, isi Akumulasi Manual."},
		{"", "9. Opsional: sheet Karyawan dan Transaksi (transaksi mulai Tanggal Saldo Awal). Lalu unggah berkas ini di menu Impor Data Awal."},
		{},
		{"", "AKUN TAMBAHAN YANG DISARANKAN"},
	}
	tabel := func(judul string, header []string, isi [][]string) (int, int) {
		gaya[fmt.Sprintf("B%d", len(rows)+1)] = "tebal"
		rows = append(rows, []string{"", judul})
		baris := []string{""}
		for i, h := range header {
			gaya[fmt.Sprintf("%s%d", string(rune('B'+i)), len(rows)+1)] = "header"
			baris = append(baris, h)
		}
		rows = append(rows, baris)
		awal := len(rows) + 1
		for _, r := range isi {
			for i := range r {
				gaya[fmt.Sprintf("%s%d", string(rune('B'+i)), len(rows)+1)] = "garis"
			}
			rows = append(rows, append([]string{""}, r...))
		}
		akhir := len(rows)
		rows = append(rows, []string{})
		return awal, akhir
	}
	rows = rows[:len(rows)-1] // judul tabel pertama ditulis oleh tabel()
	tabel("AKUN TAMBAHAN YANG DISARANKAN", []string{"Akun", "Letak di template", "Kapan dipakai / contoh"}, [][]string{
		{"Bank (dipisah dari Kas)", "Kas dan Bank", "Tiap rekening punya saldo & rekening koran sendiri, memudahkan rekonsiliasi."},
		{"Cadangan Kerugian Piutang", "Piutang (sel di atas tabel)", "Piutang yang kemungkinan tak tertagih, terutama umur > 90 hari."},
		{"Biaya Dibayar di Muka", "Aset Lancar Lain", "Sewa, asuransi, langganan yang dibayar untuk periode mendatang."},
		{"Pajak Dibayar di Muka", "Aset Lancar Lain", "PPh 23 dipotong pelanggan, PPh 22, angsuran PPh 25."},
		{"Uang Muka Pembelian", "Aset Lancar Lain", "DP ke supplier yang barangnya belum diterima."},
		{"Piutang Lain-lain", "Aset Lancar Lain", "Pinjaman karyawan, piutang pemilik, klaim asuransi."},
		{"Akumulasi Penyusutan", "Aset Tetap (otomatis)", "Garis lurus fiskal dari bulan perolehan s.d. bulan sebelum Tanggal Saldo Awal; bisa ditimpa Akumulasi Manual."},
		{"Utang Pajak", "Utang", "PPh 21/23/4(2)/25/Final dan PPN kurang bayar yang belum disetor."},
		{"Biaya Masih Harus Dibayar", "Utang", "Gaji, listrik, jasa profesional yang sudah terjadi tapi belum dibayar."},
		{"Pendapatan Diterima di Muka", "Utang", "DP pelanggan untuk pesanan yang belum diserahkan."},
		{"Utang Bank / Pinjaman", "Utang", "Sisa pokok pinjaman bank, KUR, leasing."},
		{"Saldo Laba (Laba Ditahan)", "Modal", "Akumulasi laba / rugi tahun lalu — sering jadi angka penyeimbang (minus bila rugi)."},
		{"Prive / Dividen", "Modal", "Pengambilan pemilik atau dividen yang sudah dibagikan; isi angka positif, otomatis mengurangi ekuitas."},
	})
	var kf [][]string
	for _, k := range KelompokAsetList {
		tarif := fmt.Sprintf("Garis lurus %s%% / tahun", formatBps(k.TarifGLBps))
		if k.TarifSMBps > 0 {
			tarif += fmt.Sprintf("; saldo menurun %s%%", formatBps(k.TarifSMBps))
		} else {
			tarif += "; hanya garis lurus"
		}
		kf = append(kf, []string{labelKelompokFiskal(k), strconv.Itoa(k.MasaManfaat), tarif})
	}
	kfAwal, kfAkhir := tabel("DAFTAR KELOMPOK FISKAL (PMK 72/2023) — tanah tidak disusutkan dan belum didukung daftar aset",
		[]string{"Kelompok Fiskal", "Masa Manfaat (bulan)", "Tarif"}, kf)
	var jh [][]string
	for _, j := range JenisHartaList {
		k, _ := KelompokOf(j.Kelompok)
		status := "PERLU DICEK"
		if j.Terverifikasi {
			status = "Terverifikasi"
		}
		jh = append(jh, []string{j.Nama, labelKelompokFiskal(k), j.Lampiran, status})
	}
	jhAwal, jhAkhir := tabel("DAFTAR JENIS HARTA (kolom Jenis Harta di sheet Aset Tetap)",
		[]string{"Jenis Harta", "Kelompok Fiskal", "Rujukan Lampiran PMK", "Status Pemetaan"}, jh)
	var jt [][]string
	for _, j := range JenisList {
		jt = append(jt, []string{j.Nama, j.Arah})
	}
	tabel("DAFTAR JENIS TRANSAKSI (sheet Transaksi)", []string{"Jenis", "Arah uang"}, jt)

	sh := ImportSheet{Nama: sheetPetunjuk, Rows: rows, Format: &FormatTabel{Header: -1, Kolom: 5,
		Lebar: []float64{3, 34, 30, 64, 16}, Gaya: gaya, Status: []string{"C8"}, Kunci: true}}
	return sh, [2]int{jhAwal, jhAkhir}, [2]int{kfAwal, kfAkhir}
}

// formatBps: 2500 → "25", 625 → "6,25".
func formatBps(bps int) string {
	s := strconv.FormatFloat(float64(bps)/100, 'f', -1, 64)
	return strings.ReplaceAll(s, ".", ",")
}

// neracaSheet: Neraca Saldo Awal, seluruhnya rumus dari kolom isian sheet rincian (bukan dari
// kolom rumus) supaya baris yang disisipkan pengguna tetap terhitung. Mengembalikan nomor
// baris Excel baris TOTAL (dipakai rumus saran saldo laba di sheet Modal).
func neracaSheet() (ImportSheet, int) {
	kb := func(k string) string { return rentang(sheetKasBank, k) }
	pt := func(k string) string { return rentang(sheetPiutang, k) }
	al := func(k string) string { return rentang(sheetAsetLancar, k) }
	at := func(k string) string { return rentang(sheetAsetTetap, k) }
	ut := func(k string) string { return rentang(sheetUtang, k) }
	md := func(k string) string { return rentang(sheetModal, k) }
	// Rumus tanpa "=" supaya bisa dijumlah untuk baris "jenis lain".
	sumifs := func(nilai, kriteria, j string) string { return fmt.Sprintf(`SUMIFS(%s,%s,"%s")`, nilai, kriteria, j) }
	kasJ := func(j string) string { return sumifs(kb("F"), kb("C"), j) }
	alJ := func(j string) string { return sumifs(al("F"), al("C"), j) + "-" + sumifs(al("G"), al("C"), j) }
	utJ := func(j string) string { return sumifs(ut("H"), ut("C"), j) + "-" + sumifs(ut("I"), ut("C"), j) }
	mdJ := func(j string) string { return sumifs(md("E"), md("C"), j) }
	// lain: total sheet dikurangi jenis yang punya baris sendiri = jenis yang diketik bebas.
	lain := func(total string, f func(string) string, jenis []string) string {
		var b strings.Builder
		for _, j := range jenis {
			b.WriteString("-(" + f(j) + ")")
		}
		return "=" + total + b.String()
	}

	rows := [][]string{
		{"NERACA SALDO AWAL"},
		{"Terisi otomatis dari sheet rincian — tidak perlu diketik. Status harus SEIMBANG sebelum diunggah."},
		{rumusPerTanggal},
		{}, // Status, diisi setelah baris TOTAL diketahui
		{}, // Selisih
		{"Kode Akun", "Nama Akun", "Kelompok", "Debit (Rp)", "Kredit (Rp)", "Sumber Rincian"},
	}
	gaya := map[string]string{"A3": "catatan", "B4": "tebal", "D4": "rumus_tebal", "F4": "catatan", "B5": "tebal", "D5": "rumus_tebal"}
	grup := func(nama string) {
		r := len(rows) + 1
		for _, c := range "ABCDEF" {
			gaya[fmt.Sprintf("%c%d", c, r)] = "garis_tebal"
		}
		rows = append(rows, []string{nama})
	}
	// kode kosong = tidak ada satu akun yang pasti (mis. Utang Pajak menggabung beberapa akun PPh).
	akun := func(kode, nama, kelompok, debit, kredit, sumber string) {
		rows = append(rows, []string{kode, nama, kelompok, debit, kredit, sumber})
	}
	jenisKas := []string{"Kas", "Bank"}
	grup("ASET")
	akun("1101", "Kas", "Aset Lancar", "="+kasJ("Kas"), "", sheetKasBank)
	akun("1102", "Bank", "Aset Lancar", "="+kasJ("Bank"), "", sheetKasBank)
	akun("", "Kas & bank jenis lain (akun tersendiri)", "Aset Lancar", lain("SUM("+kb("F")+")", kasJ, jenisKas), "", sheetKasBank)
	akun("1201", "Piutang Usaha", "Aset Lancar", fmt.Sprintf("=SUM(%s)-SUM(%s)", pt("F"), pt("G")), "", sheetPiutang)
	akun("1201", "Cadangan Kerugian Piutang (pengurang)", "Aset Lancar", "", "=N(Piutang!$I$4)", sheetPiutang)
	akun("1301", "Persediaan", "Aset Lancar", fmt.Sprintf("=SUMPRODUCT(%s,%s)", rentang(sheetPersediaan, "F"), rentang(sheetPersediaan, "G")), "", sheetPersediaan)
	akun("1302", "Perlengkapan", "Aset Lancar", fmt.Sprintf("=SUMPRODUCT(%s,%s)", rentang(sheetPerlengkapan, "D"), rentang(sheetPerlengkapan, "E")), "", sheetPerlengkapan)
	akun("1401", "Biaya Dibayar di Muka", "Aset Lancar", "="+alJ("Biaya Dibayar di Muka"), "", sheetAsetLancar)
	akun("1403", "Pajak Dibayar di Muka (PPh)", "Aset Lancar", "="+alJ("Pajak Dibayar di Muka"), "", sheetAsetLancar)
	akun("1404", "Uang Muka Pembelian", "Aset Lancar", "="+alJ("Uang Muka Pembelian"), "", sheetAsetLancar)
	akun("", "Piutang Lain-lain", "Aset Lancar", "="+alJ("Piutang Lain-lain"), "", sheetAsetLancar)
	akun("", "Aset lancar jenis lain (akun tersendiri)", "Aset Lancar",
		lain(fmt.Sprintf("SUM(%s)-SUM(%s)", al("F"), al("G")), alJ, jenisAsetLancarList), "", sheetAsetLancar)
	akun("1501", "Aset Tetap (harga perolehan)", "Aset Tetap", fmt.Sprintf("=SUM(%s)", at("F")), "", sheetAsetTetap)
	akun("1502", "Akumulasi Penyusutan & Amortisasi", "Aset Tetap", "", fmt.Sprintf("=SUM(%s)", at("J")), sheetAsetTetap)
	grup("LIABILITAS")
	akun("2101", "Utang Usaha", "Liabilitas", "", "="+utJ("Utang Usaha"), sheetUtang)
	akun("", "Utang Pajak", "Liabilitas", "", "="+utJ("Utang Pajak"), sheetUtang)
	akun("2104", "Beban Yang Masih Harus Dibayar", "Liabilitas", "", "="+utJ("Biaya Masih Harus Dibayar"), sheetUtang)
	akun("", "Pendapatan Diterima di Muka", "Liabilitas", "", "="+utJ("Pendapatan Diterima di Muka"), sheetUtang)
	akun("2105", "Hutang Lain-lain", "Liabilitas", "", "="+utJ("Utang Lain-lain"), sheetUtang)
	akun("2102", "Hutang Bank", "Liabilitas", "", "="+utJ("Utang Bank / Pinjaman"), sheetUtang)
	akun("", "Utang jenis lain (akun tersendiri)", "Liabilitas", "",
		lain(fmt.Sprintf("SUM(%s)-SUM(%s)", ut("H"), ut("I")), utJ, jenisUtangList), sheetUtang)
	grup("EKUITAS")
	akun("3101", "Modal Disetor", "Ekuitas", "", "="+mdJ("Modal Disetor"), sheetModal)
	akun("3101", "Tambahan Modal Disetor", "Ekuitas", "", "="+mdJ("Tambahan Modal Disetor"), sheetModal)
	akun("", "Ekuitas jenis lain", "Ekuitas", "", lain("SUM("+md("E")+")", mdJ, jenisEkuitasList), sheetModal)
	akun("3201", "Saldo Laba (Laba Ditahan)", "Ekuitas", "", "="+mdJ("Saldo Laba"), sheetModal)
	akun("3102", "Prive", "Ekuitas", "="+mdJ("Prive"), "", sheetModal)
	akun("3201", "Dividen", "Ekuitas", "="+mdJ("Dividen"), "", sheetModal)
	akhirAkun := len(rows) // baris Excel terakhir akun
	total := akhirAkun + 1
	rows = append(rows, []string{"", "TOTAL", "", fmt.Sprintf("=SUM(D7:D%d)", akhirAkun), fmt.Sprintf("=SUM(E7:E%d)", akhirAkun), ""})
	gaya[fmt.Sprintf("B%d", total)] = "garis_tebal"
	gaya[fmt.Sprintf("D%d", total)] = "rumus_tebal"
	gaya[fmt.Sprintf("E%d", total)] = "rumus_tebal"
	akhirTabel := len(rows)
	d, e := fmt.Sprintf("$D$%d", total), fmt.Sprintf("$E$%d", total)
	// Status & selisih di atas tabel, supaya terlihat tanpa menggulir ke bawah.
	rows[3] = []string{"", "Status", "", fmt.Sprintf(`=IF(ROUND(%s-%s,0)=0,"SEIMBANG ✓","BELUM SEIMBANG")`, d, e), "",
		"Bila belum seimbang: cek kelengkapan rincian, lalu lihat 'Saran Saldo Laba' di sheet Modal."}
	rows[4] = []string{"", "Selisih (Debit − Kredit)", "", fmt.Sprintf("=%s-%s", d, e)}
	rng := func(kol string) string { return fmt.Sprintf("%s7:%s%d", kol, kol, akhirAkun) }
	rows = append(rows,
		[]string{},
		[]string{"", "Ringkasan Posisi"},
		[]string{"", "Total Aset (bersih)", "", fmt.Sprintf(`=SUMIFS(%s,%s,"Aset*")-SUMIFS(%s,%s,"Aset*")`, rng("D"), rng("C"), rng("E"), rng("C"))},
		[]string{"", "Total Liabilitas", "", fmt.Sprintf(`=SUMIFS(%s,%s,"Liabilitas")`, rng("E"), rng("C"))},
		[]string{"", "Total Ekuitas", "", fmt.Sprintf(`=SUMIFS(%s,%s,"Ekuitas")-SUMIFS(%s,%s,"Ekuitas")`, rng("E"), rng("C"), rng("D"), rng("C"))},
	)
	gaya[fmt.Sprintf("B%d", total+2)] = "tebal"
	return ImportSheet{Nama: sheetNeraca, Rows: rows, Format: &FormatTabel{Header: barisHeaderRinci, Kolom: 6, Akhir: akhirTabel,
		Lebar: []float64{10, 38, 14, 18, 18, 18}, Angka: []int{3, 4}, Gaya: gaya, Status: []string{"D4"}, Referensi: true, Kunci: true}}, total
}

// ---------------------------------------------------------------------------------------
// Parser
// ---------------------------------------------------------------------------------------

// barisRinci: pembaca satu sheet rincian (header dicari lewat colOf, nomor baris Excel dijaga).
type barisRinci struct {
	rows   [][]string
	header int
	cols   map[string]int
}

func bacaRinci(rows [][]string, colOf func(string) string, wajib ...string) (barisRinci, bool) {
	h, cols, ok := rowReader(rows, colOf, wajib)
	return barisRinci{rows: rows, header: h, cols: cols}, ok
}

func (b barisRinci) get(i int, k string) string {
	if c, ok := b.cols[k]; ok && c < len(b.rows[i]) {
		return strings.TrimSpace(b.rows[i][c])
	}
	return ""
}

// kosong: semua kolom isian (keys) kosong — baris siap-isi yang hanya berisi rumus.
func (b barisRinci) kosong(i int, keys ...string) bool {
	return !slices.ContainsFunc(keys, func(k string) bool { return b.get(i, k) != "" })
}

// nominalRinci: angka rupiah, boleh minus ("-1.000" / "(1.000)"); kosong = 0, ada=false.
func nominalRinci(raw string) (n int64, ada, ok bool) {
	if raw == "" {
		return 0, false, true
	}
	neg := strings.HasPrefix(raw, "-") || (strings.HasPrefix(raw, "(") && strings.HasSuffix(raw, ")"))
	v, ok := parseRupiah(strings.Trim(raw, "-() "))
	if neg {
		v = -v
	}
	return v, true, ok
}

// jumlahRinci: kuantitas, boleh desimal ("2.5" dari sel Excel atau "2,5" diketik).
func jumlahRinci(raw string) (float64, bool) {
	if raw == "" {
		return 0, false
	}
	if f, err := strconv.ParseFloat(raw, 64); err == nil {
		return f, true
	}
	f, err := strconv.ParseFloat(strings.ReplaceAll(strings.ReplaceAll(raw, ".", ""), ",", "."), 64)
	return f, err == nil
}

// gabungIsi menggabung teks yang tidak kosong dengan pemisah sep.
func gabungIsi(sep string, s ...string) string {
	var out []string
	for _, x := range s {
		if x = strings.TrimSpace(x); x != "" {
			out = append(out, x)
		}
	}
	return strings.Join(out, sep)
}

// tanggalSaldoAwal membaca Tanggal Saldo Awal di sheet Petunjuk (sel kanan label).
func tanggalSaldoAwal(rows [][]string) (string, int, string) {
	for i, row := range rows {
		for c, v := range row {
			n := norm(v)
			if !strings.Contains(n, "tanggal saldo awal") && !strings.Contains(n, "tanggal mulai") {
				continue
			}
			for _, x := range row[c+1:] {
				if x = strings.TrimSpace(x); x != "" {
					if t, ok := parseTanggal(x); ok {
						return t, i + 1, ""
					}
					return "", i + 1, fmt.Sprintf("Tanggal Saldo Awal %q tidak dikenali (isi tanggal, mis. 2026-01-01)", x)
				}
			}
			return "", i + 1, "Tanggal Saldo Awal belum diisi"
		}
	}
	return "", 0, "baris Tanggal Saldo Awal tidak ditemukan"
}

// parseRinci membaca workbook template rinci ke d.Opening (Rincian per akun), d.Assets, d.Stok.
func parseRinci(sheets []ImportSheet, d *OnboardingData) {
	byKind := map[string]ImportSheet{}
	for _, sh := range sheets {
		k := kindRinci(sh.Nama)
		if k == "" || len(sh.Rows) == 0 {
			continue
		}
		if _, dup := byKind[k]; !dup {
			byKind[k] = sh
			if k != sheetNeraca {
				d.Terbaca = append(d.Terbaca, sh.Nama)
			}
		}
	}
	tanggal := ""
	if sh, ok := byKind[sheetPetunjuk]; ok {
		t, baris, pesan := tanggalSaldoAwal(sh.Rows)
		if pesan != "" {
			d.Errors = append(d.Errors, SheetError{Sheet: sh.Nama, Baris: baris, Pesan: pesan})
		}
		tanggal = t
	} else {
		d.Errors = append(d.Errors, SheetError{Pesan: "sheet Petunjuk (berisi Tanggal Saldo Awal) tidak ditemukan; pakai template terbaru"})
	}

	var lines []SaldoAkun
	baca := func(kind string, f func(sh ImportSheet) ([]SaldoAkun, []ImportError)) {
		sh, ok := byKind[kind]
		if !ok {
			return
		}
		l, errs := f(sh)
		lines = append(lines, l...)
		d.Errors = append(d.Errors, tag(sh.Nama, errs)...)
	}
	baca(sheetKasBank, func(sh ImportSheet) ([]SaldoAkun, []ImportError) { return rinciKas(sh.Rows) })
	baca(sheetPiutang, func(sh ImportSheet) ([]SaldoAkun, []ImportError) { return rinciPiutang(sh.Rows) })
	baca(sheetPersediaan, func(sh ImportSheet) ([]SaldoAkun, []ImportError) {
		l, stok, errs := rinciPersediaan(sh.Rows)
		d.Stok = append(d.Stok, stok...)
		return l, errs
	})
	baca(sheetPerlengkapan, func(sh ImportSheet) ([]SaldoAkun, []ImportError) { return rinciPerlengkapan(sh.Rows) })
	baca(sheetAsetLancar, func(sh ImportSheet) ([]SaldoAkun, []ImportError) { return rinciAsetLancar(sh.Rows) })
	baca(sheetAsetTetap, func(sh ImportSheet) ([]SaldoAkun, []ImportError) {
		if tanggal == "" {
			return nil, nil // tanpa tanggal saldo awal akumulasi tidak bisa dihitung; error sudah dilaporkan
		}
		as, l, errs := rinciAsetTetap(sh.Rows, tanggal)
		d.Assets = append(d.Assets, as...)
		return l, errs
	})
	baca(sheetUtang, func(sh ImportSheet) ([]SaldoAkun, []ImportError) { return rinciUtang(sh.Rows) })
	baca(sheetModal, func(sh ImportSheet) ([]SaldoAkun, []ImportError) { return rinciModal(sh.Rows) })

	if tanggal == "" {
		return
	}
	o := OpeningBalance{Tanggal: tanggal, Rincian: lines}
	if err := o.TerapkanRincian(); err != nil {
		d.Errors = append(d.Errors, SheetError{Sheet: sheetNeraca, Pesan: err.Error()})
		return
	}
	if err := ValidateOpeningBalance(&o); err != nil {
		d.Errors = append(d.Errors, SheetError{Sheet: sheetNeraca, Pesan: err.Error()})
		return
	}
	d.Opening = &o
}

// gagal: pembantu laporan error per baris.
type gagal []ImportError

func (g *gagal) add(i int, format string, a ...any) {
	*g = append(*g, ImportError{i + 1, fmt.Sprintf(format, a...)})
}

func rinciKas(rows [][]string) ([]SaldoAkun, []ImportError) {
	b, ok := bacaRinci(rows, func(h string) string {
		n := norm(h)
		switch {
		case strings.Contains(n, "atas nama"):
			return "atas"
		case strings.Contains(n, "rekening"):
			return "rek"
		case strings.Contains(n, "dasar"):
			return "dasar"
		case strings.Contains(n, "nama"):
			return "nama"
		case strings.Contains(n, "jenis"):
			return "jenis"
		case strings.Contains(n, "saldo") || strings.Contains(n, "nilai"):
			return "saldo"
		case strings.Contains(n, "keterangan"):
			return "ket"
		}
		return ""
	}, "nama", "saldo")
	if !ok {
		return nil, []ImportError{{0, "header tidak ditemukan: butuh kolom Nama Kas / Bank dan Saldo"}}
	}
	var out []SaldoAkun
	var errs gagal
	for i := b.header + 1; i < len(rows); i++ {
		if b.kosong(i, "nama", "jenis", "rek", "saldo") {
			continue
		}
		n, ada, okn := nominalRinci(b.get(i, "saldo"))
		switch {
		case !okn:
			errs.add(i, "saldo %q bukan angka", b.get(i, "saldo"))
			continue
		case !ada:
			errs.add(i, "saldo wajib diisi (0 bila kosong)")
			continue
		case n < 0:
			errs.add(i, "saldo tidak boleh minus; cerukan (overdraft) dicatat di sheet Utang sebagai Utang Bank / Pinjaman")
			continue
		}
		kode, nama := "1101", "Kas"
		switch j := norm(b.get(i, "jenis")); {
		case strings.Contains(j, "bank") || (j == "" && b.get(i, "rek") != ""):
			kode, nama = "1102", "Bank"
		case strings.Contains(j, "kas") || j == "":
		default: // jenis diketik sendiri (deposito, e-wallet, …): akun tersendiri bernama jenisnya
			kode, nama = "", b.get(i, "jenis")
		}
		rek := ""
		if r := b.get(i, "rek"); r != "" {
			rek = gabungIsi(" a.n. ", "Rek. "+r, b.get(i, "atas"))
		}
		akun, ket := cmp.Or(b.get(i, "nama"), nama), gabungIsi("; ", rek, b.get(i, "dasar"), b.get(i, "ket"))
		if kode == "" {
			akun, ket = nama, gabungIsi("; ", b.get(i, "nama"), ket)
		}
		out = append(out, SaldoAkun{Kode: kode, Nama: akun, Kategori: KatKas, Nominal: n, Keterangan: ket})
	}
	return out, errs
}

// sisaRinci: nilai − sudah dibayar/dibebankan, dengan pesan yang menunjuk baris.
func sisaRinci(b barisRinci, i int, errs *gagal, kNilai, kBayar, namaNilai, namaBayar string) (int64, bool) {
	nilai, ada, ok := nominalRinci(b.get(i, kNilai))
	if !ok || !ada || nilai < 0 {
		errs.add(i, "%s %q wajib diisi angka 0 atau lebih", namaNilai, b.get(i, kNilai))
		return 0, false
	}
	bayar, _, ok := nominalRinci(b.get(i, kBayar))
	if !ok || bayar < 0 {
		errs.add(i, "%s %q bukan angka 0 atau lebih", namaBayar, b.get(i, kBayar))
		return 0, false
	}
	if bayar > nilai {
		errs.add(i, "%s (%s) melebihi %s (%s)", namaBayar, formatRpID(bayar), namaNilai, formatRpID(nilai))
		return 0, false
	}
	return nilai - bayar, true
}

// tanggalOpsional: kolom tanggal yang boleh kosong; isi yang tidak dikenali dilaporkan.
func tanggalOpsional(b barisRinci, i int, errs *gagal, k, nama string) (string, bool) {
	raw := b.get(i, k)
	if raw == "" {
		return "", true
	}
	t, ok := parseTanggal(raw)
	if !ok {
		errs.add(i, "%s %q tidak dikenali (isi tanggal, mis. 2026-01-31)", nama, raw)
	}
	return t, ok
}

func rinciPiutang(rows [][]string) ([]SaldoAkun, []ImportError) {
	b, ok := bacaRinci(rows, func(h string) string {
		n := norm(h)
		switch {
		case strings.Contains(n, "jatuh"):
			return "jt"
		case strings.Contains(n, "tgl") || strings.Contains(n, "tanggal"):
			return "tgl"
		case strings.Contains(n, "sudah") || strings.Contains(n, "dibayar"):
			return "bayar"
		case strings.Contains(n, "sisa") || strings.Contains(n, "lewat") || strings.Contains(n, "umur"):
			return "turunan"
		case strings.Contains(n, "nilai"):
			return "nilai"
		case strings.Contains(n, "pelanggan") || strings.Contains(n, "nama"):
			return "pihak"
		case strings.Contains(n, "faktur") || strings.Contains(n, "invoice") || strings.Contains(n, "nomor"):
			return "nomor"
		case strings.Contains(n, "keterangan"):
			return "ket"
		}
		return ""
	}, "pihak", "nilai")
	if !ok {
		return nil, []ImportError{{0, "header tidak ditemukan: butuh kolom Nama Pelanggan dan Nilai Faktur"}}
	}
	var out []SaldoAkun
	var errs gagal
	// Cadangan kerugian: sel tepat di kanan label "Cadangan Kerugian" di atas tabel.
	for i, row := range rows[:b.header] {
		for c, v := range row {
			if !strings.Contains(norm(v), "cadangan") || c+1 >= len(row) {
				continue
			}
			n, ada, okn := nominalRinci(strings.TrimSpace(row[c+1]))
			switch {
			case !okn:
				errs.add(i, "Cadangan Kerugian %q bukan angka", row[c+1])
			case ada && n != 0:
				out = append(out, SaldoAkun{Kode: "1201", Nama: "Cadangan Kerugian Piutang", Kategori: KatPiutang, Nominal: -abs64(n),
					Keterangan: "pengurang piutang usaha (perkiraan tak tertagih)"})
			}
		}
	}
	for i := b.header + 1; i < len(rows); i++ {
		if b.kosong(i, "pihak", "nomor", "tgl", "jt", "nilai", "bayar") {
			continue
		}
		sisa, okS := sisaRinci(b, i, &errs, "nilai", "bayar", "Nilai Faktur", "Sudah Dibayar")
		_, okT := tanggalOpsional(b, i, &errs, "tgl", "Tgl Faktur")
		jt, okJ := tanggalOpsional(b, i, &errs, "jt", "Jatuh Tempo")
		if !okS || !okT || !okJ {
			continue
		}
		if b.get(i, "pihak") == "" {
			errs.add(i, "Nama Pelanggan wajib diisi")
			continue
		}
		if sisa == 0 {
			continue // sudah lunas: bukan piutang saldo awal
		}
		ket := ""
		if jt != "" {
			ket = "jatuh tempo " + jt
		}
		out = append(out, SaldoAkun{Kode: "1201", Nama: gabungIsi(" — ", b.get(i, "pihak"), b.get(i, "nomor")), Kategori: KatPiutang,
			Nominal: sisa, Keterangan: gabungIsi("; ", ket, b.get(i, "ket"))})
	}
	return out, errs
}

// barangRinci: baris Persediaan / Perlengkapan: nama, qty × harga.
func barangRinci(b barisRinci, i int, errs *gagal) (nama string, qty float64, harga, nilai int64, ok bool) {
	nama = b.get(i, "nama")
	if nama == "" {
		errs.add(i, "nama barang wajib diisi")
		return
	}
	qty, okq := jumlahRinci(b.get(i, "qty"))
	if !okq || qty < 0 {
		errs.add(i, "Jumlah (Qty) %q wajib diisi angka 0 atau lebih", b.get(i, "qty"))
		return
	}
	harga, ada, okh := nominalRinci(b.get(i, "harga"))
	if !okh || !ada || harga < 0 {
		errs.add(i, "Harga / Satuan %q wajib diisi angka 0 atau lebih", b.get(i, "harga"))
		return
	}
	return nama, qty, harga, int64(math.Round(qty * float64(harga))), true
}

func kolomBarang(h string) string {
	n := norm(h)
	switch {
	case strings.Contains(n, "harga"):
		return "harga"
	case strings.Contains(n, "nilai"):
		return "nilai"
	case strings.Contains(n, "jumlah") || strings.Contains(n, "qty"):
		return "qty"
	case strings.Contains(n, "kode") || strings.Contains(n, "sku"):
		return "kode"
	case strings.Contains(n, "nama"):
		return "nama"
	case strings.Contains(n, "kategori"):
		return "kategori"
	case strings.Contains(n, "satuan"):
		return "satuan"
	case strings.Contains(n, "lokasi") || strings.Contains(n, "gudang"):
		return "lokasi"
	case strings.Contains(n, "keterangan"):
		return "ket"
	}
	return ""
}

// rinciPersediaan: satu baris saldo awal per kategori barang (rincian per barang masuk Stok &
// Gudang, supaya saldo awal tidak berisi ratusan baris).
func rinciPersediaan(rows [][]string) ([]SaldoAkun, []StokItem, []ImportError) {
	b, ok := bacaRinci(rows, kolomBarang, "nama", "qty", "harga")
	if !ok {
		return nil, nil, []ImportError{{0, "header tidak ditemukan: butuh kolom Nama Barang, Jumlah (Qty), dan Harga Pokok / Satuan"}}
	}
	var stok []StokItem
	var errs gagal
	type grup struct {
		nilai int64
		n     int
	}
	per := map[string]*grup{}
	var urut []string
	for i := b.header + 1; i < len(rows); i++ {
		if b.kosong(i, "nama", "kode", "qty", "harga") {
			continue
		}
		nama, qty, harga, nilai, okb := barangRinci(b, i, &errs)
		if !okb {
			continue
		}
		s := StokItem{SKU: b.get(i, "kode"), Nama: nama, Kategori: b.get(i, "kategori"), Satuan: b.get(i, "satuan"),
			Jumlah: int64(math.Round(qty)), HargaPokok: harga}
		if err := ValidateStok(&s); err != nil {
			errs.add(i, "%s", err)
			continue
		}
		stok = append(stok, s)
		kat := cmp.Or(b.get(i, "kategori"), "Barang")
		if per[kat] == nil {
			per[kat] = &grup{}
			urut = append(urut, kat)
		}
		per[kat].nilai += nilai
		per[kat].n++
	}
	var out []SaldoAkun
	for _, kat := range urut {
		out = append(out, SaldoAkun{Kode: "1301", Nama: "Persediaan — " + kat, Kategori: KatPersediaan, Nominal: per[kat].nilai,
			Keterangan: fmt.Sprintf("%d barang; rincian di Stok & Gudang", per[kat].n)})
	}
	return out, stok, errs
}

func rinciPerlengkapan(rows [][]string) ([]SaldoAkun, []ImportError) {
	b, ok := bacaRinci(rows, kolomBarang, "nama", "qty", "harga")
	if !ok {
		return nil, []ImportError{{0, "header tidak ditemukan: butuh kolom Nama Perlengkapan, Jumlah (Qty), dan Harga / Satuan"}}
	}
	var errs gagal
	var total int64
	n := 0
	for i := b.header + 1; i < len(rows); i++ {
		if b.kosong(i, "nama", "qty", "harga") {
			continue
		}
		_, _, _, nilai, okb := barangRinci(b, i, &errs)
		if !okb {
			continue
		}
		total += nilai
		n++
	}
	if n == 0 {
		return nil, errs
	}
	return []SaldoAkun{{Kode: "1302", Nama: "Perlengkapan", Kategori: KatAsetLain, Nominal: total,
		Keterangan: fmt.Sprintf("%d jenis perlengkapan", n)}}, errs
}

func rinciAsetLancar(rows [][]string) ([]SaldoAkun, []ImportError) {
	b, ok := bacaRinci(rows, func(h string) string {
		n := norm(h)
		switch {
		case strings.Contains(n, "jenis"):
			return "jenis"
		case strings.Contains(n, "sudah") || strings.Contains(n, "dibebankan"):
			return "beban"
		case strings.Contains(n, "sisa"):
			return "sisa"
		case strings.Contains(n, "nilai"):
			return "nilai"
		case strings.Contains(n, "tgl") || strings.Contains(n, "tanggal"):
			return "tgl"
		case strings.Contains(n, "pihak"):
			return "pihak"
		case strings.Contains(n, "uraian") || strings.Contains(n, "nama"):
			return "uraian"
		case strings.Contains(n, "keterangan"):
			return "ket"
		}
		return ""
	}, "jenis", "nilai")
	if !ok {
		return nil, []ImportError{{0, "header tidak ditemukan: butuh kolom Jenis dan Nilai Awal"}}
	}
	var out []SaldoAkun
	var errs gagal
	for i := b.header + 1; i < len(rows); i++ {
		if b.kosong(i, "uraian", "jenis", "pihak", "tgl", "nilai", "beban") {
			continue
		}
		var kode string
		switch j := norm(b.get(i, "jenis")); {
		case strings.Contains(j, "pajak"):
			kode = "1403"
		case strings.Contains(j, "biaya") || strings.Contains(j, "sewa"):
			kode = "1401"
		case strings.Contains(j, "uang muka"):
			kode = "1404"
		case strings.Contains(j, "piutang"):
			kode = "1202"
		}
		sisa, okS := sisaRinci(b, i, &errs, "nilai", "beban", "Nilai Awal", "Sudah Dibebankan")
		if _, okT := tanggalOpsional(b, i, &errs, "tgl", "Tgl Bayar / Dokumen"); !okS || !okT || sisa == 0 {
			continue
		}
		uraian := gabungIsi(" — ", cmp.Or(b.get(i, "uraian"), b.get(i, "jenis")), b.get(i, "pihak"))
		r := SaldoAkun{Kode: kode, Nama: uraian, Kategori: KatAsetLain, Nominal: sisa, Keterangan: b.get(i, "ket")}
		if kode == "" { // jenis diketik sendiri: akun tersendiri bernama jenisnya
			r.Nama, r.Keterangan = b.get(i, "jenis"), gabungIsi("; ", uraian, r.Keterangan)
		}
		out = append(out, r)
	}
	return out, errs
}

// rinciAsetTetap: daftar aset (register) + satu baris saldo awal aset tetap bila semua baris
// punya Nilai Buku tersimpan; tanpa itu nilai buku dihitung sistem (LengkapiAsetTetap).
func rinciAsetTetap(rows [][]string, tanggal string) ([]Asset, []SaldoAkun, []ImportError) {
	b, ok := bacaRinci(rows, func(h string) string {
		n := norm(h)
		switch {
		case strings.Contains(n, "manual"):
			return "manual"
		case strings.Contains(n, "dipakai"):
			return "dipakai"
		case strings.Contains(n, "dihitung"):
			return "dihitung"
		case strings.Contains(n, "nilai buku"):
			return "nbv"
		case strings.Contains(n, "masa") || strings.Contains(n, "umur"):
			return "masa"
		case strings.Contains(n, "harga"):
			return "harga"
		case strings.Contains(n, "tgl") || strings.Contains(n, "tanggal"):
			return "tgl"
		case strings.Contains(n, "kelompok"):
			return "kelompok"
		case strings.Contains(n, "jenis"):
			return "jenis"
		case strings.Contains(n, "nama"):
			return "nama"
		case strings.Contains(n, "lokasi") || strings.Contains(n, "pemakai"):
			return "lokasi"
		case strings.Contains(n, "keterangan"):
			return "ket"
		}
		return ""
	}, "nama", "tgl", "harga")
	if !ok {
		return nil, nil, []ImportError{{0, "header tidak ditemukan: butuh kolom Nama Aset, Tgl Perolehan, dan Harga Perolehan"}}
	}
	var out []Asset
	var errs gagal
	var nbvTotal int64
	semuaNBV := true
	sebelum := prevPeriod(tanggal[:7])
	for i := b.header + 1; i < len(rows); i++ {
		if b.kosong(i, "nama", "jenis", "kelompok", "tgl", "harga", "manual") {
			continue
		}
		a := Asset{Nama: b.get(i, "nama"), Metode: MetodeGarisLurus, Keterangan: gabungIsi("; ", b.get(i, "lokasi"), b.get(i, "ket"))}
		if a.Nama == "" {
			errs.add(i, "Nama Aset wajib diisi")
			continue
		}
		if strings.Contains(norm(b.get(i, "kelompok")+" "+b.get(i, "jenis")), "tanah") {
			errs.add(i, "tanah tidak disusutkan dan belum didukung daftar aset; keluarkan dari sheet ini (catat terpisah bersama konsultan)")
			continue
		}
		j, okJ := findJenisHarta(b.get(i, "jenis"))
		if kel := b.get(i, "kelompok"); kel != "" {
			k, okK := findKelompok(kel)
			if !okK {
				errs.add(i, "Kelompok Fiskal %q tidak dikenal; pilih dari daftar", kel)
				continue
			}
			a.Kelompok = k.ID
			if okJ {
				a.TakBerwujud = j.TakBerwujud
				if j.Kelompok == k.ID {
					a.JenisHarta = j.ID
				}
			}
		} else if okJ {
			a.JenisHarta = j.ID
		} else {
			errs.add(i, "isi Kelompok Fiskal (atau Jenis Harta dari daftar di Petunjuk)")
			continue
		}
		t, okT := parseTanggal(b.get(i, "tgl"))
		if !okT {
			errs.add(i, "Tgl Perolehan %q tidak dikenali (isi tanggal, mis. 2025-01-31)", b.get(i, "tgl"))
			continue
		}
		a.TanggalPerolehan = t
		if t[:7] >= tanggal[:7] {
			errs.add(i, "diperoleh %s, tidak sebelum bulan Tanggal Saldo Awal (%s): catat lewat menu Aset / transaksi Beli Alat, bukan saldo awal", t, tanggal)
			continue
		}
		harga, ada, okH := nominalRinci(b.get(i, "harga"))
		if !okH || !ada || harga <= 0 {
			errs.add(i, "Harga Perolehan %q wajib diisi angka lebih dari 0", b.get(i, "harga"))
			continue
		}
		a.HargaPerolehan = harga
		// Akumulasi awal: manual bila diisi, selain itu nilai rumus yang tersimpan di berkas.
		kAkum := ""
		for _, k := range []string{"manual", "dipakai", "dihitung"} {
			if b.get(i, k) != "" {
				kAkum = k
				break
			}
		}
		if kAkum != "" {
			v, _, okA := nominalRinci(b.get(i, kAkum))
			if !okA || v < 0 || v > harga {
				errs.add(i, "Akumulasi %q harus angka antara 0 dan harga perolehan", b.get(i, kAkum))
				continue
			}
			a.AkumulasiAwal, a.PerTanggal = v, tanggal
		}
		if err := ValidateAsset(&a); err != nil {
			errs.add(i, "%s", err)
			continue
		}
		if k, _ := KelompokOf(a.Kelompok); kAkum == "manual" && monthsElapsed(a.TanggalPerolehan, sebelum) >= k.MasaManfaat && a.AkumulasiAwal != harga {
			errs.add(i, "masa manfaat fiskal (%d bulan) sudah habis per Tanggal Saldo Awal: akumulasi harus sama dengan harga perolehan", k.MasaManfaat)
			continue
		}
		if v, ada, okN := nominalRinci(b.get(i, "nbv")); ada && okN {
			nbvTotal += v
		} else {
			semuaNBV = false
		}
		out = append(out, a)
	}
	var lines []SaldoAkun
	if len(out) > 0 && semuaNBV {
		lines = append(lines, SaldoAkun{Kode: "1501", Nama: "Aset tetap (nilai buku)", Kategori: KatAsetTetap, Nominal: nbvTotal,
			Keterangan: fmt.Sprintf("%d aset; rincian di sheet Aset Tetap", len(out))})
	}
	return out, lines, errs
}

func rinciUtang(rows [][]string) ([]SaldoAkun, []ImportError) {
	b, ok := bacaRinci(rows, func(h string) string {
		n := norm(h)
		switch {
		case strings.Contains(n, "jenis"):
			return "jenis"
		case strings.Contains(n, "jatuh"):
			return "jt"
		case strings.Contains(n, "tgl") || strings.Contains(n, "tanggal"):
			return "tgl"
		case strings.Contains(n, "dokumen") || strings.Contains(n, "nomor"):
			return "nomor"
		case strings.Contains(n, "sudah") || strings.Contains(n, "dibayar"):
			return "bayar"
		case strings.Contains(n, "sisa"):
			return "sisa"
		case strings.Contains(n, "nilai"):
			return "nilai"
		case strings.Contains(n, "kreditur") || strings.Contains(n, "pihak"):
			return "pihak"
		case strings.Contains(n, "uraian") || strings.Contains(n, "nama"):
			return "uraian"
		case strings.Contains(n, "keterangan"):
			return "ket"
		}
		return ""
	}, "jenis", "nilai")
	if !ok {
		return nil, []ImportError{{0, "header tidak ditemukan: butuh kolom Jenis Utang dan Nilai Awal"}}
	}
	var out []SaldoAkun
	var errs gagal
	for i := b.header + 1; i < len(rows); i++ {
		if b.kosong(i, "uraian", "jenis", "pihak", "nomor", "tgl", "jt", "nilai", "bayar") {
			continue
		}
		var kode, kat string
		switch j := norm(b.get(i, "jenis")); {
		case strings.Contains(j, "bank") || strings.Contains(j, "pinjaman") || strings.Contains(j, "leasing"):
			kode, kat = "2102", KatPinjaman
		case strings.Contains(j, "usaha") || strings.Contains(j, "dagang") || strings.Contains(j, "supplier"):
			kode, kat = "2101", KatUtang
		case strings.Contains(j, "pajak"):
			kode, kat = "2110", KatKewajibanLain
		case strings.Contains(j, "masih harus") || strings.Contains(j, "biaya"):
			kode, kat = "2104", KatKewajibanLain
		case strings.Contains(j, "diterima di muka") || strings.Contains(j, "diterima dimuka") || strings.Contains(j, "pendapatan"):
			kode, kat = "2105", KatKewajibanLain
		case strings.Contains(j, "lain"):
			kode, kat = "2105", KatKewajibanLain
		case strings.Contains(j, "kredit") || strings.Contains(j, "koperasi") || strings.Contains(j, "cicilan"):
			kat = KatPinjaman // akun pinjaman; jenisnya jadi nama baris
		default: // jenis diketik sendiri: akun tersendiri bernama jenisnya
			kat = KatKewajibanLain
		}
		sisa, okS := sisaRinci(b, i, &errs, "nilai", "bayar", "Nilai Awal", "Sudah Dibayar")
		_, okT := tanggalOpsional(b, i, &errs, "tgl", "Tgl Dokumen")
		jt, okJ := tanggalOpsional(b, i, &errs, "jt", "Jatuh Tempo")
		if !okS || !okT || !okJ || sisa == 0 {
			continue
		}
		ket := ""
		if jt != "" {
			ket = "jatuh tempo " + jt
		}
		uraian := gabungIsi(" — ", cmp.Or(b.get(i, "uraian"), b.get(i, "jenis")), b.get(i, "pihak"))
		r := SaldoAkun{Kode: kode, Nama: uraian, Kategori: kat, Nominal: sisa, Keterangan: gabungIsi("; ", b.get(i, "nomor"), ket, b.get(i, "ket"))}
		if kode == "" {
			r.Nama, r.Keterangan = b.get(i, "jenis"), gabungIsi("; ", uraian, r.Keterangan)
		}
		out = append(out, r)
	}
	return out, errs
}

func rinciModal(rows [][]string) ([]SaldoAkun, []ImportError) {
	b, ok := bacaRinci(rows, func(h string) string {
		n := norm(h)
		switch {
		case strings.Contains(n, "saran"):
			return "" // kolom rumus penyeimbang, bukan isian
		case strings.Contains(n, "jenis"):
			return "jenis"
		case strings.Contains(n, "dasar") || strings.Contains(n, "dokumen"):
			return "dasar"
		case strings.Contains(n, "nilai") || strings.Contains(n, "jumlah"):
			return "nilai"
		case strings.Contains(n, "nama") || strings.Contains(n, "uraian"):
			return "nama"
		case strings.Contains(n, "keterangan"):
			return "ket"
		}
		return ""
	}, "jenis", "nilai")
	if !ok {
		return nil, []ImportError{{0, "header tidak ditemukan: butuh kolom Jenis Ekuitas dan Nilai"}}
	}
	var out []SaldoAkun
	var errs gagal
	for i := b.header + 1; i < len(rows); i++ {
		if b.kosong(i, "nama", "jenis", "dasar", "nilai") {
			continue
		}
		n, ada, okn := nominalRinci(b.get(i, "nilai"))
		if !okn || !ada {
			errs.add(i, "Nilai %q wajib diisi angka", b.get(i, "nilai"))
			continue
		}
		jenis := b.get(i, "jenis")
		s := SaldoAkun{Nama: gabungIsi(" — ", jenis, b.get(i, "nama")), Keterangan: gabungIsi("; ", b.get(i, "dasar"), b.get(i, "ket")), Nominal: n}
		switch j := norm(jenis); {
		case strings.Contains(j, "prive") || strings.Contains(j, "dividen"):
			// Pengambilan pemilik / dividen mengurangi ekuitas; diisi positif di Excel.
			s.Kode, s.Kategori, s.Nominal = "3102", KatModal, -abs64(n)
		case strings.Contains(j, "laba") || strings.Contains(j, "rugi"):
			s.Kode, s.Kategori = "3201", KatLabaDitahan // boleh minus (rugi)
		case strings.Contains(j, "modal"):
			s.Kode, s.Kategori = "3101", KatModal
		default: // jenis diketik sendiri (agio saham, …): bagian modal
			s.Kategori = KatModal
		}
		if n < 0 && s.Kategori == KatModal {
			errs.add(i, "Nilai %s tidak boleh minus; prive / dividen diisi positif dengan Jenis Prive / Dividen", jenis)
			continue
		}
		out = append(out, s)
	}
	return out, errs
}
