import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { ArrowLeft, Eye } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { KoreksiCard } from '@/components/KoreksiCard'
import {
  analisaKlien, arsipKlien, aturLangganan, kunciBaru, getKlien, hapusKlien, mulaiLihat, pulihkanKlien, ubahKlien, type AnalisaKlien, getPlans, getRingkasanMaster, listKlien,
  type IdentitasUsaha, type KlienDetail, type KlienRow, type Paket, type RingkasanMaster,
} from '@/lib/api'
import { AlasanDialog, HapusPermanenDialog } from './KelolaDialog'
import { formatRelatif, formatRp, formatTanggal } from '@/lib/format'
import { STATUS_LABEL, STATUS_PILL } from '@/lib/tiket'

const STATUS: Record<KlienRow['status'], [string, string]> = {
  aktif: ['pill-green', 'Aktif'], trial: ['pill-amber', 'Masa coba'], habis: ['pill-red', 'Habis'],
}
export const StatusPill = ({ s }: { s: KlienRow['status'] }) => <span className={'pill-status ' + STATUS[s][0]}>{STATUS[s][1]}</span>

function Gagal({ error }: { error: string }) {
  return error ? <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start' }}>{error}</div> : null
}

function Kpi({ label, value, sub, tone }: { label: string; value: string | number; sub?: string; tone?: string }) {
  return (
    <div className="card" style={{ padding: 18 }}>
      <div className="kpi-label">{label}</div>
      <div className="kpi-value tabular" style={tone ? { color: tone } : undefined}>{value}</div>
      {sub && <div className="text-xs text-muted-foreground" style={{ marginTop: 4 }}>{sub}</div>}
    </div>
  )
}

function KlienMini({ rows, onBuka, kanan, kosong }: { rows: KlienRow[]; onBuka: (id: number) => void; kanan: (k: KlienRow) => React.ReactNode; kosong: string }) {
  if (rows.length === 0) return <p className="text-sm text-muted-foreground" style={{ padding: '12px 0' }}>{kosong}</p>
  return (
    <ul className="list-rows">
      {rows.map((k) => (
        <li key={k.id} className="list-row" style={{ alignItems: 'center', gap: 12 }}>
          <button className="text-left" style={{ minWidth: 0, flex: 1 }} onClick={() => onBuka(k.id)}>
            <div className="text-sm font-semibold truncate">{k.nama}</div>
            <div className="text-xs text-muted-foreground truncate">{[k.owner_nama, k.kota].filter(Boolean).join(' · ') || k.owner_email}</div>
          </button>
          {kanan(k)}
        </li>
      ))}
    </ul>
  )
}

export function Ringkasan({ onBuka }: { onBuka: (id: number) => void }) {
  const [r, setR] = useState<RingkasanMaster | null>(null)
  const [error, setError] = useState('')
  useEffect(() => { getRingkasanMaster().then(setR, (e: Error) => setError(e.message)) }, [])
  return (
    <>
      <PageHeader title="Ringkasan" subtitle="Kondisi klien Master Pajak hari ini. Angka keuangan klien tidak ditampilkan di Panel Master." />
      <Gagal error={error} />
      {r && (
        <>
          <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(170px, 1fr))' }}>
            <Kpi label="Klien" value={r.klien} sub={`${r.aktif} aktif · ${r.trial} masa coba · ${r.habis} habis`} />
            {r.mrr !== undefined && <Kpi label="Pendapatan bulanan (MRR)" value={formatRp(r.mrr)} sub={`dari ${r.aktif} langganan aktif`} />}
            <Kpi label="Perlu perhatian" value={r.perlu_perhatian.length} tone={r.perlu_perhatian.length ? 'var(--accent-red)' : undefined} sub="habis, segera habis, atau 14 hari tidak dibuka" />
            <Kpi label="Tiket terbuka" value={r.tiket_terbuka} sub="baru + sedang dikerjakan" />
          </div>
          <div className="grid gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(100%, 340px), 1fr))', alignItems: 'start' }}>
            <div className="card" style={{ padding: 20 }}>
              <div className="chart-title">Perlu perhatian</div>
              <KlienMini rows={r.perlu_perhatian} onBuka={onBuka} kosong="Semua klien aman."
                kanan={(k) => <span className="text-xs text-muted-foreground" style={{ whiteSpace: 'nowrap' }}>{k.alasan}</span>} />
            </div>
            {r.belum_ditugaskan ? (
              <div className="card" style={{ padding: 20 }}>
                <div className="chart-title">Belum ditugaskan</div>
                <div className="chart-subtitle">Tiketnya hanya terlihat oleh yang melihat semua klien. Tugaskan di Tim &amp; Akses.</div>
                <KlienMini rows={r.belum_ditugaskan} onBuka={onBuka} kosong="Semua klien sudah punya penanggung jawab."
                  kanan={(k) => <StatusPill s={k.status} />} />
              </div>
            ) : (
              <div className="card" style={{ padding: 20 }}>
                <div className="chart-title">Terakhir aktif</div>
                <KlienMini rows={r.terakhir_aktif} onBuka={onBuka} kosong="Belum ada klien."
                  kanan={(k) => <span className="text-xs text-muted-foreground" style={{ whiteSpace: 'nowrap' }}>{formatRelatif(k.terakhir_aktif)}</span>} />
              </div>
            )}
          </div>
        </>
      )}
    </>
  )
}

type Filter = 'semua' | 'aktif' | 'trial' | 'habis' | 'perhatian' | 'arsip'
const FILTER: [Filter, string][] = [['semua', 'Semua'], ['aktif', 'Aktif'], ['trial', 'Masa coba'], ['habis', 'Habis'], ['perhatian', 'Perlu perhatian'], ['arsip', 'Diarsipkan']]

export function KlienList({ onBuka, bisaLihat, bisaLangganan = false, onLihat }: { onBuka: (id: number) => void; bisaLihat: boolean; bisaLangganan?: boolean; onLihat: () => void }) {
  const [lihat, setLihat] = useState<KlienRow | null>(null)
  const [langganan, setLangganan] = useState<KlienRow | null>(null)
  const [rows, setRows] = useState<KlienRow[] | null>(null)
  const [error, setError] = useState('')
  const [filter, setFilter] = useState<Filter>('semua')
  const [q, setQ] = useState('')
  const muat = () => listKlien().then(setRows, (e: Error) => setError(e.message))
  useEffect(() => { void muat() }, [])
  const tampil = useMemo(() => (rows ?? []).filter((k) =>
    (filter === 'arsip' ? !!k.arsip : !k.arsip) &&
    (filter === 'semua' || filter === 'arsip' || (filter === 'perhatian' ? k.perlu_perhatian : k.status === filter)) &&
    [k.nama, k.owner_nama, k.owner_email, k.kota].join(' ').toLowerCase().includes(q.trim().toLowerCase())), [rows, filter, q])
  return (
    <>
      <PageHeader title="Klien" subtitle={rows ? `${rows.length} usaha yang bisa Anda lihat` : 'Memuat…'} />
      <Gagal error={error} />
      <div className="card" style={{ padding: 20 }}>
        <div className="flex flex-wrap items-center gap-3" style={{ marginBottom: 12 }}>
          <div className="tabbar" role="tablist" aria-label="Saring status">
            {FILTER.map(([f, l]) => (
              <button key={f} role="tab" className="tab-btn" aria-selected={filter === f} onClick={() => setFilter(f)}>{l}</button>
            ))}
          </div>
          <Input aria-label="Cari klien" placeholder="Cari usaha, owner, kota" value={q} onChange={(e) => setQ(e.target.value)} style={{ maxWidth: 260, marginLeft: 'auto' }} />
        </div>
        <div className="table-scroll">
          <table className="data-table" style={{ minWidth: 900 }}>
            <thead>
              <tr>
                <th>Usaha</th><th>Paket</th><th>Status</th><th>Berlaku s/d</th><th>Terakhir aktif</th>
                <th style={{ textAlign: 'right' }}>Transaksi bulan ini</th><th>Profil</th><th>Ditangani</th>
                {(bisaLihat || bisaLangganan) && <th style={{ textAlign: 'right' }}>Aksi</th>}
              </tr>
            </thead>
            <tbody>
              {tampil.map((k) => (
                <tr key={k.id} onClick={() => onBuka(k.id)} style={{ cursor: 'pointer' }}>
                  <td>
                    <button className="text-left font-semibold" onClick={() => onBuka(k.id)}>{k.nama}</button>
                    {k.demo && <span className="pill-status pill-status-muted" style={{ fontSize: 10, marginLeft: 6 }}>contoh</span>}
                    {k.arsip && <span className="pill-status pill-red" style={{ fontSize: 10, marginLeft: 6 }}>diarsipkan</span>}
                    <div className="text-xs text-muted-foreground">{[k.owner_nama, k.kota, k.jenis, k.pkp && 'PKP'].filter(Boolean).join(' · ')}</div>
                  </td>
                  <td>{k.paket ?? <span className="text-muted-foreground">—</span>}</td>
                  <td><StatusPill s={k.status} /></td>
                  <td className="tabular">{formatTanggal(k.sampai)}</td>
                  <td>{k.perlu_perhatian && k.alasan?.startsWith('Tidak aktif')
                    ? <span className="pill-status pill-red">{formatRelatif(k.terakhir_aktif)}</span>
                    : formatRelatif(k.terakhir_aktif)}</td>
                  <td className="tabular" style={{ textAlign: 'right' }}>{k.transaksi_bulan_ini.toLocaleString('id-ID')}</td>
                  <td>{k.onboarding ? <span className="pill-status pill-green">Lengkap</span> : <span className="pill-status pill-status-muted">Belum</span>}</td>
                  <td>{k.penangan.length ? k.penangan.map((p) => p.nama.split(' ')[0]).join(', ') : <span className="pill-status pill-amber">Belum ada</span>}</td>
                  {(bisaLihat || bisaLangganan) && (
                    <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }} onClick={(e) => e.stopPropagation()}>
                      {bisaLangganan && !k.demo && !k.arsip && (
                        <Button size="sm" variant="outline" className="rounded-full" style={{ marginRight: 6 }} onClick={() => setLangganan(k)}>
                          Atur langganan
                        </Button>
                      )}
                      {bisaLihat && !k.arsip && (
                        <Button size="sm" variant="outline" className="rounded-full" onClick={() => setLihat(k)} aria-label={`Lihat sebagai ${k.nama}`}>
                          <Eye /> Lihat sebagai
                        </Button>
                      )}
                    </td>
                  )}
                </tr>
              ))}
              {rows && tampil.length === 0 && (
                <tr><td colSpan={9} className="text-center text-muted-foreground" style={{ padding: 24 }}>Tidak ada klien yang cocok.</td></tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
      {lihat && <LihatDialog open onClose={() => setLihat(null)} klien={lihat} tiket={[]} onMulai={onLihat} />}
      {langganan && <LanggananDialog open onClose={() => setLangganan(null)} klien={langganan} onSaved={() => { setLangganan(null); void muat() }} />}
    </>
  )
}

const ORDER_PILL: Record<string, string> = { lunas: 'pill-green', manual: 'pill-amber', pending: 'pill-status-muted', gagal: 'pill-red' }

export function KlienDetailView({ id, bisaLangganan, bisaLihat, bisaUbah, bisaHapus, onLihat, onKembali }: {
  id: number; bisaLangganan: boolean; bisaLihat: boolean; bisaUbah: boolean; bisaHapus: boolean; onLihat: () => void; onKembali: () => void
}) {
  const [d, setD] = useState<KlienDetail | null>(null)
  const [error, setError] = useState('')
  const [dialog, setDialog] = useState(false)
  const [lihat, setLihat] = useState(false)
  const [edit, setEdit] = useState(false)
  const [arsip, setArsip] = useState(false)
  const [hapus, setHapus] = useState(false)
  const muat = () => getKlien(id).then(setD, (e: Error) => setError(e.message))
  useEffect(() => { void muat() }, [id]) // eslint-disable-line react-hooks/exhaustive-deps
  const k = d?.klien
  return (
    <>
      <div>
        <Button variant="ghost" size="sm" onClick={onKembali}><ArrowLeft /> Klien</Button>
      </div>
      <PageHeader title={k?.nama ?? 'Memuat…'} subtitle={k ? [k.owner_nama, k.owner_email, k.owner_wa && `WA ${k.owner_wa}`, k.kota].filter(Boolean).join(' · ') : ''}>
        <div className="flex flex-wrap gap-2">
          {k && bisaLihat && !k.arsip && <button className="btn-cta" onClick={() => setLihat(true)}>Lihat sebagai klien</button>}
          {k && bisaLangganan && !k.demo && <Button variant="outline" className="h-10 rounded-full" onClick={() => setDialog(true)}>Atur langganan</Button>}
          {k && bisaUbah && d?.identitas && <Button variant="outline" className="h-10 rounded-full" onClick={() => setEdit(true)}>Edit data usaha</Button>}
          {k && bisaUbah && !k.demo && !k.arsip && <Button variant="outline" className="h-10 rounded-full" onClick={() => setArsip(true)}>Arsipkan</Button>}
          {k && bisaUbah && k.arsip && <Button variant="outline" className="h-10 rounded-full" onClick={() => void pulihkanKlien(k.id).then(muat, (e: Error) => setError(e.message))}>Pulihkan dari arsip</Button>}
          {k && bisaHapus && k.arsip && <Button variant="destructive" className="h-10 rounded-full" onClick={() => setHapus(true)}>Hapus permanen</Button>}
        </div>
      </PageHeader>
      <Gagal error={error} />
      {k?.arsip && (
        <div className="card" style={{ padding: '12px 16px', borderColor: 'var(--text-danger)' }} role="status">
          <strong>Diarsipkan {formatTanggal(k.arsip)}.</strong> {k.arsip_alasan} · Semua pengguna usaha ini tidak bisa masuk sampai dipulihkan.
        </div>
      )}
      {d && k && (
        <>
          <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(170px, 1fr))' }}>
            <Kpi label="Langganan" value={k.paket ?? '—'} sub={`${STATUS[k.status][1]}${k.sampai ? ' s/d ' + formatTanggal(k.sampai) : ''}`} />
            <Kpi label="Terakhir aktif" value={formatRelatif(k.terakhir_aktif)} sub={k.daftar ? `daftar ${formatTanggal(k.daftar)}` : undefined} />
            <Kpi label="Transaksi bulan ini" value={k.transaksi_bulan_ini.toLocaleString('id-ID')} sub={`${k.user} user · profil ${k.onboarding ? 'lengkap' : 'belum diisi'}`} />
            <Kpi label="Ditangani" value={k.penangan.map((p) => p.nama).join(', ') || '—'} sub={k.klu ? `KLU ${k.klu}${k.pkp ? ' · PKP' : ''}` : undefined} />
          </div>
          <div className="grid gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(100%, 380px), 1fr))', alignItems: 'start' }}>
            <div className="card" style={{ padding: 20 }}>
              <div className="chart-title">Pengguna</div>
              <div className="table-scroll">
                <table className="data-table">
                  <thead><tr><th>Nama</th><th>Role</th><th>Terakhir aktif</th></tr></thead>
                  <tbody>
                    {d.users.map((u) => (
                      <tr key={u.id}>
                        <td><div className="font-semibold">{u.name}</div><div className="text-xs text-muted-foreground">{u.email}</div></td>
                        <td>{u.role}{!u.active && <span className="pill-status pill-red" style={{ marginLeft: 6 }}>nonaktif</span>}</td>
                        <td>{formatRelatif(u.last_seen)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
            <div className="card" style={{ padding: 20 }}>
              <div className="chart-title">Tiket</div>
              {d.tiket.length === 0 ? <p className="text-sm text-muted-foreground" style={{ paddingTop: 8 }}>Tidak ada tiket.</p> : (
                <ul className="list-rows">
                  {d.tiket.map((t) => (
                    <li key={t.id} className="list-row" style={{ gap: 12 }}>
                      <span style={{ minWidth: 0 }}><span className="tabular text-xs text-muted-foreground">{t.nomor}</span> <span className="text-sm">{t.judul}</span></span>
                      <span className={'pill-status ' + STATUS_PILL[t.status]}>{STATUS_LABEL[t.status]}</span>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>
          <div className="card" style={{ padding: 20 }}>
            <div className="chart-title">Riwayat langganan</div>
            {d.orders.length === 0 ? <p className="text-sm text-muted-foreground" style={{ paddingTop: 8 }}>Belum ada pembayaran.</p> : (
              <div className="table-scroll">
                <table className="data-table">
                  <thead><tr><th>Order</th><th>Paket</th><th style={{ textAlign: 'right' }}>Jumlah</th><th>Status</th><th>Tanggal</th></tr></thead>
                  <tbody>
                    {[...d.orders].reverse().map((o) => (
                      <tr key={o.id}>
                        <td className="tabular text-xs">{o.id}</td><td>{o.paket}</td>
                        <td className="tabular" style={{ textAlign: 'right' }}>{formatRp(o.jumlah)}</td>
                        <td><span className={'pill-status ' + (ORDER_PILL[o.status] ?? 'pill-status-muted')}>{o.status}</span></td>
                        <td className="tabular">{formatTanggal(o.waktu)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
          {bisaLihat && <AnalisaCard id={k.id} />}
          {d.koreksi.length > 0 && (
            <>
              <div className="chart-title">Usulan koreksi</div>
              {d.koreksi.map((x) => <KoreksiCard key={x.id} k={x} sub={`${x.oleh} · ${formatRelatif(x.waktu)}`} />)}
            </>
          )}
          <LihatDialog open={lihat} onClose={() => setLihat(false)} klien={k} tiket={d.tiket.filter((t) => t.status !== 'selesai').map((t) => `${t.nomor}: ${t.judul}`)} onMulai={onLihat} />
          <LanggananDialog open={dialog} onClose={() => setDialog(false)} klien={k} onSaved={() => { setDialog(false); void muat() }} />
          {d.identitas && edit && <EditUsahaDialog klienId={k.id} awal={d.identitas} onClose={() => setEdit(false)} onSaved={() => { setEdit(false); void muat() }} />}
          <AlasanDialog open={arsip} onClose={() => setArsip(false)} judul={`Arsipkan ${k.nama}`} tombol="Arsipkan"
            teks="Usaha disembunyikan dari daftar dan semua penggunanya tidak bisa masuk. Data tetap tersimpan dan bisa dipulihkan kapan saja."
            onKirim={async (alasan) => { await arsipKlien(k.id, alasan); setArsip(false); await muat() }} />
          <HapusPermanenDialog open={hapus} onClose={() => setHapus(false)} judul={`Hapus permanen ${k.nama}`}
            teks="Seluruh data usaha (transaksi, laporan, aset, invoice, karyawan) dan semua akun penggunanya dihapus dari database."
            kunci={d.identitas?.nama_usaha || k.nama} labelKunci="nama usaha"
            onHapus={async (konfirmasi) => { await hapusKlien(k.id, konfirmasi); onKembali() }} />
        </>
      )}
    </>
  )
}

/** Atur langganan klien: coba semua fitur (masa coba), aktifkan paket manual, atau perpanjang
 *  masa coba. Semua paket & masa coba membuka seluruh fitur; alasan masuk audit log. */
export function LanggananDialog({ open, onClose, klien, onSaved }: { open: boolean; onClose: () => void; klien: { id: number; nama: string }; onSaved: () => void }) {
  const [aksi, setAksi] = useState<'coba' | 'trial' | 'aktifkan'>('coba')
  const [paket, setPaket] = useState('bulanan')
  const [hari, setHari] = useState('14')
  const [alasan, setAlasan] = useState('Uji coba semua fitur')
  const [plans, setPlans] = useState<Paket[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  // Satu Idempotency-Key per buka dialog: klik ganda / retry tidak menambah hari dua kali.
  const kunci = useMemo(() => (open ? kunciBaru() : ''), [open])
  useEffect(() => { if (open) getPlans().then((r) => setPlans(r.paket), () => {}) }, [open])
  async function simpan(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await aturLangganan(klien.id, aksi === 'aktifkan' ? { aksi, paket, alasan } : { aksi: 'trial', hari: Number(hari), alasan }, kunci)
      setAlasan('')
      onSaved()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <form onSubmit={simpan} className="grid gap-4">
          <DialogHeader>
            <DialogTitle>Atur langganan · {klien.nama}</DialogTitle>
            <DialogDescription>
              Semua fitur terbuka selama masa coba atau paket aktif. Pakai <strong>Coba semua fitur</strong> untuk uji coba tanpa pembayaran;
              <strong> Aktifkan paket</strong> untuk pembayaran transfer, promo, atau kompensasi. Alasan masuk audit log.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-1.5">
            <Label htmlFor="lg-aksi">Aksi</Label>
            <Select value={aksi} onValueChange={(v) => {
              const a = v as 'coba' | 'trial' | 'aktifkan'
              setAksi(a)
              if (a === 'coba') { setHari('14'); setAlasan('Uji coba semua fitur') } else if (alasan === 'Uji coba semua fitur') setAlasan('')
            }}>
              <SelectTrigger id="lg-aksi" className="w-full"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="coba">Coba semua fitur (masa coba gratis)</SelectItem>
                <SelectItem value="aktifkan">Aktifkan paket berbayar (manual)</SelectItem>
                <SelectItem value="trial">Perpanjang masa coba</SelectItem>
              </SelectContent>
            </Select>
          </div>
          {aksi === 'aktifkan' ? (
            <div className="grid gap-1.5">
              <Label htmlFor="lg-paket">Paket</Label>
              <Select value={paket} onValueChange={setPaket}>
                <SelectTrigger id="lg-paket" className="w-full"><SelectValue /></SelectTrigger>
                <SelectContent>
                  {plans.map((p) => <SelectItem key={p.id} value={p.id}>{p.nama} · {formatRp(p.harga)} · {p.hari} hari</SelectItem>)}
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">Masa aktif bertambah dari tanggal berakhir sekarang. Tidak membuka garansi 7 hari.</p>
            </div>
          ) : (
            <div className="grid gap-1.5">
              <Label htmlFor="lg-hari">{aksi === 'coba' ? 'Lama masa coba (hari)' : 'Tambah berapa hari'}</Label>
              <Input id="lg-hari" type="number" min={1} max={90} required value={hari} onChange={(e) => setHari(e.target.value)} />
              {aksi === 'coba' && <p className="text-xs text-muted-foreground">Klien langsung bisa memakai semua menu. Setelah masa coba habis, menu kembali terkunci sampai berlangganan.</p>}
            </div>
          )}
          <div className="grid gap-1.5">
            <Label htmlFor="lg-alasan">Alasan</Label>
            <Input id="lg-alasan" required maxLength={300} placeholder="mis. Transfer BCA 20 Okt, bukti di WA" value={alasan} onChange={(e) => setAlasan(e.target.value)} />
          </div>
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

function EditUsahaDialog({ klienId, awal, onClose, onSaved }: { klienId: number; awal: IdentitasUsaha; onClose: () => void; onSaved: () => void }) {
  const [v, setV] = useState<IdentitasUsaha>(awal)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const field = (key: keyof IdentitasUsaha, label: string) => (
    <div className="grid gap-1.5">
      <Label htmlFor={`eu-${key}`}>{label}</Label>
      <Input id={`eu-${key}`} value={v[key] ?? ''} onChange={(e) => setV({ ...v, [key]: e.target.value })} required={key === 'nama_usaha'} />
    </div>
  )
  async function simpan(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await ubahKlien(klienId, v)
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
            <DialogTitle>Edit data usaha</DialogTitle>
            <DialogDescription>Identitas usaha saja. Bentuk usaha, PTKP, dan PKP tetap diubah pemilik di Profil Usaha. Perubahan tercatat di audit log.</DialogDescription>
          </DialogHeader>
          {field('nama_usaha', 'Nama usaha')}
          {field('alamat', 'Alamat')}
          {field('npwp', 'NPWP')}
          {field('telepon', 'Telepon usaha')}
          {field('email', 'Email usaha')}
          <p className="text-xs text-muted-foreground">Lokasi: {[awal.kecamatan, awal.kota, awal.provinsi].filter(Boolean).join(', ') || '—'} (diubah pemilik di Profil Usaha)</p>
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

function LihatDialog({ open, onClose, klien, tiket, onMulai }: { open: boolean; onClose: () => void; klien: KlienRow; tiket: string[]; onMulai: () => void }) {
  const [alasan, setAlasan] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  async function mulai(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await mulaiLihat(klien.id, alasan)
      onMulai()
    } catch (err) {
      setError((err as Error).message)
      setBusy(false)
    }
  }
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <form onSubmit={mulai} className="grid gap-4">
          <DialogHeader>
            <DialogTitle>Lihat sebagai {klien.nama}</DialogTitle>
            <DialogDescription>
              Anda akan melihat laporan klien apa adanya, <strong>tanpa bisa mengubah</strong>, selama 2 jam. Akses ini tercatat di audit log.
            </DialogDescription>
          </DialogHeader>
          {tiket.length > 0 && (
            <div className="flex flex-wrap gap-2">
              {tiket.map((t) => (
                <button key={t} type="button" className="pill-status pill-status-muted" onClick={() => setAlasan(`Tiket ${t}`)}>{t}</button>
              ))}
            </div>
          )}
          <div className="grid gap-1.5">
            <Label htmlFor="lh-alasan">Alasan</Label>
            <Input id="lh-alasan" required maxLength={300} placeholder="mis. Pendampingan pembukuan bulanan" value={alasan} onChange={(e) => setAlasan(e.target.value)} />
          </div>
          {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>Batal</Button>
            <Button type="submit" disabled={busy}>{busy ? 'Membuka…' : 'Mulai lihat'}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

const pctBps = (bps: number) => (bps < 0 ? '−' : '') + (Math.abs(bps) / 100).toFixed(2).replace('.', ',') + '%'
const STATUS_NPM: Record<string, [string, string]> = {
  wajar: ['pill-green', 'Setara / di atas rata-rata'], di_bawah: ['pill-red', 'Di bawah rata-rata'], kurang_data: ['pill-status-muted', 'Data belum cukup'],
}

function Ref({ r, judul }: { r: string; judul: string }) {
  return (
    <span className="pill-status tabular" title={judul}
      style={{ fontSize: 11, color: 'var(--accent-blue)', border: '1px solid color-mix(in srgb, var(--accent-blue) 40%, transparent)' }}>{r}</span>
  )
}

/** Analisa AI per klien: angka dihitung sistem, AI hanya menulis penilaian bersumber. Dijalankan
 *  atas permintaan (memanggil model & tercatat di audit log). */
function AnalisaCard({ id }: { id: number }) {
  const [a, setA] = useState<AnalisaKlien | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  async function jalankan() {
    setBusy(true)
    setError('')
    try {
      setA(await analisaKlien(id))
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }
  const f = a?.fakta
  const judulRef = (r: string) => a?.sumber.find((s) => s.ref === r)?.judul ?? (r.startsWith('BM-') ? `Benchmark nasional ${f?.klu_nama}` : '')
  return (
    <div className="card" style={{ padding: 20, display: 'grid', gridTemplateColumns: 'minmax(0, 1fr)', gap: 12 }}>
      <div className="flex flex-wrap items-center gap-3">
        <div className="chart-title">Analisa AI</div>
        <span className="text-xs text-muted-foreground">Angka dihitung sistem · AI hanya menulis penilaian dan wajib menyebut sumber</span>
        <Button size="sm" variant="outline" style={{ marginLeft: 'auto' }} disabled={busy} onClick={() => void jalankan()}>{busy ? 'Menganalisa…' : a ? 'Jalankan ulang' : 'Jalankan analisa'}</Button>
      </div>
      {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
      {f && (
        <>
          <div className="table-scroll">
            <table className="data-table tabular" style={{ minWidth: 520 }}>
              <thead><tr><th>Ukuran</th><th style={{ textAlign: 'right' }}>Klien</th><th style={{ textAlign: 'right' }}>Benchmark nasional{f.klu ? ` · KLU ${f.klu}` : ''}</th><th>Sumber</th></tr></thead>
              <tbody>
                <tr><td>Margin laba bersih (NPM)</td><td style={{ textAlign: 'right' }}>{pctBps(f.npm_bps)}</td><td style={{ textAlign: 'right' }}>{f.benchmark_npm_bps !== undefined ? pctBps(f.benchmark_npm_bps) : '—'}</td><td>{f.klu && <Ref r={`BM-${f.klu}`} judul={judulRef(`BM-${f.klu}`)} />}</td></tr>
                <tr><td>Beban pajak / omzet (CTTOR)</td><td style={{ textAlign: 'right' }} className="text-muted-foreground">—</td><td style={{ textAlign: 'right' }}>{f.benchmark_cttor_bps !== undefined ? pctBps(f.benchmark_cttor_bps) : '—'}</td><td>{f.klu && <Ref r={`BM-${f.klu}`} judul={judulRef(`BM-${f.klu}`)} />}</td></tr>
                <tr><td>Omzet setahun (rata-rata × 12)</td><td style={{ textAlign: 'right' }}>{formatRp(f.omzet_setahun)}</td><td /><td /></tr>
              </tbody>
            </table>
          </div>
          <div className="flex flex-wrap items-center gap-2 text-sm">
            <span className={'pill-status ' + STATUS_NPM[f.status_npm][0]}>{STATUS_NPM[f.status_npm][1]}</span>
            {f.klu_nama && <span className="text-muted-foreground">{f.klu_nama} · {f.sumber_benchmark}</span>}
            {f.risiko?.map((r) => <span key={r} className="pill-status pill-amber">{r}</span>)}
          </div>
          {a!.analisa.tersedia ? (
            <div className="grid gap-2">
              <p className="text-sm" style={{ borderLeft: '3px solid var(--accent-blue)', paddingLeft: 10 }}><strong>Penilaian:</strong> {a!.analisa.penilaian}</p>
              {a!.analisa.poin?.map((p, i) => (
                <div key={i} className="text-sm">
                  <strong>{p.judul}</strong> {p.sumber.map((r) => <Ref key={r} r={r} judul={judulRef(r)} />)}
                  <p className="text-muted-foreground">{p.isi}</p>
                </div>
              ))}
              <p className="text-xs text-muted-foreground">Model: {a!.analisa.model}. Penilaian AI bukan nasihat final; hal yang menyangkut SPT diverifikasi tim pajak.</p>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">{a!.analisa.alasan}</p>
          )}
          {a!.sumber.length > 0 && (
            <details className="text-sm">
              <summary className="cursor-pointer text-muted-foreground">{a!.sumber.length} sumber yang ditemukan untuk klien ini</summary>
              <ul className="mt-2 grid gap-1">{a!.sumber.map((s, i) => <li key={i}><Ref r={s.ref} judul={s.judul} /> {s.judul}</li>)}</ul>
            </details>
          )}
        </>
      )}
    </div>
  )
}
