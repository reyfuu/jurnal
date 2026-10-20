import { useCallback, useEffect, useMemo, useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { Input } from '@/components/ui/input'
import { getCatatanMaster, type CatatanMasterRow, type View } from '@/lib/api'
import { TINGKAT } from '@/lib/catatan'
import { formatRelatif, formatTanggal } from '@/lib/format'
import { VIEW_LABEL } from '@/lib/nav'

// Catatan konsultan semua klien yang boleh dilihat. Aturan pengingatnya ada di halaman Pengingat.

type Saring = 'terbuka' | 'selesai' | 'semua'

export function CatatanMaster() {
  const [rows, setRows] = useState<CatatanMasterRow[] | null>(null)
  const [saring, setSaring] = useState<Saring>('terbuka')
  const [q, setQ] = useState('')
  const [error, setError] = useState('')
  const muat = useCallback(() => getCatatanMaster().then((r) => setRows(r.catatan), (e: Error) => setError(e.message)), [])
  useEffect(() => { void muat() }, [muat])
  const tampil = useMemo(() => (rows ?? []).filter((c) => (saring === 'semua' || c.status === saring) &&
    [c.klien_nama, c.isi, c.oleh].join(' ').toLowerCase().includes(q.trim().toLowerCase())), [rows, saring, q])

  return (
    <>
      <PageHeader title="Catatan Konsultan" subtitle="Catatan tim pada laporan klien dan tindak lanjutnya. Tulis catatan dari mode Lihat sebagai klien; frekuensi pengingatnya diatur di menu Pengingat." />
      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', whiteSpace: 'normal' }}>{error}</div>}

      <div className="card" style={{ padding: 20 }}>
        <div className="flex flex-wrap items-center gap-3" style={{ marginBottom: 12 }}>
          <div className="tabbar" role="tablist" aria-label="Saring status">
            {(['terbuka', 'selesai', 'semua'] as Saring[]).map((s) => (
              <button key={s} role="tab" className="tab-btn" aria-selected={saring === s} onClick={() => setSaring(s)}>
                {s === 'terbuka' ? 'Belum diperbaiki' : s === 'selesai' ? 'Sudah diperbaiki' : 'Semua'}
              </button>
            ))}
          </div>
          <Input aria-label="Cari catatan" placeholder="Cari klien, isi, penulis" value={q} onChange={(e) => setQ(e.target.value)} style={{ maxWidth: 260, marginLeft: 'auto' }} />
        </div>
        <div className="table-scroll">
          <table className="data-table" style={{ minWidth: 900 }}>
            <thead><tr><th>Klien</th><th>Halaman</th><th>Tingkat</th><th>Catatan</th><th>Oleh</th><th>Status</th></tr></thead>
            <tbody>
              {tampil.map((c) => (
                <tr key={`${c.klien_id}-${c.id}`}>
                  <td className="font-semibold">{c.klien_nama}</td>
                  <td className="text-sm">{VIEW_LABEL[c.halaman as View] ?? c.halaman}{c.periode && <div className="text-xs text-muted-foreground">{c.periode}</div>}</td>
                  <td><span className={'pill-status ' + TINGKAT[c.tingkat].pill}>{TINGKAT[c.tingkat].label}</span></td>
                  <td className="text-sm" style={{ whiteSpace: 'normal', maxWidth: 360 }}>{c.isi}</td>
                  <td className="text-sm">{c.oleh}<div className="text-xs text-muted-foreground">{formatTanggal(c.dibuat)}</div></td>
                  <td className="text-sm">
                    {c.status === 'selesai'
                      ? <><span className="pill-status pill-green">Sudah diperbaiki</span><div className="text-xs text-muted-foreground">{c.selesai_oleh} · {formatTanggal(c.selesai_pada)}</div></>
                      : <><span className="pill-status pill-amber">Belum</span><div className="text-xs text-muted-foreground">{c.jumlah_pengingat}x diingatkan{c.diingatkan_pada ? ` · ${formatRelatif(c.diingatkan_pada)}` : ''}</div></>}
                  </td>
                </tr>
              ))}
              {rows && tampil.length === 0 && <tr><td colSpan={6} className="text-center text-muted-foreground" style={{ padding: 24 }}>Tidak ada catatan.</td></tr>}
            </tbody>
          </table>
        </div>
      </div>
    </>
  )
}
