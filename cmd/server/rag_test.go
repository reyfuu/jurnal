package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"pajak/internal/llm"
	"pajak/internal/rag"
	"strings"
	"testing"
)

func TestRAGAccessAndUnavailable(t *testing.T) {
	srv := primbonServer(t)
	for _, route := range []struct{ method, path, body string }{{"GET", "/api/v1/primbon/rag/status", ""}, {"GET", "/api/v1/primbon/rag/cari?q=UMKM", ""}, {"POST", "/api/v1/primbon/rag/tanya", `{"query":"UMKM"}`}} {
		if r := do(srv, nil, route.method, route.path, route.body); r.Code != http.StatusUnauthorized {
			t.Fatalf("anonymous %s: %d", route.path, r.Code)
		}
		if r := do(srv, login(t, srv, "kasir@kedaikopisenja.id"), route.method, route.path, route.body); r.Code != http.StatusForbidden {
			t.Fatalf("cashier: %d", r.Code)
		}
		if r := do(srv, login(t, srv, "owner@kedaikopisenja.id"), route.method, route.path, route.body); r.Code != http.StatusServiceUnavailable {
			t.Fatalf("no DB %s: %d %s", route.path, r.Code, r.Body)
		}
	}
}

func TestPostgresRAGRetrievesPageCitations(t *testing.T) {
	pgTest(t)
	t.Setenv("LLM_API_KEY", "")
	srv := NewServer()
	store := rag.Store{DB: srv.db}
	d := rag.Document{ID: "rule-1", Code: "PP 55/2022", Title: "Pajak UMKM", Status: "Diubah", File: "a.pdf"}
	if err := store.Save(context.Background(), d, rag.Fingerprint(d, "content"), []rag.Page{{Number: 4, Text: "Ketentuan omzet UMKM."}}); err != nil {
		t.Fatal(err)
	}
	owner := login(t, srv, "owner@kedaikopisenja.id")
	for _, req := range []struct{ method, path, body string }{{"GET", "/api/v1/primbon/rag/cari?q=UMKM", ""}, {"POST", "/api/v1/primbon/rag/tanya", `{"query":"Bagaimana pajak untuk UMKM?"}`}} {
		res := do(srv, owner, req.method, req.path, req.body)
		if res.Code != 200 || !strings.Contains(res.Body.String(), `"halaman":4`) || !strings.Contains(res.Body.String(), `"status":"Diubah"`) {
			t.Fatalf("citation: %d %s", res.Code, res.Body)
		}
	}
	newUser := register(t, srv, registerBody)
	if r := do(srv, newUser, http.MethodGet, "/api/v1/primbon/rag/status", ""); r.Code != http.StatusPaymentRequired {
		t.Fatalf("unpaid access: %d", r.Code)
	}
}

func TestRAGAnswerRequiresEvidenceAndAllowlistedReferences(t *testing.T) {
	calls := 0
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"{\"jawaban\":\"Lihat ketentuan.\",\"rujukan\":[\"R1\",\"FAKE\"]}"}}]}`))
	}))
	defer fake.Close()
	c := llm.New(llm.Config{APIKey: "test", BaseURL: fake.URL, Model: "test"})
	out := answerRAG(context.Background(), c, "UMKM", nil)
	if calls != 0 || out.Tersedia {
		t.Fatal("model called without evidence")
	}
	hits := []rag.Hit{{Ref: "R1", Title: "PP 55", Text: "Pasal 1", Page: 2}}
	out = answerRAG(context.Background(), c, "UMKM", hits)
	if !out.Tersedia || len(out.Rujukan) != 1 || out.Rujukan[0] != "R1" {
		t.Fatalf("references: %+v", out)
	}
	out = answerRAG(context.Background(), nil, "UMKM", hits)
	if out.Tersedia || !strings.Contains(out.Alasan, "AI") || len(out.Sumber) != 1 {
		t.Fatalf("disabled: %+v", out)
	}
}
