package coa

import (
	"errors"
	"fmt"
	"slices"

	"pajak/internal/finance"
)

// Catat cepat multi-akun: pengguna mengisi baris jurnal (akun, debit / kredit) seperti jurnal
// umum. Supaya omzet, pajak, dashboard, dan laporan ringkas tetap memakai data yang sama dengan
// CoA, catatan itu disimpan sebagai beberapa transaksi biasa yang masing-masing berpasangan
// dengan akun lawan (kas/bank/piutang/utang) dan akun pokoknya ditetapkan (Akun).
//
// Syarat: satu sisi (debit atau kredit) berisi hanya akun lawan, sisi lain hanya akun pokok.

type BarisCatat struct {
	Akun       string `json:"akun"`
	Keterangan string `json:"keterangan,omitempty"`
	Debit      int64  `json:"debit"`
	Kredit     int64  `json:"kredit"`
	JatuhTempo string `json:"jatuh_tempo,omitempty"` // akun utang / piutang
}

// Jenis lawan: rekening (kas/bank/e-wallet; lunas), piutang usaha, utang usaha (belum lunas).
const (
	LawanRekening = "rekening"
	LawanPiutang  = "piutang"
	LawanUtang    = "utang"
)

// JenisLawan: "" bila akun bukan akun lawan.
func JenisLawan(a Account, peta Peta) string {
	switch {
	case a.Kelompok == AsetLancar && slices.Contains([]string{"cash", "bank", "ewallet"}, a.Subtype):
		return LawanRekening
	case a.Kode == peta.Kode(PeranPiutang):
		return LawanPiutang
	case a.Kode == peta.Kode(PeranUtangUsaha):
		return LawanUtang
	}
	return ""
}

// JenisUntukAkun: jenis transaksi untuk akun pokok di sisi tertentu (masuk = pokok di kredit).
// Jenis yang akun otomatisnya sama dipakai lebih dulu; selain itu dari kelompok akunnya.
func JenisUntukAkun(a Account, masuk bool, p finance.BusinessProfile, peta Peta) (string, error) {
	arah := "keluar"
	if masuk {
		arah = "masuk"
	}
	for _, j := range finance.JenisList {
		if j.Arah == arah && AutoAccount(j.Nama, p, peta) == a.Kode {
			return j.Nama, nil
		}
	}
	if masuk {
		switch a.Kelompok {
		case PendapatanUsaha:
			return "Penjualan", nil
		case PendapatanLain:
			return "Pendapatan Lain", nil
		case LiabilitasJangka:
			return "Terima Pinjaman", nil
		case Ekuitas:
			if a.Normal == Kredit {
				return "Setor Modal", nil
			}
		}
	} else {
		switch {
		case a.Subtype == "inventory" || a.Kelompok == HPP:
			return "Beli Bahan / Stok", nil
		case a.Kelompok == AsetTidakLancar && (a.Subtype == "fixed_asset" || a.Subtype == "intangible"):
			return "Beli Alat / Aset", nil
		case a.Kelompok == BebanOperasional || a.Kelompok == BebanLain:
			return "Biaya Lain", nil
		case a.Kelompok == LiabilitasPendek || a.Kelompok == LiabilitasJangka:
			return "Bayar Cicilan", nil
		}
	}
	sisi := "debit"
	if masuk {
		sisi = "kredit"
	}
	return "", fmt.Errorf("akun %s %s belum bisa dicatat di sisi %s lewat Catat cepat", a.Kode, a.Nama, sisi)
}

// PecahCatat mengubah baris jurnal menjadi transaksi (tanpa ID, tanggal, pihak — diisi pemanggil).
// Akun pokok dipasangkan dengan akun lawan berurutan; satu baris pokok bisa terbagi ke beberapa
// lawan (mis. sebagian tunai, sisanya utang).
func PecahCatat(baris []BarisCatat, akun []Account, p finance.BusinessProfile, peta Peta) ([]finance.Transaction, error) {
	type bagian struct {
		b     BarisCatat
		a     Account
		lawan string
		n     int64
	}
	var debit, kredit []bagian
	var totalD, totalK int64
	for i, b := range baris {
		if b.Debit == 0 && b.Kredit == 0 {
			continue
		}
		if b.Debit < 0 || b.Kredit < 0 || (b.Debit > 0 && b.Kredit > 0) {
			return nil, fmt.Errorf("baris %d: isi debit atau kredit saja, tidak boleh minus", i+1)
		}
		a, ok := Find(akun, b.Akun)
		if !ok {
			return nil, fmt.Errorf("baris %d: akun %q tidak ada di daftar akun usaha ini", i+1, b.Akun)
		}
		x := bagian{b: b, a: a, lawan: JenisLawan(a, peta)}
		if b.Debit > 0 {
			x.n, totalD = b.Debit, totalD+b.Debit
			debit = append(debit, x)
		} else {
			x.n, totalK = b.Kredit, totalK+b.Kredit
			kredit = append(kredit, x)
		}
	}
	if len(debit) == 0 || len(kredit) == 0 {
		return nil, errors.New("isi minimal satu baris debit dan satu baris kredit")
	}
	if totalD != totalK {
		return nil, fmt.Errorf("debit %s dan kredit %s belum seimbang", finance.FormatRupiahPenuh(totalD), finance.FormatRupiahPenuh(totalK))
	}
	semuaLawan := func(xs []bagian) bool {
		return !slices.ContainsFunc(xs, func(x bagian) bool { return x.lawan == "" })
	}
	semuaRekening := func(xs []bagian) bool {
		return !slices.ContainsFunc(xs, func(x bagian) bool { return x.lawan != LawanRekening })
	}
	var pokok, lawan []bagian
	masuk := false
	switch {
	case semuaLawan(kredit) && !semuaLawan(debit):
		pokok, lawan = debit, kredit
	case semuaLawan(debit) && !semuaLawan(kredit):
		pokok, lawan, masuk = kredit, debit, true
	case semuaLawan(debit) && semuaLawan(kredit):
		// antar rekening sendiri = Pindah Dana; piutang/utang tetap lewat menu Piutang & Utang
		if !semuaRekening(debit) || !semuaRekening(kredit) {
			return nil, errors.New("pelunasan piutang/utang dicatat di menu Piutang & Utang; Catat cepat hanya untuk pindah dana antar Kas/Bank")
		}
		pokok, lawan = debit, kredit
	default:
		return nil, errors.New("satu sisi harus berisi akun Kas/Bank, Piutang Usaha, atau Utang Usaha saja (sisi pembayaran); sisi lainnya akun pendapatan, beban, persediaan, atau aset")
	}
	for _, l := range lawan {
		if masuk && l.lawan == LawanUtang || !masuk && l.lawan == LawanPiutang {
			return nil, fmt.Errorf("akun %s %s tidak cocok di sisi ini; uang masuk diterima di Kas/Bank atau jadi Piutang, uang keluar dari Kas/Bank atau jadi Utang", l.a.Kode, l.a.Nama)
		}
	}
	var out []finance.Transaction
	li := 0
	for _, po := range pokok {
		jenis := finance.JenisPindahDana
		if po.lawan != LawanRekening {
			var err error
			if jenis, err = JenisUntukAkun(po.a, masuk, p, peta); err != nil {
				return nil, err
			}
		}
		for sisa := po.n; sisa > 0; {
			l := &lawan[li]
			n := min(sisa, l.n)
			t := finance.Transaction{Jenis: jenis, Nominal: n, Keterangan: po.b.Keterangan, Akun: po.a.Kode, Lunas: l.lawan == LawanRekening}
			if jenis == finance.JenisPindahDana {
				if l.lawan != LawanRekening || l.a.Kode == po.a.Kode {
					return nil, fmt.Errorf("pindah dana: rekening asal dan tujuan harus Kas/Bank yang berbeda (%s %s)", po.a.Kode, po.a.Nama)
				}
				if masuk { // pokok di sisi kredit = rekening asal
					t.Akun, t.Lawan = l.a.Kode, po.a.Kode
				}
			}
			if t.Lunas && t.Lawan == "" {
				t.Lawan = l.a.Kode
			} else if !t.Lunas {
				t.JatuhTempo = l.b.JatuhTempo
			}
			out = append(out, t)
			sisa -= n
			if l.n -= n; l.n == 0 {
				li++
			}
		}
	}
	return out, nil
}

// SusunCatat: kebalikan PecahCatat untuk layar ubah — transaksi satu grup jadi baris jurnal
// (pokok per akun+keterangan, lawan per akun+jatuh tempo).
func SusunCatat(txs []finance.Transaction, p finance.BusinessProfile, peta Peta) []BarisCatat {
	var pokok, lawan []BarisCatat
	tambah := func(xs *[]BarisCatat, b BarisCatat) {
		for i := range *xs {
			x := &(*xs)[i]
			if x.Akun == b.Akun && x.Keterangan == b.Keterangan && x.JatuhTempo == b.JatuhTempo {
				x.Debit += b.Debit
				x.Kredit += b.Kredit
				return
			}
		}
		*xs = append(*xs, b)
	}
	for _, t := range txs {
		j, _ := finance.LookupJenis(t.Jenis)
		masuk := j.Arah == "masuk"
		lw := CounterAccount(t, peta)
		tempo := ""
		if !t.Lunas {
			tempo = t.JatuhTempo
		}
		if masuk {
			tambah(&lawan, BarisCatat{Akun: lw, Debit: t.Nominal, JatuhTempo: tempo})
			tambah(&pokok, BarisCatat{Akun: AccountFor(t, p, peta), Keterangan: t.Keterangan, Kredit: t.Nominal})
		} else {
			tambah(&pokok, BarisCatat{Akun: AccountFor(t, p, peta), Keterangan: t.Keterangan, Debit: t.Nominal})
			tambah(&lawan, BarisCatat{Akun: lw, Kredit: t.Nominal, JatuhTempo: tempo})
		}
	}
	return append(pokok, lawan...)
}
