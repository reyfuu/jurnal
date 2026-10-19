package main

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"time"

	"pajak/internal/ai"
	"pajak/internal/auth"
	"pajak/internal/finance"
	"pajak/internal/xlsx"
)

// ponytail: stdlib net/http + RWMutex in-memory state handles 100 UMKM users with <20MB RAM and sub-ms latency.

const sessionCookie = "sid"

type Server struct {
	mu        sync.RWMutex
	snapshots map[string]finance.DashboardSnapshot // seed per periode; KPI dihitung ulang dari txs
	txs       []finance.Transaction
	nextTxID  int
	seedTx    map[string]int // jumlah transaksi seed per periode, untuk total transaksi tenant
	users     *auth.Store
	loginMu   sync.Mutex
	failures  map[string]loginFailure // per IP
	aiEngine  *ai.Engine
	mux       *http.ServeMux
}

func NewServer() *Server {
	txs := finance.SeedTransactions()
	snapshots := make(map[string]finance.DashboardSnapshot)
	seedTx := make(map[string]int)
	for _, p := range finance.AvailablePeriods() {
		snapshots[p.ID] = finance.SeedKedaiKopiSenja(p.ID)
		seedTx[p.ID] = len(finance.ForPeriod(txs, p.ID))
	}

	s := &Server{
		snapshots: snapshots,
		txs:       txs,
		nextTxID:  len(txs) + 1,
		seedTx:    seedTx,
		users:     auth.NewStoreWithPassword(cmp.Or(os.Getenv("SEED_PASSWORD"), auth.DemoPassword)),
		failures:  map[string]loginFailure{},
		aiEngine:  ai.NewEngine(),
		mux:       http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// ponytail: CSP belum dipasang; perlu audit inline style/script SPA dulu.
	h := w.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	s.mux.ServeHTTP(w, r)
}

// Batas brute force login: maxLoginFailures gagal per IP dalam loginWindow → 429.
const (
	maxLoginFailures = 5
	loginWindow      = 15 * time.Minute
)

type loginFailure struct {
	count int
	since time.Time
}

// clientIP memakai X-Real-IP dari nginx; container hanya bisa diakses lewat nginx (127.0.0.1).
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

func (s *Server) routes() {
	s.mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	s.mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
	s.mux.HandleFunc("GET /api/v1/auth/me", s.guard("", s.handleMe))
	s.mux.HandleFunc("GET /api/v1/periods", s.guard("", s.handlePeriods))

	s.mux.HandleFunc("GET /api/v1/dashboard", s.guard("dashboard", s.handleDashboard))
	s.mux.HandleFunc("GET /api/v1/reports/financial", s.guard("laporan-keuangan", s.handleFinancial))
	s.mux.HandleFunc("GET /api/v1/reports/cashflow", s.guard("arus-kas", s.handleCashflow))
	s.mux.HandleFunc("GET /api/v1/reports/balance-sheet", s.guard("laporan-keuangan", s.handleBalanceSheet))
	s.mux.HandleFunc("POST /api/v1/pricing/simulate", s.guard("simulasi-harga", s.handleSimulatePrice))
	s.mux.HandleFunc("POST /api/v1/ai/chat", s.guard("", s.handleAIChat))
	s.mux.HandleFunc("GET /api/v1/transactions", s.guard("transaksi", s.handleListTransactions))
	s.mux.HandleFunc("POST /api/v1/transactions", s.guard("transaksi", s.handleCreateTransaction))
	s.mux.HandleFunc("GET /api/v1/transactions/template", s.guard("transaksi", s.handleTransactionsTemplate))
	s.mux.HandleFunc("POST /api/v1/transactions/import", s.guard("transaksi-kelola", s.handleImportTransactions))
	s.mux.HandleFunc("PUT /api/v1/transactions/{id}", s.guard("transaksi-kelola", s.handleUpdateTransaction))
	s.mux.HandleFunc("DELETE /api/v1/transactions/{id}", s.guard("transaksi-kelola", s.handleDeleteTransaction))

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
	ip := clientIP(r)
	s.loginMu.Lock()
	f := s.failures[ip]
	if time.Since(f.since) > loginWindow {
		f = loginFailure{}
	}
	s.loginMu.Unlock()
	if f.count >= maxLoginFailures {
		http.Error(w, "terlalu banyak percobaan login, coba lagi dalam 15 menit", http.StatusTooManyRequests)
		return
	}
	token, u, err := s.users.Login(req.Email, req.Password)
	s.loginMu.Lock()
	if err != nil {
		if f.count == 0 {
			f.since = time.Now()
		}
		f.count++
		s.failures[ip] = f
	} else {
		delete(s.failures, ip)
	}
	s.loginMu.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/", MaxAge: int(auth.SessionTTL.Seconds()),
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: isHTTPS(r),
	})
	writeJSON(w, newMe(u))
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		s.users.Logout(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: isHTTPS(r)})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, newMe(currentUser(r)))
}

func (s *Server) handlePeriods(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, finance.AvailablePeriods())
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
	return finance.ToLedger(finance.ForPeriod(s.txs, periodOrDefault(r)))
}

// ponytail: nilai stok tetap seed (Rp8,25 jt) sampai modul stok menghitung persediaan dari transaksi.
const inventoryValue int64 = 8_250_000

func (s *Server) handleFinancial(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, finance.BuildIncomeStatement(s.ledger(r)))
}

func (s *Server) handleCashflow(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, finance.BuildCashFlow(s.ledger(r), finance.OpeningCash(periodOrDefault(r))))
}

func (s *Server) handleBalanceSheet(w http.ResponseWriter, r *http.Request) {
	opening := finance.OpeningCash(periodOrDefault(r))
	// Modal awal = kas awal + stok awal (tanpa utang awal).
	writeJSON(w, finance.BuildBalanceSheet(s.ledger(r), opening, inventoryValue, opening+inventoryValue))
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	period := periodOrDefault(r)
	s.mu.RLock()
	snap, ok := s.snapshots[period]
	if !ok {
		period = "2026-08"
		snap = s.snapshots[period]
	}
	txs := finance.ForPeriod(s.txs, period)
	seedN := s.seedTx[period]
	s.mu.RUnlock()

	entries := finance.ToLedger(txs)
	finance.ApplyKPIs(&snap, finance.BuildIncomeStatement(entries), finance.BuildCashFlow(entries, finance.OpeningCash(period)), txs, seedN)
	snap.CashWarnings = finance.CashWarnings(txs, finance.OpeningCash(period))
	writeJSON(w, snap)
}

type simulateReq struct {
	COGS            int64  `json:"cogs"`
	TargetProfit    int64  `json:"target_profit"`
	FeeBps          int    `json:"fee_bps"`
	FixedFee        int64  `json:"fixed_fee"`
	FixedCost       *int64 `json:"fixed_cost"` // kosong = total beban operasional periode berjalan
	MonthlyVolume   int64  `json:"monthly_volume"`
	TargetNetProfit int64  `json:"target_net_profit"`
}

type simulateResp struct {
	finance.PriceSimulationResult
	BreakEven      *finance.BreakEvenResult `json:"break_even,omitempty"`
	BreakEvenError string                   `json:"break_even_error,omitempty"`
}

func (s *Server) handleSimulatePrice(w http.ResponseWriter, r *http.Request) {
	var req simulateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	sim := finance.SimulateChannelPrice(req.COGS, req.TargetProfit, req.FeeBps, req.FixedFee)
	fixed := finance.BuildIncomeStatement(s.ledger(r)).Opex
	if req.FixedCost != nil {
		fixed = *req.FixedCost
	}
	resp := simulateResp{PriceSimulationResult: sim}
	be, err := finance.BreakEven(sim.RecommendedPrice, sim.COGS+sim.ChannelFee, fixed, req.MonthlyVolume, req.TargetNetProfit)
	if err != nil {
		resp.BreakEvenError = err.Error()
	} else {
		resp.BreakEven = &be
	}
	writeJSON(w, resp)
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
	ledger := finance.ToLedger(finance.ForPeriod(s.txs, period))
	s.mu.RUnlock()

	resp := s.aiEngine.Answer(r.Context(), req, snap, ledger)
	writeJSON(w, resp)
}

// --- Transaksi ---

type txListResp struct {
	Transactions []finance.Transaction `json:"transactions"`
	Jenis        []finance.Jenis       `json:"jenis"`
	CanEdit      bool                  `json:"can_edit"`
}

func (s *Server) handleListTransactions(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	txs := slices.Clone(s.txs)
	s.mu.RUnlock()
	writeJSON(w, txListResp{Transactions: txs, Jenis: finance.JenisList, CanEdit: currentUser(r).Role.Can("transaksi-kelola")})
}

// decodeTx membaca dan memvalidasi body; menulis 400 dan mengembalikan false bila gagal.
func decodeTx(w http.ResponseWriter, r *http.Request) (finance.Transaction, bool) {
	var t finance.Transaction
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "data transaksi tidak valid", http.StatusBadRequest)
		return t, false
	}
	if err := finance.Validate(&t); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return t, false
	}
	return t, true
}

// txResp: transaksi tersimpan + peringatan kas minus di bulan transaksi (tidak memblokir simpan).
type txResp struct {
	finance.Transaction
	Warnings []finance.CashWarning `json:"warnings,omitempty"`
}

// withWarnings memanggil CashWarnings untuk periode transaksi; pemanggil memegang s.mu.
func (s *Server) withWarnings(t finance.Transaction) txResp {
	period := t.Tanggal[:7]
	return txResp{t, finance.CashWarnings(finance.ForPeriod(s.txs, period), finance.OpeningCash(period))}
}

func (s *Server) handleCreateTransaction(w http.ResponseWriter, r *http.Request) {
	t, ok := decodeTx(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	t.ID = s.nextTxID
	s.nextTxID++
	s.txs = append(s.txs, t)
	resp := s.withWarnings(t)
	s.mu.Unlock()
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, resp)
}

// txIndex harus dipanggil dengan s.mu terkunci.
func (s *Server) txIndex(r *http.Request) int {
	id, _ := strconv.Atoi(r.PathValue("id"))
	return slices.IndexFunc(s.txs, func(t finance.Transaction) bool { return t.ID == id })
}

func (s *Server) handleUpdateTransaction(w http.ResponseWriter, r *http.Request) {
	t, ok := decodeTx(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.txIndex(r)
	if i < 0 {
		http.Error(w, "transaksi tidak ditemukan", http.StatusNotFound)
		return
	}
	t.ID = s.txs[i].ID
	s.txs[i] = t
	writeJSON(w, s.withWarnings(t))
}

func (s *Server) handleDeleteTransaction(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.txIndex(r)
	if i < 0 {
		http.Error(w, "transaksi tidak ditemukan", http.StatusNotFound)
		return
	}
	s.txs = slices.Delete(s.txs, i, i+1)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleTransactionsTemplate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="template-transaksi.xlsx"`)
	_ = xlsx.Write(w, []xlsx.Sheet{
		{Name: "Transaksi", Rows: finance.TemplateRows()},
		{Name: "Daftar Jenis", Rows: finance.JenisSheetRows()},
	})
}

type importResp struct {
	Valid    int                   `json:"valid"`
	Errors   []finance.ImportError `json:"errors"`
	Sample   []finance.Transaction `json:"sample"`
	Imported int                   `json:"imported"`
}

// handleImportTransactions: tanpa ?commit=1 hanya pratinjau (tidak ada yang disimpan).
func (s *Server) handleImportTransactions(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	f, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "unggah berkas .xlsx (maks 5 MB) di field file", http.StatusBadRequest)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "gagal membaca berkas", http.StatusBadRequest)
		return
	}
	rows, err := xlsx.Read(data)
	if err != nil {
		http.Error(w, err.Error()+"; simpan ulang sebagai .xlsx atau pakai template", http.StatusBadRequest)
		return
	}
	txs, errs := finance.ParseImport(rows)
	res := importResp{Valid: len(txs), Errors: errs, Sample: txs[:min(5, len(txs))]}
	if r.URL.Query().Get("commit") == "1" {
		s.mu.Lock()
		for _, t := range txs {
			t.ID = s.nextTxID
			s.nextTxID++
			s.txs = append(s.txs, t)
		}
		s.mu.Unlock()
		res.Imported = len(txs)
	}
	writeJSON(w, res)
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
