# App Flow — Fitur lanjutan checklist tester (Okt 2026)

**Status:** Draft, 22 Okt 2026 · **Input:** [FRD](FRD.md) · Nama menu mengikuti sidebar di `frontend/src/lib/nav.ts`.

Notasi: `Menu → Layar → aksi`. ✅ = hasil yang dilihat pengguna. ⚠ = jalur galat.

## Fase 1

### F-01 Transaksi bulan berjalan
```
Sidebar Transaksi
  → daftar terfilter "Oktober 2026" (bulan WIB hari ini)        ✅ teks "Menampilkan Oktober 2026 · Lihat semua di Jurnal Umum"
  → ganti filter bulan ke "Semua bulan" / "September 2026"     ✅ daftar berubah, pilihan diingat sampai tab ditutup
  → klik "Lihat semua di Jurnal Umum"                         ✅ Laporan → Jurnal Umum, rentang bulan yang sama
```

### F-02 Urutan akun saldo awal
```
Pengaturan → Saldo Awal → Ubah saldo awal (dialog)
  → Tambah akun → baris baru di bawah
  → isi kode "1105" + nama "Bank BNI" → keluar dari kolom kode  ✅ baris pindah ke antara 1104 dan 1201
  → Simpan                                                    ✅ urutan tersimpan sesuai kode
```

### F-03 Kursor angka
```
Layar mana pun dengan kolom rupiah (Saldo Awal, Transaksi, Invoice, Stok)
  → klik di tengah "1.250.000", hapus satu digit              ✅ "150.000", kursor tetap di posisi digit yang sama
  → tempel "2.500.000"                                         ✅ terbaca 2500000, kursor di akhir
```

### F-04 Label laporan
```
Laporan → Laporan Keuangan → tab Laba Rugi                    ✅ "Total beban operasional" (tanpa "(tunai)")
  → tab Neraca                                                ✅ "Laba Ditahan"; tiap akun pinjaman satu baris + "Total pinjaman"
  → Unduh PDF                                                 ✅ label sama dengan layar
```

### F-05 Saldo minus
```
Saldo Awal → isi Laba Ditahan negatif → Simpan                ✅ tanpa catatan "Ada saldo minus"
  → bila neraca awal tidak seimbang dan disimpan paksa        ✅ catatan "Ada selisih saldo awal Rp…" (akun 3299)
Dashboard                                                     ✅ banner saldo minus tidak menyebut laba ditahan
```

### F-06 Keterangan satu
```
Catat → Catat transaksi (jurnal manual)
  → isi Keterangan di kepala, isi baris akun debit/kredit (tanpa kolom keterangan per baris)
  → Simpan                                                    ✅ Jurnal Umum: keterangan sama di semua baris
  → "Isi dari foto nota"                                      ✅ AI mengisi Keterangan kepala
```

## Fase 2

### F-07 Unduh daftar per rentang
```
Laporan → Laporan Keuangan → baris "Daftar piutang, utang…"
  → isi Dari 1 Apr 2026, Sampai 30 Sep 2026
  → Unduh Excel / Unduh PDF                                   ✅ berkas daftar-2026-04-01_2026-09-30, judul menyebut rentang
  ⚠ Dari > Sampai                                             ✅ pesan "Tanggal mulai harus sebelum tanggal akhir", tombol nonaktif
```

### F-08 Pindah Dana
```
Transaksi → Tambah → Jenis "Pindah Dana"
  → Dari rekening: Bank BCA · Ke rekening: Kas · Nominal 2.000.000 · Biaya admin 6.500 (opsional)
  → Simpan                                                    ✅ dua baris di daftar (pindah + biaya admin), satu nomor jurnal
  ⚠ Dari = Ke                                                 ✅ "Rekening asal dan tujuan harus berbeda"

Catat → Catat transaksi: Debit Kas, Kredit Bank BCA → Simpan  ✅ tersimpan sebagai Pindah Dana (dulu ditolak)

Catat → Isi dari foto nota (bukti transfer BCA → BRI milik usaha)
  → pratinjau AI                                              ✅ saran jenis "Pindah Dana", Dari BCA Ke BRI

Transaksi → Impor Mutasi Bank (rekening BRI)
  → baris masuk yang cocok dengan Pindah Dana BCA → BRI       ✅ ditandai "Sudah dicatat (Pindah Dana JU-…)", tidak dibukukan ulang
```

### F-09 Nomor jurnal
```
Transaksi → Tambah → Simpan                                   ✅ kolom "No. jurnal" JU-202610-0013
Laporan → Jurnal Umum → cari "JU-202610-0013"                 ✅ satu entri
Transaksi → ubah tanggal ke bulan lain → Simpan               ✅ nomor tetap JU-202610-0013
```

### F-10 Daftar pelanggan
```
Invoice (atau Tagihan setelah F-15) → tab Pelanggan
  → pertama kali kosong: tombol "Ambil dari data lama"        ✅ daftar terisi nama unik dari invoice & transaksi
  → Tambah pelanggan → isi nama, alamat, NPWP → Simpan
  ⚠ nama sudah ada                                             ✅ "Pelanggan dengan nama ini sudah ada"
Invoice → Buat invoice → kolom Pelanggan (cari/pilih)         ✅ alamat & NPWP terisi otomatis
  → ketik nama baru                                           ✅ pilihan "Tambah 'PT Baru' sebagai pelanggan"
Hapus pelanggan yang sudah punya invoice                      ⚠ "Masih dipakai di 3 invoice; nonaktifkan saja" + tombol Nonaktifkan
```

### F-11 Riwayat per pihak
```
Pelanggan / Supplier → klik nama   (juga dari Invoice, kolom Pihak di Transaksi, Umur Piutang)
  → Halaman "PT Maju"
      ringkasan: total penjualan (rentang), piutang terbuka, jatuh tempo terdekat
      tabel: tanggal · no. jurnal/invoice · keterangan · tagihan · bayar · saldo
  → ubah rentang / Unduh Excel                                ✅ isi mengikuti rentang
  → klik baris                                                ✅ buka transaksi atau invoice terkait
```

## Fase 3

### F-12 Selisih persediaan
```
Stok & Gudang → tab Stok opname → isi jumlah fisik → Simpan
  → ringkasan sebelum simpan: "Kurang 1 kg Kopi Arabika = Rp100.000 → Selisih Persediaan"
  → Simpan                                                    ✅ jurnal Dr 5106 / Cr 1301 pada tanggal opname
Laporan → Laba Rugi                                           ✅ baris "Selisih persediaan" di bawah HPP
Tutup Buku Bulanan → ceklis                                   ✅ "Selisih persediaan bulan ini: Rp100.000 (kurang)"
```

### F-13 Aset dan persediaan per tanggal
```
Laporan → Laporan Keuangan → tab Aset & Penyusutan → "Per tanggal" 30 Sep 2026
                                                              ✅ nilai perolehan, akumulasi, nilai buku per 30 Sep
Stok & Gudang → "Per tanggal" 30 Sep 2026                     ✅ jumlah & nilai per barang, total = persediaan di Neraca per 30 Sep
                                                              ✅ tombol ubah/hapus nonaktif selama melihat tanggal lampau
```

### F-14 Saldo awal dari daftar
```
Pengaturan → Saldo Awal → Ubah
  → baris Persediaan (terkunci) → "Rinci"                     ✅ buka Stok awal (barang × jumlah × harga pokok)
  → baris Piutang Usaha (terkunci) → "Rinci"
      → Tambah: pihak, no. bukti, jatuh tempo, nominal → Simpan
                                                              ✅ baris Piutang Usaha = jumlah rincian
  → Simpan saldo awal                                         ✅ neraca awal tanpa selisih dari daftar
Tagihan → Piutang                                             ✅ piutang awal muncul berlabel "Saldo awal", bisa dilunasi
Usaha yang sudah tutup buku                                   ⚠ baris tidak dikunci; peringatan "Total piutang awal Rp… ≠ rincian Rp…"
```

### F-15 Menu Tagihan
```
Sekarang: Sidebar → grup Tagihan → submenu Piutang & Utang / Invoice (dua halaman)
Sesudah:  Sidebar → Tagihan
  → tab Invoice · Piutang · Utang · Pelanggan (sesuai izin)   ✅ satu halaman
Tautan lama #/invoice atau #/piutang-utang                    ✅ dialihkan ke tab yang sesuai
```

### F-16 Lupa password
```
Masuk → "Lupa password?"
  → isi email → Kirim                                         ✅ "Bila email terdaftar, tautan reset sudah dikirim. Berlaku 30 menit."
Email → klik tautan → halaman Atur ulang password
  → password baru 2× → Simpan                                 ✅ "Password diganti. Silakan masuk." → halaman Masuk
  ⚠ tautan kedaluwarsa / sudah dipakai                        ✅ "Tautan tidak berlaku lagi" + tombol "Kirim ulang"
  ⚠ 4× permintaan dalam 1 jam                                 ✅ "Terlalu banyak permintaan, coba lagi nanti"
Perangkat lain yang masih login                               ✅ keluar otomatis
```
