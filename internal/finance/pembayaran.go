package finance

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"
)

// Pembayaran sebagian (Accurate-like M1). Satu dokumen Pembayaran menerima uang dari pelanggan
// (piutang) atau membayar pemasok (utang) dan dialokasikan ke satu atau beberapa transaksi
// belum lunas. Perhitungan (kas, piutang/utang, umur, jurnal) hanya membaca Transaction.Cicilan;
// Alokasi di dokumen adalah catatan isi dokumennya. Status lunas transaksi bercicilan diturunkan
// dari cicilannya. Batal = cicilan dicabut, dokumen tetap ada bertanda batal beserta alasannya.

const (
	BayarPiutang = "piutang"
	BayarUtang   = "utang"
)

// Cicilan: bagian pembayaran yang dialokasikan ke satu transaksi.
type Cicilan struct {
	PembayaranID int    `json:"pembayaran_id"`
	Tanggal      string `json:"tanggal"`
	Nominal      int64  `json:"nominal"`
	Lawan        string `json:"lawan,omitempty"` // akun kas/bank; kosong = otomatis
}

type Alokasi struct {
	TransaksiID int   `json:"transaksi_id"`
	Nominal     int64 `json:"nominal"`
}

type BatalPembayaran struct {
	Alasan string    `json:"alasan"`
	Oleh   string    `json:"oleh,omitempty"`
	Pada   time.Time `json:"pada"`
}

type Pembayaran struct {
	ID         int       `json:"id"`
	Tanggal    string    `json:"tanggal"`
	Jenis      string    `json:"jenis"` // piutang | utang
	Pihak      string    `json:"pihak,omitempty"`
	Lawan      string    `json:"lawan,omitempty"`
	Keterangan string    `json:"keterangan,omitempty"`
	Alokasi    []Alokasi `json:"alokasi"`
	Total      int64     `json:"total"`
	// Kunci: Idempotency-Key permintaan yang membuat dokumen ini (unik per usaha).
	Kunci  string           `json:"kunci,omitempty"`
	Oleh   string           `json:"oleh,omitempty"`
	Dibuat time.Time        `json:"dibuat,omitzero"`
	Batal  *BatalPembayaran `json:"batal,omitempty"`
}

// posBolehDicicil: piutang/utang yang boleh dibayar lewat dokumen Pembayaran — usaha (omzet,
// pembelian, biaya, PPN) dan aset yang dibeli tempo. Modal, pinjaman, cicilan pinjaman, dan
// prive punya alurnya sendiri.
func posBolehDicicil(pos string) bool { return posOperasi[pos] || pos == "aset" }

// Dibayar: jumlah yang sudah dibayar sampai akhir tanggal sampai.
func (t Transaction) Dibayar(sampai string) int64 {
	if len(t.Cicilan) == 0 {
		if lunasPada(t, sampai) {
			return t.Nominal
		}
		return 0
	}
	var n int64
	for _, c := range t.Cicilan {
		if c.Tanggal <= sampai {
			n += c.Nominal
		}
	}
	return n
}

// Sisa: yang belum dibayar sampai hari ini (semua cicilan dihitung).
func (t Transaction) Sisa() int64 {
	if len(t.Cicilan) == 0 {
		if t.Lunas {
			return 0
		}
		return t.Nominal
	}
	n := t.Nominal
	for _, c := range t.Cicilan {
		n -= c.Nominal
	}
	return n
}

// GerakKas: uang yang benar-benar bergerak untuk transaksi ini — tiap cicilan, atau satu gerak
// sebesar nominal di tanggal lunas untuk transaksi lunas tanpa cicilan.
func (t Transaction) GerakKas() []Cicilan {
	if len(t.Cicilan) > 0 {
		return t.Cicilan
	}
	if t.Lunas {
		return []Cicilan{{Tanggal: tanggalKas(t), Nominal: t.Nominal, Lawan: t.Lawan}}
	}
	return nil
}

// TurunkanLunas: status lunas transaksi bercicilan dari cicilannya. Lunas sebagian = belum lunas
// (TanggalLunas kosong, sesuai aturan Validate).
func TurunkanLunas(t *Transaction) {
	t.Lunas, t.TanggalLunas = false, ""
	if len(t.Cicilan) == 0 || t.Sisa() != 0 {
		return
	}
	t.Lunas = true
	terakhir := slices.MaxFunc(t.Cicilan, func(a, b Cicilan) int { return strings.Compare(a.Tanggal, b.Tanggal) }).Tanggal
	if terakhir != t.Tanggal {
		t.TanggalLunas = terakhir
	}
}

// ValidatePembayaran merapikan dokumen dan menghitung totalnya dari alokasi.
func ValidatePembayaran(p *Pembayaran) error {
	p.Tanggal, p.Keterangan, p.Lawan = strings.TrimSpace(p.Tanggal), strings.TrimSpace(p.Keterangan), strings.TrimSpace(p.Lawan)
	if _, err := time.Parse(time.DateOnly, p.Tanggal); err != nil {
		return errors.New("tanggal bayar wajib diisi (YYYY-MM-DD)")
	}
	if p.Jenis != BayarPiutang && p.Jenis != BayarUtang {
		return errors.New("jenis pembayaran harus piutang atau utang")
	}
	if len(p.Alokasi) == 0 {
		return errors.New("pilih minimal satu tagihan yang dibayar")
	}
	p.Total = 0
	for _, a := range p.Alokasi {
		if a.Nominal <= 0 {
			return errors.New("nominal tiap tagihan harus lebih dari 0")
		}
		p.Total += a.Nominal
	}
	return nil
}

// Bayar menerapkan pembayaran p ke txs dan mengembalikan salinan baru (txs tidak diubah).
// Ditolak bila alokasi melebihi sisa, arahnya salah, tanggal bayar sebelum transaksi, atau
// transaksinya bukan piutang/utang usaha.
func Bayar(txs []Transaction, p Pembayaran) ([]Transaction, error) {
	if err := ValidatePembayaran(&p); err != nil {
		return nil, err
	}
	arah := "masuk"
	if p.Jenis == BayarUtang {
		arah = "keluar"
	}
	out := slices.Clone(txs)
	dipakai := map[int]bool{}
	for _, a := range p.Alokasi {
		i := slices.IndexFunc(out, func(t Transaction) bool { return t.ID == a.TransaksiID })
		if i < 0 {
			return nil, fmt.Errorf("transaksi #%d tidak ditemukan", a.TransaksiID)
		}
		t := out[i]
		j, _ := jenisOf(t.Jenis)
		switch {
		case dipakai[t.ID]:
			return nil, fmt.Errorf("transaksi #%d dipilih dua kali", t.ID)
		case j.Arah != arah:
			return nil, fmt.Errorf("transaksi #%d bukan %s", t.ID, p.Jenis)
		case !posBolehDicicil(j.Pos):
			return nil, fmt.Errorf("%s #%d tidak dibayar lewat Pembayaran (pakai menu transaksinya)", t.Jenis, t.ID)
		case p.Tanggal < t.Tanggal:
			return nil, fmt.Errorf("tanggal bayar sebelum tanggal transaksi #%d (%s)", t.ID, t.Tanggal)
		case t.Sisa() == 0:
			return nil, fmt.Errorf("transaksi #%d sudah lunas", t.ID)
		case a.Nominal > t.Sisa():
			return nil, fmt.Errorf("pembayaran %s melebihi sisa transaksi #%d (%s)", FormatRupiahPenuh(a.Nominal), t.ID, FormatRupiahPenuh(t.Sisa()))
		}
		dipakai[t.ID] = true
		t.Cicilan = append(slices.Clone(t.Cicilan), Cicilan{PembayaranID: p.ID, Tanggal: p.Tanggal, Nominal: a.Nominal, Lawan: p.Lawan})
		TurunkanLunas(&t)
		out[i] = t
	}
	return out, nil
}

// BatalBayar mencabut cicilan pembayaran id dari semua transaksi (salinan baru).
func BatalBayar(txs []Transaction, id int) []Transaction {
	out := slices.Clone(txs)
	for i, t := range out {
		if !slices.ContainsFunc(t.Cicilan, func(c Cicilan) bool { return c.PembayaranID == id }) {
			continue
		}
		t.Cicilan = slices.DeleteFunc(slices.Clone(t.Cicilan), func(c Cicilan) bool { return c.PembayaranID == id })
		if len(t.Cicilan) == 0 {
			t.Cicilan = nil
		}
		TurunkanLunas(&t)
		out[i] = t
	}
	return out
}

// samaTx: dua transaksi identik, termasuk cicilannya (Transaction tidak bisa dibandingkan dengan
// == karena punya field slice).
func samaTx(a, b Transaction) bool {
	if len(a.Cicilan) == 0 {
		a.Cicilan = nil
	}
	if len(b.Cicilan) == 0 {
		b.Cicilan = nil
	}
	return reflect.DeepEqual(a, b)
}

// ErrBatalkanDulu: transaksi yang sudah dibayar lewat Pembayaran tidak diubah/dihapus langsung.
var ErrBatalkanDulu = errors.New("transaksi ini sudah punya pembayaran; batalkan pembayarannya dulu di Piutang & Utang, baru ubah atau hapus")
