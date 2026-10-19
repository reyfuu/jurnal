import { CashWarningBanner } from '@/components/CashWarningBanner'
import { CalendarDays, ChevronDown } from 'lucide-react'
import { CashChart, DailyChart, MainChart, Sparkline } from '@/components/charts'
import { PageHeader } from '@/components/PageHeader'
import type { Dashboard as Data, View } from '@/lib/api'
import { formatRp } from '@/lib/format'

const Dot = ({ color }: { color: string }) => <span className="dot" style={{ background: color }} />

export function Dashboard({
  d,
  canSee,
  onNavigate,
  period,
  periods,
  onPeriodChange,
}: {
  d: Data
  canSee: (v: View) => boolean
  onNavigate: (v: View) => void
  period?: string
  periods?: { id: string; label: string }[]
  onPeriodChange?: (p: string) => void
}) {
  const k = d.kpis
  const trend = [
    { label: 'Omzet', kpi: k.omzet, color: '#eab308', badge: 'badge-up' },
    { label: 'Laba bersih', kpi: k.laba_bersih, color: '#10b981', badge: 'badge-up' },
    { label: 'Margin laba', kpi: k.margin_laba, color: '#d4a338', badge: 'pts-up' },
  ]
  const piutangLink = canSee('piutang-utang')

  return (
    <>
      <CashWarningBanner warnings={d.cash_warnings} />
      <PageHeader title="Dashboard" subtitle="Gambaran seluruh bisnis dalam satu layar">
        <div style={{ position: 'relative', display: 'inline-flex', alignItems: 'center' }}>
          <CalendarDays size={14} style={{ position: 'absolute', left: 10, pointerEvents: 'none', color: '#9ba3af' }} />
          <select
            className="period-select tabular"
            style={{ appearance: 'none', paddingLeft: 30, paddingRight: 28, cursor: 'pointer' }}
            value={period || d.period || '2026-08'}
            onChange={(e) => onPeriodChange?.(e.target.value)}
          >
            {periods && periods.length > 0 ? (
              periods.map((p) => (
                <option key={p.id} value={p.id} style={{ background: '#101318', color: '#fff' }}>
                  {p.label}
                </option>
              ))
            ) : (
              <option value={d.period || '2026-08'}>{d.period_label}</option>
            )}
          </select>
          <ChevronDown size={12} style={{ position: 'absolute', right: 10, pointerEvents: 'none', color: '#9ba3af' }} />
        </div>
      </PageHeader>

      <div className="kpi-grid">
        {trend.map((t) => (
          <div className="card" key={t.label}>
            <div>
              <div className="kpi-label">{t.label}</div>
              <div className="kpi-value tabular">{t.kpi.display}</div>
              <div className="kpi-meta-row">
                <span className={t.badge}>{t.kpi.badge_text}</span>
                <span>{t.kpi.sub_text}</span>
              </div>
            </div>
            <Sparkline values={t.kpi.sparkline} color={t.color} />
          </div>
        ))}

        <div className="card">
          <div>
            <div className="kpi-label">Saldo kas &amp; bank</div>
            <div className="kpi-value tabular">{k.saldo_kas.display}</div>
            <div className="kpi-meta-row">
              <span className="badge-up">{k.saldo_kas.badge_text}</span>
              <div style={{ width: '100%', marginTop: 2 }}>{k.saldo_kas.sub_text}</div>
            </div>
          </div>
          <Sparkline values={k.saldo_kas.sparkline} color="#38bdf8" />
        </div>

        <div className="card">
          <div>
            <div className="kpi-label">Target omzet</div>
            <div className="kpi-value tabular">{k.target_omzet.display}</div>
            <div className="kpi-meta-row">
              <span>{k.target_omzet.surplus_display}</span>
            </div>
          </div>
          <div className="progress-track">
            <div className="progress-bar" style={{ width: '100%' }} />
          </div>
        </div>

        <div
          className="card"
          style={piutangLink ? { cursor: 'pointer' } : undefined}
          onClick={piutangLink ? () => onNavigate('piutang-utang') : undefined}
        >
          <div>
            <div className="kpi-label">Piutang belum tertagih</div>
            <div className="kpi-value tabular">{k.piutang.unpaid_display}</div>
            <div style={{ fontSize: 11, color: 'var(--text-secondary)' }}>{k.piutang.invoice_count} tagihan</div>
            <div className="overdue-alert tabular">{k.piutang.overdue_30d_display}</div>
            <div className="payable-note tabular">{k.piutang.payable_unpaid_display}</div>
          </div>
        </div>
      </div>

      <div className="charts-middle">
        <div className="card" style={{ padding: '20px 22px' }}>
          <div>
            <div className="chart-title">Omzet, biaya, dan laba</div>
            <div className="chart-subtitle">11 bulan terakhir</div>
            <div className="legend-row">
              <span className="legend-item"><Dot color="#eab308" />Omzet</span>
              <span className="legend-item"><Dot color="#d97706" />Total biaya</span>
              <span className="legend-item"><Dot color="#10b981" />Laba bersih</span>
            </div>
          </div>
          <MainChart points={d.monthly_trend_11m} />
        </div>

        <div className="right-charts-stack">
          <div className="card" style={{ padding: '18px 20px' }}>
            <div>
              <div className="chart-title">Omzet harian {d.period_label}</div>
              <div className="chart-subtitle">Rata-rata Rp3 jt per hari.</div>
              <div className="legend-row" style={{ marginTop: 8, marginBottom: 6 }}>
                <span className="legend-item"><Dot color="#eab308" />Di atas target</span>
                <span className="legend-item"><Dot color="#785c12" />Di bawah target</span>
                <span className="legend-item">— Target harian</span>
              </div>
            </div>
            <DailyChart days={d.daily_omzet} target={d.daily_target} />
          </div>

          <div className="card" style={{ padding: '18px 20px' }}>
            <div>
              <div className="chart-title">Saldo kas per bulan</div>
              <div className="chart-subtitle">
                Kas bisa turun walau laba positif
                {canSee('laporan-keuangan') && (
                  <>
                    :{' '}
                    <button onClick={() => onNavigate('laporan-keuangan')} style={{ color: '#94a3b8', textDecoration: 'underline' }}>
                      cek tab Arus Kas di Laporan Keuangan.
                    </button>
                  </>
                )}
              </div>
              <div className="legend-row" style={{ marginTop: 8, marginBottom: 4 }}>
                <span className="legend-item"><Dot color="#38bdf8" />Kas akhir bulan</span>
              </div>
            </div>
            <CashChart points={d.monthly_trend_11m} />
          </div>
        </div>
      </div>

      <div className="bottom-grid">
        <div className="card" style={{ padding: '18px 20px' }}>
          <div>
            <div className="chart-title">Omzet per produk</div>
            <div className="chart-subtitle">5 terlaris dari {d.total_products} produk</div>
            <div className="list-rows">
              {d.top_products.map((p) => (
                <div className="list-row" key={p.name}>
                  <div>
                    <div style={{ fontWeight: 600 }}>{p.name}</div>
                    <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                      {p.qty.toLocaleString('id-ID')} terjual · {p.pct}% omzet
                    </div>
                  </div>
                  <div className="tabular" style={{ fontWeight: 700, color: '#facc15' }}>{formatRp(p.omzet)}</div>
                </div>
              ))}
            </div>
          </div>
        </div>

        <div className="card" style={{ padding: '18px 20px' }}>
          <div>
            <div className="chart-title">Omzet per channel</div>
            <div className="chart-subtitle" style={{ color: '#fb7185', fontWeight: 600 }}>{d.total_app_fees_label}</div>
            <div className="list-rows">
              {d.channels.map((c) => (
                <div className="list-row" key={c.name}>
                  <div>
                    <div style={{ fontWeight: 600 }}>{c.name}</div>
                    <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                      Potongan: {formatRp(c.fee)} · Bersih: {formatRp(c.net)}
                    </div>
                  </div>
                  <div className="tabular" style={{ fontWeight: 700 }}>{formatRp(c.gross)}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </>
  )
}
