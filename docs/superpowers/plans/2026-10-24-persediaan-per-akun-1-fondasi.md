# Persediaan per Akun — Tahap 1: Fondasi & Jurnal — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Setiap barang punya akun persediaan dari CoA. Neraca menampilkan persediaan per akun yang selalu sama dengan nilai kartu stok. Pada metode perpetual, pembelian, produksi, penjualan (HPP) dan opname barang dijurnal per kejadian.

**Architecture:**
- Akun barang ditentukan saat jurnal dihitung: `StokItem.Akun` bila diisi dan sah, selain itu ditebak dari teks `Kategori` lama dengan mencocokkan nama akun persediaan CoA, lalu jatuh ke akun peran persediaan. Karena itu tidak perlu migrasi data.
- Jurnal tetap dihitung ulang di `jurnalPeriode` (`cmd/server/coa.go`):
  - saldo awal persediaan dipecah per akun;
  - perpetual mendapat baris jurnal dari baris kartu stok;
  - satu penyesuaian tanggal 28 diganti **penyesuaian sisa per akun persediaan** (target = nilai kartu per akun). Penyesuaian ini menyapu transaksi tanpa barang dan menutup bug "terhitung dua kali" di akun persediaan selain akun peran.

**Tech Stack:** Go 1.24+ (`cmd/server`, `internal/coa`, `internal/finance`), React 19 + TypeScript (`frontend/src`), vitest.

**Spec:** `docs/superpowers/specs/2026-10-24-persediaan-per-akun-design.md`

**Cakupan tahap ini:** keputusan 1–3 dan 6 di spec, keputusan pembuat soal data lama, biaya lain produksi, barang setengah jadi (1303), dan saldo awal persediaan.
**Bukan tahap ini:**
- Tahap 2: wajib pilih barang di semua jalur input, rincian beberapa barang per transaksi, invoice per baris, impor mutasi "sama → pindah akun bayar".
- Tahap 3: periodik wajib persediaan awal/akhir per akun dan blokir Tutup Buku.

## Global Constraints

- Akun persediaan = akun CoA dengan `Subtype == "inventory"`. `inventory_contra` (1309) **bukan** akun persediaan barang.
- Laba bersih CoA = laba laporan ringkas. Invariant tes `TestKartuStok_MasukLaporanKeuangan` dan `TestBukuBerjalan_CoASamaDenganRingkas` wajib tetap lolos tanpa mengubah angka harapannya.
- Total saldo akun persediaan pada akhir periode (kartu aktif) = `StokPeriode.PersediaanAkhir`.
- Usaha dengan satu akun persediaan (dagang) menghasilkan neraca & laba **identik** dengan sebelum perubahan.
- Periode sebelum `KartuStok.Mulai` dan usaha tanpa kartu stok: perilaku sekarang dipertahankan.
- Neraca saldo tiap periode tetap seimbang (debit = kredit).
- Commit tanpa trailer/atribusi AI (AGENTS.md); penulis commit = konfigurasi git repo.
- Go: `PATH="/c/Program Files/Go/bin:$PATH"`; tes `go test ./...`. Dua tes yang memang gagal di Windows (`TestStateSurvivesRestart`, `TestSimpanGagalBukanSukses`) dikecualikan dari penilaian.

## Review Focus

1. **Akun barang dihapus/berganti template** sehingga `StokItem.Akun` tidak ada lagi di CoA → barang jatuh ke akun peran persediaan, tanpa error dan tanpa saldo hilang (tes di Task 2).
2. **Pembulatan pembagian nilai hasil produksi** ke beberapa barang hasil → jurnal produksi tetap seimbang per dokumen produksi (tes di Task 5).
3. **Bulan sebelum kartu stok dimulai, atau usaha tanpa kartu** → laporan bulan itu sama persis dengan sebelum perubahan (tes di Task 4).
4. **Pembelian perpetual tanpa barang (`StokID == 0`)** → masuk akun peran persediaan, lalu disapu penyesuaian sisa ke HPP. Total persediaan tetap = kartu (tes di Task 5).
5. **Biaya lain produksi yang sengaja diberi akun beban** → tercatat sebagai beban, tidak menambah nilai barang hasil di kartu maupun jurnal (tes di Task 6).

---

### Task 1: Akun 1303 & fungsi bantu akun persediaan

**Files:**
- Modify: `internal/coa/manufaktur.go` (daftar akun sekitar baris 54–60)
- Create: `internal/coa/persediaan.go`
- Test: `internal/coa/persediaan_test.go`, perbarui hitungan akun di `internal/coa/template_test.go` (`TestManufaktur_TemplateSah`, 156 → 157)

**Interfaces:**
- Produces: `func AkunPersediaan(akun []Account) []Account`: akun `Subtype == "inventory"`, urutan asli.
- Produces: `func AkunPersediaanBarang(kategori string, akun []Account, peta Peta) string`: tebakan kode akun dari teks kategori lama.

- [ ] **Step 1: Tulis tes `TestAkunPersediaanBarang_Manufaktur` dan `TestAkunPersediaanBarang_Dagang`**

```go
func TestAkunPersediaanBarang_Manufaktur(t *testing.T) {
	peta := PetaDari(Manufaktur)
	for kat, mau := range map[string]string{
		"Bahan Baku": "1301", "bahan penolong": "1302", "Kemasan": "1302",
		"Barang Setengah Jadi": "1303", "Barang Dalam Proses": "1304",
		"Produk Jadi": "1305", "Barang Jadi": "1305", "Barang": "1301", "": "1301",
	} {
		if got := AkunPersediaanBarang(kat, Manufaktur, peta); got != mau {
			t.Errorf("%q → %s, mau %s", kat, got, mau)
		}
	}
	if n := len(AkunPersediaan(Manufaktur)); n != 7 { // 1301 1302 1303 1304 1305 1306 1307
		t.Errorf("akun persediaan manufaktur %d, mau 7", n)
	}
}

func TestAkunPersediaanBarang_Dagang(t *testing.T) {
	peta := PetaDari(Master)
	for _, kat := range []string{"Bahan Baku", "Produk Jadi", "Kemasan", "Barang"} {
		if got := AkunPersediaanBarang(kat, Master, peta); got != peta.Kode(PeranPersediaan) {
			t.Errorf("dagang %q → %s, mau akun peran", kat, got)
		}
	}
}
```

- [ ] **Step 2:** `go test ./internal/coa -run AkunPersediaan` → FAIL (fungsi belum ada)
- [ ] **Step 3: Tambah akun & fungsi.**
  - `m("1303", "Persediaan Barang Setengah Jadi", "inventory", Debit, AsetLancar, "Barang olahan yang disimpan dan dipakai lagi di produksi berikutnya")` setelah 1302.
  - `AkunPersediaanBarang` mencocokkan nama akun (huruf kecil) **dalam urutan ini**:
    1. "setengah jadi"
    2. "dalam proses"
    3. "jadi" (pada teks kategori) → akun bernama "barang jadi"
    4. "penolong" atau "kemasan" → akun bernama "penolong"/"kemasan"
    5. "bahan baku" → akun bernama "bahan baku"
    6. selain itu → `peta.Kode(PeranPersediaan)`
  - Hanya akun dari `AkunPersediaan(akun)` yang boleh dipilih.
- [ ] **Step 4:** `go test ./internal/coa` → PASS (termasuk `TestManufaktur_TemplateSah` dengan 157 akun, `TestTemplateSaldoAwal_Manufaktur`)
- [ ] **Step 5: Commit** `feat(coa): akun 1303 setengah jadi & penentu akun persediaan barang`

### Task 2: Akun per barang di data & Stok & Gudang

**Files:**
- Modify: `internal/finance/tagihan.go` (`StokItem`, `ValidateStok`)
- Modify: `cmd/server/` handler stok (`handleCreateStok`, `handleUpdateStok`, daftar stok GET)
- Create: `cmd/server/akun_stok.go`
- Modify: `frontend/src/lib/api.ts` (tipe `StokItem`), `frontend/src/views/PiutangStok.tsx` (form barang ~:316, :487), `frontend/src/views/TambahBarang.tsx` (:13–14)
- Test: `cmd/server/akun_stok_test.go`

**Interfaces:**
- Produces: `StokItem.Akun string \`json:"akun,omitempty"\``. Kosong = ditebak dari Kategori.
- Produces: `func (s *tenant) akunStokLocked(all []coa.Account, peta coa.Peta) func(stokID int) string`. Pemanggil memegang `s.mu`. Hasil: `it.Akun` bila ada di `coa.AkunPersediaan(all)`, selain itu `coa.AkunPersediaanBarang(it.Kategori, all, peta)`. StokID tak dikenal → akun peran.
- Produces (API): GET daftar stok menambah per item `akun_efektif` dan pada respons `pilihan_akun: [{kode,nama}]` (akun persediaan aktif).

- [ ] **Step 1: Tulis tes `TestStok_AkunPersediaan`**
  - Usaha manufaktur. Buat barang `{"nama":"Kain","kategori":"Bahan Baku"}` → `akun_efektif == "1301"`.
  - Ubah akun ke `"1303"` → `akun_efektif == "1303"`.
  - Buat barang dengan `"akun":"4101"` → HTTP 400 "akun persediaan tidak dikenal".
  - Simulasikan akun yang tidak ada lagi: set `Akun = "1399"` langsung di state → `akun_efektif` jatuh ke `"1301"` (Review Focus 1).
- [ ] **Step 2:** `go test ./cmd/server -run TestStok_AkunPersediaan` → FAIL
- [ ] **Step 3:** Implementasi field, validasi di handler (akun harus salah satu `coa.AkunPersediaan(all)` bila tidak kosong), `akunStokLocked`, dan field API.
  - Form barang di frontend mengganti datalist kategori dengan `<select>` akun persediaan (label "Kategori (akun persediaan)"), berisi `pilihan_akun`. Nilai awal = `akun_efektif`.
  - Bila hanya ada satu akun persediaan, select disembunyikan.
- [ ] **Step 4:** `go test ./cmd/server -run 'TestStok|TestKartuStok'` → PASS. Lalu `cd frontend && npx tsc -b && npx vitest run` → PASS.
- [ ] **Step 5: Commit** `feat(stok): kategori barang = akun persediaan CoA`

### Task 3: Nilai kartu stok per akun

**Files:**
- Modify: `internal/finance/stok_laporan.go`
- Test: `internal/finance/stok_laporan_test.go`

**Interfaces:**
- Produces: `func (k KartuStok) NilaiPerAkun(sampai string, akunOf func(stokID int) string) map[string]int64`: nilai saldo tiap barang pada akhir tanggal `sampai` (`SaldoNilai` baris terakhir ≤ sampai), dijumlah per akun. Bila `Mulai == ""` atau `Mulai > sampai` → map kosong.

- [ ] **Step 1: Tulis tes `TestNilaiPerAkun`**
  - Tiga barang: id 1 → "1301", id 2 → "1301", id 3 → "1305".
  - Baris kartu dengan SaldoNilai akhir 100, 50, 200 sebelum tanggal batas, dan satu baris id 3 sesudah batas bernilai 999.
  - Hasil: `{"1301":150, "1305":200}`; jumlahnya = `k.Rentang(dari, sampai).PersediaanAkhir`.
- [ ] **Step 2:** `go test ./internal/finance -run TestNilaiPerAkun` → FAIL
- [ ] **Step 3:** Implementasi.
- [ ] **Step 4:** → PASS
- [ ] **Step 5: Commit** `feat(stok): nilai kartu stok per akun persediaan`

### Task 4: Saldo awal per akun & penyesuaian sisa per akun (perpetual & periodik)

**Files:**
- Modify: `internal/coa/journal.go` (`OpeningBerjalan`, `PenyesuaianPersediaan`)
- Modify: `internal/coa/reports.go` (`IncomeStatementKartu`)
- Modify: `cmd/server/coa.go` (`jurnalPeriode` baris 121–183), `cmd/server/main.go` (`openingFor`)
- Test: `cmd/server/persediaan_akun_test.go`; perbarui pemanggil `OpeningBerjalan` di `internal/coa/journal_test.go`

**Interfaces:**
- Consumes: `akunStokLocked` (Task 2), `NilaiPerAkun` (Task 3), `AkunPersediaan` (Task 1).
- Produces: `OpeningBerjalan(kas, piutang int64, persediaan map[string]int64, utang, pinjaman, labaDitahan int64, assetAwal, lain map[string]int64, peta Peta, p finance.BusinessProfile) map[string]int64`. `persediaan` per kode akun; jumlahnya ikut `bersih`.
- Produces: `PenyesuaianPersediaan(tanggal, akun string, selisih int64, p finance.BusinessProfile, peta Peta) []Line`, dengan akun persediaan sebagai parameter.
- Produces: `IncomeStatementKartu` menghitung persediaan awal/akhir sebagai **jumlah semua akun `Subtype == "inventory"`** di neraca saldo, bukan hanya akun peran.

**Aturan saldo awal persediaan per akun** (`persediaanAwalPerAkunLocked(period, all, peta) map[string]int64` di `cmd/server/coa.go`):
1. Kartu aktif pada periode itu → `k.NilaiPerAkun(hari terakhir bulan sebelumnya, akunOf)`. Bila bulan memuat titik nol → nilai baris `KartuAwal` per akun.
2. Selain itu → baris `KatPersediaan` saldo awal (`ob.Rincian`) per `Kode` bila kodenya akun persediaan, sisanya (dan `ob.Persediaan` tanpa rincian) ke akun peran.

Jumlah map selalu = nilai lama `opening.Inventory`.

**Aturan penyesuaian sisa:**
- Hanya bila kartu aktif.
- Untuk tiap akun persediaan: `target = NilaiPerAkun(akhir bulan)[kode]`, `selisih = target − saldo akun itu di neraca saldo` → `PenyesuaianPersediaan(period+"-28", kode, selisih, …)`, dengan ket "Penyesuaian persediaan ke nilai kartu stok".
- `j.penyesuaian` = jumlah semua selisih (untuk `IncomeStatementKartu`).

- [ ] **Step 1: Tulis tes**
  - `TestPersediaanAkun_PeriodikDuaAkun`:
    - Manufaktur periodik.
    - Kartu dimulai dengan Kain (Bahan Baku) 10 × Rp10.000 dan Kaos (Produk Jadi) 5 × Rp40.000.
    - Beli Kain 5 × Rp10.000 (StokID Kain).
    - Akhir bulan: saldo 1301 = 150.000 dan 1305 = 200.000 di neraca saldo (`/api/v1/reports/coa`); laba CoA = laba ringkas; neraca saldo seimbang.
  - `TestPersediaanAkun_DobelTertutup`:
    - Kartu aktif.
    - Catat cepat/transaksi dengan `akun:"1302"` sebesar 300.000 tanpa barang.
    - Total semua akun persediaan = `PersediaanAkhir` kartu (bukan +300.000).
  - `TestPersediaanAkun_SebelumKartuSama`:
    - Data contoh dagang: laporan CoA bulan sebelum titik nol dan bulan berjalan identik dengan nilai yang dihitung sebelum perubahan.
    - Pakai angka dari `TestBukuBerjalan_CoASamaDenganRingkas`, yang harus tetap lolos tanpa diubah (Review Focus 3).
- [ ] **Step 2:** `go test ./cmd/server -run TestPersediaanAkun` → FAIL
- [ ] **Step 3:** Implementasi aturan di atas. Ganti blok kartu di `jurnalPeriode` dan panggil `OpeningBerjalan` dengan map.
- [ ] **Step 4:** `go test ./internal/... ./cmd/server` → PASS, termasuk `TestKartuStok_MasukLaporanKeuangan`, `TestBukuBerjalan_CoASamaDenganRingkas`, `TestJurnalUmum_*`, `TestBukuBesar_*`, `TestSaldoMinus_*`.
- [ ] **Step 5: Commit** `feat(jurnal): saldo awal & penyesuaian persediaan per akun`

### Task 5: Jurnal perpetual per kejadian (beli, produksi, jual, opname)

**Files:**
- Modify: `internal/coa/journal.go` (`JournalBerjalan`)
- Create: `internal/coa/jurnal_kartu.go`
- Modify: `cmd/server/coa.go` (`jurnalPeriode`)
- Test: `internal/coa/jurnal_kartu_test.go`, `cmd/server/persediaan_akun_test.go`

**Interfaces:**
- Consumes: `akunStokLocked` (Task 2), `PenyesuaianPersediaan` per akun (Task 4).
- Produces: `JournalBerjalan(txs []finance.Transaction, period string, p finance.BusinessProfile, peta Peta, akunStok func(stokID int) string) []Line`. Untuk perpetual, transaksi `Beli Bahan / Stok` dengan `StokID > 0` dan `Akun == ""` didebit `akunStok(StokID)`. `akunStok` boleh nil (perilaku lama).
- Produces: `func JurnalKartu(baris []finance.BarisKartu, dari, sampai string, akunStok func(int) string, peta Peta) []Line`, hanya untuk perpetual, baris bertanggal [dari, sampai]:
  - `KartuJual`: Dr `PeranHPP` / Cr akun barang sebesar `-Nilai`, `TxID = TransaksiID`.
  - `KartuOpname`: `Nilai > 0` → Dr akun barang / Cr `PeranHPP`; `< 0` → kebalikannya.
  - Produksi, per `GerakID`:
    - `KartuProduksiPakai`: Cr akun barang sebesar `-Nilai`.
    - `KartuProduksiMasuk`: Dr akun barang sebesar `Nilai − Biaya`.
    - Bila debit ≠ kredit karena pembulatan, selisih ditambahkan ke baris debit terbesar.
  - `KartuAwal`, `KartuBeli`: tanpa baris (sudah lewat saldo awal / transaksi).
  - `Koreksi`: tanpa baris (disapu penyesuaian sisa).
- Produces (ket Jurnal Umum): produksi "Produksi — <keterangan gerak atau nama produk>", jual "HPP — <ket transaksi>", opname "Selisih stok opname". Baris kartu tidak punya `TransaksiID` untuk produksi/opname (TxID 0), jadi muncul sebagai bukti `SIS-…`.

- [ ] **Step 1: Tulis tes**
  - `TestJurnalKartu_ProduksiBertahap` (unit, `internal/coa`):
    - Baris kartu: pakai Kain 1301 −60.000; masuk Potongan 1303 +80.000 dengan Biaya 20.000 (gerak 1); pakai Potongan 1303 −80.000 dan Benang 1302 −10.000; masuk Kaos 1305 +90.000 (gerak 2); jual Kaos −45.000 (TransaksiID 9); opname Benang −1.000.
    - Harapan per akun (debit − kredit): 1301 −60.000, 1303 −20.000 (biaya 20.000 dijurnal oleh transaksi biayanya sendiri, bukan oleh `JurnalKartu`), 1302 −11.000, 1305 +45.000, HPP 5101 +46.000.
    - Total debit = total kredit = 196.000.
    - Gerak 1 seimbang tanpa biaya, yaitu Dr 1303 60.000 = Cr 1301 60.000.
    - Baris HPP punya TxID 9.
  - `TestJurnalKartu_PembulatanSeimbang`: satu gerak dengan dua hasil bernilai 33.333 dan 33.334 dari bahan 66.667 → total debit = total kredit (Review Focus 2).
  - `TestPersediaanAkun_PerpetualAlurManufaktur` (server, manufaktur perpetual, kartu dimulai):
    - Beli Kain (1301); catat produksi setengah jadi dengan biaya lain 20.000; produksi barang jadi; jual sebagian.
    - Saldo 1301/1303/1305 = `NilaiPerAkun`.
    - Jurnal Umum memuat baris HPP bertanggal **tanggal penjualan** (bukan tanggal 28).
    - Penyesuaian sisa = 0 untuk semua akun.
    - Laba CoA = laba ringkas.
  - `TestPersediaanAkun_PerpetualBeliTanpaBarang`: pembelian tanpa StokID 100.000 → total persediaan tetap = kartu, selisih tersapu ke HPP (Review Focus 4).
- [ ] **Step 2:** `go test ./internal/coa ./cmd/server -run 'JurnalKartu|PersediaanAkun'` → FAIL
- [ ] **Step 3:** Implementasi `JurnalKartu`, parameter baru `JournalBerjalan`. Di `jurnalPeriode`, perpetual + kartu aktif menambah baris `JurnalKartu` sebelum menghitung penyesuaian sisa.
- [ ] **Step 4:** `go test ./internal/... ./cmd/server` → PASS (semua tes invariant di Global Constraints)
- [ ] **Step 5: Commit** `feat(jurnal): jurnal perpetual per pembelian, produksi, penjualan, opname`

### Task 6: Biaya lain produksi ke akun barang hasil

**Files:**
- Modify: `cmd/server/produksi_catat.go` (:73 `akunBiaya`, :120–145)
- Test: `cmd/server/produksi_catat_test.go`

**Interfaces:**
- Consumes: `akunStokLocked` (Task 2).
- Behaviour:
  - Pada perpetual, akun bawaan biaya lain = akun persediaan **barang hasil pertama** dokumen produksi itu; periodik tetap `AutoAccount("Beli Bahan / Stok")`.
  - Biaya dengan `Akun` pilihan pengguna yang **bukan** akun persediaan tidak ditambahkan ke `g.Biaya` (tidak dikapitalisasi).

- [ ] **Step 1: Tulis tes**
  - `TestCatatProduksi_BiayaKeAkunHasil`: perpetual; produksi Kaos (1305) dengan biaya 20.000 tanpa akun → transaksi biaya berakun "1305"; `g.Biaya == 20000`.
  - `TestCatatProduksi_BiayaAkunBebanTidakDikapitalisasi`: biaya 15.000 dengan `akun` = akun beban upah di CoA → transaksi berakun beban itu; `g.Biaya == 0`; nilai Kaos di kartu = nilai bahan saja (Review Focus 5).
  - Perbarui harapan `TestCatatProduksi_LangsungTanpaKartuStok` bila menyebut akun 1301 secara eksplisit.
- [ ] **Step 2:** → FAIL
- [ ] **Step 3:** Implementasi.
- [ ] **Step 4:** `go test ./cmd/server -run 'Produksi|PersediaanAkun|KartuStok'` → PASS
- [ ] **Step 5: Commit** `feat(produksi): biaya lain ke akun barang hasil; akun beban tidak dikapitalisasi`

### Task 7: Saldo awal persediaan per akun (diisi otomatis bila ada daftar barang)

**Files:**
- Modify: `cmd/server/onboarding.go` (`openingResp`, `openingSummary`), `cmd/server/cek_saldo_awal.go` (:70–90), `cmd/server/onboarding.go` `handleOnboardingImport` (remap baris persediaan)
- Modify: `frontend/src/views/SaldoAwal.tsx`, `frontend/src/lib/api.ts`
- Test: `cmd/server/onboarding_test.go` / `cmd/server/cek_saldo_awal_test.go`

**Interfaces:**
- Produces (API `GET /api/v1/opening`): `persediaan_akun: SaldoAkun[]`. Isinya nilai daftar barang (`jumlah × harga_pokok`) per akun efektif, atau nilai hitung fisik pertama per akun bila kartu sudah dimulai. Kosong bila tidak ada daftar barang.
- Behaviour form Saldo Awal:
  - Bila `persediaan_akun` tidak kosong **dan** saldo awal belum menyimpan baris persediaan bernilai, baris persediaan diganti baris terkunci (gaya baris aset tetap, garis putus) dengan keterangan "dari daftar barang · ubah di Stok & Gudang".
  - Bila pengguna sudah menyimpan nilai persediaan, nilai itu tetap; cek kecocokan menampilkan selisih **per akun** (`cekKecocokan` satu entri per akun yang berbeda).
- Behaviour impor data awal: baris persediaan hasil `ParseOnboarding` (sekarang `Kode: "1301"`) diberi kode `coa.AkunPersediaanBarang(kategori, all, peta)` dari teks "Persediaan — <kategori>".

- [ ] **Step 1: Tulis tes**
  - `TestSaldoAwal_PersediaanPerAkunDariDaftarBarang`: barang Kain (1301, 10 × 10.000) dan Kaos (1305, 5 × 40.000) → `persediaan_akun` = `[{1301,100000},{1305,200000}]`.
  - `TestSaldoAwal_PersediaanTersimpanTidakDitimpa`: saldo awal tersimpan 1301 = 120.000 → GET opening tetap 120.000 di rincian; cek kecocokan memuat selisih 20.000 untuk 1301.
  - `TestImporDataAwal_PersediaanKeAkunKategori`: workbook dengan persediaan "Barang Jadi" → baris berkode 1305.
- [ ] **Step 2:** → FAIL
- [ ] **Step 3:** Implementasi server dan form (pakai `sisipkanUrutKode` / pola baris terkunci yang sudah ada di `SaldoAwal.tsx`).
- [ ] **Step 4:** `go test ./cmd/server -run 'SaldoAwal|ImporDataAwal|Onboarding'` → PASS; `cd frontend && npx tsc -b && npx vitest run && npx oxlint src` (peringatan tidak bertambah dari 24).
- [ ] **Step 5: Commit** `feat(saldo-awal): persediaan per akun dari daftar barang`

### Task 8: Verifikasi akhir & PR

- [ ] **Step 1:** `go test ./...` (hanya 2 kegagalan Windows yang dikenal), `cd frontend && npx tsc -b && npx vitest run && npm run build`, oxlint ≤ 24 peringatan.
- [ ] **Step 2:** Data contoh lokal (Kedai Kopi Senja, manufaktur):
  - neraca menampilkan saldo per akun persediaan;
  - total = nilai kartu;
  - tidak ada peringatan saldo minus baru di akun persediaan;
  - Laba Rugi Agustus sama dengan sebelum perubahan (catat angka sebelum mulai di ledger).
- [ ] **Step 3:** Usaha manufaktur kosong: Stok & Gudang memilih kategori dari akun persediaan; Saldo Awal menampilkan baris persediaan terkunci per akun.
- [ ] **Step 4:** Commit, push, PR ke `main` (deskripsi: tujuan, perubahan per task, hasil tes, catatan "laporan bulan lama ikut berubah hanya di bagian yang sebelumnya salah/terhitung dua kali").
