import { useEffect, useState, type FormEvent } from 'react'
import { CekAwalCard } from '@/components/CekAwalCard'
import { PageHeader } from '@/components/PageHeader'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { getTaxPlanning, type BusinessProfile, type TaxPlanning } from '@/lib/api'
import { formatJt, formatPct, formatRp } from '@/lib/format'
import { angka } from '@/lib/angka'

const fieldClass = 'h-[38px] bg-[var(--bg-input)] tabular'

// skema yang sesuai bentuk usaha saat ini, untuk ditandai "skema Anda sekarang"
const CURRENT: Record<string, string[]> = {
  op: ['pp_op', 'pasal17_op', 'nppn_op'],
  pt_perorangan: ['pp_badan'],
  cv: ['pasal31e'],
  pt: ['pasal31e', 'badan22'],
}

export function PerencanaanPajak({ profile, ptkpCodes }: { profile: BusinessProfile | null; ptkpCodes: string[] }) {
  const [r, setR] = useState<TaxPlanning | null>(null)
  const [omzet, setOmzet] = useState('')
  const [margin, setMargin] = useState('')
  const [ptkp, setPtkp] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function run(input: Parameters<typeof getTaxPlanning>[0]) {
    setBusy(true)
    setError('')
    try {
      const res = await getTaxPlanning(input)
      setR(res)
      setOmzet(String(res.input.omzet))
      setMargin(String(res.input.net_margin_bps / 100))
      setPtkp(res.input.ptkp)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  useEffect(() => { void run({}) }, []) // isi awal dari data transaksi & profil usaha

  function submit(e: FormEvent) {
    e.preventDefault()
    void run({ omzet: parseFloat(omzet || '0'), net_margin_bps: Math.round(parseFloat(margin || '0') * 100), ptkp })
  }

  const bu = r?.bentuk_usaha
  const best = bu?.skema.find((s) => s.id === bu.termurah)
  const mine = CURRENT[profile?.bentuk_usaha ?? ''] ?? []

  return (
    <>
      <PageHeader title="Perencanaan Pajak" subtitle="Bandingkan skema pajak dan bentuk usaha untuk mencari beban pajak paling efisien secara legal" />
      {profile && <div style={{ marginBottom: 16 }}><CekAwalCard profile={profile} /></div>}

      <form className="card" style={{ padding: 22 }} onSubmit={submit}>
        <div className="chart-title">Asumsi setahun (terisi otomatis dari transaksi, bisa diubah)</div>
        <div className="form-grid">
          <div>
            <Label htmlFor="tp-omzet" className="kpi-label">Omzet per Tahun (Rp)</Label>
            <Input id="tp-omzet" className={fieldClass + ' mt-1.5'} {...angka(omzet, (n) => setOmzet(String(n)))} />
          </div>
          <div>
            <Label htmlFor="tp-margin" className="kpi-label">Margin Laba Bersih (%)</Label>
            <Input id="tp-margin" type="number" step="0.1" className={fieldClass + ' mt-1.5'} value={margin} onChange={(e) => setMargin(e.target.value)} />
          </div>
          <div>
            <Label htmlFor="tp-ptkp" className="kpi-label">Status PTKP Pemilik</Label>
            <Select value={ptkp} onValueChange={setPtkp}>
              <SelectTrigger id="tp-ptkp" className={fieldClass + ' mt-1.5 w-full'}><SelectValue /></SelectTrigger>
              <SelectContent>{ptkpCodes.map((c) => <SelectItem key={c} value={c}>{c}</SelectItem>)}</SelectContent>
            </Select>
          </div>
          <div style={{ display: 'flex', alignItems: 'flex-end' }}>
            <button className="btn-cta" style={{ width: '100%' }} type="submit" disabled={busy}>Hitung Ulang</button>
          </div>
        </div>
        {error && <p className="text-sm text-destructive" role="alert" style={{ marginTop: 12 }}>{error}</p>}
      </form>

      {bu && best && (
        <>
          <div className="card" style={{ borderColor: 'var(--accent-green)' }}>
            <div className="kpi-label">Skema paling hemat</div>
            <div className="kpi-value" style={{ color: 'var(--accent-green)', fontSize: 22 }}>{best.nama}</div>
            <div className="kpi-meta-row">
              Pajak {formatRp(best.pajak)} per tahun · laba bersih {formatJt(bu.laba)} · PTKP {formatJt(bu.ptkp)}
            </div>
          </div>

          <div className="card">
            <div className="chart-title">Perbandingan {bu.skema.length} Skema Pajak (per tahun)</div>
            <table className="data-table tabular">
              <thead>
                <tr><th>Skema</th><th>Dasar perhitungan</th><th style={{ textAlign: 'right' }}>Pajak setahun</th><th style={{ textAlign: 'right' }}>Lebih mahal dari termurah</th></tr>
              </thead>
              <tbody>
                {bu.skema.map((s) => (
                  <tr key={s.id}>
                    <td>
                      {s.nama}
                      {s.id === bu.termurah && <span className="pill-status pill-green" style={{ marginLeft: 8 }}>Termurah</span>}
                      {mine.includes(s.id) && <span className="pill-status pill-amber" style={{ marginLeft: 8 }}>Bentuk usaha Anda</span>}
                    </td>
                    <td style={{ color: 'var(--text-muted)' }}>{s.keterangan}</td>
                    <td style={{ textAlign: 'right' }}>{formatRp(s.pajak)}</td>
                    <td style={{ textAlign: 'right' }}>{s.id === bu.termurah ? '-' : formatPct(s.lebih_mahal_bps)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="card">
            <div className="chart-title">Total Beban Pajak per Bentuk Badan (termasuk PPh dividen 10%)</div>
            <table className="data-table tabular">
              <thead>
                <tr><th>Bentuk</th><th style={{ textAlign: 'right' }}>PPh</th><th style={{ textAlign: 'right' }}>PPh Dividen</th><th style={{ textAlign: 'right' }}>Total</th><th style={{ textAlign: 'right' }}>% Omzet</th></tr>
              </thead>
              <tbody>
                {r!.entitas.map((e) => (
                  <tr key={e.id}>
                    <td>{e.nama}</td>
                    <td style={{ textAlign: 'right' }}>{formatRp(e.pph)}</td>
                    <td style={{ textAlign: 'right' }}>{formatRp(e.dividen)}</td>
                    <td style={{ textAlign: 'right', fontWeight: 700 }}>{formatRp(e.total)}</td>
                    <td style={{ textAlign: 'right' }}>{formatPct(e.persen_bps)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <p className="text-xs text-muted-foreground">
            Diadaptasi dari Tax Planning Calculator V2.1 (BSV Consulting). Hasil ini simulasi, bukan nasihat pajak resmi. Syarat tiap skema
            (batas waktu PPh Final, kewajiban pembukuan, notifikasi NPPN) perlu dicek bersama konsultan pajak.
          </p>
        </>
      )}
    </>
  )
}
