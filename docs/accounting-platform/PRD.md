# Product Requirements Document — Platform Akuntansi Jasa & Dagang

**Status:** Draft produk/MVP  
**Input:** [Handoff Coding Agent](../Handoff_Accounting_Platform_Coding_Agent%20%281%29.md) · [BRD](BRD.md) · [FRD](FRD.md) · [Design Brief](DESIGN_BRIEF.md)

## 1. Visi

Membantu usaha jasa dan dagang mencatat transaksi satu kali dan memahami saldo serta laporan yang konsisten, dapat direkonsiliasi, dan terlacak ke sumbernya.

Nama produk, hubungan ke Master Pajak, harga, segmen rinci, kanal distribusi, target pengguna, hosting, dan strategi komersial belum diputuskan.

## 2. Persona

- **Owner usaha:** ingin melihat keadaan usaha dan menjaga akses, periode, serta laporan.
- **Finance/bookkeeper:** perlu alur transaksi efisien, alokasi pembayaran, rekonsiliasi, dan koreksi berjejak.
- **Staf operasional:** perlu membuat draft yang tervalidasi tanpa mengubah ledger secara tidak sengaja.
- **Reviewer akuntansi:** perlu memeriksa kebijakan dan menelusuri laporan sebelum pilot/produksi.

## 3. Prinsip produk

1. Angka yang benar dan bisa ditelusuri lebih penting daripada banyaknya fitur.
2. Draft aman untuk disiapkan; posting eksplisit dan terkonfirmasi.
3. Transaksi posted immutable; koreksi selalu punya reversal dan alasan.
4. Satu istilah untuk satu konsep; Bahasa Indonesia sebagai bahasa antarmuka MVP.
5. Tidak menyatakan setara Accurate; bangun identitas dan desain produk sendiri.
6. Tanpa AI untuk perhitungan akuntansi otoritatif pada MVP.

## 4. Fitur MVP dan prioritas

| Prioritas | Hasil pengguna | Rujukan FRD |
|---|---|---|
| P0 | Setup perusahaan, COA, periode, master, saldo awal terrekonsiliasi | FR-01, FR-02 |
| P0 | Posting kas/bank dan jurnal manual tanpa jurnal tidak seimbang | FR-03 |
| P0 | Penjualan jasa/dagang, invoice, pembayaran, AR | FR-04 |
| P0 | Pembelian jasa/dagang, bill, pembayaran, AP | FR-05 |
| P0 | Stok satu gudang, moving average, HPP dan kartu stok | FR-06 |
| P0 | Reversal, audit, dan lock periode | FR-07, FR-10 |
| P0 | Laporan keuangan, aging, filter historis, drill-down | FR-08 |
| P1 | Preview import saldo awal/master dan ekspor PDF/Excel | FR-09 |

Fitur P1 tetap bagian baseline M5 handoff, tetapi demo pilot jasa terbatas dapat dimulai di M3 setelah gate akuntansi dipenuhi.

## 5. Roadmap berbasis hasil

| Tahap | Hasil |
|---|---|
| M0 Audit & desain | Gap repository, keputusan stack, model data/ERD, aturan posting dan test plan. |
| M1 Fondasi | Auth, isolasi perusahaan, COA, periode, ledger, saldo awal valid. |
| M2 Kas/bank | Modal, biaya, transfer, jurnal manual, lock dan reversal. |
| M3 Jasa | Invoice/bill jasa, pembayaran, AR/AP dan laporan dasar; kandidat pilot terbatas. |
| M4 Dagang | Item/gudang, moving average, transaksi stok, HPP dan concurrency. |
| M5 Operasional | Impor, ekspor, audit UI, fixture penerimaan dan drill-down lengkap. |
| M6 Kesiapan pilot | Review akuntansi, akses, restore test, panduan dan daftar batasan. |

Estimasi jadwal ditetapkan setelah audit repository dan kapasitas tim.

## 6. Kriteria keberhasilan

- Semua angka fixture FRD tepat, dengan jurnal seimbang dan saldo laporan/subledger/stok yang cocok.
- Pengguna dapat menelusuri angka laporan ke jurnal dan dokumen sumber.
- Kegagalan, retry, konkurensi, periode terkunci, dan isolasi perusahaan tidak menghasilkan dampak ganda/parsial atau kebocoran.
- Reviewer akuntansi menyetujui kebijakan dasar dan hasil pilot sebelum penggunaan produksi.

Belum ada sasaran numerik adopsi, retensi, performa, harga, SLA, atau biaya. Ukur setelah target pasar dan operasi ditetapkan.

## 7. Risiko dan mitigasi

| Risiko | Mitigasi produk |
|---|---|
| Asumsi akuntansi tidak cocok dengan kebijakan usaha | Validasi owner akuntansi; catat keputusan sebelum produksi. |
| Data antarperusahaan bocor | `company_id` dan izin server pada seluruh jalur serta tes isolasi. |
| Perubahan posted menghilangkan audit | Immutable posted; reversal dengan alasan. |
| Costing retrospektif mengubah laporan | Tolak mutasi tanggal mundur; revaluasi/replay di luar MVP. |
| Pengguna salah membaca saldo historis | Filter as-of dan aging memakai tanggal laporan, bukan status hari ini. |
| Referensi Accurate dianggap hasil audit | Catat edisi, langkah, input, bukti dan hal yang belum diketahui; tanpa klaim parity. |
