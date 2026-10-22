import { describe, expect, it } from 'vitest'
import { urutkan } from './urut'

describe('urutkan', () => {
  const rows = [{ n: 'Gula', h: 17500 }, { n: 'beras 10', h: 120000 }, { n: 'Beras 2', h: 26000 }]
  it('teks tanpa beda huruf besar & angka alami, angka numerik, dua arah', () => {
    expect(urutkan(rows, (r) => r.n, 'naik').map((r) => r.n)).toEqual(['Beras 2', 'beras 10', 'Gula'])
    expect(urutkan(rows, (r) => r.h, 'turun').map((r) => r.h)).toEqual([120000, 26000, 17500])
    expect(urutkan(rows, undefined, 'naik')).toBe(rows)
  })
})
