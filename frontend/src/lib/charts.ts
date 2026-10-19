// Geometri SVG murni untuk grafik dashboard; komponen di components/charts.tsx yang menggambar.
import type { DailyOmzet, MonthPoint } from './api'

const f = (n: number) => n.toFixed(1)

export function sparklinePoints(values: number[], width: number, height: number) {
  if (values.length < 2) return ''
  const min = Math.min(...values)
  const range = Math.max(...values) - min || 1
  return values
    .map((v, i) => `${f((i / (values.length - 1)) * width)},${f(height - 4 - ((v - min) / range) * (height - 8))}`)
    .join(' ')
}

export const MAIN = { w: 640, h: 250, padL: 44, padR: 16, padT: 16, padB: 28, max: 100_000_000 }

export function mainChart(points: MonthPoint[]) {
  const { w, h, padL, padR, padT, padB, max } = MAIN
  const plotW = w - padL - padR
  const plotH = h - padT - padB
  const xAt = (i: number) => padL + (i / (points.length - 1)) * plotW
  const yAt = (v: number) => padT + plotH - (v / max) * plotH

  const smooth = (key: 'omzet' | 'total_biaya' | 'laba_bersih') =>
    points
      .map((p, i) => {
        const x = xAt(i)
        const y = yAt(p[key])
        if (i === 0) return `M ${f(x)} ${f(y)}`
        const px = xAt(i - 1)
        const py = yAt(points[i - 1][key])
        return `C ${f(px + (x - px) * 0.45)} ${f(py)}, ${f(px + (x - px) * 0.55)} ${f(y)}, ${f(x)} ${f(y)}`
      })
      .join(' ')

  const lastX = xAt(points.length - 1)
  const lastPt = points[points.length - 1]
  return {
    omzet: smooth('omzet'),
    biaya: smooth('total_biaya'),
    laba: smooth('laba_bersih'),
    base: `L ${f(lastX)} ${f(padT + plotH)} L ${padL} ${f(padT + plotH)} Z`,
    yGrid: [0, 25, 50, 75, 100].map((jt) => ({ y: yAt(jt * 1_000_000), label: jt === 0 ? '0' : `${jt} jt` })),
    xLabels: points.map((p, i) => ({ x: xAt(i), label: p.month })),
    last: { x: lastX, omzet: yAt(lastPt.omzet), biaya: yAt(lastPt.total_biaya), laba: yAt(lastPt.laba_bersih) },
  }
}

export const DAILY = { w: 340, h: 115, padL: 24, padR: 8, padT: 10, padB: 18, max: 6_000_000 }

export function dailyBars(days: DailyOmzet[], target: number) {
  const { w, h, padL, padR, padT, padB, max } = DAILY
  const plotW = w - padL - padR
  const plotH = h - padT - padB
  const slot = plotW / days.length
  const barW = Math.max(4, slot - 3)
  return {
    yTarget: padT + plotH - (target / max) * plotH,
    baseY: padT + plotH,
    bars: days.map((d, i) => {
      const bh = (d.omzet / max) * plotH
      return { x: padL + i * slot, y: padT + plotH - bh, w: barW, h: bh, fill: d.above_target ? '#eab308' : '#785c12' }
    }),
  }
}

export const CASH = { w: 340, h: 110, padL: 28, padR: 10, padT: 10, padB: 20, max: 110_000_000 }

export function cashChart(points: MonthPoint[]) {
  const { w, h, padL, padR, padT, padB, max } = CASH
  const plotW = w - padL - padR
  const plotH = h - padT - padB
  const xAt = (i: number) => padL + (i / (points.length - 1)) * plotW
  const yAt = (v: number) => padT + plotH - (v / max) * plotH
  const line = points.map((p, i) => `${i === 0 ? 'M' : 'L'} ${f(xAt(i))} ${f(yAt(p.ending_cash))}`).join(' ')
  const lastX = xAt(points.length - 1)
  return {
    line,
    area: `${line} L ${f(lastX)} ${f(padT + plotH)} L ${padL} ${f(padT + plotH)} Z`,
    last: { x: lastX, y: yAt(points[points.length - 1].ending_cash) },
    midY: padT + plotH / 2,
    baseY: padT + plotH,
  }
}

export const TREND = { w: 640, h: 200, padL: 8, padR: 8, padT: 12, padB: 24 }

// trendBars: batang tren penjualan dengan skala otomatis (nilai terbesar = tinggi penuh).
export function trendBars(values: number[]) {
  const { w, h, padL, padR, padT, padB } = TREND
  const plotH = h - padT - padB
  const slot = (w - padL - padR) / Math.max(1, values.length)
  const max = Math.max(1, ...values)
  return values.map((v, i) => {
    const bh = (v / max) * plotH
    return { x: padL + i * slot + slot * 0.15, y: padT + plotH - bh, w: slot * 0.7, h: bh, cx: padL + i * slot + slot / 2 }
  })
}
