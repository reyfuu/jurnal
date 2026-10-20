import { useCallback, useEffect, useMemo, useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { aturPengingat, getCatatanMaster, type CatatanMasterRow, type Pengingat, type TingkatCatatan, type View } from '@/lib/api'
import { TINGKAT } from '@/lib/catatan'
import { formatRelatif, formatTanggal } from '@/lib/format'
import { VIEW_LABEL } from '@/lib/nav'

// Catatan konsultan semua klien yang boleh dilihat + aturan pengingat (berlaku untuk semua klien).

type Saring = 'terbuka' | 'selesai' | 'semua'

export function CatatanMaster({ bisaAtur }: { bisaAtur: boolean }) {
  const [rows, setRows] = useState<CatatanMasterRow[] | null>(null)
  const [aturan, setAturan] = useState<Pengingat | null>(null)
  const [saring, setSaring] = useState<Saring>('terbuka')
  const [q, setQ] = useState('')
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [busy, setBusy] = useState(false)
  const muat = useCallback(() => getCatatanMaster().then((r) => { setRows(r.catatan); setAturan(r.pengingat) }, (e: Error) => setError(e.message)), [])
  useEffect(() => { void muat() }, [muat])
  const tampil = useMemo(() => (rows ?? []).filter((c) => (saring === 'semua' || c.status === saring) &&
    [c.klien_nama, c.isi, c.oleh].join(' ').toLowerCase().includes(q.trim().toLowerCase())), [rows, saring, q])

  async function simpan() {
    if (!aturan) return
    setBusy(true)
    setError('')
    setInfo('')
    try {
      setAturan(await aturPengingat(aturan))
      setInfo('Pengingat disimpan; berlaku untuk semua klien.')
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <PageHeader title="Catatan Konsultan" subtitle="Catatan tim pada laporan klien dan tindak lanjutnya. Tulis catatan dari mode Lihat sebagai klien." />
      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', whiteSpace: 'normal' }}>{error}</div>}
      {info && <div className="pill-status pill-green" role="status" style={{ alignSelf: 'flex-start' }}>{info}</div>}

      {aturan && (
        <div className="card" style={{ padding: 20 }}>
          <div className="chart-title">Pengingat untuk klien</div>
          <div className="chart-subtitle" style={{ marginBottom: 12 }}>Berlaku untuk semua klien. Catatan yang belum ditandai sudah diperbaiki muncul lagi sebagai pop-up pengingat.</div>
          <div className="flex flex-wrap items-end gap-5">
            <label className="flex items-center gap-2 text-sm">
              <Switch checked={aturan.aktif} disabled={!bisaAtur} onCheckedChange={(v) => setAturan({ ...aturan, aktif: v })} /> Pengingat aktif
            </label>
            <label className="grid gap-1 text-sm">
              Ingatkan lagi setiap
              <span className="flex items-center gap-2">
                <Input type="number" min={1} max={30} style={{ width: 80 }} disabled={!bisaAtur || !aturan.aktif} value={aturan.interval_hari}
                  onChange={(e) => setAturan({ ...aturan, interval_hari: Number(e.target.value) })} /> hari
              </span>
            </label>
            <label className="grid gap-1 text-sm">
              Mulai tingkat
              <select className="input-field" disabled={!bisaAtur || !aturan.aktif} value={aturan.tingkat_min}
                onChange={(e) => setAturan({ ...aturan, tingkat_min: e.target.value as TingkatCatatan })}>
                {(Object.keys(TINGKAT) as TingkatCatatan[]).map((t) => <option key={t} value={t}>{TINGKAT[t].label} ke atas</option>)}
              </select>
            </label>
            <div className="grid gap-1 text-sm">
              Kanal
              <span className="flex flex-wrap gap-3">
                <label className="flex items-center gap-1.5"><input type="checkbox" checked readOnly disabled /> Di aplikasi</label>
                <label className="flex items-center gap-1.5 text-muted-foreground"><input type="checkbox" disabled /> WhatsApp (segera)</label>
                <label className="flex items-center gap-1.5 text-muted-foreground"><input type="checkbox" disabled /> Email (segera)</label>
              </span>
            </div>
            {bisaAtur && <Button disabled={busy} onClick={() => void simpan()}>{busy ? 'Menyimpan…' : 'Simpan pengingat'}</Button>}
          </div>
        </div>
      )}

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
