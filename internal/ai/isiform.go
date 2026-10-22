package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"pajak/internal/finance"
	"pajak/internal/llm"
)

// Isi form oleh AI: kalimat pengguna ("catat penjualan kopi 2 juta tunai") dibaca model menjadi
// usulan isian satu form. Server tidak menyimpan apa pun; pengguna melihat pratinjaunya, form
// diisi di layar, lalu pengguna sendiri yang menekan Simpan (dan bisa mengurungkan isiannya).

// IsiFormHasil: usulan isian untuk form di View. Isian memakai kunci JSON yang sama dengan
// input form di frontend (TxInput, InvoiceInput, StokInput, SupplierInput), hanya kolom yang terisi.
type IsiFormHasil struct {
	Form    string          `json:"form"`
	View    string          `json:"view"`
	Isian   json.RawMessage `json:"isian"`
	Catatan []string        `json:"catatan,omitempty"`
}

// ViewForm: halaman tempat form berada; dipakai juga untuk cek hak akses.
var ViewForm = map[string]string{"transaksi": "transaksi", "invoice": "invoice", "stok": "stok-gudang", "pemasok": "produksi-supplier"}

// JudulForm: nama form dalam bahasa pengguna.
var JudulForm = map[string]string{"transaksi": "Catat transaksi", "invoice": "Buat invoice", "stok": "Barang baru", "pemasok": "Pemasok baru"}

var (
	kataIsi = []string{"catat", "catatkan", "catatin", "input", "inputkan", "masukkan", "masukin", "tambah", "tambahkan",
		"buat", "buatkan", "bikin", "bikinin", "isi", "isikan", "isiin", "rekam"}
	// kataTanyaCara: penanda pertanyaan cara/tempat ("bagaimana cara catat penjualan?") — itu panduan menu.
	kataTanyaCara = []string{"di mana", "dimana", "dmn", "mana", "cara", "caranya", "gimana", "gmn", "bagaimana", "bgmn",
		"menu", "halaman", "kenapa", "mengapa", "apakah", "bisakah", "berapa"}
	kataForm = map[string][]string{
		"invoice": {"invoice", "invois", "faktur"},
		"transaksi": {"penjualan", "jual", "menjual", "terjual", "pendapatan", "pemasukan", "pengeluaran", "biaya", "beli",
			"membeli", "pembelian", "bayar", "membayar", "pembayaran", "gaji", "upah", "sewa", "listrik", "ongkir", "iklan",
			"modal", "setor", "pinjaman", "cicilan", "transaksi"},
		"pemasok": {"pemasok", "supplier", "suplier", "vendor"},
		"stok":    {"barang", "produk", "stok"},
	}
)

// DeteksiIsiForm: form yang diminta diisi, atau "" bila kalimat bukan perintah mengisi form.
// Perlu kata kerja (catat, buat, tambah, ...) lalu kata benda form; kata benda PERTAMA sesudah
// kata kerja yang menentukan, supaya "tambah barang ... harga jual" tetap form barang dan
// "buat invoice penjualan" tetap invoice.
func DeteksiIsiForm(q string) string {
	n := normal(q)
	if slices.ContainsFunc(kataTanyaCara, func(k string) bool { return strings.Contains(n, " "+k+" ") }) {
		return ""
	}
	kata := strings.Fields(n)
	i := slices.IndexFunc(kata, func(k string) bool { return slices.Contains(kataIsi, k) })
	if i < 0 {
		return ""
	}
	for _, k := range kata[i+1:] {
		for form, daftar := range kataForm {
			if slices.ContainsFunc(daftar, func(d string) bool { return k == d || len(d) >= 6 && bedaHuruf(k, d) <= 1 }) {
				return form
			}
		}
	}
	return ""
}

const promptIsiForm = `Kamu membantu pemilik UMKM Indonesia mengisi form aplikasi pembukuan dari kalimat mereka.

ATURAN:
1. Isi HANYA kolom yang disebut atau jelas dari kalimat. Jangan menebak atau mengarang nama, angka, atau tanggal; kolom yang tidak disebut dihilangkan.
2. Rupiah dan jumlah ditulis sebagai angka JSON tanpa titik atau "Rp": "2 juta" = 2000000, "2,5 jt" = 2500000, "50rb" = 50000.
3. Tanggal format YYYY-MM-DD. Hari ini %s, kemarin %s.
4. "catatan": maksimal 3 hal yang perlu dicek pemilik (maksimal 12 kata, bahasa awam), atau [].
5. Tanpa emoji.

FORM: %s
KOLOM:
%s

Balas HANYA JSON: {"isian":{...},"catatan":[...]}`

func kolomForm(form string) string {
	switch form {
	case "transaksi":
		var jenis []string
		for _, j := range finance.JenisList {
			if j.Pos == finance.PosPindah { // perlu rekening asal & tujuan; hanya lewat Catat
				continue
			}
			jenis = append(jenis, j.Nama)
		}
		return `- tanggal
- jenis: pilih TEPAT salah satu: ` + strings.Join(jenis, "; ") + `. Kosongkan bila tidak jelas.
- nominal: rupiah total transaksi
- keterangan: singkat, mis. "Penjualan kopi"
- pihak: nama pelanggan atau pemasok bila disebut
- item: nama barang bila disebut
- qty: jumlah barang (bilangan bulat) bila disebut
- lunas: true bila tunai/sudah dibayar, false bila belum dibayar/kredit/tempo; hilangkan bila tidak disebut`
	case "invoice":
		return `- tanggal: tanggal terbit
- jatuh_tempo: YYYY-MM-DD bila tanggalnya disebut
- tempo_hari: jumlah hari bila disebut, mis. "tempo 30 hari" = 30 (jangan hitung tanggalnya)
- pelanggan_nama, pelanggan_alamat
- items: [{"deskripsi","qty","satuan","harga_satuan"}] — harga PER SATUAN, bukan total
- diskon_rp: rupiah
- catatan_invoice: catatan yang dicetak di invoice`
	case "stok":
		return `- nama: nama barang
- kategori: mis. Bahan Baku, Produk Jadi, Barang Dagang
- satuan: pcs, kg, liter, dll.
- jumlah: stok awal (bilangan bulat dalam satuannya)
- stok_min: batas stok minimum
- harga_pokok, harga_jual: rupiah per satuan`
	case "pemasok":
		return `- nama: nama usaha pemasok
- kontak: nama orang yang dihubungi
- telepon, email, alamat
- tempo_hari: termin pembayaran dalam hari (0 = tunai)`
	}
	return ""
}

// IsiForm meminta model membaca kalimat pengguna menjadi isian form, lalu menormalkannya:
// hanya kolom form, jenis harus dari daftar, rupiah bulat ≥ 0, tanggal sah (kosong = hariIni),
// tanggal jatuh tempo dihitung di sini dari tempo_hari (model tidak berhitung).
func IsiForm(ctx context.Context, c *llm.Client, form, query, hariIni string) (*IsiFormHasil, error) {
	if c == nil || !c.Enabled() {
		return nil, errors.New("AI belum aktif")
	}
	if _, ok := ViewForm[form]; !ok {
		return nil, fmt.Errorf("form %q tidak dikenal", form)
	}
	hari, err := time.Parse(time.DateOnly, hariIni)
	if err != nil {
		return nil, err
	}
	sistem := fmt.Sprintf(promptIsiForm, hariIni, hari.AddDate(0, 0, -1).Format(time.DateOnly), JudulForm[form], kolomForm(form))
	maks := 400
	if form == "invoice" {
		maks = 800
	}
	raw, err := c.Complete(ctx, []llm.Message{{Role: "system", Content: sistem}, {Role: "user", Content: query}}, true, maks)
	if err != nil {
		return nil, err
	}
	var out struct {
		Isian   json.RawMessage `json:"isian"`
		Catatan []string        `json:"catatan"`
	}
	if err := json.Unmarshal([]byte(llm.ExtractJSON(raw)), &out); err != nil || len(out.Isian) == 0 {
		return nil, errors.New("balasan model tidak berbentuk JSON yang diharapkan")
	}
	h := &IsiFormHasil{Form: form, View: ViewForm[form]}
	var isian any
	switch form {
	case "transaksi":
		isian, err = h.transaksi(out.Isian, hariIni)
	case "invoice":
		isian, err = h.invoice(out.Isian, hariIni)
	case "stok":
		isian, err = h.stok(out.Isian)
	case "pemasok":
		isian, err = h.pemasok(out.Isian)
	}
	if err != nil {
		return nil, err
	}
	if h.Isian, err = json.Marshal(isian); err != nil {
		return nil, err
	}
	if string(h.Isian) == "{}" {
		return nil, errors.New("kalimat belum berisi data yang bisa diisi")
	}
	for _, c := range out.Catatan {
		if c = teks(c, 120); c != "" && len(h.Catatan) < 4 {
			h.Catatan = append(h.Catatan, c)
		}
	}
	return h, nil
}

// bil: angka dari model; terima angka JSON atau string angka ("85000", "Rp 85.000", "2,5").
type bil float64

func (b *bil) UnmarshalJSON(d []byte) error {
	s := strings.Trim(string(d), `"`)
	s = strings.NewReplacer("Rp", "", "rp", "", " ", "").Replace(s)
	if strings.Count(s, ".") > 1 || strings.Contains(s, ",") {
		s = strings.ReplaceAll(s, ".", "") // titik ribuan gaya Indonesia
	}
	f, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		*b = 0 // angka tidak terbaca: kolom dikosongkan, bukan galat seluruh isian
		return nil
	}
	*b = bil(f)
	return nil
}

// bulat: bilangan bulat ≥ 0 (rupiah, qty, hari); negatif atau kosong = 0 (kolom dihilangkan).
func (b bil) bulat() int64 { return int64(math.Round(max(float64(b), 0))) }

// teks: dipangkas spasinya dan dibatasi panjangnya (rune) supaya isian tetap wajar di form.
func teks(s string, maks int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > maks {
		s = string(r[:maks])
	}
	return s
}

func (h *IsiFormHasil) tanggal(s, hariIni, label string) string {
	if _, err := time.Parse(time.DateOnly, s); err == nil {
		return s
	}
	h.Catatan = append(h.Catatan, label+" tidak disebut, dipakai hari ini.")
	return hariIni
}

func (h *IsiFormHasil) transaksi(raw json.RawMessage, hariIni string) (any, error) {
	var m struct {
		Tanggal, Jenis, Keterangan, Pihak, Item string
		Nominal, Qty                            bil
		Lunas                                   *bool
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	out := struct {
		Tanggal    string `json:"tanggal"`
		Jenis      string `json:"jenis,omitempty"`
		Nominal    int64  `json:"nominal,omitempty"`
		Keterangan string `json:"keterangan,omitempty"`
		Pihak      string `json:"pihak,omitempty"`
		Item       string `json:"item,omitempty"`
		Qty        int64  `json:"qty,omitempty"`
		Lunas      *bool  `json:"lunas,omitempty"`
	}{Tanggal: h.tanggal(m.Tanggal, hariIni, "Tanggal"), Nominal: m.Nominal.bulat(), Keterangan: teks(m.Keterangan, 200),
		Pihak: teks(m.Pihak, 120), Item: teks(m.Item, 120), Qty: m.Qty.bulat(), Lunas: m.Lunas}
	if _, ok := finance.LookupJenis(m.Jenis); ok {
		out.Jenis = m.Jenis
	} else {
		h.Catatan = append(h.Catatan, "Jenis transaksi belum jelas, pilih sendiri di form.")
	}
	if out.Nominal == 0 {
		h.Catatan = append(h.Catatan, "Nominal belum disebut, isi sendiri di form.")
	}
	return out, nil
}

func (h *IsiFormHasil) invoice(raw json.RawMessage, hariIni string) (any, error) {
	var m struct {
		Tanggal         string `json:"tanggal"`
		JatuhTempo      string `json:"jatuh_tempo"`
		TempoHari       bil    `json:"tempo_hari"`
		PelangganNama   string `json:"pelanggan_nama"`
		PelangganAlamat string `json:"pelanggan_alamat"`
		Items           []struct {
			Deskripsi, Satuan string
			Qty               bil
			HargaSatuan       bil `json:"harga_satuan"`
		} `json:"items"`
		DiskonRp       bil    `json:"diskon_rp"`
		CatatanInvoice string `json:"catatan_invoice"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	type item struct {
		Deskripsi   string `json:"deskripsi"`
		Qty         int64  `json:"qty"`
		Satuan      string `json:"satuan,omitempty"`
		HargaSatuan int64  `json:"harga_satuan"`
	}
	out := struct {
		Tanggal         string `json:"tanggal"`
		JatuhTempo      string `json:"jatuh_tempo,omitempty"`
		PelangganNama   string `json:"pelanggan_nama,omitempty"`
		PelangganAlamat string `json:"pelanggan_alamat,omitempty"`
		Items           []item `json:"items,omitempty"`
		DiskonRp        int64  `json:"diskon_rp,omitempty"`
		Catatan         string `json:"catatan,omitempty"`
	}{Tanggal: h.tanggal(m.Tanggal, hariIni, "Tanggal invoice"), PelangganNama: teks(m.PelangganNama, 120),
		PelangganAlamat: teks(m.PelangganAlamat, 200), DiskonRp: m.DiskonRp.bulat(), Catatan: teks(m.CatatanInvoice, 200)}
	terbit, _ := time.Parse(time.DateOnly, out.Tanggal)
	if jt, err := time.Parse(time.DateOnly, m.JatuhTempo); err == nil && !jt.Before(terbit) {
		out.JatuhTempo = m.JatuhTempo
	} else if hari := m.TempoHari.bulat(); hari > 0 && hari <= 365 {
		out.JatuhTempo = terbit.AddDate(0, 0, int(hari)).Format(time.DateOnly)
	}
	for _, it := range m.Items {
		if d := teks(it.Deskripsi, 120); d != "" && len(out.Items) < 20 {
			out.Items = append(out.Items, item{Deskripsi: d, Qty: max(it.Qty.bulat(), 1), Satuan: teks(it.Satuan, 20), HargaSatuan: it.HargaSatuan.bulat()})
		}
	}
	if out.PelangganNama == "" {
		h.Catatan = append(h.Catatan, "Nama pelanggan belum disebut, isi sendiri di form.")
	}
	return out, nil
}

func (h *IsiFormHasil) stok(raw json.RawMessage) (any, error) {
	var m struct {
		Nama, Kategori, Satuan string
		Jumlah                 bil
		StokMin                bil `json:"stok_min"`
		HargaPokok             bil `json:"harga_pokok"`
		HargaJual              bil `json:"harga_jual"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	// Jumlah barang di daftar stok bilangan bulat dalam satuannya (nilai = jumlah × harga pokok).
	if j := float64(m.Jumlah); j != math.Trunc(j) {
		h.Catatan = append(h.Catatan, "Jumlah stok dibulatkan; pakai satuan lebih kecil (mis. gram) bila perlu.")
	}
	return struct {
		Nama       string `json:"nama,omitempty"`
		Kategori   string `json:"kategori,omitempty"`
		Satuan     string `json:"satuan,omitempty"`
		Jumlah     int64  `json:"jumlah,omitempty"`
		StokMin    int64  `json:"stok_min,omitempty"`
		HargaPokok int64  `json:"harga_pokok,omitempty"`
		HargaJual  int64  `json:"harga_jual,omitempty"`
	}{teks(m.Nama, 120), teks(m.Kategori, 60), teks(m.Satuan, 20), m.Jumlah.bulat(), m.StokMin.bulat(), m.HargaPokok.bulat(), m.HargaJual.bulat()}, nil
}

func (h *IsiFormHasil) pemasok(raw json.RawMessage) (any, error) {
	var m struct {
		Nama, Kontak, Telepon, Email, Alamat string
		TempoHari                            bil `json:"tempo_hari"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return struct {
		Nama      string `json:"nama,omitempty"`
		Kontak    string `json:"kontak,omitempty"`
		Telepon   string `json:"telepon,omitempty"`
		Email     string `json:"email,omitempty"`
		Alamat    string `json:"alamat,omitempty"`
		TempoHari int64  `json:"tempo_hari,omitempty"`
	}{teks(m.Nama, 120), teks(m.Kontak, 120), teks(m.Telepon, 40), teks(m.Email, 120), teks(m.Alamat, 200), min(m.TempoHari.bulat(), 365)}, nil
}
