package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"pajak/internal/finance"
)

type txList struct {
	Transactions []finance.Transaction `json:"transactions"`
	Jenis        []finance.Jenis       `json:"jenis"`
	CanEdit      bool                  `json:"can_edit"`
}

func omzetAug(t *testing.T, srv *Server, c *http.Cookie) int64 {
	t.Helper()
	var is finance.IncomeStatement
	_ = json.NewDecoder(do(srv, c, http.MethodGet, "/api/v1/reports/financial?period=2026-08", "").Body).Decode(&is)
	return is.Omzet
}

func TestTransactionsCRUD_UpdatesReportsAndDashboard(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")

	var list txList
	_ = json.NewDecoder(do(srv, owner, http.MethodGet, "/api/v1/transactions", "").Body).Decode(&list)
	if len(list.Transactions) == 0 || len(list.Jenis) != 18 || !list.CanEdit {
		t.Fatalf("list: %d txs, %d jenis, can_edit=%v", len(list.Transactions), len(list.Jenis), list.CanEdit)
	}
	base := omzetAug(t, srv, owner)

	rec := do(srv, owner, http.MethodPost, "/api/v1/transactions",
		`{"tanggal":"2026-08-30","jenis":"Penjualan","nominal":5000000,"keterangan":"Paket event","channel":"Dine-in","lunas":true}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	var created finance.Transaction
	_ = json.NewDecoder(rec.Body).Decode(&created)
	if got := omzetAug(t, srv, owner); got != base+5_000_000 {
		t.Fatalf("omzet after create: want %d, got %d", base+5_000_000, got)
	}
	var snap finance.DashboardSnapshot
	_ = json.NewDecoder(do(srv, owner, http.MethodGet, "/api/v1/dashboard?period=2026-08", "").Body).Decode(&snap)
	if snap.KPIs.Omzet.Value != base+5_000_000 || snap.RecentTransactions[0].Title != "Paket event" {
		t.Fatalf("dashboard not recomputed: omzet=%d recent=%+v", snap.KPIs.Omzet.Value, snap.RecentTransactions[0])
	}

	path := "/api/v1/transactions/" + strconv.Itoa(created.ID)
	rec = do(srv, owner, http.MethodPut, path,
		`{"tanggal":"2026-08-30","jenis":"Penjualan","nominal":7000000,"keterangan":"Paket event","lunas":false}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec.Code, rec.Body)
	}
	var cf finance.CashFlow
	_ = json.NewDecoder(do(srv, owner, http.MethodGet, "/api/v1/reports/cashflow?period=2026-08", "").Body).Decode(&cf)
	if omzetAug(t, srv, owner) != base+7_000_000 || cf.NewReceivables < 7_000_000 {
		t.Fatalf("update not reflected: omzet=%d receivables=%d", omzetAug(t, srv, owner), cf.NewReceivables)
	}

	if rec := do(srv, owner, http.MethodDelete, path, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}
	if got := omzetAug(t, srv, owner); got != base {
		t.Fatalf("omzet after delete: want %d, got %d", base, got)
	}
	if rec := do(srv, owner, http.MethodDelete, path, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("delete twice: want 404, got %d", rec.Code)
	}
}

func TestTransactions_Validation(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	for _, body := range []string{
		`{"tanggal":"","jenis":"Penjualan","nominal":1000,"lunas":true}`,
		`{"tanggal":"2026-08-01","jenis":"Sedekah","nominal":1000,"lunas":true}`,
		`{"tanggal":"2026-08-01","jenis":"Penjualan","nominal":0,"lunas":true}`,
		`not json`,
	} {
		if rec := do(srv, owner, http.MethodPost, "/api/v1/transactions", body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: want 400, got %d", body, rec.Code)
		}
	}
}

func TestTransactions_KasirCanOnlyCreate(t *testing.T) {
	srv := NewServer()
	kasir := login(t, srv, "kasir@kedaikopisenja.id")

	var list txList
	_ = json.NewDecoder(do(srv, kasir, http.MethodGet, "/api/v1/transactions", "").Body).Decode(&list)
	if list.CanEdit {
		t.Fatal("kasir must not get can_edit")
	}
	if rec := do(srv, kasir, http.MethodPost, "/api/v1/transactions", `{"tanggal":"2026-08-21","jenis":"Penjualan","nominal":250000,"lunas":true}`); rec.Code != http.StatusCreated {
		t.Fatalf("kasir create: want 201, got %d", rec.Code)
	}
	id := strconv.Itoa(list.Transactions[0].ID)
	if rec := do(srv, kasir, http.MethodPut, "/api/v1/transactions/"+id, `{"tanggal":"2026-08-21","jenis":"Penjualan","nominal":1,"lunas":true}`); rec.Code != http.StatusForbidden {
		t.Fatalf("kasir update: want 403, got %d", rec.Code)
	}
	if rec := do(srv, kasir, http.MethodDelete, "/api/v1/transactions/"+id, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("kasir delete: want 403, got %d", rec.Code)
	}
}
