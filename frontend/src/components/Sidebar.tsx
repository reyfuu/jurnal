import { LogOut } from 'lucide-react'
import { NAV } from '@/lib/nav'
import type { Dashboard, Me, View } from '@/lib/api'

export function BrandMark({ size = 22 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 28 28" fill="none" aria-hidden>
      <path d="M14 3L25 23H18.5L14 14.2L9.5 23H3L14 3Z" fill="#F5C428" />
    </svg>
  )
}

interface Props {
  me: Me
  tenant?: Dashboard['tenant']
  view: View
  onNavigate: (v: View) => void
  onLogout: () => void
}

export function Sidebar({ me, tenant, view, onNavigate, onLogout }: Props) {
  const initials = me.user.name.replace(/\(.*?\)/g, '').trim().split(/\s+/).map((w) => w[0]).slice(0, 2).join('').toUpperCase()
  return (
    <aside className="sidebar">
      <div className="sidebar-main">
        <div className="sidebar-brand">
          <BrandMark />
          <div>
            <div style={{ fontSize: 13.5, fontWeight: 700 }}>Laporan Bisnis</div>
            <div style={{ fontSize: 10, color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}>by Akademi Pajak</div>
          </div>
        </div>

        <nav className="sidebar-nav">
        {NAV.map((g) => {
          const items = g.items.filter((i) => me.views.includes(i.view))
          if (items.length === 0) return null
          return (
            <div className="nav-group" key={g.group}>
              <div className="nav-group-label">{g.group}</div>
              {items.map(({ view: v, label, icon: Icon }) => (
                <button key={v} className={'nav-item' + (v === view ? ' active' : '')} onClick={() => onNavigate(v)}>
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
          <button className="nav-item" style={{ width: 'auto', padding: 6 }} onClick={onLogout} title="Keluar" aria-label="Keluar">
            <LogOut className="nav-icon" />
          </button>
        </div>
        <div className="demo-caption">Data awal berisi contoh; perubahan Anda tersimpan</div>
      </div>
    </aside>
  )
}
