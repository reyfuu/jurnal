# Tampilan HP (mobile) — desain

Tanggal: 23 Okt 2026 · Status: menunggu persetujuan

## Kenapa

Pemilik produk membuka aplikasi di HP dan tampilannya "buruk sekali dan tidak rapi". Tiga keluhan konkret:

1. **Menu**: sidebar kiri pindah ke atas sebagai strip menu yang digeser ke samping. Sebagian besar menu tersembunyi, dan kepala halaman (logo, strip menu, info usaha, kartu pengguna) memakan hampir sepertiga layar sebelum konten muncul.
2. **Buku Besar**: tabel angka panjang terpotong di kanan dan susah digeser, jadi kolom saldo awal, debit, kredit, dan saldo akhir tidak terbaca.
3. **Panel Master, Pengguna Klien dan Tim**: tombol Ubah, Reset password, Nonaktifkan, dan Hapus ada di kolom paling kanan tabel. Tombol itu terpotong layar sehingga tidak bisa dipakai.

Hasil pemeriksaan di lebar 390 px (iPhone 12/13) mengonfirmasi semua keluhan di atas. Jurnal Umum lebih parah lagi: kolom debit dan kredit sama sekali tidak terlihat. Saat ini aturan mobile hanya membuat `.data-table` bisa digulir ke samping (`index.css`, `@media (max-width: 760px)`), tanpa penanda apa pun.

Sukses berarti: di HP, semua menu terjangkau, angka di tabel terbaca tanpa geser ke samping, dan semua tombol aksi bisa diketuk.

## Keputusan pemilik produk (dari mockup)

1. **Menu: tombol ☰ di kiri atas (drawer)**, bukan tab di bawah layar.
   - Bar atas tipis: ☰, logo, dan pilihan periode (kalau halaman itu punya periode).
   - Ketuk ☰ dan menu lengkap meluncur dari kiri. Isi dan susunannya sama dengan sidebar laptop: grup, submenu yang bisa dilipat, Catat cepat, info usaha, kartu pengguna, tema, dan keluar.
   - Menu **wajib bisa ditutup lagi** supaya tidak mengganggu: tutup otomatis setelah memilih halaman, ketuk area gelap di luar menu, geser ke kiri, atau tombol Back di HP.
2. **Tabel jadi kartu di HP**, bukan tabel yang digeser. Di laptop tetap tabel seperti sekarang.
   - **Buku Besar**: satu kartu per akun berisi saldo awal, debit, kredit, dan saldo akhir dalam grid 2×2. Ketuk kartu untuk membuka mutasinya. Tiap mutasi menampilkan No. Bukti, tanggal, keterangan, jumlah (+ debit / − kredit), dan saldo berjalan. Penanda "saldo minus" dan "mulai minus" tetap ada.
   - **Jurnal Umum**: satu kartu per bukti berisi No. Bukti, tanggal, dan keterangan. Baris debit rata kiri, baris kredit menjorok.
3. **Panel Master, Pengguna Klien (dan Tim & Akses)**: tampilannya **ringkas, satu baris per user**. Tombol besar berlabel ditolak karena daftar jadi terlalu panjang kalau user-nya banyak.
   - Isi baris: titik status (hijau = aktif, abu = nonaktif), nama, lalu email · usaha di baris kecil di bawahnya.
   - Di kanan hanya dua tombol ikon kecil: ✎ (Ubah) dan ⋯.
   - ⋯ membuka panel dari bawah (*bottom sheet*) berisi Reset password, Nonaktifkan/Aktifkan, dan Hapus (merah). Konfirmasi yang sudah ada tetap dipakai.
4. **Search & filter tetap ada, tapi hemat tempat.**
   - Judul halaman, ikon 🔍, dan ikon ⚙ Filter berbagi **satu baris**. Kotak search penuh ditolak karena "kebesaran".
   - Ketuk 🔍 dan kotak search menggantikan judul di baris yang sama. Ada ✕ untuk menutup sekaligus mengosongkan.
   - Ketuk ⚙ dan semua filter halaman itu muncul di bottom sheet (rentang tanggal + pilihan cepat, akun, centang lain), lalu tombol **Terapkan**. Angka kecil di ⚙ menunjukkan jumlah filter aktif.
   - Filter aktif tampil sebagai label kecil di bawah judul. Ketuk ✕ di label untuk menghapus satu filter.

Mockup acuan ada di [`2026-10-23-tampilan-hp-mockup/`](2026-10-23-tampilan-hp-mockup/): `navigasi.html`, `tabel.html`, `kartu-detail-v2.html`, `ringkas-filter.html`, `search-ringkas.html`. File-file ini fragmen HTML yang bisa dibuka langsung di browser.

## Keputusan pembuat (bukan dari pemilik produk; mohon dicek)

Keputusan di bawah diambil dari preferensi yang sudah ditunjukkan pemilik produk, yaitu ringkas, tidak ada tombol besar, dan tidak ada yang terpotong:

- **Kartu Jurnal Umum**: setiap baris menampilkan kode + nama akun, dengan nominal di kanan. Baris debit rata kiri, baris kredit menjorok; posisi menjorok itulah yang menunjukkan sisi. Status seimbang dan total ditampilkan di atas daftar.
- **Pengguna Klien, info yang tidak muat di baris ringkas** (no HP, alamat, role, langganan dan masa berlaku, terakhir aktif, label "diarsipkan"): tampil di bagian atas lembar ⋯ sebagai rincian user. Aksi **Atur langganan / Buka semua fitur** ikut masuk lembar ⋯, jadi tidak ada informasi atau aksi yang hilang di HP.
- **Tim & Akses**: baris kecil di bawah nama berisi `role · klien ditugaskan`, ditambah "(Anda)" pada akun sendiri dan titik abu untuk akun nonaktif. Tab **Role & izin** (matriks centang) tetap tabel. Di HP tabel itu bisa digeser, dengan kolom nama izin menempel di kiri (`data-kartu="tidak"`), karena matriks centang tidak cocok dijadikan kartu.
- **Daftar Klien di Panel Master** (`views/master/Klien.tsx`): ikut pola baris ringkas. Barisnya berisi nama usaha, status langganan, dan pemilik. Ketuk baris untuk membuka detail klien (perilaku yang sudah ada). ⋯ membuka lembar berisi rincian kolom lain beserta aksi **Atur langganan** dan **Lihat sebagai**.

## Tujuan

- Semua halaman yang dipakai owner, admin, kasir, dan tim Master **rapi di lebar 360–430 px** tanpa scroll horizontal di level halaman.
- Interaksi terasa mulus: gulir alami tanpa scroll bersarang yang saling rebut, target ketuk ≥ 40 px, dan drawer serta bottom sheet beranimasi singkat (±200 ms). Bila pengguna memilih *reduce motion*, animasi tidak dipakai.
- Desktop (> 760 px) **tidak berubah**.

## Bukan tujuan

- Aplikasi native atau PWA terpasang, dan mode offline.
- Mengubah alur, isi, atau logika halaman. Ini murni tampilan.
- Mendesain ulang Landing page dan halaman login. Keduanya hanya diperiksa supaya tidak rusak.

## Anti-tujuan (gagal walau "jalan")

- Ada tombol aksi yang masih terpotong atau tidak bisa diketuk di HP.
- Angka di laporan harus digeser ke samping untuk dibaca.
- Kepala halaman tetap memakan banyak layar.
- Desktop ikut berubah atau rusak.
- Search atau filter jadi tidak bisa diakses di HP.

## Batasan

- Stack yang ada: React 19, Vite, Tailwind 4, komponen shadcn/ui, dan CSS di `frontend/src/index.css`. Breakpoint mobile yang sudah dipakai adalah `max-width: 760px`, dan breakpoint itu dipertahankan.
- Tema terang dan gelap harus tetap jalan. Pakai token warna yang ada (`--accent-red-soft`, `--text-danger`, dan seterusnya), jangan warna mentah.
- Ada sekitar 70 tabel di sekitar 45 file. Mendesain ulang semuanya satu per satu tidak realistis.
- Ikuti `AGENTS.md`: tanpa atribusi AI di commit dan PR. Uji browser hemat RAM.

## Cakupan & pendekatan

**Kerangka (semua halaman)**
- Drawer dan bar atas untuk aplikasi utama (`components/Sidebar.tsx` dan `App.tsx`) serta Panel Master (`views/master/PanelMaster.tsx`).
- Kepala halaman yang padat: periode jadi pil di bar atas, peringatan panjang (mis. saldo minus) bisa dilipat.
- Tombol "Tanya AI Pajak" di HP mengecil jadi tombol bulat ikon saja supaya tidak menutupi konten. *Ini keputusan pembuat, belum dikonfirmasi.*

**Kartu khusus (sesuai mockup)**: Buku Besar, Jurnal Umum, Pengguna Klien, dan Tim & Akses.

**Kartu otomatis untuk tabel lain**: satu pola umum yang mengubah baris tabel data menjadi kartu di HP. Isinya label kolom di kiri dan nilai di kanan. Kolom pertama jadi judul kartu, dan sel berisi tombol aksi jadi satu baris tombol kecil di bawah kartu. Pola ini berlaku otomatis untuk tabel `data-table` yang belum punya kartu khusus. Halaman dengan tabel yang hasil kartu otomatisnya buruk boleh diberi pengecualian atau kartu khusus. Cara memberi label per sel (atribut, komponen pembungkus, atau skrip yang membaca header) diserahkan ke pembuat.

**Search & filter ringkas**: dipasang di Buku Besar, Jurnal Umum, Pengguna Klien, Transaksi, dan Laporan Transaksi. Halaman lain cukup memastikan filternya tidak terpotong, yaitu turun rapi ke bawah.

**Halaman prioritas yang wajib diperiksa di 390 px**: Dashboard, Transaksi, Laporan Keuangan (semua tab), Buku Besar, Jurnal Umum, Laporan Transaksi, Tagihan, Stok & Gudang, Saldo Awal, Profil Usaha, Panel Master (Ringkasan, Klien, Pengguna Klien, Tim & Akses).

## Diserahkan ke pembuat

- Struktur komponen: misalnya komponen `KartuHP` / `BarisAksiHP`, hook `useHP()` berbasis `matchMedia('(max-width: 760px)')`, atau CSS murni.
- Cara drawer menutup lewat tombol Back, misalnya `history.pushState` saat drawer dibuka.
- Detail visual kecil (jarak, ukuran huruf) selama mengikuti mockup dan token yang ada.
- Pembagian kerja ke branch, commit, dan PR.

## Verifikasi

- Tangkapan layar headless Chrome di 390 px untuk tiap halaman prioritas, sebelum dan sesudah. Halaman tidak boleh punya scroll horizontal (`scrollWidth === innerWidth`).
- Desktop 1440 px: tangkapan layar halaman yang sama tidak berubah secara berarti.
- Uji manual di HP sungguhan lewat `http://<IP-laptop>:8080` di Wi-Fi yang sama.
- `tsc -b` dan oxlint bersih, `go test` tidak terpengaruh, dan tes unit frontend untuk logika baru bila ada (mis. pemetaan label kartu).
