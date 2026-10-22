# Fitur lanjutan dari checklist tester (Okt 2026)

Dokumen untuk 16 butir checklist tester (*List Check Fitur Master Pajak Update*, 22 Okt 2026)
yang hasil cek kodenya **sebagian** atau **belum**. Butir yang sudah selesai tidak dibahas di sini.

| Dokumen | Isi |
|---|---|
| [BRD.md](BRD.md) | Masalah bisnis, tujuan, ruang lingkup, prioritas, keputusan terbuka |
| [PRD.md](PRD.md) | Cerita pengguna, perilaku produk, ukuran keberhasilan, rilis bertahap |
| [FRD.md](FRD.md) | Persyaratan fungsional per fitur dan kriteria penerimaan |
| [TRD.md](TRD.md) | Desain teknis: berkas yang diubah, API, aturan akuntansi, tes |
| [APP_FLOW.md](APP_FLOW.md) | Alur layar per fitur |
| [BACKEND_SCHEMA.md](BACKEND_SCHEMA.md) | Perubahan skema PostgreSQL, struct Go, dan endpoint |

## Daftar fitur

Kode `F-xx` dipakai di semua dokumen. Kolom "No" = nomor baris di Excel checklist.

| Kode | No | Fitur | Status 22 Okt | Fase |
|---|---|---|---|---|
| F-01 | 26 | Transaksi menampilkan bulan berjalan secara bawaan | Belum | 1 |
| F-02 | 29 | Akun saldo awal terurut sesuai nomor akun | Belum | 1 |
| F-03 | 28 | Kursor tidak melompat saat mengedit angka | Belum | 1 |
| F-04 | 19 | Sisa perbaikan label laporan keuangan | Sebagian | 1 |
| F-05 | 23 | Saldo laba ditahan minus tidak diperingatkan di Saldo Awal | Sebagian | 1 |
| F-06 | 27 | Satu keterangan per transaksi di jurnal manual | Sebagian | 1 |
| F-07 | 14 | Unduh semua daftar per rentang tanggal | Sebagian | 2 |
| F-08 | 20 | Transfer antar rekening sendiri | Sebagian | 2 |
| F-09 | 31 | Nomor jurnal berurutan per transaksi | Sebagian | 2 |
| F-10 | 24 | Daftar pelanggan | Sebagian | 2 |
| F-11 | 25 | Riwayat transaksi per pelanggan dan pemasok | Belum | 2 |
| F-12 | 21 | Selisih persediaan ke akun khusus saat opname/tutup buku | Sebagian | 3 |
| F-13 | 22 | Daftar aset dan persediaan per tanggal | Belum | 3 |
| F-14 | 30 | Saldo awal persediaan, piutang, utang dari daftarnya | Sebagian | 3 |
| F-15 | 13 | Satu halaman Tagihan bertab (Invoice, Piutang, Utang) | Sebagian | 3 |
| F-16 | 15 | Lupa password | Belum | 3 |

Fase 1 = perbaikan kecil di frontend/label (bisa satu PR per butir, hari yang sama).
Fase 2 = fitur menengah dengan sedikit perubahan backend. Fase 3 = perubahan alur akuntansi
atau data yang perlu keputusan produk dulu (lihat BRD §6).

## Keputusan & status (22 Okt 2026)

- **Diterima:** D-1 (lupa password lewat email SMTP), D-2 (halaman Tagihan bertab), D-4 (`JU-YYYYMM-NNNN`).
- **Ditunda menunggu tim pajak:** D-3 (F-12 selisih persediaan), D-5 (label "Laba Ditahan"), D-6 (F-14 saldo
  awal per pihak). Penjelasan untuk tim pajak: [Keputusan_Pajak_D3_D5_D6.pptx](Keputusan_Pajak_D3_D5_D6.pptx).
- **Implementasi:** Fase 1 #125, Fase 2 #126 (di atas #125), Fase 3 menyusul (di atas #126).

Perbedaan implementasi dari rancangan (lebih sederhana, perilaku sama untuk pengguna):

| Fitur | Rancangan | Implementasi |
|---|---|---|
| F-03 | Input angka boleh negatif | Tidak: butir 28 hanya soal kursor; minus tetap lewat impor Excel |
| F-04 | Pinjaman satu baris per akun | Tidak dipecah: label sudah nama akun CoA ("Utang Bank"), saldo awal tidak membuat akun pinjaman tambahan |
| F-08 | Kolom `akun_tujuan` | Tanpa kolom baru: `lawan` = rekening asal, `akun` = rekening tujuan; hanya lewat Catat |
| F-10 | `invoices.pelanggan_id`, nonaktifkan pelanggan, impor Excel | Nama sebagai teks (dicocokkan tanpa beda huruf besar/kecil), hapus biasa, "Ambil dari data lama" |
| F-11 | `transactions.pihak_id` | Dicocokkan lewat nama pihak |
| F-16 | Tabel `password_resets` | Hash token di memori (30 menit); server dimulai ulang = minta tautan baru |
