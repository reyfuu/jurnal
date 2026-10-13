# Perpajakan — Indeks Dokumen

Fitur pajak untuk Jurnal AI: membantu bookkeeper mencatat, merekap, dan mengingatkan kewajiban pajak per klien. **Asisten, bukan konsultan pajak; AI tidak pernah memposting atau melapor sendiri.**

Urutan baca:

1. [BRD.md](BRD.md) — kenapa: masalah, tujuan, scope, risiko, **tabel angka regulasi yang wajib diverifikasi** (BR-xx).
2. [PRD.md](PRD.md) — apa yang dirasakan user: persona, user story MoSCoW, alur, UX, fase rilis (US-xx).
3. [FRD.md](FRD.md) — perilaku sistem yang bisa diuji: FR-xx + Given/When/Then.
4. [TRD.md](TRD.md) — cara membangunnya di codebase ini: tabel, service, hook ke Extractor/Drafter/Poster, test, rollout, **matriks traceability BR → US → FR → TR**.

Status: draft 13 Okt 2026 · Pemilik keputusan: Freddie · Belum ada kode yang diubah.

## Status implementasi (13 Okt 2026)

MVP Fase 1 (US-01..13) jalan, 15 test di `tests/Feature/TaxTest.php`. Default
keputusan terbuka: Q1 tanggal dokumen, Q2 profil kosong = PKP (perilaku lama),
Q3 jurnal manual tanpa panel pajak.

Menyimpang dari TRD (sengaja, lebih sedikit kode):
- Kode pajak, pemetaan akun, dan jatuh tempo di `config/pajak.php`, bukan tabel
  `tax_codes`/`tax_obligation_rules`. Tarif tetap data (`tax_rates`, berversi).
- Tabel `holidays` belum ada: jatuh tempo hanya digeser dari Sabtu/Minggu (V20 sebagian).
- PPN non-PKP ditambahkan ke baris rincian terbesar, bukan dibagi proporsional.
- Nominal catatan pajak saat posting diambil dari baris jurnal di akun pajaknya,
  bukan dari form; tiap pajak punya centang "Catat".
- `fromTransactions()` (mutasi bank) belum membaca pajak.

Fase 2 sebagian jalan: draft jurnal akrual/setor dari rekap (US-14), kalender
pajak lintas klien di dashboard (US-15), jurnal manual di akun pajak bermakna
tunggal tercatat otomatis — PPh 21 diisi manusia (US-18, `config/pajak.php`
`manual_map`).

Belum, dan kenapa:
- Template Excel Coretax (US-16): menunggu format impor diverifikasi (BRD V22).
- Pengingat email/WA (US-17, Fase 3), register bukti potong, PJAP.
- Pajak di mutasi bank: sesuai TRD Q5 tidak dipetakan otomatis.
- Semua angka BRD §7 masih **belum diverifikasi** konsultan.
