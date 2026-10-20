# Functional Requirements Document (FRD)
**Produk**: Laporan Bisnis by Master Pajak  
**Referensi Visual**: Desain Dashboard Laporan Bisnis (`Agustus 2026` — *Kedai Kopi Senja*)

---

## 1. Autentikasi, Pembelian & Demo Mode
- **FR-AUTH-01 (Demo Mode Instan)**: Pengunjung dapat mencoba aplikasi menggunakan dataset pratinjau `Kedai Kopi Senja (FnB · 4.425 transaksi)` dengan indikator jelas `"Data contoh tidak disimpan"`.
- **FR-AUTH-02 (Checkout & Aktivasi Rp149.000)**: Pengguna membeli akses melalui tombol `Beli Rp149.000` (integrasi QRIS/Virtual Account), akun otomatis aktif setelah webhook pembayaran sukses.
- **FR-AUTH-03 (Multi-Tenant Isolation)**: Setiap pengguna memiliki workspace bisnis terpisah (`tenant_id`). Semua query wajib terfilter berdasarkan `tenant_id`.

---

## 2. Modul RINGKASAN — Dashboard (` Gambaran seluruh bisnis dalam satu layar `)
Filter global periode: **Bulan & Tahun** (contoh: `Agustus 2026`).

### 2.1 Enam Kartu KPI Utama (Top Row)
| ID | Nama Kartu | Rumus / Logika Kalkulasi | Elemen Kontekstual & Visual |
|---|---|---|---|
| **FR-DB-01** | **Omzet** | $\sum \text{Penjualan Kotor (Gross Sales)}$ pada bulan terpilih | Nilai singkat (`Rp92 jt`), badge `% vs bulan lalu` (`▲ 4% vs bulan lalu`), sparkline kuning. |
| **FR-DB-02** | **Laba bersih** | $\text{Omzet} - \text{Potongan Channel} - \text{HPP} - \text{Biaya Operasional} - \text{Pajak}$ | Nilai singkat (`Rp25 jt`), badge `% vs bulan lalu` (`▲ 11% vs bulan lalu`), sparkline hijau. |
| **FR-DB-03** | **Margin laba** | $(\text{Laba Bersih} \div \text{Omzet}) \times 100\%$ | Persentase (`27%`), selisih poin persentase (`+1,7 poin vs bulan lalu`), sparkline kuning. |
| **FR-DB-04** | **Saldo kas & bank** | $\text{Saldo Awal Kas} + \sum \text{Kas Masuk} - \sum \text{Kas Keluar}$ s/d akhir bulan terpilih | Nilai (`Rp97 jt`), badge perubahan sejak awal bulan (`▲ 21% sejak awal bulan`), sparkline biru. |
| **FR-DB-05** | **Target omzet** | $(\text{Omzet Aktual Bulan Ini} \div \text{Target Omzet Bulanan}) \times 100\%$ | Persentase (`108%`), nominal selisih (`Lebih Rp6,8 jt` atau `Kurang Rp...`), progress bar hijau. |
| **FR-DB-06** | **Piutang belum tertagih** | $\sum \text{Sisa Tagihan Invoice Belum Lunas}$ | Total piutang (`Rp11 jt`), jumlah tagihan (`10 tagihan`), highlight merah piutang jatuh tempo `Rp9,6 jt lewat 30 hari`, serta info `Utang belum dibayar Rp7,4 jt`. |

### 2.2 Grafik & Analitik Dashboard
- **FR-DB-07 (Grafik `Omzet, biaya, dan laba`)**:
  - Menampilkan tren **11–12 bulan terakhir** (contoh: `Okt 25` s/d `Agu 26`).
  - Tiga seri kurva area/garis: **Omzet** (kuning), **Total biaya** (oranye/amber), dan **Laba bersih** (hijau).
- **FR-DB-08 (Grafik `Omzet harian [Bulan Tahun]`)**:
  - Menampilkan bar chart omzet per tanggal (hari 1 s/d 30/31) beserta teks ringkasan `Rata-rata Rp3 jt per hari`.
  - Bar berwarna kuning terang jika **Di atas target** harian, kuning redup jika **Di bawah target**, dilengkapi garis putus-putus horizontal **Target harian**.
- **FR-DB-09 (Grafik `Saldo kas per bulan`)**:
  - Menampilkan kurva **Kas akhir bulan** selama 11 bulan terakhir.
  - Menampilkan teks edukatif: *"Kas bisa turun walau laba positif: cek halaman Arus Kas."*
- **FR-DB-10 (Kartu `Omzet per produk`)**:
  - Menampilkan peringkat produk penyumbang omzet tertinggi (`5 terlaris dari N produk`), lengkap dengan kuantitas terjual, nominal omzet, dan persentase kontribusi.
- **FR-DB-11 (Kartu `Omzet per channel`)**:
  - Menampilkan breakdown omzet berdasarkan channel penjualan (Dine-in/Offline, GoFood, GrabFood, ShopeeFood, WhatsApp/B2B) serta highlight **`Potongan aplikasi bulan ini Rp4,6 jt`**.

---

## 3. Modul LAPORAN
- **FR-REP-01 (`Laporan Keuangan`)**:
  - **Laporan Laba Rugi**: Omzet Penjualan → Potongan/Diskon Channel → Penjualan Bersih → HPP (Bahan Baku/Barang Dagang) → Laba Kotor → Beban Operasional (Gaji, Sewa, Listrik/Internet, Pemasaran, Lainnya) → Laba Bersih.
  - **Neraca Sederhana**: Aset (Kas & Bank, Piutang Usaha, Nilai Persediaan Stok) vs Liabilitas (Utang Usaha) & Ekuitas.
  - Ekspor ke PDF & Excel/CSV.
- **FR-REP-02 (`Arus Kas`)**:
  - Arus Kas Masuk vs Arus Kas Keluar dikelompokkan menjadi Aktivitas Operasional, Pembelian Aset/Inventori, dan Pendanaan/Prive.
  - **Jembatan Rekonsiliasi Laba vs Kas**: Menjelaskan secara bahasa manusia kenapa Laba Bersih `Rp25 jt` berbeda dengan kenaikan/penurunan Kas (misal: karena Rp11 jt masih nyangkut di Piutang atau dipakai beli stok).

---

## 4. Modul OPERASIONAL
- **FR-OPS-01 (`Transaksi`)**:
  - Input transaksi cepat (< 10 detik): Tanggal, Jenis (Pemasukan / Pengeluaran / Transfer Kas), Produk & Qty, Channel Penjualan (otomatis hitung potongan komisi % jika ada), Metode Pembayaran (Tunai/Bank/Tempo), dan Catatan.
  - Impor transaksi massal dari file CSV/Excel.
- **FR-OPS-02 (`Piutang & Utang`)**:
  - Daftar piutang pelanggan & utang supplier dengan status (`Belum Jatuh Tempo`, `Jatuh Tempo < 30 Hari`, `Lewat 30 Hari`).
  - Pencatatan pembayaran cicilan atau pelunasan penuh yang otomatis menambah/mengurangi Saldo Kas & Bank.
- **FR-OPS-03 (`Stok & Gudang`)**:
  - Pelacakan stok bahan baku & barang jadi (Masuk, Keluar, Penyesuaian/Opname).
  - Perhitungan HPP rata-rata tertimbang (*Weighted Average Cost*) otomatis.
- **FR-OPS-04 (`Produksi & Supplier`)**:
  - Definisi resep / *Bill of Materials (BOM)* sederhana (contoh: 1 Cup Kopi Susu = 18g Biji Kopi + 150ml Susu + 20ml Gula Aren + 1 Cup/Lid).
  - Pencatatan batch produksi yang memotong stok bahan baku dan menambah stok produk jadi, serta direktori supplier.

---

## 5. Modul ALAT
- **FR-TOOL-01 (`Invoice`)**:
  - Pembuatan invoice bernomor otomatis, logo bisnis, rincian item, tanggal jatuh tempo, dan rekening tujuan.
  - Unduh PDF atau salin pesan tagihan WhatsApp sekali klik.
- **FR-TOOL-02 (`Simulasi & Harga`)**:
  - Kalkulator harga jual per channel: masukkan HPP produk + target margin + % komisi aplikasi (misal GoFood 20% + Rp1.000) → rekomendasikan harga jual di aplikasi agar margin bersih tidak tergerus.
- **FR-TOOL-03 (`Pajak UMKM`)**:
  - Rekapitulasi Peredaran Bruto (Omzet) bulanan dari Januari–Desember.
  - Pilihan profil Wajib Pajak:
    - **WP Orang Pribadi (PP 55/2022)**: Kumulatif omzet s/d **Rp500.000.000/tahun bebas PPh Final (Rp0)**; hanya omzet di atas Rp500 juta yang dikenakan tarif **0,5%**.
    - **WP Badan (PT/CV/Koperasi UMKM)**: Tarif **0,5%** dari omzet bulanan sejak rupiah pertama (selama omzet tahunan < Rp4,8 Miliar).
  - Cetak/unduh tabel Rekapitulasi Peredaran Bruto resmi untuk lampiran SPT Tahunan.

---

## 6. Modul PENGATURAN
- **FR-SET-01 (`Master Data`)**: CRUD Produk, Bahan Baku, Kategori Biaya, Channel & Komisi, Akun Kas/Bank, serta Target Omzet Bulanan/Harian.
- **FR-SET-02 (`Panduan`)**: Kamus istilah keuangan UMKM singkat, video/artikel cara pakai, dan tombol reset/keluar dari Demo Mode.

