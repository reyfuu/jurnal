import { Factory, Plus, Trash2 } from 'lucide-react'
import { Fragment, useCallback, useEffect, useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import {
  createProduk, createSupplier, deleteProduk, deleteSupplier, getProduksi, getStok, terapkanHPP,
  updateProduk, updateSupplier, TAKARAN_SKALA,
  type HPP, type ProdukInput, type ProdukRow, type ProduksiSummary, type ResepBahan,
  type StokRow, type SupplierInput, type SupplierRow,
} from '@/lib/api'
import { formatJt, formatRp } from '@/lib/format'
import { angka } from '@/lib/angka'
import { LaporanProduksiPanel, ProduksiDialog } from '@/views/KartuStok'

const RED = 'var(--text-danger)'
const GREEN = 'var(--text-profit)'
const BLUE = 'var(--accent-blue)'
const AMBER = 'var(--text-warn)'

const aksiBtn = { width: 'auto', padding: '4px 8px', fontSize: 12 } as const

/** Takaran disimpan per seribu satuan; di layar ditampilkan sebagai desimal biasa. */
const takaranKeTeks = (t: number) => (t / TAKARAN_SKALA).toLocaleString('id-ID', { maximumFractionDigits: 3 })
const teksKeTakaran = (s: string) => Math.round((parseFloat(s.replace(',', '.')) || 0) * TAKARAN_SKALA)

const PRODUK_KOSONG: ProdukInput = {
  sku: '', nama: '', satuan: 'cup', kategori: 'Produk Jadi', hasil_per_batch: 1,
  bahan: [], biaya_tenaga_kerja: 0, biaya_overhead: 0, harga_jual: 0, catatan: '',
}

const SUPPLIER_KOSONG: SupplierInput = {
  nama: '', kontak: '', telepon: '', email: '', alamat: '', tempo_hari: 0, catatan: '',
}

const toProdukInput = ({
  sku, nama, satuan, kategori, hasil_per_batch, bahan, biaya_tenaga_kerja, biaya_overhead, harga_jual, catatan,
}: ProdukRow): ProdukInput =>
  ({ sku, nama, satuan, kategori, hasil_per_batch, bahan: bahan ?? [], biaya_tenaga_kerja, biaya_overhead, harga_jual, catatan })

const toSupplierInput = ({ nama, kontak, telepon, email, alamat, tempo_hari, catatan }: SupplierRow): SupplierInput =>
  ({ nama, kontak, telepon, email, alamat, tempo_hari, catatan })

/** Rincian HPP satu produk: dari bahan sampai laba per unit. */
function RincianHPP({ h, satuan }: { h: HPP; satuan: string }) {
  const baris = [
    { label: 'Total bahan', value: h.total_bahan },
    { label: 'Tenaga kerja per batch', value: h.tenaga_kerja },
    { label: 'Overhead per batch', value: h.overhead },
  ]
  return (
    <div style={{ background: 'var(--bg-raised)', borderRadius: 10, padding: 16 }}>
      <table className="data-table tabular" style={{ marginTop: 0 }}>
        <thead>
          <tr>
            <th>Bahan</th><th style={{ textAlign: 'right' }}>Takaran</th>
            <th style={{ textAlign: 'right' }}>Harga / Satuan</th><th style={{ textAlign: 'right' }}>Biaya</th><th>Sumber</th>
          </tr>
        </thead>
        <tbody>
          {(h.bahan ?? []).length === 0 && (
            <tr><td colSpan={5} style={{ color: 'var(--text-muted)' }}>Resep belum berisi bahan.</td></tr>
          )}
          {(h.bahan ?? []).map((b, i) => (
            <tr key={i}>
              <td>
                {b.nama || '(tanpa nama)'}
                {b.peringatan && <span style={{ color: AMBER, fontSize: 11, display: 'block' }}>{b.peringatan}</span>}
              </td>
              <td style={{ textAlign: 'right' }}>{takaranKeTeks(b.takaran)} {b.satuan}</td>
              <td style={{ textAlign: 'right' }}>{formatRp(b.harga_satuan)}</td>
              <td style={{ textAlign: 'right' }}>{formatRp(b.subtotal)}</td>
              <td style={{ color: 'var(--text-muted)', fontSize: 11 }}>{b.sumber === 'stok' ? 'Stok & Gudang' : 'diisi manual'}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <div className="list-rows tabular" style={{ marginTop: 12 }}>
        {baris.map((b) => (
          <div className="list-row" key={b.label}><span>{b.label}</span><strong>{formatRp(b.value)}</strong></div>
        ))}
        <div className="list-row"><span><strong>Total biaya per batch</strong></span><strong style={{ color: BLUE }}>{formatRp(h.total_batch)}</strong></div>
        <div className="list-row">
          <span>Hasil per batch</span><strong>{h.hasil_per_batch.toLocaleString('id-ID')} {satuan}</strong>
        </div>
        <div className="list-row" style={{ border: '1px solid color-mix(in srgb, var(--accent-blue) 30%, transparent)' }}>
          <span><strong>HPP per {satuan}</strong></span><strong style={{ color: BLUE }}>{formatRp(h.per_unit)}</strong>
        </div>
        {h.harga_jual > 0 && (
          <>
            <div className="list-row"><span>Harga jual per {satuan}</span><strong>{formatRp(h.harga_jual)}</strong></div>
            <div className="list-row">
              <span><strong>Laba per {satuan}</strong></span>
              <strong style={{ color: h.laba_per_unit < 0 ? RED : GREEN }}>
                {formatRp(h.laba_per_unit)} ({(h.margin_bps / 100).toFixed(1).replace('.', ',')}%)
              </strong>
            </div>
          </>
        )}
      </div>

      {(h.peringatan ?? []).length > 0 && (
        <div className="note-box warn" style={{ marginTop: 12 }}>
          <strong>Perlu diperiksa:</strong>
          <ul style={{ margin: '6px 0 0 18px' }}>
            {(h.peringatan ?? []).map((w) => <li key={w}>{w}</li>)}
          </ul>
        </div>
      )}
    </div>
  )
}

/** Baris-baris bahan di dalam form produk. */
function EditorResep({ bahan, stok, onChange }: { bahan: ResepBahan[]; stok: StokRow[]; onChange: (b: ResepBahan[]) => void }) {
  const ubah = (i: number, patch: Partial<ResepBahan>) => onChange(bahan.map((b, j) => (j === i ? { ...b, ...patch } : b)))
  const satuanDari = (b: ResepBahan) => (b.stok_id ? stok.find((s) => s.id === b.stok_id)?.satuan ?? '' : b.satuan)

  return (
    <div style={{ marginTop: 18 }}>
      <div className="kpi-label" style={{ marginBottom: 8 }}>Bahan per Batch</div>
      <div className="table-scroll">
        <table className="data-table tabular" style={{ minWidth: 760, marginTop: 0 }}>
          <thead>
            <tr>
              <th style={{ minWidth: 240 }}>Bahan</th><th style={{ width: 150 }}>Takaran</th>
              <th style={{ width: 160 }}>Harga / Satuan</th><th />
            </tr>
          </thead>
          <tbody>
            {bahan.length === 0 && (
              <tr><td colSpan={4} style={{ color: 'var(--text-muted)' }}>Belum ada bahan. Tambah baris di bawah.</td></tr>
            )}
            {bahan.map((b, i) => (
              <tr key={i}>
                <td>
                  <select
                    className="input-field"
                    value={b.stok_id || 'manual'}
                    onChange={(e) => {
                      const v = e.target.value
                      ubah(i, v === 'manual' ? { stok_id: 0 } : { stok_id: Number(v), nama: '', harga_manual: 0 })
                    }}
                  >
                    <option value="manual">— bahan di luar stok —</option>
                    {stok.map((s) => (
                      <option key={s.id} value={s.id}>{s.nama} ({formatRp(s.harga_pokok)}/{s.satuan})</option>
                    ))}
                  </select>
                  {!b.stok_id && (
                    <div style={{ display: 'flex', gap: 6, marginTop: 6 }}>
                      <input className="input-field" placeholder="Nama bahan" value={b.nama} onChange={(e) => ubah(i, { nama: e.target.value })} />
                      <input className="input-field" style={{ maxWidth: 90 }} placeholder="satuan" value={b.satuan} onChange={(e) => ubah(i, { satuan: e.target.value })} />
                    </div>
                  )}
                </td>
                <td>
                  <input
                    className="input-field tabular" type="number" step="0.001" min={0}
                    value={b.takaran ? b.takaran / TAKARAN_SKALA : ''}
                    onChange={(e) => ubah(i, { takaran: teksKeTakaran(e.target.value) })}
                  />
                  <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>{satuanDari(b) || 'satuan bahan'}</span>
                </td>
                <td>
                  {b.stok_id ? (
                    <span style={{ color: 'var(--text-muted)', fontSize: 12 }}>
                      {formatRp(stok.find((s) => s.id === b.stok_id)?.harga_pokok ?? 0)}
                      <span style={{ display: 'block', fontSize: 11 }}>dari Stok & Gudang</span>
                    </span>
                  ) : (
                    <input
                      className="input-field tabular" {...angka(b.harga_manual, (n) => ubah(i, { harga_manual: n }))}
                    />
                  )}
                </td>
                <td>
                  <button
                    type="button" className="nav-item" style={{ ...aksiBtn, color: RED }}
                    onClick={() => onChange(bahan.filter((_, j) => j !== i))} aria-label="Hapus bahan"
                  >
                    <Trash2 size={14} aria-hidden />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <button
        type="button" className="nav-item" style={{ width: 'auto', padding: '6px 12px', marginTop: 10, display: 'inline-flex', alignItems: 'center', gap: 6 }}
        onClick={() => onChange([...bahan, { stok_id: stok[0]?.id ?? 0, nama: '', satuan: '', takaran: 0, harga_manual: 0 }])}
      >
        <Plus size={14} aria-hidden /> Tambah bahan
      </button>
    </div>
  )
}

/** bisaUbah: false di mode lihat-sebagai (hanya baca). */
export function ProduksiSupplier({ bisaUbah = true }: { bisaUbah?: boolean }) {
  const [tab, setTab] = useState<'produksi' | 'produk' | 'supplier'>('produksi')
  const [muatUlang, setMuatUlang] = useState(0)
  const [data, setData] = useState<ProduksiSummary | null>(null)
  const [stok, setStok] = useState<StokRow[]>([])
  const [stokMulai, setStokMulai] = useState('')
  // Dialog catat produksi; awal = produk yang dipilih dari baris tabel resep.
  const [produksi, setProduksi] = useState<{ awal?: ProdukRow } | null>(null)
  const [form, setForm] = useState<ProdukInput | null>(null)
  const [editId, setEditId] = useState<number | null>(null)
  const [supForm, setSupForm] = useState<SupplierInput | null>(null)
  const [supEditId, setSupEditId] = useState<number | null>(null)
  const [buka, setBuka] = useState<number | null>(null)
  const [error, setError] = useState('')
  const [pesan, setPesan] = useState('')
  const [busy, setBusy] = useState(false)

  const muat = useCallback(() => {
    getProduksi().then(setData, (e: Error) => setError(e.message))
    getStok().then((s) => { setStok(s.items ?? []); setStokMulai(s.mulai) }, () => {})
  }, [])
  useEffect(muat, [muat])

  async function simpanProduk(e: React.FormEvent) {
    e.preventDefault()
    if (!form) return
    setBusy(true)
    try {
      await (editId === null ? createProduk(form) : updateProduk(editId, form))
      setForm(null)
      setEditId(null)
      muat()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  async function hapusProduk(p: ProdukRow) {
    if (!confirm(`Hapus produk ${p.nama} beserta resepnya?`)) return
    try {
      await deleteProduk(p.id)
      muat()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function pakaiHPP(p: ProdukRow) {
    if (!confirm(`Ganti harga pokok ${p.sku} di Stok & Gudang jadi ${formatRp(p.hpp.per_unit)}?\n\nHarga pokok ikut menentukan nilai persediaan di neraca.`)) return
    setError('')
    try {
      const r = await terapkanHPP(p.id)
      setPesan(`Harga pokok ${r.sku} di Stok & Gudang kini ${formatRp(r.harga_pokok)}.`)
      muat()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function simpanSupplier(e: React.FormEvent) {
    e.preventDefault()
    if (!supForm) return
    setBusy(true)
    try {
      await (supEditId === null ? createSupplier(supForm) : updateSupplier(supEditId, supForm))
      setSupForm(null)
      setSupEditId(null)
      muat()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  async function hapusSupplier(s: SupplierRow) {
    if (!confirm(`Hapus supplier ${s.nama}?`)) return
    try {
      await deleteSupplier(s.id)
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

  const produk = data.produk ?? []
  const supplier = data.supplier ?? []
  const hppTertinggi = produk.reduce((m, p) => Math.max(m, p.hpp.per_unit), 0)

  return (
    <>
      <PageHeader
        title="Produksi & Supplier"
        subtitle={tab === 'produksi' ? 'Catat setiap kali produksi: bahan di stok berkurang, produk jadi bertambah.'
          : tab === 'produk' ? 'Daftar produk & resepnya (diisi sekali per produk). HPP dihitung dari harga bahan di Stok & Gudang.'
            : 'Daftar supplier bahan dan perkiraan utang ke masing-masing.'}
      >
        {bisaUbah && tab === 'produksi' && (
          <button className="btn-cta accent" style={{ padding: '10px 20px', fontSize: 14, display: 'inline-flex', alignItems: 'center', gap: 8 }}
            disabled={!stokMulai || produk.length === 0}
            title={!stokMulai ? 'Mulai kartu stok dulu di Stok & Gudang' : produk.length === 0 ? 'Tambah produk & resep dulu' : undefined}
            onClick={() => { setError(''); setPesan(''); setProduksi({}) }}>
            <Factory size={17} aria-hidden /> Catat produksi
          </button>
        )}
        {bisaUbah && tab === 'produk' && (
          <button className="btn-cta" style={{ padding: '8px 18px', fontSize: 13 }}
            onClick={() => { setError(''); setEditId(null); setForm({ ...PRODUK_KOSONG, bahan: [] }) }}>
            + Tambah Produk
          </button>
        )}
        {bisaUbah && tab === 'supplier' && (
          <button className="btn-cta" style={{ padding: '8px 18px', fontSize: 13 }}
            onClick={() => { setError(''); setSupEditId(null); setSupForm({ ...SUPPLIER_KOSONG }) }}>
            + Tambah Supplier
          </button>
        )}
      </PageHeader>

      <div className="tabbar" role="tablist" aria-label="Bagian produksi" style={{ marginBottom: 12 }}>
        <button role="tab" className="tab-btn" aria-selected={tab === 'produksi'} onClick={() => setTab('produksi')}>Produksi</button>
        <button role="tab" className="tab-btn" aria-selected={tab === 'produk'} onClick={() => setTab('produk')}>Produk &amp; Resep</button>
        <button role="tab" className="tab-btn" aria-selected={tab === 'supplier'} onClick={() => setTab('supplier')}>Supplier</button>
      </div>

      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', marginBottom: 12 }}>{error}</div>}
      {pesan && <div className="pill-status pill-green" style={{ alignSelf: 'flex-start', marginBottom: 12 }}>{pesan}</div>}

      {tab === 'produksi' && (!stokMulai ? (
        <div className="card" style={{ padding: 24 }}>
          <div className="chart-title">Mulai kartu stok dulu</div>
          <div className="chart-subtitle" style={{ marginBottom: 12 }}>
            Mencatat produksi butuh kartu stok supaya bahan yang terpakai dan produk jadi yang dihasilkan ikut tercatat. Mulai dari menu
            <strong> Stok &amp; Gudang → Mulai kartu stok</strong> (hitung fisik pertama).
          </div>
        </div>
      ) : produk.length === 0 ? (
        <div className="card" style={{ padding: 24 }}>
          <div className="chart-title">Belum ada produk & resep</div>
          <div className="chart-subtitle">Buka tab <strong>Produk &amp; Resep</strong> → <strong>+ Tambah Produk</strong> (sekali saja per produk), lalu kembali ke sini untuk mencatat produksi.</div>
        </div>
      ) : (
        <LaporanProduksiPanel muatUlang={muatUlang} bisaUbah={bisaUbah} bisaEkspor={bisaUbah} onChanged={() => { setMuatUlang((n) => n + 1); muat() }} />
      ))}

      {tab === 'produk' && <>
      <div className="kpi-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(210px, 1fr))', marginBottom: 16 }}>
        <div className="card" style={{ padding: '14px 18px' }}>
          <div className="kpi-label">Produk Ber-resep</div>
          <div className="kpi-value tabular" style={{ fontSize: 20 }}>{data.total_produk}</div>
          <div className="kpi-meta-row"><span>HPP tertinggi {formatRp(hppTertinggi)}</span></div>
        </div>
        <div className="card" style={{ padding: '14px 18px' }}>
          <div className="kpi-label">Resep Belum Lengkap</div>
          <div className="kpi-value tabular" style={{ fontSize: 20, color: data.produk_tanpa_hpp > 0 ? AMBER : undefined }}>{data.produk_tanpa_hpp}</div>
          <div className="kpi-meta-row"><span>ada bahan tanpa harga pokok</span></div>
        </div>
        <div className="card" style={{ padding: '14px 18px' }}>
          <div className="kpi-label">Dijual di Bawah HPP</div>
          <div className="kpi-value tabular" style={{ fontSize: 20, color: data.produk_rugi > 0 ? RED : undefined }}>{data.produk_rugi}</div>
          <div className="kpi-meta-row"><span>setiap unit terjual menambah rugi</span></div>
        </div>
        <div className="card" style={{ padding: '14px 18px' }}>
          <div className="kpi-label">Utang ke Supplier</div>
          <div className="kpi-value tabular" style={{ fontSize: 20, color: RED }}>{formatJt(data.utang_supplier)}</div>
          <div className="kpi-meta-row"><span>{data.total_supplier} supplier terdaftar</span></div>
        </div>
      </div>


      {form && (
        <form className="card" style={{ padding: 20, marginBottom: 14 }} onSubmit={simpanProduk}>
          <div className="chart-title">{editId === null ? 'Tambah Produk & Resep' : 'Ubah Produk & Resep'}</div>
          <div className="form-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))' }}>
            <label>
              <span className="kpi-label">SKU</span>
              <input className="input-field" placeholder="mis. FG-001" value={form.sku} onChange={(e) => setForm({ ...form, sku: e.target.value })} />
            </label>
            <label>
              <span className="kpi-label">Nama Produk</span>
              <input className="input-field" required value={form.nama} onChange={(e) => setForm({ ...form, nama: e.target.value })} />
            </label>
            <label>
              <span className="kpi-label">Satuan Jual</span>
              <input className="input-field" value={form.satuan} onChange={(e) => setForm({ ...form, satuan: e.target.value })} />
            </label>
            <label>
              <span className="kpi-label">Hasil per Batch</span>
              <input
                className="input-field tabular" required {...angka(form.hasil_per_batch, (n) => setForm({ ...form, hasil_per_batch: n }))}
              />
              <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>isi 1 bila resepnya untuk satu unit</span>
            </label>
            <label>
              <span className="kpi-label">Tenaga Kerja per Batch</span>
              <input
                className="input-field tabular" placeholder="0" {...angka(form.biaya_tenaga_kerja, (n) => setForm({ ...form, biaya_tenaga_kerja: n }))}
              />
            </label>
            <label>
              <span className="kpi-label">Overhead per Batch</span>
              <input
                className="input-field tabular" placeholder="0" {...angka(form.biaya_overhead, (n) => setForm({ ...form, biaya_overhead: n }))}
              />
              <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>gas, listrik, es, tisu</span>
            </label>
            <label>
              <span className="kpi-label">Harga Jual per Satuan</span>
              <input
                className="input-field tabular" placeholder="0" {...angka(form.harga_jual, (n) => setForm({ ...form, harga_jual: n }))}
              />
            </label>
            <label>
              <span className="kpi-label">Catatan</span>
              <input className="input-field" value={form.catatan ?? ''} onChange={(e) => setForm({ ...form, catatan: e.target.value })} />
            </label>
          </div>

          <EditorResep bahan={form.bahan ?? []} stok={stok} onChange={(b) => setForm({ ...form, bahan: b })} />

          {stok.length === 0 && (
            <div className="note-box warn" style={{ marginTop: 12 }}>
              Daftar Stok & Gudang masih kosong, jadi bahan hanya bisa diisi manual. Isi stok lebih dulu
              supaya HPP ikut berubah sendiri saat harga bahan naik.
            </div>
          )}

          <div style={{ display: 'flex', gap: 10, marginTop: 16 }}>
            <button className="btn-cta" style={{ padding: '9px 20px', fontSize: 13 }} type="submit" disabled={busy}>
              {busy ? 'Menyimpan…' : 'Simpan Produk'}
            </button>
            <button type="button" className="nav-item" style={{ width: 'auto', padding: '9px 16px' }} onClick={() => setForm(null)}>Batal</button>
          </div>
        </form>
      )}

      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">HPP per Produk</div>
        <div className="chart-subtitle">Klik satu baris untuk melihat rincian bahannya</div>
        <div className="table-scroll">
          <table className="data-table tabular" style={{ minWidth: 900 }}>
            <thead>
              <tr>
                <th>SKU</th><th>Produk</th><th style={{ textAlign: 'right' }}>Batch</th>
                <th style={{ textAlign: 'right' }}>HPP / Satuan</th><th style={{ textAlign: 'right' }}>Harga Jual</th>
                <th style={{ textAlign: 'right' }}>Laba</th><th style={{ textAlign: 'right' }}>Margin</th><th>HPP di Stok</th><th />
              </tr>
            </thead>
            <tbody>
              {produk.length === 0 && (
                <tr><td colSpan={9} style={{ color: 'var(--text-muted)' }}>Belum ada produk ber-resep. Tambah satu untuk mulai menghitung HPP.</td></tr>
              )}
              {produk.map((p) => (
                <Fragment key={p.id}>
                  <tr>
                    <td style={{ color: 'var(--text-muted)' }}>{p.sku || '—'}</td>
                    <td style={{ fontWeight: 600 }}>
                      <button
                        type="button" className="nav-item" style={{ width: 'auto', padding: 0, fontWeight: 600, background: 'none' }}
                        onClick={() => setBuka(buka === p.id ? null : p.id)} aria-expanded={buka === p.id}
                      >
                        {p.nama} {buka === p.id ? '▾' : '▸'}
                      </button>
                    </td>
                    <td style={{ textAlign: 'right' }}>{p.hasil_per_batch} {p.satuan}</td>
                    <td style={{ textAlign: 'right', fontWeight: 700, color: BLUE }}>{formatRp(p.hpp.per_unit)}</td>
                    <td style={{ textAlign: 'right' }}>{p.harga_jual > 0 ? formatRp(p.harga_jual) : '—'}</td>
                    <td style={{ textAlign: 'right', color: p.hpp.laba_per_unit < 0 ? RED : GREEN }}>
                      {p.harga_jual > 0 ? formatRp(p.hpp.laba_per_unit) : '—'}
                    </td>
                    <td style={{ textAlign: 'right' }}>{p.harga_jual > 0 ? `${(p.hpp.margin_bps / 100).toFixed(1).replace('.', ',')}%` : '—'}</td>
                    <td>
                      {stokMulai ? (
                        p.ada_di_stok
                          ? <span style={{ fontSize: 12 }} title="Dari kartu stok (rata-rata bergerak)">Stok {formatRp(p.hpp_stok)}</span>
                          : <span style={{ color: 'var(--text-muted)', fontSize: 12 }}>dibuat saat produksi pertama</span>
                      ) : !p.ada_di_stok ? (
                        <span style={{ color: 'var(--text-muted)', fontSize: 12 }}>SKU tidak ada di stok</span>
                      ) : p.selisih === 0 ? (
                        <span className="pill-status pill-green" style={{ fontSize: 10 }}>Sama ({formatRp(p.hpp_stok)})</span>
                      ) : (
                        <button className="nav-item" style={{ ...aksiBtn, color: 'var(--link)' }} onClick={() => void pakaiHPP(p)}>
                          Stok {formatRp(p.hpp_stok)} — pakai HPP resep
                        </button>
                      )}
                    </td>
                    <td>
                      <div style={{ display: 'flex', gap: 4 }}>
                        {stokMulai && <button className="nav-item" style={{ ...aksiBtn, color: 'var(--link)', fontWeight: 600 }} onClick={() => { setError(''); setPesan(''); setProduksi({ awal: p }) }}>Catat produksi</button>}
                        <button className="nav-item" style={aksiBtn} onClick={() => { setError(''); setEditId(p.id); setForm(toProdukInput(p)) }}>Ubah</button>
                        <button className="nav-item" style={{ ...aksiBtn, color: RED }} onClick={() => void hapusProduk(p)}>Hapus</button>
                      </div>
                    </td>
                  </tr>
                  {buka === p.id && (
                    <tr>
                      <td colSpan={9} style={{ padding: '8px 0 16px' }}>
                        {p.catatan && <p className="kpi-meta-row" style={{ marginBottom: 8 }}>{p.catatan}</p>}
                        <RincianHPP h={p.hpp} satuan={p.satuan} />
                      </td>
                    </tr>
                  )}
                </Fragment>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      </>}

      {tab === 'supplier' && (
      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Supplier</div>

        {supForm && (
          <form className="form-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', marginTop: 12 }} onSubmit={simpanSupplier}>
            <label>
              <span className="kpi-label">Nama Supplier</span>
              <input className="input-field" required value={supForm.nama} onChange={(e) => setSupForm({ ...supForm, nama: e.target.value })} />
            </label>
            <label>
              <span className="kpi-label">Nama Kontak</span>
              <input className="input-field" value={supForm.kontak} onChange={(e) => setSupForm({ ...supForm, kontak: e.target.value })} />
            </label>
            <label>
              <span className="kpi-label">Telepon</span>
              <input className="input-field" value={supForm.telepon} onChange={(e) => setSupForm({ ...supForm, telepon: e.target.value })} />
            </label>
            <label>
              <span className="kpi-label">Email</span>
              <input className="input-field" type="email" value={supForm.email} onChange={(e) => setSupForm({ ...supForm, email: e.target.value })} />
            </label>
            <label>
              <span className="kpi-label">Alamat</span>
              <input className="input-field" value={supForm.alamat} onChange={(e) => setSupForm({ ...supForm, alamat: e.target.value })} />
            </label>
            <label>
              <span className="kpi-label">Tempo Bayar (hari)</span>
              <input
                className="input-field tabular" placeholder="0" {...angka(supForm.tempo_hari, (n) => setSupForm({ ...supForm, tempo_hari: n }))}
              />
              <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>0 = bayar tunai</span>
            </label>
            <label>
              <span className="kpi-label">Catatan</span>
              <input className="input-field" value={supForm.catatan ?? ''} onChange={(e) => setSupForm({ ...supForm, catatan: e.target.value })} />
            </label>
            <div style={{ display: 'flex', alignItems: 'flex-end', gap: 10 }}>
              <button className="btn-cta" style={{ padding: '9px 20px', fontSize: 13 }} type="submit" disabled={busy}>
                {busy ? 'Menyimpan…' : 'Simpan'}
              </button>
              <button type="button" className="nav-item" style={{ width: 'auto', padding: '9px 16px' }} onClick={() => setSupForm(null)}>Batal</button>
            </div>
          </form>
        )}

        <div className="table-scroll">
          <table className="data-table tabular" style={{ minWidth: 880 }}>
            <thead>
              <tr>
                <th>Supplier</th><th>Kontak</th><th>Telepon</th><th>Alamat</th>
                <th style={{ textAlign: 'right' }}>Tempo</th><th style={{ textAlign: 'right' }}>Utang Belum Dibayar</th><th />
              </tr>
            </thead>
            <tbody>
              {supplier.length === 0 && (
                <tr><td colSpan={7} style={{ color: 'var(--text-muted)' }}>Belum ada supplier terdaftar.</td></tr>
              )}
              {supplier.map((s) => (
                <tr key={s.id}>
                  <td style={{ fontWeight: 600 }}>{s.nama}</td>
                  <td>{s.kontak || '—'}</td>
                  <td>{s.telepon || '—'}</td>
                  <td style={{ color: 'var(--text-muted)' }}>{s.alamat || '—'}</td>
                  <td style={{ textAlign: 'right' }}>{s.tempo_hari > 0 ? `${s.tempo_hari} hari` : 'Tunai'}</td>
                  <td style={{ textAlign: 'right', color: s.utang_belum_dibayar > 0 ? RED : 'var(--text-muted)' }}>
                    {s.utang_belum_dibayar > 0 ? `${formatRp(s.utang_belum_dibayar)} (${s.jumlah_tagihan} tagihan)` : '—'}
                  </td>
                  <td>
                    <div style={{ display: 'flex', gap: 4 }}>
                      <button className="nav-item" style={aksiBtn} onClick={() => { setError(''); setSupEditId(s.id); setSupForm(toSupplierInput(s)) }}>Ubah</button>
                      <button className="nav-item" style={{ ...aksiBtn, color: RED }} onClick={() => void hapusSupplier(s)}>Hapus</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <p className="kpi-meta-row" style={{ marginTop: 10 }}>
          Kolom utang dicocokkan dari <strong>nama pihak</strong> di menu Piutang &amp; Utang, jadi ia perkiraan —
          bukan saldo resmi. Kalau nama di tagihan berbeda ejaannya, utangnya tidak terhitung di sini.
        </p>
      </div>
      )}

      {produksi && (
        <ProduksiDialog produkList={produk} awal={produksi.awal} stok={stok} mulai={stokMulai} onClose={() => setProduksi(null)}
          onDone={(m) => { setProduksi(null); setPesan(m); setTab('produksi'); setMuatUlang((n) => n + 1); muat() }} />
      )}
    </>
  )
}
