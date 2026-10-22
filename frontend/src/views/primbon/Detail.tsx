import { useEffect, useRef, useState, type ReactNode } from 'react'
import { ArrowLeft, Download, ExternalLink, FileX, Paperclip } from 'lucide-react'
import { Button, buttonVariants } from '@/components/ui/button'
import { berkasPrimbonUrl, getPeraturan, simpanCatatanPrimbon, type CatatanPrimbon, type Peraturan } from '@/lib/api'
import { formatTanggal } from '@/lib/format'
import { formatWaktu, MONO, pillStatus } from './bantu'
import { TombolBookmark } from './KartuPeraturan'

const tautan = buttonVariants({ variant: 'outline', size: 'sm' })

/** Halaman satu peraturan: judul & status, naskah resmi di halaman, informasi, catatan pribadi. */
export function Detail({ uuid, kembali, onKembali }: { uuid: string; kembali: string; onKembali: () => void }) {
  const [p, setP] = useState<Peraturan | null>(null)
  const [error, setError] = useState('')
  const atas = useRef<HTMLDivElement>(null)

  useEffect(() => {
    atas.current?.scrollIntoView({ block: 'start' })
    let batal = false
    getPeraturan(uuid).then((x) => { if (!batal) setP(x) }, (e: Error) => { if (!batal) setError(e.message) })
    return () => { batal = true }
  }, [uuid])

  return (
    <>
      <div ref={atas} className="scroll-mt-4">
        <Button variant="ghost" size="sm" onClick={onKembali}><ArrowLeft /> {kembali}</Button>
      </div>
      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start' }}>{error}</div>}
      {!p && !error && <p className="workspace-subtitle">Memuat peraturan…</p>}
      {p && (
        <div className="grid items-start gap-5 lg:grid-cols-[minmax(0,1fr)_300px]">
          <div className="flex min-w-0 flex-col gap-4">
            <header className="flex flex-col gap-2">
              <div className="label-caps text-muted-foreground">{p.jenis_label}</div>
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-sm font-semibold" style={MONO}>{p.kode}</span>
                {p.status && <span className={'pill-status ' + pillStatus(p.status)} style={{ fontSize: 11 }}>{p.status}</span>}
              </div>
              <h1 className="font-heading text-2xl leading-tight font-bold tracking-tight">{p.judul}</h1>
              <div className="flex flex-wrap gap-1.5">
                {p.kategori_label.map((k) => (
                  <span key={k} className="pill-status pill-status-muted" style={{ fontSize: 11, fontWeight: 600 }}>{k}</span>
                ))}
              </div>
              <div className="mt-1">
                <TombolBookmark p={p} berlabel onUbah={(bookmark) => setP({ ...p, bookmark })} />
              </div>
            </header>
            <Naskah p={p} />
          </div>
          <aside className="flex flex-col gap-4 lg:sticky lg:top-4">
            <Informasi p={p} />
            <CatatanAnda p={p} onSimpan={(c) => setP({ ...p, catatan: c ?? undefined, ada_catatan: !!c })} />
          </aside>
        </div>
      )}
    </>
  )
}

function Naskah({ p }: { p: Peraturan }) {
  if (!p.format) {
    return (
      <section className="card items-start" style={{ padding: 24, gap: 10, justifyContent: 'flex-start' }}>
        <FileX size={28} aria-hidden="true" style={{ color: 'var(--text-secondary)' }} />
        <h2 className="text-base font-semibold">Naskah belum tersedia di MasterPajak AI</h2>
        <p className="text-sm leading-relaxed text-muted-foreground" style={{ maxWidth: 560 }}>
          Tim kami sedang melengkapi naskah resmi peraturan ini dan akan muncul di sini begitu tersedia.
          {p.sumber_url ? ' Sementara itu, Anda bisa membacanya di sumber resmi.' : ''}
        </p>
        {p.sumber_url && (
          <a className={tautan} href={p.sumber_url} target="_blank" rel="noopener noreferrer"><ExternalLink /> Buka sumber resmi</a>
        )}
      </section>
    )
  }
  return (
    <section className="card" style={{ padding: 0, gap: 0, overflow: 'hidden', justifyContent: 'flex-start' }} aria-label="Naskah resmi">
      <div className="flex flex-wrap items-center justify-between gap-2 px-4 py-3" style={{ borderBottom: '1px solid var(--border-subtle)' }}>
        <div className="label-caps text-muted-foreground">Naskah resmi · {p.format === 'pdf' ? 'PDF' : 'HTML'}</div>
        <div className="flex flex-wrap gap-1.5">
          <a className={tautan} href={berkasPrimbonUrl(p.uuid)} target="_blank" rel="noopener"><ExternalLink /> Buka di tab baru</a>
          <a className={tautan} href={berkasPrimbonUrl(p.uuid, { unduh: true })}><Download /> Unduh</a>
          {p.lampiran && (
            <a className={tautan} href={berkasPrimbonUrl(p.uuid, { lampiran: true })} target="_blank" rel="noopener"><Paperclip /> Lampiran (PDF)</a>
          )}
        </div>
      </div>
      {p.abstrak && (
        <p className="px-4 py-2 text-sm" style={{ background: 'var(--accent-orange-soft)', color: 'var(--text-warn)' }}>
          JDIH Kemenkeu baru memuat abstrak peraturan ini; naskah lengkapnya belum dipublikasikan.
        </p>
      )}
      {p.format === 'pdf' && navigator.pdfViewerEnabled === false ? (
        // Chrome Android & sejenisnya tidak menampilkan PDF di dalam halaman.
        <p className="px-4 py-6 text-sm text-muted-foreground">
          Browser ini tidak bisa menampilkan PDF di dalam halaman. Tekan <strong>Buka di tab baru</strong> atau <strong>Unduh</strong> untuk membaca naskahnya.
        </p>
      ) : (
        // latar putih: naskah HTML JDIH tidak punya warna latar dan tidak terbaca di tema gelap
        <iframe title={`Naskah ${p.kode}`} src={berkasPrimbonUrl(p.uuid)} className="block w-full"
          style={{ height: '75vh', minHeight: 420, border: 0, background: 'var(--neutral-0)' }} />
      )}
    </section>
  )
}

function hostDari(url?: string) {
  try {
    return url ? new URL(url).hostname.replace(/^www\./, '') : ''
  } catch {
    return ''
  }
}

function Informasi({ p }: { p: Peraturan }) {
  const sumber = hostDari(p.sumber_url)
  const baris: [string, ReactNode][] = [
    ['Jenis', p.jenis_label],
    ['Nomor', p.kode],
    ['Tahun', p.tahun],
    ['Ditetapkan', p.ditetapkan && formatTanggal(p.ditetapkan)],
    ['Diundangkan', p.diundangkan && formatTanggal(p.diundangkan)],
    ['Mulai berlaku', p.berlaku && formatTanggal(p.berlaku)],
    ['Status', p.status],
    ['Lembaran negara', p.sumber_ln],
    ['Halaman', p.halaman],
    ['Sumber resmi', sumber && (
      <a key="sumber" href={p.sumber_url} target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-1 underline underline-offset-2">
        {sumber} <ExternalLink size={12} aria-hidden="true" />
      </a>
    )],
  ]
  return (
    <section className="card" style={{ padding: 18, gap: 0, justifyContent: 'flex-start' }}>
      <h2 className="label-caps mb-1 text-muted-foreground">Informasi</h2>
      <dl className="text-sm">
        {baris.filter(([, v]) => v).map(([k, v]) => (
          <div key={k} className="flex justify-between gap-4 py-2" style={{ borderBottom: '1px solid var(--border-subtle)' }}>
            <dt className="shrink-0 text-muted-foreground">{k}</dt>
            <dd className="min-w-0 text-right font-semibold break-words">{v}</dd>
          </div>
        ))}
      </dl>
    </section>
  )
}

function CatatanAnda({ p, onSimpan }: { p: Peraturan; onSimpan: (c: CatatanPrimbon | null) => void }) {
  const awal = p.catatan?.isi ?? ''
  const [isi, setIsi] = useState(awal)
  const [sibuk, setSibuk] = useState(false)
  const [error, setError] = useState('')
  const berubah = isi.trim() !== awal

  async function simpan() {
    setSibuk(true)
    setError('')
    try {
      const c = await simpanCatatanPrimbon(p.uuid, isi)
      onSimpan(c)
      setIsi(c?.isi ?? '')
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setSibuk(false)
    }
  }

  return (
    <section className="card" style={{ padding: 18, gap: 0, justifyContent: 'flex-start' }}>
      <label htmlFor="primbon-catatan" className="label-caps text-muted-foreground">Catatan Anda</label>
      <textarea id="primbon-catatan" rows={7} maxLength={10000} value={isi} onChange={(e) => setIsi(e.target.value)}
        placeholder="Tulis poin penting, penerapannya di usaha Anda, atau rujukan terkait…"
        className="mt-2 w-full rounded-lg border border-input bg-[var(--bg-input)] px-3 py-2 text-sm leading-relaxed outline-none placeholder:text-muted-foreground focus-visible:border-primary focus-visible:ring-3 focus-visible:ring-primary/25"
        style={{ resize: 'vertical' }} />
      <div className="mt-2 flex items-center justify-between gap-2">
        <span className="text-xs text-muted-foreground" aria-live="polite">
          {error
            ? <span style={{ color: 'var(--text-danger)' }}>{error}</span>
            : p.catatan ? `Disimpan ${formatWaktu(p.catatan.diubah)}` : 'Belum ada catatan. Hanya Anda yang bisa membacanya.'}
        </span>
        <Button size="sm" onClick={simpan} disabled={!berubah || sibuk}>{sibuk ? 'Menyimpan…' : 'Simpan'}</Button>
      </div>
    </section>
  )
}
