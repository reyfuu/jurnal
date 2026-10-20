import { useEffect, useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { getCoA, type Account, type CoAInfo } from '@/lib/api'

const KONDISI: Partial<Record<Account['kondisi'], string>> = {
  jika_ada: 'Opsional, bila dimiliki',
  jika_pt: 'Khusus PT',
  jika_non_pt: 'Khusus OP / CV',
  jika_perpetual: 'Metode perpetual',
  jika_periodik: 'Metode periodik',
}

export function DaftarAkun() {
  const [c, setC] = useState<CoAInfo | null>(null)
  const [error, setError] = useState('')

  useEffect(() => { getCoA().then(setC, (e: Error) => setError(e.message)) }, [])

  return (
    <>
      <PageHeader title="Daftar Akun (CoA)" subtitle="Susunan akun otomatis sesuai jenis usaha, bentuk badan, dan metode persediaan di Profil Usaha" />
      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start' }}>{error}</div>}
      {c && !c.supported && (
        <div className="card" style={{ padding: 22 }}>
          <div className="chart-title">CoA belum tersedia untuk jenis usaha ini</div>
          <p className="text-sm text-muted-foreground" style={{ marginTop: 6 }}>
            CoA untuk usaha jasa dan manufaktur sedang disiapkan tim Master Pajak. Sementara itu laporan memakai format ringkas.
            Usaha perdagangan sudah memakai CoA lengkap.
          </p>
        </div>
      )}
      {c?.supported && c.accounts && (
        <>
          <div className="card">
            <div className="chart-title">Pemetaan otomatis jenis transaksi ke akun</div>
            <p className="kpi-meta-row" style={{ marginTop: 4 }}>Setiap transaksi otomatis masuk ke akun ini. Akun bisa dikoreksi per transaksi di form Transaksi.</p>
            <table className="data-table tabular">
              <thead><tr><th>Jenis transaksi</th><th>Kode</th><th>Akun</th></tr></thead>
              <tbody>
                {c.mapping?.map((m) => (
                  <tr key={m.jenis}><td>{m.jenis}</td><td>{m.kode}</td><td>{m.nama}</td></tr>
                ))}
              </tbody>
            </table>
          </div>

          {c.kelompok.map((k) => {
            const rows = c.accounts!.filter((a) => a.kelompok === k)
            if (!rows.length) return null
            return (
              <div className="card" key={k}>
                <div className="chart-title">{k}</div>
                <table className="data-table tabular">
                  <thead><tr><th style={{ width: 80 }}>Kode</th><th>Nama akun</th><th style={{ width: 110 }}>Saldo normal</th><th>Keterangan</th></tr></thead>
                  <tbody>
                    {rows.map((a) => (
                      <tr key={a.kode}>
                        <td>{a.kode}</td>
                        <td>
                          {a.nama}
                          {KONDISI[a.kondisi] && <span className={'pill-status ' + (a.opsional ? 'pill-status-muted' : 'pill-amber')} style={{ marginLeft: 8 }}>{KONDISI[a.kondisi]}</span>}
                        </td>
                        <td style={{ textTransform: 'capitalize' }}>{a.normal}</td>
                        <td style={{ color: 'var(--text-muted)', whiteSpace: 'normal' }}>{a.catatan}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )
          })}
          <p className="text-xs text-muted-foreground">Sumber: Daftar COA hasil review akuntan Master Pajak (perdagangan).</p>
        </>
      )}
    </>
  )
}
