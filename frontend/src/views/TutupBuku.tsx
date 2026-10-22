import { useEffect, useState } from 'react'
import { AlertTriangle, CheckCircle2, Info, Lock, LockOpen } from 'lucide-react'
import { PageHeader } from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { bukaBulan, getTutupBuku, tutupBulan, type PeriodOption, type TutupBuku as Ringkas, type View } from '@/lib/api'
import { formatRp } from '@/lib/format'

// Tutup buku bulanan: cek akhir bulan lalu kunci bulannya. Bulan yang ditutup (dan semua
// sebelumnya) tidak bisa diubah; Owner bisa membuka kembali bulan terakhir yang ditutup.

const IKON = { ok: CheckCircle2, peringatan: AlertTriangle, info: Info }
const WARNA = { ok: 'var(--accent-green)', peringatan: 'var(--accent-amber)', info: 'var(--text-muted)' }

export function TutupBuku({ periods, period, bisaUbah, onBuka, onChanged }: {
  periods: PeriodOption[]; period: string; bisaUbah: boolean; onBuka: (v: View) => void; onChanged: () => void
}) {
  const [bulan, setBulan] = useState(period)
  const [muatUlang, setMuatUlang] = useState(0)
  const [hasil, setHasil] = useState<{ kunci: string; d?: Ringkas; error?: string } | null>(null)
  const [busy, setBusy] = useState(false)
  const [pesan, setPesan] = useState('')
  const kunci = `${bulan}|${muatUlang}`
  useEffect(() => {
    let batal = false
    getTutupBuku(bulan).then((d) => !batal && setHasil({ kunci, d }), (e: Error) => !batal && setHasil({ kunci, error: e.message }))
    return () => { batal = true }
  }, [bulan, kunci])
  const d = hasil?.d
  const peringatan = d?.cek.filter((c) => c.status === 'peringatan').length ?? 0

  async function jalankan(f: () => Promise<Ringkas>, ok: string) {
    setBusy(true)
    setPesan('')
    try {
      await f()
      setPesan(ok)
      setMuatUlang((n) => n + 1)
      onChanged()
    } catch (e) {
      setHasil((h) => h && { ...h, error: (e as Error).message })
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <PageHeader title="Tutup Buku Bulanan" subtitle="Cek akhir bulan, lalu kunci bulan itu supaya laporan yang sudah dipakai (pajak, bank) tidak berubah." />
      <div className="flex flex-wrap items-center gap-3" style={{ marginBottom: 14 }}>
        <label htmlFor="tb-bulan" className="text-sm text-muted-foreground">Bulan</label>
        <select id="tb-bulan" className="input-field" style={{ width: 200 }} value={bulan} onChange={(e) => setBulan(e.target.value)}>
          {periods.map((p) => <option key={p.id} value={p.id}>{p.label}</option>)}
        </select>
        {d && (d.terkunci
          ? <span className="pill-status pill-green inline-flex items-center gap-1"><Lock size={12} /> Sudah ditutup</span>
          : <span className="pill-status pill-amber inline-flex items-center gap-1"><LockOpen size={12} /> Masih terbuka</span>)}
        {d?.tutup_sampai && <span className="text-xs text-muted-foreground">Buku sudah ditutup sampai {periods.find((p) => p.id === d.tutup_sampai)?.label ?? d.tutup_sampai}</span>}
      </div>
      {hasil?.error && <div className="pill-status pill-red" role="alert" style={{ whiteSpace: 'normal' }}>{hasil.error}</div>}
      {pesan && <div className="pill-status pill-green" role="status">{pesan}</div>}

      {d && <>
        <div className="kpi-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(190px, 1fr))', marginBottom: 16 }}>
          {[['Laba (rugi) bersih', d.laba], ['EBITDA', d.ebitda], ['Penyusutan & amortisasi', d.penyusutan], ['Kas akhir bulan', d.kas_akhir]].map(([l, v]) => (
            <div className="card" key={l as string} style={{ padding: '14px 18px' }}>
              <div className="kpi-label">{l}</div>
              <div className="kpi-value tabular" style={{ fontSize: 20, color: (v as number) < 0 ? 'var(--text-danger)' : undefined }}>{formatRp(v as number)}</div>
            </div>
          ))}
        </div>

        <div className="card" style={{ padding: 20 }}>
          <div className="chart-title">Daftar cek akhir bulan — {d.label}</div>
          <div className="chart-subtitle" style={{ marginBottom: 12 }}>
            {peringatan > 0 ? `${peringatan} hal perlu dicek sebelum menutup buku.` : 'Semua cek beres. Buku bulan ini siap ditutup.'}
          </div>
          <ul className="grid gap-2">
            {d.cek.map((c) => {
              const Ikon = IKON[c.status]
              return (
                <li key={c.kode + c.judul} className="flex items-start gap-3 rounded-lg border px-3 py-2.5" style={{ borderColor: 'var(--border-shell)' }}>
                  <Ikon size={20} style={{ color: WARNA[c.status], flexShrink: 0, marginTop: 1 }} aria-label={c.status} />
                  <div style={{ flex: 1 }}>
                    <div className="text-sm font-semibold">{c.judul}</div>
                    <div className="text-xs text-muted-foreground">{c.detail}</div>
                  </div>
                  {c.view && c.status === 'peringatan' && <Button size="sm" variant="outline" onClick={() => onBuka(c.view!)}>Buka</Button>}
                </li>
              )
            })}
          </ul>
          {bisaUbah && (
            <div className="flex flex-wrap items-center gap-3" style={{ marginTop: 16 }}>
              {d.bisa_tutup && (
                <Button disabled={busy} onClick={() => {
                  if (window.confirm(`Tutup buku ${d.label}? Transaksi, stok, dan aset bertanggal ${d.label} dan sebelumnya tidak bisa diubah lagi.${peringatan ? `\n\nMasih ada ${peringatan} peringatan.` : ''}`))
                    void jalankan(() => tutupBulan(d.periode), `Buku ${d.label} sudah ditutup.`)
                }}><Lock /> Tutup buku {d.label}</Button>
              )}
              {d.bisa_buka && (
                <Button variant="outline" disabled={busy} onClick={() => {
                  if (window.confirm(`Buka kembali buku ${d.label}? Data bulan ini bisa diubah lagi; laporan yang sudah dipakai bisa berubah.`))
                    void jalankan(bukaBulan, `Buku ${d.label} dibuka kembali.`)
                }}><LockOpen /> Buka kembali (Owner)</Button>
              )}
              <span className="text-xs text-muted-foreground">Penyusutan & amortisasi dicatat otomatis tiap bulan; menutup buku mengunci angkanya.</span>
            </div>
          )}
        </div>
      </>}
    </>
  )
}
