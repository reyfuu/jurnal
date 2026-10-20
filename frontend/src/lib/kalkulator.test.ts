import { describe, expect, it } from 'vitest'
import { keBps, persen, wilayahDariKota } from './kalkulator'

describe('kalkulator', () => {
  it('membaca persen gaya Indonesia ke basis poin', () => {
    expect(keBps('12,5')).toBe(1250)
    expect(keBps('0.5')).toBe(50)
    expect(keBps('')).toBe(0)
    expect(persen(175)).toBe('1,75%')
  })
  it('hanya 10 ibukota yang bisa ditebak dari nama kota', () => {
    expect(wilayahDariKota('Kota Surabaya')).toBe('ibukota10')
    expect(wilayahDariKota('Jakarta Selatan')).toBe('ibukota10')
    expect(wilayahDariKota('Malang')).toBe('')
    expect(wilayahDariKota(undefined)).toBe('')
  })
})
