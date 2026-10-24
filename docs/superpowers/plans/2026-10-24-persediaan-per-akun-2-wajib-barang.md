# Persediaan per akun — Tahap 2: wajib barang & rincian multi-barang — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Pada metode perpetual, setiap Penjualan dan Beli Bahan / Stok di semua jalur input membawa barang + qty (boleh beberapa barang per transaksi) atau ditandai eksplisit "Bukan barang (jasa)"; jawaban "sama" di impor mutasi bisa memindahkan akun bayar ke rekening bank.

**Architecture:** `finance.Transaction` mendapat `Rincian []RincianBarang` dan `BukanBarang bool`; `StokID`/`QtyStok` lama dibaca sebagai rincian satu baris lewat satu fungsi (`BarisBarang`). Kartu stok dan jurnal perpetual membaca rincian per baris. Satu aturan server (`cekBarangLocked`) dipanggil oleh semua jalur simpan. Frontend memakai satu komponen pemilih rincian (+ barang baru di tempat) di form Transaksi, Catat cepat, impor mutasi, dan invoice.

**Tech Stack:** Go (`cmd/server`, `internal/finance`, `internal/coa`), React 19 + TS + Vite + vitest, oxlint, JSON store & Postgres (`pgstore.go`, `schema.sql`).

**Spec:** `docs/superpowers/specs/2026-10-24-persediaan-per-akun-design.md` (tabel aturan wajib per metode, "Keputusan pembuat": qty wajib, + barang baru, jasa, multi-barang, mutasi "sama").

## Global Constraints

- Aturan wajib barang **hanya** untuk `MetodePersediaan == "perpetual"`; periodik dan usaha tanpa daftar barang tidak berubah.
- Setiap jalur simpan di Task 5–9 memanggil `cekBarangLocked` (Task 3) per transaksi dan `cekTanpaMinusLocked` (Task 4) atas keadaan usulan sebelum menyimpan.
- Hanya jenis `Penjualan` (`finance.JenisJual`) dan `Beli Bahan / Stok` (`finance.JenisBeliStok`) yang membawa barang.
- Data lama (`stok_id`/`qty_stok` tunggal, tanpa `rincian`) harus menghasilkan kartu stok, jurnal, dan laporan **identik** dengan sekarang.
- `QtyStok` dan `qty_stok` rincian tetap per seribu satuan (`finance.TakaranSkala`).
- Invarian: `TestKartuStok_MasukLaporanKeuangan`, `TestBukuBerjalan_CoASamaDenganRingkas`, semua `TestPersediaanAkun_*` lolos tanpa mengubah angka harapan.
- Postgres: tiap field baru wajib punya kolom (migrasi `ADD COLUMN IF NOT EXISTS` di `schema.sql`) dan ikut `TestPostgresSimpanMuatSemuaField`.
- Teks UI persis: opsi `"Bukan barang (jasa)"`, tombol `"+ Barang baru"`, galat server `"pilih barang & jumlahnya, atau tandai Bukan barang (jasa)"`.
- Commit tanpa trailer/atribusi AI (AGENTS.md). Go: `PATH="/c/Program Files/Go/bin:$PATH"`. `TestStateSurvivesRestart`, `TestSimpanGagalBukanSukses` gagal di Windows (dikecualikan). oxlint tetap 24 peringatan.

## Keputusan pembuat dalam rencana ini (mohon dicek)

1. **(Diputuskan pemilik produk.) Perpetual: barang selalu wajib.** Bila kartu stok belum dimulai, Penjualan/Beli Bahan / Stok tanpa `BukanBarang` ditolak dengan `"mulai kartu stok dulu (hitung fisik pertama) sebelum mencatat penjualan/pembelian barang"`. Pengecualian satu-satunya: transaksi bertanggal **sebelum** tanggal mulai kartu stok (catatan stok belum ada).
1a. **(Diputuskan pemilik produk.) Perpetual: stok tidak boleh minus.** Perubahan apa pun yang membuat saldo qty suatu barang di kartu stok minus (penjualan, pemakaian produksi, hapus/ubah pembelian, opname, hapus dokumen) ditolak. Data lama yang sudah minus sebelum aturan ini tidak memblokir perubahan lain.
2. **Nilai per baris rincian:** `Σ nilai ≤ Nominal`. Untuk pembelian, nilai baris = nilai barang yang masuk kartu; sisanya (mis. ongkir jasa) tetap ke akun bawaan seperti sekarang. Rincian satu baris dengan nilai 0 otomatis bernilai Nominal. Untuk penjualan, nilai baris hanya informasi (HPP tetap dari kartu FIFO/rata-rata).
3. **Impor Excel:** satu baris Excel = satu barang (kolom baru `barang` = SKU atau nama persis, `jumlah_barang`); nilai `jasa` di kolom `barang` = Bukan barang (jasa). Multi-barang lewat impor Excel di luar tahap ini.
4. **Catat cepat:** rincian berlaku untuk satu-satunya transaksi Penjualan/Beli yang dihasilkan catatan itu; bila catatan menghasilkan lebih dari satu transaksi berbarang, server menolak dengan pesan agar dipecah.
5. **Edit transaksi lama tanpa barang** (perpetual, sesudah tanggal mulai) wajib dilengkapi saat disimpan ulang. Pelunasan (`lunasiTxLocked`) dan sinkron invoice lama yang sudah dibukukan **tidak** diblokir.
6. **Pindah akun bayar (mutasi "sama"):** dialog mutasi mendapat pilihan rekening bank (bawaan = rekening bawaan). Bila catatan manual yang cocok dibayar dari akun yang bukan rekening itu dan bukan transaksi invoice/Grup, aksi `"Sama — pindah akun bayar ke <rekening>"` ditawarkan; tanpa itu perilaku "lewati" tetap.

## Review Focus

1. Transaksi lama dengan `stok_id` tunggal (tanpa `rincian`) diubah tanpa menyentuh barang → tersimpan dengan rincian setara, kartu & laporan identik (tes Task 1 dan Task 2).
2. Pelunasan piutang/utang atas penjualan perpetual lama tanpa barang tetap berhasil (tes Task 3).
3. Invoice diubah (baris barang dihapus/ditambah) setelah dibukukan → rincian transaksi Penjualan-nya ikut, kartu stok mengikuti (tes Task 9).
4. Baris mutasi yang cocok dengan catatan manual berbarang dijawab "sama" → tidak ada transaksi baru dan kartu stok tidak berubah (tes Task 7).
5. Barang yang dipakai di rincian (bukan hanya `stok_id`) tidak boleh dihapus dari daftar barang (tes Task 1).

---

### Task 1: Model rincian barang & validasi

**Files:**
- Modify: `internal/finance/transaction.go` (struct `Transaction` :73-108, `Validate` :133-142)
- Modify: `cmd/server/kartustok.go` (`validStok` :44-52, `stokDipakai` :56)
- Modify: `cmd/server/schema.sql`, `cmd/server/pgstore_test.go`, `frontend/src/lib/api.ts` (type `Tx`)
- Test: `internal/finance/rincian_test.go`, `cmd/server/rincian_test.go`

**Interfaces:**
- Produces: `type RincianBarang struct { StokID int \`json:"stok_id"\`; QtyStok int64 \`json:"qty_stok"\`; Nilai int64 \`json:"nilai,omitempty"\` }`.
- Produces: `Transaction.Rincian []RincianBarang \`json:"rincian,omitempty"\``, `Transaction.BukanBarang bool \`json:"bukan_barang,omitempty"\``.
- Produces: `func (t Transaction) BarisBarang() []RincianBarang`: `Rincian` bila ada; selain itu `StokID>0` → `[{StokID, QtyStok, Nominal}]`; selain itu nil. Baris bernilai 0 pada rincian satu baris → Nominal.
- `Validate` (aturan baru): `Rincian` dan `StokID` tidak boleh diisi bersamaan; tiap baris `StokID>0`, `QtyStok>0`, `Nilai>=0`; `Σ Nilai ≤ Nominal`; barang hanya untuk Penjualan/Beli Bahan / Stok; `BukanBarang` tidak boleh bersama barang dan hanya untuk dua jenis itu.
- Produces: `func NormalkanBarang(t *Transaction)`: `StokID/QtyStok` tunggal → `Rincian` satu baris (nilai = Nominal), lalu `StokID/QtyStok` dikosongkan. Dipanggil di jalur simpan server sesudah `Validate`; data lama di penyimpanan tidak ditulis ulang.
- Produces (frontend): `RincianBarang = { stok_id: number; qty_stok: number; nilai?: number }`, `Tx.rincian?: RincianBarang[]`, `Tx.bukan_barang?: boolean`.

- [ ] **Step 1: Tes** `TestBarisBarang_DataLamaJadiSatuBaris` (StokID 3, QtyStok 2000, Nominal 50.000 → `[{3,2000,50000}]`), `TestNormalkanBarang` (hasil `BarisBarang()` sebelum = sesudah normalisasi; Review Focus 1), `TestValidate_Rincian` (tabel: rincian+stok_id → galat; qty 0 → galat; Σ nilai 120.000 > Nominal 100.000 → galat; rincian pada "Gaji & Upah" → galat; bukan_barang + rincian → galat; dua baris 60.000+40.000 Nominal 100.000 → sah), `TestStok_TidakBolehHapusBarangDiRincian` (server: barang hanya muncul di `rincian` transaksi → DELETE 409), dan tambah `rincian`/`bukan_barang` ke `TestPostgresSimpanMuatSemuaField`.
- [ ] **Step 2:** `go test ./internal/finance ./cmd/server -run 'BarisBarang|Validate_Rincian|TidakBolehHapusBarangDiRincian|PostgresSimpanMuat'` → FAIL
- [ ] **Step 3:** Implementasi; `validStok` memeriksa setiap baris `BarisBarang()`; `stokDipakai` juga; `schema.sql`: `ALTER TABLE transactions ADD COLUMN IF NOT EXISTS rincian JSONB`, `... bukan_barang BOOLEAN NOT NULL DEFAULT false`.
- [ ] **Step 4:** `go test ./internal/... ./cmd/server` → PASS; `cd frontend && npx tsc -b`
- [ ] **Step 5: Commit** `feat(transaksi): rincian barang & tanda bukan barang`

### Task 2: Kartu stok & jurnal membaca rincian

**Files:**
- Modify: `internal/finance/kartustok.go` (`HitungKartuMetode` :415-431)
- Modify: `internal/coa/journal.go` (`JournalBerjalan` :154)
- Test: `internal/finance/kartustok_rincian_test.go`, `cmd/server/persediaan_akun_test.go`

**Interfaces:**
- Consumes: `Transaction.BarisBarang()` (Task 1).
- Kartu: tiap baris rincian menjadi event sendiri (beli: `masuk(StokID, QtyStok, Nilai)`; jual: `keluar(StokID, QtyStok)`), `TransaksiID` sama, urutan sesuai urutan rincian.
- Jurnal perpetual pembelian (`Akun == ""`): Dr `akunStok(baris.StokID)` sebesar `baris.Nilai` per baris; sisa `Nominal − Σ Nilai` (>0) ke akun bawaan seperti transaksi tanpa barang; Cr tetap satu baris sebesar Nominal.

- [ ] **Step 1: Tes** `TestKartu_RincianDuaBarang` (beli Kain 3.000 mili Rp30.000 + Benang 1.000 mili Rp5.000 dalam satu transaksi Nominal 40.000 → dua baris KartuBeli TransaksiID sama, nilai 30.000 & 5.000); `TestKartu_DataLamaSamaDenganRincian` (kartu dari transaksi `StokID` tunggal == kartu dari transaksi `Rincian` setara, `reflect.DeepEqual` pada `Baris`); `TestPersediaanAkun_PerpetualBeliDuaAkun` (server: satu pembelian Kain 1301 Rp30.000 + Benang 1302 Rp5.000 + ongkir Rp5.000 → saldo 1301 +30.000, 1302 +5.000, sisa 5.000 tersapu penyesuaian ke HPP; laba CoA = ringkas; neraca saldo seimbang); `TestPersediaanAkun_PenjualanDuaBarang` (satu penjualan dua barang → dua baris HPP dalam satu entri Jurnal Umum berbukti sama).
- [ ] **Step 2:** → FAIL
- [ ] **Step 3:** Implementasi.
- [ ] **Step 4:** `go test ./internal/... ./cmd/server` → PASS (semua invarian Global Constraints)
- [ ] **Step 5: Commit** `feat(stok): kartu & jurnal perpetual per baris rincian barang`

### Task 3: Aturan wajib barang di server & form Transaksi

**Files:**
- Create: `cmd/server/wajib_barang.go`
- Modify: `cmd/server/main.go` (`(*tenant).decodeTx` :1098, create :1128, update :1156)
- Modify: `frontend/src/views/Transaksi.tsx` (:401, :449-480), `frontend/src/views/KartuStok.tsx` (dialog beli :36-100)
- Create: `frontend/src/components/PilihRincianBarang.tsx`, `frontend/src/components/BarangBaruDialog.tsx`, `frontend/src/lib/rincian.ts` (+ `rincian.test.ts`)
- Test: `cmd/server/wajib_barang_test.go`

**Interfaces:**
- Produces: `func (s *tenant) cekBarangLocked(t finance.Transaction) error` — galat `"pilih barang & jumlahnya, atau tandai Bukan barang (jasa)"` bila perpetual, jenis Penjualan/Beli Bahan / Stok, `!BukanBarang`, `len(BarisBarang())==0`, dan (kartu belum dimulai **atau** `Tanggal >= k.Mulai`). Bila kartu belum dimulai, pesannya `"mulai kartu stok dulu (hitung fisik pertama) sebelum mencatat penjualan/pembelian barang"` — juga bila barang diisi tapi kartu belum dimulai. Transaksi bertanggal sebelum `k.Mulai` → nil. Dipanggil di create/update transaksi (HTTP 400). Tidak dipanggil di `lunasiTxLocked`.
- Produces (frontend): `<PilihRincianBarang jenis nominal value onChange barang wajib />` dengan `value: { rincian: RincianBarang[]; bukan_barang: boolean }`; baris = select barang (opsi terakhir `"+ Barang baru"`), qty (`parseQty`), nilai (hanya untuk Beli), tombol tambah/hapus baris, centang `"Bukan barang (jasa)"`. `<BarangBaruDialog pilihanAkun onSelesai(barang) onBatal />` membuat satu barang (nama, satuan, akun persediaan; SKU via `buatSKU`) lewat `createStok`.
- Produces: `lib/rincian.ts` — `sisaNominal(nominal, rincian)`, `rincianSah(value, wajib)` (pesan galat sama dengan server).

- [ ] **Step 1: Tes** `TestWajibBarang_PerpetualTolakTanpaBarang` (POST Penjualan tanpa barang sesudah mulai kartu → 400 berpesan persis), `_BukanBarangBoleh`, `_PeriodikBoleh`, `_KartuBelumDimulaiDitolak` (perpetual tanpa kartu → 400 pesan "mulai kartu stok dulu…"), `_SebelumMulaiKartuBoleh`, `_PelunasanTransaksiLamaTetapBoleh` (Review Focus 2), `_EditLamaWajibLengkap`; vitest `rincian.test.ts` (sisa nominal, rincianSah dengan/ tanpa wajib).
- [ ] **Step 2:** → FAIL
- [ ] **Step 3:** Implementasi server; ganti select barang tunggal di Transaksi.tsx dan dialog beli di KartuStok.tsx dengan `PilihRincianBarang`; prop `wajib` = perpetual dan tanggal ≥ mulai (dari `getPilihanBarang().mulai` + profil).
- [ ] **Step 4:** `go test ./cmd/server -run 'WajibBarang|Transaksi|KartuStok'`; `cd frontend && npx tsc -b && npx vitest run && npx oxlint src` (24)
- [ ] **Step 5: Commit** `feat(transaksi): wajib barang di perpetual, rincian multi-barang & barang baru di tempat`

### Task 4: Stok tidak boleh minus (perpetual)

**Files:**
- Create: `cmd/server/tanpa_minus.go`
- Modify: `cmd/server/main.go` (create/update/delete transaksi), `cmd/server/kartustok.go` (`handleStokOpname` :180, produksi :207, hapus gerak), `cmd/server/produksi_catat.go`
- Test: `cmd/server/tanpa_minus_test.go`

**Interfaces:**
- Consumes: `BarisKartu.Minus`, `kartuLocked()`, `finance.HitungKartuMetode`.
- Produces: `func (s *tenant) cekTanpaMinusLocked(txs []finance.Transaction, gerak []finance.GerakStok) error` — hitung kartu dari keadaan **usulan** (`txs`, `gerak`); bila perpetual dan ada barang yang punya baris `Minus` di kartu usulan tetapi tidak minus di kartu sekarang (pada tanggal yang sama atau sesudahnya), galat `"stok <nama> tidak cukup pada <tanggal ID>: tersedia <qty> <satuan>, dibutuhkan <qty> <satuan>"` (qty memakai `formatQty`; tanggal `finance.FormatTanggalID`). Periodik → nil. Dipanggil sebelum menyimpan di semua jalur yang mengubah kartu: transaksi (create/update/delete), opname, produksi, hapus dokumen gerak, dan oleh Task 5–9 di jalurnya masing-masing (HTTP 409).
- Frontend: tidak ada komponen baru; galat server tampil apa adanya di form.

- [ ] **Step 1: Tes** `TestTanpaMinus_PenjualanMelebihiStokDitolak` (stok awal Kaos 5, jual 6 → 409 pesan persis), `_HapusPembelianYangDipakaiDitolak`, `_ProduksiBahanKurangDitolak`, `_MinusLamaTidakMemblokir` (data lama sudah minus, penjualan barang lain tetap boleh), `_PeriodikBoleh`.
- [ ] **Step 2:** → FAIL · **Step 3:** Implementasi · **Step 4:** `go test ./internal/... ./cmd/server` → PASS (tes lama yang sengaja membuat minus pada perpetual diperbarui dengan sengaja; daftar perubahannya di laporan)
- [ ] **Step 5: Commit** `feat(stok): perpetual menolak stok minus`

### Task 5: Catat cepat & foto nota

**Files:**
- Modify: `cmd/server/catat.go` (`catatReq` :19, `susunTxCatat` :80), `internal/coa/catat.go` (`PecahCatat` :93)
- Modify: `frontend/src/views/CatatJurnal.tsx` (bacaNota :120)
- Test: `cmd/server/catat_test.go`, `frontend/src/views/catatNota.test.ts`

**Interfaces:**
- Consumes: `cekBarangLocked`, `PilihRincianBarang` (Task 3).
- `catatReq` mendapat `Rincian []finance.RincianBarang \`json:"rincian,omitempty"\``, `BukanBarang bool \`json:"bukan_barang,omitempty"\``; ditempelkan ke satu-satunya transaksi Penjualan/Beli hasil `PecahCatat`; lebih dari satu → 400 `"catatan ini berisi lebih dari satu penjualan/pembelian barang; pecah jadi beberapa catatan"`. `SusunCatat` mengembalikan rincian untuk layar ubah.
- Frontend: `PilihRincianBarang` muncul bila baris catatan menyentuh akun persediaan/HPP (debit) atau pendapatan usaha (kredit). Hasil foto nota mengisi rincian dengan mencocokkan `item[].nama` ke nama barang (persis, tanpa beda huruf besar); yang tidak cocok dibiarkan kosong dengan saran `"+ Barang baru"`.

- [ ] **Step 1: Tes** `TestCatat_PerpetualPenjualanWajibBarang`, `TestCatat_RincianTersimpanDanTampilSaatUbah`, `TestCatat_DuaTransaksiBerbarangDitolak`; vitest `cocokkanNota` (nama sama beda huruf → cocok; tidak ada → kosong).
- [ ] **Step 2:** → FAIL · **Step 3:** Implementasi · **Step 4:** tes server + frontend (tsc, vitest, oxlint 24) → PASS
- [ ] **Step 5: Commit** `feat(catat): rincian barang di catat cepat & foto nota`

### Task 6: Impor Excel transaksi

**Files:**
- Modify: `internal/finance/import.go` (`ParseImport` :160, `column` :96, template), `cmd/server/main.go` (`handleImportTransactions` :1238, `handleTransactionsTemplate`)
- Modify: `frontend/src/views/ImportDialog.tsx` (tampilkan galat per baris apa adanya)
- Test: `internal/finance/import_test.go`, `cmd/server/import_barang_test.go`

**Interfaces:**
- Kolom baru opsional `barang` (SKU atau nama persis, tanpa beda huruf besar) dan `jumlah_barang` (desimal, jadi per seribu). Nilai `jasa` di kolom `barang` → `BukanBarang`. Pencocokan barang di server (`ParseImport` hanya membawa teks: field baru `ImportRow`/hasil dengan `BarangTeks`, `JumlahBarang`), lalu `cekBarangLocked` per baris; galat per baris `"baris N: barang \"X\" tidak ada di daftar barang"` atau pesan wajib.
- Template Excel menambah dua kolom itu dengan contoh.

- [ ] **Step 1: Tes** `TestImporTransaksi_BarangDariSKUDanNama`, `_PerpetualTanpaBarangGalatPerBaris`, `_JasaBoleh`, `_KolomLamaTetapJalan` (berkas tanpa kolom baru di periodik → hasil sama dengan sekarang).
- [ ] **Step 2:** → FAIL · **Step 3:** Implementasi · **Step 4:** → PASS
- [ ] **Step 5: Commit** `feat(impor): barang & jumlah di impor Excel transaksi`

### Task 7: Impor mutasi — barang per baris

**Files:**
- Modify: `internal/finance/mutasi.go` (`MutasiRow` :30, `MutasiKeTransaksi` :508), `cmd/server/mutasi.go` (commit :230)
- Modify: `frontend/src/views/MutasiDialog.tsx`
- Test: `internal/finance/mutasi_test.go`, `cmd/server/mutasi_barang_test.go`

**Interfaces:**
- `MutasiRow` mendapat `Rincian []RincianBarang \`json:"rincian,omitempty"\``, `BukanBarang bool \`json:"bukan_barang,omitempty"\``; `MutasiKeTransaksi` menyalinnya; commit memanggil `cekBarangLocked` per transaksi baru (galat per baris, baris lain tetap tersimpan seperti perilaku galat sekarang).
- UI: baris berjenis Penjualan/Beli (dan tidak `lunasi`/lewati) menampilkan tombol `"Barang…"` yang membuka `PilihRincianBarang` dalam lembar/dialog; ringkasan baris menampilkan jumlah barang atau "jasa"; tombol Bukukan nonaktif dengan alasan bila ada baris wajib yang belum lengkap.

- [ ] **Step 1: Tes** `TestMutasiKeTransaksi_MembawaRincian`, `TestMutasi_PerpetualTanpaBarangGalatPerBaris`, `TestMutasi_DuplikatLewatiTidakMengubahKartu` (Review Focus 4).
- [ ] **Step 2:** → FAIL · **Step 3:** Implementasi · **Step 4:** → PASS
- [ ] **Step 5: Commit** `feat(mutasi): pilih barang per baris mutasi`

### Task 8: Mutasi "sama" — pindah akun bayar

**Files:**
- Modify: `cmd/server/mutasi.go` (`mutasiCommitReq`, commit), `internal/finance/mutasi.go` (`MutasiRow`), `frontend/src/views/MutasiDialog.tsx` (:238-267, :316)
- Test: `cmd/server/mutasi_pindah_test.go`

**Interfaces:**
- `mutasiCommitReq` mendapat `Rekening string \`json:"rekening,omitempty"\`` (kode akun rekening bank; kosong = `rekeningBawaanLocked()`; harus akun berjenis rekening, divalidasi seperti `BayarDari` di `produksi_catat.go`).
- `MutasiRow` mendapat `Pindah int \`json:"pindah,omitempty"\`` = ID transaksi manual yang akun bayarnya dipindah. Commit: verifikasi ulang dengan `CariMirip` (sama seperti `Lunasi`), tolak bila transaksi itu ber-InvoiceID/Grup atau bulannya ditutup; set `Lawan = rekening`; tidak membuat transaksi baru. Respons commit mendapat `dipindah int \`json:"dipindah"\``.
- UI: pilih rekening di atas tabel (bawaan terpilih); aksi baris duplikat ke-4 `"Sama — pindah akun bayar ke <nama rekening>"` hanya bila `mirip` bukan invoice, akun bayarnya berbeda dari rekening terpilih. Pratinjau mengirim `mirip.lawan` (tambah field `Lawan` ke `Mirip` di `internal/finance/mirip.go`).

- [ ] **Step 1: Tes** `TestMutasi_SamaPindahAkunBayar` (catatan manual dibayar Kas Besar 1102; mutasi rekening 1111 jawab pindah → Lawan 1111, jumlah transaksi tetap, saldo kas & bank bergeser), `_PindahTransaksiInvoiceDitolak`, `_PindahBulanTutupDitolak`.
- [ ] **Step 2:** → FAIL · **Step 3:** Implementasi · **Step 4:** → PASS (+ tsc/vitest/oxlint)
- [ ] **Step 5: Commit** `feat(mutasi): jawaban sama bisa memindah akun bayar ke rekening bank`

### Task 9: Invoice per baris

**Files:**
- Modify: `internal/finance/invoice.go` (`InvoiceItem` :49, `ValidateInvoice`), `cmd/server/invoice_buku.go` (`jagaTxInvoiceLocked` :42), `cmd/server/invoice.go` (create/update: panggil aturan wajib), `internal/finance/import_invoice.go` (`itemInvoice`), `cmd/server/schema.sql` (`invoice_items`), `cmd/server/pgstore.go` bila tulis/baca item eksplisit
- Modify: `frontend/src/views/Invoice.tsx` (:306-326)
- Test: `cmd/server/invoice_barang_test.go`, `internal/finance/import_invoice_test.go`, `cmd/server/pgstore_test.go`

**Interfaces:**
- `InvoiceItem` mendapat `StokID int \`json:"stok_id,omitempty"\``, `BukanBarang bool \`json:"bukan_barang,omitempty"\``. Validasi: keduanya tidak bersamaan.
- Transaksi Penjualan invoice: `Rincian` = item ber-`StokID`, `QtyStok = qty × TakaranSkala`, `Nilai = qty × harga` dikurangi bagian diskon proporsional (pembulatan sisa ke baris terbesar) sehingga `Σ Nilai ≤ DPP`. `BukanBarang = true` bila semua item bukan barang.
- Aturan wajib (perpetual, tanggal ≥ mulai kartu, saat status bukan draft): tiap item harus `StokID` atau `BukanBarang`; galat 400 `"baris N: pilih barang atau tandai Bukan barang (jasa)"`. Invoice lama yang sudah dibukukan dan tidak diubah tidak diblokir saat sinkron.
- Impor invoice: kolom opsional `barang` (SKU/nama) per baris item; `jasa` = bukan barang.
- UI: tiap baris item mendapat select barang (+ `"+ Barang baru"` via `BarangBaruDialog`) dan pilihan `"Bukan barang (jasa)"`; memilih barang mengisi deskripsi, satuan, harga jual bila masih kosong.
- Postgres: `ALTER TABLE invoice_items ADD COLUMN IF NOT EXISTS stok_id INT`, `... bukan_barang BOOLEAN NOT NULL DEFAULT false`.

- [ ] **Step 1: Tes** `TestInvoice_PerpetualItemWajibBarang`, `TestInvoice_RincianIkutItemDanDiskon` (2 item barang + 1 jasa, diskon Rp10.000 → rincian 2 baris, Σ nilai = DPP − nilai jasa setelah diskon proporsional), `TestInvoice_UbahItemKartuIkut` (Review Focus 3), `TestInvoice_LamaTanpaBarangTetapSinkron`, `TestParseInvoiceFile_KolomBarang`, pgstore round-trip item `stok_id`.
- [ ] **Step 2:** → FAIL · **Step 3:** Implementasi · **Step 4:** → PASS (+ tsc/vitest/oxlint)
- [ ] **Step 5: Commit** `feat(invoice): barang per baris invoice`

### Task 10: Verifikasi akhir & PR

- [ ] **Step 1:** `go vet ./...`, `go test ./...` (hanya 2 kegagalan Windows), `cd frontend && npx tsc -b && npx vitest run && npm run build`, oxlint 24.
- [ ] **Step 2:** Data contoh Kedai Kopi Senja: laporan CoA & keuangan Juni–Oktober 2026 identik dengan `main` (probe sama seperti tahap 1).
- [ ] **Step 3:** Uji layar (headless, satu sesi): usaha manufaktur perpetual yang kartunya sudah dimulai — form Transaksi menolak penjualan tanpa barang, "+ Barang baru" membuat barang dan langsung terpilih, invoice dengan barang, mutasi dengan jawaban "sama — pindah akun bayar".
- [ ] **Step 4:** Commit, push, PR ke `main` (tujuan, perubahan per task, hasil tes, catatan aturan baru untuk pengguna perpetual).
