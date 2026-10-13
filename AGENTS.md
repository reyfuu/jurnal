# AGENTS.md — Jurnal AI

Aturan proyek yang portabel untuk semua agent (Claude Code, Codex, Gemini, dll).
Sumber kebenaran konteks: [docs/HANDOFF.md](docs/HANDOFF.md). Baca itu dulu.

## Proyek

Aplikasi Laravel berdiri sendiri: **upload → AI baca → review manusia → posting → laporan**.
Sumber dokumen: foto struk, PDF invoice, screenshot mutasi bank, teks tempel.

**Aturan nomor satu: AI tidak pernah memposting sendiri.** Hasil baca selalu
mendarat di layar review. Yang masuk buku hanya angka yang sudah dilihat manusia.

## Status folder ini

Source diekstrak (13 Okt 2026) dari `origin/claude/quirky-cori-kbvckj:jurnal-ai/`
di `~/skinku-b2b-portal` via `git archive` — commit `9be6dcd`. Folder ini belum
repo git sendiri. Setup lokal Mac sudah jalan: SQLite, seed + DemoSeeder, 134 test hijau.
Login lokal: `admin@example.com` / `password123` (diubah langsung di DB; seed ulang
mengembalikan ke bawaan `admin@jurnal.test` / `password123`). Jangan taruh
`SEED_ADMIN_*` di `.env` lokal — `SeederGuardTest` ikut membacanya dan merah.

AI belum tersambung: key 9router di `~/api/.env` ditolak (401). Isi
`OPENAI_API_KEY` / `OPENAI_API_BASE` / `AI_MODEL` di `.env` dengan key valid
(model wajib vision), lalu `php artisan config:clear`.

## Stack & perintah

- PHP 8.3+ (lokal Mac: 8.5 via Homebrew), Composer, Laravel, SQLite (lokal) / MySQL (produksi).
- `composer install` · `php artisan migrate` · `php artisan test` · `php artisan serve --port=8001`
- Seed demo **hanya lokal**: `php artisan db:seed --class=DemoSeeder`.
- Sebelum diekspos publik: `php artisan jurnal:cek-publik`.

## Keputusan desain yang menahan beban (jangan dibongkar)

Tiap poin punya test. Test merah = kode yang salah, bukan test-nya.

- **Klien = isolasi penuh.** COA, jurnal, laporan per klien. `JournalPoster` menolak akun milik klien lain.
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

## Urutan kerja berikutnya (dari HANDOFF §7)

1. Tunnel: `jurnal:cek-publik` lalu `cloudflared tunnel --url http://127.0.0.1:8001`.
2. Tuntaskan checklist uji manual `TESTING-LOKAL.md` §4 — fokus kejujuran AI (nota buram → warning, bukan angka karangan).
3. Laporan Arus Kas (operasi / investasi / pendanaan dari mutasi akun kas/bank).
4. Queue untuk pembacaan AI.
5. Ekspor laporan PDF/Excel.
5b. Perpajakan MVP sudah jalan (docs/perpajakan/README.md) — tarif wajib diverifikasi konsultan sebelum dipakai klien.
6. Putuskan opsi A/B (HANDOFF §2) — keputusan user, bukan agent.

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
