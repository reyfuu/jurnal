# Handoff — Jurnal AI

Status per 13 Oktober 2026. Branch: `claude/quirky-cori-kbvckj`
(139 berkas, ~21.800 baris, 134 test hijau).

---

## 1. Apa ini

Aplikasi Laravel **berdiri sendiri** di dalam folder `jurnal-ai/`. Punya
`composer.json`, `.env`, migration, dan test sendiri — siap dipisah jadi repo
tersendiri kapan saja (`git subtree split`).

Alurnya: **upload → AI baca → review manusia → posting → laporan**.
Sumber dokumen: foto struk, PDF invoice, screenshot mutasi bank, teks tempel.

Satu aturan yang tidak boleh dilanggar: **AI tidak pernah memposting sendiri.**
Hasil bacanya selalu mendarat di layar review untuk dikoreksi. Yang masuk buku
adalah angka yang sudah dilihat manusia.

## 2. Hubungannya dengan portal B2B SKINKU — TIDAK ADA

Ini pertanyaan yang paling sering muncul, jadi ditulis tegas:

- **Nol referensi silang.** Tidak menyentuh `acc_journals` / `acc_accounts`
  milik portal B2B.
- **Database terpisah.** Default SQLite satu berkas; produksi MySQL sendiri.
- **Klien "SKINKU" di dalam aplikasi hanyalah baris di tabel `clients`** —
  bukan tarikan data dari portal. Semua jurnalnya dari `DemoSeeder` (karangan).

Foldernya menumpang di repo `skinku-b2b-portal` semata supaya bisa di-commit
dan dibuka dari app Claude, bukan karena ada ketergantungan.

**Keputusan terbuka yang belum dieksekusi:** portal B2B sudah punya modul
akuntansi lengkap (COA, jurnal, impor mutasi, Laba Rugi, Neraca, Arus Kas)
yang tersambung ke data operasional asli. Untuk SKINKU sendiri, Jurnal AI
adalah duplikat — dua buku besar untuk satu bisnis itu resep angka beda.
Rekomendasi yang sudah disampaikan dan belum diputuskan:

- **A.** Jurnal AI khusus klien eksternal; SKINKU tetap dibukukan di portal.
- **B.** Port `DocumentExtractor` + `JournalDrafter` + layar review ke dalam
  modul akuntansi portal, supaya SKINKU dapat "foto struk → jurnal" tanpa
  memecah buku.

## 3. Keputusan desain yang menahan beban

Jangan dibongkar tanpa alasan kuat — tiap poin punya test yang menjaganya.

- **Klien = isolasi penuh, bukan filter.** COA, jurnal, laporan terpisah per
  klien. `JournalPoster` menolak jurnal yang memakai akun milik klien lain;
  tanpa itu laporan dua klien tercampur diam-diam.
- **Semua angka dari AI dicuci ulang di PHP.** `Rupiah::parse()` menormalkan
  "Rp 1.250.000", "1,250,000", "85rb", "1,5jt", "(50.000)". Tanggal `03/04/2026`
  dibaca 3 April (gaya Indonesia); tanggal ngawur (1970, 2090) ditolak.
- **Kode akun karangan tidak pernah lolos.** Kode di luar COA klien dialihkan
  ke akun default + warning. Jurnal tidak boleh gagal hanya karena model salah
  mengarang nomor.
- **Draft dijamin balance.** Sisi pembayaran dihitung sebagai sisa. Tapi
  ketidakcocokan dengan total dokumen **dilaporkan sebagai warning**, bukan
  ditambal diam-diam.
- **Anti-dobel dua lapis.** `documents.content_hash` menandai dokumen identik;
  `journals.fingerprint` menahan posting ganda walau struk difoto ulang.
- **Posting gagal sebagian = batalkan semua.** Setengah mutasi bank di buku
  lebih berbahaya daripada nol.
- **Jangan hapus, nonaktifkan.** Klien & akun yang sudah punya jurnal posted
  tidak bisa dihapus. Jurnal salah di-*void*, bukan dihapus.
- **Pagar kredensial.** Seeder menolak jalan di produksi tanpa
  `SEED_ADMIN_PASSWORD` yang kuat; `jurnal:cek-publik` memblokir tunnel selama
  APP_DEBUG menyala atau masih ada sandi bawaan. Aturan sandi tunggal ada di
  `app/Support/SandiLemah.php` (cek **kandungan** kata, bukan kesamaan persis —
  "password1234" panjangnya cukup tapi sama buruknya).

## 4. Status tiap bagian

| Bagian | Status |
|---|---|
| Engine double-entry + COA 50 akun per klien | Selesai |
| Baca dokumen (gambar, PDF, teks) + normalisasi | Selesai |
| Layar review + posting + tombol balik debit/kredit | Selesai |
| Jurnal manual, void, hak akses admin/staff | Selesai |
| Neraca Saldo, Laba Rugi, Neraca | Selesai |
| Provider AI (OpenAI / Anthropic / OpenAI-compatible + cadangan) | Selesai |
| Peluncur Windows (JALANKAN / RESET-DATA / AKTIFKAN-SQLITE / TUNNEL / SIAPKAN-PUBLIK) | Selesai |
| **Laporan Arus Kas** | **Belum ada** |
| Jurnal penutup akhir periode | Belum ada (laba berjalan langsung di ekuitas) |
| Ekspor PDF / Excel laporan | Belum ada |
| Pembacaan asinkron (queue) | Belum ada — upload menunggu AI 5–20 detik |
| Multi-user per klien | Belum ada — semua user melihat semua klien |

## 5. Di mana Freddie berhenti

**Testing lokal di Windows, berhasil.** Aplikasi jalan di `127.0.0.1:8001`,
AI tersambung ke **9router** (model `cx/gpt-5.6-so1`), sudah terbukti membaca
bukti transfer m-Banking BCA dan menyusun jurnalnya.

**Deploy ke `jurnal.skinku.id` DIBATALKAN.** Sudah sampai tahap clone +
composer install + `.env` di server, lalu dihentikan karena friksi mengedit
`.env` lewat nano di SSH. Database `u864765086_jurnal` dan folder
`~/jurnal-src` direncanakan dihapus. `DEPLOY-JURNAL.md` tetap disimpan dan
masih berlaku kalau suatu saat dilanjutkan.

**Gantinya: tunnel.** PC Freddie jadi server, diekspos lewat cloudflared
(`TUNNEL.bat`). Per akhir sesi, `SIAPKAN-PUBLIK.bat` baru dikirim dan
**belum dikonfirmasi berhasil dijalankan**.

**Checklist uji manual di `TESTING-LOKAL.md` bagian 4 belum dituntaskan.**
Yang paling berguna dan belum diuji: screenshot mutasi banyak baris, nota
tulis tangan, dan posting dobel.

## 6. Lingkungan Freddie — penting untuk sesi berikutnya

- **Windows**, PHP 8.3 dari winget. `pdo_sqlite` awalnya mati, sudah
  diaktifkan lewat `AKTIFKAN-SQLITE.bat` (permanen di `php.ini`).
- **PowerShell 5.1** — `&&` tidak didukung, pakai `;` atau satu baris per Enter.
- **Paste multi-baris ke terminal sering menyatu** dan jadi satu perintah
  rusak. Ini sudah tiga kali memakan waktu. Berikan **satu perintah per blok**,
  atau lebih baik: bungkus jadi `.bat` yang tinggal diklik.
- Preferensinya jelas: **satu klik > perintah terminal**. Setiap kali solusi
  diberikan sebagai rangkaian perintah, sesinya tersendat.
- Claude di sesi ini jalan di container cloud — **tidak bisa menyentuh folder
  di PC Freddie**. Setiap perbaikan dikirim sebagai ZIP. Kalau ini mengganggu,
  jalurnya: `claude remote-control` dari dalam folder `Jurnal-AI`.

## 7. Langkah berikutnya, berurut

1. **Pastikan `SIAPKAN-PUBLIK.bat` + `TUNNEL.bat` berhasil** — ini titik
   berhenti terakhir dan belum dikonfirmasi.
2. **Tuntaskan checklist uji manual** (`TESTING-LOKAL.md` §4). Fokus ke kasus
   yang menguji kejujuran AI: nota buram harus memunculkan warning, bukan
   angka karangan.
3. **Laporan Arus Kas** — sudah ditanyakan Freddie, datanya lengkap, tinggal
   mengelompokkan mutasi akun kas/bank jadi aktivitas operasi, investasi,
   pendanaan.
4. **Queue untuk pembacaan AI** — mulai terasa di atas ~50 dokumen/hari.
5. **Ekspor laporan PDF/Excel** — syarat praktis kalau dijual sebagai jasa
   pembukuan; klien pasti minta laporan yang bisa dikirim.
6. **Putuskan A atau B** dari bagian 2.

## 8. Berkas yang perlu dibaca lebih dulu

| Berkas | Isi |
|---|---|
| `README.md` | Gambaran sistem, cara jalan, keputusan desain |
| `TESTING-LOKAL.md` | Setup Windows/Mac + checklist uji manual per area |
| `DEPLOY-JURNAL.md` | Deploy ke Hostinger (belum dieksekusi, masih berlaku) |
| `CARA-PAKAI.txt` | Petunjuk satu halaman untuk pengguna akhir |
| `app/Services/Intake/DocumentExtractor.php` | Prompt + normalisasi hasil AI — inti produk |
| `app/Services/Accounting/JournalPoster.php` | Satu-satunya pintu pembuatan jurnal |

## 9. Yang tidak boleh dilakukan

- `migrate:fresh` / `migrate:refresh` di mana pun yang ada data nyata.
- `db:seed --class=DemoSeeder` di luar lokal — menyuntik 18 jurnal karangan.
- Mengarahkan `DB_DATABASE` ke database portal B2B.
- `APP_DEBUG=true` saat aplikasi bisa diakses dari internet.
- Menghapus validasi balance, pagar isolasi klien, atau anti-dobel demi
  diff yang lebih pendek. Semuanya punya test; kalau test merah, itu bukan
  test yang salah.
