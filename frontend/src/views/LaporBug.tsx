import { useEffect, useState, type ClipboardEvent, type FormEvent } from 'react'
import { ImagePlus, X } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { bugScreenshotUrl, createBug, listBugs, type BugReport } from '@/lib/api'

// LaporBug: laporan disimpan di server (data/bugs.json); tiap usaha melihat laporannya sendiri.
export function LaporBug({ halamanAwal }: { halamanAwal: string }) {
  const [judul, setJudul] = useState('')
  const [deskripsi, setDeskripsi] = useState('')
  const [halaman, setHalaman] = useState(halamanAwal)
  const [shot, setShot] = useState<File | null>(null)
  const [preview, setPreview] = useState('')
  const [list, setList] = useState<BugReport[]>([])
  const [error, setError] = useState('')
  const [sent, setSent] = useState(false)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    listBugs().then(setList, (e: Error) => setError(e.message))
  }, [])

  // pratinjau lokal; URL dilepas saat gambar diganti/dihapus
  useEffect(() => {
    if (!shot) return setPreview('')
    const url = URL.createObjectURL(shot)
    setPreview(url)
    return () => URL.revokeObjectURL(url)
  }, [shot])

  function pickImage(f: File | undefined) {
    if (!f) return
    if (!f.type.startsWith('image/')) return setError('Lampiran harus berupa gambar.')
    if (f.size > 5 << 20) return setError('Ukuran gambar maksimal 5 MB.')
    setError('')
    setShot(f)
  }

  // tempel screenshot langsung dari clipboard (Ctrl+V) di mana pun dalam form
  function onPaste(e: ClipboardEvent) {
    const f = [...e.clipboardData.files].find((x) => x.type.startsWith('image/'))
    if (f) {
      e.preventDefault()
      pickImage(f)
    }
  }

  async function submit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    setSent(false)
    try {
      const b = await createBug({ judul, deskripsi, halaman, screenshot: shot })
      setList((x) => [b, ...x])
      setJudul('')
      setDeskripsi('')
      setShot(null)
      setSent(true)
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <PageHeader title="Lapor Bug" subtitle="Ceritakan masalah yang Anda temui; tim kami akan menindaklanjuti" />
      <form className="card grid gap-4" style={{ padding: 22, maxWidth: 760, justifyContent: 'stretch' }} onSubmit={submit} onPaste={onPaste}>
        <div className="grid gap-1.5">
          <Label htmlFor="bug-judul">Judul singkat</Label>
          <Input id="bug-judul" required maxLength={200} value={judul} onChange={(e) => setJudul(e.target.value)} placeholder="mis. Tombol simpan transaksi tidak merespons" />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="bug-halaman">Menu / halaman</Label>
          <Input id="bug-halaman" value={halaman} onChange={(e) => setHalaman(e.target.value)} placeholder="mis. Transaksi" />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="bug-deskripsi">Apa yang terjadi?</Label>
          <textarea
            id="bug-deskripsi" required maxLength={5000} rows={6} value={deskripsi} onChange={(e) => setDeskripsi(e.target.value)}
            placeholder="Langkah yang Anda lakukan, apa yang diharapkan, dan apa yang terjadi."
            className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 dark:bg-input/30"
          />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="bug-shot">Screenshot / foto (opsional)</Label>
          {preview ? (
            <div className="relative justify-self-start">
              <img src={preview} alt="Pratinjau screenshot" className="max-h-56 rounded-lg border border-border" />
              <Button type="button" variant="secondary" size="icon" className="absolute right-2 top-2 size-8" aria-label="Hapus screenshot" onClick={() => setShot(null)}>
                <X aria-hidden="true" />
              </Button>
            </div>
          ) : (
            <label htmlFor="bug-shot" className="flex cursor-pointer items-center gap-3 rounded-lg border border-dashed border-input px-4 py-5 text-sm text-muted-foreground hover:border-ring">
              <ImagePlus className="size-5 shrink-0" aria-hidden="true" />
              <span>Pilih gambar atau ambil foto. Bisa juga tempel screenshot dengan Ctrl+V. Maks 5 MB.</span>
            </label>
          )}
          <input id="bug-shot" type="file" accept="image/*" className="sr-only" onChange={(e) => { pickImage(e.target.files?.[0]); e.target.value = '' }} />
        </div>
        {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
        {sent && <p className="text-sm" style={{ color: 'var(--accent-green)' }} role="status">Terima kasih, laporan Anda terkirim.</p>}
        <Button type="submit" className="justify-self-start rounded-full px-5 font-bold" disabled={busy}>{busy ? 'Mengirim…' : 'Kirim laporan'}</Button>
      </form>

      {list.length > 0 && (
        <div className="card" style={{ padding: 22, maxWidth: 760, marginTop: 16 }}>
          <div className="chart-title">Laporan terkirim</div>
          <ul className="mt-3 grid gap-3">
            {list.map((b) => (
              <li key={b.id} className="border-b border-border pb-3 last:border-0 last:pb-0">
                <div className="text-sm font-semibold">{b.judul}</div>
                <div className="text-xs text-muted-foreground">{new Date(b.waktu).toLocaleString('id-ID')} · {b.user_email}{b.halaman && ` · ${b.halaman}`}</div>
                <p className="mt-1 whitespace-pre-wrap text-sm text-muted-foreground">{b.deskripsi}</p>
                {b.screenshot && (
                  <a href={bugScreenshotUrl(b.id)} target="_blank" rel="noreferrer" className="mt-2 inline-block">
                    <img src={bugScreenshotUrl(b.id)} alt={`Screenshot: ${b.judul}`} loading="lazy" className="max-h-32 rounded-md border border-border" />
                  </a>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}
    </>
  )
}
