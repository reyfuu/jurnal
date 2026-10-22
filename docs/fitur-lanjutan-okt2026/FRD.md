# Functional Requirements Document — Fitur lanjutan checklist tester (Okt 2026)

**Status:** Draft, 22 Okt 2026 · **Input:** [BRD](BRD.md) · [PRD](PRD.md) · **Desain teknis:** [TRD](TRD.md)

## 1. Aturan lintas fitur

- Semua uang dalam rupiah utuh (`int64`), tampil penuh dengan pemisah ribuan; negatif dalam kurung.
- Semua tanggal WIB, format simpan `YYYY-MM-DD`.
- Izin ditegakkan di server lewat izin menu (`guard`) yang ada. Kasir tidak melihat laporan keuangan.
- Periode yang sudah **tutup buku** menolak perubahan data yang memengaruhi periode itu
  (mengikuti aturan tutup buku yang berlaku sekarang).
- Setiap fitur punya status kosong, memuat, dan galat dengan pesan bahasa awam.

## 2. Persyaratan per fitur

### F-01 — Transaksi bawaan bulan berjalan (No 26)
**FR:** Saat menu Transaksi dibuka tanpa filter tersimpan, filter bulan = bulan berjalan (WIB).
Pilihan bulan lain dan "Semua bulan" tetap tersedia. Pilihan terakhir disimpan di `sessionStorage`.
Di atas tabel ada teks "Menampilkan Oktober 2026 · Lihat semua di Jurnal Umum".
**Penerimaan:** Buka Transaksi pada 22 Okt → hanya transaksi Okt 2026; pilih "Semua bulan" → semua tampil;
muat ulang halaman di tab yang sama → pilihan terakhir tetap.

### F-02 — Urutan akun saldo awal (No 29)
**FR:** Baris di dialog Saldo Awal diurutkan menaik berdasarkan kode akun. Baris baru tanpa kode
tetap di bawah sampai kodenya diisi; setelah kode diisi (blur) baris pindah ke posisinya.
Saat disimpan, urutan rincian = urutan kode.
**Penerimaan:** Tambah akun "1105 Bank BNI" → setelah kode diisi, baris berada di antara 1104 dan 1201, bukan di bawah Modal.

### F-03 — Kursor angka (No 28)
**FR:** Semua input angka berpemisah ribuan (`angka()`) mempertahankan posisi kursor relatif
terhadap jumlah digit di kirinya setelah format ulang. Berlaku untuk menghapus, menyisip, dan menempel.
Input boleh negatif bila kolomnya mengizinkan (saldo awal laba ditahan).
**Penerimaan:** Pada `1.250.000`, letakkan kursor setelah "2", hapus → `150.000` dengan kursor setelah "1".

### F-04 — Sisa label laporan (No 19)
**FR:**
| Lokasi | Sekarang | Menjadi |
|---|---|---|
| Laba Rugi (layar + PDF) | Total beban operasional (tunai) | Total beban operasional |
| Neraca (layar + PDF) | Laba (Rugi) Ditahan | Laba Ditahan |
| Neraca: kewajiban jangka panjang/pendek | satu baris pinjaman gabungan | satu baris per akun pinjaman (subtype `loan`) dengan nama akun CoA; subtotal "Total pinjaman" bila > 1 akun |
**Penerimaan:** Usaha dengan 2 akun pinjaman (2102 Utang Bank BRI, 2103 Utang Bank BCA) → Neraca menampilkan dua baris dan total; label lain sesuai tabel.

### F-05 — Saldo minus laba ditahan (No 23)
**FR:** Catatan amber "Ada saldo minus" di dialog Saldo Awal tidak memuat akun bersubtype `retained`/`retained_earnings`/`current_earnings`.
Akun 3299 "Laba Ditahan - Selisih Saldo Awal" diperlakukan khusus: diperingatkan bila ≠ 0 dengan pesan "Ada selisih saldo awal Rp…".
Dashboard memakai aturan yang sama.
**Penerimaan:** Laba ditahan −Rp5.000.000 → tidak ada catatan; 3299 = Rp10.000 → muncul catatan selisih.

### F-06 — Satu keterangan per transaksi (No 27)
**FR:** Halaman Catat (jurnal manual) hanya punya satu kolom Keterangan di kepala transaksi.
Kolom keterangan per baris dihapus dari layar. Keterangan kepala disalin ke setiap baris saat disimpan
(laporan per baris tetap terisi). Isi dari foto nota mengisi keterangan kepala.
**Penerimaan:** Catat 3 baris dengan keterangan "Belanja bahan 22 Okt" → Jurnal Umum menampilkan keterangan itu di ketiga baris.

### F-07 — Unduh semua daftar per rentang (No 14)
**FR:** Di Laporan Keuangan, baris "Daftar piutang, utang, invoice, barang, pelanggan, pemasok" punya
pemilih Dari–Sampai (bawaan = periode aktif). Excel dan PDF memakai rentang itu: transaksi/invoice
dalam rentang, saldo piutang/utang dan stok **per tanggal Sampai**. Judul berkas dan judul halaman menyebut rentang.
**Penerimaan:** Rentang 1 Apr–30 Sep 2026 → berkas `daftar-2026-04-01_2026-09-30.xlsx`, isi sesuai rentang.

### F-08 — Pindah dana antar rekening sendiri (No 20)
**FR:** Jenis transaksi baru **Pindah Dana** (arah netral): pilih akun asal dan akun tujuan dari kelompok
kas/bank/e-wallet/saldo marketplace, nominal, tanggal, biaya admin opsional (ke 9102 Beban Administrasi Bank).
Jurnal: Dr tujuan / Cr asal; biaya admin: Dr 9102 / Cr asal. Tidak masuk laba rugi kecuali biaya admin.
Catat cepat (jurnal manual) yang kedua sisinya kas/bank dibukukan sebagai Pindah Dana, bukan ditolak.
Impor mutasi: baris yang cocok dengan Pindah Dana di rekening lain ditandai sebagai pasangan, tidak dibukukan ulang.
Isi dari foto bukti transfer: bila pengirim dan penerima sama-sama rekening usaha, AI menyarankan Pindah Dana.
**Penerimaan:** Pindah Rp2.000.000 Bank BCA → Kas dengan admin Rp6.500 → saldo BCA −2.006.500, Kas +2.000.000, laba rugi hanya beban admin.

### F-09 — Nomor jurnal berurutan (No 31)
**FR:** Setiap transaksi (atau satu grup catatan) mendapat nomor `JU-YYYYMM-NNNN` saat pertama disimpan,
berdasarkan bulan tanggal transaksi dan urutan per usaha. Nomor tidak berubah bila diedit (termasuk bila tanggal
pindah bulan) dan tidak dipakai ulang bila dihapus. Nomor tampil di Transaksi, Jurnal Umum, Buku Besar, dan PDF,
dan bisa dicari. Transaksi yang sudah ada diberi nomor sekali (urut tanggal, lalu id).
Entri yang dibuat sistem tanpa transaksi (penyusutan, penyesuaian persediaan, tutup buku) yang sekarang
tampil sebagai `JU-YYYYMM-n` diganti menjadi `SIS-YYYYMM-n` supaya tidak tertukar dengan nomor jurnal pengguna;
`BYR-` untuk pelunasan dan `SA` untuk saldo awal tetap. Referensi lama `TRX-<id>` tetap bisa dicari.
**Penerimaan:** Dua transaksi Okt → JU-202610-0001, -0002; hapus -0001, catat baru → -0003.

### F-10 — Daftar pelanggan (No 24)
**FR:** Master pelanggan: nama (wajib, unik per usaha tanpa beda huruf besar/kecil), kontak, telepon, email,
alamat, NPWP, tempo hari, catatan. CRUD + impor Excel + "Ambil dari data lama" (nama unik dari invoice dan transaksi).
Form invoice memilih pelanggan dari daftar (atau ketik baru → tersimpan sebagai pelanggan).
Pelanggan yang masih dipakai tidak bisa dihapus, hanya dinonaktifkan.
**Penerimaan:** Ambil dari data lama pada data contoh → daftar terisi tanpa nama ganda; pilih pelanggan di invoice → alamat & NPWP terisi.

### F-11 — Riwayat per pihak (No 25)
**FR:** Klik nama pelanggan/supplier (di daftar, invoice, transaksi, umur piutang) membuka halaman detail:
ringkasan (total penjualan/pembelian dalam rentang, piutang/utang terbuka, jatuh tempo terdekat) dan tabel
riwayat (transaksi, invoice, pembayaran) dengan filter rentang dan unduh Excel. Pencocokan memakai `pihak_id`;
transaksi lama tanpa id dicocokkan berdasarkan nama.
**Penerimaan:** Pelanggan "PT Maju" dengan 3 invoice dan 1 pelunasan → keempatnya tampil berurutan tanggal dengan saldo berjalan.

### F-12 — Selisih persediaan (No 21)
**FR:** Akun baru 5106 "Selisih Persediaan" (kelompok HPP). Saat opname disimpan, selisih nilai
(jumlah fisik − jumlah buku) dinilai dengan metode nilai stok usaha (FIFO bawaan atau rata-rata
bergerak, sama seperti kartu stok sekarang) dan membentuk jurnal pada tanggal opname:
kurang = Dr 5106 / Cr 1301, lebih = Dr 1301 / Cr 5106. Harga pokok barang tidak berubah.
Penyesuaian akhir periode (`PenyesuaianPersediaan`) tidak lagi menghitung ulang selisih opname yang sudah dijurnal.
Ceklis Tutup Buku menampilkan total selisih bulan itu. Laba rugi menampilkan baris "Selisih persediaan" di bawah HPP.
**Penerimaan:** Stok kopi buku 10 kg @Rp100.000, fisik 9 kg → jurnal Dr 5106 Rp100.000 / Cr 1301 Rp100.000; neraca tidak menambah modal.

### F-13 — Aset dan persediaan per tanggal (No 22)
**FR:** Tab Aset & Penyusutan dan halaman Stok & Gudang punya pemilih "Per tanggal" (bawaan = hari ini / akhir periode laporan).
Aset: nilai perolehan, akumulasi penyusutan, dan nilai buku per tanggal itu (penyusutan dihitung sampai bulan tanggal itu).
Persediaan: jumlah dan nilai per barang dari kartu stok sampai tanggal itu. Total sama dengan baris Neraca pada tanggal yang sama.
**Penerimaan:** Neraca per 30 Sep = persediaan Rp12.400.000 → Stok & Gudang per 30 Sep menampilkan total Rp12.400.000.

### F-14 — Saldo awal dari daftar (No 30)
**FR:** Di Saldo Awal, baris Persediaan, Piutang Usaha, dan Utang Usaha tidak bisa diketik; nilainya
= jumlah rinciannya: persediaan dari Stok Awal (barang & jumlah per tanggal saldo awal), piutang/utang dari
daftar "Piutang/Utang awal" per pihak (nama, nomor bukti, jatuh tempo, nominal). Tombol "Rinci" membuka rincian.
Rincian piutang/utang awal muncul di menu Piutang & Utang dan bisa dilunasi seperti tagihan biasa
(menggantikan `pelunasan` saldo awal yang sekarang).
Usaha yang sudah tutup buku: baris tetap seperti sekarang, dengan peringatan bila total ≠ rincian.
**Penerimaan:** Rincian piutang 2 pihak (Rp3 jt + Rp2 jt) → baris Piutang Usaha = Rp5 jt dan terkunci.

### F-15 — Menu Tagihan (No 13)
**Kondisi sekarang:** sidebar sudah punya grup **Tagihan** berisi dua submenu, Piutang & Utang dan Invoice
(`frontend/src/lib/nav.ts:32-37`), dan perannya sudah dibedakan (effc2a0). Yang belum: keduanya masih dua halaman.
**FR:** Grup Tagihan menjadi satu halaman **Tagihan** dengan tab Invoice · Piutang · Utang · Pelanggan. Izin: pengguna yang dulu punya salah satu izin melihat tab yang sesuai.
Alamat lama (`#/invoice`, `#/piutang-utang`) dialihkan ke tab yang sesuai.
**Penerimaan:** Admin dengan izin piutang-utang saja → menu Tagihan dengan tab Piutang dan Utang.

### F-16 — Lupa password (No 15)
**FR:** Halaman Masuk punya tautan "Lupa password?". Pengguna mengisi email; sistem selalu membalas
"Bila email terdaftar, tautan dikirim." Email berisi tautan sekali pakai berlaku 30 menit.
Halaman reset meminta password baru (min. 8 karakter) dua kali. Setelah berhasil: semua sesi lama berakhir,
token tidak bisa dipakai lagi, pengguna diarahkan ke Masuk. Batas: 3 permintaan per jam per email dan per IP.
Akun nonaktif tidak menerima email.
**Penerimaan:** Minta reset → email masuk → ganti password → login dengan password baru berhasil, password lama gagal, sesi lain keluar.

## 3. Ketergantungan

- F-11 bergantung pada F-10 (pelanggan) dan memakai master supplier yang ada.
- F-15 sebaiknya setelah F-10 supaya tab Pelanggan langsung ada.
- F-14 piutang/utang awal memakai tabel `tagihan` (lihat BACKEND_SCHEMA).
- F-12 dan F-13 sama-sama membaca `gerak_stok`; kerjakan F-12 dulu.
