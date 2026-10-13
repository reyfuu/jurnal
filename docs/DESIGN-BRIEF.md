# Design Brief — Jurnal AI UI/UX

Status: draft 13 Okt 2026 · Pemilik keputusan: Freddie · Pelaksana: agent/dev berikutnya

**Design Read:** Membangun *workspace pembukuan* untuk bookkeeper & pemilik UMKM
Indonesia dengan nada **Monastic Utility / Swiss Ledger**: tenang, padat data,
dapat dipercaya. Ciri khasnya adalah **"Ledger Seam"**, yaitu garis debit|kredit yang selalu
terlihat dan berubah warna seketika saat jurnal tidak balance.

---

## 1. Masalah UI sekarang

Hasil audit cepat terhadap 19 view Blade (1.613 baris) dan `resources/css/app.css`:

| # | Masalah | Dampak |
|---|---|---|
| 1 | Brand indigo `#4f46e5` + logo "JA" generik | Tampilan template AI, tidak punya identitas |
| 2 | Font system stack | Angka & teks tidak punya karakter; tabular hanya di `font-mono` |
| 3 | Semua blok dibungkus `card` (border+shadow) | Kartu bertumpuk, hierarki datar, layar review padat terasa sesak |
| 4 | Hijau/merah dipakai untuk pendapatan/beban | Salah makna: beban bukan "error". Merah seharusnya khusus untuk *tidak balance / gagal* |
| 5 | Nav atas hanya Dashboard/Klien, sub-nav klien terpisah | Konteks klien aktif mudah hilang, padahal **isolasi klien = aturan inti** |
| 6 | Layar review (`documents/show`, 307 baris) tidak membedakan angka *AI* vs *sudah dikoreksi manusia* | Melemahkan aturan "AI tidak pernah memposting sendiri" |
| 7 | Tidak ada focus ring konsisten, tidak ada `prefers-reduced-motion`, tidak ada dark mode | Aksesibilitas & kenyamanan kerja lama |
| 8 | Warning AI berupa kotak kuning panjang | Mudah di-skip; seharusnya menempel di baris yang bermasalah |

## 2. Prinsip (urut prioritas)

1. **Kejujuran AI terlihat.** Tiap angka hasil AI punya status visual: `AI · ragu xx%`, `AI · yakin`, `dikoreksi`. Manusia selalu tahu angka mana yang belum dilihatnya.
2. **Balance di atas segalanya.** Status debit=kredit selalu terlihat (sticky), tidak perlu scroll.
3. **Klien aktif tidak boleh ambigu.** Nama klien + warna/inisialnya ada di header di setiap halaman dalam konteks klien.
4. **Padat tapi bernapas.** Tabel adalah bentuk utama; grouping dilakukan dengan whitespace & garis tipis, bukan kartu.
5. **Merah = salah, bukan beban.** Warna semantik hanya untuk status, bukan untuk jenis akun.

## 3. Sumber referensi & cara memakainya

Stack tetap **Laravel Blade + Tailwind v4 + vanilla JS** (`resources/js/app.js`).
**Jangan menambah React/Vue.** Library di bawah dipakai sebagai *referensi pola &
token*, lalu di-port ke `@utility` di `app.css` atau partial Blade.

| Sumber | Yang diambil | Yang TIDAK diambil |
|---|---|---|
| **Impeccable** (skill) | Design Read, larangan slop (indigo-gradient, kartu seragam, font default), self-critique pass, motion token | Hero/landing pattern, typography display raksasa: ini alat kerja |
| **shadcn/ui** | Sistem token semantik (`--background`, `--foreground`, `--muted`, `--border`, `--ring`, `--destructive`), anatomi: Button, Input, Select, Table, Badge, Dialog, Tabs, Command (⌘K), Sonner toast, Sidebar | Komponen React & Radix; tiru markup/state-nya dalam Blade |
| **Kokonut UI** | Detail micro-interaction: tombol aksi dengan state loading, input dengan prefix "Rp", file dropzone, kartu statistik ringkas | Animasi dekoratif, efek glow/gradient |
| **21st.dev** | Referensi bentuk: data table dengan sticky header, empty state, upload dropzone, stat tile + sparkline, command palette | Komponen yang butuh framer-motion / dependency berat |
| **UI/UX Pro Max** (belum terpasang; prinsip diterapkan manual) | Rule produk *Fintech / Accounting SaaS*: palet dengan kepercayaan tinggi, kontras AA, angka tabular, target sentuh ≥ 44px, form error inline, loading state < 100 ms feedback | Gaya "glassmorphism" / "neumorphism" |

## 4. Fondasi visual

### 4.1 Token warna (gaya shadcn, didefinisikan di `@theme` + `:root`)

Netral dingin-hangat (stone) + satu aksen tinta hijau-ledger. Indigo dibuang.

| Token | Light | Dark | Pakai untuk |
|---|---|---|---|
| `--background` | `#fafaf9` | `#0c0c0b` | kanvas |
| `--surface` | `#ffffff` | `#161614` | panel, tabel |
| `--muted` | `#f2f1ef` | `#1f1f1c` | header tabel, area sekunder |
| `--border` | `#e4e2de` | `#2b2a27` | garis 1px |
| `--foreground` | `#1c1b19` | `#f2f1ee` | teks utama |
| `--muted-foreground` | `#6b6862` | `#a19e97` | teks sekunder (cek AA ≥ 4.5:1) |
| `--primary` | `#1c1b19` | `#f2f1ee` | tombol utama (gaya shadcn: hitam pekat) |
| `--accent` | `#0f6b55` (ledger green) | `#3fbf9a` | link, fokus, klien aktif, logo |
| `--ai` | `#b45309` (amber) | `#f5a524` | penanda angka dari AI / ragu |
| `--ok` | `#15803d` | `#4ade80` | balance, posted |
| `--destructive` | `#b91c1c` | `#f87171` | tidak balance, gagal, void |
| `--ring` | `--accent` @ 40% | idem | focus ring |

Aturan: pendapatan/beban **tidak** diberi hijau/merah. Laba bersih negatif boleh
memakai `--destructive` hanya pada angka total, dengan tanda minus/kurung.

### 4.2 Tipografi

- **Sans:** Geist (variable). **Angka & kode akun:** Geist Mono, `tabular-nums`.
- Self-host lewat `@fontsource-variable/geist` + `@fontsource-variable/geist-mono`, karena aplikasi harus bisa jalan **tanpa internet**. Jangan pakai CDN font.
- Skala (rem): 12 · 13 · 14 (body tabel) · 16 (body form) · 20 (judul halaman) · 28 (angka stat). Tidak ada `clamp()` display, karena ini alat kerja.
- Label: 11–12px, uppercase, `tracking-[0.08em]`, `--muted-foreground`.
- Angka rupiah: rata kanan, mono, pemisah titik, negatif dalam kurung `(50.000)`, konsisten dengan `Rupiah::format()`.

### 4.3 Ruang, radius, elevasi

- Grid 4px. Padding sel tabel `px-3 py-2`; baris padat opsional `py-1.5`.
- Radius: 6px (input/tombol), 10px (panel). Pill hanya untuk badge.
- Elevasi: default **border 1px tanpa shadow**. Shadow hanya untuk lapisan melayang (dropdown, dialog, toast): `0 1px 2px rgb(0 0 0/.05), 0 8px 24px rgb(0 0 0/.08)`.

### 4.4 Motion

- `--ease-snappy: cubic-bezier(.2,0,0,1)`; 120–180 ms untuk hover/focus/badge.
- Ledger Seam: transisi warna 150 ms, tanpa animasi lain.
- Wajib blok `@media (prefers-reduced-motion: reduce)`.

## 5. Shell & navigasi

- **Sidebar kiri** (gaya shadcn Sidebar), bisa diciutkan, disimpan di `localStorage`:
  - Atas: logo (wordmark "Jurnal" + tanda ledger, bukan "JA").
  - **Client switcher** (dropdown dengan inisial berwarna + nama + badge `sendiri/klien`). Berganti klien = berganti seluruh konteks.
  - Menu klien: Dokumen (badge jumlah perlu review) · Jurnal · Akun (COA) · Laporan ▸ Neraca Saldo / Laba Rugi / Neraca.
  - Bawah: Klien (admin), user + role, Keluar.
- **Topbar tipis:** breadcrumb `Klien › Dokumen › Review`, period picker, tombol `⌘K`.
- **Command palette (⌘K):** loncat ke klien, "Upload dokumen", "Jurnal manual", laporan. Vanilla JS + `<dialog>`.
- Mobile: sidebar jadi drawer (`<dialog>`), tabel scroll horizontal dengan kolom pertama sticky.

## 6. Layar prioritas (urut pengerjaan)

### P1. Review dokumen (`documents/show`), inti produk

Layout dua kolom ≥ 1024px:
- **Kiri (40%) — Bukti asli:** preview gambar/PDF dengan zoom & rotate, sticky. Teks tempel tampil sebagai blok mono.
- **Kanan (60%) — Draft jurnal:**
  - Ringkasan dokumen (vendor, tanggal, no. invoice, total) sebagai *definition list*, bukan kartu.
  - Tabel baris jurnal: Akun (select dengan pencarian, kode mono + nama) · Keterangan · Debit · Kredit.
  - Tiap sel hasil AI memakai **titik amber + tooltip** `AI · ragu 62%`. Setelah diedit, penandanya berubah jadi `dikoreksi` (netral). Sel dengan kode akun yang dialihkan ke default menampilkan ikon peringatan inline.
  - **Warning menempel di baris**, ditambah ringkasan jumlah warning di atas tabel (klik untuk loncat).
  - Multi-blok (mutasi bank): tiap blok berupa section dengan checkbox "Posting" dan Ledger Seam mini.
- **Ledger Seam (sticky bottom bar):** `Debit Rp x | Kredit Rp y` + status: hijau "Balance", merah "Selisih Rp z". Tombol **Posting ke buku** non-aktif saat selisih (validasi server tetap ada, karena ini hanya UX).
- Status duplikat (content_hash / fingerprint) tampil sebagai banner di atas dengan link ke dokumen lama.
- Shortcut: `⌘↵` posting, `Tab` antarsel, `Esc` batal edit.

### P2. Upload dokumen (`documents/create`)

- Dropzone besar (gaya 21st/Kokonut) untuk banyak file, termasuk paste dari clipboard (`paste` event, screenshot mutasi).
- Pilihan jenis sebagai *segmented control*: Foto struk · PDF invoice · Mutasi bank · Teks.
- Progress per file + teks jujur: "AI membaca, 5–20 detik". Tidak ada spinner tanpa keterangan.

### P3. Dashboard

- Baris stat ringkas (bukan 4 kartu identik): satu stat utama **"Perlu direview"** besar + aksi, tiga stat lain kecil inline.
- Tabel posisi klien tetap ada; angka netral, laba negatif dalam kurung.
- Banner "Otak AI belum terpasang" diganti jadi status di sidebar (titik merah/hijau) + halaman bantuan.

### P4. Laporan (Neraca Saldo, Laba Rugi, Neraca)

- Gaya dokumen cetak: lebar maks ~880px, judul + nama klien + periode, garis total tunggal/ganda seperti laporan akuntansi.
- Cek balance tampil sebagai baris status di bawah, bukan badge melayang.
- `@media print` sudah rapi dan menjadi dasar ekspor PDF nanti (HANDOFF §7.5).

### P5. Jurnal, Akun, Klien, Login

- Tabel dengan header sticky, filter di atas, empty state yang menjelaskan langkah berikutnya.
- Void = aksi dengan dialog konfirmasi `<dialog>` (bukan `confirm()`), alasan wajib.
- Login: satu kolom, wordmark, toggle "Tampilkan password" yang sudah ada dipertahankan (dijadikan ikon mata dengan `aria-pressed`).

## 7. Komponen yang dibuat (port ke Blade)

Dibuat sebagai `@utility` di `app.css` + Blade anonymous component di
`resources/views/components/` **hanya kalau dipakai ≥ 2 tempat**.

`button` (primary / secondary / ghost / destructive / sm / loading) · `input`,
`input-rupiah` (prefix Rp, mono, rata kanan) · `select` + combobox akun ·
`table` (sticky header, sel numerik) · `badge` (ok / warn / bad / mute / ai) ·
`ai-marker` · `ledger-seam` · `dialog` · `toast` (ganti `partials/flash`) ·
`dropzone` · `empty-state` · `sidebar` + `client-switcher` · `command-palette`.

## 8. Aksesibilitas & kualitas (wajib lolos)

- Kontras teks ≥ 4.5:1 di light & dark. Status tidak boleh hanya mengandalkan warna (selalu ada ikon/teks: "Balance", "Selisih").
- Focus ring 2px `--ring` di semua elemen interaktif; urutan tab logis di tabel review.
- Target sentuh ≥ 40px di mobile; semua aksi bisa lewat keyboard.
- `lang="id"`, label terhubung ke input, error form inline + `aria-describedby`.
- Lighthouse a11y ≥ 95 untuk login, dashboard, review.

## 9. Batasan teknis

- Tanpa React/Vue/Alpine baru. JS tetap vanilla di `resources/js/app.js` (sekarang 120 baris); tiap fitur JS harus tetap jalan tanpa JS untuk alur inti (posting = form POST biasa).
- Dependency baru yang diizinkan: hanya font `@fontsource-variable/*`. Ikon: SVG inline (set Lucide, disalin per ikon, bukan paket penuh).
- Hasil build (`public/build`) tetap di-commit, karena aplikasi harus bisa jalan tanpa `npm`.
- **Tidak mengubah logika**: validasi balance, isolasi klien, dan anti-dobel tetap di server. 134 test harus tetap hijau; tambahkan test feature hanya untuk perilaku baru (mis. client switcher tidak membocorkan klien lain).

## 10. Urutan eksekusi & definisi selesai

1. Token + font + `@utility` baru di `app.css` (visual berubah global, markup belum).
2. Shell: sidebar, client switcher, topbar, toast.
3. P1 Review dokumen + Ledger Seam + AI marker.
4. P2 Upload → P3 Dashboard → P4 Laporan (+ print CSS) → P5 sisanya.
5. Dark mode (token sudah siap; tinggal toggle + `prefers-color-scheme`).

Tiap langkah selesai bila: `php artisan test` hijau · `npm run build` sukses ·
dicek di browser 375px & 1440px, light & dark · lolos self-critique Impeccable
(*Distill, Sharpen, Contrast*).

## 11. Keputusan terbuka (perlu jawaban Freddie)

1. Aksen **ledger green** `#0f6b55` cocok, atau ada warna brand jasa pembukuan yang mau dipakai?
2. Sidebar kiri, atau tetap nav atas (lebih sedikit perubahan)?
3. Dark mode masuk fase pertama atau ditunda?
4. Wordmark: "Jurnal AI" tetap, atau nama produk baru kalau mau dijual sebagai jasa?

## 12. Status eksekusi (13 Okt 2026)

Keputusan: **sidebar** (Freddie). Default yang dipakai sampai diputuskan lain:
aksen **terakota** `#b5532a` (diganti dari ledger green atas pilihan user; penanda AI digeser ke mustard `#ca8a04` supaya tidak bentrok), dark mode ditunda, wordmark "Jurnal AI".

Selesai: token + Geist self-host (§4), shell sidebar + client switcher + drawer
mobile (§5), P1 review (bukti sticky, penanda AI → "dikoreksi", Ledger Seam yang
mematikan tombol saat tidak balance), P3 dashboard, warna semantik laporan, login
dengan tombol mata. 134 test hijau, 13 halaman 200.

Belum: command palette ⌘K, dropzone + paste di upload (P2), print CSS laporan (P4),
dialog void (P5), toast, dark mode.
