import { useEffect, useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { KoreksiCard } from '@/components/KoreksiCard'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { listKoreksi, putuskanKoreksi, type KoreksiRow } from '@/lib/api'
import { formatRelatif } from '@/lib/format'

/** Pemilik usaha meninjau usulan koreksi dari konsultan MasterPajak. Pembukuan baru berubah setelah disetujui. */
export function UsulanKoreksi({ onChanged }: { onChanged: () => void }) {
  const [rows, setRows] = useState<KoreksiRow[] | null>(null)
  const [catatan, setCatatan] = useState<Record<number, string>>({})
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(0)
  const muat = () => listKoreksi().then(setRows, (e: Error) => setError(e.message))
  useEffect(() => { void muat() }, [])

  async function putuskan(k: KoreksiRow, setuju: boolean) {
    setBusy(k.id)
    setError('')
    try {
      await putuskanKoreksi(k.id, setuju, catatan[k.id] ?? '')
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(0)
      await muat()
      onChanged()
    }
  }

  const menunggu = rows?.filter((k) => k.status === 'menunggu') ?? []
  const riwayat = rows?.filter((k) => k.status !== 'menunggu') ?? []
  return (
    <>
      <PageHeader title="Usulan Koreksi" subtitle="Dari konsultan MasterPajak yang menangani usaha Anda. Pembukuan tidak berubah sampai Anda menyetujui." />
      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start' }}>{error}</div>}
      {rows && menunggu.length === 0 && <div className="card text-sm text-muted-foreground" style={{ padding: 20 }}>Tidak ada usulan yang menunggu keputusan Anda.</div>}
      {menunggu.map((k) => (
        <KoreksiCard key={k.id} k={k} sub={`dari ${k.oleh} · ${formatRelatif(k.waktu)}`} aksi={
          <div className="flex flex-wrap items-center gap-2">
            <Input aria-label={`Catatan untuk ${k.nomor}`} placeholder="Catatan untuk konsultan (opsional)" style={{ flex: 1, minWidth: 220 }}
              value={catatan[k.id] ?? ''} onChange={(e) => setCatatan((c) => ({ ...c, [k.id]: e.target.value }))} />
            <Button variant="outline" className="text-destructive hover:text-destructive" disabled={busy === k.id} onClick={() => void putuskan(k, false)}>Tolak</Button>
            <Button disabled={busy === k.id} onClick={() => void putuskan(k, true)}>Setujui &amp; terapkan</Button>
          </div>
        } />
      ))}
      {riwayat.length > 0 && (
        <>
          <div className="chart-title" style={{ marginTop: 8 }}>Riwayat</div>
          {riwayat.map((k) => <KoreksiCard key={k.id} k={k} sub={`dari ${k.oleh} · ${formatRelatif(k.waktu)}`} />)}
        </>
      )}
    </>
  )
}
