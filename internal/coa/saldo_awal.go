package coa

import (
	"slices"
	"strconv"
	"strings"

	"pajak/internal/finance"
)

// Layar Saldo Awal memakai CoA usaha itu sendiri (dagang / manufaktur), bukan daftar tetap:
// kode & nama akun ikut CoA-nya, kategori neraca diturunkan dari kelompok, subtype, dan peran.

// subtypeTanpaSaldoAwal: akun pengurang & penampung sistem — tidak diisi pengguna di saldo awal.
var subtypeTanpaSaldoAwal = []string{"contra_asset", "accum_depr", "accum_amort", "inventory_contra",
	"current_earnings", "drawing", "equity_contra"}

func peranAkun(a Account, peta Peta) string {
	if a.Peran != "" {
		return a.Peran
	}
	for r, k := range peta {
		if k == a.Kode {
			return r
		}
	}
	return ""
}

// KategoriSaldoAkun: kategori saldo awal satu akun neraca; "" = tidak diisi di saldo awal.
func KategoriSaldoAkun(a Account, peta Peta) string {
	if slices.Contains(subtypeTanpaSaldoAwal, a.Subtype) || strings.Contains(strings.ToLower(a.Nama), "opening balance") {
		return ""
	}
	peran := peranAkun(a, peta)
	switch a.Kelompok {
	case AsetLancar:
		switch {
		case slices.Contains([]string{"cash", "bank", "ewallet"}, a.Subtype):
			return finance.KatKas
		case a.Subtype == "inventory":
			return finance.KatPersediaan
		case peran == PeranPiutang:
			return finance.KatPiutang
		}
		return finance.KatAsetLain
	case AsetTidakLancar:
		if a.Subtype == "fixed_asset" || a.Subtype == "intangible" {
			return finance.KatAsetTetap
		}
		return finance.KatAsetLain
	case LiabilitasPendek, LiabilitasJangka:
		switch {
		case peran == PeranUtangUsaha:
			return finance.KatUtang
		case peran == PeranPinjaman || a.Subtype == "loan" || a.Subtype == "debt":
			return finance.KatPinjaman
		}
		return finance.KatKewajibanLain
	case Ekuitas:
		if peran == PeranLabaDitahan || strings.HasPrefix(a.Subtype, "retained") {
			return finance.KatLabaDitahan
		}
		return finance.KatModal
	}
	return ""
}

// TemplateSaldoAwal: baris bawaan layar Saldo Awal dari akun neraca aktif. Aset tetap tidak
// diketik di sini: barisnya datang dari daftar aset (NilaiBukuPerAkun). keterangan = petunjuk
// per kode dari template lama, dipakai bila nama akunnya sama.
func TemplateSaldoAwal(active []Account, peta Peta, petunjuk []finance.SaldoAkun) []finance.SaldoAkun {
	var out []finance.SaldoAkun
	for _, a := range active {
		kat := KategoriSaldoAkun(a, peta)
		if kat == "" || kat == finance.KatAsetTetap {
			continue
		}
		b := finance.SaldoAkun{Kode: a.Kode, Nama: a.Nama, Kategori: kat}
		if i := slices.IndexFunc(petunjuk, func(p finance.SaldoAkun) bool { return strings.EqualFold(p.Nama, a.Nama) }); i >= 0 {
			b.Keterangan = petunjuk[i].Keterangan
		}
		out = append(out, b)
	}
	return out
}

// NilaiBukuPerAkun: nilai buku awal daftar aset per akun harta CoA (aset yang diperoleh sebelum
// periode), sebagai baris saldo awal kategori aset tetap. Jumlahnya = AssetSummary.NilaiBukuAwal.
func NilaiBukuPerAkun(as finance.AssetSummary, period string, akun []Account, peta Peta) []finance.SaldoAkun {
	per := map[string]int64{}
	jumlah := map[string]int{}
	for _, r := range as.Assets {
		if diperolehPadaPeriode(r, period) || r.TanggalPerolehan[:min(7, len(r.TanggalPerolehan))] > period {
			continue
		}
		harta, _, _ := akunAset(r.Asset, peta)
		per[harta] += r.NilaiBukuAwal
		jumlah[harta]++
	}
	var out []finance.SaldoAkun
	for _, a := range akun {
		if n, ok := per[a.Kode]; ok {
			out = append(out, finance.SaldoAkun{Kode: a.Kode, Nama: a.Nama, Kategori: finance.KatAsetTetap, Nominal: n,
				Keterangan: "nilai buku " + strconv.Itoa(jumlah[a.Kode]) + " aset di daftar aset"})
			delete(per, a.Kode)
		}
	}
	for kode, n := range per { // akun peran tidak ada di daftar akun (seharusnya tidak terjadi)
		out = append(out, finance.SaldoAkun{Kode: kode, Nama: "Aset tetap", Kategori: finance.KatAsetTetap, Nominal: n})
	}
	return out
}

// kataAkun: kata bermakna nama akun (huruf kecil, "hutang" = "utang", tanpa kata sambung).
func kataAkun(s string) []string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return ' '
	}, s)
	var out []string
	for _, w := range strings.Fields(s) {
		if w == "hutang" {
			w = "utang"
		}
		if slices.Contains([]string{"dan", "di", "yang", "atau", "ke", "dari", "untuk", "lain", "lainnya", "rekening"}, w) {
			continue
		}
		out = append(out, w)
	}
	return out
}

// CocokAkunSaldo mencari akun neraca yang namanya cocok dengan ketikan pengguna. persis = nama
// sama (barisnya akun itu sendiri); selain itu semua kata nama akun ada di ketikan ("Bank BCA"
// → akun Bank): kategorinya dipakai, tapi barisnya akun baru dengan kode sendiri.
func CocokAkunSaldo(nama string, active []Account, peta Peta) (acc Account, persis, ok bool) {
	kata := kataAkun(nama)
	if len(kata) == 0 {
		return Account{}, false, false
	}
	terbaik := 0
	for _, a := range active {
		if KategoriSaldoAkun(a, peta) == "" {
			continue
		}
		ka := kataAkun(a.Nama)
		if len(ka) > 0 && slices.Equal(ka, kata) {
			return a, true, true
		}
		cocok := 0
		for _, w := range ka {
			if slices.Contains(kata, w) {
				cocok++
			}
		}
		if cocok == len(ka) && cocok > terbaik {
			acc, terbaik, ok = a, cocok, true
		}
	}
	return acc, false, ok
}

// KodeBaruSaldo: kode 4 angka berikutnya yang belum dipakai di kelompok akun kategori itu (mis.
// rekening bank kedua di CoA dagang → 1105). dipakai = kode yang sudah ada di layar.
func KodeBaruSaldo(kategori string, active []Account, peta Peta, dipakai []string) string {
	// Kelompok kode mengikuti akun utama kategorinya (mis. aset lancar lain → Biaya Dibayar
	// Dimuka 14xx), bukan akun pertama yang kebetulan berkategori sama.
	utama := map[string]string{finance.KatKas: PeranKas, finance.KatPiutang: PeranPiutang, finance.KatPersediaan: PeranPersediaan,
		finance.KatAsetLain: PeranDibayarDimuka, finance.KatUtang: PeranUtangUsaha, finance.KatPinjaman: PeranPinjaman,
		finance.KatKewajibanLain: PeranUtangLain, finance.KatModal: PeranModal, finance.KatLabaDitahan: PeranLabaDitahan}
	var awalan string
	if k := peta.Kode(utama[kategori]); len(k) == 4 {
		awalan = k[:2]
	}
	for _, a := range active {
		if awalan != "" {
			break
		}
		if KategoriSaldoAkun(a, peta) == kategori {
			awalan = a.Kode[:2]
			break
		}
	}
	if awalan == "" {
		return ""
	}
	terpakai := map[string]bool{}
	for _, a := range active {
		terpakai[a.Kode] = true
	}
	for _, k := range dipakai {
		terpakai[strings.TrimSpace(k)] = true
	}
	maks := 0
	for k := range terpakai {
		if strings.HasPrefix(k, awalan) && len(k) == 4 {
			if n, err := strconv.Atoi(k); err == nil && n > maks {
				maks = n
			}
		}
	}
	if maks == 0 {
		maks, _ = strconv.Atoi(awalan + "00")
	}
	for n := maks + 1; n%100 != 0; n++ {
		if k := strconv.Itoa(n); !terpakai[k] {
			return k
		}
	}
	return ""
}

// kataKategori: kata yang lazim diketik pemilik untuk nama akun, di luar istilah akuntansi.
var kataKategori = []struct {
	kategori string
	kata     []string
}{
	{finance.KatPinjaman, []string{"pinjaman", "kredit", "kur", "leasing", "cicilan", "koperasi"}},
	{finance.KatKas, []string{"tabungan", "giro", "deposito", "ovo", "gopay", "dana", "shopeepay", "brankas", "bca", "bri", "bni", "mandiri", "bsi", "cimb", "danamon", "permata", "btn", "jago", "seabank", "blu"}},
	{finance.KatAsetTetap, []string{"mobil", "motor", "truk", "pickup", "pick", "kendaraan", "mesin", "laptop", "komputer", "printer", "tanah", "bangunan", "gedung", "ruko", "rumah", "gudang", "etalase", "freezer", "kulkas", "oven", "peralatan", "inventaris", "furnitur", "mebel", "kandang"}},
	{finance.KatPersediaan, []string{"stok", "persediaan", "bahan", "barang"}},
}

// TebakKategoriNama: kategori dari kata di nama akun ("Tabungan Mandiri" → kas, "Mobil pickup"
// → aset tetap), lalu istilah akuntansi (finance.KategoriSaldoDari). Dipakai bila AI tidak aktif.
func TebakKategoriNama(nama string) string {
	kata := kataAkun(nama)
	for _, k := range kataKategori {
		for _, w := range kata {
			if slices.Contains(k.kata, w) {
				// "Utang ke Bank Mandiri" = pinjaman bank, bukan rekening kas.
				if k.kategori == finance.KatKas && slices.Contains(kata, "utang") {
					return finance.KatPinjaman
				}
				return k.kategori
			}
		}
	}
	if k, ok := finance.KategoriSaldoDari(nama); ok {
		return k.ID
	}
	return ""
}
