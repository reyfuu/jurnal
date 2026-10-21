package finance

import (
	"errors"
	"fmt"
	"strings"
)

// Saldo awal per akun: tiap baris satu akun (dari CoA atau ditambah sendiri oleh klien) dengan
// kategorinya. Kategori, bukan kode akun, yang menentukan ke pos neraca mana barisnya masuk,
// supaya klien bisa menambah akun sendiri ("Rekening BCA kedua") tanpa mengenal kode CoA.

type SaldoAkun struct {
	Kode       string `json:"kode,omitempty"`
	Nama       string `json:"nama"`
	Kategori   string `json:"kategori"`
	Nominal    int64  `json:"nominal"`
	Keterangan string `json:"keterangan,omitempty"`
}

// Kategori saldo awal. Label dipakai di template Excel (dropdown) dan layar.
const (
	KatKas           = "kas"
	KatPiutang       = "piutang"
	KatPersediaan    = "persediaan"
	KatAsetLain      = "aset_lain"
	KatAsetTetap     = "aset_tetap"
	KatUtang         = "utang"
	KatPinjaman      = "pinjaman"
	KatKewajibanLain = "kewajiban_lain"
	KatModal         = "modal"
	KatLabaDitahan   = "laba_ditahan"
)

type KategoriSaldo struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Sisi  string `json:"sisi"` // harta | kewajiban | modal
}

// KategoriSaldoList: urutan tampil. Label diawali sisinya ("Harta: …") supaya rumus SUMIF di
// template bisa menjumlah per sisi, dan total panjangnya muat di dropdown Excel (≤255 huruf).
var KategoriSaldoList = []KategoriSaldo{
	{KatKas, "Harta: Kas & Bank", "harta"},
	{KatPiutang, "Harta: Piutang Usaha", "harta"},
	{KatPersediaan, "Harta: Persediaan", "harta"},
	{KatAsetLain, "Harta: Aset Lancar Lain", "harta"},
	{KatAsetTetap, "Harta: Aset Tetap (nilai buku)", "harta"},
	{KatUtang, "Kewajiban: Utang Usaha", "kewajiban"},
	{KatPinjaman, "Kewajiban: Pinjaman", "kewajiban"},
	{KatKewajibanLain, "Kewajiban: Lain-lain", "kewajiban"},
	{KatModal, "Modal: Modal Disetor", "modal"},
	{KatLabaDitahan, "Modal: Laba Ditahan", "modal"},
}

// KategoriSaldoDari mengenali kategori dari id atau label (juga ketikan bebas yang mirip).
func KategoriSaldoDari(s string) (KategoriSaldo, bool) {
	n := norm(s)
	for _, k := range KategoriSaldoList {
		if n == norm(k.Label) || n == k.ID || n == norm(strings.ReplaceAll(k.ID, "_", " ")) {
			return k, true
		}
	}
	has := func(w ...string) bool {
		for _, x := range w {
			if strings.Contains(n, x) {
				return true
			}
		}
		return false
	}
	id := ""
	switch {
	case has("laba ditahan", "saldo laba", "rugi ditahan"):
		id = KatLabaDitahan
	case has("modal", "ekuitas"):
		id = KatModal
	case has("aset tetap", "nilai buku", "peralatan", "kendaraan", "gedung", "mesin"):
		id = KatAsetTetap
	case has("pinjaman", "hutang bank", "utang bank", "kredit bank", "leasing"):
		id = KatPinjaman
	case has("kewajiban", "utang pajak", "hutang pajak", "masih harus dibayar") && !has("utang usaha", "hutang usaha"):
		id = KatKewajibanLain
	case has("utang", "hutang"):
		id = KatUtang
	case has("piutang usaha") || n == "piutang":
		id = KatPiutang
	case has("persediaan", "stok"):
		id = KatPersediaan
	case has("kas", "bank", "tunai", "wallet", "rekening"):
		id = KatKas
	case has("harta", "aset", "uang muka", "dibayar dimuka", "dibayar di muka", "piutang"):
		id = KatAsetLain
	}
	for _, k := range KategoriSaldoList {
		if k.ID == id {
			return k, true
		}
	}
	return KategoriSaldo{}, false
}

// TerapkanRincian: rapikan & periksa tiap baris, lalu turunkan angka per pos dari Rincian.
// Tanpa rincian (saldo awal lama / isian pos langsung) tidak mengubah apa pun.
func (o *OpeningBalance) TerapkanRincian() error {
	if len(o.Rincian) == 0 {
		return nil
	}
	o.Kas, o.Piutang, o.Persediaan, o.AsetLain, o.AsetTetap = 0, 0, 0, 0, 0
	o.Utang, o.Pinjaman, o.KewajibanLain, o.ModalDisetor, o.LabaDitahan = 0, 0, 0, 0, 0
	out := o.Rincian[:0]
	for i, r := range o.Rincian {
		r.Kode, r.Nama, r.Keterangan = strings.TrimSpace(r.Kode), strings.TrimSpace(r.Nama), strings.TrimSpace(r.Keterangan)
		if r.Nama == "" && r.Nominal == 0 {
			continue // baris kosong
		}
		k, ok := KategoriSaldoDari(r.Kategori)
		if !ok {
			return fmt.Errorf("baris %d (%s): kategori akun %q tidak dikenal; pilih salah satu kategori Harta / Kewajiban / Modal", i+1, r.Nama, r.Kategori)
		}
		r.Kategori = k.ID
		if r.Nama == "" {
			r.Nama = k.Label
		}
		if r.Nominal < 0 && k.ID != KatLabaDitahan {
			return fmt.Errorf("baris %d (%s): nominal tidak boleh negatif; hanya Laba Ditahan yang boleh minus (rugi)", i+1, r.Nama)
		}
		switch k.ID {
		case KatKas:
			o.Kas += r.Nominal
		case KatPiutang:
			o.Piutang += r.Nominal
		case KatPersediaan:
			o.Persediaan += r.Nominal
		case KatAsetLain:
			o.AsetLain += r.Nominal
		case KatAsetTetap:
			o.AsetTetap += r.Nominal
		case KatUtang:
			o.Utang += r.Nominal
		case KatPinjaman:
			o.Pinjaman += r.Nominal
		case KatKewajibanLain:
			o.KewajibanLain += r.Nominal
		case KatModal:
			o.ModalDisetor += r.Nominal
		case KatLabaDitahan:
			o.LabaDitahan += r.Nominal
		}
		out = append(out, r)
	}
	o.Rincian = out
	if len(out) == 0 {
		return errors.New("saldo awal belum diisi")
	}
	return nil
}
