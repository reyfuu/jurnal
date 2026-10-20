import { useEffect, useState } from 'react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { getCoA, perbaruiCoA, type Account, type CoAInfo } from '@/lib/api'

const KONDISI: Partial<Record<Account['kondisi'], string>> = {
  jika_ada: 'Opsional, bila dimiliki',
  jika_pt: 'Khusus PT',
  jika_non_pt: 'Khusus OP / CV',
  jika_perpetual: 'Metode perpetual',
  jika_periodik: 'Metode periodik',
}

/** bisaPerbarui: Owner boleh memindahkan usaha ke versi template COA terbaru. */
export function DaftarAkun({ bisaPerbarui }: { bisaPerbarui: boolean }) {
  const [c, setC] = useState<CoAInfo | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => { getCoA().then(setC, (e: Error) => setError(e.message)) }, [])
  async function perbarui() {
    setBusy(true)
    setError('')
    try {
      await perbaruiCoA()
      setC(await getCoA())
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }
  const p = c?.pembaruan

  return (
    <>
      <PageHeader title="Daftar Akun (CoA)" subtitle="Susunan akun otomatis sesuai jenis usaha, bentuk badan, dan metode persediaan di Profil Usaha" />
      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', whiteSpace: 'normal' }}>{error}</div>}
      {p && (
        <div className="card" style={{ padding: 20, borderColor: 'var(--accent-amber)' }} role="status">
          <div className="chart-title">Versi baru daftar akun tersedia (v{p.versi})</div>
          <p className="text-sm text-muted-foreground" style={{ marginTop: 4 }}>
            Tim MasterPajak memperbarui template. Usaha Anda masih memakai v{c?.template?.versi} sampai Anda menerapkan versi baru.
          </p>
          <ul className="text-sm" style={{ margin: '10px 0', display: 'grid', gap: 4 }}>
            {p.diff.tambah.map((a) => <li key={'t' + a.kode}><span style={{ color: 'var(--accent-green)' }}>+ Akun baru</span> {a.kode} {a.nama}</li>)}
            {p.diff.ubah.map((a) => <li key={'u' + a.kode}><span style={{ color: 'var(--accent-amber)' }}>~ Berubah</span> {a.kode} {a.nama}</li>)}
            {p.diff.hapus.map((a) => <li key={'h' + a.kode}><span style={{ color: 'var(--accent-red)' }}>− Dihapus</span> {a.kode} {a.nama}</li>)}
          </ul>
          {bisaPerbarui
            ? <Button className="justify-self-start" style={{ width: 'fit-content' }} disabled={busy} onClick={() => void perbarui()}>{busy ? 'Menerapkan…' : `Terapkan v${p.versi}`}</Button>
            : <p className="text-xs text-muted-foreground">Hanya Owner yang bisa menerapkan versi baru.</p>}
        </div>
      )}
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
