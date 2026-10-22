# Technical Requirements Document — Fitur lanjutan checklist tester (Okt 2026)

**Status:** Draft, 22 Okt 2026 · **Basis kode:** `origin/main` d9b467c · **Input:** [FRD](FRD.md) · **Skema:** [BACKEND_SCHEMA](BACKEND_SCHEMA.md)

## 1. Arsitektur yang berlaku

- **Backend:** Go 1.27, satu binary `cmd/server`. State per usaha (`tenant`) ada di memori dengan `t.mu`;
  setiap request yang mengubah data dibungkus `s.persisting(...)` lalu ditulis ke PostgreSQL lewat
  `pgstore.go` (kolom = tag JSON struct). Skema idempoten di `cmd/server/schema.sql` dijalankan setiap boot.
- **Jurnal tidak disimpan.** Jurnal berpasangan dihitung dari `transactions` + aset + stok + saldo awal
  (`internal/coa/journal.go`, `Journal`, `PenyesuaianPersediaan`). Fitur yang menambah dampak akuntansi
  menambah baris di perhitungan ini, bukan tabel jurnal baru.
- **Frontend:** React 19 + Vite, `frontend/src/views/*`, API di `frontend/src/lib/api.ts`, input angka lewat
  `frontend/src/lib/angka.ts`.
- **Izin:** `s.guard("<view>", ...)` di `cmd/server/main.go`; daftar view per peran di `internal/auth/auth.go`.

## 2. Desain per fitur

### Fase 1

**F-01 Transaksi bulan berjalan.** `frontend/src/views/Transaksi.tsx:47`: nilai awal `bulan` dari
`sessionStorage['mp:transaksi-bulan']` atau `bulanIni()` (WIB, pola yang sama dengan `today()` di berkas itu).
Tulis ke `sessionStorage` saat filter berubah. Tidak ada perubahan backend (filter sudah di klien). Tes Vitest
untuk fungsi pemilih bulan awal.

**F-02 Urutan saldo awal.** `frontend/src/views/SaldoAwal.tsx:95-99` (`tampil`) dan `:396` (Tambah akun):
urutkan `baris` dengan `kode.localeCompare` numerik setelah `onBlur` kolom kode dan sebelum kirim. Backend
`internal/coa/saldo_awal.go` mengurutkan `Rincian` per kode saat `IsiKodeSaldo` supaya data lama juga rapi.

**F-03 Kursor angka.** `frontend/src/lib/angka.ts`: `angka()` mengembalikan `onChange` yang menghitung
jumlah digit di kiri `selectionStart` sebelum format, lalu setelah render memasang `setSelectionRange` pada
posisi digit yang sama (lewat `requestAnimationFrame` atau `useLayoutEffect` pada ref). Tambah opsi
`{ negatif: true }` supaya `bacaAngka` tidak membuang tanda minus untuk kolom yang boleh negatif.
Tes Vitest: hapus/sisip di tengah, tempel teks berformat, minus.

**F-04 Label laporan.**
- `internal/finance/laporan_baris.go:159` dan `frontend/src/views/Reports.tsx:244`: hapus "(tunai)".
- `laporan_baris.go:195` dan `Reports.tsx:332`: "Laba Ditahan".
- Pinjaman per akun: `laporan_baris.go:187` membuat satu `BarisLaporan` per akun bersubtype `loan` dari
  saldo per akun di `internal/coa/reports.go`; nama dari `BeriNama` (`cmd/server/nama_akun.go`). `Reports.tsx:323`
  memetakan daftar baris, bukan satu baris tetap. PDF (`internal/finance/laporan_pdf.go`) ikut karena memakai baris yang sama.
- Tes: golden test baris Neraca dengan 2 akun pinjaman.

**F-05 Saldo minus.** `frontend/src/views/SaldoAwal.tsx:435-439`: filter baris memakai daftar subtype yang
sama dengan `subtypeBolehMinus` di `cmd/server/saldo_minus.go:29`. Pindahkan daftar itu ke respons
`/api/v1/coa` (field `boleh_minus` per akun) supaya klien dan server satu sumber. Keluarkan 3299 dari
`subtypeBolehMinus` dan tampilkan pesan selisih tersendiri.

**F-06 Keterangan satu.** `frontend/src/views/CatatJurnal.tsx:281,297`: hapus kolom keterangan baris.
Saat simpan, `keterangan` baris = keterangan kepala. Backend `internal/coa/catat.go` tetap menerima
`keterangan` baris (kompatibel dengan klien lama dan impor), memakai kepala bila baris kosong.

### Fase 2

**F-07 Unduh daftar per rentang.** Backend sudah membaca `dari`/`sampai` (`rentangDari`,
`cmd/server/laporan_transaksi.go:25`; `handleDaftarXlsx/PDF`, `cmd/server/daftar.go:102,117`).
Ubah `daftarUrl` (`frontend/src/lib/api.ts:1911`) menjadi `(dari, sampai, format)` dan tambah dua input
tanggal di `Reports.tsx:118-123`. Pastikan saldo piutang/utang dan stok di `daftar.go` dihitung per `sampai`
(bukan "sekarang"); stok per tanggal memakai fungsi dari F-13 (kerjakan bagian stok setelah F-13, sementara
pakai nilai sekarang dengan catatan di judul).

**F-08 Pindah Dana.**
- `internal/finance/transaction.go`: tambah `{"Pindah Dana", "keluar", "pindah"}` ke `JenisList`. Tidak ada kolom
  baru: model transaksi sudah punya `Lawan` (kode kas/bank sisi pembayaran, diisi `pakaiRekeningBawaanLocked`,
  `cmd/server/rekening.go:38`) dan `Akun` (akun kategori). Untuk Pindah Dana, `Lawan` = rekening asal dan
  `Akun` = rekening tujuan; validasi: keduanya berperan `LawanRekening` (`internal/coa/catat.go:28`) dan berbeda.
- `internal/coa/journal.go` `Journal`: pola dua baris yang ada untuk jenis keluar (Dr `Akun` / Cr `Lawan`) sudah
  menghasilkan Dr tujuan / Cr asal; cukup izinkan `Akun` berupa akun kas/bank untuk jenis ini. Biaya admin dicatat
  sebagai transaksi kedua jenis `Biaya Lain` akun 9102 dalam grup yang sama.
- `internal/finance` laporan kas (`berjalan.go`) dan arus kas: pos `pindah` tidak masuk pendapatan/beban;
  arus kas per akun bergerak, total kas tidak berubah.
- `internal/coa/catat.go:139`: kasus `semuaLawan(debit) && semuaLawan(kredit)` dengan kedua sisi
  kas/bank (bukan piutang/utang) → bentuk transaksi Pindah Dana; piutang/utang tetap ditolak dengan pesan lama.
- Impor mutasi (`cmd/server/mutasi.go`) dan `finance.Mirip`: Pindah Dana dicocokkan di kedua rekening.
- AI nota (`internal/ai/nota.go`): bila rekening pengirim dan penerima ada di daftar rekening usaha, saran jenis = Pindah Dana.
- Tes: jurnal seimbang, saldo per akun, laba rugi tidak berubah, catat cepat kas→bank tidak lagi galat.

**F-09 Nomor jurnal.**
- Kolom `transactions.no_jurnal` + penghitung per bulan `tenants.nomor_jurnal JSONB` (`{"202610": 12}`).
- Diberikan di satu tempat: fungsi `(*tenant).beriNomorJurnalLocked(tanggal)` dipanggil dari semua jalur buat
  transaksi pengguna (form Transaksi, Catat, impor Excel, impor mutasi, foto nota). Satu grup catatan = satu nomor.
- Migrasi saat boot (`pgstore.go` load): transaksi tanpa `no_jurnal` diberi nomor urut (tanggal, id) sekali;
  idempoten karena hanya mengisi yang kosong.
- `cmd/server/jurnalumum.go:94-102`: `Bukti` = `no_jurnal` bila ada; entri sistem `SIS-YYYYMM-n`.
  Pencarian `cocokJurnal` mencocokkan `no_jurnal`, `TRX-<id>`, dan `no_bukti`.
- Tes: urutan per bulan, hapus tidak memakai ulang, edit tanggal tidak mengubah nomor, migrasi idempoten.

**F-10 Pelanggan.** Tabel `pelanggan` meniru `supplier` (lihat BACKEND_SCHEMA). Handler baru
`cmd/server/pelanggan.go` (list/create/update/delete/import/ambil-dari-data-lama) dengan izin `invoice`.
`invoices.pelanggan_id` opsional; `Invoice.tsx` memakai combobox pelanggan. Validasi nama unik mengikuti pola
1596656 (SKU/nama master tidak kembar). `internal/finance/daftar.go:32` `RekapPelanggan` membaca master bila ada.

**F-11 Riwayat pihak.** Endpoint `GET /api/v1/pihak/{jenis}/{id}/riwayat?dari&sampai` (jenis = pelanggan|supplier)
menggabungkan transaksi (`pihak_id` atau `lower(pihak) = lower(nama)`), invoice, tagihan, dan pelunasan,
diurutkan tanggal dengan saldo berjalan piutang/utang. Kolom baru `transactions.pihak_id` diisi saat pihak dipilih
dari daftar. Frontend: `views/RiwayatPihak.tsx`, dibuka dari daftar pelanggan/supplier, Invoice, Transaksi
(kolom Pihak), dan `UmurPiutang.tsx`. Unduh Excel memakai `internal/xlsx`.

### Fase 3

**F-12 Selisih persediaan.**
- CoA: akun `5106 Selisih Persediaan` subtype `inventory_variance`, kelompok HPP, peran baru `PeranSelisihPersediaan`
  (`internal/coa/peran.go`, `coa.go`); usaha lama mendapat akun ini lewat mekanisme akun tambahan (`TambahanDiLuarFile`).
- `internal/finance/kartustok.go:397-411` sudah menghitung `selisih` dan `nilai` per baris opname
  (`OpnameNilai`, `SelisihOpname`). `Journal` menambah baris Dr/Cr 5106 vs 1301 untuk setiap gerak opname pada tanggalnya.
- `PenyesuaianPersediaan` (`journal.go:96`) dan pemanggilnya (`cmd/server/coa.go:166-180`) mengurangi
  `SelisihOpname` yang sudah dijurnal supaya tidak dihitung dua kali.
- Laba rugi (`laporan_baris.go`): baris "Selisih persediaan" di bawah HPP. Tutup buku (`cmd/server/tutupbuku.go`
  `ringkasTutup`): cek baru "Selisih persediaan bulan ini Rp…".
- Tes: opname kurang/lebih, metode FIFO dan rata-rata, periodik dan perpetual; total HPP + selisih sama dengan perhitungan lama.

**F-13 Per tanggal.**
- `GET /api/v1/assets?pada=YYYY-MM-DD` (`cmd/server/assets.go:24`): penyusutan sampai bulan `pada`
  (fungsi penyusutan yang sama dengan Neraca per tanggal 13eb940).
- `GET /api/v1/stok?pada=YYYY-MM-DD` (`cmd/server/tagihan.go:75`): replay `gerak_stok` sampai `pada`
  dengan fungsi kartu stok (`internal/finance/kartustok.go`) → jumlah & nilai per barang.
- Frontend: pemilih tanggal di `Reports.tsx:183` (tab Aset) dan `PiutangStok.tsx` (Stok & Gudang); saat
  `pada` ≠ hari ini, form ubah dinonaktifkan (tampilan sejarah).
- Tes: total stok per tanggal = baris persediaan Neraca per tanggal yang sama (data contoh).

**F-14 Saldo awal dari daftar.**
- Persediaan: `Stok awal` sudah ada (langkah 2, `SaldoAwal.tsx:236-242`, `:115-119`); jadikan baris
  persediaan read-only = Σ nilai stok awal per tanggal saldo awal (fungsi F-13 dengan `pada` = tanggal saldo awal).
- Piutang/utang: rincian per pihak disimpan sebagai baris `tagihan` dengan `saldo_awal = true`, dibayar lewat
  pelunasan biasa. Migrasi: `opening_balances.pelunasan` lama dikonversi menjadi pelunasan tagihan saldo awal
  untuk satu baris "Saldo awal (tanpa rincian)" per jenis, sehingga angka lama tidak berubah.
- `cmd/server/cek_saldo_awal.go:31`: cek piutang/utang terhadap Σ tagihan saldo awal.
- Usaha yang sudah tutup buku: tidak dimigrasi paksa; tampilkan peringatan selisih.

**F-15 Menu Tagihan.** Grup sidebar Tagihan (`frontend/src/lib/nav.ts:32-37`, submenu `piutang-utang` dan `invoice`)
→ satu entri `tagihan` dengan tab; `App.tsx` merender
`InvoiceView`/`PiutangStok` per tab; `lib/rute.ts` mengalihkan `#/invoice` dan `#/piutang-utang`. Izin server
tetap (`invoice`, `piutang-utang`); daftar view peran di `auth.go` menambah `tagihan` bila salah satu izin ada.

**F-16 Lupa password.**
- Tabel `password_resets` (hash token SHA-256, user, kedaluwarsa, dipakai). Token 32 byte acak, dikirim hanya di email.
- `internal/auth`: `MintaReset(email) (token, user, ok)` dan `ResetDenganToken(token, pwBaru)` (memakai
  `hashPassword` dan `revoke` yang sama dengan `ResetPassword`, `auth.go:463`).
- Endpoint publik `POST /api/v1/auth/lupa-password` dan `POST /api/v1/auth/reset-password`, rate limit memakai
  `reserveAttempt` yang sudah dipakai `handleRegister`.
- Pengirim email: paket kecil `internal/surel` (SMTP `net/smtp` stdlib) dengan env `SMTP_HOST`, `SMTP_PORT`,
  `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`; tanpa konfigurasi → endpoint tetap membalas sama dan mencatat log
  peringatan (dev: tautan dicetak ke log).
- Frontend: `views/LupaPassword.tsx`, `views/ResetPassword.tsx`, tautan di `Login.tsx`; rute `#/reset?token=`.
- Tes: token sekali pakai, kedaluwarsa, email tak terdaftar tidak bocor, sesi lama dicabut.

## 3. Kualitas dan rilis

- Setiap PR: `go vet ./... && go test ./...`, `cd frontend && npx vitest run && npm run lint && npm run build`
  (build memakai `tsc -b`). Fitur dengan logika akuntansi (F-04, F-08, F-09, F-12, F-13, F-14) wajib tes Go.
- Benchmark `go test ./... -run '^$' -bench .` dijalankan untuk F-08, F-09, F-12, F-13 (menyentuh jalur
  Transaksi → Jurnal → Laporan, lihat agent.md §1.B).
- Gerbang agent-skills: `/agent-skills:review`, `ponytail-review`, `/agent-skills:ship` sebelum PR.
- Deploy: `pg_dump` manual, `deploy.sh` (dua kali bila `deploy.sh` berubah, F-16 menambah env SMTP).
