import { useEffect, useState, type FormEvent } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { createBug, listBugs, type BugReport } from '@/lib/api'

// LaporBug: laporan disimpan di server (data/bugs.json); tiap usaha melihat laporannya sendiri.
export function LaporBug({ halamanAwal }: { halamanAwal: string }) {
  const [judul, setJudul] = useState('')
  const [deskripsi, setDeskripsi] = useState('')
  const [halaman, setHalaman] = useState(halamanAwal)
  const [list, setList] = useState<BugReport[]>([])
  const [error, setError] = useState('')
  const [sent, setSent] = useState(false)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    listBugs().then(setList, (e: Error) => setError(e.message))
  }, [])

  async function submit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    setSent(false)
    try {
      const b = await createBug({ judul, deskripsi, halaman })
      setList((x) => [b, ...x])
      setJudul('')
      setDeskripsi('')
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
      <form className="card grid gap-4" style={{ padding: 22, maxWidth: 760, justifyContent: 'stretch' }} onSubmit={submit}>
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
              </li>
            ))}
          </ul>
        </div>
      )}
    </>
  )
}
