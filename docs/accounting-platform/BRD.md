# Business Requirements Document — Platform Akuntansi Jasa & Dagang

**Status:** Draft baseline untuk validasi bisnis dan akuntansi  
**Sumber utama:** [Handoff Coding Agent](../Handoff_Accounting_Platform_Coding_Agent%20%281%29.md)  
**Batas penting:** Produk ini belum dipastikan terkait dengan Master Pajak. Accurate hanya referensi perilaku yang boleh diamati secara sah; dokumen ini tidak mengklaim parity atau kompatibilitas.

## 1. Ringkasan

Platform web ini membantu usaha jasa dan dagang mencatat transaksi satu kali dan memperoleh saldo kas, piutang, utang, persediaan, serta laporan keuangan yang konsisten dan dapat ditelusuri ke dokumen sumber. MVP memprioritaskan ketepatan pembukuan dan alur posting yang dapat diaudit daripada banyaknya modul.

## 2. Masalah bisnis

Usaha membutuhkan catatan yang menghubungkan transaksi operasional dengan jurnal, saldo subledger, stok, dan laporan. Pencatatan yang terpisah menyulitkan rekonsiliasi, pelacakan piutang/utang, serta penjelasan asal suatu angka. Pengguna juga berisiko mengubah transaksi posted tanpa jejak koreksi yang dapat dipercaya.

## 3. Tujuan dan hasil

- Catat penjualan, pembelian, kas/bank, jurnal, dan persediaan melalui workflow yang saling terhubung.
- Jaga double-entry, saldo subledger, nilai stok, dan laporan tetap seimbang serta dapat direkonsiliasi.
- Lacak perubahan, posting, reversal, lock periode, dan impor dengan identitas pengguna, waktu, serta alasan yang sesuai.
- Sediakan laporan yang dapat ditelusuri kembali ke jurnal dan dokumen sumber.
- Buat UI berbahasa Indonesia yang usable di desktop dan tetap berfungsi pada layar kecil.

## 4. Pengguna dan pemangku kepentingan

| Peran | Kebutuhan utama |
|---|---|
| Owner | Mengatur perusahaan, pengguna, periode, membuka kembali periode dengan alasan, dan melihat laporan. |
| Finance | Mengelola master dan saldo awal, membuat serta mem-posting transaksi, melihat laporan dan audit. |
| Staf | Mencatat draft transaksi sesuai hak akses dan mengikuti proses posting yang diizinkan. |
| Penanggung jawab akuntansi | Memvalidasi kebijakan akun, costing, rounding, saldo awal, dan hasil laporan sebelum pilot/produksi. |

Hak akses rinci harus dikunci sebelum implementasi. API/server menjadi otoritas izin; penyembunyian tombol UI bukan pengamanan.

## 5. Ruang lingkup

### MVP

- Satu perusahaan aktif per sesi, IDR, tahun buku dan periode fiskal.
- Owner, finance, dan staf dengan otorisasi di server.
- COA, mapping akun posting, pelanggan/pemasok, barang/jasa, satuan, satu gudang.
- Saldo awal GL, AR/AP per pihak, dan stok yang dapat direkonsiliasi.
- Kas/bank: setoran modal, penerimaan, pengeluaran, dan transfer.
- Invoice penjualan dan bill pembelian dengan pembayaran parsial/penuh, AR/AP, serta jurnal otomatis.
- Jurnal manual, draft/posting/reversal, audit, dan lock periode.
- Persediaan perpetual dengan moving weighted average; tanpa stok negatif dan tanpa posting tanggal mundur.
- Buku besar, neraca saldo, laba rugi, neraca, umur piutang dan utang.
- Impor saldo awal/master dengan pratinjau dan ekspor PDF/Excel.

### Di luar MVP

Multi-currency/konsolidasi, multi-gudang, manufaktur, varian/serial/batch, quotation/SO/delivery independen, PO/goods receipt independen, retur parsial, rekonsiliasi bank, aset tetap, pajak, marketplace, payroll, AI, bank feed, dan migrasi otomatis dari Accurate.

## 6. Ukuran keberhasilan MVP

- Fixture penerimaan utama pada [FRD](FRD.md) menghasilkan angka yang ditetapkan dan semua jurnal seimbang.
- Setiap laporan dan saldo subledger pada fixture cocok dengan GL dan dapat ditelusuri ke sumber.
- Pengujian concurrency, idempotency, tenant isolation, periode terkunci, reversal, dan impor menolak kondisi tidak valid tanpa data parsial.
- Penanggung jawab akuntansi meninjau keputusan baseline dan menyetujui hasil pilot terbatas sebelum penggunaan produksi.

Target jumlah pengguna, SLA, performa, biaya, hosting, dan tenggat belum ditentukan; jangan mengarang angka bisnis untuknya.

## 7. Asumsi dan keputusan terbuka

Asumsi kerja yang memerlukan validasi: basis akrual, double-entry, IDR, perpetual moving weighted average, satu gudang, dan titik posting invoice/bill sekaligus menjadi titik mutasi stok. Stack, nama produk, repository, hosting, kapasitas, integrasi, edisi Accurate referensi, dan relasi dengan Master Pajak masih terbuka. Lihat [TRD](TRD.md) untuk konsekuensi teknis.

## 8. Batas etika dan otorisasi

Implementasi harus memakai kode, identitas, dan desain sendiri. Observasi Accurate hanya dengan akses sah dan data dummy; jangan menyalin source/binary atau melewati lisensi. Handoff ini tidak mengotorisasi akses akun pihak lain, deployment produksi, atau migrasi destruktif.
