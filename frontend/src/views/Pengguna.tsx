import { Check, Pencil, Plus, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { NAV } from '@/lib/nav'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import {
  getRiwayatAksesTim, type AuditRow,
  createUser, deleteUser, listUsers, updateUser, type Role, type RoleInfo, type User, type UserInput,
} from '@/lib/api'

const ROLE_ORDER: Role[] = ['owner', 'admin', 'kasir']
const ROLE_PILL: Record<Role, string> = { owner: 'pill-brand', admin: 'pill-green', kasir: 'pill-status-muted', mp: 'pill-status-muted', lihat: 'pill-status-muted' }
// Izin di luar menu sidebar (dicek server lewat guard yang sama).
const EXTRA_PERMISSIONS = [{ view: 'transaksi-kelola', label: 'Ubah & hapus transaksi' }] as const

const EMPTY: UserInput = { name: '', email: '', role: 'kasir', active: true, password: '' }

type Editing = { id: number | null; input: UserInput } | null

export function Pengguna({ currentUserId }: { currentUserId: number }) {
  const [users, setUsers] = useState<User[]>([])
  const [roles, setRoles] = useState<Record<Role, RoleInfo> | null>(null)
  const [editing, setEditing] = useState<Editing>(null)
  const [deleting, setDeleting] = useState<User | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const refresh = useCallback(async () => {
    const res = await listUsers()
    setUsers(res.users)
    setRoles(res.roles)
  }, [])

  useEffect(() => {
    refresh().catch((e: Error) => setError(e.message))
  }, [refresh])

  async function save(e: FormEvent) {
    e.preventDefault()
    if (!editing) return
    setBusy(true)
    setError('')
    try {
      if (editing.id === null) await createUser(editing.input)
      else await updateUser(editing.id, editing.input)
      setEditing(null)
      await refresh()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  async function confirmDelete() {
    if (!deleting) return
    setBusy(true)
    setError('')
    try {
      await deleteUser(deleting.id)
      setDeleting(null)
      await refresh()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const set = (patch: Partial<UserInput>) => setEditing((ed) => ed && { ...ed, input: { ...ed.input, ...patch } })

  return (
    <>
      <PageHeader title="Pengguna & Akses" subtitle="Kelola siapa yang bisa masuk dan menu apa saja yang boleh dibuka tiap role.">
        <button className="btn-cta inline-flex items-center gap-1.5" onClick={() => { setError(''); setEditing({ id: null, input: EMPTY }) }}>
          <Plus size={16} /> Tambah Pengguna
        </button>
      </PageHeader>

      {error && !editing && !deleting && (
        <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start' }}>{error}</div>
      )}

      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Daftar Pengguna</div>
        <div className="chart-subtitle">{users.length} akun · password tidak pernah ditampilkan</div>
        <table className="data-table">
          <thead>
            <tr>
              <th>Nama</th><th>Email</th><th>Role</th><th>Status</th><th>Dibuat</th><th style={{ textAlign: 'right' }}>Aksi</th>
            </tr>
          </thead>
          <tbody>
            {users.map((u) => {
              const self = u.id === currentUserId
              return (
                <tr key={u.id}>
                  <td style={{ fontWeight: 600 }}>
                    {u.name}
                    {self && <span style={{ color: 'var(--text-muted)', fontWeight: 400 }}> (Anda)</span>}
                  </td>
                  <td style={{ color: 'var(--text-secondary)' }}>{u.email}</td>
                  <td><span className={'pill-status ' + ROLE_PILL[u.role]}>{roles?.[u.role]?.label ?? u.role}</span></td>
                  <td>
                    <span className={'pill-status ' + (u.active ? 'pill-green' : 'pill-red')}>{u.active ? 'Aktif' : 'Nonaktif'}</span>
                  </td>
                  <td className="tabular" style={{ color: 'var(--text-secondary)' }}>{u.created_at}</td>
                  <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label={`Ubah ${u.name}`}
                      onClick={() => { setError(''); setEditing({ id: u.id, input: { name: u.name, email: u.email, role: u.role, active: u.active, password: '' } }) }}
                    >
                      <Pencil />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="text-destructive hover:text-destructive"
                      aria-label={`Hapus ${u.name}`}
                      title={self ? 'Tidak bisa menghapus akun sendiri' : undefined}
                      disabled={self}
                      onClick={() => { setError(''); setDeleting(u) }}
                    >
                      <Trash2 />
                    </Button>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      {roles && (
        <div className="card" style={{ padding: 20 }}>
          <div className="chart-title">Hak Akses per Role</div>
          <div className="chart-subtitle">Role bersifat tetap. Menu yang tidak diizinkan disembunyikan dan API-nya menolak akses.</div>
          <table className="data-table">
            <thead>
              <tr>
                <th>Menu</th>
                {ROLE_ORDER.map((r) => (
                  <th key={r} style={{ textAlign: 'center' }} title={roles[r].description}>{roles[r].label}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {[...NAV.flatMap((g) => g.items), ...EXTRA_PERMISSIONS].map((item) => (
                <tr key={item.view}>
                  <td>{item.label}</td>
                  {ROLE_ORDER.map((r) => (
                    <td key={r} style={{ textAlign: 'center' }}>
                      {roles[r].views.includes(item.view)
                        ? <Check size={16} className="inline" color="var(--text-profit)" aria-label="boleh" />
                        : <span style={{ color: 'var(--text-muted)' }} aria-label="tidak">–</span>}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Dialog open={editing !== null} onOpenChange={(open) => !open && setEditing(null)}>
        <DialogContent>
          <form onSubmit={save} className="grid gap-4">
            <DialogHeader>
              <DialogTitle>{editing?.id === null ? 'Tambah Pengguna' : 'Ubah Pengguna'}</DialogTitle>
              <DialogDescription>
                {editing?.id === null ? 'Akun baru langsung bisa login dengan password ini.' : 'Kosongkan password bila tidak ingin mengganti.'}
              </DialogDescription>
            </DialogHeader>
            {editing && (
              <>
                <div className="grid gap-1.5">
                  <Label htmlFor="u-name">Nama</Label>
                  <Input id="u-name" required value={editing.input.name} onChange={(e) => set({ name: e.target.value })} />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="u-email">Email</Label>
                  <Input id="u-email" type="email" required value={editing.input.email} onChange={(e) => set({ email: e.target.value })} />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="u-role">Role</Label>
                  <Select value={editing.input.role} onValueChange={(v) => set({ role: v as Role })}>
                    <SelectTrigger id="u-role" className="w-full"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      {ROLE_ORDER.map((r) => (
                        <SelectItem key={r} value={r}>{roles?.[r].label ?? r}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {roles && <p className="text-xs text-muted-foreground">{roles[editing.input.role].description}</p>}
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="u-pass">Password</Label>
                  <Input
                    id="u-pass"
                    type="password"
                    autoComplete="new-password"
                    minLength={8}
                    required={editing.id === null}
                    placeholder={editing.id === null ? 'Minimal 8 karakter' : '••••••••  (tidak diubah)'}
                    value={editing.input.password}
                    onChange={(e) => set({ password: e.target.value })}
                  />
                </div>
                <div className="flex items-center justify-between">
                  <Label htmlFor="u-active">Akun aktif</Label>
                  <Switch id="u-active" checked={editing.input.active} onCheckedChange={(v) => set({ active: v })} />
                </div>
                {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
              </>
            )}
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setEditing(null)}>Batal</Button>
              <Button type="submit" disabled={busy}>{busy ? 'Menyimpan…' : 'Simpan'}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={deleting !== null} onOpenChange={(open) => !open && setDeleting(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Hapus pengguna?</DialogTitle>
            <DialogDescription>
              Akun <strong>{deleting?.name}</strong> ({deleting?.email}) akan dihapus dan langsung tidak bisa login.
              Bila hanya ingin menahan akses sementara, nonaktifkan saja akunnya.
            </DialogDescription>
          </DialogHeader>
          {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleting(null)}>Batal</Button>
            <Button variant="destructive" disabled={busy} onClick={confirmDelete}>Hapus</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <RiwayatAksesTim />
    </>
  )
}

/** Transparansi untuk pemilik usaha: siapa dari tim Master Pajak yang membuka data usaha ini,
 *  kapan, untuk apa, dan perubahan apa yang mereka lakukan — tercatat atas nama anggota tim. */
function RiwayatAksesTim() {
  const [rows, setRows] = useState<AuditRow[] | null>(null)
  const [error, setError] = useState('')
  useEffect(() => { getRiwayatAksesTim().then(setRows, (e: Error) => setError(e.message)) }, [])
  return (
    <div className="card" style={{ padding: 20 }}>
      <div className="chart-title">Riwayat akses tim Master Pajak</div>
      <div className="chart-subtitle">
        Setiap kali tim kami membuka atau mengusulkan perubahan pada data usaha Anda, tercatat di sini atas nama anggota tim tersebut, bukan atas nama Anda.
        Mode “lihat sebagai” hanya baca; koreksi baru berlaku setelah Anda setujui.
      </div>
      {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
      {rows && rows.length === 0 && <p className="text-sm text-muted-foreground" style={{ paddingTop: 10 }}>Belum ada akses dari tim Master Pajak.</p>}
      {rows && rows.length > 0 && (
        <div className="table-scroll" style={{ maxHeight: 360, overflowY: 'auto', marginTop: 10 }}>
          <table className="data-table">
            <thead><tr><th>Waktu</th><th>Oleh</th><th>Aktivitas</th><th>Keterangan</th></tr></thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.id}>
                  <td className="tabular text-sm" style={{ whiteSpace: 'nowrap' }}>{new Date(r.waktu).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' })}</td>
                  <td>
                    {r.peran === 'Owner klien'
                      ? <div className="font-semibold">Anda (pemilik usaha)</div>
                      : <><div className="text-xs text-muted-foreground">Tim Master Pajak</div><div className="font-semibold">{r.nama} <span className="font-normal text-muted-foreground">({r.peran})</span></div></>}
                  </td>
                  <td style={{ whiteSpace: 'normal' }}>{r.aksi}</td>
                  <td className="text-sm text-muted-foreground" style={{ whiteSpace: 'normal' }}>{r.keterangan || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
