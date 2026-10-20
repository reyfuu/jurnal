import { useEffect, useState } from 'react'
import { cekAwal, type BusinessProfile, type CekAwal } from '@/lib/api'

// 2 desimal, sama dengan PDF DJP dan pesan dari server
const formatPct = (bps: number) => (bps / 100).toFixed(2).replace('.', ',') + '%'

const STATUS: Record<CekAwal['status'], { label: string; color: string }> = {
  wajar: { label: 'Setara / di atas rata-rata', color: 'var(--accent-green)' },
  di_bawah: { label: 'Di bawah rata-rata', color: 'var(--accent-red)' },
  kurang_data: { label: 'Belum ada data', color: 'var(--text-secondary)' },
}
const LEVEL: Record<CekAwal['risiko'][number]['level'], { label: string; color: string }> = {
  tinggi: { label: 'Risiko tinggi', color: 'var(--accent-red)' },
  sedang: { label: 'Perlu dipertimbangkan', color: 'var(--accent-amber-bright)' },
  info: { label: 'Info', color: 'var(--text-secondary)' },
}

// CekAwalCard: NPM vs rasio benchmark DJP + risiko pajak. Dihitung backend (tax.CekAwal),
// dipakai di ringkasan onboarding dan Perencanaan Pajak.
export function CekAwalCard({ profile }: { profile: BusinessProfile }) {
  const [c, setC] = useState<CekAwal | null>(null)
  const [error, setError] = useState('')
  const key = JSON.stringify(profile)

  useEffect(() => {
    let live = true
    cekAwal(JSON.parse(key)).then((r) => live && setC(r), (e: Error) => live && setError(e.message))
    return () => { live = false }
  }, [key])

  if (error) return <p className="text-sm text-destructive" role="alert">{error}</p>
  if (!c) return <p className="text-sm text-muted-foreground">Menghitung cek awal…</p>
  const st = STATUS[c.status]
  return (
    <section className="grid gap-4 rounded-lg border border-border p-4" aria-labelledby="cek-awal-title">
      <div>
        <h3 id="cek-awal-title" className="text-sm font-bold">Cek awal: benchmark DJP & risiko pajak</h3>
        <p className="mt-1 text-xs text-muted-foreground">
          NPM (laba sebelum pajak ÷ omzet) dan CTTOR (pajak ÷ omzet) dibandingkan rata-rata nasional DJP{c.klu ? ` untuk KLU ${c.klu.kode} ${c.klu.nama}` : ''}. Sumber: {c.sumber}.
        </p>
      </div>
      {c.status !== 'kurang_data' && c.klu && (
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="rounded-md border border-border p-3">
            <div className="text-xs text-muted-foreground">NPM</div>
            <div className="mt-1 flex flex-wrap items-baseline gap-x-2"><span className="text-2xl font-bold tabular">{formatPct(c.npm_bps)}</span><span className="text-xs text-muted-foreground tabular">nasional {formatPct(c.klu.npm_bps)}</span></div>
            <div className="mt-1 text-xs font-semibold" style={{ color: st.color }}>{st.label}</div>
          </div>
          <div className="rounded-md border border-border p-3">
            <div className="text-xs text-muted-foreground">CTTOR</div>
            <div className="mt-1 flex flex-wrap items-baseline gap-x-2"><span className="text-2xl font-bold tabular">{formatPct(c.cttor_bps)}</span><span className="text-xs text-muted-foreground tabular">nasional {formatPct(c.klu.cttor_bps)}</span></div>
          </div>
        </div>
      )}
      <p className="text-sm leading-relaxed">{c.pesan}</p>
      {c.cttor_pesan && <p className="text-sm leading-relaxed text-muted-foreground">{c.cttor_pesan}</p>}
      {c.risiko.length > 0 ? (
        <ul className="grid gap-3">
          {c.risiko.map((r) => (
            <li key={r.judul} className="border-l-2 pl-3" style={{ borderColor: LEVEL[r.level].color }}>
              <div className="text-xs font-semibold" style={{ color: LEVEL[r.level].color }}>{LEVEL[r.level].label}</div>
              <div className="text-sm font-semibold">{r.judul}</div>
              <p className="text-sm text-muted-foreground">{r.isi}</p>
            </li>
          ))}
        </ul>
      ) : c.status !== 'kurang_data' && <p className="text-sm text-muted-foreground">Tidak ada risiko pajak menonjol dari data ini.</p>}
    </section>
  )
}
