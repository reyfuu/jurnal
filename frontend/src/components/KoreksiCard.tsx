import type { ReactNode } from 'react'
import type { KoreksiRow, KoreksiStatus, Tx } from '@/lib/api'
import { formatRp } from '@/lib/format'
import { formatTanggal, labelBulan } from '@/lib/transaksi'

const KOREKSI_STATUS: Record<KoreksiStatus, [string, string]> = {
  menunggu: ['pill-amber', 'Menunggu keputusan'],
  disetujui: ['pill-green', 'Disetujui'],
  ditolak: ['pill-red', 'Ditolak'],
  kedaluwarsa: ['pill-status-muted', 'Kedaluwarsa'],
}

// Kolom yang dibandingkan; urutan = urutan tampil.
const KOLOM: [keyof Tx, string, (t: Tx) => string][] = [
  ['tanggal', 'Tanggal', (t) => formatTanggal(t.tanggal)],
  ['jenis', 'Jenis', (t) => t.jenis],
  ['akun', 'Akun', (t) => t.akun || 'otomatis'],
  ['nominal', 'Nominal', (t) => formatRp(t.nominal)],
  ['keterangan', 'Keterangan', (t) => t.keterangan || '—'],
  ['pihak', 'Pihak', (t) => t.pihak || '—'],
  ['lunas', 'Status bayar', (t) => (t.lunas ? 'Lunas' : 'Belum lunas')],
]

/** Satu usulan koreksi: perbandingan sebelum/sesudah, alasan, dampak laba, dan aksi (opsional). */
export function KoreksiCard({ k, sub, aksi }: { k: KoreksiRow; sub: ReactNode; aksi?: ReactNode }) {
  const [pill, label] = KOREKSI_STATUS[k.status]
  const baris = KOLOM.filter(([f]) => k.jenis === 'hapus' || k.sebelum[f] !== k.usulan?.[f] || f === 'keterangan')
  return (
    <article className="card" style={{ padding: 18, display: 'grid', gridTemplateColumns: 'minmax(0, 1fr)', gap: 12 }}>
      <div className="flex flex-wrap items-center gap-2">
        <span className="tabular text-xs text-muted-foreground">{k.nomor}</span>
        <span className="font-semibold">{k.jenis === 'hapus' ? 'Hapus' : 'Ubah'} transaksi {formatTanggal(k.sebelum.tanggal)}</span>
        <span className={'pill-status ' + pill}>{label}</span>
        <span className="text-xs text-muted-foreground" style={{ marginLeft: 'auto' }}>{sub}</span>
      </div>
      <div className="table-scroll">
        <table className="data-table" style={{ minWidth: 460 }}>
          <thead><tr><th>Kolom</th><th>Sekarang</th><th>Diusulkan</th></tr></thead>
          <tbody>
            {baris.map(([f, l, fmt]) => {
              const beda = k.jenis === 'hapus' || k.sebelum[f] !== k.usulan?.[f]
              return (
                <tr key={f}>
                  <td className="text-muted-foreground">{l}</td>
                  <td style={beda ? { textDecoration: 'line-through', color: 'var(--text-secondary)' } : undefined}>{fmt(k.sebelum)}</td>
                  <td style={beda ? { fontWeight: 600, color: k.jenis === 'hapus' ? 'var(--accent-red)' : 'var(--accent-green)' } : undefined}>
                    {k.jenis === 'hapus' ? '(dihapus)' : fmt(k.usulan!)}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      <p className="text-sm" style={{ borderLeft: '3px solid var(--accent-amber)', paddingLeft: 10 }}><strong>Alasan:</strong> {k.alasan}</p>
      {k.dampak.some((d) => d.laba_sesudah !== d.laba_sebelum) && (
        <div className="flex flex-wrap gap-2 text-sm">
          {k.dampak.map((d) => {
            const selisih = d.laba_sesudah - d.laba_sebelum
            return (
              <span key={d.periode} className="rounded-md border border-border" style={{ padding: '4px 10px' }}>
                Laba {labelBulan(d.periode)}{' '}
                <strong className="tabular" style={{ color: selisih < 0 ? 'var(--accent-red)' : 'var(--accent-green)' }}>
                  {selisih < 0 ? formatRp(selisih) : `+${formatRp(selisih)}`}
                </strong>
              </span>
            )
          })}
          <span className="text-xs text-muted-foreground" style={{ alignSelf: 'center' }}>bila disetujui; penyusutan aset dihitung terpisah</span>
        </div>
      )}
      {k.status !== 'menunggu' && (k.catatan || k.diputuskan_oleh) && (
        <p className="text-xs text-muted-foreground">
          {k.diputuskan_oleh && `Diputuskan ${k.diputuskan_oleh}`}{k.catatan && ` · “${k.catatan}”`}
          {k.status === 'kedaluwarsa' && ' · transaksinya sudah berubah sebelum usulan diputuskan'}
        </p>
      )}
      {aksi}
    </article>
  )
}
