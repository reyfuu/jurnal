import { Upload } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { MenuImportDialog } from '@/views/MenuImportDialog'
import {
  createStok, createTagihan, deleteStok, deleteTagihan, importStok, importTagihan,
  STOK_TEMPLATE_URL, TAGIHAN_TEMPLATE_URL, getStok, getTagihan, updateStok, updateTagihan,
  type StokInput, type StokRow, type StokSummary, type TagihanInput, type TagihanRow, type TagihanSummary,
} from '@/lib/api'
import { formatJt, formatRp } from '@/lib/format'

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

const TAGIHAN_KOSONG: TagihanInput = {
  nomor: '', pihak: '', jenis: 'RECEIVABLE', jatuh_tempo: '', nominal: 0, lunas: false, keterangan: '',
}

export function PiutangUtang() {
  const [data, setData] = useState<TagihanSummary | null>(null)
  const [form, setForm] = useState<TagihanInput | null>(null)
  const [editId, setEditId] = useState<number | null>(null)
  const [impor, setImpor] = useState(false)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [cari, setCari] = useState('')
  const [jenisFilter, setJenisFilter] = useState('ALL')
  const [statusFilter, setStatusFilter] = useState('ALL')

  const muat = useCallback(() => { getTagihan().then(setData, (e: Error) => setError(e.message)) }, [])
  useEffect(muat, [muat])

  async function simpan(e: React.FormEvent) {
    e.preventDefault()
    if (!form) return
    setBusy(true)
    try {
      await (editId === null ? createTagihan(form) : updateTagihan(editId, form))
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
      <PageHeader title="Piutang & Utang" subtitle="Pantau tagihan pelanggan dan kewajiban ke supplier supaya arus kas tidak macet.">
        <div style={{ display: 'flex', gap: 8 }}>
          <button className="nav-item" style={{ width: 'auto', padding: '8px 14px', display: 'inline-flex', alignItems: 'center', gap: 6 }} onClick={() => setImpor(true)}>
            <Upload size={15} aria-hidden /> Impor Excel
          </button>
          <button
            className="btn-cta" style={{ padding: '8px 18px', fontSize: 13 }}
            onClick={() => { setError(''); setEditId(null); setForm({ ...TAGIHAN_KOSONG }) }}
          >
            + Tambah Tagihan
          </button>
        </div>
      </PageHeader>

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
        <form className="card" style={{ padding: 20, marginBottom: 14 }} onSubmit={simpan}>
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
                className="input-field tabular" type="number" min={1} required value={form.nominal || ''}
                onChange={(e) => setForm({ ...form, nominal: Number(e.target.value) })}
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
                    {list.length === 0 ? 'Belum ada tagihan tercatat. Tambah satu per satu atau impor dari Excel.' : 'Tidak ada tagihan yang cocok dengan filter.'}
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

      <MenuImportDialog
        open={impor} onOpenChange={setImpor} onImported={muat}
        title="Impor piutang & utang dari Excel"
        description="Wajib: Pelanggan/Supplier, Jatuh Tempo, Nominal. Kolom Jenis menentukan arah tagihan — periksa pengelompokannya di pratinjau sebelum diimpor."
        templateUrl={TAGIHAN_TEMPLATE_URL} templateName="template-piutang-utang.xlsx"
        existing={list.length} satuan="tagihan"
        doImport={importTagihan}
        renderPreview={(r) => (
          <table className="data-table tabular" style={{ marginTop: 0 }}>
            <thead><tr><th>Jenis</th><th>No.</th><th>Pihak</th><th>Jatuh Tempo</th><th style={{ textAlign: 'right' }}>Nominal</th></tr></thead>
            <tbody>
              {(r.tagihan ?? []).slice(0, 8).map((t, i) => (
                <tr key={i}>
                  <td style={{ color: t.jenis === 'RECEIVABLE' ? GREEN : RED, fontWeight: 600 }}>{t.jenis === 'RECEIVABLE' ? 'Piutang' : 'Utang'}</td>
                  <td>{t.nomor || '—'}</td><td>{t.pihak}</td><td>{t.jatuh_tempo}</td>
                  <td style={{ textAlign: 'right' }}>{formatRp(t.nominal)}</td>
                </tr>
              ))}
              {(r.tagihan ?? []).length > 8 && (
                <tr><td colSpan={5} style={{ color: 'var(--text-muted)' }}>…dan {(r.tagihan ?? []).length - 8} baris lainnya</td></tr>
              )}
            </tbody>
          </table>
        )}
      />
    </>
  )
}

// --- Stok & Gudang ---

const STOK_KOSONG: StokInput = {
  sku: '', nama: '', kategori: 'Bahan Baku', jumlah: 0, satuan: 'pcs', stok_min: 0, harga_pokok: 0, harga_jual: 0,
}

const STATUS_CLS: Record<StokRow['status'], string> = {
  Habis: 'pill-red',
  'Stok Menipis': 'pill-amber',
  Aman: 'pill-green',
}

export function StokGudang() {
  const [data, setData] = useState<StokSummary | null>(null)
  const [form, setForm] = useState<StokInput | null>(null)
  const [editId, setEditId] = useState<number | null>(null)
  const [impor, setImpor] = useState(false)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [cari, setCari] = useState('')
  const [kategoriFilter, setKategoriFilter] = useState('ALL')
  const [statusFilter, setStatusFilter] = useState('ALL')

  const muat = useCallback(() => { getStok().then(setData, (e: Error) => setError(e.message)) }, [])
  useEffect(muat, [muat])

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
  const kategoris = [...new Set(list.map((it) => it.kategori))].sort()
  const tampil = list.filter((it) => {
    if (kategoriFilter !== 'ALL' && it.kategori !== kategoriFilter) return false
    if (statusFilter !== 'ALL' && it.status !== statusFilter) return false
    const q = cari.trim().toLowerCase()
    if (q && !it.sku.toLowerCase().includes(q) && !it.nama.toLowerCase().includes(q)) return false
    return true
  })

  return (
    <>
      <PageHeader title="Stok & Gudang" subtitle="Daftar bahan baku & produk jadi, nilai persediaan, dan peringatan stok menipis.">
        <div style={{ display: 'flex', gap: 8 }}>
          <button className="nav-item" style={{ width: 'auto', padding: '8px 14px', display: 'inline-flex', alignItems: 'center', gap: 6 }} onClick={() => setImpor(true)}>
            <Upload size={15} aria-hidden /> Impor Excel
          </button>
          <button
            className="btn-cta" style={{ padding: '8px 18px', fontSize: 13 }}
            onClick={() => { setError(''); setEditId(null); setForm({ ...STOK_KOSONG }) }}
          >
            + Tambah Barang
          </button>
        </div>
      </PageHeader>

      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', marginBottom: 12 }}>{error}</div>}

      <div className="kpi-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(210px, 1fr))', marginBottom: 16 }}>
        <Kpi label="Item Tercatat" value={String(data.total_item)} meta={`${tampil.length} tampil sesuai filter`} />
        <Kpi label="Nilai Persediaan" value={formatJt(data.nilai_persediaan)} color={BLUE} meta="jumlah × harga pokok" />
        <Kpi label="Stok Menipis" value={String(data.menipis)} color={data.menipis > 0 ? 'var(--text-warn)' : undefined} meta="sudah di bawah batas minimum" />
        <Kpi label="Stok Habis" value={String(data.habis)} color={data.habis > 0 ? RED : undefined} meta="jumlahnya nol" />
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
            <option value="Habis">Habis</option>
            <option value="Stok Menipis">Stok Menipis</option>
            <option value="Aman">Aman</option>
          </select>
        </div>
      </div>

      {form && (
        <form className="card" style={{ padding: 20, marginBottom: 14 }} onSubmit={simpan}>
          <div className="chart-title">{editId === null ? 'Tambah Barang' : 'Ubah Barang'}</div>
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
            <label>
              <span className="kpi-label">Jumlah</span>
              <input
                className="input-field tabular" type="number" min={0} required value={form.jumlah || 0}
                onChange={(e) => setForm({ ...form, jumlah: Number(e.target.value) })}
              />
            </label>
            <label>
              <span className="kpi-label">Satuan</span>
              <input className="input-field" value={form.satuan} onChange={(e) => setForm({ ...form, satuan: e.target.value })} />
            </label>
            <label>
              <span className="kpi-label">Stok Minimum</span>
              <input
                className="input-field tabular" type="number" min={0} value={form.stok_min || 0}
                onChange={(e) => setForm({ ...form, stok_min: Number(e.target.value) })}
              />
            </label>
            <label>
              <span className="kpi-label">Harga Pokok / Satuan</span>
              <input
                className="input-field tabular" type="number" min={0} value={form.harga_pokok || 0}
                onChange={(e) => setForm({ ...form, harga_pokok: Number(e.target.value) })}
              />
            </label>
            <label>
              <span className="kpi-label">Harga Jual / Satuan</span>
              <input
                className="input-field tabular" type="number" min={0} value={form.harga_jual || 0}
                onChange={(e) => setForm({ ...form, harga_jual: Number(e.target.value) })}
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

      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Daftar Stok</div>
        <div className="chart-subtitle">Bisa diisi sekaligus lewat tombol Impor Excel, atau lewat Pengaturan → Impor Data Awal</div>
        <div className="table-scroll">
          <table className="data-table tabular" style={{ minWidth: 940 }}>
            <thead>
              <tr>
                <th>SKU</th><th>Nama Barang</th><th>Kategori</th>
                <th style={{ textAlign: 'right' }}>Jumlah</th><th style={{ textAlign: 'right' }}>Min.</th>
                <th style={{ textAlign: 'right' }}>Harga Pokok</th><th style={{ textAlign: 'right' }}>Harga Jual</th>
                <th style={{ textAlign: 'right' }}>Margin</th><th style={{ textAlign: 'right' }}>Nilai</th><th>Status</th><th />
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
              {tampil.map((it) => (
                <tr key={it.id}>
                  <td style={{ color: 'var(--text-muted)' }}>{it.sku || '—'}</td>
                  <td style={{ fontWeight: 600 }}>{it.nama}</td>
                  <td>{it.kategori}</td>
                  <td style={{ textAlign: 'right' }}>{it.jumlah.toLocaleString('id-ID')} {it.satuan}</td>
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
