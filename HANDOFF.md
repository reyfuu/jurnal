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
go test -v ./...
go build -o bin/laporan-bisnis ./cmd/server
./bin/laporan-bisnis
# Buka http://localhost:8080
```

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
- Tampilan Frontend [web/index.html](file:///home/reyfuu/pajak/web/index.html): Halaman Laporan Keuangan menampilkan Laba Rugi dan Neraca bersanding dengan badge `✓ Aktiva = Pasiva (Seimbang)`.
- Container Docker `laporan-bisnis` aktif di `http://localhost:8080`.

## Langkah Berikutnya & Target Deploy VPS
1. Konfirmasi detail VPS target (IP, user SSH, port, key/password) atau pembuatan script otomatisasi `deploy.sh`.
2. Melengkapi Piutang & Utang (pelunasan / penagihan invoice).
3. Migrasi persistensi PostgreSQL sesuai skema di [TRD.md](file:///home/reyfuu/pajak/docs/TRD.md).


