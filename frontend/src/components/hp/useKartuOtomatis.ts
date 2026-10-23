import { useEffect } from 'react'
import { kolomHeader, labelSel, selAksi, useHP } from '@/lib/hp'

const KELAS = 'kartu-otomatis'

/** Isi data-label tiap sel dari header tabel, supaya CSS HP bisa menampilkan baris sebagai kartu. */
function labeli(tabel: HTMLTableElement) {
  if (!tabel.tHead?.rows.length) return
  // Semua baris header (rowSpan/colSpan), mis. "Masuk · Beli" di Mutasi stok.
  const header = kolomHeader([...tabel.tHead.rows].map((r) => [...r.cells].map((c) =>
    ({ teks: (c.textContent ?? '').trim(), span: c.colSpan || 1, rowSpan: c.rowSpan || 1 })))).map((teks) => ({ teks, span: 1 }))
  // baris total di tfoot ikut diberi label
  for (const body of [...tabel.tBodies, ...(tabel.tFoot ? [tabel.tFoot] : [])]) {
    for (const tr of body.rows) {
      const sel = [...tr.cells]
      const label = labelSel(header, sel.map((c) => c.colSpan || 1))
      sel.forEach((c, i) => {
        if (c.dataset.label !== label[i]) c.dataset.label = label[i]
        // pilihan (Select / combobox) bukan tombol aksi
        const tombol = [...c.querySelectorAll('button, a.btn, [role="button"]')].filter((b) => b.getAttribute('role') !== 'combobox')
        c.classList.toggle('sel-aksi', i > 0 && selAksi(c.textContent ?? '', tombol.map((b) => b.textContent ?? '')))
      })
      // Baris satu sel selebar tabel: judul kelompok (section-head) atau pesan (mis. "Belum ada barang.")
      const penuh = sel.length === 1 && label[0] === ''
      tr.classList.toggle('kartu-judul-kelompok', penuh && sel[0].classList.contains('section-head'))
      tr.classList.toggle('kartu-baris-pesan', penuh && !sel[0].classList.contains('section-head'))
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
    // Tabel di area kerja + di dialog (dirender di luar area kerja, mis. pratinjau impor).
    const semua = () => [...el.querySelectorAll<HTMLTableElement>('table.data-table'),
      ...document.querySelectorAll<HTMLTableElement>('[role="dialog"] table.data-table')]
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
    pengamat.observe(document.body, { childList: true, subtree: true, characterData: true })
    return () => { pengamat.disconnect(); if (jadwal) cancelAnimationFrame(jadwal) }
  }, [hp, el])
}
