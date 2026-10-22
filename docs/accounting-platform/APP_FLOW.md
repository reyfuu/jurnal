# App Flow — Platform Akuntansi Jasa & Dagang

**Status:** Alur baseline MVP  
**Rujukan:** [FRD](FRD.md) · [PRD](PRD.md) · [Design Brief](DESIGN_BRIEF.md)

## 1. Setup perusahaan dan saldo awal

```mermaid
flowchart TD
  A[Masuk] --> B{Membership perusahaan valid?}
  B -- Tidak --> X[Tolak akses dan catat hasil]
  B -- Ya --> C{Setup selesai?}
  C -- Tidak --> D[Profil perusahaan dan tahun buku]
  D --> E[COA dan mapping akun]
  E --> F[Periode fiskal]
  F --> G{Ada saldo awal?}
  G -- Tidak --> H[Mulai saldo nol]
  G -- Ya --> I[Input GL + rincian AR/AP + stok]
  I --> J[Validasi dan rekonsiliasi preview]
  J --> K{Seimbang dan disetujui?}
  K -- Tidak --> I
  K -- Ya --> L[Commit batch saldo awal]
  H --> M[Dashboard perusahaan]
  L --> M
  C -- Ya --> M
```

## 2. Invoice penjualan

```mermaid
flowchart TD
  A[Penjualan] --> B[Buat draft invoice]
  B --> C[Isi pihak, tanggal, jatuh tempo, item/jasa, kuantitas, harga]
  C --> D[Validasi akun, periode, perusahaan, dan stok]
  D --> E{Valid?}
  E -- Tidak --> F[Tampilkan error per field/baris]
  F --> C
  E -- Ya --> G[Preview jurnal, AR, dan dampak stok/HPP]
  G --> H{Konfirmasi posting?}
  H -- Tidak --> B
  H -- Ya --> I[Posting atomik dengan idempotency key]
  I --> J{Berhasil?}
  J -- Tidak --> K[Rollback; tampilkan sebab dan retry aman]
  J -- Ya --> L[Invoice POSTED + jurnal + mutasi stok]
  L --> M[Cetak / lihat detail / catat pembayaran]
  M --> N[Alokasikan pembayaran ke invoice]
  N --> O{Alokasi tidak melebihi outstanding?}
  O -- Tidak --> P[Tolak dan tampilkan saldo tersisa]
  O -- Ya --> Q[Posting kas dan alokasi atomik]
  Q --> R[Perbarui status unpaid/partial/paid]
```

Untuk jasa, alur invoice sama tetapi tidak menimbulkan mutasi stok/HPP persediaan.

## 3. Bill pembelian dan pembayaran

```mermaid
flowchart TD
  A[Pembelian] --> B[Buat draft bill]
  B --> C[Isi pemasok, tanggal, jatuh tempo, item/jasa, kuantitas, biaya]
  C --> D[Validasi akun, periode, perusahaan, dan nilai]
  D --> E{Valid?}
  E -- Tidak --> F[Tampilkan error per field/baris]
  F --> C
  E -- Ya --> G[Pilih opsi bayar langsung penuh atau simpan sebagai utang]
  G --> H{Bayar penuh saat posting?}
  H -- Ya --> I[Masukkan rekening dan detail pembayaran]
  H -- Tidak --> J[Bill dengan AP outstanding]
  I --> K[Preview bill + pembayaran, jurnal, dan stok]
  J --> L[Preview bill, jurnal, dan stok]
  K --> M{Konfirmasi bill + pembayaran?}
  L --> N{Konfirmasi bill?}
  M -- Tidak --> B
  N -- Tidak --> B
  M -- Ya --> O[Commit bill + payment atomik dengan idempotency key]
  N -- Ya --> P[Commit bill + AP atomik dengan idempotency key]
  O --> Q[Perbarui kartu stok, GL, dan status lunas]
  P --> R[Bill POSTED; AP outstanding]
  R --> S[Pembayaran parsial/penuh dari aksi Bayar]
  S --> T[Validasi alokasi tidak melebihi outstanding]
  T --> U[Posting kas dan alokasi atomik]
  U --> V[Perbarui status dan kartu pemasok]
```

Untuk jasa, bill tidak menerima stok dan tidak menghitung HPP persediaan.

## 4. Kas/bank dan jurnal manual

```mermaid
flowchart TD
  A[Kas & Bank / Jurnal] --> B{Jenis transaksi}
  B -- Setoran modal --> C[Debit Bank; kredit Modal]
  B -- Penerimaan/pengeluaran --> D[Pilih akun lawan dan isi pihak/keterangan]
  B -- Transfer --> E[Pilih rekening sumber dan tujuan]
  B -- Jurnal manual --> F[Isi baris debit dan kredit]
  C --> G[Isi tanggal, rekening, nominal]
  D --> H[Isi tanggal, rekening, nominal]
  E --> I[Isi tanggal dan nominal]
  F --> J[Validasi akun aktif dan total debit = kredit]
  G --> K[Preview jurnal dan periode]
  H --> K
  I --> K
  J --> K
  K --> L{Periode terbuka dan valid?}
  L -- Tidak --> M[Tampilkan sebab; jangan simpan dampak]
  L -- Ya --> N[Konfirmasi posting]
  N --> O[Commit jurnal dan audit atomik]
  O --> P[Detail posted dan tautan ke ledger]
```

## 5. Koreksi dan reversal

1. Pengguna membuka dokumen posted dan memilih **Buat reversal**.
2. Sistem memeriksa izin, periode terbuka, reversal sebelumnya, pembayaran aktif, serta aturan stok.
3. Pengguna memasukkan alasan; sistem menampilkan preview akun dan dampak pembalik.
4. Jika valid dan dikonfirmasi, reversal serta jurnal/alokasi/mutasi terkait disimpan atomik dengan referensi silang.
5. Sistem menampilkan hasil dan tautan dokumen asal/pengganti.

Jika ada pembayaran aktif, balik alokasi/pembayaran terlebih dahulu. Reversal stok ditolak jika item/gudang telah memiliki mutasi sesudah dokumen tersebut.

## 6. Laporan dan drill-down

`Laporan → pilih jenis → pilih periode/as-of date → lihat ringkasan/detail → buka baris → jurnal → dokumen sumber`. Filter yang sama dipakai untuk ekspor PDF/Excel. Saldo historis menghitung transaksi hingga tanggal laporan.

## 7. Impor dan kondisi umum

`Pilih jenis impor → unggah → pemeriksaan format/ukuran → preview dan error per baris → perbaiki sumber atau batalkan → validasi saldo/duplicate → konfirmasi commit → hasil batch dan audit`. Retry batch yang sama tidak boleh menggandakan data.

Semua alur memiliki loading, empty, error, validasi, dan akses-terlarang state. Kegagalan posting tidak meninggalkan data parsial; error menyebut tindakan perbaikan yang aman.
