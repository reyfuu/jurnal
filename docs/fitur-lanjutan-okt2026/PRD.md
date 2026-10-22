# Product Requirements Document — Fitur lanjutan checklist tester (Okt 2026)

**Status:** Draft, 22 Okt 2026 · **Input:** [BRD](BRD.md) · **Turunan:** [FRD](FRD.md), [TRD](TRD.md), [APP_FLOW](APP_FLOW.md), [BACKEND_SCHEMA](BACKEND_SCHEMA.md)

## 1. Masalah dalam satu kalimat

Pemilik UMKM sudah bisa mencatat dan membaca laporan di MasterPajak AI, tetapi masih menemui
16 hambatan kecil-menengah yang membuat mereka mencatat di luar aplikasi, ragu pada laporan,
atau tidak bisa masuk sendiri.

## 2. Persona

| Persona | Kebutuhan utama di putaran ini |
|---|---|
| **Bu Sari, Owner kedai** | Lihat transaksi bulan ini saja, transfer antar rekening, lupa password |
| **Dina, Admin Keuangan** | Input saldo awal tanpa selisih, keterangan sekali per transaksi, nomor jurnal untuk ditempel di bukti |
| **Pak Budi, konsultan Tim Master** | Label laporan baku, aset & persediaan per tanggal, selisih persediaan terpisah |

## 3. Cerita pengguna

| Kode | Sebagai | Saya ingin | Supaya |
|---|---|---|---|
| F-01 | Owner | Membuka Transaksi dan langsung melihat bulan berjalan | Tidak bingung dengan daftar panjang semua bulan |
| F-02 | Admin | Akun saldo awal yang saya tambah langsung pindah ke urutan nomornya | Neraca awal terbaca rapi seperti CoA |
| F-03 | Admin | Mengedit angka di tengah tanpa kursor melompat ke akhir | Koreksi angka cepat dan tidak salah ketik |
| F-04 | Konsultan | Label laporan baku: tanpa "(tunai)", "Laba Ditahan", pinjaman per akun | Laporan bisa langsung dipakai untuk SPT/analisis |
| F-05 | Owner | Tidak diperingatkan saat laba ditahan minus | Rugi tahun lalu dianggap wajar, bukan kesalahan input |
| F-06 | Admin | Menulis keterangan sekali per transaksi | Input jurnal manual lebih cepat |
| F-07 | Owner | Mengunduh semua daftar (piutang, utang, barang, pelanggan…) untuk rentang tanggal | Arsip triwulan/tahunan sekali klik |
| F-08 | Owner | Mencatat pindah dana Bank A → Bank B atau Kas → Bank | Saldo tiap rekening cocok dengan rekening koran |
| F-09 | Admin | Setiap transaksi punya nomor jurnal tetap (JU-202610-0001) | Bisa ditulis di bukti fisik dan dicari kembali |
| F-10 | Owner | Punya daftar pelanggan seperti daftar supplier | Data pelanggan rapi dan dipakai ulang di invoice |
| F-11 | Owner | Membuka satu pelanggan/supplier dan melihat semua transaksinya | Tahu riwayat belanja dan tagihan tiap pihak |
| F-12 | Konsultan | Selisih stok opname tercatat di akun "Selisih Persediaan" | HPP murni dari penjualan, selisih terlihat terpisah |
| F-13 | Konsultan | Melihat daftar aset & nilai persediaan per tanggal tertentu | Cocok dengan neraca per tanggal yang sama |
| F-14 | Admin | Saldo awal persediaan, piutang, utang terisi dari daftar barang & tagihan | Neraca awal tidak selisih dengan daftar pendukung |
| F-15 | Owner | Satu halaman Tagihan bertab untuk invoice, piutang, dan utang | Tidak bingung dua halaman yang terasa sama |
| F-16 | Semua | Mengatur ulang password sendiri dari halaman Masuk | Tidak perlu menghubungi tim |

## 4. Perilaku produk ringkas

- **F-01:** filter bulan Transaksi bawaan = bulan berjalan (WIB); pilihan "Semua bulan" tetap ada;
  pilihan pengguna diingat per browser selama sesi. Tautan "Lihat semua jurnal" ke Jurnal Umum.
- **F-02:** baris saldo awal diurutkan berdasarkan kode akun setiap kali kode diisi dan saat simpan.
- **F-03:** posisi kursor dipertahankan relatif terhadap digit saat titik ribuan ditambah/dihapus.
- **F-04:** lihat FRD §F-04 untuk daftar label persis.
- **F-05:** catatan "Ada saldo minus" di dialog Saldo Awal tidak menghitung akun berjenis laba ditahan;
  akun 3299 "Selisih Saldo Awal" tetap diperingatkan bila ≠ 0.
- **F-06:** jurnal manual hanya punya kolom Keterangan di kepala transaksi; kolom per baris dihapus.
- **F-07:** kontrol Dari–Sampai di samping tombol "Unduh semua daftar", bawaan = periode laporan.
- **F-08:** jenis transaksi baru **Pindah Dana** (dari akun kas/bank ke akun kas/bank lain).
- **F-09:** nomor `JU-YYYYMM-NNNN` diberikan saat transaksi disimpan, tidak berubah saat diedit,
  tidak dipakai ulang saat dihapus.
- **F-10:** menu Daftar Pelanggan (tab di Invoice/Tagihan) dengan CRUD dan impor dari nama pelanggan yang sudah ada.
- **F-11:** halaman detail pihak: ringkasan (total transaksi, piutang/utang terbuka) + tabel riwayat.
- **F-12:** opname dengan selisih langsung membentuk jurnal Selisih Persediaan pada tanggal opname.
- **F-13:** tab Aset & Penyusutan dan Stok & Gudang punya pemilih "Per tanggal".
- **F-14:** baris Persediaan, Piutang Usaha, Utang Usaha di Saldo Awal terkunci dan menjumlah rinciannya;
  tombol "Rinci" membuka daftar barang / daftar piutang-utang awal per pihak.
- **F-15:** grup sidebar Tagihan (sudah ada) menjadi satu halaman dengan tab Invoice · Piutang · Utang · Pelanggan.
- **F-16:** tautan "Lupa password?" di halaman Masuk → email berisi tautan reset (30 menit).

## 5. Ukuran keberhasilan

| Metrik | Target |
|---|---|
| Butir checklist berstatus Sudah dan dicentang tester | 31/31 |
| Transaksi yang ditolak dengan pesan "perpindahan antar kas/bank" | 0 setelah F-08 |
| Selisih saldo awal yang disimpan paksa ke 3299 pada usaha baru | turun ≥ 50% setelah F-14 |
| Permintaan reset password ke Tim Master | 0 setelah F-16 |

## 6. Rilis

| Rilis | Isi | Syarat rilis |
|---|---|---|
| R1 | Fase 1 (F-01…F-06) | Vitest + build hijau, uji tester pada produksi |
| R2 | Fase 2 (F-07…F-11) | Go test + migrasi skema idempoten, pg_dump sebelum deploy |
| R3 | Fase 3 (F-12…F-16) | Keputusan D-1…D-6 di BRD, uji akuntansi dengan data contoh Kedai Kopi Senja |

## 7. Di luar lingkup

Kirim invoice lewat email/WA, multi-gudang, persetujuan berjenjang, dan pengingat jatuh tempo otomatis.
