# Tampilan HP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Aplikasi utama dan Panel Master rapi dan nyaman dipakai di HP (360–430 px): menu jadi drawer ☰, tabel jadi kartu, aksi Panel Master ringkas, search & filter hemat tempat. Desktop tidak berubah.

**Architecture:** Semua perubahan hanya aktif di `@media (max-width: 760px)` (CSS) atau saat `useHP()` true (JSX khusus HP). Sidebar yang ada dipakai ulang sebagai drawer. Tabel `data-table` biasa diubah jadi kartu oleh CSS memakai atribut `data-label` yang diisi otomatis dari header tabel. Empat halaman (Buku Besar, Jurnal Umum, Pengguna Klien, Tim & Akses) punya JSX kartu khusus.

**Tech Stack:** React 19, TypeScript, Vite, Tailwind 4 + `frontend/src/index.css`, lucide-react, vitest (tanpa DOM: hanya logika murni).

**Spec:** `docs/superpowers/specs/2026-10-23-tampilan-hp-design.md`

**Branch:** `feat/tampilan-hp` dari `fix/beban-marketplace-tanpa-kurung` (PR #135), karena kartu Buku Besar memakai No. Bukti & penanda saldo minus dari #134. PR base = `fix/beban-marketplace-tanpa-kurung`.

## Global Constraints

- Breakpoint HP: `max-width: 760px` (sama dengan yang ada). Di atas itu tampilan tidak boleh berubah.
- Warna hanya lewat token yang ada (`--bg-card`, `--text-primary`, `--text-muted`, `--text-danger`, `--accent-red-soft`, `--border-subtle`, `--bg-sidebar`, …); tema terang & gelap harus jalan.
- Target ketuk ≥ 40 px; animasi ±200 ms, dimatikan di `prefers-reduced-motion: reduce`.
- Tidak ada scroll horizontal di level halaman pada 390 px (`document.documentElement.scrollWidth === innerWidth`).
- Tanpa dependency baru. Commit tanpa atribusi AI (AGENTS.md). Penulis commit: Ravent.
- Teks UI Bahasa Indonesia, gaya yang sama dengan aplikasi.

## Review Focus

1. **Drawer + tombol Back HP**: Back saat drawer terbuka menutup drawer, bukan meninggalkan halaman; Back saat tertutup tetap navigasi biasa (riwayat `#/…` tidak rusak).
2. **Ubah ukuran layar / putar HP**: drawer terbuka lalu layar jadi > 760 px → sidebar desktop normal, tidak ada scrim/scroll terkunci yang tertinggal.
3. **Tabel dengan `colSpan`** (baris judul kelompok, baris total) di kartu otomatis: label kolom tidak bergeser; baris satu sel lebar tampil sebagai judul kelompok.
4. **Tabel yang dimuat/diubah setelah render** (filter, ganti periode, dialog impor): label kartu otomatis ikut diperbarui.
5. **Aksi berbahaya dari ⋯** (Hapus, Nonaktifkan) tetap melewati dialog konfirmasi yang sudah ada; aksi yang tidak diizinkan tidak muncul di sheet.

---

### Task 1: Dasar — `useHP` dan pemetaan label kolom

**Files:**
- Create: `frontend/src/lib/hp.ts`
- Test: `frontend/src/lib/hp.test.ts`

**Interfaces:**
- Produces: `useHP(): boolean` (true bila `matchMedia('(max-width: 760px)')` cocok; `useSyncExternalStore`, aman bila `matchMedia` tidak ada → false).
- Produces: `labelSel(header: { teks: string; span: number }[], sel: number[]): string[]` — `sel` = colSpan tiap sel satu baris; hasil = label per sel, yaitu teks header di kolom awal sel itu (`''` bila di luar header). Baris satu sel dengan span ≥ jumlah kolom header → `['']`.

- [ ] **Step 1: Tulis tes `hp.test.ts`**

```ts
import { describe, expect, it } from 'vitest'
import { labelSel } from './hp'
const H = [{ teks: 'Kode', span: 1 }, { teks: 'Akun', span: 1 }, { teks: 'Debit', span: 1 }, { teks: 'Kredit', span: 1 }]
describe('labelSel', () => {
  it('satu sel per kolom', () => expect(labelSel(H, [1, 1, 1, 1])).toEqual(['Kode', 'Akun', 'Debit', 'Kredit']))
  it('colSpan menggeser label', () => expect(labelSel(H, [2, 1, 1])).toEqual(['Kode', 'Debit', 'Kredit']))
  it('baris judul kelompok', () => expect(labelSel(H, [4])).toEqual(['']))
  it('header ber-colSpan', () => expect(labelSel([{ teks: 'Tanggal', span: 1 }, { teks: 'Nilai', span: 2 }], [1, 1, 1])).toEqual(['Tanggal', 'Nilai', 'Nilai']))
  it('sel lebih banyak dari header', () => expect(labelSel(H.slice(0, 2), [1, 1, 1])).toEqual(['Kode', 'Akun', '']))
})
```

- [ ] **Step 2: `npx vitest run src/lib/hp.test.ts`** → FAIL (modul belum ada)
- [ ] **Step 3: Implementasi `hp.ts`** (dua fungsi di atas)
- [ ] **Step 4: `npx vitest run src/lib/hp.test.ts`** → PASS
- [ ] **Step 5: Commit** `feat(hp): useHP & pemetaan label kolom kartu`

### Task 2: Drawer menu ☰ (aplikasi utama & Panel Master)

**Files:**
- Create: `frontend/src/components/hp/Laci.tsx`
- Modify: `frontend/src/components/Sidebar.tsx`, `frontend/src/App.tsx` (sekitar baris 271–346: shell, bar periode), `frontend/src/views/master/PanelMaster.tsx` (baris 73–139), `frontend/src/index.css` (blok `@media (max-width: 760px)` sekitar baris 1032 — ganti strip menu horizontal)

**Interfaces:**
- Consumes: `useHP()` (Task 1).
- Produces: `useLaci(): { buka: boolean; setBuka(b: boolean): void }` — saat dibuka mendorong satu entri riwayat (`history.pushState({ laci: true }, '')`) dan menutup di `popstate`; menutup sendiri bila `useHP()` jadi false; mengunci scroll body selagi terbuka.
- Produces: `<BarAtasHP onMenu={() => void} judul?: ReactNode>{children}</BarAtasHP>` — bar atas tipis (☰ kiri, logo, `children` di kanan untuk pil periode), hanya dirender saat `useHP()`.
- Produces: `<Laci buka onTutup>{sidebar}</Laci>` — membungkus `<aside class="sidebar">`: kelas `laci-terbuka`, scrim yang menutup saat diketuk, geser kiri ≥ 60 px menutup, `Escape` menutup.
- `Sidebar` dan nav Panel Master memanggil `onNavigate`/`go` lalu `setBuka(false)`.

- [ ] **Step 1: Implementasi `Laci.tsx`** (`useLaci`, `BarAtasHP`, `Laci`)
- [ ] **Step 2: Pasang di `App.tsx`**: `BarAtasHP` di atas `.app-shell`, pil periode (select periode yang sama) di dalamnya; bar periode di workspace disembunyikan di HP (`.period-bar-hp-sembunyi` di media query). Navigasi menutup drawer.
- [ ] **Step 3: Pasang di `PanelMaster.tsx`** (judul "Panel Master", tanpa periode); `go()` menutup drawer.
- [ ] **Step 4: CSS HP**: hapus aturan strip horizontal (`.sidebar-nav` flex-row, `.nav-group{display:contents}`, label disembunyikan, dll.); `.sidebar` jadi `position: fixed; inset: 0 auto 0 0; width: min(300px, 84vw); transform: translateX(-100%)`, `.laci-terbuka .sidebar` → `translateX(0)`; scrim `rgba(10,14,30,.45)`; grup & submenu tampil seperti desktop; `.tenant-box` tetap di bawah drawer.
- [ ] **Step 5: Peringatan saldo minus dapat dilipat di HP** (`components/PeringatanSaldoMinus.tsx`): di HP default terlipat menampilkan "Ada N saldo minus ▾"; ketuk membuka daftar. Desktop tetap terbuka.
- [ ] **Step 6: Tombol "Tanya AI Pajak" di HP** jadi bulat ikon saja 48 px (CSS di HP; label disembunyikan, `aria-label` tetap).
- [ ] **Step 7: Verifikasi** `npx tsc -b` bersih; tangkapan 390 px Dashboard (drawer tertutup & terbuka) dan 1440 px Dashboard tidak berubah; uji Back menutup drawer (Review Focus 1) dan resize > 760 px (Review Focus 2) lewat skrip CDP.
- [ ] **Step 8: Commit** `feat(hp): menu jadi drawer ☰ dengan bar atas tipis`

### Task 3: Kartu otomatis untuk tabel data

**Files:**
- Create: `frontend/src/components/hp/useKartuOtomatis.ts`
- Modify: `frontend/src/App.tsx`, `frontend/src/views/master/PanelMaster.tsx` (panggil hook pada elemen `.workspace`), `frontend/src/index.css`

**Interfaces:**
- Consumes: `useHP()`, `labelSel()` (Task 1).
- Produces: `useKartuOtomatis(root: RefObject<HTMLElement | null>): void` — saat `useHP()`: untuk tiap `table.data-table` di `root` yang tidak ber-`data-kartu="khusus"`, beri kelas `kartu-otomatis` dan isi `data-label` tiap `td` dari `labelSel` (header = `thead tr:last-child th`). Diperbarui lewat `MutationObserver` (childList, subtree) dengan penundaan satu frame. Saat bukan HP: hapus kelas.
- Tabel yang tidak cocok dijadikan kartu boleh diberi `data-kartu="khusus"` (dirender sendiri) atau `data-kartu="tidak"` (tetap tabel yang bisa digeser).

- [ ] **Step 1: Implementasi hook**
- [ ] **Step 2: CSS HP untuk `.kartu-otomatis`**: `thead` disembunyikan; `tr` = kartu (`--bg-card`, radius 12, padding 10–12, jarak 8); `td` = baris flex `label kiri (::before attr(data-label), --text-muted, 11px) — nilai kanan`; `td:first-child` = judul kartu tebal tanpa label; `td[data-label=""]` penuh lebar; baris satu sel lebar = judul kelompok (huruf kecil kapital, tanpa kotak kartu); `td:has(button)` = deret tombol kecil rata kanan; hapus aturan lama `.data-table{display:block;overflow-x:auto;white-space:nowrap}` untuk kartu otomatis (tetap berlaku bagi `data-kartu="tidak"`).
- [ ] **Step 3: Verifikasi** tangkapan 390 px: Transaksi, Tagihan, Stok & Gudang, Saldo Awal, Laporan Transaksi, Panel Master Klien — tanpa scroll horizontal halaman, label benar pada tabel ber-colSpan (Review Focus 3), dan label ikut berubah setelah ganti filter (Review Focus 4). Tabel yang hasilnya buruk: tandai `data-kartu="tidak"` dan catat di PR.
- [ ] **Step 4: Commit** `feat(hp): tabel data jadi kartu otomatis di HP`

### Task 4: Komponen HP bersama — kepala ringkas & lembar bawah

**Files:**
- Create: `frontend/src/components/hp/KepalaHP.tsx`, `frontend/src/components/hp/LembarBawah.tsx`
- Modify: `frontend/src/index.css`

**Interfaces:**
- Produces: `<LembarBawah buka judul onTutup>{children}</LembarBawah>` — panel dari bawah dengan scrim, pegangan, `role="dialog"`, `aria-modal`, `Escape`/ketuk scrim menutup, fokus pindah ke panel, tinggi maks 85vh dengan gulir dalam.
- Produces: `<KepalaHP judul cari onCari placeholderCari jumlahFilter onFilter chips={{ label: string; onHapus(): void }[]} />` — satu baris: judul, ikon 🔍 (`Search`), ikon ⚙ (`SlidersHorizontal`) dengan lencana jumlah filter. Ketuk 🔍 → input menggantikan judul (autofocus), ✕ menutup & mengosongkan (`onCari('')`). Di bawahnya label filter aktif yang bisa dihapus. Dipakai hanya saat `useHP()`.

- [ ] **Step 1: Implementasi kedua komponen + CSS**
- [ ] **Step 2: `npx tsc -b`** bersih
- [ ] **Step 3: Commit** `feat(hp): kepala ringkas & lembar bawah untuk HP`

### Task 5: Buku Besar & Jurnal Umum versi kartu

**Files:**
- Modify: `frontend/src/views/BukuBesar.tsx`, `frontend/src/views/JurnalUmum.tsx`

**Interfaces:**
- Consumes: `useHP()`, `KepalaHP`, `LembarBawah` (Task 1, 4).

- [ ] **Step 1: Buku Besar di HP** (tabel desktop diberi `data-kartu="khusus"` dan tidak dirender di HP): `KepalaHP` (cari = filter akun yang sudah ada, ⚙ = rentang tanggal + pilihan cepat + akun + "Tampilkan akun tanpa saldo" di `LembarBawah` dengan tombol Terapkan); judul kelompok; kartu akun (nama, label "saldo minus", grid 2×2 saldo awal/debit/kredit/saldo akhir dengan warna merah yang sama dengan desktop); ketuk kartu membuka mutasi: No. Bukti, tanggal, keterangan, jumlah `+debit`/`−kredit`, saldo berjalan, baris "mulai minus" disorot. Ekspor Excel tetap tersedia (di lembar filter atau ikon di kepala).
- [ ] **Step 2: Jurnal Umum di HP**: `KepalaHP` (cari = `q` yang sudah ada; ⚙ = rentang + akun); satu kartu per bukti: No. Bukti, tanggal, keterangan; baris debit rata kiri, kredit menjorok, nominal kanan; total & status seimbang di atas daftar.
- [ ] **Step 3: Verifikasi** tangkapan 390 px kedua halaman (kartu terbuka & filter terbuka) dan 1440 px tidak berubah; `npx tsc -b`.
- [ ] **Step 4: Commit** `feat(hp): Buku Besar & Jurnal Umum versi kartu di HP`

### Task 6: Panel Master — Pengguna Klien, Tim & Akses, Klien ringkas

**Files:**
- Modify: `frontend/src/views/master/PenggunaKlien.tsx` (tabel baris ~60–97), `frontend/src/views/master/Tim.tsx` (tabel baris ~75–100), `frontend/src/views/master/Klien.tsx` (daftar klien)

**Interfaces:**
- Consumes: `useHP()`, `KepalaHP`, `LembarBawah`.

- [ ] **Step 1: Pengguna Klien di HP**: satu baris per user — titik status (hijau aktif / abu nonaktif), nama (terpotong …), di bawahnya email · usaha; kanan: ✎ (Ubah, bila `bisaKelola`) dan ⋯. ⋯ membuka `LembarBawah` berjudul nama user: di atas rincian kolom yang tidak muat (no HP, alamat, role, langganan & masa berlaku, terakhir aktif, diarsipkan), di bawah hanya aksi yang diizinkan: Atur langganan / Buka semua fitur (`bisaLangganan`, aksi yang sama dengan kolom langganan), Reset password (`bisaReset`), Nonaktifkan/Aktifkan (`bisaKelola`), Hapus merah (`bisaHapus`) — masing-masing memanggil handler/dialog konfirmasi yang sudah ada (Review Focus 5). Tab Semua/Aktif/Nonaktif jadi chip; kotak cari lewat `KepalaHP`.
- [ ] **Step 2: Tim & Akses di HP**: baris kecil `role · klien ditugaskan`, "(Anda)" untuk akun sendiri, titik abu bila nonaktif; ✎ = ubah role & klien; ⋯ = Nonaktifkan/Aktifkan, Hapus. Tabel tab "Role & izin" diberi `data-kartu="tidak"` dan kolom pertamanya `position: sticky; left: 0` di HP.
- [ ] **Step 2b: Daftar Klien di HP**: baris ringkas (nama usaha, status langganan, pemilik); ketuk baris = buka detail (perilaku yang ada); ⋯ = rincian kolom lain + Atur langganan (`bisaLangganan`) + Lihat sebagai (`bisaLihat`).
- [ ] **Step 3: Verifikasi** tangkapan 390 px (daftar & sheet terbuka) sebagai master; aksi Hapus memunculkan dialog konfirmasi; 1440 px tidak berubah.
- [ ] **Step 4: Commit** `feat(hp): Pengguna Klien & Tim ringkas dengan menu aksi ⋯`

### Task 7: Search & filter ringkas di Transaksi & Laporan Transaksi, sapuan akhir, PR

**Files:**
- Modify: `frontend/src/views/Transaksi.tsx`, `frontend/src/views/LaporanTransaksi.tsx`, lalu file mana pun yang ketahuan rusak di sapuan.

- [ ] **Step 1: Transaksi & Laporan Transaksi di HP**: kotak cari & filter yang ada dipindah ke `KepalaHP` + `LembarBawah` (nilai filter dan logika tidak berubah).
- [ ] **Step 2: Sapuan 390 px** semua halaman prioritas di spec (Dashboard, Transaksi, Laporan Keuangan semua tab, Buku Besar, Jurnal Umum, Laporan Transaksi, Tagihan, Stok & Gudang, Saldo Awal, Profil Usaha, Panel Master Ringkasan/Klien/Pengguna Klien/Tim) + Landing & login: `scrollWidth === innerWidth`, tidak ada tombol terpotong; perbaiki yang rusak.
- [ ] **Step 3: Sapuan 1440 px** halaman yang sama: tidak berubah dibanding sebelum branch.
- [ ] **Step 4:** `npx tsc -b`, `npm run lint` (tanpa peringatan baru), `npx vitest run`, `npm run build`, `go test ./...` (hanya 2 kegagalan Windows yang sudah diketahui).
- [ ] **Step 5: Commit, push, PR** (base `fix/beban-marketplace-tanpa-kurung`), deskripsi berisi tangkapan sebelum/sesudah & cara uji di HP (`http://<IP-laptop>:8080`).
