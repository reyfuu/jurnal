package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"

	"pajak/internal/finance"
)

// Ekspor Laporan Keuangan: PDF per tab / semua laporan sekaligus, dan perbandingan beberapa
// bulan (JSON untuk layar, PDF untuk diunduh). Semua angka dari t.report, sumber yang sama
// dengan layar Laporan Keuangan.

func (t *tenant) bagianLaporan(rep finance.PeriodReport, bagian, metode string) ([]finance.BagianLaporan, bool, error) {
	if metode != "" && metode != "langsung" && metode != "tidak-langsung" {
		return nil, false, fmt.Errorf("metode arus kas %q tidak dikenal", metode)
	}
	tl := metode == "tidak-langsung"
	switch bagian {
	case "", "semua":
		return []finance.BagianLaporan{finance.BagianLabaRugi(rep.Income), finance.BagianNeraca(rep.BalanceSheet), finance.BagianArusKas(rep.CashFlow, tl)}, false, nil
	case "laba-rugi":
		return []finance.BagianLaporan{finance.BagianLabaRugi(rep.Income)}, false, nil
	case "neraca":
		return []finance.BagianLaporan{finance.BagianNeraca(rep.BalanceSheet)}, false, nil
	case "arus-kas":
		return []finance.BagianLaporan{finance.BagianArusKas(rep.CashFlow, tl)}, false, nil
	case "aset":
		return nil, true, nil
	}
	return nil, false, fmt.Errorf("bagian laporan %q tidak dikenal", bagian)
}

var namaBerkasBagian = map[string]string{"": "Laporan-Keuangan", "semua": "Laporan-Keuangan", "laba-rugi": "Laba-Rugi",
	"neraca": "Neraca", "arus-kas": "Arus-Kas", "aset": "Register-Aset"}

func (t *tenant) handleLaporanPDF(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	period := periodOrDefault(r)
	if !finance.PeriodeSah(period) {
		http.Error(w, "periode tidak dikenal", http.StatusBadRequest)
		return
	}
	rep := t.report(period)
	bagian, aset, err := t.bagianLaporan(rep, q.Get("bagian"), q.Get("metode"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	label, labelNeraca, berkas := finance.LabelPeriode(period), finance.TanggalAkhirPeriode(period), period
	// Rentang tanggal: laba rugi & arus kas dari..sampai; neraca tetap posisi akhir periode.
	dari, sampai, rentang, err := rentangLaporan(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if rentang {
		lr := t.laporanRentang(dari, sampai)
		label, berkas = finance.TanggalPanjang(dari)+" s.d. "+finance.TanggalPanjang(sampai), dari+"_"+sampai
		for i, b := range bagian {
			switch b.Kode {
			case "laba-rugi":
				bagian[i] = finance.BagianLabaRugi(lr.Income)
			case "arus-kas":
				bagian[i] = finance.BagianArusKas(lr.CashFlow, q.Get("metode") == "tidak-langsung")
			}
		}
	}
	// Neraca per tanggal bebas (?tanggal=YYYY-MM-DD), mis. "Per 15 Agustus 2026".
	if tgl := q.Get("tanggal"); tgl != "" {
		bs, err := t.neracaPada(tgl)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for i, b := range bagian {
			if b.Kode == "neraca" {
				bagian[i] = finance.BagianNeraca(bs)
			}
		}
		labelNeraca = finance.TanggalPanjang(tgl)
		if q.Get("bagian") == "neraca" {
			berkas = tgl
		}
	}
	bagian = finance.BeriNamaSemua(bagian, t.namaAkunLaporan())
	var ringkasAset *finance.AssetSummary
	if aset {
		ringkasAset = &rep.Assets
	}
	var buf bytes.Buffer
	if err := finance.RenderLaporanPDF(&buf, t.currentProfile().NamaUsaha, label, labelNeraca, bagian, ringkasAset); err != nil {
		log.Printf("render laporan PDF %s: %v", period, err)
		http.Error(w, "gagal membuat PDF", http.StatusInternalServerError)
		return
	}
	kirimPDF(w, fmt.Sprintf("%s-%s.pdf", namaBerkasBagian[q.Get("bagian")], berkas), buf.Bytes())
}

func kirimPDF(w http.ResponseWriter, nama string, isi []byte) {
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, nama))
	_, _ = w.Write(isi)
}

// banding menyusun perbandingan laba rugi, neraca, dan arus kas (metode langsung) untuk
// bulan-bulan yang dipilih pengguna.
func (t *tenant) banding(w http.ResponseWriter, r *http.Request) (finance.Banding, bool) {
	periode, err := finance.ValidasiPeriodeBanding(r.URL.Query().Get("periode"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return finance.Banding{}, false
	}
	laporan := make([]finance.PeriodeLaporan, 0, len(periode))
	nama := t.namaAkunLaporan()
	for _, p := range periode {
		rep := t.report(p)
		laporan = append(laporan, finance.PeriodeLaporan{Periode: p, Bagian: finance.BeriNamaSemua([]finance.BagianLaporan{
			finance.BagianLabaRugi(rep.Income), finance.BagianNeraca(rep.BalanceSheet), finance.BagianArusKas(rep.CashFlow, false)}, nama)})
	}
	return finance.SusunBanding(laporan), true
}

func (t *tenant) handleBanding(w http.ResponseWriter, r *http.Request) {
	if b, ok := t.banding(w, r); ok {
		writeJSON(w, b)
	}
}

func (t *tenant) handleBandingPDF(w http.ResponseWriter, r *http.Request) {
	b, ok := t.banding(w, r)
	if !ok {
		return
	}
	var buf bytes.Buffer
	if err := finance.RenderBandingPDF(&buf, t.currentProfile().NamaUsaha, b); err != nil {
		log.Printf("render banding PDF: %v", err)
		http.Error(w, "gagal membuat PDF", http.StatusInternalServerError)
		return
	}
	n := len(b.Kolom)
	nama := "Perbandingan-" + b.Kolom[0].Periode
	if n > 1 {
		nama += "-sd-" + b.Kolom[n-1].Periode
	}
	kirimPDF(w, nama+".pdf", buf.Bytes())
}
