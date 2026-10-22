package main

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"pajak/internal/auth"
	"pajak/internal/finance"
	"pajak/internal/katalog"
	"pajak/internal/master"
	"pajak/internal/pengetahuan"
)

// Penyimpanan PostgreSQL (DATABASE_URL). Data tetap dilayani dari memori; Postgres adalah
// sumber kebenaran yang dimuat saat boot dan ditulis setelah setiap perubahan. Kolom tabel
// = tag JSON struct Go, jadi baris dibaca dengan to_jsonb dan ditulis dengan
// jsonb_populate_recordset tanpa pemetaan kolom manual.
//
// ponytail: simpan = hapus lalu tulis ulang semua baris satu usaha dalam satu transaksi SQL;
// cukup untuk ribuan transaksi per usaha. Ganti ke upsert per baris bila satu usaha
// melewati ~50 rb transaksi atau simpan terasa lambat.

//go:embed schema.sql
var schemaSQL string

func openDB(url string) (*sql.DB, error) {
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	cfg.RuntimeParams["timezone"] = "UTC" // to_jsonb(timestamptz) selalu berformat RFC 3339 yang bisa dibaca Go
	db := stdlib.OpenDB(*cfg)
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("skema: %w", err)
	}
	return db, nil
}

// insertRows menulis rows (slice struct) ke table; tid nil = baris sudah membawa tenant_id.
func insertRows(tx *sql.Tx, table string, tid any, rows any) error {
	b, err := json.Marshal(rows)
	if err != nil || string(b) == "null" { // slice nil: tidak ada baris
		return err
	}
	_, err = tx.Exec(`INSERT INTO `+table+` SELECT * FROM jsonb_populate_recordset(null::`+table+`, lb_rows($1, $2))`, b, tid)
	if err != nil {
		return fmt.Errorf("%s: %w", table, err)
	}
	return nil
}

// selectJSON menjalankan query yang mengembalikan satu nilai jsonb lalu men-decode ke dst.
func selectJSON(db *sql.DB, dst any, query string, args ...any) error {
	var b []byte
	if err := db.QueryRow(query, args...).Scan(&b); err != nil || b == nil { // NULL: dst tetap kosong
		return err
	}
	return json.Unmarshal(b, dst)
}

// rowsOf: semua baris satu usaha sebagai array JSON (tanpa tenant_id); tanpa baris = null,
// jadi slice tetap nil seperti usaha baru.
func rowsOf(table, order string) string {
	return `SELECT jsonb_agg(to_jsonb(t) - 'tenant_id' ORDER BY ` + order + `) FROM ` + table + ` t WHERE tenant_id = $1`
}

// one: nol atau satu baris dari pointer opsional.
func one[T any](p *T) []*T {
	if p == nil {
		return nil
	}
	return []*T{p}
}

type tenantRow struct {
	ID              int            `json:"id"`
	NextTxID        int            `json:"next_tx_id"`
	NextAssetID     int            `json:"next_asset_id"`
	NextKaryawanID  int            `json:"next_karyawan_id"`
	NextInvoiceID   int            `json:"next_invoice_id"`
	NextTagihanID   int            `json:"next_tagihan_id"`
	NextStokID      int            `json:"next_stok_id"`
	NextProdukID    int            `json:"next_produk_id"`
	NextSupplierID  int            `json:"next_supplier_id"`
	NextKoreksiID   int            `json:"next_koreksi_id"`
	NextCatatanID   int            `json:"next_catatan_id"`
	CoaTemplate     int            `json:"coa_template"`
	TrialSampai     time.Time      `json:"trial_sampai,omitzero"`
	AktifSampai     time.Time      `json:"aktif_sampai,omitzero"`
	EksporPertama   time.Time      `json:"ekspor_pertama,omitzero"`
	GaransiDiklaim  time.Time      `json:"garansi_diklaim,omitzero"`
	GaransiAlasan   string         `json:"garansi_alasan,omitempty"`
	Arsip           time.Time      `json:"arsip,omitzero"`
	ArsipAlasan     string         `json:"arsip_alasan,omitempty"`
	NextGerakID     int            `json:"next_gerak_id"`
	TutupBuku       string         `json:"tutup_buku,omitempty"`
	NomorJurnal     map[string]int `json:"nomor_jurnal,omitempty"`
	NextPelangganID int            `json:"next_pelanggan_id"`
}

type invoiceItemRow struct {
	InvoiceID int `json:"invoice_id"`
	No        int `json:"no"`
	finance.InvoiceItem
}

// produkBahanRow: satu baris resep, anak dari produk — sama polanya dengan invoice_items.
type produkBahanRow struct {
	ProdukID int `json:"produk_id"`
	No       int `json:"no"`
	finance.ResepBahan
}

// saveTenantDB menulis ulang semua data satu usaha.
func saveTenantDB(db *sql.DB, t *tenant) error {
	st := t.snapshot()
	t.mu.RLock()
	prof := t.profile
	t.mu.RUnlock()
	l := st.Langganan
	var items []invoiceItemRow
	for _, inv := range st.Invoices {
		for i, it := range inv.Items {
			items = append(items, invoiceItemRow{inv.ID, i + 1, it})
		}
	}
	var bahan []produkBahanRow
	for _, p := range st.Produk {
		for i, b := range p.Bahan {
			bahan = append(bahan, produkBahanRow{p.ID, i + 1, b})
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// ON DELETE CASCADE membersihkan semua tabel anak usaha ini.
	if _, err := tx.Exec(`DELETE FROM tenants WHERE id = $1`, t.id); err != nil {
		return err
	}
	row := tenantRow{t.id, st.NextTxID, st.NextAssetID, st.NextKaryawanID, st.NextInvoiceID,
		st.NextTagihanID, st.NextStokID, st.NextProdukID, st.NextSupplierID, st.NextKoreksiID, st.NextCatatanID, st.CoaTemplate, l.TrialSampai, l.AktifSampai,
		l.EksporPertama, l.GaransiDiklaim, l.GaransiAlasan, st.Arsip, st.ArsipAlasan, st.NextGerakID, st.TutupBuku, st.NomorJurnal, st.NextPelangganID}
	if err := insertRows(tx, "tenants", nil, []tenantRow{row}); err != nil {
		return err
	}
	for _, r := range []struct {
		table string
		rows  any
	}{
		{"business_profiles", one(prof)},
		{"opening_balances", one(st.Opening)},
		{"transactions", st.Transactions},
		{"assets", st.Assets},
		{"karyawan", st.Karyawan},
		{"invoices", st.Invoices},
		{"invoice_items", items},
		{"tagihan", st.Tagihan},
		{"stok", st.Stok},
		{"produk", st.Produk},
		{"produk_bahan", bahan},
		{"supplier", st.Supplier},
		{"pelanggan", st.Pelanggan},
		{"koreksi", st.Koreksi},
		{"catatan_konsultan", st.Catatan},
		{"gerak_stok", st.Gerak},
		{"billing_orders", l.Orders},
	} {
		if err := insertRows(tx, r.table, t.id, r.rows); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// saveAuthDB menulis ulang semua user & sesi.
func saveAuthDB(db *sql.DB, snap auth.StoreSnapshot) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM sessions; DELETE FROM users`); err != nil {
		return err
	}
	if err := insertRows(tx, "users", nil, snap.Users); err != nil {
		return err
	}
	if err := insertRows(tx, "sessions", nil, snap.Sessions); err != nil {
		return err
	}
	return tx.Commit()
}

func insertBugDB(db *sql.DB, rep bugReport) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := insertRows(tx, "bug_reports", nil, []bugReport{rep}); err != nil {
		return err
	}
	return tx.Commit()
}

// updateBugDB menyimpan status & balasan tiket (isi laporan tidak pernah berubah).
func updateBugDB(db *sql.DB, rep bugReport) error {
	balasan, err := json.Marshal(rep.Balasan)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE bug_reports SET status = NULLIF($2, ''), balasan = $3 WHERE id = $1`, rep.ID, rep.Status, balasan)
	return err
}

// saveMasterDB menulis ulang role & anggota tim, lalu MENAMBAHKAN baris audit baru.
func saveMasterDB(db *sql.DB, st master.State, auditBaru []master.Catatan) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM mp_roles; DELETE FROM mp_anggota; DELETE FROM mp_pengaturan`); err != nil {
		return err
	}
	for kunci, isi := range map[string]any{"pengingat": st.Pengingat, "jadwal": st.Jadwal} {
		if reflect.ValueOf(isi).IsNil() {
			continue // belum pernah diatur: bawaan
		}
		b, _ := json.Marshal(isi)
		if _, err := tx.Exec(`INSERT INTO mp_pengaturan (kunci, isi) VALUES ($1, $2)`, kunci, string(b)); err != nil {
			return err
		}
	}
	for _, r := range []struct {
		table string
		rows  any
	}{{"mp_roles", st.Roles}, {"mp_anggota", st.Anggota}, {"mp_audit", auditBaru}} {
		if err := insertRows(tx, r.table, nil, r.rows); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func loadMasterDB(db *sql.DB) (master.State, error) {
	st := master.NewState()
	var roles []master.Role
	for _, q := range []struct {
		dst   any
		query string
	}{
		{&roles, `SELECT jsonb_agg(to_jsonb(r)) FROM mp_roles r`},
		{&st.Anggota, `SELECT jsonb_agg(to_jsonb(a) ORDER BY user_id) FROM mp_anggota a`},
		{&st.Audit, `SELECT jsonb_agg(to_jsonb(x) ORDER BY id) FROM mp_audit x`},
	} {
		if err := selectJSON(db, q.dst, q.query); err != nil {
			return st, err
		}
	}
	if len(roles) > 0 { // DB lama tanpa tim: role bawaan
		st.Roles = roles
	}
	var p finance.Pengingat
	if err := selectJSON(db, &p, `SELECT (SELECT isi FROM mp_pengaturan WHERE kunci = 'pengingat')`); err != nil {
		return st, err
	}
	if p.IntervalHari > 0 {
		st.Pengingat = &p
	}
	var j finance.DaftarJadwal
	if err := selectJSON(db, &j, `SELECT (SELECT isi FROM mp_pengaturan WHERE kunci = 'jadwal')`); err != nil {
		return st, err
	}
	if j.NextID > 0 {
		st.Jadwal = &j
	}
	for _, c := range st.Audit {
		st.NextAuditID = max(st.NextAuditID, c.ID+1)
	}
	return st, nil
}

// saveKatalogDB menulis ulang semua versi template COA & benchmark (puluhan baris).
func saveKatalogDB(db *sql.DB, st katalog.State) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM coa_template; DELETE FROM benchmark_versi; DELETE FROM parameter_pajak`); err != nil {
		return err
	}
	if err := insertRows(tx, "parameter_pajak", nil, st.Parameter); err != nil {
		return err
	}
	if err := insertRows(tx, "coa_template", nil, st.Template); err != nil {
		return err
	}
	if err := insertRows(tx, "benchmark_versi", nil, st.Benchmark); err != nil {
		return err
	}
	return tx.Commit()
}

// loadKatalogDB: DB tanpa katalog (sebelum fitur ini) = katalog bawaan.
func loadKatalogDB(db *sql.DB) (katalog.State, error) {
	st := katalog.NewState()
	var tm []katalog.Template
	var bm []katalog.Benchmark
	if err := selectJSON(db, &tm, `SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM coa_template t`); err != nil {
		return st, err
	}
	if err := selectJSON(db, &bm, `SELECT jsonb_agg(to_jsonb(b) ORDER BY id) FROM benchmark_versi b`); err != nil {
		return st, err
	}
	var pm []katalog.ParameterVersi
	if err := selectJSON(db, &pm, `SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM parameter_pajak p`); err != nil {
		return st, err
	}
	if len(pm) > 0 {
		st.Parameter = pm
	}
	for _, p := range st.Parameter {
		st.NextParameterID = max(st.NextParameterID, p.ID+1)
	}
	if len(tm) > 0 {
		st.Template = tm
	}
	if len(bm) > 0 {
		st.Benchmark = bm
	}
	for _, t := range st.Template {
		st.NextTemplateID = max(st.NextTemplateID, t.ID+1)
	}
	for _, b := range st.Benchmark {
		st.NextBenchmarkID = max(st.NextBenchmarkID, b.ID+1)
	}
	return st, nil // versi bawaan disegarkan di NewServer (segarkanBawaan)
}

// savePengetahuanDB menulis ulang artikel & sumber pengetahuan.
// ponytail: tulis ulang semua baris tiap perubahan; isi PDF bisa ratusan KB per sumber —
// pindah ke upsert per baris bila sumber melewati ~200 atau simpan terasa lambat.
func savePengetahuanDB(db *sql.DB, st pengetahuan.State) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM artikel; DELETE FROM sumber_pengetahuan`); err != nil {
		return err
	}
	if err := insertRows(tx, "artikel", nil, st.Artikel); err != nil {
		return err
	}
	if err := insertRows(tx, "sumber_pengetahuan", nil, st.Sumber); err != nil {
		return err
	}
	return tx.Commit()
}

func loadPengetahuanDB(db *sql.DB) (pengetahuan.State, error) {
	st := pengetahuan.NewState()
	if err := selectJSON(db, &st.Artikel, `SELECT jsonb_agg(to_jsonb(a) ORDER BY id) FROM artikel a`); err != nil {
		return st, err
	}
	if err := selectJSON(db, &st.Sumber, `SELECT jsonb_agg(to_jsonb(x) ORDER BY id) FROM sumber_pengetahuan x`); err != nil {
		return st, err
	}
	for _, a := range st.Artikel {
		st.NextArtikelID = max(st.NextArtikelID, a.ID+1)
	}
	for _, x := range st.Sumber {
		st.NextSumberID = max(st.NextSumberID, x.ID+1)
	}
	return st, nil
}

// loadDB memuat semua data; ok=false bila DB masih kosong (belum ada usaha).
func (s *Server) loadDB() (ok bool, err error) {
	var ids []int
	if err := selectJSON(s.db, &ids, `SELECT coalesce(jsonb_agg(id ORDER BY id), '[]') FROM tenants`); err != nil {
		return false, err
	}
	if len(ids) == 0 {
		return false, nil
	}
	s.tenants = map[int]*tenant{}
	for _, id := range ids {
		t := newTenant(id)
		if id == 1 {
			t = demoTenant()
		}
		if err := loadTenantDB(s.db, t); err != nil {
			return false, fmt.Errorf("usaha %d: %w", id, err)
		}
		s.tenants[id] = t
	}
	var snap auth.StoreSnapshot
	if err := selectJSON(s.db, &snap.Users, `SELECT coalesce(jsonb_agg(to_jsonb(u) ORDER BY id), '[]') FROM users u`); err != nil {
		return false, err
	}
	if err := selectJSON(s.db, &snap.Sessions, `SELECT coalesce(jsonb_agg(to_jsonb(x)), '[]') FROM sessions x WHERE expires > now()`); err != nil {
		return false, err
	}
	s.users = auth.Restore(snap)
	s.bugs = &bugStore{path: dataPath("bugs.json"), mem: map[string][]byte{}}
	if s.master, err = loadMasterDB(s.db); err != nil {
		return false, fmt.Errorf("tim MasterPajak AI: %w", err)
	}
	if s.katalog, err = loadKatalogDB(s.db); err != nil {
		return false, fmt.Errorf("katalog: %w", err)
	}
	if s.pengetahuan, err = loadPengetahuanDB(s.db); err != nil {
		return false, fmt.Errorf("pengetahuan: %w", err)
	}
	return true, selectJSON(s.db, &s.bugs.list, `SELECT coalesce(jsonb_agg(to_jsonb(b) ORDER BY id), '[]') FROM bug_reports b`)
}

func loadTenantDB(db *sql.DB, t *tenant) error {
	var row tenantRow
	if err := selectJSON(db, &row, `SELECT to_jsonb(t) FROM tenants t WHERE id = $1`, t.id); err != nil {
		return err
	}
	st := state{NextTxID: row.NextTxID, NextAssetID: row.NextAssetID, NextKaryawanID: row.NextKaryawanID, NextInvoiceID: row.NextInvoiceID,
		NextTagihanID: row.NextTagihanID, NextStokID: row.NextStokID, NextProdukID: row.NextProdukID, NextSupplierID: row.NextSupplierID,
		NextKoreksiID: row.NextKoreksiID, NextCatatanID: row.NextCatatanID, NextGerakID: row.NextGerakID, TutupBuku: row.TutupBuku, NomorJurnal: row.NomorJurnal, NextPelangganID: row.NextPelangganID, CoaTemplate: row.CoaTemplate, Arsip: row.Arsip, ArsipAlasan: row.ArsipAlasan,
		Langganan: &langganan{TrialSampai: row.TrialSampai, AktifSampai: row.AktifSampai,
			EksporPertama: row.EksporPertama, GaransiDiklaim: row.GaransiDiklaim, GaransiAlasan: row.GaransiAlasan}}
	var items []invoiceItemRow
	var bahan []produkBahanRow
	for _, q := range []struct {
		dst   any
		query string
	}{
		{&st.Transactions, rowsOf("transactions", "id")},
		{&st.Assets, rowsOf("assets", "id")},
		{&st.Karyawan, rowsOf("karyawan", "id")},
		{&st.Invoices, rowsOf("invoices", "id")},
		{&items, rowsOf("invoice_items", "invoice_id, no")},
		{&st.Tagihan, rowsOf("tagihan", "id")},
		{&st.Stok, rowsOf("stok", "id")},
		{&st.Produk, rowsOf("produk", "id")},
		{&bahan, rowsOf("produk_bahan", "produk_id, no")},
		{&st.Supplier, rowsOf("supplier", "id")},
		{&st.Pelanggan, rowsOf("pelanggan", "id")},
		{&st.Koreksi, rowsOf("koreksi", "id")},
		{&st.Catatan, rowsOf("catatan_konsultan", "id")},
		{&st.Gerak, rowsOf("gerak_stok", "id")},
		{&st.Langganan.Orders, rowsOf("billing_orders", "waktu, id")},
	} {
		if err := selectJSON(db, q.dst, q.query, t.id); err != nil {
			return err
		}
	}
	byID := map[int]int{} // id invoice → indeks
	for i, inv := range st.Invoices {
		byID[inv.ID] = i
	}
	for _, it := range items {
		inv := &st.Invoices[byID[it.InvoiceID]]
		inv.Items = append(inv.Items, it.InvoiceItem)
	}
	produkByID := map[int]int{} // id produk → indeks
	for i, p := range st.Produk {
		produkByID[p.ID] = i
	}
	for _, b := range bahan {
		p := &st.Produk[produkByID[b.ProdukID]]
		p.Bahan = append(p.Bahan, b.ResepBahan)
	}
	if err := selectJSON(db, &st.Opening, `SELECT to_jsonb(o) - 'tenant_id' FROM opening_balances o WHERE tenant_id = $1`, t.id); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	var prof *finance.BusinessProfile
	if err := selectJSON(db, &prof, `SELECT to_jsonb(p) - 'tenant_id' FROM business_profiles p WHERE tenant_id = $1`, t.id); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	t.restore(&st)
	t.profile = prof
	return nil
}
