import { useEffect, useState, type FormEvent } from 'react'
import { CheckCircle2, MessageSquarePlus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { PageHeader } from '@/components/PageHeader'
import { catatanDiingatkan, getCatatan, selesaiCatatan, tulisCatatan, type CatatanKonsultan, type TingkatCatatan, type View } from '@/lib/api'
import { TINGKAT } from '@/lib/catatan'
import { formatTanggal } from '@/lib/format'
import { VIEW_LABEL } from '@/lib/nav'

const textarea = 'w-full rounded-md border border-input bg-[var(--bg-input)] px-3 py-2 text-sm'
const halamanLabel = (h: string) => VIEW_LABEL[h as View] ?? h

/** Tombol mengambang saat lihat-sebagai: catatan menempel ke halaman & periode yang dibuka. */
export function TulisCatatanTombol({ klienId, halaman, periode }: { klienId: number; halaman: View; periode: string }) {
  const [buka, setBuka] = useState(false)
  const [tingkat, setTingkat] = useState<TingkatCatatan>('perbaiki')
  const [isi, setIsi] = useState('')
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [busy, setBusy] = useState(false)
  async function kirim(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await tulisCatatan(klienId, { halaman, periode, tingkat, isi })
      setIsi('')
      setBuka(false)
      setInfo('Catatan terkirim ke klien.')
      setTimeout(() => setInfo(''), 3000)
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }
  return (
    <>
      <div style={{ position: 'fixed', right: 24, bottom: 24, zIndex: 40, display: 'grid', gap: 8, justifyItems: 'end' }}>
        {info && <span className="pill-status pill-green" role="status">{info}</span>}
        <Button className="rounded-full shadow-lg" onClick={() => setBuka(true)}><MessageSquarePlus /> Catatan</Button>
      </div>
      <Dialog open={buka} onOpenChange={setBuka}>
        <DialogContent>
          <form onSubmit={kirim} className="grid gap-3">
            <DialogHeader>
              <DialogTitle>Catatan untuk klien</DialogTitle>
              <DialogDescription>Menempel ke halaman <strong>{halamanLabel(halaman)}</strong> periode {periode}. Klien menandai sudah diperbaiki; tercatat atas nama Anda.</DialogDescription>
            </DialogHeader>
            <fieldset className="grid gap-2">
              <legend className="text-sm font-medium">Tingkat</legend>
              {(Object.keys(TINGKAT) as TingkatCatatan[]).map((t) => (
                <label key={t} className="flex cursor-pointer items-start gap-2 text-sm">
                  <input type="radio" name="ct-tingkat" className="mt-1" checked={tingkat === t} onChange={() => setTingkat(t)} />
                  <span><span className="font-semibold">{TINGKAT[t].label}</span> <span className="text-muted-foreground">— {TINGKAT[t].ket}</span></span>
                </label>
              ))}
            </fieldset>
            <label className="grid gap-1.5 text-sm font-medium">
              Isi catatan
              <textarea className={textarea} rows={5} required maxLength={2000} value={isi} onChange={(e) => setIsi(e.target.value)}
                placeholder="mis. Saldo kas Agustus tidak cocok dengan rekening koran; mohon cek transaksi tanggal 15–20." />
            </label>
            {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setBuka(false)}>Batal</Button>
              <Button type="submit" disabled={busy || !isi.trim()}>{busy ? 'Mengirim…' : 'Kirim catatan'}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  )
}

function KartuCatatan({ c, bisaTindak, onSelesai, onBuka }: { c: CatatanKonsultan; bisaTindak: boolean; onSelesai?: () => void; onBuka?: (v: View) => void }) {
  const t = TINGKAT[c.tingkat]
  return (
    <div className="card" style={{ padding: 16, opacity: c.status === 'selesai' ? 0.7 : 1 }}>
      <div className="flex flex-wrap items-center gap-2">
        <span className={'pill-status ' + t.pill}>{t.label}</span>
        <span className="text-sm text-muted-foreground">{halamanLabel(c.halaman)}{c.periode ? ` · ${c.periode}` : ''}</span>
        {c.status === 'selesai' && <span className="pill-status pill-green">Sudah diperbaiki</span>}
      </div>
      <p className="mt-2 text-sm leading-relaxed" style={{ whiteSpace: 'pre-wrap' }}>{c.isi}</p>
      <div className="mt-2 text-xs text-muted-foreground">
        Tim Master Pajak — {c.oleh}{c.peran ? ` (${c.peran})` : ''} · {formatTanggal(c.dibuat)}
        {c.status === 'selesai' && c.selesai_oleh && ` · ditandai selesai oleh ${c.selesai_oleh} ${formatTanggal(c.selesai_pada)}`}
      </div>
      {c.status === 'terbuka' && (
        <div className="mt-3 flex flex-wrap gap-2">
          {onBuka && c.halaman && <Button size="sm" variant="outline" className="rounded-full" onClick={() => onBuka(c.halaman as View)}>Buka {halamanLabel(c.halaman)}</Button>}
          {bisaTindak && onSelesai && <Button size="sm" className="rounded-full" onClick={onSelesai}><CheckCircle2 /> Sudah diperbaiki</Button>}
        </div>
      )}
    </div>
  )
}

/** Halaman klien: semua catatan konsultan, yang terbuka di atas. */
export function CatatanKlienView({ bisaTindak, onBuka, onBerubah }: { bisaTindak: boolean; onBuka: (v: View) => void; onBerubah: () => void }) {
  const [daftar, setDaftar] = useState<CatatanKonsultan[] | null>(null)
  const [error, setError] = useState('')
  const muat = () => getCatatan().then((r) => setDaftar(r.catatan), (e: Error) => setError(e.message))
  useEffect(() => { void muat() }, [])
  const terbuka = (daftar ?? []).filter((c) => c.status === 'terbuka')
  const selesai = (daftar ?? []).filter((c) => c.status === 'selesai')
  async function tandai(id: number) {
    try {
      await selesaiCatatan(id)
      await muat()
      onBerubah()
    } catch (e) {
      setError((e as Error).message)
    }
  }
  return (
    <>
      <PageHeader title="Catatan Konsultan" subtitle="Catatan dari tim Master Pajak tentang laporan Anda. Perbaiki datanya, lalu tandai sudah diperbaiki." />
      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start' }}>{error}</div>}
      {daftar && daftar.length === 0 && <p className="text-sm text-muted-foreground">Belum ada catatan dari konsultan.</p>}
      {terbuka.length > 0 && <div className="chart-title">Perlu ditindaklanjuti ({terbuka.length})</div>}
      {terbuka.map((c) => <KartuCatatan key={c.id} c={c} bisaTindak={bisaTindak} onSelesai={() => void tandai(c.id)} onBuka={onBuka} />)}
      {!bisaTindak && terbuka.length > 0 && <p className="text-xs text-muted-foreground">Hanya Owner atau Admin yang bisa menandai sudah diperbaiki.</p>}
      {selesai.length > 0 && <div className="chart-title" style={{ marginTop: 8 }}>Sudah diperbaiki</div>}
      {selesai.map((c) => <KartuCatatan key={c.id} c={c} bisaTindak={false} />)}
    </>
  )
}

/** Pop-up pengingat: catatan terbuka yang jatuh tempo sesuai aturan Pengingat Panel Master. */
export function PengingatCatatan({ onLihat }: { onLihat: () => void }) {
  const [due, setDue] = useState<CatatanKonsultan[]>([])
  useEffect(() => {
    getCatatan().then((r) => setDue(r.catatan.filter((c) => r.pengingat.includes(c.id))), () => {})
  }, [])
  const tutup = (lanjut: boolean) => {
    void catatanDiingatkan(due.map((c) => c.id)).catch(() => {})
    setDue([])
    if (lanjut) onLihat()
  }
  return (
    <Dialog open={due.length > 0} onOpenChange={(o) => !o && tutup(false)}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Pengingat dari konsultan</DialogTitle>
          <DialogDescription>{due.length} catatan belum ditandai sudah diperbaiki.</DialogDescription>
        </DialogHeader>
        <ul className="grid gap-2" style={{ maxHeight: 320, overflowY: 'auto' }}>
          {due.map((c) => (
            <li key={c.id} className="rounded-md border p-3 text-sm">
              <span className={'pill-status ' + TINGKAT[c.tingkat].pill} style={{ marginRight: 6 }}>{TINGKAT[c.tingkat].label}</span>
              <span className="text-muted-foreground">{halamanLabel(c.halaman)}</span>
              <p className="mt-1">{c.isi}</p>
            </li>
          ))}
        </ul>
        <DialogFooter>
          <Button variant="outline" onClick={() => tutup(false)}>Nanti</Button>
          <Button onClick={() => tutup(true)}>Lihat catatan</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
