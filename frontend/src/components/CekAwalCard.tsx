import { useEffect, useState } from 'react'
import { cekAwal, type BusinessProfile, type CekAwal, type SaranBentuk } from '@/lib/api'
import { formatRp } from '@/lib/format'
import { InfoTip, TebalAngka } from './InfoTip'

// Penjelasan singkat istilah di kartu cek awal; poin yang perlu diperhatikan ditebalkan.
export function PenjelasanNPM() {
  return (
    <>
      <span><strong>NPM (Net Profit Margin)</strong> = laba bersih ÷ omzet. Artinya: dari setiap Rp100 penjualan, berapa rupiah yang menjadi <strong>laba bersih</strong>.</span>
      <span>Gunanya: kantor pajak membandingkan NPM usaha dengan <strong>rata-rata usaha sejenis</strong>. NPM yang <strong>jauh di bawah rata-rata</strong> bisa dianggap ada omzet yang belum dicatat atau biaya yang terlalu besar, sehingga lebih mungkin <strong>ditanya atau diperiksa</strong>.</span>
    </>
  )
}

export function PenjelasanCTTOR() {
  return (
    <>
      <span><strong>CTTOR (Corporate Tax to Turnover Ratio)</strong> = pajak penghasilan ÷ omzet. Artinya: dari setiap Rp100 penjualan, berapa rupiah yang dibayar sebagai <strong>pajak</strong>.</span>
      <span>Gunanya: menunjukkan apakah pajak yang dibayar <strong>wajar dibanding usaha sejenis</strong>. CTTOR yang <strong>jauh di bawah rata-rata</strong> sering menjadi alasan kantor pajak meminta penjelasan. Untuk <strong>PPh Final 0,5%</strong>, angka ini sudah pasti dari omzet, jadi hanya sebagai informasi.</span>
    </>
  )
}

// 2 desimal, sama dengan data sumber dan pesan dari server
const formatPct = (bps: number) => (bps / 100).toFixed(2).replace('.', ',') + '%'

const STATUS: Record<CekAwal['status'], { label: string; color: string }> = {
  wajar: { label: 'Setara / di atas rata-rata', color: 'var(--accent-green)' },
  di_bawah: { label: 'Di bawah rata-rata', color: 'var(--accent-red)' },
  kurang_data: { label: 'Belum ada data', color: 'var(--text-secondary)' },
}
const LEVEL: Record<CekAwal['risiko'][number]['level'], { label: string; color: string }> = {
  tinggi: { label: 'Risiko tinggi', color: 'var(--accent-red)' },
  sedang: { label: 'Perlu dipertimbangkan', color: 'var(--accent-orange)' },
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
      </div>
      {!sama && (
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="relative rounded-md border border-border p-3">
            <span className="absolute right-2 top-2"><InfoTip label="penghematan pajak"><span>Selisih pajak setahun antara skema <strong>Anda sekarang</strong> dan skema <strong>yang disarankan</strong>, dalam persen dari pajak sekarang.</span></InfoTip></span>
            <div className="text-xs text-muted-foreground">Penghematan pajak</div>
            <div className="mt-1 text-2xl font-bold tabular" style={{ color: 'var(--accent-green)' }}>{formatPct(s.hemat_bps)}</div>
            <div className="text-xs text-muted-foreground">lebih kecil dari skema Anda sekarang</div>
          </div>
          <div className="relative rounded-md border border-border p-3">
            <span className="absolute right-2 top-2"><InfoTip label="hemat per tahun"><span>Perkiraan pajak yang <strong>tidak perlu dibayar</strong> setahun bila memakai skema yang disarankan. Dihitung dari omzet & laba yang Anda isi; angka sebenarnya mengikuti pembukuan.</span></InfoTip></span>
            <div className="text-xs text-muted-foreground">Hemat per tahun</div>
            <div className="mt-1 text-2xl font-bold tabular" style={{ color: 'var(--accent-green)' }}>{formatRp(s.hemat)}</div>
            <div className="text-xs text-muted-foreground">{formatRp(s.hemat ? Math.round(s.hemat / 12) : 0)} per bulan</div>
          </div>
        </div>
      )}
      <ul className="grid gap-2">
        {s.alasan.map((a) => <li key={a} className="text-sm leading-relaxed"><TebalAngka teks={a} /></li>)}
      </ul>
      <div>
        <div className="text-xs font-semibold text-muted-foreground">Perbandingan {s.skema.length} skema pajak per tahun (% dari omzet)</div>
        <ul className="mt-2 grid gap-2.5">
          {(() => {
            const urut = [...s.skema].sort((a, b) => a.pajak - b.pajak)
            const maks = Math.max(1, ...urut.map((k) => k.pajak))
            return urut.map((k) => {
              const warnaBar = k.id === s.termurah ? 'var(--accent-green)' : k.id === s.sekarang ? 'var(--accent-amber-bright)' : 'var(--border-strong)'
              return (
                <li key={k.id} className="grid gap-1 rounded-md text-sm" style={k.id === s.termurah ? { background: 'color-mix(in srgb, var(--accent-green) 14%, transparent)', padding: '6px 8px', margin: '0 -8px' } : undefined}>
                  <div className="flex flex-wrap items-baseline justify-between gap-x-3">
                    <span>
                      {k.nama}
                      {k.id === s.termurah && <span className="pill-status pill-green" style={{ marginLeft: 6 }}>Disarankan</span>}
                      {k.id === s.sekarang && k.id !== s.termurah && <span className="pill-status pill-amber" style={{ marginLeft: 6 }}>Sekarang</span>}
                    </span>
                    <span className="tabular">
                      {formatRp(k.pajak)}
                      <span className="text-xs text-muted-foreground"> · {s.omzet > 0 ? formatPct(Math.round((k.pajak * 10_000) / s.omzet)) : '0,00%'}</span>
                    </span>
                  </div>
                  <div className="h-1.5 rounded-full" style={{ background: 'var(--border-subtle)' }} aria-hidden="true">
                    <div className="h-1.5 rounded-full" style={{ width: `${Math.max(2, (k.pajak / maks) * 100)}%`, background: warnaBar }} />
                  </div>
                </li>
              )
            })
          })()}
        </ul>
      </div>
      <ul className="grid gap-1">
        {s.catatan.map((c) => <li key={c} className="text-xs text-muted-foreground">{c}</li>)}
      </ul>
    </section>
  )
}

// CekAwalCard: NPM vs rasio benchmark nasional + risiko pajak. Dihitung backend (tax.CekAwal),
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
        <h3 id="cek-awal-title" className="text-sm font-bold">Cek awal: benchmark nasional & risiko pajak</h3>
        <p className="mt-1 text-xs text-muted-foreground">
          <strong>NPM</strong> (laba bersih ÷ omzet) dan <strong>CTTOR</strong> (pajak ÷ omzet) dibandingkan dengan rata-rata nasional{c.klu ? ` untuk KLU ${c.klu.kode} ${c.klu.nama}` : ''}. Arahkan kursor ke tanda tanya untuk penjelasannya.
        </p>
      </div>
      {c.status !== 'kurang_data' && c.klu && (
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="relative rounded-md border border-border p-3">
            <span className="absolute right-2 top-2"><InfoTip label="NPM"><PenjelasanNPM /></InfoTip></span>
            <div className="text-xs text-muted-foreground">NPM</div>
            <div className="mt-1 flex flex-wrap items-baseline gap-x-2"><span className="text-2xl font-bold tabular">{formatPct(c.npm_bps)}</span><span className="text-xs text-muted-foreground tabular">nasional {formatPct(c.klu.npm_bps)}</span></div>
            <div className="mt-1 text-xs font-semibold" style={{ color: st.color }}>{st.label}</div>
          </div>
          <div className="relative rounded-md border border-border p-3">
            <span className="absolute right-2 top-2"><InfoTip label="CTTOR"><PenjelasanCTTOR /></InfoTip></span>
            <div className="text-xs text-muted-foreground">CTTOR</div>
            <div className="mt-1 flex flex-wrap items-baseline gap-x-2"><span className="text-2xl font-bold tabular">{formatPct(c.cttor_bps)}</span><span className="text-xs text-muted-foreground tabular">nasional {formatPct(c.klu.cttor_bps)}</span></div>
          </div>
        </div>
      )}
      <p className="text-sm leading-relaxed"><TebalAngka teks={c.pesan} /></p>
      {c.cttor_pesan && <p className="text-sm leading-relaxed text-muted-foreground"><TebalAngka teks={c.cttor_pesan} /></p>}
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
