import { useCallback, useEffect, useState } from 'react'
import { angka } from '@/lib/angka'
import {
  createPelanggan, deletePelanggan, getPelanggan, pelangganDariData, updatePelanggan,
  type Pelanggan, type PelangganInput,
} from '@/lib/api'

// Daftar pelanggan (checklist tester No 24), pola sama dengan daftar Supplier.

const KOSONG: PelangganInput = { nama: '', kontak: '', telepon: '', email: '', alamat: '', npwp: '', tempo_hari: 0, catatan: '' }

/** onBerubah: daftar terbaru (mis. untuk pilihan nama di form invoice). onRiwayat: buka riwayat pelanggan. */
export function DaftarPelanggan({ onBerubah, onRiwayat }: { onBerubah?: (d: Pelanggan[]) => void; onRiwayat?: (p: Pelanggan) => void }) {
  const [list, setList] = useState<Pelanggan[] | null>(null)
  const [form, setForm] = useState<PelangganInput | null>(null)
  const [editId, setEditId] = useState<number | null>(null)
  const [error, setError] = useState('')
  const [pesan, setPesan] = useState('')
  const [busy, setBusy] = useState(false)

  const muat = useCallback(() => getPelanggan().then((d) => { setList(d); onBerubah?.(d) }, (e: Error) => setError(e.message)), [onBerubah])
  useEffect(() => { void muat() }, [muat])

  /** f mengembalikan pesan sukses (atau memakai ok). */
  async function jalan(f: () => Promise<unknown>, ok: string) {
    setBusy(true); setError(''); setPesan('')
    try {
      const r = await f()
      setPesan(typeof r === 'string' ? r : ok)
      await muat()
    } catch (e) { setError((e as Error).message) } finally { setBusy(false) }
  }
  async function simpan(e: React.FormEvent) {
    e.preventDefault()
    if (!form) return
    await jalan(() => (editId === null ? createPelanggan(form) : updatePelanggan(editId, form)), editId === null ? 'Pelanggan ditambahkan.' : 'Pelanggan diperbarui.')
    setForm(null); setEditId(null)
  }
  const isi = (k: keyof PelangganInput, label: string, type = 'text') => (
    <label>
      <span className="kpi-label">{label}</span>
      <input className="input-field" type={type} required={k === 'nama'} value={(form?.[k] as string) ?? ''} onChange={(e) => setForm((f) => f && { ...f, [k]: e.target.value })} />
    </label>
  )

  return (
    <div className="card" style={{ padding: 20, marginBottom: 14 }}>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div className="chart-title">Daftar pelanggan</div>
          <div className="chart-subtitle">Nama, alamat, dan NPWP terisi otomatis saat membuat invoice.</div>
        </div>
        <div className="flex gap-2">
          <button type="button" className="nav-item" style={{ width: 'auto', padding: '8px 14px', fontSize: 13 }} disabled={busy}
            onClick={() => void jalan(async () => `${(await pelangganDariData()).ditambah} pelanggan ditambahkan dari data lama.`, '')}
            title="Ambil nama pelanggan dari transaksi penjualan & invoice yang sudah ada">
            Ambil dari data lama
          </button>
          <button type="button" className="btn-cta" style={{ padding: '8px 16px', fontSize: 13 }} onClick={() => { setEditId(null); setForm({ ...KOSONG }) }}>+ Pelanggan</button>
        </div>
      </div>
      {error && <p className="text-sm text-destructive" role="alert" style={{ marginTop: 8 }}>{error}</p>}
      {pesan && <p className="text-sm" role="status" style={{ marginTop: 8 }}>{pesan}</p>}

      {form && (
        <form className="form-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', marginTop: 12 }} onSubmit={simpan}>
          {isi('nama', 'Nama pelanggan')}
          {isi('kontak', 'Nama kontak')}
          {isi('telepon', 'Telepon')}
          {isi('email', 'Email', 'email')}
          {isi('alamat', 'Alamat')}
          {isi('npwp', 'NPWP')}
          <label>
            <span className="kpi-label">Tempo bayar (hari)</span>
            <input className="input-field tabular" placeholder="0" {...angka(form.tempo_hari, (n) => setForm({ ...form, tempo_hari: n }))} />
          </label>
          {isi('catatan', 'Catatan')}
          <div style={{ display: 'flex', gap: 10, alignItems: 'end' }}>
            <button className="btn-cta" style={{ padding: '9px 18px', fontSize: 13 }} type="submit" disabled={busy}>{busy ? 'Menyimpan…' : 'Simpan'}</button>
            <button type="button" className="nav-item" style={{ width: 'auto', padding: '9px 14px' }} onClick={() => setForm(null)}>Batal</button>
          </div>
        </form>
      )}

      <div className="table-scroll" style={{ marginTop: 12 }}>
        <table className="data-table">
          <thead><tr><th>Nama</th><th>Telepon</th><th>Email</th><th>Alamat</th><th>NPWP</th><th style={{ textAlign: 'right' }}>Tempo</th><th /></tr></thead>
          <tbody>
            {list === null ? <tr><td colSpan={7}>Memuat…</td></tr> : list.length === 0 ? (
              <tr><td colSpan={7} className="text-muted-foreground">Belum ada pelanggan. Tambah sendiri atau ambil dari data lama.</td></tr>
            ) : list.map((p) => (
              <tr key={p.id}>
                <td>{onRiwayat ? <button type="button" className="underline underline-offset-4" onClick={() => onRiwayat(p)}>{p.nama}</button> : p.nama}</td>
                <td>{p.telepon}</td><td>{p.email}</td><td>{p.alamat}</td><td>{p.npwp}</td>
                <td className="tabular" style={{ textAlign: 'right' }}>{p.tempo_hari ? `${p.tempo_hari} hari` : 'Tunai'}</td>
                <td style={{ whiteSpace: 'nowrap', textAlign: 'right' }}>
                  <button type="button" className="nav-item" style={{ width: 'auto', padding: '4px 10px', fontSize: 12 }} onClick={() => { setEditId(p.id); setForm({ ...KOSONG, ...p }) }}>Ubah</button>
                  <button type="button" className="nav-item" style={{ width: 'auto', padding: '4px 10px', fontSize: 12, color: 'var(--text-danger)' }}
                    onClick={() => { if (confirm(`Hapus pelanggan ${p.nama}? Invoice yang sudah ada tidak berubah.`)) void jalan(() => deletePelanggan(p.id), 'Pelanggan dihapus.') }}>Hapus</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
