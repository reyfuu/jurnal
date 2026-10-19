# Design Brief — Laporan Bisnis by Akademi Pajak

> Dokumen ringkas ini mendampingi [BRD.md](file:///home/reyfuu/pajak/docs/BRD.md), [FRD.md](file:///home/reyfuu/pajak/docs/FRD.md), [TRD.md](file:///home/reyfuu/pajak/docs/TRD.md), dan [PRD.md](file:///home/reyfuu/pajak/docs/PRD.md). Spesifikasi sistem desain teknis lengkap tersedia di [DESIGN.md](file:///home/reyfuu/pajak/DESIGN.md).

---

## 1. Ringkasan Proyek Desain
- **Nama Produk**: Laporan Bisnis (`by Akademi Pajak`)
- **Tagline UI**: *"Gambaran seluruh bisnis dalam satu layar"*
- **Target Pengguna**: Pemilik UMKM Indonesia (FnB seperti *Kedai Kopi Senja*, Ritel, Jasa, Produksi) yang menginginkan kejelasan laba bersih, arus kas, piutang macet, dan pajak UMKM tanpa kerumitan software akuntansi tradisional.
- **Harga & CTA Utama**: `Beli Rp149.000` (Tombol aksen kuning emas di sudut kanan atas).

---

## 2. Masalah UX yang Diselesaikan oleh Desain Ini
1. **Ilusi Omzet Besar**: Pemilik UMKM sering merasa bisnisnya ramai, padahal tergerus biaya bahan baku dan komisi aplikasi (`Potongan aplikasi bulan ini Rp4,6 jt`). Desain ini menyandingkan **Omzet (`Rp92 jt`)**, **Laba bersih (`Rp25 jt`)**, dan **Margin laba (`27%`)** secara berdampingan di baris pertama.
2. **Anomali "Untung di Kertas tapi Kas Kosong"**: Kartu **Saldo kas & bank (`Rp97 jt`)** dan grafik **Saldo kas per bulan** secara eksplisit mengedukasi pengguna: *"Kas bisa turun walau laba positif: cek halaman Arus Kas."*
3. **Tagihan Pelanggan Lewat Jatuh Tempo**: Kartu ke-6 (**Piutang belum tertagih `Rp11 jt`**) langsung menyorot warna merah **`Rp9,6 jt lewat 30 hari`** dan menyandingkannya dengan kewajiban **`Utang belum dibayar Rp7,4 jt`**.

---

## 3. Struktur Visual & Hierarki Informasi
1. **Top Marketing Navbar**: Menampilkan identitas brand, 4 menu informasi (`Fitur`, `Cara kerja`, `Keamanan data`, `Harga`), tombol `Masuk`, dan tombol konversi `Beli Rp149.000`.
2. **Left Sidebar (`232px`)**: Mengelompokkan 12 fitur ke dalam 5 kategori logis (`RINGKASAN`, `LAPORAN`, `OPERASIONAL`, `ALAT`, `PENGATURAN`) plus kartu status tenant demo di bagian bawah (`KK · Kedai Kopi Senja · FnB · 4.425 transaksi`).
3. **Baris 1 — 6 KPI Cards**: Ringkasan metrik makro dengan *sparkline* tren, *progress bar* target, dan *aging alert* piutang.
4. **Baris 2 — Bento Analytics (65% : 35%)**:
   - Kiri: Grafik komparasi multi-seri 11 bulan (`Omzet`, `Total biaya`, `Laba bersih`).
   - Kanan Atas: Evaluasi disiplin target harian (`Omzet harian Agustus 2026` — `Rata-rata Rp3 jt per hari`).
   - Kanan Bawah: Tren likuiditas bulanan (`Saldo kas per bulan`).
5. **Baris 3 — Breakdown Operasional (50% : 50%)**:
   - Kiri: `Omzet per produk` (`5 terlaris dari 7 produk`).
   - Kanan: `Omzet per channel` (`Potongan aplikasi bulan ini Rp4,6 jt`).

---

## 4. Aturan Implementasi Komponen (`shadcn/ui` + `Kokonut UI` + `21st.dev`)
- Warna utama brand: **navy `#222852`** (keputusan pengguna, 19 Oktober 2026), dengan aksen emas `#F5C428` / `#EAB308`. Navy tampil pada login, onboarding, sidebar, favicon, dan tema browser; teks tombol emas memakai navy. Teks sekunder sidebar `#BFC5D8` dan muted `#AAB1CA`. Detail penggunaan ada di `DESIGN.md` §2.
- Gunakan primitif `Card`, `Badge`, `Select`, dan `Tooltip` dari **`shadcn/ui`** yang dikustomisasi ke palet *Dark Obsidian* (`#0F1115` / `#161920`).
- Gunakan pola kartu finansial dari **`Kokonut UI`** dan transisi halus dari **`21st.dev`**.
- Pastikan semua nominal dan persentase memakai `font-variant-numeric: tabular-nums`.

