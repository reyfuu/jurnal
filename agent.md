# AGENT.md — Laporan Bisnis (by Akademi Pajak)

> **Project**: Laporan Bisnis by Akademi Pajak — SaaS Pembukuan, Operasional & Pajak UMKM ("Gambaran seluruh bisnis dalam satu layar")  
> **Target Scale**: 100 Active UMKM Tenants (~4.000–10.000 transaksi/tenant/tahun)  
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
4. **`grillme` & `grillme with docs`**
   - Setiap ada ambiguitas bisnis/teknis/visual, lakukan interogasi desain singkat & tajam (*grill me*) yang mengacu langsung pada dokumen [BRD](file:///home/reyfuu/pajak/docs/BRD.md), [FRD](file:///home/reyfuu/pajak/docs/FRD.md), [TRD](file:///home/reyfuu/pajak/docs/TRD.md), [PRD](file:///home/reyfuu/pajak/docs/PRD.md), [DESIGN_BRIEF](file:///home/reyfuu/pajak/docs/DESIGN_BRIEF.md), dan [DESIGN.md](file:///home/reyfuu/pajak/DESIGN.md).
5. **`skillspector`**
   - Audit kesiapan skill di awal sesi; pastikan tidak ada regresi aturan dari skill yang terpasang.
6. **`handoff`**
   - Setiap akhir sesi atau sebelum pergantian konteks, perbarui status progres, keputusan arsitektur, dan *next steps* agar sesi berikutnya bisa langsung melanjutkan tanpa kehilangan konteks.

### B. Engineering Discipline & Code Quality
7. **`ponytail` (Lazy Senior Dev / YAGNI Mode — Active Every Response)**
   - Terapkan *The Ponytail Ladder*:
     1. *Does this need to exist at all?* (YAGNI — kapasitas target adalah **100 user**, jangan bangun infrastruktur sekelas 1 juta user seperti Kafka/Kubernetes/Microservices).
     2. *Already in this codebase?* Gunakan ulang helper/tipe yang ada.
     3. *Stdlib / Native platform first* (`net/http` / `database/sql` di Go, atau native `Bun.serve` / `Elysia` di Bun, constraint PostgreSQL di DB).
     4. Tandai kompromi kapasitas yang disengaja dengan komentar `// ponytail: <alasan & batas upgrade>`.
8. **`tdd` (Test-Driven Development)**
   - **Wajib Red-Green-Refactor** terutama pada domain kritikal finansial:
     - Kalkulasi Omzet, HPP (COGS), Laba Kotor, Laba Bersih, dan Margin Laba.
     - Kalkulasi Arus Kas vs Laba Akrual ("Kas bisa turun walau laba positif").
     - Kalkulasi Pajak UMKM (PPh Final 0,5% PP 55/2022, batas bebas pajak Rp500 juta/tahun untuk WP OP, potongan aplikasi/channel).
     - Penuaan Piutang & Utang (*aging schedule* > 30 hari).
   - Jangan pernah menggunakan `float64` / `number` floating-point biasa untuk uang; gunakan `int64` (Rupiah penuh / sen) atau `decimal`.
9. **`improve-codebase-architecture` & `setup-matt-pocock-skills`**
   - Struktur modular berbasis **Vertical Slices / Domain Modules** (`dashboard`, `ledger`, `cashflow`, `transactions`, `receivables`, `inventory`, `production`, `invoicing`, `pricing_sim`, `tax_umkm`, `master_data`).
   - *Make illegal states unrepresentable*: gunakan strict schema validation (TypeBox di Elysia / strict structs & sqlc di Go), branded types untuk ID & nominal Rupiah, dan eksplisit error handling.

### C. UI/UX, Visual Craft & Verification
10. **`antislop`, `taste`, & `impeccable`**
    - **DILARANG KERAS** membuat UI generik "AI slop" (gradien ungu-biru murahan, bayangan berlebihan, kartu tanpa hierarki, teks placeholder asal-asalan).
    - Ikuti estetika referensi **Laporan Bisnis by Akademi Pajak**:
      - **Dark Obsidian Theme**: Background utama `#0B0C0E`, surface kartu `#121418`, border halus `#1F2229` (`1px solid rgba(255,255,255,0.06)`).
      - **Brand Accent**: Amber/Gold `#EAB308` / `#F59E0B` untuk metrik utama (Omzet, CTA `Beli Rp149.000`, active nav state).
      - **Semantic Financial Colors**: Emerald `#10B981` (Laba Bersih / tren positif), Sky Blue `#38BDF8` (Saldo Kas & Bank), Rose/Coral `#F43F5E` (peringatan piutang lewat 30 hari).
      - **Typography**: Angka finansial wajib menggunakan `font-variant-numeric: tabular-nums` agar sejajar sempurna.
11. **`ui ux pro max`, `21dev` (`21st.dev`), `shadcn/ui`, & `kokonut ui`**
    - Gunakan primitif `shadcn/ui` (Radix + Tailwind) yang diperkaya dengan komponen finansial dari `Kokonut UI` dan micro-interactions dari `21st.dev`.
    - Setiap KPI card di Dashboard wajib menampilkan:
      1. Label metrik yang jelas (`Omzet`, `Laba bersih`, `Margin laba`, `Saldo kas & bank`, `Target omzet`, `Piutang belum tertagih`).
      2. Nilai utama singkat & mudah dibaca (`Rp92 jt`, `Rp25 jt`, `27%`, `108%`).
      3. Badge perbandingan periode (`▲ 4% vs bulan lalu`, `+1,7 poin vs bulan lalu`) + mini sparkline / progress bar.
12. **`agent browser`**
    - Setiap perubahan frontend wajib diverifikasi secara visual menggunakan browser agent untuk memastikan tata letak 6 KPI card atas, 3 grafik tengah, 2 kartu bawah, dan sidebar navigasi presisi di desktop & mobile.

---

## 2. Domain Architecture Map (Laporan Bisnis)

Sesuai referensi desain produk **Laporan Bisnis by Akademi Pajak**, sistem terdiri dari 5 grup navigasi utama:

1. **RINGKASAN**
   - `Dashboard`: Ringkasan 6 KPI utama, grafik tren 11 bulan (`Omzet, biaya, dan laba`), grafik bar `Omzet harian` vs target harian, grafik area `Saldo kas per bulan`, `Omzet per produk` (Top 5), dan `Omzet per channel` (termasuk potongan aplikasi).
2. **LAPORAN**
   - `Laporan Keuangan`: Laba Rugi (Income Statement), Neraca (Balance Sheet), Perubahan Modal.
   - `Arus Kas`: Arus kas operasional, investasi, pendanaan, serta rekonsiliasi selisih Laba vs Kas.
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
   - `Master Data`: Produk, Kategori Biaya, Channel Penjualan & % Komisi, Akun Kas/Bank, Target Bulanan/Harian, Profil Usaha.
   - `Panduan`: Dokumentasi interaktif & *demo mode* (`Kedai Kopi Senja — Data contoh tidak disimpan`).

---

## 3. Aturan Teknis Skala 100 User (ElysiaJS vs Go)

- **Target Beban**: 100 tenant UMKM aktif (~500 ribu baris transaksi/tahun total seluruh sistem, ~5–20 request/detik saat jam sibuk).
- **Kapasitas Server**: Cukup **1 VPS (2 vCPU, 2–4 GB RAM)** dengan **Single-Node PostgreSQL 16/17** + **Single Binary Backend (Go atau Bun/Elysia)**.
- **Dilarang Over-Engineering (`ponytail`)**:
  - ❌ Jangan gunakan microservices, Kafka, RabbitMQ, atau Kubernetes.
  - ❌ Jangan gunakan Redis terpisah kecuali dibutuhkan nanti; gunakan in-memory cache (`sync.Map`/LRU di Go atau `Map` di Bun) untuk data agregasi dashboard yang di-invalidate saat tenant menambah transaksi.
  - ✅ Gunakan **Multi-tenancy via `tenant_id` + PostgreSQL Row-Level Security (RLS)** atau *strict tenant middleware*.
  - ✅ Semua nilai uang disimpan sebagai `BIGINT` (Rupiah utuh, tanpa desimal pecahan sen kecuali kurs asing diperlukan).

---

## 4. Checklist Sesi Sebelum Commit / Selesai

- [ ] Apakah semua perubahan sudah mengikuti `ponytail` (kode paling simpel, tanpa abstraksi spekulatif)?
- [ ] Apakah logika finansial & pajak sudah memiliki unit test (`tdd`) yang lulus 100%?
- [ ] Apakah dokumen (`BRD.md`, `FRD.md`, `TRD.md`, `PRD.md`) dan `graphify` tetap sinkron?
- [ ] Apakah UI sudah bebas dari *AI slop* (`antislop`, `taste`, `impeccable`) dan terverifikasi lewat `agent browser`?
- [ ] Apakah ringkasan `handoff` sudah ditulis untuk sesi berikutnya?

