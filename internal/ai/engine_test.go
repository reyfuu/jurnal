package ai

import (
	"context"
	"strings"
	"testing"

	"pajak/internal/finance"
)

func TestAnswer_SummarizeReport(t *testing.T) {
	snap := finance.SeedKedaiKopiSenja("2026-08")
	ledger := finance.SeedLedger("2026-08")
	engine := NewEngine()

	resp := engine.Answer(context.Background(), QueryRequest{
		Query:  "Ringkas laporan bulan ini",
		Period: "2026-08",
	}, snap, ledger)

	if !strings.Contains(resp.Answer, "92.000.000") && !strings.Contains(resp.Answer, "92 jt") && !strings.Contains(resp.Answer, "92,0") {
		t.Fatalf("expected omzet in answer, got: %s", resp.Answer)
	}
	if !strings.Contains(resp.Answer, "25.000.000") && !strings.Contains(resp.Answer, "25 jt") && !strings.Contains(resp.Answer, "25,0") {
		t.Fatalf("expected laba bersih in answer, got: %s", resp.Answer)
	}
	if len(resp.ActionSuggestions) == 0 {
		t.Fatal("expected action suggestions")
	}
}

func TestAnswer_TaxPP55Rules(t *testing.T) {
	snap := finance.SeedKedaiKopiSenja("2026-08")
	ledger := finance.SeedLedger("2026-08")
	engine := NewEngine()

	resp := engine.Answer(context.Background(), QueryRequest{
		Query: "Bagaimana aturan pajak UMKM PP 55 tahun 2022?",
	}, snap, ledger)

	lower := strings.ToLower(resp.Answer)
	if !strings.Contains(lower, "500") || !strings.Contains(lower, "0,5%") && !strings.Contains(lower, "0.5%") {
		t.Fatalf("expected 500jt threshold and 0.5 percent in tax answer, got: %s", resp.Answer)
	}
	if !strings.Contains(lower, "orang pribadi") {
		t.Fatalf("expected distinction for orang pribadi, got: %s", resp.Answer)
	}
}

func TestAnswer_ReceivablesAndCashflow(t *testing.T) {
	snap := finance.SeedKedaiKopiSenja("2026-08")
	ledger := finance.SeedLedger("2026-08")
	engine := NewEngine()

	resp := engine.Answer(context.Background(), QueryRequest{
		Query: "Bagaimana kondisi arus kas dan piutang saya?",
	}, snap, ledger)

	if !strings.Contains(resp.Answer, "97.000.000") && !strings.Contains(resp.Answer, "97 jt") && !strings.Contains(resp.Answer, "97,0") {
		t.Fatalf("expected saldo kas 97jt in answer, got: %s", resp.Answer)
	}
	if !strings.Contains(strings.ToLower(resp.Answer), "shopeepay") && !strings.Contains(strings.ToLower(resp.Answer), "piutang") {
		t.Fatalf("expected mention of piutang, got: %s", resp.Answer)
	}
}


func TestAnswer_NoEmoji(t *testing.T) {
	engine := NewEngine()
	snap := finance.SeedKedaiKopiSenja("2026-08")
	ledger := finance.SeedLedger("2026-08")
	for _, q := range []string{"Ringkas laporan bulan ini", "aturan pajak UMKM", "arus kas dan piutang", "simulasi harga GoFood", "halo"} {
		resp := engine.Answer(context.Background(), QueryRequest{Query: q, Period: "2026-08"}, snap, ledger)
		for _, s := range append([]string{resp.Answer}, resp.ActionSuggestions...) {
			for _, r := range s {
				if (r >= 0x1F300 && r <= 0x1FAFF) || (r >= 0x2600 && r <= 0x27BF) {
					t.Fatalf("query %q: emoji %q in %q", q, r, s)
				}
			}
		}
	}
}
