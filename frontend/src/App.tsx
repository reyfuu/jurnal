import { useCallback, useEffect, useState } from 'react'
import { Sidebar } from '@/components/Sidebar'
import { Button } from '@/components/ui/button'
import { VIEW_LABEL } from '@/lib/nav'
import { LaporBug } from '@/views/LaporBug'
import { Landing } from '@/views/Landing'
import { Langganan, statusText } from '@/views/Langganan'
import { ApiError, getBusiness, getMe, getPeriods, loadAll, logout, type AppData, type BusinessInfo, type Me, type PeriodOption, type View } from '@/lib/api'
import { DaftarAkun } from '@/views/DaftarAkun'
import { Dashboard } from '@/views/Dashboard'
import { Login } from '@/views/Login'
import { LaporanKeuangan } from '@/views/Reports'
import { LaporanPenjualan } from '@/views/LaporanPenjualan'
import { InvoiceView } from '@/views/Invoice'
import { KalkulatorPajak } from '@/views/KalkulatorPajak'
import { PajakUMKM, StokModule } from '@/views/Operations'
import { SimulasiHarga } from '@/views/SimulasiHarga'
import { PiutangUtang, StokGudang } from '@/views/PiutangStok'
import { ProduksiSupplier } from '@/views/ProduksiSupplier'
import { Transaksi } from '@/views/Transaksi'
import { Pengguna } from '@/views/Pengguna'
import { PerencanaanPajak } from '@/views/PerencanaanPajak'
import { AlatGratis } from '@/views/AlatGratis'
import { Onboarding, OnboardingShell, ProfilUsaha } from '@/views/ProfilUsaha'
import { ImporDataAwal } from '@/views/ImporDataAwal'
import { KaryawanView } from '@/views/Karyawan'
import { AIAgentBubble } from '@/components/AIAgentBubble'
import { PanelMaster } from '@/views/master/PanelMaster'

type Session = { status: 'checking' } | { status: 'anon' } | { status: 'in'; me: Me }

export default function App() {
  const [session, setSession] = useState<Session>({ status: 'checking' })
  const [data, setData] = useState<AppData | null>(null)
  const [view, setView] = useState<View>('dashboard')
  const [bugFrom, setBugFrom] = useState<View>('dashboard') // halaman asal saat membuka Lapor Bug
  const [auth, setAuth] = useState<'masuk' | 'daftar' | null>(null) // null = landing page
  const [period, setPeriod] = useState<string>('2026-08')
  const [periods, setPeriods] = useState<PeriodOption[]>([])
  const [error, setError] = useState('')
  const [business, setBusiness] = useState<BusinessInfo | null>(null)
  const [layarBayar, setLayarBayar] = useState<'langganan' | 'alat'>('langganan') // usaha belum berlangganan

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
    if (m.views.includes('panel-master')) return // akun tim Master Pajak: tanpa data usaha
    setView(m.views[0])
    getBusiness().then(setBusiness, fail)
    getPeriods().then(setPeriods).catch(() => {}) // butuh sesi: jangan dipanggil sebelum login
    void reload(m)
  }, [reload, fail])

  useEffect(() => {
    getMe().then(signIn, () => setSession({ status: 'anon' }))
  }, [signIn])

  if (session.status === 'checking') return null
  if (!me) {
    return auth
      ? <Login key={auth} onLogin={signIn} initialDaftar={auth === 'daftar'} onBack={() => setAuth(null)} />
      : <Landing onMasuk={() => setAuth('masuk')} onDaftar={() => setAuth('daftar')} />
  }
  async function onLogout() {
    await logout().catch(() => {})
    setSession({ status: 'anon' })
    setData(null)
    setBusiness(null)
    setAuth(null)
  }

  if (me.views.includes('panel-master')) return <PanelMaster me={me} onLogout={onLogout} />
  if (!business) return error ? <p className="workspace-subtitle" role="alert" style={{ padding: 24 }}>{error}</p> : null

  const canSee = (v: View) => me.views.includes(v)
  const navigate = (v: View) => {
    if (!canSee(v)) return
    if (v === 'lapor-bug' && view !== 'lapor-bug') setBugFrom(view)
    setView(v)
  }

  const changePeriod = (newP: string) => {
    setPeriod(newP)
    void reload(me, newP)
  }

  const onProfileSaved = (p: NonNullable<BusinessInfo['profile']>) => {
    setBusiness({ ...business!, profile: p })
    void reload(me!, period) // nama & kategori tenant ikut berubah
  }

  if (!business.profile) {
    return <Onboarding canEdit={canSee('profil-usaha')} onSaved={onProfileSaved} onLogout={onLogout} provinsi={business.options.provinsi} klu={business.options.klu} />
  }

  const onPaid = () => getMe().then((m) => { setSession({ status: 'in', me: m }); void reload(m) }, fail)

  // Bayar dulu: setelah onboarding, usaha tanpa langganan aktif hanya melihat layar Langganan.
  if (me.langganan.status === 'habis') {
    return (
      <OnboardingShell onLogout={onLogout}>
        {canSee('kalkulator-pajak') && (
          <div className="tabbar" role="tablist" aria-label="Langganan atau alat gratis" style={{ marginBottom: 20 }}>
            <button role="tab" className="tab-btn" aria-selected={layarBayar === 'langganan'} onClick={() => setLayarBayar('langganan')}>Langganan</button>
            <button role="tab" className="tab-btn" aria-selected={layarBayar === 'alat'} onClick={() => setLayarBayar('alat')}>Alat Gratis</button>
          </div>
        )}
        {layarBayar === 'alat' && canSee('kalkulator-pajak') ? (
          <AlatGratis profile={business.profile} ptkpCodes={business.options.ptkp} onLangganan={() => setLayarBayar('langganan')} />
        ) : canSee('langganan') ? (
          <Langganan onPaid={onPaid} />
        ) : (
          <p className="text-sm">Langganan usaha belum aktif. Hubungi Owner Anda untuk memilih paket, lalu muat ulang halaman ini.</p>
        )}
      </OnboardingShell>
    )
  }

  return (
    <main>
      <div className="app-shell">
        <Sidebar me={me} tenant={data?.dash.tenant} view={view} onNavigate={navigate} onLogout={onLogout} />
        <section className="workspace">
          {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start' }}>{error}</div>}
          {me.langganan.status === 'trial' && view !== 'langganan' && ( // usaha lama yang masih masa coba
            <div className="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-lg border px-4 py-3 text-sm" style={{ borderColor: 'var(--border-shell)' }} role="status">
              <span>{statusText(me.langganan)}</span>
              {canSee('langganan') && <Button size="sm" variant="accent" onClick={() => navigate('langganan')}>Aktifkan langganan</Button>}
            </div>
          )}
          {canSee('profil-usaha') && view !== 'profil-usaha' && (!business.profile.kota || !business.profile.omzet_tahunan || !business.profile.klu) && (
            <div className="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-lg border px-4 py-3 text-sm" style={{ borderColor: 'color-mix(in srgb, var(--accent-orange) 30%, transparent)', background: 'var(--accent-orange-soft)' }} role="status">
              <span><strong>Lengkapi profil usaha.</strong> Isi bidang usaha (KLU), lokasi, serta perkiraan omzet &amp; laba per tahun supaya NPM dan CTTOR bisa dibandingkan dengan rata-rata DJP.</span>
              <Button size="sm" onClick={() => navigate('profil-usaha')}>Lengkapi sekarang</Button>
            </div>
          )}
          {data && (
            <div className="period-toolbar" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 18, background: 'var(--bg-raised)', padding: '10px 16px', borderRadius: 10, border: '1px solid var(--border-shell)' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                  Periode Laporan:
                </span>
                <span className="pill-status pill-brand" style={{ fontSize: 12 }}>
                  {data.dash.period_label}
                </span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <label htmlFor="global-period-select" style={{ fontSize: 12, color: 'var(--text-muted)' }}>Ganti Periode:</label>
                <select
                  id="global-period-select"
                  className="period-select tabular"
                  style={{ background: 'var(--bg-input)', border: '1px solid var(--border-strong)', color: 'var(--text-primary)', borderRadius: 8, padding: '5px 12px', fontSize: 12, cursor: 'pointer' }}
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
        return data!.report && data!.balanceSheet && data!.cash && data!.assets && (
          <LaporanKeuangan
            r={data!.report}
            bs={data!.balanceSheet}
            cf={data!.cash}
            assets={data!.assets}
            period={period}
            periodLabel={d.period_label}
            onAssetsChanged={() => { void reload(me!, period); getBusiness().then(setBusiness, fail) }} // metode penyusutan bisa dipilih dari tab Aset
          />
        )
      case 'laporan-penjualan':
        return <LaporanPenjualan period={period} periodLabel={d.period_label} />
      case 'transaksi':
        return <Transaksi onChanged={() => reload(me!, period)} />
      case 'piutang-utang':
        return <PiutangUtang />
      case 'stok-gudang':
        return <StokGudang />
      case 'produksi-supplier':
        return <ProduksiSupplier />
      case 'invoice':
        return <InvoiceView />
      case 'simulasi-harga':
        return <SimulasiHarga period={period} />
      case 'pajak-umkm':
        return <PajakUMKM d={d} />
      case 'kalkulator-pajak':
        return <KalkulatorPajak />
      case 'perencanaan-pajak':
        return <PerencanaanPajak profile={business!.profile} ptkpCodes={business!.options.ptkp} />
      case 'profil-usaha':
        return <ProfilUsaha info={business!} onSaved={onProfileSaved} />
      case 'master-data':
        return <DaftarAkun />
      case 'impor-data':
        return <ImporDataAwal onChanged={() => reload(me!, period)} />
      case 'karyawan':
        return <KaryawanView />
      case 'langganan':
        return <Langganan onPaid={onPaid} />
      case 'lapor-bug':
        return <LaporBug halamanAwal={VIEW_LABEL[bugFrom]} />
      case 'pengguna':
        return <Pengguna currentUserId={me!.user.id} />
      default:
        return <StokModule d={d} title={VIEW_LABEL[view].toUpperCase()} />
    }
  }
}
