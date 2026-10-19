package finance

import (
	"errors"
	"strings"
)

// Produksi & Supplier: resep (BOM) per produk, HPP yang dihitung darinya, dan daftar supplier.
//
// HPP tidak diketik langsung. Ia dihitung dari harga pokok bahan yang ada di Stok & Gudang,
// supaya kenaikan harga bahan langsung terlihat di HPP produk — kalau HPP cuma angka yang
// diketik sekali, ia akan terus dipakai menetapkan harga jual lama setelah harga bahan naik.

// TakaranSkala: takaran bahan disimpan sebagai bilangan bulat per seribu satuan bahan,
// jadi 1500 berarti 1,5 satuan. Pecahan dibutuhkan karena resep nyata memakai 0,15 liter susu
// atau 18 gram biji, sementara nominal rupiah tetap bilangan bulat (tidak ada float di uang).
const TakaranSkala = 1000

type ResepBahan struct {
	// StokID merujuk ke StokItem. 0 berarti bahan di luar daftar stok — harga & namanya
	// diisi manual, dipakai untuk bahan kecil yang tidak dilacak per gudang.
	StokID      int    `json:"stok_id"`
	Nama        string `json:"nama"`
	Satuan      string `json:"satuan"`
	Takaran     int64  `json:"takaran"`      // per seribu; lihat TakaranSkala
	HargaManual int64  `json:"harga_manual"` // per satuan, hanya dipakai bila StokID 0
}

type Produk struct {
	ID       int    `json:"id"`
	SKU      string `json:"sku"`
	Nama     string `json:"nama"`
	Satuan   string `json:"satuan"` // satuan produk jadi, mis. cup, pcs, botol
	Kategori string `json:"kategori"`
	// HasilPerBatch: berapa unit produk jadi dari satu kali resep di bawah. Kalau resepnya
	// untuk satu unit, isi 1.
	HasilPerBatch    int64        `json:"hasil_per_batch"`
	Bahan            []ResepBahan `json:"bahan"`
	BiayaTenagaKerja int64        `json:"biaya_tenaga_kerja"` // per batch
	BiayaOverhead    int64        `json:"biaya_overhead"`     // per batch (gas, listrik, kemasan)
	HargaJual        int64        `json:"harga_jual"`         // per unit
	Catatan          string       `json:"catatan,omitempty"`
}

func ValidateProduk(p *Produk) error {
	for _, s := range []*string{&p.SKU, &p.Nama, &p.Satuan, &p.Kategori, &p.Catatan} {
		*s = strings.TrimSpace(*s)
	}
	if p.Nama == "" {
		return errors.New("nama produk wajib diisi")
	}
	if p.HasilPerBatch <= 0 {
		return errors.New("hasil per batch harus lebih dari 0 (isi 1 bila resepnya untuk satu unit)")
	}
	if p.BiayaTenagaKerja < 0 || p.BiayaOverhead < 0 || p.HargaJual < 0 {
		return errors.New("biaya dan harga jual tidak boleh negatif")
	}
	if p.Satuan == "" {
		p.Satuan = "pcs"
	}
	if p.Kategori == "" {
		p.Kategori = "Produk Jadi"
	}
	bahan := make([]ResepBahan, 0, len(p.Bahan))
	for i := range p.Bahan {
		b := p.Bahan[i]
		b.Nama = strings.TrimSpace(b.Nama)
		b.Satuan = strings.TrimSpace(b.Satuan)
		if b.StokID == 0 && b.Nama == "" {
			continue // baris kosong dari form, bukan kesalahan
		}
		if b.Takaran <= 0 {
			return errors.New("takaran bahan " + namaAtau(b.Nama, "tanpa nama") + " harus lebih dari 0")
		}
		if b.HargaManual < 0 {
			return errors.New("harga bahan tidak boleh negatif")
		}
		bahan = append(bahan, b)
	}
	p.Bahan = bahan
	return nil
}

func namaAtau(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// HPPBahan: satu baris rincian bahan beserta biayanya.
type HPPBahan struct {
	Nama        string `json:"nama"`
	Satuan      string `json:"satuan"`
	Takaran     int64  `json:"takaran"` // per seribu
	HargaSatuan int64  `json:"harga_satuan"`
	Subtotal    int64  `json:"subtotal"`
	// Sumber: "stok" bila harganya dari Stok & Gudang, "manual" bila diisi sendiri.
	Sumber     string `json:"sumber"`
	Peringatan string `json:"peringatan,omitempty"`
}

// HPP: rincian harga pokok satu produk.
type HPP struct {
	Bahan         []HPPBahan `json:"bahan"`
	TotalBahan    int64      `json:"total_bahan"`
	TenagaKerja   int64      `json:"tenaga_kerja"`
	Overhead      int64      `json:"overhead"`
	TotalBatch    int64      `json:"total_batch"`
	HasilPerBatch int64      `json:"hasil_per_batch"`
	PerUnit       int64      `json:"per_unit"`
	HargaJual     int64      `json:"harga_jual"`
	LabaPerUnit   int64      `json:"laba_per_unit"`
	MarginBps     int        `json:"margin_bps"`
	Peringatan    []string   `json:"peringatan"`
}

// bagiKeAtas membulatkan ke atas. HPP yang kurang dari seharusnya membuat harga jual
// ditetapkan terlalu rendah, jadi pembulatan diarahkan ke sisi yang aman.
func bagiKeAtas(a, b int64) int64 {
	if b == 0 {
		return 0
	}
	if a%b == 0 {
		return a / b
	}
	return a/b + 1
}

// HitungHPP menghitung harga pokok satu produk dari resepnya dan harga bahan di stok.
//
// Bahan yang merujuk stok tapi tidak ditemukan, atau yang harga pokoknya masih 0, tidak
// membuat perhitungan gagal — biayanya dihitung 0 dan diberi peringatan, supaya jelas bahwa
// HPP-nya belum lengkap dan bukan sekadar murah.
func HitungHPP(p Produk, stok []StokItem) HPP {
	byID := make(map[int]StokItem, len(stok))
	for _, s := range stok {
		byID[s.ID] = s
	}
	h := HPP{HasilPerBatch: p.HasilPerBatch, TenagaKerja: p.BiayaTenagaKerja, Overhead: p.BiayaOverhead, HargaJual: p.HargaJual}
	h.Bahan = make([]HPPBahan, 0, len(p.Bahan))
	for _, b := range p.Bahan {
		row := HPPBahan{Nama: b.Nama, Satuan: b.Satuan, Takaran: b.Takaran, Sumber: "manual", HargaSatuan: b.HargaManual}
		if b.StokID != 0 {
			row.Sumber = "stok"
			s, ok := byID[b.StokID]
			if !ok {
				row.Peringatan = "bahan tidak ada lagi di Stok & Gudang; biayanya dihitung 0"
				row.HargaSatuan = 0
			} else {
				row.Nama = s.Nama
				row.Satuan = s.Satuan
				row.HargaSatuan = s.HargaPokok
				if s.HargaPokok == 0 {
					row.Peringatan = "harga pokok bahan ini masih 0 di Stok & Gudang"
				}
			}
		} else if b.HargaManual == 0 {
			row.Peringatan = "harga bahan belum diisi"
		}
		row.Subtotal = bagiKeAtas(row.HargaSatuan*b.Takaran, TakaranSkala)
		h.TotalBahan += row.Subtotal
		h.Bahan = append(h.Bahan, row)
		if row.Peringatan != "" {
			h.Peringatan = append(h.Peringatan, row.Nama+": "+row.Peringatan)
		}
	}
	h.TotalBatch = h.TotalBahan + h.TenagaKerja + h.Overhead
	h.PerUnit = bagiKeAtas(h.TotalBatch, p.HasilPerBatch)
	if len(p.Bahan) == 0 {
		h.Peringatan = append(h.Peringatan, "resep belum berisi bahan, HPP hanya dari tenaga kerja & overhead")
	}
	if h.HargaJual > 0 {
		h.LabaPerUnit = h.HargaJual - h.PerUnit
		h.MarginBps = int(h.LabaPerUnit * 10_000 / h.HargaJual)
		if h.LabaPerUnit < 0 {
			h.Peringatan = append(h.Peringatan, "harga jual di bawah HPP: setiap unit terjual menambah rugi")
		}
	}
	return h
}

type Supplier struct {
	ID        int    `json:"id"`
	Nama      string `json:"nama"`
	Kontak    string `json:"kontak"`
	Telepon   string `json:"telepon"`
	Email     string `json:"email"`
	Alamat    string `json:"alamat"`
	TempoHari int    `json:"tempo_hari"` // termin pembayaran, 0 = tunai
	Catatan   string `json:"catatan,omitempty"`
}

func ValidateSupplier(s *Supplier) error {
	for _, p := range []*string{&s.Nama, &s.Kontak, &s.Telepon, &s.Email, &s.Alamat, &s.Catatan} {
		*p = strings.TrimSpace(*p)
	}
	if s.Nama == "" {
		return errors.New("nama supplier wajib diisi")
	}
	if s.TempoHari < 0 {
		return errors.New("tempo pembayaran tidak boleh negatif")
	}
	return nil
}

// ProdukRow: produk beserta HPP hasil hitung dan perbandingannya dengan HPP di Stok & Gudang.
type ProdukRow struct {
	Produk
	HPP HPP `json:"hpp"`
	// HPPStok: harga pokok produk ini yang tercatat di Stok & Gudang (dicocokkan lewat SKU).
	// Ada = -1 bila SKU-nya tidak ada di stok, supaya 0 tidak rancu dengan "belum diisi".
	HPPStok   int64 `json:"hpp_stok"`
	AdaDiStok bool  `json:"ada_di_stok"`
	// Selisih: HPP resep - HPP di stok. Bukan 0 berarti HPP di stok sudah basi.
	Selisih int64 `json:"selisih"`
}

// SupplierRow: supplier beserta utang yang belum dibayar kepadanya.
type SupplierRow struct {
	Supplier
	// UtangBelumDibayar dicocokkan dari nama pihak di Piutang & Utang. Pencocokan nama memang
	// rapuh, karena itu angkanya disebut sebagai perkiraan di tampilan, bukan saldo resmi.
	UtangBelumDibayar int64 `json:"utang_belum_dibayar"`
	JumlahTagihan     int   `json:"jumlah_tagihan"`
}

type ProduksiSummary struct {
	Produk   []ProdukRow   `json:"produk"`
	Supplier []SupplierRow `json:"supplier"`
	// HPPTerbesar & MarginTerkecil untuk ringkasan di atas halaman.
	TotalProduk    int   `json:"total_produk"`
	TotalSupplier  int   `json:"total_supplier"`
	ProdukRugi     int   `json:"produk_rugi"`      // harga jual di bawah HPP
	ProdukTanpaHPP int   `json:"produk_tanpa_hpp"` // resep belum lengkap
	UtangSupplier  int64 `json:"utang_supplier"`
}

func SummarizeProduksi(produk []Produk, supplier []Supplier, stok []StokItem, tagihan []Tagihan) ProduksiSummary {
	out := ProduksiSummary{
		Produk:        make([]ProdukRow, 0, len(produk)),
		Supplier:      make([]SupplierRow, 0, len(supplier)),
		TotalProduk:   len(produk),
		TotalSupplier: len(supplier),
	}
	bySKU := make(map[string]StokItem, len(stok))
	for _, s := range stok {
		if s.SKU != "" {
			bySKU[strings.ToUpper(s.SKU)] = s
		}
	}
	for _, p := range produk {
		row := ProdukRow{Produk: p, HPP: HitungHPP(p, stok)}
		if s, ok := bySKU[strings.ToUpper(p.SKU)]; ok && p.SKU != "" {
			row.AdaDiStok = true
			row.HPPStok = s.HargaPokok
			row.Selisih = row.HPP.PerUnit - s.HargaPokok
		}
		if row.HPP.LabaPerUnit < 0 {
			out.ProdukRugi++
		}
		if len(row.HPP.Peringatan) > 0 {
			out.ProdukTanpaHPP++
		}
		out.Produk = append(out.Produk, row)
	}
	// Utang per supplier: dijumlahkan dari tagihan PAYABLE yang belum lunas.
	utang := map[string]struct {
		total int64
		n     int
	}{}
	for _, t := range tagihan {
		if t.Lunas || t.Jenis != TagihanUtang {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(t.Pihak))
		v := utang[k]
		v.total += t.Nominal
		v.n++
		utang[k] = v
	}
	for _, s := range supplier {
		v := utang[strings.ToLower(s.Nama)]
		out.Supplier = append(out.Supplier, SupplierRow{Supplier: s, UtangBelumDibayar: v.total, JumlahTagihan: v.n})
		out.UtangSupplier += v.total
	}
	return out
}
