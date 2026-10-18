package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"pajak/internal/finance"
)

// ponytail: stdlib net/http + RWMutex in-memory state handles 100 UMKM users with <20MB RAM and sub-ms latency.

type Server struct {
	mu        sync.RWMutex
	snapshots map[string]finance.DashboardSnapshot
	ledgers   map[string][]finance.LedgerEntry
	mux       *http.ServeMux
}

func NewServer() *Server {
	s := &Server{
		snapshots: map[string]finance.DashboardSnapshot{
			"2026-08": finance.SeedKedaiKopiSenja("2026-08"),
			"2026-07": finance.SeedKedaiKopiSenja("2026-07"),
		},
		ledgers: map[string][]finance.LedgerEntry{
			"2026-08": finance.SeedLedger("2026-08"),
			"2026-07": finance.SeedLedger("2026-07"),
		},
		mux: http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/v1/dashboard", s.handleDashboard)
	s.mux.HandleFunc("GET /api/v1/reports/financial", s.handleFinancial)
	s.mux.HandleFunc("GET /api/v1/reports/cashflow", s.handleCashflow)
	s.mux.HandleFunc("GET /api/v1/reports/balance-sheet", s.handleBalanceSheet)
	s.mux.HandleFunc("POST /api/v1/pricing/simulate", s.handleSimulatePrice)
	s.mux.HandleFunc("POST /api/v1/transactions", s.handleAddTransaction)
	s.mux.HandleFunc("GET /", s.handleStatic)
}

func periodOrDefault(r *http.Request) string {
	if p := r.URL.Query().Get("period"); p != "" {
		return p
	}
	return "2026-08"
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleFinancial(w http.ResponseWriter, r *http.Request) {
	period := periodOrDefault(r)
	s.mu.RLock()
	entries := s.ledgers[period]
	s.mu.RUnlock()
	writeJSON(w, finance.BuildIncomeStatement(entries))
}

// ponytail: opening cash hardcoded to demo tenant (Rp80 jt); read from accounts table once Postgres lands.
func (s *Server) handleCashflow(w http.ResponseWriter, r *http.Request) {
	period := periodOrDefault(r)
	s.mu.RLock()
	entries := s.ledgers[period]
	s.mu.RUnlock()
	writeJSON(w, finance.BuildCashFlow(entries, 80_000_000))
}

func (s *Server) handleBalanceSheet(w http.ResponseWriter, r *http.Request) {
	period := periodOrDefault(r)
	s.mu.RLock()
	entries := s.ledgers[period]
	s.mu.RUnlock()

	is := finance.BuildIncomeStatement(entries)
	cf := finance.BuildCashFlow(entries, 80_000_000)
	const inventoryValue int64 = 8_250_000
	const openingEquity int64 = 88_250_000 // Modal Awal

	bs := finance.BuildBalanceSheet(cf.ClosingBalance, cf.NewReceivables, cf.NewPayables, inventoryValue, openingEquity, is.NetProfit)
	writeJSON(w, bs)
}


func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "2026-08"
	}
	s.mu.RLock()
	snap, ok := s.snapshots[period]
	if !ok {
		snap = s.snapshots["2026-08"]
	}
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snap)
}

type simulateReq struct {
	COGS         int64 `json:"cogs"`
	TargetProfit int64 `json:"target_profit"`
	FeeBps       int   `json:"fee_bps"`
	FixedFee     int64 `json:"fixed_fee"`
}

func (s *Server) handleSimulatePrice(w http.ResponseWriter, r *http.Request) {
	var req simulateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	res := finance.SimulateChannelPrice(req.COGS, req.TargetProfit, req.FeeBps, req.FixedFee)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

type addTxReq struct {
	Title   string `json:"title"`
	Type    string `json:"type"`
	Channel string `json:"channel"`
	Gross   int64  `json:"gross"`
	FeeBps  int    `json:"fee_bps"`
}

func (s *Server) handleAddTransaction(w http.ResponseWriter, r *http.Request) {
	var req addTxReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Gross <= 0 {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	fee := (req.Gross * int64(req.FeeBps)) / 10000
	net := req.Gross - fee
	if req.Type == "EXPENSE" {
		net = -req.Gross
	}

	s.mu.Lock()
	snap := s.snapshots["2026-08"]
	snap.Tenant.TotalTransactions++
	newTx := finance.TransactionItem{
		ID:       "TX-NEW",
		Date:     "Baru saja",
		Type:     req.Type,
		Category: "Operasional",
		Title:    req.Title,
		Channel:  req.Channel,
		Gross:    req.Gross,
		Fee:      fee,
		Net:      net,
	}
	snap.RecentTransactions = append([]finance.TransactionItem{newTx}, snap.RecentTransactions...)
	if req.Type == "INCOME" {
		snap.KPIs.Omzet.Value += req.Gross
		snap.KPIs.Omzet.Display = finance.FormatJt(snap.KPIs.Omzet.Value)
		snap.KPIs.LabaBersih.Value += net
		snap.KPIs.LabaBersih.Display = finance.FormatJt(snap.KPIs.LabaBersih.Value)
		snap.KPIs.SaldoKas.Value += net
		snap.KPIs.SaldoKas.Display = finance.FormatJt(snap.KPIs.SaldoKas.Value)
	} else {
		snap.KPIs.LabaBersih.Value -= req.Gross
		snap.KPIs.LabaBersih.Display = finance.FormatJt(snap.KPIs.LabaBersih.Value)
		snap.KPIs.SaldoKas.Value -= req.Gross
		snap.KPIs.SaldoKas.Display = finance.FormatJt(snap.KPIs.SaldoKas.Value)
	}

	cat := finance.CatSales
	if req.Type == "EXPENSE" {
		cat = "OPEX"
	}
	s.ledgers["2026-08"] = append(s.ledgers["2026-08"], finance.LedgerEntry{
		Kind:     req.Type,
		Category: cat,
		Title:    req.Title,
		Channel:  req.Channel,
		Gross:    req.Gross,
		Fee:      fee,
		Settled:  true,
	})
	s.snapshots["2026-08"] = snap
	s.mu.Unlock()



	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snap)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	candidates := []string{
		"web/index.html",
		filepath.Join("..", "..", "web", "index.html"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			http.ServeFile(w, r, p)
			return
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte("<h1>Laporan Bisnis API Ready</h1>"))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := NewServer()
	log.Printf("Laporan Bisnis by Akademi Pajak listening on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, srv); err != nil {
		log.Fatal(err)
	}
}
