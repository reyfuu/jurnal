# Persediaan per akun & HPP per transaksi (Proyek A) — desain

Tanggal: 24 Okt 2026 · Status: menunggu persetujuan · Lanjutan: Proyek B (jurnal disimpan/posting) menyusul sesudah A.

## Kenapa

Klien manufaktur punya alur bertahap:
1. Membeli bahan baku dan bahan penolong.
2. Mengolah sebagian bahan menjadi barang setengah jadi.
3. Memakai kombinasi bahan baku, penolong, dan setengah jadi untuk membuat barang jadi.
4. Menjual barang jadi.

Pemilik produk ingin pembukuan mengikuti alur ini: tiap jenis barang punya **akun persediaan sendiri**, dan **HPP terhitung per produksi/penjualan**, tidak per bulan.

Kondisi kode sekarang (hasil pemetaan, 24 Okt 2026):

- **Semua persediaan masuk satu akun.** Di pembukuan, semuanya memakai `PeranPersediaan` (1301 di dagang maupun manufaktur). Saldo awal per akun persediaan (1301/1302/1304/1305/…) dijumlah lalu ditaruh di 1301.
- **Produksi tanpa jurnal.** Catat produksi (bahan keluar, hasil masuk) **tidak membuat jurnal**; perpindahan itu hanya tercatat di kartu stok.
- **Penjualan tanpa HPP per transaksi.** Penjualan tidak menjurnal HPP. HPP hanya muncul dari **satu jurnal penyesuaian per bulan** (tanggal 28) berupa selisih nilai kartu stok dengan saldo 1301.
- **Kategori barang (`StokItem.Kategori`) teks bebas.** Nilainya tidak konsisten ("Produk Jadi" vs "Barang Jadi"); tidak ada "Barang Setengah Jadi".
- **Barang tidak tercatat di jalur impor.** Pembelian dari impor Excel, mutasi bank, foto nota, dan invoice tidak mencatat barang (`StokID`).
- **Ada bug "terhitung dua kali".** Saat kartu stok aktif, nilai yang dicatat ke akun persediaan selain 1301 (lewat koreksi akun atau Catat cepat) ikut terhitung lagi.
- **Kartu stok sudah lengkap.** Kartu stok sudah menghitung nilai **per barang per gerakan** (FIFO/rata-rata), termasuk biaya lain produksi (upah, listrik) yang menambah nilai hasil. Jadi nilai yang dibutuhkan jurnal sudah ada.
- **Jurnal tidak disimpan.** Jurnal dihitung ulang dari transaksi setiap kali laporan dibuka.

## Keputusan pemilik produk

1. **Kategori barang = akun persediaan dari CoA usaha itu.**
   - Kategori tidak lagi teks bebas dan tidak ditetapkan di kode. Di Stok & Gudang, pengguna memilih salah satu **akun persediaan** di Daftar Akun usahanya (akun bersubtype persediaan).
   - Usaha ternak, misalnya, hanya melihat akun persediaan miliknya; tidak ada "suku cadang" bila CoA-nya tidak punya.
   - Kategori baru = tambah akun persediaan di Daftar Akun.
2. **Metode perpetual: HPP per transaksi.**
   - **Pembelian barang:** Dr akun persediaan barang itu.
   - **Catat produksi:** Cr akun persediaan tiap bahan yang dipakai; Dr akun persediaan hasil (setengah jadi atau jadi) sebesar nilai bahan + biaya lain produksi. Barang setengah jadi bisa menjadi hasil di satu produksi dan bahan di produksi berikutnya.
   - **Biaya lain produksi** (upah, listrik, dll.) **menambah nilai persediaan hasil**. Biaya ini baru menjadi HPP saat barangnya terjual.
   - **Penjualan barang:** Dr HPP / Cr akun persediaan barang itu, sebesar nilai keluar menurut kartu stok (FIFO/rata-rata).
   - **Stok opname:** selisihnya dijurnal di akun persediaan barang itu.
3. **Metode periodik tetap seperti cara periodik** (HPP dihitung di akhir periode). Bedanya, penyesuaian akhir bulan dihitung **per akun persediaan**, bukan satu total. Ini juga menutup bug terhitung dua kali.
4. **Cegah dobel dipakai ulang.**
   - Saat impor mutasi bank, baris yang cocok (arah sama, nominal sama, tanggal berdekatan) dengan catatan manual ditanyakan ke pengguna, seperti fitur `finance.CariMirip` yang sudah ada.
   - Jawaban **"sama"**: tidak ada jurnal baru; catatan manual (yang sudah berisi barang) yang dipakai.
   - Jawaban **"tidak sama"**: jurnal baru.
5. **Aturan wajib dibedakan per metode persediaan:**

   | | Perpetual | Periodik |
   |---|---|---|
   | Pilih barang & qty di setiap pembelian/penjualan | **Wajib** di semua jalur: form Transaksi, impor Excel, impor mutasi bank, foto nota, Catat cepat (bila menyentuh akun persediaan), **invoice** (per baris). Kartu stok wajib dimulai | Tidak wajib |
   | Persediaan awal | dari hitung fisik pertama kartu stok | **Wajib diisi per akun persediaan** di Saldo Awal; 0 boleh tapi harus diisi sadar |
   | Persediaan akhir bulan | otomatis dari kartu stok | **Wajib diisi per akun persediaan** lewat stok opname akhir bulan (cukup total nilai per akun) |
   | Tutup Buku | seperti sekarang | **Diblokir** sampai persediaan akhir bulan itu diisi |

   - Penjualan tanpa barang tidak boleh pada perpetual, karena HPP-nya tidak akan tercatat.
   - Pada periodik, HPP = persediaan awal + pembelian − persediaan akhir, per akun. Sekarang periodik tanpa opname membuat persediaan awal = akhir, sehingga HPP = semua pembelian; aturan ini menutupnya.
   - Usaha yang belum memakai daftar barang tetap bisa mencatat transaksi pada periodik.
6. **Laporan bulan-bulan lama ikut berubah** mengikuti aturan baru. Ini diterima, karena jurnal belum disimpan; Proyek B yang akan menguncinya.

## Keputusan pembuat (bukan dari pemilik produk; mohon dicek)

- **Jumlah (qty) wajib bersama barang.** Kartu stok butuh qty untuk harga per unit.
- **"+ Barang baru" di tempat.** Di layar impor, foto nota, dan invoice, ada pilihan membuat barang baru (nama, satuan, akun persediaan) tanpa pindah menu.
- **Pengecualian jasa.** Penjualan/pembelian yang memang bukan barang tetap boleh, dengan pilihan eksplisit **"Bukan barang (jasa)"**; contohnya ongkos jasa, sewa, atau maklon. Pilihan ini tidak menyentuh persediaan atau HPP. Tanpa pilihan ini, transaksi jasa di usaha manufaktur tidak bisa dicatat.
- **Pemetaan otomatis kategori teks lama ke akun:**

  | Teks lama | Akun |
  |---|---|
  | "Bahan Baku" | akun persediaan bahan baku (1301 manufaktur) |
  | "Bahan Penolong", "Kemasan" | akun bahan penolong (1302 manufaktur) |
  | "Produk Jadi", "Barang Jadi" | akun barang jadi (1305 manufaktur) |
  | "Barang Dalam Proses" | 1304 |
  | Lainnya / dagang | akun `PeranPersediaan` |

  Pemetaan berjalan sekali dan bisa diubah pengguna per barang. Pencocokan memakai nama akun di CoA usaha itu, bukan kode tetap, supaya template lain tetap jalan.
- **Barang hasil produksi.** Barang jadi otomatis yang dibuat dari produksi (sekarang `Kategori: "Produk Jadi"`) memakai akun barang jadi CoA, atau akun persediaan pilihan di form produk.
- **Saldo awal persediaan diisi otomatis.** Baris persediaan di Saldo Awal terisi otomatis per akun dari daftar barang/hitung fisik pertama dan dikunci, sama seperti baris aset tetap.
- **Dagang.** Bila CoA hanya punya satu akun persediaan, perilaku praktis tidak berubah; pengguna tidak dipaksa memilih kategori (satu-satunya akun terpilih otomatis).
- **Data lama / sebelum kartu stok dimulai.** Transaksi tanpa barang dan periode sebelum tanggal mulai kartu stok tetap memakai akun persediaan bawaan seperti sekarang. Selisih yang tidak dijelaskan gerakan kartu tetap dibereskan oleh **penyesuaian sisa** akhir bulan: penyesuaian per akun hanya untuk bagian tanpa barang, lawannya HPP. Dengan begitu laporan lama tidak rusak. Begitu semua transaksi baru berbarang, penyesuaian sisa ini akan nol dengan sendirinya.
- **Biaya lain produksi.**
  - Transaksi pembayaran biaya lain yang ditautkan ke satu produksi (`Grup`) didebit langsung ke **akun persediaan hasil produksi itu**, sehingga tidak terhitung dua kali.
  - Bila pengguna **sengaja memilih akun beban** untuk baris biaya, pilihan itu dihormati: dicatat sebagai beban dan **tidak** ditambahkan ke nilai hasil, baik di kartu maupun di jurnal.
- **Satu transaksi bisa berisi beberapa barang.**
  - Satu baris mutasi bank, satu nota, atau satu invoice boleh berisi beberapa barang. Transaksi diberi **rincian barang** (barang, qty, nilai per baris), bukan dipecah jadi banyak transaksi.
  - Dengan begitu nomor jurnal, piutang/utang per invoice, PPN, dan pencocokan dobel (nominal = baris bank) tetap utuh.
  - Data lama (`StokID`/`QtyStok` tunggal) dibaca sebagai rincian satu baris.
- **Saldo awal persediaan.**
  - **Sumber nilai:** dari daftar barang (jumlah × harga pokok) sampai kartu stok dimulai, lalu dari hitung fisik pertama.
  - **Diisi otomatis dan dikunci hanya bila ada daftar barang.** Usaha tanpa daftar barang tetap mengetik saldo awal persediaan seperti sekarang.
  - **Nilai yang sudah diketik pengguna tidak ditimpa.** Bila berbeda, cek kecocokan yang sudah ada (`cek_saldo_awal.go`) menampilkan selisihnya per akun.
- **Barang setengah jadi.**
  - Template manufaktur ditambah akun **1303 Persediaan Barang Setengah Jadi**: barang olahan yang **disimpan** dan dipakai lagi di produksi berikutnya.
  - 1304 Barang Dalam Proses tetap ada untuk yang masih di lini produksi. Kode 1303 masih kosong di template.
- **Cegah dobel saat impor mutasi** sudah berjalan lewat `CariMirip`. Yang baru hanya satu: bila pengguna menjawab **"sama"** dan catatan manualnya dibayar dari akun lain (mis. kas), aplikasi **menawarkan** memindahkan akun bayarnya ke rekening bank mutasi itu, karena mutasi bank membuktikan uangnya keluar dari bank.
- **Jasa** (tanpa CoA lengkap) di luar cakupan.

## Tujuan

- Neraca menampilkan persediaan **per akun** (bahan baku, penolong, setengah jadi, barang jadi, dst.), dengan total sama dengan nilai kartu stok.
- Pada metode perpetual, **setiap** pembelian, produksi, penjualan, dan opname barang menghasilkan jurnal di akun persediaan yang benar pada **tanggal kejadiannya**.
- Laba bersih per periode tetap konsisten antara laporan CoA dan laporan ringkas, seperti yang dijaga tes `TestKartuStok_MasukLaporanKeuangan` / `TestBukuBerjalan_CoASamaDenganRingkas` sekarang.
- Perpetual: tidak ada transaksi barang tanpa barang di jalur mana pun (kecuali yang ditandai "bukan barang").
- Periodik: HPP selalu memakai persediaan awal & akhir per akun yang diisi pengguna, tidak pernah lagi "HPP = semua pembelian".

## Bukan tujuan

- Menyimpan jurnal permanen / posting (Proyek B).
- Akuntansi aset biologis (PSAK 69) khusus ternak.
- Biaya standar, varians, dan alokasi overhead otomatis; biaya lain tetap diisi manual per produksi seperti sekarang.
- Multi-gudang.

## Anti-tujuan (gagal walau "jalan")

- Neraca persediaan per akun tidak sama dengan nilai kartu stok.
- Satu kejadian tercatat dua kali, baik di jurnal maupun di stok.
- Laba berubah hanya karena pemecahan akun. Pemecahan seharusnya memindahkan saldo antar akun neraca, bukan mengubah laba, kecuali di bagian yang memang sebelumnya salah (bug dobel).
- Pengguna dagang biasa jadi repot memilih kategori yang tidak perlu.
- Transaksi lama tanpa barang membuat laporan error atau kosong. Transaksi lama tetap tampil; yang tanpa barang tetap memakai akun persediaan bawaan seperti sekarang.

## Batasan

- **Stack yang ada:** Go (`cmd/server`, `internal/coa`, `internal/finance`), React (`frontend/src`), penyimpanan JSON / Postgres (`pgstore.go`). Perubahan struktur data harus kompatibel dengan data lama di kedua penyimpanan.
- **Jurnal tetap dihitung ulang** (`jurnalPeriode`, `cmd/server/coa.go`) sampai Proyek B.
- **Tes yang mengunci perilaku lama** perlu diperbarui dengan sengaja, bukan dihapus: daftar di laporan pemetaan, mis. `TestAutoAccount`, `TestTemplateSaldoAwal_Manufaktur`, `TestKartuStok_*`, `TestCatatProduksi_*`.
- **Tidak ada atribusi AI** di commit dan PR (AGENTS.md).

## Diserahkan ke pembuat

- Cara menyimpan akun per barang (mis. `StokItem.Akun` berisi kode akun), cara migrasinya, dan perlakuan bila akun dihapus.
- Bentuk jurnal per produksi, penjualan, dan opname di `jurnalPeriode`, mis. dari baris kartu stok per gerakan, menggantikan penyesuaian tanggal 28 pada metode perpetual.
- Tampilan pemilihan barang di layar impor mutasi, impor Excel, foto nota, Catat cepat, dan baris invoice (mis. menambah `stok_id`/qty pada `InvoiceItem`).
- Urutan pengerjaan dan pembagian PR.

## Verifikasi

- **Tes Go baru:**
  - Pembelian barang di akun X menambah saldo X.
  - Produksi bertahap (bahan → setengah jadi → jadi) dengan biaya lain menghasilkan jurnal per akun yang seimbang, dan nilai hasil = bahan + biaya.
  - Penjualan menghasilkan HPP sesuai FIFO/rata-rata.
  - Opname menghasilkan selisih di akun barang.
  - Metode periodik menghasilkan penyesuaian per akun.
  - Total persediaan per akun = nilai kartu stok.
  - Laba CoA = laba ringkas.
  - Bug dobel tertutup.
  - Impor/invoice tanpa barang ditolak, kecuali ditandai "bukan barang".
- **Data contoh manufaktur:** neraca menampilkan saldo per akun persediaan, dan tidak ada peringatan saldo minus baru.
- **Uji manual di layar:** impor mutasi dengan baris yang cocok catatan manual (dijawab "sama" dan "tidak sama"), invoice dengan pilihan barang, dan Catat produksi.
