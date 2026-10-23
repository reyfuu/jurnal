import { useState, type ReactNode } from 'react'
import { Bookmark, BookmarkCheck, FileText, StickyNote } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { setBookmarkPrimbon, type Peraturan } from '@/lib/api'
import { jenisSingkat, keteranganNaskah } from './bantu'
import { pecahSorot } from './sorot'
import { tagPeraturan } from './tag'

/** Bookmark langsung berubah di layar; dikembalikan bila gagal disimpan. */
export function TombolBookmark({ p, onUbah, berlabel }: { p: Peraturan; onUbah: (aktif: boolean) => void; berlabel?: boolean }) {
  const [sibuk, setSibuk] = useState(false)
  const [error, setError] = useState('')
  async function ubah() {
    const aktif = !p.bookmark
    setSibuk(true)
    setError('')
    onUbah(aktif)
    try {
      await setBookmarkPrimbon(p.uuid, aktif)
    } catch (e) {
      onUbah(!aktif)
      setError((e as Error).message)
    } finally {
      setSibuk(false)
    }
  }
  const Icon = p.bookmark ? BookmarkCheck : Bookmark
  const teks = p.bookmark ? `Hapus ${p.kode} dari bookmark` : `Simpan ${p.kode} ke bookmark`
  const ikon = <Icon className={p.bookmark ? 'text-[var(--text-brand)]' : undefined} />
  return (
    <span className="inline-flex shrink-0 flex-col items-end gap-1">
      {berlabel ? (
        <Button variant="outline" size="sm" onClick={ubah} disabled={sibuk} aria-pressed={p.bookmark}>
          {ikon}
          {p.bookmark ? 'Tersimpan di bookmark' : 'Bookmark'}
        </Button>
      ) : (
        <Button variant="ghost" size="icon" onClick={ubah} disabled={sibuk} aria-pressed={p.bookmark} aria-label={teks} title={teks}>
          {ikon}
        </Button>
      )}
      {error && <span role="alert" className="max-w-[220px] text-xs text-[var(--text-danger)]">{error}</span>}
    </span>
  )
}

/** Teks dengan kata kunci ditandai (aman: dirender sebagai teks, bukan HTML). */
export function Sorot({ teks, q }: { teks: string; q?: string }) {
  if (!q?.trim()) return <>{teks}</>
  return (
    <>
      {pecahSorot(teks, q).map((x, i) => (x.cocok
        ? <mark key={i} className="rounded-sm bg-[color-mix(in_srgb,var(--accent-amber-bright)_32%,transparent)] px-px text-inherit">{x.teks}</mark>
        : <span key={i}>{x.teks}</span>))}
    </>
  )
}

const MAKS_LABEL = 2

/** Status & perubahan (dua tag pertama tagPeraturan) di depan; kategori & tag topik sesudahnya. */
function pisahTag(p: Peraturan) {
  const semua = tagPeraturan(p)
  const status = (p.status?.trim() || 'Status belum diketahui').toLocaleLowerCase('id-ID')
  const perubahan = p.perubahan?.trim().toLocaleLowerCase('id-ID')
  const n = perubahan && perubahan !== status ? 2 : 1
  return { kepala: semua.slice(0, n), topik: semua.slice(n) }
}

/** Satu peraturan di daftar. Seluruh kartu bisa diklik untuk membuka detail; bookmark terpisah. */
export function KartuPeraturan({ p, q, onBuka, onUbah, children }: {
  p: Peraturan
  q?: string
  onBuka: () => void
  onUbah: (p: Peraturan) => void
  children?: ReactNode
}) {
  const { kepala, topik } = pisahTag(p)
  const tampil = topik.slice(0, MAKS_LABEL)
  const lebih = topik.length - tampil.length
  return (
    <article data-kartu={p.uuid}
      className="card relative justify-start px-[18px] py-4 has-[[data-buka]:focus-visible]:ring-3 has-[[data-buka]:focus-visible]:ring-[var(--ring)]">
      <div className="flex items-start gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-1.5 text-xs">
            {kepala.map((t) => (
              <span key={t.label} className={`pill-status inline-flex items-center gap-1.5 text-[11px] font-semibold ${t.warna}`}>
                <span className="size-1.5 rounded-full bg-current" aria-hidden="true" />
                {t.label}
              </span>
            ))}
            <span className="pill-status pill-brand text-[11px]" title={p.jenis_label}>{jenisSingkat(p)}</span>
            {p.tahun ? <span className="tabular text-muted-foreground">{p.tahun}</span> : null}
          </div>
          <h3 className="mt-2">
            {/* tombol "melebar" ke seluruh kartu (after:inset-0); kontrol lain di atasnya (z-10) */}
            <button type="button" data-buka onClick={onBuka}
              className="block w-full cursor-pointer text-left outline-none after:absolute after:inset-0 after:rounded-[16px] after:content-[''] hover:[&_.judul]:underline">
              <span className="block font-mono text-[13px] font-semibold text-[var(--text-brand)]"><Sorot teks={p.kode} q={q} /></span>
              <span className="judul mt-0.5 line-clamp-2 text-base leading-snug font-semibold underline-offset-2" title={p.judul}>
                <Sorot teks={p.judul} q={q} />
              </span>
            </button>
          </h3>
          <div className="mt-2.5 flex flex-wrap items-center gap-x-3 gap-y-1.5 text-xs text-muted-foreground">
            {tampil.length > 0 && (
              <span className="flex flex-wrap items-center gap-1.5" title={topik.map((t) => t.label).join(', ')}>
                {tampil.map((t) => <span key={t.label} className="pill-status pill-status-muted text-[11px] font-semibold">{t.label}</span>)}
                {lebih > 0 && <span className="text-[11px] font-semibold">+{lebih}</span>}
              </span>
            )}
            <span className="inline-flex items-center gap-1">
              <FileText size={13} aria-hidden="true" />
              {keteranganNaskah(p)}
            </span>
            {p.ada_catatan && (
              <span className="inline-flex items-center gap-1">
                <StickyNote size={13} aria-hidden="true" />
                Ada catatan
              </span>
            )}
          </div>
          {children}
        </div>
        <div className="relative z-10 -mt-1 -mr-1">
          <TombolBookmark p={p} onUbah={(bookmark) => onUbah({ ...p, bookmark })} />
        </div>
      </div>
    </article>
  )
}

/** Kerangka kartu selagi daftar pertama kali dimuat. */
export function KerangkaKartu() {
  const balok = 'rounded-md bg-[color-mix(in_srgb,var(--text-primary)_8%,transparent)] motion-safe:animate-pulse'
  return (
    <div className="card justify-start gap-2.5 px-[18px] py-4" aria-hidden="true">
      <div className="flex gap-1.5"><div className={balok + ' h-5 w-16'} /><div className={balok + ' h-5 w-12'} /></div>
      <div className={balok + ' h-3.5 w-40'} />
      <div className={balok + ' h-4 w-full max-w-[560px]'} />
      <div className={balok + ' h-3 w-56'} />
    </div>
  )
}
