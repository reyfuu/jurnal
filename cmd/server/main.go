package main

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"maps"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"pajak/internal/ai"
	"pajak/internal/auth"
	"pajak/internal/finance"
	"pajak/internal/katalog"
	"pajak/internal/llm"
	"pajak/internal/master"
	"pajak/internal/pengetahuan"
	"pajak/internal/primbon"
	"pajak/internal/xlsx"
)

// ponytail: stdlib net/http + RWMutex in-memory state handles 100 UMKM users with <20MB RAM and sub-ms latency.

const sessionCookie = "sid"

// tenant: data satu usaha. Usaha 1 = data contoh Kedai Kopi Senja; usaha lain hasil register, mulai kosong.
type tenant struct {
	id          int
	demo        bool // data contoh: saldo awal & dashboard memakai seed
	mu          sync.RWMutex
	snapshots   map[string]finance.DashboardSnapshot // seed per periode; KPI dihitung ulang dari txs
	txs         []finance.Transaction
	nextTxID    int
	assets      []finance.Asset // register aset: dasar penyusutan & amortisasi
	nextAssetID int
	// opening: neraca awal onboarding; nil = belum diisi (usaha demo memakai saldo awal seed).
	opening        *finance.OpeningBalance
	karyawan       []finance.Karyawan
	nextKaryawanID int
	invoices       []finance.Invoice // faktur penjualan; dicetak ke PDF lewat internal/pdf
	nextInvoiceID  int
	tagihan        []finance.Tagihan
	nextTagihanID  int
	stok           []finance.StokItem
	nextStokID     int
	// produk: resep (BOM) per produk; HPP-nya dihitung dari harga pokok bahan di stok.
	produk         []finance.Produk
	nextProdukID   int
	supplier       []finance.Supplier
	nextSupplierID int
	// koreksi: usulan tim Master Pajak; mengubah txs hanya setelah disetujui pemilik.
	koreksi       []finance.Koreksi
	nextKoreksiID int
	// catatan: catatan konsultan pada laporan usaha ini; klien menandai selesai.
	catatan       []finance.CatatanKonsultan
	nextCatatanID int
	// gerak: dokumen kartu stok (stok awal/titik nol, opname, produksi). Pergerakan karena beli
	// & jual berasal dari transaksi bertanda barang; kartu dihitung ulang dari keduanya.
	gerak       []finance.GerakStok
	nextGerakID int
	// tutupBuku: bulan terakhir yang sudah ditutup (YYYY-MM); data bertanggal sampai akhir
	// bulan itu tidak bisa diubah lagi. Kosong = belum ada yang ditutup.
	tutupBuku string
	// coaTemplateID: versi template COA yang dipatok untuk usaha ini; 0 = ikut versi terbaru.
	coaTemplateID int
	// arsip: diarsipkan tim Master Pajak (soft delete); semua user usaha ini tidak bisa masuk.
	arsip       time.Time
	arsipAlasan string
	seedTx      map[string]int           // jumlah transaksi seed per periode, untuk total transaksi tenant
	profile     *finance.BusinessProfile // nil = Owner belum mengisi (onboarding)
	langganan   langganan                // masa coba & pembayaran Midtrans
}

type Server struct {
	tenantsMu sync.RWMutex
	tenants   map[int]*tenant
	users     *auth.Store
	saveMu    sync.Mutex // serialisasi penulisan state
	loginMu   sync.Mutex
	failures  map[string]loginFailure // per IP
	aiEngine  *ai.Engine
	llm       *llm.Client // nil-aman: tanpa LLM_API_KEY, analisis AI mati dan laporan tetap jalan
	mux       *http.ServeMux
	bugs      *bugStore
	db        *sql.DB // nil = berkas JSON di DATA_DIR (atau memori saja)
	// bugHook: penerusan laporan bug ke pengembang; nil-aman, tanpa BUG_WEBHOOK_URL mati.
	bugHook *bugHook
	// master: tim Master Pajak (role, penugasan klien, audit log) untuk Panel Master.
	masterMu sync.Mutex
	master   master.State
	lihat    map[int]lihatGrant // user id → lihat-sebagai aktif (memori saja; restart = selesai)
	// katalog: template COA & benchmark nasional berversi; versi aktifnya dipasang lewat pasangKatalog.
	katalogMu sync.Mutex
	// kuotaAI: waktu pemakaian fitur AI per kunci (mis. "klu|<user>", "mutasi|<usaha>"),
	// untuk kuota harian (lihat jatahAI di klu.go).
	kluMu   sync.Mutex
	kuotaAI map[string][]time.Time
	katalog katalog.State
	// pengetahuan: artikel pajak & sumber yang dibaca AI (Panel Master, Pustaka Pajak klien).
	pengetahuanMu sync.Mutex
	pengetahuan   pengetahuan.State
	// primbon: katalog peraturan resmi (REGULASI_DIR); primbonUser: bookmark & catatan per pengguna.
	primbon     *primbon.Katalog
	primbonUser *primbonStore
}

// newTenant membuat usaha kosong (hasil register).
func newTenant(id int) *tenant {
	t := &tenant{id: id, snapshots: map[string]finance.DashboardSnapshot{}, seedTx: map[string]int{},
		nextTxID: 1, nextAssetID: 1, nextKaryawanID: 1, nextInvoiceID: 1, nextTagihanID: 1, nextStokID: 1,
		nextProdukID: 1, nextSupplierID: 1, nextKoreksiID: 1, nextCatatanID: 1, nextGerakID: 1}
	for _, p := range finance.AvailablePeriods() {
		t.snapshots[p.ID] = finance.EmptySnapshot(p.ID)
	}
	return t
}

// demoTenant: usaha 1 berisi data contoh Kedai Kopi Senja.
func demoTenant() *tenant {
	t := &tenant{id: 1, demo: true, txs: finance.SeedTransactions(), assets: finance.SeedAssets(),
		invoices: finance.SeedInvoices(), tagihan: finance.SeedTagihan(), stok: finance.SeedStok(),
		produk: finance.SeedProduk(), supplier: finance.SeedSupplier(),
		snapshots: map[string]finance.DashboardSnapshot{}, seedTx: map[string]int{}, nextKaryawanID: 1, nextKoreksiID: 1, nextCatatanID: 1, nextGerakID: 1}
	t.nextTxID, t.nextAssetID = len(t.txs)+1, len(t.assets)+1
	t.nextInvoiceID = len(t.invoices) + 1
	t.nextTagihanID, t.nextStokID = len(t.tagihan)+1, len(t.stok)+1
	t.nextProdukID, t.nextSupplierID = len(t.produk)+1, len(t.supplier)+1
	for _, p := range finance.AvailablePeriods() {
		t.snapshots[p.ID] = finance.SeedKedaiKopiSenja(p.ID)
		t.seedTx[p.ID] = len(finance.ForPeriod(t.txs, p.ID))
	}
	return t
}

// restore menimpa data tenant dengan isi berkas state.
func (t *tenant) restore(st *state) {
	t.txs, t.nextTxID = st.Transactions, st.NextTxID
	// state.json dari versi sebelum register aset tidak punya daftar aset: pakai seed.
	if st.NextAssetID > 0 {
		t.assets, t.nextAssetID = st.Assets, st.NextAssetID
	}
	t.opening = st.Opening
	// state sebelum fitur langganan: masa coba dihitung sejak versi ini berjalan (nilai newTenant).
	if st.Langganan != nil {
		t.langganan = *st.Langganan
	}
	if st.NextKaryawanID > 0 {
		t.karyawan, t.nextKaryawanID = st.Karyawan, st.NextKaryawanID
	}
	// state dari versi sebelum modul faktur tidak punya daftar invoice: biarkan apa adanya
	// (usaha demo tetap memakai seed, usaha hasil register tetap kosong).
	if st.NextInvoiceID > 0 {
		t.invoices, t.nextInvoiceID = st.Invoices, st.NextInvoiceID
	}
	if st.NextTagihanID > 0 {
		t.tagihan, t.nextTagihanID = st.Tagihan, st.NextTagihanID
	}
	if st.NextStokID > 0 {
		t.stok, t.nextStokID = st.Stok, st.NextStokID
	}
	if st.NextProdukID > 0 {
		t.produk, t.nextProdukID = st.Produk, st.NextProdukID
	}
	if st.NextSupplierID > 0 {
		t.supplier, t.nextSupplierID = st.Supplier, st.NextSupplierID
	}
	if st.NextKoreksiID > 0 {
		t.koreksi, t.nextKoreksiID = st.Koreksi, st.NextKoreksiID
	}
	if st.NextCatatanID > 0 {
		t.catatan, t.nextCatatanID = st.Catatan, st.NextCatatanID
	}
	if st.NextGerakID > 0 {
		t.gerak, t.nextGerakID = st.Gerak, st.NextGerakID
	}
	t.coaTemplateID = st.CoaTemplate
	t.arsip, t.arsipAlasan = st.Arsip, st.ArsipAlasan
	t.tutupBuku = st.TutupBuku
}

func NewServer() *Server {
	s := &Server{
		llm:      llm.New(llm.ConfigFromEnv()),
		failures: map[string]loginFailure{},
		aiEngine: ai.NewEngine(),
		mux:      http.NewServeMux(),
		bugHook:  newBugHook(),
		lihat:    map[int]lihatGrant{},
	}
	loaded := false
	if url := os.Getenv("DATABASE_URL"); url != "" {
		var err error
		if s.db, err = openDB(url); err != nil {
			log.Fatalf("Postgres: %v", err)
		}
		if loaded, err = s.loadDB(); err != nil {
			log.Fatalf("memuat data dari Postgres: %v", err)
		}
	}
	if !loaded {
		// DB kosong: impor berkas JSON lama (atau seed) lalu simpan. Mode JSON: seed hanya
		// disimpan bila state.json belum ada; SEED_PASSWORD hanya dipakai saat itu.
		if fromFiles := s.loadFiles(); s.db != nil || !fromFiles {
			s.saveAll()
		}
	}
	if segarkanBawaan(&s.katalog) {
		// Katalog lama baru mendapat CoA manufaktur bawaan: simpan supaya ID-nya tetap.
		if err := s.saveKatalog(s.katalog); err != nil {
			log.Printf("PERINGATAN: gagal menyimpan CoA manufaktur bawaan ke katalog: %v", err)
		}
	}
	pasangKatalog(s.katalog)
	s.bootstrapMaster()
	s.adoptOrphans()
	s.primbon = primbon.Buka(cmp.Or(os.Getenv("REGULASI_DIR"), filepath.Join("docs", "peraturan")))
	s.loadPrimbon()
	s.routes()
	return s
}

// adoptOrphans membuat usaha kosong untuk user yang usahanya tidak termuat (mis. berkas usaha
// gagal tersimpan), supaya ID itu tidak diberikan ke pendaftar berikutnya (audit C3).
func (s *Server) adoptOrphans() {
	for _, u := range s.users.Snapshot().Users {
		if u.Role != auth.MP && s.tenants[u.TenantID] == nil { // akun tim memang tanpa usaha
			log.Printf("PERINGATAN: user %d memakai usaha %d yang tidak ada; dibuat kosong", u.ID, u.TenantID)
			s.tenants[u.TenantID] = newTenant(u.TenantID)
		}
	}
}

// loadFiles memuat data dari berkas JSON di DATA_DIR (atau seed); true bila state.json ada.
func (s *Server) loadFiles() bool {
	demo := demoTenant()
	users := auth.NewStoreWithPassword(cmp.Or(os.Getenv("SEED_PASSWORD"), auth.DemoPassword))
	st, err := loadState(demo.file("state"))
	if err != nil {
		log.Fatalf("state.json rusak, perbaiki atau pindahkan dulu: %v", err) // jangan timpa data user dengan seed
	}
	if st != nil {
		demo.restore(st)
		users = auth.Restore(st.Auth)
	}
	demo.profile = loadProfile(demo.file("business"))
	tenants, err := loadTenants()
	if err != nil {
		log.Fatalf("data usaha rusak, perbaiki atau pindahkan dulu: %v", err)
	}
	tenants[1] = demo
	s.tenants, s.users, s.bugs = tenants, users, loadBugs()
	s.master = loadMasterFile()
	s.katalog = loadKatalogFile()
	s.pengetahuan = loadPengetahuanFile()
	return st != nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// ponytail: CSP belum dipasang; perlu audit inline style/script SPA dulu.
	h := w.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	if strings.HasPrefix(r.URL.Path, "/api/") {
		h.Set("Cache-Control", "no-store") // data usaha tidak boleh disimpan cache CDN/browser
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		// Audit C8: tolak permintaan pengubah dari situs lain (Fetch Metadata). Webhook
		// Midtrans (server-ke-server) tidak mengirim header ini.
		if site := r.Header.Get("Sec-Fetch-Site"); site == "cross-site" || site == "same-site" {
			http.Error(w, "permintaan lintas situs ditolak", http.StatusForbidden)
			return
		}
		mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		// Login & daftar tidak butuh cookie, jadi form HTML lintas situs bisa memakainya di
		// browser tanpa Fetch Metadata: hanya terima JSON.
		if (r.URL.Path == "/api/v1/auth/login" || r.URL.Path == "/api/v1/auth/register") && formTypes[mt] {
			http.Error(w, "kirim sebagai application/json", http.StatusUnsupportedMediaType)
			return
		}
		if mt != "multipart/form-data" { // unggahan punya batas sendiri (5–6 MB)
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		}
	}
	s.mux.ServeHTTP(w, r)
}

// formTypes: Content-Type yang bisa dikirim form HTML lintas situs tanpa preflight CORS.
var formTypes = map[string]bool{"text/plain": true, "application/x-www-form-urlencoded": true, "multipart/form-data": true}

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
	s.mux.HandleFunc("POST /api/v1/auth/login", s.persisting(s.handleLogin))
	s.mux.HandleFunc("POST /api/v1/auth/register", s.persisting(s.handleRegister))
	s.mux.HandleFunc("POST /api/v1/auth/logout", s.persisting(s.handleLogout))
	s.mux.HandleFunc("GET /api/v1/auth/me", s.guard("", s.handleMe))
	s.mux.HandleFunc("GET /api/v1/periods", s.guard("", s.handlePeriods))

	s.mux.HandleFunc("GET /api/v1/dashboard", s.guard("dashboard", s.on((*tenant).handleDashboard)))
	s.mux.HandleFunc("GET /api/v1/reports/financial", s.guard("laporan-keuangan", s.on((*tenant).handleFinancial)))
	s.mux.HandleFunc("GET /api/v1/reports/cashflow", s.guard("laporan-keuangan", s.on((*tenant).handleCashflow)))
	s.mux.HandleFunc("GET /api/v1/business", s.guard("", s.on((*tenant).handleGetBusiness)))
	s.mux.HandleFunc("GET /api/v1/wilayah", s.guard("", handleWilayah))
	s.mux.HandleFunc("POST /api/v1/business/cek-awal", s.guard("", s.on((*tenant).handleCekAwal)))
	s.mux.HandleFunc("POST /api/v1/business/klu-saran", s.guard("", s.handleSaranKLU))
	s.mux.HandleFunc("PUT /api/v1/business", s.guard("profil-usaha", s.persisting(s.on((*tenant).handlePutBusiness))))
	s.mux.HandleFunc("POST /api/v1/business/metode-penyusutan", s.guard("profil-usaha", s.persisting(s.on((*tenant).handlePilihMetode))))
	s.mux.HandleFunc("POST /api/v1/tax/planning", s.guard("perencanaan-pajak", s.on((*tenant).handleTaxPlanning)))
	s.mux.HandleFunc("GET /api/v1/coa", s.guard("", s.on((*tenant).handleCoA)))
	s.mux.HandleFunc("GET /api/v1/kalkulator", s.guard("kalkulator-pajak", s.handleKalkulatorMaster))
	s.mux.HandleFunc("POST /api/v1/kalkulator/pph21", s.guard("kalkulator-pajak", s.handleHitungPPh21))
	s.mux.HandleFunc("POST /api/v1/kalkulator/potong", s.guard("kalkulator-pajak", s.handleHitungPotong))
	s.mux.HandleFunc("POST /api/v1/kalkulator/pph-badan", s.guard("kalkulator-pajak", s.handleHitungPPhBadan))
	s.mux.HandleFunc("POST /api/v1/kalkulator/pkp", s.guard("kalkulator-pajak", s.handleCekPKP))
	s.mux.HandleFunc("GET /api/v1/reports/ai", s.guard("laporan-keuangan", s.on(s.handleAIReport)))
	s.mux.HandleFunc("GET /api/v1/reports/coa", s.guard("laporan-keuangan", s.on((*tenant).handleReportCoA)))
	s.mux.HandleFunc("GET /api/v1/reports/coa.xlsx", s.guard("laporan-keuangan", s.exporting(s.on((*tenant).handleReportCoAXlsx))))
	s.mux.HandleFunc("GET /api/v1/reports/buku-besar", s.guard("buku-besar", s.on((*tenant).handleBukuBesar)))
	s.mux.HandleFunc("GET /api/v1/reports/transaksi", s.guard("laporan-transaksi", s.on((*tenant).handleLaporanTransaksi)))
	s.mux.HandleFunc("GET /api/v1/reports/transaksi.xlsx", s.guard("laporan-transaksi", s.exporting(s.on((*tenant).handleLaporanTransaksiXlsx))))
	s.mux.HandleFunc("GET /api/v1/reports/buku-besar.xlsx", s.guard("buku-besar", s.exporting(s.on((*tenant).handleBukuBesarXlsx))))
	s.mux.HandleFunc("GET /api/v1/reports/sales", s.guard("laporan-penjualan", s.on((*tenant).handleSalesReport)))
	s.mux.HandleFunc("GET /api/v1/reports/balance-sheet", s.guard("laporan-keuangan", s.on((*tenant).handleBalanceSheet)))
	s.mux.HandleFunc("GET /api/v1/reports/pdf", s.guard("laporan-keuangan", s.exporting(s.on((*tenant).handleLaporanPDF))))
	s.mux.HandleFunc("GET /api/v1/reports/banding", s.guard("laporan-keuangan", s.on((*tenant).handleBanding)))
	s.mux.HandleFunc("GET /api/v1/reports/banding.pdf", s.guard("laporan-keuangan", s.exporting(s.on((*tenant).handleBandingPDF))))
	s.mux.HandleFunc("POST /api/v1/pricing/simulate", s.guard("simulasi-harga", s.on((*tenant).handleSimulatePrice)))
	s.mux.HandleFunc("POST /api/v1/simulasi", s.guard("simulasi-harga", s.on((*tenant).handleSimulasi)))
	s.mux.HandleFunc("POST /api/v1/ai/chat", s.guard("", s.handleAIChat))
	s.mux.HandleFunc("GET /api/v1/transactions", s.guard("transaksi", s.on((*tenant).handleListTransactions)))
	s.mux.HandleFunc("POST /api/v1/transactions", s.persisting(s.guard("transaksi", s.on((*tenant).handleCreateTransaction))))
	s.mux.HandleFunc("GET /api/v1/transactions/template", s.guard("transaksi", s.handleTransactionsTemplate))
	s.mux.HandleFunc("POST /api/v1/transactions/import", s.persisting(s.guard("transaksi-kelola", s.on((*tenant).handleImportTransactions))))
	s.mux.HandleFunc("POST /api/v1/transactions/mutasi", s.guard("transaksi-kelola", s.handleMutasiPreview))
	s.mux.HandleFunc("POST /api/v1/transactions/mutasi/commit", s.persisting(s.guard("transaksi-kelola", s.on((*tenant).handleMutasiCommit))))
	s.mux.HandleFunc("PUT /api/v1/transactions/{id}", s.persisting(s.guard("transaksi-kelola", s.on((*tenant).handleUpdateTransaction))))
	s.mux.HandleFunc("DELETE /api/v1/transactions/{id}", s.persisting(s.guard("transaksi-kelola", s.on((*tenant).handleDeleteTransaction))))

	s.mux.HandleFunc("GET /api/v1/assets", s.guard("laporan-keuangan", s.on((*tenant).handleListAssets)))
	s.mux.HandleFunc("POST /api/v1/assets", s.persisting(s.guard("laporan-keuangan", s.on((*tenant).handleCreateAsset))))
	s.mux.HandleFunc("PUT /api/v1/assets/{id}", s.persisting(s.guard("laporan-keuangan", s.on((*tenant).handleUpdateAsset))))
	s.mux.HandleFunc("DELETE /api/v1/assets/{id}", s.persisting(s.guard("laporan-keuangan", s.on((*tenant).handleDeleteAsset))))

	s.mux.HandleFunc("GET /api/v1/onboarding/template", s.guard("impor-data", s.on((*tenant).handleOnboardingTemplate)))
	s.mux.HandleFunc("POST /api/v1/onboarding/import", s.persisting(s.guard("impor-data", s.on((*tenant).handleOnboardingImport))))
	s.mux.HandleFunc("GET /api/v1/opening", s.guard("laporan-keuangan", s.on((*tenant).handleGetOpening)))
	s.mux.HandleFunc("PUT /api/v1/opening", s.persisting(s.guard("saldo-awal", s.on((*tenant).handlePutOpening))))

	s.mux.HandleFunc("GET /api/v1/karyawan", s.guard("karyawan", s.on((*tenant).handleListKaryawan)))
	s.mux.HandleFunc("POST /api/v1/karyawan", s.persisting(s.guard("karyawan", s.on((*tenant).handleCreateKaryawan))))
	s.mux.HandleFunc("PUT /api/v1/karyawan/{id}", s.persisting(s.guard("karyawan", s.on((*tenant).handleUpdateKaryawan))))
	s.mux.HandleFunc("DELETE /api/v1/karyawan/{id}", s.persisting(s.guard("karyawan", s.on((*tenant).handleDeleteKaryawan))))

	s.mux.HandleFunc("GET /api/v1/tagihan", s.guard("piutang-utang", s.on((*tenant).handleListTagihan)))
	s.mux.HandleFunc("POST /api/v1/tagihan", s.persisting(s.guard("piutang-utang", s.on((*tenant).handleCreateTagihan))))
	s.mux.HandleFunc("GET /api/v1/tagihan/template", s.guard("piutang-utang", s.handleTagihanTemplate))
	s.mux.HandleFunc("POST /api/v1/tagihan/import", s.persisting(s.guard("piutang-utang", s.on((*tenant).handleImportTagihan))))
	s.mux.HandleFunc("PUT /api/v1/tagihan/{id}", s.persisting(s.guard("piutang-utang", s.on((*tenant).handleUpdateTagihan))))
	s.mux.HandleFunc("DELETE /api/v1/tagihan/{id}", s.persisting(s.guard("piutang-utang", s.on((*tenant).handleDeleteTagihan))))

	s.mux.HandleFunc("GET /api/v1/stok", s.guard("stok-gudang", s.on((*tenant).handleListStok)))
	s.mux.HandleFunc("POST /api/v1/stok", s.persisting(s.guard("stok-gudang", s.on((*tenant).handleCreateStok))))
	s.mux.HandleFunc("GET /api/v1/stok/template", s.guard("stok-gudang", s.handleStokTemplate))
	s.mux.HandleFunc("POST /api/v1/stok/import", s.persisting(s.guard("stok-gudang", s.on((*tenant).handleImportStok))))
	s.mux.HandleFunc("PUT /api/v1/stok/{id}", s.persisting(s.guard("stok-gudang", s.on((*tenant).handleUpdateStok))))
	s.mux.HandleFunc("DELETE /api/v1/stok/{id}", s.persisting(s.guard("stok-gudang", s.on((*tenant).handleDeleteStok))))

	s.mux.HandleFunc("GET /api/v1/produksi", s.guard("produksi-supplier", s.on((*tenant).handleListProduksi)))
	s.mux.HandleFunc("POST /api/v1/produksi/produk", s.persisting(s.guard("produksi-supplier", s.on((*tenant).handleCreateProduk))))
	s.mux.HandleFunc("PUT /api/v1/produksi/produk/{id}", s.persisting(s.guard("produksi-supplier", s.on((*tenant).handleUpdateProduk))))
	s.mux.HandleFunc("DELETE /api/v1/produksi/produk/{id}", s.persisting(s.guard("produksi-supplier", s.on((*tenant).handleDeleteProduk))))
	s.mux.HandleFunc("POST /api/v1/produksi/produk/{id}/terapkan-hpp", s.persisting(s.guard("produksi-supplier", s.on((*tenant).handleTerapkanHPP))))
	s.mux.HandleFunc("POST /api/v1/produksi/supplier", s.persisting(s.guard("produksi-supplier", s.on((*tenant).handleCreateSupplier))))
	s.mux.HandleFunc("PUT /api/v1/produksi/supplier/{id}", s.persisting(s.guard("produksi-supplier", s.on((*tenant).handleUpdateSupplier))))
	s.mux.HandleFunc("DELETE /api/v1/produksi/supplier/{id}", s.persisting(s.guard("produksi-supplier", s.on((*tenant).handleDeleteSupplier))))

	s.mux.HandleFunc("GET /api/v1/invoice", s.guard("invoice", s.on((*tenant).handleListInvoice)))
	s.mux.HandleFunc("POST /api/v1/invoice", s.persisting(s.guard("invoice", s.on((*tenant).handleCreateInvoice))))
	s.mux.HandleFunc("PUT /api/v1/invoice/{id}", s.persisting(s.guard("invoice", s.on((*tenant).handleUpdateInvoice))))
	s.mux.HandleFunc("DELETE /api/v1/invoice/{id}", s.persisting(s.guard("invoice", s.on((*tenant).handleDeleteInvoice))))
	s.mux.HandleFunc("GET /api/v1/invoice/{id}/pdf", s.guard("invoice", s.exporting(s.on((*tenant).handleInvoicePDF))))

	s.mux.HandleFunc("GET /api/v1/users", s.guard("pengguna", s.handleListUsers))
	s.mux.HandleFunc("POST /api/v1/users", s.persisting(s.guard("pengguna", s.handleCreateUser)))
	s.mux.HandleFunc("PUT /api/v1/users/{id}", s.persisting(s.guard("pengguna", s.handleUpdateUser)))
	s.mux.HandleFunc("DELETE /api/v1/users/{id}", s.persisting(s.guard("pengguna", s.handleDeleteUser)))

	s.mux.HandleFunc("GET /api/v1/billing/plans", s.handlePlans)
	s.mux.HandleFunc("GET /api/v1/billing", s.guard("", s.handleBilling))
	s.mux.HandleFunc("POST /api/v1/billing/checkout", s.persisting(s.guard("langganan", s.handleCheckout)))
	s.mux.HandleFunc("POST /api/v1/billing/garansi", s.persisting(s.guard("langganan", s.handleKlaimGaransi)))
	s.mux.HandleFunc("POST /api/v1/billing/refresh", s.guard("langganan", s.handleBillingRefresh))
	s.mux.HandleFunc("POST /api/v1/billing/midtrans/notify", s.handleMidtransNotify)

	s.mux.HandleFunc("GET /api/v1/bugs", s.guard("lapor-bug", s.handleListBugs))
	s.mux.HandleFunc("POST /api/v1/bugs", s.guard("lapor-bug", s.handleCreateBug))
	s.mux.HandleFunc("GET /api/v1/bugs/info", s.guard("lapor-bug", s.handleBugInfo))
	s.mux.HandleFunc("GET /api/v1/bugs/{id}/screenshot", s.guard("lapor-bug", s.handleBugScreenshot))

	s.masterRoutes()
	s.katalogRoutes()
	s.pengetahuanRoutes()
	s.primbonRoutes()
	s.mux.HandleFunc("GET /api/v1/koreksi", s.guard("usulan-koreksi", s.on((*tenant).handleListKoreksi)))
	s.mux.HandleFunc("POST /api/v1/koreksi/{id}/setujui", s.persisting(s.guard("usulan-koreksi", s.handleSetujuiKoreksi)))
	s.mux.HandleFunc("POST /api/v1/koreksi/{id}/tolak", s.persisting(s.guard("usulan-koreksi", s.handleTolakKoreksi)))

	s.mux.HandleFunc("GET /", s.handleStatic)
}

type ctxKey struct{}

func currentUser(r *http.Request) auth.User { return r.Context().Value(ctxKey{}).(auth.User) }

// tenantOf: usaha milik user yang login; hanya dipanggil di balik guard.
func (s *Server) tenantOf(r *http.Request) *tenant {
	s.tenantsMu.RLock()
	defer s.tenantsMu.RUnlock()
	return s.tenants[currentUser(r).TenantID]
}

// on menjalankan handler data terhadap usaha milik user yang login.
func (s *Server) on(h func(*tenant, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { h(s.tenantOf(r), w, r) }
}

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
		s.users.Touch(u.ID, time.Now())
		ctx := r.Context()
		// Akun tim tidak punya usaha: selain Panel Master hanya boleh membaca /auth/me, atau data
		// satu usaha lewat lihat-sebagai (hanya baca). Tanpa ini endpoint "cukup login" (view "")
		// akan berjalan atas usaha yang tidak ada.
		if u.Role == auth.MP && view != "panel-master" && r.URL.Path != "/api/v1/auth/me" {
			g, ok := s.lihatUntuk(u.ID, time.Now())
			if !ok {
				http.Error(w, "akun tim Master Pajak hanya membuka Panel Master", http.StatusForbidden)
				return
			}
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				http.Error(w, "mode lihat sebagai klien hanya baca", http.StatusForbidden)
				return
			}
			if view != "" {
				s.catatBukaLihat(u, view)
			}
			u, ctx = sebagaiKlien(u, g), context.WithValue(ctx, lihatKey{}, true)
		}
		if view != "" && !u.Role.Can(view) {
			http.Error(w, "akses ditolak untuk role ini", http.StatusForbidden)
			return
		}
		if s.diarsip(u.TenantID) {
			http.Error(w, pesanArsip, http.StatusForbidden)
			return
		}
		if s.lockedWrite(r, u.TenantID) {
			http.Error(w, "Langganan belum aktif. Pilih paket di menu Langganan untuk mulai memakai aplikasi.", http.StatusPaymentRequired)
			return
		}
		h(w, r.WithContext(context.WithValue(ctx, ctxKey{}, u)))
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
	User      auth.User       `json:"user"`
	RoleLabel string          `json:"role_label"`
	Views     []string        `json:"views"`
	Langganan langgananStatus `json:"langganan"`
	Garansi   garansiStatus   `json:"garansi"`
	Master    *masterInfo     `json:"master,omitempty"` // hanya akun tim Master Pajak
	Lihat     *lihatInfo      `json:"lihat,omitempty"`  // akun tim yang sedang lihat-sebagai
	// KoreksiMenunggu: usulan koreksi yang menunggu keputusan pemilik usaha.
	KoreksiMenunggu int `json:"koreksi_menunggu,omitempty"`
	// CatatanTerbuka: catatan konsultan yang belum ditandai selesai; CatatanIngat: yang jatuh
	// tempo diingatkan (pop-up pengingat).
	CatatanTerbuka int `json:"catatan_terbuka,omitempty"`
	CatatanIngat   int `json:"catatan_ingat,omitempty"`
	// Jadwal: pengingat tenggat laporan & pajak yang sedang dalam jendela H-n (Panel Master).
	Jadwal []finance.PengingatJadwal `json:"jadwal,omitempty"`
}

func (s *Server) newMe(u auth.User) meResp {
	me := meResp{User: u, RoleLabel: auth.Roles[u.Role].Label, Views: auth.Roles[u.Role].Views}
	if u.Role == auth.MP {
		me.Master = s.masterInfoFor(u)
		if me.Lihat = s.lihatInfoFor(u); me.Lihat != nil {
			// Menu klien versi baca; layar bayar tidak relevan untuk tim.
			me.Views, me.Langganan = auth.Roles[auth.Lihat].Views, langgananStatus{Status: "aktif"}
		}
		return me
	}
	s.tenantsMu.RLock()
	t := s.tenants[u.TenantID]
	s.tenantsMu.RUnlock()
	if t != nil {
		me.Langganan = t.langgananStatus(time.Now())
		t.mu.RLock()
		me.Garansi = statusGaransi(t.langganan, time.Now())
		t.mu.RUnlock()
		if u.Role.Can("usulan-koreksi") {
			me.KoreksiMenunggu = t.koreksiMenunggu()
		}
		me.CatatanTerbuka, me.CatatanIngat = s.catatanTerbuka(t)
		me.Jadwal = s.jadwalAktif(t)
	}
	return me
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct{ Email, Password string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	key := "login|" + clientIP(r)
	if !s.reserveAttempt(key) {
		http.Error(w, "terlalu banyak percobaan login, coba lagi dalam 15 menit", http.StatusTooManyRequests)
		return
	}
	token, u, err := s.users.Login(req.Email, req.Password)
	if err == nil {
		s.refundAttempt(key)
		if s.diarsip(u.TenantID) {
			s.users.Logout(token)
			http.Error(w, pesanArsip, http.StatusForbidden)
			return
		}
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	setSession(w, r, token)
	writeJSON(w, s.newMe(u))
}

// reserveAttempt memotong satu jatah percobaan untuk key SEBELUM password diperiksa, dalam
// satu kunci. Permintaan serentak tidak bisa lolos bersama (audit C2). false = jatah habis.
func (s *Server) reserveAttempt(key string) bool {
	s.loginMu.Lock()
	defer s.loginMu.Unlock()
	f := s.failures[key]
	if time.Since(f.since) > loginWindow {
		f = loginFailure{since: time.Now()}
	}
	if f.count >= maxLoginFailures {
		return false
	}
	f.count++
	s.failures[key] = f
	return true
}

// refundAttempt mengembalikan satu jatah setelah login sukses. Bucket tidak dikosongkan,
// supaya login sah penyerang tidak mereset hitungan tebakan (audit C4).
func (s *Server) refundAttempt(key string) {
	s.loginMu.Lock()
	defer s.loginMu.Unlock()
	if f := s.failures[key]; f.count > 0 {
		f.count--
		s.failures[key] = f
	}
}

func setSession(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/", MaxAge: int(auth.SessionTTL.Seconds()),
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: isHTTPS(r),
	})
}

// versiPDP: versi Kebijakan Privasi yang disetujui pendaftar (frontend KebijakanPrivasi.tsx).
// Naikkan bila isi kebijakan berubah, supaya persetujuan lama bisa dibedakan.
const versiPDP = "2026-10"

// handleRegister: daftar mandiri (terbuka untuk umum). Pendaftar menjadi
// Owner sebuah usaha baru yang kosong, lalu langsung login dan masuk wizard onboarding.
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name, Email, Password, WA string
		SetujuPDP                 bool `json:"setuju_pdp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !req.SetujuPDP {
		http.Error(w, "persetujuan kebijakan privasi (UU PDP) wajib", http.StatusBadRequest)
		return
	}
	wa, err := auth.NormalizeWA(req.WA)
	if err != nil {
		http.Error(w, "nomor WhatsApp tidak valid", http.StatusBadRequest)
		return
	}
	// Setiap pendaftaran memakai satu jatah per IP (tanpa pengembalian): maks 5 akun per 15 menit.
	if !s.reserveAttempt("register|" + clientIP(r)) {
		http.Error(w, "terlalu banyak percobaan, coba lagi dalam 15 menit", http.StatusTooManyRequests)
		return
	}
	s.tenantsMu.Lock()
	id := slices.Max(slices.Collect(maps.Keys(s.tenants))) + 1
	s.tenants[id] = newTenant(id)
	s.tenantsMu.Unlock()
	if _, err := s.users.Create(auth.Input{Name: req.Name, Email: req.Email, Password: req.Password, Role: auth.Owner, Active: true, TenantID: id,
		WA: wa, PersetujuanPDP: time.Now().UTC().Truncate(time.Second), VersiPDP: versiPDP}); err != nil {
		s.tenantsMu.Lock()
		delete(s.tenants, id)
		s.tenantsMu.Unlock()
		writeAuthErr(w, err)
		return
	}
	token, u, err := s.users.Login(req.Email, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	setSession(w, r, token)
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, s.newMe(u))
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		s.users.Logout(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: isHTTPS(r)})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.newMe(currentUser(r)))
}

type usersResp struct {
	Users []auth.User                 `json:"users"`
	Roles map[auth.Role]auth.RoleInfo `json:"roles"`
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	// Role tim Master Pajak bukan pilihan untuk user usaha (auth menolaknya juga).
	roles := maps.Clone(auth.Roles)
	delete(roles, auth.MP)
	delete(roles, auth.Lihat)
	writeJSON(w, usersResp{Users: s.users.List(currentUser(r).TenantID), Roles: roles})
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
	in.TenantID = currentUser(r).TenantID
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
	u, err := s.users.Update(currentUser(r).TenantID, id, in)
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
	if err := s.users.Delete(currentUser(r).TenantID, id); err != nil {
		writeAuthErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Laporan ---

// bukuAwalLocked: saldo awal pembukuan berjalan — neraca awal onboarding bila sudah diisi;
// demo memakai saldo awal Januari data contoh; usaha lain mulai dari nol di bulan data pertama.
// Pemanggil memegang s.mu.
func (s *tenant) bukuAwalLocked() finance.OpeningBalance {
	if s.opening != nil {
		return *s.opening
	}
	if s.demo {
		return finance.OpeningBalance{Tanggal: "2026-01-01", Kas: finance.OpeningCash("2026-01"), Persediaan: finance.SeedInventory}
	}
	awal := finance.BulanBerjalan()
	for _, t := range s.txs {
		if len(t.Tanggal) >= 7 && t.Tanggal[:7] < awal {
			awal = t.Tanggal[:7]
		}
	}
	return finance.OpeningBalance{Tanggal: awal + "-01"}
}

// openingFor: saldo awal satu periode (posisi akhir bulan sebelumnya) dari buku berjalan.
// Pemanggil harus memegang s.mu.
func (s *tenant) openingFor(period string) finance.ReportOpening {
	ob := s.bukuAwalLocked()
	p := finance.PosisiPada(ob, s.txs, period+"-00")
	inv := ob.Persediaan
	if sp := s.kartuLocked().Periode(period); sp.Aktif {
		inv = sp.PersediaanAwal
	}
	return finance.ReportOpening{Cash: p.Kas, Receivables: p.Piutang, Payables: p.Utang, Loans: p.Pinjaman, Inventory: inv}
}

// reportLocked: laporan berjalan satu periode; pemanggil memegang s.mu.
func (s *tenant) reportLocked(period string) finance.PeriodReport {
	return finance.BuildLaporanBerjalan(s.txs, s.assets, period, s.bukuAwalLocked(), s.kartuLocked())
}

// report menghitung seluruh laporan satu periode dari state saat ini. Semua endpoint laporan
// memakai fungsi ini supaya laba rugi, arus kas, neraca, dan register aset tidak saling bertentangan.
func (s *tenant) report(period string) finance.PeriodReport {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rep := s.reportLocked(period)
	if rep.Assets.MetodeUsaha = s.metodeUsaha(); rep.Assets.MetodeUsaha == "" {
		saran, seragam := finance.MetodeSeragam(s.assets)
		rep.Assets.MetodeSaran, rep.Assets.MetodeCampur = saran, !seragam
	}
	return rep
}

func (s *tenant) handleFinancial(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.report(periodOrDefault(r)).Income)
}

// handleSalesReport: granularity day|week untuk satu periode, month untuk tren seluruh periode.
func (s *tenant) handleSalesReport(w http.ResponseWriter, r *http.Request) {
	g := r.URL.Query().Get("granularity")
	s.mu.RLock()
	txs := s.txs
	if g != "month" {
		txs = finance.ForPeriod(s.txs, periodOrDefault(r))
	}
	rep := finance.SalesReport(txs, g)
	s.mu.RUnlock()
	writeJSON(w, rep)
}

func (s *tenant) handleCashflow(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.report(periodOrDefault(r)).CashFlow)
}

func (s *tenant) handleBalanceSheet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.report(periodOrDefault(r)).BalanceSheet)
}

func (s *tenant) handleDashboard(w http.ResponseWriter, r *http.Request) {
	period := periodOrDefault(r)
	s.mu.RLock()
	snap := s.snapshotLocked(period)
	txs := finance.ForPeriod(s.txs, period)
	seedN := s.seedTx[period]
	if s.profile != nil {
		snap.Tenant.Name, snap.Tenant.Category = s.profile.NamaUsaha, s.profile.JenisLabel()
	}
	s.mu.RUnlock()

	rep := s.report(period)
	finance.ApplyKPIs(&snap, rep.Income, rep.CashFlow, txs, seedN)
	// Grafik tren & sparkline ikut dihitung ulang supaya tidak bertentangan dengan KPI.
	s.mu.RLock()
	history := finance.TrendHistory(s.txs, period, s.reportLocked)
	s.mu.RUnlock()
	finance.ApplyTrend(&snap, history)
	snap.CashWarnings = finance.CashWarnings(txs, rep.CashFlow.OpeningBalance)

	// Piutang & utang di dashboard dari buku besar (transaksi belum lunas + saldo awal), sama
	// dengan Neraca dan menu Piutang & Utang. Acuan: akhir periode atau hari ini bila lebih awal.
	s.mu.RLock()
	acuan := time.Now()
	if t, err := time.Parse("2006-01", period); err == nil {
		if akhir := t.AddDate(0, 1, -1); akhir.Before(acuan) {
			acuan = akhir
		}
	}
	tagihan := s.tagihanBukuLocked(acuan.Format(time.DateOnly))
	stok := finance.TerapkanKartu(append([]finance.StokItem(nil), s.stok...), s.kartuLocked())
	s.mu.RUnlock()
	rekap := finance.SummarizeTagihan(tagihan, acuan)
	snap.Obligations = finance.ToObligations(tagihan, acuan)
	snap.Inventory = finance.ToInventory(stok)
	// KPI "Piutang belum tertagih" ikut dari rekap yang sama. Tanpa ini kartunya tetap memakai
	// angka seed bulanan dengan jumlah faktur dipatok 8, dan dashboard menyebut angka yang
	// berbeda dari halaman yang ditautkannya.
	snap.KPIs.Piutang = rekap.ToPiutangCard()
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
	// MarginBps > 0: harga dihitung dari margin yang diinginkan (mis. 6000 = 60%), bukan
	// TargetProfit per pcs.
	MarginBps int `json:"margin_bps"`
}

type simulateResp struct {
	finance.PriceSimulationResult
	BreakEven      *finance.BreakEvenResult `json:"break_even,omitempty"`
	BreakEvenError string                   `json:"break_even_error,omitempty"`
}

func (s *tenant) handleSimulatePrice(w http.ResponseWriter, r *http.Request) {
	var req simulateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	sim := finance.SimulateChannelPrice(req.COGS, req.TargetProfit, req.FeeBps, req.FixedFee)
	if req.MarginBps > 0 {
		var err error
		if sim, err = finance.HargaUntukMargin(req.COGS, req.FeeBps, req.FixedFee, req.MarginBps); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	// Beban tetap break-even memakai beban operasional kas; penyusutan tidak ikut karena
	// tidak memengaruhi kas yang harus ditutup penjualan.
	fixed := s.report(periodOrDefault(r)).Income.Opex
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
	if !finance.PeriodeSah(period) {
		period = finance.BulanBerjalan()
	}
	t := s.tenantOf(r)
	t.mu.RLock()
	snap := t.snapshotLocked(period)
	t.mu.RUnlock()

	// Pertanyaan "di mana / bagaimana cara" dijawab panduan menu + tautan sesuai hak akses.
	u := currentUser(r)
	boleh := func(v string) bool { return u.Role.Can(v) || v == "catatan" }
	panduan, cocok := ai.CariPanduan(req.Query, boleh)
	if !cocok && ai.TanyaMenu(req.Query) && s.llm != nil && s.llm.Enabled() && s.jatahAI("panduan|"+strconv.Itoa(u.ID), 40) {
		if p, err := ai.PilihPanduan(r.Context(), s.llm, req.Query, boleh); err == nil {
			panduan, cocok = p, true
		}
	}
	if !cocok && ai.TanyaMenu(req.Query) {
		panduan, cocok = ai.PanduanUmum(boleh), true
	}
	if cocok && panduan.Kuat {
		writeJSON(w, ai.QueryResponse{Answer: panduan.Jawaban, Category: "panduan", Tautan: panduan.Tautan, ActionSuggestions: []string{
			"Di mana saya mencatat transaksi?", "Bagaimana cara impor mutasi bank?", "Ringkas laporan bulan ini"}})
		return
	}
	resp := s.aiEngine.Answer(r.Context(), req, snap, t.report(period))
	if cocok {
		resp.Tautan = panduan.Tautan
	}
	writeJSON(w, resp)
}

// --- Transaksi ---

type txListResp struct {
	Transactions []finance.Transaction `json:"transactions"`
	Jenis        []finance.Jenis       `json:"jenis"`
	CanEdit      bool                  `json:"can_edit"`
}

func (s *tenant) handleListTransactions(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	txs := slices.Clone(s.txs)
	s.mu.RUnlock()
	writeJSON(w, txListResp{Transactions: nonNil(txs), Jenis: finance.JenisList, CanEdit: currentUser(r).Role.Can("transaksi-kelola") && !sedangLihat(r)})
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

// decodeTx + cek akun koreksi terhadap CoA aktif. Tautan invoice hanya diatur server.
func (s *tenant) decodeTx(w http.ResponseWriter, r *http.Request) (finance.Transaction, bool) {
	t, ok := decodeTx(w, r)
	t.InvoiceID = 0
	if ok && !s.validAkun(t) {
		http.Error(w, "akun "+t.Akun+" tidak ada di daftar akun usaha ini", http.StatusBadRequest)
		return t, false
	}
	if ok && !s.validStok(t) {
		http.Error(w, "barang yang dipilih tidak ada di Stok & Gudang", http.StatusBadRequest)
		return t, false
	}
	return t, ok
}

// txResp: transaksi tersimpan + peringatan kas minus di bulan transaksi (tidak memblokir simpan).
type txResp struct {
	finance.Transaction
	Warnings []finance.CashWarning `json:"warnings,omitempty"`
}

// withWarnings memanggil CashWarnings untuk periode transaksi; pemanggil memegang s.mu.
func (s *tenant) withWarnings(t finance.Transaction) txResp {
	period := t.Tanggal[:7]
	return txResp{t, finance.CashWarnings(finance.ForPeriod(s.txs, period), s.openingFor(period).Cash)}
}

func (s *tenant) handleCreateTransaction(w http.ResponseWriter, r *http.Request) {
	t, ok := s.decodeTx(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	if err := s.cekTxLocked(t); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	t.ID = s.nextTxID
	s.nextTxID++
	s.txs = append(s.txs, t)
	resp := s.withWarnings(t)
	s.mu.Unlock()
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, resp)
}

// txIndex harus dipanggil dengan s.mu terkunci.
func (s *tenant) txIndex(r *http.Request) int {
	id, _ := strconv.Atoi(r.PathValue("id"))
	return slices.IndexFunc(s.txs, func(t finance.Transaction) bool { return t.ID == id })
}

func (s *tenant) handleUpdateTransaction(w http.ResponseWriter, r *http.Request) {
	t, ok := s.decodeTx(w, r)
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
	if s.txs[i].InvoiceID != 0 {
		http.Error(w, errTxInvoice.Error(), http.StatusConflict)
		return
	}
	if err := s.cekTerbukaLocked(tanggalTx(s.txs[i])...); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err := s.cekTxLocked(t); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	t.ID = s.txs[i].ID
	s.txs[i] = t
	writeJSON(w, s.withWarnings(t))
}

func (s *tenant) handleDeleteTransaction(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.txIndex(r)
	if i < 0 {
		http.Error(w, "transaksi tidak ditemukan", http.StatusNotFound)
		return
	}
	if err := s.cekTerbukaLocked(tanggalTx(s.txs[i])...); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if s.txs[i].InvoiceID != 0 {
		http.Error(w, errTxInvoice.Error(), http.StatusConflict)
		return
	}
	// Transaksi yang menjadi pembayaran aset terdaftar: menghapusnya membuat aset di neraca
	// kehilangan pasangan kas/utangnya, jadi harus lewat menu Aset.
	if j := slices.IndexFunc(s.assets, func(a finance.Asset) bool { return a.TransaksiID == s.txs[i].ID }); j >= 0 {
		http.Error(w, "transaksi ini adalah pembelian aset \""+s.assets[j].Nama+"\"; ubah atau hapus dari menu Aset supaya neraca tetap seimbang", http.StatusConflict)
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
func (s *tenant) handleImportTransactions(w http.ResponseWriter, r *http.Request) {
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
			if err := s.cekTxLocked(t); err != nil {
				s.mu.Unlock()
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
		}
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
		// index.html selalu divalidasi ulang supaya deploy baru langsung terpakai (tanpa ini browser
		// & Cloudflare menahan versi lama); asset Vite ber-hash aman di-cache selamanya.
		if strings.HasPrefix(r.URL.Path, "/assets/") && p != filepath.Join(dir, "index.html") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		http.ServeFile(w, r, p)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte("<h1>Master Pajak API Ready</h1><p>Frontend belum di-build: jalankan <code>cd frontend &amp;&amp; npm run build</code>.</p>"))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := NewServer()
	log.Printf("Master Pajak listening on http://localhost:%s", port)
	hs := &http.Server{
		Addr: ":" + port, Handler: srv,
		ReadHeaderTimeout: 10 * time.Second, ReadTimeout: time.Minute,
		WriteTimeout: 3 * time.Minute, IdleTimeout: 2 * time.Minute, // laporan AI bisa lama
	}
	if err := hs.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
