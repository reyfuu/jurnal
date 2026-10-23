package coa

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"pajak/internal/finance"
)

// Layar Saldo Awal memakai CoA usaha itu sendiri (dagang / manufaktur), bukan daftar tetap:
// kode & nama akun ikut CoA-nya, kategori neraca diturunkan dari kelompok, subtype, dan peran.

// KodeSelisihSaldoAwal: akun laba ditahan penampung selisih saldo awal yang disimpan walau belum
// seimbang (Harta ≠ Utang + Modal). Tidak diketik pengguna; diisi server saat simpan paksa.
const KodeSelisihSaldoAwal = "3299"

// AkunSelisihSaldoAwal: akun penampung di CoA usaha ini; template tanpa akun itu memakai akun
// Laba Ditahan biasa.
func AkunSelisihSaldoAwal(all []Account, peta Peta) Account {
	if a, ok := Find(all, KodeSelisihSaldoAwal); ok {
		return a
	}
	a, _ := Find(all, peta.Kode(PeranLabaDitahan))
	return a
}

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
		if kat == "" || kat == finance.KatAsetTetap || a.Kode == KodeSelisihSaldoAwal {
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
		switch w {
		case "hutang":
			w = "utang"
		case "dimuka": // "Dibayar Dimuka" = "Dibayar di Muka"
			w = "muka"
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
	var awalan string
	if k := peta.Kode(peranKategori[kategori]); len(k) == 4 {
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

// peranKategori: akun peran tiap kategori saldo awal.
var peranKategori = map[string]string{finance.KatKas: PeranKas, finance.KatPiutang: PeranPiutang, finance.KatPersediaan: PeranPersediaan,
	finance.KatAsetLain: PeranDibayarDimuka, finance.KatUtang: PeranUtangUsaha, finance.KatPinjaman: PeranPinjaman,
	finance.KatKewajibanLain: PeranUtangLain, finance.KatModal: PeranModal, finance.KatLabaDitahan: PeranLabaDitahan}

// KodeSaldo: kode untuk baris saldo awal baru. Kategori yang boleh berakun sendiri mendapat kode
// baru (baru = true); kategori lain dibukukan ke akun perannya, jadi kodenya kode akun itu.
func KodeSaldo(kategori string, active []Account, peta Peta, dipakai []string) (kode string, baru bool) {
	if _, ok := akunSendiri[kategori]; ok {
		k := KodeBaruSaldo(kategori, active, peta, dipakai)
		return k, k != ""
	}
	return peta.Kode(peranKategori[kategori]), false
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

// CatatanSaldoAwal menandai akun yang ditambahkan klien lewat Saldo Awal, bukan dari template.
const CatatanSaldoAwal = "Ditambahkan sendiri di Saldo Awal"

// akunSendiri: kategori saldo awal yang barisnya boleh menjadi akun tersendiri, beserta akun
// peran yang ditiru (kelompok, saldo normal, subtype).
//
// ponytail: piutang, persediaan, utang usaha, pinjaman, dan modal tetap di akun perannya karena
// sub-bukunya (pelunasan saldo awal, kartu stok, cicilan) menggerakkan akun peran. Akun
// tersendiri untuk kategori itu baru masuk akal setelah sub-bukunya per akun.
var akunSendiri = map[string]string{finance.KatKas: PeranKas, finance.KatAsetLain: PeranDibayarDimuka, finance.KatKewajibanLain: PeranUtangLain}

// AkunSaldoAwal: daftar akun ditambah akun dari baris saldo awal yang kodenya belum ada di
// daftar (akun yang ditambahkan sendiri). Diturunkan dari saldo awal tersimpan, jadi tidak perlu
// tabel sendiri; all tidak diubah.
func AkunSaldoAwal(rincian []finance.SaldoAkun, all []Account, peta Peta) []Account {
	out := all
	for _, r := range rincian {
		peran, ok := akunSendiri[r.Kategori]
		if !ok || len(r.Kode) != 4 {
			continue
		}
		if _, ada := Find(out, r.Kode); ada {
			continue
		}
		if len(out) == len(all) {
			out = slices.Clone(all) // template dipakai bersama semua usaha
		}
		induk, _ := Find(all, peta.Kode(peran))
		i := slices.IndexFunc(out, func(a Account) bool { return a.Kode > r.Kode })
		if i < 0 {
			i = len(out)
		}
		out = slices.Insert(out, i, Account{Kode: r.Kode, Nama: r.Nama, Subtype: induk.Subtype, Normal: induk.Normal,
			Kelompok: induk.Kelompok, Kondisi: Selalu, Catatan: CatatanSaldoAwal})
	}
	return out
}

// IsiKodeSaldo memberi kode pada baris saldo awal tanpa kode (akun ditambah sendiri, atau jenis
// yang diketik bebas di template Excel): nama sama dengan baris lain → kodenya; nama sama
// dengan akun CoA berkategori sama → akun itu; selain itu kode baru di kelompoknya. Kategori
// yang tetap di akun perannya tidak diberi kode.
func IsiKodeSaldo(rincian []finance.SaldoAkun, active []Account, peta Peta) {
	kode := map[string]string{} // nama (huruf kecil) → kode
	var dipakai []string
	for _, r := range rincian {
		if r.Kode == "" {
			continue
		}
		dipakai = append(dipakai, r.Kode)
		if _, ok := akunSendiri[r.Kategori]; ok {
			kode[strings.ToLower(strings.TrimSpace(r.Nama))] = cmp.Or(kode[strings.ToLower(strings.TrimSpace(r.Nama))], r.Kode)
		}
	}
	for i := range rincian {
		r := &rincian[i]
		if _, ok := akunSendiri[r.Kategori]; !ok || r.Kode != "" {
			continue
		}
		nama := strings.ToLower(strings.TrimSpace(r.Nama))
		k := kode[nama]
		if a, persis, ok := CocokAkunSaldo(r.Nama, active, peta); k == "" && ok && persis && KategoriSaldoAkun(a, peta) == r.Kategori {
			k = a.Kode
		}
		if k == "" {
			k = KodeBaruSaldo(r.Kategori, active, peta, dipakai)
		}
		r.Kode, kode[nama] = k, k
		dipakai = append(dipakai, k)
	}
}

// KasAwalLain: saldo awal baris kas yang punya akun sendiri (akun CoA bernama sama, mis. Bank,
// atau akun tambahan), per kode. Sisanya tetap di akun Kas.
func KasAwalLain(ob finance.OpeningBalance, akun []Account, peta Peta) map[string]int64 {
	out := map[string]int64{}
	for _, r := range ob.Rincian {
		if r.Kategori == finance.KatKas && r.Kode != peta.Kode(PeranKas) && kodeSendiri(r, akun) {
			out[r.Kode] += r.Nominal
		}
	}
	return out
}

// kodeSendiri: baris saldo awal dibukukan ke akun berkodenya sendiri bila akun itu memang akun
// yang sama (nama sama) atau ditambahkan sendiri lewat Saldo Awal. Kode template saldo awal lama
// mengikuti CoA dagang; di CoA lain kode yang sama bisa berarti akun lain (manufaktur: 1401 =
// Uang Muka Pembelian).
func kodeSendiri(r finance.SaldoAkun, akun []Account) bool {
	acc, ok := Find(akun, r.Kode)
	return ok && r.Kode != "" && (acc.Catatan == CatatanSaldoAwal || strings.EqualFold(strings.TrimSpace(acc.Nama), strings.TrimSpace(r.Nama)))
}

// CatatanTambahan menandai akun yang ditambahkan klien di Daftar Akun.
const CatatanTambahan = "Ditambahkan sendiri di Daftar Akun"

// AkunTambahanKlien: daftar akun ditambah akun buatan klien (profil.AkunTambahan). Jenisnya
// disalin dari akun induk, tanpa peran jurnal otomatis; all tidak diubah.
func AkunTambahanKlien(list []finance.AkunTambahan, all []Account) []Account {
	out := all
	for _, t := range list {
		induk, ok := Find(all, t.Induk)
		if _, ada := Find(out, t.Kode); !ok || ada || len(t.Kode) != 4 {
			continue
		}
		if len(out) == len(all) {
			out = slices.Clone(all)
		}
		i := slices.IndexFunc(out, func(a Account) bool { return a.Kode > t.Kode })
		if i < 0 {
			i = len(out)
		}
		out = slices.Insert(out, i, Account{Kode: t.Kode, Nama: t.Nama, Subtype: induk.Subtype, Normal: induk.Normal,
			Kelompok: induk.Kelompok, Kondisi: Selalu, Catatan: CatatanTambahan})
	}
	return out
}

// IndukNomor: akun yang jenisnya ditiru akun baru bernomor kode — akun terdekat di seri ratusan
// yang sama (1103 → 1102 / 1101); seri yang belum ada memakai akun pertama di kelompok pilihan.
// Error bila nomor tidak 4 angka, sudah dipakai, atau seri baru tanpa kelompok.
func IndukNomor(all []Account, kode string, kelompok Kelompok) (string, error) {
	if n, err := strconv.Atoi(kode); err != nil || len(kode) != 4 || n < 1000 {
		return "", errors.New("nomor akun harus 4 angka, mis. 1103")
	}
	if a, ada := Find(all, kode); ada {
		return "", fmt.Errorf("nomor %s sudah dipakai akun %s", kode, a.Nama)
	}
	induk := ""
	for _, a := range all { // all urut kode: akun terakhir di bawah kode, selain itu yang pertama di atasnya
		if a.Kode[:min(2, len(a.Kode))] != kode[:2] {
			continue
		}
		if a.Kode < kode || induk == "" {
			induk = a.Kode
		}
	}
	if induk != "" {
		return induk, nil
	}
	if kelompok == "" {
		return "", fmt.Errorf("seri %sxx belum ada di daftar akun; pilih kelompoknya", kode[:2])
	}
	for _, a := range all {
		if a.Kelompok == kelompok {
			return a.Kode, nil
		}
	}
	return "", fmt.Errorf("kelompok %q tidak dikenal", kelompok)
}
