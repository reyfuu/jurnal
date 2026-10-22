package finance

import (
	"cmp"
	"slices"
	"strconv"
	"time"
)

// Kartu stok: riwayat pergerakan tiap barang dan nilainya dengan metode FIFO atau rata-rata
// bergerak (pilihan usaha).
//
// Sumbernya dua: transaksi yang memilih barang (Beli Bahan / Stok menambah, Penjualan
// mengurangi) dan dokumen stok (stok awal = titik nol, stok opname, produksi). Kartu tidak
// disimpan — selalu dihitung ulang dari sumbernya, jadi mengubah atau menghapus transaksi
// otomatis memperbaiki stok tanpa pencatatan ganda. Pergerakan sebelum titik nol diabaikan.
//
// Jumlah disimpan per seribu satuan (sama dengan TakaranSkala) supaya 2,5 kg atau 0,15 liter
// tetap tepat tanpa pecahan desimal.

const (
	GerakAwal     = "awal"     // titik nol: hasil hitung fisik pertama + harga pokoknya
	GerakOpname   = "opname"   // hitung fisik; selisihnya jadi penyesuaian
	GerakProduksi = "produksi" // bahan keluar (Qty < 0), barang jadi masuk (Qty > 0)

	KartuAwal          = "awal"
	KartuBeli          = "beli"
	KartuProduksiMasuk = "produksi_masuk"
	KartuProduksiPakai = "produksi_pakai"
	KartuJual          = "jual"
	KartuOpname        = "opname"

	JenisBeliStok = "Beli Bahan / Stok"
	JenisJual     = "Penjualan"
)

type BarisGerak struct {
	StokID    int   `json:"stok_id"`
	Qty       int64 `json:"qty,omitempty"`        // produksi: per seribu, + masuk / − keluar
	Fisik     int64 `json:"fisik,omitempty"`      // awal & opname: hasil hitung fisik (per seribu)
	HargaUnit int64 `json:"harga_unit,omitempty"` // awal: harga pokok per satuan
	// Standar: produksi — jumlah menurut resep × batch (per seribu, tanpa tanda), untuk
	// melihat selisih pemakaian/hasil aktual terhadap resep. 0 = di luar resep.
	Standar int64 `json:"standar,omitempty"`
}

type GerakStok struct {
	ID         int          `json:"id"`
	Tanggal    string       `json:"tanggal"`
	Jenis      string       `json:"jenis"`
	Baris      []BarisGerak `json:"baris"`
	ProdukID   int          `json:"produk_id,omitempty"`
	Batch      int64        `json:"batch,omitempty"`
	Keterangan string       `json:"keterangan,omitempty"`
	// Biaya: biaya lain produksi (tenaga harian, gas, ongkir) yang ikut menjadi nilai produk jadi;
	// Grup: transaksi pembayaran biaya itu (Transaction.Grup), dihapus bersama dokumen ini.
	Biaya  int64     `json:"biaya,omitempty"`
	Grup   int       `json:"grup,omitempty"`
	Oleh   string    `json:"oleh,omitempty"`
	Dibuat time.Time `json:"dibuat"`
}

type BarisKartu struct {
	Tanggal     string `json:"tanggal"`
	StokID      int    `json:"stok_id"`
	Jenis       string `json:"jenis"`
	TransaksiID int    `json:"transaksi_id,omitempty"`
	GerakID     int    `json:"gerak_id,omitempty"`
	Channel     string `json:"channel,omitempty"`
	Keterangan  string `json:"keterangan,omitempty"`
	Qty         int64  `json:"qty"`   // per seribu, bertanda
	Nilai       int64  `json:"nilai"` // rupiah, bertanda
	// Koreksi: saat stok minus ditutup pembelian, sisa stok dinilai ulang dengan harga masuk;
	// selisihnya koreksi HPP barang yang sebelumnya keluar dengan harga perkiraan.
	Koreksi int64 `json:"koreksi,omitempty"`
	// Biaya: bagian biaya lain produksi di nilai barang jadi yang masuk (produksi_masuk).
	Biaya      int64 `json:"biaya,omitempty"`
	SaldoQty   int64 `json:"saldo_qty"`
	SaldoNilai int64 `json:"saldo_nilai"`
	Minus      bool  `json:"minus,omitempty"`
}

type PosisiStok struct {
	Qty   int64 `json:"qty"`
	Nilai int64 `json:"nilai"`
}

type KartuStok struct {
	Mulai  string             `json:"mulai"`  // tanggal titik nol; kosong = kartu stok belum dimulai
	Metode string             `json:"metode"` // fifo | rata_rata
	Baris  []BarisKartu       `json:"baris"`
	Posisi map[int]PosisiStok `json:"-"`
}

type kejadian struct {
	tanggal string
	urut    int // urutan dalam satu hari
	id      int
	jalan   func()
}

// Metode penilaian persediaan (Pasal 10 ayat 6 UU PPh): FIFO atau rata-rata. Dipilih sekali
// per usaha (BusinessProfile.MetodeNilaiStok); pergantian butuh persetujuan DJP.
const (
	MetodeFIFO     = "fifo"
	MetodeRataRata = "rata_rata"
)

// penilai menyimpan saldo tiap barang dan menilai barang masuk/keluar menurut metodenya.
type penilai interface {
	// awal menimpa saldo barang dengan hasil hitung fisik pertama.
	awal(id int, qty, nilai int64)
	// masuk menambah stok; mengembalikan koreksi nilai bila stok sebelumnya minus.
	masuk(id int, q, biaya int64) int64
	// keluar mengurangi stok; mengembalikan nilai yang keluar (positif).
	keluar(id int, q int64) int64
	// hargaTambah: nilai q unit lebih (opname) — harga pokok terakhir yang diketahui.
	hargaTambah(id int, q int64) int64
	posisi(id int) (qty, nilai int64)
	ids() []int
}

// NamaMetodeStok: label metode untuk laporan & ekspor.
func NamaMetodeStok(m string) string {
	if m == MetodeFIFO {
		return "FIFO"
	}
	return "rata-rata bergerak"
}

func penilaiUntuk(metode string, harga map[int]int64) penilai {
	if metode == MetodeFIFO {
		return &penilaiFIFO{harga: harga, st: map[int]*itemFIFO{}}
	}
	return &penilaiRata{harga: harga, st: map[int]*nilaiItem{}}
}

// --- Rata-rata bergerak ---

type nilaiItem struct {
	qty, nilai int64
	// rata-rata terakhir saat stok masih positif, untuk menilai barang keluar saat stok habis
	akhirQty, akhirNilai int64
}

type penilaiRata struct {
	harga map[int]int64
	st    map[int]*nilaiItem
}

func (p *penilaiRata) item(id int) *nilaiItem {
	if p.st[id] == nil {
		p.st[id] = &nilaiItem{}
	}
	return p.st[id]
}

func (p *penilaiRata) ingat(v *nilaiItem) {
	if v.qty > 0 {
		v.akhirQty, v.akhirNilai = v.qty, v.nilai
	}
}

func (p *penilaiRata) awal(id int, qty, nilai int64) {
	v := p.item(id)
	v.qty, v.nilai = qty, nilai
	p.ingat(v)
}

func (p *penilaiRata) hargaTambah(id int, q int64) int64 {
	v := p.item(id)
	switch {
	case v.qty > 0:
		return mulDiv(v.nilai, q, v.qty)
	case v.akhirQty > 0:
		return mulDiv(v.akhirNilai, q, v.akhirQty)
	}
	return mulDiv(p.harga[id], q, TakaranSkala)
}

func (p *penilaiRata) keluar(id int, q int64) int64 {
	v := p.item(id)
	biaya := p.hargaTambah(id, q)
	v.qty -= q
	v.nilai -= biaya
	p.ingat(v)
	return biaya
}

func (p *penilaiRata) masuk(id int, q, biaya int64) int64 {
	v := p.item(id)
	sebelum := v.qty
	v.qty += q
	v.nilai += biaya
	var koreksi int64
	if sebelum < 0 && v.qty > 0 {
		benar := mulDiv(biaya, v.qty, q)
		koreksi = benar - v.nilai
		v.nilai = benar
	}
	p.ingat(v)
	return koreksi
}

func (p *penilaiRata) posisi(id int) (int64, int64) { v := p.item(id); return v.qty, v.nilai }

func (p *penilaiRata) ids() []int {
	out := make([]int, 0, len(p.st))
	for id := range p.st {
		out = append(out, id)
	}
	return out
}

// --- FIFO: barang yang masuk lebih dulu keluar lebih dulu ---

type lapisFIFO struct{ qty, nilai int64 }

type itemFIFO struct {
	lapis []lapisFIFO
	// kurangQty/kurangNilai: stok minus (keluar melebihi saldo) dan nilai yang sudah dibebankan
	// untuknya; dibereskan oleh barang masuk berikutnya (selisihnya jadi koreksi).
	kurangQty, kurangNilai int64
	// harga pokok per unit terakhir (qty, nilai) untuk menilai stok minus & lebih opname
	akhirQty, akhirNilai int64
}

type penilaiFIFO struct {
	harga map[int]int64
	st    map[int]*itemFIFO
}

func (p *penilaiFIFO) item(id int) *itemFIFO {
	if p.st[id] == nil {
		p.st[id] = &itemFIFO{}
	}
	return p.st[id]
}

func (p *penilaiFIFO) awal(id int, qty, nilai int64) {
	v := p.item(id)
	*v = itemFIFO{}
	if qty > 0 {
		v.lapis = []lapisFIFO{{qty, nilai}}
		v.akhirQty, v.akhirNilai = qty, nilai
	}
}

func (p *penilaiFIFO) hargaTambah(id int, q int64) int64 {
	if v := p.item(id); v.akhirQty > 0 {
		return mulDiv(v.akhirNilai, q, v.akhirQty)
	}
	return mulDiv(p.harga[id], q, TakaranSkala)
}

func (p *penilaiFIFO) keluar(id int, q int64) int64 {
	v := p.item(id)
	var biaya int64
	for q > 0 && len(v.lapis) > 0 {
		l := &v.lapis[0]
		if l.qty <= q {
			biaya += l.nilai
			q -= l.qty
			v.lapis = v.lapis[1:]
			continue
		}
		n := mulDiv(l.nilai, q, l.qty)
		l.qty -= q
		l.nilai -= n
		biaya += n
		q = 0
	}
	if q > 0 { // stok habis: sisanya dinilai harga pokok terakhir sampai barang masuk lagi
		n := p.hargaTambah(id, q)
		v.kurangQty += q
		v.kurangNilai += n
		biaya += n
	}
	return biaya
}

func (p *penilaiFIFO) masuk(id int, q, biaya int64) int64 {
	v := p.item(id)
	v.akhirQty, v.akhirNilai = q, biaya
	var koreksi int64
	if v.kurangQty > 0 {
		tutup := min(q, v.kurangQty)
		sebenarnya := mulDiv(biaya, tutup, q)
		dibebankan := mulDiv(v.kurangNilai, tutup, v.kurangQty)
		koreksi = dibebankan - sebenarnya
		v.kurangQty -= tutup
		v.kurangNilai -= dibebankan
		q -= tutup
		biaya -= sebenarnya
	}
	if q > 0 {
		v.lapis = append(v.lapis, lapisFIFO{q, biaya})
	}
	return koreksi
}

func (p *penilaiFIFO) posisi(id int) (qty, nilai int64) {
	v := p.item(id)
	for _, l := range v.lapis {
		qty += l.qty
		nilai += l.nilai
	}
	return qty - v.kurangQty, nilai - v.kurangNilai
}

func (p *penilaiFIFO) ids() []int {
	out := make([]int, 0, len(p.st))
	for id := range p.st {
		out = append(out, id)
	}
	return out
}

// HitungKartu menghitung kartu stok dengan rata-rata bergerak (lihat HitungKartuMetode).
func HitungKartu(stok []StokItem, txs []Transaction, gerak []GerakStok) KartuStok {
	return HitungKartuMetode(MetodeRataRata, stok, txs, gerak)
}

// HitungKartuMetode menghitung kartu stok dari stok awal (titik nol), transaksi bertanda barang,
// stok opname, dan produksi, dengan metode penilaian FIFO atau rata-rata bergerak.
func HitungKartuMetode(metode string, stok []StokItem, txs []Transaction, gerak []GerakStok) KartuStok {
	k := KartuStok{Posisi: map[int]PosisiStok{}, Metode: metode}
	if metode != MetodeFIFO {
		k.Metode = MetodeRataRata
	}
	awal := -1
	for i, g := range gerak {
		if g.Jenis == GerakAwal && (awal < 0 || g.Tanggal < gerak[awal].Tanggal) {
			awal = i
		}
	}
	if awal < 0 {
		return k
	}
	k.Mulai = gerak[awal].Tanggal

	harga := map[int]int64{}
	for _, s := range stok {
		harga[s.ID] = s.HargaPokok
	}
	nilai := penilaiUntuk(k.Metode, harga)
	catat := func(b BarisKartu) {
		b.SaldoQty, b.SaldoNilai = nilai.posisi(b.StokID)
		b.Minus = b.SaldoQty < 0
		k.Baris = append(k.Baris, b)
	}
	keluar, masuk, rata := nilai.keluar, nilai.masuk, nilai.hargaTambah

	var ev []kejadian
	for _, g := range gerak {
		if g.Tanggal < k.Mulai || (g.Jenis == GerakAwal && g.ID != gerak[awal].ID) {
			continue
		}
		g := g
		switch g.Jenis {
		case GerakAwal:
			ev = append(ev, kejadian{g.Tanggal, 0, g.ID, func() {
				for _, b := range g.Baris {
					n := mulDiv(b.Fisik, b.HargaUnit, TakaranSkala)
					nilai.awal(b.StokID, b.Fisik, n)
					catat(BarisKartu{Tanggal: g.Tanggal, StokID: b.StokID, Jenis: KartuAwal, GerakID: g.ID, Keterangan: "Stok awal (titik nol)", Qty: b.Fisik, Nilai: n})
				}
			}})
		case GerakProduksi:
			ev = append(ev, kejadian{g.Tanggal, 2, g.ID, func() {
				var pakai, hasil int64
				for _, b := range g.Baris {
					if b.Qty < 0 {
						biaya := keluar(b.StokID, -b.Qty)
						pakai += biaya
						catat(BarisKartu{Tanggal: g.Tanggal, StokID: b.StokID, Jenis: KartuProduksiPakai, GerakID: g.ID, Keterangan: g.Keterangan, Qty: b.Qty, Nilai: -biaya})
					} else if b.Qty > 0 {
						hasil += b.Qty
					}
				}
				// Nilai barang jadi = bahan terpakai + biaya lain produksi, dibagi per jumlah hasil.
				sisa, sisaLain := pakai+g.Biaya, g.Biaya
				for _, b := range g.Baris {
					if b.Qty <= 0 {
						continue
					}
					biaya, lain := mulDiv(pakai+g.Biaya, b.Qty, hasil), mulDiv(g.Biaya, b.Qty, hasil)
					hasil -= b.Qty
					if hasil == 0 {
						biaya, lain = sisa, sisaLain // baris terakhir menerima sisa pembulatan
					}
					sisa, sisaLain = sisa-biaya, sisaLain-lain
					kor := masuk(b.StokID, b.Qty, biaya)
					catat(BarisKartu{Tanggal: g.Tanggal, StokID: b.StokID, Jenis: KartuProduksiMasuk, GerakID: g.ID, Keterangan: g.Keterangan, Qty: b.Qty, Nilai: biaya, Koreksi: kor, Biaya: lain})
				}
			}})
		case GerakOpname:
			ev = append(ev, kejadian{g.Tanggal, 4, g.ID, func() {
				for _, b := range g.Baris {
					qty, _ := nilai.posisi(b.StokID)
					selisih := b.Fisik - qty
					var nilai, kor int64
					switch {
					case selisih < 0:
						nilai = -keluar(b.StokID, -selisih)
					case selisih > 0:
						nilai = rata(b.StokID, selisih)
						kor = masuk(b.StokID, selisih, nilai)
					}
					catat(BarisKartu{Tanggal: g.Tanggal, StokID: b.StokID, Jenis: KartuOpname, GerakID: g.ID, Keterangan: cmp.Or(g.Keterangan, "Stok opname"), Qty: selisih, Nilai: nilai, Koreksi: kor})
				}
			}})
		}
	}
	for _, t := range txs {
		if t.StokID == 0 || t.QtyStok <= 0 || t.Tanggal < k.Mulai {
			continue
		}
		t := t
		switch t.Jenis {
		case JenisBeliStok:
			ev = append(ev, kejadian{t.Tanggal, 1, t.ID, func() {
				kor := masuk(t.StokID, t.QtyStok, t.Nominal)
				catat(BarisKartu{Tanggal: t.Tanggal, StokID: t.StokID, Jenis: KartuBeli, TransaksiID: t.ID, Keterangan: cmp.Or(t.Keterangan, t.Pihak), Qty: t.QtyStok, Nilai: t.Nominal, Koreksi: kor})
			}})
		case JenisJual:
			ev = append(ev, kejadian{t.Tanggal, 3, t.ID, func() {
				biaya := keluar(t.StokID, t.QtyStok)
				catat(BarisKartu{Tanggal: t.Tanggal, StokID: t.StokID, Jenis: KartuJual, TransaksiID: t.ID, Channel: t.Channel, Keterangan: cmp.Or(t.Keterangan, t.Pihak), Qty: -t.QtyStok, Nilai: -biaya})
			}})
		}
	}
	slices.SortStableFunc(ev, func(a, b kejadian) int {
		return cmp.Or(cmp.Compare(a.tanggal, b.tanggal), cmp.Compare(a.urut, b.urut), cmp.Compare(a.id, b.id))
	})
	for _, e := range ev {
		e.jalan()
	}
	for _, id := range nilai.ids() {
		q, n := nilai.posisi(id)
		k.Posisi[id] = PosisiStok{Qty: q, Nilai: n}
	}
	return k
}

// PosisiPada: saldo satu barang setelah semua pergerakan sampai tanggal itu (inklusif).
// Pada: salinan kartu dengan posisi setiap barang per tanggal itu (laporan stok per tanggal).
func (k KartuStok) Pada(tanggal string) KartuStok {
	posisi := make(map[int]PosisiStok, len(k.Posisi))
	for id := range k.Posisi {
		posisi[id] = k.PosisiPada(id, tanggal)
	}
	k.Posisi = posisi
	return k
}

func (k KartuStok) PosisiPada(stokID int, tanggal string) PosisiStok {
	var p PosisiStok
	for _, b := range k.Baris {
		if b.Tanggal > tanggal {
			break
		}
		if b.StokID == stokID {
			p = PosisiStok{Qty: b.SaldoQty, Nilai: b.SaldoNilai}
		}
	}
	return p
}

// FormatQty: 2500 → "2,5"; dipakai di ekspor & pesan.
func FormatQty(q int64) string {
	neg := q < 0
	if neg {
		q = -q
	}
	s := strconv.FormatInt(q/TakaranSkala, 10)
	if r := q % TakaranSkala; r != 0 {
		d := strconv.FormatInt(r+TakaranSkala, 10)[1:] // 3 digit dengan nol di depan
		for d[len(d)-1] == '0' {
			d = d[:len(d)-1]
		}
		s += "," + d
	}
	if neg {
		return "-" + s
	}
	return s
}

// --- Laporan mutasi stok ---

type MutasiItem struct {
	StokID        int    `json:"stok_id"`
	SKU           string `json:"sku"`
	Nama          string `json:"nama"`
	Kategori      string `json:"kategori"`
	Satuan        string `json:"satuan"`
	AwalQty       int64  `json:"awal_qty"`
	AwalNilai     int64  `json:"awal_nilai"`
	BeliQty       int64  `json:"beli_qty"`
	BeliNilai     int64  `json:"beli_nilai"`
	ProduksiQty   int64  `json:"produksi_qty"` // barang jadi masuk dari produksi
	ProduksiNilai int64  `json:"produksi_nilai"`
	JualQty       int64  `json:"jual_qty"`
	JualNilai     int64  `json:"jual_nilai"` // = HPP penjualan
	PakaiQty      int64  `json:"pakai_qty"`  // bahan dipakai produksi
	PakaiNilai    int64  `json:"pakai_nilai"`
	OpnameQty     int64  `json:"opname_qty"` // bertanda: + lebih, − kurang
	OpnameNilai   int64  `json:"opname_nilai"`
	KoreksiNilai  int64  `json:"koreksi_nilai"`
	AkhirQty      int64  `json:"akhir_qty"`
	AkhirNilai    int64  `json:"akhir_nilai"`
	Minus         bool   `json:"minus"`
}

type LaporanMutasi struct {
	Dari   string       `json:"dari"`
	Sampai string       `json:"sampai"`
	Mulai  string       `json:"mulai"`
	Items  []MutasiItem `json:"items"`
	Total  MutasiItem   `json:"total"`
	// HPPJual: nilai pokok barang terjual; SelisihOpname: nilai bersih penyesuaian opname
	// (negatif = barang hilang/rusak). Dipakai laporan keuangan setelah titik nol.
	HPPJual       int64 `json:"hpp_jual"`
	SelisihOpname int64 `json:"selisih_opname"`
}

func (m *MutasiItem) tambah(b BarisKartu) {
	switch b.Jenis {
	case KartuAwal:
		m.AwalQty += b.Qty
		m.AwalNilai += b.Nilai
	case KartuBeli:
		m.BeliQty += b.Qty
		m.BeliNilai += b.Nilai
	case KartuProduksiMasuk:
		m.ProduksiQty += b.Qty
		m.ProduksiNilai += b.Nilai
	case KartuJual:
		m.JualQty -= b.Qty
		m.JualNilai -= b.Nilai
	case KartuProduksiPakai:
		m.PakaiQty -= b.Qty
		m.PakaiNilai -= b.Nilai
	case KartuOpname:
		m.OpnameQty += b.Qty
		m.OpnameNilai += b.Nilai
	}
	m.KoreksiNilai += b.Koreksi
}

func (m *MutasiItem) jumlahkan(x MutasiItem) {
	m.AwalQty += x.AwalQty
	m.AwalNilai += x.AwalNilai
	m.BeliQty += x.BeliQty
	m.BeliNilai += x.BeliNilai
	m.ProduksiQty += x.ProduksiQty
	m.ProduksiNilai += x.ProduksiNilai
	m.JualQty += x.JualQty
	m.JualNilai += x.JualNilai
	m.PakaiQty += x.PakaiQty
	m.PakaiNilai += x.PakaiNilai
	m.OpnameQty += x.OpnameQty
	m.OpnameNilai += x.OpnameNilai
	m.KoreksiNilai += x.KoreksiNilai
	m.AkhirQty += x.AkhirQty
	m.AkhirNilai += x.AkhirNilai
}

// Mutasi merekap pergerakan tiap barang dalam rentang tanggal [dari, sampai].
func (k KartuStok) Mutasi(stok []StokItem, dari, sampai string) LaporanMutasi {
	l := LaporanMutasi{Dari: dari, Sampai: sampai, Mulai: k.Mulai, Items: []MutasiItem{}}
	per := map[int]*MutasiItem{}
	for _, s := range stok {
		per[s.ID] = &MutasiItem{StokID: s.ID, SKU: s.SKU, Nama: s.Nama, Kategori: s.Kategori, Satuan: s.Satuan}
	}
	for _, b := range k.Baris {
		m := per[b.StokID]
		if m == nil || b.Tanggal > sampai {
			continue
		}
		if b.Tanggal < dari {
			m.AwalQty, m.AwalNilai = b.SaldoQty, b.SaldoNilai
			m.AkhirQty, m.AkhirNilai = b.SaldoQty, b.SaldoNilai
			continue
		}
		m.tambah(b)
		m.AkhirQty, m.AkhirNilai = b.SaldoQty, b.SaldoNilai
	}
	for _, s := range stok {
		m := per[s.ID]
		m.Minus = m.AkhirQty < 0
		l.Items = append(l.Items, *m)
		l.Total.jumlahkan(*m)
	}
	l.Total.Nama = "Total"
	l.HPPJual = l.Total.JualNilai
	l.SelisihOpname = l.Total.OpnameNilai
	return l
}
