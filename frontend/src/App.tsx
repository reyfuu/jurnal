import { useCallback, useEffect, useState } from 'react'
import { Sidebar } from '@/components/Sidebar'
import { Button } from '@/components/ui/button'
import { BEBAS_TERKUNCI, VIEW_LABEL } from '@/lib/nav'
import { FiturTerkunci } from '@/components/FiturTerkunci'
import { CatatanKlienView, PengingatCatatan, TulisCatatanTombol } from '@/components/CatatanKonsultan'
import { PengingatJadwalBanner } from '@/components/PengingatJadwal'
import { PageHeader } from '@/components/PageHeader'
import { LaporanAI } from '@/views/LaporanAI'
import { LaporBug } from '@/views/LaporBug'
import { Landing } from '@/views/Landing'
import { Langganan, statusText } from '@/views/Langganan'
import { ApiError, getBusiness, getMe, getPeriods, loadAll, logout, selesaiLihat, type AppData, type BusinessInfo, type Me, type PeriodOption, type View } from '@/lib/api'
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
import { UsulanKoreksi } from '@/views/UsulanKoreksi'
import { Pustaka } from '@/views/Pustaka'

type Session = { status: 'checking' } | { status: 'anon' } | { status: 'in'; me: Me }

export default function App() {
  const [session, setSession] = useState<Session>({ status: 'checking' })
  const [data, setData] = useState<AppData | null>(null)
  const [view, setView] = useState<View>('dashboard')
  const [bugFrom, setBugFrom] = useState<View>('dashboard') // halaman asal saat membuka Lapor Bug
  const [bukaTab, setBukaTab] = useState<{ tab?: string; n: number }>({ n: 0 })
  const [auth, setAuth] = useState<'masuk' | 'daftar' | null>(null) // null = landing page
  const [period, setPeriod] = useState<string>('2026-08')
  const [periods, setPeriods] = useState<PeriodOption[]>([])
  const [error, setError] = useState('')
  const [business, setBusiness] = useState<BusinessInfo | null>(null)
  const [layarBayar, setLayarBayar] = useState<'langganan' | 'alat'>('langganan') // usaha belum berlangganan
  // Belum berlangganan tapi memilih "Masuk ke aplikasi": workspace dengan menu bergembok.
  const [masukTerkunci, setMasukTerkunci] = useState(() => { try { return sessionStorage.getItem('masuk-terkunci') === '1' } catch { return false } })
  const ingatMasuk = (v: boolean) => { setMasukTerkunci(v); try { sessionStorage.setItem('masuk-terkunci', v ? '1' : '') } catch { /* mode privat */ } }

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
    if (m.views.includes('panel-master')) return // akun tim Master Pajak: tanpa data usaha (kecuali lihat-sebagai)
    setView(m.langganan.status === 'habis' ? 'laporan-ai' : m.views[0]) // belum berlangganan: menu pertama yang terbuka
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
    ingatMasuk(false)
  }

  // Masuk/keluar lihat-sebagai: muat ulang me; data usaha dibuang supaya tidak tertukar.
  const gantiMode = () => { setBusiness(null); setData(null); getMe().then(signIn, fail) }
  if (me.views.includes('panel-master')) return <PanelMaster me={me} onLogout={onLogout} onLihat={gantiMode} />
  if (!business) return error ? <p className="workspace-subtitle" role="alert" style={{ padding: 24 }}>{error}</p> : null

  const canSee = (v: View) => me.views.includes(v)
  const terkunci = me.langganan.status === 'habis'
  const navigate = (v: View, tab?: string) => {
    if (!canSee(v) && v !== 'catatan' && !(terkunci && (v === 'laporan-ai' || v === 'alat-gratis'))) return
    if (v === 'lapor-bug' && view !== 'lapor-bug') setBugFrom(view)
    setBukaTab({ tab, n: Date.now() }) // tab awal halaman tujuan, mis. dari tautan Tanya AI
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

  if (!business.profile && me.lihat) { // lihat-sebagai hanya baca: klien yang belum onboarding tidak punya laporan
    return (
      <OnboardingShell onLogout={onLogout}>
        <div className="grid gap-3">
          <h1 className="text-xl font-bold">{me.lihat.nama} belum mengisi profil usaha</h1>
          <p className="text-sm text-muted-foreground">Belum ada laporan yang bisa dilihat. Mode lihat sebagai klien hanya baca, jadi profil tidak bisa diisi dari sini.</p>
          <Button className="justify-self-start rounded-full" onClick={() => void selesaiLihat().finally(gantiMode)}>Kembali ke Panel Master</Button>
        </div>
      </OnboardingShell>
    )
  }
  if (!business.profile) {
    return <Onboarding canEdit={canSee('profil-usaha')} onSaved={onProfileSaved} onLogout={onLogout} provinsi={business.options.provinsi} klu={business.options.klu} />
  }

  const onPaid = () => getMe().then((m) => { setSession({ status: 'in', me: m }); void reload(m) }, fail)

  // Bayar dulu: setelah onboarding, usaha tanpa langganan aktif hanya melihat layar Langganan.
  if (terkunci && !masukTerkunci) {
    return (
      <OnboardingShell onLogout={onLogout} onMasuk={() => { ingatMasuk(true); setView('laporan-ai') }}>
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
        <Sidebar me={me} tenant={data?.dash.tenant} view={view} onNavigate={navigate} onLogout={onLogout} terkunci={terkunci} />
        <section className="workspace">
          {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start' }}>{error}</div>}
          {terkunci && view !== 'langganan' && (
            <div className="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-lg border px-4 py-3 text-sm" role="status"
              style={{ borderColor: 'color-mix(in srgb, var(--accent-orange) 30%, transparent)', background: 'var(--accent-orange-soft)' }}>
              <span><strong>Anda belum berlangganan.</strong> Yang terbuka: Laporan AI (dari data pendaftaran) dan Alat Gratis. Fitur lain terbuka setelah memilih paket.</span>
              {canSee('langganan') && <Button size="sm" variant="accent" onClick={() => navigate('langganan')}>Pilih paket</Button>}
            </div>
          )}
          {me.lihat && (
            <div className="mb-4 flex flex-wrap items-center gap-3 rounded-lg border px-4 py-3 text-sm" role="status"
              style={{ borderColor: 'var(--accent-amber)', background: 'repeating-linear-gradient(135deg, color-mix(in srgb, var(--accent-amber) 12%, transparent) 0 12px, transparent 12px 24px)' }}>
              <strong style={{ color: 'var(--accent-amber)' }}>Lihat sebagai klien · baca saja</strong>
              <span>{me.lihat.nama} · alasan “{me.lihat.alasan}” · tercatat di audit log · berakhir {new Date(me.lihat.sampai).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })}</span>
              <Button size="sm" variant="outline" className="rounded-full" style={{ marginLeft: 'auto' }} onClick={() => void selesaiLihat().finally(gantiMode)}>Kembali ke Panel Master</Button>
            </div>
          )}
          {!me.lihat && !!me.jadwal?.length && <PengingatJadwalBanner jadwal={me.jadwal} />}
          {!!me.catatan_terbuka && !me.lihat && view !== 'catatan' && (
            <div className="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-lg border px-4 py-3 text-sm" style={{ borderColor: 'var(--accent-amber)' }} role="status">
              <span><strong>{me.catatan_terbuka} catatan dari konsultan Master Pajak</strong> belum ditandai sudah diperbaiki.</span>
              <Button size="sm" className="rounded-full font-bold" onClick={() => navigate('catatan')}>Lihat catatan</Button>
            </div>
          )}
          {!!me.koreksi_menunggu && view !== 'usulan-koreksi' && (
            <div className="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-lg border px-4 py-3 text-sm" style={{ borderColor: 'var(--accent-amber)' }} role="status">
              <span><strong>{me.koreksi_menunggu} usulan koreksi</strong> dari konsultan Master Pajak menunggu keputusan Anda. Pembukuan belum berubah.</span>
              <Button size="sm" className="rounded-full font-bold" onClick={() => navigate('usulan-koreksi')}>Tinjau</Button>
            </div>
          )}
          {me.langganan.status === 'trial' && view !== 'langganan' && ( // usaha lama yang masih masa coba
            <div className="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-lg border px-4 py-3 text-sm" style={{ borderColor: 'var(--border-shell)' }} role="status">
              <span>{statusText(me.langganan)}</span>
              {canSee('langganan') && <Button size="sm" variant="accent" onClick={() => navigate('langganan')}>Aktifkan langganan</Button>}
            </div>
          )}
          {canSee('profil-usaha') && !me.lihat && view !== 'profil-usaha' && (!business.profile.kota || !business.profile.omzet_tahunan || !business.profile.klu) && (
            <div className="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-lg border px-4 py-3 text-sm" style={{ borderColor: 'color-mix(in srgb, var(--accent-orange) 30%, transparent)', background: 'var(--accent-orange-soft)' }} role="status">
              <span><strong>Lengkapi profil usaha.</strong> Isi bidang usaha (KLU), lokasi, serta perkiraan omzet &amp; laba per tahun supaya NPM dan CTTOR bisa dibandingkan dengan rata-rata nasional.</span>
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
      {!me.lihat && !terkunci && <AIAgentBubble period={period} onBuka={navigate} />}
      {me.lihat && me.master?.izin.includes('catatan') && <TulisCatatanTombol klienId={me.lihat.klien_id} halaman={view} periode={period} />}
      {!me.lihat && !!me.catatan_ingat && <PengingatCatatan onLihat={() => navigate('catatan')} />}
    </main>
  )

  function renderView() {
    const d = data!.dash
    if (terkunci && !BEBAS_TERKUNCI.includes(view)) {
      return <FiturTerkunci nama={VIEW_LABEL[view] ?? 'Fitur ini'} onLangganan={canSee('langganan') ? () => navigate('langganan') : undefined} />
    }
    switch (view) {
      case 'catatan':
        return <CatatanKlienView bisaTindak={me!.user.role === 'owner' || me!.user.role === 'admin'} onBuka={navigate}
          onBerubah={() => getMe().then((m) => setSession({ status: 'in', me: m }), fail)} />
      case 'laporan-ai':
        return (
          <>
            <PageHeader title="Laporan AI" subtitle="Analisis usaha Anda oleh AI. Sebelum ada transaksi, analisis memakai data pendaftaran." />
            <LaporanAI key={period} period={period} />
          </>
        )
      case 'alat-gratis':
        return <AlatGratis profile={business!.profile} ptkpCodes={business!.options.ptkp} onLangganan={() => navigate('langganan')} />
      case 'dashboard':
        return <Dashboard d={d} canSee={canSee} onNavigate={navigate} period={period} periods={periods} onPeriodChange={changePeriod} />
      case 'laporan-keuangan':
        return data!.report && data!.balanceSheet && data!.cash && data!.assets && (
          <LaporanKeuangan
            key={bukaTab.n}
            tabAwal={bukaTab.tab}
            r={data!.report}
            bs={data!.balanceSheet}
            cf={data!.cash}
            assets={data!.assets}
            period={period}
            periodLabel={d.period_label}
            periods={periods}
            bisaEkspor={!me!.lihat}
            onAssetsChanged={() => { void reload(me!, period); getBusiness().then(setBusiness, fail) }} // metode penyusutan bisa dipilih dari tab Aset
          />
        )
      case 'laporan-penjualan':
        return <LaporanPenjualan period={period} periodLabel={d.period_label} />
      case 'transaksi':
        return <Transaksi onChanged={() => reload(me!, period)}
          lihat={me!.lihat && { klienId: me!.lihat.klien_id, bolehUsul: !!me!.master?.izin.includes('usul_koreksi') }} />
      case 'piutang-utang':
        return <PiutangUtang />
      case 'stok-gudang':
        return <StokGudang bisaUbah={!me!.lihat} />
      case 'produksi-supplier':
        return <ProduksiSupplier />
      case 'invoice':
        return <InvoiceView />
      case 'simulasi-harga':
        return <SimulasiHarga period={period} />
      case 'pajak-umkm':
        return <PajakUMKM d={d} />
      case 'kalkulator-pajak':
        return <KalkulatorPajak profile={business?.profile ?? null} />
      case 'perencanaan-pajak':
        return <PerencanaanPajak profile={business!.profile} ptkpCodes={business!.options.ptkp} />
      case 'profil-usaha':
        return <ProfilUsaha info={business!} onSaved={onProfileSaved} />
      case 'master-data':
        return <DaftarAkun bisaPerbarui={canSee('profil-usaha') && !me!.lihat} />
      case 'impor-data':
        return <ImporDataAwal onChanged={() => reload(me!, period)} />
      case 'karyawan':
        return <KaryawanView />
      case 'langganan':
        return <Langganan onPaid={onPaid} />
      case 'pustaka':
        return <Pustaka />
      case 'usulan-koreksi':
        return <UsulanKoreksi onChanged={() => { void reload(me!, period); getMe().then((m) => setSession({ status: 'in', me: m }), fail) }} />
      case 'lapor-bug':
        return <LaporBug halamanAwal={VIEW_LABEL[bugFrom]} />
      case 'pengguna':
        return <Pengguna currentUserId={me!.user.id} />
      default:
        return <StokModule d={d} title={VIEW_LABEL[view].toUpperCase()} />
    }
  }
}
