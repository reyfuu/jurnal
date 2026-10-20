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

/** Props siap sebar ke <input>/<Input>: `<input {...angka(nilai, setNilai)} />`. */
export const angka = (v: number | string | undefined | null, set: (n: number) => void) => ({
  type: 'text',
  inputMode: 'numeric' as const,
  value: ribuan(v),
  onChange: (e: ChangeEvent<HTMLInputElement>) => set(bacaAngka(e.target.value)),
})
