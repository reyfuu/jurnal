# FRD — Perpajakan Jurnal AI

Status: draft 13 Okt 2026 · Dari: [PRD](PRD.md) · Implementasi: [TRD](TRD.md) · Matriks traceability: [TRD §11](TRD.md#11-matriks-traceability)

Konvensi: **AC** = acceptance criteria (Given/When/Then). "Tarif berlaku" = baris `tax_rates` dengan `effective_from ≤ tanggal < effective_to` (atau `effective_to` null). Semua nilai pajak dalam **rupiah bulat (integer)**. Angka regulasi di contoh hanya ilustrasi dari seed [BRD §7](BRD.md#7-angka-regulasi--perlu-diverifikasi-konsultan-pajak).

---

## A. Profil pajak klien

### FR-01 Profil pajak klien — Must
Tiap klien punya: `taxpayer_type` (orang_pribadi | badan), `tax_id` (NPWP 16 digit / NIK), `nitku` (opsional, 22 digit), `is_pkp` (bool), `pkp_since` (tanggal, wajib jika PKP), `pph_regime` (final_umkm | normal), `regime_start_year`, `final_umkm_revenue_accounts` (daftar akun omzet, default semua akun subtype `sales`/`service`).
- **AC1** Given admin di form klien, When menyimpan `is_pkp = true` tanpa `pkp_since`, Then ditolak dengan pesan "Tanggal pengukuhan PKP wajib diisi".
- **AC2** Given klien baru tanpa profil pajak, When halaman Pajak dibuka, Then tampil empty state "Lengkapi profil pajak" dan rekap tidak dihitung.
- **AC3** Given staff, When membuka form profil pajak, Then hanya bisa melihat (read-only); simpan hanya admin.

### FR-02 Validasi & normalisasi NPWP/NIK — Must
- Input dibersihkan ke digit saja (titik, strip, spasi dibuang).
- 16 digit = valid. 15 digit = diterima dengan warning "Format NPWP lama — perbarui ke 16 digit". Panjang lain = ditolak.
- **AC1** Given input `09.254.294.3-407.000`, When disimpan, Then tersimpan `092542943407000` + warning format lama.
- **AC2** Given input 12 digit, When disimpan, Then ditolak.
- **AC3** Given NPWP tersimpan, When tampil di daftar/tabel, Then dimasking `3201••••••••0001` (4 awal + 4 akhir); tampil penuh hanya di form profil untuk admin.

### FR-03 Rezim berlaku per tanggal — Must
Sistem menentukan perlakuan per transaksi dari profil: PKP jika `is_pkp` dan `tanggal ≥ pkp_since`; Final UMKM jika `pph_regime = final_umkm` dan `tahun ≥ regime_start_year`.
- **AC1** Given klien PKP sejak 1 Jul 2026, When invoice pembelian bertanggal 20 Jun 2026 diproses, Then diperlakukan non-PKP (PPN ke biaya, FR-11).
- **AC2** Given perubahan profil, When disimpan, Then jurnal yang sudah posted **tidak** diubah; hanya draft berikutnya yang terpengaruh.

## B. Master kode & tarif

### FR-04 Master kode pajak — Must
Kode awal (seed, bisa ditambah admin): `PPN`, `PPN_MEWAH`, `PPH21`, `PPH22`, `PPH23_JASA`, `PPH23_MODAL` (dividen/bunga/royalti), `PPH4_2_SEWA`, `PPH4_2_KONSTRUKSI`, `PPH_FINAL_UMKM`. Tiap kode punya `kind` (vat | withholding | final), label, dan subtype akun sasaran per arah.
- **AC1** Given kode dipakai di catatan pajak posted, When admin mencoba menghapus, Then ditolak; hanya bisa dinonaktifkan.

### FR-05 Tarif ber-tanggal berlaku — Must
Tarif = `rate` (persen), `dpp_numerator/dpp_denominator` (mis. 11/12; default 1/1), `rate_no_tax_id` (opsional, tarif tanpa NPWP), `rounding` (floor | half_up), `effective_from`, `effective_to`, `source` (dasar hukum), `verified_at/by`.
- **AC1** Given tarif PPN 12% × 11/12 berlaku sejak 2025-01-01, When dihitung untuk invoice 2026-03-10 DPP harga jual Rp1.000.000, Then pajak hitung = Rp110.000.
- **AC2** Given admin menambah versi baru berlaku 2027-01-01, When disimpan, Then `effective_to` versi lama otomatis = 2026-12-31 dan tidak ada rentang tumpang tindih.
- **AC3** Given versi tarif sudah dipakai catatan pajak posted, When admin mencoba mengubah angkanya, Then ditolak — wajib membuat versi baru.
- **AC4** Given tidak ada tarif berlaku untuk tanggal transaksi, When draft disusun, Then baris pajak diberi warning "Tarif {kode} untuk {tanggal} belum ada" dan nilai hitung kosong; posting tetap boleh dengan nilai dari dokumen.
- **AC5** Given tarif `verified_at` kosong, When dipakai, Then panel menampilkan badge "Tarif belum diverifikasi konsultan".

## C. Ekstraksi & normalisasi

### FR-06 Ekstraksi pajak per jenis — Must
Hasil AI memuat `taxes[]`: `type`, `label` (teks asli di dokumen), `base` (DPP), `rate_percent`, `amount`, `withheld` (bool), `withheld_by` (us | counterparty), `document_number` (no. faktur/bukpot), `counterparty_tax_id`. Field `tax` skalar lama tetap diisi = jumlah pajak yang **ditambahkan** ke tagihan (kompatibilitas).
- **AC1** Given invoice "DPP 916.667 · PPN 12% 110.000", When dibaca, Then `taxes` berisi satu entri `type=ppn, base=916667, amount=110000`.
- **AC2** Given invoice jasa "Subtotal 5.000.000 · PPh 23 (2%) (100.000)", When dibaca, Then entri `type=pph23, withheld=true, amount=100000`, dan `tax` tidak memasukkan PPh 23.
- **AC3** Given AI mengirim `type` di luar daftar, When dinormalisasi, Then diubah ke `other` + warning "Jenis pajak tidak dikenali".

### FR-07 Normalisasi angka — Must
Semua `base`, `amount`, `rate_percent` dicuci `Rupiah::parse()` lalu dibulatkan ke rupiah integer; `amount` negatif/kurung dijadikan absolut dan arah ditentukan oleh `withheld`.
- **AC1** Given `amount: "(100.000)"`, When dinormalisasi, Then `amount = 100000`.
- **AC2** Given `rate_percent: "12%"` atau `0.12`, When dinormalisasi, Then dibaca 12 (nilai ≤ 1 dikali 100).
- **AC3** Given `counterparty_tax_id`, When dinormalisasi, Then dibersihkan seperti FR-02 dan **tidak** pernah ditambah oleh sistem.

### FR-08 Hitung ulang & toleransi (tidak ada koreksi diam-diam) — Must
Untuk tiap entri, sistem menghitung `computed = round_mode(base × rate × dpp_factor)` dengan tarif berlaku di tanggal dokumen.
- Selisih `|amount − computed| ≤ toleransi` (default Rp1 × jumlah item, nilai di config) → status **cocok**.
- Lebih → status **selisih** + warning di baris: "PPN di dokumen Rp110.500, hitung sistem Rp110.000 (selisih Rp500)". Nilai yang dipakai draft = **nilai dokumen**; tidak ada yang ditimpa.
- `base` kosong → diturunkan dari subtotal item, ditandai "DPP diturunkan" + confidence turun.
- **AC1** Given amount AI 110.000 dan computed 110.000, Then status cocok, tanpa warning.
- **AC2** Given amount AI 120.000 dan computed 110.000, Then warning tampil di baris pajak dan di ringkasan jumlah warning; draft memakai 120.000.
- **AC3** Given tidak ada `taxes` tapi `tax > 0` (dokumen lama), Then diperlakukan sebagai satu entri `ppn` + warning "Jenis pajak diasumsikan PPN".

## D. Jurnal

### FR-09 Pemetaan akun pajak — Must
Akun dicari di COA klien berdasarkan **subtype**, bukan kode hardcoded:

| Kode | Arah | Subtype akun | Template baru (TRD §2.4) |
|---|---|---|---|
| PPN | pembelian (PKP) | `tax_in` | 1402 PPN Masukan (sudah ada) |
| PPN | penjualan (PKP) | `vat_out` | 2106 PPN Keluaran |
| PPH21 / PPH23_* / PPH4_2_* | kita pemotong | `wht_payable_{kode}` | 2107 Utang PPh 21 · 2108 Utang PPh 23 · 2109 Utang PPh 4(2) |
| PPH23_* / PPH22 / PPH4_2_* | dipotong pihak lain | `prepaid_tax` | 1403 Pajak Dibayar Dimuka |
| PPH_FINAL_UMKM | akrual | `final_payable` | 2110 Utang PPh Final |
| PPH22 | kita pemungut (jarang) | — | fallback 2103 + warning |

- **AC1** Given akun subtype yang dicari tidak ada/aktif di COA klien, When draft disusun, Then dipakai `2103 Hutang Pajak` (atau akun `tax_out` pertama) + warning "Akun {subtype} tidak ada — tambahkan via Sinkron COA".
- **AC2** Given akun hasil pemetaan, Then akun itu selalu milik klien yang sama (isolasi; JournalPoster tetap menolak akun klien lain).

### FR-10 Aturan penyusunan draft — Must

| Skenario | Debit | Kredit |
|---|---|---|
| Beli, PKP, PPN | Rincian + PPN Masukan | Bayar/Hutang = rincian + PPN − PPh dipotong |
| Beli, PPh dipotong oleh kita | Rincian | Utang PPh xx + Bayar/Hutang (sisa) |
| Jual, PKP, PPN | Bayar/Piutang | Pendapatan + PPN Keluaran |
| Jual, PPh dipotong pelanggan | Bayar/Piutang (sisa) + Pajak Dibayar Dimuka | Pendapatan |

- **AC1** Given invoice jasa 5.000.000, PPN 550.000, PPh 23 dipotong 100.000, klien PKP, Then draft: D Beban 5.000.000, D PPN Masukan 550.000, K Utang PPh 23 100.000, K Bank 5.450.000; balance.
- **AC2** Draft selalu balance (sisi pembayaran = sisa) — invariant lama tetap berlaku.
- **AC3** Given semua test `JournalDrafterTest` lama (field `tax` saja), Then hasilnya tidak berubah untuk klien PKP.

### FR-11 PPN klien non-PKP — Must
- **AC1** Given klien non-PKP dan invoice pembelian ber-PPN 110.000 atas beban 1.000.000, Then draft: D Beban 1.110.000 (PPN dibebankan ke akun rincian, proporsional jika banyak item; sisa pembulatan ke item terbesar), tanpa baris PPN Masukan; info "Non-PKP — PPN digabung ke biaya".
- **AC2** Given klien non-PKP dan dokumen penjualan memuat PPN, Then warning keras "Klien non-PKP tidak boleh memungut PPN — periksa dokumen" dan tidak dibuat baris PPN Keluaran.

### FR-12 Edit di layar review — Must
User dapat mengubah jenis, arah (`withheld_by`), DPP, nilai, no. faktur/bukpot, NPWP lawan, atau menghapus entri pajak. Server menghitung ulang status cocok/selisih saat posting; nilai dari form = nilai yang masuk buku.
- **AC1** Given user mengubah nilai PPN, When halaman dimuat ulang setelah error validasi, Then nilai editan user dipertahankan (old input).

### FR-13 Posting atomik — Must
Catatan pajak (`tax_entries`) disimpan **dalam transaksi DB yang sama** dengan jurnalnya via `JournalPoster`.
- **AC1** Given satu dari tiga draft gagal validasi, Then tidak ada jurnal **maupun** catatan pajak yang tersimpan.
- **AC2** Given fingerprint sudah posted, Then draft dilewati dan tidak ada catatan pajak ganda.
- **AC3** Given jurnal di-void, Then catatan pajaknya otomatis keluar dari semua rekap (rekap hanya membaca jurnal `posted`).

### FR-14 Konsistensi catatan pajak vs baris jurnal — Must
Berlaku untuk entri yang dikreditkan/terutang (bukan PPN non-PKP yang digabung ke biaya).
- **AC1** Given catatan pajak PPN Masukan 110.000 untuk jurnal X, When posting, Then total baris jurnal X di akun pemetaannya harus = 110.000; jika tidak, posting ditolak dengan pesan "Nilai pajak di panel ≠ baris jurnal akun {akun}".

## E. Rekap & laporan

### FR-15 Rekap PPN masa — Must (klien PKP)
Isi: daftar PPN Keluaran & Masukan di masa (tanggal faktur, lawan, NPWP lawan termasking, no. faktur, DPP, PPN, tautan dokumen) + total + `Kurang/(Lebih) bayar = Keluaran − Masukan`.
- **AC1** Given 2 penjualan PPN 110.000 & 220.000 dan 1 pembelian PPN 55.000 di Sep 2026, Then kurang bayar = 275.000.
- **AC2** Given satu jurnal di masa itu di-void, Then rekap tidak memuatnya.
- **AC3** Given klien non-PKP, Then menu PPN tidak tampil dan route mengembalikan 404.

### FR-16 Rekap PPh potong/pungut — Must
Per kode PPh: daftar transaksi (tanggal, lawan, NPWP/NIK lawan, bruto, tarif, PPh, no. bukpot) + total. Dua bagian: "Kita potong (wajib setor)" dan "Dipotong pihak lain (kredit pajak)".
- **AC1** Given transaksi PPh 23 dipotong tanpa NPWP lawan, Then baris ditandai "NPWP lawan kosong".

### FR-17 Rekap PPh Final UMKM — Must (rezim final)
`Omzet bruto` = total kredit akun omzet (FR-01) pada jurnal posted di masa − debit akun yang sama (retur/koreksi). Untuk OP: `bagian bebas` = min(omzet masa, sisa batas bebas tahun berjalan) dengan batas dari tabel tarif. `PPh = round_mode((omzet − bagian bebas) × tarif)`.
- **AC1** Given OP, omzet Jan–Agu kumulatif Rp450 jt, omzet Sep Rp80 jt, batas Rp500 jt, tarif 0,5%, Then bagian bebas Sep = 50 jt, PPh Sep = 150.000.
- **AC2** Given badan, Then bagian bebas = 0.
- **AC3** Given omzet kumulatif melewati batas rezim (Rp4,8 M), Then banner "Omzet melewati batas rezim Final UMKM — konsultasikan perubahan rezim" (tidak mengubah apa pun otomatis).
- **AC4** Given tarif/batas berubah di tengah tahun, Then tiap masa memakai versi berlaku di hari pertama masa itu.

### FR-18 Cek silang rekap vs buku — Must
- **AC1** Given rekap PPN masa, Then total PPN Masukan & Keluaran sama dengan mutasi akun `tax_in` & `vat_out` di Neraca Saldo masa yang sama; jika beda, tampil baris status merah "Rekap ≠ buku: selisih Rp x (jurnal manual tanpa catatan pajak?)" + daftar jurnal penyebab.

### FR-19 Ekspor — Must (CSV) / Should (Excel Coretax)
- CSV UTF-8, pemisah `;`, angka tanpa pemisah ribuan, tanggal `YYYY-MM-DD`; baris pertama = disclaimer; nama berkas `pajak-{slug}-{kode}-{YYYY-MM}.csv`.
- **AC1** Given rekap PPN Sep 2026, When ekspor, Then isi CSV = baris yang tampil di layar, total sama.
- **AC2** Template Excel Coretax **ditandai "perlu diverifikasi" (BRD V22)** dan tidak dirilis sebelum diuji impor di akun Coretax nyata.

## F. Kalender & status

### FR-20 Kalender kewajiban — Must
Kewajiban per masa diturunkan dari profil + tabel aturan jatuh tempo (`tax_obligation_rules`, data-driven): mis. PKP → "Setor & lapor PPN"; ada transaksi PPh dipotong → "Setor & lapor PPh xx"; Final UMKM → "Setor PPh Final". Jatuh tempo di hari libur digeser ke hari kerja berikutnya memakai tabel `holidays` (diisi admin).
- **AC1** Given klien Final UMKM, masa Sep 2026, aturan setor tanggal 15 bulan berikut, Then kewajiban "Setor PPh Final UMKM — 15 Okt 2026".
- **AC2** Given jatuh tempo jatuh di hari Minggu/libur terdaftar, Then digeser ke hari kerja berikutnya.
- **AC3** Status: `belum` · `sudah setor` · `sudah lapor` · `nihil`. Merah hanya untuk "lewat jatuh tempo & belum".

### FR-21 Pencatatan setor/lapor — Must
Form: jumlah setor, tanggal, NTPN (16 karakter alfanumerik), kode billing (opsional), tanggal lapor, nomor BPE/NTTE, lampiran (opsional).
- **AC1** Given jumlah setor ≠ PPh rekap masa itu, Then warning (bukan blokir) "Setoran berbeda dari rekap Rp x".
- **AC2** Given status diubah, Then tercatat di audit trail (FR-22).
- **AC3** Sistem **tidak pernah** mengubah status menjadi "sudah setor/lapor" sendiri.

## G. Audit, hak akses, keamanan

### FR-22 Audit trail — Must
Dicatat append-only: perubahan profil pajak, tarif, aturan jatuh tempo, status setor/lapor, posting/void jurnal ber-pajak — user, waktu, klien, nilai sebelum/sesudah. Catatan pajak menyimpan snapshot tarif (`rate`, faktor DPP, `tax_rate_id`) dan `created_by`.
- **AC1** Given tarif diperbarui, When catatan pajak lama dibuka, Then tetap menampilkan tarif yang dipakai saat posting.
- **AC2** Halaman audit hanya untuk admin, terfilter per klien.

### FR-23 Hak akses — Must

| Aksi | admin | staff |
|---|---|---|
| Lihat rekap, kalender, ekspor | ✓ | ✓ |
| Review & posting dokumen ber-pajak | ✓ | ✓ |
| Catat status setor/lapor | ✓ | ✓ |
| Ubah profil pajak klien | ✓ | — |
| Kelola kode, tarif, aturan jatuh tempo, hari libur | ✓ | — |
| Lihat audit trail | ✓ | — |

### FR-24 Isolasi klien — Must
- **AC1** Given URL rekap/ekspor/status klien A diakses dengan parameter entri milik klien B, Then 404 (pola `assertOwned`).
- **AC2** Given tarif & kode = master global, Then tidak ada data klien di dalamnya; catatan pajak & status selalu ber-`client_id`.

### FR-25 Disclaimer — Must
- **AC1** Setiap halaman Pajak dan setiap ekspor memuat: "Alat bantu pencatatan, bukan nasihat pajak. Angka wajib diverifikasi konsultan pajak sebelum disetor/dilapor."

### FR-26 Perlindungan NPWP/NIK — Must
- **AC1** NPWP/NIK klien tidak dimasukkan ke prompt AI (prompt hanya memuat nama klien & COA, seperti sekarang).
- **AC2** Kolom NPWP/NIK disimpan terenkripsi; pencarian memakai hash.
