import { useEffect, useRef, useState, type ReactNode } from 'react'
import { ArrowLeft, BookOpen, Check, Download, ExternalLink, FileX, Link2, Paperclip } from 'lucide-react'
import { Button, buttonVariants } from '@/components/ui/button'
import { berkasPrimbonUrl, getPeraturan, simpanCatatanPrimbon, type CatatanPrimbon, type Peraturan } from '@/lib/api'
import { formatTanggal } from '@/lib/format'
import { formatWaktu } from './bantu'
import { TombolBookmark } from './KartuPeraturan'
import { TagPeraturan } from './TagPeraturan'

const tautan = buttonVariants({ variant: 'outline', size: 'sm' })

/** Halaman satu peraturan: judul & status, naskah resmi di halaman, informasi, catatan pribadi. */
export function Detail({ uuid, kembali, onKembali }: { uuid: string; kembali: string; onKembali: () => void }) {
  const [p, setP] = useState<Peraturan | null>(null)
  const [error, setError] = useState('')
  const [coba, setCoba] = useState(0)
  const judul = useRef<HTMLHeadingElement>(null)

  useEffect(() => {
    window.scrollTo({ top: 0 })
    let batal = false
    getPeraturan(uuid).then((x) => { if (!batal) { setP(x); setError('') } }, (e: Error) => { if (!batal) setError(e.message) })
    return () => { batal = true }
  }, [uuid, coba])

  // Keyboard & pembaca layar langsung berada di judul peraturan yang dibuka.
  const adaP = !!p
  useEffect(() => { if (adaP) judul.current?.focus({ preventScroll: true }) }, [adaP])

  return (
    <>
      <div>
        <Button variant="ghost" size="sm" className="-ml-2" onClick={onKembali}><ArrowLeft /> {kembali}</Button>
      </div>
      {error && (
        <div className="card flex-row flex-wrap items-center justify-between gap-3 text-sm" role="alert">
          <span className="text-[var(--text-danger)]">{error}</span>
          <Button size="sm" variant="outline" onClick={() => setCoba((n) => n + 1)}>Coba lagi</Button>
        </div>
      )}
      {!p && !error && <p className="workspace-subtitle" role="status">Memuat peraturan…</p>}
      {p && (
        <div className="grid items-start gap-5 lg:grid-cols-[minmax(0,1fr)_300px]">
          <div className="flex min-w-0 flex-col gap-4">
            <header className="flex flex-col gap-2">
              <div className="label-caps text-muted-foreground">{p.jenis_label}</div>
              <div className="font-mono text-sm font-semibold text-[var(--text-brand)]">{p.kode}</div>
              <h1 ref={judul} tabIndex={-1} className="font-heading text-2xl leading-tight font-bold tracking-tight outline-none">{p.judul}</h1>
              <TagPeraturan p={p} />
              <div className="mt-1 flex flex-wrap items-start gap-2">
                <TombolBookmark p={p} berlabel onUbah={(bookmark) => setP({ ...p, bookmark })} />
                <SalinTautan />
              </div>
            </header>
            <Naskah p={p} />
          </div>
          <aside className="flex flex-col gap-4 lg:sticky lg:top-4">
            <Informasi p={p} />
            <CatatanAnda p={p} onSimpan={(c) => setP((x) => x && { ...x, catatan: c ?? undefined, ada_catatan: !!c })} />
          </aside>
        </div>
      )}
    </>
  )
}

/** Salin alamat halaman ini (#/primbon-pajak/p/<uuid>) untuk dibagikan ke rekan satu usaha. */
function SalinTautan() {
  const [hasil, setHasil] = useState<'' | 'ok' | 'gagal'>('')
  useEffect(() => {
    if (!hasil) return
    const t = setTimeout(() => setHasil(''), 2500)
    return () => clearTimeout(t)
  }, [hasil])
  return (
    <span className="inline-flex flex-col gap-1">
      <Button variant="outline" size="sm" onClick={() => navigator.clipboard.writeText(window.location.href).then(() => setHasil('ok'), () => setHasil('gagal'))}>
        {hasil === 'ok' ? <Check /> : <Link2 />}
        {hasil === 'ok' ? 'Tautan disalin' : 'Salin tautan'}
      </Button>
      <span className="sr-only" aria-live="polite">{hasil === 'ok' ? 'Tautan disalin' : ''}</span>
      {hasil === 'gagal' && <span role="alert" className="text-xs text-[var(--text-danger)]">Tidak bisa menyalin; salin dari bilah alamat.</span>}
    </span>
  )
}

function Naskah({ p }: { p: Peraturan }) {
  if (!p.format) {
    return (
      <section className="card items-start justify-start gap-2.5 p-6">
        <FileX size={28} aria-hidden="true" className="text-[var(--text-secondary)]" />
        <h2 className="text-base font-semibold">Naskah belum tersedia di MasterPajak AI</h2>
        <p className="max-w-[560px] text-sm leading-relaxed text-muted-foreground">
          Tim kami sedang melengkapi naskah resmi peraturan ini dan akan muncul di sini begitu tersedia.
          {p.sumber_url ? ' Sementara itu, Anda bisa membacanya di sumber resmi.' : ''}
        </p>
        {p.sumber_url && (
          <a className={tautan} href={p.sumber_url} target="_blank" rel="noopener noreferrer"><ExternalLink /> Buka sumber resmi</a>
        )}
      </section>
    )
  }
  const tanpaViewer = p.format === 'pdf' && navigator.pdfViewerEnabled === false // Chrome Android & sejenisnya
  return (
    <section className="card justify-start gap-0 overflow-hidden p-0" aria-label="Naskah resmi">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-[var(--border-subtle)] px-4 py-3">
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
        <p className="bg-[var(--accent-orange-soft)] px-4 py-2 text-sm text-[var(--text-warn)]">
          JDIH Kemenkeu baru memuat abstrak peraturan ini; naskah lengkapnya belum dipublikasikan.
        </p>
      )}
      {/* Layar kecil: naskah di dalam halaman terlalu sempit untuk dibaca, jadi dibuka penuh. */}
      <div className={'flex-col items-stretch gap-2 px-4 py-5 ' + (tanpaViewer ? 'flex' : 'flex md:hidden')}>
        <a className={buttonVariants({ size: 'lg' })} href={berkasPrimbonUrl(p.uuid)} target="_blank" rel="noopener">
          <BookOpen /> Baca naskah {p.kode}
        </a>
        {tanpaViewer && (
          <p className="text-sm text-muted-foreground">Browser ini tidak bisa menampilkan PDF di dalam halaman, jadi naskah dibuka di tab baru.</p>
        )}
      </div>
      {!tanpaViewer && (
        // latar putih: naskah HTML JDIH tidak punya warna latar dan tidak terbaca di tema gelap
        <iframe title={`Naskah ${p.kode}`} src={berkasPrimbonUrl(p.uuid)} loading="lazy"
          className="hidden h-[75vh] min-h-[420px] w-full border-0 bg-[var(--neutral-0)] md:block" />
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
    ['Status', p.status || 'Belum diketahui'],
    ['Perubahan', p.perubahan],
    ['Tanggal acuan status', p.status_pada && formatTanggal(p.status_pada)],
    ['Metadata diverifikasi', p.diverifikasi_pada && formatTanggal(p.diverifikasi_pada)],
    ['Sumber verifikasi', p.sumber_verifikasi && (
      <a key="verifikasi" href={p.sumber_verifikasi} target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-1 underline underline-offset-2">
        {hostDari(p.sumber_verifikasi)} <ExternalLink size={12} aria-hidden="true" />
      </a>
    )],
    ['Lembaran negara', p.sumber_ln],
    ['Halaman', p.halaman],
    ['Sumber resmi', sumber && (
      <a key="sumber" href={p.sumber_url} target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-1 underline underline-offset-2">
        {sumber} <ExternalLink size={12} aria-hidden="true" />
      </a>
    )],
  ]
  return (
    <section className="card justify-start gap-0 p-[18px]">
      <h2 className="label-caps mb-1 text-muted-foreground">Informasi</h2>
      <dl className="text-sm">
        {baris.filter(([, v]) => v).map(([k, v]) => (
          <div key={k} className="flex justify-between gap-4 border-b border-[var(--border-subtle)] py-2 last:border-b-0">
            <dt className="shrink-0 text-muted-foreground">{k}</dt>
            <dd className="min-w-0 text-right font-semibold break-words">{v}</dd>
          </div>
        ))}
      </dl>
    </section>
  )
}

const JEDA_SIMPAN = 1000

/** Catatan tersimpan sendiri 1 detik setelah berhenti mengetik; sisa ketikan disimpan saat
 *  halaman ditinggalkan, dan browser memperingatkan bila ditutup sebelum tersimpan. */
function CatatanAnda({ p, onSimpan }: { p: Peraturan; onSimpan: (c: CatatanPrimbon | null) => void }) {
  const [isi, setIsi] = useState(p.catatan?.isi ?? '')
  const [tersimpan, setTersimpan] = useState({ isi: p.catatan?.isi ?? '', diubah: p.catatan?.diubah })
  const [sibuk, setSibuk] = useState(false)
  const [error, setError] = useState('')
  const berubah = isi.trim() !== tersimpan.isi.trim()

  const terbaru = useRef({ isi, berubah, onSimpan, uuid: p.uuid })
  useEffect(() => { terbaru.current = { isi, berubah, onSimpan, uuid: p.uuid } })

  useEffect(() => {
    if (!berubah || sibuk) return
    const t = setTimeout(async () => {
      const kirim = terbaru.current.isi
      setSibuk(true)
      setError('')
      try {
        const c = await simpanCatatanPrimbon(terbaru.current.uuid, kirim)
        setTersimpan({ isi: c?.isi ?? '', diubah: c?.diubah })
        terbaru.current.onSimpan(c)
      } catch (e) {
        setError((e as Error).message)
      } finally {
        setSibuk(false)
      }
    }, error ? JEDA_SIMPAN * 5 : JEDA_SIMPAN)
    return () => clearTimeout(t)
  }, [isi, berubah, sibuk, error])

  useEffect(() => {
    if (!berubah) return
    const tahan = (e: BeforeUnloadEvent) => e.preventDefault()
    window.addEventListener('beforeunload', tahan)
    return () => window.removeEventListener('beforeunload', tahan)
  }, [berubah])

  // Pindah ke peraturan / halaman lain sebelum jeda habis: simpan sisanya di belakang layar.
  useEffect(() => () => {
    const x = terbaru.current
    if (x.berubah) void simpanCatatanPrimbon(x.uuid, x.isi).then((c) => x.onSimpan(c), () => {})
  }, [])

  const status = error
    ? <span className="text-[var(--text-danger)]">Gagal menyimpan: {error}. Dicoba lagi otomatis.</span>
    : sibuk ? 'Menyimpan…'
      : berubah ? 'Belum tersimpan…'
        : tersimpan.diubah ? `Tersimpan ${formatWaktu(tersimpan.diubah)}` : 'Hanya Anda yang bisa membaca catatan ini.'

  return (
    <section className="card justify-start gap-0 p-[18px]">
      <label htmlFor="primbon-catatan" className="label-caps text-muted-foreground">Catatan Anda</label>
      <textarea id="primbon-catatan" rows={7} maxLength={10000} value={isi} onChange={(e) => setIsi(e.target.value)}
        placeholder="Tulis poin penting, penerapannya di usaha Anda, atau rujukan terkait…" aria-describedby="primbon-catatan-status"
        className="mt-2 w-full resize-y rounded-lg border border-input bg-[var(--bg-input)] px-3 py-2 text-sm leading-relaxed outline-none placeholder:text-muted-foreground focus-visible:border-primary focus-visible:ring-3 focus-visible:ring-primary/25" />
      <p id="primbon-catatan-status" className="mt-2 text-xs text-muted-foreground" aria-live="polite">{status}</p>
    </section>
  )
}
