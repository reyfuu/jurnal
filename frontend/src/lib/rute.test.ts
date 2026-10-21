import { describe, expect, it } from 'vitest'
import { bacaRute, tulisRute } from './rute'

describe('rute di hash URL', () => {
  it('menulis halaman dan periode', () => {
    expect(tulisRute('buku-besar', '2026-08')).toBe('#/buku-besar?periode=2026-08')
    expect(tulisRute('dashboard', '')).toBe('#/dashboard')
  })
  it('membaca balik hasil tulisRute', () => {
    expect(bacaRute('#/buku-besar?periode=2026-08')).toEqual({ view: 'buku-besar', periode: '2026-08' })
    expect(bacaRute('#/transaksi')).toEqual({ view: 'transaksi', periode: '' })
  })
  it('mengabaikan jangkar biasa dan isian rusak', () => {
    expect(bacaRute('#harga')).toEqual({ view: '', periode: '' })
    expect(bacaRute('')).toEqual({ view: '', periode: '' })
    expect(bacaRute('#/<script>?periode=2026-13x')).toEqual({ view: '', periode: '' })
  })
})
