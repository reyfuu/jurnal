package finance

import (
	"errors"
	"strings"
	"time"

	"pajak/internal/tax"
)

// Master karyawan: data dasar untuk onboarding dan rekap beban gaji.
//
// ponytail: belum menghitung PPh 21 maupun membuat transaksi gaji otomatis. Status PTKP
// sudah disimpan supaya kalkulator PPh 21 (Fase D) tinggal membacanya.

type Karyawan struct {
	ID           int    `json:"id"`
	Nama         string `json:"nama"`
	Jabatan      string `json:"jabatan"`
	TanggalMasuk string `json:"tanggal_masuk"` // YYYY-MM-DD
	GajiPokok    int64  `json:"gaji_pokok"`
	Tunjangan    int64  `json:"tunjangan"`
	PTKP         string `json:"ptkp"` // status PTKP, mis. "K/1"
	NPWP         string `json:"npwp"`
	Aktif        bool   `json:"aktif"`
}

// PenghasilanBulanan = gaji pokok + tunjangan tetap.
func (k Karyawan) PenghasilanBulanan() int64 { return k.GajiPokok + k.Tunjangan }

func ValidateKaryawan(k *Karyawan) error {
	for _, s := range []*string{&k.Nama, &k.Jabatan, &k.TanggalMasuk, &k.PTKP, &k.NPWP} {
		*s = strings.TrimSpace(*s)
	}
	if k.Nama == "" {
		return errors.New("nama karyawan wajib diisi")
	}
	if _, err := time.Parse(time.DateOnly, k.TanggalMasuk); err != nil {
		return errors.New("tanggal masuk wajib diisi (format YYYY-MM-DD)")
	}
	if k.GajiPokok <= 0 {
		return errors.New("gaji pokok harus lebih dari 0")
	}
	if k.Tunjangan < 0 {
		return errors.New("tunjangan tidak boleh negatif")
	}
	if k.PTKP == "" {
		k.PTKP = "TK/0"
	}
	if _, ok := tax.PTKP[k.PTKP]; !ok {
		return errors.New("status PTKP tidak dikenal (mis. TK/0, K/1)")
	}
	return nil
}

// KaryawanSummary: rekap yang dipakai halaman Karyawan.
type KaryawanSummary struct {
	Karyawan   []Karyawan `json:"karyawan"`
	PTKPCodes  []string   `json:"ptkp_codes"`
	TotalAktif int        `json:"total_aktif"`
	// BebanGajiBulanan = total penghasilan bulanan karyawan aktif.
	BebanGajiBulanan int64 `json:"beban_gaji_bulanan"`
}

func SummarizeKaryawan(list []Karyawan) KaryawanSummary {
	s := KaryawanSummary{Karyawan: list, PTKPCodes: tax.PTKPCodes}
	for _, k := range list {
		if k.Aktif {
			s.TotalAktif++
			s.BebanGajiBulanan += k.PenghasilanBulanan()
		}
	}
	return s
}
