-- Skema PostgreSQL Laporan Bisnis. Dijalankan setiap boot (idempoten).
-- Nama kolom = tag JSON struct Go, supaya baris bisa dibaca/ditulis lewat to_jsonb /
-- jsonb_populate_recordset tanpa pemetaan kolom manual (lihat pgstore.go).
-- Uang BIGINT rupiah utuh; tarif dalam basis poin (INT).

CREATE TABLE IF NOT EXISTS tenants (
	id               INT PRIMARY KEY,
	next_tx_id       INT NOT NULL,
	next_asset_id    INT NOT NULL,
	next_karyawan_id INT NOT NULL,
	next_invoice_id  INT NOT NULL,
	next_tagihan_id  INT NOT NULL DEFAULT 1,
	next_stok_id     INT NOT NULL DEFAULT 1,
	next_produk_id   INT NOT NULL DEFAULT 1,
	next_supplier_id INT NOT NULL DEFAULT 1,
	trial_sampai     TIMESTAMPTZ,
	aktif_sampai     TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS business_profiles (
	tenant_id             INT PRIMARY KEY REFERENCES tenants ON DELETE CASCADE,
	nama_usaha            TEXT,
	jenis_usaha           TEXT,
	bentuk_usaha          TEXT,
	ptkp                  TEXT,
	pkp                   BOOLEAN NOT NULL,
	metode_persediaan     TEXT,
	klu                   TEXT,
	omzet_tahunan         BIGINT,
	laba_sebelum_pajak    BIGINT,
	alamat                TEXT,
	kecamatan             TEXT,
	kota                  TEXT,
	provinsi              TEXT,
	npwp                  TEXT,
	telepon               TEXT,
	email                 TEXT,
	bank_nama             TEXT,
	bank_rekening         TEXT,
	bank_atas_nama        TEXT,
	ppn_bps               INT,
	penandatangan_nama    TEXT,
	penandatangan_jabatan TEXT
);

CREATE TABLE IF NOT EXISTS opening_balances (
	tenant_id     INT PRIMARY KEY REFERENCES tenants ON DELETE CASCADE,
	tanggal       DATE NOT NULL,
	kas           BIGINT NOT NULL,
	piutang       BIGINT NOT NULL,
	persediaan    BIGINT NOT NULL,
	utang         BIGINT NOT NULL,
	pinjaman      BIGINT NOT NULL,
	modal_disetor BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS transactions (
	tenant_id     INT NOT NULL REFERENCES tenants ON DELETE CASCADE,
	id            INT NOT NULL,
	tanggal       DATE NOT NULL,
	jenis         TEXT,
	nominal       BIGINT NOT NULL,
	keterangan    TEXT,
	item          TEXT,
	qty           INT,
	channel       TEXT,
	pihak         TEXT,
	lunas         BOOLEAN NOT NULL,
	tanggal_lunas DATE,
	akun          TEXT,
	PRIMARY KEY (tenant_id, id)
);
CREATE INDEX IF NOT EXISTS transactions_tenant_tanggal ON transactions (tenant_id, tanggal);

CREATE TABLE IF NOT EXISTS assets (
	tenant_id         INT NOT NULL REFERENCES tenants ON DELETE CASCADE,
	id                INT NOT NULL,
	nama              TEXT,
	jenis_harta       TEXT,
	kelompok          TEXT,
	metode            TEXT,
	tanggal_perolehan DATE NOT NULL,
	harga_perolehan   BIGINT NOT NULL,
	tak_berwujud      BOOLEAN NOT NULL,
	transaksi_id      INT,
	keterangan        TEXT,
	PRIMARY KEY (tenant_id, id)
);

CREATE TABLE IF NOT EXISTS karyawan (
	tenant_id     INT NOT NULL REFERENCES tenants ON DELETE CASCADE,
	id            INT NOT NULL,
	nama          TEXT,
	jabatan       TEXT,
	tanggal_masuk DATE NOT NULL,
	gaji_pokok    BIGINT NOT NULL,
	tunjangan     BIGINT NOT NULL,
	ptkp          TEXT,
	npwp          TEXT,
	aktif         BOOLEAN NOT NULL,
	PRIMARY KEY (tenant_id, id)
);

CREATE TABLE IF NOT EXISTS invoices (
	tenant_id        INT NOT NULL REFERENCES tenants ON DELETE CASCADE,
	id               INT NOT NULL,
	nomor            TEXT,
	no_po            TEXT,
	status           TEXT,
	tanggal          DATE NOT NULL,
	jatuh_tempo      DATE,
	pelanggan_nama   TEXT,
	pelanggan_alamat TEXT,
	pelanggan_npwp   TEXT,
	diskon_rp        BIGINT NOT NULL,
	ppn_bps          INT NOT NULL,
	catatan          TEXT,
	PRIMARY KEY (tenant_id, id),
	UNIQUE (tenant_id, nomor)
);

CREATE TABLE IF NOT EXISTS invoice_items (
	tenant_id    INT NOT NULL,
	invoice_id   INT NOT NULL,
	no           INT NOT NULL, -- urutan baris di faktur
	deskripsi    TEXT,
	qty          BIGINT NOT NULL,
	satuan       TEXT,
	harga_satuan BIGINT NOT NULL,
	PRIMARY KEY (tenant_id, invoice_id, no),
	FOREIGN KEY (tenant_id, invoice_id) REFERENCES invoices ON DELETE CASCADE
);

-- Piutang & Utang. jatuh_tempo DATE, bukan teks: kolom ini yang menentukan keterlambatan,
-- dan keterlambatan dihitung saat dibaca, bukan disimpan.
CREATE TABLE IF NOT EXISTS tagihan (
	tenant_id   INT NOT NULL REFERENCES tenants ON DELETE CASCADE,
	id          INT NOT NULL,
	nomor       TEXT,
	pihak       TEXT,
	jenis       TEXT NOT NULL, -- RECEIVABLE | PAYABLE
	jatuh_tempo DATE NOT NULL,
	nominal     BIGINT NOT NULL,
	lunas       BOOLEAN NOT NULL,
	keterangan  TEXT,
	PRIMARY KEY (tenant_id, id)
);

CREATE TABLE IF NOT EXISTS stok (
	tenant_id   INT NOT NULL REFERENCES tenants ON DELETE CASCADE,
	id          INT NOT NULL,
	sku         TEXT,
	nama        TEXT,
	kategori    TEXT,
	jumlah      BIGINT NOT NULL,
	satuan      TEXT,
	stok_min    BIGINT NOT NULL,
	harga_pokok BIGINT NOT NULL,
	harga_jual  BIGINT NOT NULL,
	PRIMARY KEY (tenant_id, id)
);

CREATE TABLE IF NOT EXISTS produk (
	tenant_id          INT NOT NULL REFERENCES tenants ON DELETE CASCADE,
	id                 INT NOT NULL,
	sku                TEXT,
	nama               TEXT,
	satuan             TEXT,
	kategori           TEXT,
	hasil_per_batch    BIGINT NOT NULL,
	biaya_tenaga_kerja BIGINT NOT NULL,
	biaya_overhead     BIGINT NOT NULL,
	harga_jual         BIGINT NOT NULL,
	catatan            TEXT,
	PRIMARY KEY (tenant_id, id)
);

-- Resep (BOM) per produk. takaran disimpan per seribu satuan bahan (lihat finance.TakaranSkala):
-- 180 berarti 0,18 kg. stok_id 0 = bahan di luar daftar stok, harganya di harga_manual.
CREATE TABLE IF NOT EXISTS produk_bahan (
	tenant_id    INT NOT NULL,
	produk_id    INT NOT NULL,
	no           INT NOT NULL, -- urutan baris di resep
	stok_id      INT NOT NULL,
	nama         TEXT,
	satuan       TEXT,
	takaran      BIGINT NOT NULL,
	harga_manual BIGINT NOT NULL,
	PRIMARY KEY (tenant_id, produk_id, no),
	FOREIGN KEY (tenant_id, produk_id) REFERENCES produk ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS supplier (
	tenant_id  INT NOT NULL REFERENCES tenants ON DELETE CASCADE,
	id         INT NOT NULL,
	nama       TEXT,
	kontak     TEXT,
	telepon    TEXT,
	email      TEXT,
	alamat     TEXT,
	tempo_hari INT NOT NULL,
	catatan    TEXT,
	PRIMARY KEY (tenant_id, id)
);

CREATE TABLE IF NOT EXISTS billing_orders (
	id        TEXT PRIMARY KEY,
	tenant_id INT NOT NULL REFERENCES tenants ON DELETE CASCADE,
	paket     TEXT NOT NULL,
	jumlah    BIGINT NOT NULL,
	status    TEXT NOT NULL,
	waktu     TIMESTAMPTZ NOT NULL
);

-- Garansi 7 hari (Okt 2026): waktu lunas per order; ekspor pertama & klaim per usaha.
ALTER TABLE billing_orders ADD COLUMN IF NOT EXISTS dibayar_pada TIMESTAMPTZ;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS ekspor_pertama TIMESTAMPTZ;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS garansi_diklaim TIMESTAMPTZ;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS garansi_alasan TEXT;

-- users & sessions lintas usaha (email unik global); disimpan terpisah dari data usaha.
CREATE TABLE IF NOT EXISTS users (
	id         INT PRIMARY KEY,
	tenant_id  INT NOT NULL,
	name       TEXT NOT NULL,
	email      TEXT NOT NULL UNIQUE,
	role       TEXT NOT NULL,
	active     BOOLEAN NOT NULL,
	created_at DATE NOT NULL,
	salt       TEXT NOT NULL, -- base64, sama dengan bentuk JSON
	hash       TEXT NOT NULL  -- PBKDF2-SHA256, base64
);

-- Kolom tambahan users (Okt 2026): nomor WA & persetujuan UU PDP saat daftar mandiri.
ALTER TABLE users ADD COLUMN IF NOT EXISTS wa TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS persetujuan_pdp TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS versi_pdp TEXT;

CREATE TABLE IF NOT EXISTS sessions (
	token   TEXT PRIMARY KEY,
	user_id INT NOT NULL,
	expires TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS bug_reports (
	id         INT PRIMARY KEY,
	tenant_id  INT NOT NULL,
	user_email TEXT,
	judul      TEXT,
	deskripsi  TEXT,
	halaman    TEXT,
	user_agent TEXT,
	waktu      TIMESTAMPTZ NOT NULL,
	screenshot TEXT -- nama berkas di $DATA_DIR/bugs/
);

-- Panel Master (Okt 2026): aktivitas user, status & balasan tiket, tim MasterPajak.
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_seen TIMESTAMPTZ;
ALTER TABLE bug_reports ADD COLUMN IF NOT EXISTS status TEXT;
ALTER TABLE bug_reports ADD COLUMN IF NOT EXISTS balasan JSONB;

CREATE TABLE IF NOT EXISTS mp_roles (
	id    TEXT PRIMARY KEY,
	nama  TEXT NOT NULL,
	kunci BOOLEAN NOT NULL,
	izin  JSONB
);

-- Anggota tim: user_id menunjuk users (role 'mp'); klien = array id usaha yang ditugaskan.
CREATE TABLE IF NOT EXISTS mp_anggota (
	user_id INT PRIMARY KEY,
	role    TEXT NOT NULL,
	klien   JSONB
);

-- Audit log Panel Master: hanya ditambah, tidak pernah diubah atau dihapus aplikasi.
CREATE TABLE IF NOT EXISTS mp_audit (
	id         INT PRIMARY KEY,
	waktu      TIMESTAMPTZ NOT NULL,
	user_id    INT NOT NULL,
	nama       TEXT,
	peran      TEXT,
	aksi       TEXT NOT NULL,
	klien_id   INT,
	keterangan TEXT
);

-- Usulan koreksi (Okt 2026): tim mengusulkan, pemilik usaha memutuskan. sebelum = potret
-- transaksi saat diusulkan; usulan = transaksi pengganti (NULL untuk hapus).
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS next_koreksi_id INT NOT NULL DEFAULT 1;
CREATE TABLE IF NOT EXISTS koreksi (
	tenant_id       INT NOT NULL REFERENCES tenants ON DELETE CASCADE,
	id              INT NOT NULL,
	transaksi_id    INT NOT NULL,
	jenis           TEXT NOT NULL,
	sebelum         JSONB NOT NULL,
	usulan          JSONB,
	alasan          TEXT NOT NULL,
	oleh_id         INT NOT NULL,
	oleh            TEXT,
	waktu           TIMESTAMPTZ NOT NULL,
	status          TEXT NOT NULL,
	catatan         TEXT,
	diputuskan_oleh TEXT,
	diputuskan_pada TIMESTAMPTZ,
	PRIMARY KEY (tenant_id, id)
);

-- Katalog Panel Master (Okt 2026): template COA per jenis usaha & benchmark DJP, berversi.
-- Versi 1 (bawaan) selalu diisi ulang dari kode saat boot. tenants.coa_template = versi template
-- yang dipatok untuk usaha itu (0 = ikut versi terbaru).
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS coa_template INT NOT NULL DEFAULT 0;
CREATE TABLE IF NOT EXISTS coa_template (
	id      INT PRIMARY KEY,
	jenis   TEXT NOT NULL,
	versi   INT NOT NULL,
	status  TEXT NOT NULL,
	akun    JSONB NOT NULL,
	catatan TEXT,
	oleh    TEXT,
	waktu   TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS benchmark_versi (
	id     INT PRIMARY KEY,
	nama   TEXT NOT NULL,
	sumber TEXT,
	status TEXT NOT NULL,
	baris  JSONB NOT NULL,
	oleh   TEXT,
	waktu  TIMESTAMPTZ
);

-- lb_rows: siapkan array JSON untuk jsonb_populate_recordset. String kosong jadi NULL
-- (kolom DATE menolak ''), dan tenant_id ditempel bila tid tidak NULL.
CREATE OR REPLACE FUNCTION lb_rows(rows jsonb, tid int) RETURNS jsonb LANGUAGE sql IMMUTABLE AS $$
	SELECT coalesce(jsonb_agg(
		coalesce((SELECT jsonb_object_agg(k, v) FROM jsonb_each(e) AS f(k, v) WHERE v <> '""'::jsonb), '{}'::jsonb)
		|| CASE WHEN tid IS NULL THEN '{}'::jsonb ELSE jsonb_build_object('tenant_id', tid) END
	), '[]'::jsonb)
	FROM jsonb_array_elements(rows) AS e
$$;
