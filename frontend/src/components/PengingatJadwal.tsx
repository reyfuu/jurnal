import { useState } from 'react'
import { CalendarClock, Check } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { PengingatJadwal } from '@/lib/api'
import { formatTanggal } from '@/lib/format'

// Banner tenggat laporan & pajak dari jadwal Panel Master. "Sudah dikerjakan" hanya
// menyembunyikan tenggat itu di browser ini (localStorage); tenggat berikutnya tetap muncul.

const kunciSelesai = (j: PengingatJadwal) => `jadwal-selesai:${j.id}:${j.jatuh_tempo}`

function sudah(j: PengingatJadwal) {
  try {
    return localStorage.getItem(kunciSelesai(j)) === '1'
  } catch {
    return false
  }
}

const sisa = (n: number) => (n <= 0 ? 'hari ini' : n === 1 ? 'besok' : `${n} hari lagi`)

export function PengingatJadwalBanner({ jadwal }: { jadwal: PengingatJadwal[] }) {
  const [, setUlang] = useState(0)
  const tampil = jadwal.filter((j) => !sudah(j))
  if (tampil.length === 0) return null
  const tandai = (j: PengingatJadwal) => {
    try {
      localStorage.setItem(kunciSelesai(j), '1')
    } catch { /* penyimpanan diblokir: banner tetap tampil */ }
    setUlang((n) => n + 1)
  }
  return (
    <div className="mb-4 grid gap-2 rounded-lg border px-4 py-3 text-sm" style={{ borderColor: 'var(--accent-blue)' }} role="status" aria-label="Pengingat tenggat">
      {tampil.map((j) => (
        <div key={j.id} className="flex flex-wrap items-center gap-3">
          <CalendarClock size={18} style={{ color: j.sisa_hari <= 1 ? 'var(--text-danger)' : 'var(--accent-blue)', flexShrink: 0 }} />
          <span className="min-w-0 flex-1">
            <strong>{j.judul}</strong> · jatuh tempo {formatTanggal(j.jatuh_tempo)}{' '}
            <span style={{ color: j.sisa_hari <= 1 ? 'var(--text-danger)' : undefined, fontWeight: 600 }}>({sisa(j.sisa_hari)})</span>
            {j.keterangan && <span className="block text-xs text-muted-foreground">{j.keterangan}</span>}
          </span>
          <Button size="sm" variant="outline" className="rounded-full" onClick={() => tandai(j)}><Check /> Sudah dikerjakan</Button>
        </div>
      ))}
    </div>
  )
}
