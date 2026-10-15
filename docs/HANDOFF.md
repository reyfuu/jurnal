# Handoff — Jurnal AI

> Diperbarui: 15 Okt 2026 · Untuk: agent berikutnya (Codex / lainnya) · Repo: `github.com/reyfuu/jurnal` (private, branch `main`)
> Ringkas: app sudah live di **<https://pajak.aipreneur.co.id>**. 194 test hijau. Fitur utama selesai; yang tersisa
> kebanyakan blocker dari luar (key AI, keputusan akuntan) plus beberapa pekerjaan operasional.

Baca berurutan: **`AGENTS.md`** (aturan, keputusan desain yang dijaga test, larangan) → dokumen ini → `docs/DESIGN-BRIEF.md`
kalau menyentuh UI → `docs/perpajakan/` kalau menyentuh pajak.

---

## 1. Tujuan

Aplikasi pembukuan multi-klien untuk jasa pembukuan/konsultan pajak: **upload bukti → AI baca → manusia review → posting → laporan
& rekap pajak.** Aturan nomor satu: **AI tidak pernah memposting sendiri.** Pemakai: admin, supervisor, akuntan, anggota
(staf/klien yang upload). Sedang diuji akuntan sungguhan di server (perusahaan "Tes MP-Dagang" & "Tes MP-Manufaktur").

Tidak ada hubungan apa pun dengan portal B2B SKINKU (asal repo diekstrak). Klien "SKINKU" di DB lokal hanyalah data contoh.

## 2. Keadaan sekarang

**Jalan & teruji (ada test):**

- Engine double-entry, COA per klien (3 template), jurnal manual, void, anti-dobel, review dokumen + posting.
- Baca dokumen: gambar, PDF (teks diekstrak lokal dulu — PHP murni `PdfTextExtractor`, lalu `pdftotext` bila ada — baru dikirim utuh ke AI kalau gagal), teks tempel, mutasi bank multi-baris.
- Laporan: Neraca Saldo, Laba Rugi, Neraca, Arus Kas (metode langsung); ekspor CSV + cetak/PDF.
- Perpajakan MVP + Fase 2: profil pajak klien, tarif berversi, rekap PPN/PPh/Final UMKM, kalender jatuh tempo, draft jurnal setor.
- Peran & akses per klien dengan batas: supervisor 3 · akuntan 5 · anggota 10 perusahaan (`User::CLIENT_LIMITS`). Admin semua.
- AI **Gemini** (endpoint OpenAI-compatible, `OpenAiProvider` dipakai ulang). Key diisi admin di **/pengaturan** (terenkripsi di
  tabel `settings`). Model default alias `gemini-flash-latest`.
- Bubble **"Tanya pajak"** (kanan bawah, halaman klien) — `TaxChatController`, konteks hanya klien itu, read-only.
- COA: CRUD lewat modal, **impor/ekspor Excel** (`app/Support/Xlsx.php`, tanpa library). Impor = upsert per kode, all-or-nothing.
- **Template COA Dagang = review akuntan** (`~/Downloads/Daftar COA.xlsx` di Mac user, 3 sheet). Lihat §4.
- Data dummy produksi-aman: `php artisan jurnal:demo` / `--hapus` (12 perusahaan `demo-*`, 5 akun `@demo.jurnal.id`).

**Belum jalan / belum terbukti:**

- **AI belum pernah menjawab sungguhan di produksi.** Key Gemini di server **kosong**. Key di `~/api/.env` (`GOOGLE_API_KEY`)
  ditolak Google: `429 — Your project has exceeded its monthly spending cap`. App sudah menampilkannya dengan benar sebagai
  "Kuota/saldo habis untuk model gemini-flash-latest — cek billing." Jadi pembacaan dokumen & Tanya pajak dengan Gemini
  **belum diuji end-to-end** (hanya dengan `Http::fake` + uji lama via OpenRouter `gpt-4.1-mini`).
- Tampilan HP (≤ 400px) belum dicek di browser — window Chrome tidak bisa diperkecil di bawah ~1200px saat sesi terakhir.
- Template **Manufaktur & Jasa belum di-review akuntan** (masih skema lama).

## 3. Produksi & deploy

| | |
|---|---|
| URL | <https://pajak.aipreneur.co.id> (Cloudflare wildcard DNS → VPS) |
| Server | `ssh aipreneur-vps` → root@31.97.66.119 (Hostinger, Ubuntu 24.04). **VPS dipakai bersama** banyak layanan lain (omni, campaignos, n8n, system-aipreneur, docker) — jangan sentuh selain milik Jurnal. |
| App | `/var/www/jurnal`, nginx `/etc/nginx/sites-available/jurnal-pajak`, php8.4-fpm, Let's Encrypt (webroot `/var/www/html`) |
| DB | MariaDB `jurnal` / user `jurnal` (sandi hanya di `/var/www/jurnal/.env`) |
| Kredensial | sandi admin awal `/root/jurnal-admin.txt` (**masih ada — sandinya pernah ditempel di chat, minta user ganti lalu hapus file**); sandi akun demo `/root/jurnal-demo.txt`. Jangan tulis isinya ke repo/chat. |
| Env penting | `APP_ENV=production`, `APP_DEBUG=false`, `AI_PROVIDER=gemini`, `GEMINI_MODEL=gemini-flash-latest`, `AI_QUEUE=false` (AI dibaca sinkron, nginx `fastcgi_read_timeout 180s`), `SESSION_ENCRYPT=true` |

**Deploy = rsync dari Mac, bukan git pull** (GitHub hanya penyimpan kode). Urutan yang terbukti aman:

```bash
npm run build                                   # server tidak punya npm; public/build ikut dikirim
rsync -az --exclude .git --exclude node_modules --exclude vendor --exclude .env --exclude '.env.*' \
  --exclude 'database/*.sqlite' --exclude 'storage/logs/*' --exclude 'storage/app/private/*' --exclude 'storage/app/dokumen' \
  --exclude 'storage/framework/sessions/*' --exclude 'storage/framework/views/*' --exclude 'storage/framework/cache/data/*' \
  --exclude 'bootstrap/cache/*.php' --exclude graphify-out --exclude .claude --exclude public/hot \
  ./ aipreneur-vps:/var/www/jurnal/
ssh aipreneur-vps 'cd /var/www/jurnal && chown -R www-data:www-data storage bootstrap/cache \
  && sudo -u www-data HOME=/tmp php artisan migrate --force \
  && sudo -u www-data HOME=/tmp php artisan optimize \
  && sudo -u www-data HOME=/tmp php artisan jurnal:cek-publik'
# composer.json berubah? tambahkan: COMPOSER_ALLOW_SUPERUSER=1 composer install --no-dev --optimize-autoloader
```

## 4. COA: dua skema kode (paling gampang salah)

- **Perdagangan** (`ChartOfAccounts::DAGANG`, 72 akun dari sheet "COA Baru" + 1407 Piutang Karyawan, 2116 Utang PPh 22,
  2117 Utang PPh Badan Ps. 29 — keputusan user): pendapatan di luar usaha **8xxx**, beban di luar usaha **9xxx**
  (9103 Beban Pajak), **tanpa 2103**. Akun bersyarat dari kolom klien:
  `legal_form` = pt → 2115 Utang Dividen; selain pt/kosong → 3102 Prive. `inventory_method` = perpetual (default) → 5101 HPP;
  periodik → 5102/5103/5105. Gedung/kendaraan (1503–1506) dibuat **nonaktif**.
- **Manufaktur & Jasa**: skema lama — 4901, **7xxx** (7103 Beban Pajak), ada 2103.
- Mesin yang mengenali keduanya: `ReportService::incomeStatement()` (rev. `8…` → `other_income`; beban `7…`/`9…` → `other`),
  `ChartOfAccounts::TAX_EXPENSE_CODES` (draft akrual PPh Final), `TaxCalculator::accountFor()` (fallback 2103 hanya ada di skema lama).
- **Batasan yang disadari:** pajak "other" yang *dipungut* (mis. PB1 restoran) tidak punya akun di Dagang → baris pajak dilewati,
  layar review menampilkan warning selisih. Kalau sering, usulkan akun "Utang Pajak Daerah (PB1)" ke user/akuntan.
- Migrasi `2026_10_15_000002_coa_dagang_revisi_akuntan` memanggil `ChartOfAccounts::revisiDagang()` (idempoten; kode diganti
  **di tempat** sehingga jurnal lama tetap menunjuk akun yang sama). Sudah jalan di lokal & produksi.
- Bukti kesesuaian: skrip pembanding sheet vs template → 72/72 cocok (kode, nama, tipe, subtype, saldo normal).
  Test: `CoaTemplateTest::test_template_dagang_sesuai_daftar_akuntan`.

## 5. Jebakan yang sudah memakan waktu (jangan diulang)

| Gejala | Penyebab | Yang benar |
|---|---|---|
| Produksi 500 `Class "Laravel\Pail\PailServiceProvider" not found` | rsync ikut mengirim `bootstrap/cache/packages.php` dari Mac (berisi paket dev) | exclude `bootstrap/cache/*.php`; kalau terlanjur: hapus file itu lalu `package:discover` sebagai www-data |
| Produksi 500 `Class "config" does not exist` | `config:cache` dijalankan sebagai root (umask 077) → www-data tak bisa baca | semua artisan di server: `sudo -u www-data HOME=/tmp php artisan …` |
| Test AI lolos/gagal acak, `configured()` true padahal `phpunit.xml` mengosongkan key | shell user meng-export `OPENAI_API_KEY` lama; `force="true"` di phpunit tidak menang | test yang butuh "tanpa key" set `config(['ai.openai.key' => ''])` sendiri; server lokal: `env -u OPENAI_API_KEY php artisan serve --port=8001` |
| `Http::fake` kedua tidak berlaku | fake bertumpuk; stub pertama yang cocok menang | fake per test, jangan di `setUp` kalau ada test yang butuh respons lain |
| Pesan "rate limit sementara" padahal kuota habis | Gemini OpenAI-compat membungkus error dalam array `[{"error":{…}}]` | sudah ditangani `OpenAiProvider::errorDetail()` (+ test) |
| Blade 500 "Undefined variable $row" | `@php(...)` sebaris lalu blok `@php … @endphp` di bawahnya → Blade menelan isi di antaranya | pakai bentuk blok `@php … @endphp` saja |
| Blade ParseError di atribut | `@json([...])` multi-baris tidak didukung | `data-x="{{ json_encode([...]) }}"` |
| Subtype 8101 tidak terganti di migrasi | key array `'4901'` jadi **int** di PHP → `=== '4901'` selalu false | bandingkan `(string) $key`; `revisiDagang()` sekarang juga memperbaiki sisa `other` → `other_income` |
| Halaman "Semua klien" menampilkan klien terakhir seolah aktif | `@foreach ($clients as $client)` bocor ke layout | layout mengambil klien HANYA dari `request()->route('client')` (+ test regresi) |

## 6. Langkah berikutnya (urut)

1. **Key Gemini**: user menaikkan spend cap di <https://ai.studio/spend> atau memberi key baru → admin isi di /pengaturan →
   klik "Tes koneksi AI". Lalu uji end-to-end di produksi: upload struk/PDF/mutasi + 2–3 pertanyaan di bubble Tanya pajak.
   Catat temuan di `TESTING-LOKAL.md` §5. **Jangan** isi key ke `.env` repo / chat.
2. **Keamanan & operasional server** (minta izin user dulu — VPS bersama):
   - user ganti sandi admin, lalu `rm /root/jurnal-admin.txt`;
   - backup harian: dump DB `jurnal` + `storage/app/private` (dokumen upload) + `.env` (APP_KEY — tanpa ini NPWP & key AI
     terenkripsi tidak bisa dibuka);
   - `apt install poppler-utils` (pdftotext; tanpa ini PDF ber-font CID dikirim utuh ke AI);
   - `APP_TIMEZONE=Asia/Jakarta` di `.env` server kalau user setuju (sekarang UTC).
3. **Sebelum klien asli masuk:** `sudo -u www-data php artisan jurnal:demo --hapus` di server (hapus 12 perusahaan & 5 akun demo).
   "Tes MP-Dagang" & "Tes MP-Manufaktur" milik akuntan — **jangan dihapus**. "Tes MP-Dagang" belum mengisi Bentuk badan usaha.
4. **Review akuntan untuk Manufaktur & Jasa** — kalau akuntan mengirim file serupa, ulangi pola §4 (konstanta daftar + migrasi
   `revisi…()` + test pembanding sheet). Keputusan user 15 Okt: perubahan umum (8/9xxx dst.) **hanya Dagang** sampai diminta.
5. Opsional yang sudah ditawarkan ke user, belum diminta: `./deploy.sh` satu perintah; ekspor laporan ke `.xlsx` (pakai `Xlsx`).
6. Cek tampilan HP (375px) halaman utama: dashboard, review dokumen, COA modal, bubble chat.

## 7. Peta berkas

| Berkas | Isi |
|---|---|
| `app/Services/Intake/DocumentExtractor.php` | Prompt + normalisasi hasil AI — inti produk |
| `app/Services/Intake/JournalDrafter.php` | Hasil baca → draft jurnal balance (+ baris pajak) |
| `app/Services/Accounting/JournalPoster.php` | Satu-satunya pintu jurnal (balance, isolasi klien, rollback) |
| `app/Services/Accounting/ChartOfAccounts.php` | 3 template COA, `seedFor()`, `revisiDagang()` |
| `app/Services/Ai/AiProviderFactory.php` | Pilih otak AI (key Gemini dari `settings` menang atas `.env`) |
| `app/Services/Reporting/ReportService.php` | Semua laporan; pengelompokan Laba Rugi dua skema |
| `app/Services/Tax/{TaxRecap,TaxCalculator}.php`, `config/pajak.php` | Rekap & akun pajak per subtype |
| `app/Http/Controllers/{AccountController,TaxChatController,SettingController}.php` | COA modal+impor/ekspor, Tanya pajak, Pengaturan AI |
| `app/Support/Xlsx.php` | Baca/tulis .xlsx minimal (ZipArchive) |
| `app/Console/Commands/DemoDataCommand.php` | `jurnal:demo` / `--hapus` |
| `app/Models/User.php` | Peran, `ABILITIES`, `CLIENT_LIMITS` |
| `resources/js/app.js` | Semua JS (vanilla): modal, palet ⌘K, dropzone, bubble chat, batas perusahaan |

## 8. Aturan kerja dari user (wajib)

- Bahasa Indonesia, ringkas. Jalankan & verifikasi dengan output nyata (`php artisan test`), jangan klaim tanpa bukti.
- **Commit tanpa trailer co-author AI** (user menghapus "Co-Authored-By: Claude" dari seluruh riwayat sebelum push pertama).
  Author tetap `reyfuu`. Push ke `origin main` hanya bila user meminta / alurnya sudah disepakati.
- Minimalis (ponytail): reuse kode yang ada → stdlib → fitur native → baru kode baru; tandai jalan pintas dengan `// ponytail:`.
- Logika uang/parsing/isolasi wajib meninggalkan test. Test merah = kode yang salah, bukan test-nya.
- Aksi ke luar (deploy, ubah server bersama, kirim pesan) → konfirmasi user dulu kecuali sudah diminta eksplisit.
- Lokal: Mac, PHP 8.5, Laravel 13, SQLite. Login lokal `admin@example.com` / `password123`; akun demo lokal sandi `Coba-Lokal-2026`.
