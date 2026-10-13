# TRD — Perpajakan Jurnal AI

Status: draft 13 Okt 2026 · Dari: [FRD](FRD.md) · Invariant wajib: [AGENTS.md](../../AGENTS.md) §"Keputusan desain yang menahan beban"

Prinsip teknis: **tambah, jangan bongkar.** Semua migration additive; semua jurnal tetap lewat `JournalPoster`; test lama (134) wajib hijau tanpa diubah, kecuali yang memang menguji perilaku pajak lama (lihat §8).

---

## 1. Kondisi kode hari ini (titik sentuh)

| Berkas | Perilaku pajak sekarang | Perubahan |
|---|---|---|
| `app/Services/Accounting/ChartOfAccounts.php` | `1402 PPN Masukan` (`tax_in`), `2103 Hutang Pajak` (`tax_out`), `7103 Beban Pajak`. `seedFor()` idempoten. | Tambah 5 akun (TR-06). |
| `app/Services/Intake/DocumentExtractor.php` | Skema JSON punya `tax` skalar; `normalize()` hanya `Rupiah::parse($raw['tax'])`; cek silang `rincian + tax − diskon ≈ total`. | Tambah `taxes[]` + `normalizeTaxes()` (TR-08). |
| `app/Services/Intake/JournalDrafter.php` | `fromItems()`: `tax` selalu di sisi rincian → `1402` (beli) / `2103` (jual). PPh dipotong = salah arah; non-PKP tidak dibedakan. | Ganti blok pajak dengan loop `taxes` berbasis profil (TR-09). |
| `app/Services/Accounting/JournalPoster.php` | `record($header, $lines, $status)` dalam satu `DB::transaction`. | Parameter opsional `$taxEntries` disimpan di transaksi yang sama (TR-10). |
| `app/Http/Controllers/DocumentController.php` | `post()` loop draft dalam `DB::transaction`, gagal sebagian = rollback, fingerprint dicek. | Validasi + teruskan `drafts.*.taxes.*` (TR-11). |
| `clients.npwp` | `string(30)` bebas, plaintext. | Digantikan `tax_id` terenkripsi (TR-01). |
| `app/Services/Reporting/ReportService.php` | Agregasi per `journals.period`, status posted. | Dipakai ulang untuk cek silang (TR-12). |

## 2. Data model

Semua tabel klien ber-`client_id` FK `constrained()`; nilai pajak `unsignedBigInteger` (rupiah bulat).

### TR-01 `clients` — kolom profil pajak (migration `add_tax_profile_to_clients_table`)

| Kolom | Tipe | Catatan |
|---|---|---|
| `taxpayer_type` | string(20) null | `orang_pribadi` \| `badan` |
| `tax_id` | text null | NPWP 16 / NIK, cast `encrypted` |
| `tax_id_hash` | string(64) null, index | `hash_hmac('sha256', digits, app.key)` untuk pencarian/duplikat |
| `nitku` | string(22) null | |
| `is_pkp` | boolean default false | |
| `pkp_since` | date null | |
| `pph_regime` | string(20) default `normal` | `normal` \| `final_umkm` |
| `regime_start_year` | unsignedSmallInteger null | |
| `revenue_account_ids` | json null | akun omzet Final UMKM; null = semua akun subtype `sales`,`service` |

Migration menyalin `npwp` → `tax_id` (dinormalisasi). Kolom `npwp` **tidak di-drop** di MVP; drop di migration terpisah setelah verifikasi. Logika per tanggal di model: `Client::isPkpOn(string $date): bool`, `Client::isFinalUmkmOn(string $date): bool` (tanpa service baru).

### TR-02 Master global `tax_codes` & `tax_rates`

`tax_codes`: `id`, `code` string(30) unique, `name`, `kind` (`vat`\|`withholding`\|`final`), `subtype_in` string(40) null (aset: PPN Masukan / dibayar dimuka), `subtype_out` string(40) null (liabilitas: PPN Keluaran / utang PPh), `is_active`, timestamps.

`tax_rates`:

| Kolom | Tipe | Catatan |
|---|---|---|
| `tax_code_id` | FK | |
| `applies_to` | string(20) null | `orang_pribadi` \| `badan` \| null = semua |
| `rate_bp` | unsignedInteger | basis poin: 12% = 1200, 0,5% = 50, 1,75% = 175 |
| `rate_no_tax_id_bp` | unsignedInteger null | tarif tanpa NPWP lawan |
| `dpp_numerator` / `dpp_denominator` | unsignedInteger default 1/1 | DPP nilai lain 11/12 |
| `exempt_threshold` | unsignedBigInteger null | Rp500 jt bebas OP UMKM per tahun |
| `regime_cap` | unsignedBigInteger null | Rp4,8 M batas rezim |
| `rounding` | string(10) default `floor` | `floor` \| `half_up` |
| `effective_from` | date | |
| `effective_to` | date null | |
| `source`, `notes` | string / text | dasar hukum |
| `verified_at`, `verified_by` | timestamp / FK users null | diisi saat konsultan menyetujui |
| `created_by` | FK users null | |

Index `unique(tax_code_id, applies_to, effective_from)`. Non-overlap dijaga di aplikasi (`TaxRateController@store` menutup versi sebelumnya dalam transaksi). Baris yang direferensikan `tax_entries` tidak bisa di-update (cek di controller + test).

### TR-03 `tax_entries` — catatan pajak per jurnal

| Kolom | Tipe | Catatan |
|---|---|---|
| `client_id` | FK | isolasi + index |
| `journal_id` | FK cascadeOnDelete | status & periode diambil dari `journals` (tidak diduplikasi) |
| `account_id` | FK accounts | akun pajak yang dipakai; wajib ada di baris jurnal ini |
| `document_id` | FK null nullOnDelete | |
| `tax_code_id` | FK | |
| `tax_rate_id` | FK null | null = tidak ada tarif berlaku |
| `direction` | string(4) | `in` (aset/kredit pajak) \| `out` (liabilitas) |
| `withheld` | boolean | PPh potong |
| `base` | unsignedBigInteger | DPP/bruto |
| `amount` | unsignedBigInteger | nilai yang masuk buku (dari form) |
| `computed_amount` | unsignedBigInteger null | hitung sistem |
| `rate_bp_snapshot`, `dpp_factor_snapshot` | uint / string(10) | jejak tarif saat posting |
| `match_status` | string(10) | `match` \| `mismatch` \| `no_rate` \| `manual` |
| `noncreditable` | boolean default false | PPN non-PKP (tidak masuk rekap PPN) |
| `counterparty_name` | string null | |
| `counterparty_tax_id` | text null | cast `encrypted` |
| `document_number` | string(60) null | no. faktur / bukpot |
| `created_by` | FK users null | |

Index: `(client_id, tax_code_id)`, `journal_id`. Rekap selalu `join journals` + `journals.status = posted` + `journals.period` → void otomatis keluar, sama seperti `ReportService`.

### TR-04 Kalender: `tax_obligation_rules`, `tax_filings`, `holidays`

- `tax_obligation_rules` (global, data-driven): `tax_code_id`, `action` (`pay`\|`file`), `trigger` (`pkp`\|`final_umkm`\|`has_withholding`), `months_after` tinyint default 1, `day_of_month` tinyint null, `end_of_month` bool, `effective_from`, `effective_to`, `source`.
- `tax_filings` (per klien, dibuat hanya saat user mencatat status): `client_id`, `tax_code_id`, `period` string(7), `status` (`belum`\|`setor`\|`lapor`\|`nihil`), `paid_amount` ubigint null, `paid_at` date, `ntpn` string(16) null, `billing_code` string(15) null, `filed_at` date, `receipt_number` string(60) null, `note`, `updated_by`, timestamps. `unique(client_id, tax_code_id, period)`.
- `holidays`: `date` unique, `name`. Diisi admin; tanpa API eksternal.

Kewajiban (obligation) **dihitung on-the-fly** dari profil + rules + ada/tidaknya `tax_entries` — tidak disimpan. ponytail: tidak ada tabel jadwal; tambah kalau butuh pengingat terjadwal (Fase 3).

### TR-05 `audit_logs`

`user_id`, `client_id` null, `subject_type`, `subject_id`, `action`, `before` json, `after` json, `created_at`. Index `(client_id, created_at)`. Helper satu fungsi `App\Support\Audit::log(...)` dipanggil eksplisit di controller profil/tarif/status/void. Append-only (tidak ada route update/delete).

### TR-06 Template COA

Tambah ke `ChartOfAccounts::template()`:

| Kode | Nama | Tipe | Subtype |
|---|---|---|---|
| 1403 | Pajak Dibayar Dimuka | asset | `prepaid_tax` |
| 2106 | PPN Keluaran | liability | `vat_out` |
| 2107 | Utang PPh 21 | liability | `wht_payable_pph21` |
| 2108 | Utang PPh 23 | liability | `wht_payable_pph23` |
| 2109 | Utang PPh 4(2) | liability | `wht_payable_pph4_2` |
| 2110 | Utang PPh Final | liability | `final_payable` |

Klien lama: tombol `clients.coa.sync` (sudah ada, idempoten, melewati kode yang sudah dipakai). Pemetaan akun via `tax_codes.subtype_in/out` → akun aktif pertama dengan subtype itu; tidak ketemu → akun `tax_out` (2103) + warning. Test COA memakai `count(template())`, jadi aman.

## 3. Uang & pembulatan (TR-07)

- Pajak = **integer rupiah**. `journal_lines.debit/credit` tetap `decimal(18,2)` (tidak dimigrasi); nilai pajak ditulis sebagai `x.00`.
- Rumus tanpa float: `raw = base × rate_bp × num`, `den = 10000 × den_dpp`.
  - `floor`: `intdiv(raw, den)` · `half_up`: `intdiv(2×raw + den, 2×den)`.
  - Batas aman: base ≤ 10¹³ × 1200 × 12 ≈ 1,4×10¹⁷ < `PHP_INT_MAX`. Base di atas 10¹³ ditolak (validasi).
- Toleransi cocok: `max(1, jumlah item) × config('pajak.toleransi_rupiah', 1)`.
- PPN non-PKP dibagi proporsional ke item; sisa pembulatan ke item terbesar supaya total tepat.
- `rate_percent` dari AI → bp: `(int) round(parse × 100)`; nilai ≤ 1 dianggap pecahan (×100 dulu).

## 4. Service

| ID | Kelas | Tanggung jawab | Catatan |
|---|---|---|---|
| TR-07 | `App\Services\Tax\TaxCalculator` | `rateFor(code, date, ?taxpayerType): ?TaxRate` · `compute(base, TaxRate, hasTaxId): int` · `status(amount, computed, items): string` | Murni; satu-satunya tempat rumus pajak. |
| TR-12 | `App\Services\Tax\TaxRecap` | `ppn(client, period)` · `withholding(client, period)` · `finalUmkm(client, period)` · `reconcile(client, period)` · `obligations(client, period)` | Satu kelas; query `tax_entries ⋈ journals` + `ReportService::movements()` untuk cek silang. |
| — | `App\Support\TaxId` | `normalize(string): array{digits, warning}` · `mask()` | Statik, pola sama dengan `Rupiah`. |
| — | `App\Support\Audit` | `log(...)` | Satu fungsi. |

Tidak ada interface, repository, atau event bus. ponytail: tambah kalau ada implementasi kedua.

## 5. Hook ke alur yang ada

### TR-08 `DocumentExtractor`
- **Prompt (`systemPrompt`)**: ganti aturan "PPN/PPh taruh terpisah di tax" dengan:
  - tambahkan ke skema `"taxes": [{"type": "ppn"|"ppn_mewah"|"pph21"|"pph22"|"pph23"|"pph4_2"|"other", "label": string, "base": number|null, "rate_percent": number|null, "amount": number, "withheld": boolean, "document_number": string|null, "counterparty_tax_id": string|null}]`;
  - aturan: "PPh yang MENGURANGI tagihan → withheld=true. Salin angka persis seperti tertulis; jangan menghitung pajak sendiri. Tidak ada di dokumen → null.";
  - `tax` tetap ada = jumlah pajak yang menambah tagihan (kompatibilitas).
  - Prompt **tidak** memuat profil/NPWP klien (FR-26).
- **`normalize()`**: panggil `normalizeTaxes($raw, $client, $docDate, count($items))` → tiap entri dicuci `Rupiah::parse`, di-`round` ke int, `type` divalidasi, `computed_amount` & `match_status` dari `TaxCalculator`, warning per selisih. Dokumen lama tanpa `taxes` tapi `tax > 0` → satu entri `ppn` + warning (FR-08 AC3).
- Cek silang total diubah: `rincian + Σ pajak tambah − Σ pajak withheld − diskon ≈ total` (beberapa invoice mencantumkan total sebelum potong PPh → kedua bentuk diterima, warning hanya jika keduanya meleset).

### TR-09 `JournalDrafter::fromItems`
Blok `if ($tax >= 0.005)` diganti loop `extraction['taxes']`:

```
foreach taxes as t:
  ppn & !client->isPkpOn(date)      → tambahkan ke baris rincian (proporsional), entri noncreditable
  ppn & beli                        → D akun subtype_in   (1402)
  ppn & jual                        → K akun subtype_out  (2106)
  withheld & beli (kita potong)     → K akun subtype_out  (2108 dst.)
  withheld & jual (dipotong lawan)  → D akun subtype_in   (1403)
settlement = sisa (tidak berubah) → draft tetap selalu balance
```
Tiap draft mendapat kunci baru `taxes` (array entri siap posting, berisi `account_id`). `fromTransactions()` (mutasi bank) tidak diubah di MVP.

### TR-10 `JournalPoster::record($header, $lines, $status = POSTED, array $taxEntries = [])`
- Sebelum transaksi: per akun pajak, `Σ baris jurnal di akun itu (sisi sesuai direction) = Σ amount entri akun itu` → jika tidak, `AccountingException` (FR-14). Entri `noncreditable` (PPN non-PKP, akunnya akun beban) dikecualikan. Akun entri pasti milik klien karena wajib muncul di `$lines` yang sudah lolos `assertAccountsBelongTo`.
- Di dalam `DB::transaction` yang sama: `$journal->taxEntries()->createMany(...)` dengan `client_id` dari header (bukan dari input).
- Pemanggil lama (tanpa argumen ke-4) tidak berubah.

### TR-11 `DocumentController::post` + `documents/show`
- Validasi tambahan: `drafts.*.taxes.*.{tax_code, account_id, direction, withheld, base, amount, document_number, counterparty_name, counterparty_tax_id}`; `amount`/`base` `integer|min:0|max:10000000000000`.
- Server menghitung ulang `computed_amount`/`match_status` (tidak percaya hidden input) sebelum `record()`.
- Rollback semua & anti-dobel fingerprint: tidak berubah — entri pajak ikut transaksi luar yang sudah ada.
- View: partial baru `documents/_tax-panel.blade.php` di layar review (kolom: Jenis · DPP · Tarif · Pajak AI · Hitung · Status · No. faktur), memakai `ai-marker` & warning per baris sesuai DESIGN-BRIEF; JS vanilla untuk hitung ulang tampilan (validasi server tetap otoritatif).

## 6. Routes, controller, view (TR-13)

```
// dalam group auth+active, pola /klien/{client}/... seperti routes/web.php
GET  /klien/{client}/pajak                     TaxController@index        tax.index (ringkasan masa + kalender)
GET  /klien/{client}/pajak/ppn                 TaxController@ppn          tax.ppn
GET  /klien/{client}/pajak/pph                 TaxController@withholding  tax.pph
GET  /klien/{client}/pajak/final-umkm          TaxController@finalUmkm    tax.final
GET  /klien/{client}/pajak/ekspor/{jenis}      TaxController@export       tax.export
POST /klien/{client}/pajak/status              TaxController@storeFiling  tax.filing
GET  /klien/{client}/pajak/profil              TaxController@profile      tax.profile
// admin
PUT  /klien/{client}/pajak/profil              TaxController@updateProfile
GET|POST /pajak/tarif                          TaxRateController@index|store
GET|POST /pajak/hari-libur                     TaxRateController@holidays|storeHoliday
GET  /audit                                    TaxRateController@audit    (filter ?klien=)
```

- Periode via `partials/period-picker` yang sudah ada (`?period=YYYY-MM`).
- `ppn`/`final-umkm` → 404 jika profil tidak relevan (FR-15 AC3).
- Setiap aksi yang menerima id entri/filing memakai pola `assertOwned` (`abort_unless(... client_id === $client->id, 404)`).
- Views: `resources/views/tax/{index,ppn,pph,final,profile,rates,audit}.blade.php`; tab "Pajak" di `partials/client-nav.blade.php` sampai sidebar DESIGN-BRIEF jadi; partial `tax/_disclaimer.blade.php` dipakai di semua halaman.

### TR-14 Ekspor
`response()->streamDownload()` + `fputcsv($fh, $row, ';')`, BOM UTF-8 supaya Excel membaca rupiah/aksara benar. Tanpa dependency baru. Template Excel Coretax = Fase 2, setelah BRD V22 terverifikasi (cek dulu apakah `phpoffice/phpspreadsheet` memang perlu; CSV mungkin cukup untuk konverter DJP).

## 7. Keamanan & performa

### TR-15 Keamanan
- `tax_id`, `counterparty_tax_id`: cast `encrypted`. **Risiko:** `APP_KEY` hilang = NPWP hilang → wajib dicatat di backup/`DEPLOY-JURNAL.md`.
- Masking via `TaxId::mask()` di semua tabel & ekspor (ekspor ke Coretax butuh NPWP penuh → kolom penuh hanya di ekspor, akses dicatat di `audit_logs`).
- CSV injection: sel yang diawali `= + - @` diberi prefiks `'`.
- NPWP/NIK tidak pernah masuk prompt AI.
- Admin-only route memakai middleware `admin` yang sudah ada.
- Keterbatasan diketahui: semua user melihat semua klien (HANDOFF §4) — tidak diselesaikan di sini.

### Performa
- Rekap = satu query agregat per jenis per masa, ditopang index `journals(client_id, period, status)` yang sudah ada + `tax_entries(client_id, tax_code_id)`.
- Final UMKM kumulatif = satu query `period BETWEEN 'YYYY-01' AND period`.
- Tanpa cache. ponytail: tambah cache kalau rekap > 300 ms pada data nyata.
- Prompt bertambah ±25 baris → token naik kecil; tidak mengubah kebutuhan queue (HANDOFF §7.4).

## 8. Test yang ditambah (TR-16)

**Unit**
- `TaxCalculatorTest`: 11/12 × 12% (1.000.000 → 110.000), floor vs half_up, tarif tanpa NPWP, tidak ada tarif berlaku → null, pemilihan versi per tanggal (batas `effective_from`/`to`), base > 10¹³ ditolak.
- `TaxIdTest`: normalisasi titik/strip, 15 digit → warning, panjang salah → invalid, mask.

**Feature**
- `TaxExtractionTest`: `taxes[]` dinormalisasi (`"(100.000)"`, `"12%"`, `0.12`), type asing → `other`, back-compat `tax` skalar → entri ppn + warning, selisih → warning dan nilai dokumen **tidak** diubah.
- `TaxDrafterTest`: PKP beli PPN (1402) · PKP jual (2106) · non-PKP kapitalisasi proporsional + sisa pembulatan · PPh 23 kita potong (2108, bayar = sisa) · PPh 23 dipotong pelanggan (1403) · kombinasi PPN + PPh 23 (FR-10 AC1) · akun subtype hilang → 2103 + warning · semua draft balance.
- `TaxPostingTest`: entri tersimpan bersama jurnal · satu draft gagal → 0 jurnal & 0 entri · fingerprint dobel → 0 entri baru · entri ≠ baris jurnal → ditolak (FR-14) · akun klien lain → ditolak · void → keluar dari rekap.
- `TaxRecapTest`: PPN kurang bayar (FR-15 AC1) · PPh potput dua bagian · Final UMKM OP lewat batas Rp500 jt (FR-17 AC1) · badan tanpa bagian bebas · tarif berubah tengah tahun · reconcile rekap = Neraca Saldo, dan mendeteksi jurnal manual tanpa entri.
- `TaxRateTest`: versi baru menutup versi lama · tarif terpakai tidak bisa diubah · tarif belum diverifikasi → badge.
- `TaxObligationTest`: jatuh tempo per aturan, geser hari libur/Minggu, status manual saja.
- `TaxAccessTest`: staff tidak bisa ubah profil/tarif/audit (403) · entri/filing klien lain → 404 · menu PPN non-PKP → 404.
- `TaxExportTest`: CSV = isi layar, disclaimer baris 1, NPWP termasking di layar, CSV injection di-escape.
- **Regresi:** seluruh suite lama hijau. `JournalDrafterTest` baris 88–94 (`tax` 110.000 → 1402) tetap hijau untuk klien PKP; factory klien test diset `is_pkp = true` **atau** default perilaku "profil kosong = perlakuan lama (PKP)" — pilih salah satu di Q2.

Verifikasi: `php artisan test` lalu `graphify update .`.

## 9. Rollout (TR-17)

1. Migration additive (TR-01..05). Tidak ada `migrate:fresh`.
2. `TaxMasterSeeder` — idempoten, **tanpa data demo**, aman di produksi (berbeda dari `DemoSeeder`); isi kode, tarif, aturan jatuh tempo dari BRD §7 dengan `verified_at = null`.
3. Template COA + tombol Sinkron COA per klien.
4. Gerbang alami: menu Pajak hanya aktif setelah profil pajak klien diisi; klien tanpa profil = perilaku drafter lama.
5. Pilot 1 klien PKP + 1 klien Final UMKM selama satu masa; bandingkan dengan rekap Excel manual; selisih Rp0 sebelum dibuka ke klien lain.
6. Konsultan mengisi kolom Status BRD §7 → admin set `verified_at` di tarif.
7. Fase 2 baru dimulai setelah item 5–6 lulus.

## 10. Pertanyaan teknis terbuka

| # | Pertanyaan | Default sementara |
|---|---|---|
| Q1 | Tanggal acuan tarif & masa: tanggal faktur, tanggal posting, atau tanggal bayar (PPh terutang saat bayar/akrual)? | Tanggal dokumen (= tanggal jurnal). |
| Q2 | Klien tanpa profil: perlakukan sebagai PKP (perilaku lama) atau non-PKP? | PKP (tidak mengubah data lama); banner "Profil pajak belum diisi". |
| Q3 | Jurnal manual (`JournalController`) perlu panel pajak di MVP? | Tidak; FR-18 menandai jurnal manual yang menyentuh akun pajak tanpa entri. |
| Q4 | PPN Masukan yang dikreditkan di masa lain (pengkreditan mundur/maju) | Dikreditkan di masa faktur; override per entri = Fase 2. |
| Q5 | Mutasi bank yang berisi setoran pajak (kode billing) — otomatis dipetakan ke Utang PPh? | Tidak; AI cukup menyarankan akun utang pajak lewat COA. |
| Q6 | Pembulatan PPN per faktur vs per item di Coretax | Per dokumen (`floor`), dikonfirmasi BRD V4. |
| Q7 | Perlu `phpoffice/phpspreadsheet` untuk template Coretax? | Tidak sampai V22 terverifikasi. |
| Q8 | Rotasi `APP_KEY` dengan kolom terenkripsi | Pakai `APP_PREVIOUS_KEYS` Laravel; dokumentasikan di deploy. |

## 11. Matriks traceability

| BR | US | FR | TR |
|---|---|---|---|
| BR-01 Pajak terpecah & akun benar | US-03, US-05, US-07 | FR-06, FR-07, FR-09, FR-10, FR-12 | TR-02, TR-06, TR-08, TR-09, TR-11 |
| BR-02 Ikut profil klien | US-01, US-06 | FR-01, FR-02, FR-03, FR-11 | TR-01, TR-09 |
| BR-03 Hitung ulang, tanpa tambal diam | US-04 | FR-07, FR-08, FR-14 | TR-07, TR-08, TR-10, TR-11 |
| BR-04 Rekap satu sumber angka | US-08, US-09, US-10, US-14 | FR-13, FR-15, FR-16, FR-17, FR-18 | TR-03, TR-10, TR-12 |
| BR-05 Tidak telat | US-11, US-15, US-17 | FR-20, FR-21 | TR-04, TR-12, TR-13 |
| BR-06 Bawa ke Coretax | US-12, US-16 | FR-19 | TR-14 |
| BR-07 Tarif data-driven | US-02 | FR-04, FR-05 | TR-02, TR-07, TR-17 |
| BR-08 Audit | US-13 | FR-22 | TR-03 (snapshot), TR-05 |
| BR-09 Paket jasa | US-08..12 | FR-15..19 | TR-12..14 |
| (lintas) Isolasi, akses, keamanan | US-01, US-02, US-13 | FR-23, FR-24, FR-25, FR-26 | TR-10, TR-13, TR-15 |
| (lintas) Kualitas | semua | semua AC | TR-16 |
