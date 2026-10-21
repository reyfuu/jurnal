import { Fragment, useEffect, useMemo, useState } from 'react'
import { ChevronDown, ChevronRight, FileSpreadsheet } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { bukuBesarXlsxUrl, getBukuBesar, type AkunBukuBesar, type BukuBesar as DataBukuBesar } from '@/lib/api'
import { formatRp, formatTanggal } from '@/lib/format'
import { ekspor } from '@/lib/garansi'

const RED = 'var(--text-danger)'

/**
 * Buku Besar: gambaran besar semua akun (saldo awal, debit, kredit, saldo akhir per kelompok),
 * dan rincian tiap akun dengan saldo berjalan saat barisnya dibuka. Saldo mengikuti saldo normal
 * akun, jadi akun kredit (utang, modal, pendapatan) tampil positif seperti di neraca.
 */
export function BukuBesar({ period, bisaEkspor }: { period: string; bisaEkspor: boolean }) {
  const [data, setData] = useState<DataBukuBesar | null>(null)
  const [error, setError] = useState('')
  const [cari, setCari] = useState('')
  const [semua, setSemua] = useState(false)
  const [buka, setBuka] = useState<Set<string>>(new Set())

  useEffect(() => {
    let hidup = true
    getBukuBesar(period).then((d) => { if (hidup) { setData(d); setError('') } }, (e: Error) => { if (hidup) setError(e.message) })
    return () => { hidup = false }
  }, [period])

  const tampil = useMemo(() => {
    const q = cari.trim().toLowerCase()
    return (data?.akun ?? []).filter((a) =>
      (semua || a.saldo_awal !== 0 || a.debit !== 0 || a.kredit !== 0 || a.saldo_akhir !== 0) &&
      (!q || a.kode.includes(q) || a.nama.toLowerCase().includes(q)))
  }, [data, cari, semua])

  const kelompok = useMemo(() => {
    const out: { nama: string; akun: AkunBukuBesar[] }[] = []
    for (const a of tampil) {
      const g = out.find((x) => x.nama === a.kelompok)
      if (g) g.akun.push(a)
      else out.push({ nama: a.kelompok, akun: [a] })
    }
    return out
  }, [tampil])

  const toggle = (kode: string) => setBuka((s) => {
    const n = new Set(s)
    if (n.has(kode)) n.delete(kode)
    else n.add(kode)
    return n
  })

  if (error) return <p className="pill-status pill-red" role="alert">{error}</p>
  if (!data) return <p className="workspace-subtitle">Memuat buku besar…</p>

  return (
    <>
      <PageHeader title="Buku Besar" subtitle={`${data.period_label} · saldo awal, mutasi, dan saldo akhir setiap akun`}>
        {data.supported && bisaEkspor && (
          <Button variant="outline" onClick={() => void ekspor(bukuBesarXlsxUrl(period))}><FileSpreadsheet aria-hidden="true" /> Ekspor Excel</Button>
        )}
      </PageHeader>

      {!data.supported ? (
        <div className="card" style={{ padding: 22 }}>
          <div className="chart-title">Buku besar belum tersedia untuk jenis usaha ini</div>
          <p className="text-sm text-muted-foreground" style={{ marginTop: 6 }}>Buku besar memakai daftar akun (CoA) jenis usaha Anda; CoA untuk jenis usaha ini sedang disiapkan tim Master Pajak.</p>
        </div>
      ) : (
        <>
          <div className="flex flex-wrap items-center gap-3" style={{ marginBottom: 12 }}>
            <input className="input-field" style={{ maxWidth: 280 }} placeholder="Cari kode atau nama akun" aria-label="Cari akun" value={cari} onChange={(e) => setCari(e.target.value)} />
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={semua} onChange={(e) => setSemua(e.target.checked)} /> Tampilkan akun tanpa saldo
            </label>
            <span className={'pill-status ' + (data.seimbang ? 'pill-green' : 'pill-red')}>
              {data.seimbang ? 'Debit = Kredit (seimbang)' : 'Debit ≠ Kredit'}
            </span>
            <button type="button" className="text-sm underline underline-offset-4" onClick={() => setBuka(buka.size ? new Set() : new Set(tampil.filter((a) => a.baris.length).map((a) => a.kode)))}>
              {buka.size ? 'Tutup semua rincian' : 'Buka semua rincian'}
            </button>
          </div>

          <div className="card" style={{ padding: 0 }}>
            <div className="table-scroll">
              <table className="data-table tabular" style={{ minWidth: 860 }}>
                <thead>
                  <tr>
                    <th style={{ width: 90 }}>Kode</th><th>Akun</th>
                    <th style={{ textAlign: 'right' }}>Saldo awal</th><th style={{ textAlign: 'right' }}>Debit</th>
                    <th style={{ textAlign: 'right' }}>Kredit</th><th style={{ textAlign: 'right' }}>Saldo akhir</th>
                  </tr>
                </thead>
                <tbody>
                  {kelompok.length === 0 && <tr><td colSpan={6} className="text-muted-foreground">Tidak ada akun yang cocok.</td></tr>}
                  {kelompok.map((g) => (
                    <Fragment key={g.nama}>
                      <tr><td colSpan={6} className="section-head" style={{ background: 'var(--bg-raised)' }}>{g.nama}</td></tr>
                      {g.akun.map((a) => {
                        const terbuka = buka.has(a.kode)
                        const bisaBuka = a.baris.length > 0
                        return (
                          <Fragment key={a.kode}>
                            <tr style={{ cursor: bisaBuka ? 'pointer' : 'default' }} onClick={() => bisaBuka && toggle(a.kode)}>
                              <td>
                                {bisaBuka
                                  ? <button type="button" aria-expanded={terbuka} aria-label={`${terbuka ? 'Tutup' : 'Buka'} rincian ${a.nama}`} className="inline-flex items-center gap-1" onClick={(e) => { e.stopPropagation(); toggle(a.kode) }}>
                                      {terbuka ? <ChevronDown size={14} aria-hidden="true" /> : <ChevronRight size={14} aria-hidden="true" />}{a.kode}
                                    </button>
                                  : <span style={{ paddingLeft: 18 }}>{a.kode}</span>}
                              </td>
                              <td>{a.nama}{bisaBuka && <span className="text-xs text-muted-foreground"> · {a.baris.length} mutasi</span>}</td>
                              <td style={{ textAlign: 'right' }}>{formatRp(a.saldo_awal)}</td>
                              <td style={{ textAlign: 'right' }}>{a.debit ? formatRp(a.debit) : '–'}</td>
                              <td style={{ textAlign: 'right' }}>{a.kredit ? formatRp(a.kredit) : '–'}</td>
                              <td style={{ textAlign: 'right', fontWeight: 600, color: a.saldo_akhir < 0 ? RED : undefined }}>{formatRp(a.saldo_akhir)}</td>
                            </tr>
                            {terbuka && (
                              <tr>
                                <td colSpan={6} style={{ padding: '4px 12px 14px 36px', background: 'var(--bg-raised)' }}>
                                  <table className="data-table tabular" style={{ marginTop: 0 }}>
                                    <thead><tr><th style={{ width: 110 }}>Tanggal</th><th>Keterangan</th><th style={{ textAlign: 'right' }}>Debit</th><th style={{ textAlign: 'right' }}>Kredit</th><th style={{ textAlign: 'right' }}>Saldo</th></tr></thead>
                                    <tbody>
                                      <tr><td /><td className="text-muted-foreground">Saldo awal</td><td /><td /><td style={{ textAlign: 'right' }}>{formatRp(a.saldo_awal)}</td></tr>
                                      {a.baris.map((b, i) => (
                                        <tr key={i}>
                                          <td>{formatTanggal(b.tanggal)}</td>
                                          <td>{b.keterangan}{b.tx_id ? <span className="text-xs text-muted-foreground"> · #{b.tx_id}</span> : null}</td>
                                          <td style={{ textAlign: 'right' }}>{b.debit ? formatRp(b.debit) : ''}</td>
                                          <td style={{ textAlign: 'right' }}>{b.kredit ? formatRp(b.kredit) : ''}</td>
                                          <td style={{ textAlign: 'right' }}>{formatRp(b.saldo)}</td>
                                        </tr>
                                      ))}
                                      <tr style={{ fontWeight: 600 }}><td /><td>Saldo akhir</td><td style={{ textAlign: 'right' }}>{formatRp(a.debit)}</td><td style={{ textAlign: 'right' }}>{formatRp(a.kredit)}</td><td style={{ textAlign: 'right' }}>{formatRp(a.saldo_akhir)}</td></tr>
                                    </tbody>
                                  </table>
                                </td>
                              </tr>
                            )}
                          </Fragment>
                        )
                      })}
                    </Fragment>
                  ))}
                  <tr style={{ fontWeight: 700 }}>
                    <td /><td>Total mutasi bulan ini</td><td />
                    <td style={{ textAlign: 'right' }}>{formatRp(data.total_debit)}</td>
                    <td style={{ textAlign: 'right' }}>{formatRp(data.total_kredit)}</td><td />
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
          <p className="text-xs text-muted-foreground" style={{ marginTop: 10 }}>
            Saldo mengikuti saldo normal akun: akun harta & beban bertambah di debit, akun utang, modal & pendapatan bertambah di kredit.
            Saldo awal = saldo akhir bulan sebelumnya (dari menu Saldo Awal untuk bulan pertama). Angkanya sama dengan Neraca &amp; Laba Rugi.
          </p>
        </>
      )}
    </>
  )
}
