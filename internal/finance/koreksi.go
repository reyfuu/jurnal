package finance

import (
	"errors"
	"slices"
	"strings"
	"time"
)

// Koreksi: usulan tim MasterPajak untuk mengubah atau menghapus satu transaksi klien. Pembukuan
// tidak berubah sampai pemilik usaha menyetujuinya. Sebelum = potret transaksi saat diusulkan;
// bila transaksinya kemudian berubah, usulan basi dan tidak boleh diterapkan.
type Koreksi struct {
	ID          int          `json:"id"`
	TransaksiID int          `json:"transaksi_id"`
	Jenis       string       `json:"jenis"` // ubah | hapus
	Sebelum     Transaction  `json:"sebelum"`
	Usulan      *Transaction `json:"usulan,omitempty"` // hanya untuk ubah
	Alasan      string       `json:"alasan"`
	OlehID      int          `json:"oleh_id"`
	Oleh        string       `json:"oleh"`
	Waktu       time.Time    `json:"waktu"`
	Status      string       `json:"status"`
	// Catatan & keputusan pemilik usaha.
	Catatan        string    `json:"catatan,omitempty"`
	DiputuskanOleh string    `json:"diputuskan_oleh,omitempty"`
	DiputuskanPada time.Time `json:"diputuskan_pada,omitzero"`
}

const (
	KoreksiUbah  = "ubah"
	KoreksiHapus = "hapus"

	KoreksiMenunggu    = "menunggu"
	KoreksiDisetujui   = "disetujui"
	KoreksiDitolak     = "ditolak"
	KoreksiKedaluwarsa = "kedaluwarsa" // transaksi berubah/hilang sebelum diputuskan
)

// ErrKoreksiBasi: transaksi sudah berbeda dari potret saat usulan dibuat.
var ErrKoreksiBasi = errors.New("transaksi sudah berubah sejak usulan dibuat")

func txIndex(txs []Transaction, id int) int {
	return slices.IndexFunc(txs, func(t Transaction) bool { return t.ID == id })
}

// SiapkanKoreksi memvalidasi usulan terhadap transaksi saat ini dan mengisi potret Sebelum.
// Cek akun CoA (bergantung profil usaha) dilakukan pemanggil.
func SiapkanKoreksi(k *Koreksi, txs []Transaction) error {
	k.Alasan = strings.TrimSpace(k.Alasan)
	if k.Alasan == "" || len([]rune(k.Alasan)) > 500 {
		return errors.New("alasan wajib diisi (maks 500 karakter)")
	}
	i := txIndex(txs, k.TransaksiID)
	if i < 0 {
		return errors.New("transaksi tidak ditemukan")
	}
	k.Sebelum = txs[i]
	switch k.Jenis {
	case KoreksiHapus:
		k.Usulan = nil
	case KoreksiUbah:
		if k.Usulan == nil {
			return errors.New("isi transaksi yang diusulkan")
		}
		u := *k.Usulan
		u.ID = k.TransaksiID
		if err := Validate(&u); err != nil {
			return err
		}
		if u == k.Sebelum {
			return errors.New("usulan sama dengan transaksi sekarang")
		}
		k.Usulan = &u
	default:
		return errors.New("jenis koreksi harus ubah atau hapus")
	}
	k.Status = KoreksiMenunggu
	return nil
}

// TerapkanKoreksi mengembalikan daftar transaksi baru (txs tidak diubah).
func TerapkanKoreksi(txs []Transaction, k Koreksi) ([]Transaction, error) {
	i := txIndex(txs, k.TransaksiID)
	if i < 0 || txs[i] != k.Sebelum {
		return nil, ErrKoreksiBasi
	}
	out := slices.Clone(txs)
	if k.Jenis == KoreksiHapus {
		return slices.Delete(out, i, i+1), nil
	}
	out[i] = *k.Usulan
	return out, nil
}

// DampakKoreksi: laba bersih satu bulan sebelum & sesudah koreksi diterapkan (tanpa penyusutan
// dari register aset, yang tidak ikut berubah).
type DampakKoreksi struct {
	Periode     string `json:"periode"`
	LabaSebelum int64  `json:"laba_sebelum"`
	LabaSesudah int64  `json:"laba_sesudah"`
}

// HitungDampak: bulan yang tersentuh koreksi, urut. Memakai potret Sebelum supaya dampak tetap
// terbaca walau transaksinya sudah berubah.
func HitungDampak(txs []Transaction, k Koreksi) []DampakKoreksi {
	before := slices.Clone(txs)
	if i := txIndex(before, k.TransaksiID); i >= 0 {
		before[i] = k.Sebelum
	}
	after := slices.DeleteFunc(slices.Clone(before), func(t Transaction) bool { return t.ID == k.TransaksiID })
	periode := []string{k.Sebelum.Tanggal[:min(7, len(k.Sebelum.Tanggal))]}
	if k.Jenis == KoreksiUbah && k.Usulan != nil {
		after = append(after, *k.Usulan)
		periode = append(periode, k.Usulan.Tanggal[:min(7, len(k.Usulan.Tanggal))])
	}
	slices.Sort(periode)
	periode = slices.Compact(periode)
	laba := func(txs []Transaction, p string) int64 {
		return BuildIncomeStatement(ToLedger(ForPeriod(txs, p)), 0, 0).NetProfit
	}
	out := make([]DampakKoreksi, 0, len(periode))
	for _, p := range periode {
		out = append(out, DampakKoreksi{p, laba(before, p), laba(after, p)})
	}
	return out
}
