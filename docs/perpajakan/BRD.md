# BRD — Perpajakan Jurnal AI

Status: draft 13 Okt 2026 · Pemilik: Freddie · Lanjut ke: [PRD](PRD.md) · [FRD](FRD.md) · [TRD](TRD.md)

## 1. Masalah

Jurnal AI sudah bisa membaca invoice dan memposting jurnal, tapi **pajaknya diperlakukan sebagai satu angka gelondongan**:

| # | Kondisi kode hari ini | Akibat bisnis |
|---|---|---|
| P1 | `DocumentExtractor` hanya punya satu field `tax` (skalar). PPN, PPh 23, PPh 4(2) bercampur. | Bookkeeper harus memecah manual; rawan salah akun. |
| P2 | `JournalDrafter` selalu menaruh `tax` di sisi rincian (debit untuk pembelian) → `1402 PPN Masukan`, atau `2103 Hutang Pajak` untuk penjualan. | PPh yang **dipotong** (mengurangi pembayaran) malah ditambahkan ke pembayaran. Jurnal tetap balance, tapi **salah secara pajak** — dan diam. |
| P3 | Klien non-PKP tetap mendapat PPN di `1402 PPN Masukan`. | Non-PKP tidak bisa mengkreditkan PPN; seharusnya masuk harga perolehan/beban. Aset pajak fiktif di Neraca. |
| P4 | COA template hanya punya `1402 PPN Masukan`, `2103 Hutang Pajak` (satu ember), `7103 Beban Pajak`. Tidak ada PPN Keluaran, Utang PPh per jenis, atau PPh dibayar dimuka. | Rekap per jenis pajak mustahil tanpa membaca memo satu per satu. |
| P5 | `clients.npwp` hanya string bebas 30 karakter; tidak ada status PKP atau rezim PPh. | Sistem tidak tahu aturan mana yang berlaku untuk klien. |
| P6 | Tidak ada rekap masa, kalender jatuh tempo, atau ekspor. | Rekap pajak dikerjakan di Excel terpisah dari buku → dua sumber angka. |

## 2. Tujuan bisnis

| ID | Tujuan | Ukuran |
|---|---|---|
| BR-01 | Pajak di dokumen terpisah per jenis (PPN / PPh 21 / 22 / 23 / 4(2) / Final UMKM) dan masuk akun yang benar. | ≥ 95% invoice ber-PPN di uji manual masuk akun benar tanpa koreksi akun. |
| BR-02 | Perlakuan pajak mengikuti profil klien (PKP/non-PKP, rezim PPh). | 0 jurnal klien non-PKP ke PPN Masukan (cek otomatis). |
| BR-03 | Angka pajak tidak pernah dipercaya mentah dari AI; selisih dilaporkan, bukan ditambal. | 100% baris pajak punya cek DPP × tarif; selisih > toleransi = warning terlihat. |
| BR-04 | Rekap pajak bulanan per klien langsung dari buku (satu sumber angka). | Rekap masa = saldo akun pajak di Neraca Saldo (selisih Rp0). |
| BR-05 | Bookkeeper tidak telat setor/lapor. | Kalender jatuh tempo per klien; 0 kewajiban terlewat di klien pilot selama 3 bulan. |
| BR-06 | Hasil rekap bisa dibawa ke Coretax DJP tanpa ketik ulang semuanya. | Ekspor CSV/Excel; format Coretax = *perlu diverifikasi* sebelum diklaim. |
| BR-07 | Tarif & aturan berubah tanpa deploy kode. | Semua tarif di tabel ber-tanggal berlaku; 0 tarif hardcoded (cek review). |
| BR-08 | Jejak audit: siapa mengubah apa, kapan, berdasarkan tarif versi mana. | Tiap baris pajak menyimpan snapshot tarif + user. |
| BR-09 | Nilai jual jasa pembukuan naik: "pembukuan + rekap pajak" sebagai satu paket. | Dipakai di ≥ 2 klien eksternal setelah MVP. |

## 3. Penerima manfaat

| Pihak | Manfaat |
|---|---|
| Bookkeeper jasa pembukuan (operator utama) | Pecah pajak otomatis, rekap masa sekali klik, kalender semua klien di satu tempat. |
| Klien UMKM (pemilik usaha) | Rekap PPh Final 0,5% dari omzet & pengingat setor; laporan yang bisa diserahkan ke konsultan. |
| SKINKU internal (klien `internal`) | Sama seperti klien lain. **Catatan:** keputusan A/B di HANDOFF §2 belum diambil — jangan bangun integrasi khusus portal B2B. |
| Konsultan pajak mitra | Menerima rekap terstruktur + bukti dokumen, bukan tumpukan foto. |

## 4. Scope

**In (MVP):** profil pajak klien · master kode & tarif ber-tanggal · ekstraksi pajak per jenis + hitung ulang di PHP · jurnal otomatis ke akun pajak yang benar (lewat layar review) · rekap PPN / PPh potput / PPh Final UMKM per masa · kalender jatuh tempo + status setor/lapor manual · ekspor CSV/Excel · audit trail.

**Out (MVP):** submit langsung ke Coretax DJP (API/H2H) · membuat Faktur Pajak / Bukti Potong resmi · hitung PPh 21 TER dari data gaji · SPT Tahunan · PPh Badan & rekonsiliasi fiskal · PPnBM · bea cukai/impor · nasihat pajak.

## 5. Risiko

| ID | Risiko | Dampak | Mitigasi |
|---|---|---|---|
| R1 | **Tanggung jawab hukum:** user menganggap angka aplikasi = kepatuhan pajak. | Sanksi/denda ke klien; reputasi & potensi klaim ke penyedia jasa. | Disclaimer tetap di setiap layar Pajak & ekspor: "Alat bantu pencatatan. Bukan nasihat pajak. Verifikasi dengan konsultan pajak." Persetujuan ToS jasa pembukuan mencantumkan batas tanggung jawab (perlu legal review). |
| R2 | AI salah baca / salah klasifikasi pajak. | Jurnal & rekap salah. | AI **tidak pernah** posting/lapor. Semua angka dihitung ulang di PHP, selisih = warning di baris. Posting hanya lewat layar review. |
| R3 | Regulasi berubah (tarif, DPP nilai lain, batas omzet, jatuh tempo). | Hitungan basi. | Tarif data-driven ber-tanggal; tabel §7 wajib direview konsultan sebelum go-live & tiap awal tahun. |
| R4 | Format impor Coretax berubah / tidak cocok. | Ekspor ditolak DJP. | MVP klaim "CSV rekap", bukan "file impor Coretax". Template Coretax hanya setelah diuji di akun nyata. |
| R5 | Data pribadi (NIK sebagai NPWP) bocor. | Pelanggaran UU PDP (UU 27/2022). | Enkripsi kolom, masking di UI, tidak dikirim ke AI kecuali ada di dokumen itu sendiri. |
| R6 | Semua user melihat semua klien (HANDOFF §4). | Staf melihat data pajak klien yang bukan tugasnya. | Dicatat sebagai keterbatasan; multi-user per klien di luar scope tapi jadi syarat sebelum >1 bookkeeper eksternal. |
| R7 | Perubahan drafter merusak invariant (balance, isolasi, anti-dobel, all-or-nothing). | Buku rusak. | Semua jurnal tetap lewat `JournalPoster`; test lama wajib tetap hijau. |

## 6. Asumsi

- A1. Pengguna adalah bookkeeper yang paham dasar pajak; aplikasi tidak mengajari pajak.
- A2. Klien mayoritas UMKM: orang pribadi atau badan kecil, banyak yang non-PKP dan memakai PPh Final 0,5%.
- A3. Mata uang hanya IDR; nilai pajak dalam rupiah penuh (tanpa sen).
- A4. Faktur pajak & bukti potong resmi tetap dibuat user di Coretax; aplikasi hanya mencatat nomornya.
- A5. Satu klien = satu wajib pajak (satu NPWP). Cabang/NITKU dicatat sebagai teks, tidak dimodelkan.
- A6. Periode pajak = bulan kalender (`journals.period` YYYY-MM), sejalan dengan masa pajak.

## 7. Angka regulasi — **Perlu diverifikasi konsultan pajak**

Asumsi per **13 Oktober 2026**. Semua angka di bawah menjadi **data awal (seed)** tabel tarif, bukan konstanta kode. Kolom "Status" diisi konsultan sebelum go-live.

| # | Item | Asumsi | Dasar (perkiraan) | Status |
|---|---|---|---|---|
| V1 | Tarif PPN umum | 12% sejak 1 Jan 2025 | UU HPP (UU 7/2021) | ☐ |
| V2 | DPP barang/jasa non-mewah | DPP nilai lain = 11/12 × harga jual → PPN efektif 11% | PMK 131/2024 | ☐ |
| V3 | Barang mewah (PPnBM) | PPN 12% × harga jual penuh | PMK 131/2024 | ☐ |
| V4 | Pembulatan PPN/PPh | Dibulatkan ke bawah ke rupiah penuh per dokumen | Praktik e-Faktur/Coretax | ☐ |
| V5 | PPh Final UMKM | 0,5% × omzet bruto bulanan | PP 55/2022 (ubah PP 23/2018) | ☐ |
| V6 | Omzet tidak kena pajak (OP UMKM) | Rp500 juta pertama per tahun pajak | PP 55/2022 | ☐ |
| V7 | Batas omzet rezim Final UMKM | ≤ Rp4,8 miliar per tahun | PP 55/2022 | ☐ |
| V8 | Jangka waktu Final UMKM | OP 7 th; CV/firma/koperasi 4 th; PT 3 th — **status perpanjangan untuk OP setelah 2024/2025 belum pasti** | PP 55/2022 + kebijakan lanjutan | ☐ |
| V9 | PPh 23 jasa | 2% × bruto; 4% (100% lebih tinggi) jika tanpa NPWP | UU PPh ps. 23 | ☐ |
| V10 | PPh 23 dividen/bunga/royalti/hadiah | 15% × bruto | UU PPh ps. 23 | ☐ |
| V11 | PPh 4(2) sewa tanah/bangunan | 10% × bruto | PP 34/2017 | ☐ |
| V12 | PPh 4(2) jasa konstruksi | Berjenjang 1,75%–6% per kualifikasi | PP 9/2022 | ☐ |
| V13 | PPh 22 | Beragam per objek (mis. impor 2,5%/7,5%; tanpa NPWP lebih tinggi) | PMK 34/2017 dst. | ☐ |
| V14 | PPh 21 | Tarif Efektif Rata-rata (TER) kategori A/B/C per bulan; Desember hitung ulang pasal 17 | PP 58/2023, PMK 168/2023 | ☐ |
| V15 | NIK sebagai NPWP | OP penduduk: NIK 16 digit = NPWP; badan: NPWP 16 digit; cabang: NITKU 22 digit | PMK 112/2022 | ☐ |
| V16 | Sistem administrasi | Coretax DJP berlaku sejak 1 Jan 2025 (faktur, bupot, SPT) | PMK 81/2024 | ☐ |
| V17 | Batas setor PPh masa (potput & Final UMKM) | Tanggal 15 bulan berikutnya | PMK 81/2024 | ☐ |
| V18 | Batas lapor SPT Masa PPh | Tanggal 20 bulan berikutnya; Final UMKM yang sudah setor dianggap lapor | PMK 81/2024 | ☐ |
| V19 | Batas setor & lapor SPT Masa PPN | Akhir bulan berikutnya (setor sebelum lapor) | PMK 81/2024 | ☐ |
| V20 | Jatuh tempo pada hari libur | Bergeser ke hari kerja berikutnya | PMK 81/2024 | ☐ |
| V21 | Batas wajib PKP | Omzet > Rp4,8 miliar per tahun | PMK 197/2013 | ☐ |
| V22 | Format impor Coretax (faktur/bupot) | XML dengan template konverter Excel dari DJP | Situs DJP | ☐ |

## 8. Stakeholder

| Peran | Siapa | Kepentingan |
|---|---|---|
| Pemilik produk & keputusan | Freddie | Scope, harga paket jasa, risiko hukum. |
| Pengguna utama | Bookkeeper (admin/staff) | Kecepatan & akurasi review. |
| Validator regulasi | Konsultan pajak mitra (belum ditunjuk) | Isi kolom Status di §7, review ekspor. |
| Klien | UMKM eksternal + SKINKU internal | Kepatuhan, tidak telat setor. |
| Pengembang | Agent/dev berikutnya | Invariant di AGENTS.md tetap utuh. |
| Legal | (belum ditunjuk) | Disclaimer & batas tanggung jawab. |
