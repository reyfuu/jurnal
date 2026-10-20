import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Pencil, Plus, Trash2 } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { aturJadwal, aturPengingat, getCatatanMaster, getJadwalMaster, type JadwalPengingat, type Pengingat, type TingkatCatatan, type UntukJadwal } from '@/lib/api'
import { TINGKAT } from '@/lib/catatan'

// Semua pengingat yang berlaku untuk seluruh klien: jadwal tenggat laporan & pajak, dan
// pengingat catatan konsultan yang belum diperbaiki. Hanya izin `pengaturan` yang bisa mengubah.

const BULAN = ['Januari', 'Februari', 'Maret', 'April', 'Mei', 'Juni', 'Juli', 'Agustus', 'September', 'Oktober', 'November', 'Desember']
const UNTUK: Record<UntukJadwal, string> = { semua: 'Semua klien', op: 'Orang pribadi', badan: 'Badan (CV, PT, dll.)', pkp: 'PKP (wajib PPN)' }

const kapan = (j: JadwalPengingat) =>
  j.frekuensi === 'bulanan' ? `Setiap tanggal ${j.tanggal}${j.tanggal >= 29 ? ' (atau hari terakhir bulan)' : ''}` : `Setiap ${j.tanggal} ${BULAN[(j.bulan ?? 1) - 1]}`

export function PengingatMaster({ bisaAtur }: { bisaAtur: boolean }) {
  const [jadwal, setJadwal] = useState<JadwalPengingat[] | null>(null)
  const [edit, setEdit] = useState<JadwalPengingat | null>(null)
  const [error, setError] = useState('')
  const muat = useCallback(() => getJadwalMaster().then((r) => setJadwal(r.item), (e: Error) => setError(e.message)), [])
  useEffect(() => { void muat() }, [muat])

  async function simpan(daftar: JadwalPengingat[]) {
    const r = await aturJadwal(daftar)
    setJadwal(r.item)
  }
  const jalankan = (f: () => Promise<void>) => { setError(''); void f().catch((e: Error) => setError(e.message)) }

  return (
    <>
      <PageHeader title="Pengingat" subtitle="Tenggat laporan & pajak serta pengingat catatan konsultan. Berlaku untuk semua klien dan tampil di aplikasi mereka.">
        {bisaAtur && (
          <button className="btn-cta inline-flex items-center gap-1.5"
            onClick={() => setEdit({ id: 0, judul: '', keterangan: '', frekuensi: 'bulanan', tanggal: 15, hari_sebelum: 3, untuk: 'semua', aktif: true })}>
            <Plus size={16} /> Tambah jadwal
          </button>
        )}
      </PageHeader>
      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', whiteSpace: 'normal' }}>{error}</div>}

      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Jadwal laporan & pajak</div>
        <div className="chart-subtitle" style={{ marginBottom: 12 }}>
          Klien melihat banner mulai H-n sebelum jatuh tempo, sesuai bentuk usaha & status PKP di profil mereka. Kanal: di aplikasi (WhatsApp & email menyusul).
        </div>
        <div className="table-scroll">
          <table className="data-table" style={{ minWidth: 860 }}>
            <thead><tr><th>Aktif</th><th>Jadwal</th><th>Jatuh tempo</th><th>Mulai diingatkan</th><th>Berlaku untuk</th>{bisaAtur && <th style={{ textAlign: 'right' }}>Aksi</th>}</tr></thead>
            <tbody>
              {(jadwal ?? []).map((j) => (
                <tr key={j.id} style={{ opacity: j.aktif ? 1 : 0.55 }}>
                  <td>
                    <Switch checked={j.aktif} disabled={!bisaAtur} aria-label={`Aktifkan ${j.judul}`}
                      onCheckedChange={(v) => jalankan(() => simpan(jadwal!.map((x) => x.id === j.id ? { ...x, aktif: v } : x)))} />
                  </td>
                  <td style={{ whiteSpace: 'normal', maxWidth: 340 }}>
                    <div className="font-semibold">{j.judul}
                      {j.perlu_verifikasi && <span className="pill-status pill-amber" style={{ marginLeft: 6, fontSize: 10 }} title="Tenggat bawaan berdasarkan aturan pajak; cocokkan dengan aturan terbaru lalu tandai sudah dicek">perlu dicek tim pajak</span>}
                    </div>
                    {j.keterangan && <div className="text-xs text-muted-foreground">{j.keterangan}</div>}
                  </td>
                  <td className="text-sm">{kapan(j)}</td>
                  <td className="text-sm">{j.hari_sebelum === 0 ? 'Hari H saja' : `H-${j.hari_sebelum}`}</td>
                  <td className="text-sm">{UNTUK[j.untuk]}</td>
                  {bisaAtur && (
                    <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
                      <Button variant="ghost" size="icon-sm" aria-label={`Ubah ${j.judul}`} onClick={() => setEdit(j)}><Pencil /></Button>
                      <Button variant="ghost" size="icon-sm" aria-label={`Hapus ${j.judul}`}
                        onClick={() => window.confirm(`Hapus jadwal "${j.judul}"?`) && jalankan(() => simpan(jadwal!.filter((x) => x.id !== j.id)))}><Trash2 /></Button>
                    </td>
                  )}
                </tr>
              ))}
              {jadwal?.length === 0 && <tr><td colSpan={6} className="text-center text-muted-foreground" style={{ padding: 24 }}>Belum ada jadwal.</td></tr>}
            </tbody>
          </table>
        </div>
      </div>

      <PengingatCatatanCard bisaAtur={bisaAtur} />

      {edit && (
        <JadwalDialog awal={edit} onClose={() => setEdit(null)}
          onSimpan={async (j) => {
            await simpan(j.id ? jadwal!.map((x) => x.id === j.id ? j : x) : [...jadwal!, j])
            setEdit(null)
          }} />
      )}
    </>
  )
}

function JadwalDialog({ awal, onClose, onSimpan }: { awal: JadwalPengingat; onClose: () => void; onSimpan: (j: JadwalPengingat) => Promise<void> }) {
  const [j, setJ] = useState(awal)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const set = (p: Partial<JadwalPengingat>) => setJ((x) => ({ ...x, ...p }))
  async function kirim(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await onSimpan({ ...j, bulan: j.frekuensi === 'tahunan' ? j.bulan ?? 1 : undefined })
    } catch (err) {
      setError((err as Error).message)
      setBusy(false)
    }
  }
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <form onSubmit={kirim} className="grid gap-3">
          <DialogHeader>
            <DialogTitle>{awal.id ? 'Ubah jadwal' : 'Tambah jadwal'}</DialogTitle>
            <DialogDescription>Berlaku untuk semua klien yang cocok dengan pilihan "Berlaku untuk".</DialogDescription>
          </DialogHeader>
          <div className="grid gap-1.5"><Label htmlFor="jd-judul">Judul</Label>
            <Input id="jd-judul" required maxLength={80} placeholder="mis. Setor PPh Final UMKM" value={j.judul} onChange={(e) => set({ judul: e.target.value })} /></div>
          <div className="grid gap-1.5"><Label htmlFor="jd-ket">Keterangan untuk klien</Label>
            <Input id="jd-ket" maxLength={300} value={j.keterangan ?? ''} onChange={(e) => set({ keterangan: e.target.value })} /></div>
          <div className="grid grid-cols-2 gap-3">
            <div className="grid gap-1.5"><Label htmlFor="jd-frek">Frekuensi</Label>
              <select id="jd-frek" className="input-field" value={j.frekuensi} onChange={(e) => set({ frekuensi: e.target.value as JadwalPengingat['frekuensi'], bulan: e.target.value === 'tahunan' ? j.bulan ?? 1 : undefined })}>
                <option value="bulanan">Bulanan</option><option value="tahunan">Tahunan</option>
              </select></div>
            <div className="grid gap-1.5"><Label htmlFor="jd-tgl">Tanggal jatuh tempo</Label>
              <Input id="jd-tgl" type="number" min={1} max={31} required value={j.tanggal} onChange={(e) => set({ tanggal: Number(e.target.value) })} /></div>
            {j.frekuensi === 'tahunan' && (
              <div className="grid gap-1.5"><Label htmlFor="jd-bln">Bulan</Label>
                <select id="jd-bln" className="input-field" value={j.bulan ?? 1} onChange={(e) => set({ bulan: Number(e.target.value) })}>
                  {BULAN.map((b, i) => <option key={b} value={i + 1}>{b}</option>)}
                </select></div>
            )}
            <div className="grid gap-1.5"><Label htmlFor="jd-h">Mulai diingatkan (hari sebelum)</Label>
              <Input id="jd-h" type="number" min={0} max={60} required value={j.hari_sebelum} onChange={(e) => set({ hari_sebelum: Number(e.target.value) })} /></div>
            <div className="grid gap-1.5"><Label htmlFor="jd-untuk">Berlaku untuk</Label>
              <select id="jd-untuk" className="input-field" value={j.untuk} onChange={(e) => set({ untuk: e.target.value as UntukJadwal })}>
                {(Object.keys(UNTUK) as UntukJadwal[]).map((u) => <option key={u} value={u}>{UNTUK[u]}</option>)}
              </select></div>
          </div>
          {j.frekuensi === 'bulanan' && j.tanggal >= 29 && <p className="text-xs text-muted-foreground">Bulan yang lebih pendek memakai hari terakhirnya.</p>}
          {awal.perlu_verifikasi && (
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={!j.perlu_verifikasi} onChange={(e) => set({ perlu_verifikasi: !e.target.checked })} />
              Tanggal sudah dicek tim pajak sesuai aturan terbaru
            </label>
          )}
          <label className="flex items-center gap-3 text-sm"><Switch checked={j.aktif} onCheckedChange={(v) => set({ aktif: v })} /> Aktif</label>
          {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>Batal</Button>
            <Button type="submit" disabled={busy}>{busy ? 'Menyimpan…' : 'Simpan'}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

/** Pengingat untuk catatan konsultan yang belum ditandai sudah diperbaiki klien. */
function PengingatCatatanCard({ bisaAtur }: { bisaAtur: boolean }) {
  const [aturan, setAturan] = useState<Pengingat | null>(null)
  const [pesan, setPesan] = useState<{ ok: boolean; teks: string } | null>(null)
  const [busy, setBusy] = useState(false)
  useEffect(() => { getCatatanMaster().then((r) => setAturan(r.pengingat), (e: Error) => setPesan({ ok: false, teks: e.message })) }, [])
  if (!aturan) return pesan && <div className="pill-status pill-red" role="alert">{pesan.teks}</div>

  async function simpan() {
    setBusy(true)
    setPesan(null)
    try {
      setAturan(await aturPengingat(aturan!))
      setPesan({ ok: true, teks: 'Pengingat catatan disimpan; berlaku untuk semua klien.' })
    } catch (e) {
      setPesan({ ok: false, teks: (e as Error).message })
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="card" style={{ padding: 20 }}>
      <div className="chart-title">Pengingat catatan konsultan</div>
      <div className="chart-subtitle" style={{ marginBottom: 12 }}>Catatan yang belum ditandai sudah diperbaiki muncul lagi sebagai pop-up di aplikasi klien.</div>
      {pesan && <div className={'pill-status ' + (pesan.ok ? 'pill-green' : 'pill-red')} role="status" style={{ marginBottom: 10, display: 'inline-block' }}>{pesan.teks}</div>}
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
  )
}
