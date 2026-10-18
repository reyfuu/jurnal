import { describe, expect, it } from 'vitest'
import { formatJt, formatPct, formatRp, formatSignedJt } from './format'

describe('formatRp', () => {
  it('memakai titik ribuan dan tanda minus di depan', () => {
    expect(formatRp(1_680_000)).toBe('Rp1.680.000')
    expect(formatRp(-3_450_000)).toBe('-Rp3.450.000')
    expect(formatRp(0)).toBe('Rp0')
  })
})

describe('formatJt', () => {
  it('bulat tanpa desimal, pecahan pakai koma', () => {
    expect(formatJt(24_000_000)).toBe('Rp24 jt')
    expect(formatJt(6_800_000)).toBe('Rp6,8 jt')
  })
  it('formatSignedJt memberi + atau -', () => {
    expect(formatSignedJt(16_000_000)).toBe('+Rp16 jt')
    expect(formatSignedJt(-2_500_000)).toBe('-Rp2,5 jt')
  })
})

describe('formatPct', () => {
  it('basis poin ke persen satu desimal berkoma', () => {
    expect(formatPct(2608)).toBe('26,1%')
  })
})
