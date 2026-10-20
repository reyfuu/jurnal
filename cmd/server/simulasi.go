package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"time"

	"pajak/internal/finance"
)

// Simulasi usaha di menu Simulasi & Harga: dasar = rata-rata 3 bulan terakhir yang ada
// penjualannya (sampai periode yang dipilih), lalu skenario dari slider dihitung di sini.

const bulanDasarSimulasi = 3

type simulasiResp struct {
	Dasar      finance.DasarSimulasi   `json:"dasar"`
	Hasil      finance.HasilSimulasi   `json:"hasil"`
	TitikImpas finance.TitikImpasUsaha `json:"titik_impas"`
	Produk     []finance.PorsiProduk   `json:"produk"`
}

// hariKalender: jumlah hari dalam bulan "YYYY-MM". Titik impas per hari memakai hari kalender
// (bukan hari yang tercatat ada penjualan): data contoh & usaha yang mencatat rekap mingguan
// akan terlihat buka 4–5 hari saja.
func hariKalender(period string) int64 {
	if d, err := time.Parse("2006-01", period); err == nil {
		return int64(d.AddDate(0, 1, -1).Day())
	}
	return 30
}

func (s *tenant) dasarSimulasi(sampai string) finance.DasarSimulasi {
	var ids []string
	for _, p := range finance.AvailablePeriods() {
		if p.ID <= sampai {
			ids = append(ids, p.ID)
		}
	}
	slices.Sort(ids) // "YYYY-MM": urutan teks = urutan waktu
	var bulan []finance.BulanSimulasi
	for i := len(ids) - 1; i >= 0 && len(bulan) < bulanDasarSimulasi; i-- {
		if inc := s.report(ids[i]).Income; inc.Omzet > 0 {
			bulan = append(bulan, finance.BulanSimulasi{Periode: ids[i], Income: inc, HariBuka: hariKalender(ids[i])})
		}
	}
	slices.Reverse(bulan)
	return finance.RataRataSimulasi(bulan)
}

// produkSimulasi: produk resep (modal = HPP resep) lalu item yang terjual di bulan dasar tanpa
// resep (harga = rata-rata harga jual, modal belum diketahui).
func (s *tenant) produkSimulasi(d finance.DasarSimulasi) []finance.PorsiProduk {
	s.mu.RLock()
	sum := finance.SummarizeProduksi(slices.Clone(s.produk), nil, slices.Clone(s.stok), nil)
	type jual struct{ nominal, qty int64 }
	terjual := map[string]*jual{}
	var urut []string
	for _, p := range d.Periode {
		for _, t := range finance.ForPeriod(s.txs, p) {
			if t.Jenis != "Penjualan" || strings.TrimSpace(t.Item) == "" || t.Qty <= 0 {
				continue
			}
			if terjual[t.Item] == nil {
				terjual[t.Item] = &jual{}
				urut = append(urut, t.Item)
			}
			terjual[t.Item].nominal += t.Nominal
			terjual[t.Item].qty += int64(t.Qty)
		}
	}
	s.mu.RUnlock()
	out := []finance.PorsiProduk{}
	for _, p := range sum.Produk {
		if p.HargaJual <= 0 {
			continue
		}
		row := finance.PorsiBalikModal(d, p.HargaJual, p.HPP.PerUnit)
		row.Nama = p.Nama
		out = append(out, row)
	}
	for _, nama := range urut {
		if slices.ContainsFunc(out, func(p finance.PorsiProduk) bool { return strings.EqualFold(p.Nama, nama) }) {
			continue
		}
		j := terjual[nama]
		out = append(out, finance.PorsiProduk{Nama: nama, Harga: (j.nominal + j.qty/2) / j.qty})
	}
	return out
}

func (s *tenant) handleSimulasi(w http.ResponseWriter, r *http.Request) {
	var sk finance.Skenario
	if err := json.NewDecoder(r.Body).Decode(&sk); err != nil {
		http.Error(w, "skenario tidak valid", http.StatusBadRequest)
		return
	}
	// Batas slider: harga & jualan -90%..+100%, modal -90%..+100%.
	for _, v := range []int64{sk.HargaBps, sk.VolumeBps, sk.ModalBps} {
		if v < -9_000 || v > 10_000 {
			http.Error(w, "perubahan harus antara -90% dan +100%", http.StatusBadRequest)
			return
		}
	}
	d := s.dasarSimulasi(periodOrDefault(r))
	h := finance.Simulasikan(d, sk)
	writeJSON(w, simulasiResp{Dasar: d, Hasil: h, TitikImpas: finance.HitungTitikImpas(h, d.HariBuka), Produk: s.produkSimulasi(d)})
}
