import { describe, expect, it } from 'vitest'
import { kalimatPerubahan, labelPeriode } from './simulasi'

describe('kalimatPerubahan', () => {
  it('sama dengan sekarang bila tidak berubah', () => {
    expect(kalimatPerubahan(22_940_079, 22_940_079)).toBe('Sama dengan kondisi sekarang. Geser slider atau pilih skenario di kiri.')
  })
  it('menyebut selisih rupiah dan persen', () => {
    expect(kalimatPerubahan(22_940_079, 27_305_446)).toBe('Naik Rp4.365.367 (+19%) dibanding sekarang.')
    expect(kalimatPerubahan(22_940_079, 19_940_079)).toBe('Turun Rp3.000.000 (−13%) dibanding sekarang.')
  })
  it('rugi disebut rugi', () => {
    expect(kalimatPerubahan(10_000_000, -2_000_000)).toBe('Turun Rp12.000.000 (−120%) dibanding sekarang — usaha jadi rugi.')
  })
})

describe('labelPeriode', () => {
  it('menyingkat bulan dasar', () => {
    expect(labelPeriode(['2026-07', '2026-08', '2026-09'])).toBe('Jul 26, Agu 26, Sep 26')
  })
})
