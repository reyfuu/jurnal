# Functional Requirements Document — Platform Akuntansi Jasa & Dagang

**Status:** Draft baseline untuk validasi proses akuntansi  
**Input:** [Handoff Coding Agent](../Handoff_Accounting_Platform_Coding_Agent%20%281%29.md) · [BRD](BRD.md) · [TRD](TRD.md) · [PRD](PRD.md)

## 1. Aturan lintas fungsi

- Mata uang IDR; basis akrual; double-entry.
- Lifecycle dokumen: `DRAFT → POSTED → REVERSED`; draft juga boleh `CANCELLED`. Status pembayaran terpisah: `UNPAID`, `PARTIALLY_PAID`, `PAID`, dihitung dari alokasi pembayaran posted yang belum dibalik.
- Draft tidak mengubah GL, stok, AR, atau AP. Posted tidak boleh diedit/dihapus; koreksi menggunakan reversal dengan referensi silang dan alasan.
- Posting memvalidasi perusahaan, akun aktif, periode terbuka, nomor unik, baris nonkosong, jumlah/alokasi valid, dan saldo stok. Semua dampak tersimpan atau batal bersama.
- Semua layar memiliki loading, empty, error, validasi yang jelas, pencarian, pagination server, filter, serta konfirmasi untuk posting/reversal.

## 2. Pengguna dan izin

| Kemampuan | Owner | Finance | Staf |
|---|---:|---:|---:|
| Setup perusahaan, pengguna, periode | Ya | Tidak | Tidak |
| Kelola akun, master, saldo awal | Ya | Ya* | Tidak |
| Buat draft transaksi | Ya | Ya | Ya* |
| Posting/reversal | Ya | Ya* | Tidak* |
| Lihat laporan dan audit | Ya | Ya* | Sesuai izin* |
| Lock/reopen periode | Owner | Tidak | Tidak |

`*` Hak finance/staf perlu persetujuan pemilik produk. Izin tetap ditegakkan di server.

## 3. Persyaratan fungsional

### FR-01 — Setup perusahaan dan periode

Owner membuat profil perusahaan, menetapkan IDR/tahun buku, membuat periode, dan mengatur COA serta mapping akun posting. Perusahaan baru memulai dari saldo nol atau menjalankan wizard saldo awal yang mengelompokkan GL, AR/AP terperinci, dan stok tanpa penghitungan ganda.

**Penerimaan:** Periode terbuka menerima transaksi; periode terkunci menolak posting/reversal/saldo awal lewat UI, API, dan impor. Hanya owner dapat membuka kembali periode dan wajib memberi alasan yang tercatat.

### FR-02 — Master

Pengguna berizin mengelola COA, pelanggan/pemasok (pihak dapat berjenis customer, supplier, atau keduanya), item inventory/service, satuan dasar, dan satu gudang. Nomor akun dan identitas master mengikuti uniqueness perusahaan.

**Penerimaan:** Item jasa tidak memicu mutasi stok/HPP persediaan. Referensi master dari perusahaan lain selalu ditolak.

### FR-03 — Kas/bank dan jurnal manual

Pengguna membuat draft setoran modal, penerimaan non-penjualan, pengeluaran, transfer, dan jurnal manual. Jurnal manual menampilkan preview debit/kredit seimbang; akun kontrol yang dapat merusak subledger dibatasi.

**Penerimaan:** Posting hanya berhasil jika debit=kredit dan periode terbuka. Transfer mendebit bank tujuan dan mengkredit bank sumber dalam satu posting.

### FR-04 — Penjualan, invoice, dan piutang

Pengguna membuat draft invoice jasa/barang, memvalidasi pihak, tanggal, akun, baris, harga, jatuh tempo, dan stok; lalu mem-posting, mencetak, dan mencatat pembayaran parsial/penuh dengan alokasi. Invoice barang yang diposting sekaligus mengeluarkan stok dan menghitung HPP.

**Penerimaan:** Invoice kredit mem-posting Piutang/Pendapatan. Pembayaran pelanggan mendebit Bank dan mengkredit Piutang. Alokasi tidak boleh melebihi outstanding. Status pembayaran dihitung dari alokasi posted yang belum dibalik.

### FR-05 — Pembelian, bill, dan utang

Pengguna membuat draft bill jasa/barang, memvalidasi pemasok, tanggal, akun, baris, jatuh tempo, lalu mem-posting serta membuat pembayaran parsial/penuh. Bill barang yang diposting sekaligus menerima stok dan memperbarui moving weighted average. Bill langsung lunas diproses atomik bersama pembayarannya.

**Penerimaan:** Bill barang mendebit Persediaan dan mengkredit Utang; pembayaran pemasok mendebit Utang dan mengkredit Bank. Tidak ada jurnal ganda pada bill langsung lunas.

### FR-06 — Persediaan

Pengguna berizin melihat kuantitas dan nilai per item, kartu mutasi, unit cost snapshot, dan tautan ke dokumen sumber. MVP memakai satu gudang serta satuan dasar per item.

**Penerimaan:** Moving weighted average memakai aritmetika Decimal; kuantitas pecahan didukung; stok negatif ditolak. Mutasi bertanggal sebelum mutasi terakhir item/gudang ditolak. Penjualan seluruh stok tersisa menghabiskan nilai tersisa tanpa residu.

### FR-07 — Koreksi dan reversal

Pengguna berizin dapat membalik dokumen posted sekali pada periode terbuka, dengan alasan dan referensi ke dokumen asal. Dokumen pengganti menunjuk dokumen yang dibalik. Pembayaran/alokasi aktif dibalik lebih dahulu. Untuk stok, reversal hanya tersedia bila transaksi adalah mutasi terbaru item/gudang dan tidak ada pergerakan berikutnya.

**Penerimaan:** Reversal kedua, edit/hapus langsung dokumen posted, reversal di periode terkunci, atau reversal stok dengan mutasi setelahnya ditolak dengan alasan yang dapat dipahami.

### FR-08 — Laporan dan drill-down

Pengguna melihat buku besar, neraca saldo, laba rugi, neraca, umur piutang, dan umur utang dengan filter periode/tanggal as-of. Dari baris laporan dapat ditelusuri ke jurnal dan dokumen asal. Saldo historis dihitung menurut tanggal laporan, bukan outstanding hari ini.

**Penerimaan:** Trial balance seimbang; AR/AP cocok dengan kontrol GL; nilai stok cocok dengan akun inventory; neraca seimbang; PDF/Excel mengikuti filter layar.

### FR-09 — Impor dan ekspor

Pengguna berizin mengunggah master atau saldo awal, melihat preview, error per baris, dan hasil validasi sebelum commit. Kebijakan batch harus eksplisit: batch invalid tidak boleh ter-post sebagian secara diam-diam. Retry tidak menggandakan data.

**Penerimaan:** File valid menghasilkan satu batch terlacak; file/row invalid dilaporkan dengan sebab; file berbahaya/besar/jenis salah ditolak; ekspor hanya memuat data perusahaan dan filter yang diizinkan.

### FR-10 — Audit

Audit mencatat aktor, waktu, perusahaan, tindakan, jenis/ID sumber, alasan untuk tindakan koreksi atau reopen, dan hasilnya. Draft, posting, pembayaran, reversal, import, lock/reopen harus dapat ditelusuri.

**Penerimaan:** Pengguna perusahaan A tidak dapat membaca/menulis data B lewat ID langsung, pencarian, laporan, ekspor, maupun lampiran.

## 4. Fixture penerimaan utama

Perusahaan baru, saldo awal nol, tanpa pajak, satu periode terbuka, transaksi berurutan tanggal:

1. Setor modal ke bank Rp10.000.000.
2. Bill 10 unit A @Rp100.000; posting dan bayar penuh.
3. Invoice kredit 2 unit A @Rp150.000.
4. Terima pembayaran pelanggan Rp200.000 dan alokasikan ke invoice.
5. Catat biaya operasi Rp50.000 dari bank.

| Saldo/hasil akhir | Nilai |
|---|---:|
| Bank | Rp9.150.000 |
| Piutang | Rp100.000 |
| Stok A | 8 unit |
| Nilai persediaan | Rp800.000 |
| Utang | Rp0 |
| Pendapatan | Rp300.000 |
| HPP | Rp200.000 |
| Beban operasi | Rp50.000 |
| Laba | Rp50.000 |
| Total aset | Rp10.050.000 |
| Modal + laba | Rp10.050.000 |

Semua jurnal seimbang, AR dan inventory cocok dengan GL, dan neraca seimbang. As-of sebelum pembayaran pelanggan, piutang harus Rp300.000.

## 5. Fixture costing dan kegagalan wajib

- Beli 10 unit @Rp100.000 lalu 10 unit @Rp120.000: moving average Rp110.000. Jual 2 unit: HPP Rp220.000, stok 18 unit bernilai Rp1.980.000.
- Posting key sama dua kali memberi satu dampak; kegagalan di tengah transaksi tidak meninggalkan data parsial.
- Dua request bersamaan atas stok terakhir: hanya transaksi yang valid dapat berhasil.
- Tolak pembayaran di atas outstanding, stok negatif, tanggal stok mundur, nomor duplikat, akun nonaktif, dokumen tanpa baris, periode terkunci, dan referensi perusahaan lain.
- Tolak edit posted, reversal kedua, reversal stok yang bukan mutasi terakhir, dan import retry yang menggandakan dampak.
