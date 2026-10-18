import { useCallback, useEffect, useState } from 'react'
import { Sidebar } from '@/components/Sidebar'
import { VIEW_LABEL } from '@/lib/nav'
import { addTransaction, ApiError, getMe, loadAll, logout, type AppData, type Me, type View } from '@/lib/api'
import { Dashboard } from '@/views/Dashboard'
import { Login } from '@/views/Login'
import { ArusKas, LaporanKeuangan } from '@/views/Reports'
import { PajakUMKM, PiutangUtang, SimulasiHarga, StokModule, Transaksi } from '@/views/Operations'
import { Pengguna } from '@/views/Pengguna'

type Session = { status: 'checking' } | { status: 'anon' } | { status: 'in'; me: Me }

export default function App() {
  const [session, setSession] = useState<Session>({ status: 'checking' })
  const [data, setData] = useState<AppData | null>(null)
  const [view, setView] = useState<View>('dashboard')
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

  const reload = useCallback(async (m: Me) => {
    try {
      setData(await loadAll(m.views))
      setError('')
    } catch (e) {
      fail(e)
    }
  }, [fail])

  const signIn = useCallback((m: Me) => {
    setSession({ status: 'in', me: m })
    setView(m.views[0])
    void reload(m)
  }, [reload])

  useEffect(() => {
    getMe().then(signIn, () => setSession({ status: 'anon' }))
  }, [signIn])

  if (session.status === 'checking') return null
  if (!me) return <Login onLogin={signIn} />

  const canSee = (v: View) => me.views.includes(v)
  const navigate = (v: View) => canSee(v) && setView(v)

  async function onAddTransaction(tx: Parameters<typeof addTransaction>[0]) {
    try {
      await addTransaction(tx) // POST dulu, baru laporan dibaca ulang
      await reload(me!)
    } catch (e) {
      fail(e)
    }
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
          {data ? renderView() : !error && <p className="workspace-subtitle">Memuat data…</p>}
        </section>
      </div>
    </main>
  )

  function renderView() {
    const d = data!.dash
    switch (view) {
      case 'dashboard':
        return <Dashboard d={d} canSee={canSee} onNavigate={navigate} />
      case 'laporan-keuangan':
        return data!.report && data!.balanceSheet && <LaporanKeuangan r={data!.report} bs={data!.balanceSheet} periodLabel={d.period_label} />
      case 'arus-kas':
        return data!.report && data!.cash && <ArusKas c={data!.cash} netProfit={data!.report.net_profit} />
      case 'transaksi':
        return <Transaksi d={d} onAdd={onAddTransaction} />
      case 'piutang-utang':
      case 'invoice':
        return <PiutangUtang d={d} />
      case 'simulasi-harga':
        return <SimulasiHarga />
      case 'pajak-umkm':
        return <PajakUMKM d={d} />
      case 'pengguna':
        return <Pengguna currentUserId={me!.user.id} />
      default:
        return <StokModule d={d} title={VIEW_LABEL[view].toUpperCase()} />
    }
  }
}
