# Business Requirements Document (BRD)
**Produk**: Laporan Bisnis by Akademi Pajak  
**Tagline**: *"Gambaran seluruh bisnis dalam satu layar"*  
**Target Skala Awal**: 100 UMKM Aktif (Early Adopters / Batch Pertama)  
**Model Harga**: Rp149.000 (Sekali bayar / Akses tahunan terjangkau untuk UMKM)

---

## 1. Latar Belakang & Masalah Bisnis (Problem Statement)
Pemilik UMKM di Indonesia (FnB seperti *Kedai Kopi Senja*, ritel, jasa, dan manufaktur kecil) menghadapi 4 masalah utama:
1. **Buta Finansial Harian**: Omzet terlihat besar di aplikasi kasir/marketplace, tetapi kas di bank justru menyusut karena potongan komisi aplikasi (GoFood/Grab/Shopee), biaya bahan baku yang bocor, dan piutang pelanggan yang macet.
2. **Bingung Bedakan Laba vs Arus Kas**: Banyak pemilik bisnis mengira laba positif berarti uang kas tersedia, padahal tertahan di piutang (>30 hari) atau stok menumpuk.
3. **Alat Akuntansi Terlalu Rumit & Mahal**: Software ERP/akuntansi konvensional mengharuskan pemilik UMKM memahami jurnal debit-kredit dan berbiaya Rp300.000–Rp1.500.000/bulan.
4. **Ketakutan & Ketidaktahuan Pajak UMKM**: Pemilik UMKM kesulitan menghitung PPh Final 0,5% (PP 55/2022) dan tidak tahu bahwa Wajib Pajak Orang Pribadi memiliki kuota bebas pajak hingga omzet Rp500 juta/tahun.

---

## 2. Tujuan Bisnis (Business Objectives)
| ID | Objektif | Target terukur (Skala 100 User) |
|---|---|---|
| **BO-01** | Akuisisi Batch Pertama | Mencapai **100 pembeli aktif** dengan harga **Rp149.000** (Total Gross Revenue awal: **Rp14.900.000**). |
| **BO-02** | Efisiensi Infrastruktur (*Ponytail Principle*) | Biaya server bulanan **< Rp150.000/bulan** (1 VPS kecil), sehingga *gross margin* infrastruktur > 90% tahun pertama. |
| **BO-03** | Time-to-Value Cepat | User baru memahami kondisi bisnisnya dalam **< 3 menit** melalui *Demo Mode* (`Kedai Kopi Senja`) dan input transaksi tanpa istilah akuntansi rumit. |
| **BO-04** | Kepatuhan Pajak UMKM | 100% pengguna dapat mengekspor rekap peredaran bruto (omzet) bulanan & tahunan siap pakai untuk pelaporan SPT Tahunan / PPh Final 0,5%. |

---

## 3. Target Segmen Pengguna (Target Audience)
1. **FnB & Kedai / Resto Kecil** (Contoh: *Kedai Kopi Senja* — ~4.000+ transaksi/tahun, multi-channel offline + GoFood/GrabFood/ShopeeFood, butuh pantau HPP resep & potongan aplikasi).
2. **Pedagang Online & Ritel** (Toko Shopee/Tokopedia/TikTok Shop + offline yang pusing menghitung potongan admin platform & stok gudang).
3. **UMKM B2B / Jasa / Produksi Kecil** (Memiliki tagihan invoice ke pelanggan dengan termin pembayaran dan risiko piutang lewat 30 hari).

---

## 4. Ruang Lingkup Bisnis (Scope)
### In-Scope (MVP 100 User)
- Landing page konversi tinggi (`Fitur`, `Cara kerja`, `Keamanan data`, `Harga`, `Masuk`, `Beli Rp149.000`) + *Interactive Live Demo* (`Kedai Kopi Senja`).
- Modul **Ringkasan**: Dashboard eksekutif 1 layar (6 KPI utama, grafik tren 11 bulan, omzet harian vs target, saldo kas bulanan, omzet per produk, omzet per channel & potongan aplikasi).
- Modul **Laporan**: Laporan Keuangan (Laba Rugi & Neraca sederhana) dan Laporan Arus Kas (dengan edukasi rekonsiliasi Laba vs Kas).
- Modul **Operasional**: Transaksi, Piutang & Utang (dengan *aging* > 30 hari), Stok & Gudang, Produksi & Supplier.
- Modul **Alat**: Pembuat Invoice, Simulasi & Harga (kalkulator HPP & margin setelah potongan aplikasi), dan Pajak UMKM (PP 55/2022).
- Modul **Pengaturan**: Master Data & Panduan.

### Out-of-Scope (Fase 1 — YAGNI)
- Integrasi mutasi bank otomatis berbayar mahal (Brick/Ayoconnect) → Cukup input cepat & impor CSV/Excel mutasi.
- Multi-cabang kompleks / konsolidasi holding perusahaan.
- Pemotongan pajak korporasi kompleks (PPh 21 karyawan berlapis, PPh 23/26, e-Faktur PPN PKP) → Fokus khusus pada Pajak UMKM.

---

## 5. Model Finansial & Unit Economics (Skala 100 User)
- **Pendapatan 100 User**: `100 × Rp149.000 = Rp14.900.000`
- **Biaya Payment Gateway (QRIS 0,7%)**: `~Rp104.300`
- **Biaya Server (VPS 2 vCPU / 2–4 GB RAM + Backup S3/R2)**: `~Rp80.000–Rp120.000 / bulan` (`~Rp1.200.000 / tahun`)
- **Net Margin Operasional Sistem**: **~91%** (Sangat sehat karena menggunakan arsitektur efisien berbasis **Go** atau **Elysia/Bun**).

