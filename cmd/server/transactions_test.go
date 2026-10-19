package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"pajak/internal/finance"
	"pajak/internal/xlsx"
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

func xlsxUpload(t *testing.T, rows [][]string) (*bytes.Buffer, string) {
	t.Helper()
	var file bytes.Buffer
	if err := xlsx.Write(&file, []xlsx.Sheet{{Name: "Transaksi", Rows: rows}}); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "transaksi.xlsx")
	_, _ = fw.Write(file.Bytes())
	_ = mw.Close()
	return &body, mw.FormDataContentType()
}

func upload(srv *Server, c *http.Cookie, url string, body *bytes.Buffer, ctype string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, url, body)
	req.Header.Set("Content-Type", ctype)
	req.AddCookie(c)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func TestTransactionsTemplate(t *testing.T) {
	srv := NewServer()
	kasir := login(t, srv, "kasir@kedaikopisenja.id")
	rec := do(srv, kasir, http.MethodGet, "/api/v1/transactions/template", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Disposition"), ".xlsx") {
		t.Fatalf("template: %d %v", rec.Code, rec.Header())
	}
	rows, err := xlsx.Read(rec.Body.Bytes())
	if err != nil || len(rows) < 2 || rows[0][0] != "Tanggal" {
		t.Fatalf("template content: %v %q", err, rows)
	}
}

func TestTransactionsImport_PreviewThenCommit(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	base := omzetAug(t, srv, owner)
	rows := [][]string{
		{"Tanggal", "Jenis", "Nominal", "Keterangan"},
		{"2026-08-29", "Penjualan", "3.000.000", "Impor rekap kasir"},
		{"2026-08-29", "Sedekah", "100000", "jenis salah"},
	}

	body, ctype := xlsxUpload(t, rows)
	rec := upload(srv, owner, "/api/v1/transactions/import", body, ctype)
	var res importResp
	_ = json.NewDecoder(rec.Body).Decode(&res)
	if rec.Code != http.StatusOK || res.Valid != 1 || len(res.Errors) != 1 || res.Errors[0].Baris != 3 || res.Imported != 0 {
		t.Fatalf("preview: %d %+v", rec.Code, res)
	}
	if omzetAug(t, srv, owner) != base {
		t.Fatal("preview must not save anything")
	}

	body, ctype = xlsxUpload(t, rows)
	rec = upload(srv, owner, "/api/v1/transactions/import?commit=1", body, ctype)
	_ = json.NewDecoder(rec.Body).Decode(&res)
	if rec.Code != http.StatusOK || res.Imported != 1 {
		t.Fatalf("commit: %d %+v", rec.Code, res)
	}
	if got := omzetAug(t, srv, owner); got != base+3_000_000 {
		t.Fatalf("omzet after import: want %d, got %d", base+3_000_000, got)
	}
}

func TestTransactionsImport_RejectsBadFileAndKasir(t *testing.T) {
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "data.csv")
	_, _ = fw.Write([]byte("tanggal,jenis\n"))
	_ = mw.Close()
	if rec := upload(srv, owner, "/api/v1/transactions/import", &body, mw.FormDataContentType()); rec.Code != http.StatusBadRequest {
		t.Fatalf("non-xlsx: want 400, got %d", rec.Code)
	}

	kasir := login(t, srv, "kasir@kedaikopisenja.id")
	b, ct := xlsxUpload(t, finance.TemplateRows())
	if rec := upload(srv, kasir, "/api/v1/transactions/import?commit=1", b, ct); rec.Code != http.StatusForbidden {
		t.Fatalf("kasir import: want 403, got %d", rec.Code)
	}
}

func TestCashWarningOnBigExpense(t *testing.T) {
	srv := asOwner(t, NewServer())
	body := `{"tanggal":"2026-08-01","jenis":"Beli Alat / Aset","nominal":9000000000,"keterangan":"Mesin roasting","lunas":true}`
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(body)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d %s", rec.Code, rec.Body)
	}
	var created struct {
		ID       int                   `json:"id"`
		Warnings []finance.CashWarning `json:"warnings"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&created)
	if created.ID == 0 || len(created.Warnings) == 0 || created.Warnings[0].Tanggal != "2026-08-01" {
		t.Fatalf("transaction must be saved with cash warning, got %+v", created)
	}

	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/dashboard?period=2026-08", nil))
	var snap finance.DashboardSnapshot
	_ = json.NewDecoder(rec.Body).Decode(&snap)
	if len(snap.CashWarnings) == 0 {
		t.Fatal("dashboard must expose cash warnings")
	}
}
