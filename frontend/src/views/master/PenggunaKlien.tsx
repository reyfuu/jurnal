import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react'
import { KeyRound, Pencil, Trash2, UserCheck, UserX } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { getPenggunaKlien, hapusPenggunaKlien, resetPasswordKlien, ubahPenggunaKlien, type PenggunaKlien } from '@/lib/api'
import { formatRelatif, formatTanggal } from '@/lib/format'
import { HapusPermanenDialog, KonfirmasiDialog } from './KelolaDialog'
import { LanggananDialog } from './Klien'

// Pengguna klien: semua akun usaha (bukan tim) yang boleh dilihat anggota — kontak, usaha, status.
// Nonaktif = soft delete (bisa diaktifkan lagi); hapus permanen hanya untuk akun nonaktif.

type Saring = 'semua' | 'aktif' | 'nonaktif'

const STATUS_LG: Record<PenggunaKlien['langganan']['status'], [string, string]> = { aktif: ['pill-green', 'Aktif'], trial: ['pill-amber', 'Masa coba'], habis: ['pill-red', 'Habis'] }

export function PenggunaKlienView({ bisaKelola, bisaReset, bisaHapus, bisaLangganan = false }: { bisaKelola: boolean; bisaReset: boolean; bisaHapus: boolean; bisaLangganan?: boolean }) {
  const [langganan, setLangganan] = useState<PenggunaKlien | null>(null)
  const [rows, setRows] = useState<PenggunaKlien[] | null>(null)
  const [error, setError] = useState('')
  const [q, setQ] = useState('')
  const [saring, setSaring] = useState<Saring>('semua')
  const [edit, setEdit] = useState<PenggunaKlien | null>(null)
  const [hapus, setHapus] = useState<PenggunaKlien | null>(null)
  const [nonaktif, setNonaktif] = useState<PenggunaKlien | null>(null)
  const [reset, setReset] = useState<{ u: PenggunaKlien; pw?: string } | null>(null)
  const muat = useCallback(() => getPenggunaKlien().then(setRows, (e: Error) => setError(e.message)), [])
  useEffect(() => { void muat() }, [muat])

  const tampil = useMemo(() => (rows ?? []).filter((u) =>
    (saring === 'semua' || (saring === 'aktif') === u.active) &&
    [u.name, u.email, u.wa, u.usaha, u.kota].join(' ').toLowerCase().includes(q.trim().toLowerCase())), [rows, q, saring])

  const setAktif = (u: PenggunaKlien, aktif: boolean) => ubahPenggunaKlien(u.id, { name: u.name, email: u.email, wa: u.wa ?? '', aktif })

  async function jalankanReset(u: PenggunaKlien) {
    try {
      const r = await resetPasswordKlien(u.id)
      setReset({ u, pw: r.password })
    } catch (e) {
      setError((e as Error).message)
      setReset(null)
    }
  }

  return (
    <>
      <PageHeader title="Pengguna Klien" subtitle={rows ? `${rows.length} akun pengguna di usaha yang bisa Anda lihat` : 'Memuat…'} />
      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', whiteSpace: 'normal' }}>{error}</div>}
      <div className="card" style={{ padding: 20 }}>
        <div className="flex flex-wrap items-center gap-3" style={{ marginBottom: 12 }}>
          <div className="tabbar" role="tablist" aria-label="Saring status">
            {(['semua', 'aktif', 'nonaktif'] as Saring[]).map((s) => (
              <button key={s} role="tab" className="tab-btn" aria-selected={saring === s} onClick={() => setSaring(s)}>{s[0].toUpperCase() + s.slice(1)}</button>
            ))}
          </div>
          <Input aria-label="Cari pengguna" placeholder="Cari nama, email, no HP, usaha" value={q} onChange={(e) => setQ(e.target.value)} style={{ maxWidth: 280, marginLeft: 'auto' }} />
        </div>
        <div className="table-scroll">
          <table className="data-table" style={{ minWidth: 1300 }}>
            <thead><tr><th>Nama</th><th>No HP</th><th>Usaha</th><th>Alamat usaha</th><th>Role</th><th>Langganan</th><th>Status</th><th>Terakhir aktif</th><th style={{ textAlign: 'right' }}>Aksi</th></tr></thead>
            <tbody>
              {tampil.map((u) => (
                <tr key={u.id}>
                  <td><div className="font-semibold">{u.name}</div><div className="text-xs text-muted-foreground">{u.email}</div></td>
                  <td className="tabular">{u.wa || <span className="text-muted-foreground">—</span>}</td>
                  <td>{u.usaha}{u.arsip && <span className="pill-status pill-red" style={{ marginLeft: 6, fontSize: 10 }}>diarsipkan</span>}</td>
                  <td className="text-sm" style={{ whiteSpace: 'normal', maxWidth: 240 }}>{[u.alamat, u.kota].filter(Boolean).join(', ') || <span className="text-muted-foreground">—</span>}</td>
                  <td>{u.role_label}</td>
                  <td>
                    <span className={'pill-status ' + STATUS_LG[u.langganan.status][0]}>{STATUS_LG[u.langganan.status][1]}</span>
                    {u.langganan.sampai && <div className="text-xs text-muted-foreground">s/d {formatTanggal(u.langganan.sampai)}</div>}
                    {bisaLangganan && !u.demo && !u.arsip && (
                      <Button variant="link" size="sm" className="h-auto p-0 text-xs" onClick={() => setLangganan(u)}>
                        {u.langganan.status === 'habis' ? 'Buka semua fitur' : 'Atur langganan'}
                      </Button>
                    )}
                  </td>
                  <td>{u.active ? <span className="pill-status pill-green">Aktif</span> : <span className="pill-status pill-status-muted">Nonaktif</span>}</td>
                  <td>{formatRelatif(u.last_seen)}</td>
                  <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
                    {bisaKelola && <Button variant="ghost" size="icon-sm" aria-label={`Ubah ${u.name}`} onClick={() => setEdit(u)}><Pencil /></Button>}
                    {bisaReset && <Button variant="ghost" size="icon-sm" aria-label={`Reset password ${u.name}`} title="Reset password" onClick={() => setReset({ u })}><KeyRound /></Button>}
                    {bisaKelola && (u.active
                      ? <Button variant="ghost" size="sm" title="Soft delete: tidak bisa masuk, data tetap" onClick={() => setNonaktif(u)}><UserX /> Nonaktifkan</Button>
                      : <Button variant="ghost" size="sm" onClick={() => void setAktif(u, true).then(muat, (e: Error) => setError(e.message))}><UserCheck /> Aktifkan</Button>)}
                    {bisaHapus && <Button variant="ghost" size="sm" className="text-destructive" title="Hard delete: hapus permanen" onClick={() => setHapus(u)}><Trash2 /> Hapus</Button>}
                  </td>
                </tr>
              ))}
              {rows && tampil.length === 0 && <tr><td colSpan={9} className="text-center text-muted-foreground" style={{ padding: 24 }}>Tidak ada pengguna yang cocok.</td></tr>}
            </tbody>
          </table>
        </div>
        <p className="mt-3 text-xs text-muted-foreground">Nonaktifkan (soft delete) = pengguna tidak bisa masuk, data tetap, bisa diaktifkan lagi. Hapus (hard delete) = akun dihapus permanen; akun aktif dinonaktifkan dulu otomatis. Semua tindakan tercatat di audit log.</p>
      </div>

      {langganan && (
        <LanggananDialog open klien={{ id: langganan.klien_id, nama: langganan.usaha }} onClose={() => setLangganan(null)}
          onSaved={() => { setLangganan(null); void muat() }} />
      )}
      {edit && <EditPengguna u={edit} onClose={() => setEdit(null)} onSaved={() => { setEdit(null); void muat() }} />}

      <Dialog open={!!reset} onOpenChange={(o) => !o && setReset(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Reset password {reset?.u.name}</DialogTitle>
            <DialogDescription>
              {reset?.pw ? 'Password sementara di bawah hanya tampil sekali. Sampaikan langsung ke pengguna dan minta ia menggantinya setelah masuk.'
                : 'Password lama tidak berlaku lagi dan semua sesi pengguna ini berakhir.'}
            </DialogDescription>
          </DialogHeader>
          {reset?.pw && <div className="rounded-md border p-3 text-center text-lg font-bold tabular" style={{ letterSpacing: 1 }}>{reset.pw}</div>}
          <DialogFooter>
            {reset?.pw
              ? <Button type="button" onClick={() => { void navigator.clipboard?.writeText(reset.pw!); setReset(null) }}>Salin & tutup</Button>
              : <>
                <Button type="button" variant="outline" onClick={() => setReset(null)}>Batal</Button>
                <Button type="button" onClick={() => void jalankanReset(reset!.u)}>Buat password sementara</Button>
              </>}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {hapus && (
        <HapusPermanenDialog open onClose={() => setHapus(null)} judul={`Hapus permanen ${hapus.name}`}
          teks={(hapus.active ? 'Akun ini masih aktif; akan dinonaktifkan dulu lalu dihapus. ' : '') + 'Akun pengguna ini dihapus dari database. Data usahanya tetap ada.'}
          kunci={hapus.email} labelKunci="email pengguna"
          onHapus={async (k) => {
            if (hapus.active) await setAktif(hapus, false)
            await hapusPenggunaKlien(hapus.id, k); setHapus(null); await muat()
          }} />
      )}
      {nonaktif && (
        <KonfirmasiDialog open onClose={() => setNonaktif(null)} judul={`Nonaktifkan ${nonaktif.name}?`} tombol="Nonaktifkan"
          teks="Soft delete: pengguna tidak bisa masuk lagi dan sesinya berakhir, tetapi akun dan data usahanya tetap tersimpan. Bisa diaktifkan kembali kapan saja."
          onYa={async () => { await setAktif(nonaktif, false); setNonaktif(null); await muat() }} />
      )}
    </>
  )
}

function EditPengguna({ u, onClose, onSaved }: { u: PenggunaKlien; onClose: () => void; onSaved: () => void }) {
  const [v, setV] = useState({ name: u.name, email: u.email, wa: u.wa ?? '', aktif: u.active })
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  async function simpan(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await ubahPenggunaKlien(u.id, v)
      onSaved()
    } catch (err) {
      setError((err as Error).message)
      setBusy(false)
    }
  }
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <form onSubmit={simpan} className="grid gap-3">
          <DialogHeader>
            <DialogTitle>Ubah pengguna</DialogTitle>
            <DialogDescription>{u.usaha} · {u.role_label}. Mengganti email mengakhiri sesi pengguna.</DialogDescription>
          </DialogHeader>
          <div className="grid gap-1.5"><Label htmlFor="ep-nama">Nama</Label><Input id="ep-nama" required value={v.name} onChange={(e) => setV({ ...v, name: e.target.value })} /></div>
          <div className="grid gap-1.5"><Label htmlFor="ep-email">Email</Label><Input id="ep-email" type="email" required value={v.email} onChange={(e) => setV({ ...v, email: e.target.value })} /></div>
          <div className="grid gap-1.5"><Label htmlFor="ep-wa">No HP / WhatsApp</Label><Input id="ep-wa" value={v.wa} onChange={(e) => setV({ ...v, wa: e.target.value })} /></div>
          <label className="flex items-center gap-3 text-sm">
            <Switch checked={v.aktif} onCheckedChange={(c) => setV({ ...v, aktif: c })} />
            {v.aktif ? 'Aktif — bisa masuk' : 'Nonaktif — tidak bisa masuk (data tetap)'}
          </label>
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
