import { useState, type ReactNode } from 'react'
import { Search, SlidersHorizontal, X } from 'lucide-react'

export interface ChipFilter { label: string; onHapus: () => void }

/**
 * Kepala halaman ringkas di HP: judul, ikon cari, dan ikon filter dalam satu baris. Ketuk 🔍 →
 * kotak cari menggantikan judul; ✕ menutup sekaligus mengosongkan. Filter aktif tampil sebagai
 * label kecil yang bisa dihapus satu per satu.
 */
export function KepalaHP({ judul, cari, onCari, placeholderCari = 'Cari…', jumlahFilter = 0, onFilter, chips = [], aksi }: {
  judul: ReactNode
  cari?: string
  onCari?: (q: string) => void
  placeholderCari?: string
  jumlahFilter?: number
  onFilter?: () => void
  chips?: ChipFilter[]
  /** tombol ikon tambahan di kanan (mis. ekspor) */
  aksi?: ReactNode
}) {
  const [bukaCari, setBukaCari] = useState(!!cari)
  const tutupCari = () => { onCari?.(''); setBukaCari(false) }
  return (
    <div className="kepala-hp">
      <div className="kepala-hp-baris">
        {bukaCari && onCari ? (
          <label className="kepala-hp-cari">
            <Search size={15} aria-hidden="true" />
            <input autoFocus value={cari ?? ''} placeholder={placeholderCari} aria-label={placeholderCari} onChange={(e) => onCari(e.target.value)} />
            <button type="button" aria-label="Tutup pencarian" onClick={tutupCari}><X size={15} aria-hidden="true" /></button>
          </label>
        ) : (
          <h1 className="kepala-hp-judul">{judul}</h1>
        )}
        {onCari && !bukaCari && (
          <button type="button" className="kepala-hp-ikon" aria-label="Cari" onClick={() => setBukaCari(true)}><Search size={16} aria-hidden="true" /></button>
        )}
        {aksi}
        {onFilter && (
          <button type="button" className="kepala-hp-ikon" aria-label={jumlahFilter ? `Filter (${jumlahFilter} aktif)` : 'Filter'} onClick={onFilter}>
            <SlidersHorizontal size={16} aria-hidden="true" />
            {jumlahFilter > 0 && <span className="kepala-hp-lencana">{jumlahFilter}</span>}
          </button>
        )}
      </div>
      {chips.length > 0 && (
        <div className="kepala-hp-chips">
          {chips.map((c) => (
            <button type="button" key={c.label} className="kepala-hp-chip" onClick={c.onHapus} aria-label={`Hapus filter ${c.label}`}>
              {c.label} <X size={12} aria-hidden="true" />
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
