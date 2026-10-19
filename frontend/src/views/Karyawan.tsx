import { useCallback, useEffect, useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import {
  createKaryawan, deleteKaryawan, getKaryawan, updateKaryawan,
  type Karyawan, type KaryawanInput, type KaryawanSummary,
} from '@/lib/api'
import { formatJt, formatRp } from '@/lib/format'

const RED = '#fb7185'
const BLUE = '#38bdf8'

const KOSONG: KaryawanInput = {
  nama: '', jabatan: '', tanggal_masuk: '', gaji_pokok: 0, tunjangan: 0, ptkp: 'TK/0', npwp: '', aktif: true,
}

function toInput(k: Karyawan): KaryawanInput {
  const { id: _id, ...rest } = k
  return rest
}

export function KaryawanView() {
  const [data, setData] = useState<KaryawanSummary | null>(null)
  const [form, setForm] = useState<KaryawanInput | null>(null)
  const [editId, setEditId] = useState<number | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const muat = useCallback(() => {
    getKaryawan().then(setData, (e: Error) => setError(e.message))
  }, [])

  useEffect(muat, [muat])

  async function simpan(e: React.FormEvent) {
    e.preventDefault()
    if (!form) return
    setBusy(true)
    try {
      await (editId === null ? createKaryawan(form) : updateKaryawan(editId, form))
      setForm(null)
      setEditId(null)
      muat()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  async function hapus(k: Karyawan) {
    if (!confirm(`Hapus ${k.nama} dari daftar karyawan?`)) return
    try {
      await deleteKaryawan(k.id)
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

  const list = data.karyawan ?? []

  return (
    <>
      <PageHeader title="Karyawan" subtitle="Data dasar karyawan & rekap beban gaji bulanan">
        <button
          className="btn-cta"
          style={{ padding: '8px 18px', fontSize: 13 }}
          onClick={() => { setError(''); setEditId(null); setForm({ ...KOSONG }) }}
        >
          + Tambah Karyawan
        </button>
      </PageHeader>

      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', marginBottom: 12 }}>{error}</div>}

      <div className="kpi-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(210px, 1fr))' }}>
        <div className="card">
          <div className="kpi-label">Karyawan Aktif</div>
          <div className="kpi-value tabular">{data.total_aktif}</div>
          <div className="kpi-meta-row"><span>dari {list.length} tercatat</span></div>
        </div>
        <div className="card">
          <div className="kpi-label">Beban Gaji per Bulan</div>
          <div className="kpi-value tabular" style={{ color: BLUE }}>{formatJt(data.beban_gaji_bulanan)}</div>
          <div className="kpi-meta-row"><span>gaji pokok + tunjangan, karyawan aktif</span></div>
        </div>
      </div>

      {form && (
        <form className="card" style={{ padding: 20, marginBottom: 14 }} onSubmit={simpan}>
          <div className="chart-title">{editId === null ? 'Tambah Karyawan' : 'Ubah Karyawan'}</div>
          <div className="form-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))' }}>
            <label>
              <span className="kpi-label">Nama</span>
              <input className="input-field" required value={form.nama} onChange={(e) => setForm({ ...form, nama: e.target.value })} />
            </label>
            <label>
              <span className="kpi-label">Jabatan</span>
              <input className="input-field" value={form.jabatan} onChange={(e) => setForm({ ...form, jabatan: e.target.value })} />
            </label>
            <label>
              <span className="kpi-label">Tanggal Masuk</span>
              <input
                className="input-field" type="date" required value={form.tanggal_masuk}
                onChange={(e) => setForm({ ...form, tanggal_masuk: e.target.value })}
              />
            </label>
            <label>
              <span className="kpi-label">Gaji Pokok</span>
              <input
                className="input-field tabular" type="number" min={1} required value={form.gaji_pokok || ''}
                onChange={(e) => setForm({ ...form, gaji_pokok: Number(e.target.value) })}
              />
            </label>
            <label>
              <span className="kpi-label">Tunjangan Tetap</span>
              <input
                className="input-field tabular" type="number" min={0} value={form.tunjangan || ''}
                onChange={(e) => setForm({ ...form, tunjangan: Number(e.target.value) })}
              />
            </label>
            <label>
              <span className="kpi-label">Status PTKP</span>
              <select className="input-field" value={form.ptkp} onChange={(e) => setForm({ ...form, ptkp: e.target.value })}>
                {data.ptkp_codes.map((c) => <option key={c} value={c}>{c}</option>)}
              </select>
            </label>
            <label>
              <span className="kpi-label">NPWP</span>
              <input className="input-field" value={form.npwp} onChange={(e) => setForm({ ...form, npwp: e.target.value })} />
            </label>
            <label>
              <span className="kpi-label">Status</span>
              <select
                className="input-field"
                value={form.aktif ? 'aktif' : 'nonaktif'}
                onChange={(e) => setForm({ ...form, aktif: e.target.value === 'aktif' })}
              >
                <option value="aktif">Aktif</option>
                <option value="nonaktif">Tidak aktif / resign</option>
              </select>
            </label>
          </div>
          <div style={{ display: 'flex', gap: 10, marginTop: 16 }}>
            <button className="btn-cta" style={{ padding: '9px 20px', fontSize: 13 }} type="submit" disabled={busy}>
              {busy ? 'Menyimpan…' : 'Simpan'}
            </button>
            <button type="button" className="nav-item" style={{ width: 'auto', padding: '9px 16px' }} onClick={() => setForm(null)}>
              Batal
            </button>
          </div>
        </form>
      )}

      <div className="card" style={{ padding: 20 }}>
        <div className="chart-title">Daftar Karyawan</div>
        <div className="chart-subtitle">Bisa diisi sekaligus lewat Pengaturan → Impor Data Awal</div>
        <div className="table-scroll">
          <table className="data-table tabular" style={{ minWidth: 820 }}>
            <thead>
              <tr>
                <th>Nama</th>
                <th>Jabatan</th>
                <th>Masuk</th>
                <th style={{ textAlign: 'right' }}>Gaji Pokok</th>
                <th style={{ textAlign: 'right' }}>Tunjangan</th>
                <th style={{ textAlign: 'right' }}>Penghasilan / Bln</th>
                <th>PTKP</th>
                <th>Status</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {list.length === 0 && (
                <tr><td colSpan={9} style={{ color: 'var(--text-muted)' }}>Belum ada karyawan tercatat.</td></tr>
              )}
              {list.map((k) => (
                <tr key={k.id}>
                  <td style={{ fontWeight: 600 }}>{k.nama}</td>
                  <td>{k.jabatan || '—'}</td>
                  <td>{k.tanggal_masuk}</td>
                  <td style={{ textAlign: 'right' }}>{formatRp(k.gaji_pokok)}</td>
                  <td style={{ textAlign: 'right' }}>{formatRp(k.tunjangan)}</td>
                  <td style={{ textAlign: 'right', color: BLUE }}>{formatRp(k.gaji_pokok + k.tunjangan)}</td>
                  <td>{k.ptkp}</td>
                  <td>
                    <span className={'pill-status ' + (k.aktif ? 'pill-green' : 'pill-status-muted')} style={{ fontSize: 10 }}>
                      {k.aktif ? 'Aktif' : 'Tidak aktif'}
                    </span>
                  </td>
                  <td style={{ whiteSpace: 'nowrap' }}>
                    <button
                      className="nav-item" style={{ width: 'auto', padding: '4px 8px', fontSize: 12 }}
                      onClick={() => { setError(''); setEditId(k.id); setForm(toInput(k)) }}
                    >
                      Ubah
                    </button>
                    <button
                      className="nav-item" style={{ width: 'auto', padding: '4px 8px', fontSize: 12, color: RED }}
                      onClick={() => void hapus(k)}
                    >
                      Hapus
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="note-box warn">
        Halaman ini baru menyimpan data dasar. Perhitungan <strong>PPh 21</strong> dan pembuatan transaksi gaji
        otomatis belum ada — gaji masih dicatat manual lewat menu Transaksi (jenis "Gaji &amp; Upah"). Status
        PTKP sudah disimpan supaya kalkulator PPh 21 nanti tinggal membacanya.
      </div>
    </>
  )
}
