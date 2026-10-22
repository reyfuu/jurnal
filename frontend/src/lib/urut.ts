import { useMemo, useState } from 'react'

// Urutkan tabel dengan klik judul kolom: klik pertama naik (A→Z, kecil→besar), klik lagi turun,
// klik ketiga kembali ke urutan bawaan.

export type Arah = 'naik' | 'turun'
export type Pengurut<T> = Record<string, (r: T) => string | number>

export function urutkan<T>(rows: T[], ambil: ((r: T) => string | number) | undefined, arah: Arah): T[] {
  if (!ambil) return rows
  const kali = arah === 'naik' ? 1 : -1
  return [...rows].sort((a, b) => {
    const x = ambil(a), y = ambil(b)
    const c = typeof x === 'number' && typeof y === 'number' ? x - y : String(x).localeCompare(String(y), 'id', { numeric: true, sensitivity: 'base' })
    return c * kali
  })
}

export function useUrut<T>(rows: T[], kolom: Pengurut<T>) {
  const [kunci, setKunci] = useState<string | null>(null)
  const [arah, setArah] = useState<Arah>('naik')
  const hasil = useMemo(() => urutkan(rows, kunci ? kolom[kunci] : undefined, arah), [rows, kunci, arah, kolom])
  const atur = (k: string) => {
    if (kunci !== k) { setKunci(k); setArah('naik') }
    else if (arah === 'naik') setArah('turun')
    else { setKunci(null); setArah('naik') }
  }
  return { rows: hasil, kunci, arah, atur }
}

/** Hanya status urutan (kolom & arah), untuk komponen yang menghitung barisnya sesudah return awal. */
export function useKolomUrut() {
  const [kunci, setKunci] = useState<string | null>(null)
  const [arah, setArah] = useState<Arah>('naik')
  const atur = (k: string) => {
    if (kunci !== k) { setKunci(k); setArah('naik') }
    else if (arah === 'naik') setArah('turun')
    else { setKunci(null); setArah('naik') }
  }
  return { kunci, arah, atur }
}
