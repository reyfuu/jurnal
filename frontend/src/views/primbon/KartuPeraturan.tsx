import { useState, type ReactNode } from 'react'
import { Bookmark, BookmarkCheck, FileText, StickyNote } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { setBookmarkPrimbon, type Peraturan } from '@/lib/api'
import { jenisSingkat, keteranganNaskah, MONO, pillStatus } from './bantu'

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
  const teks = p.bookmark ? 'Hapus dari bookmark' : 'Simpan ke bookmark'
  return (
    <span className="inline-flex shrink-0 flex-col items-end gap-1">
      {berlabel ? (
        <Button variant="outline" size="sm" onClick={ubah} disabled={sibuk} aria-pressed={p.bookmark}>
          <Icon style={p.bookmark ? { color: 'var(--text-brand)' } : undefined} />
          {p.bookmark ? 'Tersimpan di bookmark' : 'Bookmark'}
        </Button>
      ) : (
        <Button variant="ghost" size="icon-sm" onClick={ubah} disabled={sibuk} aria-pressed={p.bookmark} aria-label={teks} title={teks}>
          <Icon style={p.bookmark ? { color: 'var(--text-brand)' } : undefined} />
        </Button>
      )}
      {error && <span role="alert" className="text-xs" style={{ color: 'var(--text-danger)', maxWidth: 220 }}>{error}</span>}
    </span>
  )
}

/** Satu peraturan di daftar: jenis, status, tahun, nomor — judul, kategori, ketersediaan naskah. */
export function KartuPeraturan({ p, onBuka, onUbah, children }: {
  p: Peraturan
  onBuka: () => void
  onUbah: (p: Peraturan) => void
  children?: ReactNode
}) {
  return (
    <article className="card" style={{ padding: '16px 18px', justifyContent: 'flex-start' }}>
      <div className="flex items-start gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-1.5 text-xs">
            <span className="pill-status pill-brand" style={{ fontSize: 11 }} title={p.jenis_label}>{jenisSingkat(p)}</span>
            {p.status && <span className={'pill-status ' + pillStatus(p.status)} style={{ fontSize: 11 }}>{p.status}</span>}
            {p.tahun ? <span className="tabular text-muted-foreground">· {p.tahun}</span> : null}
          </div>
          <h3 className="mt-2 text-base leading-snug font-semibold">
            <button type="button" className="cursor-pointer text-left underline-offset-2 hover:underline" onClick={onBuka}>
              <span style={{ ...MONO, fontSize: '0.92em' }}>{p.kode}</span>
              <span className="text-muted-foreground"> — </span>
              {p.judul}
            </button>
          </h3>
          <div className="mt-2.5 flex flex-wrap items-center gap-1.5 text-xs">
            {p.kategori_label.map((k) => (
              <span key={k} className="pill-status pill-status-muted" style={{ fontSize: 11, fontWeight: 600 }}>{k}</span>
            ))}
            <span className="inline-flex items-center gap-1 text-muted-foreground">
              <FileText size={13} aria-hidden="true" />
              {keteranganNaskah(p)}
            </span>
            {p.ada_catatan && (
              <span className="inline-flex items-center gap-1 text-muted-foreground">
                <StickyNote size={13} aria-hidden="true" />
                Ada catatan
              </span>
            )}
          </div>
          {children}
        </div>
        <TombolBookmark p={p} onUbah={(bookmark) => onUbah({ ...p, bookmark })} />
      </div>
    </article>
  )
}
