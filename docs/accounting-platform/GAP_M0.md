# M0 — Audit celah & keputusan (fitur ala Accurate di Master Pajak)

**Tanggal:** 22 Oktober 2026
**Keputusan pemilik:** fitur akuntansi jasa & dagang dibangun **di dalam Master Pajak** (bukan aplikasi terpisah). Milestone pertama: **pembayaran sebagian**.
**Rujukan:** [Handoff](../Handoff_Accounting_Platform_Coding_Agent%20%281%29.md) · [BRD](BRD.md) · [FRD](FRD.md) · [TRD](TRD.md)

Accurate hanya referensi perilaku. Tidak ada observasi langsung Accurate di dokumen ini, dan tidak ada klaim kesetaraan fitur.

## 1. Kondisi Master Pajak sebelum M1

Transaksi Master Pajak satu sisi (jenis + nominal + `Lunas`). Jurnal double-entry diturunkan ulang per bulan dari transaksi; tidak ada tabel jurnal. Data hidup di memori per usaha dan disimpan ulang utuh ke Postgres setiap perubahan.

| Kebutuhan handoff | Status | Catatan |
|---|---|---|
| Double-entry, uang `int64` rupiah | Ada | Jurnal turunan (`internal/coa/journal.go`), bukan jurnal tersimpan |
| Isolasi perusahaan | Ada | Satu user = satu usaha, kunci per usaha |
| Invoice penjualan | Ada (dokumen) | Draft/terkirim/lunas; baris invoice tanpa barang (tidak menggerakkan stok) |
| **Pembayaran sebagian & alokasi** | **Belum → M1** | Sebelumnya `Lunas` semua-atau-tidak |
| Tagihan pembelian (bill) | Belum | Utang = transaksi pembelian/biaya belum lunas |
| Umur piutang/utang dari jatuh tempo | Sebagian → **M1** | Sebelumnya dihitung dari tanggal transaksi |
| Saldo historis per tanggal | Ada | `PosisiPada`, `UmurSaldo ?per=` |
| Persediaan perpetual rata-rata bergerak | Ada (opsional) | Bawaan usaha baru FIFO; HPP di jurnal = penyesuaian akhir bulan |
| Tolak stok minus / tanggal mundur | Belum | Hanya peringatan |
| Posted immutable + reversal | Belum | Transaksi bisa diubah/hapus di bulan terbuka |
| Kunci periode | Ada | Tutup buku per bulan; buka kembali Owner tanpa alasan |
| Audit trail usaha | Sebagian | Hanya aksi tim Master Pajak & keputusan koreksi |
| **Idempotency key** | **Belum → M1 (+ Midtrans)** | |
| Neraca saldo, buku besar, jurnal umum | Ada | Neraca per akhir bulan saja |
| Transfer antar rekening | Belum | |

## 2. Yang dikerjakan M1

- Dokumen **Pembayaran** (piutang/utang): satu pembayaran, boleh sebagian, boleh ke beberapa tagihan pihak yang sama, ke rekening kas/bank pilihan.
- **Idempotency-Key** per dokumen: klik ganda / retry mengembalikan dokumen yang sama.
- **Batal** = reversal beralasan: sekali saja, hanya di bulan yang belum ditutup; dokumen tetap tercatat sebagai dibatalkan.
- Status lunas transaksi dan status invoice **diturunkan** dari cicilan. Transaksi, invoice, aset, dan grup catat yang sudah dibayar tidak bisa diubah nominal/tanggalnya atau dihapus sebelum pembayarannya dibatalkan.
- **Umur dari jatuh tempo** (tanpa jatuh tempo: tanggal + 30 hari, sama dengan dasbor), kelompok baru "Belum jatuh tempo", nilai = sisa per tanggal.
- **Jurnal**: tagihan yang dibayar setelah tanggal transaksi selalu lewat Piutang/Utang, walau masih sebulan (Dr Piutang di tanggal jual; Dr Kas/Bank, Cr Piutang di tanggal bayar), dan rekening kas/bank pilihan (`Lawan`) kini dipakai di jurnal per akun.
- Impor mutasi bank: transfer sebesar **sisa** tagihan dicocokkan sebagai pelunasan; transfer yang sudah dicatat sebagai cicilan dikenali sebagai dobel.

## 3. Keputusan (asumsi kerja, validasi dengan penanggung jawab akuntansi)

1. **Alokasi disimpan di transaksi** (`transactions.cicilan`), bukan tabel terpisah. Semua perhitungan hanya membaca cicilan transaksi; `pembayaran.alokasi` adalah isi dokumen (catatan). Alasan: transaksi tetap satu baris (kartu stok, aset, deteksi dobel tidak melihat "potongan"), dan tidak ada dua sumber yang bisa berbeda.
2. **Yang bisa dibayar lewat Pembayaran**: piutang/utang usaha (penjualan, pendapatan lain, pembelian, biaya, PPN) dan aset yang dibeli tempo. Modal, pinjaman, cicilan pinjaman, dan prive tetap lewat menu transaksinya.
3. **Lebih bayar ditolak** (uang muka pelanggan/pemasok di luar M1).
4. **Alokasi ke invoice** dibagi ke transaksi DPP lalu PPN-nya.
5. **Batal = cabut cicilan**, bukan jurnal pembalik tersimpan. Setara secara angka karena batal hanya boleh di bulan yang belum ditutup: tidak ada bulan tertutup yang berubah.
6. **Lunasi** (tombol lama) pada transaksi yang sudah dicicil = pembayaran sebesar sisanya.

## 4. Bukti pengujian

- `cmd/server/fixture_akuntansi_test.go`: fixture handoff §11 lewat API (modal 10 jt, beli 10 unit @100 rb, jual kredit 2 unit @150 rb, terima 200 rb, biaya 50 rb) → bank 9.150.000, piutang 100.000, stok 8 unit / 800.000, pendapatan 300.000, HPP 200.000, laba 50.000, aset = kewajiban + modal = 10.050.000; piutang per 9 Jan = 300.000; neraca saldo seimbang. Fixture costing: HPP 220.000, 18 unit = 1.980.000. Uji mutasi: tanpa langkah pembayaran tes gagal di bank, piutang, dan neraca saldo.
- `internal/finance/pembayaran_test.go`: posisi per tanggal, aturan tolak, copy-on-write, batal, umur dari jatuh tempo, invarian laporan berjalan dengan cicilan lintas bulan, peringatan kas harian, deteksi dobel.
- `internal/coa/cicilan_test.go`: jurnal cicilan sebulan lewat Piutang, rekening pilihan, saldo bawaan.
- `cmd/server/pembayaran_test.go`: API (idempoten, tolak, batal, tutup buku), invoice sebagian/lunas/batal.
- Round-trip Postgres (`TestPostgresSimpanMuatSemuaField`) mencakup `pembayaran` dan `transactions.cicilan`.
- Benchmark 10.000 transaksi: `JournalBerjalan`+`SaldoBawaan` ±2,5 ms (sebelumnya ±2,8 ms), `BuildLaporanBerjalan` ±10,5 ms (sebelumnya ±9,3 ms).

## 5. Batasan M1

- Pihak masih teks bebas (belum master pelanggan/pemasok berbasis ID).
- Belum ada uang muka / lebih bayar, nota kredit, retur.
- Invoice belum memuat barang, jadi penjualan lewat invoice tidak menggerakkan stok; fixture memakai transaksi Penjualan bertanda barang.
- Nomor bukti jurnal umum untuk pelunasan masih `BYR-<id transaksi>`; dokumen pembayaran bernomor `PB-<id>`.

## 6. Roadmap berikutnya

| Milestone | Isi |
|---|---|
| M2 Audit & reversal | Log audit untuk semua perubahan transaksi; buka tutup buku wajib alasan & tercatat; transaksi terposting tidak diedit langsung tetapi dibalik |
| M3 Kontrol stok | Tolak stok minus & tanggal mundur; HPP per penjualan (bukan penyesuaian akhir bulan); barang di baris invoice; bawaan rata-rata bergerak |
| M4 Kas/bank & master | Transfer antar rekening; master pelanggan/pemasok ber-ID; tagihan pembelian (bill) sebagai dokumen |
| M5 Laporan | Neraca per tanggal bebas; layar neraca saldo; drill-down laporan → jurnal → dokumen; ekspor umur piutang/utang |
| M6 Kesiapan pilot | Review akuntansi, uji restore backup, panduan & daftar batasan |
