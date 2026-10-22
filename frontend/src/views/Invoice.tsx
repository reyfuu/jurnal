import { useEffect, useState, useRef } from 'react'
import { ThUrut } from '@/components/ThUrut'
import { urutkan, useKolomUrut, type Pengurut } from '@/lib/urut'
import { useGulirKeForm } from '@/lib/gulir'
import { bacaDraft, hapusDraft, useSimpanDraft } from '@/lib/draft'

const DRAFT_INV = 'invoice-baru'
import { PageHeader } from '@/components/PageHeader'
import { IsiAIBanner } from '@/components/IsiAIBanner'
import { useIsiAI, type SinyalAI } from '@/lib/useIsiAI'
import { MenuImportDialog } from '@/views/MenuImportDialog'
import {
  bukukanInvoice, createInvoice, deleteInvoice, getInvoices, getPelanggan, importInvoice, INVOICE_TEMPLATE_URL, invoicePdfUrl, updateInvoice,
  type InvoiceInfo, type InvoiceInput, type InvoiceItem, type InvoiceRow, type Pelanggan,
} from '@/lib/api'
import { DaftarPelanggan } from '@/views/Pelanggan'
import { RiwayatPihakDialog } from '@/views/RiwayatPihak'
import { ekspor } from '@/lib/garansi'
import { formatRp } from '@/lib/format'
import { angka } from '@/lib/angka'

const AMBER = 'var(--text-warn)'

const KOLOM_INVOICE: Pengurut<InvoiceRow> = {
  nomor: (r) => r.nomor, pelanggan: (r) => r.pelanggan_nama, terbit: (r) => r.tanggal, tempo: (r) => r.jatuh_tempo,
  status: (r) => r.status, total: (r) => r.totals.total,
}
const RED = 'var(--text-danger)'
const GREEN = 'var(--accent-green)'

const BARIS_KOSONG: InvoiceItem = { deskripsi: '', qty: 1, satuan: '', harga_satuan: 0 }

const STATUS_LABEL: Record<string, string> = {
  draft: 'Draft', terkirim: 'Belum Dibayar', lunas: 'Lunas',
}

function kosong(nomor: string, ppnBps: number): InvoiceInput {
  // Menurut WIB: toISOString() memakai UTC sehingga sebelum pukul 07.00 jatuh ke kemarin.
  const hariIni = new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Jakarta' })
  const tempo = new Date(Date.now() + 30 * 864e5).toLocaleDateString('sv-SE', { timeZone: 'Asia/Jakarta' })
  return {
    nomor, no_po: '', status: 'draft', tanggal: hariIni, jatuh_tempo: tempo,
    pelanggan_nama: '', pelanggan_alamat: '', pelanggan_npwp: '',
    items: [{ ...BARIS_KOSONG }], diskon_rp: 0, ppn_bps: ppnBps, catatan: '', tanggal_lunas: '',
  }
}

const hariIniWIB = () => new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Jakarta' })

function toInput(r: InvoiceRow): InvoiceInput {
  return {
    nomor: r.nomor, no_po: r.no_po ?? '', status: r.status, tanggal: r.tanggal,
    jatuh_tempo: r.jatuh_tempo, pelanggan_nama: r.pelanggan_nama,
    pelanggan_alamat: r.pelanggan_alamat ?? '', pelanggan_npwp: r.pelanggan_npwp ?? '',
    items: r.items.map((i) => ({ ...i })), diskon_rp: r.diskon_rp,
    ppn_bps: r.ppn_bps, catatan: r.catatan ?? '', tanggal_lunas: r.tanggal_lunas ?? '', dibukukan: r.dibukukan,
  }
}

/** Total dihitung dengan rumus yang sama dengan server: PPN dari DPP, bukan dari subtotal. */
function hitung(f: InvoiceInput) {
  const subtotal = f.items.reduce((s, i) => s + i.qty * i.harga_satuan, 0)
  const diskon = Math.min(f.diskon_rp || 0, subtotal)
  const dpp = subtotal - diskon
  const ppn = Math.trunc((dpp * (f.ppn_bps || 0)) / 10000)
  return { subtotal, diskon, dpp, ppn, total: dpp + ppn }
}

/** isiAI: isian dari Tanya AI untuk invoice baru (bisa diurungkan). */
export function InvoiceView({ isiAI }: { isiAI?: SinyalAI }) {
  const [info, setInfo] = useState<InvoiceInfo | null>(null)
  const [form, setForm] = useState<InvoiceInput | null>(null)
  const formRef = useRef<HTMLFormElement>(null)
  useGulirKeForm(formRef, !!form)
  const [editId, setEditId] = useState<number | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [impor, setImpor] = useState(false)
  const [lihatPelanggan, setLihatPelanggan] = useState(false)
  const [riwayat, setRiwayat] = useState<string | null>(null)
  const [pelanggan, setPelanggan] = useState<Pelanggan[]>([])
  useEffect(() => { getPelanggan().then(setPelanggan, () => {}) }, [])
  // Pilih nama dari daftar pelanggan: alamat & NPWP yang masih kosong ikut terisi.
  const pilihPelanggan = (nama: string) => setForm((f) => {
    if (!f) return f
    const p = pelanggan.find((x) => x.nama.toLowerCase() === nama.trim().toLowerCase())
    return { ...f, pelanggan_nama: nama, pelanggan_alamat: f.pelanggan_alamat || p?.alamat || '', pelanggan_npwp: f.pelanggan_npwp || p?.npwp || '' }
  })
  const urut = useKolomUrut()
  // Invoice baru yang belum tersimpan disimpan sebagai draft (error / muat ulang tidak menghapusnya).
  useSimpanDraft(form && editId === null ? DRAFT_INV : null, form)

  const muat = () => getInvoices().then(setInfo, (e) => setError((e as Error).message))
  useEffect(() => { void muat() }, [])
  const isiAi = useIsiAI<{ form: InvoiceInput | null; editId: number | null }, Partial<InvoiceInput>>(isiAI, 'invoice', {
    ambil: () => ({ form, editId }),
    pasang: (s) => { setForm(s.form); setEditId(s.editId) },
    isi: (isian) => { setError(''); setEditId(null); setForm({ ...kosong(info!.nomor_berikutnya, info!.ppn_bps), ...isian }) },
    terbuka: form !== null,
    siap: info !== null, // nomor & PPN invoice baru datang dari info
  })

  if (!info) return <p className="workspace-subtitle">{error || 'Memuat faktur…'}</p>

  const rows = info.summary.invoices ?? []
  const s = info.summary

  function buka(r?: InvoiceRow) {
    setError('')
    setEditId(r ? r.id : null)
    setForm(r ? toInput(r) : bacaDraft<InvoiceInput>(DRAFT_INV) ?? kosong(info!.nomor_berikutnya, info!.ppn_bps))
  }

  async function simpan(e: React.FormEvent) {
    e.preventDefault()
    if (!form) return
    setBusy(true)
    try {
      await (editId === null ? createInvoice(form) : updateInvoice(editId, form))
      if (editId === null) hapusDraft(DRAFT_INV)
      setForm(null)
      setEditId(null)
      await muat()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  async function hapus(r: InvoiceRow) {
    if (!confirm(`Hapus faktur ${r.nomor} untuk ${r.pelanggan_nama}?`)) return
    try {
      await deleteInvoice(r.id)
      await muat()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  // Invoice terbit yang dibuat sebelum invoice tersambung ke pembukuan: belum jadi omzet/piutang.
  const belumBuku = rows.filter((r) => !r.dibukukan && r.status !== 'draft')
  const draft = rows.filter((r) => r.status === 'draft')

  async function bukukanSemua() {
    if (!confirm(`Bukukan ${belumBuku.length} invoice sekarang? Penjualannya akan tercatat di Transaksi (dan jadi piutang bila belum lunas). Kalau penjualan ini sudah Anda catat manual, batalkan dan hapus transaksi manualnya dulu supaya tidak dobel.`)) return
    setBusy(true)
    try {
      const h = await bukukanInvoice()
      if (h.gagal.length) setError('Sebagian gagal dibukukan: ' + h.gagal.join('; '))
      await muat()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const t = form ? hitung(form) : null
  const ubahBaris = (i: number, patch: Partial<InvoiceItem>) =>
    setForm((f) => f && { ...f, items: f.items.map((it, n) => (n === i ? { ...it, ...patch } : it)) })

  return (
    <>
      <PageHeader title="Invoice" subtitle="Terbitkan tagihan ke pelanggan dan unduh PDF. Invoice yang dikirim otomatis tercatat sebagai penjualan dan muncul di Piutang & Utang untuk dipantau & ditagih.">
        <button className="nav-item" style={{ width: 'auto', padding: '8px 14px', fontSize: 13 }} onClick={() => setLihatPelanggan((v) => !v)} aria-expanded={lihatPelanggan}>
          Daftar pelanggan
        </button>
        <button className="nav-item" style={{ width: 'auto', padding: '8px 14px', fontSize: 13 }} onClick={() => setImpor(true)}>
          Impor Excel
        </button>
        <button className="btn-cta" style={{ padding: '8px 18px', fontSize: 13 }} onClick={() => buka()}>
          + Buat Invoice
        </button>
      </PageHeader>

      {lihatPelanggan && <DaftarPelanggan onBerubah={setPelanggan} onRiwayat={(p) => setRiwayat(p.nama)} />}
      <RiwayatPihakDialog nama={riwayat} onTutup={() => setRiwayat(null)} />

      <MenuImportDialog
        open={impor} onOpenChange={setImpor} onImported={() => void muat()}
        title="Impor invoice dari Excel"
        description="Satu baris per item; baris bernomor sama jadi satu invoice. Status kosong = Terkirim (langsung masuk penjualan & piutang). Nomor yang sudah ada dan bulan yang sudah tutup buku dilewati."
        templateUrl={INVOICE_TEMPLATE_URL} templateName="template-invoice.xlsx"
        existing={0} satuan="invoice"
        doImport={(f, commit) => importInvoice(f, commit)}
        renderPreview={(r) => (
          <table className="data-table tabular" style={{ marginTop: 0 }}>
            <thead><tr><th>Nomor</th><th>Tanggal</th><th>Pelanggan</th><th>Status</th><th style={{ textAlign: 'right' }}>Item</th><th style={{ textAlign: 'right' }}>Total</th></tr></thead>
            <tbody>
              {(r.invoice ?? []).slice(0, 10).map((x, i) => (
                <tr key={i}>
                  <td>{x.nomor}</td><td>{x.tanggal}</td><td>{x.pelanggan_nama}</td>
                  <td>{STATUS_LABEL[x.status] ?? x.status}</td>
                  <td style={{ textAlign: 'right' }}>{x.items.length}</td>
                  <td style={{ textAlign: 'right' }}>{formatRp(hitung(x).total)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      />

      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', marginBottom: 12 }}>{error}</div>}

      {info.profil_lengkap && info.profil_lengkap.length > 0 && (
        <div className="card" style={{ padding: '12px 18px', marginBottom: 14, borderColor: 'color-mix(in srgb, var(--text-warn) 35%, transparent)' }}>
          <div style={{ fontSize: 13, color: AMBER, fontWeight: 600 }}>Lengkapi Profil Usaha sebelum mengirim faktur</div>
          <div style={{ fontSize: 12.5, color: 'var(--text-muted)', marginTop: 4 }}>
            Belum diisi: {info.profil_lengkap.join(', ')}. Faktur tetap bisa dicetak, tapi bagian itu akan kosong di PDF.
          </div>
        </div>
      )}

      {belumBuku.length > 0 && (
        <div className="note-box warn" style={{ marginTop: 0, marginBottom: 14, display: 'flex', gap: 14, alignItems: 'center', flexWrap: 'wrap' }}>
          <span style={{ flex: 1, minWidth: 260 }}>
            <strong>{belumBuku.length} invoice belum masuk pembukuan</strong> ({belumBuku.map((r) => r.nomor).join(', ')}), jadi belum muncul sebagai penjualan,
            piutang, atau kas di laporan. Invoice ini dibuat sebelum invoice tersambung otomatis ke pembukuan.
          </span>
          <button className="btn-cta" style={{ padding: '8px 16px', fontSize: 13 }} disabled={busy} onClick={() => void bukukanSemua()}>Bukukan sekarang</button>
        </div>
      )}
      {draft.length > 0 && (
        <div className="note-box" style={{ marginTop: 0, marginBottom: 14 }}>
          {draft.length} invoice masih <strong>Draft</strong> ({draft.map((r) => r.nomor).join(', ')}): belum dihitung sebagai tagihan maupun piutang.
          Klik <strong>Ubah</strong> lalu ganti status ke <strong>Terkirim</strong> saat invoice dikirim ke pelanggan — otomatis tercatat di Piutang &amp; Utang.
        </div>
      )}

      <div className="kpi-grid" style={{ gridTemplateColumns: 'repeat(3, 1fr)', marginBottom: 16 }}>
        <div className="card" style={{ padding: '14px 18px' }}>
          <div className="kpi-label">Total Tagihan Terbit</div>
          <div className="kpi-value tabular" style={{ fontSize: 20 }}>{formatRp(s.total_tagihan)}</div>
        </div>
        <div className="card" style={{ padding: '14px 18px' }}>
          <div className="kpi-label">Belum Dibayar</div>
          <div className="kpi-value tabular" style={{ fontSize: 20, color: AMBER }}>{formatRp(s.belum_dibayar)}</div>
        </div>
        <div className="card" style={{ padding: '14px 18px' }}>
          <div className="kpi-label">Lewat Jatuh Tempo</div>
          <div className="kpi-value tabular" style={{ fontSize: 20, color: s.jumlah_terlambat ? RED : undefined }}>
            {formatRp(s.terlambat)}
          </div>
          <div className="kpi-meta-row">{s.jumlah_terlambat} faktur</div>
        </div>
      </div>

      {form && (
        <form ref={formRef} style={{ scrollMarginTop: 16, padding: 20, marginBottom: 14 }} className="card" onSubmit={simpan}>
          <div className="chart-title">{editId === null ? 'Buat Invoice' : `Ubah ${form.nomor}`}</div>
          {isiAi.aktif !== null && <div style={{ marginTop: 12 }}><IsiAIBanner ai={isiAi} /></div>}

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(190px, 1fr))', gap: 12, marginTop: 14 }}>
            <Field label="No. Invoice">
              <input className="input-field" value={form.nomor ?? ''} onChange={(e) => setForm({ ...form, nomor: e.target.value })} />
            </Field>
            <Field label="No. PO pelanggan" hint="Dicetak di faktur bila diisi">
              <input className="input-field" value={form.no_po ?? ''} placeholder="mis. PO-KDM-2026-0317"
                onChange={(e) => setForm({ ...form, no_po: e.target.value })} />
            </Field>
            <Field label="Status">
              <select className="input-field" value={form.status}
                onChange={(e) => setForm({ ...form, status: e.target.value, tanggal_lunas: e.target.value === 'lunas' ? (form.tanggal_lunas || hariIniWIB()) : '' })}>
                {info.status.map((v) => <option key={v} value={v}>{STATUS_LABEL[v] ?? v}</option>)}
              </select>
            </Field>
            {form.status === 'lunas' && (
              <Field label="Tanggal dibayar">
                <input className="input-field" type="date" required min={form.tanggal} value={form.tanggal_lunas ?? ''} onChange={(e) => setForm({ ...form, tanggal_lunas: e.target.value })} />
              </Field>
            )}
            <Field label="Tanggal terbit">
              <input className="input-field" type="date" value={form.tanggal} onChange={(e) => setForm({ ...form, tanggal: e.target.value })} />
            </Field>
            <Field label="Jatuh tempo">
              <input className="input-field" type="date" value={form.jatuh_tempo} onChange={(e) => setForm({ ...form, jatuh_tempo: e.target.value })} />
            </Field>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(190px, 1fr))', gap: 12, marginTop: 12 }}>
            <Field label="Nama pelanggan">
              <input className="input-field" required list="daftar-pelanggan" value={form.pelanggan_nama} onChange={(e) => pilihPelanggan(e.target.value)} />
              <datalist id="daftar-pelanggan">{pelanggan.map((p) => <option key={p.id} value={p.nama} />)}</datalist>
            </Field>
            <Field label="NPWP pelanggan">
              <input className="input-field" value={form.pelanggan_npwp ?? ''} onChange={(e) => setForm({ ...form, pelanggan_npwp: e.target.value })} />
            </Field>
            <Field label="Alamat pelanggan">
              <input className="input-field" value={form.pelanggan_alamat ?? ''} onChange={(e) => setForm({ ...form, pelanggan_alamat: e.target.value })} />
            </Field>
          </div>

          <div className="chart-subtitle" style={{ marginTop: 18, marginBottom: 6 }}>Rincian</div>
          <div className="table-scroll">
            <table className="data-table tabular" style={{ minWidth: 700 }}>
              <thead>
                <tr>
                  <th style={{ minWidth: 220 }}>Deskripsi</th>
                  <th style={{ width: 80 }}>Qty</th>
                  <th style={{ width: 90 }}>Satuan</th>
                  <th style={{ width: 150, textAlign: 'right' }}>Harga Satuan</th>
                  <th style={{ width: 140, textAlign: 'right' }}>Jumlah</th>
                  <th style={{ width: 40 }} />
                </tr>
              </thead>
              <tbody>
                {form.items.map((it, i) => (
                  <tr key={i}>
                    <td><input className="input-field" required value={it.deskripsi} onChange={(e) => ubahBaris(i, { deskripsi: e.target.value })} /></td>
                    <td><input className="input-field tabular" {...angka(it.qty, (n) => ubahBaris(i, { qty: n }))} /></td>
                    <td><input className="input-field" value={it.satuan ?? ''} placeholder="pcs" onChange={(e) => ubahBaris(i, { satuan: e.target.value })} /></td>
                    <td><input className="input-field tabular" style={{ textAlign: 'right' }} {...angka(it.harga_satuan, (n) => ubahBaris(i, { harga_satuan: n }))} /></td>
                    <td style={{ textAlign: 'right' }}>{formatRp(it.qty * it.harga_satuan)}</td>
                    <td>
                      {form.items.length > 1 && (
                        <button type="button" className="nav-item" title="Hapus baris"
                          style={{ width: 'auto', padding: '4px 9px', fontSize: 13, color: RED }}
                          onClick={() => setForm({ ...form, items: form.items.filter((_, n) => n !== i) })}>×</button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <button type="button" className="nav-item" style={{ width: 'auto', padding: '6px 14px', marginTop: 8, fontSize: 13 }}
            onClick={() => setForm({ ...form, items: [...form.items, { ...BARIS_KOSONG }] })}>
            + Tambah baris
          </button>

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))', gap: 16, marginTop: 16 }}>
            <div>
              <Field label="Diskon (Rp)">
                <input className="input-field tabular" {...angka(form.diskon_rp, (n) => setForm({ ...form, diskon_rp: n }))} />
              </Field>
              <Field label="PPN (basis poin)" hint={form.ppn_bps ? `${form.ppn_bps / 100}% — dari tarif di Profil Usaha` : 'Non-PKP: tidak memungut PPN'}>
                <input className="input-field" type="number" min={0} max={10000} value={form.ppn_bps}
                  onChange={(e) => setForm({ ...form, ppn_bps: Number(e.target.value) })} />
              </Field>
              <Field label="Catatan">
                <input className="input-field" value={form.catatan ?? ''} onChange={(e) => setForm({ ...form, catatan: e.target.value })} />
              </Field>
            </div>
            <div className="list-rows tabular" style={{ alignSelf: 'end' }}>
              <Total label="Subtotal" v={t!.subtotal} />
              {t!.diskon > 0 && <><Total label="Diskon" v={-t!.diskon} /><Total label="Dasar Pengenaan Pajak" v={t!.dpp} /></>}
              {form.ppn_bps > 0 && <Total label={`PPN ${form.ppn_bps / 100}%`} v={t!.ppn} />}
              <Total label="Total Tagihan" v={t!.total} bold />
            </div>
          </div>

          {editId !== null && form.dibukukan === false && rows.find((r) => r.id === editId)?.status !== 'draft' ? (
            <label className="note-box warn" style={{ display: 'flex', gap: 8, alignItems: 'flex-start', marginTop: 12 }}>
              <input type="checkbox" checked={false} onChange={() => setForm({ ...form, dibukukan: true })} />
              <span><strong>Bukukan invoice ini ke pembukuan.</strong> Invoice ini dibuat sebelum invoice tersambung ke pembukuan, jadi belum tercatat.
                Centang hanya bila penjualannya <strong>belum</strong> Anda catat manual di Transaksi, supaya tidak dobel.</span>
            </label>
          ) : (
            <p className="text-xs text-muted-foreground" style={{ marginTop: 12 }}>
              {form.status === 'draft'
                ? 'Draft belum dicatat ke pembukuan. Ubah status ke Terkirim saat invoice dikirim ke pelanggan.'
                : `Tercatat otomatis: Penjualan ${formatRp(t!.dpp)}${t!.ppn > 0 ? ` + PPN Keluaran ${formatRp(t!.ppn)} (utang PPN, bukan omzet)` : ''}, sebagai ${form.status === 'lunas' ? 'kas masuk' : 'piutang di menu Piutang & Utang'}.`}
            </p>
          )}

          <div style={{ display: 'flex', gap: 10, marginTop: 16 }}>
            <button className="btn-cta" style={{ padding: '9px 20px', fontSize: 13 }} disabled={busy}>
              {busy ? 'Menyimpan…' : editId === null ? 'Simpan Invoice' : 'Simpan Perubahan'}
            </button>
            <button type="button" className="nav-item" style={{ width: 'auto', padding: '9px 16px' }}
              onClick={() => { if (editId === null) hapusDraft(DRAFT_INV); setForm(null); setEditId(null) }}>Batal</button>
          </div>
        </form>
      )}

      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Daftar Invoice</div>
        <div className="chart-subtitle">Unduh PDF untuk dikirim ke pelanggan</div>
        <div className="table-scroll">
          <table className="data-table tabular" style={{ minWidth: 880 }}>
            <thead>
              <tr>
                <ThUrut k="nomor" urut={urut}>No. Invoice</ThUrut>
                <ThUrut k="pelanggan" urut={urut}>Pelanggan</ThUrut>
                <ThUrut k="terbit" urut={urut}>Terbit</ThUrut>
                <ThUrut k="tempo" urut={urut}>Jatuh Tempo</ThUrut>
                <ThUrut k="status" urut={urut}>Status</ThUrut>
                <ThUrut k="total" urut={urut} kanan>Total</ThUrut>
                <th style={{ width: 170 }} />
              </tr>
            </thead>
            <tbody>
              {rows.length === 0 && (
                <tr><td colSpan={7} style={{ color: 'var(--text-muted)' }}>Belum ada faktur. Klik "Buat Invoice" untuk menerbitkan yang pertama.</td></tr>
              )}
              {urutkan(rows, urut.kunci ? KOLOM_INVOICE[urut.kunci] : undefined, urut.arah).map((r) => (
                <tr key={r.id}>
                  <td>
                    <div style={{ fontWeight: 600 }}>{r.nomor}</div>
                    {r.no_po && <div style={{ fontSize: 11.5, color: 'var(--text-muted)' }}>PO {r.no_po}</div>}
                  </td>
                  <td>{r.pelanggan_nama}</td>
                  <td>{r.tanggal}</td>
                  <td>
                    {r.jatuh_tempo}
                    {r.hari_terlambat > 0 && <div style={{ fontSize: 11.5, color: RED }}>telat {r.hari_terlambat} hari</div>}
                  </td>
                  <td>
                    <span className={'pill-status ' + (r.status === 'lunas' ? 'pill-green' : r.hari_terlambat > 0 ? 'pill-red' : 'pill-amber')}>
                      {r.hari_terlambat > 0 && r.status !== 'lunas' ? 'Lewat Tempo' : STATUS_LABEL[r.status] ?? r.status}
                    </span>
                    {r.status === 'draft' && <div style={{ fontSize: 11, marginTop: 3, color: 'var(--text-muted)' }}>belum jadi piutang</div>}
                    {r.status !== 'draft' && (
                      <div style={{ fontSize: 11, marginTop: 3, color: r.dibukukan ? 'var(--text-muted)' : AMBER }}>
                        {r.dibukukan ? `tercatat · trx #${r.transaksi_id}${r.transaksi_ppn_id ? ` & #${r.transaksi_ppn_id}` : ''}` : 'belum dibukukan'}
                      </div>
                    )}
                  </td>
                  <td style={{ textAlign: 'right', color: r.status === 'lunas' ? GREEN : undefined }}>{formatRp(r.totals.total)}</td>
                  <td style={{ display: 'flex', gap: 6, justifyContent: 'flex-end' }}>
                    <button className="btn-cta" style={{ padding: '6px 12px', fontSize: 12 }} onClick={() => void ekspor(invoicePdfUrl(r.id))}>PDF</button>
                    <button className="nav-item" style={{ width: 'auto', padding: '4px 8px', fontSize: 12 }} onClick={() => buka(r)}>Ubah</button>
                    <button className="nav-item" style={{ width: 'auto', padding: '4px 8px', fontSize: 12, color: RED }} onClick={() => hapus(r)}>Hapus</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </>
  )
}

function Field({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) {
  return (
    <label style={{ display: 'block', marginBottom: 10 }}>
      <span className="kpi-label" style={{ display: 'block', marginBottom: 4 }}>{label}</span>
      {children}
      {hint && <span style={{ display: 'block', fontSize: 11.5, color: 'var(--text-muted)', marginTop: 3 }}>{hint}</span>}
    </label>
  )
}

function Total({ label, v, bold }: { label: string; v: number; bold?: boolean }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'space-between', gap: 20, padding: '7px 0', fontWeight: bold ? 700 : 400, fontSize: bold ? 15 : 13 }}>
      <span style={{ color: bold ? undefined : 'var(--text-muted)' }}>{label}</span>
      <span>{formatRp(v)}</span>
    </div>
  )
}
