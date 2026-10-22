package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"pajak/internal/finance"
	"pajak/internal/llm"
)

// Baca nota / struk / bukti transfer dari foto: model membaca tanggal, total, pihak, dan
// barangnya, lalu menyarankan jenis transaksi dari daftar aplikasi. Hasilnya hanya pratinjau;
// pengguna memeriksa dan menyimpannya sendiri.

type BarisNota struct {
	Nama  string `json:"nama"`
	Qty   string `json:"qty,omitempty"`
	Harga int64  `json:"harga,omitempty"`
}

type SaranNota struct {
	Tanggal    string      `json:"tanggal"`
	Jenis      string      `json:"jenis"`
	Nominal    int64       `json:"nominal"`
	Pihak      string      `json:"pihak,omitempty"`
	Keterangan string      `json:"keterangan,omitempty"`
	Lunas      bool        `json:"lunas"`
	Item       []BarisNota `json:"item,omitempty"`
	Alasan     string      `json:"alasan,omitempty"`
	Yakin      string      `json:"yakin"`   // tinggi | sedang | rendah
	Catatan    []string    `json:"catatan"` // hal yang perlu dicek pengguna
	Model      string      `json:"model,omitempty"`
}

const promptNota = `Kamu membantu pemilik UMKM Indonesia mencatat transaksi dari foto nota, struk, kuitansi, faktur, atau bukti transfer.

ATURAN:
1. Baca hanya yang tertulis di gambar. Jangan mengarang angka. Bila tidak terbaca, kosongkan.
2. "nominal": TOTAL yang dibayar/diterima dalam rupiah, bilangan bulat tanpa titik (mis. 125000). Pakai total akhir setelah diskon & pajak.
3. "tanggal": format YYYY-MM-DD. Tahun dua digit = 20xx.
4. "jenis": pilih SATU dari daftar jenis sesuai arahnya. Nota pembelian dari toko/supplier = uang KELUAR; nota yang diterbitkan usaha ini ke pelanggan = uang MASUK. Ragu = "".
5. "pihak": nama toko / supplier / pelanggan. "keterangan": ringkas apa yang dibeli/dijual (maks 8 kata).
6. "lunas": false hanya bila jelas belum dibayar (mis. tertulis tempo, kredit, belum lunas).
7. "item": maksimal 10 baris barang yang terbaca (nama, qty sebagai teks, harga total baris).
8. "yakin": tinggi | sedang | rendah — seberapa jelas fotonya. "alasan": maksimal 15 kata kenapa jenis itu.
9. "catatan": hal yang perlu dicek pengguna (mis. "tanggal tidak terbaca", "ada PPN 11%").

Balas HANYA JSON: {"tanggal":"","jenis":"","nominal":0,"pihak":"","keterangan":"","lunas":true,"item":[],"yakin":"","alasan":"","catatan":[]}`

// BacaNota mengirim foto ke model pembaca gambar. hariIni dipakai bila tanggal tak terbaca.
func BacaNota(ctx context.Context, c *llm.Client, usaha, mime string, foto []byte, hariIni string) (SaranNota, error) {
	if c == nil || !c.Enabled() {
		return SaranNota{}, fmt.Errorf("AI belum aktif")
	}
	var masuk, keluar []string
	for _, j := range finance.JenisList {
		if j.Pos == finance.PosPindah { // perlu rekening asal & tujuan; hanya lewat Catat
			continue
		}
		if j.Arah == "masuk" {
			masuk = append(masuk, j.Nama)
		} else {
			keluar = append(keluar, j.Nama)
		}
	}
	teks := fmt.Sprintf("Usaha: %s\nHari ini: %s\nJenis untuk uang MASUK: %s\nJenis untuk uang KELUAR: %s",
		usaha, hariIni, strings.Join(masuk, "; "), strings.Join(keluar, "; "))
	raw, model, err := c.CompleteGambar(ctx, promptNota, teks, mime, foto, true, 900)
	if err != nil {
		return SaranNota{}, err
	}
	var s SaranNota
	if err := json.Unmarshal([]byte(llm.ExtractJSON(raw)), &s); err != nil {
		return SaranNota{}, fmt.Errorf("balasan model bukan JSON")
	}
	s.Model = model
	return RapikanNota(s, hariIni), nil
}

// RapikanNota menjaga hasil model tetap aman dipakai form: jenis harus ada di daftar, tanggal
// sah dan tidak di masa depan, nominal positif. Yang dibuang dicatat supaya pengguna tahu.
func RapikanNota(s SaranNota, hariIni string) SaranNota {
	s.Catatan = append([]string{}, s.Catatan...)
	if _, ok := finance.LookupJenis(s.Jenis); !ok {
		if s.Jenis != "" {
			s.Catatan = append(s.Catatan, "jenis \""+s.Jenis+"\" tidak dikenal; pilih jenisnya sendiri")
		}
		s.Jenis = ""
	}
	if t, err := time.Parse(time.DateOnly, s.Tanggal); err != nil || s.Tanggal > hariIni || t.Year() < 2000 {
		if s.Tanggal != "" {
			s.Catatan = append(s.Catatan, "tanggal \""+s.Tanggal+"\" tidak masuk akal; diganti hari ini")
		} else {
			s.Catatan = append(s.Catatan, "tanggal tidak terbaca; diisi hari ini")
		}
		s.Tanggal = hariIni
	}
	if s.Nominal < 0 {
		s.Nominal = 0
	}
	if s.Nominal == 0 {
		s.Catatan = append(s.Catatan, "total tidak terbaca; isi nominalnya sendiri")
	}
	switch s.Yakin {
	case "tinggi", "sedang", "rendah":
	default:
		s.Yakin = "sedang"
	}
	if len(s.Item) > 10 {
		s.Item = s.Item[:10]
	}
	s.Pihak, s.Keterangan, s.Alasan = strings.TrimSpace(s.Pihak), strings.TrimSpace(s.Keterangan), strings.TrimSpace(s.Alasan)
	return s
}
