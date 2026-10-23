package finance

import (
	"fmt"
	"strconv"
	"strings"
)

// No. Referensi transaksi: TRX-yymm-0001, urut per bulan tanggal transaksi (transaksi backdate
// mendapat nomor berikutnya di bulannya, nomor yang ada tidak berubah). Nomor yang dibatalkan
// tidak dipakai ulang.

// Pembatalan: jejak transaksi yang dibatalkan. Asal: dari mana pembatalan terjadi —
// "transaksi" / "catatan" (dibatalkan pengguna, bisa dipulihkan Owner), atau dokumen lain
// ("invoice", "aset", "produksi") yang membawa transaksinya.
type Pembatalan struct {
	Oleh   string `json:"oleh"`
	Waktu  string `json:"waktu"`
	Alasan string `json:"alasan"`
	Asal   string `json:"asal"`
}

// BisaDipulihkan: hanya pembatalan langsung oleh pengguna; transaksi milik dokumen lain kembali
// lewat dokumennya.
func (p *Pembatalan) BisaDipulihkan() bool {
	return p != nil && (p.Asal == "transaksi" || p.Asal == "catatan")
}

// awalanRef: "2026-09-28" → "TRX-2609-".
func awalanRef(tanggal string) string {
	if len(tanggal) < 7 {
		return "TRX-0000-"
	}
	return "TRX-" + tanggal[2:4] + tanggal[5:7] + "-"
}

// RefBerikut: nomor berikutnya di bulan tanggal itu, dari semua nomor yang pernah dipakai
// (aktif maupun batal).
func RefBerikut(dipakai []string, tanggal string) string {
	aw := awalanRef(tanggal)
	maks := 0
	for _, r := range dipakai {
		if n, err := strconv.Atoi(strings.TrimPrefix(r, aw)); err == nil && strings.HasPrefix(r, aw) {
			maks = max(maks, n)
		}
	}
	return fmt.Sprintf("%s%04d", aw, maks+1)
}

// kunciRef: transaksi yang berbagi satu nomor — satu catatan (Grup) atau satu invoice.
func kunciRef(t Transaction) string {
	switch {
	case t.Grup != 0:
		return "g" + strconv.Itoa(t.Grup)
	case t.InvoiceID != 0:
		return "i" + strconv.Itoa(t.InvoiceID)
	}
	return "t" + strconv.Itoa(t.ID)
}

// BeriRef memberi nomor pada transaksi aktif tanpa nomor yang ID-nya >= mulai, urut ID.
// Anggota catatan / invoice yang sudah bernomor membagikan nomornya. batal ikut dihitung supaya
// nomor yang dibatalkan tidak terpakai lagi. Mengubah txs di tempat.
func BeriRef(txs []Transaction, batal []Transaction, mulai int) {
	dipakai := make([]string, 0, len(txs)+len(batal))
	perKunci := map[string]string{}
	for _, list := range [][]Transaction{txs, batal} {
		for _, t := range list {
			if t.Ref != "" {
				dipakai = append(dipakai, t.Ref)
				perKunci[kunciRef(t)] = t.Ref
			}
		}
	}
	for i := range txs {
		t := &txs[i]
		if t.Ref != "" || t.ID < mulai || mulai <= 0 {
			continue
		}
		k := kunciRef(*t)
		if r, ok := perKunci[k]; ok && t.Grup+t.InvoiceID != 0 {
			t.Ref = r
			continue
		}
		t.Ref = RefBerikut(dipakai, t.Tanggal)
		dipakai = append(dipakai, t.Ref)
		perKunci[k] = t.Ref
	}
}
