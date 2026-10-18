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

## Fase 2 (sedang berjalan) — Laporan Keuangan & Arus Kas
- Selesai (TDD): [ledger.go](file:///home/reyfuu/pajak/internal/finance/ledger.go) menghitung Laba Rugi dan Arus Kas dari `LedgerEntry`, dengan invarian `laba − piutang baru + utang baru = kenaikan kas`.
- Endpoint: `GET /api/v1/reports/financial` dan `GET /api/v1/reports/cashflow`; halaman Laporan Keuangan dan Arus Kas membaca endpoint ini, bukan angka hardcode.
- Celah yang diketahui:
  - Ledger masih `SeedLedger` statis; `POST /api/v1/transactions` belum menulis ke ledger, jadi laporan belum ikut berubah.
  - KPI dashboard (`SeedKedaiKopiSenja`) dan ledger adalah dua sumber angka yang cocok secara manual. Gabungkan jadi satu sumber.
  - Saldo awal kas `80_000_000` masih hardcode di handler.
  - Neraca belum ada (FRD FR-REP-01).
  - Frontend belum diuji di browser (hanya cek sintaks JS dan respons API).

## Langkah Selanjutnya (Sesi Berikutnya)
- Turunkan KPI dashboard dari ledger, dan buat POST transaksi menulis ke ledger.
- Neraca sederhana + tes.

- Koneksi opsional ke PostgreSQL 16/17 menggunakan skema di [TRD.md](file:///home/reyfuu/pajak/docs/TRD.md) (saat ini menggunakan in-memory store sesuai prinsip `ponytail` Fase 1).
- Ekspor PDF untuk Invoice & Kertas Kerja Pajak UMKM PP 55/2022.
- Integrasi webhook pembayaran QRIS Rp149.000.

