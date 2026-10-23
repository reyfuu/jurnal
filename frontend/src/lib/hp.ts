import { useSyncExternalStore } from 'react'

// Tampilan HP: sama dengan breakpoint ponsel di index.css.
const KUERI_HP = '(max-width: 760px)'

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
