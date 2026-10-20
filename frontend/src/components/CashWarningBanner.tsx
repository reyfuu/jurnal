import { AlertTriangle } from 'lucide-react'
import type { CashWarning } from '@/lib/api'
import { formatRp } from '@/lib/format'

const tgl = (iso: string) => new Date(iso + 'T00:00:00').toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' })

// Saldo minus tidak memblokir simpan: UMKM sering telat mencatat pemasukan, jadi cukup diingatkan.
export function CashWarningBanner({ warnings }: { warnings?: CashWarning[] | null }) {
  if (!warnings?.length) return null
  const first = warnings[0]
  const lowest = warnings.reduce((m, w) => (w.saldo < m.saldo ? w : m), first)
  const penyebab = [...new Set(warnings.flatMap((w) => w.penyebab ?? []))]
  return (
    <div className="card" role="alert" style={{ borderColor: 'var(--text-danger)', background: 'color-mix(in srgb, var(--text-danger) 8%, transparent)', display: 'flex', gap: 12, alignItems: 'flex-start' }}>
      <AlertTriangle size={18} color="var(--text-danger)" style={{ flexShrink: 0, marginTop: 2 }} aria-hidden />
      <div style={{ fontSize: 13, lineHeight: 1.5 }}>
        <div style={{ fontWeight: 700, color: 'var(--text-danger)' }}>
          Saldo kas minus mulai {tgl(first.tanggal)}
          {warnings.length > 1 && ` (${warnings.length} hari)`}, terendah {formatRp(lowest.saldo)} pada {tgl(lowest.tanggal)}
        </div>
        <div style={{ color: 'var(--text-muted)' }}>
          Kas tidak mungkin di bawah nol. Kemungkinan ada pemasukan yang belum dicatat, atau nominal pengeluaran salah.
          {penyebab.length > 0 && ` Periksa pengeluaran: ${penyebab.map((id) => '#' + id).join(', ')}.`}
        </div>
      </div>
    </div>
  )
}
