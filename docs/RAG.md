# Infrastruktur RAG peraturan

Status: backend dan worker lokal; belum dideploy ke VPS. Koleksi 1.512 entri memakai `docs/peraturan/index.csv`; naskah yang belum ada dapat dilengkapi nanti. RAG tidak membaca seluruh koleksi saat pengguna bertanya.

## Komponen

- PostgreSQL: `rag_document` menyimpan metadata, fingerprint, dan status; `rag_chunk` menyimpan teks per halaman dan indeks GIN (`simple`, agar nomor peraturan dan istilah Indonesia tetap tersedia).
- Worker `cmd/rag`: satu dokumen pada satu waktu, transaksi per dokumen, tidak mengunduh URL. Perubahan isi atau metadata memicu indeks ulang. Impor ulang melewati fingerprint yang sudah siap; potongan lama diganti atomik. Menambahkan berkas yang tadinya hilang cukup diikuti impor ulang.
- PDF/HTML diekstrak sekali saat impor. Batas per berkas 128 MiB, teks 16 MiB, 10.000 halaman; pelanggaran dicatat sebagai error, bukan diam-diam dipotong. PDF tanpa teks diberi `needs_ocr`; OCR belum diotomatisasi.
- Sumber resmi bersama terpisah dari data transaksi/tenant dan dari sumber internal tim. Hanya dokumen `ready` yang dicari. `missing`, `error`, `needs_ocr` terlihat di laporan worker dan statistik.
- API memakai autentikasi, izin `primbon-pajak`, dan langganan yang sama dengan katalog. Tidak ada endpoint publik untuk mengimpor atau mengompresi file.

Pencarian saat ini berbasis kata kunci berbobot, dengan pembersihan kata tanya Indonesia. Ini belum pencarian semantik/hybrid: model embedding, dimensinya, biaya, dan perubahan image database untuk pgvector perlu dipilih kemudian. Tidak ada API embedding berbayar yang dipanggil saat impor.

## Menyiapkan worker di VPS

Jalankan setelah kode ini ditinjau. Tidak perlu mengganti image PostgreSQL atau mengubah volume database yang sudah ada. Cadangkan database sebelum menerapkan versi server baru; tabel RAG dibuat dengan `IF NOT EXISTS` saat boot maupun sebelum impor.

```bash
docker build -f Dockerfile.rag -t laporan-bisnis-rag:local .

# Pratinjau: tanpa DATABASE_URL, tanpa perubahan file/database.
docker run --rm --memory=1g --cpus=1 \
  -v /var/www/peraturan:/peraturan:ro \
  laporan-bisnis-rag:local --dir /peraturan --limit 20
```

Untuk menerapkan, sediakan `DATABASE_URL` melalui lingkungan shell yang aman. Gunakan host `laporan-bisnis-db` dan network `laporan-bisnis-net` bila mengikuti `deploy.sh`. Jangan menuliskan password ke laporan/log atau commit. Worker tidak membaca `.env` secara otomatis.

```bash
mkdir -p /var/www/peraturan-compressed

docker run --rm --memory=1g --cpus=1 \
  --network laporan-bisnis-net -e DATABASE_URL \
  -v /var/www/peraturan:/peraturan:ro \
  -v /var/www/peraturan-compressed:/compressed \
  laporan-bisnis-rag:local --dir /peraturan --apply \
  --compressed-dir /compressed --limit 20 > rag-sample.jsonl
```

Periksa hasil sampel, penggunaan RAM, dan mutu teks sebelum menghapus `--limit 20` untuk seluruh katalog. `--limit` membatasi entri pertama katalog; hilangkan untuk memeriksa semuanya. Laporan berupa JSON per dokumen dan ringkasan akhir. File hilang tidak menghentikan impor; kegagalan DB menghentikannya. Worker keluar nonzero jika ada error, kebutuhan OCR, atau kompresi gagal. Jalankan ulang untuk melanjutkan, tanpa menghapus indeks.

Untuk impor berikutnya tanpa kompresi, hilangkan `--compressed-dir`: checksum tetap diperiksa dan hanya dokumen berubah yang diekstrak ulang. Tidak ada penjadwalan otomatis atau penghapusan dokumen yang hilang dari CSV. Jika dokumen harus ditarik dari indeks, operator dapat menandainya dengan `state='error'` dan alasan, lalu menghapusnya dari katalog impor; jangan menafsirkan status hukum `Dicabut` sebagai instruksi penghapusan dokumen historis.

## Kompresi

Worker memakai library `pdfcpu` yang sudah dipakai proyek, bukan layanan online. Hasil ditulis ke folder terpisah dengan nama SHA256 sumber. Folder turunan wajib berada di luar folder sumber. PDF asli tidak diubah, termasuk bila memiliki tanda tangan digital. Salinan turunan bukan pengganti naskah bertanda tangan.

Hasil hanya disimpan bila lebih kecil, lolos validasi PDF, jumlah halaman sama, dan teks per halaman tetap sama setelah normalisasi spasi. Optimasi struktur tidak menjamin pengurangan besar untuk PDF scan yang sudah terkompresi. OCR, pengecilan resolusi gambar, dan penggantian file katalog tidak dilakukan. Salinan terkompresi belum otomatis dipakai viewer Primbon; viewer tetap memakai naskah asli.

Kompresi adalah pekerjaan offline di worker, bukan dalam request HTTP. Batas container di atas melindungi aplikasi utama dari pemakaian RAM/CPU berlebihan. Parser native mengecek pembatalan antarhalaman; `context` tidak dapat memotong parser yang sedang mengerjakan satu halaman, sehingga batas proses/container tetap diperlukan untuk PDF rusak.

## API

- `GET /api/v1/primbon/rag/status`: jumlah ready/missing/needs_ocr/error dan potongan siap.
- `GET /api/v1/primbon/rag/cari?q=UMKM`: maksimal enam potongan, metadata dan nomor halaman. Halaman HTML bernilai 1 (halaman logis, bukan nomor halaman PDF).
- `POST /api/v1/primbon/rag/tanya` dengan JSON `{"query":"Bagaimana pajak untuk UMKM?"}`: sumber plus jawaban dari konfigurasi `LLM_*` yang sudah ada. Batas pertanyaan 2.000 karakter, body 16 KiB, kuota AI 40/user/hari.

Tanpa LLM, pencarian dan potongan sumber tetap tersedia. Tanpa bukti relevan, model tidak dipanggil. Rujukan jawaban dibatasi ke ref potongan yang benar-benar dikirim; metadata status/berlaku ditampilkan sebagai informasi sumber, bukan jaminan aturan masih berlaku pada tanggal pertanyaan. Penilaian kebenaran hukum tetap memerlukan pemeriksaan naskah dan perubahan aturan. Endpoint ini belum dihubungkan ke bubble AI frontend.

## Optimasi penyimpanan pengetahuan lama

Perubahan artikel/sumber tim memakai perbandingan state lama dan baru di bawah lock yang sudah ada. Hanya baris berubah yang di-upsert dan ID yang dihapus yang dihapus, dalam satu transaksi. Baris besar yang tidak berubah tidak ditulis ulang. Ini tidak memindahkan seluruh koleksi regulasi ke state memori aplikasi dan tidak mengubah logika keuangan.

## Pengujian

```bash
go test ./...
# Khusus DB disposable! Tes berikut menghapus tabel/skema pengujian.
RAG_TEST_DATABASE_URL=postgres://... go test ./internal/rag -count=1
TEST_DATABASE_URL=postgres://... go test ./cmd/server -run 'TestPostgresPengetahuan|TestPostgresRAG' -count=1
go build ./cmd/server ./cmd/rag
```

Jangan arahkan variabel database tes ke database produksi. Mutu retrieval seluruh koleksi, rasio kompresi nyata, dan latensi produksi belum dapat disimpulkan tanpa naskah VPS dan benchmark sampel.
