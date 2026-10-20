package finance

import (
	"errors"
	"slices"
	"strings"
	"time"
)

// Catatan konsultan: tim Master Pajak menulis catatan pada halaman & periode laporan klien
// (mis. "saldo kas tidak cocok dengan rekening koran"). Klien menandai sudah diperbaiki; catatan
// terbuka diingatkan lagi sesuai Pengingat (pengaturan Master, berlaku untuk semua klien).

const (
	TingkatInfo     = "info"
	TingkatPerbaiki = "perbaiki"
	TingkatPenting  = "penting"

	CatatanTerbuka = "terbuka"
	CatatanSelesai = "selesai"
)

var tingkatUrut = []string{TingkatInfo, TingkatPerbaiki, TingkatPenting}

type CatatanKonsultan struct {
	ID          int       `json:"id"`
	Halaman     string    `json:"halaman"`           // view aplikasi, mis. laporan-keuangan
	Periode     string    `json:"periode,omitempty"` // YYYY-MM yang sedang dibuka
	Tingkat     string    `json:"tingkat"`
	Isi         string    `json:"isi"`
	TransaksiID int       `json:"transaksi_id,omitempty"`
	OlehID      int       `json:"oleh_id"`
	Oleh        string    `json:"oleh"`
	Peran       string    `json:"peran,omitempty"`
	Dibuat      time.Time `json:"dibuat"`
	Status      string    `json:"status"`
	SelesaiOleh string    `json:"selesai_oleh,omitempty"`
	SelesaiPada time.Time `json:"selesai_pada,omitzero"`
	// DiingatkanPada: terakhir pengingat ditampilkan ke klien; JumlahPengingat: berapa kali.
	DiingatkanPada  time.Time `json:"diingatkan_pada,omitzero"`
	JumlahPengingat int       `json:"jumlah_pengingat"`
}

// Pengingat: aturan pengingat catatan terbuka, diatur di Panel Master.
type Pengingat struct {
	Aktif        bool   `json:"aktif"`
	IntervalHari int    `json:"interval_hari"` // catatan terbuka diingatkan lagi setiap N hari
	TingkatMin   string `json:"tingkat_min"`   // tingkat terendah yang diingatkan
	// Kanal: "aplikasi" sekarang; "whatsapp"/"email" disiapkan untuk nanti.
	Kanal []string `json:"kanal"`
}

func PengingatBawaan() Pengingat {
	return Pengingat{Aktif: true, IntervalHari: 3, TingkatMin: TingkatPerbaiki, Kanal: []string{"aplikasi"}}
}

var ErrCatatan = errors.New("catatan tidak valid")

func (p *Pengingat) Validasi() error {
	switch {
	case p.IntervalHari < 1 || p.IntervalHari > 30:
		return errors.New("frekuensi pengingat harus 1–30 hari")
	case !slices.Contains(tingkatUrut, p.TingkatMin):
		return errors.New("tingkat minimal tidak dikenal")
	}
	p.Kanal = []string{"aplikasi"} // ponytail: WA & email belum tersambung
	return nil
}

// RapikanCatatan memvalidasi isian dari konsultan.
func RapikanCatatan(c *CatatanKonsultan) error {
	c.Isi, c.Halaman, c.Periode = strings.TrimSpace(c.Isi), strings.TrimSpace(c.Halaman), strings.TrimSpace(c.Periode)
	switch {
	case c.Isi == "" || len([]rune(c.Isi)) > 2000:
		return errors.Join(ErrCatatan, errors.New("isi catatan wajib (maks 2.000 huruf)"))
	case !slices.Contains(tingkatUrut, c.Tingkat):
		return errors.Join(ErrCatatan, errors.New("tingkat harus info, perbaiki, atau penting"))
	case c.TransaksiID < 0:
		return ErrCatatan
	}
	return nil
}

// PerluDiingatkan: catatan terbuka, setingkat atau di atas tingkat minimal, dan sudah lewat
// IntervalHari sejak dibuat / terakhir diingatkan.
func (c CatatanKonsultan) PerluDiingatkan(p Pengingat, now time.Time) bool {
	if !p.Aktif || c.Status != CatatanTerbuka || slices.Index(tingkatUrut, c.Tingkat) < slices.Index(tingkatUrut, p.TingkatMin) {
		return false
	}
	acuan := c.Dibuat
	if c.DiingatkanPada.After(acuan) {
		acuan = c.DiingatkanPada
	}
	return !now.Before(acuan.Add(time.Duration(p.IntervalHari) * 24 * time.Hour))
}
