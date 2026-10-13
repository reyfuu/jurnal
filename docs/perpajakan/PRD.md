# PRD — Perpajakan Jurnal AI

Status: draft 13 Okt 2026 · Sumber kebutuhan: [BRD](BRD.md) · Detail perilaku: [FRD](FRD.md)

Prinsip produk yang diwarisi (tidak bisa ditawar):
1. **AI tidak pernah memposting, menyetor, atau melapor.** Semua pajak mendarat di layar review.
2. **Angka pajak dihitung ulang di PHP** dari DPP × tarif berlaku; selisih = warning di baris, bukan koreksi diam-diam.
3. **Klien = isolasi penuh.** Profil, rekap, kalender, ekspor per klien.
4. **Tarif = data**, ber-tanggal berlaku, bukan konstanta kode.

## 1. Persona

| ID | Persona | Konteks | Butuh |
|---|---|---|---|
| PS-1 | **Rina, bookkeeper jasa pembukuan** (role `staff`) | Pegang 8–15 klien UMKM, upload 30–80 dokumen/hari. | Pajak terpecah otomatis, tahu kapan harus setor apa untuk tiap klien. |
| PS-2 | **Freddie, pemilik jasa / admin** (role `admin`) | Atur klien, COA, tarif; jual paket pembukuan + rekap pajak. | Profil pajak benar, tarif bisa diperbarui tanpa developer, jejak audit. |
| PS-3 | **Bu Sari, pemilik UMKM** (bukan user aplikasi di MVP) | Non-PKP, PPh Final 0,5%. | Menerima rekap & angka setoran bulanan yang bisa dipercaya. |
| PS-4 | **Konsultan pajak mitra** (bukan user di MVP) | Terima rekap dari bookkeeper. | Ekspor rapi, ada nomor faktur/bukpot & tautan dokumen sumber. |

## 2. User story (MoSCoW)

| ID | Sebagai | Saya ingin | Supaya | Prioritas | BR |
|---|---|---|---|---|---|
| US-01 | admin | mengisi profil pajak klien (PKP/non-PKP, NPWP/NIK, rezim PPh, tanggal PKP) | aturan pajak yang dipakai sistem sesuai klien | **Must** | BR-02 |
| US-02 | admin | melihat & menambah versi tarif dengan tanggal berlaku | perubahan regulasi tidak butuh deploy | **Must** | BR-07 |
| US-03 | staff | melihat pajak di invoice terpecah per jenis (PPN, PPh 23, dll.) dengan DPP, tarif, dan nilai | tidak memecah manual | **Must** | BR-01 |
| US-04 | staff | melihat tanda jika pajak AI ≠ DPP × tarif berlaku | salah baca ketahuan sebelum posting | **Must** | BR-03 |
| US-05 | staff | draft jurnal otomatis memakai akun pajak yang benar (PPN Masukan/Keluaran, Utang PPh xx, PPh Dibayar Dimuka) | buku & rekap konsisten | **Must** | BR-01 |
| US-06 | staff | PPN pada klien non-PKP otomatis masuk ke biaya, bukan PPN Masukan | Neraca tidak berisi aset pajak fiktif | **Must** | BR-02 |
| US-07 | staff | menandai PPh sebagai "dipotong" (kita pemotong) atau "dipotong pihak lain" | arah jurnal & pembayaran benar | **Must** | BR-01 |
| US-08 | staff | melihat rekap PPN masa (Keluaran − Masukan = kurang/lebih bayar) | tahu angka setor PPN | **Must** | BR-04 |
| US-09 | staff | melihat rekap PPh potput per jenis per masa beserta daftar transaksinya | tahu angka setor & bukpot yang harus dibuat | **Must** | BR-04 |
| US-10 | staff | melihat rekap PPh Final UMKM dari omzet bulanan (termasuk sisa Rp500 jt bebas untuk OP) | tahu angka setoran 0,5% | **Must** | BR-04 |
| US-11 | staff | melihat kalender jatuh tempo setor/lapor per klien & menandai "sudah setor (NTPN)" / "sudah lapor" | tidak ada yang telat | **Must** | BR-05 |
| US-12 | staff | mengekspor rekap ke CSV | dibawa ke Coretax / konsultan | **Must** | BR-06 |
| US-13 | admin | melihat jejak audit perubahan profil, tarif, status setor/lapor | bisa ditelusuri saat ada sengketa | **Must** | BR-08 |
| US-14 | staff | membuat draft jurnal setoran/akrual dari rekap (PPh Final, PPN kurang bayar) yang lalu saya posting sendiri | tidak ketik ulang | Should | BR-04 |
| US-15 | staff | melihat kalender pajak semua klien di dashboard | prioritas harian jelas | Should | BR-05 |
| US-16 | staff | ekspor dalam format Excel template impor Coretax | upload langsung ke Coretax | Should (Fase 2, setelah verifikasi V22) | BR-06 |
| US-17 | staff | menerima pengingat (email/WA) H-3 jatuh tempo | tidak bergantung buka aplikasi | Could | BR-05 |
| US-18 | staff | mencatat PPh 21 dari input gaji manual (nilai PPh 21 diisi manusia) | utang PPh 21 tercatat | Could | BR-01 |
| US-19 | staff | sistem menghitung PPh 21 TER otomatis dari data karyawan | — | **Won't (MVP)** | — |
| US-20 | staff | submit SPT/faktur langsung ke Coretax | — | **Won't (MVP)** | — |

## 3. Alur utama

### F1 — Invoice pembelian ber-PPN, klien PKP (US-03/04/05)
1. Rina upload PDF invoice supplier Rp1.000.000 + PPN Rp110.000.
2. AI membaca → `taxes: [{type: ppn, dpp: 916.667, rate: 12, amount: 110.000}]`.
3. PHP menghitung ulang dengan tarif berlaku di tanggal invoice: 1.000.000 × 11/12 × 12% = 110.000 → **cocok** (badge netral).
4. Layar review, panel **Pajak**: Jenis · DPP · Tarif · Pajak (AI) · Pajak (hitung) · Status · No. Faktur.
5. Draft: `D 6xxx Beban 1.000.000 · D 1402 PPN Masukan 110.000 · K 2101/1102 1.110.000`.
6. Rina posting → jurnal + catatan pajak tersimpan dalam satu transaksi.

### F1b — Invoice yang sama, klien non-PKP (US-06)
Draft: `D 6xxx Beban 1.110.000 · K Bank 1.110.000`. Info di panel: "Klien non-PKP — PPN digabung ke biaya." Catatan pajak tetap disimpan (jenis `ppn_noncreditable`) untuk audit, tidak masuk rekap PPN.

### F2 — Bayar jasa, klien memotong PPh 23 (US-07)
Invoice jasa Rp5.000.000, vendor ber-NPWP. Draft: `D 6109 Beban Jasa Profesional 5.000.000 · K Utang PPh 23 100.000 · K Bank 4.900.000`. Jika vendor tanpa NPWP → tarif tanpa-NPWP dari tabel + warning "Bukti potong wajib dibuat di Coretax".

### F3 — Klien menerima pembayaran yang dipotong PPh 23 oleh pelanggan (US-07)
`D Bank 4.900.000 · D PPh 23 Dibayar Dimuka 100.000 · K 4102 Pendapatan Jasa 5.000.000`. Warning: "Minta bukti potong ke pelanggan."

### F4 — Penjualan klien PKP
`D Piutang/Bank 1.110.000 · K 4101 Penjualan 1.000.000 · K PPN Keluaran 110.000`.

### F5 — Rekap PPh Final UMKM bulanan (US-10, US-14)
1. Menu Pajak › PPh Final UMKM › pilih masa.
2. Sistem: omzet bruto = kredit akun pendapatan usaha posted di masa itu (tanpa Pendapatan Lain-lain; akun yang dihitung bisa diatur per klien).
3. OP: kurangi sisa batas bebas tahun berjalan (kumulatif sejak Januari).
4. Tampil: Omzet · Bagian bebas · DKP · Tarif · PPh Final · Jatuh tempo · Status.
5. Tombol "Buat draft jurnal" → form jurnal manual terisi (`D 7103 Beban Pajak · K Utang PPh Final`), Rina sendiri yang posting.

### F6 — Rekap PPN masa (US-08)
PPN Keluaran − PPN Masukan (yang dikreditkan di masa itu) = kurang bayar / lebih bayar. Daftar faktur di bawahnya dengan tautan ke dokumen.

### F7 — Kalender & status (US-11, US-15)
Daftar kewajiban per masa (diturunkan dari profil klien + aturan jatuh tempo di tabel): `PPh Final UMKM Sep 2026 · setor 15 Okt · Belum`. Rina klik → isi NTPN, tanggal setor, jumlah → status "Sudah setor". Lapor → nomor BPE → "Sudah lapor". Merah hanya untuk **lewat jatuh tempo & belum** (sesuai DESIGN-BRIEF: merah = salah/gagal).

## 4. Catatan UX (selaras [DESIGN-BRIEF](../DESIGN-BRIEF.md))

- **Navigasi:** menu **Pajak** di dalam konteks klien, setelah Laporan: `Pajak ▸ Ringkasan · PPN · PPh Potput · PPh Final UMKM · Kalender · Profil Pajak`. Selama sidebar belum dibangun, tambahkan tab "Pajak" di `partials/client-nav.blade.php`.
- Item menu yang tidak relevan disembunyikan, bukan dinonaktifkan: non-PKP tidak melihat "PPN"; rezim normal tidak melihat "PPh Final UMKM".
- **Panel Pajak di layar review** memakai `ai-marker` amber untuk angka AI; setelah diedit jadi `dikoreksi`. Status cocok/selisih menempel di baris, bukan kotak kuning panjang.
- Angka rupiah mono, rata kanan, `Rupiah::format()`; tarif ditampilkan `12% × 11/12` agar DPP nilai lain terlihat.
- Disclaimer satu baris tetap di footer setiap halaman Pajak dan di baris pertama ekspor.
- Badge rezim di header klien: `PKP` / `Non-PKP` · `Final 0,5%` / `Normal`.
- Laporan rekap bergaya dokumen cetak (lebar ~880px, `@media print`), konsisten dengan P4 DESIGN-BRIEF.
- Tanpa React/Vue; Blade + Tailwind + vanilla JS.

## 5. Fase rilis

| Fase | Isi | Story |
|---|---|---|
| **MVP (Fase 1)** | Profil pajak · master tarif ber-tanggal · ekstraksi `taxes[]` + hitung ulang · draft ke akun pajak benar + non-PKP · PPh dipotong/dipotong pihak lain · rekap PPN, PPh potput, Final UMKM · kalender per klien + status manual · ekspor CSV · audit trail | US-01..13 |
| Fase 2 | Draft jurnal dari rekap · kalender lintas klien di dashboard · template Excel Coretax (setelah V22 terverifikasi) · PPh 21 input manual | US-14..16, US-18 |
| Fase 3 | Pengingat email/WA · register bukti potong · evaluasi integrasi via PJAP resmi | US-17 |

## 6. Non-goals

- Submit/impor otomatis ke Coretax DJP, pembuatan faktur pajak atau bukti potong resmi.
- Perhitungan PPh 21 TER, SPT Tahunan, PPh Badan, rekonsiliasi fiskal, PPnBM, kepabeanan.
- Memberi rekomendasi pajak ("sebaiknya pakai rezim X").
- AI menentukan status PKP/rezim klien — itu input manusia.
- Integrasi apa pun dengan portal B2B SKINKU (keputusan A/B belum diambil).

## 7. Metrik keberhasilan produk

| Metrik | Target MVP |
|---|---|
| Invoice ber-PPN masuk akun benar tanpa koreksi akun | ≥ 95% (uji manual 40 dokumen) |
| Waktu menyusun rekap masa per klien | < 2 menit (dari ±30 menit Excel) |
| Rekap = saldo akun pajak di Neraca Saldo | Selisih Rp0, otomatis dicek test |
| Kewajiban lewat jatuh tempo tanpa status | 0 di klien pilot, 3 bulan |
