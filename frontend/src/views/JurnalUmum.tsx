import { Fragment, useEffect, useState } from 'react'
import { FileSpreadsheet } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { getJurnalUmum, jurnalUmumXlsxUrl, type FilterJurnalUmum, type JurnalUmum as DataJurnal, type View } from '@/lib/api'
import { formatRp, formatTanggal } from '@/lib/format'
import { ekspor } from '@/lib/garansi'
import { pilihanCepat, rentangBulan } from '@/lib/rentang'
import { useHP } from '@/lib/hp'
import { KepalaHP } from '@/components/hp/KepalaHP'
import { FilterRentangAkun } from '@/components/hp/FilterRentangAkun'

const PER_HALAMAN = 300

/**
 * Jurnal Umum: semua entri jurnal urut tanggal, per bukti (transaksi, pelunasan, penyusutan,
 * penyesuaian), baris debit lalu kredit per akun. Angkanya dari jurnal yang sama dengan Buku Besar
 * dan laporan keuangan, jadi totalnya selalu cocok.
 */
export function JurnalUmum({ period, bisaEkspor, navigate }: { period: string; bisaEkspor: boolean; navigate?: (v: View) => void }) {
  const [f, setF] = useState<FilterJurnalUmum>(() => rentangBulan(period))
  const [ketik, setKetik] = useState('')
  const [data, setData] = useState<DataJurnal | null>(null)
  const [error, setError] = useState('')
  const [batas, setBatas] = useState(PER_HALAMAN)

  useEffect(() => { setF((x) => ({ ...x, ...rentangBulan(period) })) }, [period])
  // Kata kunci dicari di server (tampilan = ekspor Excel); tunggu jeda mengetik.
  useEffect(() => {
    const t = setTimeout(() => setF((x) => (x.q ?? '') === ketik.trim() ? x : { ...x, q: ketik.trim() || undefined }), 300)
    return () => clearTimeout(t)
  }, [ketik])
  useEffect(() => {
    if (f.dari > f.sampai) { setError('Tanggal sampai tidak boleh sebelum tanggal dari.'); return }
    let hidup = true
    getJurnalUmum(f).then((d) => { if (hidup) { setData(d); setError(''); setBatas(PER_HALAMAN) } }, (e: Error) => { if (hidup) setError(e.message) })
    return () => { hidup = false }
  }, [f])
  const ubah = (patch: Partial<FilterJurnalUmum>) => setF((x) => ({ ...x, ...patch }))
  const hp = useHP()
  const [bukaFilter, setBukaFilter] = useState(false)

  if (!data) return error ? <p className="pill-status pill-red" role="alert">{error}</p> : <p className="workspace-subtitle">Memuat jurnal umum…</p>
  const tampil = data.entri.slice(0, batas)
  const lebihBanyak = data.entri.length > tampil.length && (
    <div style={{ marginTop: 10 }}>
      <Button variant="outline" size="sm" onClick={() => setBatas((b) => b + PER_HALAMAN)}>
        Tampilkan {Math.min(PER_HALAMAN, data.entri.length - tampil.length)} bukti lagi (dari {data.entri.length - tampil.length} tersisa)
      </Button>
    </div>
  )

  if (hp && data.supported) {
    // HP: satu kartu per bukti; filter di lembar bawah, cari lewat ikon di kepala halaman.
    const bulanIni = rentangBulan(period)
    const rentangLain = f.dari !== bulanIni.dari || f.sampai !== bulanIni.sampai
    const akun = data.pilihan.find((a) => a.kode === f.akun)
    return (
      <>
        <KepalaHP judul="Jurnal Umum" cari={ketik} onCari={setKetik} placeholderCari="Keterangan, no. bukti, akun"
          jumlahFilter={(rentangLain ? 1 : 0) + (f.akun ? 1 : 0)} onFilter={() => setBukaFilter(true)}
          aksi={bisaEkspor && <button type="button" className="kepala-hp-ikon" aria-label="Ekspor Excel" onClick={() => void ekspor(jurnalUmumXlsxUrl(f))}><FileSpreadsheet size={16} aria-hidden="true" /></button>}
          chips={[
            ...(rentangLain ? [{ label: data.period_label, onHapus: () => ubah(bulanIni) }] : []),
            ...(akun ? [{ label: `${akun.kode} ${akun.nama}`, onHapus: () => ubah({ akun: undefined }) }] : []),
          ]} />
        <div className="hp-ringkas tabular">
          <span>{data.period_label} · {data.entri.length} bukti</span>
          <span className={'pill-status ' + (data.seimbang ? 'pill-green' : 'pill-red')}>{data.seimbang ? 'Seimbang' : 'Debit ≠ Kredit'}</span>
          <span>Debit {formatRp(data.total_debit)} · Kredit {formatRp(data.total_kredit)}</span>
        </div>
        {error && <p className="pill-status pill-red" role="alert" style={{ marginBottom: 10 }}>{error}</p>}
        {tampil.length === 0 && <p className="text-sm text-muted-foreground">{f.akun || f.q ? 'Tidak ada bukti jurnal yang cocok.' : `Belum ada jurnal di ${data.period_label}.`}</p>}
        <div>{tampil.map((e) => (
          <div className="kartu-hp tabular" key={`${e.bukti}|${e.tx_id ?? ''}|${e.tanggal}|${e.keterangan}`}>
            <div className="kartu-hp-judul"><span>{e.bukti}</span><span className="text-xs text-muted-foreground" style={{ fontWeight: 400 }}>{formatTanggal(e.tanggal)}</span></div>
            <div className="kartu-hp-sub" style={{ marginBottom: 6 }}>{e.keterangan}</div>
            {e.baris.map((b, i) => (
              // akun kredit menjorok, sesuai kebiasaan penulisan jurnal
              <div key={i} className={'jurnal-hp-baris' + (b.kredit ? ' kredit' : '')}>
                <span>{b.kode} {b.nama}</span><span>{formatRp(b.debit || b.kredit)}</span>
              </div>
            ))}
          </div>
        ))}</div>
        {lebihBanyak}
        <FilterRentangAkun buka={bukaFilter} onTutup={() => setBukaFilter(false)} nilai={f} period={period} pilihan={data.pilihan}
          onTerapkan={(n) => ubah({ dari: n.dari, sampai: n.sampai, akun: n.akun })} />
      </>
    )
  }

  return (
    <>
      <PageHeader title="Jurnal Umum" subtitle={`${data.period_label} · ${data.entri.length} bukti jurnal, debit & kredit per akun`}>
        {data.supported && bisaEkspor && (
          <Button variant="outline" onClick={() => void ekspor(jurnalUmumXlsxUrl(f))}><FileSpreadsheet aria-hidden="true" /> Ekspor Excel</Button>
        )}
      </PageHeader>

      {!data.supported ? (
        <div className="card" style={{ padding: 22 }}>
          <div className="chart-title">Jurnal umum belum tersedia untuk jenis usaha ini</div>
          <p className="text-sm text-muted-foreground" style={{ marginTop: 6 }}>Jurnal memakai daftar akun (CoA) jenis usaha Anda; CoA untuk jenis usaha ini sedang disiapkan tim MasterPajak AI.</p>
        </div>
      ) : (
        <>
          <div className="card" style={{ padding: 16, marginBottom: 14 }}>
            <div className="flex flex-wrap items-end gap-3">
              <label className="grid gap-1"><span className="kpi-label">Dari tanggal</span>
                <input className="input-field" type="date" value={f.dari} max={f.sampai} onChange={(e) => e.target.value && ubah({ dari: e.target.value })} /></label>
              <label className="grid gap-1"><span className="kpi-label">Sampai tanggal</span>
                <input className="input-field" type="date" value={f.sampai} min={f.dari} onChange={(e) => e.target.value && ubah({ sampai: e.target.value })} /></label>
              <label className="grid gap-1" style={{ minWidth: 240 }}><span className="kpi-label">Akun</span>
                <select className="input-field" value={f.akun ?? ''} onChange={(e) => ubah({ akun: e.target.value || undefined })}>
                  <option value="">Semua akun</option>
                  {data.pilihan.map((a) => <option key={a.kode} value={a.kode}>{a.kode} · {a.nama}</option>)}
                </select></label>
              <label className="grid gap-1" style={{ minWidth: 220 }}><span className="kpi-label">Cari</span>
                <input className="input-field" placeholder="Keterangan, no. bukti, akun" value={ketik} onChange={(e) => setKetik(e.target.value)} /></label>
              <div className="flex flex-wrap gap-1.5">
                {pilihanCepat(period).map((p) => (
                  <button key={p.label} type="button" className="pill-status" style={{ cursor: 'pointer', outline: f.dari === p.dari && f.sampai === p.sampai ? '1px solid var(--text-brand)' : undefined }}
                    onClick={() => ubah({ dari: p.dari, sampai: p.sampai })}>{p.label}</button>
                ))}
              </div>
            </div>
            {error && <p className="pill-status pill-red" role="alert" style={{ marginTop: 10 }}>{error}</p>}
          </div>

          <div className="flex flex-wrap items-center gap-3" style={{ marginBottom: 12 }}>
            <span className={'pill-status ' + (data.seimbang ? 'pill-green' : 'pill-red')}>
              {data.seimbang ? 'Debit = Kredit (seimbang)' : 'Debit ≠ Kredit'}
            </span>
            <span className="text-sm text-muted-foreground tabular">Total debit {formatRp(data.total_debit)} · kredit {formatRp(data.total_kredit)}</span>
          </div>

          <div className="card" style={{ padding: 0 }}>
            <div className="table-scroll">
              <table className="data-table tabular" style={{ minWidth: 900 }}>
                <thead>
                  <tr>
                    <th style={{ width: 110 }}>Tanggal</th><th style={{ width: 130 }}>No. bukti</th><th>Keterangan / akun</th>
                    <th style={{ width: 70 }}>Kode</th><th style={{ textAlign: 'right', width: 150 }}>Debit</th><th style={{ textAlign: 'right', width: 150 }}>Kredit</th>
                  </tr>
                </thead>
                <tbody>
                  {tampil.length === 0 && (
                    <tr><td colSpan={6} className="text-muted-foreground" style={{ whiteSpace: 'normal' }}>
                      {f.akun || f.q ? 'Tidak ada bukti jurnal yang cocok.' : (
                        <div style={{ padding: '10px 0' }}>
                          <div className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>Belum ada jurnal di {data.period_label}.</div>
                          <div className="text-sm" style={{ marginTop: 4 }}>Jurnal terbentuk otomatis dari saldo awal dan transaksi yang dicatat.</div>
                          {navigate && <Button size="sm" style={{ marginTop: 10 }} onClick={() => navigate('transaksi')}>Catat transaksi</Button>}
                        </div>
                      )}
                    </td></tr>
                  )}
                  {tampil.map((e) => (
                    <Fragment key={`${e.bukti}|${e.tx_id ?? ''}|${e.tanggal}|${e.keterangan}`}>
                      <tr style={{ background: 'var(--bg-raised)' }}>
                        <td style={{ whiteSpace: 'nowrap' }}>{formatTanggal(e.tanggal)}</td>
                        <td style={{ fontWeight: 600, whiteSpace: 'nowrap' }}>{e.bukti}</td>
                        <td colSpan={4} style={{ whiteSpace: 'normal' }}>{e.keterangan}</td>
                      </tr>
                      {e.baris.map((b, i) => (
                        <tr key={i}>
                          <td /><td />
                          {/* Akun kredit menjorok, sesuai kebiasaan penulisan jurnal. */}
                          <td style={{ paddingLeft: b.kredit ? 40 : 16 }}>{b.nama}</td>
                          <td className="text-muted-foreground">{b.kode}</td>
                          <td style={{ textAlign: 'right' }}>{b.debit ? formatRp(b.debit) : ''}</td>
                          <td style={{ textAlign: 'right' }}>{b.kredit ? formatRp(b.kredit) : ''}</td>
                        </tr>
                      ))}
                    </Fragment>
                  ))}
                  <tr style={{ fontWeight: 700 }}>
                    <td /><td /><td>Total {data.entri.length > tampil.length ? `(${data.entri.length} bukti)` : ''}</td><td />
                    <td style={{ textAlign: 'right' }}>{formatRp(data.total_debit)}</td>
                    <td style={{ textAlign: 'right' }}>{formatRp(data.total_kredit)}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
          {lebihBanyak}
          <p className="text-xs text-muted-foreground" style={{ marginTop: 10 }}>
            No. bukti: SA = saldo awal pembukuan, JU = transaksi, BYR = pelunasan transaksi bulan sebelumnya, SIS = entri otomatis sistem (penyusutan, penyesuaian persediaan, tutup buku).
            Ubah jurnal lewat transaksinya di menu Transaksi, atau ajukan koreksi akun.
          </p>
        </>
      )}
    </>
  )
}
