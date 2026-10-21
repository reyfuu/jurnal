import { describe, expect, it } from 'vitest'
import { cekLabaOmzet, cekMargin, keBps, persen, wilayahDariKota } from './kalkulator'

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

describe('batas margin laba & PKP', () => {
  it('laba bersih harus di antara -omzet dan omzet', () => {
    expect(cekLabaOmzet(1_000, 1_000)).toBe('') // margin 100%
    expect(cekLabaOmzet(-1_000, 1_000)).toBe('') // margin -100%
    expect(cekLabaOmzet(0, 0)).toBe('')
    const lebih = cekLabaOmzet(1_500_000_000, 1_000_000_000)
    expect(lebih).toContain('lebih besar dari omzet')
    expect(lebih).toContain('150%')
    expect(lebih).toContain('periksa lagi')
    expect(cekLabaOmzet(-1_000_000_001, 1_000_000_000)).toContain('Kerugian')
  })
  it('margin persen -100% sampai 100%, boleh koma dan minus', () => {
    for (const ok of ['100', '-100', '12,5', '-20', '', '-']) expect(cekMargin(ok)).toBe('')
    for (const salah of ['100,01', '-150', '150']) expect(cekMargin(salah)).toContain('-100% dan 100%')
  })
})
