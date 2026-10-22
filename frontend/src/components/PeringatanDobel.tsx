import { useEffect, useState } from 'react'
import { Copy } from 'lucide-react'
import { getDobel, type Tx } from '@/lib/api'
import { formatRp, formatTanggal } from '@/lib/format'

const ASAL: Record<string, string> = { foto: 'foto nota', mutasi: 'impor mutasi' }
const ket = (t: Tx) => `#${t.id} ${formatTanggal(t.tanggal)} · ${t.jenis}${t.keterangan ? ` · ${t.keterangan}` : ''}${t.sumber ? ` (${ASAL[t.sumber] ?? t.sumber})` : ''}`

/**
 * Pemeriksaan menyeluruh transaksi dobel: pasangan transaksi lunas di periode ini dengan arah &
 * nominal sama dan tanggal berdekatan (mis. dicatat dari foto nota lalu ikut impor mutasi).
 */
export function PeringatanDobel({ period, versi }: { period: string; versi: unknown }) {
  const [pasangan, setPasangan] = useState<[Tx, Tx][]>([])
  useEffect(() => {
    let hidup = true
    getDobel(period).then((p) => { if (hidup) setPasangan(p) }, () => { if (hidup) setPasangan([]) })
    return () => { hidup = false }
  }, [period, versi])
  if (pasangan.length === 0) return null
  return (
    <div className="note-box warn" role="alert" style={{ marginTop: 0, marginBottom: 14 }}>
      <div className="flex items-center gap-2" style={{ fontWeight: 700 }}>
        <Copy size={16} aria-hidden="true" /> {pasangan.length} pasang transaksi mungkin tercatat dua kali
      </div>
      <ul style={{ margin: '6px 0 0 18px', listStyle: 'disc' }}>
        {pasangan.slice(0, 8).map(([a, b]) => (
          <li key={`${a.id}-${b.id}`} className="text-sm">{formatRp(a.nominal)}: {ket(a)} ↔ {ket(b)}</li>
        ))}
      </ul>
      <p className="text-xs" style={{ marginTop: 6, opacity: 0.85 }}>Arah &amp; nominal sama dengan tanggal berdekatan. Kalau memang sama, hapus salah satunya di daftar transaksi.</p>
    </div>
  )
}
