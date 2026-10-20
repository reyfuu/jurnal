import { useEffect, useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { KoreksiCard } from '@/components/KoreksiCard'
import { listKoreksiMaster, type KoreksiRow } from '@/lib/api'
import { formatRelatif } from '@/lib/format'

/** Semua usulan koreksi di klien yang bisa dilihat anggota tim. */
export function KoreksiMaster() {
  const [rows, setRows] = useState<KoreksiRow[] | null>(null)
  const [error, setError] = useState('')
  useEffect(() => { listKoreksiMaster().then(setRows, (e: Error) => setError(e.message)) }, [])
  return (
    <>
      <PageHeader title="Usulan Koreksi" subtitle="Koreksi tidak langsung mengubah pembukuan. Angka di laporan klien baru berubah setelah pemilik usaha menyetujui." />
      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start' }}>{error}</div>}
      {rows?.length === 0 && (
        <div className="card text-sm text-muted-foreground" style={{ padding: 20 }}>
          Belum ada usulan. Buka klien, pilih <strong>Lihat sebagai klien</strong>, lalu klik transaksi yang perlu dikoreksi.
        </div>
      )}
      {rows?.map((k) => <KoreksiCard key={`${k.klien_id}-${k.id}`} k={k} sub={`${k.klien_nama} · ${k.oleh} · ${formatRelatif(k.waktu)}`} />)}
    </>
  )
}
