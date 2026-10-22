import type { ChangeEvent } from 'react'

// Input bilangan bulat (rupiah, jumlah, hari) yang tampil dengan titik ribuan saat diketik.
// Bukan untuk input desimal (persen, takaran) maupun tahun.

/** 10000000 → "10.000.000"; 0/kosong → "" supaya placeholder terlihat. */
export const ribuan = (v: number | string | undefined | null) => {
  const n = Number(v)
  return v === '' || v == null || !n ? '' : n.toLocaleString('id-ID', { maximumFractionDigits: 0 })
}

/** "Rp 10.000.000" → 10000000; kosong → 0. */
export const bacaAngka = (s: string) => Number(s.replace(/\D/g, '')) || 0

/** Posisi kursor di teks berformat tepat setelah digit ke-`digitKiri` (titik ribuan dilewati). */
export const posisiKursor = (teks: string, digitKiri: number) => {
  if (digitKiri <= 0) return 0
  let n = 0
  for (let i = 0; i < teks.length; i++) if (/\d/.test(teks[i]) && ++n === digitKiri) return i + 1
  return teks.length
}

/** Props siap sebar ke <input>/<Input>: `<input {...angka(nilai, setNilai)} />`.
 *  Titik ribuan disusun ulang setiap ketikan; kursor dikembalikan ke digit yang sama supaya
 *  mengedit di tengah angka tidak melompat ke akhir. */
export const angka = (v: number | string | undefined | null, set: (n: number) => void) => ({
  type: 'text',
  inputMode: 'numeric' as const,
  value: ribuan(v),
  onChange: (e: ChangeEvent<HTMLInputElement>) => {
    const el = e.target
    const n = bacaAngka(el.value)
    // digit di kiri kursor; nol di depan ikut hilang saat diformat, jadi tidak dihitung
    const digitKiri = el.value.slice(0, el.selectionStart ?? el.value.length).replace(/\D/g, '').replace(/^0+/, '').length
    set(n)
    // setelah React menulis nilai berformat (yang memindahkan kursor ke akhir)
    if (typeof requestAnimationFrame === 'function' && document.activeElement === el) {
      requestAnimationFrame(() => {
        const p = posisiKursor(el.value, digitKiri)
        el.setSelectionRange(p, p)
      })
    }
  },
})
