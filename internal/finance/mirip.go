package finance

import (
	"slices"
	"time"
)

// Cegah jurnal dobel: uang masuk/keluar yang sama bisa tercatat dua kali, mis. dicatat dari foto
// nota lalu ikut lagi saat impor mutasi bank. Transaksi dianggap mirip bila arahnya sama,
// nominalnya sama, dan tanggal uangnya bergerak berselisih paling lama ToleransiHariMirip hari
// (bank sering membukukan sehari-dua sesudahnya). Transaksi yang belum lunas juga dicocokkan:
// mutasi itu kemungkinan pelunasannya, bukan transaksi baru.

const (
	ToleransiHariMirip = 3
	// tagihan belum lunas dicari sampai sejauh ini sebelum tanggal mutasi
	maksHariTagihan = 180
)

// Mirip: transaksi (atau satu invoice: DPP + PPN-nya) yang kemungkinan sama.
type Mirip struct {
	ID         int    `json:"id"` // transaksi pertama; untuk invoice: transaksi DPP-nya
	InvoiceID  int    `json:"invoice_id,omitempty"`
	Tanggal    string `json:"tanggal"`
	Jenis      string `json:"jenis"`
	Keterangan string `json:"keterangan,omitempty"`
	Pihak      string `json:"pihak,omitempty"`
	Nominal    int64  `json:"nominal"`
	Sumber     string `json:"sumber,omitempty"`
	// BelumLunas: transaksi masih piutang/utang — mutasi ini bisa dicatat sebagai pelunasannya.
	BelumLunas bool `json:"belum_lunas"`
	// Lawan: rekening kas/bank transaksi itu (Transaction.Lawan; pratinjau mutasi di server
	// mengisinya dengan akun bayar efektif). Grup: transaksi bagian catatan jurnal multi-akun.
	// Keduanya dipakai menawarkan "sama — pindah akun bayar" di impor mutasi.
	Lawan string `json:"lawan,omitempty"`
	Grup  int    `json:"grup,omitempty"`
}

func selisihHari(a, b string) int {
	ta, e1 := time.Parse(time.DateOnly, a)
	tb, e2 := time.Parse(time.DateOnly, b)
	if e1 != nil || e2 != nil {
		return 1 << 30
	}
	d := int(ta.Sub(tb).Hours() / 24)
	if d < 0 {
		d = -d
	}
	return d
}

// CariMirip: transaksi yang kemungkinan sama dengan uang sebesar nominal ke arah arah pada
// tanggal. kecuali: ID transaksi yang diabaikan (mis. yang sedang diubah).
func CariMirip(txs []Transaction, tanggal string, nominal int64, arah string, kecuali map[int]bool) []Mirip {
	var out []Mirip
	invoice := map[int]*Mirip{}
	for _, t := range txs {
		j, ok := jenisOf(t.Jenis)
		// Pindah Dana keluar dari rekening asal dan masuk ke rekening tujuan: cocok di kedua arah.
		if !ok || (j.Arah != arah && t.Jenis != JenisPindahDana) || kecuali[t.ID] {
			continue
		}
		if t.InvoiceID != 0 && !t.Lunas {
			// Invoice belum lunas: transfernya sebesar DPP + PPN, jadi dijumlah per invoice.
			m := invoice[t.InvoiceID]
			if m == nil {
				m = &Mirip{ID: t.ID, InvoiceID: t.InvoiceID, Tanggal: t.Tanggal, Jenis: t.Jenis, Keterangan: t.Keterangan, Pihak: t.Pihak, Sumber: t.Sumber, BelumLunas: true}
				invoice[t.InvoiceID] = m
			}
			m.Nominal += t.Nominal
			continue
		}
		if t.Nominal != nominal {
			continue
		}
		m := Mirip{ID: t.ID, InvoiceID: t.InvoiceID, Tanggal: t.Tanggal, Jenis: t.Jenis, Keterangan: t.Keterangan, Pihak: t.Pihak, Nominal: t.Nominal, Sumber: t.Sumber, BelumLunas: !t.Lunas, Lawan: t.Lawan, Grup: t.Grup}
		switch {
		case t.Lunas && selisihHari(tanggalKas(t), tanggal) <= ToleransiHariMirip:
			out = append(out, m)
		case !t.Lunas && t.Tanggal <= tanggal && selisihHari(t.Tanggal, tanggal) <= maksHariTagihan:
			out = append(out, m)
		}
	}
	for _, m := range invoice {
		if m.Nominal == nominal && m.Tanggal <= tanggal && selisihHari(m.Tanggal, tanggal) <= maksHariTagihan {
			out = append(out, *m)
		}
	}
	// Yang paling dekat tanggalnya dulu.
	slices.SortStableFunc(out, func(a, b Mirip) int { return selisihHari(a.Tanggal, tanggal) - selisihHari(b.Tanggal, tanggal) })
	return out
}

// PasanganMirip: pasangan transaksi lunas di periode yang kemungkinan dobel (arah & nominal
// sama, tanggal uang berdekatan), selain transaksi satu catatan / invoice yang sama.
func PasanganMirip(txs []Transaction, period string) [][2]Transaction {
	var calon []Transaction
	for _, t := range txs {
		if t.Lunas && len(tanggalKas(t)) >= 7 && tanggalKas(t)[:7] == period {
			calon = append(calon, t)
		}
	}
	slices.SortStableFunc(calon, func(a, b Transaction) int { return int(a.Nominal - b.Nominal) })
	var out [][2]Transaction
	for i := range calon {
		for k := i + 1; k < len(calon) && calon[k].Nominal == calon[i].Nominal; k++ {
			a, b := calon[i], calon[k]
			ja, _ := jenisOf(a.Jenis)
			jb, _ := jenisOf(b.Jenis)
			sama := (a.Grup != 0 && a.Grup == b.Grup) || (a.InvoiceID != 0 && a.InvoiceID == b.InvoiceID)
			if ja.Arah == jb.Arah && !sama && selisihHari(tanggalKas(a), tanggalKas(b)) <= ToleransiHariMirip {
				out = append(out, [2]Transaction{a, b})
			}
		}
	}
	return out
}
