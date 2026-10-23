import { useSyncExternalStore } from 'react'

// Tampilan HP: sama dengan breakpoint ponsel di index.css.
export const KUERI_HP = '(max-width: 760px)'

function langganan(ubah: () => void) {
  if (typeof window === 'undefined' || !window.matchMedia) return () => {}
  const mq = window.matchMedia(KUERI_HP)
  mq.addEventListener('change', ubah)
  return () => mq.removeEventListener('change', ubah)
}
const cocok = () => typeof window !== 'undefined' && !!window.matchMedia && window.matchMedia(KUERI_HP).matches

/** true bila layar selebar HP (≤ 760 px); ikut berubah saat layar diputar / diubah ukurannya. */
export function useHP(): boolean {
  return useSyncExternalStore(langganan, cocok, () => false)
}

/**
 * Label per kolom dari semua baris header (mendukung rowSpan & colSpan). Kolom di bawah header
 * kelompok diberi label "Kelompok · Sub", mis. "Masuk · Beli".
 */
export function kolomHeader(baris: { teks: string; span: number; rowSpan: number }[][]): string[] {
  const grid: string[][] = baris.map(() => [])
  baris.forEach((cells, r) => {
    let c = 0
    for (const h of cells) {
      while (grid[r][c] !== undefined) c++ // dilewati: terisi rowSpan dari baris atas
      for (let dr = 0; dr < Math.max(1, h.rowSpan); dr++) {
        if (!grid[r + dr]) continue
        for (let dc = 0; dc < Math.max(1, h.span); dc++) grid[r + dr][c + dc] = h.teks
      }
      c += Math.max(1, h.span)
    }
  })
  const lebar = Math.max(0, ...grid.map((g) => g.length))
  return Array.from({ length: lebar }, (_, c) =>
    [...new Set(grid.map((g) => g[c]).filter((t): t is string => !!t))].join(' · '))
}

/**
 * Label kolom untuk tiap sel satu baris tabel (kartu HP): teks header di kolom tempat sel itu
 * mulai. `sel` = colSpan tiap sel. Baris satu sel yang selebar tabel (judul kelompok) → [''].
 */
export function labelSel(header: { teks: string; span: number }[], sel: number[]): string[] {
  const kolom = header.flatMap((h) => Array<string>(Math.max(1, h.span)).fill(h.teks))
  if (sel.length === 1 && sel[0] >= kolom.length) return ['']
  let mulai = 0
  return sel.map((span) => {
    const label = kolom[mulai] ?? ''
    mulai += Math.max(1, span)
    return label
  })
}

/**
 * Sel tabel yang isinya hanya tombol (Ubah, Hapus, ikon …) = deret aksi di kartu HP. Tombol di
 * tengah kalimat (mis. tautan saran) tidak membuat sel itu jadi sel aksi.
 */
export function selAksi(teksSel: string, teksTombol: string[]): boolean {
  if (teksTombol.length === 0) return false
  let sisa = teksSel
  for (const t of teksTombol) if (t) sisa = sisa.replace(t, '')
  return sisa.trim() === ''
}
