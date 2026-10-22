# Technical Requirements Document — Platform Akuntansi Jasa & Dagang

**Status:** Draft, stack-agnostic sampai repository dan kemampuan tim diaudit  
**Input:** [Handoff Coding Agent](../Handoff_Accounting_Platform_Coding_Agent%20%281%29.md) · [BRD](BRD.md) · [FRD](FRD.md)

## 1. Prinsip teknis

1. Audit dan pertahankan stack existing jika repository tersedia; jangan membangun ulang tanpa alasan konkret.
2. Jika belum ada repository, mulai dari modular monolith dan database relasional transaksional. Jangan menambah microservices, vector database, atau AI untuk hitung saldo inti.
3. Semua jalur UI, API, dan impor memakai layanan domain posting yang sama.
4. Dokumen posted immutable; koreksi memakai reversal dan dokumen pengganti.
5. Ledger posted adalah sumber kebenaran laporan. Proyeksi/cache harus dapat dibangun ulang dan direkonsiliasi.

## 2. Keputusan baseline (belum disahkan untuk produksi)

| Area | Baseline handoff | Catatan |
|---|---|---|
| Buku | Akrual, double-entry, IDR | Validasi dengan penanggung jawab akuntansi. |
| Perusahaan | Satu perusahaan aktif per sesi; setiap data memiliki `company_id` | Semua query, ekspor, pencarian, dan lampiran harus terisolasi. |
| Persediaan | Satu gudang; perpetual moving weighted average | Tidak mengklaim sebagai default Accurate. |
| Presisi | Quantity `NUMERIC(20,6)`, unit cost `NUMERIC(24,8)`, ledger amount `NUMERIC(24,2)` | Gunakan Decimal; jangan float. Rounding half-up; ledger adalah otoritas. |
| Stok | Tolak stok negatif dan mutasi bertanggal sebelum mutasi terakhir item/gudang | Revaluasi historis dan replay costing di luar MVP. |
| Posting | Posting invoice/bill menjadi titik pengeluaran/penerimaan stok | Delivery/receipt independen di luar MVP. |

## 3. Arsitektur logis

- **UI web** menangani draft, validasi yang membantu pengguna, preview posting, pencarian, tabel, dan drill-down.
- **API/auth** memeriksa identitas, membership perusahaan, hak tindakan, payload, dan batas impor/ekspor.
- **Domain accounting** menampung aturan COA, posting, AR/AP, alokasi pembayaran, costing stok, lock periode, dan laporan.
- **Database relasional** menyimpan dokumen sumber, jurnal, mutasi stok, alokasi, audit, dan idempotency dengan constraint serta foreign key.
- **Import/export** memakai domain dan filter otorisasi yang sama; batch impor tidak boleh membuat posting parsial tanpa kebijakan eksplisit.

Jangan mengunci framework/ORM sebelum audit repository, deployment, kompetensi tim, dan batas transaksi database.

## 4. Model data konseptual

Entitas minimum (nama tabel dapat mengikuti konvensi stack): `companies`, `users`, `company_memberships`, `roles/permissions`, `fiscal_periods`, `accounts`, `posting_account_mappings`, `parties`, `items`, `units`, `warehouses`, `sales_invoices/lines`, `purchase_bills/lines`, `payments`, `payment_allocations`, `journal_entries/lines`, `stock_movements`, `opening_balance_batches/details`, `audit_events`, `idempotency_records`, `import_batches/errors`.

Gunakan constraint database untuk uniqueness dan integritas lintas referensi. Saldo awal AR/AP menyimpan rincian pihak dan outstanding; saldo awal stok harus cocok dengan akun kontrol persediaan. Hindari menghitung saldo awal dua kali antara GL dan subledger.

## 5. Kontrak posting dan integritas

Posting harus atomik untuk dokumen, jurnal, mutasi stok, alokasi, dan audit. Satu kegagalan me-rollback seluruh perubahan. Terapkan idempotency key dengan unique constraint. Kunci dokumen dan item/gudang terkait saat posting agar dua request bersamaan tidak menjual stok yang sama. Pastikan total debit=kredit pada presisi ledger.

| Peristiwa | Debit | Kredit |
|---|---|---|
| Setoran modal | Bank | Modal |
| Bill barang | Persediaan | Utang usaha |
| Pembayaran pemasok | Utang usaha | Bank |
| Invoice kredit | Piutang usaha | Pendapatan |
| Barang terjual | HPP | Persediaan |
| Pembayaran pelanggan | Bank | Piutang usaha |
| Biaya melalui bank | Beban terkait | Bank |
| Transfer | Bank tujuan | Bank sumber |

Bill yang langsung dibayar mem-posting bill dan pembayaran terkait dalam satu transaksi database, tanpa jurnal ganda.

## 6. Keamanan dan isolasi

- Otorisasi server untuk setiap operasi; validasi `company_id` pada semua object ID dan relasi.
- Cegah akses lintas perusahaan melalui detail, search, import, export, laporan, dan lampiran.
- Kredensial melalui environment/secret manager; jangan commit secret.
- Validasi MIME/format, ukuran, nama, dan isi file impor; batasi ekspor/lampiran sesuai otorisasi.
- Data keuangan tidak dikirim ke model AI secara default. AI tahap lanjut tidak menulis journal posted atau saldo otoritatif.
- Periode terkunci memblokir posting, reversal, dan saldo awal pada semua jalur.

## 7. Antarmuka logis

Endpoint spesifik ditentukan setelah audit stack. API minimum harus menyediakan operasi terotorisasi untuk company/setup, COA, parties/items, fiscal periods, invoice/bill draft-post-reverse, payments/allocations, cash/bank, journals, stock card, reports/drill-down, opening balance/import preview/commit, audit, lock/reopen, dan ekspor. Mutasi state memakai idempotency key dan mengembalikan referensi posting serta error validasi terstruktur. Jangan mengekspos operasi yang melewati domain posting.

## 8. Pengujian dan observabilitas

- Unit test Decimal, rounding, moving average, jurnal, AR/AP, reversal, dan lock.
- Integration test transaksi atomik, constraint, idempotency, serta rollback.
- Concurrency test untuk stok terakhir dan double-submit.
- Tenant-isolation test pada API, query, import/export, dan lampiran.
- Acceptance test memakai fixture angka di [FRD](FRD.md); bandingkan GL, subledger, stok, dan laporan.
- Restore test di lingkungan uji sebelum klaim backup siap produksi.

Target performa, retensi, RPO/RTO, hosting, dan mekanisme backup belum diputuskan. Tetapkan setelah repository dan kebutuhan operasi diaudit.
