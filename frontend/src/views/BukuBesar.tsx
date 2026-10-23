import { Fragment, useEffect, useMemo, useState } from 'react'
import { ChevronDown, ChevronRight, FileSpreadsheet } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { bukuBesarXlsxUrl, getBukuBesar, type AkunBukuBesar, type BukuBesar as DataBukuBesar, type FilterBukuBesar, type View } from '@/lib/api'
import { formatRp, formatTanggal } from '@/lib/format'
import { ekspor } from '@/lib/garansi'
import { pilihanCepat, rentangBulan } from '@/lib/rentang'

const RED = 'var(--text-danger)'

/**
 * Buku Besar: gambaran besar semua akun (saldo awal, debit, kredit, saldo akhir per kelompok),
 * dan rincian tiap akun dengan saldo berjalan saat barisnya dibuka. Saldo mengikuti saldo normal
 * akun, jadi akun kredit (utang, modal, pendapatan) tampil positif seperti di neraca.
 */
export function BukuBesar({ period, bisaEkspor, navigate }: { period: string; bisaEkspor: boolean; navigate?: (v: View) => void }) {
  const [f, setF] = useState<FilterBukuBesar>(() => rentangBulan(period))
  const [data, setData] = useState<DataBukuBesar | null>(null)
  const [error, setError] = useState('')
  const [cari, setCari] = useState('')
  const [semua, setSemua] = useState(false)
  const [buka, setBuka] = useState<Set<string>>(new Set())

  // Ganti periode laporan di atas = rentang ikut ke bulan itu (akun yang dipilih tetap).
  useEffect(() => { setF((x) => ({ ...x, ...rentangBulan(period) })) }, [period])
  useEffect(() => {
    if (f.dari > f.sampai) { setError('Tanggal sampai tidak boleh sebelum tanggal dari.'); return }
    let hidup = true
    getBukuBesar(f).then((d) => {
      if (!hidup) return
      setData(d)
      setError('')
      if (f.akun) setBuka(new Set([f.akun])) // riwayat satu akun: langsung tampil rinciannya
    }, (e: Error) => { if (hidup) setError(e.message) })
    return () => { hidup = false }
  }, [f])
  const ubah = (patch: Partial<FilterBukuBesar>) => setF((x) => ({ ...x, ...patch }))

  const tampil = useMemo(() => {
    const q = cari.trim().toLowerCase()
    return (data?.akun ?? []).filter((a) =>
      (semua || a.saldo_awal !== 0 || a.debit !== 0 || a.kredit !== 0 || a.saldo_akhir !== 0) &&
      (!q || a.kode.includes(q) || a.nama.toLowerCase().includes(q)))
  }, [data, cari, semua])

  // Rentang tanpa saldo & mutasi sama sekali (usaha baru): jelaskan kenapa kosong.
  const kosong = !!data && data.akun.every((a) => !a.saldo_awal && !a.debit && !a.kredit && !a.saldo_akhir)

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

  if (!data) return error ? <p className="pill-status pill-red" role="alert">{error}</p> : <p className="workspace-subtitle">Memuat buku besar…</p>

  return (
    <>
      <PageHeader title="Buku Besar" subtitle={`${data.period_label} · saldo awal, mutasi, dan saldo akhir ${f.akun ? 'akun terpilih' : 'setiap akun'}`}>
        {data.supported && bisaEkspor && (
          <Button variant="outline" onClick={() => void ekspor(bukuBesarXlsxUrl(f))}><FileSpreadsheet aria-hidden="true" /> Ekspor Excel</Button>
        )}
      </PageHeader>

      {data.supported && (
        <div className="card" style={{ padding: 16, marginBottom: 14 }}>
          <div className="flex flex-wrap items-end gap-3">
            <label className="grid gap-1"><span className="kpi-label">Dari tanggal</span>
              <input className="input-field" type="date" value={f.dari} max={f.sampai} onChange={(e) => e.target.value && ubah({ dari: e.target.value })} /></label>
            <label className="grid gap-1"><span className="kpi-label">Sampai tanggal</span>
              <input className="input-field" type="date" value={f.sampai} min={f.dari} onChange={(e) => e.target.value && ubah({ sampai: e.target.value })} /></label>
            <label className="grid gap-1" style={{ minWidth: 260 }}><span className="kpi-label">Akun</span>
              <select className="input-field" value={f.akun ?? ''} onChange={(e) => ubah({ akun: e.target.value || undefined })}>
                <option value="">Semua akun (gambaran besar)</option>
                {data.pilihan.map((a) => <option key={a.kode} value={a.kode}>{a.kode} · {a.nama}</option>)}
              </select></label>
            <div className="flex flex-wrap gap-1.5">
              {pilihanCepat(period).map((p) => (
                <button key={p.label} type="button" className="pill-status" style={{ cursor: 'pointer', outline: f.dari === p.dari && f.sampai === p.sampai ? '1px solid var(--text-brand)' : undefined }}
                  onClick={() => ubah({ dari: p.dari, sampai: p.sampai })}>{p.label}</button>
              ))}
            </div>
          </div>
          {error && <p className="pill-status pill-red" role="alert" style={{ marginTop: 10 }}>{error}</p>}
          <p className="text-xs text-muted-foreground" style={{ marginTop: 8 }}>
            Rentang lintas bulan: saldo berjalan menyambung, akun pendapatan & beban menjumlah seluruh rentang, dan laba bulan-bulan sebelum tanggal dari sudah masuk Laba Ditahan.
          </p>
        </div>
      )}

      {!data.supported ? (
        <div className="card" style={{ padding: 22 }}>
          <div className="chart-title">Buku besar belum tersedia untuk jenis usaha ini</div>
          <p className="text-sm text-muted-foreground" style={{ marginTop: 6 }}>Buku besar memakai daftar akun (CoA) jenis usaha Anda; CoA untuk jenis usaha ini sedang disiapkan tim MasterPajak AI.</p>
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
                  {kelompok.length === 0 && (
                    <tr><td colSpan={6} className="text-muted-foreground" style={{ whiteSpace: 'normal' }}>
                      {kosong && !cari.trim() ? (
                        <div style={{ padding: '10px 0' }}>
                          <div className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>Belum ada saldo awal maupun transaksi di {data.period_label}.</div>
                          <div className="text-sm" style={{ marginTop: 4 }}>
                            Buku Besar terisi dari saldo awal dan transaksi. Isi saldo awal dulu, lalu catat transaksi (atau impor mutasi rekening).
                          </div>
                          <div className="flex flex-wrap gap-2" style={{ marginTop: 10 }}>
                            {navigate && <Button size="sm" onClick={() => navigate('saldo-awal')}>Isi saldo awal</Button>}
                            {navigate && <Button size="sm" variant="outline" onClick={() => navigate('transaksi')}>Catat transaksi</Button>}
                            <Button size="sm" variant="outline" onClick={() => setSemua(true)}>Lihat semua akun (CoA)</Button>
                          </div>
                        </div>
                      ) : 'Tidak ada akun yang cocok.'}
                    </td></tr>
                  )}
                  {kelompok.map((g) => (
                    <Fragment key={g.nama}>
                      <tr><td colSpan={6} className="section-head" style={{ background: 'var(--bg-raised)' }}>{g.nama}</td></tr>
                      {g.akun.map((a) => {
                        const terbuka = buka.has(a.kode)
                        const bisaBuka = a.baris.length > 0
                        // Saldo berlawanan dengan saldo normal = tanda salah catat (kecuali laba/rugi).
                        const salah = (n: number) => n < 0 && !a.boleh_minus
                        const bermasalah = salah(a.saldo_awal) || salah(a.saldo_akhir) || a.baris.some((b) => salah(b.saldo))
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
                              <td>
                                {a.nama}{bisaBuka && <span className="text-xs text-muted-foreground"> · {a.baris.length} mutasi</span>}
                                {bermasalah && <span className="pill-status pill-red" style={{ marginLeft: 8 }}>saldo minus</span>}
                              </td>
                              <td style={{ textAlign: 'right', color: salah(a.saldo_awal) ? RED : undefined }}>{formatRp(a.saldo_awal)}</td>
                              <td style={{ textAlign: 'right' }}>{a.debit ? formatRp(a.debit) : '–'}</td>
                              <td style={{ textAlign: 'right' }}>{a.kredit ? formatRp(a.kredit) : '–'}</td>
                              <td style={{ textAlign: 'right', fontWeight: 600, color: salah(a.saldo_akhir) ? RED : undefined }}>{formatRp(a.saldo_akhir)}</td>
                            </tr>
                            {terbuka && (
                              <tr>
                                <td colSpan={6} style={{ padding: '4px 12px 14px 36px', background: 'var(--bg-raised)' }}>
                                  <table className="data-table tabular" style={{ marginTop: 0 }}>
                                    <thead><tr><th style={{ width: 110 }}>Tanggal</th><th style={{ width: 140 }}>No. Bukti</th><th>Keterangan</th><th style={{ textAlign: 'right' }}>Debit</th><th style={{ textAlign: 'right' }}>Kredit</th><th style={{ textAlign: 'right' }}>Saldo</th></tr></thead>
                                    <tbody>
                                      <tr><td /><td /><td className="text-muted-foreground">Saldo awal</td><td /><td /><td style={{ textAlign: 'right', color: salah(a.saldo_awal) ? RED : undefined }}>{formatRp(a.saldo_awal)}</td></tr>
                                      {a.baris.map((b, i) => {
                                        // Baris tempat saldo berbalik minus: titik awal mencari jurnal yang salah.
                                        const mulaiMinus = salah(b.saldo) && !salah(i ? a.baris[i - 1].saldo : a.saldo_awal)
                                        return (
                                          <tr key={i} style={mulaiMinus ? { background: 'var(--accent-red-soft)' } : undefined}>
                                            <td>{formatTanggal(b.tanggal)}</td>
                                            <td style={{ fontWeight: 600, whiteSpace: 'nowrap' }}>{b.bukti}</td>
                                            <td>
                                              {b.keterangan}
                                              {mulaiMinus && <span className="pill-status pill-red" style={{ marginLeft: 8 }}>mulai minus</span>}
                                            </td>
                                            <td style={{ textAlign: 'right' }}>{b.debit ? formatRp(b.debit) : ''}</td>
                                            <td style={{ textAlign: 'right' }}>{b.kredit ? formatRp(b.kredit) : ''}</td>
                                            <td style={{ textAlign: 'right', color: salah(b.saldo) ? RED : undefined }}>{formatRp(b.saldo)}</td>
                                          </tr>
                                        )
                                      })}
                                      <tr style={{ fontWeight: 600 }}><td /><td /><td>Saldo akhir</td><td style={{ textAlign: 'right' }}>{formatRp(a.debit)}</td><td style={{ textAlign: 'right' }}>{formatRp(a.kredit)}</td><td style={{ textAlign: 'right' }}>{formatRp(a.saldo_akhir)}</td></tr>
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
                    <td /><td>Total mutasi {f.akun ? 'akun ini' : 'dalam rentang'}</td><td />
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
