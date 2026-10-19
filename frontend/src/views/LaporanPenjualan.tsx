import { useEffect, useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { getSalesReport, type Granularity, type SalesGroup, type SalesReport } from '@/lib/api'
import { TREND, trendBars } from '@/lib/charts'
import { formatJt, formatRp } from '@/lib/format'

const GRANULARITY: { value: Granularity; label: string }[] = [
  { value: 'day', label: 'Harian' },
  { value: 'week', label: 'Mingguan' },
  { value: 'month', label: 'Bulanan (semua periode)' },
]

function TrendChart({ points }: { points: SalesReport['trend'] }) {
  const bars = trendBars(points.map((p) => p.omzet))
  const step = Math.ceil(points.length / 10) // label secukupnya agar tidak bertumpuk
  return (
    <svg viewBox={`0 0 ${TREND.w} ${TREND.h}`} style={{ width: '100%', height: 210, display: 'block', marginTop: 8 }} role="img" aria-label="Tren omzet penjualan">
      {bars.map((b, i) => (
        <g key={points[i].label}>
          <rect x={b.x} y={b.y} width={b.w} height={b.h} rx={2} fill="#eab308">
            <title>{`${points[i].label}: ${formatRp(points[i].omzet)} (${points[i].count} transaksi)`}</title>
          </rect>
          {i % step === 0 && (
            <text x={b.cx} y={TREND.h - 6} textAnchor="middle" fill="#616875" fontSize="10">{points[i].label.slice(-5)}</text>
          )}
        </g>
      ))}
    </svg>
  )
}

// BarList: daftar peringkat dengan batang proporsional terhadap baris teratas.
function BarList({ title, rows }: { title: string; rows: { name: string; value: number; meta: string }[] }) {
  const max = Math.max(1, ...rows.map((r) => r.value))
  return (
    <div className="card">
      <div className="chart-title">{title}</div>
      {rows.length === 0 && <p className="workspace-subtitle">Belum ada penjualan.</p>}
      <div style={{ display: 'grid', gap: 10, marginTop: 10 }}>
        {rows.map((r) => (
          <div key={r.name}>
            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, gap: 8 }}>
              <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{r.name}</span>
              <span className="tabular" style={{ flexShrink: 0 }}>{formatRp(r.value)}</span>
            </div>
            <div style={{ height: 6, borderRadius: 3, background: 'rgba(255,255,255,0.05)', marginTop: 4 }}>
              <div style={{ height: 6, borderRadius: 3, background: '#eab308', width: `${(r.value / max) * 100}%` }} />
            </div>
            <div className="kpi-meta-row" style={{ marginBottom: 0, marginTop: 3 }}>{r.meta}</div>
          </div>
        ))}
      </div>
    </div>
  )
}

const groupRows = (g: SalesGroup[]) =>
  g.map((x) => ({ name: x.name, value: x.omzet, meta: `${x.count} transaksi${x.piutang ? ` · belum lunas ${formatRp(x.piutang)}` : ''}` }))

export function LaporanPenjualan({ period, periodLabel }: { period: string; periodLabel: string }) {
  const [granularity, setGranularity] = useState<Granularity>('day')
  const [r, setR] = useState<SalesReport | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    setError('')
    getSalesReport(period, granularity).then(setR, (e: Error) => setError(e.message))
  }, [period, granularity])

  const cards = r ? [
    { label: 'Total Omzet Penjualan', value: formatJt(r.summary.omzet) },
    { label: 'Jumlah Transaksi', value: r.summary.transaksi.toLocaleString('id-ID') },
    { label: 'Rata-rata per Transaksi', value: formatRp(r.summary.rata_rata) },
    { label: 'Penjualan Belum Lunas', value: formatJt(r.summary.piutang), color: r.summary.piutang > 0 ? '#facc15' : undefined },
  ] : []

  return (
    <>
      <PageHeader title="Laporan Penjualan" subtitle={granularity === 'month' ? 'Tren seluruh periode tersedia' : `${periodLabel} · omzet, produk terlaris, channel & pelanggan`}>
        <Select value={granularity} onValueChange={(v) => setGranularity(v as Granularity)}>
          <SelectTrigger className="tx-filter" aria-label="Kelompokkan tren"><SelectValue /></SelectTrigger>
          <SelectContent>
            {GRANULARITY.map((g) => <SelectItem key={g.value} value={g.value}>{g.label}</SelectItem>)}
          </SelectContent>
        </Select>
      </PageHeader>

      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start' }}>{error}</div>}
      {r && (
        <>
          <div className="kpi-grid" style={{ gridTemplateColumns: 'repeat(4, minmax(0, 1fr))' }}>
            {cards.map((c) => (
              <div className="card" key={c.label}>
                <div className="kpi-label">{c.label}</div>
                <div className="kpi-value tabular" style={{ color: c.color }}>{c.value}</div>
              </div>
            ))}
          </div>

          <div className="card">
            <div className="chart-title">Tren Omzet {GRANULARITY.find((g) => g.value === granularity)!.label}</div>
            {r.trend.length ? <TrendChart points={r.trend} /> : <p className="workspace-subtitle">Belum ada penjualan.</p>}
          </div>

          <div className="bottom-grid grid-3">
            <BarList title="10 Produk Terlaris" rows={r.top_items.map((i) => ({ name: i.item, value: i.omzet, meta: `${i.qty.toLocaleString('id-ID')} pcs` }))} />
            <BarList title="Penjualan per Channel" rows={groupRows(r.by_channel)} />
            <BarList title="10 Pelanggan Teratas" rows={groupRows(r.by_customer)} />
          </div>
        </>
      )}
    </>
  )
}
