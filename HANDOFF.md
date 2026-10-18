# HANDOFF.md — Status Sesi & Konteks Berikutnya

## Ringkasan yang Sudah Selesai
1. **[agent.md](file:///home/reyfuu/pajak/agent.md) & [AGENTS.md](file:///home/reyfuu/pajak/AGENTS.md)**:
   - Mengunci protokol wajib setiap sesi: `rtk`, `graphify`, `grillme`, `superpowers`, `antislop`, `taste`, `21dev`, `shadcn/ui`, `ponytail`, `ui ux pro max`, `kokonut ui`, `impeccable`, `grillme with docs`, `skillspector`, `agent browser`, `tdd`, `handoff`, dan `improve-codebase-architecture` / `setup-matt-pocock-skills`.
2. **Dokumen Spesifikasi Lengkap (Skala 100 User UMKM)**:
   - [BRD.md](file:///home/reyfuu/pajak/docs/BRD.md): Unit economics Rp149.000 × 100 user, biaya VPS < Rp150rb/bulan, margin > 90%.
   - [FRD.md](file:///home/reyfuu/pajak/docs/FRD.md): Spesifikasi fungsional 6 KPI utama, 3 grafik tengah, 2 kartu bawah, laporan keuangan, arus kas, piutang >30 hari, simulasi harga komisi aplikasi, dan Pajak UMKM PP 55/2022.
   - [TRD.md](file:///home/reyfuu/pajak/docs/TRD.md): Analisis komparasi mendalam **ElysiaJS (Bun) vs Go (Golang)** untuk 100 user UMKM, skema PostgreSQL (`BIGINT` Rupiah), dan kontrak API.
   - [PRD.md](file:///home/reyfuu/pajak/docs/PRD.md): Persona UMKM, spesifikasi warna & tipografi *Dark Obsidian Fintech*, user stories, dan roadmap.
   - [DESIGN.md](file:///home/reyfuu/pajak/DESIGN.md) & [DESIGN_BRIEF.md](file:///home/reyfuu/pajak/docs/DESIGN_BRIEF.md): Design Brief eksekutif, Design Tokens lengkap (hex/RGBA), aturan tipografi (`tabular-nums`), arsitektur layout Bento Grid, anatomi 6 KPI card & 3 grafik utama, serta *Anti-Slop Guardrails*.
3. **Implementasi Fase 1 (Go + TDD + Interactive Web UI)**:
   - Engine finansial & pajak UMKM (`internal/finance/engine.go` & `internal/finance/engine_test.go`) dibangun menggunakan **TDD (Red-Green-Refactor)**.
   - Server HTTP Go (`cmd/server/main.go` & `cmd/server/main_test.go`) dan antarmuka [web/index.html](file:///home/reyfuu/pajak/web/index.html) persis mengikuti desain referensi **Laporan Bisnis by Akademi Pajak** (`Kedai Kopi Senja`).

## Cara Menjalankan Aplikasi
```bash
# Backend (Go ≥ 1.24, go.mod memakai crypto/pbkdf2 stdlib)
go test ./...
# Frontend (Vite + React + shadcn/ui di frontend/, hasil build ke web/dist)
cd frontend && npm ci && npx vitest run && npm run build && cd ..
go build -o bin/laporan-bisnis ./cmd/server
./bin/laporan-bisnis
# Buka http://localhost:8080 → halaman login
# Dev frontend dengan hot reload: jalankan server Go di :8080, lalu `cd frontend && npm run dev` (proxy /api ke :8080)
```

Catatan mesin lokal: `go` di PATH masih 1.22.4 sementara `GOROOT` menunjuk ke 1.27.1, sehingga build gagal ("version does not match"). Pakai `/home/reyfuu/.local/opt/go1.27.1/bin/go` atau rapikan PATH.

## Fase 3 — Frontend React, Login & Manajemen Akses
- `web/index.html` (vanilla) diganti SPA di [frontend/](file:///home/reyfuu/pajak/frontend). Tampilan dipertahankan: kelas CSS lama ada di `frontend/src/index.css`, token shadcn dipetakan ke palet Dark Obsidian. Grafik SVG tetap buatan sendiri (`src/lib/charts.ts`, ber-test).
- Navbar marketing dihapus; pintu masuk sekarang halaman login.
- Auth ([internal/auth](file:///home/reyfuu/pajak/internal/auth/auth.go)): user in-memory, password PBKDF2-SHA256, sesi cookie `sid` HttpOnly. Akun demo (password `senja2026`): `owner@`, `admin@`, `kasir@kedaikopisenja.id`.
- Role tetap: **Owner** (semua + Pengguna & Akses), **Admin Keuangan** (semua kecuali kelola pengguna), **Kasir** (Dashboard + Transaksi). `auth.Roles` adalah satu-satunya sumber hak akses: server menolak API lewat `guard(view)` (401/403), frontend menyembunyikan menu dari `views` di `/auth/me`.
- Endpoint baru: `POST /api/v1/auth/login`, `POST /api/v1/auth/logout`, `GET /api/v1/auth/me`, `GET|POST /api/v1/users`, `PUT|DELETE /api/v1/users/{id}` (khusus Owner). Pengaman: tidak bisa hapus akun sendiri, minimal satu Owner aktif, email unik.
- Dockerfile kini multi-stage: Node (build frontend) → Go → scratch.

## Fase 2 (Sedang Berjalan & Terverifikasi) — Laporan Keuangan, Arus Kas & Neraca
- Selesai (TDD): [ledger.go](file:///home/reyfuu/pajak/internal/finance/ledger.go) menghitung:
  1. Laba Rugi (`IncomeStatement`)
  2. Arus Kas (`CashFlow`) dengan invarian rekonsiliasi: `Laba - Piutang Baru + Utang Baru = Kenaikan Kas`
  3. Neraca (`BalanceSheet`) dengan formula `Aset (Kas + Piutang + Persediaan) = Kewajiban (Utang) + Modal (Modal Awal + Laba Berjalan)` (`is_balanced: true`).
- Endpoint aktif:
  - `GET /api/v1/reports/financial`
  - `GET /api/v1/reports/cashflow`
  - `GET /api/v1/reports/balance-sheet`
  - `POST /api/v1/transactions`: Transaksi baru otomatis masuk ke ledger dan meng-update seluruh laporan & KPI dashboard secara terpadu.
- Halaman Laporan Keuangan menampilkan Laba Rugi dan Neraca bersanding dengan badge `✓ Aktiva = Pasiva (Seimbang)`.

## Status Deploy VPS & Produksi (Selesai & Live — Fase 3 Aktif)
- **VPS**: `31.97.66.119` (hostname: `hermes`)
- **Domain**: [https://pajak.aipreneur.co.id](https://pajak.aipreneur.co.id)
- **Aplikasi Aktif**: Single Page Application (React 19 + Vite + Tailwind/shadcn tokens) dengan sistem autentikasi RBAC (Owner, Admin Keuangan, Kasir), backend Go, serta **AI Agent Bubble Konsultan Bisnis & Pajak**.
- **AI Agent Bubble**: Melayang di pojok kanan bawah, terhubung ke engine Go (`POST /api/v1/ai/chat`) untuk meringkas laporan eksekutif, menganalisis arus kas/piutang >30 hari, dan menjawab aturan pajak UMKM (PP 55/2022).
- **Panduan Penggunaan PDF**: Tersedia lengkap 8 halaman beserta tangkapan layar antarmuka asli di [docs/Panduan_Laporan_Bisnis_Akademi_Pajak.pdf](file:///home/reyfuu/pajak/docs/Panduan_Laporan_Bisnis_Akademi_Pajak.pdf).
- **Akun Demo**:
  - Owner: `owner@kedaikopisenja.id` (password: `senja2026`)
  - Admin: `admin@kedaikopisenja.id` (password: `senja2026`)
  - Kasir: `kasir@kedaikopisenja.id` (password: `senja2026`)
- **Container**: Docker `laporan-bisnis` aktif di port `8085` (`0.0.0.0:8085->8080/tcp`).
- **Nginx Reverse Proxy**: `/etc/nginx/sites-available/jurnal-pajak` dikonfigurasi ke `http://127.0.0.1:8085` dengan sertifikat SSL Let's Encrypt aktif.
- **9Router**: Diperbarui ke versi terbaru `v0.5.104` di server hermes (layanan systemd `9router.service` aktif).
- **GitHub Repository**: [https://github.com/reyfuu/jurnal.git](https://github.com/reyfuu/jurnal.git)
  - Branch `backup/main-lama`: Menyimpan snapshot kode Laravel lama.
  - Branch `main`: Kode baru Go + React Vite + Auth RBAC + AI Agent Bubble (Laporan Bisnis).

## Fase 4 — CRUD Transaksi & Impor Excel (branch `feat/transaksi-crud`, belum di-merge)
Meniru demo `bisnis.pakarpajak.id/app?demo=1#transaksi`.
- **Sumber kebenaran = transaksi bertanggal** ([internal/finance/transaction.go](internal/finance/transaction.go)): 18 jenis (Penjualan, Beli Bahan / Stok, Gaji & Upah, Setor Modal, Bayar Cicilan, Ambil Uang Pribadi, dst.) dipetakan ke pos laporan. `ToLedger` menurunkan buku besar untuk laporan dan AI. Seed 8 bulan di [seed.go](internal/finance/seed.go) diurai jadi transaksi oleh `SeedTransactions`.
- **Laporan dan KPI dashboard dihitung ulang** dari transaksi periode tersebut (`ApplyKPIs`); grafik riwayat dan sparkline tetap seed. Neraca tetap seimbang untuk modal, pinjaman, aset, cicilan, dan prive. Kas awal per periode diambil dari `OpeningCash(period)`.
- **Endpoint**:
  - `GET|POST /api/v1/transactions` untuk semua role yang punya menu Transaksi;
  - `PUT|DELETE /api/v1/transactions/{id}` butuh izin `transaksi-kelola` (Owner, Admin);
  - `GET /api/v1/transactions/template` (.xlsx);
  - `POST /api/v1/transactions/import` (pratinjau; `?commit=1` untuk menyimpan; Owner/Admin).
- **Impor Excel tanpa dependensi**: [internal/xlsx](internal/xlsx/xlsx.go) membaca dan menulis .xlsx dengan stdlib, sudah diuji bolak-balik dengan LibreOffice. [finance.ParseImport](internal/finance/import.go) mengenali header bebas, alias jenis ("gaji", "kulakan", ...), tanggal YYYY-MM-DD / DD/MM/YYYY / seri Excel, dan nominal "Rp 1.250.000"; baris bermasalah dilaporkan per nomor baris.
- **Frontend**: [views/Transaksi.tsx](frontend/src/views/Transaksi.tsx) dan [views/ImportDialog.tsx](frontend/src/views/ImportDialog.tsx).
- **Catatan**: tanggal default "Catat transaksi" adalah hari ini. Transaksi di luar 8 periode seed (Jan–Agu 2026) tampil di daftar tapi belum bisa dipilih sebagai periode laporan.

## Langkah Berikutnya (Pengembangan Lanjutan)
1. Melengkapi Piutang & Utang (pelunasan / penagihan invoice).
1a. Daftar periode laporan dibuat dinamis dari bulan-bulan transaksi (sekarang statis Jan–Agu 2026).
2. Migrasi persistensi PostgreSQL sesuai skema di [TRD.md](file:///home/reyfuu/pajak/docs/TRD.md), termasuk tabel `users` & `sessions` (sekarang user in-memory).
3. Menambahkan rate limit dan kedaluwarsa sesi pada login.



