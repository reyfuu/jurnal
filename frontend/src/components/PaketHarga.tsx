import { useState } from 'react'
import { Check } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { SegeraHadir } from '@/components/SegeraHadir'
import type { Paket } from '@/lib/api'
import { formatRp } from '@/lib/format'

type Periode = 'bulanan' | 'tahunan'

// Isi tiap tingkat (sama dengan landing page). Fitur bertanda segera = belum bisa dipakai;
// semua paket saat ini membuka seluruh aplikasi.
const TINGKAT: { id: string; nama: string; populer?: boolean; fitur: [string, segera?: boolean][] }[] = [
  { id: 'starter', nama: 'Starter', fitur: [['Foto nota via WhatsApp jadi jurnal', true], ['4 laporan keuangan otomatis'], ['Konsultan Pajak AI'], ['Zoom mingguan bersama tim ahli']] },
  { id: 'pro', nama: 'Pro', populer: true, fitur: [['Semua fitur Starter'], ['Pajak masukan otomatis jadi kredit pajak', true], ['Rekonsiliasi bulanan vs tahunan'], ['Ekspor XML siap Coretax', true]] },
  { id: 'premium', nama: 'Premium Master', fitur: [['Semua fitur Pro'], ['Akses webinar eksklusif'], ['Pembahasan SP2DK dan ekualisasi'], ['Pendampingan pemeriksaan dan audit']] },
]

const periodeDari = (p: Paket): Periode => (p.hari >= 365 ? 'tahunan' : 'bulanan')

/**
 * Pilihan periode (bulanan/tahunan) + kartu 3 tingkat. Dipakai di landing (tombol daftar) dan
 * menu Langganan (tombol bayar). pilihan: id paket yang disorot (dipilih di landing).
 */
export function PaketHarga({ paket, onPilih, label, busy, disabled, pilihan }: {
  paket: Paket[]
  onPilih: (p: Paket) => void
  label: (p: Paket) => string
  busy?: string
  disabled?: boolean
  pilihan?: string
}) {
  const [periode, setPeriode] = useState<Periode>(() => {
    const p = paket.find((x) => x.id === pilihan)
    return p ? periodeDari(p) : 'bulanan'
  })
  const cari = (tingkat: string, per: Periode) => paket.find((p) => p.tingkat === tingkat && periodeDari(p) === per)

  return (
    <div className="@container flex flex-col items-center">
      <div className="inline-flex rounded-full border border-[var(--border-subtle)] p-1" role="group" aria-label="Periode tagihan">
        {(['bulanan', 'tahunan'] as const).map((per) => (
          <button key={per} type="button" aria-pressed={periode === per} onClick={() => setPeriode(per)}
            className="rounded-full px-[18px] py-2 text-sm font-semibold text-[var(--text-secondary)] aria-pressed:bg-primary aria-pressed:text-primary-foreground">
            {per === 'bulanan' ? 'Bulanan' : 'Tahunan'}
            {per === 'tahunan' && <span className="ml-1.5 text-xs text-[var(--accent-green)]">hemat 2 bulan</span>}
          </button>
        ))}
      </div>
      {/* 3 kolom hanya bila wadahnya lebar (landing); di layar bayar yang sempit kartu bertumpuk */}
      <div className="mt-8 grid w-full gap-4 text-left @3xl:grid-cols-3">
        {TINGKAT.map((t) => {
          const p = cari(t.id, periode)
          if (!p) return null
          const lain = cari(t.id, periode === 'bulanan' ? 'tahunan' : 'bulanan')
          const alt = periode === 'bulanan'
            ? lain && `atau ${formatRp(lain.harga)} per tahun`
            : `setara ${formatRp(Math.round(p.harga / 12_000) * 1000)} per bulan`
          return (
            <article key={t.id}
              className={`relative flex flex-col gap-3.5 rounded-[18px] bg-[var(--bg-card)] p-[26px] ${t.populer ? 'border-2 border-[var(--ink)] shadow-[var(--shadow-3)]' : 'border border-[var(--border-subtle)]'} ${pilihan === p.id ? 'ring-2 ring-[var(--yellow-500)] ring-offset-2 ring-offset-[var(--bg-card)]' : ''}`}>
              {t.populer && (
                <span className="absolute -top-3 left-1/2 -translate-x-1/2 whitespace-nowrap rounded-full bg-[var(--yellow-500)] px-3 py-1 text-xs font-bold text-[var(--navy-900)]">Paling populer</span>
              )}
              <h3 className="font-heading text-[19px] font-semibold">{t.nama}</h3>
              {p.harga_normal ? (
                <div className="text-base font-medium text-[var(--text-secondary)] line-through decoration-[var(--error)] decoration-2">{formatRp(p.harga_normal)}</div>
              ) : null}
              <div className="font-heading text-4xl font-bold leading-none text-[var(--text-brand)] tabular">
                {formatRp(p.harga)} <small className="font-sans text-sm font-medium text-[var(--text-secondary)]">/ {periode === 'bulanan' ? 'bulan' : 'tahun'}</small>
              </div>
              <p className="-mt-1.5 min-h-[1.4em] text-[13px] text-[var(--text-muted)]">{alt}</p>
              <ul className="grid gap-2.5 border-t border-[var(--border-subtle)] pt-3.5 text-[15px]">
                {t.fitur.map(([f, segera]) => (
                  <li key={f} className="flex gap-2.5">
                    <Check className="mt-[3px] size-[18px] shrink-0 text-[var(--text-brand)]" aria-hidden="true" />
                    <span>{f}{segera && <SegeraHadir />}</span>
                  </li>
                ))}
              </ul>
              <Button size="lg" variant={t.populer ? 'default' : 'outline'} className="mt-auto w-full" disabled={disabled || !!busy}
                onClick={() => onPilih(p)}>
                {busy === p.id ? 'Membuka pembayaran…' : label(p)}
              </Button>
            </article>
          )
        })}
      </div>
    </div>
  )
}
