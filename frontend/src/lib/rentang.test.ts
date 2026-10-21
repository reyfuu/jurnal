import { describe, expect, it } from 'vitest'
import { pilihanCepat, rentangBulan } from './rentang'

describe('rentang tanggal laporan', () => {
  it('bulan penuh, termasuk Februari kabisat', () => {
    expect(rentangBulan('2026-08')).toEqual({ dari: '2026-08-01', sampai: '2026-08-31' })
    expect(rentangBulan('2028-02')).toEqual({ dari: '2028-02-01', sampai: '2028-02-29' })
  })
  it('pilihan cepat melewati pergantian tahun', () => {
    const p = Object.fromEntries(pilihanCepat('2026-01').map((x) => [x.label, x]))
    expect(p['Bulan lalu']).toMatchObject({ dari: '2025-12-01', sampai: '2025-12-31' })
    expect(p['3 bulan terakhir']).toMatchObject({ dari: '2025-11-01', sampai: '2026-01-31' })
    expect(p['Tahun ini']).toMatchObject({ dari: '2026-01-01', sampai: '2026-01-31' })
  })
})
