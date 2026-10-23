import { describe, expect, it } from 'vitest'
import { bacaRute, pecahRute, tulisRute } from './rute'

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
    expect(bacaRute('#/buku-besar?periode=2026-13x')).toEqual({ view: 'buku-besar', periode: '' })
  })
  it('sub-alamat & parameter halaman tidak mengganggu halaman dan periode', () => {
    const h = tulisRute('primbon-pajak', '2026-08', { sub: 'p/a1', params: new URLSearchParams({ q: 'pph 21' }) })
    expect(h).toBe('#/primbon-pajak/p/a1?periode=2026-08&q=pph+21')
    expect(bacaRute(h)).toEqual({ view: 'primbon-pajak', periode: '2026-08' })
    const r = pecahRute(h)!
    expect([r.sub, r.params.get('q'), r.params.has('periode')]).toEqual(['p/a1', 'pph 21', false])
    expect(pecahRute('#harga')).toBeNull()
  })
})
