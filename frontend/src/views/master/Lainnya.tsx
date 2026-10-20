import { useEffect, useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { listAudit, listKlien, listPembayaran, type AuditRow, type PembayaranRow } from '@/lib/api'
import { formatRp, formatTanggal } from '@/lib/format'

const ORDER: Record<string, [string, string]> = {
  lunas: ['pill-green', 'Lunas'], manual: ['pill-amber', 'Manual'], pending: ['pill-status-muted', 'Menunggu'], gagal: ['pill-red', 'Gagal'],
}

export function Pembayaran() {
  const [rows, setRows] = useState<PembayaranRow[] | null>(null)
  const [error, setError] = useState('')
  useEffect(() => { listPembayaran().then(setRows, (e: Error) => setError(e.message)) }, [])
  return (
    <>
      <PageHeader title="Pembayaran" subtitle="Order langganan semua klien yang bisa Anda lihat. Garansi 7 hari gugur begitu klien mengekspor data." />
      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start' }}>{error}</div>}
      <div className="card" style={{ padding: 20 }}>
        <div className="table-scroll">
          <table className="data-table" style={{ minWidth: 760 }}>
            <thead><tr><th>Order</th><th>Klien</th><th>Paket</th><th style={{ textAlign: 'right' }}>Jumlah</th><th>Status</th><th>Tanggal</th><th>Garansi 7 hari</th></tr></thead>
            <tbody>
              {rows?.map((r) => {
                const [pill, label] = ORDER[r.order.status] ?? ['pill-status-muted', r.order.status]
                return (
                  <tr key={r.order.id}>
                    <td className="tabular text-xs">{r.order.id}</td>
                    <td>{r.klien_nama}</td>
                    <td>{r.order.paket}</td>
                    <td className="tabular" style={{ textAlign: 'right' }}>{formatRp(r.order.jumlah)}</td>
                    <td><span className={'pill-status ' + pill}>{label}</span></td>
                    <td className="tabular">{formatTanggal(r.order.waktu)}</td>
                    <td>{!r.garansi ? <span className="text-muted-foreground">—</span>
                      : r.garansi.berlaku ? <span className="pill-status pill-green">Berlaku s/d {formatTanggal(r.garansi.sampai)}</span>
                        : <span className="text-xs text-muted-foreground">Tidak berlaku: {r.garansi.alasan}</span>}</td>
                  </tr>
                )
              })}
              {rows?.length === 0 && <tr><td colSpan={7} className="text-center text-muted-foreground" style={{ padding: 24 }}>Belum ada pembayaran.</td></tr>}
            </tbody>
          </table>
        </div>
      </div>
    </>
  )
}

export function AuditLog() {
  const [rows, setRows] = useState<AuditRow[] | null>(null)
  const [error, setError] = useState('')
  const [nama, setNama] = useState<Record<number, string>>({})
  useEffect(() => {
    listAudit().then(setRows, (e: Error) => setError(e.message))
    // Nama usaha untuk kolom Klien; role tanpa izin lihat klien tetap melihat nomornya.
    listKlien().then((ks) => setNama(Object.fromEntries(ks.map((k) => [k.id, k.nama]))), () => {})
  }, [])
  return (
    <>
      <PageHeader title="Audit Log" subtitle="Siapa di tim membuka atau mengubah apa, di klien mana. Catatan tidak bisa diubah atau dihapus." />
      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start' }}>{error}</div>}
      <div className="card" style={{ padding: 20 }}>
        <div className="table-scroll">
          <table className="data-table" style={{ minWidth: 760 }}>
            <thead><tr><th>Waktu</th><th>Oleh</th><th>Aksi</th><th>Klien</th><th>Alasan / catatan</th></tr></thead>
            <tbody>
              {rows?.map((a) => (
                <tr key={a.id}>
                  <td className="tabular" style={{ whiteSpace: 'nowrap' }}>{new Date(a.waktu).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' })}</td>
                  <td><div className="font-semibold">{a.nama}</div><div className="text-xs text-muted-foreground">{a.peran}</div></td>
                  <td>{a.aksi}</td>
                  <td>{a.klien_id ? nama[a.klien_id] ?? `#${a.klien_id}` : '—'}</td>
                  <td className="text-muted-foreground">{a.keterangan || '—'}</td>
                </tr>
              ))}
              {rows?.length === 0 && <tr><td colSpan={5} className="text-center text-muted-foreground" style={{ padding: 24 }}>Belum ada catatan.</td></tr>}
            </tbody>
          </table>
        </div>
      </div>
    </>
  )
}
