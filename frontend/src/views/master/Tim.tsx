import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Pencil, Plus, Trash2, UserCheck, UserX } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { HapusPermanenDialog, KonfirmasiDialog } from './KelolaDialog'
import { getTim, hapusAnggotaTim, hapusRole, simpanRole, tambahAnggota, ubahAnggota, type AnggotaRow, type Izin, type MasterRole, type TimData } from '@/lib/api'
import { formatRelatif } from '@/lib/format'
import { useHP } from '@/lib/hp'
import { AksiLembar, LembarBawah } from '@/components/hp/LembarBawah'
import { BarisRingkas } from '@/components/hp/BarisRingkas'

type Form = { id: number | null; nama: string; email: string; password: string; role: string; klien: number[]; aktif: boolean }

export function TimAkses({ selfId, bisaHapus }: { selfId: number; bisaHapus: boolean }) {
  const [hapus, setHapus] = useState<AnggotaRow | null>(null)
  const [nonaktif, setNonaktif] = useState<AnggotaRow | null>(null)
  const [tim, setTim] = useState<TimData | null>(null)
  const [tab, setTab] = useState<'anggota' | 'role'>('anggota')
  const [form, setForm] = useState<Form | null>(null)
  const [roleBaru, setRoleBaru] = useState<{ nama: string; izin: Izin[] } | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const hp = useHP()
  const [aksi, setAksi] = useState<AnggotaRow | null>(null) // lembar ⋯ di HP
  const muat = useCallback(() => getTim().then(setTim, (e: Error) => setError(e.message)), [])
  useEffect(() => { void muat() }, [muat])

  const roleNama = (id: string) => tim?.roles.find((r) => r.id === id)?.nama ?? id
  const lihatSemua = (id: string) => id === 'master' || !!tim?.roles.find((r) => r.id === id)?.izin.includes('klien_semua')

  async function jalankan(f: () => Promise<unknown>, selesai?: () => void) {
    setBusy(true)
    setError('')
    try {
      await f()
      selesai?.()
      await muat()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  function simpanAnggota(e: FormEvent) {
    e.preventDefault()
    if (!form) return
    void jalankan(() => form.id === null
      ? tambahAnggota({ nama: form.nama, email: form.email, password: form.password, role: form.role, klien: form.klien })
      : ubahAnggota(form.id, { role: form.role, klien: form.klien, aktif: form.aktif }), () => setForm(null))
  }

  const toggleIzin = (r: MasterRole, i: Izin) =>
    void jalankan(() => simpanRole({ id: r.id, nama: r.nama, izin: r.izin.includes(i) ? r.izin.filter((x) => x !== i) : [...r.izin, i] }))

  const setAktif = (a: AnggotaRow, aktif: boolean) => ubahAnggota(a.user.id, { role: a.role, klien: a.klien, aktif })
  const set = (p: Partial<Form>) => setForm((f) => f && { ...f, ...p })
  const edit = (a: AnggotaRow) => { setError(''); setForm({ id: a.user.id, nama: a.user.name, email: a.user.email, password: '', role: a.role, klien: a.klien, aktif: a.user.active }) }

  return (
    <>
      <PageHeader title="Tim & Akses" subtitle="Akun tim MasterPajak AI, role beserta izinnya, dan klien yang ditugaskan ke tiap anggota.">
        {tab === 'anggota'
          ? <button className="btn-cta inline-flex items-center gap-1.5" onClick={() => { setError(''); setForm({ id: null, nama: '', email: '', password: '', role: 'konsultan', klien: [], aktif: true }) }}><Plus size={16} /> Tambah anggota</button>
          : <button className="btn-cta inline-flex items-center gap-1.5" onClick={() => { setError(''); setRoleBaru({ nama: '', izin: ['klien_lihat'] }) }}><Plus size={16} /> Role baru</button>}
      </PageHeader>
      <div className="tabbar" role="tablist" aria-label="Tim atau role">
        <button role="tab" className="tab-btn" aria-selected={tab === 'anggota'} onClick={() => setTab('anggota')}>Anggota tim</button>
        <button role="tab" className="tab-btn" aria-selected={tab === 'role'} onClick={() => setTab('role')}>Role &amp; izin</button>
      </div>
      {error && !form && !roleBaru && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start' }}>{error}</div>}

      {tim && tab === 'anggota' && hp && (
        // HP: satu baris per anggota; ⋯ membuka aksi lain di lembar bawah.
        <div className="daftar-ringkas">
          {tim.anggota.map((a) => {
            const klien = lihatSemua(a.role) ? 'Semua klien'
              : a.klien.map((id) => tim.klien.find((k) => k.id === id)?.nama ?? `#${id}`).join(', ') || 'Belum ada klien'
            const bisaLain = a.user.id !== selfId
            return (
              <BarisRingkas key={a.user.id} label={a.user.name} aktif={a.user.active}
                judul={<>{a.user.name}{a.user.id === selfId && <span className="text-muted-foreground font-normal"> (Anda)</span>}</>}
                sub={`${a.role ? roleNama(a.role) : 'Tanpa role'} · ${klien}`}
                labelUbah="Ubah role & klien" onUbah={() => edit(a)} onLainnya={bisaLain ? () => setAksi(a) : undefined} />
            )
          })}
        </div>
      )}
      <LembarBawah buka={!!aksi} judul={aksi?.user.name ?? ''} onTutup={() => setAksi(null)}>
        {aksi && (
          <>
            <dl className="lembar-rinci">
              <dt>Email</dt><dd>{aksi.user.email}</dd>
              <dt>Role</dt><dd>{aksi.role ? roleNama(aksi.role) : 'Tanpa role'}</dd>
              <dt>Status</dt><dd>{aksi.user.active ? 'Aktif' : 'Nonaktif'}</dd>
              <dt>Terakhir aktif</dt><dd>{formatRelatif(aksi.user.last_seen)}</dd>
            </dl>
            {aksi.user.active
              ? <AksiLembar ikon={<UserX />} onClick={() => { setAksi(null); setNonaktif(aksi) }}>Nonaktifkan</AksiLembar>
              : <AksiLembar ikon={<UserCheck />} disabled={busy} onClick={() => { setAksi(null); void jalankan(() => setAktif(aksi, true)) }}>Aktifkan</AksiLembar>}
            {bisaHapus && <AksiLembar ikon={<Trash2 />} bahaya onClick={() => { setAksi(null); setHapus(aksi) }}>Hapus permanen</AksiLembar>}
          </>
        )}
      </LembarBawah>

      {tim && tab === 'anggota' && !hp && (
        <div className="card" style={{ padding: 20 }}>
          <div className="table-scroll">
            <table className="data-table" style={{ minWidth: 720 }}>
              <thead><tr><th>Nama</th><th>Role</th><th>Klien ditugaskan</th><th>Terakhir aktif</th><th style={{ textAlign: 'right' }}>Aksi</th></tr></thead>
              <tbody>
                {tim.anggota.map((a) => (
                  <tr key={a.user.id}>
                    <td>
                      <div className="font-semibold">{a.user.name}{a.user.id === selfId && <span className="text-muted-foreground font-normal"> (Anda)</span>}</div>
                      <div className="text-xs text-muted-foreground">{a.user.email}</div>
                    </td>
                    <td>
                      <span className={'pill-status ' + (a.role === 'master' ? 'pill-brand' : 'pill-status-muted')}>{a.role ? roleNama(a.role) : 'Tanpa role'}</span>
                      {!a.user.active && <span className="pill-status pill-red" style={{ marginLeft: 6 }}>nonaktif</span>}
                    </td>
                    <td className="text-sm">{lihatSemua(a.role) ? <span className="text-muted-foreground">Semua klien</span>
                      : a.klien.map((id) => tim.klien.find((k) => k.id === id)?.nama ?? `#${id}`).join(', ') || <span className="text-muted-foreground">Belum ada</span>}</td>
                    <td>{formatRelatif(a.user.last_seen)}</td>
                    <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
                      <Button variant="ghost" size="icon-sm" aria-label={`Ubah ${a.user.name}`} title="Ubah role & klien" onClick={() => edit(a)}><Pencil /></Button>
                      {a.user.id !== selfId && (a.user.active
                        ? <Button variant="ghost" size="sm" title="Soft delete: tidak bisa masuk, data tetap" onClick={() => setNonaktif(a)}><UserX /> Nonaktifkan</Button>
                        : <Button variant="ghost" size="sm" disabled={busy} onClick={() => void jalankan(() => setAktif(a, true))}><UserCheck /> Aktifkan</Button>)}
                      {bisaHapus && a.user.id !== selfId && (
                        <Button variant="ghost" size="sm" className="text-destructive" title="Hard delete: hapus permanen" onClick={() => setHapus(a)}><Trash2 /> Hapus</Button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {tim && tab === 'role' && (
        <div className="card" style={{ padding: 20 }}>
          <div className="chart-subtitle">Centang untuk mengubah izin; tersimpan langsung dan tercatat di audit log. Role Master selalu punya semua izin.</div>
          <div className="table-scroll">
            {/* Matriks centang: di HP tetap tabel geser dengan kolom izin menempel di kiri. */}
            <table className="data-table tabel-matriks" data-kartu="tidak" style={{ minWidth: 320 + tim.roles.length * 120 }}>
              <thead>
                <tr>
                  <th>Izin</th>
                  {tim.roles.map((r) => (
                    <th key={r.id} style={{ textAlign: 'center' }}>
                      {r.nama}
                      {!r.kunci && !['konsultan', 'cs'].includes(r.id) && (
                        <Button variant="ghost" size="icon-sm" aria-label={`Hapus role ${r.nama}`} disabled={busy} onClick={() => void jalankan(() => hapusRole(r.id))}><Trash2 /></Button>
                      )}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {tim.izin.map((i) => (
                  <tr key={i.id}>
                    <td><div className="font-semibold">{i.label}</div>{i.keterangan && <div className="text-xs text-muted-foreground">{i.keterangan}</div>}</td>
                    {tim.roles.map((r) => (
                      <td key={r.id} style={{ textAlign: 'center' }}>
                        <input type="checkbox" aria-label={`${r.nama}: ${i.label}`} checked={r.kunci || r.izin.includes(i.id)} disabled={r.kunci || busy}
                          onChange={() => toggleIzin(r, i.id)} style={{ width: 16, height: 16, accentColor: 'var(--primary)' }} />
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <Dialog open={form !== null} onOpenChange={(o) => !o && setForm(null)}>
        <DialogContent>
          {form && tim && (
            <form onSubmit={simpanAnggota} className="grid gap-4">
              <DialogHeader>
                <DialogTitle>{form.id === null ? 'Tambah anggota tim' : `Ubah ${form.nama}`}</DialogTitle>
                <DialogDescription>{form.id === null ? 'Akun langsung bisa masuk ke Panel Master dengan password ini.' : form.email}</DialogDescription>
              </DialogHeader>
              {form.id === null && (
                <>
                  <div className="grid gap-1.5"><Label htmlFor="tm-nama">Nama</Label><Input id="tm-nama" required value={form.nama} onChange={(e) => set({ nama: e.target.value })} /></div>
                  <div className="grid gap-1.5"><Label htmlFor="tm-email">Email</Label><Input id="tm-email" type="email" required value={form.email} onChange={(e) => set({ email: e.target.value })} /></div>
                  <div className="grid gap-1.5"><Label htmlFor="tm-pass">Password awal</Label><Input id="tm-pass" type="password" autoComplete="new-password" minLength={8} required value={form.password} onChange={(e) => set({ password: e.target.value })} placeholder="Minimal 8 karakter" /></div>
                </>
              )}
              <div className="grid gap-1.5">
                <Label htmlFor="tm-role">Role</Label>
                <Select value={form.role} onValueChange={(v) => set({ role: v })}>
                  <SelectTrigger id="tm-role" className="w-full"><SelectValue placeholder="Pilih role" /></SelectTrigger>
                  <SelectContent>{tim.roles.map((r) => <SelectItem key={r.id} value={r.id}>{r.nama}</SelectItem>)}</SelectContent>
                </Select>
              </div>
              {!lihatSemua(form.role) && (
                <fieldset className="grid gap-1.5">
                  <legend className="text-sm font-medium" style={{ marginBottom: 6 }}>Klien yang ditugaskan</legend>
                  <div className="grid gap-1 rounded-lg border border-border" style={{ maxHeight: 220, overflowY: 'auto', padding: 8 }}>
                    {tim.klien.map((k) => (
                      <label key={k.id} className="flex items-center gap-2 text-sm" style={{ padding: '4px 6px', cursor: 'pointer' }}>
                        <input type="checkbox" checked={form.klien.includes(k.id)} style={{ accentColor: 'var(--primary)' }}
                          onChange={(e) => set({ klien: e.target.checked ? [...form.klien, k.id] : form.klien.filter((x) => x !== k.id) })} />
                        {k.nama}
                      </label>
                    ))}
                  </div>
                </fieldset>
              )}
              {form.id !== null && (
                <div className="flex items-center justify-between">
                  <Label htmlFor="tm-aktif">{form.aktif ? 'Akun aktif' : 'Nonaktif (soft delete) — tidak bisa masuk, bisa diaktifkan lagi'}</Label>
                  <Switch id="tm-aktif" checked={form.aktif} disabled={form.id === selfId} onCheckedChange={(v) => set({ aktif: v })} />
                </div>
              )}
              {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setForm(null)}>Batal</Button>
                <Button type="submit" disabled={busy}>{busy ? 'Menyimpan…' : 'Simpan'}</Button>
              </DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>

      <Dialog open={roleBaru !== null} onOpenChange={(o) => !o && setRoleBaru(null)}>
        <DialogContent>
          {roleBaru && tim && (
            <form className="grid gap-4" onSubmit={(e) => { e.preventDefault(); void jalankan(() => simpanRole(roleBaru), () => { setRoleBaru(null); setTab('role') }) }}>
              <DialogHeader>
                <DialogTitle>Role baru</DialogTitle>
                <DialogDescription>Izin bisa diubah lagi kapan saja di tabel Role &amp; izin.</DialogDescription>
              </DialogHeader>
              <div className="grid gap-1.5"><Label htmlFor="rl-nama">Nama role</Label><Input id="rl-nama" required placeholder="mis. Akuntan Senior" value={roleBaru.nama} onChange={(e) => setRoleBaru({ ...roleBaru, nama: e.target.value })} /></div>
              <div className="grid gap-1 rounded-lg border border-border" style={{ padding: 8 }}>
                {tim.izin.map((i) => (
                  <label key={i.id} className="flex items-start gap-2 text-sm" style={{ padding: '4px 6px', cursor: 'pointer' }}>
                    <input type="checkbox" style={{ accentColor: 'var(--primary)', marginTop: 3 }} checked={roleBaru.izin.includes(i.id)}
                      onChange={(e) => setRoleBaru({ ...roleBaru, izin: e.target.checked ? [...roleBaru.izin, i.id] : roleBaru.izin.filter((x) => x !== i.id) })} />
                    <span><span className="font-semibold">{i.label}</span>{i.keterangan && <span className="block text-xs text-muted-foreground">{i.keterangan}</span>}</span>
                  </label>
                ))}
              </div>
              {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setRoleBaru(null)}>Batal</Button>
                <Button type="submit" disabled={busy}>{busy ? 'Menyimpan…' : 'Buat role'}</Button>
              </DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>
      {hapus && (
        <HapusPermanenDialog open onClose={() => setHapus(null)} judul={`Hapus permanen ${hapus.user.name}`}
          teks={(hapus.user.active ? 'Akun ini masih aktif; akan dinonaktifkan dulu lalu dihapus. ' : '') + 'Akun anggota tim ini dan penugasannya dihapus. Catatan audit yang dibuatnya tetap tersimpan.'}
          kunci={hapus.user.email} labelKunci="email anggota"
          onHapus={async (k) => {
            if (hapus.user.active) await setAktif(hapus, false)
            await hapusAnggotaTim(hapus.user.id, k); setHapus(null); await muat()
          }} />
      )}
      {nonaktif && (
        <KonfirmasiDialog open onClose={() => setNonaktif(null)} judul={`Nonaktifkan ${nonaktif.user.name}?`} tombol="Nonaktifkan"
          teks="Soft delete: anggota tidak bisa masuk lagi dan sesinya berakhir, tetapi akun, penugasan, dan catatan audit tetap tersimpan. Bisa diaktifkan kembali kapan saja."
          onYa={async () => { await setAktif(nonaktif, false); setNonaktif(null); await muat() }} />
      )}
    </>
  )
}
