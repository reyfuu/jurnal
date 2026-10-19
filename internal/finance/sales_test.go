package finance

import (
	"reflect"
	"testing"
)

func TestSalesReport(t *testing.T) {
	txs := []Transaction{
		{Tanggal: "2026-08-03", Jenis: "Penjualan", Nominal: 300_000, Item: "Kopi Susu", Qty: 15, Channel: "GoFood", Lunas: true},
		{Tanggal: "2026-08-03", Jenis: "Penjualan", Nominal: 100_000, Item: "Croissant", Qty: 4, Channel: "Dine-in", Lunas: true},
		{Tanggal: "2026-08-11", Jenis: "Penjualan", Nominal: 500_000, Item: "Kopi Susu", Qty: 25, Channel: "B2B Tempo", Pihak: "PT Maju", Lunas: false},
		{Tanggal: "2026-08-11", Jenis: "Sewa Tempat", Nominal: 9_000_000, Lunas: true}, // bukan penjualan
	}
	r := SalesReport(txs, "week")
	if r.Summary != (SalesSummary{Omzet: 900_000, Transaksi: 3, RataRata: 300_000, Qty: 44, Piutang: 500_000}) {
		t.Fatalf("summary %+v", r.Summary)
	}
	if want := []SalesPoint{{"2026-W32", 400_000, 2}, {"2026-W33", 500_000, 1}}; !reflect.DeepEqual(r.Trend, want) {
		t.Fatalf("trend %+v", r.Trend)
	}
	if r.TopItems[0] != (SalesItem{Item: "Kopi Susu", Qty: 40, Omzet: 800_000}) {
		t.Fatalf("top item %+v", r.TopItems[0])
	}
	if r.ByCustomer[0].Name != "PT Maju" || r.ByCustomer[1].Name != "Umum" || r.ByCustomer[0].Piutang != 500_000 {
		t.Fatalf("by customer %+v", r.ByCustomer)
	}
	if got := SalesReport(txs, "day").Trend; len(got) != 2 || got[0].Label != "2026-08-03" {
		t.Fatalf("day trend %+v", got)
	}
	if got := SalesReport(txs, "month").Trend; len(got) != 1 || got[0].Label != "2026-08" {
		t.Fatalf("month trend %+v", got)
	}
}
