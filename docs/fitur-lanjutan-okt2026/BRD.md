# Business Requirements Document — Fitur lanjutan checklist tester (Okt 2026)

**Status:** Draft, 22 Okt 2026 · **Pemilik:** tim MasterPajak AI (reyfuu, Ravent) · **Sumber:** checklist tester *List Check Fitur Master Pajak Update* dan cek kode `origin/main` (d9b467c)

## 1. Latar belakang

Tester menguji MasterPajak AI dengan alur pemilik UMKM: mengisi saldo awal, mencatat
transaksi harian, menerbitkan invoice, opname stok, lalu membaca laporan. Dari 31 kendala,
16 sudah selesai. Sisanya 16 butir: 10 sudah dikerjakan sebagian, 6 belum sama sekali.

Kendala yang tersisa berdampak pada tiga hal:

1. **Kepercayaan pada angka.** Label laporan yang belum baku, selisih persediaan yang tercampur
   di HPP, dan daftar aset yang tidak bisa dilihat per tanggal membuat laporan sulit dicocokkan
   dengan pembukuan manual atau dibawa ke konsultan pajak.
2. **Beban kerja harian.** Transfer antar rekening sendiri ditolak, keterangan harus diisi per
   baris, kursor angka melompat, dan riwayat transaksi tidak bisa dilihat per pelanggan.
   Pengguna terpaksa mencari jalan memutar atau mencatat di luar aplikasi.
3. **Akses dan retensi.** Pengguna yang lupa password tidak bisa masuk tanpa menghubungi tim.

## 2. Tujuan bisnis

| Kode | Tujuan | Ukuran |
|---|---|---|
| BO-1 | Laporan keuangan bisa dipakai apa adanya oleh konsultan pajak | Tidak ada temuan label/struktur laporan dari tester pada putaran uji berikutnya |
| BO-2 | Semua uang masuk/keluar bisa dicatat di aplikasi, termasuk transfer antar rekening | 0 kasus "harus dicatat di luar aplikasi" dari tester |
| BO-3 | Saldo awal lebih jarang selisih | Saldo awal persediaan/piutang/utang selalu sama dengan daftar pendukungnya |
| BO-4 | Pengguna bisa memulihkan akses sendiri | Permintaan reset password ke tim turun ke nol setelah F-16 rilis |
| BO-5 | Checklist tester tertutup | 31/31 butir berstatus "Sudah" dan dicentang tester |

## 3. Pemangku kepentingan

| Pihak | Kepentingan |
|---|---|
| Pemilik usaha (Owner) | Pencatatan cepat, laporan benar, akses mandiri |
| Admin Keuangan / Kasir | Input harian tanpa hambatan, riwayat per pihak |
| Tim Master (konsultan) | Laporan baku untuk analisis dan SPT; nomor jurnal untuk referensi |
| Tester | Kriteria penerimaan jelas per butir |
| Developer (reyfuu, Ravent) | Lingkup kecil per PR, tidak saling bentrok |

## 4. Ruang lingkup

**Termasuk:** 16 fitur F-01 s.d. F-16 (lihat [README](README.md)).

**Tidak termasuk:**
- Butir checklist yang sudah selesai (1–12, 16–18).
- Multi-gudang, multi-mata uang, dan siklus dokumen DRAFT/POSTED/REVERSED (sudah dibahas di
  [docs/accounting-platform](../accounting-platform/BRD.md), di luar putaran ini).
- Pengiriman notifikasi WhatsApp umum (hanya dipakai bila dipilih untuk F-16, lihat §6).

## 5. Prioritas dan fase

| Fase | Fitur | Alasan |
|---|---|---|
| 1 — cepat | F-01, F-02, F-03, F-04, F-05, F-06 | Frontend/label saja, risiko rendah, langsung terasa oleh tester |
| 2 — menengah | F-07, F-08, F-09, F-10, F-11 | Backend kecil, menutup jalan memutar di input harian |
| 3 — butuh keputusan | F-12, F-13, F-14, F-15, F-16 | Mengubah alur akuntansi, data, atau menu; perlu keputusan §6 |

Setiap fitur dikirim sebagai PR sendiri. PR yang menyentuh berkas yang sama (mis.
`Reports.tsx`, `SaldoAwal.tsx`) ditumpuk berurutan supaya tidak bentrok.

## 6. Keputusan terbuka

| No | Pertanyaan | Usulan | Dampak bila ditunda |
|---|---|---|---|
| D-1 | F-16: kirim tautan reset lewat email atau kode lewat WhatsApp? | **Email** (SMTP, tautan berlaku 30 menit). Semua akun punya email; WA belum punya penyedia | F-16 tidak bisa mulai |
| D-2 | F-15: sidebar sudah punya grup "Tagihan" berisi submenu Piutang & Utang dan Invoice. Cukup itu, atau jadikan satu halaman bertab? | **Satu halaman Tagihan bertab** (Invoice, Piutang, Utang, Pelanggan); data dan izin tetap. Bila tester puas dengan grup sidebar, F-15 cukup ditandai selesai | Tester tetap melihat dua halaman terpisah |
| D-3 | F-12: nomor dan nama akun selisih persediaan | **5106 "Selisih Persediaan"** di kelompok HPP; saldo debit = beban, kredit = pendapatan | Selisih tetap tercampur di HPP |
| D-4 | F-09: format nomor jurnal | **`JU-YYYYMM-NNNN`** per usaha per bulan; transaksi lama diberi nomor sekali saat migrasi | Referensi tetap dihitung saat tampil |
| D-5 | F-04: label laba ditahan | **"Laba Ditahan"** (bukan "Laba (Rugi) Ditahan"); baris tahun berjalan tetap "Laba (Rugi) Tahun Berjalan" | — |
| D-6 | F-14: saldo awal piutang/utang wajib dirinci per pihak? | **Ya bila ada nilainya**; total di neraca awal = jumlah rincian | Selisih saldo awal tetap mungkin |

## 7. Risiko

| Risiko | Mitigasi |
|---|---|
| Migrasi nomor jurnal (F-09) mengubah tampilan referensi lama | Nomor lama (`TRX-`, `BYR-`) tetap bisa dicari; migrasi idempoten |
| Perubahan akun selisih persediaan (F-12) mengubah HPP periode lalu | Berlaku mulai opname setelah rilis; periode yang sudah tutup buku tidak dihitung ulang |
| Saldo awal dari daftar (F-14) mengubah neraca awal usaha yang sudah berjalan | Hanya untuk usaha yang belum tutup buku pertama; usaha lama diberi peringatan selisih, bukan diubah |
| Email reset (F-16) disalahgunakan untuk enumerasi akun | Respons selalu sama ("bila email terdaftar, tautan dikirim"); batas 3 permintaan per jam per email/IP |

## 8. Asumsi

- Penyimpanan PostgreSQL mengikuti pola yang ada: satu tabel per entitas dengan `(tenant_id, id)`
  dan penghitung `next_*_id` di `tenants` (lihat [BACKEND_SCHEMA.md](BACKEND_SCHEMA.md)).
- Izin per menu (`guard("<view>")`) dipertahankan; fitur baru memakai izin menu yang sudah ada
  kecuali disebut lain.
- Produksi dideploy lewat `deploy.sh` setelah PR di-merge dan `pg_dump` manual.
