import { useEffect, useState } from 'react'
import { ChevronDown, Lock, LogOut, type LucideIcon } from 'lucide-react'
import { Logo } from '@/components/Logo'
import { ThemeToggle } from '@/components/ThemeToggle'
import { BEBAS_TERKUNCI, NAV, NAV_CATAT, NAV_COBA } from '@/lib/nav'
import type { Dashboard, Me, View } from '@/lib/api'

interface Props {
  me: Me
  tenant?: Dashboard['tenant']
  view: View
  onNavigate: (v: View) => void
  onLogout: () => void
  /** Belum berlangganan: menu Coba gratis di atas, menu lain tampil bergembok. */
  terkunci?: boolean
  /** Catat cepat di grup Catat Harian (kosong = tidak boleh mencatat, mis. lihat-sebagai). */
  onCatat?: (v: View, aksi: string) => void
}

const KUNCI_GRUP = 'nav-grup-terbuka'
const grupAktif = (v: View) => [NAV_COBA, ...NAV].find((g) => g.items.some((i) => i.view === v))?.group

// Belum pernah diatur: Catat Harian terbuka, supaya Catat cepat langsung terlihat.
function bacaGrup(): string[] {
  try { return JSON.parse(localStorage.getItem(KUNCI_GRUP) ?? '["Catat Harian"]') } catch { return ['Catat Harian'] }
}

export function Sidebar({ me, tenant, view, onNavigate, onLogout, terkunci, onCatat }: Props) {
  // Grup menu bisa dibuka-tutup (submenu); grup halaman yang sedang dibuka selalu terbuka.
  const [terbuka, setTerbuka] = useState<string[]>(() => [...new Set([...bacaGrup(), grupAktif(view) ?? ''])])
  useEffect(() => {
    const g = grupAktif(view)
    if (g) setTerbuka((t) => (t.includes(g) ? t : [...t, g]))
  }, [view])
  useEffect(() => { try { localStorage.setItem(KUNCI_GRUP, JSON.stringify(terbuka)) } catch { /* abaikan */ } }, [terbuka])
  const toggle = (g: string) => setTerbuka((t) => (t.includes(g) ? t.filter((x) => x !== g) : [...t, g]))
  const initials = me.user.name.replace(/\(.*?\)/g, '').trim().split(/\s+/).map((w) => w[0]).slice(0, 2).join('').toUpperCase()
  return (
    <aside className="sidebar dark">
      <div className="sidebar-main">
        <div className="sidebar-brand">
          <Logo tinggi={24} />
        </div>

        <nav className="sidebar-nav">
        {(terkunci ? [NAV_COBA, ...NAV] : NAV).map((g) => {
          const items = g.items.filter((i) => me.views.includes(i.view) || (terkunci && g === NAV_COBA))
          if (items.length === 0) return null
          // Grup satu menu (Ringkasan) & menu coba gratis tampil langsung, tanpa kepala grup.
          const GrupIcon = (g as { icon?: LucideIcon }).icon
          const lipat = !!GrupIcon
          const buka = !lipat || terbuka.includes(g.group)
          const adaAktif = items.some((i) => i.view === view)
          return (
            <div className={'nav-group' + (lipat ? ' nav-group-lipat' : '')} key={g.group}>
              {GrupIcon ? (
                <button type="button" className={'nav-item nav-group-head' + (adaAktif && !buka ? ' active' : '')} aria-expanded={buka} onClick={() => toggle(g.group)}>
                  <GrupIcon className="nav-icon" strokeWidth={2} />
                  <span style={{ flex: 1 }}>{g.group}</span>
                  <ChevronDown size={14} aria-hidden="true" style={{ transition: 'transform .15s', transform: buka ? 'rotate(180deg)' : undefined }} />
                </button>
              ) : <div className="nav-group-label">{g.group}</div>}
              <div className={lipat ? 'nav-sub' + (buka ? '' : ' tertutup') : undefined}>
              {items.map(({ view: v, label, icon: Icon }) => {
                const gembok = terkunci && !BEBAS_TERKUNCI.includes(v)
                return (
                  <button key={v} className={'nav-item' + (v === view ? ' active' : '')} onClick={() => onNavigate(v)}
                    style={gembok ? { opacity: 0.6 } : undefined} aria-label={gembok ? `${label} (terkunci)` : undefined}>
                    <Icon className="nav-icon" strokeWidth={2} />
                    {label}
                    {gembok && <Lock size={13} style={{ marginLeft: 'auto' }} aria-hidden="true" />}
                  </button>
                )
              })}
              {g.group === 'Catat Harian' && onCatat && (() => {
                const catat = NAV_CATAT.filter((c) => me.views.includes(c.view))
                return catat.length > 0 && (
                  <>
                    <div className="nav-sub-label">Catat cepat</div>
                    {catat.map(({ label, ket, view: v, aksi, icon: Icon }) => (
                      <button key={label} type="button" className="nav-item nav-catat" title={ket} onClick={() => onCatat(v, aksi)}>
                        <Icon className="nav-icon" strokeWidth={2} />
                        {label}
                      </button>
                    ))}
                  </>
                )
              })()}
              </div>
            </div>
          )
        })}
        </nav>
      </div>

      <div className="tenant-box">
        {tenant && (
          <div className="tenant-meta tabular" style={{ padding: '0 6px 8px' }}>
            {tenant.name} · {tenant.category} · {tenant.total_transactions.toLocaleString('id-ID')} transaksi
          </div>
        )}
        <div className="tenant-pill">
          <div className="tenant-avatar">{initials}</div>
          <div style={{ minWidth: 0, flex: 1 }}>
            <div className="tenant-name truncate">{me.user.name}</div>
            <div className="tenant-meta">{me.role_label}</div>
          </div>
          <ThemeToggle className="nav-item" style={{ width: 'auto', padding: 6 }} />
          <button className="nav-item" style={{ width: 'auto', padding: 6 }} onClick={onLogout} title="Keluar" aria-label="Keluar">
            <LogOut className="nav-icon" />
          </button>
        </div>
        {tenant?.is_demo && <div className="demo-caption">Data awal berisi contoh; perubahan Anda tersimpan</div>}
      </div>
    </aside>
  )
}
