package finance

import (
	"cmp"
	"errors"
	"fmt"
	"strconv"
	"time"

	"pajak/internal/tax"
)

// Rekonsiliasi bulanan vs tahunan PPh Final 0,5% (PP 55/2022 jo. PP 20/2026). Untuk UMKM,
// setoran bulanan itulah laporan bulanannya (setor = lapor), jadi setiap masa dibandingkan:
// PPh terutang menurut pembukuan saat ini vs yang sudah disetor untuk masa itu. Selisih
// muncul bila setoran keliru atau pembukuan berubah setelah disetor; totalnya menjadi kurang
// / lebih bayar di SPT Tahunan (lampiran L-3B).

const JenisSetorPPhFinal = "Setor PPh Final"

// ToleransiRekon: selisih di bawah ini dianggap pembulatan.
const ToleransiRekon = 100

type RekonBulan struct {
	Masa           string `json:"masa"` // YYYY-MM
	Bulan          string `json:"bulan"`
	Omzet          int64  `json:"omzet"`
	OmzetKumulatif int64  `json:"omzet_kumulatif"`
	DasarPajak     int64  `json:"dasar_pajak"` // omzet kena pajak (setelah kuota bebas OP)
	PPhTerutang    int64  `json:"pph_terutang"`
	PPhDisetor     int64  `json:"pph_disetor"`
	Selisih        int64  `json:"selisih"` // disetor − terutang; negatif = kurang bayar
	JatuhTempo     string `json:"jatuh_tempo"`
	// Status: cocok | kurang | lebih | belum_jatuh_tempo | nihil | berjalan (masa belum berakhir)
	Status string `json:"status"`
	// PajakLain: transaksi "Pajak & Izin" di bulan berikutnya — mungkin setoran PPh Final yang
	// belum ditandai jenisnya.
	PajakLain   int64 `json:"pajak_lain,omitempty"`
	PPNKeluaran int64 `json:"ppn_keluaran,omitempty"`
	PPNDisetor  int64 `json:"ppn_disetor,omitempty"`
}

type Rekonsiliasi struct {
	Tahun         int          `json:"tahun"`
	Subjek        string       `json:"subjek"` // OP | Badan
	BebasOP       int64        `json:"bebas_op"`
	TarifBps      int64        `json:"tarif_bps"`
	PKP           bool         `json:"pkp"`
	Bulan         []RekonBulan `json:"bulan"`
	TotalOmzet    int64        `json:"total_omzet"`
	TotalTerutang int64        `json:"total_terutang"`
	TotalDisetor  int64        `json:"total_disetor"`
	// KurangBayar: terutang − disetor untuk masa yang sudah lewat jatuh tempo (negatif = lebih
	// bayar). Selesai: semua masa tahun itu sudah lewat jatuh tempo, jadi KurangBayar adalah
	// angka kurang / lebih bayar SPT Tahunan.
	KurangBayar int64    `json:"kurang_bayar"`
	Selesai     bool     `json:"selesai"`
	PerluDicek  int      `json:"perlu_dicek"` // masa berstatus kurang / lebih
	Peringatan  []string `json:"peringatan,omitempty"`
}

// SubjekPPhFinal: orang pribadi mendapat kuota bebas Rp500 jt; PT perorangan, CV, dan PT tidak.
func SubjekPPhFinal(p BusinessProfile) string {
	if p.BentukUsaha == "" || p.BentukUsaha == "op" {
		return "OP"
	}
	return "Badan"
}

// MasaSetor: masa yang dilunasi setoran (data lama tanpa masa = bulan sebelum tanggal setor).
func MasaSetor(t Transaction) string {
	return cmp.Or(t.MasaPajak, bulanGeser(t.Tanggal, -1))
}

// bulanGeser: "2026-07-10" / "2026-07" digeser n bulan → "YYYY-MM"; "" bila tidak sah.
func bulanGeser(tanggal string, n int) string {
	if len(tanggal) < 7 {
		return ""
	}
	d, err := time.Parse("2006-01", tanggal[:7])
	if err != nil {
		return ""
	}
	return d.AddDate(0, n, 0).Format("2006-01")
}

// rapikanMasa: masa hanya untuk setoran pajak, bawaan bulan sebelumnya, tidak boleh sesudah
// bulan setor.
func rapikanMasa(t *Transaction) error {
	if t.Jenis != JenisSetorPPhFinal && t.Jenis != "Setor PPN" {
		t.MasaPajak = ""
		return nil
	}
	if t.MasaPajak == "" {
		t.MasaPajak = bulanGeser(t.Tanggal, -1)
		return nil
	}
	if _, err := time.Parse("2006-01", t.MasaPajak); err != nil || len(t.MasaPajak) != 7 {
		return errors.New("masa pajak harus berformat YYYY-MM")
	}
	if t.MasaPajak > t.Tanggal[:7] {
		return errors.New("masa pajak tidak boleh sesudah bulan setor")
	}
	return nil
}

// RekonsiliasiTahun menyusun rekonsiliasi setahun. omzet: omzet pembukuan per bulan (Jan–Des);
// hariIni (YYYY-MM-DD) menentukan masa yang sudah berakhir dan jatuh tempo (tgl 15 bulan berikut).
func RekonsiliasiTahun(tahun int, omzet [12]int64, txs []Transaction, p BusinessProfile, hariIni string) Rekonsiliasi {
	subjek := SubjekPPhFinal(p)
	hitung := CalculateUMKMTax(subjek, omzet[:])
	pr := tax.P()
	r := Rekonsiliasi{Tahun: tahun, Subjek: subjek, BebasOP: hitung.YearlyExemption, TarifBps: pr.UMKMTarifBps, PKP: p.PKP,
		TotalOmzet: hitung.TotalGross, TotalTerutang: hitung.TotalTaxDue}
	idx := map[string]int{}
	for i, m := range hitung.Months {
		masa := fmt.Sprintf("%04d-%02d", tahun, i+1)
		idx[masa] = i
		r.Bulan = append(r.Bulan, RekonBulan{Masa: masa, Bulan: m.MonthName, Omzet: m.GrossOmzet, OmzetKumulatif: m.CumulativeOmzet,
			DasarPajak: m.TaxableAmount, PPhTerutang: m.TaxDue, JatuhTempo: bulanGeser(masa, 1) + "-15"})
	}
	tambah := func(masa string, f func(*RekonBulan)) {
		if i, ok := idx[masa]; ok {
			f(&r.Bulan[i])
		}
	}
	for _, t := range txs {
		switch t.Jenis {
		case JenisSetorPPhFinal:
			tambah(MasaSetor(t), func(b *RekonBulan) { b.PPhDisetor += t.Nominal })
		case "Setor PPN":
			tambah(MasaSetor(t), func(b *RekonBulan) { b.PPNDisetor += t.Nominal })
		case "PPN Keluaran":
			tambah(bulanGeser(t.Tanggal, 0), func(b *RekonBulan) { b.PPNKeluaran += t.Nominal })
		case "Pajak & Izin":
			tambah(bulanGeser(t.Tanggal, -1), func(b *RekonBulan) { b.PajakLain += t.Nominal })
		}
	}
	for i := range r.Bulan {
		b := &r.Bulan[i]
		b.Selisih = b.PPhDisetor - b.PPhTerutang
		r.TotalDisetor += b.PPhDisetor
		switch {
		case b.Masa >= hariIni[:7]:
			b.Status = "berjalan"
		case b.PPhTerutang == 0 && b.PPhDisetor == 0:
			b.Status = "nihil"
		case abs64(b.Selisih) < ToleransiRekon:
			b.Status = "cocok"
		case b.Selisih > 0:
			b.Status = "lebih"
		case hariIni <= b.JatuhTempo:
			b.Status = "belum_jatuh_tempo"
		default:
			b.Status = "kurang"
		}
		switch b.Status {
		case "kurang", "lebih":
			r.PerluDicek++
			fallthrough
		case "cocok", "nihil":
			r.KurangBayar -= b.Selisih
		}
	}
	r.Selesai = hariIni > r.Bulan[11].JatuhTempo
	if p.BentukUsaha == "cv" || p.BentukUsaha == "pt" {
		r.Peringatan = append(r.Peringatan, "CV, firma, dan PT (selain PT Perorangan) hanya boleh memakai PPh Final 0,5% sampai jangka waktu menurut PP 55/2022 berakhir (CV dan firma 4 tahun pajak, PT 3 tahun pajak sejak terdaftar; PP 20/2026). Sesudahnya PPh dihitung dengan tarif umum, jadi angka di sini tidak berlaku.")
	}
	if r.TotalOmzet > pr.BatasOmzetUMKM {
		r.Peringatan = append(r.Peringatan, fmt.Sprintf("Omzet %d melewati Rp4,8 miliar. Tahun ini tetap dikenai PPh Final 0,5%%, tetapi mulai %d usaha ini tidak bisa lagi memakainya (PP 20/2026 Pasal 57 ayat (4)).", tahun, tahun+1))
	}
	return r
}

// LabelStatusRekon: teks status untuk layar, Excel, dan PDF.
var LabelStatusRekon = map[string]string{
	"cocok": "Cocok", "kurang": "Kurang bayar", "lebih": "Lebih bayar",
	"belum_jatuh_tempo": "Belum jatuh tempo", "nihil": "Nihil", "berjalan": "Masa berjalan",
}

// Daftar: rekonsiliasi sebagai tabel (unduhan Excel / PDF), ditutup baris setahun.
func (r Rekonsiliasi) Daftar() Daftar {
	n := func(v int64) string { return strconv.FormatInt(v, 10) }
	d := Daftar{Judul: fmt.Sprintf("Rekonsiliasi PPh Final %d", r.Tahun),
		Kolom:  []string{"Masa", "Omzet", "Omzet kumulatif", "Dasar kena pajak", "PPh terutang", "PPh disetor", "Selisih", "Jatuh tempo", "Status"},
		Rupiah: []int{1, 2, 3, 4, 5, 6}}
	if r.PKP {
		d.Kolom = append(d.Kolom, "PPN Keluaran", "PPN disetor")
		d.Rupiah = append(d.Rupiah, 9, 10)
	}
	for _, b := range r.Bulan {
		row := []string{b.Bulan + " " + b.Masa[:4], n(b.Omzet), n(b.OmzetKumulatif), n(b.DasarPajak), n(b.PPhTerutang), n(b.PPhDisetor), n(b.Selisih), b.JatuhTempo, LabelStatusRekon[b.Status]}
		if r.PKP {
			row = append(row, n(b.PPNKeluaran), n(b.PPNDisetor))
		}
		d.Baris = append(d.Baris, row)
	}
	akhir := "Lebih bayar"
	if r.KurangBayar > 0 {
		akhir = "Kurang bayar"
	} else if r.KurangBayar == 0 {
		akhir = "Cocok"
	}
	if r.Selesai {
		akhir += " (SPT Tahunan)"
	} else {
		akhir += " s.d. masa yang jatuh tempo"
	}
	total := []string{fmt.Sprintf("Setahun %d", r.Tahun), n(r.TotalOmzet), n(r.TotalOmzet), "", n(r.TotalTerutang), n(r.TotalDisetor), n(-r.KurangBayar), "", akhir}
	if r.PKP {
		var k, s int64
		for _, b := range r.Bulan {
			k, s = k+b.PPNKeluaran, s+b.PPNDisetor
		}
		total = append(total, n(k), n(s))
	}
	d.Baris = append(d.Baris, total)
	return d
}
