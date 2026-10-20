package finance

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Impor mutasi rekening bank. Mutasi bank tidak menyebut jenis transaksi, hanya keterangan
// apa adanya dari bank ("TRSF E-BANKING CR", "BIAYA ADM", "QRIS CR"), jadi jenisnya ditebak
// dari kata kunci di keterangan itu.
//
// Tebakan tidak pernah langsung dibukukan. Setiap baris membawa tingkat keyakinan dan alasan
// kenapa ia ditebak begitu, lalu seluruhnya ditampilkan untuk dikoreksi sebelum disimpan —
// salah kelompok di sini bukan cuma salah laporan, tapi bisa menggelembungkan omzet yang
// jadi dasar PPh final.

// Tingkat keyakinan tebakan.
const (
	KeyakinanTinggi = "tinggi" // kata kunci khas, mis. "BIAYA ADM", "GAJI", "QRIS"
	KeyakinanSedang = "sedang" // biasanya benar, tapi tergantung konteks usaha
	KeyakinanRendah = "rendah" // hanya arah uangnya yang pasti; jenisnya wajib diperiksa
	KeyakinanAI     = "ai"     // baris rendah yang jenisnya disarankan AI; tetap bisa dikoreksi
)

// MutasiRow: satu baris mutasi beserta tebakan jenisnya.
type MutasiRow struct {
	Baris      int    `json:"baris"` // nomor baris di berkas, supaya bisa ditunjuk saat keliru
	Tanggal    string `json:"tanggal"`
	Keterangan string `json:"keterangan"`
	Nominal    int64  `json:"nominal"`
	Arah       string `json:"arah"`  // "masuk" | "keluar"
	Jenis      string `json:"jenis"` // tebakan; kosong = harus dipilih sendiri
	Alasan     string `json:"alasan"`
	Keyakinan  string `json:"keyakinan"`
	// Duplikat: sudah ada transaksi dengan tanggal & nominal sama. Tidak selalu benar
	// (dua penjualan bisa sama nominal di hari yang sama), karena itu hanya ditandai.
	Duplikat bool `json:"duplikat"`
	// Pilih dipakai saat commit: baris yang tidak dipilih tidak dibukukan.
	Pilih bool `json:"pilih"`
}

// MutasiGrup: rekap per jenis untuk diperiksa sebelum submit.
type MutasiGrup struct {
	Jenis  string `json:"jenis"`
	Arah   string `json:"arah"`
	Jumlah int    `json:"jumlah"`
	Total  int64  `json:"total"`
}

type MutasiHasil struct {
	Rows        []MutasiRow   `json:"rows"`
	Errors      []ImportError `json:"errors"`
	Ringkasan   []MutasiGrup  `json:"ringkasan"`
	TotalMasuk  int64         `json:"total_masuk"`
	TotalKeluar int64         `json:"total_keluar"`
	// PerluDiperiksa: baris berkeyakinan rendah atau belum punya jenis.
	PerluDiperiksa int `json:"perlu_diperiksa"`
	Duplikat       int `json:"duplikat"`
	// DibantuAI: baris berkeyakinan rendah yang jenisnya disarankan AI. AIPesan: kenapa AI
	// tidak dipakai (belum aktif, kuota habis, gagal) — kosong bila dipakai atau tidak perlu.
	DibantuAI int    `json:"dibantu_ai"`
	AIPesan   string `json:"ai_pesan,omitempty"`
	// Format menyebut kolom apa yang dikenali, supaya jelas berkasnya terbaca sebagai apa.
	Format string `json:"format"`
	// PerBulan: jumlah baris per bulan, supaya pengguna bisa memastikan mutasinya masuk ke
	// bulan yang benar sebelum dibukukan. Peringatan: hal yang perlu dicek (format tanggal
	// terbalik, tanggal di masa depan, pergantian tahun).
	PerBulan   []MutasiBulan `json:"per_bulan"`
	Peringatan []string      `json:"peringatan,omitempty"`
}

type MutasiBulan struct {
	Bulan  string `json:"bulan"` // YYYY-MM
	Label  string `json:"label"`
	Jumlah int    `json:"jumlah"`
}

type aturan struct {
	kata  []string
	jenis string
	// arah membatasi aturan ke satu arah uang saja; kosong = berlaku dua arah.
	arah      string
	keyakinan string
	alasan    string
}

// aturanMutasi diperiksa berurutan, yang pertama cocok dipakai. Urutannya penting: kata yang
// lebih khas harus di atas kata yang lebih umum — "pajak bunga" (debit pajak atas jasa giro)
// memuat "bunga", dan "biaya adm" memuat "adm".
var aturanMutasi = []aturan{
	// Keluar — penanda khas bank
	{[]string{"pajak bunga", "pjk bunga", "pph bunga"}, "Pajak & Izin", "keluar", KeyakinanTinggi, "pajak atas bunga bank"},
	{[]string{"biaya adm", "biaya admin", "adm", "administrasi", "biaya transfer", "biaya rtgs", "biaya sms", "biaya notifikasi", "materai", "biaya kartu", "biaya buku"}, "Biaya Lain", "keluar", KeyakinanTinggi, "biaya administrasi bank"},
	{[]string{"pajak", "pph", "ppn", "djp", "e billing", "ebilling", "mpn", "bea", "retribusi", "perizinan", "izin", "oss"}, "Pajak & Izin", "keluar", KeyakinanTinggi, "pembayaran pajak atau perizinan"},
	{[]string{"gaji", "payroll", "upah", "thr", "bonus karyawan", "bpjs"}, "Gaji & Upah", "keluar", KeyakinanTinggi, "pembayaran gaji atau tunjangan"},
	{[]string{"sewa", "rent", "kontrakan", "ruko"}, "Sewa Tempat", "keluar", KeyakinanTinggi, "pembayaran sewa"},
	{[]string{"pln", "listrik", "token listrik", "pdam", "air minum", "indihome", "telkom", "biznet", "first media", "myrepublic", "internet", "wifi", "telkomsel", "pulsa", "indosat", "xl axiata"}, "Listrik, Air, Internet", "keluar", KeyakinanTinggi, "tagihan utilitas atau internet"},
	{[]string{"angsuran", "cicilan", "leasing", "installment", "kredit kendaraan", "adira", "fif", "bussan"}, "Bayar Cicilan", "keluar", KeyakinanTinggi, "angsuran pinjaman atau leasing"},
	{[]string{"iklan", "ads", "facebook", "meta platforms", "google ads", "tiktok ads", "promosi", "influencer", "endorse"}, "Iklan & Promosi", "keluar", KeyakinanTinggi, "belanja iklan atau promosi"},
	{[]string{"ongkir", "jne", "j t express", "jnt", "sicepat", "anteraja", "ninja xpress", "gosend", "grabexpress", "lalamove", "deliveree", "tol", "parkir", "bbm", "pertamina", "shell", "bensin", "solar"}, "Ongkir & Transport", "keluar", KeyakinanTinggi, "biaya kirim atau transportasi"},
	{[]string{"kemasan", "packaging", "sablon", "paper bag", "plastik", "cup", "dus", "kardus", "label"}, "Kemasan", "keluar", KeyakinanTinggi, "belanja kemasan"},
	{[]string{"service", "servis", "perbaikan", "maintenance", "sparepart", "teknisi", "atk", "alat tulis", "perlengkapan", "cleaning"}, "Perawatan & Perlengkapan", "keluar", KeyakinanTinggi, "perawatan atau perlengkapan"},
	{[]string{"roastery", "supplier", "grosir", "indogrosir", "lotte", "pasar", "sayur", "daging", "ayam", "beras", "gula", "susu", "bahan baku", "biji kopi", "distributor"}, "Beli Bahan / Stok", "keluar", KeyakinanSedang, "nama pemasok atau bahan baku"},
	{[]string{"tarik tunai", "penarikan", "atm", "cash withdrawal", "prive", "uang pribadi"}, "Ambil Uang Pribadi", "keluar", KeyakinanRendah, "tarik tunai — bisa jadi kas operasional, bukan prive; pastikan sendiri"},

	// Masuk
	{[]string{"bunga", "jasa giro", "interest"}, "Pendapatan Lain", "masuk", KeyakinanTinggi, "bunga atau jasa giro bank"},
	{[]string{"qris", "edc", "settlement", "setoran tunai", "setor tunai", "kartu debit", "kartu kredit"}, "Penjualan", "masuk", KeyakinanTinggi, "setelan mesin kasir atau setoran penjualan"},
	{[]string{"gofood", "grabfood", "shopeefood", "gojek", "grab", "shopee", "tokopedia", "tiktok", "blibli", "lazada", "bukalapak", "marketplace"}, "Penjualan", "masuk", KeyakinanTinggi, "pencairan dari aplikasi atau marketplace"},
	{[]string{"refund", "pengembalian", "reversal", "koreksi"}, "Pendapatan Lain", "masuk", KeyakinanSedang, "pengembalian dana"},
	{[]string{"setor modal", "modal", "injeksi", "tambahan modal"}, "Setor Modal", "masuk", KeyakinanSedang, "setoran modal pemilik"},
	{[]string{"pinjaman", "kur", "pencairan kredit", "modal kerja", "pencairan"}, "Terima Pinjaman", "masuk", KeyakinanSedang, "pencairan pinjaman"},
}

// cocok mencari kata kunci di keterangan yang sudah dinormalkan. Kata pendek (≤4 huruf)
// dicocokkan sebagai kata utuh, supaya "adm" tidak ikut cocok di tengah kata lain.
func cocok(keterangan, kata string) bool {
	if len(kata) <= 4 && !strings.Contains(kata, " ") {
		return strings.Contains(" "+keterangan+" ", " "+kata+" ")
	}
	return strings.Contains(keterangan, kata)
}

// TebakJenis menebak jenis transaksi dari keterangan mutasi dan arah uangnya.
//
// Tanpa kata kunci yang cocok, jenisnya tidak dikosongkan tapi diisi default per arah dengan
// keyakinan rendah: baris yang kosong jenisnya mudah terlewat saat diperiksa, sementara baris
// berlabel "rendah" tampil menonjol dan tetap tidak bisa lolos tanpa dilihat.
func TebakJenis(keterangan, arah string) (jenis, keyakinan, alasan string) {
	k := norm(keterangan)
	for _, a := range aturanMutasi {
		if a.arah != "" && a.arah != arah {
			continue
		}
		for _, kata := range a.kata {
			if cocok(k, kata) {
				return a.jenis, a.keyakinan, a.alasan + ` (kata kunci "` + kata + `")`
			}
		}
	}
	if arah == "masuk" {
		return "Penjualan", KeyakinanRendah, "uang masuk tanpa penanda yang dikenali; periksa pengirimnya — bisa jadi setoran modal atau pinjaman, bukan penjualan"
	}
	return "Biaya Lain", KeyakinanRendah, "uang keluar tanpa penanda yang dikenali; pilih jenis bebannya sendiri"
}

// mutasiKolom mengenali header mutasi bank. Nama kolomnya berbeda antar bank: Keterangan /
// Uraian / Berita, Mutasi / Nominal, Debet-Kredit terpisah atau satu kolom bertanda DB/CR.
func mutasiKolom(h string) string {
	n := norm(h)
	switch {
	case strings.Contains(n, "tanggal") || n == "tgl" || strings.Contains(n, "date"):
		return "tanggal"
	case strings.Contains(n, "keterangan") || strings.Contains(n, "uraian") || strings.Contains(n, "berita") || strings.Contains(n, "deskripsi") || strings.Contains(n, "description") || strings.Contains(n, "transaksi"):
		return "keterangan"
	case strings.Contains(n, "debet") || strings.Contains(n, "debit"):
		return "debet"
	case strings.Contains(n, "kredit") || strings.Contains(n, "credit"):
		return "kredit"
	case strings.Contains(n, "saldo") || strings.Contains(n, "balance"):
		return "saldo"
	case strings.Contains(n, "mutasi") || strings.Contains(n, "nominal") || strings.Contains(n, "jumlah") || strings.Contains(n, "amount"):
		return "mutasi"
	case n == "db cr" || n == "dbcr" || strings.Contains(n, "tipe") || strings.Contains(n, "type"):
		return "tanda"
	}
	return ""
}

// tandaArah membaca penanda arah uang pada kolom mutasi gabungan ("DB", "CR", "D", "K").
func tandaArah(s string) string {
	switch norm(s) {
	case "cr", "c", "k", "kredit", "credit", "masuk", "in":
		return "masuk"
	case "db", "d", "dr", "debet", "debit", "keluar", "out":
		return "keluar"
	}
	return ""
}

// ParseMutasi membaca baris mutasi bank jadi MutasiRow beserta tebakan jenisnya.
//
// tahunDefault dipakai bila tanggalnya tanpa tahun ("01/09"), yang biasa pada cetakan mutasi
// bulanan; 0 berarti pakai tahun berjalan. existing dipakai menandai kemungkinan duplikat.
func ParseMutasi(rows [][]string, tahunDefault int, existing []Transaction) MutasiHasil {
	var h MutasiHasil
	headerAt, cols := -1, map[string]int{}
	// Header mutasi bank sering didahului beberapa baris kop (nama bank, nomor rekening,
	// periode), jadi dicari sampai 15 baris pertama, bukan hanya baris pertama.
	for i := 0; i < len(rows) && i < 15 && headerAt < 0; i++ {
		m := map[string]int{}
		for c, cell := range rows[i] {
			if k := mutasiKolom(cell); k != "" {
				if _, ada := m[k]; !ada {
					m[k] = c
				}
			}
		}
		_, adaTanggal := m["tanggal"]
		_, adaKet := m["keterangan"]
		punyaNominal := false
		for _, k := range []string{"debet", "kredit", "mutasi"} {
			if _, ok := m[k]; ok {
				punyaNominal = true
			}
		}
		if adaTanggal && punyaNominal && (adaKet || len(m) >= 3) {
			headerAt, cols = i, m
		}
	}
	if headerAt < 0 {
		h.Errors = append(h.Errors, ImportError{Pesan: "kolom mutasi tidak dikenali; berkas harus punya kolom Tanggal, Keterangan, dan Debet/Kredit atau Mutasi"})
		return h
	}

	terbaca := make([]string, 0, len(cols))
	for _, k := range []string{"tanggal", "keterangan", "debet", "kredit", "mutasi", "tanda", "saldo"} {
		if _, ok := cols[k]; ok {
			terbaca = append(terbaca, k)
		}
	}
	h.Format = strings.Join(terbaca, ", ")

	if tahunDefault == 0 {
		tahunDefault = time.Now().Year()
	}
	// Kunci duplikat: tanggal + nominal. Keterangan tidak ikut karena tulisan bank hampir
	// pasti berbeda dari keterangan yang diketik manual untuk transaksi yang sama.
	sudahAda := map[string]bool{}
	for _, t := range existing {
		sudahAda[t.Tanggal+"|"+strconv.FormatInt(t.Nominal, 10)] = true
	}

	get := func(row []string, key string) string {
		c, ok := cols[key]
		if !ok || c >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[c])
	}

	// Urutan hari/bulan dibaca dari seluruh berkas: satu tanggal seperti 01/25 sudah cukup
	// membuktikan formatnya bulan/hari. Tanpa bukti, dianggap hari/bulan (format bank Indonesia).
	var semuaTanggal []string
	for i := headerAt + 1; i < len(rows); i++ {
		semuaTanggal = append(semuaTanggal, get(rows[i], "tanggal"))
	}
	bulanDulu := urutanBulanDulu(semuaTanggal)
	if bulanDulu {
		h.Peringatan = append(h.Peringatan, "Tanggal di berkas ini berformat bulan/tanggal (mis. 01/25/2026), jadi dibaca sebagai bulan/tanggal. Cek kolom tanggal di bawah.")
	}
	tahunDari := tahunUntukBulan(semuaTanggal, tahunDefault, bulanDulu)

	for i := headerAt + 1; i < len(rows); i++ {
		row := rows[i]
		noBaris := i + 1
		if kosong(row) {
			continue
		}
		tgl, ok := parseTanggalMutasi(get(row, "tanggal"), tahunDari, bulanDulu)
		if !ok {
			// Baris kaki cetakan ("Saldo Akhir", "Mutasi Kredit") tidak punya tanggal:
			// dilewati tanpa dilaporkan sebagai kesalahan.
			if get(row, "keterangan") != "" && get(row, "tanggal") != "" {
				h.Errors = append(h.Errors, ImportError{Baris: noBaris, Pesan: "tanggal tidak terbaca: " + get(row, "tanggal")})
			}
			continue
		}
		nominal, arah := bacaNominal(get(row, "debet"), get(row, "kredit"), get(row, "mutasi"), get(row, "tanda"))
		if nominal <= 0 {
			h.Errors = append(h.Errors, ImportError{Baris: noBaris, Pesan: "nominal tidak terbaca atau nol"})
			continue
		}
		if arah == "" {
			h.Errors = append(h.Errors, ImportError{Baris: noBaris, Pesan: "arah uang tidak jelas: tidak ada kolom Debet/Kredit maupun penanda DB/CR"})
			continue
		}
		ket := get(row, "keterangan")
		jenis, yakin, alasan := TebakJenis(ket, arah)
		r := MutasiRow{
			Baris: noBaris, Tanggal: tgl, Keterangan: ket, Nominal: nominal, Arah: arah,
			Jenis: jenis, Keyakinan: yakin, Alasan: alasan, Pilih: true,
			Duplikat: sudahAda[tgl+"|"+strconv.FormatInt(nominal, 10)],
		}
		if r.Duplikat {
			h.Duplikat++
			r.Pilih = false // dugaan duplikat tidak dicentang; pengguna yang memutuskan
		}
		if yakin == KeyakinanRendah {
			h.PerluDiperiksa++
		}
		if arah == "masuk" {
			h.TotalMasuk += nominal
		} else {
			h.TotalKeluar += nominal
		}
		h.Rows = append(h.Rows, r)
	}
	h.Ringkasan = rekapMutasi(h.Rows)
	hariIni := time.Now().Format(time.DateOnly)
	masaDepan := 0
	for _, r := range h.Rows {
		if r.Tanggal > hariIni {
			masaDepan++
		}
		b := r.Tanggal[:7]
		if n := len(h.PerBulan); n > 0 && h.PerBulan[n-1].Bulan == b {
			h.PerBulan[n-1].Jumlah++
			continue
		}
		if i := slices.IndexFunc(h.PerBulan, func(x MutasiBulan) bool { return x.Bulan == b }); i >= 0 {
			h.PerBulan[i].Jumlah++
			continue
		}
		h.PerBulan = append(h.PerBulan, MutasiBulan{Bulan: b, Label: LabelPeriode(b), Jumlah: 1})
	}
	slices.SortFunc(h.PerBulan, func(a, b MutasiBulan) int { return strings.Compare(a.Bulan, b.Bulan) })
	if masaDepan > 0 {
		h.Peringatan = append(h.Peringatan, fmt.Sprintf("%d baris bertanggal di masa depan — cek isian Tahun mutasi atau format tanggal berkas.", masaDepan))
	}
	return h
}

var polaHariBulan = regexp.MustCompile(`^(\d{1,2})[/-](\d{1,2})(?:[/-]\d{2,4})?$`)

// urutanBulanDulu: true bila ada tanggal yang hanya masuk akal sebagai bulan/hari (bagian
// kedua > 12) dan tidak ada yang hanya masuk akal sebagai hari/bulan.
func urutanBulanDulu(tanggal []string) bool {
	hariDulu, bulanDulu := false, false
	for _, s := range tanggal {
		m := polaHariBulan.FindStringSubmatch(strings.TrimSpace(s))
		if m == nil {
			continue
		}
		a, _ := strconv.Atoi(m[1])
		b, _ := strconv.Atoi(m[2])
		hariDulu = hariDulu || a > 12
		bulanDulu = bulanDulu || b > 12
	}
	return bulanDulu && !hariDulu
}

// tahunUntukBulan: tahun untuk tanggal tanpa tahun. Bila berkas melewati pergantian tahun
// (ada Desember dan Januari), tahun yang diisi pengguna adalah tahun bagian Januari-nya, jadi
// bulan Juli–Desember mundur satu tahun.
func tahunUntukBulan(tanggal []string, tahun int, bulanDulu bool) func(time.Month) int {
	ada := map[time.Month]bool{}
	for _, s := range tanggal {
		if t, ok := tanggalTanpaTahun(s, bulanDulu); ok {
			ada[t.Month()] = true
		}
	}
	lintas := ada[time.December] && ada[time.January]
	return func(m time.Month) int {
		if lintas && m >= time.July {
			return tahun - 1
		}
		return tahun
	}
}

func tanggalTanpaTahun(s string, bulanDulu bool) (time.Time, bool) {
	s = strings.TrimSpace(s)
	layouts := []string{"02/01", "2/1", "02-01", "2-1"}
	if bulanDulu {
		layouts = []string{"01/02", "1/2", "01-02", "1-2"}
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	if t, err := time.Parse("02 Jan", s); err == nil {
		return t, true
	}
	return time.Time{}, false
}

func kosong(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// bacaNominal menangani dua bentuk: kolom Debet & Kredit terpisah, atau satu kolom Mutasi
// dengan penanda DB/CR (bisa di kolom sendiri atau menempel di angkanya, "150.000 DB").
func bacaNominal(debet, kredit, mutasi, tanda string) (int64, string) {
	if n, ok := parseRupiah(debet); ok && n > 0 {
		return n, "keluar"
	}
	if n, ok := parseRupiah(kredit); ok && n > 0 {
		return n, "masuk"
	}
	if mutasi == "" {
		return 0, ""
	}
	arah := tandaArah(tanda)
	m := mutasi
	// Penanda menempel di angka: "1.250.000 DB" atau "1.250.000CR".
	for _, suf := range []string{"DB", "CR", "D", "K", "db", "cr"} {
		if strings.HasSuffix(strings.TrimSpace(m), suf) {
			if a := tandaArah(suf); a != "" && arah == "" {
				arah = a
			}
			m = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(m), suf))
			break
		}
	}
	neg := strings.HasPrefix(m, "-") || (strings.HasPrefix(m, "(") && strings.HasSuffix(m, ")"))
	m = strings.Trim(m, "-()")
	n, ok := parseRupiah(m)
	if !ok || n <= 0 {
		return 0, ""
	}
	if arah == "" {
		// Satu kolom bertanda minus: negatif keluar, positif masuk.
		if neg {
			arah = "keluar"
		} else {
			arah = "masuk"
		}
	}
	return n, arah
}

// parseTanggalMutasi menerima format umum cetakan bank, termasuk tanggal tanpa tahun
// ("01/09") dan nama bulan ("01 Sep 2026").
func parseTanggalMutasi(s string, tahunDari func(time.Month) int, bulanDulu bool) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	if bulanDulu {
		for _, layout := range []string{"01/02/2006", "1/2/2006", "01-02-2006", "1-2-2006"} {
			if t, err := time.Parse(layout, s); err == nil {
				return t.Format(time.DateOnly), true
			}
		}
	}
	if t, ok := parseTanggal(s); ok {
		return t, true
	}
	for _, layout := range []string{"02 Jan 2006", "02 January 2006", "2 Jan 2006", "02-Jan-2006", "02 Jan 06"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format(time.DateOnly), true
		}
	}
	// Tanpa tahun: "01/09", "01-09", "01 SEP" — tahunnya dari isian pengguna.
	if t, ok := tanggalTanpaTahun(s, bulanDulu); ok {
		return time.Date(tahunDari(t.Month()), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Format(time.DateOnly), true
	}
	return "", false
}

func rekapMutasi(rows []MutasiRow) []MutasiGrup {
	urut := make([]string, 0, 8)
	per := map[string]*MutasiGrup{}
	for _, r := range rows {
		if !r.Pilih {
			continue
		}
		g, ok := per[r.Jenis]
		if !ok {
			g = &MutasiGrup{Jenis: r.Jenis, Arah: r.Arah}
			per[r.Jenis] = g
			urut = append(urut, r.Jenis)
		}
		g.Jumlah++
		g.Total += r.Nominal
	}
	out := make([]MutasiGrup, 0, len(urut))
	for _, j := range urut {
		out = append(out, *per[j])
	}
	return out
}

// MutasiKeTransaksi mengubah baris yang sudah diperiksa jadi transaksi. Baris yang tidak
// dipilih dilewati; baris yang jenisnya tidak dikenal dilaporkan dan tidak dibukukan.
func MutasiKeTransaksi(rows []MutasiRow) ([]Transaction, []ImportError) {
	var out []Transaction
	var errs []ImportError
	for _, r := range rows {
		if !r.Pilih {
			continue
		}
		t := Transaction{
			Tanggal: r.Tanggal, Jenis: r.Jenis, Nominal: r.Nominal,
			Keterangan: r.Keterangan, Lunas: true, Channel: "Mutasi Bank",
		}
		if err := Validate(&t); err != nil {
			errs = append(errs, ImportError{Baris: r.Baris, Pesan: err.Error()})
			continue
		}
		// Arah jenis yang dipilih harus sama dengan arah uang di rekening: "Penjualan" untuk
		// baris debet berarti salah pilih, dan akan membalik laporan.
		if j, ok := LookupJenis(t.Jenis); ok && j.Arah != r.Arah {
			errs = append(errs, ImportError{Baris: r.Baris, Pesan: "jenis \"" + t.Jenis + "\" adalah uang " + j.Arah + ", tapi barisnya uang " + r.Arah})
			continue
		}
		out = append(out, t)
	}
	return out, errs
}

// --- bantuan AI untuk baris yang tidak dikenali kata kunci ---

// KelompokMutasi: baris berkeyakinan rendah yang keterangannya sama (setelah nomor disamarkan)
// dan arahnya sama. Dikirim ke AI sekali per kelompok supaya hemat token.
type KelompokMutasi struct {
	Kunci      string `json:"-"`
	Keterangan string `json:"keterangan"` // nomor ≥4 digit diganti "#" (rekening, referensi)
	Arah       string `json:"arah"`
	Nominal    int64  `json:"nominal"` // nominal baris pertama, sebagai gambaran
	Baris      []int  `json:"-"`       // indeks di Rows
}

type SaranJenis struct{ Jenis, Alasan string }

// SamarkanAngka: deret angka ≥4 digit (rekening, nomor referensi, kartu) diganti "#", spasi
// dirapikan, huruf besar. Dipakai sebelum keterangan mutasi dikirim ke layanan AI.
func SamarkanAngka(s string) string {
	var b strings.Builder
	digit := 0
	flush := func(buf string) {
		if digit >= 4 {
			b.WriteString("#")
		} else {
			b.WriteString(buf)
		}
	}
	var buf strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			buf.WriteRune(r)
			digit++
			continue
		}
		if digit > 0 {
			flush(buf.String())
			buf.Reset()
			digit = 0
		}
		b.WriteRune(r)
	}
	if digit > 0 {
		flush(buf.String())
	}
	return strings.ToUpper(strings.Join(strings.Fields(b.String()), " "))
}

func (h *MutasiHasil) KelompokRendah() []KelompokMutasi {
	var out []KelompokMutasi
	idx := map[string]int{}
	for i, r := range h.Rows {
		if r.Keyakinan != KeyakinanRendah {
			continue
		}
		ket := SamarkanAngka(r.Keterangan)
		kunci := r.Arah + "|" + ket
		if j, ok := idx[kunci]; ok {
			out[j].Baris = append(out[j].Baris, i)
			continue
		}
		idx[kunci] = len(out)
		out = append(out, KelompokMutasi{Kunci: kunci, Keterangan: ket, Arah: r.Arah, Nominal: r.Nominal, Baris: []int{i}})
	}
	return out
}

// TerapkanSaran memasang saran AI per kelompok (kunci = KelompokMutasi.Kunci). Jenis yang tidak
// ada di JenisList atau berlawanan arah diabaikan; baris yang dibantu tidak lagi dihitung
// "perlu diperiksa" tetapi tetap ditandai Saran AI.
func (h *MutasiHasil) TerapkanSaran(saran map[string]SaranJenis) {
	for _, g := range h.KelompokRendah() {
		s, ok := saran[g.Kunci]
		if !ok || !jenisArah(s.Jenis, g.Arah) {
			continue
		}
		for _, i := range g.Baris {
			h.Rows[i].Jenis, h.Rows[i].Keyakinan = s.Jenis, KeyakinanAI
			h.Rows[i].Alasan = "Saran AI: " + strings.TrimSpace(s.Alasan)
			h.PerluDiperiksa--
			h.DibantuAI++
		}
	}
	h.Ringkasan = rekapMutasi(h.Rows)
}

func jenisArah(jenis, arah string) bool {
	for _, j := range JenisList {
		if j.Nama == jenis {
			return j.Arah == arah
		}
	}
	return false
}
