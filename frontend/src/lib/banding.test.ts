import { describe, expect, it } from 'vitest'
import { bulanRentang, tahunBerjalan } from './banding'

const bulan = (dari: string, n: number) => Array.from({ length: n }, (_, i) => {
  const d = new Date(Number(dari.slice(0, 4)), Number(dari.slice(5, 7)) - 1 + i, 1)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
})

describe('pilihan bulan Banding', () => {
  const ada = bulan('2025-01', 22) // Jan 2025 – Okt 2026

  it('tahun berjalan: Januari sampai bulan aktif di tahun yang sama', () => {
    expect(tahunBerjalan(ada, '2026-10')).toEqual(bulan('2026-01', 10))
    expect(tahunBerjalan(ada, '2025-03')).toEqual(['2025-01', '2025-02', '2025-03'])
  })

  it('tahun berjalan di Januari hanya satu bulan', () => {
    expect(tahunBerjalan(ada, '2026-01')).toEqual(['2026-01'])
  })

  it('rentang dari–sampai, urutan input boleh terbalik', () => {
    expect(bulanRentang(ada, '2026-03', '2026-06')).toEqual(['2026-03', '2026-04', '2026-05', '2026-06'])
    expect(bulanRentang(ada, '2026-06', '2026-03')).toEqual(['2026-03', '2026-04', '2026-05', '2026-06'])
  })

  it('rentang lebih dari 12 bulan dipotong ke 12 bulan terakhir', () => {
    const r = bulanRentang(ada, '2025-01', '2026-10')
    expect(r).toHaveLength(12)
    expect(r[0]).toBe('2025-11')
    expect(r.at(-1)).toBe('2026-10')
  })

  it('hanya bulan yang ada datanya', () => {
    expect(bulanRentang(['2026-01', '2026-03'], '2026-01', '2026-03')).toEqual(['2026-01', '2026-03'])
  })
})
