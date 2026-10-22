# Backend Schema — Fitur lanjutan checklist tester (Okt 2026)

**Status:** Draft, 22 Okt 2026 · **Basis:** `cmd/server/schema.sql` dan `cmd/server/pgstore.go` di `origin/main` d9b467c · **Input:** [TRD](TRD.md)

## 1. Aturan skema yang wajib diikuti

1. **Idempoten.** Semua perubahan ditambahkan di akhir `cmd/server/schema.sql` dengan
   `CREATE TABLE IF NOT EXISTS` / `ALTER TABLE … ADD COLUMN IF NOT EXISTS`. Berkas ini dijalankan setiap boot.
2. **Nama kolom = tag JSON struct Go.** `pgstore.go` membaca/menulis lewat `to_jsonb` /
   `jsonb_populate_recordset`, tanpa pemetaan manual.
3. **Uang `BIGINT` rupiah utuh, tanggal `DATE`, waktu `TIMESTAMPTZ`, tarif basis poin `INT`.**
4. **Data usaha ditulis ulang utuh.** `saveTenantDB` menghapus baris `tenants` lalu menulis semua tabel anak.
   Tabel usaha baru **wajib** `tenant_id INT NOT NULL REFERENCES tenants ON DELETE CASCADE`, kunci
   `(tenant_id, id)`, didaftarkan di daftar tabel `saveTenantDB`/`loadTenantDB`, dan penghitungnya
   (`next_<x>_id`) ditambahkan ke `tenants` serta `tenantRow` (struct posisional, urutan harus sama).
5. **Data lintas usaha** (users, sessions, reset password) tidak memakai FK ke `users`, karena
   `saveAuthDB` menulis ulang tabel `users`; baris yatim dibersihkan saat boot (pola `primbon_bookmark`).

## 2. Ringkasan perubahan

| Fitur | Tabel | Perubahan |
|---|---|---|
| F-01…F-07 | — | Tidak ada perubahan skema |
| F-08 Pindah Dana | `transactions` | Tidak ada kolom baru (`akun` = tujuan, `lawan` = asal); jenis baru di `finance.JenisList` |
| F-09 Nomor jurnal | `transactions`, `tenants` | `no_jurnal TEXT`, `tenants.nomor_jurnal JSONB` |
| F-10 Pelanggan | **`pelanggan`** (baru), `invoices`, `tenants` | tabel baru, `invoices.pelanggan_id INT`, `tenants.next_pelanggan_id` |
| F-11 Riwayat pihak | `transactions`, `tagihan` | `pihak_id INT`, `pihak_jenis TEXT` |
| F-12 Selisih persediaan | — (CoA di kode) | akun `5106`; data opname sudah di `gerak_stok` |
| F-13 Per tanggal | — | dihitung dari `assets` dan `gerak_stok` |
| F-14 Saldo awal dari daftar | `tagihan` | `saldo_awal BOOLEAN`, `tanggal DATE` |
| F-15 Menu Tagihan | — | Hanya frontend dan daftar view peran |
| F-16 Lupa password | **`password_resets`** (baru) | tabel lintas usaha |

## 3. DDL (tambahkan di akhir `schema.sql`)

```sql
-- F-09 Nomor jurnal (Okt 2026): JU-YYYYMM-NNNN per usaha per bulan, tidak dipakai ulang.
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS no_jurnal TEXT;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS nomor_jurnal JSONB; -- {"202610": 12}
CREATE UNIQUE INDEX IF NOT EXISTS transactions_no_jurnal
	ON transactions (tenant_id, no_jurnal) WHERE no_jurnal IS NOT NULL AND grup IS NULL;

-- F-10 Pelanggan (Okt 2026): master pelanggan, pola sama dengan supplier.
CREATE TABLE IF NOT EXISTS pelanggan (
	tenant_id  INT NOT NULL REFERENCES tenants ON DELETE CASCADE,
	id         INT NOT NULL,
	nama       TEXT NOT NULL,
	kontak     TEXT,
	telepon    TEXT,
	email      TEXT,
	alamat     TEXT,
	npwp       TEXT,
	tempo_hari INT NOT NULL DEFAULT 0,
	catatan    TEXT,
	aktif      BOOLEAN NOT NULL DEFAULT TRUE,
	PRIMARY KEY (tenant_id, id)
);
CREATE UNIQUE INDEX IF NOT EXISTS pelanggan_nama ON pelanggan (tenant_id, lower(nama));
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS next_pelanggan_id INT NOT NULL DEFAULT 1;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS pelanggan_id INT;

-- F-11 Riwayat per pihak: tautan opsional ke master (kosong = cocokkan nama).
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS pihak_id INT;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS pihak_jenis TEXT; -- pelanggan | supplier
ALTER TABLE tagihan ADD COLUMN IF NOT EXISTS pihak_id INT;
CREATE INDEX IF NOT EXISTS transactions_pihak ON transactions (tenant_id, pihak_jenis, pihak_id);

-- F-14 Piutang/utang saldo awal dirinci per pihak sebagai tagihan.
ALTER TABLE tagihan ADD COLUMN IF NOT EXISTS saldo_awal BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE tagihan ADD COLUMN IF NOT EXISTS tanggal DATE; -- tanggal bukti asal (opsional)

-- F-16 Lupa password: token sekali pakai, disimpan sebagai hash. Tanpa FK ke users
-- (saveAuthDB menulis ulang users); baris user yang sudah dihapus dibuang saat boot.
CREATE TABLE IF NOT EXISTS password_resets (
	token_hash TEXT PRIMARY KEY,        -- hex SHA-256 dari token di tautan
	user_id    INT NOT NULL,
	dibuat     TIMESTAMPTZ NOT NULL,
	kedaluwarsa TIMESTAMPTZ NOT NULL,   -- dibuat + 30 menit
	dipakai    TIMESTAMPTZ,             -- terisi saat berhasil reset
	ip         TEXT
);
CREATE INDEX IF NOT EXISTS password_resets_user ON password_resets (user_id, dibuat);
```

> `no_jurnal` unik hanya untuk transaksi tanpa grup; anggota satu grup catatan berbagi nomor yang sama.

## 4. Struct Go

```go
// internal/finance/transaction.go — tambahan field Transaction
NoJurnal   string `json:"no_jurnal,omitempty"`   // F-09, diatur server
PihakID    int    `json:"pihak_id,omitempty"`    // F-11
PihakJenis string `json:"pihak_jenis,omitempty"` // "pelanggan" | "supplier"

// internal/finance/transaction.go — JenisList (F-08)
{"Pindah Dana", "keluar", "pindah"}, // Lawan = rekening asal, Akun = rekening tujuan

// internal/finance (berkas baru pelanggan.go) — F-10
type Pelanggan struct {
	ID        int    `json:"id"`
	Nama      string `json:"nama"`
	Kontak    string `json:"kontak,omitempty"`
	Telepon   string `json:"telepon,omitempty"`
	Email     string `json:"email,omitempty"`
	Alamat    string `json:"alamat,omitempty"`
	NPWP      string `json:"npwp,omitempty"`
	TempoHari int    `json:"tempo_hari"`
	Catatan   string `json:"catatan,omitempty"`
	Aktif     bool   `json:"aktif"`
}

// Invoice — tambahan
PelangganID int `json:"pelanggan_id,omitempty"`

// Tagihan — tambahan (F-11, F-14)
PihakID   int    `json:"pihak_id,omitempty"`
SaldoAwal bool   `json:"saldo_awal"`
Tanggal   string `json:"tanggal,omitempty"`

// tenant state / snapshot — tambahan
Pelanggan       []finance.Pelanggan `json:"pelanggan"`
NextPelangganID int                 `json:"next_pelanggan_id"`
NomorJurnal     map[string]int      `json:"nomor_jurnal"` // "202610" -> nomor terakhir

// internal/auth — F-16 (disimpan di tabel password_resets)
type resetRow struct {
	TokenHash   string     `json:"token_hash"`
	UserID      int        `json:"user_id"`
	Dibuat      time.Time  `json:"dibuat"`
	Kedaluwarsa time.Time  `json:"kedaluwarsa"`
	Dipakai     *time.Time `json:"dipakai,omitempty"`
	IP          string     `json:"ip,omitempty"`
}
```

**CoA (F-12)** — `internal/coa/coa.go`, kelompok HPP, berlaku untuk semua jenis usaha:

```go
a("5106", "Selisih Persediaan", "inventory_variance", Debit, HPP, Selalu,
	"Selisih hitung fisik (opname); debit = kurang, kredit = lebih"),
```
Peran baru `PeranSelisihPersediaan = "selisih_persediaan"` di `internal/coa/peran.go` dengan kode bawaan `5106`.
Usaha lama mendapat akun ini lewat `TambahanDiLuarFile`.

## 5. Endpoint baru dan berubah

| Metode & jalur | Fitur | Izin (`guard`) | Keterangan |
|---|---|---|---|
| `GET /api/v1/reports/daftar.{xlsx,pdf}?dari&sampai` | F-07 | laporan-keuangan | Sudah ada; frontend mulai mengirim rentang |
| `POST /api/v1/transactions` (jenis `Pindah Dana`) | F-08 | transaksi | Validasi `akun`/`lawan` kas/bank berbeda |
| `GET /api/v1/reports/jurnal?q=JU-202610-0001` (dan `.xlsx`) | F-09 | jurnal-umum | Sudah ada; `Bukti` = `no_jurnal`, pencarian ikut nomor |
| `GET/POST /api/v1/pelanggan`, `PUT/DELETE /api/v1/pelanggan/{id}` | F-10 | invoice | `DELETE` ditolak 409 bila dipakai → nonaktifkan |
| `POST /api/v1/pelanggan/import`, `POST /api/v1/pelanggan/dari-data` | F-10 | invoice | Impor Excel / ambil nama unik dari invoice & transaksi |
| `GET /api/v1/pihak/{jenis}/{id}/riwayat?dari&sampai[&format=xlsx]` | F-11 | transaksi | jenis = pelanggan \| supplier |
| `GET /api/v1/assets?pada=YYYY-MM-DD` | F-13 | laporan-keuangan | `pada` kosong = perilaku lama (`period`) |
| `GET /api/v1/stok?pada=YYYY-MM-DD` | F-13 | stok-gudang | Replay `gerak_stok` sampai `pada` |
| `GET/POST /api/v1/opening/tagihan`, `PUT/DELETE /api/v1/opening/tagihan/{id}` | F-14 | saldo-awal (sama dengan `PUT /api/v1/opening`) | Rincian piutang/utang awal. `POST /api/v1/tagihan` biasa sengaja tetap tidak ada (effc2a0) |
| `POST /api/v1/auth/lupa-password` `{email}` | F-16 | publik, rate limit | Selalu 202 dengan pesan sama |
| `POST /api/v1/auth/reset-password` `{token, password}` | F-16 | publik, rate limit | 204; cabut semua sesi user |

## 6. Migrasi data saat boot

| Fitur | Langkah | Idempoten karena |
|---|---|---|
| F-09 | Transaksi tanpa `no_jurnal` diberi nomor urut (tanggal, id) per bulan; `nomor_jurnal` = maksimum per bulan | Hanya mengisi yang kosong |
| F-10 | Tidak otomatis; pengguna menekan "Ambil dari data lama" | — |
| F-12 | Akun 5106 ditambahkan ke CoA usaha yang belum punya | Cek kode akun sebelum menambah |
| F-14 | Usaha belum tutup buku: `opening_balances.piutang/utang` > 0 tanpa rincian → satu baris `tagihan` saldo awal "Saldo awal (tanpa rincian)"; `opening_balances.pelunasan` dipindah menjadi pelunasan baris itu | Hanya bila belum ada `tagihan.saldo_awal` |
| F-16 | Hapus baris `password_resets` milik user yang sudah dihapus dan yang kedaluwarsa > 7 hari | Penghapusan saja |

Semua migrasi dijalankan setelah `pg_dump` manual sebelum deploy dan diuji dengan `TEST_DATABASE_URL`
(`cmd/server/pgstore_test.go`).
