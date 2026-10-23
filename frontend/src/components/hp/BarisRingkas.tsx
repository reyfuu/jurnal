import type { ReactNode } from 'react'
import { MoreHorizontal, Pencil } from 'lucide-react'

/**
 * Satu baris daftar ringkas di HP (Panel Master): titik status, judul, keterangan kecil, lalu
 * tombol ikon ✎ (ubah) dan ⋯ (aksi lain di lembar bawah). Ketuk baris = onBuka (bila ada).
 */
export function BarisRingkas({ aktif, judul, sub, onBuka, onUbah, onLainnya, labelUbah = 'Ubah', label }: {
  /** titik hijau (true) / abu (false); undefined = tanpa titik */
  aktif?: boolean
  judul: ReactNode
  sub?: ReactNode
  onBuka?: () => void
  onUbah?: () => void
  onLainnya?: () => void
  labelUbah?: string
  /** nama untuk aria-label tombol, mis. nama user */
  label: string
}) {
  const isi = (
    <>
      {aktif !== undefined && <span className={'baris-ringkas-titik' + (aktif ? ' aktif' : '')} aria-label={aktif ? 'aktif' : 'nonaktif'} role="img" />}
      <span className="baris-ringkas-teks">
        <b>{judul}</b>
        {sub && <span>{sub}</span>}
      </span>
    </>
  )
  return (
    <div className="baris-ringkas">
      {onBuka
        ? <button type="button" className="baris-ringkas-utama" onClick={onBuka}>{isi}</button>
        : <div className="baris-ringkas-utama">{isi}</div>}
      {onUbah && <button type="button" className="kepala-hp-ikon" aria-label={`${labelUbah} ${label}`} onClick={onUbah}><Pencil size={15} aria-hidden="true" /></button>}
      {onLainnya && <button type="button" className="kepala-hp-ikon" aria-label={`Aksi lain untuk ${label}`} onClick={onLainnya}><MoreHorizontal size={17} aria-hidden="true" /></button>}
    </div>
  )
}
