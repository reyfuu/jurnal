# MasterPajak AI — Laporan Bisnis

Aplikasi pembukuan dan pajak untuk UMKM Indonesia. Pengguna mencatat transaksi dengan bahasa
sehari-hari; jurnal, laporan keuangan, dan hitungan pajak UMKM tersusun otomatis.

Backend Go (satu binary yang juga menyajikan frontend), frontend React 19 + Vite + shadcn/ui,
penyimpanan PostgreSQL (atau berkas JSON / memori untuk pengembangan).

## Fitur utama

- **Transaksi**: catat penjualan, biaya, pembelian, produksi, dan pelunasan; isi dari foto nota
  (dibaca AI, fotonya disimpan sebagai bukti); impor mutasi bank dengan pencegah jurnal dobel.
- **Operasional**: Stok & Gudang (kartu stok, stok opname), Produksi & Supplier (resep, HPP),
  Piutang & Utang (dengan umur piutang), Invoice + PDF.
- **Akuntansi**: Laporan Keuangan (laba rugi, neraca, arus kas), Buku Besar, Jurnal Umum,
  Daftar Akun (CoA), Saldo Awal, Tutup Buku Bulanan, Laporan Transaksi & Penjualan.
- **Pajak**: Pajak UMKM (PP 55/2022), Kalkulator Pajak, Perencanaan Pajak, Aturan Pajak
  (katalog peraturan resmi dengan bookmark & catatan), Artikel Pajak.
- **AI**: Laporan AI, Tanya AI (menjawab, menunjukkan menu, dan mengisi form dengan pratinjau).
- **Akun & langganan**: pendaftaran mandiri, peran Owner / Admin Keuangan / Kasir, Alat Gratis
  untuk pengunjung, langganan lewat Midtrans dengan garansi 7 hari uang kembali.
- **Panel Master**: tim konsultan melihat klien, menulis catatan, dan mengusulkan koreksi.

## Menjalankan secara lokal

Butuh Go ≥ 1.24 (Docker memakai 1.27) dan Node 22.

```bash
# frontend → web/dist
cd frontend && npm ci && npm run build && cd ..

# backend (menyajikan web/dist dan /api)
go run ./cmd/server
# buka http://localhost:8080
```

Tanpa `DATABASE_URL` dan `DATA_DIR`, data hanya di memori dan berisi data contoh. Akun demo
lokal: `owner@kedaikopisenja.id`, `admin@kedaikopisenja.id`, `kasir@kedaikopisenja.id`,
password `senja2026` (ganti lewat `SEED_PASSWORD`).

Pengembangan frontend dengan hot reload: jalankan server Go di :8080, lalu
`cd frontend && npm run dev` (Vite mem-proxy `/api` ke :8080).

## Tes

```bash
go vet ./... && go test ./...
cd frontend && npx vitest run && npm run lint && npm run build
```

Tes penyimpanan PostgreSQL hanya berjalan bila `TEST_DATABASE_URL` diisi.

## Konfigurasi (environment)

| Variabel | Fungsi |
|---|---|
| `PORT` | Port HTTP (bawaan 8080) |
| `DATABASE_URL` | PostgreSQL; bila kosong memakai `DATA_DIR` |
| `DATA_DIR` | Folder data (state JSON, foto bukti nota); kosong = memori saja |
| `SEED_PASSWORD` | Password akun contoh saat data pertama kali dibuat |
| `MASTER_EMAIL`, `MASTER_PASSWORD` | Membuat akun Master pertama bila belum ada |
| `MIDTRANS_SERVER_KEY`, `MIDTRANS_CLIENT_KEY`, `MIDTRANS_PRODUCTION` | Pembayaran langganan (`MIDTRANS_PRODUCTION=true` untuk produksi) |
| `LLM_API_KEY`, `LLM_BASE_URL`, `LLM_MODEL`, `LLM_TIMEOUT` | Model AI (API kompatibel OpenAI). `LLM_MODEL` boleh berisi beberapa model dipisah koma, dicoba berurutan. Tanpa kunci, fitur AI memakai jawaban cadangan |
| `LLM_VISION_API_KEY`, `LLM_VISION_BASE_URL`, `LLM_VISION_MODEL` | Model khusus pembaca foto nota; yang kosong ikut `LLM_*` |
| `REGULASI_DIR` | Folder katalog peraturan (`index.csv` + naskah); bawaan `docs/peraturan` |
| `BUG_WEBHOOK_URL`, `BUG_WEBHOOK_TIMEOUT` | Meneruskan laporan bug ke webhook |

Jangan commit nilai rahasia; di server simpan di `.env` yang tidak masuk git.

## Struktur

```
cmd/server/      server HTTP, handler API, penyimpanan
internal/        ai, auth, coa, finance, katalog, llm, master, midtrans, pdf,
                 pengetahuan, primbon, tax, xlsx
frontend/        React + Vite (hasil build ke web/dist)
docs/            BRD, FRD, PRD, TRD, desain, panduan
scripts/         generator data (KLU, norma, wilayah)
deploy.sh        build image Docker dan jalankan container + PostgreSQL
```

## Dokumen lain

- [HANDOFF.md](HANDOFF.md): riwayat fase, keputusan, dan status terkini.
- [DESIGN.md](DESIGN.md): sistem desain dan brand.
- [docs/RAG.md](docs/RAG.md): impor regulasi bertahap, indeks pencarian PostgreSQL, API RAG, dan kompresi PDF di worker terpisah.
- [docs/PRIMBON-METADATA.md](docs/PRIMBON-METADATA.md): tag dan status peraturan melalui metadata kecil; naskah tetap di VPS.
- [agent.md](agent.md) dan [AGENTS.md](AGENTS.md): aturan kerja untuk agen AI di repo ini.
- [docs/](docs/): dokumen produk (BRD, FRD, PRD, TRD) dan `docs/accounting-platform/`.
