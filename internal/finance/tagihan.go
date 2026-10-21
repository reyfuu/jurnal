package finance

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Tagihan & StokItem adalah data usaha yang bisa diimpor dan disimpan, berbeda dari
// ObligationItem dan InventoryItem yang merupakan bentuk tampilan di dashboard.
//
// Dipisah karena keduanya menyimpan hal yang berbeda: di sini tanggal jatuh tempo dan jumlah
// stok disimpan sebagai data sungguhan (YYYY-MM-DD, angka), sementara bentuk tampilan memuat
// hasil turunan seperti "lewat 52 hari" dan "Stok Menipis". Turunan dihitung saat dirender,
// supaya tidak basi ketika tanggal berganti.

const (
	TagihanPiutang = "RECEIVABLE" // pelanggan berutang ke kita
	TagihanUtang   = "PAYABLE"    // kita berutang ke supplier
)

type Tagihan struct {
	ID         int    `json:"id"`
	Nomor      string `json:"nomor"` // no. invoice (piutang) atau no. PO (utang)
	Pihak      string `json:"pihak"`
	Jenis      string `json:"jenis"`       // RECEIVABLE | PAYABLE
	JatuhTempo string `json:"jatuh_tempo"` // YYYY-MM-DD
	Nominal    int64  `json:"nominal"`
	Lunas      bool   `json:"lunas"`
	Keterangan string `json:"keterangan,omitempty"`
}

func ValidateTagihan(t *Tagihan) error {
	for _, s := range []*string{&t.Nomor, &t.Pihak, &t.Jenis, &t.JatuhTempo, &t.Keterangan} {
		*s = strings.TrimSpace(*s)
	}
	t.Jenis = strings.ToUpper(t.Jenis)
	switch t.Jenis {
	case "", "PIUTANG":
		t.Jenis = TagihanPiutang
	case "UTANG", "HUTANG":
		t.Jenis = TagihanUtang
	case TagihanPiutang, TagihanUtang:
	default:
		return errors.New("jenis tagihan harus Piutang atau Utang")
	}
	if t.Pihak == "" {
		return errors.New("nama pelanggan/supplier wajib diisi")
	}
	if _, err := time.Parse(time.DateOnly, t.JatuhTempo); err != nil {
		return errors.New("jatuh tempo wajib diisi (format YYYY-MM-DD atau DD/MM/YYYY)")
	}
	if t.Nominal <= 0 {
		return errors.New("nominal harus lebih dari 0")
	}
	return nil
}

// HariTerlambat: 0 bila lunas atau belum jatuh tempo.
func (t Tagihan) HariTerlambat(acuan time.Time) int {
	if t.Lunas {
		return 0
	}
	jt, err := time.Parse(time.DateOnly, t.JatuhTempo)
	if err != nil || !acuan.After(jt) {
		return 0
	}
	return int(acuan.Sub(jt).Hours() / 24)
}

// ToObligations merender tagihan jadi bentuk yang dipakai dashboard. Tagihan yang sudah lunas
// tidak ikut: daftar Piutang & Utang hanya berguna untuk yang belum selesai.
func ToObligations(list []Tagihan, acuan time.Time) []ObligationItem {
	out := make([]ObligationItem, 0, len(list))
	for _, t := range list {
		if t.Lunas {
			continue
		}
		jt, _ := time.Parse(time.DateOnly, t.JatuhTempo)
		out = append(out, ObligationItem{
			InvoiceNo:    t.Nomor,
			Counterparty: t.Pihak,
			Kind:         t.Jenis,
			DueDate:      tanggalPendek(jt),
			DaysOverdue:  t.HariTerlambat(acuan),
			Amount:       t.Nominal,
		})
	}
	return out
}

// StokItem: persediaan yang bisa diimpor. Jumlah disimpan sebagai angka beserta satuannya,
// bukan teks seperti "42,5 kg", supaya bisa dibandingkan dengan batas minimum.
type StokItem struct {
	ID         int    `json:"id"`
	SKU        string `json:"sku"`
	Nama       string `json:"nama"`
	Kategori   string `json:"kategori"` // mis. Bahan Baku, Produk Jadi
	Jumlah     int64  `json:"jumlah"`   // dalam satuan terkecil; lihat Satuan
	Satuan     string `json:"satuan"`
	StokMin    int64  `json:"stok_min"` // batas peringatan; 0 = tanpa peringatan
	HargaPokok int64  `json:"harga_pokok"`
	HargaJual  int64  `json:"harga_jual"`
}

func ValidateStok(s *StokItem) error {
	for _, p := range []*string{&s.SKU, &s.Nama, &s.Kategori, &s.Satuan} {
		*p = strings.TrimSpace(*p)
	}
	if s.Nama == "" {
		return errors.New("nama barang wajib diisi")
	}
	if s.Jumlah < 0 || s.StokMin < 0 {
		return errors.New("jumlah dan stok minimum tidak boleh negatif")
	}
	if s.HargaPokok < 0 || s.HargaJual < 0 {
		return errors.New("harga tidak boleh negatif")
	}
	if s.Kategori == "" {
		s.Kategori = "Barang"
	}
	return nil
}

// Status: dihitung dari jumlah terhadap stok minimum, bukan disimpan, supaya tidak basi.
func (s StokItem) Status() string {
	switch {
	case s.Jumlah == 0:
		return "Habis"
	case s.StokMin > 0 && s.Jumlah <= s.StokMin:
		return "Stok Menipis"
	}
	return "Aman"
}

// MarginBps: margin kotor per unit. 0 bila harga jual atau pokok belum diisi.
func (s StokItem) MarginBps() int {
	if s.HargaJual <= 0 || s.HargaPokok <= 0 {
		return 0
	}
	return int((s.HargaJual - s.HargaPokok) * 10_000 / s.HargaJual)
}

func ToInventory(list []StokItem) []InventoryItem {
	out := make([]InventoryItem, 0, len(list))
	for _, s := range list {
		jumlah := fmt.Sprintf("%d", s.Jumlah)
		if s.Satuan != "" {
			jumlah += " " + s.Satuan
		}
		status := s.Status()
		if status == "Aman" && s.MarginBps() > 0 {
			status = fmt.Sprintf("Margin %d%%", s.MarginBps()/100)
		}
		out = append(out, InventoryItem{
			SKU: s.SKU, Name: s.Nama, Type: s.Kategori,
			Stock: jumlah, COGS: s.HargaPokok, SellPrice: s.HargaJual, Status: status,
		})
	}
	return out
}

// StokSummary: rekap yang dipakai halaman Stok & Gudang.
type StokSummary struct {
	Items           []StokRow `json:"items"`
	TotalItem       int       `json:"total_item"`
	NilaiPersediaan int64     `json:"nilai_persediaan"` // jumlah x harga pokok
	Menipis         int       `json:"menipis"`
	Habis           int       `json:"habis"`
	// Mulai: titik nol kartu stok (kosong = jumlah masih diketik manual). Minus: barang yang
	// stok sistemnya di bawah nol — pembelian/produksi belum dicatat atau salah jumlah.
	Mulai string `json:"mulai"`
	Minus int    `json:"minus"`
}

// StokRow: satu baris stok beserta turunannya. Turunan dikirim dari sini, bukan dihitung ulang
// di tampilan, supaya ambang "stok menipis" dan rumus margin hanya ada di satu tempat.
type StokRow struct {
	StokItem
	Status    string `json:"status"`
	MarginBps int    `json:"margin_bps"`
	Nilai     int64  `json:"nilai"` // jumlah x harga pokok
	// JumlahMili: jumlah per seribu satuan (2500 = 2,5). Dilacak: jumlah & nilainya dari
	// kartu stok, bukan diketik.
	JumlahMili int64 `json:"jumlah_mili"`
	Dilacak    bool  `json:"dilacak"`
	Minus      bool  `json:"minus"`
}

func SummarizeStok(list []StokItem) StokSummary {
	s := StokSummary{Items: make([]StokRow, 0, len(list)), TotalItem: len(list)}
	for _, it := range list {
		nilai := it.Jumlah * it.HargaPokok
		status := it.Status()
		s.Items = append(s.Items, StokRow{StokItem: it, Status: status, MarginBps: it.MarginBps(), Nilai: nilai, JumlahMili: it.Jumlah * TakaranSkala})
		s.NilaiPersediaan += nilai
		switch status {
		case "Stok Menipis":
			s.Menipis++
		case "Habis":
			s.Habis++
		}
	}
	return s
}

// TagihanSummary: rekap yang dipakai halaman Piutang & Utang.
type TagihanSummary struct {
	Items            []TagihanRow `json:"items"`
	TotalPiutang     int64        `json:"total_piutang"`
	TotalUtang       int64        `json:"total_utang"`
	PiutangTerlambat int64        `json:"piutang_terlambat"`
	JumlahTerlambat  int          `json:"jumlah_terlambat"`
}

type TagihanRow struct {
	Tagihan
	HariTerlambat int `json:"hari_terlambat"`
}

func SummarizeTagihan(list []Tagihan, acuan time.Time) TagihanSummary {
	s := TagihanSummary{Items: make([]TagihanRow, 0, len(list))}
	for _, t := range list {
		telat := t.HariTerlambat(acuan)
		s.Items = append(s.Items, TagihanRow{Tagihan: t, HariTerlambat: telat})
		if t.Lunas {
			continue
		}
		if t.Jenis == TagihanPiutang {
			s.TotalPiutang += t.Nominal
			if telat > 0 {
				s.PiutangTerlambat += t.Nominal
				s.JumlahTerlambat++
			}
		} else {
			s.TotalUtang += t.Nominal
		}
	}
	return s
}

var bulanPendek = [...]string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}

// tanggalPendek: "2026-07-10" → "10 Jul 2026", format yang sudah dipakai dashboard.
func tanggalPendek(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return fmt.Sprintf("%02d %s %d", t.Day(), bulanPendek[t.Month()-1], t.Year())
}

// ToPiutangCard merender rekap tagihan jadi KPI "Piutang belum tertagih" di dashboard.
//
// Dipakai supaya KPI dan daftar Piutang & Utang membaca sumber yang sama. Sebelumnya kartu ini
// diisi dari angka seed bulanan dengan jumlah faktur dipatok 8, sehingga begitu daftarnya jadi
// data sungguhan, dashboard menyebut "8 tagihan" sementara halamannya menampilkan 5.
//
// Hanya tagihan yang belum lunas yang dihitung: kartu ini tentang uang yang masih di luar.
func (s TagihanSummary) ToPiutangCard() PiutangCard {
	var jumlah int
	for _, t := range s.Items {
		if !t.Lunas {
			jumlah++
		}
	}
	return PiutangCard{
		UnpaidTotal:          s.TotalPiutang,
		UnpaidDisplay:        FormatRupiahPenuh(s.TotalPiutang),
		InvoiceCount:         jumlah,
		Overdue30dTotal:      s.PiutangTerlambat,
		Overdue30dDisplay:    fmt.Sprintf("%s lewat jatuh tempo", FormatRupiahPenuh(s.PiutangTerlambat)),
		PayableUnpaidTotal:   s.TotalUtang,
		PayableUnpaidDisplay: fmt.Sprintf("Utang supplier %s", FormatRupiahPenuh(s.TotalUtang)),
	}
}

// TerapkanKartu: salinan daftar stok dengan jumlah (dibulatkan ke bawah) dan harga pokok
// rata-rata dari kartu stok. Tanpa titik nol, daftar dikembalikan apa adanya.
func TerapkanKartu(list []StokItem, k KartuStok) []StokItem {
	if k.Mulai == "" {
		return list
	}
	out := slices.Clone(list)
	for i := range out {
		p := k.Posisi[out[i].ID]
		out[i].Jumlah = p.Qty / TakaranSkala
		if p.Qty > 0 {
			out[i].HargaPokok = mulDiv(p.Nilai, TakaranSkala, p.Qty)
		}
	}
	return out
}

// RingkasStok: ringkasan Stok & Gudang; setelah titik nol jumlah, nilai, dan status
// mengikuti kartu stok.
func RingkasStok(list []StokItem, k KartuStok) StokSummary {
	if k.Mulai == "" {
		return SummarizeStok(list)
	}
	s := SummarizeStok(TerapkanKartu(list, k))
	s.Mulai, s.NilaiPersediaan, s.Menipis, s.Habis = k.Mulai, 0, 0, 0
	for i := range s.Items {
		r := &s.Items[i]
		p := k.Posisi[r.ID]
		r.JumlahMili, r.Nilai, r.Dilacak, r.Minus = p.Qty, p.Nilai, true, p.Qty < 0
		switch {
		case p.Qty < 0:
			r.Status = "Minus"
			s.Minus++
		case p.Qty == 0:
			r.Status = "Habis"
			s.Habis++
		case r.StokMin > 0 && p.Qty <= r.StokMin*TakaranSkala:
			r.Status = "Stok Menipis"
			s.Menipis++
		default:
			r.Status = "Aman"
		}
		s.NilaiPersediaan += p.Nilai
	}
	return s
}
