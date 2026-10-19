import { describe, expect, it } from 'vitest'
import { cashChart, dailyBars, mainChart, sparklinePoints, trendBars } from './charts'

describe('sparklinePoints', () => {
  it('memetakan min ke bawah dan max ke atas dengan padding 4px', () => {
    expect(sparklinePoints([0, 10], 140, 28)).toBe('0.0,24.0 140.0,4.0')
  })
  it('kosong bila titik < 2', () => {
    expect(sparklinePoints([5], 140, 28)).toBe('')
  })
})

const trend = [
  { month: 'Jul 26', omzet: 50_000_000, total_biaya: 25_000_000, laba_bersih: 25_000_000, ending_cash: 55_000_000 },
  { month: 'Agu 26', omzet: 100_000_000, total_biaya: 0, laba_bersih: 0, ending_cash: 110_000_000 },
]

describe('mainChart', () => {
  it('kurva mulai di padL dan titik akhir di kanan', () => {
    const c = mainChart(trend)
    expect(c.omzet.startsWith('M 44.0 119.0')).toBe(true) // 16 + 206 - 0.5*206
    expect(c.last).toEqual({ x: 624, omzet: 16, biaya: 222, laba: 222 })
    expect(c.yGrid.map((g) => g.label)).toEqual(['0', '25 jt', '50 jt', '75 jt', '100 jt'])
    expect(c.xLabels).toEqual([{ x: 44, label: 'Jul 26' }, { x: 624, label: 'Agu 26' }])
  })
})

describe('dailyBars', () => {
  it('garis target di Rp3 jt dan warna redup di bawah target', () => {
    const c = dailyBars([{ day: 1, omzet: 6_000_000, above_target: true }, { day: 2, omzet: 0, above_target: false }], 3_000_000)
    expect(c.yTarget).toBeCloseTo(53.5)
    expect(c.bars[0]).toMatchObject({ y: 10, h: 87, fill: '#eab308' })
    expect(c.bars[1].fill).toBe('#785c12')
  })
})

describe('cashChart', () => {
  it('area ditutup ke garis dasar', () => {
    const c = cashChart(trend)
    expect(c.line).toBe('M 28.0 50.0 L 330.0 10.0')
    expect(c.area.endsWith('L 330.0 90.0 L 28 90.0 Z')).toBe(true)
    expect(c.last).toEqual({ x: 330, y: 10 })
  })
})

describe('trendBars', () => {
  it('nilai terbesar setinggi area plot, sisanya proporsional', () => {
    const [a, b] = trendBars([50, 100])
    expect(b.h).toBe(164)
    expect(a.h).toBe(82)
    expect(a.y + a.h).toBe(176)
  })
  it('aman untuk data kosong / nol', () => {
    expect(trendBars([])).toEqual([])
    expect(trendBars([0])[0].h).toBe(0)
  })
})
