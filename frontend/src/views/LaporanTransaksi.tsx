import { useEffect, useMemo, useState } from 'react'
import { FileSpreadsheet } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { getLaporanTransaksi, laporanTransaksiXlsxUrl, type FilterTransaksi, type LaporanTransaksi as Data } from '@/lib/api'
import { formatRp, formatTanggal } from '@/lib/format'
import { ekspor } from '@/lib/garansi'
import { pilihanCepat, rentangBulan } from '@/lib/rentang'
import { useHP } from '@/lib/hp'
import { KepalaHP } from '@/components/hp/KepalaHP'
import { LembarBawah } from '@/components/hp/LembarBawah'

const GREEN = 'var(--accent-green)'
const RED = 'var(--text-danger)'
const HALAMAN = 200

/**
 * Laporan Transaksi: transaksi dalam rentang tanggal bebas dengan filter jenis, arah, status
 * bayar, channel & kata kunci; total dan rekap per jenis. Disaring di server, jadi ekspor Excel
 * memuat baris yang persis sama dengan yang tampil.
 */
export function LaporanTransaksi({ period, bisaEkspor }: { period: string; bisaEkspor: boolean }) {
  const [f, setF] = useState<FilterTransaksi>(() => rentangBulan(period))
  const [cari, setCari] = useState('')
  const [data, setData] = useState<Data | null>(null)
  const [error, setError] = useState('')
  const [batas, setBatas] = useState(HALAMAN)
  const hp = useHP()
  const [bukaFilter, setBukaFilter] = useState(false)

  // Ganti periode laporan di atas = rentang ikut ke bulan itu.
  useEffect(() => { setF((x) => ({ ...x, ...rentangBulan(period) })) }, [period])
  // Kata kunci diterapkan setelah berhenti mengetik sebentar.
  useEffect(() => {
    const t = setTimeout(() => setF((x) => (x.cari === cari ? x : { ...x, cari })), 350)
    return () => clearTimeout(t)
  }, [cari])
  useEffect(() => {
    if (f.dari > f.sampai) { setError('Tanggal sampai tidak boleh sebelum tanggal dari.'); return }
    let hidup = true
    getLaporanTransaksi(f).then((d) => { if (hidup) { setData(d); setError(''); setBatas(HALAMAN) } }, (e: Error) => { if (hidup) setError(e.message) })
    return () => { hidup = false }
  }, [f])

  const ubah = (patch: Partial<FilterTransaksi>) => setF((x) => ({ ...x, ...patch }))
  const arahJenis = useMemo(() => new Map((data?.jenis_pilihan ?? []).map((j) => [j.nama, j.arah])), [data])
  const selisih = (data?.total_masuk ?? 0) - (data?.total_keluar ?? 0)

  const bisaUnduh = bisaEkspor && !!data && data.jumlah > 0
  const filterIsi = (
    <>
        <div className="flex flex-wrap items-end gap-3">
          <label className="grid gap-1"><span className="kpi-label">Dari tanggal</span>
            <input className="input-field" type="date" value={f.dari} max={f.sampai} onChange={(e) => e.target.value && ubah({ dari: e.target.value })} /></label>
          <label className="grid gap-1"><span className="kpi-label">Sampai tanggal</span>
            <input className="input-field" type="date" value={f.sampai} min={f.dari} onChange={(e) => e.target.value && ubah({ sampai: e.target.value })} /></label>
          <div className="flex flex-wrap gap-1.5">
            {pilihanCepat(period).map((p) => (
              <button key={p.label} type="button" className="pill-status" style={{ cursor: 'pointer', outline: f.dari === p.dari && f.sampai === p.sampai ? '1px solid var(--text-brand)' : undefined }}
                onClick={() => ubah({ dari: p.dari, sampai: p.sampai })}>{p.label}</button>
            ))}
          </div>
        </div>
        <div className="flex flex-wrap items-end gap-3" style={{ marginTop: 12 }}>
          <label className="grid gap-1"><span className="kpi-label">Jenis</span>
            <select className="input-field" value={f.jenis ?? ''} onChange={(e) => ubah({ jenis: e.target.value })}>
              <option value="">Semua jenis</option>
              {(data?.jenis_pilihan ?? []).map((j) => <option key={j.nama} value={j.nama}>{j.nama}</option>)}
            </select></label>
          <label className="grid gap-1"><span className="kpi-label">Arah uang</span>
            <select className="input-field" value={f.arah ?? ''} onChange={(e) => ubah({ arah: e.target.value })}>
              <option value="">Masuk & keluar</option><option value="masuk">Uang masuk</option><option value="keluar">Uang keluar</option>
            </select></label>
          <label className="grid gap-1"><span className="kpi-label">Status bayar</span>
            <select className="input-field" value={f.status ?? ''} onChange={(e) => ubah({ status: e.target.value })}>
              <option value="">Semua</option><option value="lunas">Lunas</option><option value="belum">Belum lunas</option>
            </select></label>
          {!!data?.channel?.length && (
            <label className="grid gap-1"><span className="kpi-label">Channel</span>
              <select className="input-field" value={f.channel ?? ''} onChange={(e) => ubah({ channel: e.target.value })}>
                <option value="">Semua channel</option>
                {data.channel.map((c) => <option key={c} value={c}>{c}</option>)}
              </select></label>
          )}
          {!hp && (
            <label className="grid gap-1" style={{ flex: '1 1 200px' }}><span className="kpi-label">Cari</span>
              <input className="input-field" placeholder="keterangan, barang, pelanggan / supplier" value={cari} onChange={(e) => setCari(e.target.value)} /></label>
          )}
        </div>
    </>
  )
  const bulanIni = rentangBulan(period)
  // HP: filter aktif sebagai label yang bisa dihapus.
  const chipFilter = [
    ...(f.dari !== bulanIni.dari || f.sampai !== bulanIni.sampai ? [{ label: `${formatTanggal(f.dari)} – ${formatTanggal(f.sampai)}`, onHapus: () => ubah(bulanIni) }] : []),
    ...(f.jenis ? [{ label: f.jenis, onHapus: () => ubah({ jenis: '' }) }] : []),
    ...(f.arah ? [{ label: f.arah === 'masuk' ? 'Uang masuk' : 'Uang keluar', onHapus: () => ubah({ arah: '' }) }] : []),
    ...(f.status ? [{ label: f.status === 'lunas' ? 'Lunas' : 'Belum lunas', onHapus: () => ubah({ status: '' }) }] : []),
    ...(f.channel ? [{ label: f.channel, onHapus: () => ubah({ channel: '' }) }] : []),
  ]

  return (
    <>
      {hp ? (
        <>
          <KepalaHP judul="Laporan Transaksi" cari={cari} onCari={setCari} placeholderCari="Keterangan, barang, pelanggan / supplier"
            jumlahFilter={chipFilter.length} onFilter={() => setBukaFilter(true)} chips={chipFilter}
            aksi={bisaUnduh && <button type="button" className="kepala-hp-ikon" aria-label="Ekspor Excel" onClick={() => void ekspor(laporanTransaksiXlsxUrl(f))}><FileSpreadsheet size={16} aria-hidden="true" /></button>} />
          <LembarBawah buka={bukaFilter} judul="Filter" onTutup={() => setBukaFilter(false)}>
            <div className="grid gap-2" style={{ paddingTop: 4 }}>
              {filterIsi}
              <Button style={{ marginTop: 6 }} onClick={() => setBukaFilter(false)}>Selesai</Button>
            </div>
          </LembarBawah>
        </>
      ) : (
        <>
          <PageHeader title="Laporan Transaksi" subtitle="Semua transaksi dalam rentang tanggal yang Anda pilih, lengkap dengan filter dan rekap per jenis">
            {bisaUnduh && (
              <Button variant="outline" onClick={() => void ekspor(laporanTransaksiXlsxUrl(f))}><FileSpreadsheet aria-hidden="true" /> Ekspor Excel</Button>
            )}
          </PageHeader>
          <div className="card" style={{ padding: 16, marginBottom: 14 }}>{filterIsi}</div>
        </>
      )}

      {error && <p className="pill-status pill-red" role="alert" style={{ whiteSpace: 'normal' }}>{error}</p>}
      {!data ? <p className="workspace-subtitle">Memuat transaksi…</p> : (
        <>
          <div className="grid gap-3 sm:grid-cols-4" style={{ marginBottom: 14 }}>
            {[
              ['Jumlah transaksi', data.jumlah.toLocaleString('id-ID'), undefined],
              ['Uang masuk', formatRp(data.total_masuk), GREEN],
              ['Uang keluar', formatRp(data.total_keluar), RED],
              ['Selisih masuk − keluar', formatRp(selisih), selisih < 0 ? RED : GREEN],
            ].map(([l, v, c]) => (
              <div key={l} className="card" style={{ padding: 14 }}>
                <div className="kpi-label">{l}</div>
                <div className="tabular" style={{ fontSize: 20, fontWeight: 700, color: c }}>{v}</div>
              </div>
            ))}
          </div>
          {data.belum_lunas > 0 && <p className="text-sm text-muted-foreground" style={{ marginBottom: 10 }}>Termasuk transaksi belum lunas senilai <strong>{formatRp(data.belum_lunas)}</strong>.</p>}

          {data.per_jenis.length > 0 && (
            <div className="card" style={{ padding: 16, marginBottom: 14 }}>
              <div className="chart-title">Rekap per jenis</div>
              <div className="table-scroll"><table className="data-table tabular">
                <thead><tr><th>Jenis</th><th>Arah</th><th style={{ textAlign: 'right' }}>Jumlah</th><th style={{ textAlign: 'right' }}>Total</th></tr></thead>
                <tbody>
                  {data.per_jenis.map((g) => (
                    <tr key={g.jenis} style={{ cursor: 'pointer' }} onClick={() => ubah({ jenis: f.jenis === g.jenis ? '' : g.jenis })} title="Klik untuk menyaring jenis ini">
                      <td>{g.jenis}</td><td>{g.arah === 'masuk' ? 'Masuk' : 'Keluar'}</td>
                      <td style={{ textAlign: 'right' }}>{g.jumlah.toLocaleString('id-ID')}</td>
                      <td style={{ textAlign: 'right', color: g.arah === 'masuk' ? GREEN : RED }}>{formatRp(g.total)}</td>
                    </tr>
                  ))}
                </tbody>
              </table></div>
            </div>
          )}

          <div className="card" style={{ padding: 0 }}>
            <div className="table-scroll"><table className="data-table tabular" style={{ minWidth: 980 }}>
              <thead><tr>
                <th>Tanggal</th><th>No</th><th>Jenis</th><th>Keterangan</th><th>Pelanggan / Supplier</th><th>Channel</th><th>Status</th>
                <th style={{ textAlign: 'right' }}>Masuk</th><th style={{ textAlign: 'right' }}>Keluar</th>
              </tr></thead>
              <tbody>
                {data.transaksi.length === 0 && <tr><td colSpan={9} className="text-muted-foreground">Tidak ada transaksi yang cocok dengan filter ini.</td></tr>}
                {data.transaksi.slice(0, batas).map((t) => {
                  const masuk = arahJenis.get(t.jenis) === 'masuk'
                  return (
                    <tr key={t.id}>
                      <td>{formatTanggal(t.tanggal)}</td><td className="text-muted-foreground">#{t.id}</td><td>{t.jenis}</td>
                      <td>{t.keterangan || t.item}</td><td>{t.pihak}</td><td>{t.channel}</td>
                      <td>{t.lunas ? <span className="pill-status pill-green">Lunas{t.tanggal_lunas && t.tanggal_lunas !== t.tanggal ? ` ${formatTanggal(t.tanggal_lunas)}` : ''}</span> : <span className="pill-status pill-amber">Belum lunas</span>}</td>
                      <td style={{ textAlign: 'right', color: GREEN }}>{masuk ? formatRp(t.nominal) : ''}</td>
                      <td style={{ textAlign: 'right', color: RED }}>{masuk ? '' : formatRp(t.nominal)}</td>
                    </tr>
                  )
                })}
                {data.transaksi.length > 0 && (
                  <tr style={{ fontWeight: 700 }}><td colSpan={7}>Total</td><td style={{ textAlign: 'right' }}>{formatRp(data.total_masuk)}</td><td style={{ textAlign: 'right' }}>{formatRp(data.total_keluar)}</td></tr>
                )}
              </tbody>
            </table></div>
          </div>
          {data.transaksi.length > batas && (
            <button type="button" className="nav-item" style={{ width: 'auto', padding: '8px 14px', marginTop: 10 }} onClick={() => setBatas((b) => b + HALAMAN)}>
              Tampilkan {Math.min(HALAMAN, data.transaksi.length - batas)} lagi (dari {data.transaksi.length})
            </button>
          )}
        </>
      )}
    </>
  )
}
