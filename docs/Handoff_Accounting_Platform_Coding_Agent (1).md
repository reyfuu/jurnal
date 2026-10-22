# Handoff Coding Agent — Platform Akuntansi untuk Jasa dan Dagang

Tanggal: 21 Oktober 2026
Pemilik kebutuhan: Freddie / AIpreneur
Status: handoff perencanaan; belum ada repository, akses Accurate, atau implementasi yang diperiksa.

## 1. Instruksi utama untuk agent

Bangun aplikasi akuntansi web milik sendiri dengan workflow pembukuan jasa dan dagang sebagai fondasi. Accurate adalah referensi perilaku produk, bukan sumber kode atau desain yang harus disalin. Jangan mengklaim kompatibilitas, kesetaraan fitur, atau hasil observasi Accurate yang belum benar-benar diperiksa.

Baca dokumen ini sampai selesai. Jika ada repository, baca AGENTS.md, struktur aplikasi, skema database, autentikasi, dan test yang tersedia sebelum mengubah kode. Pertahankan stack dan konvensi existing kecuali ada alasan konkret untuk menggantinya. Jangan membangun ulang aplikasi existing dari nol.

Output pertama agent: ringkasan kondisi repository, gap terhadap handoff, keputusan teknis beserta asumsi, dan rencana implementasi berurutan. Setelah itu implementasikan per milestone dengan bukti pengujian. Dokumen ini tidak mengotorisasi deployment produksi, migrasi destruktif, atau akses ke akun pihak lain.

## 2. Konteks dan batas keputusan

Percakapan membahas cara mempelajari aplikasi Windows dari lingkungan development Linux, kemudian menerapkannya pada Accurate. Tujuannya memahami input, validasi, workflow, jurnal, stok, dan laporan agar dapat membangun produk dengan fungsi serupa.

Yang diminta pengguna: handoff agar coding agent memahami pembahasan. Daftar MVP di bawah merupakan baseline rekomendasi dari percakapan, bukan hasil audit kebutuhan bisnis atau keputusan final seluruh detail teknis.

Belum ditentukan: nama produk final, repository, stack, hosting, jumlah pengguna, target performa, integrasi, edisi Accurate referensi, dan hubungan aplikasi ini dengan Master Pajak. Jangan otomatis menggabungkannya ke Master Pajak atau aplikasi AIpreneur existing.

## 3. Tujuan produk

Pengguna usaha jasa/dagang dapat mencatat transaksi sekali dan memperoleh saldo, piutang, utang, stok, serta laporan yang konsisten dan dapat ditelusuri kembali ke dokumen sumber.

Keberhasilan MVP bukan banyaknya halaman, melainkan workflow nyata yang lengkap dan angka yang benar. Antarmuka berbahasa Indonesia, berorientasi desktop browser, dengan tabel yang tetap dapat digunakan pada layar kecil.

## 4. Cakupan baseline

| Modul | MVP | Setelah MVP |
|---|---|---|
| Perusahaan | Profil, IDR, tahun buku, periode | Multi-currency, konsolidasi |
| Akses | Owner, finance, staf; otorisasi server | Approval bertingkat |
| Akun | Chart of accounts, kategori, saldo awal | Template industri lanjutan |
| Master | Pelanggan, pemasok, barang, jasa, satuan | Varian, serial, batch |
| Kas/bank | Penerimaan, pengeluaran, transfer | Bank feed otomatis |
| Penjualan | Invoice, pembayaran sebagian/penuh, piutang | Quotation, SO, delivery terpisah |
| Pembelian | Tagihan, pembayaran sebagian/penuh, utang | PO dan goods receipt terpisah |
| Buku besar | Jurnal otomatis/manual, reversal | Alokasi lanjutan |
| Persediaan | Satu gudang, mutasi, kartu stok, HPP | Multi-gudang, manufaktur |
| Laporan | Buku besar, neraca saldo, laba rugi, neraca, umur utang/piutang | Arus kas formal, konsolidasi |
| Kontrol | Draft/posting/reversal, audit, lock periode | Approval kompleks |
| Data | Impor saldo awal/master, ekspor PDF/Excel | Migrasi otomatis Accurate |
| Lanjutan | Tidak termasuk MVP | Retur parsial, rekonsiliasi bank, aset tetap, pajak, marketplace |

AI bukan dependensi MVP. Tidak ada integrasi DJP, tarif pajak produksi, payroll, manufaktur, maupun perpajakan otomatis yang diasumsikan tersedia.

## 5. Baseline keputusan akuntansi untuk implementasi awal

Keputusan berikut adalah asumsi kerja yang wajib dicatat di README/decision log dan divalidasi dengan penanggung jawab akuntansi sebelum penggunaan produksi:

- Basis akrual, double-entry, mata uang IDR.
- Satu perusahaan aktif per sesi; semua data tetap memiliki company_id untuk isolasi.
- Satu gudang per perusahaan pada MVP; satu satuan dasar per item.
- Persediaan perpetual dengan moving weighted average; bukan klaim tentang metode default Accurate.
- Invoice penjualan yang diposting sekaligus menjadi titik pengeluaran barang. Tagihan pembelian yang diposting sekaligus menjadi titik penerimaan barang. Tidak ada delivery/goods receipt independen di MVP.
- Jasa tidak membuat mutasi stok atau HPP persediaan.
- Stok negatif ditolak; penjualan melebihi stok tidak boleh diposting.
- Tolak mutasi stok bertanggal sebelum mutasi terakhir item/gudang. Revaluasi historis dan replay costing di luar MVP.
- Kuantitas dan biaya unit dapat memiliki pecahan. Gunakan NUMERIC/Decimal di database dan library decimal di aplikasi, bukan floating-point uang.
- Usulan presisi: quantity NUMERIC(20,6), unit cost NUMERIC(24,8), ledger amount NUMERIC(24,2). Definisikan rounding half-up; nilai ledger menjadi otoritas rekonsiliasi. Penjualan seluruh sisa stok menghabiskan nilai inventory tersisa agar tidak menyisakan residu pembulatan.
- Pajak produksi ditunda. Fixture penerimaan bebas pajak; jangan hardcode tarif pajak terkini.

## 6. Status dan koreksi

Pisahkan lifecycle dokumen dari status pembayaran:

- Lifecycle: DRAFT → POSTED → REVERSED. DRAFT juga dapat menjadi CANCELLED.
- Pembayaran invoice/bill: UNPAID, PARTIALLY_PAID, PAID; dihitung dari alokasi pembayaran posted yang belum dibalik.
- DRAFT tidak memengaruhi ledger, stok, AR, atau AP.
- POSTED tidak boleh diedit atau dihapus langsung. Koreksi dilakukan melalui reversal dan dokumen pengganti dengan referensi silang serta alasan.
- Reversal kas/jurnal membalik entry sumber secara atomik; reversal hanya dapat dilakukan sekali dan dalam periode terbuka.
- Invoice/bill yang punya pembayaran aktif harus membalik pembayaran/alokasi terkait terlebih dahulu.
- Untuk stok, MVP hanya mengizinkan reversal transaksi terbaru item/gudang yang tidak memiliki pergerakan berikutnya. Selain itu blokir dengan alasan yang jelas; jangan mengarang costing koreksi.
- Retur barang sebagian adalah fitur terpisah setelah MVP, jangan menyamarkannya sebagai reversal penuh.
- Periode terkunci menolak posting, reversal, dan saldo awal pada periode itu. Pembukaan kembali hanya owner, dengan alasan dan audit.

## 7. Aturan posting wajib

| Peristiwa | Debit | Kredit |
|---|---|---|
| Setoran modal | Bank | Modal |
| Tagihan pembelian barang | Persediaan | Utang usaha |
| Bayar pemasok | Utang usaha | Bank |
| Invoice jasa/barang kredit | Piutang usaha | Pendapatan |
| Pengeluaran barang jual | HPP | Persediaan |
| Terima pembayaran pelanggan | Bank | Piutang usaha |
| Biaya dibayar bank | Beban terkait | Bank |
| Transfer antar rekening | Bank tujuan | Bank sumber |

Untuk pembelian yang langsung lunas, posting bill dan pembayaran terkait dalam satu transaksi database; jangan sekaligus membuat jurnal persediaan-kas tambahan yang menggandakan dampak.

Setiap posting harus memenuhi:

1. Total debit = total kredit pada presisi ledger.
2. Semua akun, pihak, item, gudang, dokumen, dan pembayaran milik perusahaan yang sama.
3. Akun aktif dan sesuai mapping; nomor dokumen unik per perusahaan/jenis dokumen.
4. Tidak ada posting tanpa baris, nilai tidak valid, alokasi melebihi outstanding, atau saldo awal ganda.
5. Dokumen, jurnal, mutasi stok, alokasi pembayaran, dan audit event commit/rollback bersama.
6. Idempotency key + unique constraint mencegah posting ganda saat retry/double-click.
7. Lock dokumen dan item terkait saat posting; dua transaksi serentak tidak boleh menjual stok yang sama melebihi tersedia.
8. Ledger posted menjadi sumber laporan keuangan. Cache/proyeksi saldo harus dapat dibangun ulang dan direkonsiliasi.

## 8. Data model konseptual

Sesuaikan dengan stack existing; nama tabel berikut bukan kewajiban literal.

- companies, users, company_memberships, roles/permissions.
- fiscal_periods, accounts, posting_account_mappings.
- parties dengan jenis customer/supplier/both.
- items dengan jenis inventory/service, units, warehouses.
- sales_invoices, sales_invoice_lines.
- purchase_bills, purchase_bill_lines.
- payments, payment_allocations; receipt/disbursement dibedakan eksplisit.
- journal_entries, journal_lines; sumber dokumen, tanggal, currency, reversal_of.
- stock_movements; quantity_delta, value_delta, unit_cost_snapshot, source, posting sequence.
- opening_balance_batches dan detail GL/AR/AP/stock.
- audit_events, idempotency_records, import_batches/import_errors.

Gunakan foreign key, uniqueness, dan constraint database selain validasi aplikasi. Saldo awal piutang/utang harus memiliki rincian pihak/outstanding; saldo awal stok harus cocok dengan akun kontrol persediaan. Hindari menghitung saldo awal dua kali saat mengimpor GL dan subledger.

## 9. Workflow dan layar minimum

1. Setup perusahaan → akun/mapping → periode → master → wizard saldo awal dan rekonsiliasinya.
2. Buat draft invoice → validasi → posting → lihat jurnal/stok → cetak → terima pembayaran.
3. Buat draft bill → posting → lihat penerimaan stok/utang → bayar pemasok.
4. Kas/bank: biaya, pemasukan non-penjualan, transfer; detail jurnal terkait.
5. Jurnal manual: debit/kredit dengan preview balance; pembatasan akun kontrol agar subledger tidak rusak.
6. Kartu pelanggan/pemasok: invoice/bill, pembayaran, outstanding, umur berdasarkan due date.
7. Persediaan: stok, nilai, riwayat mutasi dan link sumber.
8. Laporan: filter periode/as-of date, drill-down ke jurnal lalu dokumen; PDF dan Excel konsisten dengan filter layar.
9. Audit dan periode: siapa, kapan, tindakan, alasan, dokumen; lock/reopen periode.

Semua layar wajib punya empty/loading/error state, validasi field, pencarian, pagination server, filter, dan konfirmasi untuk posting/reversal. Tombol tersembunyi tidak menggantikan otorisasi server.

## 10. Observasi Accurate bila akses diberikan

Tidak ada observasi langsung Accurate yang dilakukan dalam handoff ini. Dokumentasi publik rujukan: https://help.accurate.id/product/fitur-aol/ dan https://accurate.id/Fitur/ . Fitur Online dan Desktop tidak boleh dianggap identik.

Jika memakai Desktop, jalankan aplikasi berlisensi di Windows/Windows VM/mesin remote. Coding boleh tetap di Linux. Gunakan data dummy perusahaan uji. Agent Linux tidak otomatis dapat mengontrol aplikasi desktop Windows; perlukan penghubung kontrol Windows yang nyata atau rekaman dan output dari operator.

Catat setiap eksperimen sebagai: edisi/versi → modul → input → tindakan → validasi → status → jurnal → dampak stok/AR/AP → laporan → bukti ekspor/screenshot → hasil yang belum diketahui.

Pelajari perilaku yang terlihat dan output yang diizinkan. Implementasikan source code, identitas, dan desain sendiri. Jangan mengakses akun/data tanpa izin, menyalin binary/source proprietari, atau melewati lisensi. Jangan membuat migrasi database/file Accurate tanpa spesifikasi format dan contoh ekspor yang sah.

## 11. Fixture penerimaan utama

Perusahaan baru, saldo awal nol, tanpa pajak, semua transaksi dalam periode terbuka dengan urutan tanggal naik.

1. Setor modal ke bank Rp10.000.000.
2. Buat bill pembelian 10 unit A @Rp100.000; posting lalu bayar penuh dari bank.
3. Posting invoice kredit 2 unit A @Rp150.000.
4. Terima pembayaran pelanggan Rp200.000 dan alokasikan ke invoice.
5. Posting biaya operasional Rp50.000 dari bank.

| Hasil | Nilai |
|---|---:|
| Bank | 9.150.000 |
| Piutang | 100.000 |
| Stok A | 8 unit |
| Nilai persediaan | 800.000 |
| Utang | 0 |
| Pendapatan | 300.000 |
| HPP | 200.000 |
| Beban operasi | 50.000 |
| Laba | 50.000 |
| Total aset | 10.050.000 |
| Modal + laba | 10.050.000 |

Seluruh jurnal seimbang; kartu piutang dan GL piutang cocok; nilai stok dan GL inventory cocok; neraca seimbang. Periksa juga laporan sebelum tanggal pembayaran: piutang masih Rp300.000. Jangan memakai outstanding hari ini untuk laporan historis.

Fixture costing tambahan: beli 10 unit @100.000 lalu 10 unit @120.000; biaya rata-rata 110.000. Jual 2 unit → HPP 220.000, sisa 18 unit bernilai 1.980.000 (tanpa pajak/biaya lain).

## 12. Pengujian kritis

- Posting key yang sama dua kali menghasilkan satu dampak; failure di tengah proses tidak meninggalkan jurnal/stok parsial.
- Dua request bersamaan terhadap stok terakhir: hanya yang valid berhasil.
- Pembayaran lebih dari outstanding, stok negatif, tanggal stok mundur, nomor duplikat, akun nonaktif ditolak.
- Draft tidak muncul dalam saldo/laporan; posted tidak dapat dimutasi langsung.
- Reversal sekali membalik dampak; reversal kedua ditolak; dependency pembayaran/stok diperiksa.
- Period lock berlaku pada semua jalur termasuk impor/API, bukan UI saja.
- Pengguna perusahaan A tidak dapat membaca/menulis data B lewat ID, pencarian, ekspor, atau lampiran.
- Uji pecahan kuantitas, diskon per baris, rounding, dan penghabisan stok terakhir tanpa residu.
- Impor punya preview/validasi dan error per baris; batch invalid tidak terposting sebagian tanpa kebijakan eksplisit. Retry batch tidak menggandakan data.
- Buku besar, trial balance, neraca, laba rugi, AR/AP, dan stok dapat direkonsiliasi.
- Backup dapat direstore di lingkungan uji; jangan mengklaim backup berhasil hanya karena file terbentuk.

## 13. Urutan implementasi dan gate

| Milestone | Hasil yang harus dapat didemokan |
|---|---|
| M0: audit dan desain | Gap repository, keputusan stack, ERD, aturan posting, test plan |
| M1: fondasi | Auth, company isolation, COA, periode, ledger, saldo awal valid |
| M2: kas/bank | Modal, biaya, transfer, jurnal manual, lock/reversal |
| M3: jasa | Invoice/bill jasa, alokasi pembayaran, AR/AP, laporan dasar |
| M4: dagang | Item/gudang, moving average, purchase/sale stock, HPP dan concurrency |
| M5: operasional | Impor, PDF/Excel, audit UI, seluruh fixture dan drill-down |
| M6: kesiapan pilot | Review akuntansi, akses, restore test, daftar keterbatasan dan panduan |

MVP lengkap hanya setelah M5 lulus; M3 adalah pilot jasa terbatas. Jangan melabelinya setara Accurate. Estimasi waktu diberikan setelah audit repository dan kapasitas tim, bukan ditebak dari dokumen ini.

## 14. Arsitektur dan keamanan

Jika belum ada repository, usulkan modular monolith + database relasional bertransaksi; pilihan framework/ORM harus mempertimbangkan kemampuan tim. Jangan menambah microservices, vector database, atau AI agent untuk perhitungan dasar.

Pisahkan domain posting/costing/reporting dari UI. Semua jalur UI, impor, dan API menggunakan layanan posting yang sama. Kredensial melalui environment/secret manager; jangan commit secret. Validasi file upload, batasi ukuran, lindungi ekspor dan lampiran, dan gunakan autentikasi/otorisasi existing yang layak. Data keuangan tidak dikirim ke model AI secara default.

## 15. AI tahap berikutnya

OCR bukti transaksi → draft; rekomendasi akun; deteksi duplikat; penjelasan laporan berbasis data. Simpan sumber dan jejak persetujuan. AI tidak mengubah posted journal, menghitung saldo otoritatif, atau mengeksekusi posting tanpa kontrol pengguna. Jawaban analitik wajib menunjuk periode/data sumber dan mengikuti hak akses.

## 16. Hasil handoff balik dari coding agent

Serahkan source code dan migration, seed fixture, cara menjalankan, konfigurasi contoh tanpa secret, test yang dijalankan beserta hasil aktual, daftar fitur selesai/belum, keputusan/asumsi, serta keterbatasan. Jika pengujian tidak dapat dijalankan, tulis penyebabnya secara spesifik. Hindari klaim selesai hanya karena UI berhasil dibuka.

## 17. Prompt pembuka siap tempel

> Baca Handoff_Accounting_Platform_Coding_Agent.md sampai selesai. Tugasmu membangun platform akuntansi jasa dan dagang dengan implementasi sendiri, menggunakan workflow Accurate hanya sebagai referensi. Mulai dengan audit repository dan AGENTS.md; laporkan gap, asumsi, desain posting, serta rencana M0–M6. Pertahankan stack existing jika layak. Jika repository belum ada, usulkan stack sederhana dan dokumentasikan keputusan. Kerjakan bertahap berdasarkan dependensi; utamakan double-entry, Decimal, atomic posting, idempotency, company isolation, stock costing, period lock, dan audit trail. Gunakan fixture angka di dokumen sebagai acceptance test, bukan angka hardcoded pada UI. Jangan mengarang hasil observasi Accurate atau mengklaim parity. Jangan deploy produksi atau menjalankan migrasi destruktif tanpa otorisasi. Laporkan implementasi nyata, bukti pengujian, dan hal yang belum selesai.
