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
- **Akun Produksi**: email seed sama (`owner@`, `admin@`, `kasir@kedaikopisenja.id`), password dari env `SEED_PASSWORD` di VPS (file `.env` di direktori repo VPS, tidak di-commit). Password demo `senja2026` hanya berlaku lokal/dev; panel akun demo di halaman login hanya muncul di `npm run dev`.
- **Container**: Docker `laporan-bisnis` di `127.0.0.1:8085->8080/tcp` (hanya lewat nginx).
- **Nginx Reverse Proxy**: `/etc/nginx/sites-available/jurnal-pajak` dikonfigurasi ke `http://127.0.0.1:8085` dengan sertifikat SSL Let's Encrypt aktif.
- **9Router**: Diperbarui ke versi terbaru `v0.5.104` di server hermes (layanan systemd `9router.service` aktif).
- **GitHub Repository**: [https://github.com/reyfuu/jurnal.git](https://github.com/reyfuu/jurnal.git)
  - Branch `backup/main-lama`: Menyimpan snapshot kode Laravel lama.
  - Branch `main`: Kode baru Go + React Vite + Auth RBAC + AI Agent Bubble (Laporan Bisnis).

## Fase 4 — Multi-Period Dummy Data & Filter Komprehensif di Seluruh View (Live & Selesai)
- **Multi-Period Data Seed (Januari s.d. Agustus 2026)**:
  - 8 bulan data realistis di [internal/finance/seed.go](file:///home/reyfuu/pajak/internal/finance/seed.go) & [internal/finance/transaction.go](file:///home/reyfuu/pajak/internal/finance/transaction.go): KPI operasional, tren 11 bulan, transaksi, performa channel, kewajiban (piutang/utang), stok inventori, dan PPh Final UMKM PP 55/2022.
  - Endpoint `GET /api/v1/periods`: mengembalikan daftar 8 periode yang tersedia beserta label bulannya.
- **Global Period Switcher**:
  - Disediakan dropdown pemilih periode di header workspace dan kartu header Dashboard.
  - Mengubah periode akan memicu reload terkoordinasi untuk `/dashboard`, `/reports/financial`, `/reports/cashflow`, `/reports/balance-sheet`, dan memberikan konteks periode ke AI Agent Bubble (`/api/v1/ai/chat`).
- **Filter Komprehensif di Seluruh Modul**:
  - **Transaksi ([views/Transaksi.tsx](file:///home/reyfuu/pajak/frontend/src/views/Transaksi.tsx))**:
    - Pencarian teks langsung (keterangan, item, pihak, channel, jenis).
    - Filter Bulan (`Semua bulan` + 8 opsi bulan).
    - Filter Arah Uang (`Semua arah`, `Pemasukan (Masuk)`, `Pengeluaran (Keluar)`).
    - Filter Channel Penjualan (`Semua channel`, `Dine-in`, `GoFood`, `GrabFood`, `ShopeeFood`, `Supplier`, `BCA Bisnis`, `B2B Tempo`).
    - Filter Jenis Transaksi (`Semua jenis` + 18 jenis transaksi).
  - **Piutang & Utang ([views/Operations.tsx](file:///home/reyfuu/pajak/frontend/src/views/Operations.tsx))**:
    - Pencarian no invoice, nama pelanggan, supplier.
    - Filter Jenis (`Semua Jenis`, `Piutang Masuk (Pelanggan)`, `Utang Keluar (Supplier)`).
    - Filter Status Umur (`Semua Status`, `Lewat 30 Hari (>30 Hari)`, `Belum Jatuh Tempo / Lancar`).
    - Mini KPI cards menampilkan total nominal piutang dan utang dari baris yang difilter.
  - **Stok & Gudang ([views/Operations.tsx](file:///home/reyfuu/pajak/frontend/src/views/Operations.tsx))**:
    - Pencarian SKU atau nama item / bahan.
    - Filter Kategori (`Semua Kategori`, `Bahan Baku`, `Produk Jadi`).
    - Filter Status Stok (`Semua Status`, `Stok Menipis (Perlu Restock)`, `Stok Aman / Resep BOM Aktif`).
- **Pengujian & Deployment**:
  - 100% Go tests lulus (`internal/finance`, `internal/auth`, `internal/ai`, `cmd/server`).
  - 100% Vitest frontend lulus (`transaksi.test.ts`, `charts.test.ts`, `format.test.ts`).
  - Di-build dan di-deploy ke VPS `31.97.66.119` di port `8085` dengan verifikasi live di [https://pajak.aipreneur.co.id](https://pajak.aipreneur.co.id).

## Fase 5 — Hardening Keamanan Produksi
- Cookie `sid` kini `Secure` di balik nginx (`isHTTPS` membaca `X-Forwarded-Proto`), `MaxAge` 12 jam; sesi kedaluwarsa di server (`auth.SessionTTL`) dan dicabut saat password diganti / user dihapus.
- Rate limit login: 5 gagal per IP (`X-Real-IP`) dalam 15 menit → 429.
- Header keamanan: `nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy`. CSP belum (ditandai `ponytail:`).
- `deploy.sh` wajib `SEED_PASSWORD`, port container hanya 127.0.0.1.
- **Wajib di nginx** `jurnal-pajak`: `proxy_set_header X-Forwarded-Proto $scheme;` dan `proxy_set_header X-Real-IP $remote_addr;` (tanpa X-Real-IP semua klien terlihat sebagai 1 IP dan header bisa dipalsukan).

## Fase A & A+ — "Staf Keuangan Virtual" untuk UMKM Menengah (Okt 2026)
Pain point baru (BRD §1 masalah 5): UMKM menengah tidak mampu atau bingung merekrut staf keuangan. Roadmap A–E ada di PRD §5.
- **Tanpa emoji** di UI, jawaban AI, dan laporan (`TestAnswer_NoEmoji`, aturan di DESIGN.md §6).
- **Simulasi Harga + BEP**: `finance.BreakEven` (BEP unit/hari/Rp, laba bersih pada volume, unit untuk target laba). Biaya tetap default = beban operasional periode berjalan.
- **Peringatan kas minus**: `finance.CashWarnings` (saldo harian dari transaksi lunas). Tampil di Dashboard dan setelah simpan transaksi; transaksi tetap tersimpan.
- **Laporan Penjualan** (pola skinku-b2b-portal): `GET /api/v1/reports/sales?granularity=day|week|month`. Seed penjualan dipecah per item menu (total tetap).
- **Sidebar sticky** setinggi layar, menu scroll sendiri (pola skinku / AIpreneur).
- **Profil usaha** (onboarding saat Owner pertama login): jenis usaha perdagangan/jasa/manufaktur, bentuk usaha OP/PT Perorangan/CV/PT, PTKP, PKP. `GET|PUT /api/v1/business`, disimpan di `$DATA_DIR/business.json` (VPS: `/var/www/laporan-bisnis/data`, volume Docker). Menu Pengaturan > Profil Usaha.
- **Perencanaan Pajak** (`internal/tax`, port inti *Tax_Planning_Calculator_V2.1.xlsm* BSV Consulting): 6 skema (PP 20/2026 OP & Badan, Pasal 17, NPPN 50%, Badan 22%, Pasal 31E) + perbandingan bentuk badan dengan PPh dividen 10%. `POST /api/v1/tax/planning`, default dari transaksi + profil. Backlog dari template: gross-up PPh 21/23/sewa, PPN dibebankan vs ditanggung, beli vs sewa aset, NPPN per KLU (tabel 1.435 norma).

## Langkah Berikutnya (Pengembangan Lanjutan)
0. **Prioritas**: persistensi SQLite (data masih in-memory, hilang tiap deploy) → multi-tenant per UMKM → KPI dihitung murni dari transaksi.
1. Melengkapi fitur cetak / ekspor laporan ke format PDF dan Excel (.xlsx) langsung dari UI.
2. Melengkapi penagihan invoice & pelunasan parsial/lunas pada modul Piutang & Utang.
3. Migrasi persistensi PostgreSQL sesuai skema di [TRD.md](file:///home/reyfuu/pajak/docs/TRD.md) saat kebutuhan database relasional diperlukan.
4. Refresh token sesi & CSP.



