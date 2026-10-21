package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"pajak/internal/llm"
)

// Panduan menu: menjawab "di mana / bagaimana cara ..." dengan penjelasan singkat dan tautan
// ke halaman aplikasi yang tepat. Tautan disaring sesuai hak akses penanya; menu yang tidak
// bisa dibukanya dijelaskan, bukan ditautkan. Pencocokan kata kunci dulu (gratis); LLM hanya
// dipakai memilih topik dari daftar bila kata kunci tidak menemukan apa pun.

// Tautan: View = id halaman di frontend; Tab opsional (mis. "banding" di Laporan Keuangan).
type Tautan struct {
	View  string `json:"view"`
	Label string `json:"label"`
	Tab   string `json:"tab,omitempty"`
}

type Panduan struct {
	Kode    string
	Jawaban string
	Tautan  []Tautan
	// Kuat: pertanyaannya memang menanyakan tempat/cara memakai menu. Lemah = topiknya cocok
	// tapi pertanyaannya soal isi (mis. aturan pajak) — tautan cukup jadi pelengkap.
	Kuat bool
	AI   bool
}

type topik struct {
	kode, ringkas string
	kata          []string
	jawaban       string
	tautan        []Tautan
}

var (
	keTransaksi = Tautan{View: "transaksi", Label: "Buka Transaksi"}
	keLaporan   = Tautan{View: "laporan-keuangan", Label: "Buka Laporan Keuangan"}
	keImporAwal = Tautan{View: "saldo-awal", Label: "Isi saldo awal"}
)

var daftarTopik = []topik{
	{kode: "isi-laporan", ringkas: "mengisi / menyusun laporan keuangan, pembukuan, laba rugi, neraca, arus kas",
		kata: []string{"laporan keuangan", "lapkeu", "laba rugi", "neraca", "arus kas", "pembukuan", "pembukuan usaha"},
		jawaban: "Laporan keuangan (Laba Rugi, Neraca, Arus Kas) **tersusun otomatis** dari transaksi yang Anda catat — tidak perlu diisi manual.\n" +
			"- **Catat transaksi** setiap ada uang masuk/keluar di menu Transaksi (bisa juga impor mutasi bank).\n" +
			"- **Isi saldo awal** sekali saja (kas, piutang, utang, stok, pinjaman) di menu Saldo Awal.\n" +
			"- **Lihat hasilnya** di Laporan Keuangan; pilih bulannya di Ganti Periode.",
		tautan: []Tautan{{View: "transaksi", Label: "Catat transaksi"}, keImporAwal, {View: "laporan-keuangan", Label: "Lihat laporan keuangan"}}},
	{kode: "unduh-laporan", ringkas: "mengunduh / mencetak laporan keuangan ke PDF",
		kata:    []string{"download laporan", "unduh laporan", "laporan keuangan pdf", "laporan pdf", "pdf", "cetak laporan", "print laporan", "ekspor laporan", "export laporan"},
		jawaban: "Di **Laporan Keuangan**, setiap tab punya tombol **Unduh PDF** (Laba Rugi, Neraca, Arus Kas, Aset). Tombol **Unduh semua laporan (PDF)** menarik Laba Rugi, Neraca, dan Arus Kas sekaligus dalam satu berkas.",
		tautan:  []Tautan{keLaporan}},
	{kode: "banding", ringkas: "membandingkan laporan beberapa bulan sekaligus",
		kata:    []string{"banding", "bandingkan", "perbandingan", "membandingkan", "compare", "beberapa bulan", "antar bulan", "per bulan"},
		jawaban: "Buka tab **Banding** di Laporan Keuangan, lalu pilih 2–12 bulan (boleh tidak berurutan). Tabel menampilkan tiap bulan dan kolom **Total** di akhir; bisa diunduh ke PDF.",
		tautan:  []Tautan{{View: "laporan-keuangan", Label: "Buka tab Banding", Tab: "banding"}}},
	{kode: "transaksi", ringkas: "mencatat pemasukan, pengeluaran, penjualan, pembelian, biaya",
		kata:    []string{"transaksi", "pemasukan", "pengeluaran", "uang masuk", "uang keluar", "penjualan harian", "pembelian", "biaya", "beban", "nota", "struk", "belanja"},
		jawaban: "Catat setiap uang masuk/keluar di menu **Transaksi** → tombol **Tambah transaksi**: pilih jenis, nominal, tanggal, dan status lunas. Banyak transaksi sekaligus bisa diimpor dari Excel atau mutasi bank.",
		tautan:  []Tautan{keTransaksi}},
	{kode: "mutasi", ringkas: "impor mutasi bank / rekening koran",
		kata:    []string{"mutasi", "mutasi bank", "rekening koran", "rekening", "bca", "mandiri", "bri", "bni", "e statement", "estatement"},
		jawaban: "Buka **Transaksi** → **Impor mutasi bank**, unggah file mutasi/rekening koran (PDF e-statement, Excel .xlsx, atau .csv). Baris dikelompokkan otomatis; yang belum yakin dibantu AI, dan Anda bisa memeriksa sebelum menyimpan.",
		tautan:  []Tautan{{View: "transaksi", Label: "Impor mutasi bank"}}},
	{kode: "buku-besar", ringkas: "buku besar / mutasi per akun / saldo per akun",
		kata:    []string{"buku besar", "general ledger", "ledger", "mutasi akun", "saldo akun", "rincian akun"},
		jawaban: "Buka menu **Buku Besar** (grup Laporan): ringkasan saldo awal, debit, kredit, dan saldo akhir setiap akun bulan ini. Klik akun untuk melihat setiap mutasinya beserta saldo berjalan; bisa diekspor ke Excel.",
		tautan:  []Tautan{{View: "buku-besar", Label: "Buka Buku Besar"}}},
	{kode: "saldo-awal", ringkas: "mengisi saldo awal / data awal / modal awal / impor Excel data lama",
		kata:    []string{"saldo awal", "data awal", "neraca awal", "modal awal", "impor data", "import data", "data lama", "template excel"},
		jawaban: "Saldo awal (kas & bank, piutang, utang, stok, pinjaman) diisi sekali di menu **Saldo Awal**; modal awal dihitung otomatis. Aset yang sudah dimiliki didaftarkan di Laporan Keuangan → Aset. Data lama yang banyak bisa diunggah sekaligus lewat template Excel di **Impor Data Awal**.",
		tautan:  []Tautan{keImporAwal, {View: "impor-data", Label: "Impor Data Awal (Excel)"}}},
	{kode: "piutang", ringkas: "piutang pelanggan, utang ke supplier, tagihan belum dibayar",
		kata:    []string{"piutang", "utang", "hutang", "ngutang", "tagihan", "belum dibayar", "belum lunas", "jatuh tempo", "tempo", "kasbon"},
		jawaban: "Daftar piutang (pelanggan belum bayar) dan utang (Anda belum bayar) ada di **Piutang & Utang**. Transaksi yang dicatat *belum lunas* otomatis masuk ke sana.",
		tautan:  []Tautan{{View: "piutang-utang", Label: "Buka Piutang & Utang"}}},
	{kode: "stok", ringkas: "stok barang, gudang, persediaan",
		kata:    []string{"stok", "gudang", "persediaan", "inventory", "barang", "stock"},
		jawaban: "Jumlah dan nilai stok dikelola di **Stok & Gudang**.",
		tautan:  []Tautan{{View: "stok-gudang", Label: "Buka Stok & Gudang"}}},
	{kode: "produksi", ringkas: "produksi, resep, bahan baku, supplier",
		kata:    []string{"produksi", "supplier", "pemasok", "resep", "bahan baku", "vendor"},
		jawaban: "Data produksi, resep/bahan baku, dan supplier ada di **Produksi & Supplier**.",
		tautan:  []Tautan{{View: "produksi-supplier", Label: "Buka Produksi & Supplier"}}},
	{kode: "invoice", ringkas: "membuat invoice / faktur penjualan untuk pelanggan",
		kata:    []string{"invoice", "faktur", "faktur penjualan", "tagih pelanggan", "nota penjualan", "kwitansi"},
		jawaban: "Buat invoice di menu **Invoice** → **Buat invoice**, lalu unduh PDF-nya untuk dikirim ke pelanggan.",
		tautan:  []Tautan{{View: "invoice", Label: "Buka Invoice"}}},
	{kode: "aset", ringkas: "aset tetap, peralatan, penyusutan",
		kata:    []string{"aset", "asset", "penyusutan", "inventaris", "peralatan", "mesin", "kendaraan", "amortisasi"},
		jawaban: "Aset dan penyusutannya dicatat di tab **Aset & Penyusutan** pada Laporan Keuangan. Metode penyusutan dipilih sekali untuk semua aset usaha.",
		tautan:  []Tautan{{View: "laporan-keuangan", Label: "Buka Aset & Penyusutan", Tab: "aset"}}},
	{kode: "pajak-umkm", ringkas: "PPh Final UMKM 0,5%, PP 55, setor pajak bulanan",
		kata:    []string{"pajak umkm", "pph final", "pp 55", "pp55", "0 5", "setor pajak", "pajak bulanan", "pajak saya"},
		jawaban: "Hitungan PPh Final UMKM per bulan (termasuk sisa batas omzet bebas pajak untuk orang pribadi) ada di menu **Pajak UMKM**.",
		tautan:  []Tautan{{View: "pajak-umkm", Label: "Buka Pajak UMKM"}}},
	{kode: "kalkulator", ringkas: "kalkulator PPh 21, PPh 23, PPh badan, cek PKP",
		kata:    []string{"pph 21", "pph21", "pph 23", "pph23", "pph badan", "kalkulator pajak", "kalkulator", "pkp"},
		jawaban: "Hitung PPh 21, PPh 23/potong pajak, PPh badan, dan cek kewajiban PKP di **Kalkulator Pajak**.",
		tautan:  []Tautan{{View: "kalkulator-pajak", Label: "Buka Kalkulator Pajak"}}},
	{kode: "perencanaan", ringkas: "perencanaan pajak, hemat pajak, pilih bentuk usaha",
		kata:    []string{"perencanaan pajak", "tax planning", "hemat pajak", "bentuk usaha", "cv atau pt", "pindah pt", "perencanaan"},
		jawaban: "Bandingkan skema pajak dan bentuk usaha (perorangan, CV, PT) di **Perencanaan Pajak**.",
		tautan:  []Tautan{{View: "perencanaan-pajak", Label: "Buka Perencanaan Pajak"}}},
	{kode: "simulasi", ringkas: "simulasi harga jual, komisi aplikasi ojol/marketplace, margin",
		kata:    []string{"harga jual", "simulasi", "komisi", "gofood", "grabfood", "shopeefood", "markup", "margin", "titik impas", "bep"},
		jawaban: "Hitung harga jual yang aman setelah komisi aplikasi, margin, dan titik impas di **Simulasi & Harga**.",
		tautan:  []Tautan{{View: "simulasi-harga", Label: "Buka Simulasi & Harga"}}},
	{kode: "laporan-penjualan", ringkas: "laporan penjualan per hari/produk/channel",
		kata:    []string{"laporan penjualan", "omzet harian", "penjualan per produk", "produk terlaris", "channel penjualan"},
		jawaban: "Rincian penjualan per hari, minggu, produk, dan channel ada di **Laporan Penjualan**.",
		tautan:  []Tautan{{View: "laporan-penjualan", Label: "Buka Laporan Penjualan"}}},
	{kode: "karyawan", ringkas: "data karyawan dan gaji",
		kata:    []string{"karyawan", "pegawai", "gaji", "payroll", "upah"},
		jawaban: "Data karyawan dan gajinya dikelola di menu **Karyawan**.",
		tautan:  []Tautan{{View: "karyawan", Label: "Buka Karyawan"}}},
	{kode: "pengguna", ringkas: "menambah pengguna/akun kasir/admin, hak akses",
		kata:    []string{"pengguna", "user", "akses", "hak akses", "kasir", "admin keuangan", "akun pengguna", "tambah akun", "undang"},
		jawaban: "Tambah akun Kasir atau Admin Keuangan dan atur aksesnya di **Pengguna & Akses** (khusus Owner).",
		tautan:  []Tautan{{View: "pengguna", Label: "Buka Pengguna & Akses"}}},
	{kode: "profil", ringkas: "profil usaha: nama, alamat, NPWP, bidang usaha (KLU), bentuk usaha",
		kata:    []string{"profil", "profil usaha", "nama usaha", "npwp", "alamat usaha", "klu", "bidang usaha", "logo"},
		jawaban: "Nama usaha, alamat, NPWP, bidang usaha (KLU), dan bentuk usaha diubah di **Profil Usaha**.",
		tautan:  []Tautan{{View: "profil-usaha", Label: "Buka Profil Usaha"}}},
	{kode: "langganan", ringkas: "berlangganan, paket, perpanjang, pembayaran langganan",
		kata:    []string{"langganan", "berlangganan", "paket", "perpanjang", "upgrade", "bayar langganan"},
		jawaban: "Pilih atau perpanjang paket di menu **Langganan**.",
		tautan:  []Tautan{{View: "langganan", Label: "Buka Langganan"}}},
	{kode: "coa", ringkas: "daftar akun / chart of account / kode akun",
		kata:    []string{"daftar akun", "coa", "chart of account", "kode akun", "akun akuntansi"},
		jawaban: "Daftar akun (CoA) usaha Anda ada di **Daftar Akun (CoA)**.",
		tautan:  []Tautan{{View: "master-data", Label: "Buka Daftar Akun"}}},
	{kode: "catatan", ringkas: "catatan / koreksi dari konsultan Master Pajak",
		kata:    []string{"catatan konsultan", "catatan dari konsultan", "usulan koreksi", "koreksi konsultan"},
		jawaban: "Catatan dari konsultan ada di **Catatan Konsultan**; usulan koreksi pembukuan menunggu persetujuan Anda di **Usulan Koreksi**.",
		tautan:  []Tautan{{View: "catatan", Label: "Buka Catatan Konsultan"}, {View: "usulan-koreksi", Label: "Buka Usulan Koreksi"}}},
	{kode: "bug", ringkas: "melaporkan error / masalah aplikasi",
		kata:    []string{"error", "bug", "lapor masalah", "tidak bisa", "gagal", "rusak", "macet"},
		jawaban: "Laporkan kendala aplikasi lewat **Lapor Bug**; tim kami akan membalas di sana.",
		tautan:  []Tautan{{View: "lapor-bug", Label: "Buka Lapor Bug"}}},
	{kode: "pustaka", ringkas: "artikel & peraturan pajak",
		kata:    []string{"artikel", "pustaka", "peraturan pajak", "baca aturan"},
		jawaban: "Artikel dan ringkasan peraturan pajak ada di **Pustaka Pajak**.",
		tautan:  []Tautan{{View: "pustaka", Label: "Buka Pustaka Pajak"}}},
}

// kataTanya: penanda pertanyaan "di mana / bagaimana cara memakai menu".
var kataTanya = []string{"di mana", "dimana", "dmn", "mana menu", "menu", "halaman", "link", "cara", "caranya", "gimana", "gmn",
	"isi", "mengisi", "ngisi", "diisi", "input", "menginput", "catat", "mencatat", "nyatet", "nyatat", "dicatat", "tambah", "menambah", "nambah",
	"masukkan", "memasukkan", "masukin", "unduh", "download", "upload", "unggah", "impor", "import", "buka", "ubah", "ganti", "atur",
	"setting", "cetak", "print", "ekspor", "export", "bikin", "buat", "membuat", "lihat", "melihat", "cek", "hitung"}

// normal: huruf kecil, tanda baca jadi spasi, diapit spasi supaya pencocokan per kata utuh.
func normal(s string) string {
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == ' ' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + 32
		}
		return ' '
	}, s)
	return " " + strings.Join(strings.Fields(s), " ") + " "
}

func ada(q, kata string) bool {
	if strings.Contains(q, " "+kata+" ") {
		return true
	}
	// Salah ketik: kata kunci (atau frasa) ≥ 6 huruf boleh beda satu huruf ("trasaksi",
	// "saldo awla"). Yang lebih pendek harus persis, supaya "stop" tidak dibaca "stok".
	if len(kata) < 6 {
		return false
	}
	kw, qw := strings.Fields(kata), strings.Fields(q)
	for i := 0; i+len(kw) <= len(qw); i++ {
		if bedaHuruf(strings.Join(qw[i:i+len(kw)], " "), kata) <= 1 {
			return true
		}
	}
	return false
}

// bedaHuruf: jarak edit (sisip, hapus, ganti, tukar dua huruf bersebelahan).
func bedaHuruf(a, b string) int {
	x, y := []rune(a), []rune(b)
	d := make([][]int, len(x)+1)
	for i := range d {
		d[i] = make([]int, len(y)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for i := 1; i <= len(x); i++ {
		for j := 1; j <= len(y); j++ {
			biaya := 1
			if x[i-1] == y[j-1] {
				biaya = 0
			}
			d[i][j] = min(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+biaya)
			if i > 1 && j > 1 && x[i-1] == y[j-2] && x[i-2] == y[j-1] {
				d[i][j] = min(d[i][j], d[i-2][j-2]+1)
			}
		}
	}
	return d[len(x)][len(y)]
}

// PanduanUmum: pertanyaan menu yang tidak dikenali — tawarkan menu utama yang bisa dibuka,
// jangan dijawab dengan ringkasan laporan.
func PanduanUmum(boleh func(view string) bool) Panduan {
	p := Panduan{Kode: "umum", Kuat: true, Tautan: []Tautan{},
		Jawaban: "Maaf, saya belum menangkap menu yang dimaksud. Coba tulis lebih spesifik (mis. *di mana mencatat penjualan?*), atau pilih menu yang paling sering dipakai:"}
	for _, l := range []Tautan{
		{View: "transaksi", Label: "Catat transaksi"},
		{View: "laporan-keuangan", Label: "Laporan Keuangan"},
		{View: "impor-data", Label: "Impor Data Awal"},
		{View: "stok-gudang", Label: "Stok & Gudang"},
		{View: "pajak-umkm", Label: "Pajak UMKM"},
		{View: "lapor-bug", Label: "Lapor Bug"},
	} {
		if boleh(l.View) {
			p.Tautan = append(p.Tautan, l)
		}
	}
	return p
}

// TanyaMenu: pertanyaan ini menanyakan tempat/cara memakai menu aplikasi.
func TanyaMenu(q string) bool {
	q = normal(q)
	return slices.ContainsFunc(kataTanya, func(k string) bool { return ada(q, k) })
}

// susunPanduan menyaring tautan sesuai akses; menu yang tidak bisa dibuka dijelaskan.
func susunPanduan(t topik, boleh func(string) bool) Panduan {
	p := Panduan{Kode: t.kode, Jawaban: t.jawaban, Tautan: []Tautan{}}
	for _, l := range t.tautan {
		if boleh(l.View) {
			p.Tautan = append(p.Tautan, l)
			continue
		}
		p.Jawaban += "\n\n*Menu " + strings.TrimPrefix(strings.TrimPrefix(l.Label, "Buka "), "Isi ") +
			" tidak tersedia untuk akun Anda. Minta Owner usaha membukakan aksesnya di Pengguna & Akses.*"
	}
	return p
}

// CariPanduan mencocokkan pertanyaan dengan topik menu lewat kata kunci (kata yang lebih
// panjang/lebih spesifik lebih berbobot). ok=false bila tidak ada topik yang cocok.
func CariPanduan(q string, boleh func(view string) bool) (Panduan, bool) {
	n := normal(q)
	terbaik, skor := -1, 0
	for i, t := range daftarTopik {
		s := 0
		for _, k := range t.kata {
			if ada(n, k) {
				s += len(k)
			}
		}
		if s > skor {
			terbaik, skor = i, s
		}
	}
	if terbaik < 0 {
		return Panduan{}, false
	}
	p := susunPanduan(daftarTopik[terbaik], boleh)
	p.Kuat = TanyaMenu(q)
	if !p.Kuat && len(p.Tautan) > 1 {
		// Hanya pelengkap jawaban lain: satu tautan cukup, utamakan halaman laporannya.
		i := max(0, slices.IndexFunc(p.Tautan, func(t Tautan) bool { return t.View == "laporan-keuangan" }))
		p.Tautan = p.Tautan[i : i+1]
	}
	return p, true
}

const promptPanduan = `Kamu asisten aplikasi pembukuan & pajak UMKM "Master Pajak". Pengguna bertanya di mana atau bagaimana melakukan sesuatu di aplikasi. Pilih 1-2 topik yang paling cocok HANYA dari daftar (kode | isi topik). Jangan mengarang kode. Bila tidak ada yang cocok, kembalikan daftar kosong.

Balas HANYA JSON: {"topik":["kode"]}`

// PilihPanduan: model memilih topik dari daftar untuk pertanyaan yang tidak cocok dengan kata
// kunci. Kode di luar daftar dibuang.
func PilihPanduan(ctx context.Context, c *llm.Client, q string, boleh func(view string) bool) (Panduan, error) {
	if c == nil || !c.Enabled() {
		return Panduan{}, fmt.Errorf("AI belum aktif")
	}
	var daftar strings.Builder
	for _, t := range daftarTopik {
		fmt.Fprintf(&daftar, "%s | %s\n", t.kode, t.ringkas)
	}
	raw, err := c.Complete(ctx, []llm.Message{
		{Role: "system", Content: promptPanduan},
		{Role: "user", Content: "Daftar topik:\n" + daftar.String() + "\nPertanyaan pengguna: " + q},
	}, true, 200)
	if err != nil {
		return Panduan{}, err
	}
	var out struct {
		Topik []string `json:"topik"`
	}
	if err := json.Unmarshal([]byte(llm.ExtractJSON(raw)), &out); err != nil {
		return Panduan{}, fmt.Errorf("balasan model bukan JSON")
	}
	for _, k := range out.Topik {
		if i := slices.IndexFunc(daftarTopik, func(t topik) bool { return t.kode == strings.TrimSpace(k) }); i >= 0 {
			p := susunPanduan(daftarTopik[i], boleh)
			p.Kuat, p.AI = true, true
			return p, nil
		}
	}
	return Panduan{}, fmt.Errorf("model tidak memilih topik dari daftar")
}
