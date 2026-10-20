import { CashWarningBanner } from '@/components/CashWarningBanner'
import { FileSpreadsheet, Info, Landmark, Pencil, Plus } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectGroup, SelectItem, SelectLabel, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
  createTransaction, deleteTransaction, getCoA, listTransactions, updateTransaction, type CashWarning, type CoAInfo, type JenisInfo, type SavedTx, type Tx, type TxInput,
} from '@/lib/api'
import { formatRp } from '@/lib/format'
import { ImportDialog } from '@/views/ImportDialog'
import { MutasiDialog } from '@/views/MutasiDialog'
import { channelsOf, filterTx, formatTanggal, isMasuk, labelBulan, monthsOf, summarize } from '@/lib/transaksi'

const PAGE = 100
const ALL = '__all__' // Radix Select tidak menerima value kosong
const AUTO = '__auto__' // akun otomatis dari jenis
const today = () => new Date().toISOString().slice(0, 10)
const EMPTY: TxInput = {
  tanggal: '', jenis: 'Penjualan', nominal: 0, keterangan: '', item: '', qty: 0, channel: '', pihak: '', lunas: true, tanggal_lunas: '',
}

type Editing = { id: number | null; input: TxInput; confirmDelete: boolean } | null

export function Transaksi({ onChanged }: { onChanged: () => Promise<void> }) {
  const [txs, setTxs] = useState<Tx[]>([])
  const [jenis, setJenis] = useState<JenisInfo[]>([])
  const [imporMutasi, setImporMutasi] = useState(false)
  const [canEdit, setCanEdit] = useState(false)
  const [filter, setFilter] = useState({ bulan: '', jenis: '', cari: '', arah: '', channel: '' })
  const [limit, setLimit] = useState(PAGE)
  const [editing, setEditing] = useState<Editing>(null)
  const [error, setError] = useState('')
  const [warnings, setWarnings] = useState<CashWarning[] | undefined>()
  const [busy, setBusy] = useState(false)
  const [importing, setImporting] = useState(false)
  const [coa, setCoa] = useState<CoAInfo | null>(null)

  const refresh = useCallback(async () => {
    const res = await listTransactions()
    setTxs(res.transactions)
    setJenis(res.jenis)
    setCanEdit(res.can_edit)
  }, [])

  useEffect(() => { getCoA().then(setCoa, () => {}) }, [])
  const autoKode = (j: string) => coa?.mapping?.find((m) => m.jenis === j)?.kode ?? ''
  const akunNama = (kode: string) => coa?.accounts?.find((a) => a.kode === kode)?.nama ?? ''
  const akunOf = (t: Pick<Tx, 'jenis' | 'akun'>) => t.akun || autoKode(t.jenis)

  useEffect(() => {
    refresh().catch((e: Error) => setError(e.message))
  }, [refresh])

  const shown = useMemo(() => filterTx(txs, filter, jenis), [txs, filter, jenis])
  const sum = useMemo(() => summarize(shown, jenis), [shown, jenis])
  const posOf = useMemo(() => Object.fromEntries(jenis.map((j) => [j.nama, j.pos])), [jenis])
  const months = useMemo(() => monthsOf(txs), [txs])
  const channels = useMemo(() => channelsOf(txs), [txs])

  const setF = (patch: Partial<typeof filter>) => { setFilter((f) => ({ ...f, ...patch })); setLimit(PAGE) }
  const set = (patch: Partial<TxInput>) => setEditing((ed) => ed && { ...ed, input: { ...ed.input, ...patch } })

  function openNew() {
    setError('')
    setEditing({ id: null, input: { ...EMPTY, tanggal: today(), jenis: filter.jenis || EMPTY.jenis }, confirmDelete: false })
  }
  function openEdit(t: Tx) {
    if (!canEdit) return
    setError('')
    const { id, ...input } = t
    setEditing({ id, input, confirmDelete: false })
  }

  async function run(fn: () => Promise<unknown>) {
    setBusy(true)
    setError('')
    try {
      const res = await fn()
      setWarnings((res as SavedTx | undefined)?.warnings)
      setEditing(null)
      await Promise.all([refresh(), onChanged()])
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  function save(e: FormEvent) {
    e.preventDefault()
    if (!editing) return
    const { id, input } = editing
    const body = { ...input, tanggal_lunas: input.lunas ? input.tanggal_lunas : '' }
    void run(() => (id === null ? createTransaction(body) : updateTransaction(id, body)))
  }

  const masukJenis = jenis.filter((j) => j.arah === 'masuk')
  const keluarJenis = jenis.filter((j) => j.arah === 'keluar')

  return (
    <>
      <PageHeader title="Transaksi" subtitle="Catat, ubah, dan cari transaksi" />

      <div className="tx-toolbar">
        <button className="btn-cta inline-flex items-center gap-1.5" onClick={openNew}>
          <Plus size={16} strokeWidth={2.4} /> Catat transaksi
        </button>
        {canEdit && (
          <>
            <Button variant="outline" className="h-10 rounded-full" onClick={() => setImporting(true)}>
              <FileSpreadsheet /> Impor Excel
            </Button>
            <Button variant="outline" className="h-10 rounded-full" onClick={() => setImporMutasi(true)}>
              <Landmark /> Impor Mutasi Bank
            </Button>
          </>
        )}
        <Input
          className="tx-search"
          aria-label="Cari transaksi"
          placeholder="Cari keterangan, produk, pelanggan"
          value={filter.cari}
          onChange={(e) => setF({ cari: e.target.value })}
        />
        <Select value={filter.bulan || ALL} onValueChange={(v) => setF({ bulan: v === ALL ? '' : v })}>
          <SelectTrigger className="tx-filter" aria-label="Filter bulan"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>Semua bulan</SelectItem>
            {months.map((m) => <SelectItem key={m} value={m}>{labelBulan(m)}</SelectItem>)}
          </SelectContent>
        </Select>
        <Select value={filter.arah || ALL} onValueChange={(v) => setF({ arah: v === ALL ? '' : v })}>
          <SelectTrigger className="tx-filter" aria-label="Filter arah"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>Semua arah</SelectItem>
            <SelectItem value="masuk">Pemasukan (Masuk)</SelectItem>
            <SelectItem value="keluar">Pengeluaran (Keluar)</SelectItem>
          </SelectContent>
        </Select>
        <Select value={filter.channel || ALL} onValueChange={(v) => setF({ channel: v === ALL ? '' : v })}>
          <SelectTrigger className="tx-filter" aria-label="Filter channel"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>Semua channel</SelectItem>
            {channels.map((c) => <SelectItem key={c} value={c}>{c}</SelectItem>)}
          </SelectContent>
        </Select>
        <Select value={filter.jenis || ALL} onValueChange={(v) => setF({ jenis: v === ALL ? '' : v })}>
          <SelectTrigger className="tx-filter" aria-label="Filter jenis"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>Semua jenis</SelectItem>
            {jenis.map((j) => <SelectItem key={j.nama} value={j.nama}>{j.nama}</SelectItem>)}
          </SelectContent>
        </Select>
      </div>

      <CashWarningBanner warnings={warnings} />
      {error && !editing && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start' }}>{error}</div>}

      <div className="kpi-grid" style={{ gridTemplateColumns: 'repeat(3, minmax(0, 1fr))' }}>
        <div className="card">
          <div className="kpi-label">Transaksi tampil</div>
          <div className="kpi-value tabular">{sum.count.toLocaleString('id-ID')}</div>
          <div className="kpi-meta-row"><span>dari {txs.length.toLocaleString('id-ID')} transaksi tercatat</span></div>
        </div>
        <div className="card">
          <div className="kpi-label">Total masuk (tampil)</div>
          <div className="kpi-value tabular" style={{ color: 'var(--text-profit)' }}>{formatRp(sum.masuk)}</div>
          <div className="kpi-meta-row">
            <span>Penjualan {formatRp(sum.omzet)}{sum.masuk > sum.omzet && ` · modal, pinjaman, pendapatan lain ${formatRp(sum.masuk - sum.omzet)}`}</span>
          </div>
        </div>
        <div className="card">
          <div className="kpi-label">Total keluar (tampil)</div>
          <div className="kpi-value tabular" style={{ color: 'var(--text-danger)' }}>{formatRp(sum.keluar)}</div>
          <div className="kpi-meta-row">
            <span>Modal bahan &amp; biaya {formatRp(sum.biayaUsaha)}{sum.keluar > sum.biayaUsaha && ` · alat, cicilan, ambil pribadi ${formatRp(sum.keluar - sum.biayaUsaha)}`}</span>
          </div>
        </div>
      </div>

      <div className="tx-note">
        <Info size={15} className="shrink-0" style={{ color: 'var(--accent-amber)', marginTop: 1 }} />
        <span>
          Angka di sini menurut <b>tanggal transaksi</b>, semua jenis. Omzet di Dashboard dan Laporan Keuangan = Penjualan saja; biaya di sana = modal
          bahan &amp; biaya. Kas masuk/keluar di Arus Kas hanya menghitung yang sudah <b>lunas</b>, jadi beda kalau ada yang belum dibayar.
        </span>
      </div>

      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Daftar transaksi</div>
        <div className="chart-subtitle">
          {canEdit ? 'Klik baris untuk mengubah, menandai lunas, atau menghapus.' : 'Ubah dan hapus transaksi hanya untuk Owner atau Admin Keuangan.'}
        </div>
        {shown.length === 0 ? (
          <p className="workspace-subtitle" style={{ padding: '24px 0' }}>
            {txs.length ? 'Tidak ada transaksi yang cocok dengan filter.' : 'Belum ada transaksi. Klik Catat transaksi untuk mulai.'}
          </p>
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                <th>Tanggal</th><th>Jenis</th>{coa?.supported && <th>Akun</th>}<th>Keterangan</th><th>Pihak</th><th style={{ textAlign: 'right' }}>Nominal</th><th>Status</th>
                {canEdit && <th />}
              </tr>
            </thead>
            <tbody>
              {shown.slice(0, limit).map((t) => {
                const masuk = isMasuk(posOf[t.jenis])
                const sub = [t.item && (t.qty ? `${t.item} × ${t.qty}` : t.item), t.channel].filter(Boolean).join(' · ')
                return (
                  <tr key={t.id} className={canEdit ? 'tx-row' : undefined} onClick={() => openEdit(t)}>
                    <td className="tabular" style={{ whiteSpace: 'nowrap' }}>{formatTanggal(t.tanggal)}</td>
                    <td><span className={'pill-status ' + (masuk ? 'pill-green' : 'pill-status-muted')}>{t.jenis}</span></td>
                    {coa?.supported && (
                      <td className="tabular" title={akunNama(akunOf(t))} style={{ color: t.akun ? 'var(--text-warn)' : 'var(--text-secondary)' }}>{akunOf(t)}</td>
                    )}
                    <td>
                      <div style={{ fontWeight: 500 }}>{t.keterangan || '-'}</div>
                      {sub && <div style={{ fontSize: 11.5, color: 'var(--text-muted)' }}>{sub}</div>}
                    </td>
                    <td style={{ color: 'var(--text-secondary)' }}>{t.pihak}</td>
                    <td className="tabular" style={{ textAlign: 'right', fontWeight: 600, color: masuk ? 'var(--text-profit)' : 'var(--text-primary)', whiteSpace: 'nowrap' }}>
                      {masuk ? '+' : '-'}{formatRp(t.nominal).replace('Rp', '')}
                    </td>
                    <td>
                      {!t.lunas
                        ? <span className="pill-status pill-amber">Belum lunas</span>
                        : t.tanggal_lunas && <span className="pill-status pill-green">Lunas {formatTanggal(t.tanggal_lunas)}</span>}
                    </td>
                    {canEdit && (
                      <td style={{ textAlign: 'right' }}>
                        <Button variant="ghost" size="icon-sm" aria-label={`Ubah ${t.keterangan || t.jenis}`} onClick={(e) => { e.stopPropagation(); openEdit(t) }}>
                          <Pencil />
                        </Button>
                      </td>
                    )}
                  </tr>
                )
              })}
            </tbody>
          </table>
        )}
        {shown.length > limit && (
          <div style={{ textAlign: 'center', marginTop: 14 }}>
            <Button variant="outline" size="sm" onClick={() => setLimit((l) => l + PAGE)}>
              Tampilkan {Math.min(PAGE, shown.length - limit)} lagi dari {(shown.length - limit).toLocaleString('id-ID')}
            </Button>
          </div>
        )}
      </div>

      <Dialog open={editing !== null} onOpenChange={(open) => !open && setEditing(null)}>
        <DialogContent className="sm:max-w-2xl">
          {editing && (
            <form onSubmit={save} className="grid gap-4">
              <DialogHeader>
                <DialogTitle>{editing.id === null ? 'Catat transaksi' : 'Ubah transaksi'}</DialogTitle>
                <DialogDescription>Wajib: tanggal, jenis, nominal. Sisanya opsional tapi bikin laporan makin lengkap.</DialogDescription>
              </DialogHeader>

              <div className="grid gap-3 sm:grid-cols-2">
                <Field id="tx-tanggal" label="Tanggal">
                  <Input id="tx-tanggal" type="date" required value={editing.input.tanggal} onChange={(e) => set({ tanggal: e.target.value })} />
                </Field>
                <Field id="tx-jenis" label="Jenis transaksi">
                  <Select value={editing.input.jenis} onValueChange={(v) => set({ jenis: v })}>
                    <SelectTrigger id="tx-jenis" className="w-full"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectGroup>
                        <SelectLabel>Uang masuk</SelectLabel>
                        {masukJenis.map((j) => <SelectItem key={j.nama} value={j.nama}>{j.nama}</SelectItem>)}
                      </SelectGroup>
                      <SelectGroup>
                        <SelectLabel>Uang keluar</SelectLabel>
                        {keluarJenis.map((j) => <SelectItem key={j.nama} value={j.nama}>{j.nama}</SelectItem>)}
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </Field>
                {coa?.supported && (
                  <Field id="tx-akun" label="Akun (CoA)">
                    <Select value={editing.input.akun || AUTO} onValueChange={(v) => set({ akun: v === AUTO ? '' : v })}>
                      <SelectTrigger id="tx-akun" className="w-full"><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value={AUTO}>Otomatis: {autoKode(editing.input.jenis)} {akunNama(autoKode(editing.input.jenis))}</SelectItem>
                        {coa.kelompok.map((k) => {
                          const accs = coa.accounts!.filter((a) => a.kelompok === k)
                          return accs.length ? (
                            <SelectGroup key={k}>
                              <SelectLabel>{k}</SelectLabel>
                              {accs.map((a) => <SelectItem key={a.kode} value={a.kode}>{a.kode} {a.nama}</SelectItem>)}
                            </SelectGroup>
                          ) : null
                        })}
                      </SelectContent>
                    </Select>
                  </Field>
                )}
                <Field id="tx-nominal" label="Nominal (Rp)">
                  <Input
                    id="tx-nominal" type="number" inputMode="numeric" min={1} required className="tabular"
                    value={editing.input.nominal || ''} onChange={(e) => set({ nominal: Number(e.target.value) })}
                  />
                </Field>
                <Field id="tx-ket" label="Keterangan">
                  <Input id="tx-ket" placeholder="Misal: rekap kasir, belanja susu" value={editing.input.keterangan} onChange={(e) => set({ keterangan: e.target.value })} />
                </Field>
                <Field id="tx-item" label="Produk / bahan" hint="Penjualan: nama produk. Beli bahan: nama bahan.">
                  <Input id="tx-item" placeholder="Opsional" value={editing.input.item} onChange={(e) => set({ item: e.target.value })} />
                </Field>
                <Field id="tx-qty" label="Jumlah">
                  <Input
                    id="tx-qty" type="number" min={0} placeholder="porsi / unit" className="tabular"
                    value={editing.input.qty || ''} onChange={(e) => set({ qty: Number(e.target.value) })}
                  />
                </Field>
                <Field id="tx-channel" label="Channel">
                  <Input id="tx-channel" placeholder="Opsional, misal GoFood" value={editing.input.channel} onChange={(e) => set({ channel: e.target.value })} />
                </Field>
                <Field id="tx-pihak" label="Pelanggan / supplier">
                  <Input id="tx-pihak" placeholder="Opsional" value={editing.input.pihak} onChange={(e) => set({ pihak: e.target.value })} />
                </Field>
                <Field id="tx-status" label="Status bayar">
                  <Select value={editing.input.lunas ? 'lunas' : 'belum'} onValueChange={(v) => set({ lunas: v === 'lunas' })}>
                    <SelectTrigger id="tx-status" className="w-full"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="lunas">Lunas</SelectItem>
                      <SelectItem value="belum">Belum Lunas</SelectItem>
                    </SelectContent>
                  </Select>
                </Field>
                {editing.input.lunas && (
                  <Field id="tx-lunas" label="Tanggal lunas" hint="Isi hanya kalau lunas tapi dibayar belakangan.">
                    <Input id="tx-lunas" type="date" min={editing.input.tanggal} value={editing.input.tanggal_lunas} onChange={(e) => set({ tanggal_lunas: e.target.value })} />
                  </Field>
                )}
              </div>

              {error && <p className="text-sm text-destructive" role="alert">{error}</p>}

              <DialogFooter className="sm:justify-between">
                {editing.id !== null ? (
                  editing.confirmDelete ? (
                    <div className="flex items-center gap-2">
                      <span className="text-sm text-muted-foreground">Hapus transaksi ini?</span>
                      <Button type="button" variant="destructive" size="sm" disabled={busy} onClick={() => void run(() => deleteTransaction(editing.id!))}>
                        Ya, hapus
                      </Button>
                      <Button type="button" variant="ghost" size="sm" onClick={() => setEditing({ ...editing, confirmDelete: false })}>Batal</Button>
                    </div>
                  ) : (
                    <Button type="button" variant="ghost" className="text-destructive hover:text-destructive" onClick={() => setEditing({ ...editing, confirmDelete: true })}>
                      Hapus
                    </Button>
                  )
                ) : <span />}
                <div className="flex gap-2">
                  <Button type="button" variant="outline" onClick={() => setEditing(null)}>Batal</Button>
                  <Button type="submit" disabled={busy}>{busy ? 'Menyimpan…' : 'Simpan'}</Button>
                </div>
              </DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>

      <ImportDialog open={importing} onOpenChange={setImporting} onImported={() => Promise.all([refresh(), onChanged()]).then(() => {})} />
      <MutasiDialog
        open={imporMutasi} onOpenChange={setImporMutasi} jenis={jenis}
        onImported={() => { void Promise.all([refresh(), onChanged()]) }}
      />
    </>
  )
}

function Field({ id, label, hint, children }: { id: string; label: string; hint?: string; children: React.ReactNode }) {
  return (
    <div className="grid content-start gap-1.5">
      <Label htmlFor={id}>{label}</Label>
      {children}
      {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
    </div>
  )
}
