package main

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"pajak/internal/finance"
	"pajak/internal/xlsx"
)

// Rekonsiliasi bulanan vs tahunan PPh Final (fitur Pro); lihat finance.RekonsiliasiTahun.

func (t *tenant) rekonsiliasi(w http.ResponseWriter, r *http.Request) (finance.Rekonsiliasi, bool) {
	now := time.Now().In(finance.WIB)
	tahun := now.Year()
	if q := r.URL.Query().Get("tahun"); q != "" {
		n, err := strconv.Atoi(q)
		if err != nil || n < 2000 || n > now.Year()+1 {
			http.Error(w, "tahun tidak valid", http.StatusBadRequest)
			return finance.Rekonsiliasi{}, false
		}
		tahun = n
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return finance.RekonsiliasiTahun(tahun, t.omzetTahunLocked(tahun, 12), t.txs, t.profileLocked(), now.Format(time.DateOnly)), true
}

// omzetTahunLocked: omzet pembukuan (sama dengan Laba Rugi) bulan 1..n tahun itu. s.mu dipegang.
func (t *tenant) omzetTahunLocked(tahun, n int) [12]int64 {
	var omzet [12]int64
	for i := range n {
		omzet[i] = t.reportLocked(fmt.Sprintf("%04d-%02d", tahun, i+1)).Income.Omzet
	}
	return omzet
}

func (t *tenant) profileLocked() finance.BusinessProfile {
	if t.profile == nil {
		return finance.BusinessProfile{}
	}
	return *t.profile
}

// cekPPhFinalLocked: butir checklist Tutup Buku untuk PPh Final masa period (paket Pro). s.mu dipegang.
func (t *tenant) cekPPhFinalLocked(period string) cekTutup {
	tahun, _ := strconv.Atoi(period[:4])
	rk := finance.RekonsiliasiTahun(tahun, t.omzetTahunLocked(tahun, 12), t.txs, t.profileLocked(), time.Now().In(finance.WIB).Format(time.DateOnly))
	bulan, _ := strconv.Atoi(period[5:7])
	b := rk.Bulan[bulan-1]
	rp := finance.FormatRupiahPenuh
	c := cekTutup{Kode: "pph_final", Judul: "PPh Final masa ini disetor", View: "pajak-umkm", Status: "ok",
		Detail: fmt.Sprintf("Terutang %s, disetor %s.", rp(b.PPhTerutang), rp(b.PPhDisetor))}
	switch b.Status {
	case "kurang":
		c.Status, c.Detail = "peringatan", fmt.Sprintf("Kurang bayar %s (terutang %s, disetor %s). Setor dengan kode billing 411128-420 lalu catat sebagai Setor PPh Final.", rp(-b.Selisih), rp(b.PPhTerutang), rp(b.PPhDisetor))
	case "lebih":
		c.Status, c.Detail = "peringatan", fmt.Sprintf("Lebih bayar %s (terutang %s, disetor %s). Periksa masa pajak setorannya.", rp(b.Selisih), rp(b.PPhTerutang), rp(b.PPhDisetor))
	case "belum_jatuh_tempo", "berjalan":
		if b.PPhTerutang > b.PPhDisetor {
			c.Status, c.Detail = "info", fmt.Sprintf("Terutang %s, disetor %s. Jatuh tempo %s.", rp(b.PPhTerutang), rp(b.PPhDisetor), finance.FormatTanggalID(b.JatuhTempo))
		}
	}
	return c
}

// pajakUMKMLocked: rekap PPh Final Januari sampai bulan period dari pembukuan (dashboard,
// halaman Pajak UMKM, fakta AI). s.mu dipegang.
func (t *tenant) pajakUMKMLocked(period string) finance.UMKMTaxReport {
	subjek := finance.SubjekPPhFinal(t.profileLocked())
	if !finance.PeriodeSah(period) {
		return finance.CalculateUMKMTax(subjek, nil)
	}
	tahun, _ := strconv.Atoi(period[:4])
	bulan, _ := strconv.Atoi(period[5:7])
	omzet := t.omzetTahunLocked(tahun, bulan)
	return finance.CalculateUMKMTax(subjek, omzet[:bulan])
}

func (t *tenant) handleRekonsiliasi(w http.ResponseWriter, r *http.Request) {
	if rk, ok := t.rekonsiliasi(w, r); ok {
		writeJSON(w, rk)
	}
}

func (t *tenant) handleRekonsiliasiXlsx(w http.ResponseWriter, r *http.Request) {
	rk, ok := t.rekonsiliasi(w, r)
	if !ok {
		return
	}
	d := rk.Daftar()
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="rekonsiliasi-pph-final-%d.xlsx"`, rk.Tahun))
	_ = xlsx.Write(w, []xlsx.Sheet{{Name: "Rekonsiliasi", Rows: append([][]string{d.Kolom}, d.Baris...)}})
}

func (t *tenant) handleRekonsiliasiPDF(w http.ResponseWriter, r *http.Request) {
	rk, ok := t.rekonsiliasi(w, r)
	if !ok {
		return
	}
	sub := "PPh Final UMKM (PP 55/2022 jo. PP 20/2026), subjek " + rk.Subjek + ". Setoran per masa vs pembukuan. Dalam Rupiah."
	var buf bytes.Buffer
	if err := finance.RenderDaftarPDF(&buf, t.currentProfile().NamaUsaha, sub, []finance.Daftar{rk.Daftar()}); err != nil {
		http.Error(w, "gagal menyusun PDF", http.StatusInternalServerError)
		return
	}
	kirimPDF(w, fmt.Sprintf("rekonsiliasi-pph-final-%d.pdf", rk.Tahun), buf.Bytes())
}
