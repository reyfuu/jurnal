import { describe, expect, it } from 'vitest'
import { angka, bacaAngka, posisiKursor, ribuan } from './angka'

describe('ribuan', () => {
  it('memberi titik pemisah ribuan', () => {
    expect(ribuan(10_000_000)).toBe('10.000.000')
    expect(ribuan(950)).toBe('950')
    expect(ribuan('2500000')).toBe('2.500.000')
  })
  it('kosong untuk 0, kosong, atau bukan angka', () => {
    expect(ribuan(0)).toBe('')
    expect(ribuan('')).toBe('')
    expect(ribuan(undefined)).toBe('')
    expect(ribuan('abc')).toBe('')
  })
})

describe('bacaAngka', () => {
  it('mengabaikan titik, spasi, dan Rp', () => {
    expect(bacaAngka('10.000.000')).toBe(10_000_000)
    expect(bacaAngka('Rp 1.250')).toBe(1250)
    expect(bacaAngka('')).toBe(0)
  })
})

describe('angka', () => {
  it('menghasilkan props input teks numerik berformat ribuan', () => {
    let nilai = 0
    const p = angka(1500000, (n) => { nilai = n })
    expect(p).toMatchObject({ type: 'text', inputMode: 'numeric', value: '1.500.000' })
    p.onChange({ target: { value: '1.500.0007' } } as React.ChangeEvent<HTMLInputElement>)
    expect(nilai).toBe(15_000_007)
  })
})

describe('posisiKursor (checklist No 28: kursor tidak melompat ke akhir)', () => {
  it('kursor tetap setelah digit yang sama walau titik ribuan bergeser', () => {
    // "1.250.000", hapus "2" (kursor setelah "1") → 150000 → "150.000", kursor setelah "1"
    expect(posisiKursor('150.000', 1)).toBe(1)
    // sisip "9" setelah "1.2" pada "1.250.000" → 1.925.0000? digit kiri = 3 → "19.250.000" posisi setelah "19.2"
    expect(posisiKursor('19.250.000', 3)).toBe(4)
  })
  it('di ujung kiri dan kanan', () => {
    expect(posisiKursor('1.000', 0)).toBe(0)
    expect(posisiKursor('1.000', 4)).toBe(5)
    expect(posisiKursor('', 0)).toBe(0)
  })
})
