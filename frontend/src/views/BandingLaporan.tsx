import { useEffect, useMemo, useState } from 'react'
import { angkaSifat } from '@/lib/angkaLaporan'
import { tanggalAkhirPeriode } from '@/lib/rentang'
import { Check, FileDown } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { bandingPdfUrl, getBanding, MAKS_BANDING, type Banding, type BagianBanding, type PeriodOption } from '@/lib/api'
import { ekspor } from '@/lib/garansi'

// Perbandingan Laporan Keuangan untuk bulan-bulan pilihan pengguna (bukan hanya dua), diakhiri
// kolom Total. Laba rugi & arus kas dijumlahkan; neraca memakai posisi bulan terakhir.

const angka = (n: number) => (n < 0 ? `(${Math.abs(n).toLocaleString('id-ID')})` : n.toLocaleString('id-ID'))

export function BandingLaporan({ periods, period, bisaEkspor }: { periods: PeriodOption[]; period: string; bisaEkspor: boolean }) {
  // Urut waktu dari yang paling awal, seperti kolom tabel.
  const urut = useMemo(() => [...periods].sort((a, b) => a.id.localeCompare(b.id)), [periods])
  const [pilih, setPilih] = useState<string[]>(() => {
    const i = urut.findIndex((p) => p.id === period)
    return i > 0 ? [urut[i - 1].id, urut[i].id] : urut.slice(-2).map((p) => p.id)
  })
  const kunci = [...pilih].sort().join(',')
  // Hasil disimpan bersama kuncinya: memuat = hasil terakhir bukan untuk pilihan saat ini.
  const [hasil, setHasil] = useState<{ kunci: string; data?: Banding; error?: string } | null>(null)
  const data = hasil?.data ?? null
  const error = hasil?.kunci === kunci ? hasil.error ?? '' : ''
  const memuat = hasil?.kunci !== kunci

  useEffect(() => {
    const periode = kunci.split(',').filter(Boolean)
    if (periode.length < 2) return
    let batal = false
    getBanding(periode).then(
      (b) => { if (!batal) setHasil({ kunci, data: b }) },
      (e: Error) => { if (!batal) setHasil((h) => ({ kunci, data: h?.data, error: e.message })) },
    )
    return () => { batal = true }
  }, [kunci])

  const toggle = (id: string) => setPilih((s) => s.includes(id) ? s.filter((x) => x !== id) : s.length >= MAKS_BANDING ? s : [...s, id])
  const terakhir = (n: number) => setPilih(urut.slice(-n).map((p) => p.id))

  return (
    <>
      <div className="card" style={{ padding: 20 }}>
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <div className="chart-title">Pilih bulan yang dibandingkan</div>
            <div className="chart-subtitle">Pilih 2 sampai {MAKS_BANDING} bulan, boleh tidak berurutan. Kolom terakhir adalah total.</div>
          </div>
          {bisaEkspor && (
            <Button disabled={pilih.length < 2} onClick={() => void ekspor(bandingPdfUrl([...pilih].sort()))}>
              <FileDown /> Unduh PDF perbandingan
            </Button>
          )}
        </div>
        <div className="flex flex-wrap gap-2" style={{ marginTop: 12 }} role="group" aria-label="Bulan">
          {urut.map((p) => {
            const aktif = pilih.includes(p.id)
            return (
              <Button key={p.id} type="button" size="sm" variant={aktif ? 'default' : 'outline'} aria-pressed={aktif} onClick={() => toggle(p.id)}>
                {aktif && <Check />} {p.label}
              </Button>
            )
          })}
        </div>
        <div className="flex flex-wrap gap-2 text-sm" style={{ marginTop: 10 }}>
          <span className="text-muted-foreground">Cepat:</span>
          <Button variant="link" size="sm" className="h-auto p-0" onClick={() => terakhir(3)}>3 bulan terakhir</Button>
          <Button variant="link" size="sm" className="h-auto p-0" onClick={() => terakhir(6)}>6 bulan terakhir</Button>
          <Button variant="link" size="sm" className="h-auto p-0" onClick={() => terakhir(Math.min(urut.length, MAKS_BANDING))}>Semua bulan</Button>
          <Button variant="link" size="sm" className="h-auto p-0" onClick={() => setPilih([])}>Kosongkan</Button>
        </div>
      </div>

      {pilih.length < 2 && <div className="note-box">Pilih minimal 2 bulan untuk dibandingkan.</div>}
      {error && <div className="pill-status pill-red" role="alert" style={{ alignSelf: 'flex-start', whiteSpace: 'normal' }}>{error}</div>}
      {pilih.length >= 2 && data && (
        <div style={{ opacity: memuat ? 0.6 : 1, display: 'grid', gap: 16 }}>
          {data.bagian.map((b) => <TabelBanding key={b.kode} b={b} kolom={data.kolom} />)}
        </div>
      )}
    </>
  )
}

function TabelBanding({ b, kolom }: { b: BagianBanding; kolom: Banding['kolom'] }) {
  const dua = kolom.length === 2
  const sticky = { position: 'sticky' as const, left: 0, background: 'var(--bg-card)', zIndex: 1 }
  return (
    <div className="card" style={{ padding: 20 }}>
      <div className="chart-title">{b.judul}</div>
      <div className="chart-subtitle" style={{ marginBottom: 10 }}>
        {b.kode === 'neraca' ? 'Posisi per tanggal terakhir tiap bulan; kolom terakhir = posisi tanggal paling akhir.' : 'Kolom Total = jumlah seluruh bulan yang dipilih.'}
      </div>
      <div className="table-scroll">
        <table className="data-table tabular" style={{ minWidth: 260 + 130 * (kolom.length + (dua ? 3 : 1)) }}>
          <thead>
            <tr>
              <th style={sticky}>Akun</th>
              {kolom.map((k) => <th key={k.periode} style={{ textAlign: 'right' }}>{b.kode === 'neraca' ? tanggalAkhirPeriode(k.periode, true) : k.label}</th>)}
              <th style={{ textAlign: 'right' }}>{b.kode === 'neraca' && kolom.length ? `Per ${tanggalAkhirPeriode(kolom[kolom.length - 1].periode, true)}` : 'Total'}</th>
              {dua && <><th style={{ textAlign: 'right' }}>Selisih</th><th style={{ textAlign: 'right' }}>%</th></>}
            </tr>
          </thead>
          <tbody>
            {b.baris.map((r) => {
              if (r.gaya === 'judul') {
                return <tr key={r.kunci}><td colSpan={kolom.length + (dua ? 4 : 2)} className="font-semibold" style={{ ...sticky, background: 'var(--muted)' }}>{r.label}</td></tr>
              }
              const tebal = r.gaya === 'total' ? 'font-semibold' : ''
              const selisih = r.nilai[1] - r.nilai[0]
              const pct = r.nilai[0] !== 0 ? (selisih / Math.abs(r.nilai[0])) * 100 : null
              return (
                <tr key={r.kunci} className={tebal}>
                  <td style={{ ...sticky, paddingLeft: r.gaya === 'sub' ? 28 : undefined, color: r.gaya === 'sub' ? 'var(--text-muted)' : undefined }}>
                    {r.huruf && <span style={{ display: 'inline-block', width: 18, fontWeight: 700, color: 'var(--text-muted)' }}>{r.huruf}</span>}
                    {r.label}
                    {r.rumus && <span style={{ display: 'block', paddingLeft: r.huruf ? 18 : 0, fontSize: 11, fontWeight: 400, color: 'var(--text-muted)' }}>{r.rumus}</span>}
                  </td>
                  {[...r.nilai, r.total].map((v, i) => {
                    const a = angkaSifat(v, r.sifat ?? '')
                    return <td key={i} style={{ textAlign: 'right', fontWeight: i === r.nilai.length ? 700 : undefined, color: a.merah ? 'var(--text-danger)' : undefined }}>{a.teks}</td>
                  })}
                  {dua && <>
                    <td style={{ textAlign: 'right', color: selisih < 0 ? 'var(--text-danger)' : selisih > 0 ? 'var(--accent-green)' : undefined }}>{angka(selisih)}</td>
                    <td style={{ textAlign: 'right', color: 'var(--text-muted)' }}>{pct === null ? '—' : `${pct > 0 ? '+' : ''}${pct.toLocaleString('id-ID', { maximumFractionDigits: 1 })}%`}</td>
                  </>}
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      {dua && <p className="mt-2 text-xs text-muted-foreground">Selisih = {kolom[1].label} dikurangi {kolom[0].label}.</p>}
    </div>
  )
}
