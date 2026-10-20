package finance

import (
	"errors"
	"math/big"
)

// Simulasi usaha untuk menu Simulasi & Harga: "kalau harga/jualan/modal/biaya tetap berubah,
// laba per bulan jadi berapa", titik impas usaha, dan porsi per hari supaya balik modal per
// produk. Dasar = rata-rata bulan terakhir yang ada penjualannya. Semua rupiah bilangan bulat;
// perkalian lewat big.Int supaya omzet ratusan miliar tidak overflow.

// DasarSimulasi: rata-rata per bulan.
type DasarSimulasi struct {
	Periode        []string `json:"periode"`
	Omzet          int64    `json:"omzet"`
	ModalBahan     int64    `json:"modal_bahan"`    // HPP / pembelian bahan
	BiayaVariabel  int64    `json:"biaya_variabel"` // potongan aplikasi/channel: ikut naik-turun dengan omzet
	BiayaTetap     int64    `json:"biaya_tetap"`    // beban operasional kas (gaji, sewa, listrik, ...)
	Penyusutan     int64    `json:"penyusutan"`     // penyusutan + amortisasi register aset
	PendapatanLain int64    `json:"pendapatan_lain"`
	Laba           int64    `json:"laba"`
	HariBuka       int64    `json:"hari_buka"` // rata-rata hari kalender per bulan dasar
}

// BulanSimulasi: laba rugi satu bulan + jumlah hari buka-nya.
type BulanSimulasi struct {
	Periode  string
	Income   IncomeStatement
	HariBuka int64
}

// mulDiv: a*b/c dibulatkan ke rupiah terdekat (setengah menjauhi nol); c > 0.
func mulDiv(a, b, c int64) int64 {
	n := new(big.Int).Mul(big.NewInt(a), big.NewInt(b))
	q, r := new(big.Int).QuoRem(n, big.NewInt(c), new(big.Int))
	if r.Abs(r).Mul(r, big.NewInt(2)).Cmp(big.NewInt(c)) >= 0 {
		if n.Sign() < 0 {
			q.Sub(q, big.NewInt(1))
		} else {
			q.Add(q, big.NewInt(1))
		}
	}
	return q.Int64()
}

// mulDivCeil: a*b/c dibulatkan ke atas; a, b ≥ 0, c > 0.
func mulDivCeil(a, b, c int64) int64 {
	n := new(big.Int).Mul(big.NewInt(a), big.NewInt(b))
	q, r := new(big.Int).QuoRem(n, big.NewInt(c), new(big.Int))
	if r.Sign() > 0 {
		q.Add(q, big.NewInt(1))
	}
	return q.Int64()
}

// RataRataSimulasi merata-rata bulan-bulan dasar (pemanggil memilih bulan yang ada penjualannya).
func RataRataSimulasi(bulan []BulanSimulasi) DasarSimulasi {
	d := DasarSimulasi{Periode: []string{}}
	n := int64(len(bulan))
	if n == 0 {
		return d
	}
	for _, b := range bulan {
		d.Periode = append(d.Periode, b.Periode)
		d.Omzet += b.Income.Omzet
		d.ModalBahan += b.Income.COGS
		d.BiayaVariabel += b.Income.ChannelFees
		d.BiayaTetap += b.Income.Opex
		d.Penyusutan += b.Income.Depreciation + b.Income.Amortization
		d.PendapatanLain += b.Income.OtherIncome
		d.HariBuka += b.HariBuka
	}
	for _, p := range []*int64{&d.Omzet, &d.ModalBahan, &d.BiayaVariabel, &d.BiayaTetap, &d.Penyusutan, &d.PendapatanLain, &d.HariBuka} {
		*p = mulDiv(*p, 1, n)
	}
	d.Laba = d.Omzet - d.ModalBahan - d.BiayaVariabel - d.BiayaTetap - d.Penyusutan + d.PendapatanLain
	return d
}

// Skenario: perubahan dalam basis poin (500 = +5%, -1000 = -10%) dan tambahan biaya tetap.
type Skenario struct {
	HargaBps    int64 `json:"harga_bps"`  // harga jual rata-rata
	VolumeBps   int64 `json:"volume_bps"` // jumlah pembeli / porsi terjual
	ModalBps    int64 `json:"modal_bps"`  // modal bahan per porsi
	TambahTetap int64 `json:"tambah_tetap"`
}

type HasilSimulasi struct {
	Omzet          int64 `json:"omzet"`
	ModalBahan     int64 `json:"modal_bahan"`
	BiayaVariabel  int64 `json:"biaya_variabel"`
	BiayaTetap     int64 `json:"biaya_tetap"`
	Penyusutan     int64 `json:"penyusutan"`
	PendapatanLain int64 `json:"pendapatan_lain"`
	Laba           int64 `json:"laba"`
	MarginBps      int64 `json:"margin_bps"`
}

func Simulasikan(d DasarSimulasi, s Skenario) HasilSimulasi {
	const satu = 10_000
	h := HasilSimulasi{BiayaTetap: d.BiayaTetap + s.TambahTetap, Penyusutan: d.Penyusutan, PendapatanLain: d.PendapatanLain}
	h.Omzet = mulDiv(mulDiv(d.Omzet, satu+s.HargaBps, satu), satu+s.VolumeBps, satu)
	h.ModalBahan = mulDiv(mulDiv(d.ModalBahan, satu+s.VolumeBps, satu), satu+s.ModalBps, satu)
	if d.Omzet > 0 { // komisi channel adalah persentase omzet
		h.BiayaVariabel = mulDiv(d.BiayaVariabel, h.Omzet, d.Omzet)
	}
	h.Laba = h.Omzet - h.ModalBahan - h.BiayaVariabel - h.BiayaTetap - h.Penyusutan + h.PendapatanLain
	if h.Omzet > 0 {
		h.MarginBps = h.Laba * satu / h.Omzet
	}
	return h
}

// TitikImpasUsaha: omzet minimal supaya tidak rugi (biaya tetap + penyusutan tertutup).
type TitikImpasUsaha struct {
	Ada          bool  `json:"ada"` // false = tiap penjualan menambah rugi
	PerBulan     int64 `json:"per_bulan"`
	PerHari      int64 `json:"per_hari"`
	BiayaTetap   int64 `json:"biaya_tetap"`    // biaya tetap + penyusutan per bulan
	SisaPer100rb int64 `json:"sisa_per_100rb"` // sisa tiap Rp100.000 jualan setelah bahan & biaya variabel
	DiAtasBps    int64 `json:"di_atas_bps"`    // (omzet - titik impas) / omzet; negatif = di bawah
}

func HitungTitikImpas(h HasilSimulasi, hariBuka int64) TitikImpasUsaha {
	t := TitikImpasUsaha{BiayaTetap: h.BiayaTetap + h.Penyusutan}
	kontribusi := h.Omzet - h.ModalBahan - h.BiayaVariabel
	if h.Omzet <= 0 || kontribusi <= 0 {
		return t
	}
	t.Ada = true
	t.PerBulan = mulDivCeil(max(t.BiayaTetap, 0), h.Omzet, kontribusi)
	t.SisaPer100rb = mulDiv(kontribusi, 100_000, h.Omzet)
	t.DiAtasBps = (h.Omzet - t.PerBulan) * 10_000 / h.Omzet
	if hariBuka > 0 {
		t.PerHari = (t.PerBulan + hariBuka - 1) / hariBuka
	}
	return t
}

// PorsiProduk: berapa porsi satu produk per hari supaya biaya tetap tertutup bila hanya produk
// ini yang dijual — untuk menilai produk andalan.
type PorsiProduk struct {
	Nama         string `json:"nama"`
	Harga        int64  `json:"harga"`
	Modal        int64  `json:"modal"`
	AdaModal     bool   `json:"ada_modal"` // false = HPP belum diketahui (belum ada resep)
	Variabel     int64  `json:"variabel"`  // komisi rata-rata per porsi
	Sisa         int64  `json:"sisa"`
	PorsiPerHari int64  `json:"porsi_per_hari"`
	Ada          bool   `json:"ada"` // false = sisa per porsi ≤ 0 atau modal belum diketahui
}

func PorsiBalikModal(d DasarSimulasi, harga, modal int64) PorsiProduk {
	p := PorsiProduk{Harga: harga, Modal: modal, AdaModal: true}
	if d.Omzet > 0 {
		p.Variabel = mulDiv(harga, d.BiayaVariabel, d.Omzet)
	}
	p.Sisa = harga - modal - p.Variabel
	if p.Sisa <= 0 || d.HariBuka <= 0 {
		return p
	}
	p.Ada = true
	p.PorsiPerHari = mulDivCeil(d.BiayaTetap+d.Penyusutan, 1, p.Sisa*d.HariBuka)
	return p
}

var ErrMarginTerlaluBesar = errors.New("komisi + margin harus di bawah 100%")

// HargaUntukMargin: harga jual minimal supaya (harga - komisi - modal) / harga ≥ margin.
func HargaUntukMargin(cogs int64, feeBps int, fixedFee int64, marginBps int) (PriceSimulationResult, error) {
	sisa := int64(10_000 - feeBps - marginBps)
	if sisa <= 0 || marginBps < 0 || feeBps < 0 {
		return PriceSimulationResult{}, ErrMarginTerlaluBesar
	}
	harga := mulDivCeil(cogs+fixedFee, 10_000, sisa)
	return SimulateChannelPrice(cogs, mulDivCeil(harga, int64(marginBps), 10_000), feeBps, fixedFee), nil
}
