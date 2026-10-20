import { useState } from 'react'
import { CreditCard, LayoutGrid, LogOut, ScrollText, Ticket, UserCog, Users, type LucideIcon } from 'lucide-react'
import { BrandMark } from '@/components/Sidebar'
import { ThemeToggle } from '@/components/ThemeToggle'
import type { Izin, Me } from '@/lib/api'
import { KlienDetailView, KlienList, Ringkasan } from './Klien'
import { TiketBoard } from './Tiket'
import { TimAkses } from './Tim'
import { AuditLog, Pembayaran } from './Lainnya'

export type Page = 'ringkasan' | 'klien' | 'tiket' | 'pembayaran' | 'tim' | 'audit'

// Menu Panel Master; izin kosong = cukup melihat klien (klien_lihat atau klien_semua).
const NAV: { group: string; items: { page: Page; label: string; icon: LucideIcon; izin?: Izin }[] }[] = [
  {
    group: 'Klien',
    items: [
      { page: 'ringkasan', label: 'Ringkasan', icon: LayoutGrid },
      { page: 'klien', label: 'Klien', icon: Users },
      { page: 'tiket', label: 'Tiket · Lapor Bug', icon: Ticket, izin: 'tiket' },
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

/** Seluruh aplikasi untuk akun tim MasterPajak: hanya metadata klien, tanpa angka keuangan. */
export function PanelMaster({ me, onLogout }: { me: Me; onLogout: () => void }) {
  const izin = me.master?.izin ?? []
  const bisa = (i?: Izin) => (i ? izin.includes(i) : izin.includes('klien_lihat') || izin.includes('klien_semua'))
  const [page, setPage] = useState<Page>('ringkasan')
  const [klienId, setKlienId] = useState<number | null>(null)
  const go = (p: Page) => { setPage(p); setKlienId(null) }
  const buka = (id: number) => { setPage('klien'); setKlienId(id) }
  const initials = me.user.name.split(/\s+/).map((w) => w[0]).slice(0, 2).join('').toUpperCase()

  if (!me.master) {
    return (
      <main className="onboarding-page" style={{ padding: 32 }}>
        <p role="alert">Akun ini belum terdaftar di tim MasterPajak. Minta Master menambahkan Anda di Tim &amp; Akses.</p>
        <button className="btn-cta" style={{ marginTop: 16 }} onClick={onLogout}>Keluar</button>
      </main>
    )
  }

  return (
    <main>
      <div className="app-shell">
        <aside className="sidebar dark">
          <div className="sidebar-main">
            <div className="sidebar-brand">
              <BrandMark />
              <div>
                <div style={{ fontSize: 13.5, fontWeight: 700 }}>Laporan Bisnis</div>
                <div style={{ fontSize: 10, color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}>Panel Master</div>
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
        <section className="workspace">
          {page === 'ringkasan' && <Ringkasan onBuka={buka} />}
          {page === 'klien' && (klienId === null
            ? <KlienList onBuka={buka} />
            : <KlienDetailView id={klienId} bisaLangganan={bisa('langganan')} onKembali={() => setKlienId(null)} />)}
          {page === 'tiket' && <TiketBoard />}
          {page === 'pembayaran' && <Pembayaran />}
          {page === 'tim' && <TimAkses selfId={me.user.id} />}
          {page === 'audit' && <AuditLog />}
        </section>
      </div>
    </main>
  )
}
