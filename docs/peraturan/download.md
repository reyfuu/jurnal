# Cara mengunduh PDF peraturan resmi (untuk agent)

Panduan untuk agent lain yang perlu menambah peraturan ke `docs/peraturan/`.

**Status: 1.497 dari 1.512 dokumen tersimpan (21 Okt 2026).** 1.168 PDF
(6 di antaranya hanya abstrak karena JDIH tidak memuat fulltext), 329 naskah
HTML resmi (298 JDIH + 31 pajak.go.id), 15 `tanpa-pdf`. Rekap per dokumen:
`docs/peraturan/rekap_download.xlsx` (`uv run scripts/peraturan/rekap-excel.py`).
Kolom Status di tabel kategori di bawah adalah catatan historis.

Sumber cadangan setelah JDIH Kemenkeu (`unduh-djp.py`):
- **pajak.go.id** untuk PER/KEP DJP dan KMK kurs. Batang tubuh peraturan ditaruh
  DJP sebagai HTML di halaman, PDF di halaman sering hanya lampiran. Jadi berkas
  utama `.htm` dan lampirannya `<nama>-lampiran.pdf` (6 dokumen). PDF naskah
  utuh (mis. PER-16/PJ/2021 `..._Salinan.pdf`) tetap didahulukan.
- **JDIH BPK** (peraturan.bpk.go.id) untuk PP/Perpres/UU: PP 16/2021, PP 18/2025.

Sisa 15 `tanpa-pdf` tidak ada di JDIH Kemenkeu, pajak.go.id, maupun JDIH BPK:
10 SE DJP (SE tidak dipublikasikan di daftar peraturan pajak.go.id), KEP-239/PJ/2026,
PENG-6/PJ/2026, PER-11/PJ/2022, Perpres 68/2025, SEMA 2/2024.
PER-01/PJ/2025: JDIH 404, terisi dari pajak.go.id.

## Alat

```bash
# 1. Panen metadata (perlu tab primbonpajak.id yang sudah login)
#    tempel scripts/peraturan/panen-metadata.js di konsol, lalu:
#    await panen(['PPN','PPh'])      -> unduh manifest-primbon.json

# 2. Unduh + verifikasi + tulis index.csv (tanpa login, resumable)
python3 scripts/peraturan/unduh.py --dry-run
python3 scripts/peraturan/unduh.py --kategori PPN PPh --jeda 3

# 3. Sisa tanpa-pdf: cari di pajak.go.id lalu JDIH BPK
python3 scripts/peraturan/unduh-djp.py --dry-run
python3 scripts/peraturan/unduh-djp.py
```

Skrip `unduh.py` aman dijalankan berulang: dokumen yang PDF-nya sudah ada dan
lolos verifikasi dilewati, dan kategori tambahan digabung ke baris yang sama.

## Batasan wajib

- Ambil **hanya PDF resmi dan metadata faktual**. Jangan ambil tab Rangkuman,
  Catatan, atau Bookmark di primbonpajak.id. Isi itu ditandai berhak cipta
  CV Abadi Tax Indonesia dan didistribusikan per pengguna.
- **Jangan pernah mengetik kredensial.** Login dilakukan user sendiri.
- Browser dipakai bersama user. **Jangan menutup tab milik user**, hanya tutup
  tab yang agent sendiri buka.
- Jangan commit PDF tanpa persetujuan user.

## Jalur tercepat: langsung dari JDIH Kemenkeu

Ini jalur yang dipakai untuk kelima dokumen Bea Materai, dan paling tahan
banting karena tidak butuh browser maupun login.

```bash
# 1. Cari slug dokumen
curl -sSL "https://jdih.kemenkeu.go.id/search?q=Bea%20Meterai" \
  | grep -oE '/dok/[a-z0-9-]+' | sort -u

# 2. Ambil URL fulltext PDF dari halaman detail
curl -sSL "https://jdih.kemenkeu.go.id/dok/pmk-78-tahun-2024" \
  | grep -oE '/api/download/[0-9a-f-]{36}/[^"\\ ]+\.pdf' | sort -u

# 3. Unduh
curl -sSL --max-time 120 -o docs/peraturan/<kategori>/<nama>.pdf \
  "https://jdih.kemenkeu.go.id/api/download/<uuid>/<FILE>.pdf"
```

Catatan: kalau muncul dua hasil, berkas dengan sufiks `abs` adalah **abstrak**,
bukan fulltext. Pilih yang tanpa `abs`.

Metadata faktual ada di payload RSC halaman detail (`Nomor`, `Judul`, `Jenis`,
`TglPenetapan`, `TglPengundangan`, `TglMulaiBerlaku`, `Sumber`). Judul bisa
memuat koma, jadi regex harus mengambil sampai tanda kutip penutup, bukan
sampai koma pertama.

## Jalur lewat primbonpajak.id (butuh login user)

1. Daftar per kategori: `https://primbonpajak.id/regulations?kategori=<KODE>&sort=terbaru`
   (paginasi `&page=N`, 20 per halaman). Tautan detail: `main a[href*="/regulations/"]`.
2. Halaman detail punya tab `Rangkuman | PDF Resmi | Riwayat`. Panel PDF Resmi
   dirender setelah diklik, isinya tautan "Buka PDF resmi →" ke sumber luar.
3. Jalur cepat tanpa klik: `fetch('/regulations/<id>')` lalu regex URL sumber
   (`jdih.kemenkeu.go.id`, `drive.google.com`, `pajak.go.id`).

Kode `kategori` (angka = jumlah dokumen per 20 Okt 2026, dari filter UI Primbon):

| # | Kode | Label | Jumlah | Status |
|---|---|---|---|---|
| 1 | `PPN` | Pajak Pertambahan Nilai | 100 | belum |
| 2 | `PPnBM` | Pajak Penjualan atas Barang Mewah | 12 | belum |
| 3 | `PPh` | Pajak Penghasilan | 144 | belum |
| 4 | `KUP` | Ketentuan Umum Perpajakan | 231 | belum |
| 5 | `PBB` | Pajak Bumi & Bangunan | 27 | belum |
| 6 | `BEA_CUKAI` | Bea & Cukai | 179 | belum |
| 7 | `PAJAK_DAERAH` | Pajak Daerah | 67 | belum |
| 8 | `UMKM` | UMKM & PPh Final | 22 | belum |
| 9 | `FAKTUR` | Faktur Pajak | 3 | **selesai** (1 PDF, 2 tanpa PDF) |
| 10 | `NPWP` | NPWP & Pendaftaran | 5 | **selesai** (4 PDF, 1 tanpa PDF) |
| 11 | `PPS` | Pengungkapan Sukarela | 1 | **selesai** |
| 12 | `CARBON_TAX` | Pajak Karbon | 4 | **selesai** |
| 13 | `TRANSFER_PRICING` | Transfer Pricing | 2 | **selesai** (1 PDF, 1 tanpa PDF) |
| 14 | `TINDAK_PIDANA` | Tindak Pidana Perpajakan | 10 | belum |
| 15 | `SENGKETA_PAJAK` | Sengketa Pajak | 6 | **selesai** (5 PDF, 1 tanpa PDF) |
| 16 | `KURS` | Kurs KMK | 52 | belum |
| 17 | `KEUNEG` | Keuangan Negara | 671 | belum |
| 18 | `PENG_SP_DJP` | Pengumuman & Siaran Pers DJP | 1 | **selesai** (tanpa PDF) |
| 19 | `BEA_MATERAI` | Bea Materai | 5 | **selesai** |

**Hati-hati soal angka.** Kolom Jumlah di atas kalau dijumlah = 1.542, tetapi
dokumen **uniknya hanya 1.512** (76 halaman x 20, halaman terakhir 12 item).
Selisih 30 muncul karena **satu dokumen bisa masuk beberapa kategori**.
Contoh terbukti: PER-11/PJ/2025 tampil di `BEA_MATERAI`, `PPh`, `PPN`, dan
`PPnBM`. Jadi:

- Jangan pakai `kategori` sebagai kunci unik. Kunci unik = UUID dokumen pada URL
  `/regulations/<uuid>`.
- Sebelum mengunduh, cek UUID sudah ada di `index.csv` atau belum. Kalau sudah,
  cukup tambahkan kategorinya ke kolom `kategori` (pisah dengan `;`), jangan
  unduh ulang PDF-nya.
- Target penyelesaian = 1.512 baris unik, bukan 1.542.

Cara mengambil ulang angka ini kalau sudah berubah (perlu tab yang sudah login,
jalankan di konsol tab primbonpajak.id):

```js
const t = await (await fetch('/regulations?sort=terbaru')).text();
const re = /name="kategori"\s+value="([^"]+)"\/><span[^>]*>([^<]+)</g;
let m, out = []; while ((m = re.exec(t))) out.push([m[1], m[2].trim()]);
console.log(out, t.match(/dari <!-- -->(\d+)/));   // angka = jumlah halaman
```

## Jebakan yang sudah terbukti

- **JDIH punya dua bentuk tautan unduh.** Dokumen baru memakai
  `/api/download/<uuid>/<FILE>.pdf`, dokumen lama (`IsNew:false`) memakai
  `/api/download/fulltext/<tahun>/<FILE>.pdf`. Regex yang mewajibkan UUID akan
  melewatkan semua dokumen lama: 6 dari 22 dokumen percontohan awalnya gagal
  karena ini.
- **Sebagian dokumen hanya punya Fulltext HTML, tanpa PDF.** Contoh: UU 14/2002
  dan UU 22/2004. Cadangannya `https://peraturan.go.id/files/<jenis><no>-<tahun>.pdf`
  (misalnya `uu14-2002.pdf`), sudah dipakai `unduh.py` secara otomatis.
- **Berkas bersufiks `abs.pdf` adalah abstrak, bukan fulltext.** PMK 78/2024
  punya keduanya. Selalu pilih yang tanpa `abs`.
- **Slug JDIH dari Primbon bisa terpotong.** Dugaan (belum dibuktikan, perlu
  sesi login): payload RSC Next.js memotong string di batas chunk, jadi regex
  pertama sempat menangkap `pmk-148-t`, `per`, `p`, `pp-1`. Enam dokumen jadi
  salah `tanpa-pdf`. `panen-metadata.js` sekarang mengambil kecocokan terpanjang.
  Cek cepat yang pasti: slug sah selalu memuat tahun 4 digit.
- **JDIH sesekali membalas 504.** Bukan tanda dokumen tidak ada. `unduh.py`
  mencoba ulang 3 kali dengan jeda menanjak.
- **Kode kategori peka huruf besar-kecil.** `PPh` dan `PPnBM` memakai campuran
  huruf. Menulis `PPH` atau `PPNBM` dibalas HTTP 200 dengan **0 hasil**, jadi
  terlihat seperti kategori kosong, bukan error. Selalu cocokkan dengan tabel
  di atas.
- **Halaman daftar butuh login.** `curl` tanpa sesi dibalas `307` ke
  `/login?next=...`. Pakai tab browser user yang sudah login, atau lewati
  Primbon dan cari langsung di JDIH.
- **Google Drive sering ditolak.** Dua dokumen (PMK 78/2024 dan PER-11/PJ/2025)
  di Primbon menunjuk ke Drive yang mengembalikan
  "the owner hasn't given you permission to download this file".
  Solusinya: cari dokumen yang sama di JDIH Kemenkeu. Keduanya ada di sana.
  Selalu cek `file <hasil>.pdf`; kalau hasilnya `HTML document`, unduhan gagal.
- **Klik lewat chrome-devtools MCP bisa lapor error padahal berhasil.**
  Klik tautan yang membuka tab baru kadang melaporkan
  "element did not become interactive" sambil tetap membuka tab. Cek
  `list_pages` sebelum mencoba ulang, supaya tidak ada tab duplikat.
- **Tab yang di belakang bikin klik gagal.** Pakai `select_page` dengan
  `bringToFront: true` lebih dulu.

## Aturan penamaan dan index

- Letak: `docs/peraturan/<kategori-kebab>/<kode>-<nomor>-<tahun>.pdf`,
  misalnya `docs/peraturan/bea-materai/pmk-78-2024.pdf`.
  Dokumen multi-kategori disimpan **sekali saja**, di kategori pertama yang
  memuatnya.
- Lampiran tambahan: sufiks `-lampiran-N`.
- `index.csv` satu berkas untuk semua kategori, **berkunci `uuid`**, kolom:
  `uuid,kategori,kode,judul,jenis,penerbit,tahun,tanggal_penetapan,tanggal_pengundangan,berlaku,sumber_ln,status,halaman,ukuran_byte,sha256,primbon_url,jdih_url,sumber_url,file`
  Kolom `kategori` boleh multi-nilai, dipisah `;`.
- Dokumen tanpa tautan PDF dicatat dengan `file` bernilai `tanpa-pdf`.
- **Resume**: lewati entri yang `file`-nya sudah ada di `index.csv` dan berkasnya
  ada di disk. Beri jeda beberapa detik antar dokumen.

## Verifikasi per berkas (wajib)

```bash
file x.pdf                                  # harus "PDF document", bukan HTML
pdfinfo x.pdf | grep -i pages               # harus sukses dan > 0
pdftotext -l 3 x.pdf - | grep -i '<nomor>'  # nomor peraturan ada di awal dokumen
```

Dokumen hasil pindai/OCR kadang mengubah angka, misalnya UU 10/2020 terbaca
`NOMOR 10 TAHUN 2O2O` (huruf O). Itu normal dan bukan tanda berkas salah;
cocokkan dengan `pdfinfo` dan judul JDIH-nya.

## Skala

Seluruh katalog = **1.512 dokumen unik**. Dari 21 PDF yang sudah terunduh,
rata-ratanya ~4,8 MB per dokumen (ada yang 383 halaman), jadi perkiraan total
**sekitar 7 GB**. Dengan jeda 3 detik, waktu berurutan sekitar 2 sampai 3 jam.

Karena itu `docs/peraturan/*/` sudah masuk `.gitignore`: hanya `index.csv` dan
`download.md` yang masuk git, PDF-nya reproducible dari kolom `sumber_url`.

Sebelum unduhan massal, pastikan ruang disk cukup (`df -h /home`) dan konfirmasi
dulu ke user, karena ini pekerjaan berjam-jam dan membebani server pemerintah.
