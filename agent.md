# AGENT.md — Laporan Bisnis (by Master Pajak)

> **Project**: Master Pajak (repo `laporan-bisnis`) — SaaS Pembukuan, Operasional & Pajak UMKM ("Gambaran seluruh bisnis dalam satu layar")  
> **Target Scale**: 1.000 Active UMKM Tenants (~4.000–10.000 transaksi/tenant/tahun), di-host di server lokal (on-premise)  
> **Core Stack**: **Go (Golang)** atau **ElysiaJS (Bun)** + PostgreSQL + Next.js/React (`shadcn/ui` + `Kokonut UI` + `21st.dev`)

---

## 1. Mandatory Session Protocol (Wajib Setiap Sesi)

Setiap sesi kerja **WAJIB** mengaktifkan dan mematuhi daftar *skills*, *mental models*, dan *toolchains* berikut tanpa terkecuali:

### A. Core Workflow & Meta-Skills
1. **`superpowers` (`using-superpowers`, `brainstorming`, `writing-plans`, `executing-plans`, `verification-before-completion`)**
   - Sebelum menulis kode, wajib cek skill yang relevan.
   - Jangan melompat ke implementasi sebelum desain/spesifikasi tervalidasi.
   - Klaim "selesai" wajib dibuktikan dengan output test/build nyata (`verification-before-completion`).
2. **`rtk` (Rust Token Killer)**
   - Gunakan `rtk` (jika tersedia di PATH, misal `rtk git status`, `rtk go test`, `rtk bun test`) atau flag output minimal untuk menghemat konteks token pada operasi CLI berulang.
3. **`graphify`**
   - Gunakan `graphify` (`graphify-out/`) sebagai sumber kebenaran peta arsitektur codebase, relasi antar-modul (Transaksi → Jurnal → Laporan Keuangan → Pajak UMKM), dan deteksi *god nodes* sebelum melakukan refactoring besar.
4. **`leantoken` ([morluto/leantoken](https://github.com/morluto/leantoken))**
   - **Lapisan pencarian & pembacaan kode pertama** untuk tugas luas (implementasi, debugging, review, arsitektur). Mulai dengan `leantoken.context`, bukan membaca file utuh, supaya konteks token tetap hemat.
   - Terpasang global sebagai MCP untuk Claude Code & Codex lewat runtime native terpin (`npx leantoken@0.1.28 setup --refresh --private-runtime --yes`; launcher `~/.local/share/leantoken/runtimes/0.1.28/leantoken`, skill di `~/.claude/skills/leantoken/`).
   - **Verifikasi wajib lewat launcher native itu sendiri**, bukan lewat `npx`:
     `~/.local/share/leantoken/runtimes/0.1.28/leantoken doctor --client claude` → `registration_health: current`.
     Dijalankan lewat `npx`, `doctor` melaporkan `stale` **walaupun registrasinya benar**, karena
     `matches_current` membandingkan entri terdaftar dengan proses doctor yang sedang berjalan (lewat npx
     itu rantai node, bukan launcher native). Jangan terkecoh lalu menjalankan `setup --refresh` berulang;
     patokan yang benar adalah `launcher_status: healthy` + `registration_health: current` dari launcher native.
   - CLI untuk pemakaian cepat tanpa MCP: `leantoken index`, `context --task "<tugas>"`, `search`, `outline`, `read`. Indeks repo ini: 234 dari 243 file (9 biner dilewati).
   - Pembagian tugas dengan `graphify`: `graphify` untuk peta arsitektur & relasi modul; `leantoken` untuk menemukan potongan kode relevan dalam batas token.
   - Catatan jcode: pemanggilan `mcp__leantoken__*` masih gagal (`unknown variant resource_link`), jadi di jcode pakai CLI-nya. Di Claude Code & Codex MCP-nya normal.
5. **`grillme` & `grillme with docs`**
   - Setiap ada ambiguitas bisnis/teknis/visual, lakukan interogasi desain singkat & tajam (*grill me*) yang mengacu langsung pada dokumen [BRD](file:///home/reyfuu/pajak/docs/BRD.md), [FRD](file:///home/reyfuu/pajak/docs/FRD.md), [TRD](file:///home/reyfuu/pajak/docs/TRD.md), [PRD](file:///home/reyfuu/pajak/docs/PRD.md), [DESIGN_BRIEF](file:///home/reyfuu/pajak/docs/DESIGN_BRIEF.md), dan [DESIGN.md](file:///home/reyfuu/pajak/DESIGN.md).
6. **`skillspector`**
   - Audit kesiapan skill di awal sesi; pastikan tidak ada regresi aturan dari skill yang terpasang.
7. **`handoff`**
   - Setiap akhir sesi atau sebelum pergantian konteks, perbarui status progres, keputusan arsitektur, dan *next steps* agar sesi berikutnya bisa langsung melanjutkan tanpa kehilangan konteks.
8. **`harness2` ([google-research/harness2](https://github.com/google-research/harness2) — Propose → Probe → Compose)**
   - **Wajib setiap akhir sesi**, bersama `handoff`: perbaiki *harness* agen proyek ini dari pelajaran sesi, tanpa menunggu aturan yang sama dilanggar dua kali. Komponen harness di proyek ini: `AGENTS.md`/`agent.md` (system prompt & guardrails), `.claude/skills/` (skills), memori proyek (memory), dan `deploy.sh`/`scripts/` (scripts).
     1. *Propose*: usulkan maksimal 3 perubahan harness (aturan yang dilanggar, langkah yang berulang, jebakan deploy/toolchain).
     2. *Probe*: uji tiap usulan pada tugas nyata sesi itu — apakah akan mencegah kesalahan tadi, dan apakah cek/tes yang diusulkan benar-benar lulus/gagal sesuai harapan.
     3. *Compose*: terapkan hanya yang lolos probe; catat yang ditolak beserta alasannya di ringkasan `handoff`.
   - Perubahan harness lewat PR seperti kode. Aturan keamanan & keuangan tidak diubah tanpa persetujuan user.
   - Framework terpasang di `~/tools/harness2` (commit `1cfe254`, venv `.venv`, demo lokal: `harness2-demo --output runs/demo --k 2`). Run berbasis model (Vertex AI, `.env`, `gcloud`) **hanya dengan persetujuan user**: ada biaya API, jaringan eksekutor tidak dibatasi, dan sandbox Codex dimatikan.

### B. Engineering Discipline & Code Quality
9. **`ponytail` (Lazy Senior Dev / YAGNI Mode — Active Every Response)**
   - Terapkan *The Ponytail Ladder*:
     1. *Does this need to exist at all?* (YAGNI — kapasitas target adalah **1.000 user**, jangan bangun infrastruktur sekelas 1 juta user seperti Kafka/Kubernetes/Microservices).
     2. *Already in this codebase?* Gunakan ulang helper/tipe yang ada.
     3. *Stdlib / Native platform first* (`net/http` / `database/sql` di Go, atau native `Bun.serve` / `Elysia` di Bun, constraint PostgreSQL di DB).
     4. Tandai kompromi kapasitas yang disengaja dengan komentar `// ponytail: <alasan & batas upgrade>`.
10. **`tdd` (Test-Driven Development)**
   - **Wajib Red-Green-Refactor** terutama pada domain kritikal finansial:
     - Kalkulasi Omzet, HPP (COGS), Laba Kotor, Laba Bersih, dan Margin Laba.
     - Kalkulasi Arus Kas vs Laba Akrual ("Kas bisa turun walau laba positif").
     - Kalkulasi Pajak UMKM (PPh Final 0,5% PP 55/2022, batas bebas pajak Rp500 juta/tahun untuk WP OP, potongan aplikasi/channel).
     - Penuaan Piutang & Utang (*aging schedule* > 30 hari).
   - Jangan pernah menggunakan `float64` / `number` floating-point biasa untuk uang; gunakan `int64` (Rupiah penuh / sen) atau `decimal`.
11. **`smokinggun` ([morluto/smokinggun](https://github.com/morluto/smokinggun))**
   - Pemindai kompleksitas statis untuk mencari *hotspot* sebelum optimasi: `npx --yes --package=smokinggun -- smokinggun scan . --format markdown` (read-only, offline, tidak menjalankan kode repo).
   - **Temuannya kandidat, bukan bukti.** Setiap temuan wajib divalidasi dengan benchmark Go nyata pada skala target `§3` (1.000 tenant, 4.000–10.000 transaksi/tenant/tahun) sebelum kode diubah. Tanpa pengukuran, perbaikan "performa" melanggar `ponytail`.
   - Hasil audit 20 Oktober 2026 (51 temuan, 45 di kode aplikasi, semuanya `medium/candidate`):
     - **Terbukti nyata & sudah diperbaiki**: `klienRows` di Panel Master memindai seluruh transaksi semua usaha pada tiap request. Diukur **di batas HTTP** (`BenchmarkMasterEndpoints`, lewat `srv.ServeHTTP`, 1.000 usaha × 4.000 transaksi/tahun), membandingkan revisi `5b8fa41` dengan sesudah perbaikan memakai file benchmark yang sama: `/master/pembayaran` 80 ms → 0,45 ms, `/master/tim` 70 ms → 0,65 ms, `/master/klien/{id}` 50 ms → 0,38 ms, `/master/ringkasan` 106 ms → 5 ms. `/master/klien` tetap ~53 ms karena halaman itu memang menampilkan kolomnya.
     - **Sisa biaya `/master/ringkasan` bukan dari loop itu**: balasannya ~388 KB untuk 1.000 usaha, karena `perlu_perhatian` dan `belum_ditugaskan` memuat baris klien utuh. Kalau mau dikecilkan, batasi jumlah baris atau kirim ringkasannya saja, bukan menyentuh `hitungTx`.
     - **Terbukti tidak relevan**: `membership-in-loop` di jalur CoA/jurnal (`coa.Find` atas 75 akun, `jenisOf` atas 19 jenis) — Transaksi → Jurnal → Neraca Saldo hanya 2 ms untuk 10.000 transaksi. Lihat `internal/coa/bench_test.go`. Jangan "optimasi" ini tanpa angka baru.
   - Dua tingkat benchmark, dipakai sesuai keperluan: `BenchmarkMasterEndpoints` (batas HTTP, angka yang dirasakan pengguna) dan `BenchmarkKlienRows` (fungsi internal, untuk menyetel satu fungsi). **Patokan klaim performa adalah yang pertama.**
   - Catatan untuk `klienRowsFilter(..., hanyaID)`: itu **murni soal performa**, karena korektness detail klien sudah dijaga `if k.ID == t.id` di handler-nya. Melumpuhkan filter itu tidak membuat satu pun test gagal (terbukti lewat uji mutasi), tetapi membuat `/master/klien/{id}` melambat dari 0,38 ms ke 48 ms. Jadi penjaganya benchmark, bukan test korektness — jangan tambahkan test korektness untuk itu.
   - Benchmark yang sudah ada wajib dijalankan ulang (`go test ./... -run '^$' -bench .`) sebelum mengubah jalur Transaksi → Jurnal → Laporan atau Panel Master.
   - **Tool morluto lain yang sudah diuji dan sengaja TIDAK dipasang** (dicatat supaya tidak dievaluasi ulang dari nol):
     - **`flameox`** ([morluto/flameox](https://github.com/morluto/flameox), dijalankan lewat `uvx flameox`; paket npm-nya hanya `setup`/`update`). 26 operasinya untuk trace Perfetto/Chrome, profil CPU Python/Node (`py-spy`, `pstats`, `cpuprofile`), PyTorch, dan kernel GPU. **Tidak ada pembaca Go pprof**, jadi untuk backend Go di proyek ini `go test -bench` + `pprof` tetap jalur yang tepat. Pasang hanya kalau nanti ada beban Python/Node/GPU.
     - Jalur SARIF-nya (`inspect_performance_candidates`) **tidak bisa dipakai bersama `smokinggun`**: 51 dari 51 hasil ditolak dengan `provider confidence is not a 0-to-1 number`, karena smokinggun menulis `confidence: "candidate"` (string ordinal). Dilaporkan ke upstream sebagai [flameox#529](https://github.com/morluto/flameox/issues/529). Jangan rancang alur yang mengandalkan rantai smokinggun → flameox sampai itu beres.
     - **`gitcontribute`** ([morluto/gitcontribute](https://github.com/morluto/gitcontribute)). Sudah diuji: `init` + `index` + `search code` berjalan (235 file terindeks). Gunanya riset sebelum menyumbang ke **repo pihak lain** (panduan kontribusi, pekerjaan terkait, PR serupa), bukan pekerjaan harian di repo sendiri. Untuk pencarian kode ia kalah dari `leantoken`: hanya nama file, tanpa nomor baris atau potongan kode. Butuh checkout bersih (`worktree is not clean` → gagal). Corpus ujinya sudah dihapus; tidak ada registrasi MCP yang ditinggalkan.
12. **`improve-codebase-architecture` & `setup-matt-pocock-skills`**
   - Struktur modular berbasis **Vertical Slices / Domain Modules** (`dashboard`, `ledger`, `cashflow`, `transactions`, `receivables`, `inventory`, `production`, `invoicing`, `pricing_sim`, `tax_umkm`, `master_data`).
   - *Make illegal states unrepresentable*: gunakan strict schema validation (TypeBox di Elysia / strict structs & sqlc di Go), branded types untuk ID & nominal Rupiah, dan eksplisit error handling.

### C. UI/UX, Visual Craft & Verification
13. **`antislop`, `taste`, & `impeccable`**
    - **DILARANG KERAS** membuat UI generik "AI slop" (gradien ungu-biru murahan, bayangan berlebihan, kartu tanpa hierarki, teks placeholder asal-asalan).
    - Ikuti **Brand Guideline Master Pajak v1.0** (`docs/brand/`, ringkasan teknis di `DESIGN.md` §2–3):
      - **Logo**: monogram M dua warna + wordmark `masterpajak` lewat komponen `Logo.tsx`; jangan menggambar ulang logo.
      - **Warna**: Biru DJP `#212C5F` (primer), Biru Royal `#26367A` (tautan, kata pajak), Kuning `#FED111` (aksen maks. 3%: satu titik/sorotan/CTA berbayar per layar), Biru Tegas `#131B40` (latar gelap). Semantik: sukses `#2E9B5B`, peringatan `#D9822B`, galat `#D64545`, info `#2F6FD6`. Di komponen selalu lewat token CSS, bukan hex.
      - **Tema**: terang default, gelap opsional (turunan palet navy).
      - **Typography**: Rubik untuk judul & wordmark, Public Sans untuk teks dan angka; angka finansial wajib `font-variant-numeric: tabular-nums`.
      - **Tone**: lugas, angka konkret, tanpa menakut-nakuti atau jargon AI (`DESIGN.md` §3.1).
12. **`ui ux pro max`, `21dev` (`21st.dev`), `shadcn/ui`, & `kokonut ui`**
    - Gunakan primitif `shadcn/ui` (Radix + Tailwind) yang diperkaya dengan komponen finansial dari `Kokonut UI` dan micro-interactions dari `21st.dev`.
    - Setiap KPI card di Dashboard wajib menampilkan:
      1. Label metrik yang jelas (`Omzet`, `Laba bersih`, `Margin laba`, `Saldo kas & bank`, `Target omzet`, `Piutang belum tertagih`).
      2. Nilai utama singkat & mudah dibaca (`Rp92 jt`, `Rp25 jt`, `27%`, `108%`).
      3. Badge perbandingan periode (`▲ 4% vs bulan lalu`, `+1,7 poin vs bulan lalu`) + mini sparkline / progress bar.
15. **`agent browser`**
    - Setiap perubahan frontend wajib diverifikasi secara visual menggunakan browser agent untuk memastikan tata letak 6 KPI card atas, 3 grafik tengah, 2 kartu bawah, dan sidebar navigasi presisi di desktop & mobile.

### D. Reverse Engineering (Situasional)
16. **`rea` ([morluto/rea](https://github.com/morluto/rea) — skill `reverse-engineer-anything` + MCP `rea`)**
    - Aktif setiap sesi, tetapi **dipakai hanya bila tugasnya memang membedah artefak jadi**: aplikasi web/Electron/JS pihak lain (mis. alur kompetitor atau halaman pembayaran Midtrans), file biner, APK, atau situs. Untuk memahami kode repo ini sendiri pakai `graphify`, bukan REA.
    - **Hanya untuk target yang sah dianalisis**: milik sendiri, berizin, atau untuk interoperabilitas/riset sesuai hukum. Tidak untuk membobol lisensi, DRM, atau mengambil data pihak lain.
    - Analisis berjalan lokal; analisis statis JS/.NET tidak menjalankan target. Penangkapan runtime menjalankan target dengan izin user, jadi **minta persetujuan user dulu**.
    - Terpasang untuk Claude Code lewat `npx rea-agents@6.7.0 setup --client claude_code --yes` (MCP di `~/.claude.json`, cadangan `~/.claude.json.rea.backup`; skill di `~/.claude/skills/reverse-engineer-anything`). Analisis biner native butuh Ghidra/Hopper/IDA, yang belum terpasang; Hopper hanya dipasang dengan persetujuan user. Perbarui dengan `npx rea-agents@latest setup --client claude_code`.

---

## 2. Domain Architecture Map (Laporan Bisnis)

Sesuai referensi desain produk **Master Pajak**, sistem terdiri dari 5 grup navigasi utama:

1. **RINGKASAN**
   - `Dashboard`: Ringkasan 6 KPI utama, grafik tren 11 bulan (`Omzet, biaya, dan laba`), grafik bar `Omzet harian` vs target harian, grafik area `Saldo kas per bulan`, `Omzet per produk` (Top 5), dan `Omzet per channel` (termasuk potongan aplikasi).
2. **LAPORAN**
   - `Laporan Keuangan` (satu menu, 4 tab): **Laba Rugi** (Income Statement) · **Neraca** (Balance Sheet) ·
     **Arus Kas** (operasi/investasi/pendanaan, disajikan dua metode — langsung & tidak langsung — plus
     rekonsiliasi selisih Laba vs Kas) · **Aset & Penyusutan** (register aset, penyusutan garis lurus &
     saldo menurun serta amortisasi menurut PMK 72/2023).
   - `Laporan Penjualan`: tren omzet per hari/minggu/bulan, per item, per channel, per pelanggan.
3. **OPERASIONAL**
   - `Transaksi`: Pencatatan pemasukan, pengeluaran, potongan marketplace/aplikasi, impor CSV/mutasi.
   - `Piutang & Utang`: Pelacakan tagihan pelanggan & supplier, reminder jatuh tempo (>30 hari).
   - `Stok & Gudang`: Kartu stok barang dagang/bahan baku, valuasi persediaan, alert stok menipis.
   - `Produksi & Supplier`: Resep/BOM (Bill of Materials — misal untuk FnB seperti *Kedai Kopi Senja*), konversi bahan baku ke barang jadi, manajemen pemasok.
4. **ALAT**
   - `Invoice`: Pembuatan & pengiriman faktur tagihan profesional (PDF/WhatsApp link) yang otomatis tercatat ke Piutang.
   - `Simulasi & Harga`: Kalkulator HPP, simulasi kenaikan harga bahan baku, simulasi diskon & komisi aplikasi (GoFood/GrabFood/ShopeeFood/Tokopedia/Shopee).
   - `Pajak UMKM`: Kalkulator & rekap omzet bulanan untuk PPh Final 0,5% (PP 55/2022), pemantauan kuota omzet bebas pajak Rp500 juta/tahun (WP OP), ekspor kertas kerja pelaporan SPT.
5. **PENGATURAN**
   - `Profil Usaha`: jenis & bentuk usaha, PTKP, status PKP.
   - `Impor Data Awal`: satu workbook .xlsx untuk onboarding — Neraca Awal (posisi kas/piutang/persediaan/utang/pinjaman di hari pertama pakai aplikasi), Aset, Karyawan, dan Transaksi. Pratinjau dulu, baru commit.
   - `Karyawan`: master data karyawan & rekap beban gaji bulanan (PPh 21 belum).
   - `Master Data`: Produk, Kategori Biaya, Channel Penjualan & % Komisi, Akun Kas/Bank, Target Bulanan/Harian.
   - `Panduan`: Dokumentasi interaktif & *demo mode* (`Kedai Kopi Senja — Data contoh tidak disimpan`).

---

## 3. Aturan Teknis Skala 1.000 User — Server Lokal (ElysiaJS vs Go)

- **Target Beban**: 1.000 tenant UMKM aktif (~5 juta baris transaksi/tahun total seluruh sistem, ~50–200 request/detik saat jam sibuk).
- **Kapasitas Server**: Cukup **1 server lokal (4 vCPU, 8 GB RAM, SSD)** dengan **Single-Node PostgreSQL 16/17** + **Single Binary Backend (Go atau Bun/Elysia)**.
- **Dilarang Over-Engineering (`ponytail`)**:
  - ❌ Jangan gunakan microservices, Kafka, RabbitMQ, atau Kubernetes.
  - ❌ Jangan gunakan Redis terpisah kecuali dibutuhkan nanti; gunakan in-memory cache (`sync.Map`/LRU di Go atau `Map` di Bun) untuk data agregasi dashboard yang di-invalidate saat tenant menambah transaksi.
  - ✅ Gunakan **Multi-tenancy via `tenant_id` + PostgreSQL Row-Level Security (RLS)** atau *strict tenant middleware*.
  - ✅ Semua nilai uang disimpan sebagai `BIGINT` (Rupiah utuh, tanpa desimal pecahan sen kecuali kurs asing diperlukan).
  - ✅ Backup PostgreSQL harian (`pg_dump`) ke disk/lokasi terpisah — server lokal tidak punya snapshot otomatis seperti VPS.
  - ✅ Akses publik (webhook Midtrans, landing) lewat reverse proxy/tunnel (mis. Cloudflare Tunnel), bukan port-forward langsung.

---

## 4. Checklist Sesi Sebelum Commit / Selesai

- [ ] Apakah semua perubahan sudah mengikuti `ponytail` (kode paling simpel, tanpa abstraksi spekulatif)?
- [ ] Apakah logika finansial & pajak sudah memiliki unit test (`tdd`) yang lulus 100%?
- [ ] Bila menyentuh jalur Transaksi → Jurnal → Laporan atau Panel Master: apakah `go test ./... -run '^$' -bench .` masih di kisaran angka yang tercatat di `§1.B` (`smokinggun`)?
- [ ] Apakah dokumen (`BRD.md`, `FRD.md`, `TRD.md`, `PRD.md`) dan `graphify` tetap sinkron?
- [ ] Apakah UI sudah bebas dari *AI slop* (`antislop`, `taste`, `impeccable`) dan terverifikasi lewat `agent browser`?
- [ ] Apakah siklus `harness2` (Propose → Probe → Compose) sudah dijalankan dan hasilnya dicatat di `handoff`?
- [ ] Apakah ringkasan `handoff` sudah ditulis untuk sesi berikutnya?

