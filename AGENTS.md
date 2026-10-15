# AGENTS.md — Jurnal AI

Aturan proyek yang portabel untuk semua agent (Claude Code, Codex, Gemini, dll).
Sumber kebenaran konteks: [docs/HANDOFF.md](docs/HANDOFF.md). Baca itu dulu.

## Proyek

Aplikasi Laravel berdiri sendiri: **upload → AI baca → review manusia → posting → laporan**.
Sumber dokumen: foto struk, PDF invoice, screenshot mutasi bank, teks tempel.

**Aturan nomor satu: AI tidak pernah memposting sendiri.** Hasil baca selalu
mendarat di layar review. Yang masuk buku hanya angka yang sudah dilihat manusia.

## Status folder ini

Repo git sendiri (branch `main`), diekstrak 13 Okt 2026 dari
`skinku-b2b-portal@9be6dcd:jurnal-ai/`. Lokal Mac jalan: SQLite + DemoSeeder.
Login lokal: `admin@example.com` / `password123` (diubah langsung di DB; seed ulang
mengembalikan ke bawaan `admin@jurnal.test`). Jangan taruh `SEED_ADMIN_*` di `.env`
lokal — `SeederGuardTest` ikut membacanya.

AI: **Gemini** — admin isi key di menu **Pengaturan AI** (`/pengaturan`, tabel `settings`,
terenkripsi APP_KEY; ganti APP_KEY = isi ulang key). Key tersimpan menang atas `.env`;
kalau kosong, jatuh ke `.env` (`AI_PROVIDER=gemini|openai|anthropic`).
Sebelumnya lokal: OpenRouter (`OPENAI_API_BASE=https://openrouter.ai/api/v1`,
`AI_MODEL=openai/gpt-4.1-mini` — `gpt-4o-mini` salah baca mutasi bank). **Shell
user meng-export `OPENAI_API_KEY` lama (ditolak) yang menang atas `.env`** —
jalankan server dengan `env -u OPENAI_API_KEY php artisan serve --port=8001`.
`phpunit.xml` mengunci env AI (`force="true"`) supaya test tidak ikut `.env`/shell.

## Stack & perintah

- PHP 8.3+ (lokal Mac: 8.5 via Homebrew), Composer, Laravel, SQLite (lokal) / MySQL (produksi).
- `composer install` · `php artisan migrate` · `php artisan test` · `php artisan serve --port=8001`
- Seed demo **hanya lokal**: `php artisan db:seed --class=DemoSeeder`.
- Data dummy (12 perusahaan, 5 akun per peran) — boleh di server karena bertanda `demo-` / `@demo.jurnal.id`:
  `php artisan jurnal:demo` · bersihkan sebelum data asli: `php artisan jurnal:demo --hapus`.
- Sebelum diekspos publik: `php artisan jurnal:cek-publik`.

## Keputusan desain yang menahan beban (jangan dibongkar)

Tiap poin punya test. Test merah = kode yang salah, bukan test-nya.

- **Klien = isolasi penuh.** COA, jurnal, laporan per klien. `JournalPoster` menolak akun milik klien lain.
- **COA per jenis usaha.** `clients.business_type` (perdagangan · manufaktur · jasa) memilih template di
  `ChartOfAccounts::template()`. Akun yang dipakai mesin & pajak wajib ada di SEMUA template (`CoaTemplateTest`).
  **Dua skema kode:** Dagang = revisi akuntan (`Daftar COA.xlsx`, 15 Okt 2026): pendapatan di luar usaha 8xxx,
  beban di luar usaha 9xxx, tanpa 2103; akun bersyarat dari `legal_form` (PT → Utang Dividen, lainnya → Prive) &
  `inventory_method` (perpetual → 5101, periodik → 5102/5103/5105). Manufaktur/Jasa = skema lama (4901, 7xxx, 2103).
  Laba Rugi mengenali keduanya (8xxx, 7xxx/9xxx); beban pajak lewat `ChartOfAccounts::TAX_EXPENSE_CODES`.
  COA diubah lewat modal; impor/ekspor Excel via `app/Support/Xlsx.php` (tanpa library, ZipArchive). Impor = upsert
  per kode di klien itu saja, satu baris salah = batal semua. Tipe akun yang sudah dipakai jurnal terkunci.
- **Tanya pajak (bubble kanan bawah).** `TaxChatController` — AI hanya diberi data klien yang sedang dibuka;
  angka buku hanya untuk peran `books.view` (anggota: profil saja). Read-only, riwayat di sessionStorage.
- **Peran & akses per klien.** Admin · supervisor · akuntan · anggota; hak per kemampuan di `User::ABILITIES`
  (middleware `ability:`). Admin melihat semua klien; peran lain hanya klien di `client_user` (middleware
  `client.access` → 404), dibatasi `User::CLIENT_LIMITS`: supervisor 3 · akuntan 5 · anggota 10 perusahaan.
  Klien yang dibuat supervisor otomatis ditugaskan padanya (ditolak kalau batas penuh).
  Anggota: upload & lihat dokumen saja. Kelola pengguna & tarif: admin. Hapus permanen: admin.
- **Angka dari AI dicuci di PHP.** `Rupiah::parse()` ("Rp 1.250.000", "85rb", "1,5jt", "(50.000)"). Tanggal gaya Indonesia (`03/04/2026` = 3 April); tahun ngawur ditolak.
- **Kode akun karangan tidak lolos** → dialihkan ke akun default + warning.
- **Draft dijamin balance**; selisih dengan total dokumen = **warning**, bukan ditambal diam-diam.
- **Anti-dobel dua lapis:** `documents.content_hash` + `journals.fingerprint`.
- **Posting gagal sebagian = rollback semua** (transaksi DB).
- **Nonaktifkan, jangan hapus.** Jurnal salah di-*void*.
- **Pagar kredensial:** `app/Support/SandiLemah.php` (cek kandungan kata). Seeder produksi wajib `SEED_ADMIN_PASSWORD` kuat.
- **Satu pintu jurnal:** `app/Services/Accounting/JournalPoster.php`. Inti produk: `app/Services/Intake/DocumentExtractor.php`.

## Dilarang

- `migrate:fresh` / `migrate:refresh` di tempat yang ada data nyata.
- `DemoSeeder` di luar lokal.
- `DB_DATABASE` mengarah ke database portal B2B SKINKU (tidak ada hubungan apa pun dengannya).
- `APP_DEBUG=true` saat bisa diakses internet.
- Menghapus validasi balance, isolasi klien, atau anti-dobel demi diff pendek.
- Commit `.env` / API key. Key ada di `~/api/.env`.

## Urutan kerja berikutnya

Selesai 13 Okt 2026: Arus Kas, queue AI (opt-in `AI_QUEUE`), ekspor laporan
(CSV + cetak/PDF), perpajakan MVP + sebagian Fase 2 (docs/perpajakan/README.md).

Uji manual dengan AI asli: lihat `TESTING-LOKAL.md` §5.

Terblokir input dari luar:
1. Saldo OpenRouter hampir habis (free tier) — top up, atau ganti key.
2. Verifikasi tarif pajak oleh konsultan (BRD §7) → tandai di /pajak/tarif.
3. Keputusan Freddie: opsi A/B (HANDOFF §2), default pajak TRD Q1–Q3, dark mode.
4. Uji dengan dokumen ASLI klien (yang sudah: dokumen sintetis, §5).

## Skill & tooling wajib

| Kapan | Pakai |
|---|---|
| Pertanyaan tentang codebase | **graphify** — kalau `graphify-out/graph.json` ada: `graphify query "<q>"` sebelum grep. Setelah ubah kode: `graphify update .` |
| Semua perintah shell | **RTK** — otomatis lewat hook. Cek hemat: `rtk gain` |
| Setiap tugas coding | **ponytail** (full) — tangga YAGNI → reuse → stdlib → native → dependency terpasang → minimal. Bug fix di fungsi bersama, bukan per-caller. Shortcut ditandai `// ponytail:` |
| Kerja UI/UX | **impeccable-design** + `docs/DESIGN-BRIEF.md` (pola shadcn/Kokonut/21st di-port ke Blade, tanpa React) |
| Sebelum fitur baru / keputusan besar (Arus Kas, queue, A vs B) | **grill-me** — tanya-jawab sampai rencana kokoh, baru tulis kode |
| Alur eksekusi | **superpowers**: `brainstorming` → `writing-plans` → `executing-plans` / `subagent-driven-development`; `test-driven-development` untuk logika akuntansi; `systematic-debugging` saat test merah; `verification-before-completion` sebelum klaim selesai |

Aturan main: grill dulu, rencana kecil, test dulu, kode paling malas yang lolos
test, verifikasi dengan output nyata (`php artisan test`), lalu `graphify update .`.

## Gaya

- Bahasa Indonesia, ringkas. Istilah dev tetap Inggris.
- Ikuti pola kode yang sudah ada; tidak ada abstraksi baru tanpa kebutuhan nyata.
- Logika non-trivial (uang, parsing, isolasi) wajib meninggalkan satu test.
- File `.bat` Windows sudah dihapus (13 Okt 2026). Instruksi jalan ada di `CARA-PAKAI.txt` (Mac/terminal).
