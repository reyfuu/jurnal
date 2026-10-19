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

## Fase B — Persistensi Data (Okt 2026)
- **Keputusan**: snapshot JSON atomik `$DATA_DIR/state.json` (transaksi, user dengan hash password, sesi aktif), bukan SQLite. Alasan: `modernc.org/sqlite` terbaru butuh Go 1.26 + dependensi besar, sementara data 1 usaha muat di memori. Ganti ke database saat multi-tenant.
- Ditulis ulang setelah setiap request pengubah data yang sukses (middleware `persisting` di `cmd/server/persist.go`): login/logout, CRUD user, CRUD + impor transaksi. Penulisan diserialisasi (`saveMu`), file 0600.
- Boot pertama tanpa `state.json`: seed dibuat dengan `SEED_PASSWORD` lalu langsung disimpan. Setelah itu **`SEED_PASSWORD` tidak berpengaruh lagi**: ganti password lewat menu Pengguna.
- `state.json` rusak → server menolak start (log fatal) agar data tidak tertimpa seed. Backup: salin `/var/www/laporan-bisnis/data/`.
- Tanpa `DATA_DIR` (test, dev lokal) tetap in-memory.

## Fase C — CoA Otomatis Perdagangan (Okt 2026)
- **Sumber**: `Daftar COA.xlsx` (sheet "COA Baru", review akuntan, 72 akun) → `internal/coa/coa.go` (`Master`). Perubahan disengaja: 2113 Utang Karyawan diganti **1407 Piutang Karyawan** (substansi kasusnya tagihan ke karyawan).
- Kondisi aktif mengikuti profil: 2115 Utang Dividen hanya PT/PT Perorangan; 3102 Prive hanya OP/CV; 5101 HPP untuk perpetual; 5102/5103/5105 untuk periodik; Gedung & Kendaraan opsional. Metode persediaan ditanyakan di onboarding (default periodik; profil lama terbaca periodik).
- **Akun otomatis**: `coa.AutoAccount` (jenis → akun) + `coa.CounterAccount` (lunas: Kas 1101 / Bank 1102 untuk BCA Bisnis & B2B / Saldo Marketplace 1104 untuk GoFood, GrabFood, ShopeeFood; belum lunas: Piutang 1201 / Hutang 2101). Koreksi per transaksi lewat field `akun` (divalidasi terhadap CoA aktif).
- **Laporan**: `coa.Journal` (double entry) → `TrialBalance` → `IncomeStatement` (format perdagangan) & `BalanceSheet`. `GET /api/v1/reports/coa`, ekspor `GET /api/v1/reports/coa.xlsx` (Laba Rugi, Neraca, Neraca Saldo). Saldo awal = kas awal periode + persediaan Rp8,25 jt, modal = jumlahnya (sama dengan neraca ringkas). Laba bersih & total aset terbukti sama dengan laporan lama (test).
- **Batasan**: persediaan akhir periodik = persediaan awal sampai stok opname dicatat; HPP perpetual menunggu kartu stok. Jasa & manufaktur menunggu file CoA dari tim (laporan tetap format ringkas).
- UI: Pengaturan > Daftar Akun (CoA), kolom & pilihan Akun di Transaksi, Laporan Keuangan per akun + Neraca Saldo + Unduh Excel.

## Fase C+ — Onboarding Bahasa Awam (Okt 2026)
> Catatan untuk Astra: bagian ini menjelaskan wizard onboarding baru, dari pertanyaan sampai pemetaannya ke profil usaha.

- **Tujuan**: Owner awam bisa mengisi profil usaha tanpa perlu paham istilah akuntansi atau pajak. Wizard menampilkan **satu pertanyaan per layar**. Setiap jawaban dipetakan diam-diam ke `BusinessProfile`. Di akhir ada layar **"Ini yang kami siapkan untuk …"** yang merangkum daftar akun, cara catat stok, skema pajak, dan PPN, lalu tombol "Simpan & mulai".
- **Kapan muncul**: saat Owner pertama kali login dan `business.json` belum ada (`App.tsx`, `if (!business.profile)`). Admin dan Kasir tetap melihat pesan "hubungi Owner".

| # | Pertanyaan | Jawaban → field | Muncul bila |
|---|---|---|---|
| 1 | Apa nama usaha Anda? | teks → `nama_usaha` | selalu |
| 2 | Usaha Anda paling mirip yang mana? | jual barang beli dari orang lain / jual jasa / membuat barang sendiri → `jenis_usaha` = `perdagangan` / `jasa` / `manufaktur` | selalu |
| 3 | Bagaimana Anda tahu sisa stok barang? | hitung sesekali / catat setiap keluar-masuk → `metode_persediaan` = `periodik` / `perpetual` | perdagangan & manufaktur |
| 4 | Usaha Anda terdaftar atas nama siapa? | pribadi / PT Perorangan / CV / PT / **tidak tahu (= `op`)** → `bentuk_usaha` | selalu |
| 5a | Apakah Anda sudah menikah? | belum → `TK`, sudah → `K` | pribadi |
| 5b | Berapa orang yang Anda tanggung? | 0–3 (maksimal 3) → angka PTKP | pribadi |
| 5c | Penghasilan pasangan dilaporkan dalam satu NPWP? | ya → `K/I` | pribadi & sudah menikah |
| 6 | Menarik PPN dan membuat faktur pajak? | ya → `pkp: true` | selalu |

- **Hasil PTKP**: `ptkp` = `TK/n`, `K/n`, atau `K/I/n`. Field ini dikosongkan untuk badan usaha. Untuk jasa, `metode_persediaan` dikosongkan.
- **Default bila ragu**: periodik, penghasilan pasangan tidak digabung, dan belum PKP.
- **Lokasi kode**:
  - `frontend/src/lib/onboarding.ts` berisi teks pertanyaan, alasan, dan opsi (`QUESTIONS`), urutan langkah dan langkah yang dilewati (`steps`), pemetaan ke profil (`toProfile`), serta teks layar akhir (`summary`). Semuanya fungsi murni.
  - Tesnya ada di `frontend/src/lib/onboarding.test.ts`.
  - UI-nya `OnboardingWizard` di `frontend/src/views/ProfilUsaha.tsx`. Komponen ini memakai ulang `ChoiceGrid`. Sejak perbaikan UX 19 Oktober 2026, memilih jawaban tetap di layar yang sama sampai menekan **Lanjut**; tersedia tombol Kembali serta "Ubah jawaban".
  - **Backend tidak berubah.** `PUT /api/v1/business` dan `finance.BusinessProfile.Validate()` tetap menjadi penjaga terakhir.
- **Menambah atau mengubah pertanyaan**:
  1. Tambah nilai di `Field`.
  2. Isi entri di `QUESTIONS`.
  3. Tambah kondisinya di `steps`.
  4. Petakan jawabannya di `toProfile`.
  5. Tulis tesnya terlebih dulu.
  6. Bila butuh field profil baru, tambahkan juga di `internal/finance/profile.go` (struct + `Validate`) dan di tipe `BusinessProfile` pada `frontend/src/lib/api.ts`.
- **Batasan & catatan**:
  - Belum ada sign-up publik. Akun masih dibuat lewat seed atau oleh Owner.
  - Teks pajak di `summary` (PP 0,5%, batas Rp500 juta, Pasal 31E Rp4,8 miliar) ditulis manual. Selaraskan bila `internal/tax/planning.go` berubah.
  - `pkp` tersimpan tetapi belum dibaca oleh backend.
  - Halaman Pengaturan > Profil Usaha masih memakai form teknis (`ProfilForm`) untuk perubahan berikutnya.

## UX Onboarding — 19 Oktober 2026

- Desain disetujui pengguna: satu pertanyaan per layar, navigasi eksplisit lewat **Lanjut**, jawaban tetap tersimpan saat Kembali/Ubah jawaban, dan ringkasan sebelum menyimpan.
- `ChoiceGrid` sekarang memakai radio native berlabel. Tombol panah/Spasi dapat memilih; kelompok radio pada Profil Usaha tetap independen. Keadaan terpilih, fokus, dan petunjuk pilihan jelas.
- Wizard menampilkan progres berdasarkan pertanyaan yang berlaku saat itu. Jumlah dapat berubah mengikuti jenis/bentuk usaha dan status menikah; keterangan ini ditampilkan di bawah progres. Fokus berpindah ke judul pertanyaan saat langkah berubah.
- Tata letak desktop/mobile memakai token yang ada, pilihan satu kolom di wizard, tombol minimal 44 px, dan ringkasan berupa daftar. Tombol navigasi wizard dinonaktifkan saat menyimpan; error simpan dapat dicoba ulang.
- File implementasi: `frontend/src/views/ProfilUsaha.tsx`, `frontend/src/index.css`. Kontrak API, pertanyaan, dan pemetaan profil tidak berubah; tidak ada dependensi baru.
- Spesifikasi: `docs/superpowers/specs/2026-10-19-onboarding-ux-design.md`; rencana: `docs/superpowers/plans/2026-10-19-onboarding-ux.md`.
- Verifikasi: build frontend berhasil; 29 tes Vitest lulus; lint tanpa error (10 warning di file lain). Browser Chromium lokal pada 1280×900, 375×812, dan 320×740: keyboard, fokus, jasa melewati stok, manufaktur/perdagangan, OP menikah & penghasilan gabung, kembali/edit jawaban, perpindahan ke PT tanpa PTKP, kegagalan simpan/retry, status sibuk, payload profil, dan masuk dashboard lulus. Radio pada halaman Profil Usaha juga diperiksa. Review kode terpisah tidak menemukan bug.
- Bukti sesi lokal: `/tmp/pajak-onboarding-check.py`, `/tmp/pajak-onboarding-name-desktop.png`, `/tmp/pajak-onboarding-summary-mobile.png`, `/tmp/pajak-onboarding-options-mobile.png`. Tidak memeriksa pembaca layar fisik atau browser selain Chromium. Belum deploy produksi.

## Revisi brand — 19 Oktober 2026

- Pengguna menetapkan navy **`#222852`** sebagai warna utama agar mengingatkan pada perpajakan/DJP. Kedua kode yang dikirim sama; aksen emas yang sudah ada dipertahankan.
- Token `--brand-navy` menjadi sumber warna login, onboarding, sidebar, dan teks CTA emas. Favicon dan meta tema browser memakai nilai yang sama. Warna teks sidebar disesuaikan agar terbaca di atas navy; permukaan laporan tetap netral.
- Pedoman diperbarui di `DESIGN.md`, salinannya `design.md`, `docs/DESIGN_BRIEF.md`, `docs/PRD.md`, dan `agent.md`.
- Build frontend berhasil. Chromium: login, onboarding mobile, dan dashboard diperiksa; warna terhitung sidebar/login `rgb(34, 40, 82)`. Kontras emas/navy 8,58:1; teks sekunder sidebar/navy 8,16:1; label sidebar/navy 6,60:1. Screenshot di `/tmp/pajak-brand-*.png`. Commit `c82fad5` terkirim setelah PR #1 digabung, sehingga dibawa ke branch lanjutan `feat/login-password-visibility`; belum deploy.

## Tambahan UX login — 19 Oktober 2026

- Kolom password login memiliki tombol mata untuk tampil/sembunyi, dengan label aksesibel dan area sentuh 44 px. Password tersembunyi saat awal dan ketika form dikirim; tombol mata tidak melakukan submit.
- Build frontend dan pemeriksaan browser klik/keyboard lulus; nilai input tetap sama saat toggle. Screenshot: `/tmp/pajak-password-toggle-mobile.png`.

## Fase C1 — Laporan Keuangan Satu Menu & Register Aset (Okt 2026)
**Perubahan navigasi**: menu `Arus Kas` dihapus; seluruh laporan keuangan jadi **satu menu dengan 4 tab** —
Laba Rugi, Neraca, Arus Kas (2 metode), dan Aset & Penyusutan. Permission `arus-kas` dihapus dari
`auth.financeViews`; semua tab ikut hak akses `laporan-keuangan`, jadi Kasir tetap tidak bisa melihatnya.
Ini menggantikan pembagian menu di agent.md §2 nomor 2.

**Arus kas dua metode** (`finance.CashFlow`, [ledger.go](file:///home/reyfuu/pajak/internal/finance/ledger.go)):
- Dipecah per aktivitas PSAK 2: `operasi_langsung`, `operasi_tidak_langsung`, `investasi`, `pendanaan`.
  Masing-masing `CashFlowSection{lines, net}`; baris penerimaan pelanggan & beban operasional punya `detail`
  (rincian per channel / per jenis biaya). Baris `non_kas: true` adalah baris informasi yang sengaja Rp 0.
- **Invarian yang dites**: `operasi_langsung.net == operasi_tidak_langsung.net`, dan
  `operasi + investasi + pendanaan == net_cash`.
- `NewReceivables`/`NewPayables` sekarang **hanya akrual operasional**, supaya penyesuaian modal kerja di
  metode tidak langsung tepat sama dengan kas operasi. Akrual investasi/pendanaan (mis. aset dibeli kredit)
  pindah ke `NewPayablesOther`/`NewReceivablesOther` — tetap jadi utang/piutang di neraca.
  `CashFlow.NonOperating` dihapus, diganti `Investasi.Net` + `Pendanaan.Net`.

**Penyusutan fiskal PMK 72/2023** ([asset.go](file:///home/reyfuu/pajak/internal/finance/asset.go)):
- Keputusan: **satu angka penyusutan, basis fiskal** — tanpa nilai residu, kelompok & tarif wajib ikut PMK.
  Angka komersial PSAK (masa manfaat sendiri + residu) dan koreksi fiskalnya belum disimpan (ditandai `ponytail:`).
- `KelompokAsetList` = tabel Pasal 2 ayat (3): Kelompok 1–4 (4/8/16/20 th; GL 25/12,5/6,25/5%; SM 50/25/12,5/10%),
  Bangunan Permanen 20 th 5%, Bangunan Tidak Permanen 10 th 10%. Bangunan **hanya garis lurus**.
  Amortisasi harta tak berwujud memakai kelompok & tarif yang sama (Pasal 9 ayat 3).
- `JenisHartaList` = subset Lampiran PMK untuk UMKM dagang/jasa/F&B; user pilih jenis harta → kelompok terisi
  otomatis, ada opsi "Lainnya" untuk pilih kelompok manual. **Belum diverifikasi baris per baris ke Lampiran
  resmi** (ditandai `ponytail:`) — wajib dicek tim pajak sebelum dipakai untuk SPT.
- `AssetSchedule` menghasilkan jadwal beban **per bulan** sepanjang masa manfaat: penyusutan dimulai bulan
  pengeluaran (Pasal 5 ayat 1), dan pada saldo menurun **akhir masa manfaat sisa nilai buku disusutkan
  sekaligus** (Pasal 2 ayat 2) sehingga nilai buku berakhir nol. Sisa pembulatan jatuh di bulan terakhir,
  jadi total jadwal selalu sama dengan harga perolehan.
- Endpoint: `GET /api/v1/assets?period=`, `POST /api/v1/assets`, `PUT|DELETE /api/v1/assets/{id}`.
  Register ikut tersimpan di `state.json` (`assets`, `next_asset_id`); `state.json` lama tanpa kunci itu
  tetap dibaca dan memakai seed aset.

**Draft aset dari transaksi**: transaksi jenis `Beli Alat / Aset` yang belum terdaftar muncul otomatis sebagai
draft (diturunkan saat hitung, tidak ditulis ke state). Draft masuk neraca sebesar harga perolehan supaya
neraca tetap seimbang, tapi belum menghasilkan beban sampai kelompok & metodenya dilengkapi.

**Satu sumber angka**: `finance.BuildPeriodReport(txs, assets, period, openingCash, inventory)` menghitung
laba rugi + arus kas + neraca + register aset sekaligus. Semua endpoint laporan, KPI dashboard, Perencanaan
Pajak, dan AI chat (`Engine.Answer` sekarang menerima `PeriodReport`) memakai fungsi ini, jadi tidak mungkin
lagi saling bertentangan.

**Dampak angka seed (Agustus 2026)** — penyusutan Rp6,55 jt masuk laba rugi:
laba bersih **Rp25,0 jt → Rp18,45 jt**, margin **27,17% → 20,05%**, total aset **Rp116,25 jt → Rp316,775 jt**,
kenaikan kas **tetap Rp17,0 jt** (penyusutan non-kas). Modal awal neraca sekarang
`kas awal + persediaan + nilai buku aset awal periode` (Rp295,325 jt) — rumus lama `kas awal + persediaan`
membuat neraca njomplang sebesar nilai buku aset.

**Belum dikerjakan (sengaja)**: baris Pajak Penghasilan di laba rugi; perubahan persediaan per periode
(nilai stok masih tetap Rp8,25 jt); `new_receivables` masih piutang yang terbit periode ini, bukan selisih
saldo awal–akhir (benar untuk satu bulan, salah untuk laporan antar-periode); penjualan/pelepasan aset;
`SparkLaba` & `monthly_trend_11m` di seed masih angka lama sehingga grafik tren belum mencerminkan penyusutan.

## Fase C2 — Impor Data Awal (Onboarding), Karyawan & Konsistensi Grafik (Okt 2026)

**Keputusan desain penting**: laporan neraca & arus kas **tidak bisa diimpor sebagai angka jadi** —
keduanya diturunkan dari transaksi + register aset, jadi angka jadi akan langsung bertabrakan dengan
hasil hitung begitu ada satu transaksi baru. Yang diimpor adalah **posisi awalnya** (neraca awal /
`OpeningBalance`), dan dari situ neraca & arus kas tiap periode terbentuk sendiri.

**Neraca awal** ([opening.go](file:///home/reyfuu/pajak/internal/finance/opening.go)):
- `OpeningBalance{Tanggal, Kas, Piutang, Persediaan, Utang, Pinjaman, ModalDisetor}`.
- Modal awal **dihitung** dari komposisinya (aktiva awal − kewajiban awal), bukan dari isian user;
  `ModalDisetor` hanya dicocokkan dan selisihnya dilaporkan, supaya salah input ketahuan tapi neraca
  tetap seimbang.
- `At(txs, period)` menurunkan `ReportOpening` per periode: **saldo kas berjalan** (saldo awal + kas
  bersih semua transaksi sebelum periode itu). Ini carry-over antar periode yang sebelumnya tidak ada.
- `BuildPeriodReport` & `BuildBalanceSheet` sekarang menerima `ReportOpening`, bukan `openingCash, inventory`.
  Konstanta `inventoryValue` di main.go dihapus, diganti `finance.SeedInventory`.
- ponytail: piutang/utang/pinjaman awal hanya dipakai di **periode onboarding**; pelunasannya lintas
  periode belum ditelusuri (butuh umur piutang per periode). Neraca tetap seimbang karena modal awal
  dihitung dari komposisi yang sama.
- `SeedOpening` (data demo) **sengaja tidak berantai**: seed menulis saldo awal tiap periode
  sendiri-sendiri untuk memperagakan kas turun walau laba positif. Kalau dirantai, kas Agustus jadi
  Rp172,5 jt (bukan Rp97 jt) dan narasi demo rusak. Tenant nyata memakai `OpeningBalance.At` yang benar.

**Impor data awal** ([onboarding.go](file:///home/reyfuu/pajak/internal/finance/onboarding.go)):
- Satu workbook, 5 sheet: `Saldo Awal`, `Aset`, `Karyawan`, `Transaksi`, `Panduan`.
  `GET /api/v1/onboarding/template` menghasilkannya; isinya lolos parsernya sendiri (ada test).
- `POST /api/v1/onboarding/import` — tanpa `?commit=1` hanya **pratinjau**, tidak menyimpan apa pun.
  Nama sheet & nama kolom bebas selama memuat kata kuncinya; tanggal `DD/MM/YYYY` dan angka
  `Rp 45.000.000` diterima (memakai `parseTanggal`/`parseRupiah` yang sudah ada).
- Baris bermasalah dilaporkan dengan **nama sheet + nomor baris Excel** (`SheetError`) dan dilewati;
  baris lain tetap masuk.
- Commit: aset & karyawan **mengganti** register lama (ini onboarding, bukan tambah data — jumlah yang
  diganti dilaporkan); transaksi **ditambahkan** karena bisa diimpor bertahap per periode.
- `xlsx.ReadAll` baru (baca semua sheet + namanya); `xlsx.Read` lama tetap ada sebagai sheet pertama.

**Karyawan** ([karyawan.go](file:///home/reyfuu/pajak/internal/finance/karyawan.go)): master data
(nama, jabatan, tanggal masuk, gaji pokok, tunjangan, PTKP, NPWP, aktif) + rekap beban gaji bulanan.
CRUD `/api/v1/karyawan`, menu Pengaturan → Karyawan (permission `karyawan`, di `financeViews`).
ponytail: **belum** menghitung PPh 21 dan belum membuat transaksi gaji otomatis; PTKP sudah disimpan
supaya kalkulator PPh 21 tinggal membacanya.

**Permission baru** `impor-data` (Owner & Admin): menu Impor Data Awal + izin mengubah neraca awal.
Kasir tertutup. `GET /api/v1/opening` ikut `laporan-keuangan`, `PUT` ikut `impor-data`.

**Grafik tren jadi konsisten dengan KPI** ([trend.go](file:///home/reyfuu/pajak/internal/finance/trend.go)):
- Dulu `monthly_trend_11m`, `SparkOmzet/SparkLaba/SparkCash`, dan badge seperti "▲ Stabil" semuanya
  angka seed, sementara KPI dihitung ulang dari transaksi — jadi penyusutan hanya muncul di KPI.
- Sekarang `TrendHistory` menghitung laporan tiap periode sampai periode yang dilihat, lalu `ApplyTrend`
  menimpa grafik, keempat sparkline, dan badge (delta % vs bulan lalu; margin dalam poin persen).
  Dijamin test: titik terakhir grafik == KPI, dan `omzet − total biaya == laba bersih`.
- Grafik demo kini 8 titik (Jan–Agu 2026, sesuai periode yang punya transaksi), bukan 11 titik seed
  yang 3 di antaranya tidak punya data.
- ponytail: menghitung ulang semua periode setiap request dashboard; pasang cache bila periode sudah
  bertahun-tahun.

**Pemetaan jenis harta disiapkan untuk dikoreksi**: `JenisHarta` dapat kolom `Lampiran` (rujukan tempat
memeriksa) dan `Terverifikasi`. Semua baris bukan-bangunan masih `false`; UI menampilkan peringatan
"N jenis harta belum diverifikasi", dropdown menandai per baris, dan sheet Panduan di template punya
kolom Status Pemetaan. Cara mengoreksi ditulis sebagai komentar langkah-demi-langkah di atas
`JenisHartaList` — satu tabel, satu tempat; jangan ubah `ID` karena dipakai aset yang sudah tersimpan.

## Fase C3 — Menu Laporan Keuangan Dipegang 4 Tab, CoA Disinkronkan (Okt 2026)

**Masalah yang diperbaiki.** Saat Fase C1 di-rebase ke `main`, menu `laporan-keuangan` dipegang
`LaporanKeuanganCoA` (Fase C) yang hanya menampilkan `fallback` bila CoA tidak didukung. Karena 4 tab
Fase C1 dipasang sebagai `fallback` itu, untuk usaha **perdagangan** tab **Arus Kas** dan
**Aset & Penyusutan** tidak pernah terjangkau sama sekali.

**Keputusan.** Menu `laporan-keuangan` dipegang penuh oleh `LaporanKeuangan` (4 tab), untuk semua jenis
usaha. `LaporanCoA.tsx` **sengaja tidak terpasang di menu mana pun** — bukan kode mati yang terlupa;
alasannya ditulis di komentar atas komponennya. Endpoint `GET /api/v1/reports/coa` dan `coa.xlsx` tetap
hidup, tinggal dipasang begitu tempatnya di navigasi diputuskan.

**Register aset ikut dijurnal ke CoA** supaya laporan per akun tidak bertentangan dengan 4 tab:
- `coa.DepreciationJournal(assets, period)` — debit akun beban, kredit akun akumulasi, per aset. Aset
  tanpa transaksi kas (mis. hasil impor) dibukukan debit akun harta / kredit 3101, sama seperti neraca ringkas.
- `coa.OpeningAssets(assets, period)` — posisi aset awal periode per akun: harga perolehan di akun harta,
  penyusutan periode sebelumnya di akun akumulasi (dipisah, bukan diringkas jadi nilai buku).
- `coa.Opening(cash, inventory, assetAwal)` — tanda tangan berubah, modal awal kini termasuk nilai buku aset.
- Pemetaan akun di `akunAset`: kendaraan/sepeda motor → 1505/1506, bangunan → 1503/1504,
  harta tak berwujud → 1507/1508/6118, selain itu → 1501/1502/6113.

**Tiga akun di luar `Daftar COA.xlsx`** (`coa.TambahanDiLuarFile`): `1507 Harta Tak Berwujud`,
`1508 Akumulasi Amortisasi`, `6118 Beban Amortisasi`. Semuanya `JikaAda`, jadi tidak muncul bagi usaha
yang tidak punya; tidak ada baris CoA lama yang diubah. Kode **6118**, bukan 6115 — 6115 sudah dipakai
Beban Pelatihan & Pengembangan. `TestMasterFromFile` tetap menjaga jumlah 72 akun dari file dengan
mengurangi tambahan ini.

**Terbukti konsisten** (Agustus 2026, profil perdagangan/CV/periodik): laba bersih CoA **18.450.000** ==
laba rugi 4 tab; total aset CoA **316.775.000** == neraca 4 tab; neraca saldo seimbang;
arus kas langsung == tidak langsung == **17.000.000**. Profil **jasa** → CoA `supported: false`,
4 tab tetap jalan.

**Header disatukan**: judul "Laporan Keuangan" kini tetap di keempat tab (hanya subtitle berganti);
`PageHeader` ganda di `Aset.tsx` dihapus dan tombol **+ Tambah Aset** pindah ke kartu Register Aset.
`Aset` tidak lagi menerima prop `periodLabel`.

## Fase C4 — Invoice & Cetak PDF (Okt 2026)

**Menu Invoice sebelumnya hanya menumpang ke `PiutangUtang`** — tidak ada entitas faktur sama
sekali, yang ada cuma `ObligationItem` dari seed (nomor, lawan transaksi, satu angka total).
Sekarang ada modul faktur penjualan penuh dengan rincian baris dan cetak PDF.

**`internal/pdf` — penulis PDF tanpa dependensi**, sejalan dengan `internal/xlsx` dan menjaga
`go.mod` tetap nol dependensi. PDF 1.4, font bawaan Helvetica (WinAnsi, jadi tidak ada font yang
perlu disertakan), API-nya `Text`/`Rect`/`HLine` dengan sumbu y dari atas halaman. `metrics.go`
memuat tabel lebar karakter AFM supaya rata kanan dan `Truncate`/`WrapText` akurat.
Yang paling mudah salah saat menulis PDF sendiri — offset tabel xref dan `/Length` tiap stream —
dikunci test tersendiri.

**`finance.Invoice`**: nomor, `NoPO` (nomor PO pelanggan), status (draft/terkirim/lunas), tanggal
terbit & jatuh tempo, identitas pelanggan, baris rincian, diskon, `PPNBps`, catatan.
Total **tidak pernah disimpan** — selalu dari `Totals()`, sehingga angka di layar, di PDF, dan di
API tidak mungkin berbeda. Yang disimpan hanya `PPNBps` saat terbit, supaya faktur lama tidak ikut
berubah ketika tarif di profil diganti. **PPN dihitung dari DPP (setelah diskon), bukan subtotal.**

**`BusinessProfile` dapat identitas penerbit**: alamat, NPWP, telepon, email, bank (nama/rekening/
atas nama), penandatangan, dan `PPNBps`. Semuanya opsional supaya wizard onboarding tidak memanjang;
`profilFakturLengkap` melaporkan yang masih kosong dan UI mengingatkan sebelum faktur dikirim.
`BusinessProfile.TarifPPNBps()` menjaga satu aturan di satu tempat: **non-PKP selalu 0** — mereka
memang tidak boleh memungut PPN.

**`finance.Terbilang`** mengeja Rupiah jadi kata untuk kaki faktur, mengikuti kaidah "seribu",
"sebelas", "sepuluh", "seratus".

**Penomoran** `NextInvoiceNumber` meneruskan deret yang ada (INV-2026-089 → INV-2026-090) dan
dimulai ulang tiap tahun terbit. Nomor kembar ditolak 409, bukan diam-diam dibuat ganda.

**Endpoint**: `GET/POST /api/v1/invoice`, `PUT/DELETE /api/v1/invoice/{id}`, dan
`GET /api/v1/invoice/{id}/pdf`. Catatan: pola `{id}.pdf` **tidak sah** di `http.ServeMux`
(wildcard harus mengisi satu segmen penuh), jadi PDF memakai segmen terpisah; nama berkas
unduhan tetap `INV-2026-090.pdf` lewat `Content-Disposition`.

**Faktur seed** (`SeedInvoices`) sengaja dibuat cocok dengan `Obligations` di seed dashboard —
nomor dan totalnya sama, dikunci test, supaya Piutang & Utang dan Invoice tidak bercerita berbeda
tentang tagihan yang sama. Totalnya Rp 11.000.000, sama dengan Piutang Usaha di neraca.

**ponytail**: belum ada pelunasan parsial, nota kredit, mata uang selain Rupiah, faktur pajak
e-Faktur, maupun pengiriman faktur lewat email. Nomor halaman hanya tercetak di halaman terakhir
karena `pdf.Doc` belum bisa menggambar ulang ke halaman sebelumnya — cukup untuk faktur yang
hampir selalu satu halaman.

## Fase C5 — Laporan AI & Kalkulator Pajak (Okt 2026)

### Laporan AI — tab ke-5 di Laporan Keuangan

`internal/llm` adalah klien **chat completions kompatibel OpenAI**, stdlib saja. Antarmuka itu
dipilih karena sudah jadi standar de facto: satu kode jalan untuk OpenAI, Groq, OpenRouter,
DeepSeek, Together, xAI, Anthropic (endpoint kompatibelnya), sampai model lokal lewat Ollama
atau LM Studio. Pindah penyedia cukup ganti env, tanpa ubah kode.

```
LLM_API_KEY   kosong = fitur AI mati, aplikasi tetap jalan
LLM_BASE_URL  default https://api.openai.com/v1
LLM_MODEL     default gpt-4o-mini
LLM_TIMEOUT   detik, default 45
```

**Model tidak pernah berhitung.** `finance.CollectFacts` merangkum SELURUH laporan satu periode
(laba rugi, arus kas, neraca, aset, pajak, tagihan, operasional, tren) jadi `BusinessFacts`.
Struktur yang sama dipakai dua pihak: halaman menampilkannya sebagai rincian, model menerimanya
sebagai bahan. Jadi mustahil model menilai angka yang berbeda dari yang dilihat pengguna.
Prompt melarang keras berhitung — di aplikasi pembukuan, angka karangan lebih berbahaya daripada
tidak ada analisis.

**Tanpa kunci, halaman tetap berguna**: rincian lengkap dan daftar `Peringatan` (fakta objektif
seperti neraca tidak seimbang, faktur lewat tempo, pemetaan PMK belum diverifikasi) tetap tampil;
hanya blok rekomendasi yang kosong, dengan petunjuk apa yang perlu diisi. Dikunci test.

Endpoint `GET /api/v1/reports/ai`. `ai.Analyze` tahan terhadap balasan model yang dibungkus
pagar kode (`llm.ExtractJSON`) dan membatasi panjang daftar supaya tata letak tidak rusak kalau
model mengabaikan prompt.

### Kalkulator Pajak — menu baru

Delapan jenis: PPh 21, PPh 22, PPh 23, PPh 4 ayat (2), PPh 15, PPh Badan, PPN, PPnBM.
Permission baru `kalkulator-pajak`. Setiap hasil menampilkan **langkah perhitungannya**, supaya
angkanya bisa ditelusuri dan tidak tampak muncul begitu saja.

- `tax.JenisPajakList` — satu tabel objek pajak, tiap baris memuat tarif (bps), DPP, status final,
  dan **rujukan aturan** supaya bisa dicek ulang. Satu-satunya tempat mengoreksi tarif.
- `tax.HitungPPh21` — metode tahunan Pasal 17: biaya jabatan 5% (maks Rp 6 juta/tahun), iuran
  pensiun, PTKP, tarif progresif. Gross dan Gross Up; tanpa NPWP +20%. Gross up diselesaikan
  dengan **iterasi sampai konvergen**, bukan rumus per lapisan — lebih tahan saat tarif berubah.
- `tax.HitungPPhBadan` — tarif 22% memakai `Pasal31E` yang sudah ada.
- PPN tarif umum dimodelkan sebagai **12% atas DPP nilai lain 11/12**, sehingga bebannya setara
  11% dari harga jual (PMK 131/2024).

**TER (PMK 168/2023) sengaja belum lengkap.** Lampiran tarifnya berupa gambar di dokumen resmi
dan sumber sekunder hanya memuat lapisan terpisah-pisah; menebak 125 baris tarif pajak tidak
dapat dipertanggungjawabkan. `tax.TERLapisan` memuat baris yang sudah terverifikasi saja, dan
`HitungTER` **menolak menghitung** di luar itu (`ErrTERBelumLengkap`) alih-alih mengembalikan
angka yang keliru. Cara melengkapi ditulis langkah-demi-langkah di atas `TERLapisan` — satu
tabel, satu tempat. UI menampilkan TER berdampingan sebagai pembanding bila sudah terisi.

**ponytail**: tarif disusun dari ketentuan yang berlaku umum dan harus diverifikasi tim pajak
sebelum dipakai untuk SPT. PPnBM bergantung klasifikasi barang yang tidak bisa diturunkan dari
nominal, jadi lapisannya dipilih manual.

### Piutang & Utang dan Stok & Gudang jadi data sungguhan

Dua halaman ini sebelumnya hanya menampilkan contoh dari `DashboardSnapshot` — tidak tersimpan,
tidak bisa disunting, dan isinya berbeda dengan dashboard. Sekarang keduanya record yang
tersimpan per usaha (`state.json`), lengkap dengan CRUD dan impor Excel.

- `finance.Tagihan` & `finance.StokItem` — data yang disimpan. Yang **tidak** disimpan: hari
  keterlambatan dan status stok. Keduanya dihitung saat dibaca (`HariTerlambat`, `Status`),
  karena nilai tersimpan basi begitu tanggal berganti atau stok diubah.
- `TagihanRow` & `StokRow` membawa turunan itu dari server, jadi ambang "stok menipis" dan rumus
  margin hanya ada di satu tempat, bukan dihitung ulang di React.
- Dashboard kini **selalu** merender daftarnya dari record tenant; usaha demo dimulai dengan
  `SeedTagihan()`/`SeedStok()`. Cabang "pakai seed tampilan" dihapus: dua tempat yang menampilkan
  tagihan yang sama tidak boleh berbeda angkanya, dan baris contoh harus bisa disunting.
- Impor per menu: `GET /api/v1/{tagihan,stok}/template` (satu sheet + Panduan) dan
  `POST /api/v1/{tagihan,stok}/import`. Tanpa `?commit=1` hanya pratinjau. `mode=tambah|ganti`
  dipilih pengguna — berbeda dari onboarding yang selalu mengganti, karena daftar ini hidup.
- Berkas dengan sheet bernama "Sheet1" tetap terbaca bila itu satu-satunya sheet (hasil
  "Save As"/Google Sheets); dua sheet asing ditolak, tidak ditebak.

**Jebakan yang sudah kena sekali**: kata **"piutang" memuat "utang"**. Urutan pemeriksaan di
`jenisTagihanDari` terbalik membuat SELURUH piutang terbaca sebagai utang — tanpa satu baris pun
gagal impor, jadi tidak terlihat dari jumlah baris. Dikunci test; jangan tukar urutannya.

### Produksi & Supplier — HPP per produk dari resep

`finance.Produk` memuat resep (BOM) yang menunjuk `StokItem` lewat `StokID`. `HitungHPP` menjumlah
biaya bahan dari **harga pokok di Stok & Gudang**, bukan angka yang diketik di produk, supaya HPP
ikut naik sendiri saat harga bahan naik. Dihitung tiap kali daftar dibaca, tidak disimpan.

- `TakaranSkala = 1000`: takaran bahan disimpan per seribu satuan (180 = 0,18 kg). Resep nyata
  butuh pecahan sementara rupiah tetap bilangan bulat; UI menampilkannya sebagai desimal.
- Pembulatan HPP per unit **ke atas** (`bagiKeAtas`). HPP yang lebih murah dari kenyataan membuat
  harga jual ditetapkan kurang — pembulatan diarahkan ke sisi yang aman.
- Bahan yang stoknya sudah dihapus atau harga pokoknya masih 0 **tidak menggagalkan** perhitungan:
  biayanya 0 dan diberi peringatan, supaya jelas HPP-nya belum lengkap, bukan sekadar murah.
- `ProdukRow.Selisih` membandingkan HPP resep dengan harga pokok di stok (dicocokkan lewat SKU).
  Bukan 0 = angka di stok sudah basi. `POST /api/v1/produksi/produk/{id}/terapkan-hpp` menyalinnya,
  sebagai aksi terpisah — harga pokok ikut menentukan nilai persediaan di neraca, jadi
  perubahannya harus keputusan pengguna, bukan efek samping menyunting resep.
- Kolom utang per supplier dicocokkan dari **nama pihak** di Piutang & Utang. Pencocokan nama
  rapuh; UI menyebutnya perkiraan, bukan saldo resmi.

### Impor mutasi bank dengan pengelompokan otomatis

Dua langkah, dan langkah pertama tidak menyimpan apa pun:
`POST /api/v1/transactions/mutasi` (pratinjau) → diperiksa di layar →
`POST /api/v1/transactions/mutasi/commit` (baris hasil koreksi).

**Commit menerima baris dari klien, bukan membaca ulang berkasnya.** Kalau berkas dibaca ulang,
koreksi pengguna hilang dan tebakan mesin yang terbukukan — padahal seluruh guna langkah
pratinjau justru supaya tebakan itu bisa diperbaiki.

- `finance.TebakJenis` memetakan keterangan bank ke `JenisList` lewat `aturanMutasi`, satu tabel
  berurutan (yang pertama cocok dipakai). **Urutannya penting**: "pajak bunga" memuat "bunga",
  "biaya adm" memuat "adm". Arah uang ikut membatasi aturan — "pinjaman" masuk = Terima Pinjaman,
  keluar = Bayar Cicilan.
- Kata kunci ≤4 huruf dicocokkan sebagai **kata utuh**, supaya "ADMIRAL" tidak terbaca "adm".
- Tiap baris membawa `Keyakinan` (tinggi/sedang/rendah) dan `Alasan`. Tanpa kata kunci yang cocok,
  jenisnya diisi default per arah dengan keyakinan **rendah** — bukan dikosongkan: baris kosong
  mudah terlewat saat diperiksa, baris berlabel "rendah" tampil menonjol. UI menghitung dan
  memperingatkan berapa yang masih perlu diperiksa, karena uang masuk yang sebenarnya setoran
  modal akan terhitung omzet, dan omzet itu dasar PPh final.
- Dugaan duplikat (tanggal + nominal sama dengan transaksi yang ada) **tidak dicentang** otomatis.
- Jenis yang arahnya bertentangan dengan arah uang di rekening ditolak di `MutasiKeTransaksi`,
  tidak dibukukan terbalik.
- Format yang terbaca: kolom Debet/Kredit terpisah, satu kolom Mutasi bertanda DB/CR (termasuk
  yang menempel di angkanya, "1.250.000 DB"), tanda minus/kurung, tanggal tanpa tahun ("01/09",
  tahunnya dari isian pengguna), `.xlsx` maupun `.csv` (pemisah `,` `;` atau tab).
- Baris kop di atas header (nama bank, periode) dan baris kaki ("Saldo Akhir") dilewati tanpa
  dilaporkan sebagai kesalahan.

### Menu Lapor Bug

`internal/support`. Laporan **selalu disimpan di server lebih dulu**, baru diteruskan — penerusan
bisa gagal karena jaringan, dan laporan yang hilang karena itu hilang selamanya sementara pelapor
sudah merasa melapor.

```
BUG_WEBHOOK_URL      kosong = laporan hanya tersimpan di server (tidak ada data keluar)
BUG_WEBHOOK_TIMEOUT  default 10s
MASTER_EMAIL         pemilik ID Master: lihat laporan lintas usaha & tanggapi; kosong = tidak ada
APP_VERSION          ikut dicatat di tiap laporan; default "dev"
```

- Payload webhook memuat `text` (Slack), `content` (Discord), dan `laporan` utuh sekaligus.
- `Pengirim.Tujuan()` hanya menyebut **host**, tidak URL penuh: webhook Slack/Discord memuat token
  di dalam path-nya.
- Laporan terpisah per usaha; hanya `MASTER_EMAIL` melihat semuanya dan bisa mengubah status.
- UI menampilkan apa saja yang ikut terkirim (nama, email, peran, nomor usaha, versi, browser)
  **sebelum** dikirim, dan menegaskan tidak ada data transaksi atau laporan keuangan yang ikut.
- `User-Agent` dibaca dari header, bukan body, supaya tidak bisa dipalsukan klien.
- Kasir ikut mendapat menu ini — dia yang paling sering menemui masalah saat memakai aplikasi.

## Langkah Berikutnya (Pengembangan Lanjutan)


0. **Prioritas**: CoA jasa & manufaktur (menunggu file tim) → stok opname / kartu stok (persediaan akhir & HPP perpetual) → Fase D (role Client + upload terkompres) → Fase E (XML Coretax) → multi-tenant (database). Sign-up publik dan multi-tenant cukup memakai ulang `OnboardingWizard` setelah register.
1. Melengkapi fitur cetak / ekspor laporan ke format PDF dan Excel (.xlsx) langsung dari UI.
2. Melengkapi penagihan invoice & pelunasan parsial/lunas pada modul Piutang & Utang.
   Terkait: `StokItem.Jumlah` masih bilangan bulat, jadi stok 42,5 kg harus dicatat 42 kg atau
   satuannya diperkecil ke gram. Perlu tipe kuantitas berpecahan (seperti `TakaranSkala` di resep)
   sebelum stok opname; mengubah arti kolom Jumlah di template yang sudah dipakai orang tidak
   boleh dilakukan diam-diam.
   Mutasi bank: kata kunci `aturanMutasi` disusun dari pola cetakan BCA/Mandiri yang umum. Baris
   berkeyakinan rendah bisa dikurangi dengan mengisi `LLM_API_KEY` dan meminta model
   mengelompokkan sisanya — slotnya sudah ada, penyambungannya belum.
3. Migrasi persistensi PostgreSQL sesuai skema di [TRD.md](file:///home/reyfuu/pajak/docs/TRD.md) saat kebutuhan database relasional diperlukan.
4. Refresh token sesi & CSP.



