package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"pajak/internal/xlsx"
)

const dagangCV = `{"nama_usaha":"Toko Senja","jenis_usaha":"perdagangan","bentuk_usaha":"cv","pkp":false,"metode_persediaan":"periodik"}`

func TestCoAEndpoints(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")

	var c struct {
		Supported bool                `json:"supported"`
		Accounts  []map[string]any    `json:"accounts"`
		Mapping   []map[string]string `json:"mapping"`
	}
	// belum ada profil → tidak didukung
	_ = json.NewDecoder(do(srv, owner, http.MethodGet, "/api/v1/coa", "").Body).Decode(&c)
	if c.Supported {
		t.Fatal("tanpa profil perdagangan CoA belum didukung")
	}

	do(srv, owner, http.MethodPut, "/api/v1/business", dagangCV)
	_ = json.NewDecoder(do(srv, owner, http.MethodGet, "/api/v1/coa", "").Body).Decode(&c)
	if !c.Supported || len(c.Accounts) < 60 || len(c.Mapping) != 18 {
		t.Fatalf("CoA perdagangan: supported=%v akun=%d mapping=%d", c.Supported, len(c.Accounts), len(c.Mapping))
	}

	var rep struct {
		Supported       bool `json:"supported"`
		IncomeStatement struct {
			LabaBersih int64 `json:"laba_bersih"`
		} `json:"income_statement"`
		BalanceSheet struct {
			Seimbang bool `json:"seimbang"`
		} `json:"balance_sheet"`
		TrialBalance []any `json:"trial_balance"`
	}
	rec := do(srv, owner, http.MethodGet, "/api/v1/reports/coa?period=2026-08", "")
	_ = json.NewDecoder(rec.Body).Decode(&rep)
	if rec.Code != http.StatusOK || !rep.Supported || !rep.BalanceSheet.Seimbang || rep.IncomeStatement.LabaBersih == 0 || len(rep.TrialBalance) == 0 {
		t.Fatalf("laporan CoA: %d %+v", rec.Code, rep)
	}

	rec = do(srv, owner, http.MethodGet, "/api/v1/reports/coa.xlsx?period=2026-08", "")
	rows, err := xlsx.Read(rec.Body.Bytes())
	if rec.Code != http.StatusOK || err != nil || len(rows) < 5 || !strings.Contains(rec.Header().Get("Content-Disposition"), ".xlsx") {
		t.Fatalf("ekspor xlsx: %d %v rows=%d", rec.Code, err, len(rows))
	}
}

func TestTransactionAkunOverride(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	srv := NewServer()
	owner := login(t, srv, "owner@kedaikopisenja.id")
	do(srv, owner, http.MethodPut, "/api/v1/business", dagangCV)

	ok := `{"tanggal":"2026-08-21","jenis":"Biaya Lain","nominal":250000,"keterangan":"Langganan Canva","lunas":true,"akun":"6114"}`
	if rec := do(srv, owner, http.MethodPost, "/api/v1/transactions", ok); rec.Code != http.StatusCreated {
		t.Fatalf("akun valid: %d %s", rec.Code, rec.Body)
	}
	bad := strings.Replace(ok, `"6114"`, `"2115"`, 1) // Utang Dividen tidak aktif untuk CV
	if rec := do(srv, owner, http.MethodPost, "/api/v1/transactions", bad); rec.Code != http.StatusBadRequest {
		t.Fatalf("akun tidak aktif harus 400, got %d", rec.Code)
	}
}
