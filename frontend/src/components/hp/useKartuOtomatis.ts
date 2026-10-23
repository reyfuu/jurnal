import { useEffect } from 'react'
import { labelSel, useHP } from '@/lib/hp'

const KELAS = 'kartu-otomatis'

/** Isi data-label tiap sel dari header tabel, supaya CSS HP bisa menampilkan baris sebagai kartu. */
function labeli(tabel: HTMLTableElement) {
  const head = tabel.tHead?.rows[tabel.tHead.rows.length - 1]
  if (!head) return
  const header = [...head.cells].map((c) => ({ teks: (c.textContent ?? '').trim(), span: c.colSpan || 1 }))
  for (const body of tabel.tBodies) {
    for (const tr of body.rows) {
      const sel = [...tr.cells]
      const label = labelSel(header, sel.map((c) => c.colSpan || 1))
      sel.forEach((c, i) => { if (c.dataset.label !== label[i]) c.dataset.label = label[i] })
      tr.classList.toggle('kartu-judul-kelompok', sel.length === 1 && label[0] === '')
    }
  }
  tabel.classList.add(KELAS)
}

/**
 * Di HP, tabel `.data-table` di dalam root tampil sebagai kartu (lihat index.css). Tabel dengan
 * data-kartu="khusus" (punya tampilan HP sendiri) atau "tidak" (tetap tabel geser) dilewati.
 */
export function useKartuOtomatis(el: HTMLElement | null) {
  const hp = useHP()
  useEffect(() => {
    if (!el) return
    const semua = () => [...el.querySelectorAll<HTMLTableElement>('table.data-table')]
    if (!hp) {
      semua().forEach((t) => t.classList.remove(KELAS))
      return
    }
    let jadwal = 0
    const jalan = () => {
      jadwal = 0
      for (const t of semua()) if (!t.dataset.kartu && !t.closest('[data-kartu]')) labeli(t)
    }
    jalan()
    // Tabel dimuat / disaring setelah render: label diperbarui sekali per frame.
    const pengamat = new MutationObserver(() => { if (!jadwal) jadwal = requestAnimationFrame(jalan) })
    pengamat.observe(el, { childList: true, subtree: true, characterData: true })
    return () => { pengamat.disconnect(); if (jadwal) cancelAnimationFrame(jadwal) }
  }, [hp, el])
}
