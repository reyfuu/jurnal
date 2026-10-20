package finance

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Jadwal pengingat laporan & pajak: diatur tim Master Pajak untuk semua klien (mis. tutup
// buku bulanan, setor PPh, lapor SPT). Klien melihatnya sebagai banner di aplikasi mulai H-n
// sebelum jatuh tempo. Tanggal dihitung di WIB.

// WIB: zona waktu tenggat. FixedZone, bukan LoadLocation, supaya tidak butuh tzdata di kontainer.
var WIB = time.FixedZone("WIB", 7*3600)

const (
	UntukSemua = "semua"
	UntukOP    = "op"    // orang pribadi (bentuk usaha op)
	UntukBadan = "badan" // CV, PT, PT perorangan, dll.
	UntukPKP   = "pkp"   // Pengusaha Kena Pajak

	MaksJadwal = 30
)

type JadwalPengingat struct {
	ID         int    `json:"id"`
	Judul      string `json:"judul"`
	Keterangan string `json:"keterangan,omitempty"`
	Frekuensi  string `json:"frekuensi"`       // bulanan | tahunan
	Tanggal    int    `json:"tanggal"`         // 1–31; melebihi hari terakhir bulan → hari terakhir
	Bulan      int    `json:"bulan,omitempty"` // 1–12, hanya untuk tahunan
	// HariSebelum: banner mulai tampil H-n (0 = hanya di hari H).
	HariSebelum int    `json:"hari_sebelum"`
	Untuk       string `json:"untuk"` // semua | op | badan | pkp
	Aktif       bool   `json:"aktif"`
	// PerluVerifikasi: tenggat bawaan berdasar aturan pajak yang belum dicek tim pajak.
	PerluVerifikasi bool `json:"perlu_verifikasi,omitempty"`
}

type DaftarJadwal struct {
	Item   []JadwalPengingat `json:"item"`
	NextID int               `json:"next_id"`
}

// JadwalBawaan: tenggat umum UMKM. Tenggat pajak ditandai PerluVerifikasi — tim pajak wajib
// mencocokkan dengan aturan terbaru (PMK 81/2024 & Coretax) sebelum tanda itu dilepas.
func JadwalBawaan() DaftarJadwal {
	item := []JadwalPengingat{
		{Judul: "Tutup buku bulanan", Keterangan: "Pastikan semua transaksi bulan lalu sudah dicatat dan laporan keuangan bulan lalu ditutup.",
			Frekuensi: "bulanan", Tanggal: 5, HariSebelum: 3, Untuk: UntukSemua, Aktif: true},
		{Judul: "Setor PPh Final UMKM", Keterangan: "Setor PPh Final 0,5% atas omzet bulan lalu.",
			Frekuensi: "bulanan", Tanggal: 15, HariSebelum: 3, Untuk: UntukSemua, Aktif: true, PerluVerifikasi: true},
		{Judul: "Lapor SPT Masa PPh", Keterangan: "Lapor SPT Masa PPh (termasuk potong/pungut) untuk masa pajak bulan lalu.",
			Frekuensi: "bulanan", Tanggal: 20, HariSebelum: 3, Untuk: UntukBadan, Aktif: true, PerluVerifikasi: true},
		{Judul: "Setor & lapor SPT Masa PPN", Keterangan: "Setor PPN kurang bayar lalu lapor SPT Masa PPN masa pajak bulan lalu.",
			Frekuensi: "bulanan", Tanggal: 31, HariSebelum: 5, Untuk: UntukPKP, Aktif: true, PerluVerifikasi: true},
		{Judul: "Tutup buku tahunan", Keterangan: "Tutup buku tahun lalu: cek saldo kas, piutang, utang, stok, dan aset sebelum menyusun SPT Tahunan.",
			Frekuensi: "tahunan", Tanggal: 31, Bulan: 1, HariSebelum: 14, Untuk: UntukSemua, Aktif: true},
		{Judul: "Lapor SPT Tahunan Orang Pribadi", Keterangan: "Batas lapor SPT Tahunan PPh Orang Pribadi tahun pajak lalu.",
			Frekuensi: "tahunan", Tanggal: 31, Bulan: 3, HariSebelum: 14, Untuk: UntukOP, Aktif: true, PerluVerifikasi: true},
		{Judul: "Lapor SPT Tahunan Badan", Keterangan: "Batas lapor SPT Tahunan PPh Badan tahun pajak lalu.",
			Frekuensi: "tahunan", Tanggal: 30, Bulan: 4, HariSebelum: 14, Untuk: UntukBadan, Aktif: true, PerluVerifikasi: true},
	}
	for i := range item {
		item[i].ID = i + 1
	}
	return DaftarJadwal{Item: item, NextID: len(item) + 1}
}

var ErrJadwal = errors.New("jadwal pengingat tidak valid")

// hariDalam: jumlah hari bulan m; tahun kabisat agar 29 Februari tetap boleh dipilih.
func hariDalam(tahun int, m time.Month) int {
	return time.Date(tahun, m+1, 0, 0, 0, 0, 0, WIB).Day()
}

func (j *JadwalPengingat) Validasi() error {
	j.Judul, j.Keterangan = strings.TrimSpace(j.Judul), strings.TrimSpace(j.Keterangan)
	switch {
	case j.Judul == "" || len([]rune(j.Judul)) > 80:
		return fmt.Errorf("%w: judul wajib diisi (maks 80 karakter)", ErrJadwal)
	case len([]rune(j.Keterangan)) > 300:
		return fmt.Errorf("%w: keterangan maks 300 karakter", ErrJadwal)
	case j.Frekuensi != "bulanan" && j.Frekuensi != "tahunan":
		return fmt.Errorf("%w: frekuensi harus bulanan atau tahunan", ErrJadwal)
	case j.Tanggal < 1 || j.Tanggal > 31:
		return fmt.Errorf("%w: tanggal 1–31", ErrJadwal)
	case j.HariSebelum < 0 || j.HariSebelum > 60:
		return fmt.Errorf("%w: mulai diingatkan 0–60 hari sebelumnya", ErrJadwal)
	case !slices.Contains([]string{UntukSemua, UntukOP, UntukBadan, UntukPKP}, j.Untuk):
		return fmt.Errorf("%w: berlaku untuk tidak dikenal", ErrJadwal)
	}
	if j.Frekuensi == "bulanan" {
		j.Bulan = 0
		return nil
	}
	if j.Bulan < 1 || j.Bulan > 12 {
		return fmt.Errorf("%w: bulan 1–12", ErrJadwal)
	}
	if j.Tanggal > hariDalam(2028, time.Month(j.Bulan)) { // 2028 kabisat
		return fmt.Errorf("%w: tanggal %d tidak ada di bulan itu", ErrJadwal, j.Tanggal)
	}
	return nil
}

// Ganti mengganti seluruh daftar (layar Master menyimpan semuanya sekaligus). Item tanpa ID
// diberi ID baru; bila satu item tidak valid, daftar tidak berubah.
func (d *DaftarJadwal) Ganti(baru []JadwalPengingat) error {
	if len(baru) > MaksJadwal {
		return fmt.Errorf("%w: maksimal %d jadwal", ErrJadwal, MaksJadwal)
	}
	out := slices.Clone(baru)
	next := max(d.NextID, 1)
	for i := range out {
		if err := out[i].Validasi(); err != nil {
			return err
		}
		if out[i].ID <= 0 {
			out[i].ID = next
			next++
		}
		next = max(next, out[i].ID+1)
	}
	d.Item, d.NextID = out, next
	return nil
}

// JatuhTempo: tenggat berikutnya yang belum lewat (hari ini terhitung), pukul 00:00 WIB.
func (j JadwalPengingat) JatuhTempo(now time.Time) time.Time {
	now = now.In(WIB)
	hariIni := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, WIB)
	pada := func(tahun int, m time.Month) time.Time {
		return time.Date(tahun, m, min(j.Tanggal, hariDalam(tahun, m)), 0, 0, 0, 0, WIB)
	}
	if j.Frekuensi == "tahunan" {
		t := pada(now.Year(), time.Month(j.Bulan))
		if t.Before(hariIni) {
			t = pada(now.Year()+1, time.Month(j.Bulan))
		}
		return t
	}
	t := pada(now.Year(), now.Month())
	if t.Before(hariIni) {
		n := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, WIB)
		t = pada(n.Year(), n.Month())
	}
	return t
}

func (j JadwalPengingat) berlakuUntuk(p *BusinessProfile) bool {
	switch j.Untuk {
	case UntukSemua:
		return true
	case UntukOP:
		return p != nil && p.BentukUsaha == "op"
	case UntukBadan:
		return p != nil && p.BentukUsaha != "" && p.BentukUsaha != "op"
	case UntukPKP:
		return p != nil && p.PKP
	}
	return false
}

type PengingatJadwal struct {
	ID         int    `json:"id"`
	Judul      string `json:"judul"`
	Keterangan string `json:"keterangan,omitempty"`
	JatuhTempo string `json:"jatuh_tempo"` // YYYY-MM-DD
	SisaHari   int    `json:"sisa_hari"`
}

// Aktif: jadwal yang sedang dalam jendela pengingat untuk usaha ini, jatuh tempo terdekat dulu.
func (d DaftarJadwal) Aktif(p *BusinessProfile, now time.Time) []PengingatJadwal {
	now = now.In(WIB)
	hariIni := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, WIB)
	out := []PengingatJadwal{}
	for _, j := range d.Item {
		if !j.Aktif || !j.berlakuUntuk(p) {
			continue
		}
		jt := j.JatuhTempo(now)
		sisa := int(jt.Sub(hariIni).Hours()/24 + 0.5)
		if sisa > j.HariSebelum {
			continue
		}
		out = append(out, PengingatJadwal{ID: j.ID, Judul: j.Judul, Keterangan: j.Keterangan, JatuhTempo: jt.Format(time.DateOnly), SisaHari: sisa})
	}
	slices.SortStableFunc(out, func(a, b PengingatJadwal) int { return strings.Compare(a.JatuhTempo, b.JatuhTempo) })
	return out
}
