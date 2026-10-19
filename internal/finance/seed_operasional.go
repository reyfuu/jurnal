package finance

// Data contoh untuk Piutang & Utang, Stok & Gudang, serta Produksi & Supplier.
//
// Sebelumnya daftar ini hanya ada sebagai tampilan di dalam DashboardSnapshot, sehingga
// dashboard menampilkan lima tagihan contoh sementara halaman Piutang & Utang kosong. Di sini
// datanya jadi record sungguhan: dashboard, halaman daftar, dan resep HPP membaca yang sama,
// dan baris contohnya bisa langsung disunting atau dihapus pengguna.

// ID stok contoh dipakai resep di SeedProduk. Diberi nama supaya resep tidak menunjuk angka
// telanjang yang bisa bergeser kalau urutan daftar berubah.
const (
	stokBijiKopi = 1
	stokSusu     = 2
	stokGulaAren = 3
	stokCup12oz  = 4
	stokMatcha   = 5
	stokOatMilk  = 6
)

func SeedTagihan() []Tagihan {
	return []Tagihan{
		{ID: 1, Nomor: "INV-2026-061", Pihak: "PT Karya Digital Mandiri", Jenis: TagihanPiutang, JatuhTempo: "2026-07-10", Nominal: 5_400_000, Keterangan: "Kopi rapat bulanan"},
		{ID: 2, Nomor: "INV-2026-068", Pihak: "Event Organizer Senayan Fest", Jenis: TagihanPiutang, JatuhTempo: "2026-07-22", Nominal: 4_200_000, Keterangan: "Booth kopi 2 hari"},
		{ID: 3, Nomor: "INV-2026-089", Pihak: "Coworking Space Kemang", Jenis: TagihanPiutang, JatuhTempo: "2026-09-05", Nominal: 1_400_000},
		{ID: 4, Nomor: "PO-SUP-112", Pihak: "Roastery Biji Kopi Gayo Lestari", Jenis: TagihanUtang, JatuhTempo: "2026-09-08", Nominal: 4_800_000, Keterangan: "Biji kopi 40 kg"},
		{ID: 5, Nomor: "PO-SUP-118", Pihak: "Kemasan Cup & Sablon Senja", Jenis: TagihanUtang, JatuhTempo: "2026-09-12", Nominal: 2_600_000, Keterangan: "Cup 12oz 5.000 pcs"},
	}
}

// SeedStok: bahan baku & produk jadi contoh. Harga pokok produk jadi (FG-) sengaja sama
// dengan hasil hitung resep di SeedProduk, supaya halaman Produksi tidak menunjukkan selisih
// pada data contoh.
func SeedStok() []StokItem {
	return []StokItem{
		{ID: stokBijiKopi, SKU: "RM-001", Nama: "Biji Kopi Arabica Gayo (Roasted)", Kategori: "Bahan Baku", Jumlah: 42, Satuan: "kg", StokMin: 15, HargaPokok: 185_000},
		{ID: stokSusu, SKU: "RM-002", Nama: "Fresh Milk Full Cream", Kategori: "Bahan Baku", Jumlah: 18, Satuan: "liter", StokMin: 24, HargaPokok: 21_500},
		{ID: stokGulaAren, SKU: "RM-003", Nama: "Gula Aren Cair", Kategori: "Bahan Baku", Jumlah: 24, Satuan: "liter", StokMin: 8, HargaPokok: 32_000},
		{ID: stokCup12oz, SKU: "RM-004", Nama: "Cup 12oz + Lid + Sablon", Kategori: "Kemasan", Jumlah: 3_400, Satuan: "pcs", StokMin: 1_000, HargaPokok: 1_150},
		{ID: stokMatcha, SKU: "RM-005", Nama: "Bubuk Matcha Premium", Kategori: "Bahan Baku", Jumlah: 4, Satuan: "kg", StokMin: 2, HargaPokok: 480_000},
		{ID: stokOatMilk, SKU: "RM-006", Nama: "Oat Milk Barista", Kategori: "Bahan Baku", Jumlah: 21, Satuan: "liter", StokMin: 12, HargaPokok: 28_000},
		{ID: 7, SKU: "FG-001", Nama: "Es Kopi Susu Senja (12oz)", Kategori: "Produk Jadi", Jumlah: 120, Satuan: "cup", StokMin: 30, HargaPokok: 8_300, HargaJual: 18_000},
		{ID: 8, SKU: "FG-002", Nama: "Matcha Oat Latte (12oz)", Kategori: "Produk Jadi", Jumlah: 60, Satuan: "cup", StokMin: 20, HargaPokok: 13_950, HargaJual: 25_000},
	}
}

// SeedProduk: resep contoh. Takaran dalam per seribu satuan bahan (lihat TakaranSkala):
// 180 berarti 0,18 kg biji kopi, 1200 berarti 1,2 liter susu.
func SeedProduk() []Produk {
	return []Produk{
		{
			ID: 1, SKU: "FG-001", Nama: "Es Kopi Susu Senja (12oz)", Satuan: "cup", Kategori: "Produk Jadi",
			HasilPerBatch: 10,
			Bahan: []ResepBahan{
				{StokID: stokBijiKopi, Takaran: 180},
				{StokID: stokSusu, Takaran: 1_200},
				{StokID: stokGulaAren, Takaran: 150},
				{StokID: stokCup12oz, Takaran: 10 * TakaranSkala},
			},
			BiayaTenagaKerja: 5_000, BiayaOverhead: 2_600, HargaJual: 18_000,
			Catatan: "Satu batch 10 cup: brew 180 g biji, susu 1,2 L, gula aren 150 ml.",
		},
		{
			ID: 2, SKU: "FG-002", Nama: "Matcha Oat Latte (12oz)", Satuan: "cup", Kategori: "Produk Jadi",
			HasilPerBatch: 10,
			Bahan: []ResepBahan{
				{StokID: stokMatcha, Takaran: 150},
				{StokID: stokOatMilk, Takaran: 1_500},
				{StokID: stokGulaAren, Takaran: 200},
				{StokID: stokCup12oz, Takaran: 10 * TakaranSkala},
			},
			BiayaTenagaKerja: 5_000, BiayaOverhead: 2_600, HargaJual: 25_000,
		},
	}
}

func SeedSupplier() []Supplier {
	return []Supplier{
		{ID: 1, Nama: "Roastery Biji Kopi Gayo Lestari", Kontak: "Pak Hendra", Telepon: "0812-5566-7788", Alamat: "Jl. Cihampelas No. 24, Bandung", TempoHari: 14, Catatan: "Kirim tiap Senin, minimum 20 kg."},
		{ID: 2, Nama: "Kemasan Cup & Sablon Senja", Kontak: "Bu Ningsih", Telepon: "0857-1122-3344", Alamat: "Kawasan Industri Pulogadung, Jakarta Timur", TempoHari: 30},
		{ID: 3, Nama: "CV Susu Segar Nusantara", Kontak: "Mas Yoga", Telepon: "0813-9090-1212", Alamat: "Lembang, Kab. Bandung Barat", TempoHari: 7, Catatan: "Susu & oat milk, kirim 2x seminggu."},
	}
}
