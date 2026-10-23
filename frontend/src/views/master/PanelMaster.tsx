import { useState } from 'react'
import { BellRing, BookOpen, Brain, Contact, MessageSquareText, CreditCard, FileCheck2, Gauge, LayoutGrid, LogOut, Network, Percent, ScrollText, Ticket, UserCog, Users, type LucideIcon } from 'lucide-react'
import { Logo } from '@/components/Logo'
import { ThemeToggle } from '@/components/ThemeToggle'
import { BarAtasHP, Laci } from '@/components/hp/Laci'
import { useLaci } from '@/components/hp/useLaci'
import { useKartuOtomatis } from '@/components/hp/useKartuOtomatis'
import type { Izin, Me } from '@/lib/api'
import { KlienDetailView, KlienList, Ringkasan } from './Klien'
import { TiketBoard } from './Tiket'
import { TimAkses } from './Tim'
import { AuditLog, Pembayaran } from './Lainnya'
import { KoreksiMaster } from './Koreksi'
import { BenchmarkView, TemplateCOAView } from './Katalog'
import { ArtikelView, BasisPengetahuan } from './Pengetahuan'
import { ParameterView } from './Parameter'
import { PenggunaKlienView } from './PenggunaKlien'
import { CatatanMaster } from './CatatanMaster'
import { PengingatMaster } from './PengingatMaster'

export type Page = 'ringkasan' | 'klien' | 'koreksi' | 'tiket' | 'pengetahuan' | 'artikel' | 'coa' | 'benchmark' | 'parameter' | 'pengguna' | 'catatan' | 'pengingat' | 'pembayaran' | 'tim' | 'audit'

// Menu Panel Master; izin kosong = cukup melihat klien (klien_lihat atau klien_semua).
const NAV: { group: string; items: { page: Page; label: string; icon: LucideIcon; izin?: Izin | Izin[] }[] }[] = [
  {
    group: 'Klien',
    items: [
      { page: 'ringkasan', label: 'Ringkasan', icon: LayoutGrid },
      { page: 'klien', label: 'Klien', icon: Users },
      { page: 'pengguna', label: 'Pengguna Klien', icon: Contact, izin: 'pengguna_lihat' },
      { page: 'koreksi', label: 'Usulan Koreksi', icon: FileCheck2, izin: ['usul_koreksi', 'klien_semua'] },
      { page: 'catatan', label: 'Catatan Konsultan', icon: MessageSquareText, izin: ['catatan', 'pengaturan', 'klien_semua'] },
      { page: 'pengingat', label: 'Pengingat', icon: BellRing, izin: ['pengaturan', 'catatan', 'klien_semua'] },
      { page: 'tiket', label: 'Tiket · Lapor Bug', icon: Ticket, izin: 'tiket' },
    ],
  },
  {
    group: 'Pengetahuan',
    items: [
      { page: 'pengetahuan', label: 'Basis Pengetahuan AI', icon: Brain, izin: 'pengetahuan' },
      { page: 'artikel', label: 'Artikel Pajak', icon: BookOpen, izin: ['artikel', 'artikel_terbit'] },
      { page: 'coa', label: 'Template COA', icon: Network, izin: 'coa_template' },
      { page: 'benchmark', label: 'Benchmark NPM Nasional', icon: Gauge, izin: 'benchmark' },
      { page: 'parameter', label: 'Parameter Pajak', icon: Percent, izin: 'parameter_pajak' },
    ],
  },
  {
    group: 'Admin',
    items: [
      { page: 'pembayaran', label: 'Pembayaran', icon: CreditCard, izin: 'pembayaran' },
      { page: 'tim', label: 'Tim & Akses', icon: UserCog, izin: 'tim' },
      { page: 'audit', label: 'Audit Log', icon: ScrollText, izin: 'audit' },
    ],
  },
]

/** Seluruh aplikasi untuk akun tim Master Pajak: hanya metadata klien, tanpa angka keuangan. */
export function PanelMaster({ me, onLogout, onLihat }: { me: Me; onLogout: () => void; onLihat: () => void }) {
  const izin = me.master?.izin ?? []
  const bisa = (i?: Izin | Izin[]) => (i ? [i].flat().some((x) => izin.includes(x)) : izin.includes('klien_lihat') || izin.includes('klien_semua'))
  const [page, setPage] = useState<Page>('ringkasan')
  const [klienId, setKlienId] = useState<number | null>(null)
  const laci = useLaci() // menu ☰ di HP
  const [ruangKerja, setRuangKerja] = useState<HTMLElement | null>(null)
  useKartuOtomatis(ruangKerja) // tabel jadi kartu di HP
  const go = (p: Page) => { setPage(p); setKlienId(null); laci.setBuka(false) }
  const buka = (id: number) => { setPage('klien'); setKlienId(id) }
  const initials = me.user.name.split(/\s+/).map((w) => w[0]).slice(0, 2).join('').toUpperCase()

  if (!me.master) {
    return (
      <main className="onboarding-page" style={{ padding: 32 }}>
        <p role="alert">Akun ini belum terdaftar di tim MasterPajak AI. Minta Master menambahkan Anda di Tim &amp; Akses.</p>
        <button className="btn-cta" style={{ marginTop: 16 }} onClick={onLogout}>Keluar</button>
      </main>
    )
  }

  return (
    <main>
      <BarAtasHP onMenu={() => laci.setBuka(true)}>
        <span className="label-caps" style={{ fontSize: 10.5, color: 'var(--text-muted)' }}>Panel Master</span>
      </BarAtasHP>
      <div className="app-shell">
        <Laci buka={laci.buka} onTutup={() => laci.setBuka(false)}>
        <aside className="sidebar dark">
          <div className="sidebar-main">
            <div className="sidebar-brand">
              <div>
                <Logo tinggi={24} />
                <div className="label-caps" style={{ marginTop: 8, fontSize: 10.5, color: 'var(--text-muted)' }}>Panel Master</div>
              </div>
            </div>
            <nav className="sidebar-nav">
              {NAV.map((g) => {
                const items = g.items.filter((i) => bisa(i.izin))
                if (items.length === 0) return null
                return (
                  <div className="nav-group" key={g.group}>
                    <div className="nav-group-label">{g.group}</div>
                    {items.map(({ page: p, label, icon: Icon }) => (
                      <button key={p} className={'nav-item' + (p === page ? ' active' : '')} onClick={() => go(p)} aria-current={p === page ? 'page' : undefined}>
                        <Icon className="nav-icon" strokeWidth={2} />
                        {label}
                      </button>
                    ))}
                  </div>
                )
              })}
            </nav>
          </div>
          <div className="tenant-box">
            <div className="tenant-meta" style={{ padding: '0 6px 8px' }}>
              {me.master.klien < 0 ? 'Semua klien' : `${me.master.klien} klien ditugaskan`}
            </div>
            <div className="tenant-pill">
              <div className="tenant-avatar">{initials}</div>
              <div style={{ minWidth: 0, flex: 1 }}>
                <div className="tenant-name truncate">{me.user.name}</div>
                <div className="tenant-meta">{me.master.role.nama}</div>
              </div>
              <ThemeToggle className="nav-item" style={{ width: 'auto', padding: 6 }} />
              <button className="nav-item" style={{ width: 'auto', padding: 6 }} onClick={onLogout} title="Keluar" aria-label="Keluar">
                <LogOut className="nav-icon" />
              </button>
            </div>
          </div>
        </aside>
        </Laci>
        <section className="workspace" ref={setRuangKerja}>
          {page === 'ringkasan' && <Ringkasan onBuka={buka} />}
          {page === 'klien' && (klienId === null
            ? <KlienList onBuka={buka} bisaLihat={bisa('lihat_sebagai')} bisaLangganan={bisa('langganan')} onLihat={onLihat} />
            : <KlienDetailView id={klienId} bisaLangganan={bisa('langganan')} bisaLihat={bisa('lihat_sebagai')} bisaUbah={bisa('klien_ubah')}
              bisaHapus={bisa('hapus_permanen')} onLihat={onLihat} onKembali={() => setKlienId(null)} />)}
          {page === 'koreksi' && <KoreksiMaster />}
          {page === 'catatan' && <CatatanMaster />}
          {page === 'pengingat' && <PengingatMaster bisaAtur={bisa('pengaturan')} />}
          {page === 'pengguna' && <PenggunaKlienView bisaKelola={bisa('pengguna_kelola')} bisaReset={bisa('pengguna_reset')} bisaHapus={bisa('hapus_permanen')} bisaLangganan={bisa('langganan')} />}
          {page === 'pengetahuan' && <BasisPengetahuan />}
          {page === 'artikel' && <ArtikelView bolehTerbit={bisa('artikel_terbit')} />}
          {page === 'coa' && <TemplateCOAView />}
          {page === 'benchmark' && <BenchmarkView />}
          {page === 'parameter' && <ParameterView />}
          {page === 'tiket' && <TiketBoard />}
          {page === 'pembayaran' && <Pembayaran />}
          {page === 'tim' && <TimAkses selfId={me.user.id} bisaHapus={bisa('hapus_permanen')} />}
          {page === 'audit' && <AuditLog />}
        </section>
      </div>
    </main>
  )
}
