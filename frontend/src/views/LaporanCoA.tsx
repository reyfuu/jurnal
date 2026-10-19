import { Download } from 'lucide-react'
import { useEffect, useState } from 'react'
import { AmountRow, PageHeader } from '@/components/PageHeader'
import { coaXlsxUrl, getCoAReport, type CoAReport, type ReportLine } from '@/lib/api'
import { formatRp } from '@/lib/format'

const RED = '#fb7185'
const GREEN = '#10b981'
const BLUE = '#38bdf8'

const label = (l: ReportLine) => (l.kode ? <><span className="tabular" style={{ color: 'var(--text-muted)', marginRight: 8 }}>{l.kode}</span>{l.nama}</> : l.nama)

function Lines({ lines, negate = false }: { lines: ReportLine[] | null | undefined; negate?: boolean }) {
  return <>{(lines ?? []).map((l) => <AmountRow key={l.kode + l.nama} label={label(l)} value={formatRp(negate ? -l.jumlah : l.jumlah)} />)}</>
}

/**
 * Laporan Keuangan per akun (CoA).
 *
 * BELUM TERPASANG DI MENU MANA PUN — dan itu disengaja, bukan kode mati yang terlupa.
 * Menu `laporan-keuangan` dipegang penuh oleh LaporanKeuangan (4 tab: Laba Rugi, Neraca,
 * Arus Kas, Aset & Penyusutan); sebelumnya komponen ini membungkusnya sebagai `fallback`,
 * sehingga tab Arus Kas & Aset tidak pernah terjangkau untuk usaha perdagangan.
 *
 * Backend-nya tetap hidup dan sudah disinkronkan: /api/v1/laporan/coa dan unduh Excel-nya
 * kini ikut menjurnal penyusutan dari register aset, jadi laba bersih & total asetnya sama
 * persis dengan 4 tab. Tinggal dipasang begitu tempatnya di navigasi diputuskan.
 */
export function LaporanKeuanganCoA({ period, periodLabel }: { period: string; periodLabel: string }) {
  const [r, setR] = useState<CoAReport | null>(null)
  const [showTB, setShowTB] = useState(false)

  useEffect(() => { getCoAReport(period).then(setR, () => setR({ supported: false, trial_balance: null })) }, [period])

  if (!r || !r.supported || !r.income_statement || !r.balance_sheet) return null
  const is = r.income_statement
  const bs = r.balance_sheet
  const tb = r.trial_balance ?? []
  const totD = tb.reduce((s, b) => s + b.debit, 0)
  const totK = tb.reduce((s, b) => s + b.kredit, 0)

  return (
    <>
      <PageHeader title="Laporan Keuangan" subtitle={`${periodLabel} · Laba Rugi & Neraca per akun (CoA perdagangan, metode ${is.metode})`}>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <span className={'pill-status ' + (bs.seimbang ? 'pill-green' : 'pill-amber')}>{bs.seimbang ? 'Aset = Liabilitas + Ekuitas' : 'Neraca tidak seimbang'}</span>
          <a className="btn-cta" href={coaXlsxUrl(period)} style={{ display: 'inline-flex', alignItems: 'center', gap: 6, padding: '8px 14px' }}>
            <Download size={15} aria-hidden /> Unduh Excel
          </a>
        </div>
      </PageHeader>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(340px, 1fr))', gap: 14 }}>
        <div className="card" style={{ padding: 20, justifyContent: 'flex-start' }}>
          <div className="chart-title">Laporan Laba Rugi</div>
          <div className="list-rows tabular" style={{ marginTop: 12 }}>
            <Lines lines={is.pendapatan} />
            <AmountRow label="Penjualan bersih" value={formatRp(is.penjualan_bersih)} color="#facc15" bold />
            {is.metode === 'periodik' && <AmountRow label={<><span className="tabular" style={{ color: 'var(--text-muted)', marginRight: 8 }}>1301</span>Persediaan awal</>} value={formatRp(is.persediaan_awal)} />}
            <Lines lines={is.hpp_lines} />
            {is.metode === 'periodik' && <AmountRow label={<><span className="tabular" style={{ color: 'var(--text-muted)', marginRight: 8 }}>1301</span>Persediaan akhir</>} value={formatRp(-is.persediaan_akhir)} />}
            <AmountRow label="Harga pokok penjualan" value={formatRp(-is.hpp)} color={RED} bold />
            <AmountRow label="Laba kotor" value={formatRp(is.laba_kotor)} bold />
            <Lines lines={is.beban_operasional} negate />
            <AmountRow label="Total beban operasional" value={formatRp(-is.total_beban_operasional)} color={RED} bold />
            <AmountRow label="Laba usaha" value={formatRp(is.laba_usaha)} bold />
            <Lines lines={is.pendapatan_lain} />
            <Lines lines={is.beban_lain} negate />
            <AmountRow label="Laba bersih" value={formatRp(is.laba_bersih)} color={GREEN} bold />
          </div>
          {(is.catatan ?? []).map((c) => <p key={c} className="kpi-meta-row" style={{ marginTop: 10 }}>{c}</p>)}
        </div>

        <div className="card" style={{ padding: 20, justifyContent: 'flex-start' }}>
          <div className="chart-title">Neraca (Posisi Keuangan)</div>
          <div className="list-rows tabular" style={{ marginTop: 12 }}>
            {bs.aset.map((s) => (
              <div key={s.kelompok}>
                <AmountRow label={s.kelompok.toUpperCase()} value={formatRp(s.total)} color={BLUE} bold />
                <Lines lines={s.lines} />
              </div>
            ))}
            <AmountRow label="TOTAL ASET" value={formatRp(bs.total_aset)} color={BLUE} bold style={{ border: '1px solid rgba(56,189,248,0.3)' }} />
            <div style={{ height: 8 }} />
            {[...bs.liabilitas, bs.ekuitas].map((s) => (
              <div key={s.kelompok}>
                <AmountRow label={s.kelompok.toUpperCase()} value={formatRp(s.total)} color={s.kelompok === 'Ekuitas' ? GREEN : RED} bold />
                <Lines lines={s.lines} />
              </div>
            ))}
            <AmountRow label="TOTAL LIABILITAS + EKUITAS" value={formatRp(bs.total_liabilitas + bs.total_ekuitas)} color={GREEN} bold style={{ border: '1px solid rgba(16,185,129,0.3)' }} />
          </div>
        </div>
      </div>

      <div className="card">
        <button type="button" className="chart-title" style={{ textAlign: 'left', cursor: 'pointer' }} onClick={() => setShowTB((v) => !v)} aria-expanded={showTB}>
          Neraca Saldo ({tb.length} akun) {showTB ? '- sembunyikan' : '- tampilkan'}
        </button>
        {showTB && (
          <table className="data-table tabular" style={{ marginTop: 8 }}>
            <thead><tr><th>Kode</th><th>Akun</th><th style={{ textAlign: 'right' }}>Debit</th><th style={{ textAlign: 'right' }}>Kredit</th></tr></thead>
            <tbody>
              {tb.map((b) => (
                <tr key={b.kode}>
                  <td>{b.kode}</td><td>{b.nama}</td>
                  <td style={{ textAlign: 'right' }}>{b.debit ? formatRp(b.debit) : ''}</td>
                  <td style={{ textAlign: 'right' }}>{b.kredit ? formatRp(b.kredit) : ''}</td>
                </tr>
              ))}
              <tr style={{ fontWeight: 700 }}>
                <td></td><td>Total</td><td style={{ textAlign: 'right' }}>{formatRp(totD)}</td><td style={{ textAlign: 'right' }}>{formatRp(totK)}</td>
              </tr>
            </tbody>
          </table>
        )}
      </div>
    </>
  )
}
