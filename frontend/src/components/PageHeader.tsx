import type { ReactNode } from 'react'

export function PageHeader({ title, subtitle, children }: { title: string; subtitle: ReactNode; children?: ReactNode }) {
  return (
    <div className="workspace-header">
      <div>
        <h1 className="workspace-title">{title}</h1>
        <p className="workspace-subtitle">{subtitle}</p>
      </div>
      {children}
    </div>
  )
}

/** Baris label–nominal di laporan keuangan. */
export function AmountRow({ label, value, color = 'var(--text-primary)', bold = false, style }: {
  label: ReactNode
  value: string
  color?: string
  bold?: boolean
  style?: React.CSSProperties
}) {
  return (
    <div className="list-row" style={style}>
      <span>{bold ? <strong>{label}</strong> : label}</span>
      <strong style={{ color }}>{value}</strong>
    </div>
  )
}
