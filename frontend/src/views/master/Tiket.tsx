import { useEffect, useState, type FormEvent } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { listTiket, tiketScreenshotUrl, ubahTiket, type TiketRow, type TiketStatus } from '@/lib/api'
import { formatRelatif } from '@/lib/format'
import { STATUS_LABEL } from '@/lib/tiket'

const KOLOM: TiketStatus[] = ['baru', 'dikerjakan', 'selesai']

export function TiketBoard() {
  const [rows, setRows] = useState<TiketRow[] | null>(null)
  const [error, setError] = useState('')
  const [buka, setBuka] = useState<TiketRow | null>(null)
  const muat = () => listTiket().then(setRows, (e: Error) => setError(e.message))
  useEffect(() => { void muat() }, [])
  return (
    <>
      <PageHeader title="Tiket · Lapor Bug" subtitle="Laporan dari klien yang bisa Anda lihat. Balasan Anda terbaca pelapor di menu Lapor Bug." />
      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start' }}>{error}</div>}
      <div className="table-scroll">
        <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(3, minmax(250px, 1fr))' }}>
          {KOLOM.map((s) => {
            const isi = (rows ?? []).filter((t) => t.status === s)
            return (
              <div key={s} className="card" style={{ padding: 12, display: 'flex', flexDirection: 'column', gap: 8 }}>
                <div className="flex items-center justify-between kpi-label" style={{ padding: '2px 4px' }}>
                  <span>{STATUS_LABEL[s]}</span><span className="tabular">{isi.length}</span>
                </div>
                {isi.map((t) => (
                  <button key={t.id} onClick={() => setBuka(t)} className="rounded-lg border border-border text-left hover:border-ring" style={{ padding: '10px 12px', background: 'var(--bg-card-hover)' }}>
                    <div className="tabular text-xs text-muted-foreground">{t.nomor} · {formatRelatif(t.waktu)}</div>
                    <div className="text-sm font-semibold">{t.judul}</div>
                    <div className="text-xs text-muted-foreground">{t.klien_nama}{t.balasan?.length ? ` · ${t.balasan.length} balasan` : ''}</div>
                  </button>
                ))}
                {rows && isi.length === 0 && <p className="text-xs text-muted-foreground" style={{ padding: 8 }}>Kosong</p>}
              </div>
            )
          })}
        </div>
      </div>
      {buka && <TiketDialog t={buka} onClose={() => setBuka(null)} onSaved={() => { setBuka(null); void muat() }} />}
    </>
  )
}

function TiketDialog({ t, onClose, onSaved }: { t: TiketRow; onClose: () => void; onSaved: () => void }) {
  const [status, setStatus] = useState<TiketStatus>(t.status)
  const [balasan, setBalasan] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  async function simpan(e: FormEvent) {
    e.preventDefault()
    if (status === t.status && !balasan.trim()) return onClose()
    setBusy(true)
    setError('')
    try {
      await ubahTiket(t.id, { status: status === t.status ? undefined : status, balasan: balasan.trim() || undefined })
      onSaved()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-2xl">
        <form onSubmit={simpan} className="grid gap-4">
          <DialogHeader>
            <DialogTitle>{t.nomor} · {t.judul}</DialogTitle>
            <DialogDescription>{t.klien_nama} · {t.user_email}{t.halaman && ` · menu ${t.halaman}`} · {new Date(t.waktu).toLocaleString('id-ID')}</DialogDescription>
          </DialogHeader>
          <div className="grid gap-2" style={{ maxHeight: '45vh', overflowY: 'auto' }}>
            <div className="rounded-lg border border-border px-3 py-2 text-sm">
              <div className="text-xs text-muted-foreground">Laporan klien</div>
              <p className="whitespace-pre-wrap">{t.deskripsi}</p>
              {t.screenshot && (
                <a href={tiketScreenshotUrl(t.id)} target="_blank" rel="noreferrer" className="mt-2 inline-block">
                  <img src={tiketScreenshotUrl(t.id)} alt={`Screenshot ${t.nomor}`} className="max-h-40 rounded-md border border-border" />
                </a>
              )}
            </div>
            {t.balasan?.map((b, i) => (
              <div key={i} className="rounded-lg px-3 py-2 text-sm" style={{ background: 'color-mix(in srgb, var(--accent-blue) 12%, transparent)' }}>
                <div className="text-xs text-muted-foreground">{b.nama} · {new Date(b.waktu).toLocaleString('id-ID')}</div>
                <p className="whitespace-pre-wrap">{b.isi}</p>
              </div>
            ))}
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="tk-balasan">Balasan untuk klien</Label>
            <textarea id="tk-balasan" rows={3} maxLength={2000} value={balasan} onChange={(e) => setBalasan(e.target.value)}
              placeholder="Tulis dengan bahasa pemilik usaha. Terlihat di menu Lapor Bug klien."
              className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 dark:bg-input/30" />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="tk-status">Status</Label>
            <Select value={status} onValueChange={(v) => setStatus(v as TiketStatus)}>
              <SelectTrigger id="tk-status" className="w-full"><SelectValue /></SelectTrigger>
              <SelectContent>{KOLOM.map((s) => <SelectItem key={s} value={s}>{STATUS_LABEL[s]}</SelectItem>)}</SelectContent>
            </Select>
          </div>
          {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>Tutup</Button>
            <Button type="submit" disabled={busy}>{busy ? 'Menyimpan…' : 'Simpan'}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
