// Package coa: Chart of Accounts perdagangan dari "Daftar COA.xlsx" (sheet COA Baru, review
// akuntan Master Pajak), pemetaan otomatis transaksi ke akun, jurnal, dan laporan per akun.
package coa

import "pajak/internal/finance"

type Normal string

const (
	Debit  Normal = "debit"
	Kredit Normal = "kredit"
)

func opposite(n Normal) Normal {
	if n == Debit {
		return Kredit
	}
	return Debit
}

type Kelompok string

const (
	AsetLancar       Kelompok = "Aset Lancar"
	AsetTidakLancar  Kelompok = "Aset Tidak Lancar"
	LiabilitasPendek Kelompok = "Liabilitas Jangka Pendek"
	LiabilitasJangka Kelompok = "Liabilitas Jangka Panjang"
	Ekuitas          Kelompok = "Ekuitas"
	PendapatanUsaha  Kelompok = "Pendapatan Usaha"
	HPP              Kelompok = "Harga Pokok Penjualan"
	BebanOperasional Kelompok = "Beban Operasional"
	PendapatanLain   Kelompok = "Pendapatan Lain-lain"
	BebanLain        Kelompok = "Beban Lain-lain"
)

// Kelompoks: urutan tampilan (sama dengan file).
var Kelompoks = []Kelompok{AsetLancar, AsetTidakLancar, LiabilitasPendek, LiabilitasJangka, Ekuitas, PendapatanUsaha, HPP, BebanOperasional, PendapatanLain, BebanLain}

// Normal: saldo normal kelompok; akun kontra kebalikannya.
func (k Kelompok) Normal() Normal {
	switch k {
	case AsetLancar, AsetTidakLancar, HPP, BebanOperasional, BebanLain:
		return Debit
	}
	return Kredit
}

type Kondisi string

const (
	Selalu        Kondisi = "selalu"
	JikaAda       Kondisi = "jika_ada"       // aset yang belum tentu dimiliki
	JikaPT        Kondisi = "jika_pt"        // PT & PT Perorangan
	JikaNonPT     Kondisi = "jika_non_pt"    // Orang Pribadi & CV
	JikaPerpetual Kondisi = "jika_perpetual" // metode persediaan perpetual
	JikaPeriodik  Kondisi = "jika_periodik"  // metode persediaan periodik
)

type Account struct {
	Kode     string   `json:"kode"`
	Nama     string   `json:"nama"`
	Subtype  string   `json:"subtype"`
	Normal   Normal   `json:"normal"`
	Kelompok Kelompok `json:"kelompok"`
	Kondisi  Kondisi  `json:"kondisi"`
	Catatan  string   `json:"catatan,omitempty"`
	Opsional bool     `json:"opsional,omitempty"`
	// Peran: peran akun di jurnal otomatis (lihat peran.go); kosong = bukan pemegang peran.
	Peran string `json:"peran,omitempty"`
}

func a(kode, nama, subtype string, n Normal, k Kelompok, kon Kondisi, catatan string) Account {
	return Account{Kode: kode, Nama: nama, Subtype: subtype, Normal: n, Kelompok: k, Kondisi: kon, Catatan: catatan}
}

func berperan(acc Account, peran string) Account {
	acc.Peran = peran
	return acc
}

// Master: 72 akun "COA Baru" + 4 akun di luar file (lihat TambahanDiLuarFile).
// Perubahan dari file: 2113 Utang Karyawan → 1407 Piutang Karyawan (keputusan user;
// substansi kasusnya tagihan perusahaan ke karyawan).
var Master = []Account{
	a("1101", "Kas", "cash", Debit, AsetLancar, Selalu, ""),
	a("1102", "Bank", "bank", Debit, AsetLancar, Selalu, ""),
	a("1103", "E-Wallet (QRIS/OVO/GoPay/Dana)", "ewallet", Debit, AsetLancar, Selalu, ""),
	a("1104", "Saldo Marketplace", "ewallet", Debit, AsetLancar, Selalu, ""),
	a("1201", "Piutang Usaha", "receivable", Debit, AsetLancar, Selalu, ""),
	a("1202", "Piutang Pihak Berelasi", "related_party_receivable", Debit, AsetLancar, Selalu, "Disajikan terpisah dari piutang usaha sesuai ketentuan pengungkapan pihak berelasi"),
	a("1301", "Persediaan Barang Dagang", "inventory", Debit, AsetLancar, Selalu, ""),
	a("1302", "Perlengkapan", "supplies", Debit, AsetLancar, Selalu, "Termasuk kemasan"),
	a("1401", "Biaya Dibayar Dimuka", "prepaid", Debit, AsetLancar, Selalu, "Beban yang dibayar di muka untuk beberapa periode (mis. sewa setahun)"),
	a("1402", "PPN Masukan", "tax_in", Debit, AsetLancar, Selalu, ""),
	a("1403", "Pajak Dibayar Dimuka (PPh)", "prepaid_tax", Debit, AsetLancar, Selalu, ""),
	a("1404", "Uang Muka Pembelian", "purchase_advance", Debit, AsetLancar, Selalu, "DP ke supplier sebelum barang diterima"),
	a("1405", "Uang Muka Karyawan", "employee_advance", Debit, AsetLancar, Selalu, "Mis. uang muka perjalanan dinas; dipertanggungjawabkan kemudian"),
	a("1406", "PPN Lebih Bayar", "vat_overpaid", Debit, AsetLancar, Selalu, "PPN Masukan > PPN Keluaran yang dikompensasikan/direstitusi"),
	a("1407", "Piutang Karyawan", "employee_receivable", Debit, AsetLancar, Selalu, "Biaya karyawan yang ditalangi perusahaan lalu dipotong dari gaji (mis. BPJS porsi karyawan)"),
	a("1501", "Peralatan & Inventaris", "fixed_asset", Debit, AsetTidakLancar, Selalu, ""),
	a("1502", "Akumulasi Penyusutan Peralatan", "contra_asset", Kredit, AsetTidakLancar, Selalu, ""),
	a("1503", "Gedung", "fixed_asset", Debit, AsetTidakLancar, JikaAda, ""),
	a("1504", "Akumulasi Penyusutan Gedung", "contra_asset", Kredit, AsetTidakLancar, JikaAda, ""),
	a("1505", "Kendaraan", "fixed_asset", Debit, AsetTidakLancar, JikaAda, ""),
	a("1506", "Akumulasi Penyusutan Kendaraan", "contra_asset", Kredit, AsetTidakLancar, JikaAda, ""),
	a("1507", "Harta Tak Berwujud", "intangible", Debit, AsetTidakLancar, JikaAda, "Software, lisensi, merek, goodwill; disusutkan sebagai amortisasi"),
	a("1508", "Akumulasi Amortisasi", "contra_asset", Kredit, AsetTidakLancar, JikaAda, ""),
	a("1601", "Aset Pajak Tangguhan", "deferred_tax_asset", Debit, AsetTidakLancar, Selalu, "Beda temporer PSAK vs ketentuan pajak yang menimbulkan manfaat pajak di masa depan"),
	a("2101", "Utang Usaha", "payable", Kredit, LiabilitasPendek, Selalu, ""),
	a("2102", "Utang Bank", "loan", Kredit, LiabilitasPendek, Selalu, ""),
	a("2104", "Beban Yang Masih Harus Dibayar", "accrued", Kredit, LiabilitasPendek, Selalu, ""),
	a("2105", "Utang Lain-lain", "payable", Kredit, LiabilitasPendek, Selalu, ""),
	a("2106", "PPN Keluaran", "vat_out", Kredit, LiabilitasPendek, Selalu, ""),
	a("2107", "Utang PPh 21", "wht_payable_pph21", Kredit, LiabilitasPendek, Selalu, ""),
	a("2108", "Utang PPh 23", "wht_payable_pph23", Kredit, LiabilitasPendek, Selalu, ""),
	a("2109", "Utang PPh 4(2)", "wht_payable_pph4_2", Kredit, LiabilitasPendek, Selalu, ""),
	a("2110", "Utang PPh Final", "final_payable", Kredit, LiabilitasPendek, Selalu, ""),
	a("2111", "PPN Kurang Bayar", "vat_underpaid", Kredit, LiabilitasPendek, Selalu, "PPN Keluaran > PPN Masukan yang belum disetor"),
	a("2112", "Utang BPJS", "bpjs_payable", Kredit, LiabilitasPendek, Selalu, "Iuran BPJS (porsi perusahaan + potongan karyawan) yang belum disetor"),
	a("2114", "Utang Pihak Berelasi", "related_party_payable", Kredit, LiabilitasPendek, Selalu, "Disajikan terpisah dari utang usaha sesuai ketentuan pengungkapan pihak berelasi"),
	berperan(a("2115", "Utang Dividen", "dividend_payable", Kredit, LiabilitasPendek, JikaPT, "Muncul otomatis bila bentuk badan usaha PT"), PeranUtangDividen),
	a("2201", "Liabilitas Pajak Tangguhan", "deferred_tax_liability", Kredit, LiabilitasJangka, Selalu, "Pasangan Aset Pajak Tangguhan"),
	a("3101", "Modal Pemilik", "capital", Kredit, Ekuitas, Selalu, ""),
	a("3102", "Prive / Pengambilan Pemilik", "drawing", Debit, Ekuitas, JikaNonPT, "Tidak muncul bila bentuk badan usaha PT"),
	a("3201", "Laba Ditahan", "retained", Kredit, Ekuitas, Selalu, ""),
	a("3299", "Laba Ditahan - Selisih Saldo Awal", "retained", Kredit, Ekuitas, JikaAda, "Selisih saldo awal yang disimpan walau belum seimbang; periksa dan pindahkan ke akun yang benar"),
	a("4101", "Penjualan", "sales", Kredit, PendapatanUsaha, Selalu, ""),
	a("4102", "Pendapatan Jasa", "service", Kredit, PendapatanUsaha, Selalu, ""),
	a("4103", "Diskon Penjualan", "contra_revenue", Debit, PendapatanUsaha, Selalu, ""),
	a("4104", "Retur Penjualan", "contra_revenue", Debit, PendapatanUsaha, Selalu, ""),
	a("5101", "Harga Pokok Penjualan", "cogs", Debit, HPP, JikaPerpetual, "Perpetual: pembelian, ongkos angkut & retur dicatat langsung ke 1301 Persediaan"),
	a("5102", "Pembelian Barang Dagang", "cogs", Debit, HPP, JikaPeriodik, "Periodik: HPP = persediaan awal + pembelian bersih - persediaan akhir"),
	a("5103", "Ongkos Angkut Pembelian", "cogs", Debit, HPP, JikaPeriodik, ""),
	a("5105", "Retur & Potongan Pembelian", "contra_cogs", Kredit, HPP, JikaPeriodik, ""),
	a("5104", "Beban Kemasan", "cogs", Debit, HPP, Selalu, ""),
	a("6101", "Beban Gaji & Upah", "opex", Debit, BebanOperasional, Selalu, ""),
	a("6117", "Beban BPJS", "opex", Debit, BebanOperasional, Selalu, "Porsi iuran BPJS Kesehatan & Ketenagakerjaan yang ditanggung perusahaan"),
	a("6102", "Beban Iklan & Promosi", "opex", Debit, BebanOperasional, Selalu, ""),
	a("6103", "Beban Komisi Affiliate & KOL", "opex", Debit, BebanOperasional, Selalu, ""),
	a("6104", "Beban Sewa", "opex", Debit, BebanOperasional, Selalu, ""),
	a("6105", "Beban Listrik, Air & Internet", "opex", Debit, BebanOperasional, Selalu, ""),
	a("6106", "Beban Transportasi & Ongkir", "opex", Debit, BebanOperasional, Selalu, ""),
	a("6107", "Beban Perlengkapan Kantor", "opex", Debit, BebanOperasional, Selalu, ""),
	a("6108", "Beban Marketplace & Payment Gateway", "opex", Debit, BebanOperasional, Selalu, ""),
	a("6109", "Beban Jasa Profesional", "opex", Debit, BebanOperasional, Selalu, ""),
	a("6110", "Beban Perjalanan & Akomodasi", "opex", Debit, BebanOperasional, Selalu, ""),
	a("6111", "Beban Konsumsi & Entertainment", "opex", Debit, BebanOperasional, Selalu, ""),
	a("6112", "Beban Pemeliharaan & Perbaikan", "opex", Debit, BebanOperasional, Selalu, ""),
	a("6113", "Beban Penyusutan", "opex", Debit, BebanOperasional, Selalu, ""),
	a("6114", "Beban Langganan Software & AI", "opex", Debit, BebanOperasional, Selalu, ""),
	a("6115", "Beban Pelatihan & Pengembangan", "opex", Debit, BebanOperasional, Selalu, ""),
	a("6116", "Beban Produksi Konten", "opex", Debit, BebanOperasional, Selalu, ""),
	a("6118", "Beban Amortisasi", "opex", Debit, BebanOperasional, JikaAda, "Amortisasi harta tak berwujud; dipisah dari 6113 agar terbaca terpisah di laba rugi"),
	a("6190", "Beban Operasional Lain-lain", "opex", Debit, BebanOperasional, Selalu, ""),
	a("8101", "Pendapatan Lain-lain", "other_income", Kredit, PendapatanLain, Selalu, "Dipisah dari pendapatan usaha"),
	a("9101", "Beban Bunga", "non_operating", Debit, BebanLain, Selalu, ""),
	a("9102", "Beban Administrasi Bank", "non_operating", Debit, BebanLain, Selalu, ""),
	a("9103", "Beban Pajak", "non_operating", Debit, BebanLain, Selalu, ""),
	a("9104", "Beban (Manfaat) Pajak Tangguhan", "deferred_tax_expense", Debit, BebanLain, Selalu, "Pasangan jurnal Aset/Liabilitas Pajak Tangguhan"),
	a("9901", "Beban Lain-lain", "non_operating", Debit, BebanLain, Selalu, ""),
}

// TambahanDiLuarFile: akun yang tidak ada di "Daftar COA.xlsx", ditambahkan supaya register
// aset bisa dijurnal ke CoA. Tanpa ini laporan per akun tidak mengenal penyusutan, sehingga
// laba bersihnya berbeda dengan laba rugi ringkas. Semuanya JikaAda, jadi usaha yang tidak
// punya harta tak berwujud tidak melihatnya. Hapus dari sini bila kelak masuk ke file resmi.
// 3299 menampung selisih saldo awal yang disimpan walau belum seimbang.
var TambahanDiLuarFile = []string{"1507", "1508", "6118", "3299"}

// Supported: CoA baru tersedia untuk perdagangan; jasa & manufaktur menunggu file dari tim.
func Supported(p finance.BusinessProfile) bool { return p.JenisUsaha == "perdagangan" }

func isPT(p finance.BusinessProfile) bool {
	return p.BentukUsaha == "pt" || p.BentukUsaha == "pt_perorangan"
}

func perpetual(p finance.BusinessProfile) bool { return p.MetodePersediaan == "perpetual" }

// Active: akun Master yang berlaku untuk profil usaha (bentuk badan & metode persediaan).
func Active(p finance.BusinessProfile) []Account {
	if !Supported(p) {
		return nil
	}
	return ActiveFrom(Master, p)
}

// ActiveFrom: akun template yang berlaku untuk profil usaha.
func ActiveFrom(template []Account, p finance.BusinessProfile) []Account {
	var out []Account
	for _, acc := range template {
		switch acc.Kondisi {
		case JikaPT:
			if !isPT(p) {
				continue
			}
		case JikaNonPT:
			if isPT(p) {
				continue
			}
		case JikaPerpetual:
			if !perpetual(p) {
				continue
			}
		case JikaPeriodik:
			if perpetual(p) {
				continue
			}
		case JikaAda:
			acc.Opsional = true
		}
		out = append(out, acc)
	}
	return out
}

// Find mencari akun di daftar.
func Find(accs []Account, kode string) (Account, bool) {
	for _, acc := range accs {
		if acc.Kode == kode {
			return acc, true
		}
	}
	return Account{}, false
}
