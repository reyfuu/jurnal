# Product Requirements Document (PRD)
**Produk**: Laporan Bisnis by Akademi Pajak  
**Versi**: 1.0 (MVP Batch 100 User)  
**Design System**: Navy & Gold, laporan pada permukaan netral gelap (`shadcn/ui` + `Kokonut UI` + `21st.dev`, patuh `antislop` & `impeccable`)

---

## 1. Visi Produk & Value Proposition
**"Gambaran seluruh bisnis dalam satu layar."**  
Laporan Bisnis by Akademi Pajak membantu pemilik UMKM melihat kesehatan bisnis secara utuh—bukan sekadar omzet kotor, melainkan **Laba Bersih sesungguhnya**, **Saldo Kas & Bank riil**, **Piutang yang macet lewat 30 hari**, **Potongan komisi aplikasi**, hingga **kewajiban Pajak UMKM 0,5%** tanpa perlu mengerti akuntansi.

---

## 2. User Personas

| Persona | Profil | Pain Point Utama | Goal di Laporan Bisnis |
|---|---|---|---|
| **Raka — Pemilik Kedai Kopi Senja (FnB)** | 4.425 transaksi, jual offline + GoFood/GrabFood | "Omzet Rp92 juta sebulan tapi kok uang di BCA cuma naik dikit? Potongan aplikasi berapa sebenarnya?" | Melihat potongan aplikasi (`Rp4,6 jt`), membandingkan omzet harian dengan target `Rp3 jt/hari`, dan menjaga margin laba `27%`. |
| **Sinta — Supplier Kue & Catering (B2B/Produksi)** | Jual ke kantor & hotel dengan sistem invoice tempo | "Laba di catatan Rp25 juta, tapi mau belanja bahan baku kasnya kurang." | Memantau kartu **Piutang belum tertagih (`Rp11 jt`)** terutama alert merah **`Rp9,6 jt lewat 30 hari`** dan mengirim invoice pengingat. |
| **Hendra — Pemilik UMKM Menengah tanpa Staf Keuangan** | Distributor sembako, omzet Rp2 M/tahun, 6 karyawan gudang & sales, pembukuan dikerjakan sendiri malam hari | "Mau rekrut admin keuangan tapi gaji Rp4,5 jt berat, dan saya tidak tahu harus cari yang bisa apa." | Transaksi otomatis masuk CoA perdagangan, laporan laba rugi & neraca siap tiap bulan, peringatan kas minus, dan BEP. Saat akhirnya merekrut, laporan menjadi acuan tugas orang tersebut. |
| **Client — Pelanggan B2B UMKM** | Hotel / kantor yang membeli dengan invoice tempo | "Tagihan saya ke kedai ini berapa saja yang belum dibayar?" | Login dengan role Client, hanya melihat invoice & sisa tagihan miliknya sendiri. |
| **Budi — Pedagang Ritel UMKM (WP Orang Pribadi)** | Omzet Rp60–90 juta/bulan | "Takut salah hitung pajak UMKM dan belum tahu kalau omzet di bawah Rp500 juta setahun bebas PPh." | Membuka halaman **Pajak UMKM**, melihat sisa kuota bebas pajak Rp500 juta, dan mengunduh rekap omzet bulanan untuk SPT. |

---

## 3. Spesifikasi UI/UX & Design System (Sesuai Referensi Gambar)

### 3.1 Palet Warna & Tipografi (`antislop` + `taste` + `impeccable`)
- **Brand Utama**: Navy `#222852` pada login, onboarding, sidebar, favicon, dan tema browser; aksen emas `#F5C428` / `#EAB308` untuk logo dan CTA, dengan teks tombol navy. Keputusan pengguna 19 Oktober 2026; detail di `DESIGN.md` §2.
- **Canvas Background**: `#090A0C` (Latar belakang utama luar) / `#0F1115` (Shell aplikasi).
- **Card Surface**: `#14171D` dengan border `1px solid rgba(255, 255, 255, 0.06)` dan radius `14px` (`rounded-xl`).
- **Active Sidebar Pill**: `rgba(234, 179, 8, 0.14)` dengan teks `#FACC15` (Amber/Emas).
- **Warna Semantik Data**:
  - **Omzet / Target / Brand Accent**: `#EAB308` (Amber Gold)
  - **Laba Bersih / Pertumbuhan Positif**: `#10B981` (Emerald Green, badge pill `rgba(16, 185, 129, 0.15)`)
  - **Total Biaya**: `#D97706` (Warm Amber/Orange)
  - **Saldo Kas & Bank**: `#38BDF8` (Sky Blue)
  - **Alert Piutang Lewat 30 Hari**: `#F43F5E` (Rose Red)
- **Tipografi**:
  - Heading & Angka KPI: `Plus Jakarta Sans` atau `Inter` SemiBold/Bold, ukuran KPI `28px–32px`, wajib `tabular-nums`.

### 3.2 Tata Letak (Layout Hierarchy)
1. **Top Marketing Header (Saat di Landing / Demo Preview)**:
   - Kiri: Logo segitiga kuning **Laporan Bisnis** `by Akademi Pajak`.
   - Kanan: Link `Fitur`, `Cara kerja`, `Keamanan data`, `Harga`, tombol ghost `Masuk`, dan tombol utama kuning bulat `Beli Rp149.000`.
2. **App Sidebar (Kiri, Lebar `240px`)**:
   - **RINGKASAN**: `Dashboard`
   - **LAPORAN**: `Laporan Keuangan`, `Arus Kas`
   - **OPERASIONAL**: `Transaksi`, `Piutang & Utang`, `Stok & Gudang`, `Produksi & Supplier`
   - **ALAT**: `Invoice`, `Simulasi & Harga`, `Pajak UMKM`
   - **PENGATURAN**: `Master Data`, `Panduan`
   - **Footer Sidebar**: Badge inisial `KK` (`Kedai Kopi Senja` · `FnB · 4.425 transaksi`) + teks status `Data contoh tidak disimpan`.
3. **Dashboard Main Area (Grid Responsif)**:
   - **Header Bar**: Judul `Dashboard`, subjudul `Gambaran seluruh bisnis dalam satu layar`, dan pemilih bulan di kanan atas (`📅 Agustus 2026`).
   - **Baris 1 (6 Kolom KPI Cards)**:
     1. `Omzet` (`Rp92 jt`, `▲ 4% vs bulan lalu`, sparkline kuning)
     2. `Laba bersih` (`Rp25 jt`, `▲ 11% vs bulan lalu`, sparkline hijau)
     3. `Margin laba` (`27%`, `+1,7 poin vs bulan lalu`, sparkline kuning)
     4. `Saldo kas & bank` (`Rp97 jt`, `▲ 21% sejak awal bulan`, sparkline biru)
     5. `Target omzet` (`108%`, `Lebih Rp6,8 jt`, progress bar hijau)
     6. `Piutang belum tertagih` (`Rp11 jt`, `10 tagihan`, `Rp9,6 jt lewat 30 hari`, `Utang belum dibayar Rp7,4 jt`)
   - **Baris 2 (Grid Asimetris 2:1)**:
     - Kiri (Lebar ~65%, tinggi penuh 2 baris): Grafik area mulus **`Omzet, biaya, dan laba`** (`11 bulan terakhir`, sumbu Y `0`, `25 jt`, `50 jt`, `75 jt`, `100 jt`, sumbu X `Okt 25` s/d `Agu 26`).
     - Kanan Atas (Lebar ~35%): Grafik bar **`Omzet harian Agustus 2026`** (`Rata-rata Rp3 jt per hari`, bar kuning terang di atas garis putus-putus target harian, kuning redup di bawah target).
     - Kanan Bawah (Lebar ~35%): Grafik area **`Saldo kas per bulan`** (`Kas bisa turun walau laba positif: cek halaman Arus Kas.`).
   - **Baris 3 (2 Kolom Sama Lebar)**:
     - Kiri: **`Omzet per produk`** (`5 terlaris dari 7 produk`).
     - Kanan: **`Omzet per channel`** (`Potongan aplikasi bulan ini Rp4,6 jt`).

---

## 4. User Stories & Acceptance Criteria (Inti)

### US-01: Melihat Kesehatan Bisnis 1 Layar di Dashboard
- **Sebagai** pemilik UMKM, **saya ingin** melihat 6 KPI utama dan 5 blok visualisasi dalam satu layar tanpa scroll panjang yang membingungkan, **agar** saya tahu dalam 10 detik apakah bisnis saya untung dan kas saya aman bulan ini.
- **Acceptance Criteria**:
  - [ ] Mengubah dropdown bulan (misal `Juli 2026` ↔ `Agustus 2026`) memperbarui keenam KPI dan seluruh grafik secara instan (`< 100ms`).
  - [ ] Jika ada piutang yang melewati `due_date` > 30 hari, nominalnya tampil dengan warna merah (`#F43F5E`) di kartu ke-6.

### US-02: Memahami Selisih Laba Bersih vs Saldo Kas
- **Sebagai** pemilik UMKM, **saya ingin** mengetahui mengapa kas saya turun atau tidak sebesar laba bersih, **agar** saya tidak salah mengambil uang kas untuk keperluan pribadi.
- **Acceptance Criteria**:
  - [ ] Grafik `Saldo kas per bulan` menampilkan subteks peringatan edukatif dan tautan ke halaman `Arus Kas`.
  - [ ] Halaman `Arus Kas` menampilkan ringkasan rekonsiliasi: `Laba Bersih + Utang Baru - Piutang Belum Tertagih - Pembelian Stok = Perubahan Kas Bersih`.

### US-03: Menghitung Potongan Aplikasi & Simulasi Harga
- **Sebagai** pemilik FnB/Ritel, **saya ingin** mengetahui total potongan komisi GoFood/GrabFood/Shopee bulan ini dan melakukan simulasi harga jual, **agar** margin laba bersih saya tidak tergerus komisi platform.
- **Acceptance Criteria**:
  - [ ] Kartu `Omzet per channel` otomatis menjumlahkan seluruh potongan komisi aplikasi pada bulan berjalan (`Potongan aplikasi bulan ini Rp4,6 jt`).
  - [ ] Halaman `Simulasi & Harga` menghitung harga jual minimum per channel untuk mencapai target margin yang diinginkan.

### US-04: Rekap Pajak UMKM (PP 55/2022)
- **Sebagai** Wajib Pajak UMKM, **saya ingin** melihat tabel omzet bulanan dan jumlah PPh Final 0,5% yang harus dibayar, **agar** saya patuh pajak tanpa perlu menyewa konsultan mahal.
- **Acceptance Criteria**:
  - [ ] Untuk WP Orang Pribadi, bulan-bulan dengan akumulasi omzet tahunan $\le \text{Rp500.000.000}$ otomatis bertarif **Rp0 (Bebas Pajak)**.
  - [ ] Bulan saat akumulasi omzet menembus Rp500 juta hanya menghitung `0,5%` dari selisih di atas Rp500 juta.

---

## 5. Roadmap Implementasi (Skala 100 User)

- **Fase 1 (Foundation & Dashboard + Demo Mode `Kedai Kopi Senja`)**:
  - Setup project, skema database PostgreSQL, seed data realistis `Kedai Kopi Senja` (4.425 transaksi, 11 bulan histori), Landing Header + Dashboard lengkap sesuai referensi visual.
- **Fase 2 (Operasional & Laporan)**:
  - CRUD Transaksi, Piutang & Utang, Stok & Gudang, Produksi & Supplier, Laporan Keuangan, dan Arus Kas.
- **Fase 3 (Alat UMKM & Monetisasi Rp149.000)**:
  - Generator Invoice PDF/WA, Kalkulator Simulasi & Harga, Modul Pajak UMKM PP 55/2022, serta integrasi checkout `Beli Rp149.000`.

### Roadmap "Staf Keuangan Virtual" (pain point UMKM menengah, Okt 2026)
- **Fase A (selesai)**: tanpa emoji di seluruh UI/AI; Simulasi Harga dengan BEP, biaya tetap & laba bersih; peringatan saldo kas minus (transaksi tetap tersimpan); Laporan Penjualan (tren harian/mingguan/bulanan, produk terlaris, per channel, per pelanggan).
- **Fase B**: persistensi SQLite (data tidak hilang saat deploy); KPI dashboard dihitung murni dari transaksi.
- **Fase C**: CoA otomatis dengan template **perdagangan, jasa, manufaktur**; transaksi dipetakan ke akun otomatis dan koreksi user diingat; Laba Rugi & Neraca per template; ekspor PDF/Excel; stok per transaksi dan peringatan stok minus.
- **Fase D**: role **Client** (pelanggan B2B melihat tagihannya sendiri); upload bukti/nota dengan kompresi otomatis (foto, PDF, Excel).
- **Fase E**: ekspor XML Coretax: rekap PPh Final 0,5%, e-Bupot PPh 21/23, e-Faktur PPN (format diverifikasi terhadap template resmi DJP).

