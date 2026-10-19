package finance

import (
	"cmp"
	"fmt"
	"slices"
	"time"
)

// Laporan Penjualan: pola halaman laporan skinku-b2b-portal (ringkasan, tren, top produk, per mitra),
// dihitung dari transaksi jenis Penjualan.

type SalesSummary struct {
	Omzet     int64 `json:"omzet"`
	Transaksi int   `json:"transaksi"`
	RataRata  int64 `json:"rata_rata"`
	Qty       int   `json:"qty"`
	Piutang   int64 `json:"piutang"` // penjualan belum lunas
}

type SalesPoint struct {
	Label string `json:"label"`
	Omzet int64  `json:"omzet"`
	Count int    `json:"count"`
}

type SalesItem struct {
	Item  string `json:"item"`
	Qty   int    `json:"qty"`
	Omzet int64  `json:"omzet"`
}

type SalesGroup struct {
	Name    string `json:"name"`
	Omzet   int64  `json:"omzet"`
	Count   int    `json:"count"`
	Piutang int64  `json:"piutang"`
}

type SalesReportResult struct {
	Granularity string       `json:"granularity"`
	Summary     SalesSummary `json:"summary"`
	Trend       []SalesPoint `json:"trend"`
	TopItems    []SalesItem  `json:"top_items"`
	ByChannel   []SalesGroup `json:"by_channel"`
	ByCustomer  []SalesGroup `json:"by_customer"`
}

func bucket(tanggal, granularity string) string {
	switch granularity {
	case "month":
		return tanggal[:7]
	case "week":
		d, _ := time.Parse(time.DateOnly, tanggal)
		y, w := d.ISOWeek()
		return fmt.Sprintf("%d-W%02d", y, w)
	}
	return tanggal
}

// group menjumlah per kunci; urutan hasil mengikuti less.
func group[T any](keys []string, m map[string]*T, less func(a, b *T) int) []T {
	out := make([]T, 0, len(keys))
	for _, k := range keys {
		out = append(out, *m[k])
	}
	slices.SortStableFunc(out, func(a, b T) int { return less(&a, &b) })
	return out
}

func SalesReport(txs []Transaction, granularity string) SalesReportResult {
	if granularity != "week" && granularity != "month" {
		granularity = "day"
	}
	r := SalesReportResult{Granularity: granularity}
	trend, items, channels, customers := map[string]*SalesPoint{}, map[string]*SalesItem{}, map[string]*SalesGroup{}, map[string]*SalesGroup{}
	var trendK, itemK, chanK, custK []string
	add := func(m map[string]*SalesGroup, keys *[]string, name string, t Transaction) {
		g, ok := m[name]
		if !ok {
			g = &SalesGroup{Name: name}
			m[name] = g
			*keys = append(*keys, name)
		}
		g.Omzet += t.Nominal
		g.Count++
		if !t.Lunas {
			g.Piutang += t.Nominal
		}
	}
	for _, t := range txs {
		if t.Jenis != "Penjualan" {
			continue
		}
		r.Summary.Omzet += t.Nominal
		r.Summary.Transaksi++
		r.Summary.Qty += t.Qty
		if !t.Lunas {
			r.Summary.Piutang += t.Nominal
		}
		b := bucket(t.Tanggal, granularity)
		if _, ok := trend[b]; !ok {
			trend[b] = &SalesPoint{Label: b}
			trendK = append(trendK, b)
		}
		trend[b].Omzet += t.Nominal
		trend[b].Count++
		item := cmp.Or(t.Item, "Tanpa nama item")
		if _, ok := items[item]; !ok {
			items[item] = &SalesItem{Item: item}
			itemK = append(itemK, item)
		}
		items[item].Qty += t.Qty
		items[item].Omzet += t.Nominal
		add(channels, &chanK, cmp.Or(t.Channel, "Lainnya"), t)
		add(customers, &custK, cmp.Or(t.Pihak, "Umum"), t)
	}
	if r.Summary.Transaksi > 0 {
		r.Summary.RataRata = r.Summary.Omzet / int64(r.Summary.Transaksi)
	}
	r.Trend = group(trendK, trend, func(a, b *SalesPoint) int { return cmp.Compare(a.Label, b.Label) })
	r.TopItems = group(itemK, items, func(a, b *SalesItem) int { return cmp.Compare(b.Omzet, a.Omzet) })
	byOmzet := func(a, b *SalesGroup) int { return cmp.Compare(b.Omzet, a.Omzet) }
	r.ByChannel = group(chanK, channels, byOmzet)
	r.ByCustomer = group(custK, customers, byOmzet)
	r.TopItems = r.TopItems[:min(10, len(r.TopItems))]
	r.ByCustomer = r.ByCustomer[:min(10, len(r.ByCustomer))]
	return r
}
