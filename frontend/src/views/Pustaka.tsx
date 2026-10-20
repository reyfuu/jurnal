import { useEffect, useMemo, useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { Input } from '@/components/ui/input'
import { listPustaka, type Artikel } from '@/lib/api'
import { formatTanggal } from '@/lib/format'

/** Artikel pajak dari tim Master Pajak yang ditandai untuk klien. */
export function Pustaka() {
  const [rows, setRows] = useState<Artikel[] | null>(null)
  const [error, setError] = useState('')
  const [kat, setKat] = useState('Semua')
  const [q, setQ] = useState('')
  const [buka, setBuka] = useState<number | null>(null)
  useEffect(() => { listPustaka().then(setRows, (e: Error) => setError(e.message)) }, [])
  const kategori = useMemo(() => ['Semua', ...new Set((rows ?? []).map((a) => a.kategori))], [rows])
  const tampil = (rows ?? []).filter((a) => (kat === 'Semua' || a.kategori === kat)
    && [a.judul, a.isi, a.dasar_hukum, ...a.tag].join(' ').toLowerCase().includes(q.trim().toLowerCase()))
  return (
    <>
      <PageHeader title="Pustaka Pajak" subtitle="Penjelasan aturan pajak dan pembukuan dari tim Master Pajak, ditulis untuk pemilik usaha." />
      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start' }}>{error}</div>}
      <div className="flex flex-wrap items-center gap-3">
        <div className="tabbar" role="tablist" aria-label="Kategori">
          {kategori.map((k) => <button key={k} role="tab" className="tab-btn" aria-selected={kat === k} onClick={() => setKat(k)}>{k}</button>)}
        </div>
        <Input aria-label="Cari artikel" placeholder="Cari judul, PP/PMK, kata kunci" value={q} onChange={(e) => setQ(e.target.value)} style={{ maxWidth: 280, marginLeft: 'auto' }} />
      </div>
      {rows && tampil.length === 0 && <div className="card text-sm text-muted-foreground" style={{ padding: 20 }}>Belum ada artikel{q || kat !== 'Semua' ? ' yang cocok' : ''}.</div>}
      {tampil.map((a) => (
        <article key={a.id} className="card" style={{ padding: 20 }}>
          <button className="text-left" style={{ width: '100%' }} onClick={() => setBuka(buka === a.id ? null : a.id)} aria-expanded={buka === a.id}>
            <div className="text-base font-semibold">{a.judul}</div>
            <div className="mt-1 flex flex-wrap gap-1.5 text-xs">
              {a.dasar_hukum && <span className="pill-status pill-amber">{a.dasar_hukum}</span>}
              <span className="pill-status pill-status-muted">{a.kategori}</span>
              {a.berlaku_sejak && <span className="text-muted-foreground">berlaku sejak {a.berlaku_sejak}</span>}
              <span className="text-muted-foreground">· diperbarui {formatTanggal(a.waktu)}</span>
            </div>
          </button>
          {buka === a.id && <p className="mt-3 whitespace-pre-wrap text-sm" style={{ maxWidth: 720, lineHeight: 1.65 }}>{a.isi}</p>}
        </article>
      ))}
    </>
  )
}
