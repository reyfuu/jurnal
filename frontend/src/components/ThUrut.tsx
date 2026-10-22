import type { CSSProperties, ReactNode } from 'react'
import { ArrowDown, ArrowUp, ArrowUpDown } from 'lucide-react'
import type { Arah } from '@/lib/urut'

/** Judul kolom yang bisa diklik untuk mengurutkan naik / turun. */
export function ThUrut({ k, urut, children, kanan, style }: {
  k: string; urut: { kunci: string | null; arah: Arah; atur: (k: string) => void }; children: ReactNode; kanan?: boolean; style?: CSSProperties
}) {
  const aktif = urut.kunci === k
  const Ikon = !aktif ? ArrowUpDown : urut.arah === 'naik' ? ArrowUp : ArrowDown
  return (
    <th aria-sort={aktif ? (urut.arah === 'naik' ? 'ascending' : 'descending') : 'none'} style={{ textAlign: kanan ? 'right' : undefined, ...style }}>
      <button type="button" onClick={() => urut.atur(k)} title="Urutkan"
        style={{ all: 'unset', cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 4, flexDirection: kanan ? 'row-reverse' : 'row', font: 'inherit', textTransform: 'inherit', letterSpacing: 'inherit' }}>
        {children}
        <Ikon size={12} aria-hidden="true" style={{ opacity: aktif ? 1 : 0.35, flexShrink: 0 }} />
      </button>
    </th>
  )
}
