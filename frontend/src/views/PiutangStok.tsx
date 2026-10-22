import { Flag, ShoppingCart, Upload } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { IsiAIBanner } from '@/components/IsiAIBanner'
import { useIsiAI, type SinyalAI } from '@/lib/useIsiAI'
import { MenuImportDialog } from '@/views/MenuImportDialog'
import { BeliBahanDialog, KartuBarangDialog, MutasiStokPanel, RiwayatGerakPanel } from '@/views/KartuStok'
import { StokOpnamePanel } from '@/views/StokOpname'
import { TambahBarangBanyak } from '@/views/TambahBarang'
import { ThUrut } from '@/components/ThUrut'
import { urutkan, useKolomUrut, type Pengurut } from '@/lib/urut'
import { useGulirKeForm } from '@/lib/gulir'
import { buatSKU } from '@/lib/sku'
import { formatQty } from '@/lib/qty'
import {
  createStok, deleteStok, deleteTagihan, importStok,
  STOK_TEMPLATE_URL, getStok, getTagihan, updateStok, updateTagihan,
  type StokInput, type StokRow, type StokSummary, type TagihanInput, type TagihanRow, type TagihanSummary,
} from '@/lib/api'
import { formatRp, formatTanggal } from '@/lib/format'
import { angka } from '@/lib/angka'

const RED = 'var(--text-danger)'
const GREEN = 'var(--text-profit)'
const BLUE = 'var(--accent-blue)'

const selectStyle = {
  height: 38, background: 'var(--bg-input)', border: '1px solid var(--border-strong)',
  color: 'var(--text-primary)', borderRadius: 8, padding: '0 12px', fontSize: 13,
} as const

const inputStyle = { flex: '1 1 240px', minWidth: 200, height: 38, background: 'var(--bg-input)' } as const

const aksiBtn = { width: 'auto', padding: '4px 8px', fontSize: 12 } as const

/** Baris tabel memuat turunan dari server; form hanya boleh berisi field yang bisa disunting. */
const toTagihanInput = ({ nomor, pihak, jenis, jatuh_tempo, nominal, lunas, keterangan }: TagihanRow): TagihanInput =>
  ({ nomor, pihak, jenis, jatuh_tempo, nominal, lunas, keterangan })

const toStokInput = ({ sku, nama, kategori, jumlah, satuan, stok_min, harga_pokok, harga_jual }: StokRow): StokInput =>
  ({ sku, nama, kategori, jumlah, satuan, stok_min, harga_pokok, harga_jual })

/**
 * Tombol Ubah & Hapus di tiap baris, sama bentuknya di kedua halaman.
 *
 * Pembungkus flex-nya perlu: `.nav-item` sendiri `display: flex`, jadi tanpa ini kedua tombol
 * jadi block dan menumpuk ke bawah alih-alih berdampingan.
 */
function Aksi({ onEdit, onDelete }: { onEdit: () => void; onDelete: () => void }) {
  return (
    <td>
      <div style={{ display: 'flex', gap: 4 }}>
        <button className="nav-item" style={aksiBtn} onClick={onEdit}>Ubah</button>
        <button className="nav-item" style={{ ...aksiBtn, color: RED }} onClick={onDelete}>Hapus</button>
      </div>
    </td>
  )
}

function Kpi({ label, value, color, meta }: { label: string; value: string; color?: string; meta: string }) {
  return (
    <div className="card" style={{ padding: '14px 18px' }}>
      <div className="kpi-label">{label}</div>
      <div className="kpi-value tabular" style={{ fontSize: 20, color }}>{value}</div>
      <div className="kpi-meta-row"><span>{meta}</span></div>
    </div>
  )
}

// --- Piutang & Utang ---

/**
 * Tagihan manual (lama): daftar yang dulu diketik terpisah dan tidak masuk pembukuan. Tidak bisa
 * ditambah lagi — tagihan baru diterbitkan lewat Invoice atau dicatat sebagai transaksi belum
 * lunas — tetapi yang lama masih bisa diubah, ditandai lunas, atau dihapus.
 */
export function PiutangUtang() {
  const [data, setData] = useState<TagihanSummary | null>(null)
  const [form, setForm] = useState<TagihanInput | null>(null)
  const formRefTagihan = useRef<HTMLFormElement>(null)
  useGulirKeForm(formRefTagihan, !!form)
  const [editId, setEditId] = useState<number | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [cari, setCari] = useState('')
  const [jenisFilter, setJenisFilter] = useState('ALL')
  const [statusFilter, setStatusFilter] = useState('ALL')

  const muat = useCallback(() => { getTagihan().then(setData, (e: Error) => setError(e.message)) }, [])
  useEffect(muat, [muat])

  async function simpan(e: React.FormEvent) {
    e.preventDefault()
    if (!form || editId === null) return
    setBusy(true)
    try {
      await updateTagihan(editId, form)
      setForm(null)
      setEditId(null)
      muat()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  async function hapus(t: TagihanRow) {
    if (!confirm(`Hapus tagihan ${t.nomor || t.pihak} sebesar ${formatRp(t.nominal)}?`)) return
    try {
      await deleteTagihan(t.id)
      muat()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  if (!data) {
    return error
      ? <p className="workspace-subtitle" role="alert">{error}</p>
      : <p className="workspace-subtitle">Memuat data…</p>
  }

  const list = data.items ?? []
  const tampil = list.filter((t) => {
    if (jenisFilter !== 'ALL' && t.jenis !== jenisFilter) return false
    if (statusFilter === 'TERLAMBAT' && t.hari_terlambat === 0) return false
    if (statusFilter === 'LANCAR' && (t.hari_terlambat > 0 || t.lunas)) return false
    if (statusFilter === 'LUNAS' && !t.lunas) return false
    const q = cari.trim().toLowerCase()
    if (q && !t.nomor.toLowerCase().includes(q) && !t.pihak.toLowerCase().includes(q)) return false
    return true
  })

  const statusTagihan = (t: TagihanRow) => {
    if (t.lunas) return { label: 'Lunas', cls: 'pill-status-muted' }
    if (t.hari_terlambat > 0) return { label: `Lewat ${t.hari_terlambat} hari`, cls: 'pill-red' }
    return { label: 'Belum jatuh tempo', cls: 'pill-green' }
  }

  return (
    <>
      <div className="note-box warn" style={{ marginBottom: 12 }}>
        Daftar lama ini diketik terpisah dan <strong>tidak masuk laporan keuangan</strong>. Tagihan baru: terbitkan lewat menu <strong>Invoice</strong> (piutang)
        atau catat pembelian <em>belum lunas</em> di Transaksi (utang). Yang di sini tinggal ditandai lunas atau dihapus setelah dicatat ulang.
      </div>
      <PageHeader title="Tagihan manual (lama)" subtitle="Hanya baca, ubah, dan hapus; tidak bisa ditambah lagi." />

      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', marginBottom: 12 }}>{error}</div>}

      <div className="kpi-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(210px, 1fr))', marginBottom: 16 }}>
        <Kpi label="Piutang Belum Tertagih" value={formatRp(data.total_piutang)} color={GREEN} meta="pelanggan belum bayar" />
        <Kpi label="Utang Belum Dibayar" value={formatRp(data.total_utang)} color={RED} meta="kewajiban ke supplier" />
        <Kpi
          label="Piutang Sudah Terlambat" value={formatRp(data.piutang_terlambat)} color="var(--text-warn)"
          meta={`${data.jumlah_terlambat} tagihan lewat jatuh tempo`}
        />
        <Kpi label="Tagihan Tercatat" value={String(list.length)} meta={`${tampil.length} tampil sesuai filter`} />
      </div>

      <div className="card" style={{ padding: '16px 20px', marginBottom: 16 }}>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12, alignItems: 'center' }}>
          <input className="input-field" style={inputStyle} placeholder="Cari no. dokumen, pelanggan, supplier…" value={cari} onChange={(e) => setCari(e.target.value)} />
          <select style={selectStyle} value={jenisFilter} onChange={(e) => setJenisFilter(e.target.value)} aria-label="Jenis tagihan">
            <option value="ALL">Semua Jenis</option>
            <option value="RECEIVABLE">Piutang (dari pelanggan)</option>
            <option value="PAYABLE">Utang (ke supplier)</option>
          </select>
          <select style={selectStyle} value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} aria-label="Status jatuh tempo">
            <option value="ALL">Semua Status</option>
            <option value="TERLAMBAT">Sudah lewat jatuh tempo</option>
            <option value="LANCAR">Belum jatuh tempo</option>
            <option value="LUNAS">Sudah lunas</option>
          </select>
        </div>
      </div>

      {form && (
        <form ref={formRefTagihan} className="card" style={{ padding: 20, marginBottom: 14, scrollMarginTop: 16 }} onSubmit={simpan}>
          <div className="chart-title">{editId === null ? 'Tambah Tagihan' : 'Ubah Tagihan'}</div>
          <div className="form-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))' }}>
            <label>
              <span className="kpi-label">Jenis</span>
              <select className="input-field" value={form.jenis} onChange={(e) => setForm({ ...form, jenis: e.target.value as TagihanInput['jenis'] })}>
                <option value="RECEIVABLE">Piutang — pelanggan berutang ke kita</option>
                <option value="PAYABLE">Utang — kita berutang ke supplier</option>
              </select>
            </label>
            <label>
              <span className="kpi-label">{form.jenis === 'RECEIVABLE' ? 'No. Invoice' : 'No. PO'}</span>
              <input className="input-field" value={form.nomor} onChange={(e) => setForm({ ...form, nomor: e.target.value })} />
            </label>
            <label>
              <span className="kpi-label">{form.jenis === 'RECEIVABLE' ? 'Pelanggan' : 'Supplier'}</span>
              <input className="input-field" required value={form.pihak} onChange={(e) => setForm({ ...form, pihak: e.target.value })} />
            </label>
            <label>
              <span className="kpi-label">Jatuh Tempo</span>
              <input className="input-field" type="date" required value={form.jatuh_tempo} onChange={(e) => setForm({ ...form, jatuh_tempo: e.target.value })} />
            </label>
            <label>
              <span className="kpi-label">Nominal</span>
              <input
                className="input-field tabular" required {...angka(form.nominal, (n) => setForm({ ...form, nominal: n }))}
              />
            </label>
            <label>
              <span className="kpi-label">Status Pembayaran</span>
              <select className="input-field" value={form.lunas ? 'lunas' : 'belum'} onChange={(e) => setForm({ ...form, lunas: e.target.value === 'lunas' })}>
                <option value="belum">Belum lunas</option>
                <option value="lunas">Sudah lunas</option>
              </select>
            </label>
            <label>
              <span className="kpi-label">Keterangan</span>
              <input className="input-field" value={form.keterangan ?? ''} onChange={(e) => setForm({ ...form, keterangan: e.target.value })} />
            </label>
          </div>
          <div style={{ display: 'flex', gap: 10, marginTop: 16 }}>
            <button className="btn-cta" style={{ padding: '9px 20px', fontSize: 13 }} type="submit" disabled={busy}>
              {busy ? 'Menyimpan…' : 'Simpan'}
            </button>
            <button type="button" className="nav-item" style={{ width: 'auto', padding: '9px 16px' }} onClick={() => setForm(null)}>Batal</button>
          </div>
        </form>
      )}

      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Daftar Piutang & Utang</div>
        <div className="chart-subtitle">Bisa diisi sekaligus lewat tombol Impor Excel, atau lewat Pengaturan → Impor Data Awal</div>
        <div className="table-scroll">
          <table className="data-table tabular" style={{ minWidth: 860 }}>
            <thead>
              <tr>
                <th>No. Dokumen</th><th>Pelanggan / Supplier</th><th>Jenis</th><th>Jatuh Tempo</th>
                <th style={{ textAlign: 'right' }}>Nominal</th><th>Status</th><th>Keterangan</th><th />
              </tr>
            </thead>
            <tbody>
              {tampil.length === 0 && (
                <tr>
                  <td colSpan={8} style={{ color: 'var(--text-muted)' }}>
                    {list.length === 0 ? 'Tidak ada tagihan manual.' : 'Tidak ada tagihan yang cocok dengan filter.'}
                  </td>
                </tr>
              )}
              {tampil.map((t) => {
                const st = statusTagihan(t)
                return (
                  <tr key={t.id}>
                    <td style={{ fontWeight: 600 }}>{t.nomor || '—'}</td>
                    <td>{t.pihak}</td>
                    <td>{t.jenis === 'RECEIVABLE' ? 'Piutang' : 'Utang'}</td>
                    <td>{t.jatuh_tempo}</td>
                    <td style={{ textAlign: 'right', fontWeight: 700, color: t.jenis === 'RECEIVABLE' ? GREEN : RED }}>{formatRp(t.nominal)}</td>
                    <td><span className={'pill-status ' + st.cls} style={{ fontSize: 10 }}>{st.label}</span></td>
                    <td style={{ color: 'var(--text-muted)' }}>{t.keterangan || '—'}</td>
                    <Aksi onEdit={() => { setError(''); setEditId(t.id); setForm(toTagihanInput(t)) }} onDelete={() => void hapus(t)} />
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </div>

    </>
  )
}

// --- Stok & Gudang ---

const KOLOM_STOK: Pengurut<StokRow> = {
  sku: (r) => r.sku, nama: (r) => r.nama, kategori: (r) => r.kategori, jumlah: (r) => r.jumlah_mili, min: (r) => r.stok_min,
  hp: (r) => r.harga_pokok, hj: (r) => r.harga_jual, margin: (r) => r.margin_bps, nilai: (r) => r.nilai, status: (r) => r.status,
}

const STATUS_CLS: Record<StokRow['status'], string> = {
  Minus: 'pill-red',
  Habis: 'pill-red',
  'Stok Menipis': 'pill-amber',
  Aman: 'pill-green',
}

/** bisaUbah: false di mode lihat-sebagai (hanya baca). */
/** isiAI: isian dari Tanya AI untuk barang baru (bisa diurungkan). */
export function StokGudang({ bisaUbah = true, aksi, isiAI }: { bisaUbah?: boolean; aksi?: string; isiAI?: SinyalAI }) {
  const [data, setData] = useState<StokSummary | null>(null)
  // aksi dari tombol + Catat / Langkah memulai: stok awal & opname langsung ke tab Stok opname.
  const [tab, setTab] = useState<'daftar' | 'opname' | 'mutasi' | 'riwayat'>(() => (aksi === 'mulai' || aksi === 'opname' ? 'opname' : 'daftar'))
  const [dialog, setDialog] = useState<'beli' | null>(() => (bisaUbah && aksi === 'beli' ? 'beli' : null))
  const [pesan, setPesan] = useState('')
  const [kartu, setKartu] = useState<StokRow | null>(null)
  const [form, setForm] = useState<StokInput | null>(null)
  const [editId, setEditId] = useState<number | null>(null)
  const [impor, setImpor] = useState(false)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [cari, setCari] = useState('')
  const [kategoriFilter, setKategoriFilter] = useState('ALL')
  const [statusFilter, setStatusFilter] = useState('ALL')
  const [tambahBanyak, setTambahBanyak] = useState(false)
  const urut = useKolomUrut()
  const formRef = useRef<HTMLDivElement>(null)
  useGulirKeForm(formRef, tambahBanyak || !!form)

  const muat = useCallback(() => { getStok().then(setData, (e: Error) => setError(e.message)) }, [])
  useEffect(muat, [muat])
  const isiAi = useIsiAI<{ tab: typeof tab; form: StokInput | null; editId: number | null; tambahBanyak: boolean }, Partial<StokInput>>(isiAI, 'stok', {
    ambil: () => ({ tab, form, editId, tambahBanyak }),
    pasang: (s) => { setTab(s.tab); setForm(s.form); setEditId(s.editId); setTambahBanyak(s.tambahBanyak) },
    isi: (isian) => {
      setError(''); setTab('daftar'); setEditId(null); setTambahBanyak(false)
      const b: StokInput = { sku: '', nama: '', kategori: 'Bahan Baku', jumlah: 0, satuan: 'pcs', stok_min: 0, harga_pokok: 0, harga_jual: 0, ...isian }
      // SKU otomatis dari nama, sama seperti tabel Tambah barang
      if (!b.sku) b.sku = buatSKU(b.nama, new Set((data?.items ?? []).map((x) => x.sku.toUpperCase()).filter(Boolean)))
      setForm(b)
    },
    terbuka: form !== null,
  })

  async function simpan(e: React.FormEvent) {
    e.preventDefault()
    if (!form) return
    setBusy(true)
    try {
      await (editId === null ? createStok(form) : updateStok(editId, form))
      setForm(null)
      setEditId(null)
      muat()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  async function hapus(it: StokRow) {
    if (!confirm(`Hapus ${it.nama} dari daftar stok?`)) return
    try {
      await deleteStok(it.id)
      muat()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  if (!data) {
    return error
      ? <p className="workspace-subtitle" role="alert">{error}</p>
      : <p className="workspace-subtitle">Memuat data…</p>
  }

  const list = data.items ?? []
  const mulai = data.mulai
  const editDilacak = editId !== null && !!list.find((x) => x.id === editId)?.dilacak
  const selesaiDialog = () => { setDialog(null); muat() }
  const kategoris = [...new Set(list.map((it) => it.kategori))].sort()
  const tampil = list.filter((it) => {
    if (kategoriFilter !== 'ALL' && it.kategori !== kategoriFilter) return false
    if (statusFilter !== 'ALL' && it.status !== statusFilter) return false
    const q = cari.trim().toLowerCase()
    if (q && !it.sku.toLowerCase().includes(q) && !it.nama.toLowerCase().includes(q)) return false
    return true
  })
  const terurut = urutkan(tampil, urut.kunci ? KOLOM_STOK[urut.kunci] : undefined, urut.arah)

  return (
    <>
      <PageHeader title="Stok & Gudang" subtitle={mulai
        ? `Kartu stok aktif sejak ${formatTanggal(mulai)}: stok bergerak otomatis dari beli, jual, produksi & opname.`
        : 'Daftar bahan baku & produk jadi, nilai persediaan, dan peringatan stok menipis.'}>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          {bisaUbah && mulai && <>
            <button className="btn-cta accent" style={{ padding: '8px 16px', fontSize: 13, display: 'inline-flex', alignItems: 'center', gap: 6 }} onClick={() => setDialog('beli')} disabled={list.length === 0}>
              <ShoppingCart size={15} aria-hidden /> Beli bahan
            </button>
          </>}
          <button className="nav-item" style={{ width: 'auto', padding: '8px 14px', display: 'inline-flex', alignItems: 'center', gap: 6 }} onClick={() => setImpor(true)}>
            <Upload size={15} aria-hidden /> Impor Excel
          </button>
        </div>
      </PageHeader>

      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', marginBottom: 12 }}>{error}</div>}

      {!mulai && (
        <div className="card" style={{ padding: 20, marginBottom: 16, borderColor: 'var(--accent-blue)' }}>
          <div className="flex flex-wrap items-center gap-4">
            <Flag size={22} style={{ color: BLUE, flexShrink: 0 }} aria-hidden />
            <div style={{ flex: '1 1 320px' }}>
              <div className="chart-title">Mulai kartu stok supaya jumlah & nilai stok akurat</div>
              <div className="chart-subtitle">
                Sekarang jumlah stok masih diketik manual. Setelah titik nol (hitung fisik pertama), stok bertambah dari pembelian & produksi,
                berkurang dari penjualan, dan dicocokkan lewat stok opname. Disarankan untuk usaha dagang & produksi; usaha jasa boleh melewatinya.
              </div>
            </div>
            {bisaUbah && <button className="btn-cta" style={{ padding: '9px 18px', fontSize: 13 }} disabled={list.length === 0} onClick={() => setTab('opname')}>Hitung stok awal</button>}
          </div>
        </div>
      )}

      {pesan && <div className="pill-status pill-green" role="status" style={{ alignSelf: 'flex-start', marginBottom: 12, whiteSpace: 'normal' }}>{pesan}</div>}
      <div className="tabbar" role="tablist" aria-label="Bagian stok" style={{ marginBottom: 12 }}>
        <button role="tab" className="tab-btn" aria-selected={tab === 'daftar'} onClick={() => setTab('daftar')}>Daftar barang</button>
        <button role="tab" className="tab-btn" aria-selected={tab === 'opname'} onClick={() => { setPesan(''); setTab('opname') }}>{mulai ? 'Stok opname' : 'Stok opname / saldo awal'}</button>
        {mulai && <button role="tab" className="tab-btn" aria-selected={tab === 'mutasi'} onClick={() => setTab('mutasi')}>Laporan mutasi</button>}
        {mulai && <button role="tab" className="tab-btn" aria-selected={tab === 'riwayat'} onClick={() => setTab('riwayat')}>Riwayat opname & produksi</button>}
      </div>
      {tab === 'opname' && <StokOpnamePanel key={mulai + list.length} items={list} mulai={mulai} bisaUbah={bisaUbah}
        onDone={(m) => { setPesan(m); setTab('daftar'); muat() }} />}
      {mulai && tab === 'mutasi' && <MutasiStokPanel mulai={mulai} bisaEkspor={bisaUbah} />}
      {mulai && tab === 'riwayat' && <RiwayatGerakPanel items={list} bisaUbah={bisaUbah} onChanged={muat} />}

      {tab === 'daftar' && <>
      <div className="kpi-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(210px, 1fr))', marginBottom: 16 }}>
        <Kpi label="Item Tercatat" value={String(data.total_item)} meta={`${tampil.length} tampil sesuai filter`} />
        <Kpi label="Nilai Persediaan" value={formatRp(data.nilai_persediaan)} color={BLUE} meta="jumlah × harga pokok" />
        <Kpi label="Stok Menipis" value={String(data.menipis)} color={data.menipis > 0 ? 'var(--text-warn)' : undefined} meta="sudah di bawah batas minimum" />
        <Kpi label="Stok Habis" value={String(data.habis)} color={data.habis > 0 ? RED : undefined} meta="jumlahnya nol" />
        {mulai && <Kpi label="Stok Minus" value={String(data.minus)} color={data.minus > 0 ? RED : undefined} meta="keluar lebih banyak dari yang tercatat masuk — cek pembelian/produksi" />}
      </div>

      <div className="card" style={{ padding: '16px 20px', marginBottom: 16 }}>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12, alignItems: 'center' }}>
          <input className="input-field" style={inputStyle} placeholder="Cari SKU atau nama barang…" value={cari} onChange={(e) => setCari(e.target.value)} />
          <select style={selectStyle} value={kategoriFilter} onChange={(e) => setKategoriFilter(e.target.value)} aria-label="Kategori">
            <option value="ALL">Semua Kategori</option>
            {kategoris.map((k) => <option key={k} value={k}>{k}</option>)}
          </select>
          <select style={selectStyle} value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} aria-label="Status stok">
            <option value="ALL">Semua Status Stok</option>
            {mulai && <option value="Minus">Minus</option>}
            <option value="Habis">Habis</option>
            <option value="Stok Menipis">Stok Menipis</option>
            <option value="Aman">Aman</option>
          </select>
        </div>
      </div>


      <div className="card grid gap-3" style={{ padding: 20 }}>
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <div className="chart-title">Daftar Stok</div>
            <div className="chart-subtitle">{mulai ? 'Klik nama barang untuk melihat kartu stoknya (riwayat masuk & keluar). Klik judul kolom untuk mengurutkan.' : 'Klik judul kolom untuk mengurutkan. Banyak barang sekaligus bisa lewat + Tambah barang atau Impor Excel.'}</div>
          </div>
          {bisaUbah && (
            <button className="btn-cta" style={{ padding: '8px 18px', fontSize: 13 }} onClick={() => { setError(''); setForm(null); setEditId(null); setTambahBanyak(true) }}>
              + Tambah barang
            </button>
          )}
        </div>
        {/* Form tambah / ubah muncul di sini, tepat di atas tabel, dan halaman bergeser ke sini. */}
        <div ref={formRef} style={{ scrollMarginTop: 16 }}>
          {tambahBanyak && (
            <TambahBarangBanyak skuTerpakai={list.map((x) => x.sku).filter(Boolean)} onBatal={() => setTambahBanyak(false)}
              onSelesai={(n) => { setTambahBanyak(false); setPesan(`${n} barang ditambahkan.`); muat() }} />
          )}
        {form && (
          <form className="grid gap-3" style={{ padding: 14, border: '2px solid var(--accent-orange)', borderRadius: 12, background: 'var(--accent-orange-soft)' }} onSubmit={simpan}>
            <div className="chart-title">{editId === null ? 'Tambah Barang' : 'Ubah Barang'}</div>
            {isiAi.aktif !== null && <IsiAIBanner ai={isiAi} />}
            <div className="form-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))' }}>
              <label>
                <span className="kpi-label">SKU</span>
                <input className="input-field" value={form.sku} onChange={(e) => setForm({ ...form, sku: e.target.value })} />
              </label>
              <label>
                <span className="kpi-label">Nama Barang</span>
                <input className="input-field" required value={form.nama} onChange={(e) => setForm({ ...form, nama: e.target.value })} />
              </label>
              <label>
                <span className="kpi-label">Kategori</span>
                <input className="input-field" list="stok-kategori" value={form.kategori} onChange={(e) => setForm({ ...form, kategori: e.target.value })} />
                <datalist id="stok-kategori">
                  {['Bahan Baku', 'Produk Jadi', 'Kemasan', 'Barang'].map((k) => <option key={k} value={k} />)}
                </datalist>
              </label>
              <label title={editDilacak ? 'Jumlah mengikuti kartu stok; ubah lewat stok opname' : undefined}>
                <span className="kpi-label">Jumlah{editDilacak ? ' (dari kartu stok)' : ''}</span>
                <input
                  className="input-field tabular" placeholder="0" disabled={editDilacak} {...angka(form.jumlah, (n) => setForm({ ...form, jumlah: n }))}
                />
              </label>
              <label>
                <span className="kpi-label">Satuan</span>
                <input className="input-field" value={form.satuan} onChange={(e) => setForm({ ...form, satuan: e.target.value })} />
              </label>
              <label>
                <span className="kpi-label">Stok Minimum</span>
                <input
                  className="input-field tabular" placeholder="0" {...angka(form.stok_min, (n) => setForm({ ...form, stok_min: n }))}
                />
              </label>
              <label title={editDilacak ? 'Harga pokok dihitung dari kartu stok (pembelian & produksi)' : undefined}>
                <span className="kpi-label">Harga Pokok / Satuan{editDilacak ? ' (dari kartu stok)' : ''}</span>
                <input
                  className="input-field tabular" placeholder="0" disabled={editDilacak} {...angka(form.harga_pokok, (n) => setForm({ ...form, harga_pokok: n }))}
                />
              </label>
              <label>
                <span className="kpi-label">Harga Jual / Satuan</span>
                <input
                  className="input-field tabular" placeholder="0" {...angka(form.harga_jual, (n) => setForm({ ...form, harga_jual: n }))}
                />
              </label>
            </div>
            <div style={{ display: 'flex', gap: 10, marginTop: 16 }}>
              <button className="btn-cta" style={{ padding: '9px 20px', fontSize: 13 }} type="submit" disabled={busy}>
                {busy ? 'Menyimpan…' : 'Simpan'}
              </button>
              <button type="button" className="nav-item" style={{ width: 'auto', padding: '9px 16px' }} onClick={() => setForm(null)}>Batal</button>
            </div>
          </form>
        )}
        </div>
        <div className="table-scroll">
          <table className="data-table tabular" style={{ minWidth: 940 }}>
            <thead>
              <tr>
                <ThUrut k="sku" urut={urut}>SKU</ThUrut><ThUrut k="nama" urut={urut}>Nama Barang</ThUrut><ThUrut k="kategori" urut={urut}>Kategori</ThUrut>
                <ThUrut k="jumlah" urut={urut} kanan>Jumlah</ThUrut><ThUrut k="min" urut={urut} kanan>Min.</ThUrut>
                <ThUrut k="hp" urut={urut} kanan>Harga Pokok</ThUrut><ThUrut k="hj" urut={urut} kanan>Harga Jual</ThUrut>
                <ThUrut k="margin" urut={urut} kanan>Margin</ThUrut><ThUrut k="nilai" urut={urut} kanan>Nilai</ThUrut><ThUrut k="status" urut={urut}>Status</ThUrut><th />
              </tr>
            </thead>
            <tbody>
              {tampil.length === 0 && (
                <tr>
                  <td colSpan={11} style={{ color: 'var(--text-muted)' }}>
                    {list.length === 0 ? 'Belum ada barang tercatat. Tambah satu per satu atau impor dari Excel.' : 'Tidak ada barang yang cocok dengan filter.'}
                  </td>
                </tr>
              )}
              {terurut.map((it) => (
                <tr key={it.id}>
                  <td style={{ color: 'var(--text-muted)' }}>{it.sku || '—'}</td>
                  <td style={{ fontWeight: 600 }}>
                    {it.dilacak
                      ? <button type="button" className="text-left hover:underline" style={{ color: 'var(--link)', fontWeight: 600 }} onClick={() => setKartu(it)}>{it.nama}</button>
                      : it.nama}
                  </td>
                  <td>{it.kategori}</td>
                  <td style={{ textAlign: 'right', color: it.minus ? RED : undefined, fontWeight: it.minus ? 700 : undefined }}>{formatQty(it.jumlah_mili)} {it.satuan}</td>
                  <td style={{ textAlign: 'right', color: 'var(--text-muted)' }}>{it.stok_min > 0 ? it.stok_min.toLocaleString('id-ID') : '—'}</td>
                  <td style={{ textAlign: 'right' }}>{formatRp(it.harga_pokok)}</td>
                  <td style={{ textAlign: 'right' }}>{it.harga_jual > 0 ? formatRp(it.harga_jual) : '—'}</td>
                  <td style={{ textAlign: 'right', color: it.margin_bps > 0 ? GREEN : 'var(--text-muted)' }}>
                    {it.margin_bps > 0 ? `${Math.round(it.margin_bps / 100)}%` : '—'}
                  </td>
                  <td style={{ textAlign: 'right', color: BLUE }}>{formatRp(it.nilai)}</td>
                  <td><span className={'pill-status ' + STATUS_CLS[it.status]} style={{ fontSize: 10 }}>{it.status}</span></td>
                  <Aksi onEdit={() => { setError(''); setEditId(it.id); setForm(toStokInput(it)) }} onDelete={() => void hapus(it)} />
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      </>}

      {dialog === 'beli' && list.length > 0 && <BeliBahanDialog items={list} onClose={() => setDialog(null)} onDone={selesaiDialog} />}
      {kartu && <KartuBarangDialog item={kartu} onClose={() => setKartu(null)} />}

      <MenuImportDialog
        open={impor} onOpenChange={setImpor} onImported={muat}
        title="Impor stok dari Excel"
        description="Wajib: Nama Barang dan Jumlah. Kolom Stok Minimum menentukan peringatan stok menipis — kosongkan bila tidak dipakai."
        templateUrl={STOK_TEMPLATE_URL} templateName="template-stok-gudang.xlsx"
        existing={list.length} satuan="barang"
        doImport={importStok}
        renderPreview={(r) => (
          <table className="data-table tabular" style={{ marginTop: 0 }}>
            <thead><tr><th>SKU</th><th>Nama</th><th>Kategori</th><th style={{ textAlign: 'right' }}>Jumlah</th><th style={{ textAlign: 'right' }}>Harga Pokok</th></tr></thead>
            <tbody>
              {(r.stok ?? []).slice(0, 8).map((it, i) => (
                <tr key={i}>
                  <td>{it.sku || '—'}</td><td>{it.nama}</td><td>{it.kategori}</td>
                  <td style={{ textAlign: 'right' }}>{it.jumlah.toLocaleString('id-ID')} {it.satuan}</td>
                  <td style={{ textAlign: 'right' }}>{formatRp(it.harga_pokok)}</td>
                </tr>
              ))}
              {(r.stok ?? []).length > 8 && (
                <tr><td colSpan={5} style={{ color: 'var(--text-muted)' }}>…dan {(r.stok ?? []).length - 8} baris lainnya</td></tr>
              )}
            </tbody>
          </table>
        )}
      />
    </>
  )
}
