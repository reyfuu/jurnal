import { describe, expect, it } from 'vitest'
import { angkaSifat } from './angkaLaporan'

describe('angkaSifat', () => {
  it('beban & HPP hitam tanpa minus, kontra & rugi merah dalam kurung', () => {
    expect(angkaSifat(-18_400_000, 'pengurang')).toEqual({ teks: '18.400.000', merah: false })
    expect(angkaSifat(500, 'pengurang')).toEqual({ teks: '(500)', merah: true })
    expect(angkaSifat(-625_000, 'kontra')).toEqual({ teks: '(625.000)', merah: true })
    expect(angkaSifat(-1_000)).toEqual({ teks: '(1.000)', merah: true })
    expect(angkaSifat(2_000)).toEqual({ teks: '2.000', merah: false })
  })
})
