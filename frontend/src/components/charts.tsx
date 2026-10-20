import { CASH, DAILY, MAIN, cashChart, dailyBars, mainChart, sparklinePoints } from '@/lib/charts'
import type { DailyOmzet, MonthPoint } from '@/lib/api'

const AXIS = 'var(--text-muted)'

export function Sparkline({ values, color }: { values: number[]; color: string }) {
  return (
    <svg className="sparkline-svg" viewBox="0 0 140 28">
      <polyline
        fill="none"
        stroke={color}
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
        points={sparklinePoints(values, 140, 28)}
      />
    </svg>
  )
}

export function MainChart({ points }: { points: MonthPoint[] }) {
  const c = mainChart(points)
  const { w, h, padL, padR } = MAIN
  return (
    <svg viewBox={`0 0 ${w} ${h}`} style={{ width: '100%', height: 265, display: 'block', marginTop: 6 }}>
      <defs>
        <linearGradient id="gradOmzet" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="var(--accent-amber)" stopOpacity="0.28" />
          <stop offset="100%" stopColor="var(--accent-amber)" stopOpacity="0.02" />
        </linearGradient>
        <linearGradient id="gradLaba" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="var(--accent-green)" stopOpacity="0.26" />
          <stop offset="100%" stopColor="var(--accent-green)" stopOpacity="0.01" />
        </linearGradient>
      </defs>
      {c.yGrid.map((g) => (
        <g key={g.label}>
          <line x1={padL} y1={g.y} x2={w - padR} y2={g.y} stroke="color-mix(in srgb, var(--text-primary) 5%, transparent)" strokeDasharray="2 3" />
          <text x={padL - 8} y={g.y + 4} textAnchor="end" fill={AXIS} fontSize="10">{g.label}</text>
        </g>
      ))}
      <path d={`${c.omzet} ${c.base}`} fill="url(#gradOmzet)" />
      <path d={`${c.laba} ${c.base}`} fill="url(#gradLaba)" />
      <path d={c.omzet} fill="none" stroke="var(--accent-amber)" strokeWidth="2.2" />
      <path d={c.biaya} fill="none" stroke="var(--accent-orange)" strokeWidth="1.8" />
      <path d={c.laba} fill="none" stroke="var(--accent-green)" strokeWidth="2.2" />
      <circle cx={c.last.x} cy={c.last.omzet} r="3.5" fill="var(--accent-amber)" />
      <circle cx={c.last.x} cy={c.last.biaya} r="3.5" fill="var(--accent-orange)" />
      <circle cx={c.last.x} cy={c.last.laba} r="3.5" fill="var(--accent-green)" />
      {c.xLabels.map((l) => (
        <text key={l.label} x={l.x} y={h - 6} textAnchor="middle" fill={AXIS} fontSize="10">{l.label}</text>
      ))}
    </svg>
  )
}

export function DailyChart({ days, target }: { days: DailyOmzet[]; target: number }) {
  const c = dailyBars(days, target)
  const { w, h, padL, padR, padT } = DAILY
  return (
    <svg viewBox={`0 0 ${w} ${h}`} style={{ width: '100%', height: 120, display: 'block' }}>
      <text x="0" y={padT + 4} fill={AXIS} fontSize="9">6 jt</text>
      <text x="0" y={c.yTarget + 3} fill={AXIS} fontSize="9">3 jt</text>
      <text x="0" y={c.baseY} fill={AXIS} fontSize="9">0</text>
      {c.bars.map((b, i) => (
        <rect key={i} x={b.x.toFixed(1)} y={b.y.toFixed(1)} width={b.w.toFixed(1)} height={b.h.toFixed(1)} rx="1.5" fill={b.fill} />
      ))}
      <line x1={padL} y1={c.yTarget} x2={w - padR} y2={c.yTarget} stroke="var(--text-secondary)" strokeWidth="1" strokeDasharray="3 3" />
      <text x={w - padR - 2} y={c.yTarget - 4} textAnchor="end" fill="var(--text-secondary)" fontSize="8.5">Target Rp3 jt</text>
      {[5, 10, 15, 20, 25, 30].map((d, i) => (
        <text key={d} x={padL + 14 + i * 50 + (i > 0 ? 1 : 0)} y={h - 3} fill={AXIS} fontSize="9">{d}</text>
      ))}
    </svg>
  )
}

export function CashChart({ points }: { points: MonthPoint[] }) {
  const c = cashChart(points)
  const { w, h, padT } = CASH
  return (
    <svg viewBox={`0 0 ${w} ${h}`} style={{ width: '100%', height: 112, display: 'block' }}>
      <defs>
        <linearGradient id="gradCash" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="var(--accent-blue)" stopOpacity="0.32" />
          <stop offset="100%" stopColor="var(--accent-blue)" stopOpacity="0.02" />
        </linearGradient>
      </defs>
      <text x="0" y={padT + 6} fill={AXIS} fontSize="9">100 jt</text>
      <text x="0" y={c.midY} fill={AXIS} fontSize="9">50 jt</text>
      <text x="0" y={c.baseY} fill={AXIS} fontSize="9">0</text>
      <path d={c.area} fill="url(#gradCash)" />
      <path d={c.line} fill="none" stroke="var(--accent-blue)" strokeWidth="2" />
      <circle cx={c.last.x} cy={c.last.y} r="3.2" fill="var(--accent-blue)" />
    </svg>
  )
}
