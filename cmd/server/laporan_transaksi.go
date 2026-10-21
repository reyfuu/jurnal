package main

import (
	"cmp"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"pajak/internal/finance"
	"pajak/internal/xlsx"
)

// Laporan Transaksi: daftar transaksi dalam rentang tanggal bebas dengan filter jenis, arah,
// status bayar, channel & kata kunci, plus total dan rekap per jenis. Disaring di server supaya
// tampilan dan ekspor Excel selalu memuat baris yang sama.

type filterTransaksi struct {
	Dari, Sampai, Jenis, Arah, Status, Channel, Cari string
}

// rentangDari: dari/sampai (YYYY-MM-DD) dari query; bila kosong, satu bulan periode laporan.
// Rentang dibatasi 36 bulan supaya permintaan tetap ringan.
func rentangDari(r *http.Request) (dari, sampai string, err error) {
	q := r.URL.Query()
	dari, sampai = q.Get("dari"), q.Get("sampai")
	if dari == "" && sampai == "" {
		p := periodOrDefault(r)
		t, e := time.Parse("2006-01", p)
		if e != nil {
			return "", "", fmt.Errorf("period harus YYYY-MM")
		}
		return t.Format(time.DateOnly), t.AddDate(0, 1, -1).Format(time.DateOnly), nil
	}
	a, e1 := time.Parse(time.DateOnly, dari)
	b, e2 := time.Parse(time.DateOnly, sampai)
	switch {
	case e1 != nil || e2 != nil:
		return "", "", fmt.Errorf("tanggal dari & sampai harus YYYY-MM-DD")
	case b.Before(a):
		return "", "", fmt.Errorf("tanggal sampai tidak boleh sebelum tanggal dari")
	case b.After(a.AddDate(3, 0, 0)):
		return "", "", fmt.Errorf("rentang paling panjang 36 bulan")
	}
	return dari, sampai, nil
}

func (f filterTransaksi) cocok(t finance.Transaction) bool {
	if t.Tanggal < f.Dari || t.Tanggal > f.Sampai {
		return false
	}
	j, _ := finance.LookupJenis(t.Jenis)
	switch {
	case f.Jenis != "" && t.Jenis != f.Jenis,
		f.Arah != "" && j.Arah != f.Arah,
		f.Status == "lunas" && !t.Lunas,
		f.Status == "belum" && t.Lunas,
		f.Channel != "" && t.Channel != f.Channel:
		return false
	}
	if q := strings.ToLower(strings.TrimSpace(f.Cari)); q != "" {
		teks := strings.ToLower(strings.Join([]string{t.Keterangan, t.Item, t.Pihak, t.Channel, t.Jenis}, " "))
		if !strings.Contains(teks, q) {
			return false
		}
	}
	return true
}

type rekapJenis struct {
	Jenis  string `json:"jenis"`
	Arah   string `json:"arah"`
	Jumlah int    `json:"jumlah"`
	Total  int64  `json:"total"`
}

type laporanTransaksiResp struct {
	Dari         string                `json:"dari"`
	Sampai       string                `json:"sampai"`
	Transaksi    []finance.Transaction `json:"transaksi"`
	Jumlah       int                   `json:"jumlah"`
	TotalMasuk   int64                 `json:"total_masuk"`
	TotalKeluar  int64                 `json:"total_keluar"`
	BelumLunas   int64                 `json:"belum_lunas"` // nilai transaksi yang belum dibayar
	PerJenis     []rekapJenis          `json:"per_jenis"`
	Channel      []string              `json:"channel"` // pilihan filter channel
	JenisPilihan []finance.Jenis       `json:"jenis_pilihan"`
}

func (s *tenant) laporanTransaksi(f filterTransaksi) laporanTransaksiResp {
	out := laporanTransaksiResp{Dari: f.Dari, Sampai: f.Sampai, Transaksi: []finance.Transaction{}, PerJenis: []rekapJenis{}, JenisPilihan: finance.JenisList}
	s.mu.RLock()
	defer s.mu.RUnlock()
	per := map[string]*rekapJenis{}
	for _, t := range s.txs {
		if t.Channel != "" && !slices.Contains(out.Channel, t.Channel) {
			out.Channel = append(out.Channel, t.Channel)
		}
		if !f.cocok(t) {
			continue
		}
		j, _ := finance.LookupJenis(t.Jenis)
		out.Transaksi = append(out.Transaksi, t)
		if j.Arah == "masuk" {
			out.TotalMasuk += t.Nominal
		} else {
			out.TotalKeluar += t.Nominal
		}
		if !t.Lunas {
			out.BelumLunas += t.Nominal
		}
		g := per[t.Jenis]
		if g == nil {
			g = &rekapJenis{Jenis: t.Jenis, Arah: j.Arah}
			per[t.Jenis] = g
		}
		g.Jumlah++
		g.Total += t.Nominal
	}
	slices.SortStableFunc(out.Transaksi, func(a, b finance.Transaction) int {
		return cmp.Or(cmp.Compare(a.Tanggal, b.Tanggal), cmp.Compare(a.ID, b.ID))
	})
	for _, g := range per {
		out.PerJenis = append(out.PerJenis, *g)
	}
	slices.SortFunc(out.PerJenis, func(a, b rekapJenis) int { return cmp.Or(cmp.Compare(a.Arah, b.Arah), cmp.Compare(b.Total, a.Total)) })
	slices.Sort(out.Channel)
	out.Jumlah = len(out.Transaksi)
	return out
}

func filterDari(r *http.Request) (filterTransaksi, error) {
	dari, sampai, err := rentangDari(r)
	q := r.URL.Query()
	return filterTransaksi{Dari: dari, Sampai: sampai, Jenis: q.Get("jenis"), Arah: q.Get("arah"), Status: q.Get("status"), Channel: q.Get("channel"), Cari: q.Get("cari")}, err
}

func (s *tenant) handleLaporanTransaksi(w http.ResponseWriter, r *http.Request) {
	f, err := filterDari(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, s.laporanTransaksi(f))
}

func (s *tenant) handleLaporanTransaksiXlsx(w http.ResponseWriter, r *http.Request) {
	f, err := filterDari(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	lt := s.laporanTransaksi(f)
	rows := [][]string{{"Tanggal", "No", "Jenis", "Arah", "Keterangan", "Produk / Bahan", "Pelanggan / Supplier", "Channel", "Status", "Tanggal Lunas", "Masuk (Rp)", "Keluar (Rp)"}}
	for _, t := range lt.Transaksi {
		j, _ := finance.LookupJenis(t.Jenis)
		masuk, keluar := "", ""
		if j.Arah == "masuk" {
			masuk = rp(t.Nominal)
		} else {
			keluar = rp(t.Nominal)
		}
		status := "Lunas"
		if !t.Lunas {
			status = "Belum Lunas"
		}
		rows = append(rows, []string{t.Tanggal, rp(int64(t.ID)), t.Jenis, j.Arah, t.Keterangan, t.Item, t.Pihak, t.Channel, status, t.TanggalLunas, masuk, keluar})
	}
	rows = append(rows, []string{"", "", "", "", "TOTAL", "", "", "", "", "", rp(lt.TotalMasuk), rp(lt.TotalKeluar)})
	rekap := [][]string{{"Jenis", "Arah", "Jumlah transaksi", "Total (Rp)"}}
	for _, g := range lt.PerJenis {
		rekap = append(rekap, []string{g.Jenis, g.Arah, rp(int64(g.Jumlah)), rp(g.Total)})
	}
	rekap = append(rekap, []string{}, []string{"Total masuk", "", "", rp(lt.TotalMasuk)}, []string{"Total keluar", "", "", rp(lt.TotalKeluar)},
		[]string{"Selisih (masuk − keluar)", "", "", rp(lt.TotalMasuk - lt.TotalKeluar)}, []string{"Belum lunas", "", "", rp(lt.BelumLunas)})
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="laporan-transaksi-%s_%s.xlsx"`, f.Dari, f.Sampai))
	_ = xlsx.Write(w, []xlsx.Sheet{{Name: "Transaksi", Rows: rows}, {Name: "Rekap", Rows: rekap}})
}
