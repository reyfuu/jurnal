import { CashWarningBanner } from '@/components/CashWarningBanner'
import { ThUrut } from '@/components/ThUrut'
import { urutkan, useKolomUrut, type Pengurut } from '@/lib/urut'
import { IsiAIBanner } from '@/components/IsiAIBanner'
import { useIsiAI, type SinyalAI } from '@/lib/useIsiAI'
import { FileSpreadsheet, Info, Landmark, Pencil, Plus } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { CatatJurnal } from '@/views/CatatJurnal'
import { JENIS_CATAT } from '@/lib/catat'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectGroup, SelectItem, SelectLabel, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
  createTransaction, deleteTransaction, getCoA, getPilihanBarang, listTransactions, updateTransaction, usulKoreksi, type CashWarning, type CoAInfo, type JenisInfo,
  type PilihanBarang, type SavedTx, type Tx, type TxInput,
} from '@/lib/api'
import { formatQty, parseQty, qtyKeTeks } from '@/lib/qty'
import { formatRp } from '@/lib/format'
import { ImportDialog } from '@/views/ImportDialog'
import { MutasiDialog } from '@/views/MutasiDialog'
import { bulanAwal, bulanWIB, channelsOf, filterTx, formatTanggal, isMasuk, labelBulan, monthsOf, summarize } from '@/lib/transaksi'
import { angka } from '@/lib/angka'

const PAGE = 100
const ALL = '__all__' // Radix Select tidak menerima value kosong
const AUTO = '__auto__' // akun otomatis dari jenis
const KUNCI_BULAN = 'mp:transaksi-bulan' // pilihan filter bulan, diingat selama tab terbuka
// Tanggal hari ini menurut WIB; toISOString() memakai UTC sehingga sebelum pukul 07.00 jatuh ke kemarin.
const today = () => new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Jakarta' })
// Isian awal transaksi yang diisi Tanya AI (dialog transaksi).
const EMPTY: TxInput = {
  tanggal: '', jenis: 'Penjualan', nominal: 0, keterangan: '', item: '', qty: 0, channel: '', pihak: '', lunas: true, tanggal_lunas: '',
}

type Editing = { id: number | null; input: TxInput; confirmDelete: boolean } | null

/** lihat: tim Master Pajak sedang "lihat sebagai klien". bolehUsul = dialog ubah menjadi form usulan
 *  koreksi; pembukuan klien baru berubah setelah pemiliknya menyetujui. */
/** aksiAwal "baru:<jenis>": langsung membuka formulir transaksi baru (dari tombol + Catat).
 *  isiAI: isian dari Tanya AI, diisi ke formulir baru (bisa diurungkan).
 *  onJurnalUmum: buka Jurnal Umum (hanya bila pengguna boleh melihatnya). */
export function Transaksi({ onChanged, lihat, aksiAwal, isiAI, onJurnalUmum }: { onChanged: () => Promise<void>; lihat?: { klienId: number; bolehUsul: boolean }; aksiAwal?: string; isiAI?: SinyalAI; onJurnalUmum?: () => void }) {
  const [txs, setTxs] = useState<Tx[]>([])
  const [jenis, setJenis] = useState<JenisInfo[]>([])
  const [imporMutasi, setImporMutasi] = useState(false)
  const [canEdit, setCanEdit] = useState(false)
  // Bawaan bulan berjalan supaya daftar tidak panjang; "Semua bulan" tetap bisa dipilih.
  const bulanIni = bulanWIB()
  const [filter, setFilter] = useState(() => {
    let tersimpan: string | null = null
    try { tersimpan = sessionStorage.getItem(KUNCI_BULAN) } catch { /* mode privat */ }
    return { bulan: bulanAwal(tersimpan, bulanIni), jenis: '', cari: '', arah: '', channel: '' }
  })
  const [limit, setLimit] = useState(PAGE)
  const [editing, setEditing] = useState<Editing>(null)
  // Catat baru & catatan jurnal multi-akun dibuka di halaman penuh (CatatJurnal), bukan dialog.
  const [halaman, setHalaman] = useState<{ jenis: string; grup?: number } | null>(() =>
    aksiAwal?.startsWith('baru') ? { jenis: aksiAwal.slice(5) || 'Penjualan' } : null)
  const [error, setError] = useState('')
  const [warnings, setWarnings] = useState<CashWarning[] | undefined>()
  const [busy, setBusy] = useState(false)
  const [importing, setImporting] = useState(false)
  const [coa, setCoa] = useState<CoAInfo | null>(null)
  const [alasan, setAlasan] = useState('')
  const [terkirim, setTerkirim] = useState('')
  const usulMode = !!lihat?.bolehUsul
  const bisaBuka = canEdit || usulMode

  const refresh = useCallback(async () => {
    const res = await listTransactions()
    setTxs(res.transactions ?? []) // usaha baru tanpa transaksi: server lama mengirim null
    setJenis(res.jenis)
    setCanEdit(res.can_edit)
  }, [])

  useEffect(() => { getCoA().then(setCoa, () => {}) }, [])
  // Barang untuk Penjualan / Beli Bahan: stok bergerak otomatis setelah kartu stok dimulai.
  const [barang, setBarang] = useState<{ mulai: string; barang: PilihanBarang[] }>({ mulai: '', barang: [] })
  const [qtyTeks, setQtyTeks] = useState('')
  useEffect(() => { getPilihanBarang().then(setBarang, () => {}) }, [])
  const autoKode = (j: string) => coa?.mapping?.find((m) => m.jenis === j)?.kode ?? ''
  const akunNama = (kode: string) => coa?.accounts?.find((a) => a.kode === kode)?.nama ?? ''
  const akunOf = (t: Pick<Tx, 'jenis' | 'akun'>) => t.akun || autoKode(t.jenis)

  useEffect(() => {
    refresh().catch((e: Error) => setError(e.message))
  }, [refresh])

  const shown = useMemo(() => filterTx(txs, filter, jenis), [txs, filter, jenis])
  const urut = useKolomUrut()
  const kolomTx: Pengurut<Tx> = {
    tanggal: (t) => t.tanggal, jenis: (t) => t.jenis, akun: (t) => t.akun || '', keterangan: (t) => t.keterangan, pihak: (t) => t.pihak,
    nominal: (t) => t.nominal, status: (t) => (t.lunas ? 1 : 0),
  }
  const sum = useMemo(() => summarize(shown, jenis), [shown, jenis])
  const posOf = useMemo(() => Object.fromEntries(jenis.map((j) => [j.nama, j.pos])), [jenis])
  // bulan terpilih ikut tampil walau transaksinya sudah tidak ada, supaya pilihan tidak kosong
  const months = useMemo(() => monthsOf(txs, ...[bulanIni, filter.bulan].filter(Boolean)), [txs, bulanIni, filter.bulan])
  const channels = useMemo(() => channelsOf(txs), [txs])

  const setF = (patch: Partial<typeof filter>) => {
    if (patch.bulan !== undefined) try { sessionStorage.setItem(KUNCI_BULAN, patch.bulan) } catch { /* mode privat */ }
    setFilter((f) => ({ ...f, ...patch }))
    setLimit(PAGE)
  }
  const set = (patch: Partial<TxInput>) => setEditing((ed) => ed && { ...ed, input: { ...ed.input, ...patch } })
  const isiAi = useIsiAI<{ editing: Editing; qtyTeks: string }, Partial<TxInput>>(isiAI, 'transaksi', {
    ambil: () => ({ editing, qtyTeks }),
    pasang: (s) => { setEditing(s.editing); setQtyTeks(s.qtyTeks) },
    isi: (isian) => {
      setError('')
      setQtyTeks('')
      getPilihanBarang().then(setBarang, () => {})
      setEditing({ id: null, input: { ...EMPTY, tanggal: today(), ...isian }, confirmDelete: false })
    },
    terbuka: editing !== null,
  })

  function openNew() {
    setHalaman({ jenis: filter.jenis && JENIS_CATAT.some((j) => j.id === filter.jenis) ? filter.jenis : 'Penjualan' })
  }
  function openEdit(t: Tx) {
    if (!bisaBuka) return
    if (t.grup && !usulMode) return setHalaman({ jenis: t.jenis, grup: t.grup })
    if (t.invoice_id) {
      setError('Transaksi ini dibuat otomatis dari invoice. Ubah lewat menu Invoice, atau tandai lunas dari menu Piutang & Utang, supaya invoice dan pembukuan tetap sama.')
      return
    }
    setError('')
    setAlasan('')
    const { id, ...input } = t
    setQtyTeks(qtyKeTeks(t.qty_stok ?? 0))
    setEditing({ id, input, confirmDelete: false })
  }

  async function run(fn: () => Promise<unknown>) {
    setBusy(true)
    setError('')
    try {
      const res = await fn()
      if (usulMode) setTerkirim(`${(res as { nomor: string }).nomor} terkirim ke pemilik usaha. Pembukuan belum berubah sampai disetujui.`)
      else setWarnings((res as SavedTx | undefined)?.warnings)
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
    if (usulMode && id !== null) return void run(() => usulKoreksi(lihat!.klienId, { transaksi_id: id, jenis: 'ubah', usulan: body, alasan }))
    void run(() => (id === null ? createTransaction(body) : updateTransaction(id, body)))
  }

  const masukJenis = jenis.filter((j) => j.arah === 'masuk')
  // Pindah Dana hanya lewat Catat (perlu rekening asal & tujuan)
  const keluarJenis = jenis.filter((j) => j.arah === 'keluar' && j.pos !== 'pindah')

  if (halaman) {
    return (
      <>
        <PageHeader title="Catat transaksi" subtitle="Isi baris jurnal (akun, debit / kredit), atau isi dari foto nota." />
        <CatatJurnal key={`${halaman.jenis}|${halaman.grup ?? ''}`} jenisAwal={halaman.jenis} grup={halaman.grup}
          onKembali={() => setHalaman(null)}
          onTersimpan={async (pesan, tanggal) => {
            setTerkirim(pesan)
            // catatan bertanggal di bulan lain: pindah ke bulannya supaya tidak terlihat hilang
            if (tanggal && filter.bulan && !tanggal.startsWith(filter.bulan)) setF({ bulan: tanggal.slice(0, 7) })
            await Promise.all([refresh(), onChanged()])
          }} />
      </>
    )
  }

  return (
    <>
      <PageHeader title="Transaksi" subtitle="Catat, ubah, dan cari transaksi" />

      {terkirim && <div className="pill-status pill-green" role="status" style={{ alignSelf: 'flex-start' }}>{terkirim}</div>}
      <div className="tx-toolbar">
        {!lihat && (
          <button className="btn-cta inline-flex items-center gap-1.5" onClick={openNew}>
            <Plus size={16} strokeWidth={2.4} /> Catat transaksi
          </button>
        )}
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
      </div>
      {/* Baris kedua: cari & filter, supaya tombol aksi tidak berdesakan dengan filter. */}
      <div className="card filter-bar">
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
      <p className="workspace-subtitle" style={{ margin: '-4px 0 0' }}>
        {filter.bulan ? `Menampilkan ${labelBulan(filter.bulan)}` : 'Menampilkan semua bulan'}
        {onJurnalUmum && <> · <button type="button" className="underline underline-offset-4" onClick={onJurnalUmum}>Lihat semua di Jurnal Umum</button></>}
      </p>

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
        <Info size={15} className="shrink-0" style={{ color: 'var(--link)', marginTop: 1 }} />
        <span>
          Angka di sini menurut <b>tanggal transaksi</b>, semua jenis. Omzet di Dashboard dan Laporan Keuangan = Penjualan saja; biaya di sana = modal
          bahan &amp; biaya. Kas masuk/keluar di Arus Kas hanya menghitung yang sudah <b>lunas</b>, jadi beda kalau ada yang belum dibayar.
        </span>
      </div>

      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Daftar transaksi</div>
        <div className="chart-subtitle">
          {usulMode ? 'Klik baris untuk mengusulkan koreksi ke pemilik usaha.'
            : canEdit ? 'Klik baris untuk mengubah, menandai lunas, atau menghapus.' : 'Ubah dan hapus transaksi hanya untuk Owner atau Admin Keuangan.'}
        </div>
        {shown.length === 0 ? (
          <p className="workspace-subtitle" style={{ padding: '24px 0' }}>
            {txs.length ? 'Tidak ada transaksi yang cocok dengan filter.' : 'Belum ada transaksi. Klik Catat transaksi untuk mulai.'}
          </p>
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                <ThUrut k="tanggal" urut={urut}>Tanggal</ThUrut><ThUrut k="jenis" urut={urut}>Jenis</ThUrut>{coa?.supported && <ThUrut k="akun" urut={urut}>Akun</ThUrut>}
                <ThUrut k="keterangan" urut={urut}>Keterangan</ThUrut><ThUrut k="pihak" urut={urut}>Pihak</ThUrut><ThUrut k="nominal" urut={urut} kanan>Nominal</ThUrut><ThUrut k="status" urut={urut}>Status</ThUrut>
                {bisaBuka && <th />}
              </tr>
            </thead>
            <tbody>
              {urutkan(shown, urut.kunci ? kolomTx[urut.kunci] : undefined, urut.arah).slice(0, limit).map((t) => {
                const masuk = isMasuk(posOf[t.jenis])
                const pindah = posOf[t.jenis] === 'pindah' // antar rekening sendiri: bukan masuk/keluar
                const sub = [t.item && (t.qty_stok ? `${t.item} × ${formatQty(t.qty_stok)}` : t.qty ? `${t.item} × ${t.qty}` : t.item), t.channel].filter(Boolean).join(' · ')
                return (
                  <tr key={t.id} className={bisaBuka ? 'tx-row' : undefined} onClick={() => openEdit(t)}>
                    <td className="tabular" style={{ whiteSpace: 'nowrap' }}>
                      {formatTanggal(t.tanggal)}
                      {t.no_jurnal && <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{t.no_jurnal}</div>}
                    </td>
                    <td><span className={'pill-status ' + (masuk ? 'pill-green' : 'pill-status-muted')}>{t.jenis}</span></td>
                    {coa?.supported && (
                      <td className="tabular" title={akunNama(akunOf(t))} style={{ color: t.akun ? 'var(--text-brand)' : 'var(--text-secondary)' }}>{akunOf(t)}</td>
                    )}
                    <td>
                      <div style={{ fontWeight: 500 }}>{t.keterangan || '-'}{!!t.invoice_id && <span className="pill-status pill-brand" style={{ marginLeft: 6, fontSize: 10 }}>dari invoice</span>}{!!t.grup && <span className="pill-status pill-brand" style={{ marginLeft: 6, fontSize: 10 }} title="Bagian dari satu catatan multi-akun">catatan #{t.grup}{t.no_bukti ? ` · ${t.no_bukti}` : ''}</span>}</div>
                      {sub && <div style={{ fontSize: 11.5, color: 'var(--text-muted)' }}>{sub}</div>}
                    </td>
                    <td style={{ color: 'var(--text-secondary)' }}>{t.pihak}</td>
                    <td className="tabular" style={{ textAlign: 'right', fontWeight: 600, color: masuk ? 'var(--text-profit)' : 'var(--text-primary)', whiteSpace: 'nowrap' }}>
                      {pindah ? '⇄ ' : masuk ? '+' : '-'}{formatRp(t.nominal).replace('Rp', '')}
                    </td>
                    <td>
                      {!t.lunas
                        ? <span className="pill-status pill-amber">Belum lunas</span>
                        : t.tanggal_lunas && <span className="pill-status pill-green">Lunas {formatTanggal(t.tanggal_lunas)}</span>}
                    </td>
                    {bisaBuka && (
                      <td style={{ textAlign: 'right' }}>
                        {!t.invoice_id && <Button variant="ghost" size="icon-sm" aria-label={`${usulMode ? 'Usulkan koreksi' : 'Ubah'} ${t.keterangan || t.jenis}`} onClick={(e) => { e.stopPropagation(); openEdit(t) }}>
                          <Pencil />
                        </Button>}
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
                <DialogTitle>{usulMode ? 'Usulkan koreksi' : editing.id === null ? 'Catat transaksi' : 'Ubah transaksi'}</DialogTitle>
                <DialogDescription>{usulMode
                  ? 'Ubah isian yang perlu dikoreksi. Pembukuan klien tidak berubah sampai pemilik usaha menyetujui.'
                  : 'Wajib: tanggal, jenis, nominal. Sisanya opsional tapi bikin laporan makin lengkap.'}</DialogDescription>
              </DialogHeader>
              <IsiAIBanner ai={isiAi} />
              <div className="grid gap-4">

              <div className="grid gap-3 sm:grid-cols-2">
                <Field id="tx-tanggal" label="Tanggal">
                  <Input id="tx-tanggal" type="date" required value={editing.input.tanggal} onChange={(e) => set({ tanggal: e.target.value })} />
                </Field>
                <Field id="tx-jenis" label="Jenis transaksi">
                  <Select value={editing.input.jenis} onValueChange={(v) => set(v === 'Penjualan' || v === 'Beli Bahan / Stok' ? { jenis: v } : { jenis: v, stok_id: undefined, qty_stok: undefined })}>
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
                    id="tx-nominal" required className="tabular"
                    {...angka(editing.input.nominal, (n) => set({ nominal: n }))}
                  />
                </Field>
                {(editing.input.jenis === 'Setor PPh Final' || editing.input.jenis === 'Setor PPN') && (
                  <Field id="tx-masa" label="Masa pajak" hint="Bulan yang dibayar pajaknya. Kosong = bulan sebelum tanggal setor.">
                    <Input id="tx-masa" type="month" max={editing.input.tanggal.slice(0, 7)} value={editing.input.masa_pajak ?? ''} onChange={(e) => set({ masa_pajak: e.target.value })} />
                  </Field>
                )}
                <Field id="tx-ket" label="Keterangan">
                  <Input id="tx-ket" placeholder="Misal: rekap kasir, belanja susu" value={editing.input.keterangan} onChange={(e) => set({ keterangan: e.target.value })} />
                </Field>
                {(() => {
                  const bolehBarang = (editing.input.jenis === 'Penjualan' || editing.input.jenis === 'Beli Bahan / Stok') && barang.barang.length > 0
                  const pilih = bolehBarang ? barang.barang.find((b) => b.id === editing.input.stok_id) : undefined
                  return <>
                    {bolehBarang && (
                      <Field id="tx-barang" label="Barang dari Stok & Gudang"
                        hint={!pilih ? 'Opsional. Pilih supaya stok & HPP ikut bergerak.'
                          : !barang.mulai ? 'Kartu stok belum dimulai, jadi stok belum bergerak.'
                            : editing.input.tanggal < barang.mulai ? `Tanggal ini sebelum titik nol stok (${barang.mulai}), jadi stok tidak bergerak.`
                              : `Stok sekarang ${formatQty(pilih.jumlah_mili)} ${pilih.satuan}.`}>
                        <select id="tx-barang" className="input-field" value={editing.input.stok_id ?? 0}
                          onChange={(e) => {
                            const b = barang.barang.find((x) => x.id === Number(e.target.value))
                            set(b ? { stok_id: b.id, item: b.nama, qty_stok: parseQty(qtyTeks) } : { stok_id: undefined, qty_stok: undefined })
                          }}>
                          <option value={0}>— tidak dari daftar stok —</option>
                          {barang.barang.map((b) => <option key={b.id} value={b.id}>{b.nama}{b.sku ? ` (${b.sku})` : ''}</option>)}
                        </select>
                      </Field>
                    )}
                    {pilih ? (
                      <Field id="tx-qty-stok" label={`Jumlah barang (${pilih.satuan || 'satuan'})`} hint="Boleh desimal, mis. 2,5">
                        <Input id="tx-qty-stok" required inputMode="decimal" className="tabular" value={qtyTeks}
                          onChange={(e) => { const v = e.target.value.replace(/[^\d.,]/g, ''); setQtyTeks(v); set({ qty_stok: parseQty(v) }) }} />
                      </Field>
                    ) : <>
                      <Field id="tx-item" label="Produk / bahan" hint="Penjualan: nama produk. Beli bahan: nama bahan.">
                        <Input id="tx-item" placeholder="Opsional" value={editing.input.item} onChange={(e) => set({ item: e.target.value })} />
                      </Field>
                      <Field id="tx-qty" label="Jumlah">
                        <Input
                          id="tx-qty" placeholder="porsi / unit" className="tabular"
                          {...angka(editing.input.qty, (n) => set({ qty: n }))}
                        />
                      </Field>
                    </>}
                  </>
                })()}
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

              {usulMode && (
                <Field id="tx-alasan" label="Alasan untuk pemilik usaha" hint="Pakai bahasa pemilik usaha, bukan istilah akuntansi. Wajib.">
                  <textarea id="tx-alasan" required rows={2} maxLength={500} value={alasan} onChange={(e) => setAlasan(e.target.value)}
                    className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 dark:bg-input/30"
                    placeholder="mis. Mesin dipakai lebih dari setahun, jadi dicatat sebagai aset" />
                </Field>
              )}

              </div>
              {error && <p className="text-sm text-destructive" role="alert">{error}</p>}

              <DialogFooter className="sm:justify-between">
                {editing.id !== null ? (
                  editing.confirmDelete ? (
                    <div className="flex items-center gap-2">
                      <span className="text-sm text-muted-foreground">{usulMode ? 'Usulkan hapus transaksi ini?' : 'Hapus transaksi ini?'}</span>
                      <Button type="button" variant="destructive" size="sm" disabled={busy || (usulMode && !alasan.trim())}
                        title={usulMode && !alasan.trim() ? 'Isi alasan dulu' : undefined}
                        onClick={() => void run(() => usulMode
                          ? usulKoreksi(lihat!.klienId, { transaksi_id: editing.id!, jenis: 'hapus', alasan })
                          : deleteTransaction(editing.id!))}>
                        {usulMode ? 'Ya, usulkan' : 'Ya, hapus'}
                      </Button>
                      <Button type="button" variant="ghost" size="sm" onClick={() => setEditing({ ...editing, confirmDelete: false })}>Batal</Button>
                    </div>
                  ) : (
                    <Button type="button" variant="ghost" className="text-destructive hover:text-destructive" onClick={() => setEditing({ ...editing, confirmDelete: true })}>
                      {usulMode ? 'Usulkan hapus' : 'Hapus'}
                    </Button>
                  )
                ) : <span />}
                <div className="flex gap-2">
                  <Button type="button" variant="outline" onClick={() => setEditing(null)}>Batal</Button>
                  <Button type="submit" disabled={busy}>{busy ? 'Menyimpan…' : usulMode ? 'Kirim usulan' : 'Simpan'}</Button>
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
