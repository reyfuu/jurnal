import { useEffect, useState } from 'react'
import { cekAwal, type BusinessProfile, type CekAwal, type SaranBentuk } from '@/lib/api'
import { formatRp } from '@/lib/format'

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

/** Saran skema pajak paling hemat (sheet "Bentuk Usaha"), dengan alasan dalam bahasa awam. */
function SaranCard({ s }: { s: SaranBentuk }) {
  const sama = s.termurah === s.sekarang
  const warna = sama ? 'var(--accent-green)' : 'var(--accent-amber-bright)'
  return (
    <section className="grid gap-3 rounded-lg border p-4" style={{ borderColor: warna }} aria-labelledby="saran-bentuk-title">
      <div>
        <div className="text-xs font-semibold" style={{ color: warna }}>Kesimpulan: bentuk usaha & skema pajak</div>
        <h3 id="saran-bentuk-title" className="mt-1 text-base font-bold">{s.judul}</h3>
        {!sama && <p className="mt-1 text-sm font-semibold tabular" style={{ color: 'var(--accent-green)' }}>Hemat sekitar {formatRp(s.hemat)} per tahun</p>}
      </div>
      <ul className="grid gap-2">
        {s.alasan.map((a) => <li key={a} className="text-sm leading-relaxed">{a}</li>)}
      </ul>
      <details>
        <summary className="cursor-pointer text-xs font-semibold text-muted-foreground">Lihat perbandingan {s.skema.length} skema</summary>
        <ul className="mt-2 grid gap-1">
          {[...s.skema].sort((a, b) => a.pajak - b.pajak).map((k) => (
            <li key={k.id} className="flex flex-wrap items-baseline justify-between gap-x-3 text-sm">
              <span>
                {k.nama}
                {k.id === s.termurah && <span className="pill-status pill-green" style={{ marginLeft: 6 }}>Disarankan</span>}
                {k.id === s.sekarang && k.id !== s.termurah && <span className="pill-status pill-amber" style={{ marginLeft: 6 }}>Sekarang</span>}
              </span>
              <span className="tabular">{formatRp(k.pajak)}</span>
            </li>
          ))}
        </ul>
      </details>
      <ul className="grid gap-1">
        {s.catatan.map((c) => <li key={c} className="text-xs text-muted-foreground">{c}</li>)}
      </ul>
    </section>
  )
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
    <div className="grid gap-4">
    {c.saran && <SaranCard s={c.saran} />}
    <section className="grid gap-4 rounded-lg border border-border p-4" aria-labelledby="cek-awal-title">
      <div>
        <h3 id="cek-awal-title" className="text-sm font-bold">Cek awal: benchmark DJP & risiko pajak</h3>
        <p className="mt-1 text-xs text-muted-foreground">
          NPM (laba bersih ÷ omzet) dan CTTOR (pajak ÷ omzet) dibandingkan rata-rata nasional DJP{c.klu ? ` untuk KLU ${c.klu.kode} ${c.klu.nama}` : ''}. Sumber: {c.sumber}.
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
    </div>
  )
}
