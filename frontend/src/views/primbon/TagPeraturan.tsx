import type { Peraturan } from '@/lib/api'
import { formatTanggal } from '@/lib/format'
import { tagPeraturan } from './tag'

export function TagPeraturan({ p }: { p: Peraturan }) {
  return (
    <div className="mt-2.5 flex min-w-0 flex-col gap-2">
      <div className="flex flex-wrap gap-1.5" aria-label="Tag peraturan">
        {tagPeraturan(p).map(tag => (
          <span key={tag.label} className={`pill-status ${tag.warna} max-w-full break-words whitespace-normal text-xs`}>{tag.label}</span>
        ))}
      </div>
      <p className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
        <span>{p.berlaku ? `Mulai berlaku ${formatTanggal(p.berlaku)}` : 'Mulai berlaku belum diketahui'}</span>
        <span title="Status dari metadata pada tanggal ini; bukan pemeriksaan otomatis atas perubahan peraturan.">
          {p.status_pada ? `Status per ${formatTanggal(p.status_pada)}` : 'Tanggal acuan status belum tersedia'}
        </span>
      </p>
    </div>
  )
}
