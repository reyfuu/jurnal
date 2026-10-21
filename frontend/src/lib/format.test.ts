import { describe, expect, it } from 'vitest'
import { formatPct, formatRelatif, formatRp, formatTanggal } from './format'

describe('formatRp', () => {
  it('memakai titik ribuan dan tanda minus di depan', () => {
    expect(formatRp(1_680_000)).toBe('Rp1.680.000')
    expect(formatRp(-3_450_000)).toBe('-Rp3.450.000')
    expect(formatRp(0)).toBe('Rp0')
  })
})

describe('formatPct', () => {
  it('basis poin ke persen satu desimal berkoma', () => {
    expect(formatPct(2608)).toBe('26,1%')
  })
})

describe('formatRelatif', () => {
  const now = new Date('2026-10-20T09:00:00Z')
  it('menyebut selisih waktu dengan bahasa sehari-hari', () => {
    expect(formatRelatif(undefined, now)).toBe('belum pernah')
    expect(formatRelatif('2026-10-20T08:59:30Z', now)).toBe('baru saja')
    expect(formatRelatif('2026-10-20T08:35:00Z', now)).toBe('25 menit lalu')
    expect(formatRelatif('2026-10-20T06:00:00Z', now)).toBe('3 jam lalu')
    expect(formatRelatif('2026-10-19T07:00:00Z', now)).toBe('kemarin')
    expect(formatRelatif('2026-10-04T09:00:00Z', now)).toBe('16 hari lalu')
  })
})

describe('formatTanggal', () => {
  it('tanggal singkat Indonesia; kosong jadi tanda strip', () => {
    expect(formatTanggal('2026-11-14T00:00:00Z')).toBe('14 Nov 2026')
    expect(formatTanggal(undefined)).toBe('—')
  })
})
