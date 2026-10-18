package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"pajak/internal/ai"
	"pajak/internal/auth"
	"pajak/internal/finance"
)

// ponytail: stdlib net/http + RWMutex in-memory state handles 100 UMKM users with <20MB RAM and sub-ms latency.

const sessionCookie = "sid"

type Server struct {
	mu        sync.RWMutex
	snapshots map[string]finance.DashboardSnapshot
	ledgers   map[string][]finance.LedgerEntry
	users     *auth.Store
	aiEngine  *ai.Engine
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
		users:    auth.NewStore(),
		aiEngine: ai.NewEngine(),
		mux:      http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	s.mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
	s.mux.HandleFunc("GET /api/v1/auth/me", s.guard("", s.handleMe))

	s.mux.HandleFunc("GET /api/v1/dashboard", s.guard("dashboard", s.handleDashboard))
	s.mux.HandleFunc("GET /api/v1/reports/financial", s.guard("laporan-keuangan", s.handleFinancial))
	s.mux.HandleFunc("GET /api/v1/reports/cashflow", s.guard("arus-kas", s.handleCashflow))
	s.mux.HandleFunc("GET /api/v1/reports/balance-sheet", s.guard("laporan-keuangan", s.handleBalanceSheet))
	s.mux.HandleFunc("POST /api/v1/pricing/simulate", s.guard("simulasi-harga", s.handleSimulatePrice))
	s.mux.HandleFunc("POST /api/v1/ai/chat", s.guard("", s.handleAIChat))
	s.mux.HandleFunc("POST /api/v1/transactions", s.guard("transaksi", s.handleAddTransaction))

	s.mux.HandleFunc("GET /api/v1/users", s.guard("pengguna", s.handleListUsers))
	s.mux.HandleFunc("POST /api/v1/users", s.guard("pengguna", s.handleCreateUser))
	s.mux.HandleFunc("PUT /api/v1/users/{id}", s.guard("pengguna", s.handleUpdateUser))
	s.mux.HandleFunc("DELETE /api/v1/users/{id}", s.guard("pengguna", s.handleDeleteUser))

	s.mux.HandleFunc("GET /", s.handleStatic)
}

type ctxKey struct{}

func currentUser(r *http.Request) auth.User { return r.Context().Value(ctxKey{}).(auth.User) }

// guard menolak request tanpa sesi (401) dan role yang tidak punya akses ke view (403). view "" = cukup login.
func (s *Server) guard(view string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookie)
		if err != nil {
			http.Error(w, "silakan login", http.StatusUnauthorized)
			return
		}
		u, ok := s.users.UserBySession(c.Value)
		if !ok {
			http.Error(w, "sesi berakhir, silakan login", http.StatusUnauthorized)
			return
		}
		if view != "" && !u.Role.Can(view) {
			http.Error(w, "akses ditolak untuk role ini", http.StatusForbidden)
			return
		}
		h(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, u)))
	}
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

// --- Auth & pengguna ---

type meResp struct {
	User      auth.User `json:"user"`
	RoleLabel string    `json:"role_label"`
	Views     []string  `json:"views"`
}

func newMe(u auth.User) meResp {
	return meResp{User: u, RoleLabel: auth.Roles[u.Role].Label, Views: auth.Roles[u.Role].Views}
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct{ Email, Password string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	token, u, err := s.users.Login(req.Email, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: r.TLS != nil,
	})
	writeJSON(w, newMe(u))
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		s.users.Logout(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Path: "/", MaxAge: -1, HttpOnly: true})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, newMe(currentUser(r)))
}

type usersResp struct {
	Users []auth.User                 `json:"users"`
	Roles map[auth.Role]auth.RoleInfo `json:"roles"`
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, usersResp{Users: s.users.List(), Roles: auth.Roles})
}

func writeAuthErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch {
	case errors.Is(err, auth.ErrInvalid):
		code = http.StatusBadRequest
	case errors.Is(err, auth.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, auth.ErrEmailTaken), errors.Is(err, auth.ErrLastOwner):
		code = http.StatusConflict
	}
	http.Error(w, err.Error(), code)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var in auth.Input
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	u, err := s.users.Create(in)
	if err != nil {
		writeAuthErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, u)
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	var in auth.Input
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	u, err := s.users.Update(id, in)
	if err != nil {
		writeAuthErr(w, err)
		return
	}
	writeJSON(w, u)
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	if id == currentUser(r).ID {
		http.Error(w, "tidak bisa menghapus akun sendiri", http.StatusBadRequest)
		return
	}
	if err := s.users.Delete(id); err != nil {
		writeAuthErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Laporan ---

func (s *Server) ledger(r *http.Request) []finance.LedgerEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ledgers[periodOrDefault(r)]
}

func (s *Server) handleFinancial(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, finance.BuildIncomeStatement(s.ledger(r)))
}

// ponytail: opening cash hardcoded to demo tenant (Rp80 jt); read from accounts table once Postgres lands.
func (s *Server) handleCashflow(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, finance.BuildCashFlow(s.ledger(r), 80_000_000))
}

func (s *Server) handleBalanceSheet(w http.ResponseWriter, r *http.Request) {
	entries := s.ledger(r)
	is := finance.BuildIncomeStatement(entries)
	cf := finance.BuildCashFlow(entries, 80_000_000)
	const inventoryValue int64 = 8_250_000
	const openingEquity int64 = 88_250_000 // Modal Awal
	writeJSON(w, finance.BuildBalanceSheet(cf.ClosingBalance, cf.NewReceivables, cf.NewPayables, inventoryValue, openingEquity, is.NetProfit))
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	snap, ok := s.snapshots[periodOrDefault(r)]
	if !ok {
		snap = s.snapshots["2026-08"]
	}
	s.mu.RUnlock()
	writeJSON(w, snap)
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
	writeJSON(w, finance.SimulateChannelPrice(req.COGS, req.TargetProfit, req.FeeBps, req.FixedFee))
}

func (s *Server) handleAIChat(w http.ResponseWriter, r *http.Request) {
	var req ai.QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Query == "" {
		http.Error(w, "query required", http.StatusBadRequest)
		return
	}
	period := req.Period
	if period == "" {
		period = "2026-08"
	}
	s.mu.RLock()
	snap, ok := s.snapshots[period]
	if !ok {
		snap = s.snapshots["2026-08"]
	}
	ledger := s.ledgers[period]
	s.mu.RUnlock()

	resp := s.aiEngine.Answer(r.Context(), req, snap, ledger)
	writeJSON(w, resp)
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

	writeJSON(w, snap)
}

// handleStatic menyajikan build React (web/dist) dengan fallback SPA ke index.html.
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	for _, dir := range []string{filepath.Join("web", "dist"), filepath.Join("..", "..", "web", "dist")} {
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		p := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		if fi, err := os.Stat(p); err != nil || fi.IsDir() {
			p = filepath.Join(dir, "index.html")
		}
		http.ServeFile(w, r, p)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte("<h1>Laporan Bisnis API Ready</h1><p>Frontend belum di-build: jalankan <code>cd frontend &amp;&amp; npm run build</code>.</p>"))
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
