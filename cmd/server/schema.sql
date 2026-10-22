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
ALTER TABLE business_profiles ADD COLUMN IF NOT EXISTS modal_awal BIGINT;
ALTER TABLE business_profiles ADD COLUMN IF NOT EXISTS akun_tambahan JSONB;
-- sumber dana aset tanpa transaksi (kas | utang | modal); kosong = belum dipilih
ALTER TABLE assets ADD COLUMN IF NOT EXISTS sumber_dana TEXT;
-- akumulasi penyusutan yang sudah ada per tanggal saldo awal (template saldo awal / pembukuan lama)
ALTER TABLE assets ADD COLUMN IF NOT EXISTS akumulasi_awal BIGINT;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS akumulasi_per DATE;
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

-- Panel Master (Okt 2026): aktivitas user, status & balasan tiket, tim Master Pajak.
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

-- Katalog Panel Master (Okt 2026): template COA per jenis usaha & benchmark nasional, berversi.
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

-- Parameter pajak (Okt 2026): tarif, PTKP, batas omzet, TER, objek potong, norma NPPN — berversi.
-- isi = tax.Parameter utuh (JSONB). Versi 1 (bawaan) diisi ulang dari kode saat boot.
CREATE TABLE IF NOT EXISTS parameter_pajak (
	id     INT PRIMARY KEY,
	nama   TEXT NOT NULL,
	dasar  TEXT,
	status TEXT NOT NULL,
	isi    JSONB NOT NULL,
	oleh   TEXT,
	waktu  TIMESTAMPTZ
);

-- Pengetahuan (Okt 2026): artikel pajak & sumber yang dibaca AI. Hanya artikel terbit dan sumber
-- terverifikasi (dipakai_ai) yang dikirim ke model.
CREATE TABLE IF NOT EXISTS artikel (
	id            INT PRIMARY KEY,
	judul         TEXT NOT NULL,
	kategori      TEXT NOT NULL,
	dasar_hukum   TEXT,
	berlaku_sejak TEXT,
	tag           JSONB,
	isi           TEXT NOT NULL,
	status        TEXT NOT NULL,
	tampil_klien  BOOLEAN NOT NULL,
	oleh          TEXT,
	diverifikasi  TEXT,
	waktu         TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS sumber_pengetahuan (
	id           INT PRIMARY KEY,
	judul        TEXT NOT NULL,
	jenis        TEXT NOT NULL,
	topik        JSONB,
	isi          TEXT NOT NULL,
	status       TEXT NOT NULL,
	dipakai_ai   BOOLEAN NOT NULL,
	oleh         TEXT,
	diverifikasi TEXT,
	waktu        TIMESTAMPTZ
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

-- Metode penyusutan usaha (Okt 2026): dipilih sekali untuk semua aset, lalu terkunci.
ALTER TABLE business_profiles ADD COLUMN IF NOT EXISTS metode_penyusutan TEXT;
ALTER TABLE business_profiles ADD COLUMN IF NOT EXISTS metode_nilai_stok TEXT;

-- Kelola data Panel Master (Okt 2026): usaha diarsipkan (soft delete) — user-nya tidak bisa masuk.
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS arsip TIMESTAMPTZ;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS arsip_alasan TEXT;

-- Kategori isi sumber pengetahuan (Okt 2026): peraturan | artikel | praktik | umum | lainnya.
ALTER TABLE sumber_pengetahuan ADD COLUMN IF NOT EXISTS kategori TEXT;

-- Catatan konsultan (Okt 2026): catatan tim pada laporan klien; klien menandai selesai.
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS next_catatan_id INT NOT NULL DEFAULT 1;
CREATE TABLE IF NOT EXISTS catatan_konsultan (
	tenant_id        INT NOT NULL REFERENCES tenants ON DELETE CASCADE,
	id               INT NOT NULL,
	halaman          TEXT,
	periode          TEXT,
	tingkat          TEXT NOT NULL,
	isi              TEXT NOT NULL,
	transaksi_id     INT,
	oleh_id          INT NOT NULL,
	oleh             TEXT,
	peran            TEXT,
	dibuat           TIMESTAMPTZ NOT NULL,
	status           TEXT NOT NULL,
	selesai_oleh     TEXT,
	selesai_pada     TIMESTAMPTZ,
	diingatkan_pada  TIMESTAMPTZ,
	jumlah_pengingat INT NOT NULL DEFAULT 0,
	PRIMARY KEY (tenant_id, id)
);
-- Pengaturan Panel Master yang berlaku untuk semua klien (kunci → JSON), mis. pengingat.
CREATE TABLE IF NOT EXISTS mp_pengaturan (
	kunci TEXT PRIMARY KEY,
	isi   JSONB NOT NULL
);

-- Kartu stok (Okt 2026): transaksi Beli Bahan / Stok & Penjualan bisa memilih barang (jumlah
-- per seribu satuan); dokumen stok = stok awal (titik nol), stok opname, produksi.
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS stok_id INT;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS qty_stok BIGINT;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS next_gerak_id INT NOT NULL DEFAULT 1;
CREATE TABLE IF NOT EXISTS gerak_stok (
	tenant_id  INT NOT NULL REFERENCES tenants ON DELETE CASCADE,
	id         INT NOT NULL,
	tanggal    DATE NOT NULL,
	jenis      TEXT NOT NULL,
	baris      JSONB NOT NULL,
	produk_id  INT,
	batch      BIGINT,
	keterangan TEXT,
	oleh       TEXT,
	dibuat     TIMESTAMPTZ NOT NULL,
	PRIMARY KEY (tenant_id, id)
);

-- Tutup buku bulanan & pelunasan saldo awal (Okt 2026).
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS tutup_buku TEXT;
ALTER TABLE opening_balances ADD COLUMN IF NOT EXISTS pelunasan JSONB;
-- saldo awal per akun (template CoA): pos tambahan neraca awal + rinciannya
ALTER TABLE opening_balances ADD COLUMN IF NOT EXISTS laba_ditahan BIGINT;
ALTER TABLE opening_balances ADD COLUMN IF NOT EXISTS aset_lain BIGINT;
ALTER TABLE opening_balances ADD COLUMN IF NOT EXISTS kewajiban_lain BIGINT;
ALTER TABLE opening_balances ADD COLUMN IF NOT EXISTS aset_tetap BIGINT;
ALTER TABLE opening_balances ADD COLUMN IF NOT EXISTS rincian JSONB;

-- Invoice ↔ pembukuan (Okt 2026): transaksi Penjualan & PPN Keluaran dibuat otomatis dari invoice.
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS invoice_id INT;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS jatuh_tempo DATE;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS grup INT;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS no_bukti TEXT;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS lawan TEXT;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS tanggal_lunas DATE;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS dibukukan BOOLEAN;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS transaksi_id INT;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS transaksi_ppn_id INT;

-- Primbon Pajak (Okt 2026): bookmark & catatan pribadi per pengguna atas peraturan katalog.
-- Tanpa FK ke users: saveAuthDB menulis ulang tabel users setiap simpan. Baris milik user
-- yang sudah dihapus dibuang saat boot (loadPrimbon).
CREATE TABLE IF NOT EXISTS primbon_bookmark (
	user_id INT NOT NULL,
	uuid    TEXT NOT NULL,
	waktu   TIMESTAMPTZ NOT NULL,
	PRIMARY KEY (user_id, uuid)
);
CREATE TABLE IF NOT EXISTS primbon_catatan (
	user_id INT NOT NULL,
	uuid    TEXT NOT NULL,
	isi     TEXT NOT NULL,
	diubah  TIMESTAMPTZ NOT NULL,
	PRIMARY KEY (user_id, uuid)
);

-- Catat produksi (Okt 2026): biaya lain produksi yang ikut nilai produk jadi, dan grup
-- transaksi pembayarannya (Transaction.Grup).
ALTER TABLE gerak_stok ADD COLUMN IF NOT EXISTS biaya BIGINT;
ALTER TABLE gerak_stok ADD COLUMN IF NOT EXISTS grup INT;

-- Cegah transaksi dobel (Okt 2026): asal pencatatan (mutasi / foto).
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS sumber TEXT;

-- Nomor jurnal (Okt 2026, checklist No 31): JU-YYYYMM-NNNN per transaksi / catatan, tidak
-- dipakai ulang; penghitung terakhir per bulan di tenants.nomor_jurnal ({"202610": 12}).
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS no_jurnal TEXT;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS nomor_jurnal JSONB;
