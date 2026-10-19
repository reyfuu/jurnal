import { useCallback, useEffect, useState } from 'react'
import { Sidebar } from '@/components/Sidebar'
import { VIEW_LABEL } from '@/lib/nav'
import { ApiError, getMe, getPeriods, loadAll, logout, type AppData, type Me, type PeriodOption, type View } from '@/lib/api'
import { Dashboard } from '@/views/Dashboard'
import { Login } from '@/views/Login'
import { ArusKas, LaporanKeuangan } from '@/views/Reports'
import { PajakUMKM, PiutangUtang, SimulasiHarga, StokModule } from '@/views/Operations'
import { Transaksi } from '@/views/Transaksi'
import { Pengguna } from '@/views/Pengguna'
import { AIAgentBubble } from '@/components/AIAgentBubble'

type Session = { status: 'checking' } | { status: 'anon' } | { status: 'in'; me: Me }

export default function App() {
  const [session, setSession] = useState<Session>({ status: 'checking' })
  const [data, setData] = useState<AppData | null>(null)
  const [view, setView] = useState<View>('dashboard')
  const [period, setPeriod] = useState<string>('2026-08')
  const [periods, setPeriods] = useState<PeriodOption[]>([])
  const [error, setError] = useState('')

  const me = session.status === 'in' ? session.me : null

  const fail = useCallback((e: unknown) => {
    if (e instanceof ApiError && e.status === 401) {
      setSession({ status: 'anon' })
      setData(null)
    } else {
      setError((e as Error).message)
    }
  }, [])

  const reload = useCallback(async (m: Me, p = period) => {
    try {
      setData(await loadAll(m.views, p))
      setError('')
    } catch (e) {
      fail(e)
    }
  }, [fail, period])

  const signIn = useCallback((m: Me) => {
    setSession({ status: 'in', me: m })
    setView(m.views[0])
    void reload(m)
  }, [reload])

  useEffect(() => {
    getMe().then(signIn, () => setSession({ status: 'anon' }))
    getPeriods().then(setPeriods).catch(() => {})
  }, [signIn])

  if (session.status === 'checking') return null
  if (!me) return <Login onLogin={signIn} />

  const canSee = (v: View) => me.views.includes(v)
  const navigate = (v: View) => canSee(v) && setView(v)

  const changePeriod = (newP: string) => {
    setPeriod(newP)
    void reload(me, newP)
  }

  async function onLogout() {
    await logout().catch(() => {})
    setSession({ status: 'anon' })
    setData(null)
  }

  return (
    <main className="frame-wrapper">
      <div className="app-shell">
        <Sidebar me={me} tenant={data?.dash.tenant} view={view} onNavigate={navigate} onLogout={onLogout} />
        <section className="workspace">
          {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start' }}>{error}</div>}
          {data && (
            <div className="period-toolbar" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 18, background: '#101318', padding: '10px 16px', borderRadius: 10, border: '1px solid #1c222c' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                  Periode Laporan:
                </span>
                <span className="pill-status pill-amber" style={{ fontSize: 12 }}>
                  {data.dash.period_label}
                </span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <label htmlFor="global-period-select" style={{ fontSize: 12, color: 'var(--text-muted)' }}>Ganti Periode:</label>
                <select
                  id="global-period-select"
                  className="period-select tabular"
                  style={{ background: '#0d0f13', border: '1px solid #262c38', color: '#fff', borderRadius: 8, padding: '5px 12px', fontSize: 12, cursor: 'pointer' }}
                  value={period}
                  onChange={(e) => changePeriod(e.target.value)}
                >
                  {periods.map((p) => (
                    <option key={p.id} value={p.id}>{p.label}</option>
                  ))}
                </select>
              </div>
            </div>
          )}
          {data ? renderView() : !error && <p className="workspace-subtitle">Memuat data…</p>}
        </section>
      </div>
      <AIAgentBubble period={period} />
    </main>
  )

  function renderView() {
    const d = data!.dash
    switch (view) {
      case 'dashboard':
        return <Dashboard d={d} canSee={canSee} onNavigate={navigate} period={period} periods={periods} onPeriodChange={changePeriod} />
      case 'laporan-keuangan':
        return data!.report && data!.balanceSheet && <LaporanKeuangan r={data!.report} bs={data!.balanceSheet} periodLabel={d.period_label} />
      case 'arus-kas':
        return data!.report && data!.cash && <ArusKas c={data!.cash} netProfit={data!.report.net_profit} />
      case 'transaksi':
        return <Transaksi onChanged={() => reload(me!, period)} />
      case 'piutang-utang':
      case 'invoice':
        return <PiutangUtang d={d} />
      case 'simulasi-harga':
        return <SimulasiHarga period={period} />
      case 'pajak-umkm':
        return <PajakUMKM d={d} />
      case 'pengguna':
        return <Pengguna currentUserId={me!.user.id} />
      default:
        return <StokModule d={d} title={VIEW_LABEL[view].toUpperCase()} />
    }
  }
}
