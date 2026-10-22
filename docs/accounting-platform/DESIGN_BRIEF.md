# Design Brief — Platform Akuntansi Jasa & Dagang

**Status:** Arah desain awal untuk validasi; belum menjadi UI spec final  
**Rujukan:** [BRD](BRD.md) · [FRD](FRD.md) · [PRD](PRD.md) · [App Flow](APP_FLOW.md)

## 1. Tantangan desain

Membuat pembukuan yang kompleks terasa aman dan mudah dipahami: pengguna harus tahu apakah dokumen masih draft atau sudah memengaruhi ledger, apa dampaknya sebelum posting, dan bagaimana menelusuri atau membalik transaksi.

## 2. Pengguna dan konteks

Pengguna owner, finance, dan staf usaha jasa/dagang menggunakan aplikasi web berbahasa Indonesia. Desktop browser menjadi konteks utama; layar kecil tetap harus mendukung formulir dan tabel esensial. Merek, warna, logo, typography, dan tingkat densitas visual belum ditentukan.

## 3. Prinsip visual dan interaksi

- Hierarki jelas antara identitas dokumen, status lifecycle, nominal, dan tindakan.
- Angka uang rata kanan dan mudah dibandingkan; format IDR konsisten; gunakan angka tabular bila font mendukung.
- Debit/kredit, saldo positif/negatif, dan status draft/posted/reversed tidak dibedakan lewat warna saja; sertakan label/ikon dan teks.
- Aksi berisiko (posting, reversal, lock/reopen, commit impor) eksplisit, punya preview dampak, konfirmasi, dan hasil yang dapat dikenali.
- Gunakan tabel untuk daftar akuntansi dengan header tetap bila sesuai, pencarian/filter/pagination, subtotal jelas, dan tautan sumber.
- Jangan meniru layout, aset, nama visual, atau brand Accurate maupun Master Pajak. Identitas visual orisinal menunggu keputusan produk.

## 4. Arsitektur informasi

1. **Ringkasan:** kondisi saldo dan periode aktif; rincian KPI final ditentukan setelah validasi kebutuhan.
2. **Penjualan:** invoice, pembayaran, pelanggan, aging AR.
3. **Pembelian:** bill, pembayaran, pemasok, aging AP.
4. **Kas & Bank:** penerimaan, pengeluaran, transfer, rekening.
5. **Persediaan:** item, stok, nilai, kartu mutasi.
6. **Akuntansi:** COA, jurnal, buku besar, neraca saldo.
7. **Laporan:** laba rugi, neraca, AR/AP aging, drill-down dan ekspor.
8. **Pengaturan:** perusahaan, pengguna, periode, saldo awal, audit.

Susunan menu final menyesuaikan alur yang paling sering dipakai dan hasil uji pengguna; jangan tambahkan dashboard atau modul yang tidak memiliki keputusan bisnis.

## 5. Pola layar penting

- **Daftar dokumen:** filter periode/status/pihak, pencarian, pagination, status yang konsisten, aksi sesuai izin.
- **Form draft:** kelompok informasi pihak/tanggal, baris, pajak (tidak diimplementasikan pada MVP), ringkasan total dan catatan.
- **Preview posting:** jurnal seimbang, perubahan AR/AP, dampak kas/stok/HPP, periode dan sumber; tombol konfirmasi terpisah.
- **Detail posted:** ringkasan, lifecycle/pembayaran, jurnal, alokasi, kartu stok terkait, audit, dan dokumen sumber.
- **Laporan:** filter tanggal yang terlihat, total dan saldo per akun, status keseimbangan, drill-down, serta ekspor dengan filter sama.
- **Saldo awal/impor:** langkah pratinjau, error per baris, peringatan rekonsiliasi, dan commit terkonfirmasi.

## 6. Responsif dan aksesibilitas

- Pada layar sempit, tabel dapat bergulir horizontal atau menampilkan ringkasan baris dan detail; jangan memotong nominal/status penting.
- Form dapat digunakan keyboard, label melekat pada field, fokus terlihat, urutan fokus logis, error diumumkan dan terhubung ke field.
- Target sentuh cukup besar; dialog konfirmasi dapat diakses keyboard dan fokus kembali ke pemicu.
- Kontras teks/status memenuhi standar aksesibilitas yang dipilih tim; warna bukan satu-satunya penanda.

## 7. Nada bahasa dan state

Gunakan Bahasa Indonesia sederhana dan istilah akuntansi konsisten, dengan bantuan singkat untuk istilah seperti posting, reversal, dan saldo as-of. Error menyebut apa yang ditolak, data apa yang tetap aman, dan langkah perbaikan. Sediakan empty/loading/error/permission-denied states serta konfirmasi untuk tindakan berdampak finansial.

## 8. Hal yang perlu diputuskan sebelum UI final

Merek dan tema visual, kepadatan tabel, urutan navigasi, dashboard KPI, izin per peran, pola notifikasi, serta perangkat yang paling dominan. Validasi dengan pengguna usaha dan reviewer akuntansi sebelum mengunci komponen dan token visual.
